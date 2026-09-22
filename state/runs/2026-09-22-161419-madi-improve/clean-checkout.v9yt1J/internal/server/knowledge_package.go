package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

//go:embed knowledge_package.sql
var knowledgePackageSchema string

func (s *Server) migrateKnowledgePackages(ctx context.Context) error {
	_, err := s.DB.Exec(ctx, knowledgePackageSchema)
	return err
}
func (s *Server) registerKnowledgePackages() {
	s.admin("GET /api/v1/admin/knowledge-packages/policy", s.getPackagePolicy)
	s.admin("PUT /api/v1/admin/knowledge-packages/policy", s.putPackagePolicy)
	s.handle("GET /api/v1/knowledge/packages/policy", s.getPackagePolicy)
	s.handle("GET /api/v1/knowledge/packages/context", func(w http.ResponseWriter, r *http.Request) {
		wid := r.URL.Query().Get("workspace_id")
		if _, err := s.packagePrincipal(r, wid); err != nil {
			apiError(w, 403, "워크스페이스 문서 조회 권한이 필요합니다")
			return
		}
		cfg, err := s.effectiveSettings(r.Context(), wid)
		if err != nil {
			respond(w, nil, err)
			return
		}
		jsonResponse(w, 200, map[string]any{"model": str(cfg, "ai_model"), "provider_url": ragDisplayURL(str(cfg, "ai_base_url")), "ai_enabled": boolean(cfg, "ai_enabled")})
	})
	s.handle("POST /api/v1/knowledge/packages", s.createKnowledgePackage)
	s.handle("GET /api/v1/knowledge/packages", s.listKnowledgePackages)
	s.handle("GET /api/v1/knowledge/packages/{id}", s.getKnowledgePackage)
	s.handle("POST /api/v1/knowledge/packages/{id}/export", s.exportKnowledgePackage)
	s.handle("DELETE /api/v1/knowledge/packages/{id}", s.deleteKnowledgePackage)
}

type knowledgePackagePolicy struct {
	Enabled        bool   `json:"enabled"`
	RetentionHours int    `json:"retention_hours"`
	TokenCounter   string `json:"token_counter"`
	AllowHTTP      bool   `json:"allow_http"`
	Version        int    `json:"version"`
}

func (s *Server) packagePolicy(ctx context.Context) (knowledgePackagePolicy, error) {
	var p knowledgePackagePolicy
	err := s.DB.QueryRow(ctx, `SELECT enabled,retention_hours,token_counter,allow_http,version FROM knowledge_package_policy WHERE id=1`).Scan(&p.Enabled, &p.RetentionHours, &p.TokenCounter, &p.AllowHTTP, &p.Version)
	return p, err
}
func (s *Server) getPackagePolicy(w http.ResponseWriter, r *http.Request) {
	p, e := s.packagePolicy(r.Context())
	respond(w, p, e)
}
func (s *Server) putPackagePolicy(w http.ResponseWriter, r *http.Request) {
	var in knowledgePackagePolicy
	if decode(r, &in) != nil || in.Version < 1 || in.Version > 2147483646 || in.RetentionHours < 1 || in.RetentionHours > 168 || !oneOf(in.TokenCounter, "estimate", "responses") {
		apiError(w, 400, "패키지 정책의 버전·보존 시간(1~168)·토큰 계산 방식을 확인하세요")
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	result, err := tx.Exec(r.Context(), `UPDATE knowledge_package_policy SET enabled=$1,retention_hours=$2,token_counter=$3,allow_http=$4,version=version+1,updated_at=now() WHERE id=1 AND version=$5`, in.Enabled, in.RetentionHours, in.TokenCounter, in.AllowHTTP, in.Version)
	if err != nil {
		respond(w, nil, err)
		return
	}
	if result.RowsAffected() != 1 {
		apiError(w, 409, "패키지 정책이 변경되었습니다")
		return
	}
	_, err = tx.Exec(r.Context(), `UPDATE knowledge_packages SET expires_at=LEAST(expires_at,created_at+make_interval(hours=>$1))`, in.RetentionHours)
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO knowledge_package_policy_history(version,actor_id,enabled,retention_hours,token_counter,allow_http) VALUES($1,$2,$3,$4,$5,$6)`, in.Version+1, current(r).ID, in.Enabled, in.RetentionHours, in.TokenCounter, in.AllowHTTP)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		respond(w, nil, err)
		return
	}
	s.audit(r, "KNOWLEDGE_PACKAGE_POLICY", "", map[string]any{"enabled": in.Enabled, "token_counter": in.TokenCounter, "retention_hours": in.RetentionHours})
	s.getPackagePolicy(w, r)
}

type packageDocumentInput struct {
	ID        string `json:"id"`
	Version   int    `json:"version"`
	Mandatory bool   `json:"mandatory"`
	Reason    string `json:"reason"`
}
type knowledgePackageInput struct {
	WorkspaceID  string                   `json:"workspace_id"`
	Purpose      string                   `json:"purpose"`
	AllowedScope string                   `json:"allowed_scope"`
	Model        string                   `json:"model"`
	TokenBudget  int                      `json:"token_budget"`
	Documents    []packageDocumentInput   `json:"documents"`
	Attachments  []packageAttachmentInput `json:"attachments"`
	Counter      string                   `json:"counter"`
	Consent      bool                     `json:"consent"`
	ModelConsent bool                     `json:"model_consent"`
}

// Same authenticated request identity is reloaded for every long-running
// token-count step. Stored packages confer no authority beyond document:read.
func (s *Server) packagePrincipal(r *http.Request, wid string) (*Principal, error) {
	initial := current(r)
	if initial == nil || initial.PluginID != "" || !hasIntegrationScope(initial, "document:read") || (initial.WorkspaceID != "" && initial.WorkspaceID != wid) {
		return nil, errRAGChanged
	}
	var p *Principal
	var err error
	if initial.TokenID != "" {
		p, err = s.tokenPrincipal(r.WithContext(context.WithValue(r.Context(), integrationCountedTokenKey{}, initial.TokenID)))
	} else {
		p, err = s.workerPrincipal(r.Context(), initial.ID, "", wid)
		cookie, e := r.Cookie("madi_session")
		var active bool
		if e != nil || s.DB.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM sessions WHERE user_id=$1 AND token_hash=$2 AND expires_at>now())`, initial.ID, digest(cookie.Value)).Scan(&active) != nil || !active {
			return nil, errRAGChanged
		}
	}
	if err != nil || p == nil || p.ID != initial.ID || !hasIntegrationScope(p, "document:read") || !s.canWorkspace(r.Context(), p, wid, false) {
		return nil, errRAGChanged
	}
	return p, nil
}

func (s *Server) createKnowledgePackage(w http.ResponseWriter, r *http.Request) {
	var in knowledgePackageInput
	if decode(r, &in) != nil || !validID(in.WorkspaceID) || !in.Consent || len(strings.TrimSpace(in.Purpose)) < 1 || len(in.Purpose) > 4000 || len(in.AllowedScope) > 4000 || len(in.Model) > 256 || strings.TrimSpace(in.Model) == "" || in.TokenBudget < 512 || in.TokenBudget > 262144 || len(in.Documents)+len(in.Attachments) < 1 || len(in.Documents)+len(in.Attachments) > 32 || !oneOf(in.Counter, "estimate", "responses") {
		apiError(w, 400, "업무 목적·모델·512~262144 토큰 예산·문서/첨부 구간 합계 1~32개·계산 방식·보관 동의를 확인하세요")
		return
	}
	seen := map[string]bool{}
	for _, d := range in.Documents {
		if !validID(d.ID) || d.Version < 1 || d.Version > 2147483647 || len(d.Reason) > 1000 || seen[d.ID] {
			apiError(w, 400, "문서 ID·버전·중복·포함 이유를 확인하세요")
			return
		}
		seen[d.ID] = true
	}
	for _, a := range in.Attachments {
		key := a.Source.FragmentID
		if !validID(a.Source.ID) || !validID(key) || a.Source.Version < 1 || a.Source.Version > 2147483647 || len(a.Reason) > 1000 || seen[key] {
			apiError(w, 400, "첨부 구간 ID·부모 버전·포함 이유·중복을 확인하세요")
			return
		}
		seen[key] = true
	}
	p, err := s.packagePrincipal(r, in.WorkspaceID)
	if err != nil {
		apiError(w, 403, "현재 문서 조회 권한이 필요합니다")
		return
	}
	policy, err := s.packagePolicy(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	if !policy.Enabled {
		apiError(w, 403, "지식 패키지 기능이 비활성화되어 있습니다")
		return
	}
	cfg, err := s.effectiveSettings(r.Context(), in.WorkspaceID)
	if err != nil {
		respond(w, nil, err)
		return
	}
	if in.Counter == "responses" && (policy.TokenCounter != "responses" || !in.ModelConsent || !hasIntegrationScope(p, "ai:execute") || !boolean(cfg, "ai_enabled") || in.Model != str(cfg, "ai_model")) {
		apiError(w, 403, "모델 토큰 계산은 관리자 허용·현재 AI 모델·ai:execute 권한·원문 전송 동의가 필요합니다")
		return
	}
	payload := knowledgePackagePayload{Format: "madi-knowledge-package-v1", Purpose: in.Purpose, AllowedScope: in.AllowedScope, Model: in.Model, TokenBudget: in.TokenBudget, Counter: in.Counter, CountNotice: "UTF-8 바이트를 토큰 예산 단위로 보수적으로 계산한 추정치입니다. 해당 모델의 정확한 토큰 수나 컨텍스트 수용을 보증하지 않습니다. 실제 도구·대화·응답 토큰은 별도로 확보하세요."}
	if in.Counter == "responses" {
		payload.SettingsFingerprint = aiHistoryProvider(cfg)
		payload.CountNotice = "설정된 모델 공급자의 단일 input_tokens 요청 결과입니다. 다른 메시지·도구·대화 이력·출력 토큰은 포함하지 않습니다."
	}
	ordered := append([]packageDocumentInput{}, in.Documents...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	candidates := []knowledgePackageQuote{}
	allRefs := []aiSource{}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	// Lock all parent documents before any extraction/settings row. Mixed
	// sources must not invert the canonical document → extraction lock order.
	parents := []string{}
	for _, d := range in.Documents {
		parents = append(parents, d.ID)
	}
	for _, a := range in.Attachments {
		parents = append(parents, a.Source.ID)
	}
	if _, err = tx.Exec(r.Context(), `SELECT id FROM documents WHERE id=ANY($1::uuid[]) ORDER BY id FOR SHARE`, parents); err != nil {
		respond(w, nil, err)
		return
	}
	bytes := 0
	for _, d := range ordered {
		var title, markdown string
		var version int
		err = tx.QueryRow(r.Context(), `SELECT title,markdown,version FROM documents WHERE id=$1 AND workspace_id=$2 AND version=$3 AND deleted_at IS NULL AND madi_document_allowed($4,id,false) FOR SHARE`, d.ID, in.WorkspaceID, d.Version, p.ID).Scan(&title, &markdown, &version)
		if err != nil {
			apiError(w, 409, "선택한 자료 또는 현재 접근 범위가 변경되었습니다. 목록을 다시 확인하세요")
			return
		}
		chunks, e := chunkMarkdown(markdown, 6144, 0)
		if e != nil {
			apiError(w, 422, "선택 자료를 안전한 원문 구간으로 분리할 수 없습니다")
			return
		}
		if len(chunks) == 0 && d.Mandatory {
			apiError(w, 422, "필수 정책 본문이 비어 있습니다")
			return
		}
		for i, c := range chunks {
			if !d.Mandatory && i >= 8 {
				payload.OmittedChunks++
				payload.OmittedBytes += len(c.Content)
				continue
			}
			bytes += len(c.Content)
			if bytes > 2<<20 {
				apiError(w, 422, "선택한 필수 정책과 원문 구간의 준비 크기는 2MiB 이하여야 합니다")
				return
			}
			reason := strings.TrimSpace(d.Reason)
			if reason == "" {
				reason = "사용자가 이 업무의 참고 자료로 선택한 문서"
			}
			score := 0
			for _, word := range aiQueryWord.FindAllString(in.Purpose, 24) {
				word = strings.ToLower(word)
				if strings.Contains(strings.ToLower(title), word) {
					score += 4
				}
				if strings.Contains(strings.ToLower(c.Heading), word) {
					score += 2
				}
				if strings.Contains(strings.ToLower(c.Content), word) {
					score++
				}
			}
			q := knowledgePackageQuote{evidenceQuote: evidenceQuote{sourceFromChunk(d.ID, title, version, c), c.Content}, Mandatory: d.Mandatory, Reason: reason, Representation: "original", Score: score}
			candidates = append(candidates, q)
		}
		allRefs = append(allRefs, aiSource{ID: d.ID, Version: version})
	}
	for _, a := range in.Attachments {
		q, e := s.packageAttachmentTx(r.Context(), tx, p, a)
		if e != nil {
			apiError(w, 409, "선택한 첨부 구간·현재 권한·추출 버전이 변경되었습니다")
			return
		}
		bytes += len(q.Text)
		if bytes > 2<<20 {
			apiError(w, 422, "선택 원문 준비 크기는 2MiB 이하여야 합니다")
			return
		}
		candidates = append(candidates, q)
		allRefs = append(allRefs, q.aiSource)
	}
	if err = s.checkEvidenceProtection(r.Context(), tx, p, in.WorkspaceID, map[string]any{"purpose": in.Purpose, "scope": in.AllowedScope, "quotes": candidates}); err != nil {
		apiError(w, 422, err.Error())
		return
	}
	var protectionRevision int
	var protectionData []byte
	if err = tx.QueryRow(r.Context(), `SELECT revision,data FROM protection_settings WHERE id=1 FOR SHARE`).Scan(&protectionRevision, &protectionData); err != nil {
		respond(w, nil, err)
		return
	}
	protectionFingerprint := digest(string(protectionData))
	_ = tx.Rollback(r.Context()) // Network token counting must not hold document locks.
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	check := func(ctx context.Context) error {
		rq := r.WithContext(ctx)
		actor, e := s.packagePrincipal(rq, in.WorkspaceID)
		if e != nil {
			return e
		}
		fresh, e := s.packagePolicy(ctx)
		if e != nil || !fresh.Enabled || fresh.Version != policy.Version {
			return errRAGChanged
		}
		var revision int
		var data []byte
		if e = s.DB.QueryRow(ctx, `SELECT revision,data FROM protection_settings WHERE id=1`).Scan(&revision, &data); e != nil || revision != protectionRevision || digest(string(data)) != protectionFingerprint {
			return errRAGChanged
		}
		var count int
		if s.DB.QueryRow(ctx, `SELECT count(*) FROM jsonb_to_recordset($1::jsonb) AS ref(id uuid,version integer) JOIN documents d ON d.id=ref.id WHERE d.workspace_id=$2 AND d.version=ref.version AND d.deleted_at IS NULL AND madi_document_allowed($3,d.id,false)`, jsonValue(allRefs), in.WorkspaceID, actor.ID).Scan(&count) != nil || count != len(allRefs) {
			return errRAGChanged
		}
		if in.Counter == "responses" {
			if e = s.validateAIStream(rq, actor, in.WorkspaceID, allRefs, cfg); e != nil {
				return e
			}
		}
		if e = s.validateAttachmentAISources(ctx, actor, allRefs); e != nil {
			return e
		}
		return nil
	}
	ctx, stop, watchErr := ragWatch(ctx, check)
	defer stop()
	counter := packageTokenCounter(estimatePackageTokens)
	if in.Counter == "responses" {
		counter = func(ctx context.Context, text string) (int, error) {
			if e := check(ctx); e != nil {
				return 0, e
			}
			return responsePackageTokens(ctx, cfg, policy.AllowHTTP, text)
		}
	}
	payload, err = fitKnowledgePackage(ctx, payload, candidates, counter)
	stop()
	if *watchErr != nil {
		err = *watchErr
	}
	// stop cancels its child context, so final validation uses the request context.
	if err == nil {
		err = check(r.Context())
	}
	if err != nil {
		apiError(w, 409, err.Error())
		return
	}
	tx, err = s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	var currentPolicy int
	if err = tx.QueryRow(r.Context(), `SELECT version FROM knowledge_package_policy WHERE id=1 AND enabled FOR SHARE`).Scan(&currentPolicy); err != nil || currentPolicy != policy.Version {
		apiError(w, 409, "패키지 정책이 변경되었습니다")
		return
	}
	refs := []aiSource{}
	unique := map[string]bool{}
	for _, q := range payload.Quotes {
		key := q.ID + ":" + q.AttachmentID
		if unique[key] {
			continue
		}
		unique[key] = true
		ref := aiSource{ID: q.ID, Version: q.Version}
		if q.AttachmentID != "" {
			ref = q.aiSource
		}
		refs = append(refs, ref)
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].ID < refs[j].ID })
	for _, ref := range refs {
		var version int
		if err = tx.QueryRow(r.Context(), `SELECT version FROM documents WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL AND madi_document_allowed($3,id,false) FOR SHARE`, ref.ID, in.WorkspaceID, p.ID).Scan(&version); err != nil || version != ref.Version {
			apiError(w, 409, "패키지 근거가 변경되었습니다. 다시 구성하세요")
			return
		}
	}
	for _, q := range payload.Quotes {
		if q.AttachmentID != "" {
			if _, e := s.attachmentCitationTx(r.Context(), tx, p, q.aiSource, true); e != nil {
				apiError(w, 409, "패키지 첨부 근거가 변경되었습니다. 다시 구성하세요")
				return
			}
		}
	}
	if err = s.checkEvidenceProtection(r.Context(), tx, p, in.WorkspaceID, payload); err != nil {
		apiError(w, 422, err.Error())
		return
	}
	scopes := []string{"document:read"}
	if in.Counter == "responses" {
		scopes = append(scopes, "ai:execute")
		fresh, e := s.ragSettingsTx(r.Context(), tx, in.WorkspaceID)
		if e != nil || !boolean(fresh, "ai_enabled") || aiHistoryProvider(fresh) != payload.SettingsFingerprint {
			apiError(w, 409, "원문 토큰 계산 후 AI 공급자 설정이 변경되었습니다")
			return
		}
	}
	if err = s.knowledgeActorTx(r, tx, in.WorkspaceID, scopes...); err != nil {
		apiError(w, 403, err.Error())
		return
	}
	plain := string(jsonValue(payload))
	if len(plain) > 4<<20 {
		apiError(w, 422, "패키지 직렬화 크기는 4MiB 이하여야 합니다")
		return
	}
	sealed, err := s.encrypt(plain)
	id := newID()
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO knowledge_packages(id,owner_id,workspace_id,source_refs,ciphertext,payload_hash,expires_at) VALUES($1,$2,$3,$4,$5,$6,now()+make_interval(hours=>$7))`, id, p.ID, in.WorkspaceID, jsonValue(refs), sealed, digest(plain), policy.RetentionHours)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		respond(w, nil, err)
		return
	}
	s.audit(r, "KNOWLEDGE_PACKAGE_CREATE", id, map[string]any{"source_count": len(refs), "token_count": payload.TokenCount, "counter": payload.Counter})
	jsonResponse(w, 201, map[string]any{"id": id, "token_count": payload.TokenCount, "token_budget": payload.TokenBudget, "counter": payload.Counter, "omitted_chunks": payload.OmittedChunks})
}

func (s *Server) listKnowledgePackages(w http.ResponseWriter, r *http.Request) {
	p, err := s.packagePrincipal(r, r.URL.Query().Get("workspace_id"))
	if err != nil {
		apiError(w, 403, "문서 조회 권한이 필요합니다")
		return
	}
	items, err := s.rows(r.Context(), `SELECT jsonb_build_object('id',p.id,'created_at',p.created_at,'expires_at',p.expires_at,'export_count',p.export_count,'source_count',jsonb_array_length(p.source_refs),'stale',EXISTS(
SELECT 1 FROM jsonb_to_recordset(p.source_refs) AS ref(id uuid,version integer,attachment_id uuid,attachment_checksum text,extraction_id uuid,extraction_revision bigint) JOIN documents d ON d.id=ref.id
WHERE d.version<>ref.version OR (ref.attachment_id IS NOT NULL AND NOT EXISTS(
SELECT 1 FROM attachments a JOIN attachment_extraction_heads h ON h.attachment_id=a.id JOIN attachment_extractions e ON e.id=h.extraction_id
JOIN attachment_extraction_settings settings ON settings.id=1 AND (settings.data->>'enabled')::boolean AND settings.revision=e.policy_revision
WHERE a.id=ref.attachment_id AND a.document_id=d.id AND a.checksum_sha256=ref.attachment_checksum AND e.id=ref.extraction_id AND e.revision=ref.extraction_revision AND e.status='ready'))))
FROM knowledge_packages p WHERE p.workspace_id=$1 AND p.owner_id=$2 AND madi_package_allowed($2,p.id) ORDER BY created_at DESC,id LIMIT 100`, r.URL.Query().Get("workspace_id"), p.ID)
	respond(w, items, err)
}
func (s *Server) readKnowledgePackage(r *http.Request, tx pgx.Tx) (map[string]any, error) {
	p := current(r)
	id := r.PathValue("id")
	if p == nil || !validID(id) || p.PluginID != "" || !hasIntegrationScope(p, "document:read") {
		return nil, pgx.ErrNoRows
	}
	var wid, sealed, hash string
	var created, expires time.Time
	var exports int
	err := tx.QueryRow(r.Context(), `SELECT workspace_id::text,ciphertext,payload_hash,created_at,expires_at,export_count FROM knowledge_packages WHERE id=$1 AND owner_id=$2 AND ($3='' OR workspace_id::text=$3) AND madi_package_allowed($2,id) FOR SHARE`, id, p.ID, p.WorkspaceID).Scan(&wid, &sealed, &hash, &created, &expires, &exports)
	if err != nil {
		return nil, err
	}
	plain, err := s.decrypt(sealed)
	var payload knowledgePackagePayload
	if err != nil || digest(plain) != hash || json.Unmarshal([]byte(plain), &payload) != nil || payload.Format != "madi-knowledge-package-v1" {
		return nil, errors.New("패키지 무결성을 확인할 수 없습니다")
	}
	if err = s.checkEvidenceProtection(r.Context(), tx, p, wid, payload); err != nil {
		return nil, err
	}
	stale := false
	for _, q := range payload.Quotes {
		var version int
		if err = tx.QueryRow(r.Context(), `SELECT version FROM documents WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL AND madi_document_allowed($3,id,false) FOR SHARE`, q.ID, wid, p.ID).Scan(&version); err != nil {
			return nil, pgx.ErrNoRows
		}
		if digest(q.Text) != q.ContentHash {
			return nil, errors.New("패키지 원문 해시가 일치하지 않습니다")
		}
		stale = stale || version != q.Version
		if q.AttachmentID != "" {
			value, e := s.attachmentCitationTx(r.Context(), tx, p, q.aiSource, false)
			if e != nil {
				return nil, pgx.ErrNoRows
			}
			stale = stale || !value.Fresh
		}
	}
	if err = s.knowledgeActorTx(r, tx, wid, "document:read"); err != nil {
		return nil, err
	}
	if !expires.After(time.Now()) {
		return nil, pgx.ErrNoRows
	}
	return map[string]any{"id": id, "workspace_id": wid, "created_at": created, "expires_at": expires, "export_count": exports, "stale": stale, "package": payload, "notice": "이 패키지는 도구 실행 권한이 아닙니다. 전달한 사본은 이후 권한 철회로 회수할 수 없습니다. 새 조회·내보내기는 현재 권한과 버전을 검사합니다."}, nil
}
func (s *Server) getKnowledgePackage(w http.ResponseWriter, r *http.Request) {
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	v, err := s.readKnowledgePackage(r, tx)
	if errors.Is(err, pgx.ErrNoRows) {
		apiError(w, 404, "현재 접근할 수 있는 지식 패키지가 없습니다")
		return
	}
	if err != nil {
		apiError(w, 409, err.Error())
		return
	}
	if current(r).TokenID != "" {
		payload := v["package"].(knowledgePackagePayload)
		delete(v, "package")
		v["token_count"], v["token_budget"], v["counter"] = payload.TokenCount, payload.TokenBudget, payload.Counter
		v["export_required"] = true
		v["notice"] = "외부 에이전트는 전달 대상과 동의를 명시한 export_knowledge_package로 현재 버전의 원문 묶음을 가져오세요. 조회 권한이 도구 실행 권한을 부여하지 않습니다."
	}
	respond(w, v, nil)
}
func (s *Server) exportKnowledgePackage(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Consent bool   `json:"consent"`
		Target  string `json:"target"`
	}
	if decode(r, &in) != nil || !in.Consent || strings.TrimSpace(in.Target) == "" || len(in.Target) > 200 {
		apiError(w, 400, "전달 대상과 사본 회수 불가에 대한 동의가 필요합니다")
		return
	}
	if !validID(r.PathValue("id")) {
		apiError(w, 404, "지식 패키지가 없습니다")
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	if _, err = tx.Exec(r.Context(), `SELECT id FROM knowledge_packages WHERE id=$1 AND owner_id=$2 FOR UPDATE`, r.PathValue("id"), current(r).ID); err != nil {
		respond(w, nil, err)
		return
	}
	v, err := s.readKnowledgePackage(r, tx)
	if err != nil {
		apiError(w, 404, "현재 접근할 수 있는 지식 패키지가 없습니다")
		return
	}
	if boolean(v, "stale") {
		apiError(w, 409, "원문 버전이 변경되었습니다. 재검토하여 새 패키지를 구성하세요")
		return
	}
	_, err = tx.Exec(r.Context(), `UPDATE knowledge_packages SET export_count=export_count+1,last_exported_at=now() WHERE id=$1`, r.PathValue("id"))
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		respond(w, nil, err)
		return
	}
	// Target may be an internal agent name; do not copy private text into audit.
	s.audit(r, "KNOWLEDGE_PACKAGE_EXPORT", r.PathValue("id"), map[string]any{"target_hash": digest(in.Target)})
	w.Header().Set("Content-Disposition", `attachment; filename="madi-knowledge-`+r.PathValue("id")+`.json"`)
	jsonResponse(w, 200, v)
}
func (s *Server) deleteKnowledgePackage(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Confirmation string `json:"confirmation"`
	}
	if decode(r, &in) != nil || in.Confirmation != "DELETE" || !validID(r.PathValue("id")) {
		apiError(w, 400, "삭제 확인이 필요합니다")
		return
	}
	if !hasIntegrationScope(current(r), "document:read") {
		apiError(w, 403, "문서 조회 권한이 필요합니다")
		return
	}
	result, err := s.DB.Exec(r.Context(), `DELETE FROM knowledge_packages WHERE id=$1 AND owner_id=$2 AND ($3='' OR workspace_id::text=$3) AND NOT EXISTS(SELECT 1 FROM jsonb_to_recordset(source_refs) AS ref(id uuid) JOIN knowledge_document_meta m ON m.document_id=ref.id WHERE m.legal_hold OR m.retain_until>now())`, r.PathValue("id"), current(r).ID, current(r).WorkspaceID)
	if err != nil {
		respond(w, nil, err)
		return
	}
	if result.RowsAffected() != 1 {
		apiError(w, 409, "패키지가 없거나 보존 정책상 삭제할 수 없습니다")
		return
	}
	s.audit(r, "KNOWLEDGE_PACKAGE_DELETE", r.PathValue("id"), nil)
	jsonResponse(w, 200, map[string]any{"deleted": true})
}
