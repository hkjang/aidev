package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/analytics"
	"github.com/hkjang/umm/internal/auth"
	"github.com/hkjang/umm/internal/dream"
	"github.com/hkjang/umm/internal/intelligence"
	"github.com/hkjang/umm/internal/presentation"
	"github.com/hkjang/umm/internal/store"
)

const secretMask = "••••••••"

func (s *Server) adminSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.Store.AllSettings(r.Context())
	if err != nil {
		writeError(w, 500, "설정을 불러오지 못했습니다.")
		return
	}
	out := map[string]any{}
	for key, raw := range settings {
		var value map[string]any
		if json.Unmarshal(raw, &value) != nil {
			continue
		}
		for _, field := range secretFields(key) {
			if secret, ok := value[field].(string); ok && secret != "" {
				value[field] = secretMask
				value[field+"_configured"] = true
			}
		}
		out[key] = value
	}
	writeJSON(w, 200, out)
}

func (s *Server) putAdminSetting(w http.ResponseWriter, r *http.Request) {
	section := chiParam(r, "section")
	// The store owns this list. Keeping a second copy here meant a new section
	// had to be added in two places, and forgetting one produced a 404 that looks
	// like the section does not exist rather than like a missed edit.
	if !store.AllowedSetting(section) {
		writeError(w, 404, "알 수 없는 설정 영역입니다.")
		return
	}
	var incoming map[string]any
	if decodeJSON(w, r, &incoming) != nil {
		writeError(w, 400, "설정 형식이 올바르지 않습니다.")
		return
	}
	if err := s.validateSetting(section, incoming); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	for _, field := range secretFields(section) {
		raw, _ := incoming[field].(string)
		if raw == "" || raw == secretMask {
			// Omit a masked secret so Store.PutSetting can merge the latest
			// ciphertext while holding the same lock as master-key rotation.
			delete(incoming, field)
		} else {
			encrypted, err := s.Cipher.Encrypt(raw)
			if err != nil {
				writeError(w, 500, "비밀 값을 암호화하지 못했습니다.")
				return
			}
			incoming[field] = "enc:" + encrypted
		}
		delete(incoming, field+"_configured")
	}
	p := principal(r)
	if err := s.Store.PutSetting(r.Context(), section, incoming, p.User.ID); err != nil {
		writeError(w, 500, "설정을 저장하지 못했습니다.")
		return
	}
	// Cached settings must not outlive the change that an administrator just
	// confirmed, so the derived caches are dropped immediately.
	if section == "ai_gateway" {
		s.Store.InvalidateEmbeddingProvider()
	}
	if section == "security" {
		s.invalidateSecurityPolicy()
	}
	if section == "intelligence" {
		s.Store.InvalidateIntelligenceSettings()
	}
	if section == analytics.SettingKey {
		s.invalidateTracking()
	}
	s.Store.Audit(r.Context(), &p.User.ID, "settings.update", "settings", section, map[string]any{})
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// secretFields names every value in a section that must be masked when read and
// encrypted when written.
//
// One list, used by both the read and the write path. They used to name their
// fields separately, so adding a second secret to a section meant remembering
// two places — and forgetting the read side would return a key in plain text.
func secretFields(section string) []string {
	return store.SecretSettingFields(section)
}

func (s *Server) validateSetting(section string, v map[string]any) error {
	switch section {
	case "general":
		raw := strings.TrimSpace(fmt.Sprint(v["public_url"]))
		u, err := url.Parse(raw)
		if err != nil || !(u.Scheme == "http" || u.Scheme == "https") || u.Host == "" {
			return errors.New("서비스 공개 URL은 http(s) 전체 주소여야 합니다")
		}
		if strings.TrimSpace(fmt.Sprint(v["service_name"])) == "" {
			return errors.New("서비스 이름이 필요합니다")
		}
		if zone := strings.TrimSpace(fmt.Sprint(v["timezone"])); zone == "" {
			return errors.New("서비스 시간대가 필요합니다")
		} else if _, err := time.LoadLocation(zone); err != nil {
			return errors.New("IANA 형식의 올바른 서비스 시간대가 필요합니다")
		}
	case "oidc":
		if enabled, _ := v["enabled"].(bool); enabled {
			raw := strings.TrimSpace(fmt.Sprint(v["issuer_url"]))
			u, err := url.Parse(raw)
			if err != nil || !(u.Scheme == "http" || u.Scheme == "https") || u.Host == "" {
				return errors.New("Keycloak Issuer URL이 올바르지 않습니다")
			}
			if strings.TrimSpace(fmt.Sprint(v["client_id"])) == "" {
				return errors.New("OIDC Client ID가 필요합니다")
			}
		}
		if err := auth.ValidateMCPSettings(v); err != nil {
			return err
		}
	case "handoff":
		return validateHandoffSettings(v)
	case "intelligence":
		// Clamping a bad value would hide the mistake: an administrator who typed
		// 40 into a standard-deviation field needs to be told it is not one,
		// rather than have umm quietly use 1.1 and behave unlike the screen says.
		for _, field := range []struct {
			key      string
			low, max float64
			unit     string
		}{
			{"related_band", 0, 4, "표준편차"},
			{"cluster_band", 0, 4, "표준편차"},
			{"strong_band", 0, 4, "표준편차"},
			{"autolink_band", 0, 4, "표준편차"},
			{"semantic_accuracy_bar", 0, 1, "0~1 비율"},
			{"semantic_purity_bar", 0, 1, "0~1 비율"},
			{"autolink_max_per_run", 1, 100, "개"},
			{"autolink_min_notes", 3, 1000, "개"},
			{"quality_cache_minutes", 1, 1440, "분"},
		} {
			raw, present := v[field.key]
			if !present {
				continue
			}
			number, ok := raw.(float64)
			if !ok || math.IsNaN(number) || number < field.low || number > field.max {
				return fmt.Errorf("%s은(는) %g~%g %s 범위여야 합니다", field.key, field.low, field.max, field.unit)
			}
		}
	case "security":
		scopes, ok := v["api_key_scopes"].([]any)
		if !ok || len(scopes) == 0 {
			return errors.New("하나 이상의 API 키 권한이 필요합니다")
		}
		// The abuse guards are optional in the payload so an older client can
		// still save this section, but a value that is present must be sane:
		// silently clamping one would hide a misconfiguration from the operator.
		for _, guard := range []struct {
			key     string
			low     float64
			high    float64
			message string
		}{
			{key: "login_max_failures", low: 3, high: 100, message: "로그인 실패 허용 횟수는 3~100회여야 합니다"},
			{key: "login_lockout_minutes", low: 1, high: 1440, message: "로그인 잠금 시간은 1~1440분이어야 합니다"},
			{key: "api_rate_per_minute", low: 30, high: 100000, message: "분당 API 요청 한도는 30~100000이어야 합니다"},
			{key: "ai_rate_per_minute", low: 1, high: 600, message: "분당 AI 요청 한도는 1~600이어야 합니다"},
			{key: "ai_daily_limit", low: 0, high: 100000, message: "하루 AI 생성 한도는 0~100000이어야 합니다"},
		} {
			raw, present := v[guard.key]
			if !present {
				continue
			}
			value, ok := raw.(float64)
			if !ok || math.Trunc(value) != value || value < guard.low || value > guard.high {
				return errors.New(guard.message)
			}
		}
	case "workflow":
		actions, ok := v["actions"].([]any)
		if !ok {
			return errors.New("검토 작업 목록이 올바르지 않습니다")
		}
		for _, action := range actions {
			if !slices.Contains([]string{"space_share", "export"}, fmt.Sprint(action)) {
				return errors.New("지원하지 않는 검토 작업입니다")
			}
		}
	case "dream":
		threshold, ok := v["quality_threshold"].(float64)
		if !ok || threshold < 0 || threshold > 1 {
			return errors.New("Dream 품질 기준은 0~1이어야 합니다")
		}
		schedule := fmt.Sprint(v["schedule"])
		if _, err := time.Parse("15:04", schedule); err != nil {
			return errors.New("Dream 생성 시간은 HH:MM 형식이어야 합니다")
		}
		frequency := fmt.Sprint(v["frequency"])
		if !slices.Contains([]string{"daily", "weekdays", "weekends", "custom", "interval"}, frequency) {
			return errors.New("Dream 생성 주기가 올바르지 않습니다")
		}
		if frequency == "custom" {
			days, ok := v["custom_days"].([]any)
			if !ok || len(days) == 0 {
				return errors.New("Dream 생성 요일을 하나 이상 선택해 주세요")
			}
			for _, day := range days {
				n, ok := day.(float64)
				if !ok || n < 1 || n > 7 {
					return errors.New("Dream 생성 요일이 올바르지 않습니다")
				}
			}
		}
		if frequency == "interval" {
			days, ok := v["interval_days"].(float64)
			if !ok || days < 2 || days > 365 {
				return errors.New("Dream N일 간격은 2~365일이어야 합니다")
			}
		}
		tokenLimit, ok := v["token_limit"].(float64)
		if !ok || math.Trunc(tokenLimit) != tokenLimit || tokenLimit < dream.MinTokenLimit || tokenLimit > dream.MaxTokenLimit {
			return fmt.Errorf("AI 응답 Token Limit은 %d~%s 사이의 정수여야 합니다", dream.MinTokenLimit, "262,144")
		}
	case "ai_gateway":
		// Through settingString, so an omitted address reads as unset rather
		// than as the literal "<nil>" — which url.Parse accepts and then fails
		// the scheme check, rejecting the request with "the URL is invalid"
		// when there is no URL at all. embedding_base_url below already guarded
		// against this; base_url did not.
		raw := settingString(v, "base_url")
		if raw != "" {
			u, err := url.Parse(raw)
			if err != nil || !(u.Scheme == "http" || u.Scheme == "https") || u.Host == "" {
				return errors.New("AI Gateway URL이 올바르지 않습니다")
			}
		}
		embedURL := settingString(v, "embedding_base_url")
		if embedURL != "" {
			// Refused rather than ignored. A malformed address here would leave
			// embeddings quietly falling back to the offline algorithm, and the
			// person who typed it would be looking at a saved setting that does
			// nothing.
			u, err := url.Parse(embedURL)
			if err != nil || !(u.Scheme == "http" || u.Scheme == "https") || u.Host == "" {
				return errors.New("임베딩 Gateway URL이 올바르지 않습니다")
			}
		}
		retention, ok := v["log_retention_days"].(float64)
		if !ok || retention < 1 || retention > 3650 {
			return errors.New("AI 로그 보존 기간은 1~3650일이어야 합니다")
		}
		timeout, ok := v["timeout_seconds"].(float64)
		if !ok || math.Trunc(timeout) != timeout || timeout < dream.MinGatewayTimeoutSeconds || timeout > dream.MaxGatewayTimeoutSeconds {
			return fmt.Errorf("AI Gateway Timeout은 %d~%d초 사이의 정수여야 합니다", dream.MinGatewayTimeoutSeconds, dream.MaxGatewayTimeoutSeconds)
		}
		retries, ok := v["max_retries"].(float64)
		if !ok || math.Trunc(retries) != retries || retries < 0 || retries > dream.MaxGatewayRetries {
			return fmt.Errorf("AI Gateway 재시도는 0~%d 사이의 정수여야 합니다", dream.MaxGatewayRetries)
		}
		if model := settingString(v, "embedding_model"); model != "" {
			if raw == "" && embedURL == "" {
				return errors.New("임베딩 모델을 사용하려면 AI Gateway 주소 또는 임베딩 Gateway 주소가 필요합니다")
			}
			if len(model) > 200 {
				return errors.New("임베딩 모델 이름은 200자 이내여야 합니다")
			}
		}
	case "ptium":
		base := settingString(v, "base_url")
		if base != "" {
			// Refused rather than ignored, for the same reason the embedding
			// address is: a saved setting that does nothing looks exactly like
			// one that works until someone tries to make a deck.
			u, err := url.Parse(base)
			if err != nil || !(u.Scheme == "http" || u.Scheme == "https") || u.Host == "" {
				return errors.New("Ptium 주소가 올바르지 않습니다")
			}
		}
		if key := settingString(v, "api_key"); base == "" && key != "" {
			return errors.New("Ptium API 키를 쓰려면 Ptium 주소가 필요합니다")
		}
		if language := settingString(v, "language"); len(language) > 32 {
			return errors.New("Ptium 언어 코드는 32자 이내여야 합니다")
		}
		if template := settingString(v, "template_id"); len(template) > 200 {
			return errors.New("Ptium 템플릿 ID는 200자 이내여야 합니다")
		}
		timeout, ok := v["timeout_seconds"].(float64)
		if !ok || math.Trunc(timeout) != timeout || timeout < 5 || timeout > 300 {
			return errors.New("Ptium Timeout은 5~300초 사이의 정수여야 합니다")
		}
	case analytics.SettingKey:
		// The package that renders the snippet is the one that knows what a
		// complete configuration is, so the rules live there and are read here
		// through the same struct the serving path reads the row into.
		raw, err := json.Marshal(v)
		if err != nil {
			return errors.New("방문 추적 설정 형식이 올바르지 않습니다")
		}
		var config analytics.Config
		if err := json.Unmarshal(raw, &config); err != nil {
			return errors.New("방문 추적 설정 형식이 올바르지 않습니다")
		}
		return config.Validate()
	}
	return nil
}

// settingString reads one setting field as trimmed text.
//
// A missing key formats as the literal "<nil>" through fmt.Sprint, which read
// as a real value would turn an unset address into a saved one. Every caller
// was writing that check out by hand, and one of them had already forgotten it.
func settingString(v map[string]any, key string) string {
	value := strings.TrimSpace(fmt.Sprint(v[key]))
	if value == "<nil>" {
		return ""
	}
	return value
}

func chiParam(r *http.Request, key string) string { return strings.TrimSpace(chi.URLParam(r, key)) }

func (s *Server) testOIDC(w http.ResponseWriter, r *http.Request) {
	if err := s.OIDC.Test(r.Context()); err != nil {
		writeError(w, 400, "OIDC 연결 실패: "+err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "message": "Keycloak Discovery와 클라이언트 설정을 확인했습니다."})
}

func (s *Server) adminUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Store.Pool.Query(r.Context(), `SELECT u.id,u.username,u.display_name,COALESCE(u.email,''),u.role,u.team_id,u.active,COALESCE(t.name,''),u.created_at FROM users u LEFT JOIN teams t ON t.id=u.team_id ORDER BY u.created_at`)
	if err != nil {
		writeError(w, 500, "사용자 목록을 불러오지 못했습니다.")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id uuid.UUID
		var username, display, email, role, teamName string
		var teamID *uuid.UUID
		var active bool
		var created time.Time
		if err := rows.Scan(&id, &username, &display, &email, &role, &teamID, &active, &teamName, &created); err != nil {
			writeError(w, 500, "사용자 목록을 읽지 못했습니다.")
			return
		}
		out = append(out, map[string]any{"id": id, "username": username, "displayName": display, "email": email, "role": role, "teamId": teamID, "teamName": teamName, "active": active, "createdAt": created})
	}
	writeJSON(w, 200, map[string]any{"users": out})
}

func (s *Server) updateUser(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "userID")
	if !ok {
		return
	}
	var body struct {
		Role     string `json:"role"`
		Active   bool   `json:"active"`
		TeamName string `json:"teamName"`
	}
	if decodeJSON(w, r, &body) != nil {
		writeError(w, 400, "사용자 설정 형식이 올바르지 않습니다.")
		return
	}
	if !slices.Contains([]string{"user", "team_lead", "admin"}, body.Role) {
		writeError(w, 400, "역할이 올바르지 않습니다.")
		return
	}
	p := principal(r)
	if id == p.User.ID && (!body.Active || body.Role != "admin") {
		writeError(w, 400, "현재 관리자 계정의 관리자 권한을 제거할 수 없습니다.")
		return
	}
	var teamID *uuid.UUID
	if strings.TrimSpace(body.TeamName) != "" {
		var t uuid.UUID
		if err := s.Store.Pool.QueryRow(r.Context(), `INSERT INTO teams(name) VALUES($1) ON CONFLICT(name) DO UPDATE SET name=EXCLUDED.name RETURNING id`, strings.TrimSpace(body.TeamName)).Scan(&t); err != nil {
			writeError(w, 500, "팀을 저장하지 못했습니다.")
			return
		}
		teamID = &t
	}
	cmd, err := s.Store.Pool.Exec(r.Context(), `UPDATE users SET role=$2,active=$3,team_id=$4,updated_at=now() WHERE id=$1`, id, body.Role, body.Active, teamID)
	if err != nil {
		writeError(w, 500, "사용자를 수정하지 못했습니다.")
		return
	}
	if cmd.RowsAffected() == 0 {
		writeError(w, 404, "사용자를 찾을 수 없습니다.")
		return
	}
	s.Store.Audit(r.Context(), &p.User.ID, "user.update", "user", id.String(), map[string]any{"role": body.Role, "active": body.Active})
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) adminMetrics(w http.ResponseWriter, r *http.Request) {
	var users, active, notes, spaces, pending, comments, onboarded, webhookFailures int64
	err := s.Store.Pool.QueryRow(r.Context(), `SELECT (SELECT count(*) FROM users),(SELECT count(*) FROM users WHERE active),(SELECT count(*) FROM notes WHERE deleted_at IS NULL),(SELECT count(*) FROM spaces),(SELECT count(*) FROM approval_requests WHERE status='pending'),(SELECT count(*) FROM note_comments WHERE deleted_at IS NULL),(SELECT count(*) FROM user_preferences WHERE onboarding_completed_at IS NOT NULL),(SELECT count(*) FROM webhook_deliveries WHERE status='failed' AND attempted_at>=now()-interval '24 hours')`).Scan(&users, &active, &notes, &spaces, &pending, &comments, &onboarded, &webhookFailures)
	if err != nil {
		writeError(w, 500, "운영 지표를 불러오지 못했습니다.")
		return
	}
	dreamMetrics, err := s.Dreams.Metrics(r.Context())
	if err != nil {
		writeError(w, 500, "Dream 운영 지표를 불러오지 못했습니다.")
		return
	}
	var evalRuns, evalPassed int64
	if err := s.Store.Pool.QueryRow(r.Context(), `SELECT count(*),count(*) FILTER (WHERE status='passed') FROM ai_eval_runs WHERE created_at>=now()-interval '30 days'`).Scan(&evalRuns, &evalPassed); err != nil {
		writeError(w, 500, "AI 평가 지표를 불러오지 못했습니다.")
		return
	}
	out := map[string]any{"users": users, "activeUsers": active, "notes": notes, "spaces": spaces, "pendingApprovals": pending, "comments": comments, "onboardedUsers": onboarded, "webhookFailures24h": webhookFailures, "aiEvalRuns30d": evalRuns, "aiEvalPassed30d": evalPassed, "dream": dreamMetrics}
	if s.Metrics != nil {
		out["http"] = s.Metrics.Snapshot()
	}
	if s.Events != nil {
		subscribers, spaces, delivered, listening := s.Events.Stats()
		out["realtime"] = map[string]any{
			"subscribers": subscribers, "spaces": spaces, "delivered": delivered, "listening": listening,
		}
	}
	writeJSON(w, 200, out)
}

func (s *Server) runDreams(w http.ResponseWriter, r *http.Request) {
	if err := s.Dreams.Trigger(r.Context()); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	p := principal(r)
	s.Store.Audit(r.Context(), &p.User.ID, "dream.run", "dream", "manual", map[string]any{})
	writeJSON(w, 202, map[string]string{"status": "queued"})
}

// adminAudit lists what was done, narrowed to the question being asked.
//
// The log records everything worth recording — a space unshared, a key
// rotated, a thought restored, other sessions ended — and it could only be
// read newest-first, all of it. Answering "who took this person out of that
// space" meant scrolling until you found it, which is not an answer anyone
// gets to.
//
// Every filter is optional and they combine. An unknown action or a username
// nobody has narrows the result to nothing rather than being ignored: a filter
// that silently does not apply is worse than an empty page, because the page
// then looks like an answer.
func (s *Server) adminAudit(w http.ResponseWriter, r *http.Request) {
	offset, ok := decodeOffsetCursor(r.URL.Query().Get("cursor"))
	if !ok {
		writeError(w, 400, "감사 로그 커서가 올바르지 않습니다.")
		return
	}
	limit := parsePageLimit(r, 100, 300)

	where := []string{}
	args := []any{}
	add := func(clause string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(clause, len(args)))
	}
	query := r.URL.Query()
	if actor := strings.TrimSpace(query.Get("actor")); actor != "" {
		// 'system' is what the listing shows for an action with no actor, so
		// asking for it has to find those rather than nothing.
		if actor == "system" {
			where = append(where, "a.actor_id IS NULL")
		} else {
			add("u.username = $%d::citext", actor)
		}
	}
	if action := strings.TrimSpace(query.Get("action")); action != "" {
		add("a.action = $%d", action)
	}
	if kind := strings.TrimSpace(query.Get("resourceType")); kind != "" {
		add("a.resource_type = $%d", kind)
	}
	if id := strings.TrimSpace(query.Get("resourceId")); id != "" {
		add("a.resource_id = $%d", id)
	}
	if from := strings.TrimSpace(query.Get("from")); from != "" {
		at, parseErr := time.Parse(time.RFC3339, from)
		if parseErr != nil {
			writeError(w, 400, "시작 시각이 올바르지 않습니다.")
			return
		}
		add("a.created_at >= $%d", at)
	}
	if to := strings.TrimSpace(query.Get("to")); to != "" {
		at, parseErr := time.Parse(time.RFC3339, to)
		if parseErr != nil {
			writeError(w, 400, "끝 시각이 올바르지 않습니다.")
			return
		}
		add("a.created_at <= $%d", at)
	}
	clause := ""
	if len(where) > 0 {
		clause = " WHERE " + strings.Join(where, " AND ")
	}
	args = append(args, limit+1, offset)

	rows, err := s.Store.Pool.Query(r.Context(), `SELECT a.id,a.action,a.resource_type,a.resource_id,a.metadata,a.created_at,COALESCE(u.username,'system') FROM audit_logs a LEFT JOIN users u ON u.id=a.actor_id`+clause+
		fmt.Sprintf(` ORDER BY a.created_at DESC,a.id DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		writeError(w, 500, "감사 로그를 불러오지 못했습니다.")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var action, resourceType, resourceID, actor string
		var metadata json.RawMessage
		var at time.Time
		if err := rows.Scan(&id, &action, &resourceType, &resourceID, &metadata, &at, &actor); err != nil {
			writeError(w, 500, "감사 로그를 읽지 못했습니다.")
			return
		}
		out = append(out, map[string]any{"id": id, "action": action, "resourceType": resourceType, "resourceId": resourceID, "metadata": metadata, "createdAt": at, "actor": actor})
	}
	if err := rows.Err(); err != nil {
		writeError(w, 500, "감사 로그를 불러오지 못했습니다.")
		return
	}
	next := ""
	if len(out) > limit {
		out = out[:limit]
		next = encodeOffsetCursor(offset + limit)
	}
	// The actions actually in use, so the screen can offer them instead of
	// asking someone to type "space.unshare" exactly. Taken from a bounded
	// window of recent rows rather than the whole table, which would be a full
	// scan on a log that only ever grows.
	actions := []string{}
	actionRows, err := s.Store.Pool.Query(r.Context(),
		`SELECT DISTINCT action FROM (SELECT action FROM audit_logs ORDER BY created_at DESC, id DESC LIMIT 5000) recent ORDER BY action`)
	if err == nil {
		for actionRows.Next() {
			var action string
			if actionRows.Scan(&action) == nil {
				actions = append(actions, action)
			}
		}
		actionRows.Close()
	}
	writeJSON(w, 200, map[string]any{"audit": out, "nextCursor": next, "actions": actions})
}

// embeddingQuality reports what the configured embedding backend actually
// measures, so an administrator can tell a semantic model from a lexical
// fallback without reading the source.
//
// The measurement runs against the gateway they configured, which means it costs
// one embedding request. The store caches the result per backend; ?refresh=true
// forces a fresh run after a settings change.
func (s *Server) embeddingQuality(w http.ResponseWriter, r *http.Request) {
	refresh := r.URL.Query().Get("refresh") == "true"
	report, err := s.Store.MeasureEmbeddingQuality(r.Context(), refresh)
	if err != nil {
		writeError(w, 502, "임베딩 백엔드를 측정하지 못했습니다.")
		return
	}
	classes := make([]map[string]any, 0, len(report.Classes))
	for _, class := range report.Classes {
		classes = append(classes, map[string]any{
			"class": string(class.Class),
			"mean":  math.Round(class.Mean*1000) / 1000,
			"min":   math.Round(class.Min*1000) / 1000,
			"max":   math.Round(class.Max*1000) / 1000,
			"count": class.Count,
		})
	}
	// fellBack tells the operator the difference that matters most: a model is
	// configured, but these numbers came from the offline algorithm instead.
	fellBack := report.Model != "" && report.Algorithm == intelligence.LocalAlgorithm
	writeJSON(w, 200, map[string]any{
		"algorithm":        report.Algorithm,
		"model":            report.Model,
		"classes":          classes,
		"discrimination":   math.Round(report.Discrimination*1000) / 1000,
		"pairwiseAccuracy": math.Round(report.PairwiseAccuracy*1000) / 1000,
		"pairs":            report.Pairs,
		"topicSeparation":  math.Round(report.TopicSeparation*1000) / 1000,
		// The thresholds this verdict was reached against, so the screen shows
		// what "semantic" meant on this deployment rather than the shipped
		// defaults an administrator may have changed.
		"accuracyBar":     report.AccuracyBar,
		"purityBar":       report.PurityBar,
		"neighbourPurity": math.Round(report.NeighbourPurity*1000) / 1000,
		"sentences":       report.Sentences,
		"semantic":        report.Semantic,
		"fellBack":        fellBack,
	})
}

// testEmbeddingGateway probes the configured embedding backend and says what
// actually happened.
//
// Until now the only way to find out whether a gateway worked was to save it and
// read the quality report, which conflates three different failures: the address
// is wrong, the model name is wrong, or the model works but is not semantic.
// This separates the first two, which are the ones an administrator can fix from
// this screen.
// testPtium checks that the configured Ptium answers and accepts the key.
//
// It lists templates, which is an authenticated read: a reply proves the
// address answers, that it is a Ptium, and that the credential works — all
// three at once. Without this the first sign of a wrong address or a rejected
// key is someone trying to make a deck and being handed a status code.
//
// The templates come back with the result so the settings screen can offer
// them by name rather than asking an administrator to paste a UUID.
func (s *Server) testPtium(w http.ResponseWriter, r *http.Request) {
	var body struct {
		BaseURL        string `json:"base_url"`
		APIKey         string `json:"api_key"`
		TimeoutSeconds int    `json:"timeout_seconds"`
	}
	if decodeJSON(w, r, &body) != nil {
		writeError(w, 400, "Ptium 정보가 올바르지 않습니다.")
		return
	}

	// A saved key arrives masked, because the settings screen never sends the
	// stored secret back. Fall back to the stored one so testing an unchanged
	// connection does not require retyping it.
	var stored presentation.Config
	hasStored := s.Store.GetSetting(r.Context(), "ptium", &stored) == nil
	storedKey := ""
	if hasStored {
		storedKey = s.Store.DecryptSetting(stored.APIKey)
	}
	key := body.APIKey
	usingStoredKey := key == "" || key == secretMask
	if hasStored && usingStoredKey {
		key = storedKey
	}
	// A pass on a key nobody has saved yet is the most misleading result this
	// screen can give: it works, the page is reloaded, the saved key is used
	// instead, and the connection stops working with nothing having changed on
	// screen. Said plainly rather than left for the reload to reveal.
	unsaved := !usingStoredKey && key != "" && key != storedKey
	base := strings.TrimSpace(body.BaseURL)
	if base == "" && hasStored {
		base = stored.BaseURL
	}
	timeout := body.TimeoutSeconds
	if timeout <= 0 {
		timeout = stored.TimeoutSeconds
	}

	client, err := presentation.NewClient(base, key, time.Duration(timeout)*time.Second)
	if err != nil {
		if errors.Is(err, presentation.ErrNotConfigured) {
			writeError(w, 400, "Ptium 주소를 입력해 주세요.")
			return
		}
		writeError(w, 400, "Ptium 주소가 올바르지 않습니다: "+err.Error())
		return
	}

	/*
	 * Listing templates is how the connection is tested, and it used to be the
	 * whole of the verdict: anything it could not do was reported as the
	 * connection failing.
	 *
	 * Templates are the optional part. A deck can be made without one, and a
	 * Ptium that keeps them somewhere else, or answers that endpoint in a shape
	 * this version does not know, is still a Ptium umm can talk to. Telling
	 * someone their connection is broken when it works sends them to check the
	 * address and the key, which are fine.
	 *
	 * So the reply is read for what it actually says. Nothing came back at all,
	 * or Ptium refused the key: the connection does not work, and that is worth
	 * stopping for. Ptium answered anything else — a 404, a 500, a body this
	 * version cannot parse — the connection works and the templates are simply
	 * unavailable.
	 */
	templates, err := client.Templates(r.Context())
	switch {
	case err == nil:
		writeJSON(w, 200, map[string]any{
			"ok":        true,
			"message":   withSaveWarning(fmt.Sprintf("Ptium에 연결했습니다. 템플릿 %d개를 찾았습니다.", len(templates)), unsaved),
			"unsaved":   unsaved,
			"templates": templates,
		})
	case isPtiumAuthFailure(err):
		writeError(w, 400, "Ptium이 API 키를 거부했습니다: "+err.Error())
	case ptiumAnswered(err):
		writeJSON(w, 200, map[string]any{
			"ok":        true,
			"message":   withSaveWarning("Ptium에 연결했습니다. 템플릿 목록은 가져오지 못했습니다 — 발표 자료를 만드는 데는 지장이 없고, 템플릿은 직접 입력할 수 있습니다.", unsaved),
			"unsaved":   unsaved,
			"warning":   err.Error(),
			"templates": []presentation.Template{},
		})
	default:
		writeError(w, 400, "Ptium 연결 실패: "+err.Error())
	}
}

// withSaveWarning appends the note that the key under test is not the saved one.
func withSaveWarning(message string, unsaved bool) string {
	if !unsaved {
		return message
	}
	return message + " 다만 이 키는 아직 저장되지 않았습니다 — 저장해야 실제 연동에 쓰입니다."
}

// isPtiumAuthFailure is Ptium saying the credential is wrong.
//
// Separated from every other reply because it is the one the person can act on
// directly, and because it must not be softened into "connected" — a key Ptium
// refuses is not a working integration.
func isPtiumAuthFailure(err error) bool {
	var status *presentation.StatusError
	if !errors.As(err, &status) {
		return false
	}
	return status.Status == http.StatusUnauthorized || status.Status == http.StatusForbidden
}

// ptiumAnswered reports whether Ptium replied at all, whatever it said.
//
// A reply means the address resolved, the connection was made and any
// credential was accepted far enough to get a response. What came back may
// still be useless for listing templates.
func ptiumAnswered(err error) bool {
	var status *presentation.StatusError
	if errors.As(err, &status) {
		return true
	}
	var shape *presentation.ShapeError
	return errors.As(err, &shape)
}

func (s *Server) testEmbeddingGateway(w http.ResponseWriter, r *http.Request) {
	var body struct {
		BaseURL          string `json:"base_url"`
		APIKey           string `json:"api_key"`
		Model            string `json:"embedding_model"`
		EmbeddingBaseURL string `json:"embedding_base_url"`
		EmbeddingAPIKey  string `json:"embedding_api_key"`
	}
	if decodeJSON(w, r, &body) != nil {
		writeError(w, 400, "게이트웨이 정보가 올바르지 않습니다.")
		return
	}
	// A saved key arrives masked, because the settings screen never sends the
	// stored secret back. Fall back to the stored one so testing an unchanged
	// gateway does not require retyping it.
	var stored struct {
		APIKey          string `json:"api_key"`
		EmbeddingAPIKey string `json:"embedding_api_key"`
	}
	hasStored := s.Store.GetSetting(r.Context(), "ai_gateway", &stored) == nil
	chatKey := body.APIKey
	if hasStored && (chatKey == "" || chatKey == secretMask) {
		chatKey = s.Store.DecryptSetting(stored.APIKey)
	}
	embedKey := body.EmbeddingAPIKey
	if hasStored && embedKey == secretMask {
		// Only the mask means "the one already saved". An empty embedding key is
		// a real answer — most embedding servers want no authentication — and
		// must not be filled in with the chat gateway's key.
		embedKey = s.Store.DecryptSetting(stored.EmbeddingAPIKey)
	}

	// Resolved exactly as the runtime resolves it, so a green test means the
	// thing that actually embeds will work.
	testURL, testKey := store.ResolveEmbeddingEndpoint(body.BaseURL, chatKey, body.EmbeddingBaseURL, embedKey)
	if strings.TrimSpace(testURL) == "" || strings.TrimSpace(body.Model) == "" {
		writeError(w, 400, "주소와 임베딩 모델 이름이 모두 필요합니다.")
		return
	}

	ctx, cancel := contextWithTimeout(r, 30*time.Second)
	defer cancel()
	provider := intelligence.Provider{Remote: &intelligence.RemoteConfig{
		BaseURL: testURL, APIKey: testKey, Model: body.Model, Timeout: 25 * time.Second,
	}}
	vectors, err := provider.EmbedStrict(ctx, []string{"연결 확인", "connection check"})
	if err != nil {
		writeJSON(w, 200, map[string]any{
			"ok":     false,
			"detail": err.Error(),
		})
		return
	}
	dimensions := 0
	if len(vectors) > 0 {
		dimensions = len(vectors[0])
	}
	writeJSON(w, 200, map[string]any{
		"ok":         true,
		"model":      body.Model,
		"dimensions": dimensions,
	})
}

// discoverEmbeddingGateways looks for an embedding gateway at the addresses umm
// itself documents, so setting one up does not require knowing what the model is
// called.
//
// The addresses are compiled in and never read from the request: probing a
// supplied address would make this screen a way of reaching whatever the server
// can reach, and finding umm's own sidecar needs nothing of the sort.
func (s *Server) discoverEmbeddingGateways(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()
	found := intelligence.DiscoverGateways(ctx, &http.Client{Timeout: 4 * time.Second})
	writeJSON(w, 200, map[string]any{"gateways": found})
}
