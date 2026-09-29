package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
)

// MCP 를 SSO 로 — 개인 키 없이, Keycloak 이 발급한 액세스 토큰으로.
//
// MCP 인가 규격(2025-06-18 이후)은 OAuth 2.1 이다. 이 서버는 리소스 서버다:
// 인증 서버가 어디인지 알리고(RFC 9728, /.well-known/oauth-protected-resource),
// 401 로 거절할 때 그 문서의 주소를 WWW-Authenticate 에 실어 보내면, 클라이언트가
// 스스로 Keycloak 에서 PKCE 로 로그인해 이 서버를 대상으로 하는 토큰을 받아 온다.
// 토큰을 발급하거나 저장하거나 세션으로 바꾸는 일은 여기 없다 — 요청마다 검사한다.
//
// 개인 키는 그대로 둔다. 사람 없이 도는 스크립트와 Keycloak 없는 설치가 여전히
// 키를 쓴다. SSO 토큰은 같은 방으로 들어오는 두 번째 문이다: 이미 웹으로 로그인해
// 등록된 활성 계정을 찾고, 관리자가 정한 범위를 주며, 키와 같은 검사를 지난다.
// 계정을 만들거나 정지된 계정을 열거나 토큰의 role 로 권한을 올리지 않는다.
// 그리고 이 문은 /mcp 에만 있다 — REST·관리 API 는 지금처럼 키와 세션만 받는다.

const mcpOAuthKeyPrefix = "mcp.oauth."

// mcpOAuthAppScopes 는 이 앱의 범위 어휘다(api_keys.scopes 와 같다).
var mcpOAuthAppScopes = []string{"read", "write", "mcp"}

// mcpOAuthSigningAlgs 는 받아들이는 서명 알고리즘이다. HS* 와 none 은 없다 —
// 공개 키로 검증할 수 없는 서명은 Keycloak 의 서명이 아니다.
var mcpOAuthSigningAlgs = []string{oidc.RS256, oidc.RS384, oidc.RS512, oidc.ES256, oidc.ES384, oidc.ES512, oidc.PS256, oidc.PS384, oidc.PS512}

// mcpOAuthConfig 는 MCP 를 SSO 토큰으로 여는 데 필요한 설정 묶음이다.
// Issuer 와 ClientID 는 웹 로그인 설정을 그대로 다시 쓴다.
type mcpOAuthConfig struct {
	Enabled bool
	// Issuer 는 Keycloak realm 의 issuer(oidc.issuer_url). 토큰의 iss 와 같아야 한다.
	Issuer string
	// Resource 는 이 서버가 주장하는 리소스 식별자(RFC 8707). 설정값만 쓴다 —
	// 요청의 Host 로 만들지 않는다. Host 는 클라이언트(공격자)가 정하는 값이라
	// 그것으로 만든 식별자를 aud 허용값에 넣으면 같은 realm 의 다른 리소스
	// 서버용 토큰이 Host 헤더 하나로 들어온다. 비어 있으면 켜지지 않는다.
	Resource string
	// Audiences 는 관리자가 적은 허용 대상. 토큰의 aud 또는 azp 와 비교한다.
	Audiences []string
	// Scopes 는 SSO 토큰 주체에게 주는 범위. 토큰의 scope 가 아니라 관리자가
	// 정한다 — Keycloak 에 이 앱의 범위 어휘를 가르치지 않아도 되게.
	Scopes []string
	// ServiceName 은 메타데이터의 resource_name 에 쓴다(general.service_name).
	ServiceName string
}

func readMCPOAuthConfig(values map[string]string) mcpOAuthConfig {
	return mcpOAuthConfig{
		Enabled:     values["mcp.oauth.enabled"] == "true",
		Issuer:      strings.TrimRight(strings.TrimSpace(values["oidc.issuer_url"]), "/"),
		Resource:    strings.TrimSpace(values["mcp.oauth.resource"]),
		Audiences:   strings.Fields(values["mcp.oauth.audience"]),
		Scopes:      strings.Fields(values["mcp.oauth.scopes"]),
		ServiceName: strings.TrimSpace(values["general.service_name"]),
	}
}

func (s *Server) loadMCPOAuth(ctx context.Context) mcpOAuthConfig {
	rows, err := s.db.Query(ctx, `SELECT key,value FROM settings WHERE key LIKE $1 OR key IN ('oidc.issuer_url','general.service_name')`, mcpOAuthKeyPrefix+"%")
	if err != nil {
		s.logger.Warn("MCP SSO 설정을 읽지 못했습니다", "error", err)
		return mcpOAuthConfig{}
	}
	defer rows.Close()
	values := map[string]string{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err == nil {
			values[key] = value
		}
	}
	return readMCPOAuthConfig(values)
}

// active 는 토큰을 실제로 받을지다. 스위치가 켜져 있어도 issuer 가 없으면
// 검증할 길이 없고, 리소스 식별자가 없으면 대상 검사의 허용값과 메타데이터의
// resource 를 설정값으로 만들 수 없으므로 꺼진 것처럼 동작한다(저장 시점에
// validate 가 막지만, 옛 데이터나 SQL 로 넣은 값은 그 검사를 거치지 않았다).
func (c mcpOAuthConfig) active() bool {
	return c.Enabled && c.Issuer != "" && c.Resource != ""
}

// metadataURL 은 거절된 클라이언트가 인증 서버를 찾으러 가는 문서의 주소다.
// 리소스 식별자에서만 만든다 — 요청의 Host 는 보지 않는다.
func (c mcpOAuthConfig) metadataURL() string {
	if parsed, err := url.Parse(c.Resource); err == nil && parsed.Host != "" {
		return parsed.Scheme + "://" + parsed.Host + "/.well-known/oauth-protected-resource" + parsed.Path
	}
	return strings.TrimSuffix(c.Resource, "/mcp") + "/.well-known/oauth-protected-resource/mcp"
}

// validate 는 저장하려는 값이 말이 되는지 본다. 켜는 조건은 issuer 와 리소스
// 식별자가 있고 범위가 이 앱의 어휘 안에 있으며 mcp 를 포함하는 것이다 — mcp 가
// 없으면 토큰이 통과해도 /mcp 가 403 을 내므로 켜 둘 이유가 없다. 리소스
// 식별자는 요청 Host 로 대신 만들지 않으므로(위 Resource 참고) 켤 때 필수다.
func (c mcpOAuthConfig) validate() error {
	if c.Resource != "" {
		parsed, err := url.Parse(c.Resource)
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
			return errors.New("MCP 리소스 식별자는 https://주소/mcp 꼴의 절대 URL 이어야 합니다(인증정보·쿼리·프래그먼트 없이)")
		}
		if parsed.Path != "/mcp" {
			return errors.New("MCP 리소스 식별자는 /mcp 로 끝나야 합니다 — 클라이언트가 실제로 접속하는 주소입니다")
		}
	}
	for _, audience := range c.Audiences {
		if len(audience) > 256 || strings.ContainsAny(audience, `"\`) {
			return errors.New("MCP 허용 대상에는 따옴표와 역슬래시를 쓸 수 없습니다")
		}
	}
	for _, scope := range c.Scopes {
		if !containsString(mcpOAuthAppScopes, scope) {
			return errors.New("MCP SSO 범위는 read, write, mcp 중에서 공백으로 구분해 적습니다")
		}
	}
	if c.Enabled {
		if c.Issuer == "" {
			return errors.New("MCP SSO 인증을 켜려면 Keycloak Issuer URL(oidc.issuer_url)이 필요합니다")
		}
		if !containsString(c.Scopes, "mcp") {
			return errors.New("MCP SSO 범위에는 mcp 가 있어야 합니다 — 없으면 토큰이 통과해도 /mcp 가 403 을 냅니다")
		}
		if c.Resource == "" {
			return errors.New("MCP SSO 인증을 켜려면 리소스 식별자(클라이언트가 접속하는 공개 주소 + /mcp)가 필요합니다 — 요청 주소로 대신 만들지 않습니다")
		}
	}
	return nil
}

// validateMCPOAuthSettings 는 저장 트랜잭션 안의 값을 통째로 읽어 검증한다.
// 요청이 일부 키만 보내도 나머지와 합친 결과가 말이 되어야 한다.
func (s *Server) validateMCPOAuthSettings(ctx context.Context, tx pgx.Tx) error {
	rows, err := tx.Query(ctx, `SELECT key,value FROM settings WHERE key LIKE $1 OR key='oidc.issuer_url'`, mcpOAuthKeyPrefix+"%")
	if err != nil {
		return err
	}
	defer rows.Close()
	values := map[string]string{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return err
		}
		values[key] = value
	}
	return readMCPOAuthConfig(values).validate()
}

// oauthProviders 는 issuer 별 discovery 결과를 캐시한다. discovery 는 Keycloak
// 으로의 왕복이고 그 뒤의 JWKS 가 모든 토큰을 검증하므로, 요청마다 하면 MCP
// 호출마다 Keycloak 지연이 앞에 붙는다. go-oidc 는 모르는 key id 를 만나면 키
// 집합을 다시 받아오므로 키 회전에 캐시 무효화가 필요 없다.
//
// discovery 는 뮤텍스 밖에서 한다. 잠근 채 Keycloak 을 기다리면 IdP 가 멎었을
// 때 JWT 모양 bearer 를 실은 모든 /mcp 요청이 그 뒤에 직렬로 선다. 같은 issuer
// 의 동시 요청은 한 번의 discovery 를 함께 기다리고(singleflight), 실패는
// mcpOAuthDiscoveryRetry 동안 음성 캐시해 그 사이의 요청은 IdP 를 두드리지
// 않고 바로 503 을 받는다.
type oauthProviders struct {
	mu       sync.Mutex
	byIssuer map[string]*oauthDiscovery
}

// oauthDiscovery 는 한 issuer 의 discovery 한 번이다. done 이 닫히면 provider
// 또는 err 가 정해져 있다.
type oauthDiscovery struct {
	done     chan struct{}
	provider *oidc.Provider
	err      error
	failedAt time.Time
}

const (
	// mcpOAuthDiscoveryTimeout 은 discovery·JWKS 한 요청의 상한이다. 요청
	// 컨텍스트와 분리된 서버 수명 컨텍스트로 가므로 이것이 유일한 상한이다.
	mcpOAuthDiscoveryTimeout = 10 * time.Second
	// mcpOAuthDiscoveryRetry 는 실패한 discovery 를 다시 시도하기까지의 간격이다.
	mcpOAuthDiscoveryRetry = 30 * time.Second
)

// mcpOAuthHTTPClient 는 Keycloak 으로 가는 discovery·JWKS 요청에 쓴다.
// http.DefaultClient 는 타임아웃이 없다.
var mcpOAuthHTTPClient = &http.Client{Timeout: mcpOAuthDiscoveryTimeout}

func (s *Server) oauthProvider(ctx context.Context, issuer string) (*oidc.Provider, error) {
	s.oauth.mu.Lock()
	entry := s.oauth.byIssuer[issuer]
	if entry != nil {
		select {
		case <-entry.done:
			if entry.err != nil && time.Since(entry.failedAt) >= mcpOAuthDiscoveryRetry {
				entry = nil // 음성 캐시가 지났다 — 다시 시도한다.
			}
		default:
			// 진행 중 — 아래에서 함께 기다린다.
		}
	}
	if entry == nil {
		entry = &oauthDiscovery{done: make(chan struct{})}
		if s.oauth.byIssuer == nil {
			s.oauth.byIssuer = map[string]*oauthDiscovery{}
		}
		s.oauth.byIssuer[issuer] = entry
		go entry.run(issuer)
	}
	s.oauth.mu.Unlock()
	select {
	case <-entry.done:
		return entry.provider, entry.err
	case <-ctx.Done():
		// 요청이 취소됐다. discovery 는 계속 돌아 다음 요청이 결과를 쓴다.
		return nil, ctx.Err()
	}
}

// run 은 discovery 를 요청과 분리된 컨텍스트로 한다. provider 는 이 컨텍스트
// (의 HTTP 클라이언트)를 뒤의 JWKS 요청에도 쓰므로 어느 한 요청에 묶여서는
// 안 되고, 타임아웃은 클라이언트가 건다. go-oidc 는 키 집합에 취소를 뗀
// 컨텍스트를 쓰므로 여기서 cancel 해도 뒤의 키 요청은 막히지 않는다.
func (d *oauthDiscovery) run(issuer string) {
	defer close(d.done)
	ctx, cancel := context.WithTimeout(context.Background(), mcpOAuthDiscoveryTimeout)
	defer cancel()
	d.provider, d.err = oidc.NewProvider(oidc.ClientContext(ctx, mcpOAuthHTTPClient), issuer)
	if d.err != nil {
		d.failedAt = time.Now()
	}
}

// looksLikeJWT 는 "키가 아닌 것"과 "우리가 받는 어떤 토큰도 아닌 것"을 가르는
// 값싼 모양 검사다. 점 두 개, 세 조각이 모두 비어 있지 않아야 한다.
func looksLikeJWT(token string) bool {
	parts := strings.Split(token, ".")
	return len(parts) == 3 && parts[0] != "" && parts[1] != "" && parts[2] != ""
}

// mcpOAuthRefusal 은 토큰을 거절한 이유다. message 는 클라이언트가 읽는 문장이고
// cause 는 로그에 남는 원인(서명·발급자·만료 중 무엇이 실패했는지)이다. 둘은
// refuseMCPToken 한 곳에서 함께 나간다 — 문장만 복사하고 원인을 버리지 않도록.
type mcpOAuthRefusal struct {
	status  int
	code    string
	message string
	cause   error
}

func (e *mcpOAuthRefusal) Error() string {
	if e.cause != nil {
		return e.code + ": " + e.cause.Error()
	}
	return e.code + ": " + e.message
}

func invalidToken(message string, cause error) *mcpOAuthRefusal {
	return &mcpOAuthRefusal{status: http.StatusUnauthorized, code: "invalid_token", message: message, cause: cause}
}

// mcpTokenIdentity 는 검증을 통과한 토큰에서 계정을 찾는 데 필요한 것이다.
type mcpTokenIdentity struct {
	Subject  string
	Username string
	Scopes   []string
}

// verifyMCPAccessToken 은 Keycloak 액세스 토큰을 검사한다: 서명·iss·exp·nbf 는
// go-oidc 가, typ·cnf·sub·대상은 여기서. 데이터베이스는 건드리지 않는다.
func (s *Server) verifyMCPAccessToken(ctx context.Context, cfg mcpOAuthConfig, raw string) (mcpTokenIdentity, *mcpOAuthRefusal) {
	var identity mcpTokenIdentity
	provider, err := s.oauthProvider(ctx, cfg.Issuer)
	if err != nil {
		return identity, &mcpOAuthRefusal{status: http.StatusServiceUnavailable, code: "sso_unavailable", message: "Keycloak 발급자 정보를 읽지 못해 SSO 토큰을 확인할 수 없습니다. 잠시 후 다시 시도하거나 관리자에게 알리세요", cause: err}
	}
	// 대상은 아래에서 직접 본다. 라이브러리는 aud 하나만 비교하고 azp 를 모른다.
	token, err := provider.Verifier(&oidc.Config{SkipClientIDCheck: true, SupportedSigningAlgs: mcpOAuthSigningAlgs}).Verify(ctx, raw)
	if err != nil {
		return identity, invalidToken("SSO 액세스 토큰이 유효하지 않습니다(서명·발급자·만료). 클라이언트에서 다시 로그인하세요", err)
	}
	var claims struct {
		Type              string `json:"typ"`
		Confirmation      any    `json:"cnf"`
		AuthorizedParty   string `json:"azp"`
		Scope             string `json:"scope"`
		PreferredUsername string `json:"preferred_username"`
		Email             string `json:"email"`
	}
	if err := token.Claims(&claims); err != nil {
		return identity, invalidToken("SSO 토큰의 내용을 읽을 수 없습니다", err)
	}
	// ID 토큰은 로그인 증거지 API 자격이 아니다. Keycloak 은 액세스 토큰에
	// typ=Bearer, ID 토큰에 typ=ID 를 넣는다.
	if strings.EqualFold(claims.Type, "ID") {
		return identity, invalidToken("ID 토큰이 아니라 액세스 토큰을 보내세요", errors.New("typ=ID"))
	}
	// cnf 가 있으면 소지자 증명(DPoP·mTLS)이 묶인 토큰이다. 여기서는 그 증명을
	// 검사할 수 없으므로 받지 않는다.
	if claims.Confirmation != nil {
		return identity, invalidToken("소지자 증명(cnf)이 묶인 토큰은 받지 않습니다", errors.New("cnf present"))
	}
	if strings.TrimSpace(token.Subject) == "" {
		return identity, invalidToken("SSO 토큰에 사용자 식별 정보(sub)가 없습니다", errors.New("sub missing"))
	}
	// 이 토큰이 이 서버를 위한 것인가. 실제 Keycloak 26 은 액세스 토큰의 aud 에
	// account 만 싣고 발급받은 클라이언트는 azp 에 담는다 — 그래서 "aud 에 리소스
	// 식별자가 있거나(Audience 매퍼), aud 또는 azp 가 관리자 목록에 있거나" 다.
	// 허용값은 설정에서만 온다. 요청의 Host 로 만든 값은 여기 없다 — 그것은
	// 토큰을 내미는 쪽이 정하는 값이라 허용값이 될 수 없다.
	resource := cfg.Resource
	accepted := append([]string{}, cfg.Audiences...)
	if resource != "" {
		accepted = append(accepted, resource)
	}
	bound := append(append([]string{}, token.Audience...), claims.AuthorizedParty)
	matched := false
	for _, value := range bound {
		if value != "" && containsString(accepted, value) {
			matched = true
			break
		}
	}
	if !matched {
		return identity, invalidToken(
			fmt.Sprintf("SSO 토큰이 이 서버를 위해 발급된 것이 아닙니다(aud=%v, azp=%q). 관리자가 허용 대상에 %q 를 적거나, Keycloak 클라이언트에 Audience 매퍼로 %q 를 더해야 합니다", token.Audience, claims.AuthorizedParty, claims.AuthorizedParty, resource),
			fmt.Errorf("audience %v / azp %q not in %v", token.Audience, claims.AuthorizedParty, accepted))
	}
	identity.Subject = token.Subject
	// 웹 로그인이 계정을 만들 때와 같은 규칙으로 사용자 이름을 정한다 — 그래야
	// 그 계정이 찾아진다.
	identity.Username = claims.PreferredUsername
	if identity.Username == "" {
		identity.Username = claims.Email
	}
	if identity.Username == "" {
		identity.Username = "oidc-" + token.Subject
	}
	scopes, refusal := mcpOAuthScopes(cfg.Scopes, strings.Fields(claims.Scope))
	if refusal != nil {
		return identity, refusal
	}
	identity.Scopes = scopes
	return identity, nil
}

// mcpOAuthScopes 는 주체에게 줄 범위다. 관리자 설정이 천장이고, 토큰이 이 앱의
// 어휘(read·write·mcp)를 실어 왔으면 그 교집합만 준다. Keycloak 의 평범한
// scope(openid profile email)는 어휘 밖이라 무시한다.
//
// 교집합이 비면 거절이다. 빈 범위를 돌려주면 안 된다 — /mcp 는 빈 범위를
// "세션 인증(범위 제한 없음)"으로 읽으므로, 천장이 read mcp 인데 write 만 실은
// 토큰이 오히려 쓰기 도구까지 모두 여는 권한 상승이 된다.
func mcpOAuthScopes(configured, carried []string) ([]string, *mcpOAuthRefusal) {
	narrowed := false
	for _, scope := range carried {
		if containsString(mcpOAuthAppScopes, scope) {
			narrowed = true
			break
		}
	}
	if !narrowed {
		return append([]string{}, configured...), nil
	}
	granted := []string{}
	for _, scope := range configured {
		if containsString(carried, scope) {
			granted = append(granted, scope)
		}
	}
	if len(granted) == 0 {
		return nil, &mcpOAuthRefusal{status: http.StatusForbidden, code: "insufficient_scope",
			message: fmt.Sprintf("SSO 토큰의 범위(%s)에 관리자가 허용한 범위(%s)가 하나도 없습니다", strings.Join(carried, " "), strings.Join(configured, " ")),
			cause:   fmt.Errorf("token scopes %v share nothing with configured %v", carried, configured)}
	}
	return granted, nil
}

// oauthPrincipal 은 액세스 토큰을 등록된 계정으로 바꾼다. 계정은 만들지 않는다 —
// 웹으로 로그인하는 순간이 등록이고, 프로그램이 토큰을 내미는 순간은 누군가를
// 등록할 자리가 아니다. source='oidc' 를 요구하므로 이름이 같은 로컬 계정이
// 토큰으로 열리는 일도 없다.
func (s *Server) oauthPrincipal(ctx context.Context, cfg mcpOAuthConfig, raw string) (User, []string, error) {
	identity, refusal := s.verifyMCPAccessToken(ctx, cfg, raw)
	if refusal != nil {
		return User{}, nil, refusal
	}
	u, err := scanUser(s.db.QueryRow(ctx, `SELECT id,username,display_name,COALESCE(email,''),employee_id,role,source,last_login_at,active
		FROM users WHERE lower(username)=lower($1) AND source='oidc' AND active=true`, identity.Username))
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, nil, &mcpOAuthRefusal{status: http.StatusUnauthorized, code: "account_not_registered",
			message: "이 SSO 계정(" + identity.Username + ")은 SeatOn 에 등록되지 않았거나 비활성입니다. 먼저 웹으로 한 번 로그인하세요",
			cause:   fmt.Errorf("no active oidc user for %q (sub %s)", identity.Username, identity.Subject)}
	}
	if err != nil {
		return User{}, nil, err
	}
	return u, identity.Scopes, nil
}

// refuseMCPToken 은 거절을 응답과 로그에 함께 적는다. 클라이언트에는 고칠 수 있는
// 문장을, 로그에는 실제 원인을 — 운영자는 로그의 cause 로 서명·발급자·만료 중
// 무엇이 실패했는지 본다.
func (s *Server) refuseMCPToken(w http.ResponseWriter, r *http.Request, cfg mcpOAuthConfig, err error) {
	var refusal *mcpOAuthRefusal
	if !errors.As(err, &refusal) {
		s.logger.Error("mcp oauth lookup failed", "error", err, "request_id", middleware.GetReqID(r.Context()))
		writeError(w, http.StatusInternalServerError, "database_error", "데이터를 처리하지 못했습니다")
		return
	}
	s.logger.Warn("mcp oauth token rejected", "code", refusal.code, "cause", refusal.cause, "remote", r.RemoteAddr, "request_id", middleware.GetReqID(r.Context()))
	if refusal.status == http.StatusUnauthorized {
		s.mcpChallenge(w, r, cfg, true)
	}
	writeError(w, refusal.status, refusal.code, refusal.message)
}

// mcpChallenge 는 401 을 초대장으로 바꾸는 헤더다: MCP 클라이언트가
// resource_metadata 를 읽고 거기서 OAuth 흐름을 시작한다. 없으면 거절은 막다른
// 길이다. /mcp 에서만, 켜져 있을 때만 붙는다 — REST 401 에 붙으면 브라우저와
// 다른 클라이언트가 엉뚱한 곳으로 간다.
func (s *Server) mcpChallenge(w http.ResponseWriter, r *http.Request, cfg mcpOAuthConfig, tokenRejected bool) {
	if r.URL.Path != "/mcp" || !cfg.active() {
		return
	}
	value := fmt.Sprintf(`Bearer realm="SeatOn", resource_metadata=%q`, cfg.metadataURL())
	if tokenRejected {
		value += `, error="invalid_token"`
	}
	w.Header().Set("WWW-Authenticate", value)
}

// unauthorized 는 인증 실패 401 이다. /mcp 라면 도전 헤더를 함께 붙인다.
func (s *Server) unauthorized(w http.ResponseWriter, r *http.Request, code, message string) {
	if r.URL.Path == "/mcp" {
		s.mcpChallenge(w, r, s.mcpOAuthConfig(r.Context()), code == "invalid_api_key")
	}
	writeError(w, http.StatusUnauthorized, code, message)
}

// protectedResourceMetadata 는 RFC 9728 문서다: 거절된 MCP 클라이언트가 인증
// 서버를 찾으러 읽는다. 인증 없이, 제품의 응답 봉투가 아니라 맨 JSON 으로 —
// 읽는 쪽은 {data:…} 를 모르는 OAuth 클라이언트 라이브러리다. 어디서 로그인하는지
// 말할 뿐 누가 로그인했는지는 말하지 않으므로 공개해도 된다.
func (s *Server) protectedResourceMetadata(w http.ResponseWriter, r *http.Request) {
	cfg := s.mcpOAuthConfig(r.Context())
	if !cfg.active() {
		writeError(w, http.StatusNotFound, "mcp_oauth_disabled", "이 서버의 MCP 는 SSO 토큰을 받지 않습니다. 개인 API 키(seat_)를 사용하세요")
		return
	}
	// 브라우저 안에서 도는 클라이언트가 읽으므로 이 문서만 CORS 로 연다.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "public, max-age=300")
	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = "SeatOn"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"resource":                 cfg.Resource,
		"authorization_servers":    []string{cfg.Issuer},
		"bearer_methods_supported": []string{"header"},
		"scopes_supported":         cfg.Scopes,
		"resource_name":            serviceName + " MCP",
	})
}
