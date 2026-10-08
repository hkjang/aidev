// Package realtime turns the PostgreSQL collaboration log into a push stream.
//
// Before v0.8.0 every open Server-Sent Events connection polled space_events
// once per second, so idle collaborators produced a query per second each. The
// hub keeps a single dedicated connection in LISTEN mode and fans notifications
// out in process, which makes the database cost independent of how many people
// have a space open.
package realtime

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Channel is the PostgreSQL NOTIFY channel written by the space_events trigger
// installed in migration 007.
const Channel = "umm_space_events"

// Subscription receives a coalesced signal whenever the space it watches gains
// events. Signals carry no payload: the reader always re-queries from the last
// sequence it delivered, so a dropped or merged signal can never skip an event.
type Subscription struct {
	hub     *Hub
	spaceID uuid.UUID
	signal  chan struct{}
	closed  sync.Once
}

// C returns the notification channel. The channel is deliberately never
// closed: request cancellation owns the reader lifetime, while leaving the
// channel open removes the possibility of a concurrent publisher sending to a
// channel that Close has just closed.
func (s *Subscription) C() <-chan struct{} { return s.signal }

// Close releases the subscription. It is safe to call more than once.
func (s *Subscription) Close() {
	s.closed.Do(func() { s.hub.remove(s) })
}

// Hub owns the LISTEN connection and the subscriber registry.
type Hub struct {
	pool *pgxpool.Pool

	mu          sync.RWMutex
	subscribers map[uuid.UUID]map[*Subscription]struct{}

	listening atomic.Bool
	delivered atomic.Uint64
}

func New(pool *pgxpool.Pool) *Hub {
	return &Hub{pool: pool, subscribers: map[uuid.UUID]map[*Subscription]struct{}{}}
}

// Listening reports whether the LISTEN connection is currently healthy. Readers
// use it to decide between a slow safety-net poll and the fast fallback poll
// that keeps collaboration working while the connection is being re-established.
func (h *Hub) Listening() bool { return h.listening.Load() }

// Stats exposes counters for the operations dashboard.
func (h *Hub) Stats() (subscribers, spaces int, delivered uint64, listening bool) {
	h.mu.RLock()
	for _, set := range h.subscribers {
		subscribers += len(set)
	}
	spaces = len(h.subscribers)
	h.mu.RUnlock()
	return subscribers, spaces, h.delivered.Load(), h.listening.Load()
}

func (h *Hub) Subscribe(spaceID uuid.UUID) *Subscription {
	sub := &Subscription{hub: h, spaceID: spaceID, signal: make(chan struct{}, 1)}
	h.mu.Lock()
	set := h.subscribers[spaceID]
	if set == nil {
		set = map[*Subscription]struct{}{}
		h.subscribers[spaceID] = set
	}
	set[sub] = struct{}{}
	h.mu.Unlock()
	return sub
}

func (h *Hub) remove(sub *Subscription) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set := h.subscribers[sub.spaceID]; set != nil {
		delete(set, sub)
		if len(set) == 0 {
			delete(h.subscribers, sub.spaceID)
		}
	}
}

// Publish wakes every subscriber of a space. Sends are non-blocking, so a
// subscriber that has not drained its previous signal simply keeps the pending
// one; the reader collapses both into a single catch-up query.
func (h *Hub) Publish(spaceID uuid.UUID) {
	h.mu.RLock()
	set := h.subscribers[spaceID]
	targets := make([]*Subscription, 0, len(set))
	for sub := range set {
		targets = append(targets, sub)
	}
	h.mu.RUnlock()
	for _, sub := range targets {
		select {
		case sub.signal <- struct{}{}:
			h.delivered.Add(1)
		default:
		}
	}
}

// setListening records a listener transition and wakes every open stream. The
// wake-up has no event payload: each stream re-queries from its own cursor and
// also re-arms its safety ticker for the new listener state. Coalescing with an
// already-pending event is safe because that event causes the same catch-up
// query and interval refresh.
func (h *Hub) setListening(listening bool) {
	if previous := h.listening.Swap(listening); previous == listening {
		return
	}
	h.mu.RLock()
	targets := make([]*Subscription, 0)
	for _, set := range h.subscribers {
		for sub := range set {
			targets = append(targets, sub)
		}
	}
	h.mu.RUnlock()
	for _, sub := range targets {
		select {
		case sub.signal <- struct{}{}:
		default:
		}
	}
}

// Run holds one connection in LISTEN mode until the context is cancelled,
// reconnecting with backoff whenever PostgreSQL or the network drops it.
func (h *Hub) Run(ctx context.Context) {
	if maxConnections := h.pool.Config().MaxConns; maxConnections < 3 {
		slog.Warn("collaboration listener disabled because the pool cannot reserve two request connections", "pool_max_conns", maxConnections)
		return
	}
	backoff := time.Second
	for ctx.Err() == nil {
		if err := h.listen(ctx); err != nil && ctx.Err() == nil {
			slog.Warn("collaboration listener disconnected", "error", err, "retry_in", backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second
	}
}

func (h *Hub) listen(ctx context.Context) error {
	conn, err := h.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err = conn.Exec(ctx, "LISTEN "+Channel); err != nil {
		return err
	}
	h.setListening(true)
	defer h.setListening(false)
	slog.Info("collaboration listener ready", "channel", Channel)
	for {
		notification, waitErr := conn.Conn().WaitForNotification(ctx)
		if waitErr != nil {
			// A dead connection must be destroyed rather than returned to the
			// pool, otherwise the next acquirer inherits a broken socket.
			conn.Hijack().Close(context.Background())
			return waitErr
		}
		if spaceID, ok := parsePayload(notification.Payload); ok {
			h.Publish(spaceID)
		}
	}
}

// parsePayload reads the "<space uuid> <sequence>" payload written by the
// trigger. The sequence is informational: readers re-query by their own cursor.
func parsePayload(payload string) (uuid.UUID, bool) {
	raw, rest, _ := strings.Cut(strings.TrimSpace(payload), " ")
	spaceID, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, false
	}
	if rest != "" {
		if _, err := strconv.ParseInt(strings.TrimSpace(rest), 10, 64); err != nil {
			return uuid.Nil, false
		}
	}
	return spaceID, true
}
