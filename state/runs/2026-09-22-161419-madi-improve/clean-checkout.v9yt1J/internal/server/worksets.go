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

//go:embed worksets.sql
var worksetsSchema string

func (s *Server) migrateWorksets(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, worksetsSchema)
	return e
}
func (s *Server) registerWorksets() {
	s.registerWorksetDocuments()
	s.handle("GET /api/v1/worksets", s.listWorksets)
	s.handle("POST /api/v1/worksets", s.saveWorkset)
	s.handle("GET /api/v1/worksets/{id}", s.getWorkset)
	s.handle("PUT /api/v1/worksets/{id}", s.saveWorkset)
	s.handle("DELETE /api/v1/worksets/{id}", s.deleteWorkset)
}

type worksetItem struct {
	Kind       string         `json:"kind"`
	ResourceID string         `json:"resource_id"`
	Context    map[string]any `json:"context"`
}
type worksetRecord struct {
	ID          string        `json:"id"`
	WorkspaceID string        `json:"workspace_id"`
	OwnerID     string        `json:"owner_id"`
	Name        string        `json:"name"`
	Kind        string        `json:"kind"`
	Version     int           `json:"version"`
	Items       []worksetItem `json:"items"`
}

func (s *Server) worksetTx(r *http.Request, tx pgx.Tx, lock bool) (worksetRecord, error) {
	var result worksetRecord
	if !personalAccessRequest(current(r)) || !validID(r.PathValue("id")) {
		return result, pgx.ErrNoRows
	}
	suffix := " FOR SHARE"
	if lock {
		suffix = " FOR UPDATE"
	}
	e := tx.QueryRow(r.Context(), "SELECT id::text,workspace_id::text,owner_id::text,name,kind,version FROM user_worksets WHERE id=$1 AND owner_id=$2"+suffix, r.PathValue("id"), current(r).ID).Scan(&result.ID, &result.WorkspaceID, &result.OwnerID, &result.Name, &result.Kind, &result.Version)
	if e != nil {
		return result, e
	}
	if e = documentAccessActorTx(r, tx, result.WorkspaceID, false); e != nil {
		return result, e
	}
	rows, e := tx.Query(r.Context(), "SELECT kind,resource_id::text,context FROM workset_items WHERE workset_id=$1 ORDER BY ordinal", result.ID)
	if e != nil {
		return result, e
	}
	defer rows.Close()
	result.Items = []worksetItem{}
	for rows.Next() {
		var item worksetItem
		if e = rows.Scan(&item.Kind, &item.ResourceID, &item.Context); e != nil {
			return result, e
		}
		result.Items = append(result.Items, item)
	}
	return result, rows.Err()
}
func (s *Server) listWorksets(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if !validID(wid) || !personalAccessRequest(current(r)) {
		apiError(w, 403, "본인 브라우저와 현재 워크스페이스를 확인하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if e = documentAccessActorTx(r, tx, wid, false); e != nil {
		apiError(w, 403, e.Error())
		return
	}
	rows, e := tx.Query(r.Context(), `SELECT jsonb_build_object('id',a.id,'name',a.name,'kind',a.kind,'version',a.version,'updated_at',a.updated_at,'item_count',(SELECT count(*) FROM workset_items i WHERE i.workset_id=a.id)) FROM user_worksets a WHERE owner_id=$1 AND workspace_id=$2 ORDER BY updated_at DESC,id LIMIT 40`, current(r).ID, wid)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var raw []byte
		if e = rows.Scan(&raw); e != nil {
			respond(w, nil, e)
			return
		}
		var item map[string]any
		if e = json.Unmarshal(raw, &item); e != nil {
			respond(w, nil, e)
			return
		}
		items = append(items, item)
	}
	if e = rows.Err(); e != nil {
		respond(w, nil, e)
		return
	}
	jsonResponse(w, 200, map[string]any{"items": items, "private": true, "limit": 40})
}
func (s *Server) getWorkset(w http.ResponseWriter, r *http.Request) {
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	item, e := s.worksetTx(r, tx, false)
	if e != nil {
		apiError(w, 404, "현재 접근 가능한 내 작업 묶음이 없습니다")
		return
	}
	resolved := []map[string]any{}
	for _, source := range item.Items {
		value, e := s.resolveWorksetItem(r, tx, item.WorkspaceID, source)
		if e != nil {
			respond(w, nil, e)
			return
		}
		resolved = append(resolved, value)
	}
	jsonResponse(w, 200, map[string]any{"workset": item, "resolved": resolved, "notice": "본인 작업 맥락의 참조만 저장합니다. 문서·데이터베이스의 현재 권한이 적용되며 원문을 브라우저 저장소에 복제하지 않습니다."})
}
func (s *Server) saveWorkset(w http.ResponseWriter, r *http.Request) {
	var in struct {
		WorkspaceID string        `json:"workspace_id"`
		Name        string        `json:"name"`
		Kind        string        `json:"kind"`
		Version     int           `json:"expected_version"`
		Items       []worksetItem `json:"items"`
	}
	if decode(r, &in) != nil || len(jsonValue(in)) > 65536 || len(strings.TrimSpace(in.Name)) == 0 || len(in.Name) > 200 || !oneOf(in.Kind, "workset", "reference") || len(in.Items) > 20 || in.Kind == "reference" && len(in.Items) > 6 || !personalAccessRequest(current(r)) {
		apiError(w, 400, "개인 묶음 이름·종류와 최대20개 참조(선반은 문서6개)를 확인하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	creating := r.PathValue("id") == ""
	var old worksetRecord
	id := r.PathValue("id")
	if creating {
		if !validID(in.WorkspaceID) {
			apiError(w, 400, "워크스페이스를 확인하세요")
			return
		}
		id = newID()
	} else {
		old, e = s.worksetTx(r, tx, true)
		if e != nil {
			apiError(w, 404, "현재 접근 가능한 내 작업 묶음이 없습니다")
			return
		}
		in.WorkspaceID = old.WorkspaceID
		if in.Version < 1 || in.Version != old.Version || in.Version >= 2147483647 {
			apiError(w, 409, "작업 묶음이 변경되었습니다. 다시 확인하세요")
			return
		}
	}
	if e = documentAccessActorTx(r, tx, in.WorkspaceID, false); e != nil {
		apiError(w, 403, e.Error())
		return
	}
	if creating {
		if _, e = tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock(hashtextextended($1,198))", current(r).ID+":"+in.WorkspaceID); e != nil {
			respond(w, nil, e)
			return
		}
		var count int
		if e = tx.QueryRow(r.Context(), "SELECT count(*) FROM user_worksets WHERE owner_id=$1 AND workspace_id=$2", current(r).ID, in.WorkspaceID).Scan(&count); e != nil {
			respond(w, nil, e)
			return
		}
		if count >= 40 {
			apiError(w, 400, "워크스페이스마다 개인 작업 묶음은 40개까지 저장합니다")
			return
		}
	}
	known := map[string]int{}
	for _, item := range old.Items {
		known[string(jsonValue(item))]++
	}
	for i := range in.Items {
		item := &in.Items[i]
		if item.Context == nil {
			item.Context = map[string]any{}
		}
		if !validID(item.ResourceID) || !oneOf(item.Kind, "document", "database", "task") || in.Kind == "reference" && item.Kind != "document" {
			apiError(w, 400, "지원하는 문서·데이터베이스·할 일 참조를 확인하세요")
			return
		}
		if e = validateWorksetContext(*item); e != nil {
			apiError(w, 400, e.Error())
			return
		}
		resolved, err := s.resolveWorksetItem(r, tx, in.WorkspaceID, *item)
		if err != nil {
			respond(w, nil, err)
			return
		}
		signature := string(jsonValue(item))
		if !boolean(resolved, "available") {
			if known[signature] < 1 {
				apiError(w, 403, "현재 접근 가능한 참조만 새로 추가할 수 있습니다")
				return
			}
			known[signature]--
		} else if item.Kind == "database" {
			if e = s.validateWorksetDatabaseContext(r, tx, in.WorkspaceID, *item); e != nil {
				apiError(w, 400, e.Error())
				return
			}
		}
	}
	protected, e := s.ProtectDocumentMetadataTx(r.Context(), tx, current(r), "", in.WorkspaceID, map[string]any{"name": strings.TrimSpace(in.Name)})
	if e != nil {
		if !WriteProtectionError(w, e) {
			respond(w, nil, e)
		}
		return
	}
	metadata, _ := protected.Value.(map[string]any)
	in.Name = str(metadata, "name")
	if creating {
		_, e = tx.Exec(r.Context(), "INSERT INTO user_worksets(id,workspace_id,owner_id,name,kind) VALUES($1,$2,$3,$4,$5)", id, in.WorkspaceID, current(r).ID, in.Name, in.Kind)
	} else {
		_, e = tx.Exec(r.Context(), "UPDATE user_worksets SET name=$2,kind=$3,version=version+1,updated_at=now() WHERE id=$1", id, in.Name, in.Kind)
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), "DELETE FROM workset_items WHERE workset_id=$1", id)
	}
	for i, item := range in.Items {
		if e != nil {
			break
		}
		_, e = tx.Exec(r.Context(), "INSERT INTO workset_items(workset_id,ordinal,kind,resource_id,context) VALUES($1,$2,$3,$4,$5)", id, i, item.Kind, item.ResourceID, jsonValue(item.Context))
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	version := old.Version + 1
	if creating {
		version = 1
	}
	jsonResponse(w, 200, map[string]any{"id": id, "version": version, "name": in.Name, "private": true})
}
func (s *Server) deleteWorkset(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Version int `json:"expected_version"`
	}
	if decode(r, &in) != nil || in.Version < 1 {
		apiError(w, 400, "확인한 묶음 버전이 필요합니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	old, e := s.worksetTx(r, tx, true)
	if e != nil {
		apiError(w, 404, "현재 접근 가능한 내 작업 묶음이 없습니다")
		return
	}
	if old.Version != in.Version {
		apiError(w, 409, "작업 묶음이 변경되었습니다")
		return
	}
	_, e = tx.Exec(r.Context(), "DELETE FROM user_worksets WHERE id=$1", old.ID)
	if e == nil {
		e = tx.Commit(r.Context())
	}
	respond(w, map[string]any{"deleted": true, "sources_deleted": false}, e)
}

var errWorksetContext = errors.New("위치 정보는 문서 버전·줄/선택 숫자, 할 일 ID, 안전한 데이터베이스 보기 구성만 저장합니다")
