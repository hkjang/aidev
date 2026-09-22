package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
)

type documentQueryDocument struct {
	ID, WorkspaceID, Markdown, Hash string
	Version                         int
	Values                          map[string]any
	BodyBytes                       int
	Details                         []any
}
type documentQueryDiagnostics struct {
	DocumentsScanned   int            `json:"documents_scanned"`
	Candidates         int            `json:"candidates"`
	ScannedBytes       int            `json:"scanned_bytes"`
	OversizedDocuments int            `json:"oversized_documents"`
	InvalidProperties  int            `json:"invalid_properties"`
	TruncatedFields    int            `json:"truncated_fields"`
	Truncated          bool           `json:"truncated"`
	Limits             map[string]int `json:"limits"`
	Notice             string         `json:"notice"`
}
type documentQueryReference struct{ ID, Hash string }
type documentQueryRelationRef struct{ SourceID, TargetID, Type, Created string }
type documentQueryMaterial struct {
	Rows        []documentQueryRow
	Documents   map[string]documentQueryDocument
	Relations   map[string]documentQueryRelationRef
	Diagnostics documentQueryDiagnostics
}

func documentQueryDocJSON(body, tasks bool) string {
	md := "''"
	details := "'[]'::jsonb"
	if body {
		md = fmt.Sprintf("CASE WHEN octet_length(d.markdown)<=%d THEN d.markdown ELSE '' END", documentQueryMaxBodyBytes)
	}
	if tasks {
		details = `coalesce((SELECT jsonb_agg(v) FROM (SELECT jsonb_build_object('task_id',t.task_id,'status',t.status,'priority',t.priority,'due_date',t.due_date,'assignee_id',CASE WHEN NOT u.disabled AND madi_document_allowed(u.id,d.id,false) THEN u.id END,'assignee_name',CASE WHEN NOT u.disabled AND madi_document_allowed(u.id,d.id,false) THEN left(u.name,200) END) v FROM task_details t LEFT JOIN users u ON u.id=t.assignee_id WHERE t.document_id=d.id ORDER BY t.task_id LIMIT 2001) bounded),'[]'::jsonb)`
	}
	return `jsonb_build_object('id',d.id,'workspace_id',d.workspace_id,'title',left(d.title,500),'title_truncated',length(d.title)>500,'owner_truncated',coalesce((SELECT length(u.name)>200 FROM users u WHERE u.id=d.owner_id),false),'status',left(d.status,32),'tags',` + documentSummaryTags + `,'tags_truncated',CASE WHEN jsonb_typeof(d.tags)='array' THEN jsonb_array_length(d.tags)>jsonb_array_length(` + documentSummaryTags + `) ELSE true END,'owner_id',d.owner_id,'owner_name',coalesce((SELECT left(u.name,200) FROM users u WHERE u.id=d.owner_id),''),'created_at',d.created_at,'updated_at',d.updated_at,'version',d.version,'classification',coalesce(k.classification,'internal'),'kind',coalesce(k.kind,'page'),'body_bytes',octet_length(d.markdown),'markdown',` + md + `,'details',` + details + `)`
}
func documentQueryDecodeDoc(raw []byte) (documentQueryDocument, error) {
	var out documentQueryDocument
	if e := json.Unmarshal(raw, &out.Values); e != nil {
		return out, e
	}
	out.ID = str(out.Values, "id")
	out.WorkspaceID = str(out.Values, "workspace_id")
	out.Version = number(out.Values, "version", 0)
	out.Markdown = str(out.Values, "markdown")
	out.BodyBytes = number(out.Values, "body_bytes", 0)
	out.Hash = digest(string(raw))
	out.Details, _ = out.Values["details"].([]any)
	return out, nil
}
func documentQueryParentTx(r *http.Request, tx pgx.Tx) (documentQueryDocument, error) {
	var d documentQueryDocument
	if !validID(r.PathValue("id")) || !hasIntegrationScope(current(r), "document:read") {
		return d, approvalProblem(404, "접근 가능한 조회 문서가 없습니다")
	}
	var size int
	e := tx.QueryRow(r.Context(), `SELECT id::text,workspace_id::text,version,octet_length(markdown),CASE WHEN octet_length(markdown)<=1048576 THEN markdown ELSE '' END FROM documents WHERE id=$1 AND deleted_at IS NULL AND madi_document_allowed($2,id,false) AND ($3='' OR workspace_id::text=$3)`, r.PathValue("id"), current(r).ID, current(r).WorkspaceID).Scan(&d.ID, &d.WorkspaceID, &d.Version, &size, &d.Markdown)
	if e != nil {
		return d, approvalProblem(404, "접근 가능한 조회 문서가 없습니다")
	}
	if size > 1<<20 {
		return d, approvalProblem(413, "조회 정의를 해석할 문서는 1 MiB 이하여야 합니다. 별도 조회 문서로 분리하세요")
	}
	d.Hash = digest(d.Markdown)
	if !featureAllowed(r.Context(), tx, current(r), d.WorkspaceID, "document-queries") {
		return d, approvalProblem(403, "문서 선언형 조회 기능이 현재 정책에서 비활성화되어 있습니다")
	}
	return d, nil
}
func documentQueryNeedsBody(q documentQueryDefinition) bool {
	return q.Source == "tasks" || len(q.Properties) > 0
}
func documentQueryCandidateSQL(body, tasks bool) string {
	// The recursive policy repeats workspace/actor and ancestry lookups for
	// every candidate. As in universalSearchSQL, a document without a parent
	// or space has no inherited ACL: evaluate its identical read policy using
	// one current workspace-membership lookup. Never apply this shortcut to
	// inherited access, or move the visible-candidate limit before the ACL.
	// Final source hashes/ACL and credential checks still use the full policy.
	return `WITH active_workspace AS MATERIALIZED (
 SELECT m.workspace_id FROM users u JOIN workspace_members m ON m.user_id=u.id
 WHERE u.id=$1 AND NOT u.disabled AND m.workspace_id=$2
)
SELECT ` + documentQueryDocJSON(body, tasks) + ` FROM documents d JOIN active_workspace membership ON membership.workspace_id=d.workspace_id LEFT JOIN knowledge_document_meta k ON k.document_id=d.id WHERE d.workspace_id=$2 AND d.deleted_at IS NULL AND
 CASE WHEN d.parent_id IS NULL AND d.space_id IS NULL THEN
  (d.visibility='workspace' OR d.owner_id=$1 OR (d.visibility='selected' AND EXISTS(SELECT 1 FROM document_shares sh WHERE sh.document_id=d.id AND sh.user_id=$1)))
 ELSE madi_document_allowed($1,d.id,false) END
 AND ($3::uuid IS NULL OR d.space_id=$3) AND (cardinality($4::uuid[])=0 OR d.id=ANY($4::uuid[])) ORDER BY d.updated_at DESC,d.id LIMIT 2001`
}
func (s *Server) documentQueryMaterialTx(r *http.Request, tx pgx.Tx, parent documentQueryDocument, q documentQueryDefinition, params map[string]any) (documentQueryMaterial, error) {
	out := documentQueryMaterial{Rows: []documentQueryRow{}, Documents: map[string]documentQueryDocument{}, Relations: map[string]documentQueryRelationRef{}, Diagnostics: documentQueryDiagnostics{Limits: map[string]int{"candidates": documentQueryMaxCandidates, "scan_bytes": documentQueryScanBytes, "document_body_bytes": documentQueryMaxBodyBytes, "rows": q.Limit, "timeout_ms": 2000}, Notice: "실행 시점의 현재 접근 가능한 후보 안에서 계산합니다. 최근 수정 문서 우선 최대 2,000개·8 MiB만 검사하므로 제한 표시는 전체 검색 결과가 아닙니다. 문자열 포함은 대소문자를 구분하며, 태그는 최대 32개 중 정확히 같은 값을 비교합니다. 동적 결과는 원문·내보내기·승인 근거에 포함되지 않습니다."}}
	if q.Source == "relations" {
		return s.documentQueryRelationsTx(r, tx, parent, q, params, out)
	}
	var space any
	if q.SpaceID != "" {
		space = q.SpaceID
	}
	ids := q.DocumentIDs
	if ids == nil {
		ids = []string{}
	}
	rows, e := tx.Query(r.Context(), documentQueryCandidateSQL(documentQueryNeedsBody(q), q.Source == "tasks"), current(r).ID, parent.WorkspaceID, space, ids)
	if e != nil {
		return out, e
	}
	defer rows.Close()
	for rows.Next() {
		if r.Context().Err() != nil {
			return out, r.Context().Err()
		}
		if out.Diagnostics.Candidates >= documentQueryMaxCandidates || out.Diagnostics.DocumentsScanned >= documentQueryMaxCandidates {
			out.Diagnostics.Truncated = true
			break
		}
		var raw []byte
		if e = rows.Scan(&raw); e != nil {
			return out, e
		}
		if out.Diagnostics.ScannedBytes+len(raw) > documentQueryScanBytes {
			out.Diagnostics.Truncated = true
			break
		}
		out.Diagnostics.ScannedBytes += len(raw)
		out.Diagnostics.DocumentsScanned++
		d, e := documentQueryDecodeDoc(raw)
		if e != nil {
			return out, e
		}
		if boolean(d.Values, "tags_truncated") {
			out.Diagnostics.TruncatedFields++
		}
		for _, field := range []string{"title_truncated", "owner_truncated"} {
			if boolean(d.Values, field) {
				out.Diagnostics.TruncatedFields++
			}
		}
		if documentQueryNeedsBody(q) && d.BodyBytes > documentQueryMaxBodyBytes {
			out.Diagnostics.OversizedDocuments++
			out.Diagnostics.Candidates++
			out.Diagnostics.Truncated = true
			continue
		}
		out.Documents[d.ID] = d
		if q.Source == "documents" {
			out.Diagnostics.Candidates++
			if len(q.Properties) > 0 {
				values, bad := documentQueryProperties(d.Markdown, q.Properties)
				for key, v := range values {
					d.Values[key] = v
				}
				if bad {
					out.Diagnostics.InvalidProperties++
				}
			}
			if documentQueryMatches(q, d.Values, params) {
				out.Rows = append(out.Rows, documentQueryRow{Values: d.Values, DocumentID: d.ID, Version: d.Version, Sources: []string{d.ID}})
			}
		} else {
			details := map[string]map[string]any{}
			for _, v := range d.Details {
				m, _ := v.(map[string]any)
				details[str(m, "task_id")] = m
			}
			if len(d.Details) > documentQueryMaxCandidates {
				out.Diagnostics.OversizedDocuments++
				out.Diagnostics.Truncated = true
				continue
			}
			for _, task := range indexMarkdown(d.Markdown).Tasks {
				if r.Context().Err() != nil {
					return out, r.Context().Err()
				}
				if out.Diagnostics.Candidates >= documentQueryMaxCandidates {
					out.Diagnostics.Truncated = true
					break
				}
				out.Diagnostics.Candidates++
				label := task.Text
				if len([]rune(label)) > 500 {
					label = string([]rune(label)[:500])
					out.Diagnostics.TruncatedFields++
				}
				v := map[string]any{"document_id": d.ID, "document_title": d.Values["title"], "text": label, "done": task.Done, "status": "todo", "priority": "normal", "due_date": nil, "assignee_id": nil, "assignee_name": nil, "version": d.Version, "line": task.Line}
				if m := details[task.ID]; m != nil && !task.Ambiguous {
					for _, key := range []string{"status", "priority", "due_date", "assignee_id", "assignee_name"} {
						v[key] = m[key]
					}
				}
				if task.Done {
					v["status"] = "done"
				} else if v["status"] == "done" {
					v["status"] = "todo"
				}
				if documentQueryMatches(q, v, params) {
					out.Rows = append(out.Rows, documentQueryRow{Values: v, DocumentID: d.ID, Version: d.Version, Line: task.Line, Sources: []string{d.ID}})
				}
			}
		}
	}
	if e = rows.Err(); e != nil {
		return out, e
	}
	rows.Close()
	documentQuerySort(q, out.Rows)
	if len(out.Rows) > q.Limit {
		out.Rows = out.Rows[:q.Limit]
		out.Diagnostics.Truncated = true
	}
	return out, nil
}

// Breadth-first traversal only expands currently readable intermediate nodes.
// There is no unrestricted recursive relation path followed by a final filter.
func (s *Server) documentQueryRelationsTx(r *http.Request, tx pgx.Tx, parent documentQueryDocument, q documentQueryDefinition, params map[string]any, out documentQueryMaterial) (documentQueryMaterial, error) {
	front := map[string][]string{}
	seeds := q.DocumentIDs
	if len(seeds) == 0 {
		seeds = []string{parent.ID}
	}
	for _, id := range seeds {
		front[id] = []string{id}
	}
	visited := map[string]bool{}
	edges := map[string]bool{}
	for depth := 1; depth <= q.Depth && len(front) > 0; depth++ {
		ids := []string{}
		for id := range front {
			ids = append(ids, id)
		}
		near, far := "source_id", "target_id"
		if q.Direction == "incoming" {
			near, far = far, near
		}
		var space any
		if q.SpaceID != "" {
			space = q.SpaceID
		}
		query := `SELECT r.source_id::text,r.target_id::text,r.type,r.created_at::text,` + documentQueryDocJSON(false, false) + ` FROM document_relations r JOIN documents d ON d.id=r.` + far + ` LEFT JOIN knowledge_document_meta k ON k.document_id=d.id JOIN documents origin ON origin.id=r.` + near + ` WHERE r.` + near + `=ANY($3::uuid[]) AND origin.workspace_id=$2 AND d.workspace_id=$2 AND origin.deleted_at IS NULL AND d.deleted_at IS NULL AND madi_document_allowed($1,origin.id,false) AND madi_document_allowed($1,d.id,false) AND ($4::uuid IS NULL OR d.space_id=$4) ORDER BY r.source_id,r.target_id,r.type LIMIT 2001`
		rows, e := tx.Query(r.Context(), query, current(r).ID, parent.WorkspaceID, ids, space)
		if e != nil {
			return out, e
		}
		next := map[string][]string{}
		for rows.Next() {
			var rel documentQueryRelationRef
			var raw []byte
			if e = rows.Scan(&rel.SourceID, &rel.TargetID, &rel.Type, &rel.Created, &raw); e != nil {
				rows.Close()
				return out, e
			}
			key := rel.SourceID + "|" + rel.TargetID + "|" + rel.Type
			if edges[key] {
				continue
			}
			edges[key] = true
			if out.Diagnostics.Candidates >= documentQueryMaxCandidates || out.Diagnostics.ScannedBytes+len(raw) > documentQueryScanBytes {
				out.Diagnostics.Truncated = true
				break
			}
			out.Diagnostics.Candidates++
			out.Diagnostics.ScannedBytes += len(raw)
			d, e := documentQueryDecodeDoc(raw)
			if e != nil {
				rows.Close()
				return out, e
			}
			out.Documents[d.ID] = d
			from := rel.SourceID
			if q.Direction == "incoming" {
				from = rel.TargetID
			}
			path := append(append([]string{}, front[from]...), d.ID)
			if !visited[d.ID] {
				visited[d.ID] = true
				next[d.ID] = path
			}
			out.Relations[key] = rel
			v := map[string]any{"source_id": rel.SourceID, "target_id": rel.TargetID, "relation_type": rel.Type, "depth": depth}
			out.Rows = append(out.Rows, documentQueryRow{Values: v, DocumentID: d.ID, Version: d.Version, Sources: path})
		}
		if e = rows.Err(); e != nil {
			rows.Close()
			return out, e
		}
		rows.Close()
		front = next
		if out.Diagnostics.Truncated {
			break
		}
	}
	// Load seed titles with the identical ACL and collect version/hash receipts.
	seedRows, e := tx.Query(r.Context(), `SELECT `+documentQueryDocJSON(false, false)+` FROM documents d LEFT JOIN knowledge_document_meta k ON k.document_id=d.id WHERE d.id=ANY($3::uuid[]) AND d.workspace_id=$2 AND d.deleted_at IS NULL AND madi_document_allowed($1,d.id,false)`, current(r).ID, parent.WorkspaceID, seeds)
	if e != nil {
		return out, e
	}
	for seedRows.Next() {
		var raw []byte
		if e = seedRows.Scan(&raw); e != nil {
			seedRows.Close()
			return out, e
		}
		if out.Diagnostics.ScannedBytes+len(raw) > documentQueryScanBytes {
			seedRows.Close()
			return out, approvalProblem(413, "관계 조회의 원문·경로 메타데이터 예산을 초과했습니다. 범위와 깊이를 줄이세요")
		}
		out.Diagnostics.ScannedBytes += len(raw)
		d, e := documentQueryDecodeDoc(raw)
		if e != nil {
			seedRows.Close()
			return out, e
		}
		out.Documents[d.ID] = d
	}
	e = seedRows.Err()
	seedRows.Close()
	if e != nil {
		return out, e
	}
	out.Diagnostics.DocumentsScanned = len(out.Documents)
	for _, d := range out.Documents {
		for _, field := range []string{"title_truncated", "owner_truncated", "tags_truncated"} {
			if boolean(d.Values, field) {
				out.Diagnostics.TruncatedFields++
			}
		}
	}
	kept := []documentQueryRow{}
	for _, row := range out.Rows {
		a, aok := out.Documents[str(row.Values, "source_id")]
		b, bok := out.Documents[str(row.Values, "target_id")]
		if !aok || !bok {
			continue
		}
		row.Values["source_title"] = a.Values["title"]
		row.Values["target_title"] = b.Values["title"]
		row.Version = out.Documents[row.DocumentID].Version
		if documentQueryMatches(q, row.Values, params) {
			kept = append(kept, row)
		}
	}
	out.Rows = kept
	documentQuerySort(q, out.Rows)
	if len(out.Rows) > q.Limit {
		out.Rows = out.Rows[:q.Limit]
		out.Diagnostics.Truncated = true
	}
	return out, nil
}

func documentQuerySourceHashes(ctx context.Context, tx pgx.Tx, p *Principal, wid string, ids []string, body, tasks bool) (map[string]string, error) {
	result := map[string]string{}
	rows, e := tx.Query(ctx, `SELECT `+documentQueryDocJSON(body, tasks)+` FROM documents d LEFT JOIN knowledge_document_meta k ON k.document_id=d.id WHERE d.workspace_id=$2 AND d.id=ANY($3::uuid[]) AND d.deleted_at IS NULL AND madi_document_allowed($1,d.id,false)`, p.ID, wid, ids)
	if e != nil {
		return result, e
	}
	defer rows.Close()
	for rows.Next() {
		var raw []byte
		if e = rows.Scan(&raw); e != nil {
			return result, e
		}
		var id struct{ ID string }
		if e = json.Unmarshal(raw, &id); e != nil {
			return result, e
		}
		result[id.ID] = digest(string(raw))
	}
	return result, rows.Err()
}
