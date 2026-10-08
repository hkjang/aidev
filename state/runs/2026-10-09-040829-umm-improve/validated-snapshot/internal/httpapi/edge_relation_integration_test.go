package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/auth"
	"github.com/hkjang/umm/internal/store"
)

// The API contract for the connection vocabulary. A caller that names a relation
// umm does not have has to learn which ones exist — silently recording the
// generic relation instead would store a connection the caller did not describe
// and hide the mistake from them.
func TestUnknownRelationIsRejectedWithTheVocabularyIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	userID, spaceID := uuid.New(), uuid.New()
	username := "edge_relation_" + strings.ReplaceAll(userID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, username); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'relations')`, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	notes := make([]uuid.UUID, 2)
	for index := range notes {
		notes[index] = uuid.New()
		if _, err := db.Pool.Exec(ctx, `INSERT INTO notes(id,space_id,author_id,content) VALUES($1,$2,$3,$4)`,
			notes[index], spaceID, userID, "생각"); err != nil {
			t.Fatal(err)
		}
	}

	authService := &auth.Service{Store: db}
	session, err := authService.CreateSession(ctx, userID, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: db}
	router := chi.NewRouter()
	router.Post("/spaces/{spaceID}/edges", server.createEdge)
	handler := authService.Middleware(auth.Require(router))

	post := func(relation string) *httptest.ResponseRecorder {
		t.Helper()
		body, _ := json.Marshal(map[string]any{"source": notes[0], "target": notes[1], "relation": relation})
		request := httptest.NewRequest(http.MethodPost, "/spaces/"+spaceID.String()+"/edges", strings.NewReader(string(body)))
		request.Header.Set("Content-Type", "application/json")
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}

	// The two values that used to let a client claim umm produced the connection.
	for _, forged := range []string{"dreamed", "expanded"} {
		response := post(forged)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("relation %q returned %d; provenance can be claimed from a request again", forged, response.Code)
		}
		var problem map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
			t.Fatalf("decode problem: %v", err)
		}
		allowed, ok := problem["allowedRelations"].([]any)
		if !ok || len(allowed) != len(store.Relations()) {
			t.Fatalf("response did not list the vocabulary: %v", problem)
		}
		for _, value := range allowed {
			if value == forged {
				t.Fatalf("%q was offered as an allowed relation", forged)
			}
		}
	}

	// A real relation still works, and the server decides the provenance.
	response := post(string(store.RelationContradicts))
	if response.Code != http.StatusOK && response.Code != http.StatusCreated {
		t.Fatalf("a valid relation returned %d: %s", response.Code, response.Body.String())
	}
	var created store.Edge
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode edge: %v", err)
	}
	if created.Relation != store.RelationContradicts {
		t.Errorf("relation=%q, want %q", created.Relation, store.RelationContradicts)
	}
	if created.Origin != store.OriginManual {
		t.Errorf("origin=%q; a line drawn through the web API is manual", created.Origin)
	}
}
