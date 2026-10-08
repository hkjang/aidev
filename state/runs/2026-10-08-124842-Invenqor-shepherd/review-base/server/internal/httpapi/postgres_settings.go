package httpapi

import (
	"net/http"
	"strings"

	"github.com/hkjang/invenqor/server/internal/storage"
	"github.com/jackc/pgx/v5"
)

type postgresSettingsInput struct {
	DSN    string `json:"dsn"`
	Reason string `json:"reason"`
}

func (s *Server) getPostgresSettings(
	response http.ResponseWriter,
	request *http.Request,
) {
	if s.bootstrapStore == nil {
		writeAPIError(
			response,
			request,
			http.StatusServiceUnavailable,
			"BOOTSTRAP_STORE_UNAVAILABLE",
			"Encrypted startup settings are unavailable.",
		)
		return
	}
	values, err := s.bootstrapStore.Load()
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	savedDSN := strings.TrimSpace(values.PostgresDSN)
	effectiveDSN := strings.TrimSpace(s.currentPostgresDSN)
	payload := map[string]any{
		"database_mode":        s.database.Mode(),
		"schema":               s.databaseSchema,
		"configured":           effectiveDSN != "" || savedDSN != "",
		"saved_configured":     savedDSN != "",
		"environment_override": s.postgresEnvironmentOverride,
		"restart_required":     savedDSN != effectiveDSN,
		"effective":            postgresDSNSummary(effectiveDSN),
		"saved":                postgresDSNSummary(savedDSN),
	}
	if failure := s.database.PostgresFailure(); failure != nil {
		payload["startup_failure"] = failure
	}
	writeJSON(response, http.StatusOK, payload)
}

func (s *Server) testPostgresSettings(
	response http.ResponseWriter,
	request *http.Request,
) {
	var input postgresSettingsInput
	if err := decodeJSON(request, &input); err != nil ||
		strings.TrimSpace(input.DSN) == "" {
		writeAPIError(
			response,
			request,
			http.StatusBadRequest,
			"INVALID_POSTGRES_DSN",
			"A PostgreSQL DSN is required.",
		)
		return
	}
	if connectionFailure := storage.CheckPostgres(
		request.Context(),
		storage.Options{
			PostgresDSN: strings.TrimSpace(input.DSN),
			Schema:      s.databaseSchema,
			Timeout:     s.databaseTimeout,
		},
	); connectionFailure != nil {
		writeJSON(response, http.StatusBadGateway, map[string]any{
			"connected": false,
			"failure":   connectionFailure,
		})
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{
		"connected": true,
		"target":    postgresDSNSummary(input.DSN),
	})
}

func (s *Server) updatePostgresSettings(
	response http.ResponseWriter,
	request *http.Request,
) {
	if s.bootstrapStore == nil {
		writeAPIError(
			response,
			request,
			http.StatusServiceUnavailable,
			"BOOTSTRAP_STORE_UNAVAILABLE",
			"Encrypted startup settings are unavailable.",
		)
		return
	}
	var input postgresSettingsInput
	if err := decodeJSON(request, &input); err != nil ||
		strings.TrimSpace(input.DSN) == "" {
		writeAPIError(
			response,
			request,
			http.StatusBadRequest,
			"INVALID_POSTGRES_DSN",
			"A PostgreSQL DSN is required.",
		)
		return
	}
	input.DSN = strings.TrimSpace(input.DSN)
	if connectionFailure := storage.CheckPostgres(
		request.Context(),
		storage.Options{
			PostgresDSN: input.DSN,
			Schema:      s.databaseSchema,
			Timeout:     s.databaseTimeout,
		},
	); connectionFailure != nil {
		writeJSON(response, http.StatusBadGateway, map[string]any{
			"saved":   false,
			"failure": connectionFailure,
		})
		return
	}
	values, err := s.bootstrapStore.Load()
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	before := postgresDSNSummary(values.PostgresDSN)
	values.PostgresDSN = input.DSN
	if err := s.bootstrapStore.Save(values); err != nil {
		s.internalError(response, request, err)
		return
	}
	after := postgresDSNSummary(input.DSN)
	s.recordAdminAudit(
		request,
		"setting.postgresql.update",
		"setting",
		"database.postgresql",
		before,
		after,
		input.Reason,
	)
	writeJSON(response, http.StatusOK, map[string]any{
		"saved":                true,
		"restart_required":     input.DSN != strings.TrimSpace(s.currentPostgresDSN),
		"environment_override": s.postgresEnvironmentOverride,
		"target":               after,
	})
}

func postgresDSNSummary(dsn string) any {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return nil
	}
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		return map[string]any{"valid": false}
	}
	return map[string]any{
		"valid":    true,
		"host":     config.Host,
		"port":     config.Port,
		"database": config.Database,
		"user":     config.User,
		"tls":      config.TLSConfig != nil,
	}
}
