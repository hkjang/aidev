package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/auth"
	"github.com/hkjang/umm/internal/presentation"
	"github.com/hkjang/umm/internal/store"
)

// The API contract for turning a space into a talk.
//
// What has to hold at this layer rather than below it: a preview changes
// nothing, making a deck needs more than a read-only key, and the three ways
// this can refuse are told apart — "connect Ptium", "there is nothing here
// yet" and "Ptium said no" are different problems with different fixes, and
// answering 500 to all of them tells a person only that it broke.

type presentationHarness struct {
	db      *store.Store
	handler http.Handler
	cookie  *http.Cookie
	userID  uuid.UUID
	spaceID uuid.UUID
	notes   []uuid.UUID
}

func presentationAPI(t *testing.T) presentationHarness {
	t.Helper()
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	userID, spaceID := uuid.New(), uuid.New()
	username := "presentation_" + strings.ReplaceAll(userID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, username); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'회고 주기 재검토')`, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	notes := make([]uuid.UUID, 2)
	contents := []string{"회고 주기를 격주로 줄여 보자", "주기가 짧으면 논의가 얕아진다"}
	for i := range notes {
		notes[i] = uuid.New()
		if _, err := db.Pool.Exec(ctx, `INSERT INTO notes(id,space_id,author_id,content,x) VALUES($1,$2,$3,$4,$5)`,
			notes[i], spaceID, userID, contents[i], i*400); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO note_edges(space_id,source_note_id,target_note_id,relation,origin,created_by)
		VALUES($1,$2,$3,'supports','manual',$4)`, spaceID, notes[1], notes[0], userID); err != nil {
		t.Fatal(err)
	}

	authService := &auth.Service{Store: db}
	session, err := authService.CreateSession(ctx, userID, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: db}
	router := chi.NewRouter()
	router.Get("/spaces/{spaceID}/presentation/preview", server.previewPresentation)
	router.Get("/spaces/{spaceID}/export/outline", server.exportOutline)
	router.Post("/spaces/{spaceID}/presentations", server.createPresentation)
	router.Get("/spaces/{spaceID}/presentations", server.listPresentations)
	router.Get("/notes/{noteID}/presentations", server.notePresentations)
	router.Post("/presentations/{linkID}/retry", server.retryPresentation)

	return presentationHarness{
		db:      db,
		handler: authService.Middleware(auth.Require(router)),
		cookie:  &http.Cookie{Name: auth.CookieName, Value: session},
		userID:  userID,
		spaceID: spaceID,
		notes:   notes,
	}
}

func (h presentationHarness) do(t *testing.T, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, path, reader)
	request.AddCookie(h.cookie)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	h.handler.ServeHTTP(recorder, request)
	return recorder
}

// The preview is the person's own storyline, and it is worth showing before
// anyone has connected Ptium at all. Refusing here would make the one part umm
// wrote itself unreachable.
func TestPreviewWorksBeforePtiumIsConnectedIntegration(t *testing.T) {
	h := presentationAPI(t)
	response := h.do(t, http.MethodGet, "/spaces/"+h.spaceID.String()+"/presentation/preview", "")
	if response.Code != 200 {
		t.Fatalf("status %d: %s", response.Code, response.Body.String())
	}
	var preview presentation.Preview
	if err := json.Unmarshal(response.Body.Bytes(), &preview); err != nil {
		t.Fatal(err)
	}
	if len(preview.Storyline.Slides) == 0 {
		t.Fatalf("no storyline: %s", response.Body.String())
	}
	if preview.Storyline.Title != "회고 주기 재검토" {
		t.Fatalf("the space's name did not become the title: %q", preview.Storyline.Title)
	}
	// The person's own sentence, not a paraphrase of it.
	if !strings.Contains(preview.Source, "회고 주기를 격주로 줄여 보자") {
		t.Fatalf("the thought was not carried through verbatim:\n%s", preview.Source)
	}
	// Never checked must not read as checked and clean.
	if preview.Checked {
		t.Fatal("a preview with no Ptium reported itself as checked")
	}
}

// Looking at what a space would become must not create anything.
func TestPreviewWritesNothingIntegration(t *testing.T) {
	h := presentationAPI(t)
	if response := h.do(t, http.MethodGet, "/spaces/"+h.spaceID.String()+"/presentation/preview", ""); response.Code != 200 {
		t.Fatalf("status %d", response.Code)
	}
	var links int
	if err := h.db.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM presentation_links WHERE space_id=$1`, h.spaceID).Scan(&links); err != nil {
		t.Fatal(err)
	}
	if links != 0 {
		t.Fatalf("a preview recorded %d links", links)
	}
}

// "Connect Ptium" is not the same problem as "it broke", and a person can only
// act on the first if it is said.
func TestCreateWithoutPtiumSaysToConnectItIntegration(t *testing.T) {
	h := presentationAPI(t)
	response := h.do(t, http.MethodPost, "/spaces/"+h.spaceID.String()+"/presentations", `{"title":"임원 보고"}`)
	if response.Code != 409 {
		t.Fatalf("status %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "Ptium") {
		t.Fatalf("the message does not name what is missing: %s", response.Body.String())
	}
	var links int
	if err := h.db.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM presentation_links WHERE space_id=$1`, h.spaceID).Scan(&links); err != nil {
		t.Fatal(err)
	}
	if links != 0 {
		t.Fatalf("a refused request still recorded %d links", links)
	}
}

// A space with nothing in it is a different problem again, and answering the
// same way as "Ptium is missing" would send someone to the settings page for
// no reason.
func TestASpaceWithNothingToPresentSaysSoIntegration(t *testing.T) {
	h := presentationAPI(t)
	ctx := context.Background()
	empty := uuid.New()
	if _, err := h.db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'빈 공간')`, empty, h.userID); err != nil {
		t.Fatal(err)
	}
	response := h.do(t, http.MethodPost, "/spaces/"+empty.String()+"/presentations", "")
	if response.Code != 400 {
		t.Fatalf("status %d: %s", response.Code, response.Body.String())
	}
}

// A preview of an empty space is the same refusal, so the UI does not offer a
// "make it" button for a space that cannot produce one.
func TestPreviewOfAnEmptySpaceRefusesIntegration(t *testing.T) {
	h := presentationAPI(t)
	empty := uuid.New()
	if _, err := h.db.Pool.Exec(context.Background(),
		`INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'빈 공간')`, empty, h.userID); err != nil {
		t.Fatal(err)
	}
	if response := h.do(t, http.MethodGet, "/spaces/"+empty.String()+"/presentation/preview", ""); response.Code != 400 {
		t.Fatalf("status %d: %s", response.Code, response.Body.String())
	}
}

// A space someone cannot read compiles to nothing rather than to a deck. The
// permission check rides along with the store call, so this is really checking
// that no path skips it.
func TestAStrangerGetsNothingFromSomeoneElsesSpaceIntegration(t *testing.T) {
	h := presentationAPI(t)
	ctx := context.Background()

	stranger := uuid.New()
	name := "presentation_stranger_" + strings.ReplaceAll(stranger.String(), "-", "")
	if _, err := h.db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, stranger, name); err != nil {
		t.Fatal(err)
	}
	authService := &auth.Service{Store: h.db}
	session, err := authService.CreateSession(ctx, stranger, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/spaces/"+h.spaceID.String()+"/presentation/preview", strings.NewReader(""))
	request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
	recorder := httptest.NewRecorder()
	h.handler.ServeHTTP(recorder, request)

	// Nothing to present, because nothing was readable — not a deck built from
	// someone else's thoughts.
	if recorder.Code == 200 {
		t.Fatalf("a stranger got a storyline from someone else's space: %s", recorder.Body.String())
	}
}

func TestListingPresentationsStartsEmptyIntegration(t *testing.T) {
	h := presentationAPI(t)
	response := h.do(t, http.MethodGet, "/spaces/"+h.spaceID.String()+"/presentations", "")
	if response.Code != 200 {
		t.Fatalf("status %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		Presentations []any `json:"presentations"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	// An empty list, not null: a client iterating it must not have to special
	// case the first visit.
	if body.Presentations == nil {
		t.Fatalf("presentations came back null: %s", response.Body.String())
	}
	if len(body.Presentations) != 0 {
		t.Fatalf("a fresh space already has talks: %s", response.Body.String())
	}
}

// The other direction, from a note. Someone about to edit a thought that decks
// quote is making a different decision from someone editing one nobody used.
func TestANoteReportsWhichTalksQuoteItIntegration(t *testing.T) {
	h := presentationAPI(t)
	ctx := context.Background()

	link, err := h.db.CreatePresentationLink(ctx, h.userID, h.spaceID, "pt_api", "임원 보고", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.db.CompletePresentationLink(ctx, h.userID, link.ID, store.PresentationReady, "# 임원 보고\n",
		[]store.SlideSource{{SlidePosition: 2, NoteID: h.notes[0]}}, 1, 0, 0, "", ""); err != nil {
		t.Fatal(err)
	}

	response := h.do(t, http.MethodGet, "/notes/"+h.notes[0].String()+"/presentations", "")
	if response.Code != 200 {
		t.Fatalf("status %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "pt_api") {
		t.Fatalf("the talk quoting this note was not reported: %s", response.Body.String())
	}

	// And a note nobody used says so rather than reporting someone else's deck.
	response = h.do(t, http.MethodGet, "/notes/"+h.notes[1].String()+"/presentations", "")
	if response.Code != 200 {
		t.Fatalf("status %d", response.Code)
	}
	if strings.Contains(response.Body.String(), "pt_api") {
		t.Fatalf("an unused note claimed a talk: %s", response.Body.String())
	}
}

// A key that may only read must not be able to send someone's thoughts to
// another service and create something there. That is why making a deck asks
// for notes:write while previewing asks for notes:read — and the difference is
// only real if it is checked.
func TestAReadOnlyKeyCannotMakeADeckIntegration(t *testing.T) {
	h := presentationAPI(t)
	ctx := context.Background()

	authService := &auth.Service{Store: h.db}
	_, readOnly, err := authService.CreateKey(ctx, h.userID, "read-only", []string{"notes:read", "spaces:read"}, 1)
	if err != nil {
		t.Fatal(err)
	}

	call := func(method, path string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(method, path, strings.NewReader(""))
		request.Header.Set("Authorization", "Bearer "+readOnly)
		recorder := httptest.NewRecorder()
		h.handler.ServeHTTP(recorder, request)
		return recorder
	}

	// Reading what the space would become is fine.
	if response := call(http.MethodGet, "/spaces/"+h.spaceID.String()+"/presentation/preview"); response.Code != 200 {
		t.Fatalf("a read-only key could not preview: %d %s", response.Code, response.Body.String())
	}
	// Making one is not.
	response := call(http.MethodPost, "/spaces/"+h.spaceID.String()+"/presentations")
	if response.Code != 403 {
		t.Fatalf("a read-only key made a deck: %d %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "notes:write") {
		t.Fatalf("the refusal does not name the missing scope: %s", response.Body.String())
	}
}

// A past deck is only useful if you can get back to it, and the address it
// lives at is computed on every read rather than stored: an administrator who
// moves Ptium would otherwise leave every past link pointing somewhere that no
// longer answers, with nothing to say which ones were stale.
func TestAPastDeckCarriesWhereToOpenItIntegration(t *testing.T) {
	h := presentationAPI(t)
	ctx := context.Background()

	link, err := h.db.CreatePresentationLink(ctx, h.userID, h.spaceID, "pt link/with space", "임원 보고", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.db.CompletePresentationLink(ctx, h.userID, link.ID, store.PresentationReady, "# 임원 보고\n",
		[]store.SlideSource{{SlidePosition: 2, NoteID: h.notes[0]}}, 1, 0, 0, "", ""); err != nil {
		t.Fatal(err)
	}

	read := func() []map[string]any {
		t.Helper()
		response := h.do(t, http.MethodGet, "/spaces/"+h.spaceID.String()+"/presentations", "")
		if response.Code != 200 {
			t.Fatalf("status %d: %s", response.Code, response.Body.String())
		}
		var body struct {
			Presentations []map[string]any `json:"presentations"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body.Presentations
	}

	// With no Ptium configured the list still comes back; there is simply
	// nowhere to click through to.
	rows := read()
	if len(rows) != 1 {
		t.Fatalf("expected one talk, got %+v", rows)
	}
	if _, ok := rows[0]["url"]; ok {
		t.Fatalf("a deck url appeared with no Ptium configured: %+v", rows[0])
	}

	if _, err := h.db.Pool.Exec(ctx,
		`INSERT INTO app_settings(key,value) VALUES('ptium', jsonb_build_object('base_url','https://ptium.internal/'))
		 ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`); err != nil {
		t.Fatal(err)
	}

	rows = read()
	deckURL, _ := rows[0]["url"].(string)
	if deckURL == "" {
		t.Fatalf("no deck url once Ptium is configured: %+v", rows[0])
	}
	// One slash, not two: the configured address may or may not end in one.
	if strings.Contains(strings.TrimPrefix(deckURL, "https://"), "//") {
		t.Fatalf("the trailing slash doubled up: %q", deckURL)
	}
	// Ptium ids are opaque to umm, so nothing may assume they are safe in a path.
	if strings.Contains(deckURL, "pt link/with space") {
		t.Fatalf("the deck id was not escaped into the url: %q", deckURL)
	}
	if !strings.HasSuffix(deckURL, "/editor") {
		t.Fatalf("the url does not open the editor: %q", deckURL)
	}
}

// The same deck has to look the same from both directions.
//
// It did not: the list of a space's talks reported a changed slide while the
// same deck reached from the note that changed reported none. The note view is
// where it matters most — you have just rewritten the thought — and it was the
// one saying nothing was wrong.
func TestBothViewsOfADeckAgreeOnWhatChangedIntegration(t *testing.T) {
	h := presentationAPI(t)
	ctx := context.Background()

	link, err := h.db.CreatePresentationLink(ctx, h.userID, h.spaceID, "pt_agree", "임원 보고", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.db.CompletePresentationLink(ctx, h.userID, link.ID, store.PresentationReady, "# 임원 보고\n",
		[]store.SlideSource{{SlidePosition: 2, NoteID: h.notes[0]}}, 1, 0, 0, "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := h.db.Pool.Exec(ctx, `UPDATE notes SET content='고쳐 썼다' WHERE id=$1`, h.notes[0]); err != nil {
		t.Fatal(err)
	}

	staleFrom := func(path string) int {
		t.Helper()
		response := h.do(t, http.MethodGet, path, "")
		if response.Code != 200 {
			t.Fatalf("%s: status %d", path, response.Code)
		}
		var body struct {
			Presentations []struct {
				ID          string `json:"id"`
				StaleSlides int    `json:"staleSlides"`
			} `json:"presentations"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		for _, p := range body.Presentations {
			if p.ID == link.ID.String() {
				return p.StaleSlides
			}
		}
		t.Fatalf("%s did not return the deck: %s", path, response.Body.String())
		return -1
	}

	fromSpace := staleFrom("/spaces/" + h.spaceID.String() + "/presentations")
	fromNote := staleFrom("/notes/" + h.notes[0].String() + "/presentations")
	if fromSpace != 1 {
		t.Fatalf("the space's list reports %d changed slides, want 1", fromSpace)
	}
	if fromNote != fromSpace {
		t.Fatalf("the two views disagree: space says %d, note says %d", fromSpace, fromNote)
	}
}

// An administrator has to be able to connect Ptium from the screen, and to
// find out whether it worked without making a deck to see.
//
// The test lists Ptium's templates, which is an authenticated read: an answer
// proves the address responds, that it is a Ptium, and that the credential is
// accepted, all at once. Both the failures a person can act on from that screen
// are told apart from each other here.
func TestConnectingPtiumFromTheAdminScreenIntegration(t *testing.T) {
	h := presentationAPI(t)

	// Stands in for Ptium. The shape is the one a real Ptium returns — data is
	// the array itself, not an object wrapping one, which is what the first
	// version of this got wrong and only a live connection revealed.
	var seenAuth string
	ptium := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		if seenAuth != "Bearer ptium_good" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"title":"Unauthorized","detail":"API 키가 올바르지 않습니다"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"t1","name":"Ptium Plum Wash","kind":"builtin"}],"meta":{"count":1}}`))
	}))
	defer ptium.Close()

	router := chi.NewRouter()
	server := &Server{Store: h.db}
	router.Post("/admin/ptium/test", server.testPtium)
	handler := (&auth.Service{Store: h.db}).Middleware(auth.Require(router))

	call := func(body string) (int, string) {
		request := httptest.NewRequest(http.MethodPost, "/admin/ptium/test", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		request.AddCookie(h.cookie)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		return recorder.Code, recorder.Body.String()
	}

	// No address is its own answer: "fill this in", not "it broke".
	if code, body := call(`{}`); code != 400 || !strings.Contains(body, "주소를 입력") {
		t.Fatalf("empty address: %d %s", code, body)
	}
	// A malformed address is a different fix again.
	if code, body := call(`{"base_url":"ptium.internal"}`); code != 400 || !strings.Contains(body, "올바르지 않습니다") {
		t.Fatalf("malformed address: %d %s", code, body)
	}
	// A refused key surfaces what Ptium said rather than a bare status.
	code, body := call(`{"base_url":"` + ptium.URL + `","api_key":"ptium_wrong","timeout_seconds":10}`)
	if code != 400 || !strings.Contains(body, "API 키가 올바르지 않습니다") {
		t.Fatalf("wrong key: %d %s", code, body)
	}

	code, body = call(`{"base_url":"` + ptium.URL + `","api_key":"ptium_good","timeout_seconds":10}`)
	if code != 200 {
		t.Fatalf("a working connection was refused: %d %s", code, body)
	}
	// The templates come back so the screen can offer them by name instead of
	// asking someone to paste a UUID from another service.
	if !strings.Contains(body, "Ptium Plum Wash") || !strings.Contains(body, `"t1"`) {
		t.Fatalf("templates were not returned: %s", body)
	}
	if seenAuth != "Bearer ptium_good" {
		t.Fatalf("the credential reached Ptium as %q", seenAuth)
	}
}

// The screen never sends a stored secret back, so testing a connection nobody
// has changed must not require retyping the key.
func TestTestingAnUnchangedPtiumUsesTheStoredKeyIntegration(t *testing.T) {
	h := presentationAPI(t)
	ctx := context.Background()

	var seenAuth string
	ptium := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer ptium.Close()

	if _, err := h.db.Pool.Exec(ctx,
		`INSERT INTO app_settings(key,value) VALUES('ptium', jsonb_build_object('base_url',$1::text,'api_key','ptium_stored','timeout_seconds',30))
		 ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`, ptium.URL); err != nil {
		t.Fatal(err)
	}

	router := chi.NewRouter()
	router.Post("/admin/ptium/test", (&Server{Store: h.db}).testPtium)
	handler := (&auth.Service{Store: h.db}).Middleware(auth.Require(router))

	// The masked value is exactly what the screen sends back for an untouched
	// secret field.
	request := httptest.NewRequest(http.MethodPost, "/admin/ptium/test",
		strings.NewReader(`{"base_url":"","api_key":"`+secretMask+`","timeout_seconds":0}`))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(h.cookie)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != 200 {
		t.Fatalf("testing an unchanged connection failed: %d %s", recorder.Code, recorder.Body.String())
	}
	if seenAuth != "Bearer ptium_stored" {
		t.Fatalf("the stored key was not used: %q", seenAuth)
	}
}

// What a person is told when making a deck fails.
//
// Every failure that was not one of the three named cases used to come back as
// the same 502 carrying the Go error that produced it — an internal address, a
// Go type name, a SQL constraint. Each said what happened, none said what to
// do, and because they were all the same shape the screen could not tell them
// apart either.
func TestPtiumFailuresSayWhoCanFixThemIntegration(t *testing.T) {
	deck := 0
	newDeck := func(w http.ResponseWriter) {
		deck++
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": fmt.Sprintf("deck-%d", deck), "title": "t"},
		})
	}
	cases := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus int
		wantKind   string
		wantDetail string
		// mustNotSay is what a person who wanted slides must never be shown.
		mustNotSay []string
	}{
		{
			name: "a refused credential is the administrator's",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(401)
				_, _ = w.Write([]byte(`{"detail":"api key is invalid or expired"}`))
			},
			wantStatus: 502, wantKind: "unauthorized",
			mustNotSay: []string{"api key is invalid or expired", "ptium status"},
		},
		{
			name: "a wrong address is the administrator's",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(404)
				_, _ = w.Write([]byte(`{"detail":"not found"}`))
			},
			wantStatus: 502, wantKind: "no-api",
			mustNotSay: []string{"ptium status"},
		},
		{
			// The one case where Ptium's own words tell the author what to
			// change, so they are repeated verbatim and marked as Ptium's.
			name: "a rejected deck is the author's, and Ptium names the slide",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					newDeck(w)
					return
				}
				w.WriteHeader(422)
				_, _ = w.Write([]byte(`{"detail":"slide 3: layout \"two-column\" is not in template basic"}`))
			},
			wantStatus: 422, wantKind: "rejected",
			wantDetail: `slide 3: layout "two-column" is not in template basic`,
		},
		{
			name: "a proxy error page is not Ptium explaining itself",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					newDeck(w)
					return
				}
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(502)
				_, _ = w.Write([]byte("<html><title>502 Bad Gateway</title><body>nginx/1.24.0</body></html>"))
			},
			wantStatus: 502, wantKind: "remote-error",
			mustNotSay: []string{"nginx", "<html>"},
		},
		{
			name: "an answer this version cannot read names no Go types",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`["unexpected","array"]`))
			},
			wantStatus: 502, wantKind: "unexpected-response",
			mustNotSay: []string{"deckEnvelope", "unmarshal"},
		},
	}

	h := presentationAPI(t)
	for _, c := range cases {
		server := httptest.NewServer(c.handler)
		if err := h.db.PutSetting(context.Background(), "ptium", map[string]any{
			"base_url": server.URL, "api_key": "ptium_test", "timeout_seconds": 5,
		}, h.userID); err != nil {
			t.Fatal(err)
		}
		response := h.do(t, http.MethodPost, "/spaces/"+h.spaceID.String()+"/presentations", `{}`)
		server.Close()

		body := response.Body.String()
		if response.Code != c.wantStatus {
			t.Errorf("%s: status %d, want %d: %s", c.name, response.Code, c.wantStatus, body)
		}
		var problem struct {
			Type        string `json:"type"`
			Title       string `json:"title"`
			Detail      string `json:"detail"`
			Failure     string `json:"failure"`
			PtiumDetail string `json:"ptiumDetail"`
			Technical   string `json:"technical"`
		}
		if err := json.Unmarshal([]byte(body), &problem); err != nil {
			t.Fatalf("%s: %v\n%s", c.name, err, body)
		}
		if problem.Failure != c.wantKind {
			t.Errorf("%s: failure %q, want %q", c.name, problem.Failure, c.wantKind)
		}
		if !strings.Contains(problem.Type, c.wantKind) {
			t.Errorf("%s: type %q does not carry the kind, so a client cannot tell it from the others", c.name, problem.Type)
		}
		// The sentence has to be umm's own and has to say something. A type and
		// a status a screen can branch on are no use if the words are still the
		// Go error.
		if problem.Title == "" || problem.Detail == "" {
			t.Errorf("%s: title %q detail %q", c.name, problem.Title, problem.Detail)
		}
		if problem.PtiumDetail != c.wantDetail {
			t.Errorf("%s: ptiumDetail %q, want %q", c.name, problem.PtiumDetail, c.wantDetail)
		}
		// This user is not an admin, so the underlying error must not be here.
		if problem.Technical != "" {
			t.Errorf("%s: a non-administrator was sent the underlying error: %q", c.name, problem.Technical)
		}
		for _, forbidden := range c.mustNotSay {
			if strings.Contains(problem.Detail, forbidden) || strings.Contains(problem.PtiumDetail, forbidden) {
				t.Errorf("%s: %q reached the reader: %s", c.name, forbidden, body)
			}
		}
	}

	// Nothing listening: the message must not carry the address it tried.
	if err := h.db.PutSetting(context.Background(), "ptium", map[string]any{
		"base_url": "http://127.0.0.1:1", "api_key": "ptium_test", "timeout_seconds": 2,
	}, h.userID); err != nil {
		t.Fatal(err)
	}
	response := h.do(t, http.MethodPost, "/spaces/"+h.spaceID.String()+"/presentations", `{}`)
	body := response.Body.String()
	if !strings.Contains(body, `"failure":"unreachable"`) {
		t.Errorf("unreachable: %s", body)
	}
	if strings.Contains(body, "127.0.0.1:1") || strings.Contains(body, "/api/v1/presentations") {
		t.Errorf("the address umm tried to reach was shown to the reader: %s", body)
	}
}

// An administrator is the one who can act on the underlying error, so they are
// the one who gets it. Without this, the test above would pass just as happily
// if `technical` were never sent to anybody.
func TestAdministratorsAreSentTheUnderlyingErrorIntegration(t *testing.T) {
	h := presentationAPI(t)
	if _, err := h.db.Pool.Exec(context.Background(), `UPDATE users SET role='admin' WHERE id=$1`, h.userID); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"detail":"api key is invalid or expired"}`))
	}))
	defer server.Close()
	if err := h.db.PutSetting(context.Background(), "ptium", map[string]any{
		"base_url": server.URL, "api_key": "ptium_test", "timeout_seconds": 5,
	}, h.userID); err != nil {
		t.Fatal(err)
	}
	response := h.do(t, http.MethodPost, "/spaces/"+h.spaceID.String()+"/presentations", `{}`)
	var problem struct {
		Technical string `json:"technical"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(problem.Technical, "401") {
		t.Fatalf("an administrator was not sent the underlying error: %q", problem.Technical)
	}
}

// A failed deck stays in the list, and the list has to say something about it
// that a person can act on. The stored error is the Go one; the kind is what
// decides the sentence.
func TestAFailedDeckRecordsWhatKindOfFailureItWasIntegration(t *testing.T) {
	h := presentationAPI(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "deck-kind", "title": "t"}})
			return
		}
		w.WriteHeader(422)
		_, _ = w.Write([]byte(`{"detail":"slide 3: layout is not in template"}`))
	}))
	defer server.Close()
	if err := h.db.PutSetting(context.Background(), "ptium", map[string]any{
		"base_url": server.URL, "api_key": "ptium_test", "timeout_seconds": 5,
	}, h.userID); err != nil {
		t.Fatal(err)
	}
	if code := h.do(t, http.MethodPost, "/spaces/"+h.spaceID.String()+"/presentations", `{}`).Code; code != 422 {
		t.Fatalf("status %d", code)
	}

	listed := h.do(t, http.MethodGet, "/spaces/"+h.spaceID.String()+"/presentations", "")
	var body struct {
		Presentations []struct {
			Status      string `json:"status"`
			FailureKind string `json:"failureKind"`
			Error       string `json:"error"`
		} `json:"presentations"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Presentations) != 1 {
		t.Fatalf("%d decks listed", len(body.Presentations))
	}
	deck := body.Presentations[0]
	if deck.Status != "failed" {
		t.Fatalf("status %q", deck.Status)
	}
	if deck.FailureKind != "rejected" {
		t.Fatalf("failureKind %q — without it the list can only show the stored Go error", deck.FailureKind)
	}
	// And the underlying message is still kept, because whoever fixes it needs
	// it even though the list does not show it.
	if deck.Error == "" {
		t.Fatal("the underlying error was not recorded")
	}
}

// A failed attempt leaves a real deck in Ptium, and pressing the button again
// used to make another one.
//
// Making a deck is two calls: Ptium opens one, then umm compiles source into
// it. When the second fails — a space large enough to run past the timeout is
// the usual way — the deck exists and is empty. Nothing said so, so a space
// that failed four times left four empty decks behind.
func TestARetryUsesTheDeckTheFailedAttemptAlreadyMadeIntegration(t *testing.T) {
	var created, applied int
	fail := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			created++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"id": fmt.Sprintf("deck-%d", created), "title": "t"},
			})
			return
		}
		applied++
		if fail {
			w.WriteHeader(504)
			_, _ = w.Write([]byte(`{"detail":"compiling took too long"}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"slideCount": 3, "warnings": []string{}}})
	}))
	defer server.Close()

	h := presentationAPI(t)
	if err := h.db.PutSetting(context.Background(), "ptium", map[string]any{
		"base_url": server.URL, "api_key": "ptium_test", "timeout_seconds": 5,
	}, h.userID); err != nil {
		t.Fatal(err)
	}

	response := h.do(t, http.MethodPost, "/spaces/"+h.spaceID.String()+"/presentations", `{}`)
	if response.Code == 201 {
		t.Fatalf("the deck was reported as made although compiling failed: %s", response.Body.String())
	}
	// The failure names the deck Ptium opened, which is what turns "it broke"
	// into something that can be finished.
	var problem struct {
		PtiumID        string `json:"ptiumId"`
		DeckLeftBehind bool   `json:"deckLeftBehind"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
		t.Fatal(err)
	}
	if !problem.DeckLeftBehind || problem.PtiumID != "deck-1" {
		t.Fatalf("the deck left in Ptium was not reported: %s", response.Body.String())
	}

	// The failed attempt is in the list, with the deck it made.
	var listed struct {
		Presentations []struct {
			ID      string `json:"id"`
			Status  string `json:"status"`
			PtiumID string `json:"ptiumId"`
		} `json:"presentations"`
	}
	if err := json.Unmarshal(h.do(t, http.MethodGet, "/spaces/"+h.spaceID.String()+"/presentations", "").Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Presentations) != 1 || listed.Presentations[0].Status != "failed" {
		t.Fatalf("the failed attempt was not recorded: %+v", listed.Presentations)
	}
	linkID, deckOfFirst := listed.Presentations[0].ID, listed.Presentations[0].PtiumID

	// Retry, and Ptium must not be asked for another deck.
	fail = false
	retry := h.do(t, http.MethodPost, "/presentations/"+linkID+"/retry", "")
	if retry.Code != 200 {
		t.Fatalf("retry: %d %s", retry.Code, retry.Body.String())
	}
	if created != 1 {
		t.Fatalf("retrying opened %d decks in Ptium; the one already there was the point", created)
	}
	if applied != 2 {
		t.Fatalf("source was applied %d times, want the failure and the retry", applied)
	}

	// One row still, now ready, still pointing at the same deck.
	listed.Presentations = nil
	if err := json.Unmarshal(h.do(t, http.MethodGet, "/spaces/"+h.spaceID.String()+"/presentations", "").Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Presentations) != 1 {
		t.Fatalf("retrying left %d rows, want the one it was retrying", len(listed.Presentations))
	}
	if got := listed.Presentations[0]; got.Status != "ready" || got.PtiumID != deckOfFirst {
		t.Fatalf("after a successful retry the row is %+v, want ready on %s", got, deckOfFirst)
	}
}

// Only a failed attempt may be retried: re-applying source to a deck that
// compiled would change it behind its owner's back.
func TestOnlyAFailedDeckMayBeRetriedIntegration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "deck-ok", "title": "t"}})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"slideCount": 3, "warnings": []string{}}})
	}))
	defer server.Close()

	h := presentationAPI(t)
	if err := h.db.PutSetting(context.Background(), "ptium", map[string]any{
		"base_url": server.URL, "api_key": "ptium_test", "timeout_seconds": 5,
	}, h.userID); err != nil {
		t.Fatal(err)
	}
	made := h.do(t, http.MethodPost, "/spaces/"+h.spaceID.String()+"/presentations", `{}`)
	if made.Code != 201 {
		t.Fatalf("status %d: %s", made.Code, made.Body.String())
	}
	var result struct {
		Link struct {
			ID string `json:"id"`
		} `json:"link"`
	}
	if err := json.Unmarshal(made.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if code := h.do(t, http.MethodPost, "/presentations/"+result.Link.ID+"/retry", "").Code; code != 404 {
		t.Fatalf("a deck that compiled was retried: %d", code)
	}
	if code := h.do(t, http.MethodPost, "/presentations/"+uuid.New().String()+"/retry", "").Code; code != 404 {
		t.Fatalf("an unknown link was retried: %d", code)
	}
}

// Both exports must treat whitespace-only titles like an omitted title, while
// preserving the thoughts and their order through the real store and service.
func TestPresentationTitleFallbackIntegration(t *testing.T) {
	h := presentationAPI(t)
	base := "/spaces/" + h.spaceID.String()
	previewFor := func(t *testing.T, query string) presentation.Preview {
		t.Helper()
		response := h.do(t, http.MethodGet, base+"/presentation/preview"+query, "")
		if response.Code != http.StatusOK {
			t.Fatalf("preview status %d: %s", response.Code, response.Body.String())
		}
		var preview presentation.Preview
		if err := json.Unmarshal(response.Body.Bytes(), &preview); err != nil {
			t.Fatal(err)
		}
		return preview
	}
	outlineFor := func(t *testing.T, query string) string {
		t.Helper()
		response := h.do(t, http.MethodGet, base+"/export/outline"+query, "")
		if response.Code != http.StatusOK {
			t.Fatalf("outline status %d: %s", response.Code, response.Body.String())
		}
		return response.Body.String()
	}
	const spaceName = "회고 주기 재검토"
	baseline := previewFor(t, "")
	baselineOutline := outlineFor(t, "")
	if len(baseline.Storyline.SlideSources()) != len(h.notes) {
		t.Fatal("baseline did not include both thoughts")
	}
	for _, tc := range []struct {
		name, title, want string
		omitted           bool
	}{
		{name: "omitted", want: spaceName, omitted: true},
		{name: "empty", want: spaceName},
		{name: "ASCII spaces", title: "   ", want: spaceName},
		{name: "tabs and newlines", title: "\t\r\n", want: spaceName},
		{name: "Unicode spaces", title: "\u00a0\u2003\u3000", want: spaceName},
		{name: "ordinary title", title: "임원 보고", want: "임원 보고"},
		{name: "padded title", title: "  임원 보고  ", want: "임원 보고"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			query := ""
			if !tc.omitted {
				query = "?" + url.Values{"title": {tc.title}}.Encode()
			}
			preview := previewFor(t, query)
			if preview.Storyline.Title != tc.want {
				t.Errorf("preview title = %q, want %q", preview.Storyline.Title, tc.want)
			}
			cover := "# " + tc.want + "\n@cover\n"
			if !strings.HasPrefix(preview.Source, cover) || strings.Count(preview.Source, "\n@cover\n") != 1 {
				t.Errorf("preview must start with exactly one cover %q: %q", cover, preview.Source)
			}
			if !reflect.DeepEqual(preview.Storyline.Slides, baseline.Storyline.Slides) {
				t.Error("title changed the thoughts or their order")
			}
			sources := preview.Storyline.SlideSources()
			if len(sources) == 0 || sources[0].SlidePosition != 2 {
				t.Errorf("first content must follow the cover at position 2: %+v", sources)
			}
			if !reflect.DeepEqual(sources, baseline.Storyline.SlideSources()) {
				t.Error("title changed thought source positions or count")
			}
			if preview.Source != strings.Replace(baseline.Source, "# "+spaceName+"\n", "# "+tc.want+"\n", 1) {
				t.Error("preview changed beyond the title")
			}
			outline := outlineFor(t, query)
			heading := "# " + tc.want + "\n\n"
			if !strings.HasPrefix(outline, heading) {
				t.Errorf("outline must start with %q: %q", heading, outline)
			}
			if outline != strings.Replace(baselineOutline, "# "+spaceName+"\n\n", heading, 1) {
				t.Error("outline changed beyond the title")
			}
		})
	}
}
