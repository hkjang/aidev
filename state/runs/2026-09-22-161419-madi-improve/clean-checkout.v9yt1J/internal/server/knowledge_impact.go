package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

//go:embed knowledge_impact.sql
var knowledgeImpactSchema string

func impactWorkflowTx(ctx context.Context, tx pgx.Tx) bool {
	var raw []byte
	var cfg map[string]any
	return tx.QueryRow(ctx, `SELECT data FROM settings WHERE id=1 FOR SHARE`).Scan(&raw) == nil && json.Unmarshal(raw, &cfg) == nil && boolean(cfg, "approval_enabled")
}

func (s *Server) migrateKnowledgeImpact(ctx context.Context) error {
	_, err := s.DB.Exec(ctx, knowledgeImpactSchema)
	if err == nil {
		_, err = s.DB.Exec(ctx, impactExceptionSchema)
	}
	return err
}
func (s *Server) registerKnowledgeImpact() {
	s.registerImpactExceptions()
	s.handle("GET /api/v1/documents/{id}/impact", s.documentImpact)
	s.handle("POST /api/v1/knowledge/impact-check", s.checkImpactSnapshot)
	s.handle("GET /api/v1/knowledge/impact-reviews", s.listImpactReviews)
	s.handle("POST /api/v1/documents/{id}/impact-reviews", s.createImpactReview)
	s.handle("PUT /api/v1/knowledge/impact-reviews/{id}", s.updateImpactReview)
	s.handle("GET /api/v1/knowledge/impact-reviews/{id}/history", s.impactReviewHistory)
}

func (s *Server) checkImpactSnapshot(w http.ResponseWriter, r *http.Request) {
	var in struct {
		WorkspaceID string     `json:"workspace_id"`
		Documents   []aiSource `json:"documents"`
		Packages    []string   `json:"packages"`
		Agents      []struct {
			ID       string `json:"id"`
			Revision int    `json:"revision"`
		} `json:"agents"`
	}
	if decode(r, &in) != nil || len(in.Documents) < 1 || len(in.Documents) > 501 || len(in.Packages) > 100 || len(in.Agents) > 100 {
		apiError(w, 400, "분석 확인 범위를 확인하세요")
		return
	}
	for _, d := range in.Documents {
		if !validID(d.ID) || d.Version < 1 {
			apiError(w, 400, "문서 버전을 확인하세요")
			return
		}
	}
	for _, id := range in.Packages {
		if !validID(id) {
			apiError(w, 400, "패키지 형식을 확인하세요")
			return
		}
	}
	for _, a := range in.Agents {
		if !validID(a.ID) || a.Revision < 1 {
			apiError(w, 400, "Agent 버전을 확인하세요")
			return
		}
	}
	p, err := s.packagePrincipal(r, in.WorkspaceID)
	if err != nil {
		apiError(w, 403, "현재 조회 권한이 필요합니다")
		return
	}
	var docs, packages, agents int
	err = s.DB.QueryRow(r.Context(), `SELECT count(*) FROM jsonb_to_recordset($1::jsonb) ref(id uuid,version integer) JOIN documents d ON d.id=ref.id WHERE d.workspace_id=$2 AND d.version=ref.version AND d.deleted_at IS NULL AND madi_document_allowed($3,d.id,false)`, jsonValue(in.Documents), in.WorkspaceID, p.ID).Scan(&docs)
	if err == nil && len(in.Packages) > 0 {
		err = s.DB.QueryRow(r.Context(), `SELECT count(*) FROM knowledge_packages WHERE id=ANY($1::uuid[]) AND workspace_id=$2 AND madi_package_allowed($3,id)`, in.Packages, in.WorkspaceID, p.ID).Scan(&packages)
	}
	if err == nil && len(in.Agents) > 0 && hasIntegrationScope(p, "ai:execute") && featureAllowed(r.Context(), s.DB, p, in.WorkspaceID, "workspace-agents") {
		err = s.DB.QueryRow(r.Context(), `SELECT count(*) FROM jsonb_to_recordset($1::jsonb) ref(id uuid,revision integer) JOIN workspace_agents a ON a.id=ref.id AND a.revision=ref.revision WHERE a.workspace_id=$2 AND a.enabled`, jsonValue(in.Agents), in.WorkspaceID).Scan(&agents)
	}
	if err != nil {
		respond(w, nil, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"valid": docs == len(in.Documents) && packages == len(in.Packages) && agents == len(in.Agents)})
}

var impactNumber = regexp.MustCompile(`[0-9]+(?:\.[0-9]+)?`)

func classifyKnowledgeChange(diff documentDiff, access, metadata bool) []string {
	out := []string{}
	if access {
		out = append(out, "access")
	}
	if metadata {
		out = append(out, "metadata")
	}
	oldNumbers, newNumbers := []string{}, []string{}
	procedure := false
	for _, row := range diff.Rows {
		if row.Kind != "add" && row.Kind != "remove" {
			continue
		}
		if row.Kind == "add" {
			newNumbers = append(newNumbers, impactNumber.FindAllString(row.Text, -1)...)
		} else {
			oldNumbers = append(oldNumbers, impactNumber.FindAllString(row.Text, -1)...)
		}
		lower := strings.ToLower(row.Text)
		for _, token := range []string{"절차", "단계", "실행", "승인", "금지", "허용", "권한", "must ", "shall ", "step ", "sudo ", "kubectl "} {
			if strings.Contains(lower, token) {
				procedure = true
			}
		}
	}
	if strings.Join(oldNumbers, "\n") != strings.Join(newNumbers, "\n") {
		out = append(out, "number")
	}
	if procedure {
		out = append(out, "procedure")
	}
	if diff.Truncated {
		out = append(out, "incomplete")
	}
	if len(out) == 0 && (diff.Added > 0 || diff.Removed > 0) {
		out = append(out, "wording_candidate")
	}
	return out
}

// Traversal follows dependent -> dependency backwards. Each frontier edge is
// ACL checked at BOTH ends; inaccessible intermediate nodes cannot reveal paths.
func (s *Server) impactDependents(r *http.Request, wid, source string, depth int) ([]map[string]any, bool, error) {
	found := map[string]bool{source: true}
	frontier := []string{source}
	out := []map[string]any{}
	limited := false
	for level := 1; level <= depth && len(frontier) > 0; level++ {
		rows, err := s.rows(r.Context(), `SELECT jsonb_build_object('id',d.id,'title',d.title,'version',d.version,'owner_id',d.owner_id,'kind',coalesce(m.kind,'page'),'via',x.target_id,'relation_type',x.type,'can_write',madi_document_allowed($3,d.id,true))
   FROM document_relations x JOIN documents d ON d.id=x.source_id JOIN documents dependency ON dependency.id=x.target_id LEFT JOIN knowledge_document_meta m ON m.document_id=d.id
   WHERE x.target_id=ANY($1::uuid[]) AND d.workspace_id=$2 AND dependency.workspace_id=$2 AND d.deleted_at IS NULL AND dependency.deleted_at IS NULL AND madi_document_allowed($3,d.id,false) AND madi_document_allowed($3,dependency.id,false)
   ORDER BY CASE x.type WHEN 'policy' THEN 0 WHEN 'execution' THEN 1 WHEN 'data' THEN 2 ELSE 3 END,d.id,x.target_id,x.type LIMIT 2001`, frontier, wid, current(r).ID)
		if err != nil {
			return nil, false, err
		}
		if len(rows) > 2000 {
			rows = rows[:2000]
			limited = true
		}
		next := []string{}
		for _, row := range rows {
			id := str(row, "id")
			if found[id] {
				continue
			}
			if len(out) >= 500 {
				limited = true
				break
			}
			found[id] = true
			row["depth"] = level
			row["can_write"] = boolean(row, "can_write") && hasIntegrationScope(current(r), "document:write")
			out = append(out, row)
			next = append(next, id)
		}
		if len(out) >= 500 {
			break
		}
		frontier = next
	}
	return out, limited, nil
}

func (s *Server) documentImpact(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) || !s.canDocument(r.Context(), current(r), id, false) {
		apiError(w, 404, "문서에 접근할 수 없습니다")
		return
	}
	from, err := strconv.Atoi(r.URL.Query().Get("from"))
	if err != nil {
		apiError(w, 400, "이전 버전을 확인하세요")
		return
	}
	depth := 1
	if raw := r.URL.Query().Get("depth"); raw != "" {
		depth, err = strconv.Atoi(raw)
	}
	if err != nil || from < 1 || from > 2147483647 || depth < 1 || depth > 5 {
		apiError(w, 400, "이전 버전과 탐색 깊이(1~5)를 확인하세요")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	doc, err := s.document(r, id)
	if err != nil {
		apiError(w, 404, "문서에 접근할 수 없습니다")
		return
	}
	wid := str(doc, "workspace_id")
	if _, err = s.packagePrincipal(r, wid); err != nil {
		apiError(w, 403, "현재 문서 조회 권한이 필요합니다")
		return
	}
	version := number(doc, "version", 0)
	if from > version {
		apiError(w, 400, "현재 버전 이전의 비교 기준을 선택하세요")
		return
	}
	before, err := s.versionSnapshot(r, id, from)
	if err != nil {
		apiError(w, 404, "비교할 이전 원문을 찾을 수 없습니다")
		return
	}
	diff := boundedDocumentDiff(str(before, "markdown"), str(doc, "markdown"))
	var access, metadata bool
	err = s.DB.QueryRow(ctx, `SELECT coalesce(bool_or(access_changed),false),coalesce(bool_or(metadata_changed),false) FROM knowledge_change_events WHERE document_id=$1 AND before_version>=$2 AND after_version<=$3`, id, from, version).Scan(&access, &metadata)
	if err != nil {
		respond(w, nil, err)
		return
	}
	dependent, limited, err := s.impactDependents(r, wid, id, depth)
	if err != nil {
		respond(w, nil, err)
		return
	}
	// Only the caller's own currently readable packages are discoverable.
	packages, err := s.rows(ctx, `SELECT jsonb_build_object('id',p.id,'created_at',p.created_at,'source_version',ref.version,'stale',ref.version<>$4) FROM knowledge_packages p CROSS JOIN LATERAL jsonb_to_recordset(p.source_refs) ref(id uuid,version integer) WHERE p.workspace_id=$1 AND p.owner_id=$2 AND ref.id=$3 AND madi_package_allowed($2,p.id) ORDER BY p.created_at DESC LIMIT 101`, wid, current(r).ID, id, version)
	if err != nil {
		respond(w, nil, err)
		return
	}
	if len(packages) > 100 {
		packages = packages[:100]
		limited = true
	}
	agents := []map[string]any{}
	if hasIntegrationScope(current(r), "ai:execute") && featureAllowed(ctx, s.DB, current(r), wid, "workspace-agents") {
		agents, err = s.rows(ctx, `SELECT jsonb_build_object('id',a.id,'name',a.name,'revision',a.revision,'reason',CASE WHEN $3::uuid=ANY(a.document_ids) THEN 'selected_document' ELSE 'selected_space' END) FROM workspace_agents a WHERE a.workspace_id=$1 AND a.enabled AND EXISTS(SELECT 1 FROM documents d WHERE d.id=$3 AND d.deleted_at IS NULL AND madi_document_allowed($2,d.id,false) AND (d.id=ANY(a.document_ids) OR d.space_id IN(WITH RECURSIVE scope AS(SELECT id FROM spaces WHERE id=ANY(a.space_ids) AND workspace_id=$1 UNION SELECT x.id FROM spaces x JOIN scope p ON x.parent_id=p.id WHERE x.workspace_id=$1) SELECT id FROM scope))) ORDER BY a.name,a.id LIMIT 101`, wid, current(r).ID, id)
		if err != nil {
			respond(w, nil, err)
			return
		}
		if len(agents) > 100 {
			agents = agents[:100]
			limited = true
		}
	}
	refs := []aiSource{{ID: id, Version: version}}
	for _, d := range dependent {
		refs = append(refs, aiSource{ID: str(d, "id"), Version: number(d, "version", 0)})
	}
	var count int
	err = s.DB.QueryRow(ctx, `SELECT count(*) FROM jsonb_to_recordset($1::jsonb) ref(id uuid,version integer) JOIN documents d ON d.id=ref.id WHERE d.version=ref.version AND d.workspace_id=$2 AND d.deleted_at IS NULL AND madi_document_allowed($3,d.id,false)`, jsonValue(refs), wid, current(r).ID).Scan(&count)
	if _, e := s.packagePrincipal(r, wid); e != nil || err != nil || count != len(refs) {
		apiError(w, 409, "분석 중 원문 또는 접근 범위가 바뀌었습니다. 다시 분석하세요")
		return
	}
	jsonResponse(w, 200, map[string]any{"source": map[string]any{"id": id, "title": doc["title"], "version": version, "can_write": s.canDocument(ctx, current(r), id, true) && hasIntegrationScope(current(r), "document:write")}, "from": from, "depth": depth, "diff": diff, "categories": classifyKnowledgeChange(diff, access, metadata), "documents": dependent, "packages": packages, "agents": agents, "limited": limited, "notice": "현재 접근 가능한 직접 연결의 역방향을 최대 5단계·500문서에서 검사합니다. 표시는 자동 확정이 아닌 규칙 기반 영향 후보이며, 첫 발견 경로를 대표로 표시합니다. 미등록 의존·미해결 위키 링크·외부에 전달된 사본은 완전하게 추적할 수 없습니다. 정책·권한 변경 이력은 이 기능 도입 이후 문서 행 변경부터 기록됩니다. 연결 문서나 Agent를 자동 수정·실행하지 않습니다."})
}

const impactReviewsSQL = `SELECT jsonb_build_object('id',r.id,'source_id',r.source_id,'source_title',s.title,'source_version',r.source_version,'target_id',r.target_id,'target_title',d.title,'target_version',r.target_version,'current_target_version',d.version,'relation_type',r.relation_type,'owner_id',r.owner_id,'owner_name',u.name,'status',r.status,'revision',r.revision,'updated_at',r.updated_at,'stale',s.version<>r.source_version,'can_write',madi_document_allowed($2,d.id,true)) FROM knowledge_impact_reviews r JOIN documents s ON s.id=r.source_id JOIN documents d ON d.id=r.target_id JOIN users u ON u.id=r.owner_id WHERE s.workspace_id=$1 AND madi_impact_allowed($2,r.id)`

func (s *Server) listImpactReviews(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if _, err := s.packagePrincipal(r, wid); err != nil {
		apiError(w, 403, "워크스페이스 조회 권한이 필요합니다")
		return
	}
	cfg, err := s.settings(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	if !boolean(cfg, "approval_enabled") {
		jsonResponse(w, 200, []any{})
		return
	}
	rows, err := s.rows(r.Context(), impactReviewsSQL+` ORDER BY r.updated_at DESC,r.id LIMIT 200`, wid, current(r).ID)
	for _, row := range rows {
		row["can_write"] = boolean(row, "can_write") && hasIntegrationScope(current(r), "document:write") && (str(row, "owner_id") == current(r).ID || s.workspaceAdmin(r, wid))
	}
	respond(w, rows, err)
}

func (s *Server) createImpactReview(w http.ResponseWriter, r *http.Request) {
	var in struct {
		TargetID      string `json:"target_id"`
		SourceVersion int    `json:"source_version"`
		TargetVersion int    `json:"target_version"`
		Type          string `json:"relation_type"`
	}
	source := r.PathValue("id")
	if decode(r, &in) != nil || !validID(source) || !validID(in.TargetID) || source == in.TargetID || in.SourceVersion < 1 || in.TargetVersion < 1 || !oneOf(in.Type, "related", "reference", "policy", "execution", "data") {
		apiError(w, 400, "현재 원문·대상 버전과 의존 관계를 확인하세요")
		return
	}
	if !hasIntegrationScope(current(r), "document:write") || !s.canDocument(r.Context(), current(r), source, true) {
		apiError(w, 403, "변경 원문 작성 권한이 필요합니다")
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	ids := []string{source, in.TargetID}
	sort.Strings(ids)
	rows, err := tx.Query(r.Context(), `SELECT id::text FROM documents WHERE id=ANY($1::uuid[]) ORDER BY id FOR SHARE`, ids)
	if err != nil {
		respond(w, nil, err)
		return
	}
	for rows.Next() {
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		respond(w, nil, err)
		return
	}
	var wid, owner string
	err = tx.QueryRow(r.Context(), `SELECT s.workspace_id::text,d.owner_id::text FROM documents s,documents d WHERE s.id=$1 AND d.id=$2 AND s.workspace_id=d.workspace_id AND s.version=$3 AND d.version=$4 AND s.deleted_at IS NULL AND d.deleted_at IS NULL AND madi_document_allowed($5,s.id,true) AND madi_document_allowed($5,d.id,false) AND madi_document_allowed(d.owner_id,s.id,false) AND madi_document_allowed(d.owner_id,d.id,true) AND EXISTS(SELECT 1 FROM document_relations x WHERE x.source_id=d.id AND x.target_id=s.id AND x.type=$6)`, source, in.TargetID, in.SourceVersion, in.TargetVersion, current(r).ID, in.Type).Scan(&wid, &owner)
	if err != nil {
		apiError(w, 409, "직접 의존 관계·원문·대상·소유자의 현재 검토 권한을 다시 확인하세요")
		return
	}
	if !impactWorkflowTx(r.Context(), tx) {
		apiError(w, 403, "관리자가 검토·승인 기능을 활성화한 경우에만 영향 검토를 요청할 수 있습니다")
		return
	}
	if err = s.knowledgeActorTx(r, tx, wid, "document:read", "document:write"); err != nil {
		apiError(w, 403, err.Error())
		return
	}
	var id string
	err = tx.QueryRow(r.Context(), `INSERT INTO knowledge_impact_reviews(source_id,source_version,target_id,target_version,relation_type,owner_id,created_by) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT DO NOTHING RETURNING id::text`, source, in.SourceVersion, in.TargetID, in.TargetVersion, in.Type, owner, current(r).ID).Scan(&id)
	if err != nil {
		apiError(w, 409, "같은 변경·대상에 대한 검토가 이미 있거나 관계가 변경되었습니다")
		return
	}
	_, err = tx.Exec(r.Context(), `INSERT INTO notifications(id,user_id,title,document_id) VALUES($1,$2,'연결 문서 변경의 영향 검토 요청',$3)`, newID(), owner, in.TargetID)
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		respond(w, nil, err)
		return
	}
	s.audit(r, "KNOWLEDGE_IMPACT_REQUEST", source, map[string]any{"review_id": id, "target_id": in.TargetID, "source_version": in.SourceVersion})
	jsonResponse(w, 201, map[string]any{"id": id, "status": "pending"})
}

func (s *Server) updateImpactReview(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Revision      int    `json:"revision"`
		TargetVersion int    `json:"target_version"`
		Status        string `json:"status"`
		Note          string `json:"note"`
	}
	id := r.PathValue("id")
	if decode(r, &in) != nil || !validID(id) || in.Revision < 1 || in.TargetVersion < 1 || !oneOf(in.Status, "pending", "no_impact", "needs_change", "done", "exception_requested") || len(strings.TrimSpace(in.Note)) == 0 || len(in.Note) > 4000 {
		apiError(w, 400, "검토 버전·현재 대상 버전·처리 상태와 근거 의견(1~4000바이트)을 확인하세요")
		return
	}
	if !hasIntegrationScope(current(r), "document:write") {
		apiError(w, 403, "문서 작성 권한이 필요합니다")
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	var source, target, owner, wid string
	var sourceVersion, revision int
	err = tx.QueryRow(r.Context(), `SELECT r.source_id::text,r.target_id::text,r.owner_id::text,s.workspace_id::text,r.source_version,r.revision FROM knowledge_impact_reviews r JOIN documents s ON s.id=r.source_id WHERE r.id=$1 AND madi_impact_allowed($2,r.id)`, id, current(r).ID).Scan(&source, &target, &owner, &wid, &sourceVersion, &revision)
	if err != nil {
		apiError(w, 404, "접근 가능한 검토를 찾을 수 없습니다")
		return
	}
	var manager bool
	if err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND role IN ('owner','admin'))`, wid, current(r).ID).Scan(&manager); err != nil {
		respond(w, nil, err)
		return
	}
	if owner != current(r).ID && !manager {
		apiError(w, 403, "담당자 또는 워크스페이스 관리자가 검토할 수 있습니다")
		return
	}
	if revision != in.Revision {
		apiError(w, 409, "다른 사용자가 검토 상태를 변경했습니다")
		return
	}
	var count int
	ids := []string{source, target}
	sort.Strings(ids)
	rows, err := tx.Query(r.Context(), `SELECT id FROM documents WHERE id=ANY($1::uuid[]) ORDER BY id FOR SHARE`, ids)
	if err != nil {
		respond(w, nil, err)
		return
	}
	for rows.Next() {
		count++
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		respond(w, nil, err)
		return
	}
	var valid bool
	err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM documents s,documents d WHERE s.id=$1 AND d.id=$2 AND s.version=$3 AND d.version=$4 AND s.deleted_at IS NULL AND d.deleted_at IS NULL AND madi_document_allowed($5,s.id,false) AND madi_document_allowed($5,d.id,true))`, source, target, sourceVersion, in.TargetVersion, current(r).ID).Scan(&valid)
	if err != nil || !valid || count != 2 {
		apiError(w, 409, "원문·대상 또는 현재 권한이 변경되었습니다. 새 영향 분석이 필요합니다")
		return
	}
	// Resource locks precede the dependent review lock, matching document
	// deletion/cascades. Recheck the CAS after taking that lock, not before.
	if err = tx.QueryRow(r.Context(), `SELECT revision FROM knowledge_impact_reviews WHERE id=$1 AND source_id=$2 AND target_id=$3 AND madi_impact_allowed($4,id) FOR UPDATE`, id, source, target, current(r).ID).Scan(&revision); err != nil || revision != in.Revision {
		apiError(w, 409, "다른 사용자가 검토 상태를 변경했습니다")
		return
	}
	p := current(r)
	if !hasIntegrationScope(p, "document:write") {
		apiError(w, 403, "현재 작성 권한이 변경되었습니다")
		return
	}
	if !impactWorkflowTx(r.Context(), tx) {
		apiError(w, 403, "현재 검토·승인 기능이 비활성화되어 있습니다")
		return
	}
	if err = s.checkEvidenceProtection(r.Context(), tx, p, wid, in.Note); err != nil {
		apiError(w, 422, err.Error())
		return
	}
	if err = s.knowledgeActorTx(r, tx, wid, "document:read", "document:write"); err != nil {
		apiError(w, 403, err.Error())
		return
	}
	sealed, err := s.encrypt(in.Note)
	if err != nil {
		respond(w, nil, err)
		return
	}
	_, err = tx.Exec(r.Context(), `UPDATE knowledge_impact_reviews SET status=$2,target_version=$3,revision=revision+1,updated_at=now() WHERE id=$1`, id, in.Status, in.TargetVersion)
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO knowledge_impact_review_events(review_id,actor_id,revision,status,note_ciphertext) VALUES($1,$2,$3,$4,$5)`, id, p.ID, revision+1, in.Status, sealed)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		respond(w, nil, err)
		return
	}
	s.audit(r, "KNOWLEDGE_IMPACT_REVIEW", source, map[string]any{"review_id": id, "revision": revision + 1, "status": in.Status})
	jsonResponse(w, 200, map[string]any{"id": id, "revision": revision + 1, "status": in.Status, "notice": "이 기록은 영향 검토이며 게시 승인이나 실행 권한이 아닙니다. 예외 요청은 예외 승인 완료를 의미하지 않습니다."})
}

func (s *Server) impactReviewHistory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		apiError(w, 404, "검토를 찾을 수 없습니다")
		return
	}
	var wid string
	if s.DB.QueryRow(r.Context(), `SELECT s.workspace_id::text FROM knowledge_impact_reviews r JOIN documents s ON s.id=r.source_id WHERE r.id=$1 AND madi_impact_allowed($2,r.id)`, id, current(r).ID).Scan(&wid) != nil {
		apiError(w, 404, "검토를 찾을 수 없습니다")
		return
	}
	p, err := s.packagePrincipal(r, wid)
	if err != nil {
		apiError(w, 403, "현재 조회 권한이 필요합니다")
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	rows, err := tx.Query(r.Context(), `SELECT e.revision,e.status,e.note_ciphertext,e.created_at FROM knowledge_impact_review_events e WHERE e.review_id=$1 AND madi_impact_allowed($2,e.review_id) ORDER BY e.revision DESC LIMIT 200`, id, p.ID)
	if err != nil {
		respond(w, nil, err)
		return
	}
	out := []map[string]any{}
	for rows.Next() {
		var revision int
		var status, note string
		var created time.Time
		if err = rows.Scan(&revision, &status, &note, &created); err != nil {
			break
		}
		plain, e := s.decrypt(note)
		if e != nil {
			err = e
			break
		}
		out = append(out, map[string]any{"revision": revision, "status": status, "note": plain, "created_at": created})
	}
	rows.Close()
	if err == nil {
		err = rows.Err()
	}
	if err == nil {
		err = s.checkEvidenceProtection(r.Context(), tx, p, wid, out)
	}
	if err != nil {
		apiError(w, 409, "현재 보호 정책으로 검토 이력을 표시할 수 없습니다")
		return
	}
	var allowed bool
	if tx.QueryRow(r.Context(), `SELECT madi_impact_allowed($1,$2)`, p.ID, id).Scan(&allowed) != nil || !allowed {
		apiError(w, 404, "검토에 접근할 수 없습니다")
		return
	}
	jsonResponse(w, 200, out)
}
