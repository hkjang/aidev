package httpapi

import (
	"net/http"

	"github.com/hkjang/invenqor/server/internal/querydsl"
	"github.com/hkjang/invenqor/server/internal/storage"
)

type queryInput struct {
	Query  string `json:"query"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

// queryGrammar publishes the field and operator list so the console can show
// what is writable instead of leaving an operator to guess and read rejections.
func (s *Server) queryGrammar(response http.ResponseWriter, request *http.Request) {
	writeJSON(response, http.StatusOK, querydsl.Describe())
}

func (s *Server) validateQuery(response http.ResponseWriter, request *http.Request) {
	var input queryInput
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(response, request, 400, "INVALID_REQUEST", "The request body is invalid.")
		return
	}
	query, err := querydsl.Parse(input.Query)
	if err != nil {
		writeJSON(response, 200, map[string]any{
			"valid": false, "error": err.Error(),
		})
		return
	}
	// Compiling is part of the answer. Parsing alone accepts a clause whose
	// value the compiler still refuses - an unreadable time, a confidence that
	// is not a number - and calling that valid sends the operator to execute
	// with an expression that is about to be rejected there.
	if _, _, err := query.SQL(
		s.database.Mode() != storage.ModeSQLiteFallback,
	); err != nil {
		writeJSON(response, 200, map[string]any{
			"valid": false, "error": err.Error(),
		})
		return
	}
	writeJSON(response, 200, map[string]any{"valid": true, "ast": query})
}

func (s *Server) executeQuery(response http.ResponseWriter, request *http.Request) {
	var input queryInput
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(response, request, 400, "INVALID_REQUEST", "The request body is invalid.")
		return
	}
	query, err := querydsl.Parse(input.Query)
	if err != nil {
		writeAPIError(response, request, 400, "INVALID_QUERY", err.Error())
		return
	}
	where, args, err := query.SQL(
		s.database.Mode() != storage.ModeSQLiteFallback,
	)
	if err != nil {
		writeAPIError(response, request, 400, "INVALID_QUERY", err.Error())
		return
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	offset := input.Offset
	if offset < 0 {
		offset = 0
	}
	if offset > 1_000_000 {
		offset = 1_000_000
	}
	// How many assets the expression actually matches, asked before the page
	// arguments join the list. "last_seen_at < \"now - 720h\"" over 1,200 stale
	// hosts used to answer with 100 items and count 100, and neither the
	// response nor the audit entry said the other 1,100 existed.
	var total int64
	if err := s.database.DB().QueryRowContext(
		request.Context(),
		`SELECT COUNT(*) FROM assets WHERE `+where,
		args...,
	).Scan(&total); err != nil {
		s.internalError(response, request, err)
		return
	}
	// One row past the limit, so a result that ends exactly on it is not
	// reported as cut short. See the truncation note below.
	args = append(args, limit+1, offset)
	rows, err := s.database.DB().QueryContext(
		request.Context(),
		`SELECT `+assetColumns+` FROM assets WHERE `+where+
			// id breaks ties in last_seen_at. Assets ingested in one batch share
			// a timestamp to the second, and without a total order a paged walk
			// would return some of them twice and skip others.
			` ORDER BY last_seen_at DESC, id LIMIT $`+itoa(len(args)-1)+
			` OFFSET $`+itoa(len(args)),
		args...,
	)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	defer rows.Close()
	items := make([]assetView, 0)
	for rows.Next() {
		asset, err := scanAsset(rows)
		if err != nil {
			s.internalError(response, request, err)
			return
		}
		items = append(items, asset)
	}
	if err := rows.Err(); err != nil {
		s.internalError(response, request, err)
		return
	}
	// truncated keeps its meaning - rows the caller asked for exist beyond the
	// ones handed over - and offset is now the way to go and get them, so a
	// script reading this endpoint with an API key is no longer stuck narrowing
	// the expression until it fits under the cap.
	truncated := len(items) > limit
	if truncated {
		items = items[:limit]
	}
	s.recordAdminAudit(
		request, "query.execute", "query", "", nil,
		map[string]any{
			"dsl": input.Query, "result_count": len(items),
			"truncated": truncated, "offset": offset, "total": total,
		}, "",
	)
	writeJSON(response, 200, map[string]any{
		"items": items, "count": len(items), "ast": query,
		"limit": limit, "offset": offset, "total": total,
		"has_more": truncated, "next_offset": offset + len(items),
		"truncated": truncated,
	})
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	buffer := [20]byte{}
	index := len(buffer)
	for value > 0 {
		index--
		buffer[index] = byte('0' + value%10)
		value /= 10
	}
	return string(buffer[index:])
}
