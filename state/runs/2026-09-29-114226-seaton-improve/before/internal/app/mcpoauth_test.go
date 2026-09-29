package app

import (
	"bytes"
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"
)

// MCP SSO 의 어려운 쪽은 "이 토큰이 이 서버를 위한 것인가" 다. 가짜 IdP 로 실제
// 키 쌍을 만들어 JWT 를 서명하고 discovery·JWKS 를 서빙해, 검증기가 실제로
// 서명·발급자·만료·대상을 보는지 확인한다. 계정 조회(데이터베이스)는 e2e
// (web/e2e/mcp-oauth.spec.ts)가 실서버로 본다.

type fakeIdP struct {
	*httptest.Server
	key *rsa.PrivateKey
}

func newFakeIdP(t *testing.T) *fakeIdP {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	idp := &fakeIdP{key: key}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"issuer": idp.URL, "jwks_uri": idp.URL + "/jwks", "authorization_endpoint": idp.URL + "/auth", "token_endpoint": idp.URL + "/token", "id_token_signing_alg_values_supported": []string{"RS256"}})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		n := base64.RawURLEncoding.EncodeToString(key.N.Bytes())
		e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())
		writeJSON(w, 200, map[string]any{"keys": []map[string]string{{"kty": "RSA", "kid": "k1", "use": "sig", "alg": "RS256", "n": n, "e": e}}})
	})
	idp.Server = httptest.NewServer(mux)
	t.Cleanup(idp.Close)
	return idp
}

func segment(v any) string {
	b, _ := json.Marshal(v)
	return base64.RawURLEncoding.EncodeToString(b)
}

// token 은 Keycloak 26 이 발급하는 액세스 토큰의 모양을 흉내낸다: typ=Bearer,
// aud=["account"], 발급받은 클라이언트는 azp 에. 호출자가 claims 로 덮어쓴다.
func (idp *fakeIdP) token(t *testing.T, claims map[string]any) string {
	t.Helper()
	now := time.Now()
	body := map[string]any{"iss": idp.URL, "sub": "sub-1", "aud": []string{"account"}, "azp": "claude-mcp", "typ": "Bearer", "iat": now.Unix(), "exp": now.Add(5 * time.Minute).Unix(), "preferred_username": "hong", "scope": "openid profile email"}
	for k, v := range claims {
		if v == nil {
			delete(body, k)
		} else {
			body[k] = v
		}
	}
	signing := segment(map[string]string{"alg": "RS256", "typ": "JWT", "kid": "k1"}) + "." + segment(body)
	digest := sha256.Sum256([]byte(signing))
	sig, err := rsa.SignPKCS1v15(rand.Reader, idp.key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return signing + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// hsToken 은 같은 내용을 HS256 으로 서명한 것 — 공개 키로 검증할 수 없다.
func (idp *fakeIdP) hsToken(claims map[string]any) string {
	now := time.Now()
	body := map[string]any{"iss": idp.URL, "sub": "sub-1", "aud": []string{"account"}, "azp": "claude-mcp", "typ": "Bearer", "exp": now.Add(5 * time.Minute).Unix()}
	for k, v := range claims {
		body[k] = v
	}
	signing := segment(map[string]string{"alg": "HS256", "typ": "JWT"}) + "." + segment(body)
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte(signing))
	return signing + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func oauthServer(t *testing.T, values map[string]string) (*Server, *bytes.Buffer) {
	t.Helper()
	logs := &bytes.Buffer{}
	s := NewServer(nil, nil, slog.New(slog.NewTextHandler(logs, nil)), fstest.MapFS{"index.html": {Data: []byte(testDocument)}}, "test", "test", "test")
	cfg := readMCPOAuthConfig(values)
	s.mcpOAuthConfig = func(context.Context) mcpOAuthConfig { return cfg }
	return s, logs
}

func enabledValues(idp *fakeIdP, extra map[string]string) map[string]string {
	values := map[string]string{"mcp.oauth.enabled": "true", "oidc.issuer_url": idp.URL + "/", "mcp.oauth.scopes": "read mcp", "general.service_name": "SeatOn"}
	for k, v := range extra {
		values[k] = v
	}
	return values
}

// routed 는 실제 라우터를 지나 GET 을 보낸다.
func routed(s *Server, path string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	s.Routes().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "https://seaton.intra"+path, nil))
	return recorder
}

func mcpCall(s *Server, bearer string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "https://seaton.intra/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
	recorder := httptest.NewRecorder()
	s.Routes().ServeHTTP(recorder, request)
	return recorder
}

func TestProtectedResourceMetadata(t *testing.T) {
	off, _ := oauthServer(t, map[string]string{"oidc.issuer_url": "https://keycloak.intra/realms/company"})
	for _, path := range []string{"/.well-known/oauth-protected-resource", "/.well-known/oauth-protected-resource/mcp"} {
		if got := routed(off, path); got.Code != http.StatusNotFound {
			t.Fatalf("꺼져 있으면 %s 는 404 여야 한다: %d %s", path, got.Code, got.Body.String())
		}
	}
	// 스위치만 켜고 issuer 가 없으면 꺼진 것처럼 동작한다.
	half, _ := oauthServer(t, map[string]string{"mcp.oauth.enabled": "true", "mcp.oauth.resource": "https://seaton.intra/mcp"})
	if got := routed(half, "/.well-known/oauth-protected-resource/mcp"); got.Code != http.StatusNotFound {
		t.Fatalf("issuer 없이는 404 여야 한다: %d", got.Code)
	}
	// 리소스 식별자가 없어도 꺼진 것이다 — 요청의 Host 로 만들어 주지 않는다.
	noResource, _ := oauthServer(t, map[string]string{"mcp.oauth.enabled": "true", "oidc.issuer_url": "https://keycloak.intra/realms/company", "mcp.oauth.scopes": "read mcp"})
	if got := routed(noResource, "/.well-known/oauth-protected-resource/mcp"); got.Code != http.StatusNotFound {
		t.Fatalf("리소스 식별자 없이는 404 여야 한다: %d %s", got.Code, got.Body.String())
	}

	on, _ := oauthServer(t, map[string]string{"mcp.oauth.enabled": "true", "oidc.issuer_url": "https://keycloak.intra/realms/company/", "mcp.oauth.resource": "https://seaton.intra/mcp", "mcp.oauth.scopes": "read mcp", "general.service_name": "좌석"})
	for _, path := range []string{"/.well-known/oauth-protected-resource", "/.well-known/oauth-protected-resource/mcp"} {
		// Host 와 프록시 헤더가 무엇이든 resource 는 설정값이다.
		request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8080"+path, nil)
		request.Header.Set("X-Forwarded-Host", "other-app.intra")
		request.Header.Set("X-Forwarded-Proto", "https")
		recorder := httptest.NewRecorder()
		on.Routes().ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", path, recorder.Code, recorder.Body.String())
		}
		if recorder.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Fatalf("메타데이터는 CORS 로 열려야 한다")
		}
		var doc map[string]any
		if err := json.Unmarshal(recorder.Body.Bytes(), &doc); err != nil {
			t.Fatal(err)
		}
		if _, wrapped := doc["error"]; wrapped || doc["data"] != nil {
			t.Fatalf("맨 JSON 이어야 한다: %s", recorder.Body.String())
		}
		if doc["resource"] != "https://seaton.intra/mcp" {
			t.Fatalf("resource = %v", doc["resource"])
		}
		servers, _ := doc["authorization_servers"].([]any)
		if len(servers) != 1 || servers[0] != "https://keycloak.intra/realms/company" {
			t.Fatalf("authorization_servers = %v (끝 / 는 떼야 한다)", doc["authorization_servers"])
		}
		if doc["resource_name"] != "좌석 MCP" {
			t.Fatalf("resource_name = %v", doc["resource_name"])
		}
		methods, _ := doc["bearer_methods_supported"].([]any)
		scopes, _ := doc["scopes_supported"].([]any)
		if len(methods) != 1 || methods[0] != "header" || len(scopes) != 2 {
			t.Fatalf("bearer_methods_supported=%v scopes_supported=%v", methods, scopes)
		}
	}

	// 요청 주소와 다른 리소스 식별자를 적어 두면 그 값이다.
	fixed, _ := oauthServer(t, map[string]string{"mcp.oauth.enabled": "true", "oidc.issuer_url": "https://keycloak.intra/realms/company", "mcp.oauth.resource": "https://seats.example.com/mcp"})
	got := routed(fixed, "/.well-known/oauth-protected-resource/mcp")
	if !strings.Contains(got.Body.String(), `"resource":"https://seats.example.com/mcp"`) {
		t.Fatalf("설정한 리소스 식별자를 내야 한다: %s", got.Body.String())
	}
}

func TestChallengeHeaderOnlyOnMCPPath(t *testing.T) {
	on, _ := oauthServer(t, map[string]string{"mcp.oauth.enabled": "true", "oidc.issuer_url": "https://keycloak.intra/realms/company", "mcp.oauth.resource": "https://seaton.intra/mcp"})
	got := mcpCall(on, "")
	if got.Code != http.StatusUnauthorized {
		t.Fatalf("토큰 없는 /mcp 는 401: %d", got.Code)
	}
	want := `Bearer realm="SeatOn", resource_metadata="https://seaton.intra/.well-known/oauth-protected-resource/mcp"`
	if h := got.Header().Get("WWW-Authenticate"); h != want {
		t.Fatalf("WWW-Authenticate = %q, want %q", h, want)
	}
	// 키가 아니고 JWT 도 아닌 값은 전과 같은 401 이다 — 새로운 말을 흘리지 않는다.
	if got := mcpCall(on, "not-a-key"); got.Code != http.StatusUnauthorized || !strings.Contains(got.Body.String(), "authentication_required") {
		t.Fatalf("정체불명 bearer: %d %s", got.Code, got.Body.String())
	}
	// REST 401 에는 붙지 않는다 — 브라우저와 다른 클라이언트가 엉뚱한 곳으로 간다.
	rest := routed(on, "/api/v1/auth/me")
	if rest.Code != http.StatusUnauthorized || rest.Header().Get("WWW-Authenticate") != "" {
		t.Fatalf("REST 401 에 도전 헤더가 붙었다: %d %q", rest.Code, rest.Header().Get("WWW-Authenticate"))
	}
	// 꺼져 있으면 /mcp 401 도 전과 같다.
	off, _ := oauthServer(t, map[string]string{"oidc.issuer_url": "https://keycloak.intra/realms/company"})
	if got := mcpCall(off, ""); got.Code != http.StatusUnauthorized || got.Header().Get("WWW-Authenticate") != "" {
		t.Fatalf("꺼진 서버의 /mcp 401 에 도전 헤더가 붙었다: %q", got.Header().Get("WWW-Authenticate"))
	}
}

func TestTokenIgnoredWhenDisabledOrOutsideMCP(t *testing.T) {
	idp := newFakeIdP(t)
	valid := idp.token(t, map[string]any{"aud": []string{"https://seaton.intra/mcp"}})
	// 꺼져 있으면 유효한 토큰도 키 전용 때와 똑같이 거절된다.
	off, logs := oauthServer(t, map[string]string{"oidc.issuer_url": idp.URL, "mcp.oauth.resource": "https://seaton.intra/mcp"})
	if got := mcpCall(off, valid); got.Code != http.StatusUnauthorized || !strings.Contains(got.Body.String(), "authentication_required") || got.Header().Get("WWW-Authenticate") != "" {
		t.Fatalf("꺼진 서버: %d %s %q", got.Code, got.Body.String(), got.Header().Get("WWW-Authenticate"))
	}
	if strings.Contains(logs.String(), "mcp oauth") {
		t.Fatalf("꺼져 있으면 토큰을 보지도 않아야 한다: %s", logs.String())
	}
	// 켜져 있어도 REST 경로는 토큰을 받지 않는다.
	on, _ := oauthServer(t, enabledValues(idp, map[string]string{"mcp.oauth.resource": "https://seaton.intra/mcp"}))
	request := httptest.NewRequest(http.MethodGet, "https://seaton.intra/api/v1/employees", nil)
	request.Header.Set("Authorization", "Bearer "+valid)
	recorder := httptest.NewRecorder()
	on.Routes().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized || !strings.Contains(recorder.Body.String(), "authentication_required") || recorder.Header().Get("WWW-Authenticate") != "" {
		t.Fatalf("REST 에서 SSO 토큰이 받아들여졌다: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestVerifyMCPAccessToken(t *testing.T) {
	idp := newFakeIdP(t)
	other := newFakeIdP(t)
	resource := "https://seaton.intra/mcp"
	s, _ := oauthServer(t, nil)
	base := readMCPOAuthConfig(enabledValues(idp, map[string]string{"mcp.oauth.resource": resource}))
	withAudience := base
	withAudience.Audiences = []string{"claude-mcp"}
	past := time.Now().Add(-10 * time.Minute).Unix()

	cases := []struct {
		name     string
		cfg      mcpOAuthConfig
		token    string
		username string // 비어 있으면 거절을 기대
		message  string // 거절 문장에 들어 있어야 하는 것
		cause    string // 로그용 원인에 들어 있어야 하는 것
	}{
		{name: "aud 에 리소스 식별자 (Audience 매퍼)", cfg: base, token: idp.token(t, map[string]any{"aud": []string{"account", resource}}), username: "hong"},
		{name: "azp 가 허용 대상에 (매퍼 없는 호환 경로)", cfg: withAudience, token: idp.token(t, nil), username: "hong"},
		{name: "aud 가 허용 대상에", cfg: withAudience, token: idp.token(t, map[string]any{"aud": "claude-mcp", "azp": "x"}), username: "hong"},
		{name: "다른 앱용 토큰", cfg: base, token: idp.token(t, nil), message: `aud=[account], azp="claude-mcp"`, cause: "audience"},
		{name: "만료", cfg: withAudience, token: idp.token(t, map[string]any{"exp": past}), message: "만료", cause: "expired"},
		{name: "아직 유효하지 않음", cfg: withAudience, token: idp.token(t, map[string]any{"nbf": time.Now().Add(time.Hour).Unix()}), message: "유효하지 않습니다", cause: "before"},
		// 우리 키로 서명했지만 iss 가 다른 realm — 서명은 맞아도 발급자에서 걸린다.
		{name: "다른 issuer", cfg: withAudience, token: idp.token(t, map[string]any{"iss": other.URL}), message: "발급자", cause: "different provider"},
		// 다른 realm 의 키로 서명한 것은 서명에서 걸린다.
		{name: "다른 realm 의 서명", cfg: withAudience, token: other.token(t, nil), message: "서명", cause: "signature"},
		{name: "HS256 서명", cfg: withAudience, token: idp.hsToken(nil), message: "서명", cause: "HS256"},
		{name: "ID 토큰", cfg: withAudience, token: idp.token(t, map[string]any{"typ": "ID"}), message: "ID 토큰", cause: "typ=ID"},
		{name: "cnf 있음", cfg: withAudience, token: idp.token(t, map[string]any{"cnf": map[string]string{"jkt": "x"}}), message: "cnf", cause: "cnf"},
		{name: "sub 없음", cfg: withAudience, token: idp.token(t, map[string]any{"sub": nil}), message: "sub", cause: "sub"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			identity, refusal := s.verifyMCPAccessToken(context.Background(), c.cfg, c.token)
			if c.username != "" {
				if refusal != nil {
					t.Fatalf("통과해야 한다: %v", refusal)
				}
				if identity.Username != c.username || identity.Subject != "sub-1" {
					t.Fatalf("identity = %+v", identity)
				}
				return
			}
			if refusal == nil {
				t.Fatalf("거절해야 한다: %+v", identity)
			}
			if !strings.Contains(refusal.message, c.message) {
				t.Fatalf("문장 %q 에 %q 가 없다", refusal.message, c.message)
			}
			if refusal.cause == nil || !strings.Contains(refusal.cause.Error(), c.cause) {
				t.Fatalf("원인 %v 에 %q 가 없다", refusal.cause, c.cause)
			}
		})
	}
	// 다른 앱용 토큰의 거절 문장은 본 것과 고칠 값을 말한다 — 운영자는 이 한 줄로 설정을 끝낸다.
	_, refusal := s.verifyMCPAccessToken(context.Background(), base, idp.token(t, nil))
	for _, want := range []string{`aud=[account]`, `azp="claude-mcp"`, `허용 대상에 "claude-mcp"`, `Audience 매퍼로 "` + resource + `"`} {
		if !strings.Contains(refusal.message, want) {
			t.Fatalf("거절 문장에 %q 가 없다: %s", want, refusal.message)
		}
	}
}

// 리소스 식별자와 허용 대상이 비어 있으면(마이그레이션 기본값) 허용값이 하나도
// 없다 — 요청의 Host 로 리소스 식별자를 만들어 허용값에 넣지 않는다. 예전에는
// Host: other-app.intra 를 붙이면 aud=https://other-app.intra/mcp 인 같은 realm 의
// 다른 앱용 토큰이 이 서버에 들어왔다.
func TestHostHeaderDoesNotBecomeAcceptedAudience(t *testing.T) {
	idp := newFakeIdP(t)
	foreign := idp.token(t, map[string]any{"aud": []string{"https://other-app.intra/mcp"}})
	// 1. 검증기 자체: 설정이 비어 있으면 어떤 aud 도 맞지 않는다.
	s, _ := oauthServer(t, nil)
	empty := readMCPOAuthConfig(enabledValues(idp, nil))
	if empty.Resource != "" || len(empty.Audiences) != 0 {
		t.Fatalf("기본값이 비어 있어야 하는 검사다: %+v", empty)
	}
	identity, refusal := s.verifyMCPAccessToken(context.Background(), empty, foreign)
	if refusal == nil || refusal.code != "invalid_token" || !strings.Contains(refusal.cause.Error(), "audience") {
		t.Fatalf("설정이 비어 있으면 대상 검사에서 거절해야 한다: identity=%+v refusal=%v", identity, refusal)
	}
	// 리소스 식별자가 있어도 Host 가 아니라 그 값과만 비교한다.
	fixed := readMCPOAuthConfig(enabledValues(idp, map[string]string{"mcp.oauth.resource": "https://seaton.intra/mcp"}))
	if _, refusal := s.verifyMCPAccessToken(context.Background(), fixed, foreign); refusal == nil {
		t.Fatal("다른 리소스 서버용 aud 가 통과했다")
	}
	// 2. 라우터를 지나서: 리소스 식별자 없이 켠 서버에 Host 를 조작한 요청은
	// 켜지지 않은 서버와 똑같이 거절된다 — 토큰을 보지도 않는다.
	server, logs := oauthServer(t, enabledValues(idp, nil))
	for _, host := range []string{"other-app.intra", "seaton.intra"} {
		request := httptest.NewRequest(http.MethodPost, "https://"+host+"/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
		request.Host = host
		request.Header.Set("X-Forwarded-Host", host)
		request.Header.Set("X-Forwarded-Proto", "https")
		request.Header.Set("Authorization", "Bearer "+idp.token(t, map[string]any{"aud": []string{"https://" + host + "/mcp"}}))
		recorder := httptest.NewRecorder()
		server.Routes().ServeHTTP(recorder, request)
		if recorder.Code != http.StatusUnauthorized || !strings.Contains(recorder.Body.String(), "authentication_required") || strings.Contains(recorder.Body.String(), `"tools"`) {
			t.Fatalf("Host %s: %d %s", host, recorder.Code, recorder.Body.String())
		}
		if recorder.Header().Get("WWW-Authenticate") != "" {
			t.Fatalf("Host %s: 리소스 식별자 없이는 도전 헤더도 없어야 한다: %q", host, recorder.Header().Get("WWW-Authenticate"))
		}
	}
	if strings.Contains(logs.String(), "mcp oauth") {
		t.Fatalf("리소스 식별자 없이는 토큰을 보지도 않아야 한다: %s", logs.String())
	}
}

func TestRefusedTokenGetsChallengeAndLoggedCause(t *testing.T) {
	idp := newFakeIdP(t)
	s, logs := oauthServer(t, enabledValues(idp, map[string]string{"mcp.oauth.resource": "https://seaton.intra/mcp"}))
	got := mcpCall(s, idp.token(t, map[string]any{"exp": time.Now().Add(-time.Hour).Unix()}))
	if got.Code != http.StatusUnauthorized || !strings.Contains(got.Body.String(), "invalid_token") {
		t.Fatalf("%d %s", got.Code, got.Body.String())
	}
	h := got.Header().Get("WWW-Authenticate")
	if !strings.Contains(h, `resource_metadata="https://seaton.intra/.well-known/oauth-protected-resource/mcp"`) || !strings.Contains(h, `error="invalid_token"`) {
		t.Fatalf("거부된 토큰의 401 에는 resource_metadata 와 error=invalid_token 이 있어야 한다: %q", h)
	}
	// 클라이언트에는 문장만 가고, 무엇이 실패했는지(여기서는 만료)는 로그에 남는다.
	if !strings.Contains(logs.String(), "mcp oauth token rejected") || !strings.Contains(logs.String(), "expired") {
		t.Fatalf("거절 원인이 로그에 없다: %s", logs.String())
	}
	// 운영자가 접근 로그의 요청과 짝지을 수 있게 요청 ID 도 함께 — 값은 라우터의
	// RequestID 미들웨어가 채운 것이라 비어 있지 않다.
	if !strings.Contains(logs.String(), "request_id=") || strings.Contains(logs.String(), `request_id=""`) || strings.Contains(logs.String(), "request_id= ") {
		t.Fatalf("거절 로그에 요청 ID 가 없다: %s", logs.String())
	}
	if strings.Contains(got.Body.String(), "expired") {
		t.Fatalf("라이브러리 오류 원문이 클라이언트로 나갔다: %s", got.Body.String())
	}
}

func TestDiscoveryFailureIsNotAnInvalidToken(t *testing.T) {
	// discovery 에 500 으로 답하는 IdP. 호출 횟수를 세어 음성 캐시를 본다.
	var hits atomic.Int32
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Error(w, "down", http.StatusInternalServerError)
	}))
	t.Cleanup(broken.Close)
	s, logs := oauthServer(t, map[string]string{"mcp.oauth.enabled": "true", "oidc.issuer_url": broken.URL, "mcp.oauth.resource": "https://seaton.intra/mcp"})
	got := mcpCall(s, "a.b.c")
	if got.Code != http.StatusServiceUnavailable || !strings.Contains(got.Body.String(), "sso_unavailable") {
		t.Fatalf("Keycloak 에 닿지 못하면 503: %d %s", got.Code, got.Body.String())
	}
	if got.Header().Get("WWW-Authenticate") != "" {
		t.Fatal("503 에 도전 헤더를 붙이면 클라이언트가 로그인 루프에 빠진다")
	}
	if !strings.Contains(logs.String(), "sso_unavailable") || !strings.Contains(logs.String(), "request_id=") {
		t.Fatalf("원인과 요청 ID 가 로그에 없다: %s", logs.String())
	}
	if hits.Load() != 1 {
		t.Fatalf("첫 요청은 discovery 를 한 번 해야 한다: %d", hits.Load())
	}
	// 실패는 음성 캐시된다 — 다음 요청은 IdP 를 다시 두드리지 않고 바로 503 이다.
	if got := mcpCall(s, "a.b.c"); got.Code != http.StatusServiceUnavailable {
		t.Fatalf("두 번째 요청도 503: %d %s", got.Code, got.Body.String())
	}
	if hits.Load() != 1 {
		t.Fatalf("실패한 discovery 를 요청마다 다시 시도했다: %d 회", hits.Load())
	}
	// 음성 캐시가 지나면 다시 시도한다.
	s.oauth.mu.Lock()
	s.oauth.byIssuer[strings.TrimRight(broken.URL, "/")].failedAt = time.Now().Add(-2 * mcpOAuthDiscoveryRetry)
	s.oauth.mu.Unlock()
	if got := mcpCall(s, "a.b.c"); got.Code != http.StatusServiceUnavailable {
		t.Fatalf("재시도도 실패면 503: %d", got.Code)
	}
	if hits.Load() != 2 {
		t.Fatalf("음성 캐시가 지나면 discovery 를 다시 해야 한다: %d 회", hits.Load())
	}
	// 취소된 요청은 discovery 를 기다리지 않고 돌아온다 — 응답을 기다릴 클라이언트가 없다.
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.oauthProvider(cancelled, "http://127.0.0.1:1/never"); !errors.Is(err, context.Canceled) {
		t.Fatalf("취소된 요청: %v", err)
	}
}

// 같은 issuer 의 동시 요청은 discovery 를 한 번만 하고, 뮤텍스 뒤에 직렬로 서지
// 않는다 — 느린 IdP 를 기다리는 동안 다른 요청이 뮤텍스를 잡을 수 있다.
func TestDiscoveryIsSharedAndOffTheMutex(t *testing.T) {
	var hits atomic.Int32
	release := make(chan struct{})
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		<-release
		http.Error(w, "down", http.StatusInternalServerError)
	}))
	t.Cleanup(slow.Close)
	s, _ := oauthServer(t, nil)
	results := make(chan error, 4)
	for i := 0; i < 4; i++ {
		go func() {
			_, err := s.oauthProvider(context.Background(), slow.URL)
			results <- err
		}()
	}
	// IdP 가 답하지 않는 동안에도 뮤텍스는 비어 있어야 한다 — 잡혀 있으면 여기서 멈춘다.
	for hits.Load() == 0 {
		time.Sleep(time.Millisecond)
	}
	locked := make(chan struct{})
	go func() {
		s.oauth.mu.Lock()
		s.oauth.mu.Unlock()
		close(locked)
	}()
	select {
	case <-locked:
	case <-time.After(2 * time.Second):
		t.Fatal("discovery 가 진행 중인 동안 뮤텍스가 잡혀 있다")
	}
	close(release)
	for i := 0; i < 4; i++ {
		if err := <-results; err == nil {
			t.Fatal("500 discovery 가 성공했다")
		}
	}
	if hits.Load() != 1 {
		t.Fatalf("동시 요청 4개가 discovery 를 %d 번 했다", hits.Load())
	}
}

func TestMCPOAuthScopesAreCappedByAdmin(t *testing.T) {
	cases := []struct {
		configured, carried, want string
		refused                   bool
	}{
		// Keycloak 의 평범한 scope 는 어휘 밖이라 관리자 설정 그대로.
		{configured: "read mcp", carried: "openid profile email", want: "read mcp"},
		{configured: "read mcp", carried: "", want: "read mcp"},
		// 토큰이 이 앱의 어휘를 실어 오면 교집합만.
		{configured: "read write mcp", carried: "openid mcp read", want: "read mcp"},
		{configured: "read mcp", carried: "write mcp", want: "mcp"},
		// 관리자가 주지 않은 범위만 요구한 토큰은 거절이다. 빈 범위를 돌려주면
		// /mcp 가 그것을 "세션 인증(무제한)"으로 읽어 오히려 전부 열린다.
		{configured: "read mcp", carried: "write", refused: true},
		{configured: "mcp", carried: "read", refused: true},
		{configured: "read mcp", carried: "openid write", refused: true},
	}
	for _, c := range cases {
		granted, refusal := mcpOAuthScopes(strings.Fields(c.configured), strings.Fields(c.carried))
		if c.refused {
			if refusal == nil || refusal.status != http.StatusForbidden || refusal.code != "insufficient_scope" {
				t.Errorf("configured=%q carried=%q: 403 insufficient_scope 로 거절해야 한다: granted=%v refusal=%v", c.configured, c.carried, granted, refusal)
			}
			continue
		}
		if refusal != nil {
			t.Errorf("configured=%q carried=%q: 거절하면 안 된다: %v", c.configured, c.carried, refusal)
			continue
		}
		if got := strings.Join(granted, " "); got != c.want {
			t.Errorf("configured=%q carried=%q: got %q want %q", c.configured, c.carried, got, c.want)
		}
	}
}

// 관리자 천장 밖의 범위만 실은 토큰은 /mcp 에 발을 들이지 못한다. 예전에는 빈
// 범위로 통과해 mcp·write 검사를 모두 건너뛰었다 — 천장이 read mcp 인데 write
// 만 든 토큰이 쓰기 도구까지 열었다.
func TestTokenOutsideAdminScopeCeilingIsRefused(t *testing.T) {
	idp := newFakeIdP(t)
	s, logs := oauthServer(t, enabledValues(idp, map[string]string{"mcp.oauth.resource": "https://seaton.intra/mcp", "mcp.oauth.scopes": "read mcp"}))
	got := mcpCall(s, idp.token(t, map[string]any{"aud": []string{"https://seaton.intra/mcp"}, "scope": "openid write"}))
	if got.Code != http.StatusForbidden || !strings.Contains(got.Body.String(), "insufficient_scope") {
		t.Fatalf("천장 밖 범위만 실은 토큰은 403 insufficient_scope: %d %s", got.Code, got.Body.String())
	}
	if strings.Contains(got.Body.String(), `"tools"`) {
		t.Fatalf("도구 목록이 열렸다: %s", got.Body.String())
	}
	if !strings.Contains(logs.String(), "mcp oauth token rejected") || !strings.Contains(logs.String(), "insufficient_scope") {
		t.Fatalf("거절 원인이 로그에 없다: %s", logs.String())
	}
}

func TestMCPOAuthSettingsValidation(t *testing.T) {
	issuer := "https://keycloak.intra/realms/company"
	ok := []map[string]string{
		{},
		{"mcp.oauth.enabled": "true", "oidc.issuer_url": issuer, "mcp.oauth.scopes": "read mcp", "mcp.oauth.resource": "https://seaton.intra/mcp"},
		{"mcp.oauth.enabled": "true", "oidc.issuer_url": issuer, "mcp.oauth.scopes": "read write mcp", "mcp.oauth.resource": "https://seaton.intra/mcp", "mcp.oauth.audience": "claude-mcp cursor"},
		{"mcp.oauth.resource": "http://localhost:8080/mcp"},
		// 꺼져 있으면 issuer 가 없어도 저장된다 — 기본 설치가 그렇다.
		{"mcp.oauth.enabled": "false", "mcp.oauth.scopes": "read mcp"},
	}
	for _, values := range ok {
		if err := readMCPOAuthConfig(values).validate(); err != nil {
			t.Errorf("%v 는 받아야 한다: %v", values, err)
		}
	}
	bad := []map[string]string{
		{"mcp.oauth.enabled": "true", "mcp.oauth.scopes": "read mcp", "mcp.oauth.resource": "https://seaton.intra/mcp"},
		{"mcp.oauth.enabled": "true", "oidc.issuer_url": issuer, "mcp.oauth.scopes": "read", "mcp.oauth.resource": "https://seaton.intra/mcp"},
		// 리소스 식별자 없이는 켤 수 없다 — 요청 주소로 대신 만들지 않는다.
		{"mcp.oauth.enabled": "true", "oidc.issuer_url": issuer, "mcp.oauth.scopes": "read mcp"},
		{"mcp.oauth.enabled": "true", "oidc.issuer_url": issuer, "mcp.oauth.scopes": "read mcp", "mcp.oauth.audience": "claude-mcp"},
		{"mcp.oauth.scopes": "read admin mcp"},
		{"mcp.oauth.resource": "seaton.intra/mcp"},
		{"mcp.oauth.resource": "https://seaton.intra/api"},
		{"mcp.oauth.resource": "https://user:pw@seaton.intra/mcp"},
		{"mcp.oauth.resource": "https://seaton.intra/mcp?x=1"},
		{"mcp.oauth.audience": `claude"mcp`},
	}
	for _, values := range bad {
		if err := readMCPOAuthConfig(values).validate(); err == nil {
			t.Errorf("%v 는 거절해야 한다", values)
		}
	}
}

func TestLooksLikeJWT(t *testing.T) {
	for _, v := range []string{"a.b.c", "eyJ.eyJ.sig"} {
		if !looksLikeJWT(v) {
			t.Errorf("%q 는 JWT 모양이다", v)
		}
	}
	for _, v := range []string{"", "seat_abc", "a.b", "a..c", ".b.c", "a.b.", "a.b.c.d"} {
		if looksLikeJWT(v) {
			t.Errorf("%q 는 JWT 모양이 아니다", v)
		}
	}
}
