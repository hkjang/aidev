package httpapi

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/auth"
	"github.com/hkjang/umm/internal/intelligence"
	"github.com/hkjang/umm/internal/store"
)

// The failure this endpoint exists to expose: an operator configures an
// embedding model, sees no error anywhere in the product, and is still running
// on the offline algorithm — so "related thoughts" keeps meaning "shares
// vocabulary". These tests cover the three answers the screen has to get right:
// nothing configured, configured but not actually in use, and genuinely working.

func embeddingQualityHarness(t *testing.T) (*store.Store, uuid.UUID, string, http.Handler) {
	t.Helper()
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	db := isolatedHTTPStore(t, dsn)
	adminID := uuid.New()
	username := "embed_quality_" + strings.ReplaceAll(adminID.String(), "-", "")
	if _, err := db.Pool.Exec(context.Background(),
		`INSERT INTO users(id,username,display_name,role) VALUES($1,$2::citext,$2::text,'admin')`,
		adminID, username); err != nil {
		t.Fatal(err)
	}
	authService := &auth.Service{Store: db}
	session, err := authService.CreateSession(context.Background(), adminID,
		auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: db}
	router := chi.NewRouter()
	router.Get("/embedding-quality", server.embeddingQuality)
	return db, adminID, session, authService.Middleware(auth.Require(auth.RequireAdmin(router)))
}

func fetchQuality(t *testing.T, handler http.Handler, session, query string) map[string]any {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/embedding-quality"+query, nil)
	request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("embedding-quality=%d body=%s", response.Code, response.Body.String())
	}
	var report map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	return report
}

// With no model configured umm must say so plainly. If this ever reports
// semantic, the warning that tells an operator to configure a model disappears
// and the product goes back to silently claiming meaning it does not have.
func TestEmbeddingQualityReportsTheDefaultAsLexicalIntegration(t *testing.T) {
	_, _, session, handler := embeddingQualityHarness(t)
	report := fetchQuality(t, handler, session, "")

	if report["semantic"] != false {
		t.Error("the offline default must not report itself as a semantic backend")
	}
	if report["fellBack"] != false {
		t.Error("nothing was configured, so nothing fell back")
	}
	if report["algorithm"] != intelligence.LocalAlgorithm {
		t.Errorf("algorithm=%v want %q", report["algorithm"], intelligence.LocalAlgorithm)
	}
	if discrimination, _ := report["discrimination"].(float64); discrimination >= 0 {
		t.Errorf("the character n-gram default scores vocabulary above meaning; got %+.3f", discrimination)
	}
}

// The case the product used to hide completely: the model is set, the gateway
// is dead, embeddings silently fall back, and every screen carries on as if
// nothing happened.
func TestEmbeddingQualityFlagsASilentFallbackIntegration(t *testing.T) {
	db, adminID, session, handler := embeddingQualityHarness(t)
	// A port nothing is listening on stands in for an unreachable gateway.
	if err := db.PutSetting(context.Background(), "ai_gateway", map[string]any{
		"base_url":        "http://127.0.0.1:1",
		"embedding_model": "bge-m3",
		"timeout_seconds": 2,
	}, adminID); err != nil {
		t.Fatal(err)
	}
	report := fetchQuality(t, handler, session, "")

	if report["fellBack"] != true {
		t.Fatalf("a configured but unreachable gateway must be reported as a fallback, got %#v", report)
	}
	if report["model"] != "bge-m3" {
		t.Errorf("the report must name the configured model, got %v", report["model"])
	}
	if report["algorithm"] != intelligence.LocalAlgorithm {
		t.Errorf("the vectors came from the local algorithm, report should say so; got %v", report["algorithm"])
	}
	if report["semantic"] != false {
		t.Error("a fallback to the offline algorithm is not a semantic backend")
	}
}

// A working backend must come back clean: semantic, no fallback warning, and
// named. This also pins the caching contract, because re-embedding 44 sentences
// on every visit to the settings screen would hammer the operator's gateway.
func TestEmbeddingQualityReportsAWorkingBackendAndCachesItIntegration(t *testing.T) {
	var calls int64
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&calls, 1)
		var request struct {
			Input []string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		type embedding struct {
			Embedding []float64 `json:"embedding"`
			Index     int       `json:"index"`
		}
		payload := struct {
			Data []embedding `json:"data"`
		}{}
		for index, text := range request.Input {
			payload.Data = append(payload.Data, embedding{Embedding: idealVector(text), Index: index})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer gateway.Close()

	db, adminID, session, handler := embeddingQualityHarness(t)
	if err := db.PutSetting(context.Background(), "ai_gateway", map[string]any{
		"base_url":        gateway.URL,
		"embedding_model": "stub-semantic",
		"timeout_seconds": 10,
	}, adminID); err != nil {
		t.Fatal(err)
	}

	report := fetchQuality(t, handler, session, "")
	if report["semantic"] != true {
		t.Fatalf("a backend that separates meaning from vocabulary must report semantic, got %#v", report)
	}
	if report["fellBack"] != false {
		t.Error("the gateway answered, so nothing fell back")
	}
	if report["model"] != "stub-semantic" {
		t.Errorf("model=%v want stub-semantic", report["model"])
	}
	if discrimination, _ := report["discrimination"].(float64); discrimination <= 0 {
		t.Errorf("discrimination=%v; a semantic backend must rank meaning above vocabulary", discrimination)
	}

	after := atomic.LoadInt64(&calls)
	fetchQuality(t, handler, session, "")
	if atomic.LoadInt64(&calls) != after {
		t.Error("a second read re-measured; the report must be cached between visits")
	}
	fetchQuality(t, handler, session, "?refresh=true")
	if atomic.LoadInt64(&calls) == after {
		t.Error("refresh=true must re-measure so a settings change can be re-checked")
	}
}

// idealVector fabricates what a good embedding model would produce: both sides
// of a paraphrase land close together, a lexical decoy's two sides land far
// apart, and nothing depends on the words themselves. It exists to test umm's
// reporting, not to stand in for an embedding algorithm.
func idealVector(text string) []float64 {
	for index, pair := range intelligence.QualityPairs {
		side := 0
		switch text {
		case pair.A:
			side = 1
		case pair.B:
			side = 2
		default:
			continue
		}
		// Angle between the two sides, by class: near-identical for a
		// paraphrase, orthogonal for a decoy.
		separation := map[intelligence.PairClass]float64{
			intelligence.ClassParaphrase:   0.10,
			intelligence.ClassRelated:      0.40,
			intelligence.ClassLexicalDecoy: 1.30,
			intelligence.ClassUnrelated:    1.45,
		}[pair.Class]
		// Give every pair its own base direction so unrelated pairs are not all
		// collapsed onto one axis.
		base := float64(index) * 0.37
		angle := base
		if side == 2 {
			angle = base + separation
		}
		// The pair plane is kept clear of the topic plane below, so the two halves
		// of the measurement cannot contaminate each other.
		return []float64{math.Cos(angle), math.Sin(angle), 0.15, 0, 0}
	}
	// Topic sentences: one direction per topic, with a small per-sentence wobble
	// so members of a topic are close without being identical.
	for topic, group := range intelligence.QualityTopics {
		for position, sentence := range group.Sentences {
			if sentence != text {
				continue
			}
			angle := float64(topic)*(math.Pi/2) + float64(position)*0.05
			return []float64{0, 0, 0, math.Cos(angle), math.Sin(angle)}
		}
	}
	return []float64{1, 0, 0, 0, 0}
}
