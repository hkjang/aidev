package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/store"

	"github.com/jackc/pgx/v5"
)

type preferences struct {
	DreamEnabled       bool       `json:"dream_enabled"`
	DreamFrequency     string     `json:"dream_frequency"`
	DreamStyle         string     `json:"dream_style"`
	DreamNotifications bool       `json:"dream_notifications"`
	IncludeOldNotes    bool       `json:"include_old_notes"`
	DreamPauseUntil    *time.Time `json:"dream_pause_until"`
	Theme              string     `json:"theme"`
	Locale             string     `json:"locale"`
	EdgeStyle          string     `json:"edge_style"`
	ReviewDigest       bool       `json:"review_digest"`
}

// preferencesPatch preserves field presence so a PUT can update one setting
// without copying a stale snapshot over unrelated settings. In particular,
// dream_pause_until uses RawMessage to distinguish an omitted field from an
// explicit null, which clears a pause.
type preferencesPatch struct {
	DreamEnabled       *bool           `json:"dream_enabled"`
	DreamFrequency     *string         `json:"dream_frequency"`
	DreamStyle         *string         `json:"dream_style"`
	DreamNotifications *bool           `json:"dream_notifications"`
	IncludeOldNotes    *bool           `json:"include_old_notes"`
	DreamPauseUntil    json.RawMessage `json:"dream_pause_until"`
	Theme              *string         `json:"theme"`
	Locale             *string         `json:"locale"`
	EdgeStyle          *string         `json:"edge_style"`
	ReviewDigest       *bool           `json:"review_digest"`
}

func (patch preferencesPatch) apply(v *preferences) error {
	if patch.DreamEnabled != nil {
		v.DreamEnabled = *patch.DreamEnabled
	}
	if patch.DreamFrequency != nil {
		v.DreamFrequency = *patch.DreamFrequency
	}
	if patch.DreamStyle != nil {
		v.DreamStyle = *patch.DreamStyle
	}
	if patch.DreamNotifications != nil {
		v.DreamNotifications = *patch.DreamNotifications
	}
	if patch.IncludeOldNotes != nil {
		v.IncludeOldNotes = *patch.IncludeOldNotes
	}
	if len(patch.DreamPauseUntil) > 0 {
		var pauseUntil *time.Time
		if err := json.Unmarshal(patch.DreamPauseUntil, &pauseUntil); err != nil {
			return err
		}
		v.DreamPauseUntil = pauseUntil
	}
	if patch.Theme != nil {
		v.Theme = *patch.Theme
	}
	if patch.Locale != nil {
		v.Locale = *patch.Locale
	}
	if patch.EdgeStyle != nil {
		v.EdgeStyle = *patch.EdgeStyle
	}
	if patch.ReviewDigest != nil {
		v.ReviewDigest = *patch.ReviewDigest
	}
	return nil
}

func (s *Server) getPreferences(w http.ResponseWriter, r *http.Request) {
	p := principal(r)
	var v preferences
	err := s.Store.Pool.QueryRow(r.Context(), `SELECT dream_enabled,dream_frequency,dream_style,dream_notifications,include_old_notes,dream_pause_until,theme,locale,edge_style,review_digest FROM user_preferences WHERE user_id=$1`, p.User.ID).Scan(&v.DreamEnabled, &v.DreamFrequency, &v.DreamStyle, &v.DreamNotifications, &v.IncludeOldNotes, &v.DreamPauseUntil, &v.Theme, &v.Locale, &v.EdgeStyle, &v.ReviewDigest)
	if err != nil {
		writeError(w, 500, "개인 설정을 불러오지 못했습니다.")
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) putPreferences(w http.ResponseWriter, r *http.Request) {
	p := principal(r)
	var patch preferencesPatch
	if decodeJSON(w, r, &patch) != nil {
		writeError(w, 400, "개인 설정 형식이 올바르지 않습니다.")
		return
	}
	var dreamCfg struct {
		AllowUserDisable bool `json:"allow_user_disable"`
	}
	_ = s.Store.GetSetting(r.Context(), "dream", &dreamCfg)

	// Lock, read, merge, and write in one transaction. Two independent partial
	// updates are therefore serialized and each one merges over the latest row.
	tx, err := s.Store.Pool.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "개인 설정을 저장하지 못했습니다.")
		return
	}
	defer tx.Rollback(r.Context())
	var v preferences
	if err = tx.QueryRow(r.Context(), `SELECT dream_enabled,dream_frequency,dream_style,dream_notifications,include_old_notes,dream_pause_until,theme,locale,edge_style,review_digest FROM user_preferences WHERE user_id=$1 FOR UPDATE`, p.User.ID).
		Scan(&v.DreamEnabled, &v.DreamFrequency, &v.DreamStyle, &v.DreamNotifications, &v.IncludeOldNotes, &v.DreamPauseUntil, &v.Theme, &v.Locale, &v.EdgeStyle, &v.ReviewDigest); err != nil {
		writeError(w, 500, "개인 설정을 불러오지 못했습니다.")
		return
	}
	if patch.apply(&v) != nil {
		writeError(w, 400, "개인 설정 형식이 올바르지 않습니다.")
		return
	}
	if !slices.Contains([]string{"ko", "en"}, v.Locale) {
		writeError(w, 400, "지원하지 않는 언어입니다.")
		return
	}
	if !slices.Contains([]string{"daily", "three_week", "weekly"}, v.DreamFrequency) {
		writeError(w, 400, "Dream 빈도가 올바르지 않습니다.")
		return
	}
	if !slices.Contains([]string{"auto", "connection", "question", "expansion", "free"}, v.DreamStyle) {
		writeError(w, 400, "Dream 스타일이 올바르지 않습니다.")
		return
	}
	if !slices.Contains([]string{"light", "dark", "system"}, v.Theme) {
		writeError(w, 400, "테마가 올바르지 않습니다.")
		return
	}
	if v.EdgeStyle == "" {
		v.EdgeStyle = "bezier"
	}
	if !slices.Contains([]string{"bezier", "smoothstep", "straight"}, v.EdgeStyle) {
		writeError(w, 400, "연결선 형태가 올바르지 않습니다.")
		return
	}
	if !dreamCfg.AllowUserDisable {
		v.DreamEnabled = true
	}
	_, err = tx.Exec(r.Context(), `UPDATE user_preferences SET dream_enabled=$2,dream_frequency=$3,dream_style=$4,dream_notifications=$5,include_old_notes=$6,dream_pause_until=$7,theme=$8,locale=$9,edge_style=$10,review_digest=$11,updated_at=now() WHERE user_id=$1`, p.User.ID, v.DreamEnabled, v.DreamFrequency, v.DreamStyle, v.DreamNotifications, v.IncludeOldNotes, v.DreamPauseUntil, v.Theme, v.Locale, v.EdgeStyle, v.ReviewDigest)
	if err != nil {
		writeError(w, 500, "개인 설정을 저장하지 못했습니다.")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "개인 설정을 저장하지 못했습니다.")
		return
	}
	writeJSON(w, 200, v)
}

type securityConfig struct {
	APIKeyScopes         []string `json:"api_key_scopes"`
	DefaultKeyDays       int      `json:"default_key_days"`
	RotationOverlapHours int      `json:"rotation_overlap_hours"`
}

func (s *Server) getSecurityConfig(r *http.Request) securityConfig {
	var cfg securityConfig
	_ = s.Store.GetSetting(r.Context(), "security", &cfg)
	if cfg.DefaultKeyDays <= 0 {
		cfg.DefaultKeyDays = 90
	}
	return cfg
}
func validateScopes(requested, allowed []string) bool {
	if len(requested) == 0 {
		return false
	}
	for _, v := range requested {
		if !slices.Contains(allowed, v) {
			return false
		}
	}
	return true
}

func (s *Server) listAPIKeys(w http.ResponseWriter, r *http.Request) {
	p := principal(r)
	keys, err := s.Auth.ListKeys(r.Context(), p.User.ID)
	if err != nil {
		writeError(w, 500, "키를 불러오지 못했습니다.")
		return
	}
	cfg := s.getSecurityConfig(r)
	writeJSON(w, 200, map[string]any{"keys": keys, "availableScopes": cfg.APIKeyScopes, "rotationOverlapHours": cfg.RotationOverlapHours})
}

func (s *Server) createAPIKey(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string   `json:"name"`
		Scopes      []string `json:"scopes"`
		ExpiresDays int      `json:"expiresDays"`
	}
	if decodeJSON(w, r, &body) != nil {
		writeError(w, 400, "키 설정 형식이 올바르지 않습니다.")
		return
	}
	cfg := s.getSecurityConfig(r)
	if !validateScopes(body.Scopes, cfg.APIKeyScopes) {
		writeError(w, 400, "허용되지 않은 키 권한입니다.")
		return
	}
	if body.ExpiresDays == 0 {
		body.ExpiresDays = cfg.DefaultKeyDays
	}
	if body.ExpiresDays < 1 || body.ExpiresDays > 3650 {
		writeError(w, 400, "키 만료 기간은 1~3650일이어야 합니다.")
		return
	}
	p := principal(r)
	key, secret, err := s.Auth.CreateKey(r.Context(), p.User.ID, body.Name, body.Scopes, body.ExpiresDays)
	if err != nil {
		writeError(w, 500, "키를 만들지 못했습니다.")
		return
	}
	s.Store.Audit(r.Context(), &p.User.ID, "api_key.create", "api_key", key.ID.String(), map[string]any{"scopes": body.Scopes})
	writeJSON(w, 201, map[string]any{"key": key, "secret": secret, "warning": "이 키는 다시 표시되지 않습니다."})
}

func (s *Server) updateAPIKey(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "keyID")
	if !ok {
		return
	}
	var body struct {
		Scopes []string `json:"scopes"`
	}
	if decodeJSON(w, r, &body) != nil {
		writeError(w, 400, "키 권한 형식이 올바르지 않습니다.")
		return
	}
	cfg := s.getSecurityConfig(r)
	if !validateScopes(body.Scopes, cfg.APIKeyScopes) {
		writeError(w, 400, "허용되지 않은 키 권한입니다.")
		return
	}
	p := principal(r)
	if err := s.Auth.UpdateKeyScopes(r.Context(), p.User.ID, id, body.Scopes); err != nil {
		writeError(w, 404, "활성 키를 찾을 수 없습니다.")
		return
	}
	s.Store.Audit(r.Context(), &p.User.ID, "api_key.permissions", "api_key", id.String(), map[string]any{"scopes": body.Scopes})
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) rotateAPIKey(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "keyID")
	if !ok {
		return
	}
	cfg := s.getSecurityConfig(r)
	p := principal(r)
	key, secret, err := s.Auth.RotateKey(r.Context(), p.User.ID, id, cfg.RotationOverlapHours)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "회전할 활성 키를 찾을 수 없습니다.")
		return
	}
	if err != nil {
		// Only the lookup for an active key answers "no rows". Everything after
		// it — the new key, the overlap, the commit — fails differently, and
		// reporting that as "there is no key to rotate" sends someone looking
		// for a key that is sitting right there in their list.
		writeError(w, 500, "키를 회전하지 못했습니다.")
		return
	}
	s.Store.Audit(r.Context(), &p.User.ID, "api_key.rotate", "api_key", id.String(), map[string]any{"replacement": key.ID, "overlapHours": cfg.RotationOverlapHours})
	writeJSON(w, 201, map[string]any{"key": key, "secret": secret, "overlapHours": cfg.RotationOverlapHours, "warning": "새 키는 다시 표시되지 않습니다."})
}

func (s *Server) revokeAPIKey(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "keyID")
	if !ok {
		return
	}
	p := principal(r)
	// Revoking and forgetting are the same verb from the outside, because from
	// the outside they are one intention: be rid of this key. The first press
	// stops it working; a key that is already revoked no longer needs stopping,
	// so the second removes the row. A key that still works can never be
	// removed by one action, which is the part worth protecting.
	if err := s.Auth.RevokeKey(r.Context(), p.User.ID, id); err != nil {
		if err := s.Auth.ForgetRevokedKey(r.Context(), p.User.ID, id); err != nil {
			writeError(w, 404, "키를 찾을 수 없습니다.")
			return
		}
		s.Store.Audit(r.Context(), &p.User.ID, "api_key.forget", "api_key", id.String(), map[string]any{})
		w.WriteHeader(http.StatusNoContent)
		return
	}
	s.Store.Audit(r.Context(), &p.User.ID, "api_key.revoke", "api_key", id.String(), map[string]any{})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) dreamHistory(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "dreams:read") {
		return
	}
	offset, ok := decodeOffsetCursor(r.URL.Query().Get("cursor"))
	if !ok {
		writeError(w, 400, "Dream 커서가 올바르지 않습니다.")
		return
	}
	limit := parsePageLimit(r, 30, 100)
	v, hasMore, err := s.Dreams.HistoryPage(r.Context(), principal(r).User.ID, limit, offset)
	if err != nil {
		writeError(w, 500, "Dream 기록을 불러오지 못했습니다.")
		return
	}
	next := ""
	if hasMore {
		next = encodeOffsetCursor(offset + limit)
	}
	// The tabs label states, not pages, so their counts cannot come from the
	// slice above. Counting is cheap next to the join the listing already did;
	// a failure here costs the labels their numbers rather than the page its
	// dreams.
	counts, err := s.Dreams.StatusCounts(r.Context(), principal(r).User.ID)
	if err != nil {
		slog.Warn("could not count dreams by state", "error", err)
		counts = nil
	}
	writeJSON(w, 200, map[string]any{"dreams": v, "nextCursor": next, "counts": counts})
}
func (s *Server) dreamFeedback(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "dreams:read") {
		return
	}
	id, ok := parseID(w, r, "dreamID")
	if !ok {
		return
	}
	var body struct {
		Action string `json:"action"`
		Reason string `json:"reason"`
	}
	if decodeJSON(w, r, &body) != nil {
		writeError(w, 400, "피드백 형식이 올바르지 않습니다.")
		return
	}
	if err := s.Dreams.FeedbackWithReason(r.Context(), principal(r).User.ID, id, body.Action, body.Reason); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) acceptDream(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "dreams:read") || !requireScope(w, r, "notes:write") {
		return
	}
	id, ok := parseID(w, r, "dreamID")
	if !ok {
		return
	}
	var body struct {
		Content string `json:"content"`
	}
	if decodeJSON(w, r, &body) != nil {
		writeError(w, 400, "Dream 채택 형식이 올바르지 않습니다.")
		return
	}
	p := principal(r)
	note, err := s.Dreams.Accept(r.Context(), p.User.ID, id, body.Content)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	s.Store.Audit(r.Context(), &p.User.ID, "dream.accept", "dream", id.String(), map[string]any{"noteId": note.ID, "spaceId": note.SpaceID})
	writeJSON(w, http.StatusCreated, note)
}

func (s *Server) regenerateDream(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "dreams:read") {
		return
	}
	id, ok := parseID(w, r, "dreamID")
	if !ok {
		return
	}
	p := principal(r)
	view, err := s.Dreams.Regenerate(r.Context(), p.User.ID, id)
	if err != nil {
		if writeAIQuotaProblem(w, r, err) {
			return
		}
		writeError(w, 400, err.Error())
		return
	}
	s.Store.Audit(r.Context(), &p.User.ID, "dream.regenerate", "dream", id.String(), map[string]any{"generation": view.Generation})
	writeJSON(w, 200, view)
}

func (s *Server) developDream(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "dreams:read") {
		return
	}
	id, ok := parseID(w, r, "dreamID")
	if !ok {
		return
	}
	var body struct {
		Mode string `json:"mode"`
	}
	if decodeJSON(w, r, &body) != nil {
		writeError(w, 400, "Dream 발전 형식이 올바르지 않습니다.")
		return
	}
	p := principal(r)
	result, err := s.Dreams.Develop(r.Context(), p.User.ID, id, body.Mode)
	if err != nil {
		if writeAIQuotaProblem(w, r, err) {
			return
		}
		writeError(w, 400, err.Error())
		return
	}
	s.Store.Audit(r.Context(), &p.User.ID, "dream.develop", "dream", id.String(), map[string]any{"mode": body.Mode})
	writeJSON(w, 200, result)
}

func (s *Server) saveDevelopedDream(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "dreams:read") || !requireScope(w, r, "notes:write") {
		return
	}
	id, ok := parseID(w, r, "dreamID")
	if !ok {
		return
	}
	var body struct {
		Content string `json:"content"`
	}
	if decodeJSON(w, r, &body) != nil {
		writeError(w, 400, "Dream 발전 결과 형식이 올바르지 않습니다.")
		return
	}
	p := principal(r)
	result, err := s.Dreams.MaterializeDevelopment(r.Context(), p.User.ID, id, body.Content)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	status := http.StatusOK
	if result.Created {
		status = http.StatusCreated
		s.Store.Audit(r.Context(), &p.User.ID, "dream.development.save", "dream", id.String(), map[string]any{"noteId": result.Note.ID, "spaceId": result.Note.SpaceID})
	}
	writeJSON(w, status, result)
}

var _ = uuid.Nil

// captureThought writes a thought down without asking where it belongs.
//
// The whole point is that it works from anywhere, so it takes no space id: the
// thought lands in the person's inbox and the question of where it goes is
// answered later, or never.
func (s *Server) captureThought(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Content string `json:"content"`
	}
	if decodeJSON(w, r, &body) != nil || strings.TrimSpace(body.Content) == "" {
		writeError(w, 400, "생각 내용을 입력해 주세요.")
		return
	}
	p := principal(r)
	note, err := s.Store.CaptureThought(r.Context(), p.User.ID, body.Content)
	if err != nil {
		slog.Warn("capture failed", "user_id", p.User.ID, "error", err)
		writeError(w, 500, "생각을 저장하지 못했습니다.")
		return
	}
	writeJSON(w, 201, note)
}

// moveNote files a thought into another space.
func (s *Server) moveNote(w http.ResponseWriter, r *http.Request) {
	noteID, ok := parseID(w, r, "noteID")
	if !ok {
		return
	}
	var body struct {
		SpaceID uuid.UUID `json:"spaceId"`
	}
	if decodeJSON(w, r, &body) != nil || body.SpaceID == uuid.Nil {
		writeError(w, 400, "옮길 공간을 지정해 주세요.")
		return
	}
	p := principal(r)
	note, removedEdges, err := s.Store.MoveNote(r.Context(), p.User.ID, noteID, body.SpaceID)
	if err != nil {
		if notFound(err) {
			writeError(w, 404, "생각을 찾을 수 없습니다.")
			return
		}
		writeError(w, 400, "생각을 옮길 수 없습니다.")
		return
	}
	// The caller needs the count to explain what happened, because connections
	// are scoped to a space and a note that leaves cannot keep them.
	writeJSON(w, 200, map[string]any{"note": note, "removedEdges": removedEdges})
}

// spaceSuggestions ranks where a captured thought might belong.
func (s *Server) spaceSuggestions(w http.ResponseWriter, r *http.Request) {
	noteID, ok := parseID(w, r, "noteID")
	if !ok {
		return
	}
	p := principal(r)
	suggestions, err := s.Store.SuggestSpaces(r.Context(), p.User.ID, noteID, 3)
	if err != nil {
		writeError(w, 500, "추천 공간을 찾지 못했습니다.")
		return
	}
	writeJSON(w, 200, map[string]any{"suggestions": suggestions})
}

// myAIUsage answers, for one person, what their own thoughts were sent out for.
//
// A session rather than an API key: this is the record of what happened to
// somebody's writing, and a key issued to a script has no business reading it.
// It is mounted under the account group, which enforces that.
//
// Two things travel with the list so it cannot be misread. The retention window,
// because a list that stops ninety days back otherwise reads as ninety quiet
// days. And whether note bodies currently leave this machine for indexing,
// because a list of chat-model calls looks like the whole story otherwise —
// embeddings go in batches that can span several people's notes in a shared
// space, so naming one person for one batch would be a guess where the policy
// is a fact.
func (s *Server) myAIUsage(w http.ResponseWriter, r *http.Request) {
	p := principal(r)

	days := 30
	if raw := strings.TrimSpace(r.URL.Query().Get("days")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 1 && n <= 365 {
			days = n
		}
	}
	since := time.Now().AddDate(0, 0, -days)

	usage, err := s.Store.PersonalAIUsage(r.Context(), p.User.ID, since)
	if err != nil {
		slog.Warn("personal ai usage failed", "user_id", p.User.ID, "error", err)
		writeError(w, 500, "AI 사용 내역을 불러오지 못했습니다.")
		return
	}

	var gateway struct {
		LogRetentionDays int `json:"log_retention_days"`
	}
	_ = s.Store.GetSetting(r.Context(), "ai_gateway", &gateway)
	if gateway.LogRetentionDays < 1 {
		gateway.LogRetentionDays = 90
	}
	usage.RetentionDays = gateway.LogRetentionDays

	// Read from the same resolver the runtime uses rather than from the setting,
	// so the answer is what actually happens rather than what is configured.
	provider := s.Store.EmbeddingProvider(r.Context())
	writeJSON(w, 200, map[string]any{
		"usage": usage,
		"since": since,
		"days":  days,
		// Whether the words in a note leave this machine to be indexed. False
		// is the default and means the offline algorithm: nothing is sent.
		"embeddingsLeaveThisMachine": provider.Model() != "",
		"embeddingModel":             provider.Model(),
		"purposes":                   store.Purposes(),
	})
}
