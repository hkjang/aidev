package store

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/intelligence"
)

// The point of moving indexing off the read path is that reading a space no
// longer depends on anybody else's server being up or quick. That is the thing
// worth testing, and it needs a gateway that is deliberately slow.

func sweepSpace(t *testing.T) (*Store, uuid.UUID, uuid.UUID) {
	t.Helper()
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	db := isolatedStore(t, dsn)
	ctx := context.Background()
	userID, spaceID := uuid.New(), uuid.New()
	name := "sweep_" + strings.ReplaceAll(userID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, name); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'색인 공간')`, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	return db, userID, spaceID
}

// A gateway that answers, eventually. This is the shape of the problem: not a
// server that is down, which a circuit breaker handles, but one that is merely
// slow — which nothing handled, because the read waited for it.
func slowGateway(t *testing.T, delay time.Duration, calls *atomic.Int64) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		time.Sleep(delay)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2,0.3]}]}`))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestReadingASpaceDoesNotWaitOnTheGatewayIntegration(t *testing.T) {
	db, userID, spaceID := sweepSpace(t)
	ctx := context.Background()

	var calls atomic.Int64
	gateway := slowGateway(t, 2*time.Second, &calls)
	if err := db.PutSetting(ctx, "ai_gateway", map[string]any{
		"base_url": gateway.URL, "embedding_model": "slow-embed", "embedding_timeout_seconds": 30,
	}, userID); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		if _, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "느린 게이트웨이 뒤의 생각"}); err != nil {
			t.Fatal(err)
		}
	}

	// Writing does not wait on it either, which is the same defect in the other
	// direction: typing a thought used to hold this connection open.
	writeStart := time.Now()
	if _, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "방금 쓴 생각"}); err != nil {
		t.Fatal(err)
	}
	if writing := time.Since(writeStart); writing > time.Second {
		t.Fatalf("writing a thought took %v — it is still waiting on the gateway", writing)
	}

	// Counted from here, so what the writes above did is not attributed to the
	// read below.
	before := calls.Load()
	start := time.Now()
	notes, _, err := db.ListNotes(ctx, userID, spaceID, "")
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 4 {
		t.Fatalf("%d notes", len(notes))
	}
	// The gateway takes two seconds per call. Before this change the read paid
	// that; now it must not pay any of it.
	if elapsed > time.Second {
		t.Fatalf("reading the space took %v — it is still waiting on the gateway", elapsed)
	}
	if n := calls.Load() - before; n != 0 {
		t.Fatalf("reading the space made %d gateway calls; a read must not index", n)
	}
}

// The work still happens, just not in front of anybody.
func TestTheSweepIndexesWhatTheReadNoLongerDoesIntegration(t *testing.T) {
	db, userID, spaceID := sweepSpace(t)
	ctx := context.Background()

	for i := 0; i < 4; i++ {
		if _, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "색인될 생각"}); err != nil {
			t.Fatal(err)
		}
	}
	var before int
	if err := db.Pool.QueryRow(ctx, `SELECT count(*) FROM note_embeddings e JOIN notes n ON n.id=e.note_id WHERE n.space_id=$1`, spaceID).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if before != 0 {
		t.Fatalf("%d embeddings existed before anything indexed them", before)
	}

	if done := db.SweepEmbeddingsOnce(ctx, &spaceID); done != 4 {
		t.Fatalf("the sweep handled %d thoughts, want 4", done)
	}
	var after int
	if err := db.Pool.QueryRow(ctx, `SELECT count(*) FROM note_embeddings e JOIN notes n ON n.id=e.note_id WHERE n.space_id=$1`, spaceID).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != 4 {
		t.Fatalf("%d embeddings after the sweep, want 4", after)
	}

	// And a second pass finds nothing, so the sweep is not re-doing settled
	// work every five seconds forever.
	if done := db.SweepEmbeddingsOnce(ctx, &spaceID); done != 0 {
		t.Fatalf("the sweep redid %d thoughts that were already current", done)
	}
}

// Changing a thought makes its vector stale, and the sweep has to notice.
func TestTheSweepPicksUpAChangedThoughtIntegration(t *testing.T) {
	db, userID, spaceID := sweepSpace(t)
	ctx := context.Background()

	note, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "처음 쓴 문장"})
	if err != nil {
		t.Fatal(err)
	}
	db.SweepEmbeddingsOnce(ctx, &spaceID)
	if done := db.SweepEmbeddingsOnce(ctx, &spaceID); done != 0 {
		t.Fatalf("not settled after the first sweep: %d", done)
	}

	note.Content = "고쳐 쓴 문장"
	if _, err := db.UpdateNote(ctx, userID, note, nil); err != nil {
		t.Fatal(err)
	}
	if done := db.SweepEmbeddingsOnce(ctx, &spaceID); done != 1 {
		t.Fatalf("the sweep handled %d thoughts after an edit, want 1", done)
	}
}

// A thought held back from analysis keeps its words off the gateway. This was
// true when indexing ran on the read, and moving it must not change it.
func TestTheSweepKeepsExcludedThoughtsOffTheGatewayIntegration(t *testing.T) {
	db, userID, spaceID := sweepSpace(t)
	ctx := context.Background()

	var calls atomic.Int64
	gateway := slowGateway(t, 0, &calls)
	if err := db.PutSetting(ctx, "ai_gateway", map[string]any{
		"base_url": gateway.URL, "embedding_model": "embed", "embedding_timeout_seconds": 5,
	}, userID); err != nil {
		t.Fatal(err)
	}

	if _, err := db.CreateNote(ctx, userID, Note{
		SpaceID: spaceID, AuthorID: userID, Content: "이 문장은 밖으로 나가면 안 됩니다", AIExcluded: true,
	}); err != nil {
		t.Fatal(err)
	}
	if done := db.SweepEmbeddingsOnce(ctx, &spaceID); done != 1 {
		t.Fatalf("the sweep skipped the excluded thought entirely: %d", done)
	}
	if n := calls.Load(); n != 0 {
		t.Fatalf("an excluded thought reached the gateway %d times", n)
	}
	// It still got a vector, locally — being held back from a gateway is not
	// being left out of search.
	var algorithm string
	if err := db.Pool.QueryRow(ctx, `
		SELECT e.algorithm FROM note_embeddings e JOIN notes n ON n.id=e.note_id WHERE n.space_id=$1`, spaceID).
		Scan(&algorithm); err != nil {
		t.Fatal(err)
	}
	if algorithm != intelligence.LocalAlgorithm {
		t.Fatalf("an excluded thought was stored under %q", algorithm)
	}

	// And an ordinary thought in the same space does reach the gateway, so the
	// check above is exclusion working rather than the gateway never being used.
	if _, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "평범한 생각"}); err != nil {
		t.Fatal(err)
	}
	db.SweepEmbeddingsOnce(ctx, &spaceID)
	if calls.Load() == 0 {
		t.Fatal("no thought reached the gateway at all, so exclusion proves nothing")
	}
}

// A thought held back does not drag the rest of its batch down with it.
//
// ensureEmbeddings resolves a whole batch to the local algorithm if any thought
// in it is excluded, which is the right call for that batch and the wrong one
// for everybody else's thoughts. So the sweep separates them before handing
// them over. Both are unindexed at the same moment here, which is the case that
// tells a split from no split — run them in separate passes and either shape
// passes.
func TestTheSweepDoesNotDowngradeASharedBatchIntegration(t *testing.T) {
	db, userID, spaceID := sweepSpace(t)
	ctx := context.Background()

	var calls atomic.Int64
	gateway := slowGateway(t, 0, &calls)
	if err := db.PutSetting(ctx, "ai_gateway", map[string]any{
		"base_url": gateway.URL, "embedding_model": "embed", "embedding_timeout_seconds": 5,
	}, userID); err != nil {
		t.Fatal(err)
	}

	held, err := db.CreateNote(ctx, userID, Note{
		SpaceID: spaceID, AuthorID: userID, Content: "밖으로 나가면 안 되는 생각", AIExcluded: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	shared, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "평범한 생각"})
	if err != nil {
		t.Fatal(err)
	}

	if done := db.SweepEmbeddingsOnce(ctx, &spaceID); done != 2 {
		t.Fatalf("the sweep handled %d thoughts, want both", done)
	}

	algorithms := map[uuid.UUID]string{}
	rows, err := db.Pool.Query(ctx, `
		SELECT e.note_id,e.algorithm FROM note_embeddings e JOIN notes n ON n.id=e.note_id WHERE n.space_id=$1`, spaceID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var algorithm string
		if err := rows.Scan(&id, &algorithm); err != nil {
			t.Fatal(err)
		}
		algorithms[id] = algorithm
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	if algorithms[held.ID] != intelligence.LocalAlgorithm {
		t.Errorf("the held-back thought was stored under %q", algorithms[held.ID])
	}
	if algorithms[shared.ID] == intelligence.LocalAlgorithm || algorithms[shared.ID] == "" {
		t.Errorf("an ordinary thought was dragged to the local algorithm by its neighbour: %q", algorithms[shared.ID])
	}
	if calls.Load() == 0 {
		t.Fatal("nothing reached the gateway at all")
	}
}

// Embedding waits on a different clock from the chat model. Sharing one number
// meant a timeout chosen to be generous to a Dream became the time somebody
// waited on a search.
func TestEmbeddingTimeoutIsItsOwnIntegration(t *testing.T) {
	db, userID, _ := sweepSpace(t)
	ctx := context.Background()

	if err := db.PutSetting(ctx, "ai_gateway", map[string]any{
		"base_url": "http://gateway.invalid", "embedding_model": "embed",
		"timeout_seconds": 300, "embedding_timeout_seconds": 7,
	}, userID); err != nil {
		t.Fatal(err)
	}
	provider := db.EmbeddingProvider(ctx)
	if provider.Remote == nil {
		t.Fatal("no remote provider was built")
	}
	if provider.Remote.Timeout != 7*time.Second {
		t.Fatalf("embedding timeout is %v; it is following the chat timeout instead of its own", provider.Remote.Timeout)
	}

	// Unset, it falls back to a short default rather than to the chat number.
	if err := db.PutSetting(ctx, "ai_gateway", map[string]any{
		"base_url": "http://gateway.invalid", "embedding_model": "embed", "timeout_seconds": 300,
	}, userID); err != nil {
		t.Fatal(err)
	}
	db.embeddings.loadedAt = time.Time{} // the settings cache would otherwise answer for the previous write
	provider = db.EmbeddingProvider(ctx)
	if provider.Remote == nil || provider.Remote.Timeout != defaultEmbeddingTimeout {
		t.Fatalf("with no embedding timeout set it used %v, want %v", provider.Remote.Timeout, defaultEmbeddingTimeout)
	}
}
