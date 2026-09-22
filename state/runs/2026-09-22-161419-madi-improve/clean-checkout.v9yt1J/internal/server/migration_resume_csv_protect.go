package server

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (s *Server) protectMigrationCSVTx(ctx context.Context, tx pgx.Tx, p *Principal, wid string, prepared *migrationPrepared) (bool, error) {
	if prepared.CSV == nil {
		return false, errors.New("CSV 정본이 없습니다")
	}
	changed := false
	clean, e := s.ProtectDocumentMetadataTx(ctx, tx, p, "", wid, map[string]any{"title": prepared.Title, "header": prepared.CSV.Header})
	if e != nil {
		return false, e
	}
	changed = clean.Changed
	var header struct {
		Title  string   `json:"title"`
		Header []string `json:"header"`
	}
	if e = json.Unmarshal(jsonValue(clean.Value), &header); e != nil {
		return false, e
	}
	if len(header.Header) != len(prepared.CSV.Header) {
		return false, errors.New("보호 정책 적용 후 CSV 헤더가 일치하지 않습니다")
	}
	prepared.Title = header.Title
	prepared.CSV.Header = header.Header
	for start := 0; start < len(prepared.CSV.Rows); {
		end := start
		size := 0
		for end < len(prepared.CSV.Rows) && end-start < 256 {
			n := len(jsonValue(prepared.CSV.Rows[end]))
			if size+n > 2<<20 && end > start {
				break
			}
			size += n
			end++
		}
		clean, e = s.ProtectDocumentMetadataTx(ctx, tx, p, "", wid, prepared.CSV.Rows[start:end])
		if e != nil {
			return false, e
		}
		changed = changed || clean.Changed
		var rows [][]string
		if e = json.Unmarshal(jsonValue(clean.Value), &rows); e != nil {
			return false, e
		}
		if len(rows) != end-start {
			return false, errors.New("보호 정책 적용 후 CSV 행 수가 변경되었습니다")
		}
		copy(prepared.CSV.Rows[start:end], rows)
		start = end
	}
	seen := map[string]bool{}
	for _, name := range prepared.CSV.Header {
		if name == "" || len(name) > 100 || seen[name] {
			return false, errors.New("보호 정책 적용 후 CSV 열 이름이 비어 있거나 중복되었습니다")
		}
		seen[name] = true
	}
	return changed, nil
}
