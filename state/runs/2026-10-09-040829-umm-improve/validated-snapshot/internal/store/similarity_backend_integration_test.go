package store

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/intelligence"
)

// The features a person actually sees — related thoughts and clusters — used
// fixed cosine cutoffs that were only ever right for the offline character
// n-gram algorithm. This exercises them against a real sentence embedding model,
// where every pair scores high, and checks they still discriminate instead of
// returning the entire workspace.
//
// Skipped unless a gateway is configured, because it needs a second embedding
// backend to be a test of anything:
//
//	UMM_EMBEDDING_TEST_URL=http://127.0.0.1:11434 \
//	UMM_EMBEDDING_TEST_MODEL=bge-m3 \
//	POSTGRES_DSN=... go test ./internal/store -run SentenceEmbedding -v
func TestClustersAndRelatedSurviveASentenceEmbeddingBackendIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	baseURL := strings.TrimSpace(os.Getenv("UMM_EMBEDDING_TEST_URL"))
	model := strings.TrimSpace(os.Getenv("UMM_EMBEDDING_TEST_MODEL"))
	if baseURL == "" || model == "" {
		t.Skip("set UMM_EMBEDDING_TEST_URL and UMM_EMBEDDING_TEST_MODEL to exercise a real embedding backend")
	}
	ctx := context.Background()
	probe := intelligence.Provider{Remote: &intelligence.RemoteConfig{BaseURL: baseURL, Model: model}}
	if _, err := probe.EmbedStrict(ctx, []string{"probe"}); err != nil {
		t.Skipf("gateway unreachable: %v", err)
	}

	db, err := Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Close()

	ownerID, spaceID := uuid.New(), uuid.New()
	ownerName := "similarity_owner_" + ownerID.String()
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, ownerID, ownerName); err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, ownerID)
	if _, err = db.Pool.Exec(ctx, `INSERT INTO user_preferences(user_id) VALUES($1)`, ownerID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'similarity backend')`, spaceID, ownerID); err != nil {
		t.Fatal(err)
	}

	// Two clearly distinct topics. A backend-aware threshold keeps them apart;
	// a constant tuned for the offline algorithm merges them.
	security := []string{
		"인증 토큰 만료 시간을 24시간으로 정했다",
		"세션 쿠키는 HttpOnly와 SameSite를 함께 설정한다",
		"로그인 실패가 반복되면 계정을 일시적으로 잠근다",
	}
	cycling := []string{
		"주말에 자전거를 타고 한강을 따라 달렸다",
		"자전거 체인에 기름을 새로 발랐다",
		"라이딩이 끝나면 스트레칭을 꼭 한다",
	}
	cyclingContent := map[string]bool{}
	for _, content := range cycling {
		cyclingContent[content] = true
	}
	var firstSecurityNote uuid.UUID
	for index, content := range append(append([]string{}, security...), cycling...) {
		noteID := uuid.New()
		if _, err = db.Pool.Exec(ctx, `INSERT INTO notes(id,space_id,author_id,content) VALUES($1,$2,$3,$4)`, noteID, spaceID, ownerID, content); err != nil {
			t.Fatal(err)
		}
		if index == 0 {
			firstSecurityNote = noteID
		}
	}

	restore := useGatewayEmbedding(t, db, ownerID, baseURL, model)
	defer restore()

	notes, _, err := db.ListNotes(ctx, ownerID, spaceID, "")
	if err != nil {
		t.Fatalf("list notes: %v", err)
	}
	if len(notes) != len(security)+len(cycling) {
		t.Fatalf("expected %d notes, got %d", len(security)+len(cycling), len(notes))
	}
	assertVectorsCameFromTheGateway(t, db, spaceID, model)

	clusters, err := db.Clusters(ctx, ownerID, spaceID)
	if err != nil {
		t.Fatalf("clusters: %v", err)
	}
	for _, cluster := range clusters {
		if len(cluster.NoteIDs) == len(notes) {
			t.Fatalf("all %d notes collapsed into one cluster; the cutoff is not backend aware", len(notes))
		}
	}
	sizes := make([]int, 0, len(clusters))
	for _, cluster := range clusters {
		sizes = append(sizes, len(cluster.NoteIDs))
	}
	t.Logf("clusters under %s: %d groups, sizes %v", model, len(clusters), sizes)

	related, err := db.RelatedNotes(ctx, ownerID, firstSecurityNote, 20)
	if err != nil {
		t.Fatalf("related notes: %v", err)
	}
	if len(related) == len(notes)-1 {
		t.Fatalf("every other note came back as related; the cutoff is not backend aware")
	}
	for _, item := range related {
		if cyclingContent[item.Note.Content] {
			t.Errorf("an unrelated-topic note was returned as related: %q (score %.3f)", item.Note.Content, item.Score)
		}
	}
	t.Logf("related to the first security note: %d of %d others", len(related), len(notes)-1)
}

// useGatewayEmbedding switches the store onto a gateway backend and returns a
// function that puts the previous configuration back. app_settings is global, so
// leaving it changed would alter every test that runs afterwards.
func useGatewayEmbedding(t *testing.T, db *Store, actor uuid.UUID, baseURL, model string) func() {
	t.Helper()
	ctx := context.Background()
	var previous map[string]any
	if err := db.GetSetting(ctx, "ai_gateway", &previous); err != nil {
		t.Fatalf("read ai_gateway settings: %v", err)
	}
	updated := map[string]any{}
	for key, value := range previous {
		updated[key] = value
	}
	updated["base_url"] = baseURL
	updated["embedding_model"] = model
	if err := db.PutSetting(ctx, "ai_gateway", updated, actor); err != nil {
		t.Fatalf("configure gateway embedding: %v", err)
	}
	db.InvalidateEmbeddingProvider()
	return func() {
		if err := db.PutSetting(context.Background(), "ai_gateway", previous, actor); err != nil {
			t.Errorf("restore ai_gateway settings: %v", err)
		}
		db.InvalidateEmbeddingProvider()
	}
}

// assertVectorsCameFromTheGateway makes the test honest: without this it would
// still pass while silently measuring the offline algorithm after a gateway
// failure fell back to it.
func assertVectorsCameFromTheGateway(t *testing.T, db *Store, spaceID uuid.UUID, model string) {
	t.Helper()
	rows, err := db.Pool.Query(context.Background(), `
		SELECT DISTINCT e.algorithm
		FROM note_embeddings e JOIN notes n ON n.id=e.note_id
		WHERE n.space_id=$1`, spaceID)
	if err != nil {
		t.Fatalf("read embedding algorithms: %v", err)
	}
	defer rows.Close()
	algorithms := []string{}
	for rows.Next() {
		var algorithm string
		if err := rows.Scan(&algorithm); err != nil {
			t.Fatalf("scan algorithm: %v", err)
		}
		algorithms = append(algorithms, algorithm)
	}
	if len(algorithms) == 0 {
		t.Fatal("no embeddings were written for the space")
	}
	for _, algorithm := range algorithms {
		if algorithm == intelligence.LocalAlgorithm {
			t.Fatalf("vectors fell back to the offline algorithm; this test would be measuring the wrong backend (%v)", algorithms)
		}
		if !strings.Contains(algorithm, model) {
			t.Fatalf("expected vectors from %q, found algorithm %q", model, algorithm)
		}
	}
}
