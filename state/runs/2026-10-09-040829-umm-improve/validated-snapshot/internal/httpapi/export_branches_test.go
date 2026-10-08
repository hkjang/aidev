package httpapi

import (
	"context"
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

// An export is what someone keeps when umm is gone.
//
// A thought that was tried and set aside reads exactly like a current one once
// the label is gone, and the reason it was set aside is the half people lose
// first — losing it at the moment they take their record elsewhere is the worst
// possible time for it to go.
func TestMarkdownExportCarriesLinesOfThinkingIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	userID, spaceID := uuid.New(), uuid.New()
	inLine, outside := uuid.New(), uuid.New()
	username := "export_line_" + strings.ReplaceAll(userID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, username); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'내보내기')`, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO notes(id,space_id,author_id,content) VALUES
		($1,$3,$4,'접어 둔 갈래 안의 생각'),($2,$3,$4,'갈래에 속하지 않은 생각')`,
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
	const reason = "플러그인 호환 부담이 통제권으로 얻는 것보다 컸습니다"
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
	router.Get("/spaces/{spaceID}/export/markdown", server.exportMarkdown)
	handler := authService.Middleware(auth.Require(router))

	request := httptest.NewRequest(http.MethodGet, "/spaces/"+spaceID.String()+"/export/markdown", nil)
	request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("export returned %d: %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, "접어 둔 갈래 안의 생각") {
		t.Fatal("the thought itself did not survive the export")
	}
	if !strings.Contains(body, "- line: `젠킨스로 이전` (abandoned)") {
		t.Errorf("the thought does not say which line it belonged to:\n%s", body)
	}
	if !strings.Contains(body, reason) {
		t.Error("the reason the line was set aside was dropped; the decision survives and the why does not")
	}
	// A thought in no line must not grow one.
	outsideBlock := body[strings.Index(body, "갈래에 속하지 않은 생각"):]
	if end := strings.Index(outsideBlock, "\n## "); end > 0 {
		outsideBlock = outsideBlock[:end]
	}
	if strings.Contains(outsideBlock, "- line:") {
		t.Errorf("a thought in no line was labelled with one:\n%s", outsideBlock)
	}
}
