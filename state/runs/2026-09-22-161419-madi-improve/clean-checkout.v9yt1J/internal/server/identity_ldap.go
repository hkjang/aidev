package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
)

func ldapConfiguration(cfg map[string]any) (*url.URL, *tls.Config, error) {
	u, e := url.Parse(str(cfg, "ldap_url"))
	if e != nil || !oneOf(u.Scheme, "ldap", "ldaps") || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return nil, nil, errors.New("LDAP URL은 ldap://호스트:389 또는 ldaps://호스트:636 형식입니다")
	}
	if u.Scheme == "ldap" && !boolean(cfg, "ldap_starttls") {
		return nil, nil, errors.New("LDAP 비밀번호 보호를 위해 LDAPS 또는 StartTLS를 사용해야 합니다")
	}
	if str(cfg, "ldap_base_dn") == "" {
		return nil, nil, errors.New("LDAP 검색 Base DN을 입력하세요")
	}
	filter := str(cfg, "ldap_user_filter")
	if strings.Count(filter, "{username}") != 1 {
		return nil, nil, errors.New("LDAP 사용자 필터에는 {username}을 한 번 포함하세요")
	}
	if _, e = ldap.CompileFilter(strings.ReplaceAll(filter, "{username}", "madi-test")); e != nil {
		return nil, nil, errors.New("LDAP 검색 필터 문법을 확인하세요")
	}
	pool, e := x509.SystemCertPool()
	if e != nil {
		pool = x509.NewCertPool()
	}
	if pem := str(cfg, "ldap_ca_pem"); pem != "" && !pool.AppendCertsFromPEM([]byte(pem)) {
		return nil, nil, errors.New("LDAP CA 인증서 PEM을 확인하세요")
	}
	return u, &tls.Config{MinVersion: tls.VersionTLS12, ServerName: u.Hostname(), RootCAs: pool}, nil
}

func authenticateLDAP(ctx context.Context, cfg map[string]any, username, password string) (externalIdentity, error) {
	var out externalIdentity
	if !boolean(cfg, "ldap_enabled") {
		return out, errors.New("LDAP 로그인이 비활성화되어 있습니다")
	}
	if strings.TrimSpace(username) == "" || password == "" || len(username) > 254 || len(password) > 4096 {
		return out, errors.New("LDAP 아이디와 비밀번호를 입력하세요")
	}
	endpoint, tlsConfig, e := ldapConfiguration(cfg)
	if e != nil {
		return out, e
	}
	dialer := &net.Dialer{Timeout: 8 * time.Second}
	connection, e := ldap.DialURL(endpoint.String(), ldap.DialWithDialer(dialer), ldap.DialWithTLSConfig(tlsConfig))
	if e != nil {
		return out, errors.New("LDAP 서버에 연결하지 못했습니다. 주소와 CA 인증서를 확인하세요")
	}
	defer connection.Close()
	connection.SetTimeout(8 * time.Second)
	stop := context.AfterFunc(ctx, func() { connection.Close() })
	defer stop()
	if endpoint.Scheme == "ldap" {
		if e = connection.StartTLS(tlsConfig); e != nil {
			return out, errors.New("LDAP StartTLS 인증서 검증에 실패했습니다")
		}
	}
	if dn := str(cfg, "ldap_bind_dn"); dn != "" {
		if e = connection.Bind(dn, str(cfg, "ldap_bind_password")); e != nil {
			return out, errors.New("LDAP 검색 계정 인증에 실패했습니다")
		}
	}
	attributes := []string{str(cfg, "ldap_subject_attribute"), str(cfg, "ldap_email_attribute"), str(cfg, "ldap_name_attribute"), str(cfg, "ldap_groups_attribute")}
	filter := strings.ReplaceAll(str(cfg, "ldap_user_filter"), "{username}", ldap.EscapeFilter(strings.TrimSpace(username)))
	result, e := connection.Search(ldap.NewSearchRequest(str(cfg, "ldap_base_dn"), ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 2, 8, false, filter, attributes, nil))
	if e != nil || len(result.Entries) != 1 {
		return out, errors.New("LDAP 아이디 또는 비밀번호가 올바르지 않습니다")
	}
	entry := result.Entries[0]
	if e = connection.Bind(entry.DN, password); e != nil {
		return out, errors.New("LDAP 아이디 또는 비밀번호가 올바르지 않습니다")
	}
	subject := entry.GetAttributeValue(str(cfg, "ldap_subject_attribute"))
	if strings.EqualFold(str(cfg, "ldap_subject_attribute"), "objectGUID") {
		subject = base64.RawURLEncoding.EncodeToString(entry.GetRawAttributeValue(str(cfg, "ldap_subject_attribute")))
	}
	if subject == "" {
		return out, errors.New("LDAP 고유 ID 속성이 없습니다. entryUUID 또는 AD objectGUID 매핑을 확인하세요")
	}
	out = externalIdentity{Provider: "ldap", Issuer: str(cfg, "ldap_url") + "|" + str(cfg, "ldap_base_dn"), Subject: subject, Email: entry.GetAttributeValue(str(cfg, "ldap_email_attribute")), Name: entry.GetAttributeValue(str(cfg, "ldap_name_attribute")), Groups: entry.GetAttributeValues(str(cfg, "ldap_groups_attribute")), VerifiedEmail: true, AutoRegister: boolean(cfg, "ldap_auto_register"), ConfigurationHash: identityConfigurationHash(cfg, "ldap")}
	return out, nil
}

func (s *Server) ldapLogin(w http.ResponseWriter, r *http.Request) {
	if !s.sameOrigin(r) {
		apiError(w, 403, "허용되지 않은 요청 출처입니다")
		return
	}
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if decode(r, &in) != nil || len(in.Username) > 254 || len(in.Password) > 4096 {
		apiError(w, 400, "LDAP 아이디와 비밀번호를 확인하세요")
		return
	}
	if !s.allowLoginAttempt(integrationClientIP(r), "ldap:"+in.Username, time.Now()) {
		w.Header().Set("Retry-After", "900")
		apiError(w, 429, "로그인 시도가 너무 많습니다. 15분 후 다시 시도하세요")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	cfg, e := s.settings(ctx)
	if e != nil {
		respond(w, nil, e)
		return
	}
	identity, e := authenticateLDAP(ctx, cfg, in.Username, in.Password)
	if e != nil {
		s.audit(r, "LOGIN_FAILED", "", map[string]any{"provider": "ldap", "username": in.Username})
		apiError(w, 401, e.Error())
		return
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(ctx)
	uid, e := s.identityUser(ctx, tx, cfg, identity)
	if e != nil {
		identityLoginError(w, e)
		return
	}
	if e = tx.Commit(ctx); e != nil {
		respond(w, nil, e)
		return
	}
	if e = s.createSession(w, r, uid); e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "LOGIN", uid, map[string]any{"provider": "ldap"})
	value, e := s.user(r, uid)
	respond(w, value, e)
}
