package presentation

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// captured is what a stub Ptium saw, so a test can assert on the request rather
// than only on what came back.
type captured struct {
	method string
	path   string
	// raw is the request line as it went over the wire. r.URL.Path is the
	// decoded form, so it shows an escaped separator as a real one and cannot
	// tell a correctly escaped id from an unescaped one.
	raw  string
	auth string
	body map[string]any
}

func stubPtium(t *testing.T, status int, response string) (*Client, *captured) {
	t.Helper()
	seen := &captured{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen.method = r.Method
		seen.path = r.URL.Path
		seen.raw = r.RequestURI
		seen.auth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &seen.body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, response)
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(server.URL, "ptium_test_key", 5*time.Second)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	return client, seen
}

func TestNewClientRefusesNothingToTalkTo(t *testing.T) {
	for _, base := range []string{"", "   "} {
		if _, err := NewClient(base, "k", 0); !errors.Is(err, ErrNotConfigured) {
			t.Fatalf("base %q: got %v, want ErrNotConfigured", base, err)
		}
	}
}

// An unset base url and a mistyped one are different problems and must not
// produce the same message: one is "connect one", the other is "fix this".
func TestNewClientRejectsSomethingThatIsNotAURL(t *testing.T) {
	for _, base := range []string{"not a url", "ptium.example.com", "ftp://ptium.example.com", "file:///etc/passwd"} {
		client, err := NewClient(base, "k", 0)
		if err == nil {
			t.Fatalf("base %q was accepted: %+v", base, client)
		}
		if errors.Is(err, ErrNotConfigured) {
			t.Fatalf("base %q reported as unconfigured rather than invalid", base)
		}
	}
}

func TestCreateDeckSendsTitleAndCredential(t *testing.T) {
	client, seen := stubPtium(t, http.StatusCreated, `{"data":{"id":"pt_1","title":"회고","status":"draft"}}`)
	deck, err := client.CreateDeck(context.Background(), "회고", "", "ko")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if deck.ID != "pt_1" {
		t.Fatalf("deck id: %q", deck.ID)
	}
	if seen.method != http.MethodPost || seen.path != "/api/v1/presentations" {
		t.Fatalf("called %s %s", seen.method, seen.path)
	}
	if seen.auth != "Bearer ptium_test_key" {
		t.Fatalf("credential not sent as bearer: %q", seen.auth)
	}
	if seen.body["title"] != "회고" || seen.body["language"] != "ko" {
		t.Fatalf("body: %+v", seen.body)
	}
}

// The whole point of compiling to source rather than to a prompt: umm never
// asks Ptium to write anything.
func TestCreateDeckNeverSendsAPrompt(t *testing.T) {
	client, seen := stubPtium(t, http.StatusCreated, `{"data":{"id":"pt_1"}}`)
	if _, err := client.CreateDeck(context.Background(), "회고", "", "ko"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, ok := seen.body["prompt"]; ok {
		t.Fatalf("a prompt was sent to ptium: %+v", seen.body)
	}
}

// Ptium's schema is additionalProperties:false, so a field it does not document
// fails the whole request. Only documented ones may be sent, and an empty
// optional is omitted rather than sent blank.
func TestCreateDeckOmitsWhatItWasNotGiven(t *testing.T) {
	client, seen := stubPtium(t, http.StatusCreated, `{"data":{"id":"pt_1"}}`)
	if _, err := client.CreateDeck(context.Background(), "회고", "", ""); err != nil {
		t.Fatalf("create: %v", err)
	}
	allowed := map[string]bool{"title": true, "templateId": true, "language": true}
	for key := range seen.body {
		if !allowed[key] {
			t.Fatalf("undocumented field %q would be rejected by ptium: %+v", key, seen.body)
		}
	}
	if _, ok := seen.body["templateId"]; ok {
		t.Fatalf("an empty templateId was sent: %+v", seen.body)
	}
}

func TestCreateDeckRejectsADeckWithNoID(t *testing.T) {
	client, _ := stubPtium(t, http.StatusCreated, `{"data":{"title":"회고"}}`)
	if _, err := client.CreateDeck(context.Background(), "회고", "", ""); err == nil {
		t.Fatal("a deck with no id was accepted; every later call would use an empty path")
	}
}

func TestCreateDeckTitleFitsPtiumsLimit(t *testing.T) {
	client, seen := stubPtium(t, http.StatusCreated, `{"data":{"id":"pt_1"}}`)
	if _, err := client.CreateDeck(context.Background(), strings.Repeat("가", 400), "", ""); err != nil {
		t.Fatalf("create: %v", err)
	}
	title, _ := seen.body["title"].(string)
	if len(title) > 200 {
		t.Fatalf("title is %d bytes; ptium caps it at 200", len(title))
	}
	// Cut on a rune boundary — half a Korean syllable is not a title.
	if !strings.HasPrefix(strings.Repeat("가", 400), title) {
		t.Fatalf("title was cut mid-character: %q", title)
	}
}

func TestCreateDeckNeverSendsAnEmptyTitle(t *testing.T) {
	client, seen := stubPtium(t, http.StatusCreated, `{"data":{"id":"pt_1"}}`)
	if _, err := client.CreateDeck(context.Background(), "   ", "", ""); err != nil {
		t.Fatalf("create: %v", err)
	}
	if title, _ := seen.body["title"].(string); strings.TrimSpace(title) == "" {
		t.Fatalf("an empty title would fail ptium's minLength: %+v", seen.body)
	}
}

func TestApplySourceCompilesAndCarriesWarnings(t *testing.T) {
	client, seen := stubPtium(t, http.StatusOK, `{"data":{"slideCount":4,"warnings":["레이아웃 없음"]}}`)
	result, err := client.ApplySource(context.Background(), "pt_1", "# 회고\n@cover\n", false)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if seen.method != http.MethodPut || seen.path != "/api/v1/presentations/pt_1/source" {
		t.Fatalf("called %s %s", seen.method, seen.path)
	}
	if result.SlideCount != 4 {
		t.Fatalf("slide count: %d", result.SlideCount)
	}
	// A deck that quietly lost a bullet is worse than one that says it did.
	if len(result.Warnings) != 1 || result.Warnings[0] != "레이아웃 없음" {
		t.Fatalf("warnings dropped: %+v", result.Warnings)
	}
}

func TestApplySourcePassesDryRunThrough(t *testing.T) {
	client, seen := stubPtium(t, http.StatusOK, `{"data":{"slideCount":4}}`)
	if _, err := client.ApplySource(context.Background(), "pt_1", "# 회고\n", true); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if seen.body["dryRun"] != true {
		t.Fatalf("dryRun did not reach ptium: %+v", seen.body)
	}
}

// A preview that silently applied would rewrite the deck someone was only
// looking at.
func TestApplySourceDefaultsToNotDryRun(t *testing.T) {
	client, seen := stubPtium(t, http.StatusOK, `{"data":{"slideCount":1}}`)
	if _, err := client.ApplySource(context.Background(), "pt_1", "# 회고\n", false); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if seen.body["dryRun"] != false {
		t.Fatalf("dryRun should be explicitly false, got %+v", seen.body)
	}
}

func TestApplySourceRefusesEmptyInput(t *testing.T) {
	client, _ := stubPtium(t, http.StatusOK, `{"data":{}}`)
	if _, err := client.ApplySource(context.Background(), "pt_1", "   \n", false); err == nil {
		t.Fatal("compiling nothing would replace a deck's slides with none")
	}
	if _, err := client.ApplySource(context.Background(), "", "# 회고\n", false); err == nil {
		t.Fatal("an empty deck id would be sent as a path")
	}
}

func TestDeckIDIsEscapedIntoThePath(t *testing.T) {
	client, seen := stubPtium(t, http.StatusOK, `{"data":{"slideCount":1}}`)
	// Ptium ids are opaque to umm, so nothing here may assume they are safe in
	// a path.
	if _, err := client.ApplySource(context.Background(), "pt/../../admin", "# 회고\n", false); err != nil {
		t.Fatalf("apply: %v", err)
	}
	// Escaped on the wire, so a server routing on the escaped path sees one
	// segment. The decoded form necessarily still reads as "..", which is why
	// this asserts on the request line instead.
	if strings.Contains(seen.raw, "/../") {
		t.Fatalf("a deck id traversed the path: %q", seen.raw)
	}
	if !strings.Contains(seen.raw, "%2F") {
		t.Fatalf("the separator in the id was not escaped: %q", seen.raw)
	}
}

// Ptium answers failures with RFC 9457, and the sentence that says what went
// wrong has to survive into umm's error.
func TestFailureKeepsPtiumsExplanation(t *testing.T) {
	client, _ := stubPtium(t, http.StatusUnprocessableEntity,
		`{"type":"about:blank","title":"Unprocessable","detail":"템플릿에 해당 레이아웃이 없습니다","status":422}`)
	_, err := client.ApplySource(context.Background(), "pt_1", "# 회고\n", false)
	if err == nil {
		t.Fatal("a 422 was treated as success")
	}
	if !strings.Contains(err.Error(), "템플릿에 해당 레이아웃이 없습니다") {
		t.Fatalf("ptium's explanation was lost: %v", err)
	}
	if !strings.Contains(err.Error(), "422") {
		t.Fatalf("the status was lost: %v", err)
	}
}

func TestFailureWithoutAProblemDocumentStillSaysSomething(t *testing.T) {
	client, _ := stubPtium(t, http.StatusBadGateway, "upstream is down")
	_, err := client.ApplySource(context.Background(), "pt_1", "# 회고\n", false)
	if err == nil || !strings.Contains(err.Error(), "upstream is down") {
		t.Fatalf("a non-problem body was replaced rather than passed through: %v", err)
	}
}

func TestUnreachablePtiumSaysSo(t *testing.T) {
	client, err := NewClient("http://127.0.0.1:1", "k", 2*time.Second)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	_, err = client.CreateDeck(context.Background(), "회고", "", "")
	if err == nil || !strings.Contains(err.Error(), "unreachable") {
		t.Fatalf("expected an unreachable error, got %v", err)
	}
}

func TestResponseThatIsNotJSONIsAnError(t *testing.T) {
	client, _ := stubPtium(t, http.StatusOK, "<html>not ptium</html>")
	if _, err := client.CreateDeck(context.Background(), "회고", "", ""); err == nil {
		t.Fatal("an HTML response was accepted as a deck")
	}
}

// Something answering on the configured port that is not Ptium must not be able
// to feed umm an unbounded body.
//
// The assertion is on how much the server managed to write, not on the error.
// The read cap and the length check are belt and braces, so removing either one
// still produces "too large" — and a test that only asserts the error passed
// with the cap taken out. What the cap actually protects is memory, so what is
// measured is whether the body was still being read long past it.
func TestAnEnormousResponseIsNotReadPastTheCap(t *testing.T) {
	const chunkSize = 64 << 10
	attempt := MaxResponseBody * 16

	written := make(chan int, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":{"id":"`)
		chunk := strings.Repeat("x", chunkSize)
		total := 0
		for total < attempt {
			n, err := io.WriteString(w, chunk)
			total += n
			if err != nil {
				break // the client stopped reading, which is the point
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
		written <- total
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "k", 30*time.Second)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	_, err = client.CreateDeck(context.Background(), "회고", "", "")
	if err == nil {
		t.Fatal("an unbounded response was accepted")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected the size to be the reason, got: %v", err)
	}

	select {
	case total := <-written:
		// Generous slack for buffering in the connection and the test server;
		// the point is orders of magnitude, not an exact byte.
		if limit := MaxResponseBody * 4; total > limit {
			t.Fatalf("server wrote %d bytes, past the %d cap by more than buffering explains", total, limit)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the handler never stopped writing")
	}
}

func TestBaseURLTrailingSlashDoesNotDoubleUp(t *testing.T) {
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":{"id":"pt_1"}}`)
	}))
	defer server.Close()

	client, err := NewClient(server.URL+"/", "k", 5*time.Second)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	if _, err := client.CreateDeck(context.Background(), "회고", "", ""); err != nil {
		t.Fatalf("create: %v", err)
	}
	if path != "/api/v1/presentations" {
		t.Fatalf("path is %q", path)
	}
}

func TestNoCredentialSendsNoHeader(t *testing.T) {
	var auth string
	var present bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		_, present = r.Header["Authorization"]
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":{"id":"pt_1"}}`)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "", 5*time.Second)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	if _, err := client.CreateDeck(context.Background(), "회고", "", ""); err != nil {
		t.Fatalf("create: %v", err)
	}
	// An empty bearer is a credential Ptium will reject with a confusing 401;
	// sending nothing lets it answer "no credential" plainly.
	if present || auth != "" {
		t.Fatalf("an empty credential was sent: %q", auth)
	}
}

func TestACancelledContextStopsTheCall(t *testing.T) {
	client, _ := stubPtium(t, http.StatusOK, `{"data":{"id":"pt_1"}}`)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.CreateDeck(ctx, "회고", "", ""); err == nil {
		t.Fatal("a cancelled context still called ptium")
	}
}

func TestTemplatesListsWhatPtiumOffers(t *testing.T) {
	client, seen := stubPtium(t, http.StatusOK,
		`{"data":[{"id":"t1","name":"KCB Executive","kind":"builtin"},{"id":"t2","name":"연간 보고"}],"meta":{"count":2}}`)
	templates, err := client.Templates(context.Background())
	if err != nil {
		t.Fatalf("templates: %v", err)
	}
	if seen.method != http.MethodGet || seen.path != "/api/v1/templates" {
		t.Fatalf("called %s %s", seen.method, seen.path)
	}
	if len(templates) != 2 || templates[1].Name != "연간 보고" {
		t.Fatalf("templates: %+v", templates)
	}
	if seen.auth != "Bearer ptium_test_key" {
		t.Fatalf("the credential was not sent: %q", seen.auth)
	}
}

// A read carries no body. Sending "null" with a JSON content type on a GET is
// what a strict server, or a proxy in front of one, rejects — and there is
// nothing to send.
func TestAReadSendsNoBody(t *testing.T) {
	var length int64
	var contentType string
	var present bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		length = r.ContentLength
		contentType = r.Header.Get("Content-Type")
		_, present = r.Header["Content-Type"]
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":[]}`)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "k", 5*time.Second)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	if _, err := client.Templates(context.Background()); err != nil {
		t.Fatalf("templates: %v", err)
	}
	if length > 0 {
		t.Fatalf("a read sent %d bytes of body", length)
	}
	if present {
		t.Fatalf("a read declared a content type: %q", contentType)
	}
}

// An empty list is not an error: a Ptium with no templates is unusual but
// answering, and the connection test has to tell that apart from a refusal.
func TestNoTemplatesIsNotAFailure(t *testing.T) {
	client, _ := stubPtium(t, http.StatusOK, `{"data":[]}`)
	templates, err := client.Templates(context.Background())
	if err != nil {
		t.Fatalf("templates: %v", err)
	}
	if len(templates) != 0 {
		t.Fatalf("templates: %+v", templates)
	}
}

func TestARefusedCredentialSurfacesFromTemplates(t *testing.T) {
	client, _ := stubPtium(t, http.StatusUnauthorized, `{"title":"Unauthorized","detail":"API 키가 올바르지 않습니다"}`)
	_, err := client.Templates(context.Background())
	if err == nil || !strings.Contains(err.Error(), "API 키가 올바르지 않습니다") {
		t.Fatalf("got %v, want ptium's explanation", err)
	}
}
