package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

//go:embed information_protection.sql
var informationProtectionSchema string

type ProtectionFinding struct {
	Kind  string `json:"kind"`
	Count int    `json:"count"`
}
type ProtectionResult struct {
	Title    string              `json:"title"`
	Markdown string              `json:"markdown"`
	Findings []ProtectionFinding `json:"findings"`
	Mode     string              `json:"mode"`
	Changed  bool                `json:"changed"`
}
type ProtectionError struct {
	Code     string
	Findings []ProtectionFinding
}

func (e ProtectionError) Error() string {
	if e.Code == "invalid_text" {
		return "문서의 UTF-8 문자 형식이나 정제 후 제목·본문 크기 제한을 확인하세요"
	}
	if e.Code == "unscannable" {
		return "이 파일은 지원하는 UTF-8 텍스트 4MB 검사 범위에 속하지 않아 관리자 정책에 따라 업로드할 수 없습니다"
	}
	if e.Code == "collaboration_policy" {
		return "정보보호 차단·마스킹 정책에서는 공동 편집의 숨은 변경 이력을 검사할 수 없어 원문 편집으로 저장해야 합니다"
	}
	if e.Code == "mask_required" {
		return "민감정보 마스킹이 필요합니다. 공동 편집 대신 원문 편집에서 정제된 내용을 명시적으로 저장하세요"
	}
	return "관리자 정보보호 정책에 따라 민감정보가 포함된 저장을 차단했습니다"
}
func ProtectionMaskRequired(result ProtectionResult) error {
	return ProtectionError{Code: "mask_required", Findings: result.Findings}
}
func WriteProtectionError(w http.ResponseWriter, e error) bool {
	var problem ProtectionError
	var pg *pgconn.PgError
	if !errors.As(e, &problem) {
		if !errors.As(e, &pg) || !strings.HasPrefix(pg.Message, "MADI_PROTECTION_") {
			return false
		}
		problem.Code = "blocked"
		if pg.Message == "MADI_PROTECTION_MASK_REQUIRED" {
			problem.Code = "mask_required"
		}
	}
	jsonResponse(w, 422, map[string]any{"error": problem.Error(), "code": problem.Code, "findings": problem.Findings})
	return true
}
func defaultProtectionSettings() map[string]any {
	return map[string]any{
		"enabled": false, "mode": "warn", "detectors": []string{"rrn", "email", "phone", "payment", "account"}, "custom_terms": []string{},
		"unscannable": "warn", "watermark_enabled": false, "watermark_min_classification": "confidential",
		"public_shares_enabled": false, "public_share_max_classification": "internal", "public_share_max_days": 30, "public_share_require_password": false, "public_share_allow_download": false,
	}
}
func validateProtectionSettings(cfg map[string]any) error {
	for _, key := range []string{"enabled", "watermark_enabled", "public_shares_enabled", "public_share_require_password", "public_share_allow_download"} {
		if _, ok := cfg[key].(bool); !ok {
			return fmt.Errorf("%s 설정은 켜기/끄기 값이어야 합니다", key)
		}
	}
	if !oneOf(str(cfg, "mode"), "warn", "block", "mask", "audit") || !oneOf(str(cfg, "unscannable"), "warn", "block") {
		return errors.New("정보보호 검사 방식을 확인하세요")
	}
	for _, key := range []string{"watermark_min_classification", "public_share_max_classification"} {
		if !oneOf(str(cfg, key), "public", "internal", "confidential", "restricted") {
			return errors.New("정보보호 등급을 확인하세요")
		}
	}
	validDays := false
	switch value := cfg["public_share_max_days"].(type) {
	case int:
		validDays = value >= 1 && value <= 365
	case float64:
		validDays = value >= 1 && value <= 365 && math.Trunc(value) == value
	}
	if !validDays {
		return errors.New("공개 공유 최대 기간은 1~365일입니다")
	}
	for _, key := range []string{"detectors", "custom_terms"} {
		encoded, e := json.Marshal(cfg[key])
		if e != nil {
			return errors.New("탐지 항목을 확인하세요")
		}
		var values []string
		if json.Unmarshal(encoded, &values) != nil || values == nil || len(values) > 100 {
			return errors.New("탐지 항목 또는 지정어는 최대 100개의 문자열 목록입니다")
		}
		seen := map[string]bool{}
		for _, value := range values {
			if seen[value] {
				return errors.New("탐지 항목과 지정어는 중복할 수 없습니다")
			}
			seen[value] = true
			if key == "detectors" && !oneOf(value, "rrn", "email", "phone", "payment", "account") {
				return errors.New("지원하는 탐지 항목을 선택하세요")
			}
			if key == "custom_terms" && (len([]rune(value)) < 2 || len([]rune(value)) > 100 || strings.TrimSpace(value) != value || strings.ContainsAny(value, "\x00\r\n") || strings.Contains("MADI_REDACTED", strings.ToUpper(value))) {
				return errors.New("지정어는 앞뒤 공백과 줄바꿈 없이 2~100자여야 합니다")
			}
		}
	}
	return nil
}
func decodeProtectionSettings(data []byte) (map[string]any, error) {
	cfg := defaultProtectionSettings()
	var stored map[string]any
	if e := json.Unmarshal(data, &stored); e != nil {
		return nil, e
	}
	for k, v := range stored {
		cfg[k] = v
	}
	return cfg, validateProtectionSettings(cfg)
}
func (s *Server) migrateInformationProtection(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, informationProtectionSchema)
	return e
}
func (s *Server) protectionSettings(ctx context.Context) (map[string]any, int64, error) {
	var raw []byte
	var revision int64
	e := s.DB.QueryRow(ctx, "SELECT data,revision FROM protection_settings WHERE id=1").Scan(&raw, &revision)
	if e != nil {
		return nil, 0, e
	}
	cfg, e := decodeProtectionSettings(raw)
	return cfg, revision, e
}
func (s *Server) registerInformationProtection() {
	s.admin("GET /api/v1/admin/information-protection", func(w http.ResponseWriter, r *http.Request) {
		cfg, revision, e := s.protectionSettings(r.Context())
		respond(w, map[string]any{"settings": cfg, "revision": revision}, e)
	})
	s.admin("PUT /api/v1/admin/information-protection", s.updateInformationProtection)
	s.admin("GET /api/v1/admin/information-protection/history", func(w http.ResponseWriter, r *http.Request) {
		rows, e := s.rows(r.Context(), "SELECT jsonb_build_object('id',h.id,'revision',h.revision,'data',h.data,'user_name',u.name,'created_at',h.created_at) FROM protection_settings_history h JOIN users u ON u.id=h.user_id ORDER BY h.created_at DESC LIMIT 100")
		respond(w, rows, e)
	})
	s.admin("GET /api/v1/admin/information-protection/events", func(w http.ResponseWriter, r *http.Request) {
		rows, e := s.rows(r.Context(), "SELECT jsonb_build_object('id',id,'document_id',document_id,'workspace_id',workspace_id,'user_id',user_id,'action',action,'mode',mode,'findings',findings,'created_at',created_at) FROM protection_events ORDER BY created_at DESC LIMIT 200")
		respond(w, rows, e)
	})
	s.handle("GET /api/v1/documents/{id}/protection", s.documentProtection)
	s.handle("POST /api/v1/documents/{id}/protection/preview", s.previewDocumentProtection)
}
func (s *Server) updateInformationProtection(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Settings map[string]any `json:"settings"`
		Revision int64          `json:"revision"`
	}
	if decode(r, &in) != nil || in.Settings == nil {
		apiError(w, 400, "정보보호 설정을 확인하세요")
		return
	}
	cfg := defaultProtectionSettings()
	for key, v := range in.Settings {
		if _, ok := cfg[key]; !ok {
			apiError(w, 400, "지원하지 않는 정보보호 설정입니다")
			return
		}
		cfg[key] = v
	}
	if e := validateProtectionSettings(cfg); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	tag, e := tx.Exec(r.Context(), "UPDATE protection_settings SET data=$1,revision=revision+1,updated_by=$2,updated_at=now() WHERE id=1 AND revision=$3", jsonValue(cfg), current(r).ID, in.Revision)
	if e == nil && tag.RowsAffected() != 1 {
		apiError(w, 409, "다른 관리자가 설정을 바꿨습니다. 다시 불러오세요")
		return
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), "INSERT INTO protection_settings_history(id,revision,data,user_id) VALUES($1,$2,$3,$4)", newID(), in.Revision+1, jsonValue(cfg), current(r).ID)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "INFORMATION_PROTECTION_CHANGE", "settings", map[string]any{"revision": in.Revision + 1})
	jsonResponse(w, 200, map[string]any{"settings": cfg, "revision": in.Revision + 1})
}
func protectionScanTx(ctx context.Context, tx pgx.Tx, cfg map[string]any, value string) (string, []ProtectionFinding, error) {
	var raw []byte
	e := tx.QueryRow(ctx, "SELECT madi_protection_analyze($1,$2,$3)", value, jsonValue(cfg["detectors"]), jsonValue(cfg["custom_terms"])).Scan(&raw)
	if e != nil {
		return "", nil, e
	}
	var result struct {
		Text     string              `json:"text"`
		Findings []ProtectionFinding `json:"findings"`
	}
	e = json.Unmarshal(raw, &result)
	return result.Text, result.Findings, e
}

// ProtectDocumentTx must run before ALL canonical document/version/outbox writes.
// A CRDT caller must reject Changed via ProtectionMaskRequired: a Markdown-only
// mask cannot sanitize the original Yjs binary or other clients' pending state.
func (s *Server) ProtectDocumentTx(ctx context.Context, tx pgx.Tx, p *Principal, documentID, workspaceID, title, markdown string) (ProtectionResult, error) {
	result := ProtectionResult{Title: title, Markdown: markdown, Findings: []ProtectionFinding{}, Mode: "off"}
	if !utf8.ValidString(title) || !utf8.ValidString(markdown) || strings.ContainsRune(title, 0) || strings.ContainsRune(markdown, 0) || len(title) > 500 || len(markdown) > 4<<20 {
		return result, ProtectionError{Code: "invalid_text"}
	}
	var raw []byte
	if e := tx.QueryRow(ctx, "SELECT data FROM protection_settings WHERE id=1 FOR SHARE").Scan(&raw); e != nil {
		return result, e
	}
	cfg, e := decodeProtectionSettings(raw)
	if e != nil {
		return result, e
	}
	if !boolean(cfg, "enabled") {
		return result, nil
	}
	result.Mode = str(cfg, "mode")
	if p != nil {
		if _, e = tx.Exec(ctx, "SELECT set_config('madi.protection_actor',$1,true)", p.ID); e != nil {
			return result, e
		}
	}
	cleanTitle, findTitle, e := protectionScanTx(ctx, tx, cfg, title)
	if e != nil {
		return result, e
	}
	cleanMarkdown, findMarkdown, e := protectionScanTx(ctx, tx, cfg, markdown)
	if e != nil {
		return result, e
	}
	result.Findings = append(findTitle, findMarkdown...)
	if len(result.Findings) == 0 {
		return result, nil
	}
	if result.Mode == "block" {
		return result, ProtectionError{Code: "blocked", Findings: result.Findings}
	}
	if result.Mode == "mask" {
		if len(cleanTitle) > 500 || len(cleanMarkdown) > 4<<20 {
			return result, ProtectionError{Code: "invalid_text"}
		}
		result.Title = cleanTitle
		result.Markdown = cleanMarkdown
		result.Changed = true
		userID := ""
		if p != nil {
			userID = p.ID
		}
		_, e = tx.Exec(ctx, "INSERT INTO protection_events(id,document_id,workspace_id,user_id,action,mode,findings) VALUES($1,NULLIF($2,'')::uuid,NULLIF($3,'')::uuid,NULLIF($4,'')::uuid,'document.mask','mask',$5)", newID(), documentID, workspaceID, userID, jsonValue(result.Findings))
	}
	return result, e
}

type ProtectionMetadataResult struct {
	Value    any
	Findings []ProtectionFinding
	Changed  bool
	Mode     string
}

func (s *Server) ProtectDocumentMetadataTx(ctx context.Context, tx pgx.Tx, p *Principal, documentID, workspaceID string, value any) (ProtectionMetadataResult, error) {
	result := ProtectionMetadataResult{Value: value, Findings: []ProtectionFinding{}, Mode: "off"}
	var raw []byte
	if e := tx.QueryRow(ctx, "SELECT data FROM protection_settings WHERE id=1 FOR SHARE").Scan(&raw); e != nil {
		return result, e
	}
	cfg, e := decodeProtectionSettings(raw)
	if e != nil {
		return result, e
	}
	if !boolean(cfg, "enabled") {
		return result, nil
	}
	result.Mode = str(cfg, "mode")
	if p != nil {
		if _, e = tx.Exec(ctx, "SELECT set_config('madi.protection_actor',$1,true)", p.ID); e != nil {
			return result, e
		}
	}
	encoded, e := json.Marshal(value)
	if e != nil || len(encoded) > 4<<20 {
		return result, errors.New("문서 메타데이터 크기와 형식을 확인하세요")
	}
	if e = tx.QueryRow(ctx, "SELECT madi_protection_json($1,$2,$3)", encoded, jsonValue(cfg["detectors"]), jsonValue(cfg["custom_terms"])).Scan(&raw); e != nil {
		return result, e
	}
	var scanned struct {
		Value      any                 `json:"value"`
		Findings   []ProtectionFinding `json:"findings"`
		Unmaskable bool                `json:"unmaskable"`
	}
	if e = json.Unmarshal(raw, &scanned); e != nil {
		return result, e
	}
	result.Findings = scanned.Findings
	if len(result.Findings) == 0 {
		return result, nil
	}
	if result.Mode == "block" {
		return result, ProtectionError{Code: "blocked", Findings: result.Findings}
	}
	if result.Mode == "mask" {
		if scanned.Unmaskable {
			return result, ProtectionError{Code: "mask_required", Findings: result.Findings}
		}
		result.Value = scanned.Value
		result.Changed = true
		actor := ""
		if p != nil {
			actor = p.ID
		}
		_, e = tx.Exec(ctx, "INSERT INTO protection_events(id,document_id,workspace_id,user_id,action,mode,findings) VALUES($1,NULLIF($2,'')::uuid,NULLIF($3,'')::uuid,NULLIF($4,'')::uuid,'metadata.mask','mask',$5)", newID(), documentID, workspaceID, actor, jsonValue(result.Findings))
	}
	return result, e
}
func (s *Server) ProtectCollaborationTx(ctx context.Context, tx pgx.Tx, p *Principal, documentID, workspaceID, title, markdown string) (ProtectionResult, error) {
	var raw []byte
	if e := tx.QueryRow(ctx, "SELECT data FROM protection_settings WHERE id=1 FOR SHARE").Scan(&raw); e != nil {
		return ProtectionResult{}, e
	}
	cfg, e := decodeProtectionSettings(raw)
	if e != nil {
		return ProtectionResult{}, e
	}
	if boolean(cfg, "enabled") && oneOf(str(cfg, "mode"), "block", "mask") {
		return ProtectionResult{}, ProtectionError{Code: "collaboration_policy"}
	}
	return s.ProtectDocumentTx(ctx, tx, p, documentID, workspaceID, title, markdown)
}
func (s *Server) documentProtection(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canDocument(r.Context(), current(r), id, false) {
		apiError(w, 404, "문서에 접근할 수 없습니다")
		return
	}
	cfg, revision, e := s.protectionSettings(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	var classification string
	e = s.DB.QueryRow(r.Context(), "SELECT madi_effective_classification($1)", id).Scan(&classification)
	watermark := boolean(cfg, "watermark_enabled") && classificationRank(classification) >= classificationRank(str(cfg, "watermark_min_classification"))
	respond(w, map[string]any{"classification": classification, "watermark": watermark, "viewer": current(r).Name, "enabled": cfg["enabled"], "mode": cfg["mode"], "revision": revision, "public_shares_enabled": cfg["public_shares_enabled"], "public_policy": map[string]any{"max_days": cfg["public_share_max_days"], "require_password": cfg["public_share_require_password"], "allow_download": cfg["public_share_allow_download"], "max_classification": cfg["public_share_max_classification"]}}, e)
}
func classificationRank(value string) int {
	switch value {
	case "restricted":
		return 3
	case "confidential":
		return 2
	case "internal":
		return 1
	}
	return 0
}

func (s *Server) protectEventMetadataTx(ctx context.Context, tx pgx.Tx, value map[string]any) (map[string]any, error) {
	var raw []byte
	if e := tx.QueryRow(ctx, "SELECT data FROM protection_settings WHERE id=1 FOR SHARE").Scan(&raw); e != nil {
		return nil, e
	}
	cfg, e := decodeProtectionSettings(raw)
	if e != nil {
		return nil, e
	}
	if !boolean(cfg, "enabled") || !oneOf(str(cfg, "mode"), "block", "mask") {
		return value, nil
	}
	if e = tx.QueryRow(ctx, "SELECT madi_protection_json($1,$2,$3)", jsonValue(value), jsonValue(cfg["detectors"]), jsonValue(cfg["custom_terms"])).Scan(&raw); e != nil {
		return nil, e
	}
	var result struct {
		Value      map[string]any `json:"value"`
		Unmaskable bool           `json:"unmaskable"`
	}
	if e = json.Unmarshal(raw, &result); e != nil {
		return nil, e
	}
	if result.Unmaskable {
		return map[string]any{"redacted": true}, nil
	}
	return result.Value, nil
}
func (s *Server) previewDocumentProtection(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canDocument(r.Context(), current(r), id, true) || !hasIntegrationScope(current(r), "document:write") {
		apiError(w, 404, "문서를 편집할 수 없습니다")
		return
	}
	var in struct {
		Title    string `json:"title"`
		Markdown string `json:"markdown"`
	}
	if decode(r, &in) != nil || len(in.Markdown) > 4<<20 || len(in.Title) > 500 {
		apiError(w, 400, "검사할 문서를 확인하세요")
		return
	}
	cfg, _, e := s.protectionSettings(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	title, tf, e := protectionScanTx(r.Context(), tx, cfg, in.Title)
	if e != nil {
		respond(w, nil, e)
		return
	}
	md, mf, e := protectionScanTx(r.Context(), tx, cfg, in.Markdown)
	respond(w, ProtectionResult{Title: title, Markdown: md, Findings: append(tf, mf...), Mode: str(cfg, "mode"), Changed: title != in.Title || md != in.Markdown}, e)
}
