package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"sort"

	"github.com/jackc/pgx/v5"
)

//go:embed database_editing.sql
var databaseEditingSchema string

func (s *Server) migrateDatabaseEditing(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, databaseEditingSchema)
	return e
}
func (s *Server) registerDatabaseEditing() {
	s.handle("GET /api/v1/databases/{id}/editing", s.databaseEditingMetadata)
	s.handle("POST /api/v1/databases/{id}/edit-cells", s.editDatabaseCells)
	s.handle("GET /api/v1/databases/{id}/views", s.listDatabaseViews)
	s.handle("POST /api/v1/databases/{id}/views", s.saveDatabaseView)
	s.handle("PUT /api/v1/databases/{id}/views/{viewID}", s.saveDatabaseView)
	s.handle("DELETE /api/v1/databases/{id}/views/{viewID}", s.deleteDatabaseView)
	s.handle("PUT /api/v1/databases/{id}/view-preference", s.saveDatabaseViewPreference)
}

type databaseEditingScope struct {
	WorkspaceID, Schema string
	Properties          []map[string]any
}

func (s *Server) databaseEditingScopeTx(r *http.Request, tx pgx.Tx, id string, write bool) (databaseEditingScope, error) {
	var value databaseEditingScope
	var raw []byte
	denied := errors.New("현재 데이터베이스 접근 권한이 없습니다")
	p := current(r)
	if !personalAccessRequest(p) || !validID(id) {
		return value, denied
	}
	e := tx.QueryRow(r.Context(), `SELECT d.workspace_id::text,d.properties FROM databases d WHERE d.id=$1 AND madi_space_allowed($2,d.space_id,$3) FOR SHARE OF d`, id, p.ID, write).Scan(&value.WorkspaceID, &raw)
	if e != nil {
		return value, denied
	}
	if e = documentAccessActorTx(r, tx, value.WorkspaceID, write); e != nil {
		return value, e
	}
	if e = json.Unmarshal(raw, &value.Properties); e != nil {
		return value, e
	}
	value.Schema = digest(string(raw))
	return value, nil
}
func (s *Server) databaseEditingMetadata(w http.ResponseWriter, r *http.Request) {
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	value, e := s.databaseEditingScopeTx(r, tx, r.PathValue("id"), false)
	if e != nil {
		apiError(w, 403, e.Error())
		return
	}
	jsonResponse(w, 200, map[string]any{"schema_fingerprint": value.Schema, "properties": value.Properties, "max_rows": 100, "max_cells": 500, "atomic": true})
}

type databaseEditCell struct {
	RowID      string `json:"row_id"`
	PropertyID string `json:"property_id"`
	Version    int    `json:"expected_version"`
	Value      any    `json:"value"`
}

func (c *databaseEditCell) UnmarshalJSON(data []byte) error {
	type alias databaseEditCell
	var keys map[string]json.RawMessage
	if e := json.Unmarshal(data, &keys); e != nil {
		return e
	}
	if _, ok := keys["value"]; !ok {
		return errors.New("셀 value 필드는 필수입니다")
	}
	return json.Unmarshal(data, (*alias)(c))
}

func (s *Server) editDatabaseCells(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in struct {
		Schema  string             `json:"schema_fingerprint"`
		Cells   []databaseEditCell `json:"cells"`
		Commit  bool               `json:"commit"`
		Consent bool               `json:"consent"`
	}
	if decode(r, &in) != nil || len(in.Cells) < 1 || len(in.Cells) > 500 || len(jsonValue(in)) > 4<<20 || len(in.Schema) != 64 || in.Commit && !in.Consent {
		apiError(w, 400, "속성 기준과 최대500셀·4MiB의 변경을 확인하고 명시적으로 동의하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	scope, e := s.databaseEditingScopeTx(r, tx, id, true)
	if e != nil {
		apiError(w, 403, e.Error())
		return
	}
	if scope.Schema != in.Schema {
		apiError(w, 409, "속성 구성이 변경되었습니다. 현재 표를 다시 읽고 붙여넣으세요")
		return
	}
	values := map[string]map[string]any{}
	versions := map[string]int{}
	properties := map[string]map[string]any{}
	for _, p := range scope.Properties {
		properties[str(p, "id")] = p
	}
	issues := []map[string]any{}
	for _, cell := range in.Cells {
		if !validID(cell.RowID) || cell.Version < 1 || cell.Version > 2147483647 || cell.PropertyID == "" {
			apiError(w, 400, "행 ID·기준 버전·속성 ID를 확인하세요")
			return
		}
		if values[cell.RowID] == nil {
			values[cell.RowID] = map[string]any{}
			versions[cell.RowID] = cell.Version
		}
		if versions[cell.RowID] != cell.Version {
			apiError(w, 400, "같은 행은 동일한 기준 버전을 사용해야 합니다")
			return
		}
		if _, exists := values[cell.RowID][cell.PropertyID]; exists {
			apiError(w, 400, "같은 셀을 중복해서 변경할 수 없습니다")
			return
		}
		values[cell.RowID][cell.PropertyID] = cell.Value
		property, exists := properties[cell.PropertyID]
		if !exists {
			issues = append(issues, map[string]any{"row_id": cell.RowID, "property_id": cell.PropertyID, "message": "삭제되었거나 지원하지 않는 속성입니다"})
			continue
		}
		if e = validateValuesForProperties([]map[string]any{property}, map[string]any{cell.PropertyID: cell.Value}); e != nil {
			issues = append(issues, map[string]any{"row_id": cell.RowID, "property_id": cell.PropertyID, "message": e.Error()})
		}
	}
	if len(values) > 100 {
		apiError(w, 400, "한 번에100행까지 변경할 수 있습니다")
		return
	}
	ids := make([]string, 0, len(values))
	for rowID := range values {
		ids = append(ids, rowID)
	}
	sort.Strings(ids)
	before := map[string]map[string]any{}
	for _, rowID := range ids {
		var raw []byte
		var version int
		e = tx.QueryRow(r.Context(), "SELECT values,version FROM database_rows WHERE database_id=$1 AND id=$2 FOR UPDATE", id, rowID).Scan(&raw, &version)
		if e != nil {
			apiError(w, 409, "변경할 행이 삭제되었거나 현재 표에 없습니다. 최신 표를 다시 확인하세요")
			return
		}
		if version != versions[rowID] {
			apiError(w, 409, "다른 편집자가 행을 변경했습니다. 현재 값과 다시 비교하세요")
			return
		}
		var original map[string]any
		if e = json.Unmarshal(raw, &original); e != nil {
			respond(w, nil, e)
			return
		}
		before[rowID] = original
		if e = s.validateValuesTx(r, tx, id, values[rowID]); e != nil {
			issues = append(issues, map[string]any{"row_id": rowID, "property_id": "", "message": e.Error()})
		}
	}
	changes := []map[string]any{}
	for _, cell := range in.Cells {
		changes = append(changes, map[string]any{"row_id": cell.RowID, "property_id": cell.PropertyID, "expected_version": cell.Version, "before": before[cell.RowID][cell.PropertyID], "after": cell.Value})
	}
	if len(jsonValue(changes)) > 8<<20 {
		apiError(w, 413, "비교할 셀 원문이8MiB를 넘습니다. 더 작은 범위로 나누세요")
		return
	}
	if !in.Commit || len(issues) > 0 {
		jsonResponse(w, 200, map[string]any{"valid": len(issues) == 0, "committed": false, "changes": changes, "errors": issues, "schema_fingerprint": scope.Schema})
		return
	}
	updated := []map[string]any{}
	for _, rowID := range ids {
		var raw []byte
		e = tx.QueryRow(r.Context(), "UPDATE database_rows SET values=values||$1::jsonb,updated_at=clock_timestamp(),updated_by=$2 WHERE id=$3 AND database_id=$4 AND EXISTS(SELECT 1 FROM databases d WHERE d.id=$4 AND madi_space_allowed($2,d.space_id,true)) RETURNING to_jsonb(database_rows)", jsonValue(values[rowID]), current(r).ID, rowID, id).Scan(&raw)
		if e == nil {
			e = s.enqueueDatabaseEvent(r, tx, "database.row.updated", id, rowID, map[string]any{"version": versions[rowID]})
		}
		if e != nil {
			respond(w, nil, e)
			return
		}
		var row map[string]any
		if e = json.Unmarshal(raw, &row); e != nil {
			respond(w, nil, e)
			return
		}
		updated = append(updated, row)
	}
	if e = tx.Commit(r.Context()); e != nil {
		respond(w, nil, e)
		return
	}
	for _, rowID := range ids {
		s.audit(r, "DATABASE_ROW_UPDATE", rowID, map[string]any{"operation": "reviewed_cells", "cells": len(values[rowID])})
	}
	jsonResponse(w, 200, map[string]any{"valid": true, "committed": true, "rows": updated, "errors": []any{}})
}
