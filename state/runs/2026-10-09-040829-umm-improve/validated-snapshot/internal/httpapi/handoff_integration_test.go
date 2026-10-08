package httpapi

import (
	"context"
	"encoding/json"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/auth"
	"github.com/hkjang/umm/internal/cryptoutil"
	"github.com/hkjang/umm/internal/store"
)

// The sending half of the handoff standard, walked through the real router:
// nothing to send to until an administrator names a service, a claim bound
// to one space and one person, collected once without a sign-in, and gone.

func TestHandoffClaimIsIssuedOnceCollectedOnceAndNeverLoggedIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	newUser := func(prefix, role string) uuid.UUID {
		t.Helper()
		id := uuid.New()
		username := prefix + strings.ReplaceAll(id.String(), "-", "")
		if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name,role) VALUES($1,$2::citext,$2::text,$3)`, id, username, role); err != nil {
			t.Fatal(err)
		}
		return id
	}
	authorID, strangerID, adminID := newUser("handoff_author_", "user"), newUser("handoff_stranger_", "user"), newUser("handoff_admin_", "admin")
	space, err := db.CreateSpace(ctx, authorID, "2026년 3분기 개편안")
	if err != nil {
		t.Fatal(err)
	}
	for _, content := range []string{"결론: 조직을 셋으로 나눈다", "근거: 지금은 한 팀이 세 제품을 본다", "다음 단계: 10월에 인원을 정한다"} {
		if _, err := db.CreateNote(ctx, authorID, store.Note{SpaceID: space.ID, AuthorID: authorID, Content: content}); err != nil {
			t.Fatal(err)
		}
	}
	empty, err := db.CreateSpace(ctx, authorID, "빈 공간")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.PutSetting(ctx, "general", map[string]any{"service_name": "umm", "public_url": "https://umm.intra/", "session_hours": 24}, adminID); err != nil {
		t.Fatal(err)
	}

	cipher, err := cryptoutil.New([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	authService := &auth.Service{Store: db}
	server := &Server{Store: db, Auth: authService, Cipher: cipher, OIDC: &auth.OIDCService{Store: db, Cipher: cipher, Sessions: authService}}
	handler := server.router()
	cookieFor := func(userID uuid.UUID) *http.Cookie {
		t.Helper()
		session, err := authService.CreateSession(ctx, userID, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
		if err != nil {
			t.Fatal(err)
		}
		return &http.Cookie{Name: auth.CookieName, Value: session}
	}
	author, stranger, admin := cookieFor(authorID), cookieFor(strangerID), cookieFor(adminID)
	call := func(method, target string, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
		t.Helper()
		var reader *strings.Reader
		if body != "" {
			reader = strings.NewReader(body)
		} else {
			reader = strings.NewReader("")
		}
		request := httptest.NewRequest(method, target, reader)
		if body != "" {
			request.Header.Set("Content-Type", "application/json")
		}
		if cookie != nil {
			request.AddCookie(cookie)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}
	targets := func() []map[string]string {
		t.Helper()
		response := call(http.MethodGet, "/api/v1/handoff/targets", "", author)
		if response.Code != http.StatusOK {
			t.Fatalf("targets: %d %s", response.Code, response.Body.String())
		}
		var payload struct {
			Targets []map[string]string `json:"targets"`
			Source  string              `json:"source"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		if payload.Source != "https://umm.intra" {
			t.Errorf("source = %q, want the configured public origin", payload.Source)
		}
		return payload.Targets
	}

	// A new installation: nowhere to send to, so the canvas has no menu.
	if got := targets(); len(got) != 0 {
		t.Fatalf("targets on a fresh installation = %v, want none", got)
	}

	// An administrator names two services. Weekly receives pptx only, and umm
	// sends Markdown, so Weekly is a real entry and not somewhere to send.
	saved := call(http.MethodPut, "/api/v1/admin/settings/handoff", `{"targets":[
		{"name":"Ptium","origin":"https://ptium.intra","formats":["markdown","docx"]},
		{"name":"Weekly","origin":"https://weekly.intra","formats":["pptx"]}]}`, admin)
	if saved.Code != http.StatusOK {
		t.Fatalf("saving targets: %d %s", saved.Code, saved.Body.String())
	}
	if got := targets(); len(got) != 1 || got[0]["name"] != "Ptium" || got[0]["origin"] != "https://ptium.intra" {
		t.Fatalf("targets = %v, want only Ptium", got)
	}
	// A target with a path is not an origin, and is refused before it is saved.
	refused := call(http.MethodPut, "/api/v1/admin/settings/handoff", `{"targets":[{"name":"Ptium","origin":"https://ptium.intra/handoff","formats":["markdown"]}]}`, admin)
	if refused.Code != http.StatusBadRequest {
		t.Errorf("an origin with a path was saved: %d %s", refused.Code, refused.Body.String())
	}
	if got := targets(); len(got) != 1 {
		t.Errorf("the refused save changed the list: %v", got)
	}

	// The author issues a claim.
	issued := call(http.MethodPost, "/api/v1/handoff/claims", `{"resource":"`+space.ID.String()+`","format":"markdown"}`, author)
	if issued.Code != http.StatusCreated {
		t.Fatalf("issuing a claim: %d %s", issued.Code, issued.Body.String())
	}
	if encoding := issued.Header().Get("Content-Encoding"); encoding != "" {
		t.Errorf("a response carrying a fresh claim was compressed (%s)", encoding)
	}
	var claim struct {
		Claim       string `json:"claim"`
		Source      string `json:"source"`
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
		Bytes       int    `json:"bytes"`
		ExpiresAt   string `json:"expires_at"`
	}
	if err := json.Unmarshal(issued.Body.Bytes(), &claim); err != nil {
		t.Fatal(err)
	}
	if len(claim.Claim) < 32 {
		t.Errorf("claim %q is shorter than 128 bits could be", claim.Claim)
	}
	if claim.Source != "https://umm.intra" || claim.Filename != "2026년 3분기 개편안.md" || claim.ContentType != "text/markdown; charset=utf-8" {
		t.Errorf("claim announces source=%q filename=%q content_type=%q", claim.Source, claim.Filename, claim.ContentType)
	}
	expires, err := time.Parse(time.RFC3339, claim.ExpiresAt)
	if err != nil {
		t.Fatalf("expires_at %q is not RFC 3339: %v", claim.ExpiresAt, err)
	}
	if until := time.Until(expires); until <= 0 || until > 5*time.Minute {
		t.Errorf("claim expires in %s, want within five minutes", until)
	}

	// The claim is not written anywhere it would outlive itself.
	var inAudit bool
	if err := db.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM audit_logs WHERE metadata::text LIKE '%' || $1 || '%')`, claim.Claim).Scan(&inAudit); err != nil {
		t.Fatal(err)
	}
	if inAudit {
		t.Error("the claim was written to the audit log")
	}
	var inTable bool
	if err := db.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM handoff_claims WHERE encode(claim_digest,'base64') = $1 OR encode(claim_digest,'escape') = $1)`, claim.Claim).Scan(&inTable); err != nil {
		t.Fatal(err)
	}
	if inTable {
		t.Error("the claim is stored as itself rather than as a digest")
	}

	// Whoever brings the claim gets the document, without a sign-in.
	collected := call(http.MethodGet, "/api/v1/handoff/claims/"+claim.Claim, "", nil)
	if collected.Code != http.StatusOK {
		t.Fatalf("collecting: %d %s", collected.Code, collected.Body.String())
	}
	if got := collected.Header().Get("Content-Type"); got != "text/markdown; charset=utf-8" {
		t.Errorf("Content-Type = %q", got)
	}
	_, params, err := mime.ParseMediaType(collected.Header().Get("Content-Disposition"))
	if err != nil {
		t.Fatalf("Content-Disposition %q is not readable: %v", collected.Header().Get("Content-Disposition"), err)
	}
	if params["filename"] != "2026년 3분기 개편안.md" {
		t.Errorf("collected as %q, want the space's name", params["filename"])
	}
	body := collected.Body.String()
	if len(body) != claim.Bytes {
		t.Errorf("collected %d bytes, the claim announced %d", len(body), claim.Bytes)
	}
	if !strings.HasPrefix(body, "# 2026년 3분기 개편안\n") || !strings.Contains(body, "조직을 셋으로 나눈다") {
		t.Errorf("the document is not the space's outline:\n%s", body)
	}
	if strings.Contains(body, "- id: `") {
		t.Errorf("the document carries backup metadata a reader would skip past:\n%s", body)
	}

	// Once. The second collector gets nothing, and is not told why.
	if again := call(http.MethodGet, "/api/v1/handoff/claims/"+claim.Claim, "", nil); again.Code != http.StatusNotFound {
		t.Errorf("a spent claim answered %d, want 404", again.Code)
	}
	if never := call(http.MethodGet, "/api/v1/handoff/claims/not-a-claim-anyone-issued", "", nil); never.Code != http.StatusNotFound {
		t.Errorf("a claim never issued answered %d, want 404", never.Code)
	}

	// Expired is spent too, before any sweep runs.
	issued = call(http.MethodPost, "/api/v1/handoff/claims", `{"resource":"`+space.ID.String()+`","format":"markdown"}`, author)
	if issued.Code != http.StatusCreated {
		t.Fatalf("issuing a second claim: %d %s", issued.Code, issued.Body.String())
	}
	if err := json.Unmarshal(issued.Body.Bytes(), &claim); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `UPDATE handoff_claims SET expires_at = now() - interval '1 second'`); err != nil {
		t.Fatal(err)
	}
	if expired := call(http.MethodGet, "/api/v1/handoff/claims/"+claim.Claim, "", nil); expired.Code != http.StatusNotFound {
		t.Errorf("an expired claim answered %d, want 404", expired.Code)
	}

	// A claim is bound to what this person may read. A stranger asking for the
	// author's space gets the same answer as for a space that does not exist,
	// and no claim comes into being.
	var before int
	if err := db.Pool.QueryRow(ctx, `SELECT count(*) FROM handoff_claims`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if theirs := call(http.MethodPost, "/api/v1/handoff/claims", `{"resource":"`+space.ID.String()+`","format":"markdown"}`, stranger); theirs.Code != http.StatusNotFound {
		t.Errorf("a stranger's claim on the author's space answered %d, want 404", theirs.Code)
	}
	var after int
	if err := db.Pool.QueryRow(ctx, `SELECT count(*) FROM handoff_claims`).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Errorf("a stranger's request left a claim behind (%d → %d rows)", before, after)
	}

	// umm sends Markdown and nothing else, and an empty space is nothing to send.
	if docx := call(http.MethodPost, "/api/v1/handoff/claims", `{"resource":"`+space.ID.String()+`","format":"docx"}`, author); docx.Code != http.StatusBadRequest {
		t.Errorf("a docx claim answered %d, want 400", docx.Code)
	}
	if nothing := call(http.MethodPost, "/api/v1/handoff/claims", `{"resource":"`+empty.ID.String()+`","format":"markdown"}`, author); nothing.Code != http.StatusBadRequest {
		t.Errorf("a claim on an empty space answered %d, want 400: %s", nothing.Code, nothing.Body.String())
	}
	if anonymous := call(http.MethodPost, "/api/v1/handoff/claims", `{"resource":"`+space.ID.String()+`","format":"markdown"}`, nil); anonymous.Code != http.StatusUnauthorized {
		t.Errorf("issuing without a sign-in answered %d, want 401", anonymous.Code)
	}
}
