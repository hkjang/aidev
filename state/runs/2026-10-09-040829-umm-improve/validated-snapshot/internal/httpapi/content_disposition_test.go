package httpapi

import (
	"context"
	"fmt"
	"mime"
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

// A download name is built from a space's name, which is a person's words.
func TestAttachmentDisposition(t *testing.T) {
	tests := []struct {
		name         string
		stem         string
		wantFilename string
		wantEncoded  string
	}{
		{
			name:         "plain ascii is unchanged",
			stem:         "umm-roadmap",
			wantFilename: "umm-roadmap.md",
			wantEncoded:  "umm-roadmap.md",
		},
		{
			// The header cannot spell this name, so the fallback says only
			// where the words were and filename* carries them.
			name:         "korean name",
			stem:         "umm-회의 노트",
			wantFilename: "umm-.md",
			wantEncoded:  "umm-회의 노트.md",
		},
		{
			// A quotation mark would close the quoted string early and hand
			// the rest of a shared space's name to the header parser.
			name:         "quotation mark cannot end the quoted string",
			stem:         `umm-my "space"`,
			wantFilename: "umm-my-space.md",
			wantEncoded:  "umm-my space.md",
		},
		{
			name:         "separators and control characters are dropped",
			stem:         "umm-../etc\\pass\x00wd",
			wantFilename: "umm-..etcpasswd.md",
			wantEncoded:  "umm-..etcpasswd.md",
		},
		{
			name:         "a name with nothing left to spell",
			stem:         "회의",
			wantFilename: "umm-space.md",
			wantEncoded:  "회의.md",
		},
		{
			name:         "an empty name still names the file",
			stem:         "   ",
			wantFilename: "umm.md",
			wantEncoded:  "umm.md",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			header := attachmentDisposition(test.stem, ".md")
			mediatype, params, err := mime.ParseMediaType(header)
			if err != nil {
				t.Fatalf("the header is not readable: %v (%q)", err, header)
			}
			if mediatype != "attachment" {
				t.Errorf("mediatype = %q, want attachment (%q)", mediatype, header)
			}
			// ParseMediaType prefers filename* the way a client must, so this
			// is the name the person actually gets.
			if params["filename"] != test.wantEncoded {
				t.Errorf("filename = %q, want %q (%q)", params["filename"], test.wantEncoded, header)
			}
			if !strings.Contains(header, `filename="`+test.wantFilename+`"`) {
				t.Errorf("the ascii fallback is not %q: %q", test.wantFilename, header)
			}
		})
	}
}

// A picture is drawn where it is, and still has to have a name: its address is
// a uuid, so without one a saved copy is called after the row it came from.
func TestInlineDispositionNamesAPictureWithoutDownloadingIt(t *testing.T) {
	header := inlineDisposition("화이트보드", ".png")
	mediatype, params, err := mime.ParseMediaType(header)
	if err != nil {
		t.Fatalf("the header is not readable: %v (%q)", err, header)
	}
	// The canvas draws this with an <img>; making it an attachment would put a
	// download in place of the picture.
	if mediatype != "inline" {
		t.Errorf("mediatype = %q, want inline (%q)", mediatype, header)
	}
	if params["filename"] != "화이트보드.png" {
		t.Errorf("filename = %q, want the label the picture arrived with (%q)", params["filename"], header)
	}
	// A client that cannot read filename* is told it is a picture, not a space.
	if !strings.Contains(header, `filename="umm-picture.png"`) {
		t.Errorf("the ascii fallback does not say this is a picture: %q", header)
	}
}

// The name is spelled out twice, so an unbounded one is a header nobody reads.
func TestAttachmentDispositionBoundsTheName(t *testing.T) {
	header := attachmentDisposition(strings.Repeat("한", 200), ".md")
	if _, _, err := mime.ParseMediaType(header); err != nil {
		t.Fatalf("the header is not readable: %v", err)
	}
	// 120 bytes is 40 Korean characters, cut between them and not inside one.
	if !strings.Contains(header, rfc5987Escape(strings.Repeat("한", 40)+".md")) {
		t.Errorf("the name was not cut at 120 bytes on a character boundary: %q", header)
	}
}

// exportNameHarness signs somebody in and gives them the two export routes an
// API client reaches for, against a real store: the backup and the document
// outline. They are different files and so must arrive under different names.
type exportNameHarness struct {
	db      *store.Store
	handler http.Handler
	cookie  *http.Cookie
	userID  uuid.UUID
}

func exportNames(t *testing.T) exportNameHarness {
	t.Helper()
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	db := isolatedHTTPStore(t, dsn)

	userID := uuid.New()
	username := "export_name_" + strings.ReplaceAll(userID.String(), "-", "")
	if _, err := db.Pool.Exec(context.Background(), `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, username); err != nil {
		t.Fatal(err)
	}
	authService := &auth.Service{Store: db}
	session, err := authService.CreateSession(context.Background(), userID, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: db}
	router := chi.NewRouter()
	router.Get("/spaces/{spaceID}/export/markdown", server.exportMarkdown)
	router.Get("/spaces/{spaceID}/export/outline", server.exportOutline)
	return exportNameHarness{
		db:      db,
		handler: authService.Middleware(auth.Require(router)),
		cookie:  &http.Cookie{Name: auth.CookieName, Value: session},
		userID:  userID,
	}
}

// space makes a space with something in it, so that either export has words to
// carry rather than refusing.
func (h exportNameHarness) space(t *testing.T, name string) uuid.UUID {
	t.Helper()
	spaceID := uuid.New()
	if _, err := h.db.Pool.Exec(context.Background(), `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,$3)`, spaceID, h.userID, name); err != nil {
		t.Fatal(err)
	}
	if _, err := h.db.Pool.Exec(context.Background(), `INSERT INTO notes(id,space_id,author_id,content) VALUES($1,$2,$3,'생각')`, uuid.New(), spaceID, h.userID); err != nil {
		t.Fatal(err)
	}
	return spaceID
}

// filename is the name the person actually gets: ParseMediaType prefers
// filename* the way a client must.
func (h exportNameHarness) filename(t *testing.T, spaceID uuid.UUID, format string) string {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/spaces/"+spaceID.String()+"/export/"+format, nil)
	request.AddCookie(h.cookie)
	response := httptest.NewRecorder()
	h.handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("%s export returned %d: %s", format, response.Code, response.Body.String())
	}
	disposition := response.Header().Get("Content-Disposition")
	mediatype, params, err := mime.ParseMediaType(disposition)
	if err != nil {
		t.Fatalf("the %s download name is not a readable header: %v (%q)", format, err, disposition)
	}
	if mediatype != "attachment" {
		t.Errorf("%s mediatype = %q, want attachment (%q)", format, mediatype, disposition)
	}
	return params["filename"]
}

// The web canvas names its own download, so this header is what an API client
// — a backup script, curl -OJ — is left with.
func TestMarkdownExportNamesTheFileIntegration(t *testing.T) {
	h := exportNames(t)
	// A name somebody could plausibly give a shared space, and which the plain
	// header form cannot carry.
	spaceID := h.space(t, `"9월" 회의`)
	if name := h.filename(t, spaceID, "markdown"); name != "umm-9월 회의.md" {
		t.Errorf("filename = %q, want the space's own name", name)
	}
}

// The outline is the other file a space leaves as, and it used to leave as
// `umm-outline.md` whichever space it came from: a second download became
// `umm-outline (1).md` in a browser and overwrote the first everywhere else.
func TestOutlineExportNamesTheFileIntegration(t *testing.T) {
	h := exportNames(t)
	spaceID := h.space(t, `"9월" 회의`)

	// The quotation marks are gone the way they are from the backup's name:
	// what is left is the space, spelled as the person spelled it.
	outline := h.filename(t, spaceID, "outline")
	if outline != "umm-outline-9월 회의.md" {
		t.Errorf("outline filename = %q, want the space's own name in it", outline)
	}
	// The backup and the outline of one space hold different things, so a
	// person who takes both must end up with two files rather than one.
	if backup := h.filename(t, spaceID, "markdown"); outline == backup {
		t.Errorf("the outline and the backup are both called %q", outline)
	}
}

// A space nobody has named still has to arrive as a file: the header has to
// parse and the name has to say what the file is.
func TestOutlineExportNamesAnUnnamedSpaceIntegration(t *testing.T) {
	h := exportNames(t)
	for _, spaceName := range []string{"", "   "} {
		t.Run(fmt.Sprintf("%q", spaceName), func(t *testing.T) {
			if name := h.filename(t, h.space(t, spaceName), "outline"); name != "umm-outline-.md" {
				t.Errorf("filename = %q, want a name that still says what the file is", name)
			}
		})
	}
}
