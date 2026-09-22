package server

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	"github.com/jackc/pgx/v5"
	dsig "github.com/russellhaering/goxmldsig"
)

const samlBrowserCookie = "madi_saml_state"

func samlConfiguration(cfg map[string]any, requireIDP bool) (*saml.ServiceProvider, error) {
	base, e := url.Parse(strings.TrimRight(str(cfg, "site_url"), "/"))
	if e != nil || base.Host == "" || base.Scheme != "https" {
		return nil, errors.New("SAML POST 로그인은 HTTPS 서비스 URL과 신뢰할 수 있는 TLS 인증서가 필요합니다")
	}
	pair, e := tls.X509KeyPair([]byte(str(cfg, "saml_sp_certificate")), []byte(str(cfg, "saml_sp_private_key")))
	if e != nil {
		return nil, errors.New("관리자 화면에서 SAML 서비스 인증서와 서명 키를 생성하세요")
	}
	key, ok := pair.PrivateKey.(*rsa.PrivateKey)
	if !ok || key.N.BitLen() < 2048 {
		return nil, errors.New("SAML 서명 키는 RSA 2048비트 이상이어야 합니다")
	}
	certificate, e := x509.ParseCertificate(pair.Certificate[0])
	if e != nil || time.Now().Before(certificate.NotBefore) || !time.Now().Before(certificate.NotAfter) {
		return nil, errors.New("SAML 서비스 인증서가 만료되었거나 유효하지 않습니다")
	}
	metadataURL := *base
	metadataURL.Path = "/api/v1/auth/saml/metadata"
	acsURL := *base
	acsURL.Path = "/api/v1/auth/saml/acs"
	sp := &saml.ServiceProvider{EntityID: metadataURL.String(), MetadataURL: metadataURL, AcsURL: acsURL, Key: key, Certificate: certificate, SignatureMethod: dsig.RSASHA256SignatureMethod, AuthnNameIDFormat: saml.PersistentNameIDFormat, AllowIDPInitiated: false, HTTPClient: integrationHTTPClient(10 * time.Second)}
	sp.ValidateAudienceRestriction = func(assertion *saml.Assertion) error {
		if assertion.Conditions == nil || len(assertion.Conditions.AudienceRestrictions) == 0 {
			return errors.New("audience required")
		}
		for _, restriction := range assertion.Conditions.AudienceRestrictions {
			if restriction.Audience.Value != sp.EntityID {
				return errors.New("audience mismatch")
			}
		}
		return nil
	}
	if !requireIDP {
		return sp, nil
	}
	metadata, e := samlsp.ParseMetadata([]byte(str(cfg, "saml_idp_metadata")))
	if e != nil || metadata.EntityID == "" || len(metadata.IDPSSODescriptors) == 0 {
		return nil, errors.New("IdP SAML metadata XML을 확인하세요")
	}
	if !metadata.ValidUntil.IsZero() && !time.Now().Before(metadata.ValidUntil) {
		return nil, errors.New("IdP metadata의 유효기간이 지났습니다")
	}
	// Trust only currently valid certificates explicitly supplied by the admin.
	valid := false
	for descriptorIndex := range metadata.IDPSSODescriptors {
		descriptor := &metadata.IDPSSODescriptors[descriptorIndex]
		for keyIndex := range descriptor.KeyDescriptors {
			key := &descriptor.KeyDescriptors[keyIndex]
			certs := []saml.X509Certificate{}
			for _, entry := range key.KeyInfo.X509Data.X509Certificates {
				raw, e := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(entry.Data), ""))
				if e != nil {
					continue
				}
				cert, e := x509.ParseCertificate(raw)
				if e == nil && !time.Now().Before(cert.NotBefore) && time.Now().Before(cert.NotAfter) {
					certs = append(certs, entry)
					if key.Use == "" || key.Use == "signing" {
						valid = true
					}
				}
			}
			key.KeyInfo.X509Data.X509Certificates = certs
		}
	}
	if !valid {
		return nil, errors.New("IdP metadata에 유효한 서명 인증서가 없습니다")
	}
	sp.IDPMetadata = metadata
	endpoint, e := url.Parse(sp.GetSSOBindingLocation(saml.HTTPRedirectBinding))
	if e != nil || endpoint.Scheme != "https" || endpoint.Host == "" || endpoint.User != nil {
		return nil, errors.New("IdP metadata에 HTTPS HTTP-Redirect 로그인 주소가 필요합니다")
	}
	return sp, nil
}

func (s *Server) samlMetadata(w http.ResponseWriter, r *http.Request) {
	cfg, e := s.settings(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	sp, e := samlConfiguration(cfg, false)
	if e != nil {
		apiError(w, 503, e.Error())
		return
	}
	metadata := sp.Metadata()
	for i := range metadata.SPSSODescriptors {
		keys := []saml.KeyDescriptor{}
		for _, key := range metadata.SPSSODescriptors[i].KeyDescriptors {
			if key.Use == "signing" {
				keys = append(keys, key)
			}
		}
		metadata.SPSSODescriptors[i].KeyDescriptors = keys
	}
	data, e := xml.MarshalIndent(metadata, "", "  ")
	if e != nil {
		respond(w, nil, e)
		return
	}
	w.Header().Set("Content-Type", "application/samlmetadata+xml")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(append([]byte(xml.Header), data...))
}

func (s *Server) samlStart(w http.ResponseWriter, r *http.Request) {
	if !s.allowLoginAttempt(integrationClientIP(r), "saml-start", time.Now()) {
		apiError(w, 429, "로그인 시도가 너무 많습니다")
		return
	}
	cfg, e := s.settings(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	if !boolean(cfg, "saml_enabled") {
		apiError(w, 404, "SAML 로그인이 비활성화되어 있습니다")
		return
	}
	sp, e := samlConfiguration(cfg, true)
	if e != nil {
		apiError(w, 503, e.Error())
		return
	}
	request, e := sp.MakeAuthenticationRequest(sp.GetSSOBindingLocation(saml.HTTPRedirectBinding), saml.HTTPRedirectBinding, saml.HTTPPostBinding)
	if e != nil {
		apiError(w, 503, "서명된 SAML 인증 요청을 생성하지 못했습니다")
		return
	}
	state, browser := randomToken(), randomToken()
	redirect, e := request.Redirect(state, sp)
	if e != nil {
		apiError(w, 503, "SAML 인증 요청 서명에 실패했습니다")
		return
	}
	if _, e = s.DB.Exec(r.Context(), `INSERT INTO saml_authn_requests(request_id,state_hash,browser_hash,configuration_hash,expires_at) VALUES($1,$2,$3,$4,now()+interval '10 minutes')`, request.ID, digest(state), digest(browser), identityConfigurationHash(cfg, "saml")); e != nil {
		respond(w, nil, e)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: samlBrowserCookie, Value: browser, Path: "/api/v1/auth/saml", HttpOnly: true, Secure: true, SameSite: http.SameSiteNoneMode, MaxAge: 600})
	http.Redirect(w, r, redirect.String(), http.StatusFound)
}

func (s *Server) samlACS(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	if e := r.ParseForm(); e != nil {
		apiError(w, 400, "SAML 응답은 2MB 이하의 폼 데이터여야 합니다")
		return
	}
	// Artifact and IdP-initiated requests are deliberately not enabled: accepting
	// either would bypass the browser-bound, one-time request correlation below.
	state, encoded := r.PostForm.Get("RelayState"), r.PostForm.Get("SAMLResponse")
	cookie, e := r.Cookie(samlBrowserCookie)
	if e != nil || len(state) < 32 || len(state) > 200 || cookie.Value == "" || encoded == "" || r.Form.Get("SAMLart") != "" {
		apiError(w, 400, "SAML 요청 상태가 유효하지 않습니다. 다시 로그인하세요")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: samlBrowserCookie, Value: "", Path: "/api/v1/auth/saml", HttpOnly: true, Secure: true, SameSite: http.SameSiteNoneMode, MaxAge: -1})
	var requestID, configurationHash string
	if e = s.DB.QueryRow(ctx, `DELETE FROM saml_authn_requests WHERE state_hash=$1 AND browser_hash=$2 AND expires_at>now() RETURNING request_id,configuration_hash`, digest(state), digest(cookie.Value)).Scan(&requestID, &configurationHash); e != nil {
		apiError(w, 400, "SAML 로그인 요청이 만료되었거나 이미 사용되었습니다")
		return
	}
	cfg, e := s.settings(ctx)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if !boolean(cfg, "saml_enabled") || configurationHash != identityConfigurationHash(cfg, "saml") {
		apiError(w, 409, "SAML 설정이 변경되었습니다. 다시 로그인하세요")
		return
	}
	sp, e := samlConfiguration(cfg, true)
	if e != nil {
		apiError(w, 503, e.Error())
		return
	}
	raw, e := base64.StdEncoding.DecodeString(encoded)
	if e != nil || len(raw) > 1<<20 {
		apiError(w, 400, "SAML 응답 인코딩을 확인하세요")
		return
	}
	// Never log InvalidResponseError.PrivateErr/Response: these contain assertions.
	assertion, e := parseSAMLAssertion(sp, raw, requestID)
	if e != nil || assertion == nil || assertion.ID == "" || assertion.Subject == nil || assertion.Subject.NameID == nil || assertion.Subject.NameID.Value == "" || assertion.IssueInstant.After(time.Now().Add(180*time.Second)) {
		s.audit(r, "LOGIN_FAILED", "", map[string]any{"provider": "saml"})
		apiError(w, 401, "SAML 서명·발급자·대상·유효기간·요청 ID 검증에 실패했습니다")
		return
	}
	values := map[string][]string{}
	for _, statement := range assertion.AttributeStatements {
		for _, attribute := range statement.Attributes {
			for _, value := range attribute.Values {
				values[attribute.Name] = append(values[attribute.Name], value.Value)
				if attribute.FriendlyName != "" && attribute.FriendlyName != attribute.Name {
					values[attribute.FriendlyName] = append(values[attribute.FriendlyName], value.Value)
				}
			}
		}
	}
	single := func(name string) string {
		if len(values[name]) != 1 {
			return ""
		}
		return values[name][0]
	}
	identity := externalIdentity{Provider: "saml", Issuer: sp.IDPMetadata.EntityID, Subject: assertion.Subject.NameID.Value, Email: single(str(cfg, "saml_email_attribute")), Name: single(str(cfg, "saml_name_attribute")), Groups: values[str(cfg, "saml_groups_attribute")], VerifiedEmail: true, AutoRegister: boolean(cfg, "saml_auto_register"), ConfigurationHash: configurationHash}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, `INSERT INTO saml_assertion_replays(issuer,assertion_hash,expires_at) VALUES($1,$2,now()+interval '24 hours')`, identity.Issuer, digest(assertion.ID)); e != nil {
		apiError(w, 401, "이미 사용된 SAML 응답입니다")
		return
	}
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
	s.audit(r, "LOGIN", uid, map[string]any{"provider": "saml", "issuer": identity.Issuer})
	http.Redirect(w, r, "/app", http.StatusFound)
}

// This SP advertises signed HTTP-POST assertions over HTTPS, not artifact or
// encrypted-assertion bindings. Bound XML structure before third-party parsing;
// recover malformed schema panics without logging assertion contents.
func parseSAMLAssertion(sp *saml.ServiceProvider, raw []byte, requestID string) (out *saml.Assertion, err error) {
	defer func() {
		if recover() != nil {
			out = nil
			err = errors.New("invalid SAML assertion structure")
		}
	}()
	decoder := xml.NewDecoder(bytes.NewReader(raw))
	depth, count, assertions := 0, 0, 0
	for {
		token, e := decoder.Token()
		if e == io.EOF {
			break
		}
		if e != nil {
			return nil, e
		}
		switch value := token.(type) {
		case xml.StartElement:
			depth++
			count++
			if depth > 64 || count > 20000 {
				return nil, errors.New("SAML XML structure limit")
			}
			if count == 1 && (value.Name.Local != "Response" || value.Name.Space != "urn:oasis:names:tc:SAML:2.0:protocol") {
				return nil, errors.New("SAML response root required")
			}
			if value.Name.Space == "urn:oasis:names:tc:SAML:2.0:assertion" && value.Name.Local == "EncryptedAssertion" {
				return nil, errors.New("configure signed non-encrypted HTTP POST assertions")
			}
			if value.Name.Space == "urn:oasis:names:tc:SAML:2.0:assertion" && value.Name.Local == "Assertion" {
				assertions++
				if assertions > 1 {
					return nil, errors.New("exactly one SAML assertion required")
				}
			}
			if value.Name.Space == "http://www.w3.org/2000/09/xmldsig#" && (value.Name.Local == "SignatureMethod" || value.Name.Local == "DigestMethod") {
				for _, a := range value.Attr {
					if a.Name.Local == "Algorithm" && (strings.Contains(strings.ToLower(a.Value), "sha1") || strings.Contains(strings.ToLower(a.Value), "md5")) {
						return nil, errors.New("obsolete SAML signature algorithm")
					}
				}
			}
		case xml.EndElement:
			depth--
		case xml.Directive:
			return nil, errors.New("SAML XML directives are not allowed")
		}
	}
	if assertions != 1 {
		return nil, errors.New("exactly one signed assertion required")
	}
	out, err = sp.ParseXMLResponse(raw, []string{requestID}, sp.AcsURL)
	if err != nil {
		return nil, err
	}
	if out.Subject == nil || len(out.Subject.SubjectConfirmations) == 0 || out.Conditions == nil || out.Conditions.NotOnOrAfter.IsZero() {
		return nil, errors.New("SAML bearer confirmation and expiry required")
	}
	for _, confirmation := range out.Subject.SubjectConfirmations {
		data := confirmation.SubjectConfirmationData
		if confirmation.Method != "urn:oasis:names:tc:SAML:2.0:cm:bearer" || data == nil || data.NotOnOrAfter.IsZero() || data.InResponseTo != requestID || data.Recipient != sp.AcsURL.String() || (!data.NotBefore.IsZero() && data.NotBefore.After(time.Now().Add(180*time.Second))) {
			return nil, errors.New("SAML bearer confirmation mismatch")
		}
	}
	return out, nil
}

func (s *Server) identityStoreSettings(ctx context.Context, tx pgx.Tx, cfg map[string]any, uid string) error {
	stored := map[string]any{}
	for key, value := range cfg {
		stored[key] = value
	}
	for _, key := range secretSettings {
		if value := str(stored, key); value != "" {
			sealed, e := s.encrypt(value)
			if e != nil {
				return e
			}
			stored[key] = sealed
		}
	}
	if _, e := tx.Exec(ctx, `INSERT INTO settings_history(id,user_id,data) SELECT $1,$2,data FROM settings WHERE id=1`, newID(), uid); e != nil {
		return e
	}
	_, e := tx.Exec(ctx, `UPDATE settings SET data=$1 WHERE id=1`, jsonValue(stored))
	return e
}

func (s *Server) samlRotateCertificate(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Confirm bool `json:"confirm"`
	}
	if decode(r, &in) != nil {
		apiError(w, 400, "인증서 생성 요청을 확인하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	var raw []byte
	if e = tx.QueryRow(r.Context(), `SELECT data FROM settings WHERE id=1 FOR UPDATE`).Scan(&raw); e != nil {
		respond(w, nil, e)
		return
	}
	cfg, e := s.decodeSettings(raw)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if str(cfg, "saml_sp_private_key") != "" && !in.Confirm {
		apiError(w, 409, "인증서를 회전하면 IdP에 새 metadata를 등록해야 합니다. confirm=true로 확인하세요")
		return
	}
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		respond(w, nil, e)
		return
	}
	serial, e := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if e != nil {
		respond(w, nil, e)
		return
	}
	now := time.Now()
	certificate := &x509.Certificate{SerialNumber: serial, Subject: pkix.Name{CommonName: "madi SAML Service Provider"}, NotBefore: now.Add(-5 * time.Minute), NotAfter: now.AddDate(1, 0, 0), KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment, BasicConstraintsValid: true}
	der, e := x509.CreateCertificate(rand.Reader, certificate, certificate, &key.PublicKey, key)
	if e != nil {
		respond(w, nil, e)
		return
	}
	private, e := x509.MarshalPKCS8PrivateKey(key)
	if e != nil {
		respond(w, nil, e)
		return
	}
	cfg["saml_sp_certificate"] = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	cfg["saml_sp_private_key"] = string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: private}))
	if e = s.identityStoreSettings(r.Context(), tx, cfg, current(r).ID); e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "SAML_CERTIFICATE_ROTATE", "system", nil)
	jsonResponse(w, 200, map[string]any{"certificate": cfg["saml_sp_certificate"], "expires_at": certificate.NotAfter, "metadata_url": strings.TrimRight(str(cfg, "site_url"), "/") + "/api/v1/auth/saml/metadata"})
}
