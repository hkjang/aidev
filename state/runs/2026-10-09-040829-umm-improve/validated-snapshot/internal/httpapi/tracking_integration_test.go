package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/analytics"
	"github.com/hkjang/umm/internal/auth"
	"github.com/hkjang/umm/internal/cryptoutil"
)

// An administrator turns visitor tracking on from the settings screen, and the
// page has to carry the snippet under a policy that stays strict — nonce on
// every tag, the snippet's origins allowed, what the browser still refuses
// written down where the administrator can see it. Off, which is the default,
// the page and its policy are exactly what they were.
func TestTrackingSnippetRunsUnderTheStrictPolicyAndOnlyWhenTurnedOnIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	adminID := uuid.New()
	username := "tracking_" + strings.ReplaceAll(adminID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name,role) VALUES($1,$2::citext,$2::text,'admin')`, adminID, username); err != nil {
		t.Fatal(err)
	}
	authService := &auth.Service{Store: db}
	session, err := authService.CreateSession(ctx, adminID, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	cipher, err := cryptoutil.New([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}

	// The collector Momento would be, answering with what it was asked for so
	// the test can see that the request arrived whole — and without the cookie.
	var collectorSawCookie, collectorSawPath string
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		collectorSawCookie = r.Header.Get("Cookie")
		collectorSawPath = r.URL.Path
		w.Header().Set("Content-Type", "text/javascript")
		_, _ = w.Write([]byte("window.momento=1"))
	}))
	t.Cleanup(collector.Close)

	webDir := t.TempDir()
	shell := `<!doctype html><html><head><meta name="csp-nonce" content="__CSP_NONCE__"></head><body><div id="root"></div><script type="module" src="/assets/index-AbCd1234.js"></script></body></html>`
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte(shell), 0o644); err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: db, Auth: authService, Cipher: cipher, OIDC: &auth.OIDCService{Store: db, Cipher: cipher, Sessions: authService}, WebDir: webDir}
	handler := server.Handler()

	do := func(method, target, body string, headers ...string) *httptest.ResponseRecorder {
		t.Helper()
		var reader *strings.Reader
		if body != "" {
			reader = strings.NewReader(body)
		} else {
			reader = strings.NewReader("")
		}
		request := httptest.NewRequest(method, target, reader)
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
		for i := 0; i+1 < len(headers); i += 2 {
			request.Header.Set(headers[i], headers[i+1])
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}
	save := func(setting map[string]any) *httptest.ResponseRecorder {
		t.Helper()
		raw, _ := json.Marshal(setting)
		return do(http.MethodPut, "/api/v1/admin/settings/analytics", string(raw), "Content-Type", "application/json")
	}
	nonceOf := regexp.MustCompile(`'nonce-([^']+)'`)
	page := func(path string) (body, policy, nonce string) {
		t.Helper()
		response := do(http.MethodGet, path, "")
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s = %d", path, response.Code)
		}
		policy = response.Header().Get("Content-Security-Policy")
		match := nonceOf.FindStringSubmatch(policy)
		if match == nil {
			t.Fatalf("no nonce in policy for %s: %s", path, policy)
		}
		return response.Body.String(), policy, match[1]
	}

	// Off — the seeded default — is a page with no snippet and a policy that
	// reports nothing and names nothing beyond umm.
	body, policy, _ := page("/today")
	if strings.Contains(body, "tracker.js") || strings.Contains(body, "/momento") {
		t.Fatalf("a snippet is served while tracking is off:\n%s", body)
	}
	if strings.Contains(policy, "report-uri") {
		t.Fatalf("the policy reports while tracking is off: %s", policy)
	}
	baseline := nonceOf.ReplaceAllString(policy, "'nonce-X'")
	if response := do(http.MethodGet, "/momento/tracker.js", ""); response.Code != http.StatusNotFound {
		t.Fatalf("the Momento proxy answers %d while tracking is off, want 404", response.Code)
	}

	// On, through the proxy: the snippet is in <head> with this response's
	// nonce, the policy asks for reports, and no outside origin appears in it.
	momento := map[string]any{
		"enabled": true, "provider": "momento", "momento_url": collector.URL, "momento_site_id": "umm-prd",
		"momento_proxy": true, "include_admin": false, "placement": "head",
	}
	if response := save(momento); response.Code != http.StatusOK {
		t.Fatalf("saving a complete Momento setting = %d: %s", response.Code, response.Body.String())
	}
	body, policy, nonce := page("/space/abc")
	head := body[:strings.Index(body, "</head>")]
	if !strings.Contains(head, `src="/momento/tracker.js"`) || !strings.Contains(head, `data-endpoint="/momento"`) {
		t.Fatalf("the Momento snippet is not in <head>:\n%s", body)
	}
	if !strings.Contains(head, `nonce="`+nonce+`"`) {
		t.Fatalf("the snippet does not carry the response nonce %s:\n%s", nonce, head)
	}
	if strings.Count(body, `nonce="`+nonce+`"`) != 2 {
		t.Fatalf("expected the bundle tag and the snippet to carry the nonce once each:\n%s", body)
	}
	if !strings.Contains(policy, "report-uri "+analytics.ReportPath) {
		t.Fatalf("the tracked page's policy does not ask for reports: %s", policy)
	}
	scriptSrc := policy[strings.Index(policy, "script-src "):]
	scriptSrc = scriptSrc[:strings.Index(scriptSrc, ";")]
	if strings.Contains(scriptSrc, "unsafe-inline") || !strings.Contains(scriptSrc, "'strict-dynamic'") {
		t.Fatalf("script-src must never be loosened with 'unsafe-inline': %s", policy)
	}
	if strings.Contains(policy, "127.0.0.1") {
		t.Fatalf("through the proxy the collector must not appear in the policy: %s", policy)
	}
	if strings.Contains(body, collector.URL) {
		t.Fatalf("through the proxy the collector must not appear in the page:\n%s", body)
	}

	// Not on the admin screens unless asked, and never on API answers.
	body, policy, _ = page("/admin/analytics")
	if strings.Contains(body, "tracker.js") || strings.Contains(policy, "report-uri") {
		t.Fatal("the admin screen is tracked without include_admin")
	}
	if nonceOf.ReplaceAllString(policy, "'nonce-X'") != baseline {
		t.Fatalf("an untracked page's policy changed:\n got %s\nwant %s", policy, baseline)
	}
	if response := do(http.MethodGet, "/api/v1/meta", ""); strings.Contains(response.Header().Get("Content-Security-Policy"), "report-uri") {
		t.Fatal("an API answer carries the tracked policy")
	}
	momento["include_admin"] = true
	if response := save(momento); response.Code != http.StatusOK {
		t.Fatalf("save = %d: %s", response.Code, response.Body.String())
	}
	if body, _, _ = page("/admin/analytics"); !strings.Contains(body, "tracker.js") {
		t.Fatal("include_admin does not put the snippet on the admin screen")
	}

	// The proxy forwards to the collector, without umm's session.
	response := do(http.MethodGet, "/momento/tracker.js", "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "window.momento=1") {
		t.Fatalf("proxy = %d %q", response.Code, response.Body.String())
	}
	if collectorSawPath != "/tracker.js" {
		t.Fatalf("the collector was asked for %q, want /tracker.js", collectorSawPath)
	}
	if collectorSawCookie != "" {
		t.Fatalf("umm's session cookie reached the collector: %q", collectorSawCookie)
	}
	if response := do(http.MethodDelete, "/momento/anything", ""); response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("DELETE through the proxy = %d, want 405", response.Code)
	}

	// A pasted snippet: its origins are read into the three directives, and
	// placement=body puts it before </body>.
	custom := map[string]any{
		"enabled": true, "provider": "custom", "placement": "body",
		"custom_snippet": `<script async src="https://cdn.tracker.example/loader.js"></script><script>fetch("https://collect.tracker.example/e")</script>`,
		"allowed_hosts":  "https://pixel.tracker.example",
	}
	if response := save(custom); response.Code != http.StatusOK {
		t.Fatalf("save = %d: %s", response.Code, response.Body.String())
	}
	body, policy, nonce = page("/today")
	tail := body[strings.Index(body, "</head>"):]
	if strings.Count(tail, `nonce="`+nonce+`"`) != 3 || strings.Contains(body[:strings.Index(body, "</head>")], "loader.js") {
		t.Fatalf("placement=body must put both snippet tags after </head>, each with the nonce:\n%s", body)
	}
	for _, directive := range []string{"script-src", "connect-src", "img-src"} {
		start := strings.Index(policy, directive+" ")
		end := strings.Index(policy[start:], ";")
		clause := policy[start : start+end]
		for _, origin := range []string{"https://cdn.tracker.example", "https://collect.tracker.example", "https://pixel.tracker.example"} {
			if !strings.Contains(clause, origin) {
				t.Fatalf("%s lacks %s: %s", directive, origin, policy)
			}
		}
	}

	// The browser reports what it refused; the administrator sees it, marked
	// by whether the setting already covers it, and can empty the list.
	report := `{"csp-report":{"document-uri":"https://umm.example/today?q=secret","blocked-uri":"https://fonts.tracker.example/x.woff2","effective-directive":"font-src","violated-directive":"font-src"}}`
	if response := do(http.MethodPost, analytics.ReportPath, report, "Content-Type", "application/csp-report"); response.Code != http.StatusNoContent {
		t.Fatalf("report = %d", response.Code)
	}
	known := `{"csp-report":{"document-uri":"https://umm.example/today","blocked-uri":"https://pixel.tracker.example/p.gif","effective-directive":"img-src"}}`
	_ = do(http.MethodPost, analytics.ReportPath, known, "Content-Type", "application/csp-report")
	var listed struct {
		Active     bool                  `json:"active"`
		Violations []analytics.Violation `json:"violations"`
	}
	if err := json.Unmarshal(do(http.MethodGet, "/api/v1/admin/analytics/violations", "").Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if !listed.Active || len(listed.Violations) != 2 {
		t.Fatalf("violations = %+v", listed)
	}
	byOrigin := map[string]analytics.Violation{}
	for _, violation := range listed.Violations {
		byOrigin[violation.Origin] = violation
	}
	if got := byOrigin["https://fonts.tracker.example"]; got.Directive != "font-src" || got.Allowed || got.Page != "/today" {
		t.Fatalf("the unknown origin is recorded wrongly, or with the page's query string: %+v", got)
	}
	if got := byOrigin["https://pixel.tracker.example"]; !got.Allowed {
		t.Fatalf("an origin the setting already allows must be marked so: %+v", got)
	}
	if response := do(http.MethodDelete, "/api/v1/admin/analytics/violations", ""); response.Code != http.StatusOK {
		t.Fatalf("forget = %d", response.Code)
	}
	if err := json.Unmarshal(do(http.MethodGet, "/api/v1/admin/analytics/violations", "").Body.Bytes(), &listed); err != nil || len(listed.Violations) != 0 {
		t.Fatalf("the list survived Forget: %+v (%v)", listed, err)
	}

	// What cannot be saved: a snippet over the limit, and one that would
	// render nothing.
	tooLong := map[string]any{"enabled": true, "provider": "custom", "custom_snippet": "<script>" + strings.Repeat("x", analytics.MaxSnippetBytes) + "</script>"}
	if response := save(tooLong); response.Code != http.StatusBadRequest {
		t.Fatalf("a snippet over %d bytes was saved: %d", analytics.MaxSnippetBytes, response.Code)
	}
	if response := save(map[string]any{"enabled": true, "provider": "momento", "momento_url": collector.URL}); response.Code != http.StatusBadRequest {
		t.Fatalf("Momento without a site id was saved: %d", response.Code)
	}

	// Off again: the page and the policy return to exactly what they were,
	// reports are no longer kept, and the proxy is closed.
	if response := save(map[string]any{"enabled": false, "provider": "momento", "momento_url": collector.URL, "momento_site_id": "umm-prd", "momento_proxy": true}); response.Code != http.StatusOK {
		t.Fatalf("save = %d: %s", response.Code, response.Body.String())
	}
	body, policy, _ = page("/today")
	if strings.Contains(body, "tracker") || nonceOf.ReplaceAllString(policy, "'nonce-X'") != baseline {
		t.Fatalf("turning tracking off did not restore the page and policy:\n%s\n%s", policy, body)
	}
	_ = do(http.MethodPost, analytics.ReportPath, report, "Content-Type", "application/csp-report")
	if err := json.Unmarshal(do(http.MethodGet, "/api/v1/admin/analytics/violations", "").Body.Bytes(), &listed); err != nil || listed.Active || len(listed.Violations) != 0 {
		t.Fatalf("reports are kept while tracking is off: %+v (%v)", listed, err)
	}
	if response := do(http.MethodGet, "/momento/tracker.js", ""); response.Code != http.StatusNotFound {
		t.Fatalf("the proxy stayed open after tracking went off: %d", response.Code)
	}
}
