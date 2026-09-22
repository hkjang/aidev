package server

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
)

// One listener per serving instance (not per editor). Rooms coalesce wakeups;
// dropped/duplicate notifications are harmless because PostgreSQL is canonical.
type collaborationRuntime struct {
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.Mutex
	rooms         map[string]map[chan struct{}]chan struct{}
	connected     atomic.Bool
	reconnects    atomic.Uint64
	notifications atomic.Uint64
	done          chan struct{}
	cacheMu       sync.Mutex
	cache         map[string]*collaborationCachedDocument
	cacheBytes    atomic.Int64
	cacheHits     atomic.Uint64
	cacheMisses   atomic.Uint64
	listenerPID   atomic.Int64
}

func (s *Server) installCollaborationNotifications(ctx context.Context) error {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, table := range []string{"documents", "document_collaboration", "collaboration_presence", "sessions", "users", "workspace_members", "document_shares", "spaces", "space_members", "settings", "workspace_settings", "protection_settings", "user_feature_flags"} {
		kind := "invalidate"
		if table == "documents" || table == "document_collaboration" || table == "collaboration_presence" {
			kind = "document"
		}
		name := pgx.Identifier{table}.Sanitize()
		events := "INSERT OR UPDATE OR DELETE"
		if table == "collaboration_presence" {
			// A heartbeat/quota-only write must not wake every peer, whose own
			// heartbeat would amplify that wake again. Actual cursor identity,
			// clock, epoch and state changes remain event-driven; liveness still
			// has the independent durable catchup timer and per-recipient checks.
			events = "INSERT OR DELETE OR UPDATE OF epoch,client_id,clock,state"
		}
		if _, err = tx.Exec(ctx, "DROP TRIGGER IF EXISTS madi_collaboration_signal ON "+name); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, "CREATE TRIGGER madi_collaboration_signal AFTER "+events+" ON "+name+" FOR EACH ROW EXECUTE FUNCTION madi_collaboration_notify('"+kind+"')"); err != nil {
			return err
		}
	}
	// Parent visibility or placement can alter descendants without changing them.
	_, err = tx.Exec(ctx, `DROP TRIGGER IF EXISTS madi_collaboration_tree_signal ON documents;
 CREATE TRIGGER madi_collaboration_tree_signal AFTER UPDATE OF visibility,parent_id,space_id,owner_id,deleted_at ON documents FOR EACH STATEMENT EXECUTE FUNCTION madi_collaboration_notify('invalidate')`)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Server) subscribeCollaboration(docID string) (*collaborationRuntime, <-chan struct{}, <-chan struct{}, func()) {
	s.collaborationMu.Lock()
	rt := s.collaboration
	if rt == nil {
		ctx, cancel := context.WithCancel(context.Background())
		rt = &collaborationRuntime{ctx: ctx, cancel: cancel, rooms: map[string]map[chan struct{}]chan struct{}{}, done: make(chan struct{})}
		s.collaboration = rt
		go s.runCollaborationListener(rt)
	}
	wake := make(chan struct{}, 1)
	authority := make(chan struct{}, 1)
	rt.mu.Lock()
	if rt.rooms[docID] == nil {
		rt.rooms[docID] = map[chan struct{}]chan struct{}{}
	}
	rt.rooms[docID][wake] = authority
	rt.mu.Unlock()
	s.collaborationMu.Unlock()
	var once sync.Once
	return rt, wake, authority, func() {
		once.Do(func() {
			s.collaborationMu.Lock()
			rt.mu.Lock()
			delete(rt.rooms[docID], wake)
			if len(rt.rooms[docID]) == 0 {
				delete(rt.rooms, docID)
			}
			empty := len(rt.rooms) == 0
			docEmpty := len(rt.rooms[docID]) == 0
			rt.mu.Unlock()
			if empty {
				rt.cancel()
				if s.collaboration == rt {
					s.collaboration = nil
				}
			}
			s.collaborationMu.Unlock()
			if docEmpty {
				rt.clearCollaborationCache(docID)
			}
		})
	}
}

func (rt *collaborationRuntime) wake(docID string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	for id, peers := range rt.rooms {
		if docID != "*" && id != docID {
			continue
		}
		for wake, authority := range peers {
			select {
			case wake <- struct{}{}:
			default:
			}
			select {
			case authority <- struct{}{}:
			default:
			}
		}
	}
}

func (s *Server) runCollaborationListener(rt *collaborationRuntime) {
	defer close(rt.done)
	defer rt.clearCollaborationCache("*")
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-rt.ctx.Done():
				return
			case <-ticker.C:
				rt.wake("*")
			}
		}
	}()
	// Independent bounded connection: an idle LISTEN must not consume a request
	// pool slot or make transactions awaiting document locks starve the pool.
	for rt.ctx.Err() == nil {
		cfg := s.DB.Config().ConnConfig.Copy()
		cfg.ConnectTimeout = 5 * time.Second
		conn, err := pgx.ConnectConfig(rt.ctx, cfg)
		if err == nil {
			rt.listenerPID.Store(int64(conn.PgConn().PID()))
			var channel string
			err = conn.QueryRow(rt.ctx, "SELECT 'madi_collab_'||md5(current_schema())").Scan(&channel)
			if err == nil {
				_, err = conn.Exec(rt.ctx, "LISTEN "+pgx.Identifier{channel}.Sanitize())
			}
			if err == nil {
				rt.connected.Store(true)
				rt.wake("*") // LISTEN committed first; then durable state catchup.
				for rt.ctx.Err() == nil {
					waitCtx, cancel := context.WithTimeout(rt.ctx, 5*time.Second)
					n, e := conn.WaitForNotification(waitCtx)
					timedOut := waitCtx.Err() != nil
					cancel()
					if e != nil {
						if rt.ctx.Err() != nil {
							break
						}
						if timedOut && !conn.IsClosed() {
							continue
						}
						break
					}
					if n.Payload == "*" || validID(n.Payload) {
						rt.notifications.Add(1)
						rt.wake(n.Payload)
					}
				}
			}
			closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			_ = conn.Close(closeCtx)
			cancel()
		}
		rt.connected.Store(false)
		rt.listenerPID.Store(0)
		rt.reconnects.Add(1)
		rt.wake("*")
		select {
		case <-rt.ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

// Called on graceful shutdown; all sockets observe rt.ctx cancellation. The
// last unsubscribe also stops its listener, so tests and idle instances release it.
func (s *Server) CloseCollaboration() {
	s.collaborationMu.Lock()
	rt := s.collaboration
	s.collaboration = nil
	if rt != nil {
		rt.cancel()
	}
	s.collaborationMu.Unlock()
	if rt != nil {
		select {
		case <-rt.done:
		case <-time.After(6 * time.Second):
		}
	}
}
