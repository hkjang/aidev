package httpapi

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/hkjang/invenqor/server/internal/diagnostics"
)

func (s *Server) recordDiagnostic(
	request *http.Request,
	event diagnostics.Event,
) {
	if event.RequestID == "" {
		event.RequestID = middleware.GetReqID(request.Context())
	}
	if err := s.diagnosticStore.Record(request.Context(), event); err != nil {
		s.logger.Error(
			"diagnostic_event_store_failed",
			"request_id", event.RequestID,
			"event_code", event.EventCode,
			"error", err,
		)
	}
}

func (s *Server) listDiagnosticLogs(
	response http.ResponseWriter,
	request *http.Request,
) {
	filter := diagnostics.Filter{
		Level: strings.ToLower(strings.TrimSpace(
			request.URL.Query().Get("level"),
		)),
		Component: strings.TrimSpace(
			request.URL.Query().Get("component"),
		),
		InstanceID: strings.TrimSpace(
			request.URL.Query().Get("instance_id"),
		),
		Query: request.URL.Query().Get("q"),
		Limit: queryInt(request, "limit", 200, 1, 500),
	}
	items, total, facets, err := s.diagnosticStore.List(
		request.Context(),
		filter,
	)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{
		"items": items,
		"total": total,
		"limit": filter.Limit,
		// Instances stays alongside the facets object for the console still
		// reading the older shape.
		"instances":   facets.Instances,
		"facets":      facets,
		"retention":   map[string]any{"days": 30, "maximum_events": 10000},
		"instance_id": s.diagnosticStore.InstanceID(),
	})
}
