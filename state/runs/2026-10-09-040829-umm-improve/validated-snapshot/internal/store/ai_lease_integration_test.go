package store

import (
	"context"
	"errors"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestLongLeaseCapacityDoesNotConsumeRequestPoolIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("pool_max_conns", "1")
	parsed.RawQuery = query.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := Open(ctx, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Close()

	tests := []struct {
		name     string
		capacity int
		begin    func(context.Context) (pgx.Tx, func(), error)
	}{
		{name: "AI", capacity: MaxAILeaseConnections, begin: db.BeginAILease},
		{name: "webhook", capacity: MaxWebhookLeaseConnections, begin: db.BeginWebhookLease},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			type activeLease struct {
				tx      pgx.Tx
				release func()
			}
			leases := make([]activeLease, 0, test.capacity)
			defer func() {
				for _, lease := range leases {
					_ = lease.tx.Rollback(context.Background())
					lease.release()
				}
			}()
			for range test.capacity {
				tx, release, beginErr := test.begin(ctx)
				if beginErr != nil {
					t.Fatal(beginErr)
				}
				leases = append(leases, activeLease{tx: tx, release: release})
			}

			requestCtx, requestCancel := context.WithTimeout(ctx, 500*time.Millisecond)
			err = db.Pool.Ping(requestCtx)
			requestCancel()
			if err != nil {
				t.Fatalf("%d active %s leases consumed the one-connection request pool: %v", test.capacity, test.name, err)
			}
			if acquired := db.Pool.Stat().AcquiredConns(); acquired != 0 {
				t.Fatalf("request-pool connections held by %s leases = %d, want 0", test.name, acquired)
			}

			blockedCtx, blockedCancel := context.WithTimeout(ctx, 150*time.Millisecond)
			_, _, err = test.begin(blockedCtx)
			blockedCancel()
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("extra %s lease exceeded bounded capacity: %v", test.name, err)
			}

			if err = leases[0].tx.Rollback(ctx); err != nil {
				t.Fatal(err)
			}
			leases[0].release()
			resumed, releaseResumed, beginErr := test.begin(ctx)
			if beginErr != nil {
				t.Fatalf("%s lease capacity did not resume after release: %v", test.name, beginErr)
			}
			if err = resumed.Rollback(ctx); err != nil {
				t.Fatal(err)
			}
			releaseResumed()
		})
	}
}
