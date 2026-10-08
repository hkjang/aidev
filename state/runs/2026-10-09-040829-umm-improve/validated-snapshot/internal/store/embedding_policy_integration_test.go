package store

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/intelligence"
)

type rejectingEmbeddingDecrypter struct{}

func (rejectingEmbeddingDecrypter) Decrypt(string) (string, error) {
	return "", errors.New("embedding credential cannot be decrypted")
}

func configureManagedEmbeddingGateway(t *testing.T, ctx context.Context, db *Store, baseURL, model string) {
	t.Helper()
	raw, err := json.Marshal(embeddingSettings{BaseURL: baseURL, EmbeddingModel: model, TimeoutSeconds: 5})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `UPDATE app_settings SET value=$1::jsonb,updated_at=clock_timestamp() WHERE key='ai_gateway'`, raw); err != nil {
		t.Fatal(err)
	}
	// Exercise the production lease path: a bounded direct connection with the
	// same schema/search_path, not a connection borrowed from the request pool.
	db.leaseConfig = db.Pool.Config().ConnConfig.Copy()
	db.aiLeaseSlots = make(chan struct{}, MaxAILeaseConnections)
	db.InvalidateEmbeddingProvider()
	provider := db.EmbeddingProvider(ctx)
	if provider.Remote == nil || !provider.Remote.SettingsManaged || provider.Algorithm() == intelligence.LocalAlgorithm {
		t.Fatal("managed embedding gateway was not loaded")
	}
}

func TestEmbeddingDispatchHoldsAIExclusionLeaseIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	for _, policy := range []string{"note", "space"} {
		t.Run(policy, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			db := isolatedStore(t, dsn)

			gatewayStarted := make(chan struct{})
			releaseGateway := make(chan struct{})
			var startOnce, releaseOnce sync.Once
			var gatewayCalls atomic.Int64
			gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gatewayCalls.Add(1)
				startOnce.Do(func() { close(gatewayStarted) })
				select {
				case <-releaseGateway:
				case <-r.Context().Done():
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"index": 0, "embedding": []float32{1, 0}}}})
			}))
			defer gateway.Close()
			defer releaseOnce.Do(func() { close(releaseGateway) })
			configureManagedEmbeddingGateway(t, ctx, db, gateway.URL, "policy-lease-model")

			userID, spaceID, noteID := uuid.New(), uuid.New(), uuid.New()
			username := "embedding_policy_" + strings.ReplaceAll(userID.String(), "-", "")
			if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, username); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'embedding policy lease')`, spaceID, userID); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Pool.Exec(ctx, `INSERT INTO notes(id,space_id,author_id,content) VALUES($1,$2,$3,'lease protected secret')`, noteID, spaceID, userID); err != nil {
				t.Fatal(err)
			}

			embedDone := make(chan error, 1)
			go func() { embedDone <- db.UpsertEmbedding(ctx, noteID, "lease protected secret", 1) }()
			select {
			case <-gatewayStarted:
			case <-ctx.Done():
				t.Fatal("remote embedding dispatch did not start")
			}

			updateStarted := make(chan struct{})
			updateDone := make(chan error, 1)
			go func() {
				close(updateStarted)
				if policy == "note" {
					_, err := db.Pool.Exec(ctx, `UPDATE notes SET ai_excluded=true WHERE id=$1`, noteID)
					updateDone <- err
					return
				}
				_, err := db.Pool.Exec(ctx, `UPDATE spaces SET ai_excluded=true WHERE id=$1`, spaceID)
				updateDone <- err
			}()
			<-updateStarted
			select {
			case err := <-updateDone:
				t.Fatalf("%s exclusion committed during remote dispatch: %v", policy, err)
			case <-time.After(250 * time.Millisecond):
			}

			releaseOnce.Do(func() { close(releaseGateway) })
			if err := <-embedDone; err != nil {
				t.Fatal(err)
			}
			if err := <-updateDone; err != nil {
				t.Fatal(err)
			}
			if err := db.UpsertEmbedding(ctx, noteID, "lease protected secret", 1); err != nil {
				t.Fatal(err)
			}
			if calls := gatewayCalls.Load(); calls != 1 {
				t.Fatalf("post-exclusion embedding reached the gateway: %d calls", calls)
			}
			var algorithm string
			if err := db.Pool.QueryRow(ctx, `SELECT algorithm FROM note_embeddings WHERE note_id=$1`, noteID).Scan(&algorithm); err != nil {
				t.Fatal(err)
			}
			if algorithm != intelligence.LocalAlgorithm {
				t.Fatalf("post-exclusion vector remained remote: %q", algorithm)
			}
		})
	}
}

func TestScopedSearchHoldsAuthorizationLeaseThroughQueryDispatchIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	for _, policy := range []string{"note exclusion", "space exclusion", "membership removal"} {
		t.Run(policy, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			db := isolatedStore(t, dsn)

			gatewayStarted := make(chan struct{})
			releaseGateway := make(chan struct{})
			var startOnce, releaseOnce sync.Once
			var gatewayCalls atomic.Int64
			gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gatewayCalls.Add(1)
				startOnce.Do(func() { close(gatewayStarted) })
				select {
				case <-releaseGateway:
				case <-r.Context().Done():
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"index": 0, "embedding": []float32{1, 0}}}})
			}))
			defer gateway.Close()
			defer releaseOnce.Do(func() { close(releaseGateway) })
			configureManagedEmbeddingGateway(t, ctx, db, gateway.URL, "search-policy-model")

			ownerID, userID, spaceID, noteID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
			ownerName := "search_owner_" + strings.ReplaceAll(ownerID.String(), "-", "")
			userName := "search_member_" + strings.ReplaceAll(userID.String(), "-", "")
			if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text),($3,$4::citext,$4::text)`, ownerID, ownerName, userID, userName); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'search policy lease')`, spaceID, ownerID); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Pool.Exec(ctx, `INSERT INTO space_members(space_id,user_id,permission) VALUES($1,$2,'view')`, spaceID, userID); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Pool.Exec(ctx, `INSERT INTO notes(id,space_id,author_id,content) VALUES($1,$2,$3,'scoped search secret')`, noteID, spaceID, ownerID); err != nil {
				t.Fatal(err)
			}

			searchDone := make(chan error, 1)
			go func() {
				_, err := db.SearchNotesHybrid(ctx, userID, SearchOptions{Query: "scoped search secret", SpaceID: &spaceID, Limit: 5})
				searchDone <- err
			}()
			select {
			case <-gatewayStarted:
			case <-ctx.Done():
				t.Fatal("remote query embedding did not start")
			}
			updateStarted := make(chan struct{})
			updateDone := make(chan error, 1)
			go func() {
				close(updateStarted)
				if policy == "note exclusion" {
					_, err := db.Pool.Exec(ctx, `UPDATE notes SET ai_excluded=true WHERE id=$1`, noteID)
					updateDone <- err
					return
				}
				if policy == "space exclusion" {
					_, err := db.Pool.Exec(ctx, `UPDATE spaces SET ai_excluded=true WHERE id=$1`, spaceID)
					updateDone <- err
					return
				}
				_, err := db.Pool.Exec(ctx, `DELETE FROM space_members WHERE space_id=$1 AND user_id=$2`, spaceID, userID)
				updateDone <- err
			}()
			<-updateStarted
			select {
			case err := <-updateDone:
				t.Fatalf("%s committed while scoped query text was in flight: %v", policy, err)
			case <-time.After(250 * time.Millisecond):
			}

			releaseOnce.Do(func() { close(releaseGateway) })
			if err := <-searchDone; err != nil {
				t.Fatal(err)
			}
			if err := <-updateDone; err != nil {
				t.Fatal(err)
			}
			if _, err := db.SearchNotesHybrid(ctx, userID, SearchOptions{Query: "scoped search secret", SpaceID: &spaceID, Limit: 5}); err != nil {
				t.Fatal(err)
			}
			if calls := gatewayCalls.Load(); calls != 1 {
				t.Fatalf("post-policy scoped query reached the gateway: %d calls", calls)
			}
		})
	}
}

func TestUnreadableEmbeddingCredentialNeverLeavesProcessIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedStore(t, dsn)
	db.Cipher = rejectingEmbeddingDecrypter{}

	var gatewayCalls atomic.Int64
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		gatewayCalls.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"index": 0, "embedding": []float32{1, 0}}}})
	}))
	defer gateway.Close()
	raw, err := json.Marshal(embeddingSettings{
		BaseURL: gateway.URL, APIKey: "enc:corrupted-ciphertext",
		EmbeddingModel: "unreadable-key-model", TimeoutSeconds: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `UPDATE app_settings SET value=$1::jsonb,updated_at=clock_timestamp() WHERE key='ai_gateway'`, raw); err != nil {
		t.Fatal(err)
	}
	db.InvalidateEmbeddingProvider()
	if provider := db.EmbeddingProvider(ctx); provider.Algorithm() != intelligence.LocalAlgorithm {
		t.Fatalf("unreadable credential produced a remote provider: %q", provider.Algorithm())
	}

	userID, spaceID, noteID := uuid.New(), uuid.New(), uuid.New()
	username := "embedding_decrypt_" + strings.ReplaceAll(userID.String(), "-", "")
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, username); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'unreadable embedding key')`, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO notes(id,space_id,author_id,content) VALUES($1,$2,$3,'must remain local')`, noteID, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	if err = db.UpsertEmbedding(ctx, noteID, "must remain local", 1); err != nil {
		t.Fatal(err)
	}
	if calls := gatewayCalls.Load(); calls != 0 {
		t.Fatalf("unreadable ciphertext reached the embedding gateway: %d calls", calls)
	}
	var algorithm string
	if err = db.Pool.QueryRow(ctx, `SELECT algorithm FROM note_embeddings WHERE note_id=$1`, noteID).Scan(&algorithm); err != nil {
		t.Fatal(err)
	}
	if algorithm != intelligence.LocalAlgorithm {
		t.Fatalf("unreadable credential persisted a remote vector: %q", algorithm)
	}
}
