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

// edgeRouter mounts every route that writes to the graph, and nothing else.
// Built here rather than from the server's own tree so the test needs no OIDC
// or realtime wiring; the guards live in the handlers, which are the same ones
// the real router mounts.
func edgeRouter(server *Server) chi.Router {
	router := chi.NewRouter()
	router.Post("/spaces/{spaceID}/edges", server.createEdge)
	router.Put("/edges/{edgeID}/reason", server.setEdgeReason)
	router.Post("/edges/{edgeID}/accept", server.acceptSuggestion)
	router.Delete("/edges/{edgeID}", server.deleteEdge)
	return router
}

// Every way of changing the graph has to ask for notes:write.
//
// Creating a connection did. Deleting one did not, and neither did accepting a
// machine suggestion into the graph — so a key issued for reading could remove
// a line somebody drew, and turn umm's guesses into edges. A guard applied to
// every door but one is the shape this keeps coming back in, so this walks all
// four doors rather than the new one.
func TestEdgeWritesAllRequireWriteScopeIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)
	server := &Server{Store: db}

	userID := uuid.New()
	username := "edge_scope_" + strings.ReplaceAll(userID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, username); err != nil {
		t.Fatal(err)
	}
	space, err := db.CreateSpace(ctx, userID, "연결 권한")
	if err != nil {
		t.Fatal(err)
	}
	first, err := db.CreateNote(ctx, userID, store.Note{SpaceID: space.ID, AuthorID: userID, Content: "회고를 격주로"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := db.CreateNote(ctx, userID, store.Note{SpaceID: space.ID, AuthorID: userID, Content: "논의가 얕아진다"})
	if err != nil {
		t.Fatal(err)
	}
	edge, err := db.CreateEdge(ctx, userID, store.Edge{
		SpaceID: space.ID, SourceID: second.ID, TargetID: first.ID, Relation: store.RelationSupports,
	})
	if err != nil {
		t.Fatal(err)
	}

	authService := &auth.Service{Store: db}
	_, readOnly, err := authService.CreateKey(ctx, userID, "read only", []string{"notes:read"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	_, writable, err := authService.CreateKey(ctx, userID, "writable", []string{"notes:read", "notes:write"}, 1)
	if err != nil {
		t.Fatal(err)
	}

	handler := authService.Middleware(auth.Require(edgeRouter(server)))
	call := func(secret, method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+secret)
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		return response
	}

	doors := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"drawing a connection", http.MethodPost, "/spaces/" + space.ID.String() + "/edges",
			`{"source":"` + first.ID.String() + `","target":"` + second.ID.String() + `","relation":"related"}`},
		{"writing why it was drawn", http.MethodPut, "/edges/" + edge.ID.String() + "/reason",
			`{"reason":"읽기 전용 키가 적었다"}`},
		{"accepting a suggestion", http.MethodPost, "/edges/" + edge.ID.String() + "/accept", ``},
		{"removing a connection", http.MethodDelete, "/edges/" + edge.ID.String(), ``},
	}
	for _, door := range doors {
		refused := call(readOnly, door.method, door.path, door.body)
		if refused.Code != http.StatusForbidden {
			t.Errorf("%s: a notes:read key got %d, want 403 — this door has no scope check",
				door.name, refused.Code)
		}
	}

	// The write key really wrote: proving the harness reaches the handlers and
	// not only their scope guard.
	reason := call(writable, http.MethodPut, "/edges/"+edge.ID.String()+"/reason",
		`{"reason":"측정치가 나중에 나왔다"}`)
	if reason.Code != http.StatusOK {
		t.Fatalf("setting a reason: %d %s", reason.Code, reason.Body.String())
	}
	var stored store.Edge
	if err := json.Unmarshal(reason.Body.Bytes(), &stored); err != nil {
		t.Fatal(err)
	}
	if stored.Reason != "측정치가 나중에 나왔다" {
		t.Fatalf("the response carried reason %q", stored.Reason)
	}

	// And the same requests get past the scope check with a write key, so the
	// refusals above are about permission rather than about the routes being
	// unreachable in this harness. Last, because the final door deletes the
	// connection everything above is addressed to.
	for _, door := range doors {
		allowed := call(writable, door.method, door.path, door.body)
		if allowed.Code == http.StatusForbidden {
			t.Errorf("%s: a notes:write key was refused too (%s)", door.name, allowed.Body.String())
		}
	}

}

// A reason past the limit comes back as a sentence about length, with the
// limit in it, rather than as a database error that says nothing to whoever
// typed it.
func TestEdgeReasonTooLongIsExplainedIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)
	server := &Server{Store: db}

	userID := uuid.New()
	username := "edge_long_" + strings.ReplaceAll(userID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, username); err != nil {
		t.Fatal(err)
	}
	space, err := db.CreateSpace(ctx, userID, "긴 이유")
	if err != nil {
		t.Fatal(err)
	}
	first, err := db.CreateNote(ctx, userID, store.Note{SpaceID: space.ID, AuthorID: userID, Content: "가"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := db.CreateNote(ctx, userID, store.Note{SpaceID: space.ID, AuthorID: userID, Content: "나"})
	if err != nil {
		t.Fatal(err)
	}
	edge, err := db.CreateEdge(ctx, userID, store.Edge{
		SpaceID: space.ID, SourceID: second.ID, TargetID: first.ID, Relation: store.RelationSupports,
	})
	if err != nil {
		t.Fatal(err)
	}
	authService := &auth.Service{Store: db}
	_, secret, err := authService.CreateKey(ctx, userID, "writable", []string{"notes:read", "notes:write"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	handler := authService.Middleware(auth.Require(edgeRouter(server)))

	body, err := json.Marshal(map[string]string{"reason": strings.Repeat("가", store.MaxEdgeReason+1)})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPut, "/edges/"+edge.ID.String()+"/reason", strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer "+secret)
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var problem map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
		t.Fatal(err)
	}
	if problem["maxLength"] != float64(store.MaxEdgeReason) {
		t.Fatalf("the answer does not say how long is allowed: %s", response.Body.String())
	}

	// And nothing was stored, so a refused reason cannot half-arrive.
	var stored string
	if err := db.Pool.QueryRow(ctx, `SELECT reason FROM note_edges WHERE id=$1`, edge.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != "" {
		t.Fatalf("a refused reason was stored as %q", stored)
	}
}
