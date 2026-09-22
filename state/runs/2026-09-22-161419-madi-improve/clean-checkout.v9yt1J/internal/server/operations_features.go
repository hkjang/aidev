package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

//go:embed operations.sql
var operationsSchema string

type featureDefinition struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	StopPolicy  string `json:"stop_policy"`
}

func featureCatalogue() []featureDefinition {
	return []featureDefinition{
		{"canvas", "캔버스", "캔버스 열람·편집과 AI 노드 실행", "새 요청 차단, 실행 중 AI 취소, 원본 보존"},
		{"plugins", "플러그인 실행", "사용자가 승인한 플러그인 및 권한 브리지 실행", "브리지 즉시 차단, 작업자 재검사·취소, 설치·원본 보존"},
		{"collaboration", "실시간 공동 편집", "Yjs 기반 동시 편집과 접속 상태", "공동 편집 연결 중단, 문서의 일반 편집·원본 보존"},
		{"database-formula", "데이터베이스 수식", "수식 계산과 수식에 의존하는 집계", "계산 차단, 기존 수식 정의·원본 행 보존"},
		{"ai-graph", "AI 지식 그래프", "AI 관계 추천과 사용자 승인", "새 실행 차단, 진행 중 분석 취소, 기존 수동 관계 보존"},
		{"workspace-agents", "워크스페이스 에이전트", "허용된 지식과 도구를 사용하는 AI 에이전트", "실행·도구 호출 차단, 진행 중 실행 취소, 설정·기록 보존"},
		{"document-queries", "문서 선언형 조회", "문서 안의 제한된 문서·할 일·관계 표 조회", "새 조회 차단, 표시 결과 폐기, Markdown 정의 보존"},
	}
}
func knownFeature(key string) bool {
	for _, definition := range featureCatalogue() {
		if definition.ID == key {
			return true
		}
	}
	return false
}
func validateFeatureFlags(value any) error {
	flags, ok := value.(map[string]any)
	if !ok || len(flags) > len(featureCatalogue()) {
		return errors.New("기능 플래그는 지원하는 기능 ID와 활성 여부의 객체여야 합니다")
	}
	for key, value := range flags {
		if !knownFeature(key) {
			return errors.New("지원하지 않는 기능 플래그입니다: " + key)
		}
		if _, ok := value.(bool); !ok {
			return errors.New("기능 플래그 값은 true 또는 false여야 합니다")
		}
	}
	return nil
}
func (s *Server) migrateOperations(ctx context.Context) error {
	_, err := s.DB.Exec(ctx, operationsSchema)
	return err
}

// Feature rollout is an additional ceiling, never a substitute for RBAC/ACL.
// No content, permission or approval policy is changed by a feature toggle.
func (s *Server) canFeature(ctx context.Context, p *Principal, wid, key string) bool {
	return featureAllowed(ctx, s.DB, p, wid, key)
}
func featureAllowed(ctx context.Context, q collaborationQuery, p *Principal, wid, key string) bool {
	if p == nil || !knownFeature(key) || !validID(wid) || (p.WorkspaceID != "" && p.WorkspaceID != wid) {
		return false
	}
	var enabled bool
	err := q.QueryRow(ctx, `SELECT madi_feature_allowed($1,$2,$3)`, p.ID, wid, key).Scan(&enabled)
	return err == nil && enabled
}
func (s *Server) requireFeature(w http.ResponseWriter, r *http.Request, wid, key string) bool {
	if s.canFeature(r.Context(), current(r), wid, key) {
		return true
	}
	name := "요청한 기능"
	for _, definition := range featureCatalogue() {
		if definition.ID == key {
			name = definition.Name
		}
	}
	apiError(w, 403, name+" 기능이 현재 서비스·워크스페이스·사용자 정책에서 비활성화되어 있습니다")
	return false
}
func (s *Server) featureFlagsFor(ctx context.Context, p *Principal, wid string) map[string]bool {
	flags := map[string]bool{}
	for _, definition := range featureCatalogue() {
		flags[definition.ID] = s.canFeature(ctx, p, wid, definition.ID)
	}
	return flags
}
func (s *Server) registerFeatureOperations() {
	s.handle("GET /api/v1/workspaces/{id}/features", s.effectiveFeatures)
	s.admin("GET /api/v1/admin/features/catalogue", func(w http.ResponseWriter, r *http.Request) { jsonResponse(w, 200, featureCatalogue()) })
	s.admin("GET /api/v1/admin/features/users/{id}", s.getUserFeatures)
	s.admin("PUT /api/v1/admin/features/users/{id}", s.putUserFeatures)
	s.admin("GET /api/v1/admin/features/users/{id}/history", s.userFeaturesHistory)
	s.admin("POST /api/v1/admin/features/users/{id}/history/{version}/restore", s.restoreUserFeatures)
}
func (s *Server) effectiveFeatures(w http.ResponseWriter, r *http.Request) {
	wid := r.PathValue("id")
	if !s.canWorkspace(r.Context(), current(r), wid, false) {
		apiError(w, 403, "워크스페이스 열람 권한이 없습니다")
		return
	}
	jsonResponse(w, 200, map[string]any{"catalogue": featureCatalogue(), "effective": s.featureFlagsFor(r.Context(), current(r), wid), "notice": "서비스·워크스페이스·사용자 설정 중 하나라도 꺼져 있으면 사용할 수 없습니다. 기능이 켜져 있어도 기존 권한과 승인·정보보호 정책을 적용합니다."})
}
func (s *Server) userFeatures(r *http.Request, id string) (map[string]any, error) {
	return s.one(r.Context(), `SELECT jsonb_build_object('user',jsonb_build_object('id',u.id,'name',u.name,'email',u.email),'data',coalesce(f.data,'{}'),'version',coalesce(f.version,0),'updated_at',f.updated_at) FROM users u LEFT JOIN user_feature_flags f ON f.user_id=u.id WHERE u.id=$1`, id)
}
func (s *Server) getUserFeatures(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		apiError(w, 404, "사용자를 찾을 수 없습니다")
		return
	}
	v, err := s.userFeatures(r, id)
	respond(w, v, err)
}

type userFeatureInput struct {
	Version int64          `json:"version"`
	Data    map[string]any `json:"data"`
}

func (s *Server) putUserFeatures(w http.ResponseWriter, r *http.Request) {
	var in userFeatureInput
	if decode(r, &in) != nil || in.Data == nil {
		apiError(w, 400, "사용자 기능 설정을 확인하세요")
		return
	}
	s.saveUserFeatures(w, r, in)
}
func (s *Server) saveUserFeatures(w http.ResponseWriter, r *http.Request, in userFeatureInput) {
	id := r.PathValue("id")
	if !validID(id) || in.Version < 0 {
		apiError(w, 400, "사용자와 현재 설정 버전을 확인하세요")
		return
	}
	if err := validateFeatureFlags(in.Data); err != nil {
		apiError(w, 400, err.Error())
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	if _, err = tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock(hashtextextended($1,46273))", id); err != nil {
		respond(w, nil, err)
		return
	}
	var exists bool
	if err = tx.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)", id).Scan(&exists); err != nil || !exists {
		apiError(w, 404, "사용자를 찾을 수 없습니다")
		return
	}
	var version int64
	err = tx.QueryRow(r.Context(), "SELECT version FROM user_feature_flags WHERE user_id=$1 FOR UPDATE", id).Scan(&version)
	if err != nil && err != pgx.ErrNoRows {
		respond(w, nil, err)
		return
	}
	if version != in.Version {
		apiError(w, 409, "다른 관리자가 사용자 기능 설정을 변경했습니다. 다시 불러오세요")
		return
	}
	_, err = tx.Exec(r.Context(), `INSERT INTO user_feature_flags(user_id,data,version,updated_by) VALUES($1,$2,$3,$4) ON CONFLICT(user_id) DO UPDATE SET data=EXCLUDED.data,version=EXCLUDED.version,updated_by=EXCLUDED.updated_by,updated_at=now()`, id, jsonValue(in.Data), version+1, current(r).ID)
	if err == nil {
		_, err = tx.Exec(r.Context(), "INSERT INTO user_feature_flags_history(user_id,version,data,actor_id) VALUES($1,$2,$3,$4)", id, version+1, jsonValue(in.Data), current(r).ID)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		respond(w, nil, err)
		return
	}
	s.audit(r, "USER_FEATURE_FLAGS_CHANGE", id, map[string]any{"version": version + 1, "flags": in.Data})
	v, err := s.userFeatures(r, id)
	respond(w, v, err)
}
func (s *Server) userFeaturesHistory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		apiError(w, 404, "사용자를 찾을 수 없습니다")
		return
	}
	v, err := s.rows(r.Context(), "SELECT jsonb_build_object('version',h.version,'data',h.data,'created_at',h.created_at,'actor_name',u.name) FROM user_feature_flags_history h JOIN users u ON u.id=h.actor_id WHERE h.user_id=$1 ORDER BY h.version DESC LIMIT 100", id)
	respond(w, v, err)
}
func (s *Server) restoreUserFeatures(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	revision := r.PathValue("version")
	var in userFeatureInput
	if !validID(id) || strings.Trim(revision, "0123456789") != "" || revision == "" || decode(r, &in) != nil {
		apiError(w, 400, "복원할 설정 버전을 확인하세요")
		return
	}
	var raw []byte
	if err := s.DB.QueryRow(r.Context(), "SELECT data FROM user_feature_flags_history WHERE user_id=$1 AND version::text=$2", id, revision).Scan(&raw); err != nil {
		apiError(w, 404, "설정 버전을 찾을 수 없습니다")
		return
	}
	if err := json.Unmarshal(raw, &in.Data); err != nil {
		respond(w, nil, err)
		return
	}
	s.saveUserFeatures(w, r, in)
}
