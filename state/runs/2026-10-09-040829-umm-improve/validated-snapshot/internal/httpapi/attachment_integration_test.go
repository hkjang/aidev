package httpapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/auth"
	"github.com/hkjang/umm/internal/store"
)

// These bytes came from a person and are served from the application's own
// origin, so the tests that matter most are about what the response says the
// body is and what a browser is allowed to do with it.

func attachmentHarness(t *testing.T) (*Server, http.Handler, *http.Cookie, uuid.UUID, uuid.UUID) {
	t.Helper()
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)
	userID := uuid.New()
	username := "attach_http_" + strings.ReplaceAll(userID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, username); err != nil {
		t.Fatal(err)
	}
	space, err := db.CreateSpace(ctx, userID, "그림 공간")
	if err != nil {
		t.Fatal(err)
	}
	note, err := db.CreateNote(ctx, userID, store.Note{SpaceID: space.ID, AuthorID: userID, Content: "사진 붙일 생각"})
	if err != nil {
		t.Fatal(err)
	}
	authService := &auth.Service{Store: db}
	session, err := authService.CreateSession(ctx, userID, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: db, Auth: authService}
	return server, authService.Middleware(auth.Require(server.router())),
		&http.Cookie{Name: auth.CookieName, Value: session}, note.ID, space.ID
}

func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	img.Set(2, 2, color.RGBA{B: 255, A: 255})
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func uploadForm(t *testing.T, filename string, data []byte) (string, *bytes.Buffer) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return writer.FormDataContentType(), &body
}

// A picture is stored, and comes back as a picture and nothing else.
func TestAttachmentIsServedAsAnImageAndNothingElseIntegration(t *testing.T) {
	_, handler, cookie, noteID, _ := attachmentHarness(t)
	data := pngBytes(t)

	contentType, body := uploadForm(t, "화이트보드.png", data)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/notes/"+noteID.String()+"/attachments", body)
	request.Header.Set("Content-Type", contentType)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("upload: %d %s", response.Code, response.Body.String())
	}
	var saved store.Attachment
	if err := json.Unmarshal(response.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}

	fetch := httptest.NewRequest(http.MethodGet, "/api/v1/attachments/"+saved.ID.String(), nil)
	fetch.AddCookie(cookie)
	got := httptest.NewRecorder()
	handler.ServeHTTP(got, fetch)
	if got.Code != http.StatusOK {
		t.Fatalf("fetch: %d %s", got.Code, got.Body.String())
	}
	if !bytes.Equal(got.Body.Bytes(), data) {
		t.Fatal("the bytes came back changed")
	}

	// What stops a stored picture becoming stored scripting: the type is the
	// one read off the bytes, the browser is told not to second-guess it, and
	// the body carries a policy that permits nothing.
	if ct := got.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("Content-Type=%q", ct)
	}
	if got.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("nosniff is missing from the picture's own response")
	}
	policy := got.Header().Get("Content-Security-Policy")
	if !strings.Contains(policy, "default-src 'none'") || !strings.Contains(policy, "sandbox") {
		t.Errorf("the picture is served under the page policy rather than one of its own: %q", policy)
	}
	if disposition := got.Header().Get("Content-Disposition"); !strings.HasPrefix(disposition, "inline") {
		t.Errorf("Content-Disposition=%q", disposition)
	}
}

// A picture is fetched from an address that is a uuid, so the name it was given
// has to travel in the header or a saved copy is called after the row.
func TestAttachmentKeepsItsNameWhenSavedIntegration(t *testing.T) {
	_, handler, cookie, noteID, _ := attachmentHarness(t)

	// The label claims a format the bytes are not. umm decided PNG by reading
	// them, so the words are kept and the true ending is added.
	contentType, body := uploadForm(t, "회의 화이트보드.svg", pngBytes(t))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/notes/"+noteID.String()+"/attachments", body)
	request.Header.Set("Content-Type", contentType)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("upload: %d %s", response.Code, response.Body.String())
	}
	var saved store.Attachment
	if err := json.Unmarshal(response.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}

	fetch := httptest.NewRequest(http.MethodGet, "/api/v1/attachments/"+saved.ID.String(), nil)
	fetch.AddCookie(cookie)
	got := httptest.NewRecorder()
	handler.ServeHTTP(got, fetch)
	if got.Code != http.StatusOK {
		t.Fatalf("fetch: %d %s", got.Code, got.Body.String())
	}

	disposition := got.Header().Get("Content-Disposition")
	mediatype, params, err := mime.ParseMediaType(disposition)
	if err != nil {
		t.Fatalf("the picture's name is not a readable header: %v (%q)", err, disposition)
	}
	// The canvas draws this with an <img>. Naming it must not download it.
	if mediatype != "inline" {
		t.Errorf("mediatype = %q, want inline (%q)", mediatype, disposition)
	}
	if params["filename"] != "회의 화이트보드.svg.png" {
		t.Errorf("filename = %q, want the label with the format umm read (%q)", params["filename"], disposition)
	}
}

// The upload's word about itself is worth nothing.
func TestAttachmentRefusesWhatIsNotAnImageIntegration(t *testing.T) {
	_, handler, cookie, noteID, _ := attachmentHarness(t)

	for name, payload := range map[string][]byte{
		"innocent.png": []byte(`<!doctype html><script>alert(document.cookie)</script>`),
		"diagram.svg":  []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`),
	} {
		contentType, body := uploadForm(t, name, payload)
		request := httptest.NewRequest(http.MethodPost, "/api/v1/notes/"+noteID.String()+"/attachments", body)
		request.Header.Set("Content-Type", contentType)
		request.AddCookie(cookie)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusUnsupportedMediaType {
			t.Errorf("%s was accepted with %d: %s", name, response.Code, response.Body.String())
		}
	}
}

// The global body cap is a megabyte and a photo is bigger. Without the
// route-aware limit this fails before the handler runs, with an error about
// nothing the person did.
func TestAttachmentLargerThanTheGlobalBodyCapIntegration(t *testing.T) {
	_, handler, cookie, noteID, _ := attachmentHarness(t)

	// A real PNG over the cap. The pixels are genuinely random rather than a
	// pattern: an arithmetic fill compresses down to a sixth of a megabyte and
	// would leave this test asserting nothing, which is what the guard below
	// caught the first time.
	img := image.NewRGBA(image.Rect(0, 0, 900, 900))
	if _, err := rand.Read(img.Pix); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}
	if out.Len() <= 1<<20 {
		t.Fatalf("the fixture is only %d bytes, which does not exceed the global cap", out.Len())
	}

	contentType, body := uploadForm(t, "큰사진.png", out.Bytes())
	request := httptest.NewRequest(http.MethodPost, "/api/v1/notes/"+noteID.String()+"/attachments", body)
	request.Header.Set("Content-Type", contentType)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("a %d byte picture was refused: %d %s", out.Len(), response.Code, response.Body.String())
	}
}

// Someone who cannot see the space cannot fetch what is on it, and is told the
// same thing they would be told about a picture that does not exist.
func TestAttachmentIsNotReadableByAStrangerIntegration(t *testing.T) {
	server, handler, cookie, noteID, _ := attachmentHarness(t)
	ctx := context.Background()

	contentType, body := uploadForm(t, "사진.png", pngBytes(t))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/notes/"+noteID.String()+"/attachments", body)
	request.Header.Set("Content-Type", contentType)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("upload: %d %s", response.Code, response.Body.String())
	}
	var saved store.Attachment
	if err := json.Unmarshal(response.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}

	stranger := uuid.New()
	name := "stranger_" + strings.ReplaceAll(stranger.String(), "-", "")
	if _, err := server.Store.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, stranger, name); err != nil {
		t.Fatal(err)
	}
	strangerSession, err := server.Auth.CreateSession(ctx, stranger, auth.SessionOrigin{UserAgent: "t", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	fetch := httptest.NewRequest(http.MethodGet, "/api/v1/attachments/"+saved.ID.String(), nil)
	fetch.AddCookie(&http.Cookie{Name: auth.CookieName, Value: strangerSession})
	got := httptest.NewRecorder()
	handler.ServeHTTP(got, fetch)
	if got.Code != http.StatusNotFound {
		t.Fatalf("a stranger got %d for somebody else's picture", got.Code)
	}
	if bytes.Contains(got.Body.Bytes(), []byte("PNG")) {
		t.Fatal("the refusal carried the picture anyway")
	}
}
