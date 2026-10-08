package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/cryptoutil"
	"github.com/hkjang/umm/internal/store"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestNewUsesGuardedDirectTransport(t *testing.T) {
	service := New(nil, nil)
	transport, ok := service.client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("webhook transport has type %T, want *http.Transport", service.client.Transport)
	}
	if transport.Proxy != nil {
		t.Fatal("webhook transport must not use a proxy")
	}
	if transport.DialContext == nil {
		t.Fatal("webhook transport must use the guarded dialer")
	}
}

func TestPublicIPRejectsPrivateAndReservedRanges(t *testing.T) {
	tests := map[string]bool{
		"8.8.8.8":              true,
		"2606:4700:4700::1111": true,
		"127.0.0.1":            false,
		"10.1.2.3":             false,
		"100.64.0.1":           false,
		"169.254.1.1":          false,
		"192.0.2.8":            false,
		"198.18.0.1":           false,
		"198.51.100.8":         false,
		"203.0.113.8":          false,
		"240.0.0.1":            false,
		"::1":                  false,
		"fc00::1":              false,
		"fe80::1":              false,
		"2001:db8::1":          false,
	}
	for raw, want := range tests {
		if got := publicIP(net.ParseIP(raw)); got != want {
			t.Errorf("publicIP(%q) = %v, want %v", raw, got, want)
		}
	}
}

func TestValidateEndpointRejectsUnsafeURLs(t *testing.T) {
	for _, raw := range []string{
		"http://example.com/hook",
		"https://user:pass@example.com/hook",
		"https://example.com:8443/hook",
		"https://example.com/hook#secret",
		"https://127.0.0.1/hook",
		"https://localhost/hook",
	} {
		if err := ValidateEndpoint(context.Background(), raw); err == nil {
			t.Errorf("ValidateEndpoint(%q) accepted an unsafe URL", raw)
		}
	}
}

func TestValidateEvents(t *testing.T) {
	if !ValidateEvents([]string{"note.created", "comment.created"}) || !ValidateEvents([]string{"*"}) {
		t.Fatal("supported webhook events were rejected")
	}
	if ValidateEvents(nil) || ValidateEvents([]string{"unknown.event"}) {
		t.Fatal("unsupported webhook events were accepted")
	}
}

func TestDurableQueueSurvivesServiceRestartIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Close()
	if err = db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	cipher, err := cryptoutil.New(bytes.Repeat([]byte{0x42}, 32))
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, err := cipher.Encrypt("restart-safe-secret")
	if err != nil {
		t.Fatal(err)
	}

	userID, ownerID, subscriptionID, eventID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	username := "webhook_restart_" + userID.String()
	ownerName := "webhook_space_owner_" + ownerID.String()
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text),($3,$4::citext,$4::text)`, userID, username, ownerID, ownerName); err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Exec(ctx, `DELETE FROM users WHERE id=$1 OR id=$2`, userID, ownerID)
	if _, err = db.Pool.Exec(ctx, `INSERT INTO webhook_subscriptions(id,owner_id,name,url,secret_ciphertext,events) VALUES($1,$2,'restart queue','https://example.com/webhook',$3,ARRAY['note.created'])`, subscriptionID, userID, ciphertext); err != nil {
		t.Fatal(err)
	}

	beforeRestart := New(db, cipher)
	queued, err := beforeRestart.Enqueue(ctx, Event{ID: eventID, Type: "note.created", ActorID: userID, Data: map[string]any{"source": "integration"}})
	if err != nil {
		t.Fatal(err)
	}
	if queued != 1 {
		t.Fatalf("persisted deliveries = %d, want 1", queued)
	}
	var deliveryID uuid.UUID
	var status string
	var payload []byte
	if err = db.Pool.QueryRow(ctx, `UPDATE webhook_deliveries SET next_attempt_at='2000-01-01',created_at='2000-01-01' WHERE subscription_id=$1 AND event_id=$2 RETURNING id,status,payload`, subscriptionID, eventID).
		Scan(&deliveryID, &status, &payload); err != nil {
		t.Fatal(err)
	}
	if status != "queued" {
		t.Fatalf("delivery status = %q, want queued", status)
	}
	var persisted Event
	if err = json.Unmarshal(payload, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.ID != eventID || persisted.Type != "note.created" || persisted.CreatedAt.IsZero() {
		t.Fatalf("persisted event is incomplete: %#v", persisted)
	}

	afterRestart := New(db, cipher)
	claimed, ok, err := afterRestart.claimNext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || claimed.ID != deliveryID || claimed.SubscriptionID != subscriptionID {
		t.Fatalf("restarted service claimed %#v, ok=%v; want delivery %s", claimed, ok, deliveryID)
	}
	if err = db.Pool.QueryRow(ctx, `SELECT status FROM webhook_deliveries WHERE id=$1`, deliveryID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "processing" {
		t.Fatalf("claimed delivery status = %q, want processing", status)
	}
	if elapsed := time.Since(persisted.CreatedAt); elapsed < 0 || elapsed > time.Minute {
		t.Fatalf("unexpected persisted event timestamp %s", persisted.CreatedAt)
	}
	if err = afterRestart.finishSuccess(ctx, claimed, http.StatusNoContent, 1); err != nil {
		t.Fatal(err)
	}
	if err = db.Pool.QueryRow(ctx, `SELECT status,payload FROM webhook_deliveries WHERE id=$1`, deliveryID).Scan(&status, &payload); err != nil {
		t.Fatal(err)
	}
	if status != "delivered" || string(payload) != "{}" {
		t.Fatalf("terminal delivery status=%q payload=%s, want delivered/{}", status, payload)
	}

	spaceID, revokedEventID := uuid.New(), uuid.New()
	if _, err = db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'webhook authorization')`, spaceID, ownerID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO space_members(space_id,user_id,permission) VALUES($1,$2,'view')`, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	queued, err = afterRestart.Enqueue(ctx, Event{ID: revokedEventID, Type: "note.created", SpaceID: spaceID, ActorID: ownerID, Data: map[string]any{"scope": "current"}})
	if err != nil {
		t.Fatal(err)
	}
	if queued != 1 {
		t.Fatalf("authorized deliveries = %d, want 1", queued)
	}
	if _, err = db.Pool.Exec(ctx, `DELETE FROM space_members WHERE space_id=$1 AND user_id=$2`, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `UPDATE webhook_deliveries SET next_attempt_at='2000-01-01',created_at='2000-01-01' WHERE subscription_id=$1 AND event_id=$2`, subscriptionID, revokedEventID); err != nil {
		t.Fatal(err)
	}
	revoked, ok, err := afterRestart.claimNext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("delivery was not claimable after access revocation")
	}
	if err = afterRestart.deliverClaimed(ctx, revoked); err == nil {
		t.Fatal("delivery proceeded after the subscription owner lost space access")
	}
	var failureCount int
	if err = db.Pool.QueryRow(ctx, `SELECT d.status,d.payload,s.failure_count FROM webhook_deliveries d JOIN webhook_subscriptions s ON s.id=d.subscription_id WHERE d.id=$1`, revoked.ID).Scan(&status, &payload, &failureCount); err != nil {
		t.Fatal(err)
	}
	if status != "failed" || string(payload) != "{}" || failureCount != 0 {
		t.Fatalf("revoked delivery status=%q payload=%s failure_count=%d, want failed/{}/0", status, payload, failureCount)
	}
	if _, err = db.Pool.Exec(ctx, `UPDATE webhook_deliveries SET attempted_at=now()-interval '31 days' WHERE subscription_id=$1`, subscriptionID); err != nil {
		t.Fatal(err)
	}
	afterRestart.cleanupDeliveries(ctx)
	var retained int
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM webhook_deliveries WHERE subscription_id=$1`, subscriptionID).Scan(&retained); err != nil {
		t.Fatal(err)
	}
	if retained != 0 {
		t.Fatalf("terminal delivery retention kept %d rows older than %d days", retained, deliveryRetentionDays)
	}
}

func TestWebhookDeliveryHoldsAuthorizationThroughDispatchIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Close()
	if err = db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	cipher, err := cryptoutil.New(bytes.Repeat([]byte{0x51}, 32))
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, err := cipher.Encrypt("authorization-lease-secret")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		mutation string
		target   string
	}{
		{name: "membership removed", mutation: `DELETE FROM space_members WHERE space_id=$1 AND user_id=$2`, target: "membership"},
		{name: "owner deactivated", mutation: `UPDATE users SET active=false WHERE id=$1`, target: "user"},
		{name: "subscription disabled", mutation: `UPDATE webhook_subscriptions SET active=false WHERE id=$1`, target: "subscription"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spaceOwnerID, subscriberID, spaceID := uuid.New(), uuid.New(), uuid.New()
			subscriptionID, firstDeliveryID := uuid.New(), uuid.New()
			ownerName := "webhook_lease_owner_" + spaceOwnerID.String()
			subscriberName := "webhook_lease_subscriber_" + subscriberID.String()
			if _, err = db.Pool.Exec(ctx, `
				INSERT INTO users(id,username,display_name) VALUES
				($1,$2::citext,$2::text),($3,$4::citext,$4::text)`, spaceOwnerID, ownerName, subscriberID, subscriberName); err != nil {
				t.Fatal(err)
			}
			defer db.Pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1 OR id=$2`, spaceOwnerID, subscriberID)
			if _, err = db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'webhook delivery lease')`, spaceID, spaceOwnerID); err != nil {
				t.Fatal(err)
			}
			if _, err = db.Pool.Exec(ctx, `INSERT INTO space_members(space_id,user_id,permission) VALUES($1,$2,'view')`, spaceID, subscriberID); err != nil {
				t.Fatal(err)
			}
			if _, err = db.Pool.Exec(ctx, `INSERT INTO webhook_subscriptions(id,owner_id,name,url,secret_ciphertext,events) VALUES($1,$2,'authorization lease','https://example.com/webhook',$3,ARRAY['note.created'])`, subscriptionID, subscriberID, ciphertext); err != nil {
				t.Fatal(err)
			}
			firstEvent, firstPayload, err := prepareEvent(Event{ID: uuid.New(), Type: "note.created", SpaceID: spaceID, ActorID: spaceOwnerID, Data: map[string]any{"body": "authorized payload"}})
			if err != nil {
				t.Fatal(err)
			}
			var firstClaimedAt time.Time
			if err = db.Pool.QueryRow(ctx, `INSERT INTO webhook_deliveries(id,subscription_id,event_id,event_type,payload,status,claimed_at) VALUES($1,$2,$3,$4,$5,'processing',now()) RETURNING claimed_at`, firstDeliveryID, subscriptionID, firstEvent.ID, firstEvent.Type, json.RawMessage(firstPayload)).Scan(&firstClaimedAt); err != nil {
				t.Fatal(err)
			}

			gatewayStarted := make(chan struct{}, 1)
			releaseGateway := make(chan struct{})
			release := func() {
				select {
				case <-releaseGateway:
				default:
					close(releaseGateway)
				}
			}
			defer release()
			var calls atomic.Int32
			service := New(db, cipher)
			service.validateEndpoint = func(context.Context, string) error { return nil }
			service.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				calls.Add(1)
				gatewayStarted <- struct{}{}
				select {
				case <-releaseGateway:
				case <-request.Context().Done():
					return nil, request.Context().Err()
				}
				return &http.Response{StatusCode: http.StatusNoContent, Body: http.NoBody, Header: make(http.Header), Request: request}, nil
			})}
			staleClaimedAt := firstClaimedAt
			if err = db.Pool.QueryRow(ctx, `UPDATE webhook_deliveries SET claimed_at=claimed_at+interval '1 second' WHERE id=$1 RETURNING claimed_at`, firstDeliveryID).Scan(&firstClaimedAt); err != nil {
				t.Fatal(err)
			}
			if err = service.deliverClaimed(ctx, delivery{ID: firstDeliveryID, SubscriptionID: subscriptionID, Payload: firstPayload, ClaimedAt: staleClaimedAt}); err != nil {
				t.Fatalf("stale delivery claim returned an error: %v", err)
			}
			if calls.Load() != 0 {
				t.Fatalf("stale delivery claim reached the external endpoint: calls=%d", calls.Load())
			}

			deliveryDone := make(chan error, 1)
			go func() {
				deliveryDone <- service.deliverClaimed(ctx, delivery{ID: firstDeliveryID, SubscriptionID: subscriptionID, Payload: firstPayload, ClaimedAt: firstClaimedAt})
			}()
			select {
			case <-gatewayStarted:
			case <-time.After(5 * time.Second):
				t.Fatal("webhook delivery did not reach the external endpoint")
			}

			mutationDone := make(chan error, 1)
			mutationArgs := []any{spaceID, subscriberID}
			if test.target == "user" {
				mutationArgs = []any{subscriberID}
			} else if test.target == "subscription" {
				mutationArgs = []any{subscriptionID}
			}
			go func() {
				_, mutationErr := db.Pool.Exec(context.Background(), test.mutation, mutationArgs...)
				mutationDone <- mutationErr
			}()
			select {
			case mutationErr := <-mutationDone:
				t.Fatalf("authorization mutation bypassed the active webhook delivery lease: %v", mutationErr)
			case <-time.After(150 * time.Millisecond):
			}
			release()

			select {
			case deliveryErr := <-deliveryDone:
				if deliveryErr != nil {
					t.Fatalf("authorized webhook delivery failed: %v", deliveryErr)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("webhook delivery did not finish after endpoint release")
			}
			select {
			case mutationErr := <-mutationDone:
				if mutationErr != nil {
					t.Fatal(mutationErr)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("authorization mutation did not resume after webhook lease release")
			}

			secondEvent, secondPayload, err := prepareEvent(Event{ID: uuid.New(), Type: "note.created", SpaceID: spaceID, ActorID: spaceOwnerID, Data: map[string]any{"body": "revoked payload"}})
			if err != nil {
				t.Fatal(err)
			}
			secondDeliveryID := uuid.New()
			var secondClaimedAt time.Time
			if err = db.Pool.QueryRow(ctx, `INSERT INTO webhook_deliveries(id,subscription_id,event_id,event_type,payload,status,claimed_at) VALUES($1,$2,$3,$4,$5,'processing',now()) RETURNING claimed_at`, secondDeliveryID, subscriptionID, secondEvent.ID, secondEvent.Type, json.RawMessage(secondPayload)).Scan(&secondClaimedAt); err != nil {
				t.Fatal(err)
			}
			if err = service.deliverClaimed(ctx, delivery{ID: secondDeliveryID, SubscriptionID: subscriptionID, Payload: secondPayload, ClaimedAt: secondClaimedAt}); err == nil {
				t.Fatal("webhook delivery after completed authorization revocation was allowed")
			}
			if calls.Load() != 1 {
				t.Fatalf("revoked webhook reached the external endpoint: calls=%d", calls.Load())
			}
			var firstStatus, secondStatus string
			var secondStoredPayload []byte
			if err = db.Pool.QueryRow(ctx, `SELECT status FROM webhook_deliveries WHERE id=$1`, firstDeliveryID).Scan(&firstStatus); err != nil {
				t.Fatal(err)
			}
			if err = db.Pool.QueryRow(ctx, `SELECT status,payload FROM webhook_deliveries WHERE id=$1`, secondDeliveryID).Scan(&secondStatus, &secondStoredPayload); err != nil {
				t.Fatal(err)
			}
			if firstStatus != "delivered" || secondStatus != "failed" || string(secondStoredPayload) != "{}" {
				t.Fatalf("delivery states after revocation = first:%q second:%q payload:%s", firstStatus, secondStatus, secondStoredPayload)
			}
		})
	}
}

func TestFinishFailurePersistsBoundedUTF8Integration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Close()
	if err = db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	userID, subscriptionID, deliveryID := uuid.New(), uuid.New(), uuid.New()
	username := "webhook_utf8_" + userID.String()
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, username); err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, userID)
	if _, err = db.Pool.Exec(ctx, `INSERT INTO webhook_subscriptions(id,owner_id,name,url,secret_ciphertext,events) VALUES($1,$2,'UTF-8 failure','https://example.com/webhook','unused',ARRAY['note.created'])`, subscriptionID, userID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO webhook_deliveries(id,subscription_id,event_id,event_type,payload,status,claimed_at) VALUES($1,$2,$3,'note.created','{}','processing',now())`, deliveryID, subscriptionID, uuid.New()); err != nil {
		t.Fatal(err)
	}

	rawError := strings.Repeat("a", 499) + "한" + string([]byte{0xff}) + "tail"
	service := New(db, nil)
	if err = service.finishFailure(ctx, delivery{ID: deliveryID, SubscriptionID: subscriptionID}, http.StatusBadGateway, errors.New(rawError), 2, true); err != nil {
		t.Fatal(err)
	}

	var status, deliveryError, lastError string
	var attemptCount, responseStatus, failureCount int
	if err = db.Pool.QueryRow(ctx, `
		SELECT d.status,d.error,d.attempt_count,d.response_status,s.failure_count,s.last_error
		FROM webhook_deliveries d JOIN webhook_subscriptions s ON s.id=d.subscription_id
		WHERE d.id=$1`, deliveryID).Scan(&status, &deliveryError, &attemptCount, &responseStatus, &failureCount, &lastError); err != nil {
		t.Fatal(err)
	}
	want := strings.Repeat("a", 499)
	if status != "failed" || deliveryError != want || lastError != want || attemptCount != 2 || responseStatus != http.StatusBadGateway || failureCount != 1 {
		t.Fatalf("persisted failure status=%q error=%q last_error=%q attempts=%d response=%d failures=%d", status, deliveryError, lastError, attemptCount, responseStatus, failureCount)
	}
	if !utf8.ValidString(deliveryError) || len(deliveryError) > 500 {
		t.Fatalf("delivery error must be valid UTF-8 within 500 bytes: len=%d valid=%v", len(deliveryError), utf8.ValidString(deliveryError))
	}
}

// Deleting a webhook has to stop what was already on its way.
//
// The three cases above all check the same shape: a delivery survives, and the
// dispatcher refuses it because the authorization behind it went away. Deleting
// the subscription is a different mechanism — the delivery rows go with it,
// through the foreign key — and it is the one a person actually performs when
// they mean "stop sending my notes there". Nothing held it.
//
// A schema rewrite that dropped the cascade would leave queued deliveries with
// a subscription that no longer exists, and revoking would stop meaning what
// the button says. This fails first instead.
func TestDeletingAWebhookDropsWhatWasAlreadyQueuedIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Close()
	if err = db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	cipher, err := cryptoutil.New(bytes.Repeat([]byte{0x62}, 32))
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, err := cipher.Encrypt("revoked-webhook-secret")
	if err != nil {
		t.Fatal(err)
	}

	ownerID, spaceID, subscriptionID := uuid.New(), uuid.New(), uuid.New()
	ownerName := "webhook_revoke_" + ownerID.String()
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, ownerID, ownerName); err != nil {
		t.Fatal(err)
	}
	// Cleaned up by hand rather than by deleting the user and hoping it
	// cascades: this package's tests share one database, and a subscription
	// left behind makes the next Enqueue queue one delivery more than the test
	// doing it expects. That is how this test first broke two of its
	// neighbours.
	defer func() {
		_, _ = db.Pool.Exec(context.Background(), `DELETE FROM webhook_subscriptions WHERE id=$1`, subscriptionID)
		_, _ = db.Pool.Exec(context.Background(), `DELETE FROM spaces WHERE id=$1`, spaceID)
		_, _ = db.Pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, ownerID)
	}()
	if _, err = db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'웹훅 끊기')`, spaceID, ownerID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `
		INSERT INTO webhook_subscriptions(id,owner_id,name,url,secret_ciphertext,events)
		VALUES($1,$2,'끊을 웹훅','https://example.com/webhook',$3,ARRAY['note.created'])`,
		subscriptionID, ownerID, ciphertext); err != nil {
		t.Fatal(err)
	}

	service := New(db, cipher)
	queued, err := service.Enqueue(ctx, Event{
		ID: uuid.New(), Type: "note.created", SpaceID: spaceID, ActorID: ownerID,
		Data: map[string]any{"body": "보내지면 안 되는 생각"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if queued != 1 {
		t.Fatalf("queued %d deliveries, want 1; the rest of this test would prove nothing", queued)
	}

	// It really is about to be sent: this is the condition the dispatcher
	// selects on. Asked of this subscription rather than by claiming the next
	// delivery in the database, because the tests here share one and claiming
	// would take whichever was due first, whoever it belonged to.
	due := func() int {
		t.Helper()
		var n int
		if err = db.Pool.QueryRow(ctx, `
			SELECT count(*) FROM webhook_deliveries
			WHERE subscription_id=$1 AND status='queued' AND next_attempt_at<=now()`, subscriptionID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if due() != 1 {
		t.Fatalf("the queued delivery was not waiting to be sent: %d due", due())
	}

	// Exactly what the delete button runs.
	command, err := db.Pool.Exec(ctx, `DELETE FROM webhook_subscriptions WHERE id=$1 AND owner_id=$2`, subscriptionID, ownerID)
	if err != nil {
		t.Fatal(err)
	}
	if command.RowsAffected() != 1 {
		t.Fatalf("deleting the webhook affected %d rows, want 1", command.RowsAffected())
	}

	var left int
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM webhook_deliveries WHERE subscription_id=$1`, subscriptionID).Scan(&left); err != nil {
		t.Fatal(err)
	}
	if left != 0 {
		t.Errorf("%d deliveries are still queued for a webhook that was deleted", left)
	}

	// And nothing of theirs is waiting to be sent any more.
	if n := due(); n != 0 {
		t.Errorf("%d deliveries are still waiting to be sent for a deleted webhook", n)
	}

	// A later event does not queue for it either.
	queued, err = service.Enqueue(ctx, Event{
		ID: uuid.New(), Type: "note.created", SpaceID: spaceID, ActorID: ownerID,
		Data: map[string]any{"body": "그 뒤에 적은 생각"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if queued != 0 {
		t.Errorf("a deleted webhook was still queued %d deliveries", queued)
	}
}
