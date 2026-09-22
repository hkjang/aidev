package server

import (
	"context"
	"errors"
	"net/http"
	"slices"
)

var errAIStreamChanged = errors.New("AI 처리 중 권한·참조 문서·공급자 설정이 변경되었습니다. 응답을 지우고 현재 권한으로 다시 요청하세요")

func (s *Server) validateAIStream(r *http.Request, initial *Principal, wid string, sources []aiSource, settings map[string]any) error {
	ctx := r.Context()
	var p *Principal
	var e error
	if initial.TokenID != "" && r.Header.Get("Authorization") != "" {
		p, e = s.tokenPrincipal(r.WithContext(context.WithValue(ctx, integrationCountedTokenKey{}, initial.TokenID)))
	} else if wid != "" {
		p, e = s.workerPrincipal(ctx, initial.ID, initial.TokenID, wid)
	} else {
		p = &Principal{}
		e = s.DB.QueryRow(ctx, "SELECT id::text,email,name,role,kind FROM users WHERE id=$1 AND NOT disabled", initial.ID).Scan(&p.ID, &p.Email, &p.Name, &p.Role, &p.Kind)
	}
	if e != nil || p == nil || p.ID != initial.ID {
		return errAIStreamChanged
	}
	if cookie, err := r.Cookie("madi_session"); err == nil && initial.TokenID == "" {
		var active bool
		if s.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM sessions WHERE token_hash=$1 AND user_id=$2 AND expires_at>now())", digest(cookie.Value), p.ID).Scan(&active) != nil || !active {
			return errAIStreamChanged
		}
	}
	if initial.ScopeRestricted {
		currentScopes := []string{}
		for _, scope := range initial.Scopes {
			if hasIntegrationScope(p, scope) {
				currentScopes = append(currentScopes, scope)
			}
		}
		p.ScopeRestricted = true
		p.Scopes = currentScopes
		p.PluginID = initial.PluginID
		p.WorkspaceID = initial.WorkspaceID
	}
	if initial.PluginID != "" {
		_, caps, err := s.pluginGrant(r, initial.PluginID, wid)
		if err != nil {
			return errAIStreamChanged
		}
		p.Scopes = slices.DeleteFunc(p.Scopes, func(scope string) bool { return !slices.Contains(caps, scope) })
		p.ScopeRestricted = true
	}
	if !hasIntegrationScope(p, "ai:execute") || (len(sources) > 0 && !hasIntegrationScope(p, "document:read")) || (wid != "" && !s.canWorkspace(ctx, p, wid, false)) {
		return errAIStreamChanged
	}
	if len(sources) > 0 {
		var count int
		if s.DB.QueryRow(ctx, `SELECT count(*) FROM jsonb_to_recordset($2::jsonb) AS x(id uuid,version integer) JOIN documents d ON d.id=x.id WHERE d.deleted_at IS NULL AND d.version=x.version AND madi_document_allowed($1,d.id,false)`, p.ID, jsonValue(sources)).Scan(&count) != nil || count != len(sources) {
			return errAIStreamChanged
		}
	}
	fresh, err := s.effectiveSettings(ctx, wid)
	if err != nil {
		return errAIStreamChanged
	}
	for _, key := range append([]string{"ai_enabled", "ai_base_url", "ai_model", "ai_api_key", "ai_max_tokens", "ai_system_prompt"}, ragSettingKeys()...) {
		if string(jsonValue(fresh[key])) != string(jsonValue(settings[key])) {
			return errAIStreamChanged
		}
	}
	if e := s.validateRAGSourceGrants(ctx, sources, fresh); e != nil {
		return errAIStreamChanged
	}
	if e := s.validateAttachmentAISources(ctx, p, sources); e != nil {
		return errAIStreamChanged
	}
	return nil
}
