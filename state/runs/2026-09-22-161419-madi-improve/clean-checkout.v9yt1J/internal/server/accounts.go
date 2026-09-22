package server

import (
	"net/http"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const userJSON = "jsonb_build_object('id',id,'email',email,'name',name,'role',role,'kind',kind,'disabled',disabled,'preferences',preferences,'created_at',created_at)"

func (s *Server) user(r *http.Request, id string) (map[string]any, error) {
	return s.one(r.Context(), "SELECT "+userJSON+" FROM users WHERE id=$1", id)
}
func (s *Server) registerCore() {
	s.mux.HandleFunc("GET /api/v1/openapi.json", s.openAPI)
	s.mux.HandleFunc("GET /api/v1/public", func(w http.ResponseWriter, r *http.Request) {
		cfg, e := s.settings(r.Context())
		if e != nil {
			respond(w, nil, e)
			return
		}
		var runbookEnabled bool
		if e = s.DB.QueryRow(r.Context(), "SELECT enabled FROM runbook_settings WHERE id=1").Scan(&runbookEnabled); e != nil {
			respond(w, nil, e)
			return
		}
		jsonResponse(w, 200, map[string]any{"name": cfg["site_name"], "version": s.Version, "oidc_enabled": cfg["oidc_enabled"], "oidc_auto_login": boolean(cfg, "oidc_enabled") && boolean(cfg, "oidc_auto_login"), "ldap_enabled": boolean(cfg, "ldap_enabled"), "saml_enabled": boolean(cfg, "saml_enabled"), "signup_enabled": cfg["signup_enabled"], "approval_enabled": cfg["approval_enabled"], "reviewer_role": cfg["reviewer_role"], "runbook_enabled": runbookEnabled})
	})
	s.mux.HandleFunc("POST /api/v1/auth/login", s.login)
	s.handle("POST /api/v1/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		if c, e := r.Cookie("madi_session"); e == nil {
			s.DB.Exec(r.Context(), "DELETE FROM sessions WHERE token_hash=$1", digest(c.Value))
		}
		http.SetCookie(w, &http.Cookie{Name: "madi_session", Value: "", Path: "/", HttpOnly: true, MaxAge: -1, SameSite: http.SameSiteLaxMode})
		s.audit(r, "LOGOUT", "", nil)
		jsonResponse(w, 200, map[string]bool{"ok": true})
	})
	s.handle("GET /api/v1/auth/me", func(w http.ResponseWriter, r *http.Request) { v, e := s.user(r, current(r).ID); respond(w, v, e) })
	s.handle("GET /api/v1/profile", func(w http.ResponseWriter, r *http.Request) { v, e := s.user(r, current(r).ID); respond(w, v, e) })
	s.handle("PUT /api/v1/profile", s.updateProfile)
	s.handle("GET /api/v1/workspaces", func(w http.ResponseWriter, r *http.Request) {
		p := current(r)
		v, e := s.rows(r.Context(), "SELECT to_jsonb(w)||jsonb_build_object('role',m.role) FROM workspaces w JOIN workspace_members m ON w.id=m.workspace_id WHERE m.user_id=$1 AND ($2='' OR w.id::text=$2) ORDER BY w.created_at", p.ID, p.WorkspaceID)
		respond(w, v, e)
	})
	s.handle("POST /api/v1/workspaces", s.createWorkspace)
	s.handle("GET /api/v1/workspaces/{id}/members", func(w http.ResponseWriter, r *http.Request) {
		if !s.canWorkspace(r.Context(), current(r), r.PathValue("id"), false) {
			apiError(w, 403, "워크스페이스 접근 권한이 없습니다")
			return
		}
		v, e := s.rows(r.Context(), "SELECT jsonb_build_object('id',u.id,'user_id',u.id,'email',u.email,'name',u.name,'role',m.role) FROM workspace_members m JOIN users u ON u.id=m.user_id WHERE m.workspace_id=$1 ORDER BY u.name", r.PathValue("id"))
		respond(w, v, e)
	})
	s.handle("PUT /api/v1/workspaces/{id}/members", s.updateMember)
	s.handle("DELETE /api/v1/workspaces/{id}/members/{userId}", s.deleteMember)
	s.handle("GET /api/v1/notifications", func(w http.ResponseWriter, r *http.Request) {
		v, e := s.rows(r.Context(), "SELECT to_jsonb(n) FROM notifications n WHERE user_id=$1 AND (document_id IS NULL OR EXISTS(SELECT 1 FROM documents d WHERE d.id=n.document_id AND d.deleted_at IS NULL AND madi_document_allowed($1,d.id,false))) ORDER BY created_at DESC LIMIT 100", current(r).ID)
		respond(w, v, e)
	})
	s.handle("POST /api/v1/notifications/{id}/read", func(w http.ResponseWriter, r *http.Request) {
		_, e := s.DB.Exec(r.Context(), "UPDATE notifications SET read_at=now() WHERE id=$1 AND user_id=$2", r.PathValue("id"), current(r).ID)
		respond(w, map[string]bool{"ok": true}, e)
	})
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if !s.sameOrigin(r) {
		apiError(w, 403, "허용되지 않은 요청 출처입니다")
		return
	}
	var in struct{ Email, Password string }
	if decode(r, &in) != nil || len(in.Email) > 254 || len(in.Password) > 256 {
		apiError(w, 400, "아이디와 비밀번호를 확인하세요")
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	ip := integrationClientIP(r)
	key := "account|" + ip + "|" + in.Email
	if !s.allowLoginAttempt(ip, in.Email, time.Now()) {
		w.Header().Set("Retry-After", "900")
		apiError(w, 429, "로그인 시도가 너무 많습니다. 15분 후 다시 시도하세요")
		return
	}
	var id, hash string
	e := s.DB.QueryRow(r.Context(), "SELECT id,password_hash FROM users WHERE lower(email)=$1 AND NOT disabled AND kind='user'", strings.ToLower(strings.TrimSpace(in.Email))).Scan(&id, &hash)
	if e != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		s.audit(r, "LOGIN_FAILED", "", map[string]string{"email": in.Email})
		apiError(w, 401, "아이디 또는 비밀번호가 올바르지 않습니다")
		return
	}
	if e = s.createSession(w, r, id); e != nil {
		respond(w, nil, e)
		return
	}
	s.limiterMu.Lock()
	delete(s.attempts, key)
	if ipAttempt, exists := s.attempts["ip|"+ip]; exists && ipAttempt.count > 0 {
		ipAttempt.count--
		s.attempts["ip|"+ip] = ipAttempt
	}
	s.limiterMu.Unlock()
	v, e := s.user(r, id)
	s.audit(r, "LOGIN", id, nil)
	respond(w, v, e)
}

const maxLoginLimiterEntries = 10000

func (s *Server) allowLoginAttempt(ip, email string, now time.Time) bool {
	s.limiterMu.Lock()
	defer s.limiterMu.Unlock()
	if s.attempts == nil {
		s.attempts = map[string]attempt{}
	}
	keys := []string{"ip|" + ip, "account|" + ip + "|" + strings.ToLower(strings.TrimSpace(email))}
	limits := []int{100, 15}
	if len(s.attempts) >= maxLoginLimiterEntries-2 && (s.limiterSweep.IsZero() || now.Sub(s.limiterSweep) >= time.Minute) {
		s.limiterSweep = now
		for key, value := range s.attempts {
			if !now.Before(value.until) {
				delete(s.attempts, key)
			}
		}
	}
	for index, key := range keys {
		value, exists := s.attempts[key]
		if !exists && len(s.attempts) >= maxLoginLimiterEntries {
			return false
		}
		if !now.Before(value.until) {
			value = attempt{until: now.Add(15 * time.Minute)}
		}
		if value.count >= limits[index] {
			return false
		}
		value.count++
		s.attempts[key] = value
	}
	return true
}
func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request) {
	var in map[string]any
	if decode(r, &in) != nil {
		apiError(w, 400, "입력값을 확인하세요")
		return
	}
	p := current(r)
	if expected, supplied := in["expected_user_id"]; supplied {
		id, ok := expected.(string)
		if !ok || id != p.ID {
			apiError(w, 409, "로그인 사용자가 변경되었습니다. 개인 설정을 다시 불러오세요")
			return
		}
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if v, ok := in["name"].(string); ok {
		if len(strings.TrimSpace(v)) < 1 || len(v) > 100 {
			apiError(w, 400, "이름은 1~100자여야 합니다")
			return
		}
		if _, e = tx.Exec(r.Context(), "UPDATE users SET name=$1 WHERE id=$2", v, p.ID); e != nil {
			respond(w, nil, e)
			return
		}
	}
	if raw, supplied := in["preferences"]; supplied {
		v, ok := raw.(map[string]any)
		if !ok {
			apiError(w, 400, "개인 설정은 객체여야 합니다")
			return
		}
		if err := validateProfilePreferences(v); err != nil {
			apiError(w, 400, err.Error())
			return
		}
		var previous map[string]any
		if err := tx.QueryRow(r.Context(), "SELECT preferences FROM users WHERE id=$1 FOR UPDATE", p.ID).Scan(&previous); err != nil {
			respond(w, nil, err)
			return
		}
		for key, value := range v {
			previous[key] = value
		}
		if len(jsonValue(previous)) > 32000 {
			apiError(w, 400, "저장된 개인 설정 전체 크기가 32KB를 초과합니다")
			return
		}
		if _, e = tx.Exec(r.Context(), "UPDATE users SET preferences=preferences||$1::jsonb WHERE id=$2", jsonValue(v), p.ID); e != nil {
			respond(w, nil, e)
			return
		}
	}
	if password := str(in, "password"); password != "" {
		if len(password) < 12 || len(password) > 72 {
			apiError(w, 400, "비밀번호는 12~72바이트여야 합니다")
			return
		}
		var old string
		tx.QueryRow(r.Context(), "SELECT password_hash FROM users WHERE id=$1", p.ID).Scan(&old)
		if bcrypt.CompareHashAndPassword([]byte(old), []byte(str(in, "current_password"))) != nil {
			apiError(w, 400, "현재 비밀번호를 확인하세요")
			return
		}
		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if _, e = tx.Exec(r.Context(), "UPDATE users SET password_hash=$1 WHERE id=$2", string(hash), p.ID); e != nil {
			respond(w, nil, e)
			return
		}
		if _, e = tx.Exec(r.Context(), "DELETE FROM sessions WHERE user_id=$1", p.ID); e != nil {
			respond(w, nil, e)
			return
		}
	}
	if e = tx.Commit(r.Context()); e != nil {
		respond(w, nil, e)
		return
	}
	if str(in, "password") != "" {
		if e = s.createSession(w, r, p.ID); e != nil {
			respond(w, nil, e)
			return
		}
	}
	s.audit(r, "PROFILE_UPDATE", p.ID, nil)
	v, e := s.user(r, p.ID)
	respond(w, v, e)
}
func (s *Server) createWorkspace(w http.ResponseWriter, r *http.Request) {
	if current(r).Role == "viewer" {
		apiError(w, 403, "워크스페이스 생성 권한이 없습니다")
		return
	}
	var in struct{ Name string }
	if decode(r, &in) != nil || strings.TrimSpace(in.Name) == "" || len(in.Name) > 160 {
		apiError(w, 400, "워크스페이스 이름을 입력하세요")
		return
	}
	id := newID()
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	_, e = tx.Exec(r.Context(), "INSERT INTO workspaces(id,name,slug) VALUES($1,$2,$3)", id, in.Name, "w-"+id[:8])
	if e == nil {
		_, e = tx.Exec(r.Context(), "INSERT INTO workspace_members VALUES($1,$2,'owner')", id, current(r).ID)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "WORKSPACE_CREATE", id, nil)
	v, e := s.one(r.Context(), "SELECT to_jsonb(w)||'{\"role\":\"owner\"}'::jsonb FROM workspaces w WHERE id=$1", id)
	respond(w, v, e)
}
func (s *Server) workspaceAdmin(r *http.Request, id string) bool {
	if !validID(id) || current(r).TokenID != "" || current(r).ScopeRestricted || current(r).Role == "viewer" {
		return false
	}
	var ok bool
	s.DB.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND role IN('owner','admin'))", id, current(r).ID).Scan(&ok)
	return ok
}
func (s *Server) updateMember(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.workspaceAdmin(r, id) {
		apiError(w, 403, "워크스페이스 관리자 권한이 필요합니다")
		return
	}
	var in struct{ Email, Role string }
	if decode(r, &in) != nil || !oneOf(in.Role, "admin", "editor", "commenter", "viewer") {
		apiError(w, 400, "사용자와 역할을 확인하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if _, e = tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(hashtextextended($1,618))`, id); e != nil {
		respond(w, nil, e)
		return
	}
	var authorized bool
	if e = tx.QueryRow(r.Context(), `SELECT true FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND role IN ('owner','admin') FOR SHARE`, id, current(r).ID).Scan(&authorized); e != nil {
		apiError(w, 403, "워크스페이스 관리자 권한이 변경되었습니다")
		return
	}
	var uid string
	if tx.QueryRow(r.Context(), "SELECT id FROM users WHERE lower(email)=$1 AND NOT disabled", strings.ToLower(strings.TrimSpace(in.Email))).Scan(&uid) != nil {
		apiError(w, 404, "먼저 서비스 관리에서 사용자를 등록하세요")
		return
	}
	tag, e := tx.Exec(r.Context(), "INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,$3) ON CONFLICT(workspace_id,user_id) DO UPDATE SET role=EXCLUDED.role WHERE workspace_members.role!='owner'", id, uid, in.Role)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if tag.RowsAffected() == 0 {
		apiError(w, 400, "소유자 역할은 변경할 수 없습니다")
		return
	}
	if e = s.identityManualMembership(r.Context(), tx, id, uid); e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "PERMISSION_CHANGE", id, map[string]string{"user_id": uid, "role": in.Role})
	respond(w, map[string]string{"user_id": uid, "role": in.Role}, e)
}
func (s *Server) deleteMember(w http.ResponseWriter, r *http.Request) {
	id, uid := r.PathValue("id"), r.PathValue("userId")
	if !s.workspaceAdmin(r, id) {
		apiError(w, 403, "워크스페이스 관리자 권한이 필요합니다")
		return
	}
	if !validID(uid) {
		apiError(w, 400, "사용자 ID가 올바르지 않습니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if _, e = tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(hashtextextended($1,618))`, id); e != nil {
		respond(w, nil, e)
		return
	}
	var authorized bool
	if e = tx.QueryRow(r.Context(), `SELECT true FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND role IN ('owner','admin') FOR SHARE`, id, current(r).ID).Scan(&authorized); e != nil {
		apiError(w, 403, "워크스페이스 관리자 권한이 변경되었습니다")
		return
	}
	tag, e := tx.Exec(r.Context(), "DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND role!='owner'", id, uid)
	if e == nil && tag.RowsAffected() == 0 {
		apiError(w, 400, "소유자는 제거할 수 없거나 사용자가 없습니다")
		return
	}
	if e == nil {
		e = s.identityManualMembership(r.Context(), tx, id, uid)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "PERMISSION_CHANGE", id, map[string]string{"removed_user": uid})
	respond(w, map[string]bool{"ok": true}, e)
}
func oneOf(v string, values ...string) bool {
	for _, x := range values {
		if x == v {
			return true
		}
	}
	return false
}
func validEmail(v string) bool {
	a, e := mail.ParseAddress(v)
	return e == nil && a.Address == v && len(v) <= 254
}
