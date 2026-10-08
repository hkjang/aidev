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

// The export writes people's words onto lines that mean something: a banner
// heading naming the space, a `## ` heading per thought, a list item per line of
// thinking. A newline inside any of those words spills the rest of them onto the
// next line, where the importer reads it as a different thing entirely — and
// nothing on the way in stops one: the API trims a space name and counts its
// characters, a title is stored as it arrives, and the web fields are
// single-line only by being <input>.

func TestOneLineKeepsTheWordsAndDropsTheBreak(t *testing.T) {
	cases := map[string]string{
		"우리 팀\n회고":      "우리 팀 회고",
		"우리 팀\r\n회고":    "우리 팀 회고",
		"우리 팀\r회고":      "우리 팀 회고",
		"  결론\n## 다음  ": "결론 ## 다음",
		"한 줄":           "한 줄",
		"":              "",
	}
	for given, want := range cases {
		if got := oneLine(given); got != want {
			t.Errorf("oneLine(%q) = %q, want %q", given, got, want)
		}
	}
}

// A space whose name holds a newline used to cost the whole restore. The banner
// announces the file as umm's own, and the importer only recognises it when the
// section's entire body is the banner; the second line of the name joined that
// body, the announcement was missed, and the file came back as anybody's
// Markdown — no ids, no positions, no connections, no lines of thinking.
func TestMarkdownExportKeepsNamesOnOneLineIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	userID, spaceID, noteID := uuid.New(), uuid.New(), uuid.New()
	username := "export_one_line_" + strings.ReplaceAll(userID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, username); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,$3)`, spaceID, userID, "우리 팀\n회고"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO notes(id,space_id,author_id,title,content) VALUES($1,$2,$3,$4,$5)`,
		noteID, spaceID, userID, "결론\n## 다음 단계", "생각의 본문"); err != nil {
		t.Fatal(err)
	}
	branch, err := db.CreateBranch(ctx, userID, spaceID, "젠킨스로\n이전", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.SetNoteBranch(ctx, userID, noteID, &branch.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ResolveBranch(ctx, userID, branch.ID, store.BranchAbandoned, "호환 부담이\n얻는 것보다 컸습니다"); err != nil {
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
	lines := strings.Split(body, "\n")

	// What the importer needs to recognise this as umm's own file: one heading
	// line naming the space, then the banner alone under it.
	if lines[0] != "# 우리 팀 회고" {
		t.Errorf("the banner heading is not the space's name on one line: %q", lines[0])
	}
	if lines[1] != "" || !strings.HasPrefix(lines[2], "Exported from umm at ") || lines[3] != "" {
		t.Errorf("the banner is no longer the whole body of its section:\n%q", strings.Join(lines[:4], "\n"))
	}

	// One thought, so one thought heading. A title that broke in two would open
	// a second one and leave half the title as its name.
	headings := []string{}
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			headings = append(headings, line)
		}
	}
	want := []string{"## 결론 ## 다음 단계", "## Lines of thinking"}
	if strings.Join(headings, "\n") != strings.Join(want, "\n") {
		t.Errorf("headings are %q, want %q", headings, want)
	}

	// The metadata list and the lines of thinking are read a line at a time, so
	// a name that runs onto a second line is a key the importer never sees.
	if !strings.Contains(body, "- line: `젠킨스로 이전` (abandoned)") {
		t.Errorf("the thought's line of thinking is not on its own line:\n%s", body)
	}
	if !strings.Contains(body, "- **젠킨스로 이전** — abandoned: 호환 부담이 얻는 것보다 컸습니다\n") {
		t.Errorf("the line of thinking and why it was set aside are not one entry:\n%s", body)
	}
}
