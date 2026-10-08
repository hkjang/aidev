package mail

import (
	"context"
	"database/sql"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/hkjang/invenqor/server/internal/apitime"
)

// ConfigLoader reads the current configuration. It is read on every
// notification so a change in the console applies without a restart and
// every Pod sees the same value.
type ConfigLoader func(context.Context) (Config, error)

// Directory resolves account identifiers to email addresses. The users table
// already knows everybody, so mail keeps no address book of its own.
type Directory interface {
	LookupEmails(context.Context, []string) (map[string]string, error)
}

// Delivery is one attempt to send one message. The body is deliberately not
// stored: subject and recipient answer "did it go out", and a stored body
// would make the delivery log itself a place secrets leak from.
type Delivery struct {
	ID           string       `json:"id"`
	Event        string       `json:"event"`
	Recipient    string       `json:"recipient"`
	Subject      string       `json:"subject"`
	ActorID      string       `json:"actor_id"`
	Status       string       `json:"status"`
	Attempts     int          `json:"attempts"`
	ErrorMessage string       `json:"error_message"`
	CreatedAt    apitime.Time `json:"created_at"`
	UpdatedAt    apitime.Time `json:"updated_at"`
}

type Summary struct {
	Total  int            `json:"total"`
	Status map[string]int `json:"status"`
}

type Page struct {
	Items   []Delivery `json:"items"`
	Summary Summary    `json:"summary"`
}

const (
	StatusQueued = "queued"
	StatusSent   = "sent"
	StatusFailed = "failed"

	// retention bounds the delivery log. Ninety days answers "did it arrive"
	// for as long as anybody asks.
	retention = 90 * 24 * time.Hour
	maxLimit  = 200
)

type Service struct {
	db        *sql.DB
	config    ConfigLoader
	directory Directory
	logger    *slog.Logger
	now       func() time.Time
	send      func(context.Context, Config, Message) error
	pending   sync.WaitGroup
	writes    atomic.Int64
}

func NewService(db *sql.DB, config ConfigLoader, directory Directory, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		db: db, config: config, directory: directory, logger: logger,
		now:  func() time.Time { return time.Now().UTC() },
		send: Deliver,
	}
}

// SetSender replaces the transport so tests can drive the service without a
// relay.
func (s *Service) SetSender(sender func(context.Context, Config, Message) error) {
	s.send = sender
}

// Wait blocks until every background delivery has been recorded. Tests use
// it; production never has to.
func (s *Service) Wait() { s.pending.Wait() }

func (s *Service) Config(ctx context.Context) (Config, error) {
	if s.config == nil {
		return Default(), nil
	}
	return s.config(ctx)
}

// Notify resolves the recipients and sends in the background so no request
// waits on a relay. The actor is dropped from the recipients: nobody is
// told about their own action. A configuration that is off, incomplete, or
// has the event switched off sends nothing, and only the incomplete case is
// logged because that one is a mistake the operator wants to hear about.
func (s *Service) Notify(ctx context.Context, notification Notification, actorID string, recipients []string) {
	config, err := s.Config(ctx)
	if err != nil {
		s.logger.Warn("mail_config_unavailable", "error", err.Error())
		return
	}
	if !config.Enabled || !config.Allows(notification.Event) {
		return
	}
	if err := config.Validate(); err != nil {
		s.logger.Warn("mail_config_incomplete", "event", notification.Event, "error", err.Error())
		return
	}
	addresses := s.resolve(ctx, recipients, actorID)
	if len(addresses) == 0 {
		return
	}
	body := notification.Render(config)
	for _, address := range addresses {
		delivery := s.record(ctx, notification, actorID, address)
		s.pending.Add(1)
		go func(delivery Delivery, message Message) {
			defer s.pending.Done()
			s.deliver(delivery, config, message)
		}(delivery, Message{To: address, Subject: notification.Subject, Body: body})
	}
}

// SendNow delivers immediately and reports the outcome, which is what the
// administrator's test button needs. It ignores the event switches but not
// mail.enabled: a test of a relay that is off would prove nothing.
func (s *Service) SendNow(ctx context.Context, notification Notification, actorID, recipient string) error {
	config, err := s.Config(ctx)
	if err != nil {
		return err
	}
	if !config.Enabled {
		return ErrDisabled
	}
	if err := config.Validate(); err != nil {
		return err
	}
	delivery := s.record(ctx, notification, actorID, recipient)
	delivery.Attempts = 1
	sendContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), config.Timeout+5*time.Second)
	defer cancel()
	err = s.send(sendContext, config, Message{To: recipient, Subject: notification.Subject, Body: notification.Render(config)})
	s.complete(sendContext, delivery, err)
	return err
}

// deliver retries once, because a relay that briefly refuses a connection is
// common and losing the notification is worse than a short wait. It runs
// detached from the request that caused it.
func (s *Service) deliver(delivery Delivery, config Config, message Message) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*config.Timeout+15*time.Second)
	defer cancel()
	var err error
	for attempt := 1; attempt <= 2; attempt++ {
		delivery.Attempts = attempt
		if err = s.send(ctx, config, message); err == nil {
			break
		}
		if attempt == 1 {
			select {
			case <-ctx.Done():
			case <-time.After(2 * time.Second):
			}
		}
	}
	s.complete(ctx, delivery, err)
}

func (s *Service) record(ctx context.Context, notification Notification, actorID, recipient string) Delivery {
	now := s.now()
	delivery := Delivery{
		ID: uuid.NewString(), Event: notification.Event, Recipient: recipient,
		Subject: trim(notification.Subject, 300), ActorID: actorID, Status: StatusQueued,
		CreatedAt: apitime.New(now), UpdatedAt: apitime.New(now),
	}
	if s.db == nil {
		return delivery
	}
	if _, err := s.db.ExecContext(context.WithoutCancel(ctx),
		`INSERT INTO mail_deliveries(
		 id,event,recipient,subject,actor_id,status,attempts,error_message,
		 created_at,updated_at
		 ) VALUES($1,$2,$3,$4,$5,$6,0,'',$7,$7)`,
		delivery.ID, delivery.Event, delivery.Recipient, delivery.Subject,
		delivery.ActorID, delivery.Status, now,
	); err != nil {
		s.logger.Warn("mail_delivery_not_recorded", "error", err.Error())
	}
	if s.writes.Add(1)%100 == 0 {
		s.prune(ctx)
	}
	return delivery
}

func (s *Service) complete(ctx context.Context, delivery Delivery, cause error) {
	status, message := StatusSent, ""
	if cause != nil {
		status, message = StatusFailed, cause.Error()
		s.logger.Warn("mail_delivery_failed",
			"event", delivery.Event, "recipient", delivery.Recipient,
			"attempts", delivery.Attempts, "error", message)
	}
	if s.db == nil {
		return
	}
	if _, err := s.db.ExecContext(context.WithoutCancel(ctx),
		`UPDATE mail_deliveries SET status=$2,attempts=$3,error_message=$4,
		 updated_at=$5 WHERE id=$1`,
		delivery.ID, status, max(delivery.Attempts, 1), trim(message, 1000), s.now(),
	); err != nil {
		s.logger.Warn("mail_delivery_status_not_recorded", "error", err.Error())
	}
}

func (s *Service) prune(ctx context.Context) {
	if _, err := s.db.ExecContext(context.WithoutCancel(ctx),
		`DELETE FROM mail_deliveries WHERE created_at < $1`,
		s.now().Add(-retention),
	); err != nil {
		s.logger.Warn("mail_deliveries_not_pruned", "error", err.Error())
	}
}

// resolve turns account identifiers into unique addresses, dropping the
// actor. An identifier that already is an address needs no directory entry.
func (s *Service) resolve(ctx context.Context, recipients []string, actorID string) []string {
	wanted := make([]string, 0, len(recipients))
	for _, recipient := range recipients {
		trimmed := strings.TrimSpace(recipient)
		if trimmed == "" || strings.EqualFold(trimmed, strings.TrimSpace(actorID)) {
			continue
		}
		wanted = append(wanted, trimmed)
	}
	if len(wanted) == 0 {
		return nil
	}
	emails := map[string]string{}
	if s.directory != nil {
		resolved, err := s.directory.LookupEmails(ctx, wanted)
		if err != nil {
			s.logger.Warn("mail_recipients_not_resolved", "error", err.Error())
			return nil
		}
		emails = resolved
	}
	seen := map[string]struct{}{}
	addresses := make([]string, 0, len(wanted))
	for _, recipient := range wanted {
		address := strings.TrimSpace(emails[recipient])
		if address == "" && isAddress(recipient) {
			address = recipient
		}
		if address == "" {
			continue
		}
		key := strings.ToLower(address)
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		addresses = append(addresses, address)
	}
	return addresses
}

// Deliveries lists what was sent, newest first, with a status breakdown of
// the whole log so the summary does not change with the filter.
func (s *Service) Deliveries(ctx context.Context, status string, limit int) (Page, error) {
	if limit < 1 || limit > maxLimit {
		limit = 50
	}
	page := Page{Items: []Delivery{}, Summary: Summary{Status: map[string]int{}}}
	if s.db == nil {
		return page, nil
	}
	query := `SELECT id,event,recipient,subject,actor_id,status,attempts,
	          error_message,created_at,updated_at FROM mail_deliveries`
	args := []any{}
	if trimmed := strings.TrimSpace(status); trimmed != "" {
		args = append(args, trimmed)
		query += ` WHERE status=$1`
	}
	args = append(args, limit)
	if len(args) == 1 {
		query += ` ORDER BY created_at DESC, id LIMIT $1`
	} else {
		query += ` ORDER BY created_at DESC, id LIMIT $2`
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return Page{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var item Delivery
		if err := rows.Scan(&item.ID, &item.Event, &item.Recipient, &item.Subject,
			&item.ActorID, &item.Status, &item.Attempts, &item.ErrorMessage,
			&item.CreatedAt, &item.UpdatedAt); err != nil {
			return Page{}, err
		}
		page.Items = append(page.Items, item)
	}
	if err := rows.Err(); err != nil {
		return Page{}, err
	}
	counts, err := s.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM mail_deliveries GROUP BY status`)
	if err != nil {
		return Page{}, err
	}
	defer counts.Close()
	for counts.Next() {
		var key string
		var count int
		if err := counts.Scan(&key, &count); err != nil {
			return Page{}, err
		}
		page.Summary.Status[key] = count
		page.Summary.Total += count
	}
	return page, counts.Err()
}

func trim(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
