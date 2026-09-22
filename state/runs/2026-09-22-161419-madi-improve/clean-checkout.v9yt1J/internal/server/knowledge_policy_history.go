package server

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
)

func (s *Server) registerKnowledgePolicyHistory() {
	s.admin("GET /api/v1/admin/knowledge-packages/policy/history", func(w http.ResponseWriter, r *http.Request) {
		v, e := s.rows(r.Context(), `SELECT to_jsonb(h)-'actor_id' FROM knowledge_package_policy_history h ORDER BY version DESC LIMIT 100`)
		respond(w, v, e)
	})
	s.admin("POST /api/v1/admin/knowledge-packages/policy/history/{version}/restore", s.restoreKnowledgePolicy(false))
	s.admin("POST /api/v1/admin/evidence-policy/history/{version}/restore", s.restoreKnowledgePolicy(true))
}
func (s *Server) restoreKnowledgePolicy(evidence bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		version, err := strconv.Atoi(r.PathValue("version"))
		var in struct {
			Version int  `json:"version"`
			Consent bool `json:"consent"`
		}
		if err != nil || version < 1 || decode(r, &in) != nil || in.Version < 1 || !in.Consent {
			apiError(w, 400, "현재 설정 버전·복원할 이력·새 설정 저장 동의를 확인하세요")
			return
		}
		query := `SELECT to_jsonb(h)-'actor_id'-'created_at' FROM knowledge_package_policy_history h WHERE version=$1`
		if evidence {
			query = `SELECT to_jsonb(h)-'actor_id'-'created_at' FROM knowledge_evidence_policy_history h WHERE version=$1`
		}
		row, err := s.one(r.Context(), query, version)
		if err != nil {
			apiError(w, 404, "설정 이력을 찾을 수 없습니다")
			return
		}
		row["version"] = in.Version
		body := jsonValue(row)
		request := r.Clone(r.Context())
		request.Body = io.NopCloser(bytes.NewReader(body))
		request.ContentLength = int64(len(body))
		if evidence {
			s.putEvidencePolicy(w, request)
		} else {
			s.putPackagePolicy(w, request)
		}
	}
}
