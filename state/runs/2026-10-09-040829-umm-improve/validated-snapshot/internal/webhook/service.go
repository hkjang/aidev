package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/cryptoutil"
	"github.com/hkjang/umm/internal/store"
	"github.com/hkjang/umm/internal/textutil"
	"github.com/jackc/pgx/v5"
)

var SupportedEvents = []string{
	"space.updated",
	"note.created", "note.updated", "note.deleted", "note.restored",
	"edge.created", "comment.created", "comment.resolved", "comment.deleted",
	"member.updated", "member.removed", "dream.accepted", "*",
}

const (
	deliveryRetentionDays = 30
	deliveryCleanupPeriod = 6 * time.Hour
)

type Event struct {
	ID         uuid.UUID `json:"id"`
	Type       string    `json:"type"`
	SpaceID    uuid.UUID `json:"spaceId"`
	ResourceID uuid.UUID `json:"resourceId,omitempty"`
	ActorID    uuid.UUID `json:"actorId"`
	Data       any       `json:"data"`
	CreatedAt  time.Time `json:"createdAt"`
}

type subscription struct {
	ID         uuid.UUID
	OwnerID    uuid.UUID
	URL        string
	Ciphertext string
	Active     bool
	Authorized bool
}

type delivery struct {
	ID             uuid.UUID
	SubscriptionID uuid.UUID
	Payload        []byte
	ClaimedAt      time.Time
}

type Service struct {
	Store            *store.Store
	Cipher           *cryptoutil.Cipher
	wake             chan struct{}
	client           *http.Client
	validateEndpoint func(context.Context, string) error
	dispatchSlots    chan struct{}
}

func New(database *store.Store, cipher *cryptoutil.Cipher) *Service {
	service := &Service{
		Store: database, Cipher: cipher, wake: make(chan struct{}, 1), validateEndpoint: ValidateEndpoint,
		dispatchSlots: make(chan struct{}, store.MaxWebhookLeaseConnections),
	}
	transport := &http.Transport{
		// A proxy could resolve the target after our public-IP check and reopen
		// the SSRF path. Webhooks always use the guarded direct dialer instead.
		Proxy:                 nil,
		DialContext:           safeDialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          20,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 8 * time.Second,
	}
	service.client = &http.Client{Timeout: 12 * time.Second, Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error {
		return errors.New("webhook redirects are not allowed")
	}}
	return service
}

func (s *Service) Start(ctx context.Context) {
	for range store.MaxWebhookLeaseConnections {
		go s.worker(ctx)
	}
	go s.cleanupLoop(ctx)
	s.signal()
}

func prepareEvent(event Event) (Event, []byte, error) {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	payload, err := json.Marshal(event)
	return event, payload, err
}

func (s *Service) signal() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// Enqueue persists one delivery per currently subscribed and authorized owner.
// The wake channel is only a latency hint; PostgreSQL remains the source of
// truth, so a full channel or process restart cannot discard accepted events.
func (s *Service) Enqueue(ctx context.Context, event Event) (int64, error) {
	event, payload, err := prepareEvent(event)
	if err != nil {
		return 0, err
	}
	tag, err := s.Store.Pool.Exec(ctx, `
		INSERT INTO webhook_deliveries(subscription_id,event_id,event_type,payload,status,next_attempt_at)
		SELECT ws.id,$3,$1,$4,'queued',now() FROM webhook_subscriptions ws
		JOIN users owner_user ON owner_user.id=ws.owner_id AND owner_user.active
		WHERE ws.active AND ($1=ANY(ws.events) OR '*'=ANY(ws.events))
		  AND ($2::uuid IS NULL OR EXISTS(
		    SELECT 1 FROM spaces sp LEFT JOIN space_members sm ON sm.space_id=sp.id AND sm.user_id=ws.owner_id
		    WHERE sp.id=$2 AND (sp.owner_id=ws.owner_id OR sm.user_id=ws.owner_id)))
		ON CONFLICT(subscription_id,event_id) DO NOTHING`, event.Type, nullableUUID(event.SpaceID), event.ID, json.RawMessage(payload))
	if err != nil {
		return 0, err
	}
	if tag.RowsAffected() > 0 {
		s.signal()
	}
	return tag.RowsAffected(), nil
}

func (s *Service) Test(ctx context.Context, subscriptionID, actorID uuid.UUID) error {
	releaseSlot, err := s.acquireDispatchSlot(ctx)
	if err != nil {
		return err
	}
	defer releaseSlot()
	event := Event{ID: uuid.New(), Type: "webhook.test", ActorID: actorID, Data: map[string]any{"message": "umm webhook connection test"}, CreatedAt: time.Now().UTC()}
	event, payload, err := prepareEvent(event)
	if err != nil {
		return err
	}
	var item delivery
	err = s.Store.Pool.QueryRow(ctx, `
		INSERT INTO webhook_deliveries(subscription_id,event_id,event_type,payload,status,claimed_at,attempted_at)
		SELECT id,$3,$4,$5,'processing',now(),now() FROM webhook_subscriptions
		WHERE id=$1 AND owner_id=$2 AND active RETURNING id,subscription_id,payload,claimed_at`,
		subscriptionID, actorID, event.ID, event.Type, json.RawMessage(payload)).
		Scan(&item.ID, &item.SubscriptionID, &item.Payload, &item.ClaimedAt)
	if err != nil {
		return err
	}
	return s.deliverClaimedWithSlot(ctx, item)
}

func (s *Service) worker(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		releaseSlot, slotErr := s.acquireDispatchSlot(ctx)
		if slotErr != nil {
			return
		}
		item, claimed, err := s.claimNext(ctx)
		if err != nil && ctx.Err() == nil {
			slog.Warn("webhook delivery claim failed", "error", err)
		}
		if claimed {
			if err := s.deliverClaimedWithSlot(ctx, item); err != nil && ctx.Err() == nil {
				slog.Warn("webhook delivery failed", "delivery_id", item.ID, "error", err)
			}
			releaseSlot()
			continue
		}
		releaseSlot()
		select {
		case <-ctx.Done():
			return
		case <-s.wake:
		case <-ticker.C:
		}
	}
}

func (s *Service) acquireDispatchSlot(ctx context.Context) (func(), error) {
	select {
	case s.dispatchSlots <- struct{}{}:
		return func() { <-s.dispatchSlots }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *Service) claimNext(ctx context.Context) (delivery, bool, error) {
	var item delivery
	err := s.Store.Pool.QueryRow(ctx, `
		WITH candidate AS (
		  SELECT id FROM webhook_deliveries
		  WHERE (status='queued' AND next_attempt_at<=now())
		     OR (status='processing' AND claimed_at<now()-interval '2 minutes')
		  ORDER BY next_attempt_at,created_at,id
		  FOR UPDATE SKIP LOCKED LIMIT 1
		)
		UPDATE webhook_deliveries delivery
		SET status='processing',claimed_at=now(),attempted_at=now()
		FROM candidate WHERE delivery.id=candidate.id
		RETURNING delivery.id,delivery.subscription_id,delivery.payload,delivery.claimed_at`).
		Scan(&item.ID, &item.SubscriptionID, &item.Payload, &item.ClaimedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return delivery{}, false, nil
	}
	return item, err == nil, err
}

func (s *Service) deliverClaimed(ctx context.Context, item delivery) error {
	releaseSlot, err := s.acquireDispatchSlot(ctx)
	if err != nil {
		return err
	}
	defer releaseSlot()
	return s.deliverClaimedWithSlot(ctx, item)
}

func (s *Service) deliverClaimedWithSlot(ctx context.Context, item delivery) error {
	var event Event
	if err := json.Unmarshal(item.Payload, &event); err != nil {
		recordErr := s.finishFailure(ctx, item, 0, err, 0, false)
		return errors.Join(err, recordErr)
	}
	tx, releaseConnection, err := s.Store.BeginWebhookLease(ctx)
	if err != nil {
		return err
	}
	leaseOpen := true
	releaseLease := func() {
		if !leaseOpen {
			return
		}
		leaseOpen = false
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = tx.Rollback(rollbackCtx)
		cancel()
		releaseConnection()
	}
	commitLease := func() error {
		if !leaseOpen {
			return pgx.ErrTxClosed
		}
		leaseOpen = false
		commitErr := tx.Commit(ctx)
		releaseConnection()
		return commitErr
	}
	defer releaseLease()

	var sub subscription
	var ownerActive bool
	err = tx.QueryRow(ctx, `
		SELECT ws.id,ws.owner_id,ws.url,ws.secret_ciphertext,ws.active,u.active
		FROM webhook_subscriptions ws JOIN users u ON u.id=ws.owner_id
		WHERE ws.id=$1
		FOR NO KEY UPDATE OF ws
		FOR SHARE OF u`, item.SubscriptionID).
		Scan(&sub.ID, &sub.OwnerID, &sub.URL, &sub.Ciphertext, &sub.Active, &ownerActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	var deliveryStatus string
	err = tx.QueryRow(ctx, `
		SELECT status FROM webhook_deliveries
		WHERE id=$1 AND subscription_id=$2 AND status='processing' AND claimed_at=$3
		FOR UPDATE`, item.ID, item.SubscriptionID, item.ClaimedAt).Scan(&deliveryStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	sub.Authorized = ownerActive
	if sub.Authorized && event.SpaceID != uuid.Nil {
		var spaceOwnerID uuid.UUID
		err = tx.QueryRow(ctx, `SELECT owner_id FROM spaces WHERE id=$1 FOR SHARE`, event.SpaceID).Scan(&spaceOwnerID)
		if errors.Is(err, pgx.ErrNoRows) {
			sub.Authorized = false
		} else if err != nil {
			return err
		} else if spaceOwnerID != sub.OwnerID {
			var memberID uuid.UUID
			err = tx.QueryRow(ctx, `SELECT user_id FROM space_members WHERE space_id=$1 AND user_id=$2 FOR SHARE`, event.SpaceID, sub.OwnerID).Scan(&memberID)
			if errors.Is(err, pgx.ErrNoRows) {
				sub.Authorized = false
			} else if err != nil {
				return err
			}
		}
	}
	finishLeaseFailure := func(deliveryErr error, status, attempts int, countSubscription bool) error {
		recordErr := s.finishFailureTx(ctx, tx, item, status, deliveryErr, attempts, countSubscription)
		if recordErr == nil {
			recordErr = commitLease()
		}
		return errors.Join(deliveryErr, recordErr)
	}
	if !sub.Active {
		err = errors.New("webhook subscription is inactive")
		return finishLeaseFailure(err, 0, 0, false)
	}
	if !sub.Authorized {
		err = errors.New("webhook subscription owner is no longer authorized for the event")
		return finishLeaseFailure(err, 0, 0, false)
	}
	if err = s.validateEndpoint(ctx, sub.URL); err != nil {
		return finishLeaseFailure(err, 0, 0, true)
	}
	secret, err := s.Cipher.Decrypt(sub.Ciphertext)
	if err != nil {
		return finishLeaseFailure(err, 0, 0, true)
	}
	payload := item.Payload
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signed := timestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signed))
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	var lastErr error
	lastStatus := 0
	attempts := 0
	for attempt := 0; attempt < 3; attempt++ {
		request, requestErr := http.NewRequestWithContext(ctx, http.MethodPost, sub.URL, bytes.NewReader(payload))
		if requestErr != nil {
			lastErr = requestErr
			break
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("User-Agent", "umm-webhooks/0.7")
		request.Header.Set("X-Umm-Event", event.Type)
		request.Header.Set("X-Umm-Delivery", event.ID.String())
		request.Header.Set("X-Umm-Timestamp", timestamp)
		request.Header.Set("X-Umm-Signature-256", signature)
		attempts++
		response, responseErr := s.client.Do(request)
		if responseErr == nil {
			lastStatus = response.StatusCode
			_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
			response.Body.Close()
			if response.StatusCode >= 200 && response.StatusCode < 300 {
				if err = s.finishSuccessTx(ctx, tx, item, response.StatusCode, attempts); err != nil {
					return err
				}
				return commitLease()
			}
			lastErr = fmt.Errorf("webhook returned HTTP %d", response.StatusCode)
			if response.StatusCode < 500 && response.StatusCode != http.StatusTooManyRequests {
				break
			}
		} else {
			lastErr = responseErr
		}
		if attempt < 2 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt+1) * 250 * time.Millisecond):
			}
		}
	}
	if lastErr == nil {
		lastErr = errors.New("webhook delivery failed")
	}
	return finishLeaseFailure(lastErr, lastStatus, attempts, true)
}

func (s *Service) finishSuccess(ctx context.Context, item delivery, status, attempts int) error {
	tx, err := s.Store.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = s.finishSuccessTx(ctx, tx, item, status, attempts); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Service) finishSuccessTx(ctx context.Context, tx pgx.Tx, item delivery, status, attempts int) error {
	if _, err := tx.Exec(ctx, `UPDATE webhook_deliveries SET status='delivered',payload='{}'::jsonb,attempt_count=attempt_count+$2,response_status=$3,error='',claimed_at=NULL,attempted_at=now(),delivered_at=now() WHERE id=$1 AND status='processing'`, item.ID, attempts, status); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE webhook_subscriptions SET failure_count=0,last_error='',last_delivered_at=now(),updated_at=now() WHERE id=$1`, item.SubscriptionID)
	return err
}

func (s *Service) finishFailure(ctx context.Context, item delivery, status int, deliveryErr error, attempts int, countSubscription bool) error {
	tx, err := s.Store.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = s.finishFailureTx(ctx, tx, item, status, deliveryErr, attempts, countSubscription); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Service) finishFailureTx(ctx context.Context, tx pgx.Tx, item delivery, status int, deliveryErr error, attempts int, countSubscription bool) error {
	message := "delivery failed"
	if deliveryErr != nil {
		message = deliveryErr.Error()
	}
	message = textutil.LimitUTF8Bytes(message, 500)
	var responseStatus any
	if status > 0 {
		responseStatus = status
	}
	if _, err := tx.Exec(ctx, `UPDATE webhook_deliveries SET status='failed',payload='{}'::jsonb,attempt_count=attempt_count+$2,response_status=$3,error=$4,claimed_at=NULL,attempted_at=now() WHERE id=$1 AND status='processing'`, item.ID, attempts, responseStatus, message); err != nil {
		return err
	}
	if countSubscription {
		if _, err := tx.Exec(ctx, `UPDATE webhook_subscriptions SET failure_count=failure_count+1,last_error=$2,active=CASE WHEN failure_count+1>=10 THEN false ELSE active END,updated_at=now() WHERE id=$1`, item.SubscriptionID, message); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) cleanupLoop(ctx context.Context) {
	s.cleanupDeliveries(ctx)
	ticker := time.NewTicker(deliveryCleanupPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.cleanupDeliveries(ctx)
		}
	}
}

func (s *Service) cleanupDeliveries(ctx context.Context) {
	if _, err := s.Store.Pool.Exec(ctx, `UPDATE webhook_deliveries SET payload='{}'::jsonb WHERE status IN ('delivered','failed') AND payload<>'{}'::jsonb`); err != nil && ctx.Err() == nil {
		slog.Warn("webhook terminal payload cleanup failed", "error", err)
	}
	if _, err := s.Store.Pool.Exec(ctx, `DELETE FROM webhook_deliveries WHERE status IN ('delivered','failed') AND attempted_at<now()-make_interval(days=>$1)`, deliveryRetentionDays); err != nil && ctx.Err() == nil {
		slog.Warn("webhook delivery retention cleanup failed", "error", err)
	}
}

func ValidateEvents(events []string) bool {
	if len(events) == 0 || len(events) > len(SupportedEvents) {
		return false
	}
	for _, event := range events {
		if !slices.Contains(SupportedEvents, event) {
			return false
		}
	}
	return true
}

func ValidateEndpoint(ctx context.Context, raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil || parsed.Fragment != "" {
		return errors.New("webhook URL must be an HTTPS URL without credentials or fragments")
	}
	if parsed.Port() != "" && parsed.Port() != "443" {
		return errors.New("webhook URL must use the default HTTPS port")
	}
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, parsed.Hostname())
	if err != nil || len(addresses) == 0 {
		return errors.New("webhook hostname could not be resolved")
	}
	for _, address := range addresses {
		if !publicIP(address.IP) {
			return errors.New("webhook URL resolves to a private or reserved address")
		}
	}
	return nil
}

func safeDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	dialer := net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
	for _, candidate := range addresses {
		if !publicIP(candidate.IP) {
			continue
		}
		connection, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(candidate.IP.String(), port))
		if dialErr == nil {
			return connection, nil
		}
		err = dialErr
	}
	if err == nil {
		err = errors.New("target has no public address")
	}
	return nil, err
}

func publicIP(ip net.IP) bool {
	if ip == nil || !ip.IsGlobalUnicast() || ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}
	for _, network := range reservedNetworks {
		if network.Contains(ip) {
			return false
		}
	}
	return true
}

var reservedNetworks = func() []*net.IPNet {
	blocks := []string{
		"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24",
		"198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4",
		"100::/64", "2001::/23", "2001:db8::/32",
	}
	networks := make([]*net.IPNet, 0, len(blocks))
	for _, block := range blocks {
		_, network, err := net.ParseCIDR(block)
		if err != nil {
			panic(err)
		}
		networks = append(networks, network)
	}
	return networks
}()

func nullableUUID(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}
