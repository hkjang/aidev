package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

//go:embed identity.sql
var identitySchema string

//go:embed identity_scim.sql
var identitySCIMSchema string

func (s *Server) migrateIdentity(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, identitySchema+identitySCIMSchema)
	return e
}

func defaultIdentitySettings() map[string]any {
	return map[string]any{
		"identity_link_policy": "manual", "identity_group_mappings": []any{}, "oidc_groups_claim": "groups",
		"ldap_enabled": false, "ldap_url": "", "ldap_starttls": true, "ldap_ca_pem": "", "ldap_bind_dn": "", "ldap_bind_password": "", "ldap_base_dn": "", "ldap_user_filter": "(&(objectClass=person)(uid={username}))", "ldap_subject_attribute": "entryUUID", "ldap_email_attribute": "mail", "ldap_name_attribute": "displayName", "ldap_groups_attribute": "memberOf", "ldap_auto_register": false,
		"saml_enabled": false, "saml_idp_metadata": "", "saml_sp_private_key": "", "saml_sp_certificate": "", "saml_email_attribute": "email", "saml_name_attribute": "displayName", "saml_groups_attribute": "groups", "saml_auto_register": false,
		"scim_enabled": false,
	}
}

func validateIdentitySettings(cfg map[string]any) error {
	for key, initial := range defaultIdentitySettings() {
		switch initial.(type) {
		case bool:
			if _, ok := cfg[key].(bool); !ok {
				return fmt.Errorf("%s는 true/false여야 합니다", key)
			}
		case string:
			if _, ok := cfg[key].(string); !ok {
				return fmt.Errorf("%s는 문자열이어야 합니다", key)
			}
		}
	}
	if !oneOf(str(cfg, "identity_link_policy"), "manual", "verified_email_non_admin") {
		return errors.New("외부 계정 연결 정책을 확인하세요")
	}
	mappings, ok := cfg["identity_group_mappings"].([]any)
	if !ok || len(mappings) > 1000 {
		return errors.New("그룹 매핑은 최대 1,000개 배열입니다")
	}
	for _, raw := range mappings {
		m, ok := raw.(map[string]any)
		if !ok || !oneOf(str(m, "provider"), "oidc", "ldap", "saml", "scim") || str(m, "group") == "" || len(str(m, "group")) > 2000 || !validID(str(m, "workspace_id")) || !oneOf(str(m, "role"), "admin", "editor", "commenter", "viewer") {
			return errors.New("그룹·대상 워크스페이스·역할을 확인하세요. 서비스 관리자/owner 매핑은 허용하지 않습니다")
		}
	}
	for _, key := range []string{"oidc_groups_claim", "ldap_subject_attribute", "ldap_email_attribute", "ldap_name_attribute", "ldap_groups_attribute", "saml_email_attribute", "saml_name_attribute", "saml_groups_attribute"} {
		v := str(cfg, key)
		if v == "" || len(v) > 250 || strings.ContainsAny(v, "\x00\r\n") {
			return fmt.Errorf("%s 속성 이름을 확인하세요", key)
		}
	}
	if boolean(cfg, "ldap_enabled") {
		if _, _, e := ldapConfiguration(cfg); e != nil {
			return e
		}
	}
	if boolean(cfg, "saml_enabled") {
		if _, e := samlConfiguration(cfg, true); e != nil {
			return e
		}
	}
	return nil
}

type externalIdentity struct {
	Provider, Issuer, Subject, Email, Name string
	Groups                                 []string
	VerifiedEmail, AutoRegister            bool
	ConfigurationHash                      string
}

func identityLoginError(w http.ResponseWriter, err error) {
	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		apiError(w, 503, "외부 계정 연결을 완료하지 못했습니다. 잠시 후 다시 시도하세요")
		return
	}
	apiError(w, 403, err.Error())
}

func identityConfigurationHash(cfg map[string]any, provider string) string {
	values := map[string]any{"site_url": cfg["site_url"]}
	for key, value := range cfg {
		if strings.HasPrefix(key, provider+"_") {
			values[key] = value
		}
	}
	return digest(string(jsonValue(values)))
}

func identityClaimGroups(claims map[string]any, path string) ([]string, error) {
	var value any = claims
	for _, part := range strings.Split(path, ".") {
		object, ok := value.(map[string]any)
		if !ok {
			return []string{}, nil
		}
		value = object[part]
	}
	if value == nil {
		return []string{}, nil
	}
	if text, ok := value.(string); ok {
		return []string{text}, nil
	}
	items, ok := value.([]any)
	if !ok || len(items) > 1000 {
		return nil, errors.New("SSO 그룹 속성은 최대 1,000개 문자열 배열이어야 합니다")
	}
	out := []string{}
	for _, item := range items {
		group, ok := item.(string)
		if !ok || len(group) > 2000 {
			return nil, errors.New("SSO 그룹 속성을 확인하세요")
		}
		out = append(out, group)
	}
	return out, nil
}

// Explicit provider/issuer/subject bindings are authoritative. Email matching
// is opt-in, excludes administrators and service accounts, and never merges
// subjects already bound to a different external identity.
func (s *Server) identityUser(ctx context.Context, tx pgx.Tx, cfg map[string]any, identity externalIdentity) (string, error) {
	// Serializes the short local provisioning transaction against policy edits;
	// directory/network authentication is completed before this lock is taken.
	if _, e := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(729421587)`); e != nil {
		return "", e
	}
	var settingsJSON []byte
	if e := tx.QueryRow(ctx, `SELECT data FROM settings WHERE id=1`).Scan(&settingsJSON); e != nil {
		return "", e
	}
	currentSettings, e := s.decodeSettings(settingsJSON)
	if e != nil {
		return "", e
	}
	cfg = currentSettings
	if identity.Provider != "scim" && !boolean(cfg, identity.Provider+"_enabled") {
		return "", errors.New("외부 로그인 제공자가 비활성화되었습니다")
	}
	if identity.ConfigurationHash != "" && identity.ConfigurationHash != identityConfigurationHash(cfg, identity.Provider) {
		return "", errors.New("외부 로그인 설정이 변경되었습니다. 다시 로그인하세요")
	}
	identity.AutoRegister = boolean(cfg, identity.Provider+"_auto_register")
	identity.Email = strings.ToLower(strings.TrimSpace(identity.Email))
	identity.Name = strings.TrimSpace(identity.Name)
	address, e := mail.ParseAddress(identity.Email)
	if e != nil || address.Address != identity.Email || len(identity.Email) > 254 || identity.Subject == "" || len(identity.Subject) > 2000 || len(identity.Issuer) > 4000 {
		return "", errors.New("외부 계정의 고유 ID와 이메일 주소를 확인하세요")
	}
	if identity.Name == "" {
		identity.Name = identity.Email
	}
	if len(identity.Name) > 250 {
		return "", errors.New("외부 계정 표시 이름이 너무 깁니다")
	}
	if len(identity.Groups) > 1000 {
		return "", errors.New("외부 계정 그룹은 최대 1,000개입니다")
	}
	for _, group := range identity.Groups {
		if len(group) > 2000 {
			return "", errors.New("외부 그룹 이름이 너무 깁니다")
		}
	}
	var uid, kind, role string
	var disabled bool
	e = tx.QueryRow(ctx, `SELECT u.id::text,u.kind,u.role,u.disabled FROM identity_links i JOIN users u ON u.id=i.user_id WHERE i.provider=$1 AND i.issuer=$2 AND i.subject=$3 FOR UPDATE OF u,i`, identity.Provider, identity.Issuer, identity.Subject).Scan(&uid, &kind, &role, &disabled)
	if errors.Is(e, pgx.ErrNoRows) && identity.Provider == "oidc" {
		// Existing madi installations retain their already-linked OIDC subjects.
		e = tx.QueryRow(ctx, `SELECT id::text,kind,role,disabled FROM users WHERE oidc_subject=$1 FOR UPDATE`, identity.Issuer+"|"+identity.Subject).Scan(&uid, &kind, &role, &disabled)
	}
	if errors.Is(e, pgx.ErrNoRows) {
		e = tx.QueryRow(ctx, `SELECT id::text,kind,role,disabled FROM users WHERE lower(email)=$1 FOR UPDATE`, identity.Email).Scan(&uid, &kind, &role, &disabled)
		if e == nil {
			if str(cfg, "identity_link_policy") != "verified_email_non_admin" || !identity.VerifiedEmail || role == "admin" || kind != "user" || disabled {
				return "", errors.New("같은 이메일의 계정이 있습니다. 관리자가 제공자·고유 ID를 명시적으로 연결해야 합니다")
			}
			var bound bool
			if e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM identity_links WHERE user_id=$1 AND provider=$2 AND issuer=$3)`, uid, identity.Provider, identity.Issuer).Scan(&bound); e != nil {
				return "", e
			}
			if bound {
				return "", errors.New("다른 외부 고유 ID에 연결된 이메일입니다")
			}
		} else if errors.Is(e, pgx.ErrNoRows) {
			if !identity.AutoRegister {
				return "", errors.New("등록되지 않은 외부 계정입니다. 관리자에게 연결을 요청하세요")
			}
			uid = newID()
			kind = "user"
			role = "editor"
			if _, e = tx.Exec(ctx, `INSERT INTO users(id,email,name,role,kind) VALUES($1,$2,$3,'editor','user')`, uid, identity.Email, identity.Name); e != nil {
				return "", e
			}
			wid := newID()
			if _, e = tx.Exec(ctx, `INSERT INTO workspaces(id,name,slug) VALUES($1,$2,$3)`, wid, identity.Name+"의 워크스페이스", "personal-"+wid[:8]); e != nil {
				return "", e
			}
			if _, e = tx.Exec(ctx, `INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,'owner')`, wid, uid); e != nil {
				return "", e
			}
		}
	}
	if e != nil {
		return "", e
	}
	if disabled || kind != "user" {
		return "", errors.New("로그인할 수 없는 계정입니다")
	}
	if _, e = tx.Exec(ctx, `INSERT INTO identity_links(id,user_id,provider,issuer,subject,groups) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(provider,issuer,subject) DO UPDATE SET groups=EXCLUDED.groups,updated_at=now()`, newID(), uid, identity.Provider, identity.Issuer, identity.Subject, jsonValue(identity.Groups)); e != nil {
		return "", e
	}
	if identity.Provider == "oidc" {
		if _, e = tx.Exec(ctx, `UPDATE users SET oidc_subject=$2 WHERE id=$1`, uid, identity.Issuer+"|"+identity.Subject); e != nil {
			return "", e
		}
	}
	if e = s.identitySyncGroups(ctx, tx, cfg, uid, identity.Provider, identity.Issuer, identity.Groups); e != nil {
		return "", e
	}
	return uid, nil
}

func identityRoleRank(role string) int {
	return map[string]int{"viewer": 1, "commenter": 2, "editor": 3, "admin": 4, "owner": 5}[role]
}

// A manually assigned membership always wins. Managed roles are maintained
// separately so removing a group cannot revoke a pre-existing manual grant.
func (s *Server) identitySyncGroups(ctx context.Context, tx pgx.Tx, cfg map[string]any, uid, provider, issuer string, groups []string) error {
	if _, e := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,12854))`, uid); e != nil {
		return e
	}
	if _, e := tx.Exec(ctx, `DELETE FROM identity_grants WHERE user_id=$1 AND provider=$2 AND issuer=$3`, uid, provider, issuer); e != nil {
		return e
	}
	desired := map[string]string{}
	groupSet := map[string]bool{}
	for _, group := range groups {
		groupSet[group] = true
	}
	for _, raw := range cfg["identity_group_mappings"].([]any) {
		m := raw.(map[string]any)
		wid := str(m, "workspace_id")
		if str(m, "provider") != provider || !groupSet[str(m, "group")] || (provider == "scim" && issuer != wid) {
			continue
		}
		role := str(m, "role")
		if identityRoleRank(role) > identityRoleRank(desired[wid]) {
			desired[wid] = role
		}
	}
	for wid, role := range desired {
		if _, e := tx.Exec(ctx, `INSERT INTO identity_grants(provider,issuer,user_id,workspace_id,role) SELECT $1,$2,$3,id,$5 FROM workspaces WHERE id=$4`, provider, issuer, uid, wid, role); e != nil {
			return e
		}
	}
	rows, e := tx.Query(ctx, `SELECT workspace_id::text FROM identity_grants WHERE user_id=$1 UNION SELECT workspace_id::text FROM identity_managed_members WHERE user_id=$1`, uid)
	if e != nil {
		return e
	}
	var workspaces []string
	for rows.Next() {
		var wid string
		if e = rows.Scan(&wid); e != nil {
			rows.Close()
			return e
		}
		workspaces = append(workspaces, wid)
	}
	rows.Close()
	if rows.Err() != nil {
		return rows.Err()
	}
	slices.Sort(workspaces)
	for _, wid := range workspaces {
		if e = s.identityReconcileMember(ctx, tx, uid, wid); e != nil {
			return e
		}
	}
	return nil
}

func (s *Server) identityReconcileMember(ctx context.Context, tx pgx.Tx, uid, wid string) error {
	var role string
	e := tx.QueryRow(ctx, `SELECT role FROM identity_grants WHERE user_id=$1 AND workspace_id=$2 ORDER BY CASE role WHEN 'admin' THEN 4 WHEN 'editor' THEN 3 WHEN 'commenter' THEN 2 ELSE 1 END DESC LIMIT 1`, uid, wid).Scan(&role)
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return e
	}
	var currentRole, managedRole string
	e = tx.QueryRow(ctx, `SELECT m.role,coalesce(i.role,'') FROM workspace_members m LEFT JOIN identity_managed_members i USING(workspace_id,user_id) WHERE m.workspace_id=$1 AND m.user_id=$2 FOR UPDATE OF m`, wid, uid).Scan(&currentRole, &managedRole)
	if errors.Is(e, pgx.ErrNoRows) {
		if role == "" {
			return nil
		}
		inserted, insertErr := tx.Exec(ctx, `INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, wid, uid, role)
		if insertErr != nil {
			return insertErr
		}
		if inserted.RowsAffected() == 0 {
			return nil
		}
		_, e = tx.Exec(ctx, `INSERT INTO identity_managed_members(workspace_id,user_id,role) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, wid, uid, role)
		return e
	}
	if e != nil {
		return e
	}
	if managedRole == "" {
		return nil
	}
	if currentRole != managedRole {
		_, e = tx.Exec(ctx, `DELETE FROM identity_managed_members WHERE workspace_id=$1 AND user_id=$2`, wid, uid)
		return e
	}
	if role == "" {
		_, e = tx.Exec(ctx, `DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, wid, uid)
		return e
	}
	if _, e = tx.Exec(ctx, `UPDATE workspace_members SET role=$3 WHERE workspace_id=$1 AND user_id=$2`, wid, uid, role); e != nil {
		return e
	}
	_, e = tx.Exec(ctx, `UPDATE identity_managed_members SET role=$3 WHERE workspace_id=$1 AND user_id=$2`, wid, uid, role)
	return e
}

func (s *Server) identityManualMembership(ctx context.Context, tx pgx.Tx, wid, uid string) error {
	_, e := tx.Exec(ctx, `DELETE FROM identity_managed_members WHERE workspace_id=$1 AND user_id=$2`, wid, uid)
	return e
}

func (s *Server) identityRefreshMappings(ctx context.Context, tx pgx.Tx, cfg map[string]any) error {
	if _, e := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(729421587)`); e != nil {
		return e
	}
	rows, e := tx.Query(ctx, `SELECT user_id::text,provider,issuer,groups FROM identity_links ORDER BY user_id,provider,issuer`)
	if e != nil {
		return e
	}
	type identity struct {
		uid, provider, issuer string
		groups                []string
	}
	var all []identity
	for rows.Next() {
		var item identity
		var data []byte
		if e = rows.Scan(&item.uid, &item.provider, &item.issuer, &data); e != nil {
			rows.Close()
			return e
		}
		if e = json.Unmarshal(data, &item.groups); e != nil {
			rows.Close()
			return e
		}
		all = append(all, item)
	}
	rows.Close()
	if rows.Err() != nil {
		return rows.Err()
	}
	for _, item := range all {
		if e = s.identitySyncGroups(ctx, tx, cfg, item.uid, item.provider, item.issuer, item.groups); e != nil {
			return e
		}
	}
	return nil
}

func (s *Server) registerIdentity() {
	s.registerSCIM()
	s.mux.HandleFunc("POST /api/v1/auth/ldap/login", s.ldapLogin)
	s.mux.HandleFunc("GET /api/v1/auth/saml/start", s.samlStart)
	s.mux.HandleFunc("GET /api/v1/auth/saml/metadata", s.samlMetadata)
	s.mux.HandleFunc("POST /api/v1/auth/saml/acs", s.samlACS)
	s.admin("POST /api/v1/admin/identity/saml/certificate", s.samlRotateCertificate)
	s.admin("GET /api/v1/admin/identity/links", func(w http.ResponseWriter, r *http.Request) {
		v, e := s.rows(r.Context(), `SELECT to_jsonb(i)||jsonb_build_object('email',u.email,'name',u.name) FROM identity_links i JOIN users u ON u.id=i.user_id ORDER BY i.created_at DESC LIMIT 10000`)
		respond(w, v, e)
	})
	s.admin("POST /api/v1/admin/identity/links", s.identityLink)
	s.admin("DELETE /api/v1/admin/identity/links/{id}", s.identityUnlink)
}

func (s *Server) identityLink(w http.ResponseWriter, r *http.Request) {
	var in map[string]any
	if decode(r, &in) != nil {
		apiError(w, 400, "외부 계정 연결 값을 확인하세요")
		return
	}
	uid, provider, issuer, subject := str(in, "user_id"), str(in, "provider"), str(in, "issuer"), str(in, "subject")
	if !validID(uid) || !oneOf(provider, "oidc", "ldap", "saml") || issuer == "" || subject == "" || len(issuer) > 4000 || len(subject) > 2000 {
		apiError(w, 400, "사용자·제공자·issuer·고유 ID를 입력하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if _, e = tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(729421587)`); e != nil {
		respond(w, nil, e)
		return
	}
	var kind string
	if e = tx.QueryRow(r.Context(), `SELECT kind FROM users WHERE id=$1 AND NOT disabled FOR UPDATE`, uid).Scan(&kind); e != nil || kind != "user" {
		apiError(w, 400, "활성 사용자 계정만 연결할 수 있습니다")
		return
	}
	id := newID()
	if _, e = tx.Exec(r.Context(), `INSERT INTO identity_links(id,user_id,provider,issuer,subject) VALUES($1,$2,$3,$4,$5)`, id, uid, provider, issuer, subject); e != nil {
		apiError(w, 409, "이미 연결된 제공자 계정 또는 사용자입니다")
		return
	}
	if provider == "oidc" {
		if _, e = tx.Exec(r.Context(), `UPDATE users SET oidc_subject=$2 WHERE id=$1`, uid, issuer+"|"+subject); e != nil {
			respond(w, nil, e)
			return
		}
	}
	if e = tx.Commit(r.Context()); e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "IDENTITY_LINK", uid, map[string]any{"provider": provider, "issuer": issuer})
	jsonResponse(w, 201, map[string]any{"id": id})
}

func (s *Server) identityUnlink(w http.ResponseWriter, r *http.Request) {
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if _, e = tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(729421587)`); e != nil {
		respond(w, nil, e)
		return
	}
	var uid, provider, issuer string
	if e = tx.QueryRow(r.Context(), `DELETE FROM identity_links WHERE id=$1 RETURNING user_id::text,provider,issuer`, r.PathValue("id")).Scan(&uid, &provider, &issuer); e != nil {
		apiError(w, 404, "외부 계정 연결을 찾을 수 없습니다")
		return
	}
	if provider == "scim" {
		apiError(w, 409, "SCIM 관리 계정은 SCIM Users API로 비활성화하거나 삭제하세요")
		return
	}
	if e = s.identitySyncGroups(r.Context(), tx, map[string]any{"identity_group_mappings": []any{}}, uid, provider, issuer, nil); e != nil {
		respond(w, nil, e)
		return
	}
	if provider == "oidc" {
		if _, e = tx.Exec(r.Context(), `UPDATE users SET oidc_subject=NULL WHERE id=$1`, uid); e != nil {
			respond(w, nil, e)
			return
		}
	}
	if _, e = tx.Exec(r.Context(), `DELETE FROM sessions WHERE user_id=$1`, uid); e != nil {
		respond(w, nil, e)
		return
	}
	if e = tx.Commit(r.Context()); e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "IDENTITY_UNLINK", uid, map[string]any{"provider": provider})
	jsonResponse(w, 200, map[string]bool{"ok": true})
}
