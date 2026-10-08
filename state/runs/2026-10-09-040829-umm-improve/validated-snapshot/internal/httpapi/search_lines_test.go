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

// Search is where a thought is met most often, so a result from a line that was
// decided against must not read like a current one.
//
// umm found this gap three times after the fact — retrieval, the export, and the
// tools an agent reads over MCP. This is the same rule at the last door.
func TestSearchLabelsResultsFromSetAsideLinesIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	userID, spaceID := uuid.New(), uuid.New()
	inLine, outside := uuid.New(), uuid.New()
	username := "search_line_" + strings.ReplaceAll(userID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, username); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'검색 갈래')`, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO notes(id,space_id,author_id,content) VALUES
		($1,$3,$4,'젠킨스로 배포를 옮기는 실험'),($2,$3,$4,'젠킨스와 무관하지만 젠킨스라는 낱말이 있는 생각')`,
		inLine, outside, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	branch, err := db.CreateBranch(ctx, userID, spaceID, "젠킨스로 이전", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.SetNoteBranch(ctx, userID, inLine, &branch.ID); err != nil {
		t.Fatal(err)
	}
	const reason = "플러그인 호환 부담이 컸습니다"
	if _, err = db.ResolveBranch(ctx, userID, branch.ID, store.BranchAbandoned, reason); err != nil {
		t.Fatal(err)
	}

	authService := &auth.Service{Store: db}
	session, err := authService.CreateSession(ctx, userID, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: db}
	router := chi.NewRouter()
	router.Get("/search", server.searchNotes)
	handler := authService.Middleware(auth.Require(router))

	request := httptest.NewRequest(http.MethodGet, "/search?q=%EC%A0%A0%ED%82%A8%EC%8A%A4", nil)
	request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("search returned %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		Notes     []store.NoteSearchResult `json:"notes"`
		NoteLines map[string]struct {
			Name       string `json:"name"`
			Status     string `json:"status"`
			Resolution string `json:"resolution"`
		} `json:"noteLines"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Notes) == 0 {
		t.Fatal("the search found nothing to label")
	}
	line, ok := body.NoteLines[inLine.String()]
	if !ok {
		t.Fatal("a result from a line that was decided against came back unlabelled")
	}
	if line.Status != store.BranchAbandoned || line.Name != "젠킨스로 이전" {
		t.Errorf("label = %+v", line)
	}
	if line.Resolution != reason {
		t.Errorf("the reason did not come with the label: %q", line.Resolution)
	}
	if _, labelled := body.NoteLines[outside.String()]; labelled {
		t.Error("a thought in no line was given one")
	}
}
