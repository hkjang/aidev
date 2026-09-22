package server

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

//go:embed canvas.sql
var canvasSchema string

type canvasNode struct {
	ID     string       `json:"id"`
	Kind   string       `json:"kind"`
	X      float64      `json:"x"`
	Y      float64      `json:"y"`
	Width  float64      `json:"width"`
	Height float64      `json:"height"`
	Color  string       `json:"color"`
	Title  string       `json:"title,omitempty"`
	Text   string       `json:"text,omitempty"`
	RefID  string       `json:"ref_id,omitempty"`
	URL    string       `json:"url,omitempty"`
	Points [][2]float64 `json:"points,omitempty"`
}
type canvasEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label,omitempty"`
	Color  string `json:"color"`
}
type canvasData struct {
	Nodes []canvasNode `json:"nodes"`
	Edges []canvasEdge `json:"edges"`
}

func (s *Server) migrateCanvas(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, canvasSchema)
	return e
}
func (s *Server) registerCanvas() {
	s.handle("GET /api/v1/canvases", s.listCanvases)
	s.handle("POST /api/v1/canvases", s.createCanvas)
	s.handle("GET /api/v1/canvases/{id}", s.getCanvas)
	s.handle("GET /api/v1/canvases/{id}/images", s.canvasImages)
	s.handle("POST /api/v1/canvases/{id}/validate", s.validateCanvasImport)
	s.handle("PUT /api/v1/canvases/{id}", s.updateCanvas)
	s.handle("DELETE /api/v1/canvases/{id}", s.deleteCanvas)
	s.handle("POST /api/v1/canvases/{id}/restore", s.restoreCanvas)
	s.handle("GET /api/v1/canvases/{id}/shares", s.canvasShares)
	s.handle("PUT /api/v1/canvases/{id}/shares", s.setCanvasShare)
	s.handle("POST /api/v1/canvases/{id}/ai/{nodeId}", s.canvasAI)
}
func validateCanvasData(data *canvasData) error {
	if data.Nodes == nil {
		data.Nodes = []canvasNode{}
	}
	if data.Edges == nil {
		data.Edges = []canvasEdge{}
	}
	if len(data.Nodes) > 500 || len(data.Edges) > 1000 || len(jsonValue(data)) > 4<<20 {
		return errors.New("캔버스는 노드 500개, 연결선 1,000개, 전체 4MB까지 지원합니다")
	}
	nodes, edges := map[string]bool{}, map[string]bool{}
	finite := func(value float64) bool {
		return !math.IsNaN(value) && !math.IsInf(value, 0) && math.Abs(value) <= 100000
	}
	for _, n := range data.Nodes {
		if !validID(n.ID) || nodes[n.ID] || !oneOf(n.Kind, "note", "document", "image", "url", "database", "ai", "diagram", "rectangle", "ellipse", "diamond", "drawing") {
			return errors.New("노드 ID 또는 유형을 확인하세요")
		}
		nodes[n.ID] = true
		if !finite(n.X) || !finite(n.Y) || !finite(n.Width) || !finite(n.Height) || n.Width < 40 || n.Height < 40 || n.Width > 4000 || n.Height > 4000 {
			return errors.New("노드 좌표 또는 크기가 올바르지 않습니다")
		}
		if !oneOf(n.Color, "mint", "lavender", "sand", "sky", "rose", "white") || len(n.Title) > 500 || len(n.Text) > 200000 {
			return errors.New("노드 색상·제목·내용의 길이를 확인하세요")
		}
		if n.Kind == "diagram" && len(n.Text) > 50000 {
			return errors.New("Mermaid 소스는 50,000바이트 이하여야 합니다")
		}
		if oneOf(n.Kind, "document", "image", "database") && !validID(n.RefID) {
			return errors.New("연결할 문서·이미지·데이터베이스를 선택하세요")
		}
		if n.Kind == "url" {
			u, e := url.Parse(n.URL)
			if e != nil || !oneOf(u.Scheme, "http", "https") || u.Host == "" || u.User != nil || len(n.URL) > 4096 {
				return errors.New("링크는 사용자 정보가 없는 HTTP 또는 HTTPS 주소여야 합니다")
			}
		}
		if len(n.Points) > 2000 {
			return errors.New("펜 획은 최대 2,000개 좌표를 지원합니다")
		}
		for _, point := range n.Points {
			if !finite(point[0]) || !finite(point[1]) {
				return errors.New("펜 좌표를 확인하세요")
			}
		}
	}
	for _, edge := range data.Edges {
		if !validID(edge.ID) || edges[edge.ID] || !nodes[edge.Source] || !nodes[edge.Target] || edge.Source == edge.Target || len(edge.Label) > 500 || !oneOf(edge.Color, "mint", "lavender", "sand", "sky", "rose", "white") {
			return errors.New("연결선의 ID·양 끝 노드·라벨·색상을 확인하세요")
		}
		edges[edge.ID] = true
	}
	return nil
}
func (s *Server) canCanvas(r *http.Request, id string, write bool) bool {
	scope := "document:read"
	if write {
		scope = "document:write"
	}
	if !validID(id) || !hasIntegrationScope(current(r), scope) {
		return false
	}
	var wid string
	var allowed bool
	if s.DB.QueryRow(r.Context(), "SELECT workspace_id::text,madi_canvas_allowed($1,id,$3) FROM canvases WHERE id=$2", current(r).ID, id, write).Scan(&wid, &allowed) != nil {
		return false
	}
	return allowed && (current(r).WorkspaceID == "" || current(r).WorkspaceID == wid) && s.canFeature(r.Context(), current(r), wid, "canvas")
}
func (s *Server) canvas(r *http.Request, id string) (map[string]any, error) {
	v, e := s.one(r.Context(), "SELECT to_jsonb(c)||jsonb_build_object('can_write',madi_canvas_allowed($2,c.id,true) AND c.deleted_at IS NULL,'can_restore',madi_canvas_allowed($2,c.id,true) AND c.deleted_at IS NOT NULL,'can_manage',c.owner_id=$2 AND madi_canvas_allowed($2,c.id,true)) FROM canvases c WHERE id=$1 AND madi_canvas_allowed($2,c.id,false) AND madi_feature_allowed($2,c.workspace_id,'canvas') AND ($3='' OR c.workspace_id::text=$3)", id, current(r).ID, current(r).WorkspaceID)
	if e == nil {
		v["can_write"] = boolean(v, "can_write") && hasIntegrationScope(current(r), "document:write")
		v["can_manage"] = boolean(v, "can_manage") && hasIntegrationScope(current(r), "document:write")
		v["can_restore"] = boolean(v, "can_restore") && hasIntegrationScope(current(r), "document:write")
	}
	return v, e
}
func (s *Server) listCanvases(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if !s.canWorkspace(r.Context(), current(r), wid, false) || !hasIntegrationScope(current(r), "document:read") {
		apiError(w, 403, "워크스페이스 조회 권한이 없습니다")
		return
	}
	if !s.requireFeature(w, r, wid, "canvas") {
		return
	}
	rows, e := s.rows(r.Context(), "SELECT (to_jsonb(c)-'data')||jsonb_build_object('node_count',jsonb_array_length(c.data->'nodes'),'can_write',madi_canvas_allowed($1,c.id,true) AND c.deleted_at IS NULL,'can_restore',madi_canvas_allowed($1,c.id,true) AND c.deleted_at IS NOT NULL) FROM canvases c WHERE c.workspace_id=$2 AND madi_canvas_allowed($1,c.id,false) AND madi_feature_allowed($1,c.workspace_id,'canvas') AND (c.deleted_at IS NOT NULL)=$3 ORDER BY c.updated_at DESC LIMIT 1000", current(r).ID, wid, r.URL.Query().Get("trash") == "true")
	for _, v := range rows {
		v["can_write"] = boolean(v, "can_write") && hasIntegrationScope(current(r), "document:write")
		v["can_restore"] = boolean(v, "can_restore") && hasIntegrationScope(current(r), "document:write")
	}
	respond(w, rows, e)
}

func (s *Server) canvasImages(w http.ResponseWriter, r *http.Request) {
	id, doc := r.PathValue("id"), r.URL.Query().Get("document_id")
	if !s.canCanvas(r, id, false) || !s.canDocument(r.Context(), current(r), doc, false) {
		apiError(w, 404, "원본 문서 접근 권한이 없습니다")
		return
	}
	var same bool
	if s.DB.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM documents d JOIN canvases c ON c.workspace_id=d.workspace_id WHERE d.id=$1 AND c.id=$2 AND d.deleted_at IS NULL)", doc, id).Scan(&same) != nil || !same {
		apiError(w, 404, "같은 워크스페이스의 원본 문서를 선택하세요")
		return
	}
	v, e := s.rows(r.Context(), "SELECT jsonb_build_object('id',id,'name',name,'content_type',content_type) FROM attachments WHERE document_id=$1 AND content_type IN ('image/png','image/jpeg','image/gif','image/webp','image/avif') ORDER BY name LIMIT 1000", doc)
	w.Header().Set("Cache-Control", "no-store")
	respond(w, v, e)
}
func (s *Server) validateCanvasImport(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canCanvas(r, id, true) {
		apiError(w, 403, "캔버스 수정 권한이 없습니다")
		return
	}
	var data canvasData
	if decode(r, &data) != nil {
		apiError(w, 400, "캔버스 JSON 형식을 확인하세요")
		return
	}
	if e := validateCanvasData(&data); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	c, e := s.canvas(r, id)
	if e != nil {
		respond(w, nil, e)
		return
	}
	var old canvasData
	_ = json.Unmarshal(jsonValue(c["data"]), &old)
	if e = s.validateCanvasReferences(r, str(c, "workspace_id"), data, old); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	jsonResponse(w, 200, data)
}
func (s *Server) canvasReference(r *http.Request, wid string, n canvasNode) (map[string]any, error) {
	switch n.Kind {
	case "document":
		if !hasIntegrationScope(current(r), "document:read") || !s.canDocument(r.Context(), current(r), n.RefID, false) {
			return nil, errors.New("연결 문서 접근 권한이 없습니다")
		}
		v, e := s.one(r.Context(), "SELECT jsonb_build_object('title',title,'snippet',left(markdown,600),'url','/app/documents/'||id,'workspace_id',workspace_id) FROM documents WHERE id=$1 AND deleted_at IS NULL", n.RefID)
		if e != nil || str(v, "workspace_id") != wid {
			return nil, errors.New("같은 워크스페이스의 문서만 연결할 수 있습니다")
		}
		return v, nil
	case "database":
		if !hasIntegrationScope(current(r), "database:read") || !s.canDatabase(r, n.RefID, false) {
			return nil, errors.New("연결 데이터베이스 접근 권한이 없습니다")
		}
		v, e := s.one(r.Context(), "SELECT jsonb_build_object('title',name,'snippet',jsonb_array_length(properties)||'개 속성','url','/app/databases/'||id,'workspace_id',workspace_id) FROM databases WHERE id=$1", n.RefID)
		if e != nil || str(v, "workspace_id") != wid {
			return nil, errors.New("같은 워크스페이스의 데이터베이스만 연결할 수 있습니다")
		}
		return v, nil
	case "image":
		v, e := s.one(r.Context(), "SELECT jsonb_build_object('title',a.name,'content_type',a.content_type,'document_id',a.document_id,'workspace_id',d.workspace_id,'url','/api/v1/attachments/'||a.id) FROM attachments a JOIN documents d ON d.id=a.document_id WHERE a.id=$1 AND d.deleted_at IS NULL", n.RefID)
		if e != nil || str(v, "workspace_id") != wid || !hasIntegrationScope(current(r), "document:read") || !s.canDocument(r.Context(), current(r), str(v, "document_id"), false) {
			return nil, errors.New("이미지 원본 문서 접근 권한이 없습니다")
		}
		if !oneOf(str(v, "content_type"), "image/png", "image/jpeg", "image/gif", "image/webp", "image/avif") {
			return nil, errors.New("PNG·JPEG·GIF·WebP·AVIF 이미지 파일만 표시할 수 있습니다")
		}
		return v, nil
	}
	return nil, nil
}
func (s *Server) validateCanvasReferences(r *http.Request, wid string, data, old canvasData) error {
	existing := map[string]canvasNode{}
	for _, n := range old.Nodes {
		existing[n.ID] = n
	}
	for _, n := range data.Nodes {
		if prior, ok := existing[n.ID]; ok && prior.Kind == n.Kind && prior.RefID == n.RefID {
			continue
		}
		if _, e := s.canvasReference(r, wid, n); e != nil {
			return e
		}
	}
	return nil
}
func (s *Server) canvasResponse(w http.ResponseWriter, r *http.Request, id string) {
	v, e := s.canvas(r, id)
	if e != nil {
		respond(w, nil, e)
		return
	}
	var data canvasData
	_ = json.Unmarshal(jsonValue(v["data"]), &data)
	resolved := map[string]any{}
	for _, n := range data.Nodes {
		if !oneOf(n.Kind, "document", "image", "database") {
			continue
		}
		value, e := s.canvasReference(r, str(v, "workspace_id"), n)
		if e != nil {
			resolved[n.ID] = map[string]any{"unavailable": true, "title": "접근할 수 없는 항목", "error": "원본이 삭제되었거나 현재 접근 권한이 없습니다"}
		} else {
			resolved[n.ID] = value
		}
	}
	v["resolved"] = resolved
	// References are resolved for each read and must never enter persistent client drafts.
	w.Header().Set("Cache-Control", "no-store")
	jsonResponse(w, 200, v)
}
func (s *Server) getCanvas(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canCanvas(r, id, false) {
		apiError(w, 404, "캔버스를 찾을 수 없거나 접근 권한이 없습니다")
		return
	}
	s.canvasResponse(w, r, id)
	s.audit(r, "CANVAS_READ", id, nil)
}
func (s *Server) createCanvas(w http.ResponseWriter, r *http.Request) {
	var in struct {
		WorkspaceID string     `json:"workspace_id"`
		SpaceID     string     `json:"space_id"`
		Title       string     `json:"title"`
		Visibility  string     `json:"visibility"`
		Data        canvasData `json:"data"`
	}
	if decode(r, &in) != nil {
		apiError(w, 400, "캔버스 입력값을 확인하세요")
		return
	}
	in.Title = strings.TrimSpace(in.Title)
	if in.Visibility == "" {
		in.Visibility = "private"
	}
	if !hasIntegrationScope(current(r), "document:write") || !s.canSpace(r.Context(), current(r), in.WorkspaceID, in.SpaceID, true) {
		apiError(w, 403, "캔버스 작성 권한이 없습니다")
		return
	}
	if !s.requireFeature(w, r, in.WorkspaceID, "canvas") {
		return
	}
	if in.Title == "" || len(in.Title) > 500 || !oneOf(in.Visibility, "private", "workspace", "selected") {
		apiError(w, 400, "제목과 공개 범위를 확인하세요")
		return
	}
	if e := validateCanvasData(&in.Data); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	if e := s.validateCanvasReferences(r, in.WorkspaceID, in.Data, canvasData{}); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	id := newID()
	inserted, e := s.DB.Exec(r.Context(), "INSERT INTO canvases(id,workspace_id,space_id,owner_id,title,visibility,data) SELECT $1,$2,NULLIF($3,'')::uuid,$4,$5,$6,$7 WHERE madi_feature_allowed($4,$2,'canvas')", id, in.WorkspaceID, in.SpaceID, current(r).ID, in.Title, in.Visibility, jsonValue(in.Data))
	if e != nil {
		respond(w, nil, e)
		return
	}
	if inserted.RowsAffected() != 1 {
		apiError(w, 403, "캔버스 기능 공개 정책이 변경되었습니다")
		return
	}
	s.audit(r, "CANVAS_CREATE", id, nil)
	s.canvasResponse(w, r, id)
}
func (s *Server) updateCanvas(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canCanvas(r, id, true) {
		apiError(w, 403, "캔버스 수정 권한이 없습니다")
		return
	}
	var in struct {
		Version    int        `json:"version"`
		Title      string     `json:"title"`
		Visibility string     `json:"visibility"`
		SpaceID    *string    `json:"space_id"`
		Data       canvasData `json:"data"`
	}
	if decode(r, &in) != nil {
		apiError(w, 400, "캔버스 입력값을 확인하세요")
		return
	}
	in.Title = strings.TrimSpace(in.Title)
	old, e := s.canvas(r, id)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if old["deleted_at"] != nil {
		apiError(w, 409, "휴지통의 캔버스를 먼저 복원하세요")
		return
	}
	if in.Visibility == "" {
		in.Visibility = str(old, "visibility")
	}
	space := str(old, "space_id")
	if in.SpaceID != nil {
		space = *in.SpaceID
	}
	if (in.Visibility != str(old, "visibility") || space != str(old, "space_id")) && !boolean(old, "can_manage") {
		apiError(w, 403, "공개 범위와 공간은 캔버스 소유자만 변경할 수 있습니다")
		return
	}
	if in.Title == "" || len(in.Title) > 500 || !oneOf(in.Visibility, "private", "workspace", "selected") || in.Version < 1 {
		apiError(w, 400, "제목·공개 범위·버전을 확인하세요")
		return
	}
	if !s.canSpace(r.Context(), current(r), str(old, "workspace_id"), space, true) {
		apiError(w, 403, "대상 공간에 쓸 권한이 없습니다")
		return
	}
	if e = validateCanvasData(&in.Data); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	var oldData canvasData
	_ = json.Unmarshal(jsonValue(old["data"]), &oldData)
	if e = s.validateCanvasReferences(r, str(old, "workspace_id"), in.Data, oldData); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	result, e := s.DB.Exec(r.Context(), "UPDATE canvases SET title=$2,visibility=$3,space_id=NULLIF($4,'')::uuid,data=$5,version=version+1,updated_at=now() WHERE id=$1 AND version=$6 AND deleted_at IS NULL AND madi_canvas_allowed($7,id,true) AND madi_feature_allowed($7,workspace_id,'canvas')", id, in.Title, in.Visibility, space, jsonValue(in.Data), in.Version, current(r).ID)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if result.RowsAffected() != 1 {
		apiError(w, 409, "다른 사용자가 캔버스를 변경했습니다. 현재 작업을 내보낸 뒤 최신 내용을 다시 불러오세요")
		return
	}
	s.audit(r, "CANVAS_UPDATE", id, map[string]any{"version": in.Version + 1})
	s.canvasResponse(w, r, id)
}
func (s *Server) deleteCanvas(w http.ResponseWriter, r *http.Request) {
	s.changeCanvasTrash(w, r, true)
}
func (s *Server) restoreCanvas(w http.ResponseWriter, r *http.Request) {
	s.changeCanvasTrash(w, r, false)
}
func (s *Server) changeCanvasTrash(w http.ResponseWriter, r *http.Request, deleted bool) {
	id := r.PathValue("id")
	if !s.canCanvas(r, id, true) {
		apiError(w, 403, "캔버스를 변경할 권한이 없습니다")
		return
	}
	query := "UPDATE canvases SET deleted_at=now(),version=version+1,updated_at=now() WHERE id=$1 AND deleted_at IS NULL AND madi_canvas_allowed($2,id,true)"
	action := "CANVAS_DELETE"
	if !deleted {
		query = "UPDATE canvases SET deleted_at=NULL,version=version+1,updated_at=now() WHERE id=$1 AND deleted_at IS NOT NULL AND madi_canvas_allowed($2,id,true)"
		action = "CANVAS_RESTORE"
	}
	_, e := s.DB.Exec(r.Context(), query, id, current(r).ID)
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, action, id, nil)
	jsonResponse(w, 200, map[string]any{"ok": true})
}
func (s *Server) canvasShares(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canCanvas(r, id, false) {
		apiError(w, 404, "캔버스를 찾을 수 없습니다")
		return
	}
	v, e := s.canvas(r, id)
	if e != nil || !boolean(v, "can_manage") {
		apiError(w, 403, "소유자만 공유 설정을 볼 수 있습니다")
		return
	}
	rows, e := s.rows(r.Context(), "SELECT jsonb_build_object('user_id',u.id,'name',u.name,'email',u.email,'permission',sh.permission) FROM canvas_shares sh JOIN users u ON u.id=sh.user_id WHERE sh.canvas_id=$1 ORDER BY u.name", id)
	respond(w, rows, e)
}
func (s *Server) setCanvasShare(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canCanvas(r, id, true) {
		apiError(w, 403, "캔버스 공유 권한이 없습니다")
		return
	}
	canvas, e := s.canvas(r, id)
	if e != nil || !boolean(canvas, "can_manage") {
		apiError(w, 403, "소유자만 공유 설정을 변경할 수 있습니다")
		return
	}
	var in struct {
		Email      string `json:"email"`
		Permission string `json:"permission"`
	}
	if decode(r, &in) != nil || !oneOf(in.Permission, "read", "write", "remove") {
		apiError(w, 400, "이메일과 권한을 확인하세요")
		return
	}
	var uid, role string
	if e = s.DB.QueryRow(r.Context(), "SELECT u.id::text,u.role FROM users u JOIN workspace_members m ON m.user_id=u.id WHERE lower(u.email)=lower($1) AND m.workspace_id=$2 AND NOT u.disabled", strings.TrimSpace(in.Email), str(canvas, "workspace_id")).Scan(&uid, &role); e != nil {
		apiError(w, 400, "같은 워크스페이스의 활성 사용자를 선택하세요")
		return
	}
	if uid == str(canvas, "owner_id") {
		apiError(w, 400, "소유자의 권한은 공유 목록에서 변경할 수 없습니다")
		return
	}
	if in.Permission != "remove" && !s.canSpace(r.Context(), &Principal{ID: uid, Role: role}, str(canvas, "workspace_id"), str(canvas, "space_id"), false) {
		apiError(w, 400, "먼저 사용자에게 캔버스가 속한 공간 접근 권한을 부여하세요")
		return
	}
	if in.Permission == "remove" {
		_, e = s.DB.Exec(r.Context(), "DELETE FROM canvas_shares WHERE canvas_id=$1 AND user_id=$2", id, uid)
	} else {
		_, e = s.DB.Exec(r.Context(), "INSERT INTO canvas_shares VALUES($1,$2,$3) ON CONFLICT(canvas_id,user_id) DO UPDATE SET permission=excluded.permission", id, uid, in.Permission)
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "CANVAS_SHARE", id, map[string]any{"user_id": uid, "permission": in.Permission})
	jsonResponse(w, 200, map[string]any{"ok": true})
}
func (s *Server) canvasAI(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canCanvas(r, id, true) || !hasIntegrationScope(current(r), "ai:execute") {
		apiError(w, 403, "캔버스 수정 및 AI 실행 권한이 필요합니다")
		return
	}
	v, e := s.canvas(r, id)
	if e != nil || v["deleted_at"] != nil {
		apiError(w, 404, "캔버스를 찾을 수 없습니다")
		return
	}
	var data canvasData
	_ = json.Unmarshal(jsonValue(v["data"]), &data)
	var node *canvasNode
	for i := range data.Nodes {
		if data.Nodes[i].ID == r.PathValue("nodeId") && data.Nodes[i].Kind == "ai" {
			node = &data.Nodes[i]
			break
		}
	}
	if node == nil || strings.TrimSpace(node.Text) == "" {
		apiError(w, 400, "AI 노드의 질문을 먼저 저장하세요")
		return
	}
	body := jsonValue(map[string]any{"workspace_id": str(v, "workspace_id"), "prompt": node.Text})
	streamCtx, cancel := context.WithCancel(r.Context())
	defer cancel()
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-streamCtx.Done():
				return
			case <-ticker.C:
				if !s.canCanvas(r.Clone(streamCtx), id, true) {
					cancel()
					return
				}
			}
		}
	}()
	req := r.Clone(streamCtx)
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.ContentLength = int64(len(body))
	s.audit(r, "CANVAS_AI_QUERY", id, map[string]any{"node_id": node.ID})
	s.aiChat(w, req)
}
