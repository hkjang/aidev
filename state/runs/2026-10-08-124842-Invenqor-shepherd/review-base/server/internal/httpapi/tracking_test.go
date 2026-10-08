package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/hkjang/invenqor/server/internal/storagetest"
)

var scriptNonce = regexp.MustCompile(`'nonce-([^']+)'`)

func consolePage(t *testing.T, server *Server, path string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d body = %s", path, response.Code, response.Body.String())
	}
	return response
}

func patchTracking(
	t *testing.T, server *Server, cookie *http.Cookie, csrf string, body map[string]any,
) *httptest.ResponseRecorder {
	t.Helper()
	return performAuthenticatedJSON(
		t, server, http.MethodPatch, "/api/v1/admin/settings/tracking",
		body, cookie, csrf,
	)
}

// A fresh installation must be indistinguishable from one that never had
// tracking: no snippet, the policy the console has always shipped with, and
// nothing written to the database.
func TestTrackingIsOffByDefault(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)

	page := consolePage(t, server, "/")
	if got := page.Header().Get("Content-Security-Policy"); got != basePagePolicy {
		t.Fatalf("default page policy = %q, want %q", got, basePagePolicy)
	}
	if strings.Contains(page.Body.String(), "nonce=") ||
		strings.Contains(page.Body.String(), "tracker.js") {
		t.Fatalf("default page carries a snippet:\n%s", page.Body.String())
	}
	var count int
	if err := runtime.DB().QueryRow(
		`SELECT COUNT(*) FROM server_metadata WHERE key=$1`, trackingPolicyKey,
	).Scan(&count); err != nil || count != 0 {
		t.Fatalf("tracking policy rows = %d, err = %v; a read must not write", count, err)
	}

	api := httptest.NewRequest(http.MethodGet, "/api/v1/system/info", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, api)
	if got := response.Header().Get("Content-Security-Policy"); got != apiPolicy {
		t.Fatalf("API policy = %q, want %q", got, apiPolicy)
	}

	proxied := httptest.NewRequest(http.MethodGet, "/momento/tracker.js", nil)
	response = httptest.NewRecorder()
	server.Handler().ServeHTTP(response, proxied)
	if response.Code != http.StatusNotFound {
		t.Fatalf("proxy while off status = %d, want 404", response.Code)
	}
}

func TestMomentoThroughTheSameOriginProxy(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	var upstreamPath, upstreamCookie string
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamPath = r.URL.Path
		upstreamCookie = r.Header.Get("Cookie")
		w.Header().Set("Set-Cookie", "momento=1")
		w.Header().Set("Content-Type", "application/javascript")
		_, _ = w.Write([]byte("window.__momento=1;"))
	}))
	defer collector.Close()

	enabled := patchTracking(t, server, cookie, csrf, map[string]any{
		"enabled": true, "provider": "momento",
		"momento_url": collector.URL + "/", "momento_site_id": "invenqor",
		"placement": "body", "reason": "pilot",
	})
	if enabled.Code != http.StatusOK {
		t.Fatalf("enable status = %d body = %s", enabled.Code, enabled.Body.String())
	}
	var policy struct {
		Active        bool     `json:"active"`
		MomentoProxy  bool     `json:"momento_proxy"`
		PolicySources []string `json:"policy_sources"`
		Version       int      `json:"version"`
	}
	if err := json.Unmarshal(enabled.Body.Bytes(), &policy); err != nil {
		t.Fatal(err)
	}
	if !policy.Active || !policy.MomentoProxy || len(policy.PolicySources) != 0 || policy.Version != 1 {
		t.Fatalf("policy after enable = %+v", policy)
	}

	// Another pod reads the same database and serves the same snippet.
	otherPod := testServer(t, runtime)
	page := consolePage(t, otherPod, "/")
	body := page.Body.String()
	header := page.Header().Get("Content-Security-Policy")
	match := scriptNonce.FindStringSubmatch(header)
	if match == nil {
		t.Fatalf("policy has no nonce: %q", header)
	}
	nonce := match[1]
	if !strings.Contains(body, `src="/momento/tracker.js"`) ||
		!strings.Contains(body, `data-endpoint="/momento"`) ||
		!strings.Contains(body, `nonce="`+nonce+`"`) {
		t.Fatalf("page lacks the proxied snippet with nonce %s:\n%s", nonce, body)
	}
	if index := strings.Index(body, "tracker.js"); index < strings.Index(body, "</head>") ||
		index > strings.LastIndex(body, "</body>") {
		t.Fatalf("placement=body put the snippet elsewhere:\n%s", body)
	}
	if strings.Contains(header, collector.URL) || strings.Contains(header, "unsafe-inline") {
		t.Fatalf("proxied Momento widened the policy: %q", header)
	}
	if !strings.HasPrefix(header, basePagePolicy+"; script-src 'self' 'nonce-") ||
		!strings.Contains(header, "; report-uri "+cspReportPath) {
		t.Fatalf("policy = %q", header)
	}
	if again := scriptNonce.FindStringSubmatch(
		consolePage(t, otherPod, "/").Header().Get("Content-Security-Policy"),
	); again == nil || again[1] == nonce {
		t.Fatal("the nonce did not change between requests")
	}
	if direct := consolePage(t, otherPod, "/index.html"); !strings.Contains(direct.Body.String(), "tracker.js") {
		t.Fatal("/index.html served the shell without the snippet")
	}

	// The proxy forwards to the collector without this Server's session.
	proxied := httptest.NewRequest(http.MethodGet, "/momento/tracker.js", nil)
	proxied.AddCookie(cookie)
	response := httptest.NewRecorder()
	otherPod.Handler().ServeHTTP(response, proxied)
	if response.Code != http.StatusOK || response.Body.String() != "window.__momento=1;" {
		t.Fatalf("proxy status = %d body = %s", response.Code, response.Body.String())
	}
	if upstreamPath != "/tracker.js" || upstreamCookie != "" {
		t.Fatalf("collector saw path %q cookie %q", upstreamPath, upstreamCookie)
	}
	if response.Header().Get("Set-Cookie") != "" {
		t.Fatal("the collector's cookie reached the browser")
	}
	if got := response.Header().Get("Content-Security-Policy"); got != apiPolicy {
		t.Fatalf("proxy response policy = %q", got)
	}

	// Switching off restores the original policy on every pod.
	disabled := patchTracking(t, server, cookie, csrf, map[string]any{"enabled": false})
	if disabled.Code != http.StatusOK {
		t.Fatalf("disable status = %d body = %s", disabled.Code, disabled.Body.String())
	}
	page = consolePage(t, otherPod, "/")
	if got := page.Header().Get("Content-Security-Policy"); got != basePagePolicy {
		t.Fatalf("policy after disable = %q", got)
	}
	if strings.Contains(page.Body.String(), "tracker.js") {
		t.Fatal("snippet survived disabling")
	}
	response = httptest.NewRecorder()
	otherPod.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/momento/tracker.js", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("proxy after disable status = %d", response.Code)
	}
}

func TestPastedSnippetOriginsEnterThePolicy(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	snippet := `<script src="https://t.corp.example/t.js"></script>
<script>window.__t={endpoint:"https://collect.corp.example/v1/events"};</script>`
	enabled := patchTracking(t, server, cookie, csrf, map[string]any{
		"enabled": true, "provider": "custom", "custom_snippet": snippet,
	})
	if enabled.Code != http.StatusOK {
		t.Fatalf("enable status = %d body = %s", enabled.Code, enabled.Body.String())
	}
	page := consolePage(t, server, "/")
	header := page.Header().Get("Content-Security-Policy")
	nonce := scriptNonce.FindStringSubmatch(header)
	if nonce == nil {
		t.Fatalf("no nonce in %q", header)
	}
	for _, directive := range []string{"script-src", "connect-src", "img-src"} {
		if !regexp.MustCompile(directive + ` [^;]*https://t\.corp\.example[^;]*https://collect\.corp\.example`).MatchString(header) {
			t.Errorf("%s lacks the snippet origins: %q", directive, header)
		}
	}
	if got := strings.Count(page.Body.String(), `nonce="`+nonce[1]+`"`); got != 2 {
		t.Fatalf("nonce on %d script tags, want 2:\n%s", got, page.Body.String())
	}
	if index := strings.Index(page.Body.String(), "t.corp.example"); index > strings.Index(page.Body.String(), "</head>") {
		t.Fatal("default placement is head")
	}

	for name, body := range map[string]map[string]any{
		"oversized": {"custom_snippet": strings.Repeat("<script></script>", 600)},
		"keyword":   {"allowed_hosts": []string{"'unsafe-inline'"}},
		"bare host": {"allowed_hosts": []string{"t.corp.example"}},
		"provider":  {"provider": "piwik"},
		"unknown":   {"unsafe_inline": true},
	} {
		refused := patchTracking(t, server, cookie, csrf, body)
		if refused.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d body = %s", name, refused.Code, refused.Body.String())
		}
	}
	if header := consolePage(t, server, "/").Header().Get("Content-Security-Policy"); strings.Contains(header, "unsafe-inline") {
		t.Fatalf("policy contains unsafe-inline: %q", header)
	}
}

func TestBlockedOriginsAreRecordedAndAllowedInOneClick(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	report := func(blocked, directive string) {
		t.Helper()
		body := `{"csp-report":{"blocked-uri":"` + blocked + `","effective-directive":"` + directive + `","document-uri":"https://invenqor.corp.example/"}}`
		request := httptest.NewRequest(http.MethodPost, cspReportPath, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/csp-report")
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, request)
		if response.Code != http.StatusNoContent {
			t.Fatalf("report status = %d", response.Code)
		}
	}
	for range 3 {
		report("https://momento.corp.example/collect/v1/events", "connect-src")
	}
	report("https://momento.corp.example/tracker.js", "script-src")
	report("chrome-extension://abc/x.js", "script-src")
	report("garbage{", "")

	listed := performAuthenticatedJSON(
		t, server, http.MethodGet, "/api/v1/admin/settings/tracking/violations",
		nil, cookie, csrf,
	)
	if listed.Code != http.StatusOK {
		t.Fatalf("list status = %d body = %s", listed.Code, listed.Body.String())
	}
	var list struct {
		Items []struct {
			Origin    string `json:"origin"`
			Directive string `json:"directive"`
			Count     int    `json:"count"`
			Allowed   bool   `json:"allowed"`
		} `json:"items"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 2 {
		t.Fatalf("items = %+v, want the two http directives only", list.Items)
	}
	for _, item := range list.Items {
		if item.Origin != "https://momento.corp.example" || item.Allowed {
			t.Fatalf("item = %+v", item)
		}
		if item.Directive == "connect-src" && item.Count != 3 {
			t.Fatalf("repeated report accumulated as %+v", item)
		}
	}

	allowed := performAuthenticatedJSON(
		t, server, http.MethodPost, "/api/v1/admin/settings/tracking/allowed-hosts",
		map[string]string{"origin": "https://momento.corp.example/"}, cookie, csrf,
	)
	if allowed.Code != http.StatusOK || !strings.Contains(allowed.Body.String(), `"allowed_hosts":["https://momento.corp.example"]`) {
		t.Fatalf("allow status = %d body = %s", allowed.Code, allowed.Body.String())
	}
	listed = performAuthenticatedJSON(
		t, server, http.MethodGet, "/api/v1/admin/settings/tracking/violations",
		nil, cookie, csrf,
	)
	if err := json.Unmarshal(listed.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	for _, item := range list.Items {
		if !item.Allowed {
			t.Fatalf("allowed origin still reported as blocked: %+v", item)
		}
	}

	cleared := performAuthenticatedJSON(
		t, server, http.MethodDelete, "/api/v1/admin/settings/tracking/violations",
		nil, cookie, csrf,
	)
	if cleared.Code != http.StatusNoContent {
		t.Fatalf("clear status = %d", cleared.Code)
	}
	listed = performAuthenticatedJSON(
		t, server, http.MethodGet, "/api/v1/admin/settings/tracking/violations",
		nil, cookie, csrf,
	)
	if !strings.Contains(listed.Body.String(), `"items":[]`) {
		t.Fatalf("list after clear = %s", listed.Body.String())
	}

	// The report endpoint needs no session and never errors, even on junk.
	junk := httptest.NewRequest(http.MethodPost, cspReportPath, strings.NewReader(strings.Repeat("x", 20000)))
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, junk)
	if response.Code != http.StatusNoContent {
		t.Fatalf("junk report status = %d", response.Code)
	}
}
