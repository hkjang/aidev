package server

import (
	"bytes"
	"context"
	_ "embed"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type captureContextKey struct{}
type captureInput struct {
	Source                 map[string]any
	RequestID, PayloadHash string
}

//go:embed capture.sql
var captureSchema string

func (s *Server) migrateCaptures(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, captureSchema)
	return e
}

func (s *Server) registerCaptures() {
	s.handle("GET /api/v1/captures", s.listCaptures)
	s.handle("POST /api/v1/captures", s.createCapture)
	s.handle("POST /api/v1/captures/{id}/classify", s.classifyCapture)
}
func (s *Server) createCapture(w http.ResponseWriter, r *http.Request) {
	var in struct {
		WorkspaceID string   `json:"workspace_id"`
		Title       string   `json:"title"`
		Text        string   `json:"text"`
		URL         string   `json:"url"`
		Tags        []string `json:"tags"`
		RequestID   string   `json:"client_request_id"`
	}
	if decode(r, &in) != nil || len(in.Text) > 4<<20 || len(in.URL) > 8192 || len(in.Tags) > 100 {
		apiError(w, 400, "수집할 내용은 4MiB, URL은 8KiB, 태그는 100개 이내입니다")
		return
	}
	if in.RequestID != "" && !validID(in.RequestID) {
		apiError(w, 400, "client_request_id는 UUID여야 합니다")
		return
	}
	in.RequestID = strings.ToLower(in.RequestID)
	source := strings.TrimSpace(in.URL)
	if source != "" {
		u, e := url.Parse(source)
		if e != nil || !oneOf(u.Scheme, "http", "https") || u.Hostname() == "" || u.User != nil || strings.ContainsAny(source, "\r\n<>") {
			apiError(w, 400, "출처는 계정 정보가 없는 HTTP(S) URL이어야 합니다")
			return
		}
		source = u.String()
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = strings.TrimSpace(strings.SplitN(in.Text, "\n", 2)[0])
		if title == "" && source != "" {
			u, _ := url.Parse(source)
			title = u.Hostname()
		}
		if title == "" {
			title = "빠르게 수집한 메모"
		}
		runes := []rune(title)
		if len(runes) > 150 {
			title = string(runes[:150])
		}
		for len(title) > 500 {
			runes = []rune(title)
			title = string(runes[:len(runes)-1])
		}
	}
	markdown := in.Text
	if source != "" {
		markdown += "\n\n출처: " + source + "\n"
	}
	payload := map[string]any{"workspace_id": in.WorkspaceID, "title": title, "markdown": markdown, "visibility": "private", "tags": in.Tags}
	input := r.Clone(context.WithValue(r.Context(), captureContextKey{}, captureInput{Source: map[string]any{"source_url": source, "origin": "quick_capture"}, RequestID: in.RequestID, PayloadHash: digest(string(jsonValue(payload)))}))
	input.Body = io.NopCloser(bytes.NewReader(jsonValue(payload)))
	s.createDocument(w, input)
}
func (s *Server) listCaptures(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if !s.canWorkspace(r.Context(), current(r), wid, false) {
		apiError(w, 403, "워크스페이스 접근 권한이 없습니다")
		return
	}
	after := r.URL.Query().Get("after")
	if after != "" && !validID(after) {
		apiError(w, 400, "수집함 페이지 위치를 확인하세요")
		return
	}
	v, e := s.rows(r.Context(), `SELECT jsonb_build_object('id',d.id,'title',d.title,'workspace_id',d.workspace_id,'version',d.version,'created_at',d.created_at,'updated_at',d.updated_at,'tags',`+documentSummaryTags+`,'tags_truncated',jsonb_array_length(d.tags)>jsonb_array_length(`+documentSummaryTags+`),'visibility',d.visibility,'preview',left(d.markdown,300),'source_url',left(k.system_metadata->>'source_url',2048)) FROM documents d JOIN knowledge_document_meta k ON k.document_id=d.id WHERE d.workspace_id=$2 AND d.owner_id=$1 AND d.deleted_at IS NULL AND k.kind='inbox' AND `+docACL+` AND ($3='' OR (d.created_at,d.id)<(SELECT created_at,id FROM documents WHERE id=NULLIF($3,'')::uuid AND owner_id=$1 AND workspace_id=$2)) ORDER BY d.created_at DESC,d.id DESC LIMIT 200`, current(r).ID, wid, after)
	respond(w, v, e)
}
func (s *Server) classifyCapture(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canDocument(r.Context(), current(r), id, true) {
		apiError(w, 403, "수집 문서 분류 권한이 없습니다")
		return
	}
	var in struct {
		Version    int    `json:"version"`
		Space      string `json:"space_id"`
		Parent     string `json:"parent_id"`
		Visibility string `json:"visibility"`
		Kind       string `json:"kind"`
	}
	if decode(r, &in) != nil || !oneOf(in.Visibility, "private", "workspace", "selected") || !oneOf(in.Kind, "page", "note", "daily", "meeting", "decision", "runbook", "template", "entity") {
		apiError(w, 400, "분류할 문서 종류와 공유 범위를 선택하세요")
		return
	}
	var wid string
	if e := s.DB.QueryRow(r.Context(), "SELECT workspace_id::text FROM documents WHERE id=$1", id).Scan(&wid); e != nil {
		respond(w, nil, e)
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	// Match every tree mutation's lock order: workspace tree lock, then row.
	if e = s.treePlacementTx(r, tx, "documents", wid, id, in.Parent); e != nil {
		apiError(w, 409, e.Error())
		return
	}
	var version int
	e = tx.QueryRow(r.Context(), "SELECT d.workspace_id::text,d.version FROM documents d JOIN knowledge_document_meta k ON k.document_id=d.id WHERE d.id=$1 AND d.owner_id=$2 AND d.deleted_at IS NULL AND k.kind='inbox' FOR UPDATE OF d", id, current(r).ID).Scan(&wid, &version)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if in.Version != version {
		apiError(w, 409, "수집 문서가 변경되었습니다. 다시 불러오세요")
		return
	}
	var allowed bool
	if in.Space != "" && !validID(in.Space) {
		apiError(w, 400, "대상 공간 ID가 올바르지 않습니다")
		return
	}
	e = tx.QueryRow(r.Context(), "SELECT madi_document_allowed($1,$2,true) AND ($3='' OR EXISTS(SELECT 1 FROM spaces WHERE id=NULLIF($3,'')::uuid AND workspace_id=$4 AND madi_space_allowed($1,id,true)))", current(r).ID, id, in.Space, wid).Scan(&allowed)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if !allowed {
		apiError(w, 403, "대상 공간의 작성 권한이 없습니다")
		return
	}
	_, e = tx.Exec(r.Context(), "UPDATE documents SET space_id=NULLIF($2,'')::uuid,parent_id=NULLIF($3,'')::uuid,visibility=$4,version=version+1,updated_at=now() WHERE id=$1", id, in.Space, in.Parent, in.Visibility)
	if e == nil {
		_, e = tx.Exec(r.Context(), "UPDATE knowledge_document_meta SET kind=$2,classification=CASE WHEN classification='internal' THEN coalesce((SELECT classification FROM spaces WHERE id=NULLIF($3,'')::uuid),'internal') ELSE classification END,updated_at=now() WHERE document_id=$1", id, in.Kind, in.Space)
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), "INSERT INTO document_versions(document_id,version,title,markdown,tags,user_id,block_metadata) SELECT id,version,title,markdown,tags,$2,block_metadata FROM documents WHERE id=$1", id, current(r).ID)
	}
	if e == nil {
		e = s.enqueueEvent(r.Context(), tx, Event{Type: "document.updated", WorkspaceID: wid, ResourceID: id, After: map[string]any{"id": id, "kind": in.Kind, "visibility": in.Visibility, "version": version + 1}})
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "CAPTURE_CLASSIFY", id, map[string]any{"kind": in.Kind, "space_id": in.Space, "visibility": in.Visibility})
	v, e := s.document(r, id)
	respond(w, v, e)
}
