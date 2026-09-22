package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type ragGenerationCohort struct {
	DocumentID    string `json:"document_id"`
	Version       int    `json:"version"`
	GrantID       string `json:"grant_id,omitempty"`
	GrantRevision int64  `json:"grant_revision,omitempty"`
	IndexID       string `json:"index_id,omitempty"`
}
type ragGenerationReport struct {
	Mode             string   `json:"mode"`
	Recall           float64  `json:"recall_at_k"`
	ExactCount       int      `json:"exact_candidates"`
	ANNCount         int      `json:"ann_candidates"`
	Documents        int      `json:"documents"`
	Truncated        bool     `json:"truncated"`
	Dimensions       int      `json:"dimensions"`
	IndexName        string   `json:"index_name,omitempty"`
	PlannedIndexUsed bool     `json:"planned_index_used"`
	Warnings         []string `json:"warnings"`
}

func ragNormalizeCohort(values []ragGenerationCohort) error {
	if len(values) < 1 || len(values) > 30 {
		return errors.New("명시적인 검증 문서 1~30개와 현재 버전이 필요합니다")
	}
	seen := map[string]bool{}
	for _, value := range values {
		if !validID(value.DocumentID) || value.Version < 1 || seen[value.DocumentID] {
			return errors.New("검증 문서 ID·버전·중복을 확인하세요")
		}
		seen[value.DocumentID] = true
	}
	sort.Slice(values, func(i, j int) bool { return values[i].DocumentID < values[j].DocumentID })
	return nil
}

func ragGenerationGrantPrincipalTx(ctx context.Context, tx pgx.Tx, g ragIndexGrant) (*Principal, error) {
	p := &Principal{ID: g.ActorID, TokenID: g.TokenID, WorkspaceID: g.WorkspaceID, PluginID: g.Constraints.PluginID, ScopeRestricted: g.Constraints.Restricted, Scopes: append([]string{}, g.Constraints.Scopes...)}
	if e := tx.QueryRow(ctx, `SELECT role,kind FROM users WHERE id=$1 AND NOT disabled FOR SHARE`, p.ID).Scan(&p.Role, &p.Kind); e != nil {
		return nil, errRAGChanged
	}
	if g.Constraints.TokenBound && g.TokenID == "" {
		return nil, errRAGChanged
	}
	if e := ragActorTx(ctx, tx, p, g.DocumentID, g.WorkspaceID, true); e != nil {
		return nil, e
	}
	return p, nil
}

// The receipt carries only IDs/versions, not source text. Every use rechecks
// both the reader and the original document indexing actor under current ACL.
func (s *Server) ragGenerationCohortTx(ctx context.Context, tx pgx.Tx, r *http.Request, g ragGeneration, values []ragGenerationCohort, requireReceipt bool) ([]ragGenerationCohort, error) {
	result := make([]ragGenerationCohort, 0, len(values))
	for _, value := range values {
		var version int
		if e := tx.QueryRow(ctx, `SELECT version FROM documents WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL FOR SHARE`, value.DocumentID, g.WorkspaceID).Scan(&version); e != nil || version != value.Version {
			return nil, errRAGChanged
		}
		if e := ragActorTx(ctx, tx, current(r), value.DocumentID, g.WorkspaceID, false); e != nil {
			return nil, e
		}
		grant, e := ragGrantForGenerationTx(ctx, tx, value.DocumentID, g.ID, true)
		if e != nil || !grant.Active || grant.Provider != g.Provider || grant.Version != version {
			return nil, errRAGChanged
		}
		if requireReceipt && (value.GrantID != grant.ID || value.GrantRevision != grant.Revision || value.IndexID != grant.JobID) {
			return nil, errRAGChanged
		}
		if _, e = ragGenerationGrantPrincipalTx(ctx, tx, grant); e != nil {
			return nil, e
		}
		var ready bool
		if e = tx.QueryRow(ctx, `SELECT status='ready' AND indexed_chunks=total_chunks AND indexed_chunks>0 AND dimensions=$2 AND document_version=$3 AND grant_revision=$4 AND provider_fingerprint=$5 FROM rag_vector_indexes WHERE id=$1 AND grant_id=$6 FOR SHARE`, grant.JobID, g.Dimensions, version, grant.Revision, g.Provider, grant.ID).Scan(&ready); e != nil || !ready {
			return nil, errRAGChanged
		}
		result = append(result, ragGenerationCohort{DocumentID: value.DocumentID, Version: version, GrantID: grant.ID, GrantRevision: grant.Revision, IndexID: grant.JobID})
	}
	return result, nil
}

func (s *Server) ragGenerationVerifyGuard(ctx context.Context, r *http.Request, g ragGeneration, cohort []ragGenerationCohort, stateRevision int64, fingerprint string) error {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if e = ragGenerationOperatorTx(ctx, tx, r, g.WorkspaceID); e != nil {
		return e
	}
	actual, e := s.ragGenerationTx(ctx, tx, g.ID, true)
	if e != nil || actual.Status == "disabled" || actual.Revision != g.Revision || actual.Provider != g.Provider {
		return errRAGChanged
	}
	if _, e = s.ragGenerationCohortTx(ctx, tx, r, actual, cohort, true); e != nil {
		return e
	}
	cfg, e := s.ragSettingsTx(ctx, tx, g.WorkspaceID)
	if e != nil || !boolean(cfg, "rag_enabled") || ragGenerationSettingsFingerprint(cfg) != fingerprint {
		return errRAGChanged
	}
	var revision int64
	if e = tx.QueryRow(ctx, `SELECT revision FROM rag_generation_state WHERE workspace_id=$1 FOR SHARE`, g.WorkspaceID).Scan(&revision); e != nil || revision != stateRevision {
		return errRAGChanged
	}
	return nil
}

func (s *Server) verifyRAGGeneration(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		apiError(w, 400, "색인 세대 ID를 확인하세요")
		return
	}
	g, e := s.ragGenerationTx(r.Context(), s.DB, id, false)
	if e != nil || !s.workspaceAdmin(r, g.WorkspaceID) {
		apiError(w, 403, "색인 세대 검증 권한이 없습니다")
		return
	}
	var in struct {
		Revision  int64                 `json:"revision"`
		Provider  string                `json:"provider_fingerprint"`
		Query     string                `json:"query"`
		Mode      string                `json:"mode"`
		Consent   bool                  `json:"consent"`
		Documents []ragGenerationCohort `json:"documents"`
	}
	if decode(r, &in) != nil || !in.Consent || strings.TrimSpace(in.Query) == "" || len(in.Query) > 4000 || !oneOf(in.Mode, "exact", "ann", "verify") || ragNormalizeCohort(in.Documents) != nil {
		apiError(w, 400, "검증 모드·질문(4000바이트 이내)·문서 1~30개와 질문 전송 동의가 필요합니다")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(ctx)
	if e = ragGenerationOperatorTx(ctx, tx, r, g.WorkspaceID); e != nil {
		ragIndexError(w, e)
		return
	}
	g, e = s.ragGenerationTx(ctx, tx, id, true)
	if e != nil || g.Status == "disabled" || g.Revision != in.Revision || g.Provider != in.Provider || g.Dimensions < 1 {
		ragIndexError(w, errRAGChanged)
		return
	}
	cohort, e := s.ragGenerationCohortTx(ctx, tx, r, g, in.Documents, false)
	if e != nil {
		ragIndexError(w, e)
		return
	}
	cfg, e := s.ragSettingsTx(ctx, tx, g.WorkspaceID)
	if e != nil || !boolean(cfg, "rag_enabled") {
		ragIndexError(w, errRAGChanged)
		return
	}
	fingerprint := ragGenerationSettingsFingerprint(cfg)
	cfg, e = s.ragGenerationConfig(ctx, tx, cfg, g.WorkspaceID, g.ID)
	if e != nil {
		ragIndexError(w, e)
		return
	}
	var stateRevision int64
	if e = tx.QueryRow(ctx, `SELECT revision FROM rag_generation_state WHERE workspace_id=$1 FOR SHARE`, g.WorkspaceID).Scan(&stateRevision); e != nil {
		ragIndexError(w, e)
		return
	}
	protection, e := s.ProtectDocumentMetadataTx(ctx, tx, current(r), "", g.WorkspaceID, map[string]any{"query": in.Query})
	if e != nil {
		ragIndexError(w, e)
		return
	}
	if protection.Changed {
		apiError(w, 422, "질문에 정보보호 정책 적용이 필요합니다. 민감한 내용을 제거하고 다시 동의하세요")
		return
	}
	if e = tx.Commit(ctx); e != nil {
		respond(w, nil, e)
		return
	}
	guard := func(check context.Context) error {
		return s.ragGenerationVerifyGuard(check, r, g, cohort, stateRevision, fingerprint)
	}
	if e = guard(ctx); e != nil {
		ragIndexError(w, e)
		return
	}
	watched, stop, guardErr := ragWatch(ctx, guard)
	defer stop()
	vectors, e := ragEmbeddings(watched, ragEmbeddingProvider(cfg), []string{in.Query})
	if e != nil {
		stop()
		if *guardErr != nil {
			ragIndexError(w, *guardErr)
		} else if ctx.Err() != nil {
			apiError(w, 408, "검증 요청이 취소되었거나 제한 시간을 초과했습니다")
		} else {
			apiError(w, 502, e.Error())
		}
		return
	}
	if len(vectors) != 1 || len(vectors[0]) != g.Dimensions {
		apiError(w, 422, "고정한 세대 차원과 질문 임베딩 응답이 다릅니다")
		return
	}
	if e = guard(watched); e != nil {
		ragIndexError(w, e)
		return
	}
	ids := make([]string, 0, len(cohort))
	for _, item := range cohort {
		ids = append(ids, item.DocumentID)
	}
	exactCfg := map[string]any{}
	for key, value := range cfg {
		exactCfg[key] = value
	}
	exactCfg["rag_backend"] = "array"
	exact, diag, e := s.ragExactVectorCandidates(watched, current(r), g.WorkspaceID, "", exactCfg, vectors[0], g.ID, ids)
	if e != nil {
		ragIndexError(w, e)
		return
	}
	report := ragGenerationReport{Mode: in.Mode, Recall: 1, ExactCount: len(exact), Documents: len(ids), Truncated: diag.Truncated, Dimensions: g.Dimensions, Warnings: append([]string{}, diag.Warnings...)}
	if len(exact) == 0 {
		apiError(w, 409, "현재 권한과 버전에서 검증할 준비된 벡터가 없습니다")
		return
	}
	if in.Mode != "exact" {
		ann, annDiag, e := s.ragANNVectorCandidates(watched, current(r), g.WorkspaceID, "", cfg, vectors[0], g.ID, ids)
		if e != nil {
			ragIndexError(w, e)
			return
		}
		report.ANNCount = len(ann)
		report.Recall = ragRecall(exact, ann)
		report.PlannedIndexUsed = annDiag.PlannedIndexUsed
		report.IndexName = annDiag.IndexName
		report.Warnings = append(report.Warnings, annDiag.Warnings...)
	}
	report.Warnings = append(report.Warnings, "선택한 문서·질문 1건과 현재 권한에서만 측정한 결과입니다. 전체 조직이나 다른 사용자의 ACL recall을 보증하지 않습니다.")
	stop()
	if *guardErr != nil {
		ragIndexError(w, *guardErr)
		return
	}
	if e = guard(ctx); e != nil {
		ragIndexError(w, e)
		return
	}
	tx, e = s.DB.Begin(ctx)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(ctx)
	if e = ragGenerationOperatorTx(ctx, tx, r, g.WorkspaceID); e != nil {
		ragIndexError(w, e)
		return
	}
	fresh, e := s.ragGenerationTx(ctx, tx, g.ID, true)
	if e != nil || fresh.Status == "disabled" || fresh.Revision != g.Revision {
		ragIndexError(w, errRAGChanged)
		return
	}
	if _, e = s.ragGenerationCohortTx(ctx, tx, r, g, cohort, true); e != nil {
		ragIndexError(w, e)
		return
	}
	actualCfg, e := s.ragSettingsTx(ctx, tx, g.WorkspaceID)
	if e != nil || ragGenerationSettingsFingerprint(actualCfg) != fingerprint {
		ragIndexError(w, errRAGChanged)
		return
	}
	var actualRevision int64
	if e = tx.QueryRow(ctx, `SELECT revision FROM rag_generation_state WHERE workspace_id=$1 FOR SHARE`, g.WorkspaceID).Scan(&actualRevision); e != nil || actualRevision != stateRevision {
		ragIndexError(w, errRAGChanged)
		return
	}
	validation := newID()
	cookie, _ := r.Cookie("madi_session")
	_, e = tx.Exec(ctx, `INSERT INTO rag_generation_validations(id,generation_id,workspace_id,actor_id,session_hash,generation_revision,state_revision,settings_fingerprint,query_hash,cohort,report,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,now()+interval '15 minutes')`, validation, g.ID, g.WorkspaceID, current(r).ID, digest(cookie.Value), g.Revision, stateRevision, fingerprint, digest(in.Query), jsonValue(cohort), jsonValue(report))
	if e == nil {
		e = tx.Commit(ctx)
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "RAG_GENERATION_VERIFY", g.ID, map[string]any{"validation_id": validation, "documents": len(ids), "mode": in.Mode, "recall_at_k": report.Recall})
	jsonResponse(w, 200, map[string]any{"validation_id": validation, "state_revision": stateRevision, "generation_revision": g.Revision, "expires_in_seconds": 900, "report": report, "can_activate": !report.Truncated && report.Recall >= 0.9})
}

// Decoding helpers keep the SQL receipt schema private to the server.
func ragDecodeValidation(cohortRaw, reportRaw []byte) ([]ragGenerationCohort, ragGenerationReport, error) {
	var cohort []ragGenerationCohort
	var report ragGenerationReport
	if e := json.Unmarshal(cohortRaw, &cohort); e != nil {
		return nil, report, e
	}
	if e := ragNormalizeCohort(cohort); e != nil {
		return nil, report, e
	}
	e := json.Unmarshal(reportRaw, &report)
	return cohort, report, e
}
