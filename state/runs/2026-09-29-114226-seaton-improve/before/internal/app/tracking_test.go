package app

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/hkjang/seaton/internal/tracking"
)

// 추적 스니펫은 정책(CSP)과 한 몸이다. 꺼져 있으면 문서와 정책이 글자 그대로
// 예전과 같아야 하고, 켜면 그 요청의 nonce 가 스니펫과 정책 양쪽에 같은 값으로
// 들어가야 한다. 데이터베이스 없이 설정을 고정해 확인한다.

const testDocument = `<!doctype html><html><head><title>SeatOn</title></head><body><div id="root"></div></body></html>`

func trackingServer(t *testing.T, values map[string]string) *Server {
	t.Helper()
	s := NewServer(nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), fstest.MapFS{"index.html": {Data: []byte(testDocument)}}, "test", "test", "test")
	config := tracking.ReadConfig(values)
	s.trackingConfig = func(context.Context) tracking.Config { return config }
	return s
}

func momentoValues(extra map[string]string) map[string]string {
	values := map[string]string{"tracking.enabled": "true", "tracking.provider": "momento", "tracking.momento_url": "https://momento.intra", "tracking.momento_site_id": "SITE_1"}
	for key, value := range extra {
		values[key] = value
	}
	return values
}

func servePath(s *Server, path string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	s.securityHeaders(s.spaHandler()).ServeHTTP(recorder, request)
	return recorder
}

func TestPolicyIsUnchangedWhenTrackingOff(t *testing.T) {
	// 이 문자열은 기능을 넣기 전 서버가 내보내던 정책 그대로다. 꺼져 있을 때
	// 여기서 한 글자라도 달라지면 "새로 설치한 곳은 아무것도 달라지지 않는다"
	// 가 깨진 것이다.
	const before = "default-src 'self'; img-src 'self' data: blob:; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'"
	if basePolicy != before {
		t.Fatalf("기본 정책이 바뀌었다: %s", basePolicy)
	}
	for _, values := range []map[string]string{{}, momentoValues(map[string]string{"tracking.enabled": "false"}), {"tracking.enabled": "true", "tracking.provider": "none"}} {
		s := trackingServer(t, values)
		response := servePath(s, "/")
		if got := response.Header().Get("Content-Security-Policy"); got != before {
			t.Fatalf("꺼져 있으면 정책이 원래대로여야 한다: %s", got)
		}
		if response.Body.String() != testDocument {
			t.Fatalf("꺼져 있으면 문서에 아무것도 붙지 않아야 한다: %s", response.Body.String())
		}
	}
}

func TestPageCarriesNonceInSnippetAndPolicy(t *testing.T) {
	s := trackingServer(t, momentoValues(nil))
	response := servePath(s, "/")
	policy := response.Header().Get("Content-Security-Policy")
	body := response.Body.String()
	var scriptSrc string
	for _, directive := range strings.Split(policy, "; ") {
		if strings.HasPrefix(directive, "script-src ") {
			scriptSrc = directive
		}
	}
	if strings.Contains(scriptSrc, "'unsafe-inline'") {
		t.Fatalf("script-src 에 'unsafe-inline' 이 들어가면 안 된다: %s", policy)
	}
	start := strings.Index(scriptSrc, "'nonce-")
	if start < 0 {
		t.Fatalf("script-src 에 nonce 가 없다: %s", policy)
	}
	nonce := scriptSrc[start+len("'nonce-"):]
	nonce = nonce[:strings.IndexByte(nonce, '\'')]
	if nonce == "" || !strings.Contains(body, `nonce="`+nonce+`"`) {
		t.Fatalf("스니펫의 nonce 가 정책의 것과 같아야 한다: policy=%s body=%s", policy, body)
	}
	if !strings.Contains(body, `src="/momento/tracker.js"`) || !strings.HasSuffix(strings.TrimSpace(body[:strings.Index(body, "</head>")]), "</script>") {
		t.Fatalf("Momento 프록시 스니펫이 </head> 앞에 있어야 한다: %s", body)
	}
	if !strings.Contains(policy, "report-uri "+cspReportPath) {
		t.Fatalf("켜져 있으면 report-uri 가 있어야 한다: %s", policy)
	}
	if strings.Contains(policy, "momento.intra") {
		t.Fatalf("프록시 구성이면 외부 출처가 정책에 나오지 않아야 한다: %s", policy)
	}
	// 요청마다 다른 nonce.
	if servePath(s, "/").Header().Get("Content-Security-Policy") == policy {
		t.Fatal("nonce 는 요청마다 달라야 한다")
	}
}

func TestPolicyAddsSnippetOriginsAndBodyPlacement(t *testing.T) {
	s := trackingServer(t, map[string]string{"tracking.enabled": "true", "tracking.provider": "custom", "tracking.placement": "body",
		"tracking.custom_snippet": `<script src="https://t.intra/t.js"></script>`, "tracking.allowed_hosts": "https://pixel.intra"})
	response := servePath(s, "/login")
	policy := response.Header().Get("Content-Security-Policy")
	for _, directive := range []string{"script-src 'self' 'nonce-", "connect-src 'self' https://t.intra https://pixel.intra", "img-src 'self' data: blob: https://t.intra https://pixel.intra"} {
		if !strings.Contains(policy, directive) {
			t.Fatalf("정책에 %q 가 있어야 한다: %s", directive, policy)
		}
	}
	if !strings.Contains(policy, "https://t.intra https://pixel.intra; connect-src") {
		t.Fatalf("script-src 에 스니펫 출처와 allowed_hosts 가 있어야 한다: %s", policy)
	}
	body := response.Body.String()
	if !strings.Contains(body, `</div><script nonce=`) || !strings.Contains(body, "</script>\n</body>") {
		t.Fatalf("body 배치는 </body> 앞이어야 한다: %s", body)
	}
}

func TestAdminAndAPIPathsGetNoSnippet(t *testing.T) {
	s := trackingServer(t, momentoValues(nil))
	for _, path := range []string{"/admin/settings", "/profile/keys"} {
		response := servePath(s, path)
		if response.Header().Get("Content-Security-Policy") != basePolicy || strings.Contains(response.Body.String(), "tracker.js") {
			t.Fatalf("include_admin 이 꺼져 있으면 %s 에 붙지 않아야 한다", path)
		}
	}
	response := servePath(trackingServer(t, momentoValues(map[string]string{"tracking.include_admin": "true"})), "/admin/settings")
	if !strings.Contains(response.Body.String(), "tracker.js") {
		t.Fatal("include_admin 을 켜면 관리 화면에도 붙어야 한다")
	}
	// API 경로는 SPA 문서가 아니므로 스니펫도 nonce 도 없다.
	response = servePath(s, "/api/v1/nothing")
	if response.Code != http.StatusNotFound || response.Header().Get("Content-Security-Policy") != basePolicy {
		t.Fatalf("API 경로는 기본 정책이어야 한다: %d %s", response.Code, response.Header().Get("Content-Security-Policy"))
	}
}

func TestCSPReportIsRecordedOnlyWhileTrackingOn(t *testing.T) {
	report := `{"csp-report":{"blocked-uri":"https://momento.intra/collect/v1/events","effective-directive":"connect-src","document-uri":"https://seaton.intra/?q=1"}}`
	off := trackingServer(t, map[string]string{})
	recorder := httptest.NewRecorder()
	off.cspReport(recorder, httptest.NewRequest(http.MethodPost, cspReportPath, strings.NewReader(report)))
	if recorder.Code != http.StatusNoContent || len(off.violations.List(tracking.Config{})) != 0 {
		t.Fatal("꺼져 있으면 신고를 받아도 기록하지 않아야 한다")
	}

	on := trackingServer(t, momentoValues(map[string]string{"tracking.momento_proxy": "false"}))
	for i := 0; i < 3; i++ {
		recorder = httptest.NewRecorder()
		on.cspReport(recorder, httptest.NewRequest(http.MethodPost, cspReportPath, strings.NewReader(report)))
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("신고는 204 여야 한다: %d %s", recorder.Code, recorder.Body.String())
		}
	}
	recorder = httptest.NewRecorder()
	on.listTrackingViolations(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/settings/tracking/violations", nil))
	var listed struct{ Items []tracking.Violation }
	if err := json.NewDecoder(recorder.Body).Decode(&listed); err != nil {
		t.Fatalf("목록을 읽지 못함: %v", err)
	}
	if len(listed.Items) != 1 || listed.Items[0].Origin != "https://momento.intra" || listed.Items[0].Count != 3 || listed.Items[0].Page != "/" || listed.Items[0].Directive != "connect-src" {
		t.Fatalf("같은 출처는 한 줄로 모여야 한다: %+v", listed.Items)
	}
	if !listed.Items[0].Allowed {
		t.Fatal("직접 연결 구성은 수집기 출처를 이미 허용하므로 allowed 여야 한다")
	}
	recorder = httptest.NewRecorder()
	on.cspReport(recorder, httptest.NewRequest(http.MethodPost, cspReportPath, strings.NewReader("not json")))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("깨진 신고는 400: %d", recorder.Code)
	}
	recorder = httptest.NewRecorder()
	on.forgetTrackingViolations(recorder, httptest.NewRequest(http.MethodDelete, "/api/v1/settings/tracking/violations", nil))
	if recorder.Code != http.StatusNoContent || len(on.violations.List(tracking.Config{})) != 0 {
		t.Fatal("비우기 뒤에는 목록이 비어야 한다")
	}
}

func TestMomentoProxyForwardsLoaderAndCollectOnly(t *testing.T) {
	type seen struct {
		method, path, cookie, authorization, forwardedFor, host string
	}
	var last seen
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		last = seen{r.Method, r.URL.Path, r.Header.Get("Cookie"), r.Header.Get("Authorization"), r.Header.Get("X-Forwarded-For"), r.Host}
		w.Header().Set("Set-Cookie", "momento=1")
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		if r.URL.Path == "/base/tracker.js" {
			w.Header().Set("Content-Type", "application/javascript")
			_, _ = w.Write([]byte("window.momento=1"))
			return
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"accepted":1}`))
	}))
	defer collector.Close()

	s := trackingServer(t, momentoValues(map[string]string{"tracking.momento_url": collector.URL + "/base/"}))
	handler := s.Routes()

	request := httptest.NewRequest(http.MethodGet, "/momento/tracker.js", nil)
	request.Header.Set("Cookie", "seaton_session=secret")
	request.Header.Set("Authorization", "Bearer key")
	request.RemoteAddr = "10.1.2.3:4444"
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Body.String() != "window.momento=1" {
		t.Fatalf("tracker.js 가 프록시되어야 한다: %d %s", recorder.Code, recorder.Body.String())
	}
	if last.path != "/base/tracker.js" || last.cookie != "" || last.authorization != "" || last.forwardedFor != "10.1.2.3" || last.host != strings.TrimPrefix(collector.URL, "http://") {
		t.Fatalf("수집기에는 쿠키·인증 없이 원래 IP 와 함께 가야 한다: %+v", last)
	}
	if recorder.Header().Get("Set-Cookie") != "" || recorder.Header().Get("Content-Security-Policy") != basePolicy {
		t.Fatalf("수집기의 쿠키와 정책은 걸러야 한다: %v", recorder.Header())
	}

	request = httptest.NewRequest(http.MethodPost, "/momento/collect/v1/events", strings.NewReader(`{"events":[]}`))
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusAccepted || last.method != http.MethodPost || last.path != "/base/collect/v1/events" {
		t.Fatalf("수집 요청이 프록시되어야 한다: %d %+v", recorder.Code, last)
	}

	// 그 밖의 경로와 메서드는 열어 주지 않는다 — 이 앱이 수집기 콘솔의 통로가 되면 안 된다.
	for _, probe := range []struct{ method, path string }{{http.MethodGet, "/momento/admin"}, {http.MethodGet, "/momento/collect/v1/events"}, {http.MethodPost, "/momento/tracker.js"}, {http.MethodGet, "/momento/"}} {
		recorder = httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(probe.method, probe.path, nil))
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("%s %s 는 404 여야 한다: %d", probe.method, probe.path, recorder.Code)
		}
	}

	// 프록시를 끄거나 추적을 끄면 경로 자체가 없다.
	for _, values := range []map[string]string{momentoValues(map[string]string{"tracking.momento_proxy": "false"}), {}} {
		recorder = httptest.NewRecorder()
		trackingServer(t, values).Routes().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/momento/tracker.js", nil))
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("프록시 구성이 아니면 404 여야 한다: %d", recorder.Code)
		}
	}
}
