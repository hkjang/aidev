package realtime

import (
	"context"
	"net/url"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPublishWakesOnlyMatchingSubscribers(t *testing.T) {
	hub := New(nil)
	watched, other := uuid.New(), uuid.New()
	subscription := hub.Subscribe(watched)
	defer subscription.Close()
	bystander := hub.Subscribe(other)
	defer bystander.Close()

	hub.Publish(watched)
	select {
	case <-subscription.C():
	case <-time.After(time.Second):
		t.Fatal("subscriber of the published space was not woken")
	}
	select {
	case <-bystander.C():
		t.Fatal("subscriber of an unrelated space was woken")
	default:
	}
}

func TestPublishCoalescesWithoutBlocking(t *testing.T) {
	hub := New(nil)
	spaceID := uuid.New()
	subscription := hub.Subscribe(spaceID)
	defer subscription.Close()

	// A reader that is busy must never stall a writer: extra signals collapse
	// into the single pending one the reader will act on.
	for range 100 {
		hub.Publish(spaceID)
	}
	if got := len(subscription.C()); got != 1 {
		t.Fatalf("expected one coalesced signal, got %d", got)
	}
}

func TestListeningTransitionsWakeEverySubscriber(t *testing.T) {
	hub := New(nil)
	first := hub.Subscribe(uuid.New())
	defer first.Close()
	second := hub.Subscribe(uuid.New())
	defer second.Close()

	assertWoken := func(name string, subscription *Subscription) {
		t.Helper()
		select {
		case <-subscription.C():
		case <-time.After(time.Second):
			t.Fatalf("%s was not woken by the listener transition", name)
		}
	}

	hub.setListening(true)
	if !hub.Listening() {
		t.Fatal("listener did not become healthy")
	}
	assertWoken("first subscriber", first)
	assertWoken("second subscriber", second)

	// Storing the same state must not cause unnecessary catch-up queries.
	hub.setListening(true)
	select {
	case <-first.C():
		t.Fatal("unchanged listener state woke a subscriber")
	default:
	}

	// A pending event may absorb the state signal. The stream still wakes once
	// and observes the current unhealthy state when it resets its poll timer.
	hub.Publish(first.spaceID)
	hub.setListening(false)
	if hub.Listening() {
		t.Fatal("listener did not become unavailable")
	}
	assertWoken("first subscriber after disconnect", first)
	assertWoken("second subscriber after disconnect", second)
	select {
	case <-first.C():
		t.Fatal("coalesced event and state transition produced two wake-ups")
	default:
	}
	if _, _, delivered, _ := hub.Stats(); delivered != 1 {
		t.Fatalf("listener-state wake-ups changed the event delivery counter: %d", delivered)
	}
}

func TestCloseIsIdempotentAndDeregisters(t *testing.T) {
	hub := New(nil)
	spaceID := uuid.New()
	subscription := hub.Subscribe(spaceID)
	subscription.Close()
	subscription.Close()

	if subscribers, spaces, _, _ := hub.Stats(); subscribers != 0 || spaces != 0 {
		t.Fatalf("expected an empty registry, got %d subscribers across %d spaces", subscribers, spaces)
	}
	// Publishing to a space nobody watches must remain a no-op.
	hub.Publish(spaceID)
	select {
	case _, open := <-subscription.C():
		if !open {
			t.Fatal("Close must not close the signal channel while a publisher may still hold a reference")
		}
	default:
	}
}

func TestPublishAndCloseCanRaceSafely(t *testing.T) {
	hub := New(nil)
	spaceID := uuid.New()
	for range 1000 {
		subscription := hub.Subscribe(spaceID)
		start := make(chan struct{})
		var workers sync.WaitGroup
		workers.Add(2)
		go func() {
			defer workers.Done()
			<-start
			hub.Publish(spaceID)
		}()
		go func() {
			defer workers.Done()
			<-start
			subscription.Close()
		}()
		close(start)
		workers.Wait()
	}
}

func TestParsePayload(t *testing.T) {
	spaceID := uuid.New()
	if got, ok := parsePayload(spaceID.String() + " 42"); !ok || got != spaceID {
		t.Fatalf("expected %s, got %s (ok=%v)", spaceID, got, ok)
	}
	if got, ok := parsePayload(spaceID.String()); !ok || got != spaceID {
		t.Fatalf("payload without a sequence should still parse, got %s (ok=%v)", got, ok)
	}
	for _, payload := range []string{"", "not-a-uuid 1", spaceID.String() + " abc"} {
		if _, ok := parsePayload(payload); ok {
			t.Fatalf("expected %q to be rejected", payload)
		}
	}
}

func TestRunReservesTwoRequestConnectionsIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	for _, maximum := range []int{1, 2} {
		t.Run(strconv.Itoa(maximum), func(t *testing.T) {
			pool := poolWithMaxConnections(t, dsn, maximum)
			hub := New(pool)
			runCtx, runCancel := context.WithCancel(context.Background())
			defer runCancel()
			done := make(chan struct{})
			go func() {
				hub.Run(runCtx)
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(500 * time.Millisecond):
				runCancel()
				select {
				case <-done:
				case <-time.After(2 * time.Second):
				}
				t.Fatalf("LISTEN hub occupied a pool with only %d request connections", maximum)
			}
			if hub.Listening() {
				t.Fatal("small-pool fallback must report the listener as unavailable")
			}
			acquireCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			connections := make([]*pgxpool.Conn, 0, maximum)
			for range maximum {
				connection, err := pool.Acquire(acquireCtx)
				if err != nil {
					t.Fatalf("request connection was unavailable: acquired=%d/%d err=%v", len(connections), maximum, err)
				}
				connections = append(connections, connection)
			}
			for _, connection := range connections {
				connection.Release()
			}
		})
	}

	t.Run("three starts listener", func(t *testing.T) {
		pool := poolWithMaxConnections(t, dsn, 3)
		hub := New(pool)
		runCtx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			hub.Run(runCtx)
			close(done)
		}()
		deadline := time.Now().Add(2 * time.Second)
		for !hub.Listening() && time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond)
		}
		if !hub.Listening() {
			cancel()
			<-done
			t.Fatal("three-connection pool did not start the collaboration listener")
		}
		acquireCtx, acquireCancel := context.WithTimeout(context.Background(), time.Second)
		first, err := pool.Acquire(acquireCtx)
		if err != nil {
			acquireCancel()
			cancel()
			<-done
			t.Fatal(err)
		}
		second, err := pool.Acquire(acquireCtx)
		acquireCancel()
		if err != nil {
			first.Release()
			cancel()
			<-done
			t.Fatalf("listener did not leave two request connections: %v", err)
		}
		second.Release()
		first.Release()

		// Kill the dedicated backend after a healthy stream has subscribed.
		// The disconnect and the subsequent reconnect must each wake it so the
		// HTTP layer can immediately re-arm its fallback ticker.
		subscription := hub.Subscribe(uuid.New())
		defer subscription.Close()
		applicationName := pool.Config().ConnConfig.RuntimeParams["application_name"]
		var listenerPID int
		lookupCtx, lookupCancel := context.WithTimeout(context.Background(), time.Second)
		err = pool.QueryRow(lookupCtx, `
			SELECT pid
			FROM pg_stat_activity
			WHERE application_name=$1 AND query=$2 AND pid<>pg_backend_pid()
			ORDER BY backend_start DESC
			LIMIT 1`, applicationName, "LISTEN "+Channel).Scan(&listenerPID)
		lookupCancel()
		if err != nil {
			cancel()
			<-done
			t.Fatalf("listener backend was not visible: %v", err)
		}
		terminateCtx, terminateCancel := context.WithTimeout(context.Background(), time.Second)
		var terminated bool
		err = pool.QueryRow(terminateCtx, `SELECT pg_terminate_backend($1)`, listenerPID).Scan(&terminated)
		terminateCancel()
		if err != nil || !terminated {
			cancel()
			<-done
			t.Fatalf("listener backend was not terminated: terminated=%v err=%v", terminated, err)
		}
		select {
		case <-subscription.C():
		case <-time.After(time.Second):
			cancel()
			<-done
			t.Fatal("listener disconnect did not wake the subscriber")
		}
		if hub.Listening() {
			cancel()
			<-done
			t.Fatal("terminated listener still reported healthy after its disconnect signal")
		}
		reconnectDeadline := time.Now().Add(3 * time.Second)
		for !hub.Listening() && time.Now().Before(reconnectDeadline) {
			time.Sleep(10 * time.Millisecond)
		}
		if !hub.Listening() {
			cancel()
			<-done
			t.Fatal("listener did not reconnect after its backend was terminated")
		}
		select {
		case <-subscription.C():
		case <-time.After(time.Second):
			cancel()
			<-done
			t.Fatal("listener recovery did not wake the subscriber")
		}

		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("LISTEN hub did not stop after cancellation")
		}
	})
}

func poolWithMaxConnections(t *testing.T, dsn string, maximum int) *pgxpool.Pool {
	t.Helper()
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("pool_max_conns", strconv.Itoa(maximum))
	query.Set("application_name", "umm_realtime_test_"+uuid.NewString())
	parsed.RawQuery = query.Encode()
	pool, err := pgxpool.New(context.Background(), parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if pool.Config().MaxConns != int32(maximum) {
		t.Fatalf("pool maximum = %d, want %d", pool.Config().MaxConns, maximum)
	}
	return pool
}
