package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/auth"
	"github.com/hkjang/umm/internal/store"
)

// The export is umm's own file, and umm has an importer. Until now the two had
// never been introduced: exporting a space and importing it back turned the
// banner into a thought, carried the id/type/canvas list into every body, and
// named every untitled thought "Thought".
//
// The importer now reads this format, which means it depends on these markers.
// This test is the other half of that agreement: change how the exporter writes
// a banner, a metadata key, a section heading or the untitled placeholder, and
// this fails here rather than silently in someone's restored space.
//
// The reader's side is tested in web/src/lib/markdown-import.test.ts, against a
// fixture of exactly this shape.
func TestMarkdownExportKeepsTheShapeTheImporterReadsIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	userID, spaceID := uuid.New(), uuid.New()
	titled, untitled, linked := uuid.New(), uuid.New(), uuid.New()
	username := "export_round_" + strings.ReplaceAll(userID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, username); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'돌아오는 공간')`, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO notes(id,space_id,author_id,title,content) VALUES
		($1,$4,$5,'제목이 있는 생각','제목이 있는 생각의 본문'),
		($2,$4,$5,'','제목이 없는 생각의 본문'),
		($3,$4,$5,'','이어진 생각의 본문')`,
		titled, untitled, linked, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO note_edges(id,space_id,source_note_id,target_note_id,relation,created_by,reason) VALUES($1,$2,$3,$4,'related',$5,$6)`,
		uuid.New(), spaceID, titled, linked, userID, exportedReason); err != nil {
		t.Fatal(err)
	}
	branch, err := db.CreateBranch(ctx, userID, spaceID, "되돌리기 실험", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.SetNoteBranch(ctx, userID, linked, &branch.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ResolveBranch(ctx, userID, branch.ID, store.BranchAdopted, "되돌아왔습니다"); err != nil {
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

	// If this file is written somewhere the reader can pick it up, do so — it
	// keeps the fixture on the other side honest without hand-copying.
	if path := os.Getenv("UMM_EXPORT_SAMPLE_PATH"); path != "" {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	// The importer recognises its own file by this line, and requires it to be
	// the whole of the section it sits in — that is what keeps someone's own
	// note quoting the phrase from being read as an export. So the banner must
	// stay alone between the space name and the first thought.
	firstThought := strings.Index(body, "\n## ")
	if firstThought < 0 {
		t.Fatalf("the export has no thoughts in it:\n%s", body)
	}
	preamble := strings.TrimSpace(body[:firstThought])
	preambleLines := strings.Split(preamble, "\n")
	if len(preambleLines) == 0 || !strings.HasPrefix(preambleLines[0], "# ") {
		t.Fatalf("the export does not open with the space name:\n%s", preamble)
	}
	banner := strings.TrimSpace(strings.Join(preambleLines[1:], "\n"))
	if !strings.HasPrefix(banner, "Exported from umm at ") {
		t.Fatalf("the banner the importer looks for is missing:\n%s", preamble)
	}
	// Alone: one line, nothing else keeping it company.
	if strings.Contains(banner, "\n") {
		t.Errorf("something else joined the banner in its section; the importer reads a banner only when it is the entire body:\n%s", preamble)
	}

	// The metadata keys the importer strips off each thought.
	//
	// Both directions matter, and the second is the one that bites. The importer
	// strips a fixed list of keys, so a key added here that it does not know
	// about is not stripped — it stays in the body, and every restored thought
	// carries a line of bookkeeping nobody wrote. Adding a key to the exporter
	// has to be a deliberate act that updates the reader too, so this fails on
	// an unknown key rather than letting it through.
	known := map[string]bool{"id": true, "type": true, "source": true, "color": true, "canvas": true, "line": true}
	for key := range known {
		if !strings.Contains(body, "- "+key+": `") {
			t.Errorf("the export no longer writes %q; the importer strips exactly these keys", key)
		}
	}
	metadataKey := regexp.MustCompile("(?m)^-\\s+([a-zA-Z]+):\\s+`")
	for _, match := range metadataKey.FindAllStringSubmatch(body, -1) {
		if !known[match[1]] {
			t.Errorf("the export writes %q, which web/src/lib/markdown-import.ts does not strip; "+
				"a restored thought would carry it in its body", match[1])
		}
	}
	// The sections the importer skips rather than importing as thoughts.
	for _, section := range []string{"## Connections", "## Lines of thinking"} {
		if !strings.Contains(body, section) {
			t.Errorf("the export no longer writes %q; the importer skips exactly these", section)
		}
	}
	// The placeholder heading the importer reads back as "no title".
	if !strings.Contains(body, "## Thought\n") {
		t.Errorf("an untitled thought no longer exports as `## Thought`; the importer would restore that word as a title:\n%s", body)
	}
	if !strings.Contains(body, "## 제목이 있는 생각") {
		t.Errorf("a titled thought lost its title:\n%s", body)
	}

	// The why behind a connection is the half that disappears first from
	// anybody's memory and the half a file is actually good at keeping. An
	// export that kept the line and dropped the reason would keep the part that
	// can be reconstructed and lose the part that cannot.
	if !strings.Contains(body, "— "+exportedReason) {
		connections := body[strings.Index(body, "## Connections"):]
		t.Errorf("the export dropped why the connection was drawn:\n%s", connections)
	}
	// In the shape the importer reads: the reason follows the connection on its
	// own line, after an em dash.
	connectionLine := regexp.MustCompile("(?m)^-\\s+`[^`]+`\\s*--[a-z_-]+-->\\s*`[^`]+`(?:\\s*\\([a-z_-]+\\))?(?:\\s*—\\s*(.*))?$")
	match := connectionLine.FindStringSubmatch(body)
	if match == nil {
		t.Fatalf("no connection line in the shape web/src/lib/markdown-import.ts parses:\n%s", body)
	}
	if strings.TrimSpace(match[1]) != exportedReason {
		t.Errorf("the importer's pattern reads the reason as %q, want %q", match[1], exportedReason)
	}
}

// exportedReason is deliberately a sentence with a space and Korean in it: a
// reason written down is prose, not an identifier.
const exportedReason = "같은 회고록을 두 번 읽고 이었다"
