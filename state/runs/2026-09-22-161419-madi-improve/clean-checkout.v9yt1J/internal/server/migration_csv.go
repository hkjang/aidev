package server

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (s *Server) importCSVTx(ctx context.Context, tx pgx.Tx, p *Principal, wid, space, name string, data migrationCSV) (string, error) {
	if !hasIntegrationScope(p, "database:write") || !s.canSpace(ctx, p, wid, space, true) {
		return "", errors.New("데이터베이스 가져오기 권한이 없습니다")
	}
	// CSV text follows the same deterministic PII policy as document metadata.
	meta := map[string]any{"name": name, "header": data.Header}
	clean, e := s.ProtectDocumentMetadataTx(ctx, tx, p, "", wid, meta)
	if e != nil {
		return "", e
	}
	var normalized struct {
		Name   string     `json:"name"`
		Header []string   `json:"header"`
		Rows   [][]string `json:"rows"`
	}
	if e = json.Unmarshal(jsonValue(clean.Value), &normalized); e != nil {
		return "", e
	}
	if strings.TrimSpace(normalized.Name) == "" || len(normalized.Name) > 200 {
		return "", errors.New("CSV 데이터베이스 이름은 비어 있지 않은 200바이트 이하의 이름이어야 합니다")
	}
	seen := map[string]bool{}
	for _, header := range normalized.Header {
		if strings.TrimSpace(header) == "" || len(header) > 100 || seen[header] {
			return "", errors.New("정보보호 적용 후 CSV 열 이름이 비어 있거나 중복되거나 100바이트를 초과합니다")
		}
		seen[header] = true
	}
	for _, row := range data.Rows {
		clean, e := s.ProtectDocumentMetadataTx(ctx, tx, p, "", wid, row)
		if e != nil {
			return "", e
		}
		var values []string
		if e = json.Unmarshal(jsonValue(clean.Value), &values); e != nil {
			return "", e
		}
		if len(values) != len(normalized.Header) {
			return "", errors.New("CSV 열 수가 일치하지 않습니다")
		}
		for _, value := range values {
			if len(value) > 64<<10 {
				return "", errors.New("정보보호 적용 후 CSV 셀 값이 64KB를 초과합니다")
			}
		}
		normalized.Rows = append(normalized.Rows, values)
	}
	id := newID()
	props := []map[string]any{}
	for _, column := range normalized.Header {
		props = append(props, map[string]any{"id": newID(), "name": column, "type": "text", "options": []string{}})
	}
	_, e = tx.Exec(ctx, `INSERT INTO databases(id,workspace_id,name,properties,space_id) VALUES($1,$2,$3,$4,NULLIF($5,'')::uuid)`, id, wid, normalized.Name, jsonValue(props), space)
	if e != nil {
		return "", e
	}
	for _, row := range normalized.Rows {
		values := map[string]any{}
		for i, text := range row {
			values[str(props[i], "id")] = text
		}
		if _, e = tx.Exec(ctx, `INSERT INTO database_rows(id,database_id,values,created_by,updated_by) VALUES($1,$2,$3,$4,$4)`, newID(), id, jsonValue(values), p.ID); e != nil {
			return "", e
		}
	}
	r := automationRequest(ctx, p, "POST", nil)
	return id, s.enqueueDatabaseEvent(r, tx, "database.created", id, "", nil)
}
