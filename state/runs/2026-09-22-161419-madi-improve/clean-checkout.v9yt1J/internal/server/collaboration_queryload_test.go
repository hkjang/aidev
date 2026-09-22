package server

import (
	"context"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type collaborationQueryTrace struct {
	mu      sync.Mutex
	counts  map[string]int
	enabled bool
}

func (trace *collaborationQueryTrace) TraceQueryStart(ctx context.Context, _ *pgx.Conn, event pgx.TraceQueryStartData) context.Context {
	category := "other"
	switch {
	case strings.Contains(event.SQL, "JOIN sessions t ON t.user_id=u.id"):
		category = "current_actor_document_acl"
	case strings.Contains(event.SQL, "madi_feature_allowed"):
		category = "current_feature_policy"
	case strings.Contains(event.SQL, "data->>'mode' IN ('block','mask')"):
		category = "current_protection_policy"
	case strings.Contains(event.SQL, "SELECT c.epoch<>$2"):
		category = "durable_sequence_check"
	case strings.Contains(event.SQL, "FROM collaboration_presence p JOIN users"):
		category = "presence_read"
	case strings.HasPrefix(event.SQL, "UPDATE collaboration_presence"):
		category = "presence_write"
	case strings.HasPrefix(event.SQL, "LISTEN "):
		category = "listen"
	}
	trace.mu.Lock()
	if trace.enabled {
		trace.counts[category]++
	}
	trace.mu.Unlock()
	return ctx
}
func (*collaborationQueryTrace) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}
func (trace *collaborationQueryTrace) begin() {
	trace.mu.Lock()
	trace.counts = map[string]int{}
	trace.enabled = true
	trace.mu.Unlock()
}
func (trace *collaborationQueryTrace) end() map[string]int {
	trace.mu.Lock()
	defer trace.mu.Unlock()
	trace.enabled = false
	return trace.counts
}

// Query counts are observations, not a synthetic 2,000-user capacity claim.
// These native protocol clients deliberately send no awareness/typing, isolating
// the connection fallback and recipient authorization cost from user activity.
func TestPostgresCollaborationConnectionQueryLoad(t *testing.T) {
	if os.Getenv("MADI_COLLABORATION_QUERY_LOAD") != "1" {
		t.Skip("opt-in three 12 second connection/query measurements")
	}
	s, admin, _, wid, _ := collaborationTestSetup(t)
	trace := &collaborationQueryTrace{}
	cfg := s.DB.Config()
	cfg.ConnConfig.Tracer = trace
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	app, err := New(t.Context(), pool, s.EncryptionKey, "query-load", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(app)
	defer server.Close()
	defer app.CloseCollaboration()
	client := newIntegrationTestClient(t, server.URL)
	client.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	ids := make([]string, 8)
	for i := range ids {
		ids[i] = str(testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "연결 수 검증", "markdown": "bounded fixture"}, 200)), "id")
	}
	measurements := []map[string]any{}
	for _, phase := range []struct {
		name     string
		count    int
		distinct bool
	}{{"one_connection_one_document", 1, false}, {"eight_connections_one_document", 8, false}, {"eight_connections_eight_documents", 8, true}} {
		baseGoroutines := runtime.NumGoroutine()
		connections := make([]*websocket.Conn, 0, phase.count)
		var readers sync.WaitGroup
		for index := 0; index < phase.count; index++ {
			id := ids[0]
			if phase.distinct {
				id = ids[index]
			}
			conn, _, err := collaborationTestDial(t, client, id, server.URL)
			if err != nil {
				t.Fatal(err)
			}
			collaborationTestRead(t, conn, "hello")
			connections = append(connections, conn)
			readers.Go(func() {
				for {
					var message collaborationMessage
					if wsjson.Read(t.Context(), conn, &message) != nil {
						return
					}
				}
			})
		}
		app.collaborationMu.Lock()
		rt := app.collaboration
		app.collaborationMu.Unlock()
		collaborationWait(t, "listener ready", func() bool { return rt.connected.Load() })
		select {
		case <-time.After(2 * time.Second):
		case <-t.Context().Done():
			t.Fatal(t.Context().Err())
		}
		trace.begin()
		started := time.Now()
		select {
		case <-time.After(12 * time.Second):
		case <-t.Context().Done():
			t.Fatal(t.Context().Err())
		}
		elapsed := time.Since(started).Seconds()
		counts := trace.end()
		var memory runtime.MemStats
		runtime.ReadMemStats(&memory)
		rt.mu.Lock()
		roomCount := len(rt.rooms)
		peerCount := 0
		for _, room := range rt.rooms {
			peerCount += len(room)
		}
		rt.mu.Unlock()
		if peerCount != phase.count || rt.listenerPID.Load() == 0 {
			t.Fatal("measurement lacks actual live connections/listener")
		}
		if counts["current_actor_document_acl"] < phase.count*8 {
			t.Fatal("current authorization fallback unexpectedly absent", counts)
		}
		for _, conn := range connections {
			_ = conn.CloseNow()
		}
		readers.Wait()
		select {
		case <-rt.done:
		case <-time.After(8 * time.Second):
			t.Fatal("listener not cleaned after last client")
		}
		collaborationWait(t, "presence cleanup", func() bool {
			var n int
			err := s.DB.QueryRow(t.Context(), "SELECT count(*) FROM collaboration_presence").Scan(&n)
			return err == nil && n == 0
		})
		collaborationWait(t, "pool requests released", func() bool { return pool.Stat().AcquiredConns() == 0 })
		if rt.cacheBytes.Load() != 0 || rt.listenerPID.Load() != 0 {
			t.Fatal("native cache or listener retained after unsubscribe")
		}
		total := 0
		for _, n := range counts {
			total += n
		}
		measurement := map[string]any{"phase": phase.name, "connections": phase.count, "rooms": roomCount, "seconds": elapsed, "queries": counts, "total_queries": total, "queries_per_second": float64(total) / elapsed, "go_heap_alloc_bytes": memory.HeapAlloc, "go_heap_inuse_bytes": memory.HeapInuse, "goroutines_before": baseGoroutines, "goroutines_after": runtime.NumGoroutine(), "listener_and_cache_released": true, "presence_rows_after": 0, "scope": "idle native WebSocket clients, no awareness or typing; one process; not capacity/SLA"}
		measurements = append(measurements, measurement)
		t.Logf("%s: %.2fs %d queries (%.2f/s) %v", phase.name, elapsed, total, float64(total)/elapsed, counts)
	}
	collaborationWriteEvidence(t, "connection-queries.json", map[string]any{"recorded_at": time.Now().UTC().Format(time.RFC3339), "measurements": measurements})
}
