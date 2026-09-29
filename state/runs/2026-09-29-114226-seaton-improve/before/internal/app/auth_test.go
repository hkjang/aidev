package app

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

// 조용한 SSO(prompt=none)의 전부는 무한 루프를 막는 것이다. 서버 쪽 규칙 —
// 관리자 설정이 꺼져 있으면 시도하지 않고, 거절은 주소에 표시로 남긴다 — 를
// 여기서 확인한다.

func TestSilentLoginRequiresAutoLoginSetting(t *testing.T) {
	plain := httptest.NewRequest("GET", "/api/v1/auth/oidc/start", nil)
	silent := httptest.NewRequest("GET", "/api/v1/auth/oidc/start?prompt=none", nil)
	if silentLoginRequested(plain, true) {
		t.Fatal("prompt=none 이 없으면 조용한 시도가 아니어야 한다")
	}
	if silentLoginRequested(silent, false) {
		t.Fatal("auto_login 이 꺼져 있으면 ?prompt=none 을 붙여도 평범한 로그인으로 바꿔야 한다")
	}
	if !silentLoginRequested(silent, true) {
		t.Fatal("auto_login 이 켜져 있고 prompt=none 을 요청하면 조용한 시도여야 한다")
	}
}

func TestOIDCAuthCodeURLCarriesPromptNoneOnlyWhenSilent(t *testing.T) {
	cfg := oauth2.Config{ClientID: "seaton", Endpoint: oauth2.Endpoint{AuthURL: "https://keycloak.intra/auth"}, RedirectURL: "https://seaton.intra/api/v1/auth/oidc/callback", Scopes: []string{"openid"}}
	for _, silent := range []bool{false, true} {
		raw := oidcAuthCodeURL(cfg, "state", "nonce", "verifier", silent)
		parsed, err := url.Parse(raw)
		if err != nil {
			t.Fatalf("인가 주소를 읽지 못함: %v", err)
		}
		got := parsed.Query().Get("prompt")
		if silent && got != "none" {
			t.Fatalf("조용한 시도에는 prompt=none 이 있어야 한다: %s", raw)
		}
		if !silent && got != "" {
			t.Fatalf("평범한 로그인에는 prompt 가 없어야 한다: %s", raw)
		}
		if parsed.Query().Get("nonce") != "nonce" || parsed.Query().Get("code_challenge_method") != "S256" {
			t.Fatalf("nonce 와 PKCE 는 그대로 있어야 한다: %s", raw)
		}
	}
}

func TestOIDCErrorRedirectMarksSilentRefusal(t *testing.T) {
	cases := []struct {
		silent bool
		err    string
		want   string
	}{
		// 세션이 없다는 평범한 대답은 주소에 표시만 남기고 오류로 알리지 않는다.
		{true, "login_required", "/login?sso=none"},
		{true, "interaction_required", "/login?sso=none"},
		{true, "consent_required", "/login?sso=none"},
		// 조용한 시도가 다른 이유로 거절되면 다시 시도하지 말라는 표시와 함께 알린다.
		{true, "invalid_scope", "/login?sso=none&error=invalid_scope"},
		// 사람이 단추를 눌러 시작한 로그인의 거절은 이전과 같이 오류로 알린다.
		{false, "login_required", "/login?error=login_required"},
		{false, "access_denied", "/login?error=access_denied"},
	}
	for _, c := range cases {
		if got := oidcErrorRedirect(c.silent, c.err); got != c.want {
			t.Errorf("silent=%v error=%s: got %s want %s", c.silent, c.err, got, c.want)
		}
	}
}

func TestSafeReturnToStaysOnSite(t *testing.T) {
	ok := []string{"/", "/admin/maps", "/admin/maps?floor=3#seat", "/profile/keys"}
	for _, v := range ok {
		if !safeReturnTo(v) {
			t.Errorf("%q 는 받아야 한다", v)
		}
	}
	bad := []string{"", "admin", "//evil.example", "/\\evil.example", "https://evil.example/", "/login\r\nSet-Cookie: x=y", "javascript:alert(1)"}
	for _, v := range bad {
		if safeReturnTo(v) {
			t.Errorf("%q 는 거절해야 한다", v)
		}
	}
}

// 사용자 권한 화면의 편집은 알림 메일이 닿을 주소를 넣는 유일한 길이고, 비활성화는
// 로그인·세션·API 키를 한꺼번에 막는다. 잘못된 주소를 받거나 자기 계정을 잠그는
// 일은 여기서 막는다.
func TestValidateUserPatch(t *testing.T) {
	f, tr := false, true
	str := func(s string) *string { return &s }
	cases := []struct {
		name      string
		in        userPatch
		target    string
		wantCode  string
		wantEmail *string
	}{
		{"빈 요청은 아무것도 바꾸지 않고 통과한다", userPatch{}, "u2", "", nil},
		{"모르는 권한은 거절한다", userPatch{Role: "owner"}, "u2", "invalid_role", nil},
		{"다른 계정은 비활성화할 수 있다", userPatch{Active: &f}, "u2", "", nil},
		{"자기 계정은 비활성화할 수 없다", userPatch{Active: &f}, "me", "self_deactivation", nil},
		{"자기 계정을 다시 켜는 것은 막지 않는다", userPatch{Active: &tr}, "me", "", nil},
		{"자기 계정의 권한은 낮출 수 없다", userPatch{Role: "employee"}, "me", "self_demotion", nil},
		{"자기 계정에 시스템 관리자를 다시 주는 것은 막지 않는다", userPatch{Role: "system_admin"}, "me", "", nil},
		{"다른 계정의 권한은 낮출 수 있다", userPatch{Role: "employee"}, "u2", "", nil},
		{"주소는 앞뒤 공백을 걷어낸다", userPatch{Email: str("  admin@corp.example ")}, "u2", "", str("admin@corp.example")},
		{"빈 주소는 지우라는 뜻으로 받는다", userPatch{Email: str("   ")}, "u2", "", str("")},
		{"@ 가 없는 주소는 거절한다", userPatch{Email: str("admin.corp.example")}, "u2", "invalid_email", nil},
		{"표시 이름이 붙은 꼴은 거절한다", userPatch{Email: str("관리자 <admin@corp.example>")}, "u2", "invalid_email", nil},
		{"주소 여러 개는 거절한다", userPatch{Email: str("a@corp.example, b@corp.example")}, "u2", "invalid_email", nil},
		{"너무 긴 주소는 거절한다", userPatch{Email: str(strings.Repeat("a", 250) + "@corp.example")}, "u2", "invalid_email", nil},
	}
	for _, c := range cases {
		in := c.in
		code, message := validateUserPatch(&in, c.target, "me")
		if code != c.wantCode {
			t.Errorf("%s: code=%q(%s) want %q", c.name, code, message, c.wantCode)
			continue
		}
		if c.wantEmail != nil && (in.Email == nil || *in.Email != *c.wantEmail) {
			t.Errorf("%s: email=%v want %q", c.name, in.Email, *c.wantEmail)
		}
	}
}
