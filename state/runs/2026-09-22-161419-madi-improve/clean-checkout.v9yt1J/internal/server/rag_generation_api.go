package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (s *Server) registerRAGGenerations() {
	s.handle("GET /api/v1/workspaces/{id}/rag-generations", s.listRAGGenerations)
	s.handle("POST /api/v1/workspaces/{id}/rag-generations", s.createRAGGeneration)
	s.handle("POST /api/v1/rag-generations/{id}/index", s.queueRAGGenerationIndex)
	s.handle("POST /api/v1/rag-generations/{id}/verify", s.verifyRAGGeneration)
	s.handle("DELETE /api/v1/rag-generations/{id}", s.deleteRAGGeneration)
	s.handle("POST /api/v1/workspaces/{id}/rag-generations/activate", s.activateRAGGeneration)
	s.RegisterJobHandler("rag.hnsw", s.runRAGGenerationIndex)
}

func (s *Server) listRAGGenerations(w http.ResponseWriter, r *http.Request) {
	wid := r.PathValue("id")
	if !s.workspaceAdmin(r, wid) {
		apiError(w, 403, "색인 세대 관리 권한이 없습니다")
		return
	}
	ids, e := s.rows(r.Context(), `SELECT jsonb_build_object('id',id,'name',name,'status',status,'revision',revision,'dimensions',dimensions,'provider_fingerprint',provider_fingerprint,'created_at',created_at,'index_job_id',index_job_id) FROM rag_generations WHERE workspace_id=$1 ORDER BY created_at DESC LIMIT 21`, wid)
	if e != nil {
		respond(w, nil, e)
		return
	}
	for _, value := range ids {
		g, e := s.ragGenerationTx(r.Context(), s.DB, str(value, "id"), false)
		if e != nil {
			respond(w, nil, e)
			return
		}
		value["provider"] = map[string]any{"base_url": ragDisplayURL(str(g.Config, "rag_embedding_base_url")), "model": str(g.Config, "rag_embedding_model"), "api_key_configured": str(g.Config, "rag_embedding_api_key") != "", "allow_http": boolean(g.Config, "rag_allow_http"), "rerank_enabled": boolean(g.Config, "rag_rerank_enabled")}
		counts, e := s.one(r.Context(), `SELECT jsonb_build_object('consented_documents',count(*),'ready_documents',count(*) FILTER(WHERE i.status='ready' AND i.document_version=d.version AND i.grant_revision=g.revision),'pending_documents',count(*) FILTER(WHERE j.status IN ('pending','running')),'failed_documents',count(*) FILTER(WHERE j.status IN ('failed','cancelled'))) FROM rag_index_grants g JOIN documents d ON d.id=g.document_id LEFT JOIN rag_vector_indexes i ON i.id=g.last_job_id LEFT JOIN automation_jobs j ON j.id=g.last_job_id WHERE g.generation_id=$1 AND g.active AND d.deleted_at IS NULL AND madi_document_allowed($2,d.id,false)`, g.ID, current(r).ID)
		if e != nil {
			respond(w, nil, e)
			return
		}
		value["index"] = counts
		job, e := s.one(r.Context(), `SELECT jsonb_build_object('status',j.status,'cancel_requested',j.cancel_requested) FROM automation_jobs j WHERE j.id=NULLIF($1,'')::uuid`, g.IndexJobID)
		if e != nil && !errors.Is(e, pgx.ErrNoRows) {
			respond(w, nil, e)
			return
		}
		value["index_job"] = job
		var ready bool
		if e = s.DB.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM pg_index i JOIN pg_class c ON c.oid=i.indexrelid JOIN pg_am am ON am.oid=c.relam WHERE c.relname=$1 AND i.indrelid='rag_vector_chunks'::regclass AND i.indisvalid AND i.indisready AND am.amname='hnsw')`, ragHNSWIndexName(g.ID)).Scan(&ready); e != nil {
			respond(w, nil, e)
			return
		}
		value["ann_ready"] = ready
	}
	var active string
	var revision int64
	var mode string
	e = s.DB.QueryRow(r.Context(), `SELECT coalesce(active_id::text,''),revision,mode FROM rag_generation_state WHERE workspace_id=$1`, wid).Scan(&active, &revision, &mode)
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		respond(w, nil, e)
		return
	}
	if mode == "" {
		mode = "exact"
	}
	cfg, e := s.effectiveSettings(r.Context(), wid)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if !s.currentSearchOperator(r, wid) {
		apiError(w, 403, "현재 세대 관리 권한이 없습니다")
		return
	}
	respond(w, map[string]any{"generations": ids, "active_id": active, "state_revision": revision, "mode": mode, "enabled": boolean(cfg, "rag_enabled"), "settings_fingerprint": ragGenerationSettingsFingerprint(cfg), "scope": "현재 열람 가능한 문서의 색인 상태만 집계합니다. 기존 세대는 새 세대를 만드는 동안 계속 검색되며, 문서 전송 동의는 세대별로 필요합니다.", "limits": map[string]any{"generations": 20, "ann_dimensions": 2000, "maximum_dimensions": 8192, "validation_documents": 30}}, nil)
}

func (s *Server) createRAGGeneration(w http.ResponseWriter, r *http.Request) {
	wid, p := r.PathValue("id"), current(r)
	if !s.workspaceAdmin(r, wid) {
		apiError(w, 403, "색인 세대 관리 권한이 없습니다")
		return
	}
	var in struct {
		Name       string         `json:"name"`
		Dimensions int            `json:"dimensions"`
		Config     map[string]any `json:"config"`
		Confirm    bool           `json:"confirm"`
	}
	if decode(r, &in) != nil || !in.Confirm || strings.TrimSpace(in.Name) == "" || len(in.Name) > 200 || in.Dimensions < 1 || in.Dimensions > 8192 {
		apiError(w, 400, "세대 이름(200바이트 이내), 고정 차원(1~8192)과 설정 확인이 필요합니다")
		return
	}
	for key, value := range in.Config {
		if !slices.Contains(ragGenerationProviderKeys, key) {
			apiError(w, 400, "세대에는 임베딩·재정렬 공급자 설정만 저장할 수 있습니다")
			return
		}
		if key == "rag_allow_http" || key == "rag_rerank_enabled" {
			if _, ok := value.(bool); !ok {
				apiError(w, 400, "세대의 허용 여부는 true/false로 지정하세요")
				return
			}
		}
		if key != "rag_allow_http" && key != "rag_rerank_enabled" && key != "rag_embedding_dimensions" {
			if text, ok := value.(string); !ok || len(text) > 65536 {
				apiError(w, 400, "세대의 공급자 문자열은 64KiB 이하로 지정하세요")
				return
			}
		}
	}
	ctx := r.Context()
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(ctx)
	if e = ragGenerationOperatorTx(ctx, tx, r, wid); e != nil {
		ragIndexError(w, e)
		return
	}
	if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('madi:rag:generations:'||$1,0))`, wid); e != nil {
		respond(w, nil, e)
		return
	}
	cfg, e := s.ragSettingsTx(ctx, tx, wid)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if !boolean(cfg, "rag_enabled") {
		apiError(w, 409, "먼저 관리자 AI 검색 설정에서 검색 AI를 활성화하세요")
		return
	}
	var count int
	if e = tx.QueryRow(ctx, `SELECT count(*) FROM rag_generations WHERE workspace_id=$1`, wid).Scan(&count); e != nil {
		respond(w, nil, e)
		return
	}
	next := map[string]any{}
	for key, value := range cfg {
		next[key] = value
	}
	for _, prefix := range []string{"rag_embedding", "rag_rerank"} {
		if _, ok := in.Config[prefix+"_base_url"]; ok && str(in.Config, prefix+"_base_url") != str(cfg, prefix+"_base_url") {
			next[prefix+"_api_key"] = ""
		}
	}
	for key, value := range in.Config {
		next[key] = value
	}
	next["rag_embedding_dimensions"] = in.Dimensions
	if e = validateRAGSettings(next); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	if str(next, "rag_embedding_base_url") == "" || str(next, "rag_embedding_model") == "" {
		apiError(w, 400, "임베딩 공급자 주소와 모델이 필요합니다")
		return
	}
	protected, e := s.ProtectDocumentMetadataTx(ctx, tx, p, "", wid, map[string]any{"name": in.Name})
	if e != nil {
		ragIndexError(w, e)
		return
	}
	var nameValue map[string]any
	if e = json.Unmarshal(jsonValue(protected.Value), &nameValue); e != nil {
		respond(w, nil, e)
		return
	}
	name := strings.TrimSpace(str(nameValue, "name"))
	if name == "" || len(name) > 200 {
		apiError(w, 422, "세대 이름을 정보보호 정책에 맞게 수정하세요")
		return
	}
	// Adopt existing grants only into an immutable copy of the very same current
	// provider. This is metadata bookkeeping, never a new indexing request.
	var stateExists bool
	if e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM rag_generation_state WHERE workspace_id=$1)`, wid).Scan(&stateExists); e != nil {
		respond(w, nil, e)
		return
	}
	additional := 1
	if !stateExists {
		additional++
	}
	if count+additional > 20 {
		apiError(w, 409, "워크스페이스당 색인 세대는 최대 20개입니다")
		return
	}
	if !stateExists {
		baseline := newID()
		baselineDimensions := number(cfg, "rag_embedding_dimensions", 0)
		if baselineDimensions == 0 {
			if e = tx.QueryRow(ctx, `SELECT CASE WHEN min(i.dimensions)=max(i.dimensions) THEN coalesce(min(i.dimensions),0) ELSE 0 END FROM rag_vector_indexes i JOIN rag_index_grants g ON g.id=i.grant_id WHERE g.workspace_id=$1 AND g.generation_id IS NULL AND i.status='ready'`, wid).Scan(&baselineDimensions); e != nil {
				respond(w, nil, e)
				return
			}
		}
		cipher, e := s.encrypt(string(jsonValue(ragGenerationStoredConfig(cfg))))
		if e != nil {
			respond(w, nil, e)
			return
		}
		if _, e = tx.Exec(ctx, `INSERT INTO rag_generations(id,workspace_id,name,status,dimensions,provider_fingerprint,config_cipher,created_by) VALUES($1,$2,'기존 설정 기준 세대','active',$6,$3,$4,$5)`, baseline, wid, ragProviderFingerprint(cfg), cipher, p.ID, baselineDimensions); e != nil {
			respond(w, nil, e)
			return
		}
		if _, e = tx.Exec(ctx, `UPDATE rag_index_grants SET generation_id=$2 WHERE workspace_id=$1 AND generation_id IS NULL`, wid, baseline); e != nil {
			respond(w, nil, e)
			return
		}
		if _, e = tx.Exec(ctx, `UPDATE rag_vector_chunks c SET generation_id=$2 FROM rag_vector_indexes i WHERE i.id=c.index_id AND i.document_id IN (SELECT id FROM documents WHERE workspace_id=$1) AND c.generation_id IS NULL`, wid, baseline); e != nil {
			respond(w, nil, e)
			return
		}
		if _, e = tx.Exec(ctx, `INSERT INTO rag_generation_state(workspace_id,active_id) VALUES($1,$2)`, wid, baseline); e != nil {
			respond(w, nil, e)
			return
		}
	}
	id := newID()
	cipher, e := s.encrypt(string(jsonValue(ragGenerationStoredConfig(next))))
	if e != nil {
		respond(w, nil, e)
		return
	}
	_, e = tx.Exec(ctx, `INSERT INTO rag_generations(id,workspace_id,name,dimensions,provider_fingerprint,config_cipher,created_by) VALUES($1,$2,$3,$4,$5,$6,$7)`, id, wid, name, in.Dimensions, ragProviderFingerprint(next), cipher, p.ID)
	if e == nil {
		e = tx.Commit(ctx)
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "RAG_GENERATION_CREATED", id, map[string]any{"dimensions": in.Dimensions, "provider_fingerprint": ragProviderFingerprint(next)})
	jsonResponse(w, 201, map[string]any{"id": id, "revision": 1, "provider_fingerprint": ragProviderFingerprint(next), "status": "building", "automatic_indexing": false})
}
