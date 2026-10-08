package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hkjang/invenqor/server/internal/agents"
	"github.com/hkjang/invenqor/server/internal/apikeys"
	"github.com/hkjang/invenqor/server/internal/auth"
	"github.com/hkjang/invenqor/server/internal/bootstrap"
	"github.com/hkjang/invenqor/server/internal/diagnostics"
	"github.com/hkjang/invenqor/server/internal/ingest"
	"github.com/hkjang/invenqor/server/internal/mail"
	"github.com/hkjang/invenqor/server/internal/spool"
	"github.com/hkjang/invenqor/server/internal/storage"
	"github.com/hkjang/invenqor/server/internal/tracking"
	"github.com/hkjang/invenqor/server/internal/updates"
	"github.com/hkjang/invenqor/server/internal/version"
	"github.com/hkjang/invenqor/server/internal/webui"
)

type Server struct {
	router                       chi.Router
	database                     *storage.Runtime
	authService                  *auth.Service
	oidcService                  *auth.OIDCService
	totpService                  *auth.TOTPService
	bootstrapManager             *auth.BootstrapManager
	agentService                 *agents.Service
	ingestService                *ingest.Service
	agentRateLimit               *agentRateLimiter
	agentEnrollmentRateLimit     *agentRateLimiter
	agentEnrollmentTokenHash     [sha256.Size]byte
	agentEnrollmentEnabled       bool
	agentEnrollmentTokenRequired bool
	spool                        *spool.Manager
	bootstrapStore               *bootstrap.Store
	updateStore                  *updates.Store
	apiKeyService                *apikeys.Service
	apiRateLimit                 *agentRateLimiter
	logger                       *slog.Logger
	currentPostgresDSN           string
	postgresEnvironmentOverride  bool
	databaseSchema               string
	databaseTimeout              time.Duration
	diagnosticStore              *diagnostics.Store
	listenAddress                string
	// violations remembers what the browser's content security policy
	// refused while visitor tracking is on. It is per process by design; see
	// tracking.Recorder.
	violations       *tracking.Recorder
	momentoTransport http.RoundTripper
	console          http.Handler
	// mail sends event notifications through the company relay. It reads
	// its configuration from the settings table on every use, so it is
	// shared by every Pod and off until an administrator turns it on.
	mail *mail.Service
}

type Options struct {
	Database                    *storage.Runtime
	AuthService                 *auth.Service
	OIDCService                 *auth.OIDCService
	TOTPService                 *auth.TOTPService
	BootstrapManager            *auth.BootstrapManager
	AgentService                *agents.Service
	IngestService               *ingest.Service
	Spool                       *spool.Manager
	BootstrapStore              *bootstrap.Store
	UpdateStore                 *updates.Store
	APIKeyService               *apikeys.Service
	Logger                      *slog.Logger
	CurrentPostgresDSN          string
	PostgresEnvironmentOverride bool
	DatabaseSchema              string
	DatabaseTimeout             time.Duration
	AgentAutoEnrollment         bool
	AgentEnrollmentToken        string
	// ListenAddress is reported by /api/v1/system/info. The console used to
	// print a hard-coded 7070 on its runtime panel, which was simply false for
	// any installation that had changed the address.
	ListenAddress string
}

func New(options Options) *Server {
	if options.AgentService == nil {
		options.AgentService = agents.NewService(options.Database.DB())
	}
	if options.IngestService == nil {
		options.IngestService = ingest.NewService(options.Database.DB())
	}
	if options.APIKeyService == nil {
		options.APIKeyService = apikeys.NewService(options.Database.DB())
	}
	server := &Server{
		router:                      chi.NewRouter(),
		database:                    options.Database,
		authService:                 options.AuthService,
		oidcService:                 options.OIDCService,
		totpService:                 options.TOTPService,
		bootstrapManager:            options.BootstrapManager,
		agentService:                options.AgentService,
		ingestService:               options.IngestService,
		agentRateLimit:              newAgentRateLimiter(120, time.Minute),
		agentEnrollmentRateLimit:    newAgentRateLimiter(30, time.Minute),
		agentEnrollmentEnabled:      options.AgentAutoEnrollment,
		spool:                       options.Spool,
		bootstrapStore:              options.BootstrapStore,
		updateStore:                 options.UpdateStore,
		apiKeyService:               options.APIKeyService,
		apiRateLimit:                newAgentRateLimiter(600, time.Minute),
		logger:                      options.Logger,
		currentPostgresDSN:          options.CurrentPostgresDSN,
		postgresEnvironmentOverride: options.PostgresEnvironmentOverride,
		databaseSchema:              options.DatabaseSchema,
		databaseTimeout:             options.DatabaseTimeout,
		diagnosticStore:             diagnostics.NewStore(options.Database.DB()),
		listenAddress:               options.ListenAddress,
		violations:                  tracking.NewRecorder(),
		momentoTransport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			ResponseHeaderTimeout: 10 * time.Second,
			IdleConnTimeout:       90 * time.Second,
			MaxIdleConns:          16,
		},
	}
	if options.AgentEnrollmentToken != "" {
		server.agentEnrollmentTokenHash = sha256.Sum256(
			[]byte(options.AgentEnrollmentToken),
		)
		server.agentEnrollmentTokenRequired = true
	}
	server.mail = mail.NewService(
		options.Database.DB(), server.loadMailConfig,
		userDirectory{db: options.Database.DB()}, options.Logger,
	)
	if server.authService != nil {
		server.authService.SetAccountLockedHook(server.onAccountLocked)
	}
	if server.databaseSchema == "" {
		server.databaseSchema = "public"
	}
	if server.databaseTimeout <= 0 {
		server.databaseTimeout = 5 * time.Second
	}
	server.routes()
	return server
}

func (s *Server) AgentEnrollmentMode(ctx context.Context) (string, error) {
	policy, _, err := s.loadAgentEnrollmentPolicy(ctx)
	if err != nil {
		return "", err
	}
	return policy.mode(), nil
}

func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) routes() {
	// The request ID is generated here and nowhere else. chi's RequestID takes
	// an inbound X-Request-Id verbatim when one is present, and this value is
	// written into every audit record and diagnostic event and is what the
	// console filters by - so accepting it would let the subject of an audit
	// record choose its own correlation identifier, and collide it with someone
	// else's on purpose. Nothing needs the inbound form: the Agent and the
	// console only ever read this header off a response.
	s.router.Use(discardInboundRequestID)
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.Recoverer)
	s.router.Use(s.securityHeaders)
	s.router.Use(s.requestLog)

	console := webui.Decorated(decorateConsole)
	s.console = console
	s.router.Get("/", console.ServeHTTP)
	s.router.Get("/health/live", s.live)
	s.router.Get("/health/ready", s.ready)
	s.router.Get("/api/v1/system/info", s.publicSystemInfo)
	s.router.Get("/api/v1/bootstrap/status", s.bootstrapStatus)
	s.router.Post("/api/v1/bootstrap/admin", s.createInitialAdmin)
	s.router.Post("/api/v1/auth/local/login", s.localLogin)
	s.router.Get("/api/v1/auth/methods", s.authMethods)
	s.router.Get("/api/v1/auth/keycloak/start", s.keycloakStart)
	s.router.Get("/api/v1/auth/keycloak/callback", s.keycloakCallback)
	s.router.Get("/v1/agent/preflight", s.agentPreflight)
	s.router.Post("/v1/agent/enroll", s.autoEnrollAgent)
	s.router.Post("/v1/agent/events", s.receiveAgentEvent)
	s.router.Get("/v1/agent/updates", s.agentUpdateManifest)
	s.router.Get("/v1/agent/updates/{artifact}/artifact", s.agentUpdateArtifact)
	s.router.Post(cspReportPath, s.receiveCSPReport)
	s.router.Get(tracking.MomentoProxyPath+"/*", s.momentoProxy)
	s.router.Post(tracking.MomentoProxyPath+"/*", s.momentoProxy)
	s.router.Group(func(external chi.Router) {
		external.Use(s.authenticateAPIKey)
		external.With(s.requirePermission("assets.read")).Get(
			"/api/v1/external/assets", s.listAssets,
		)
		external.With(s.requirePermission("assets.read")).Get(
			"/api/v1/external/assets/{assetID}", s.getAsset,
		)
		external.With(s.requirePermission("assets.write")).Post(
			"/api/v1/external/assets", s.createAsset,
		)
		external.With(s.requirePermission("assets.write")).Patch(
			"/api/v1/external/assets/{assetID}", s.updateAsset,
		)
		external.With(s.requirePermission("assets.delete")).Delete(
			"/api/v1/external/assets/{assetID}", s.deleteAsset,
		)
		external.With(s.requirePermission("assets.delete")).Post(
			"/api/v1/external/assets/{assetID}/restore", s.restoreAsset,
		)
		external.With(s.requirePermission("relations.read")).Get(
			"/api/v1/external/assets/{assetID}/relations", s.assetRelations,
		)
		external.With(s.requirePermission("relations.write")).Post(
			"/api/v1/external/assets/{assetID}/relations", s.createAssetRelation,
		)
		external.With(s.requirePermission("relations.write")).Delete(
			"/api/v1/external/assets/{assetID}/relations/{relationID}", s.deleteAssetRelation,
		)
		external.With(s.requirePermission("queries.execute")).Post(
			"/api/v1/external/query/validate", s.validateQuery,
		)
		external.With(s.requirePermission("queries.execute")).Post(
			"/api/v1/external/query/execute", s.executeQuery,
		)
		external.With(s.requirePermission("mcp.access")).Get("/mcp", s.mcpGet)
		external.With(s.requirePermission("mcp.access")).Post("/mcp", s.mcpPost)
	})
	s.router.Group(func(protected chi.Router) {
		protected.Use(s.authenticate)
		protected.Get("/api/v1/auth/me", s.me)
		protected.With(s.requireCSRF).Post("/api/v1/auth/logout", s.logout)
		protected.With(s.requireCSRF).Post(
			"/api/v1/auth/password/change",
			s.changePassword,
		)
		protected.With(s.requireCSRF).Post(
			"/api/v1/auth/totp/setup",
			s.setupTOTP,
		)
		protected.With(s.requireCSRF).Post(
			"/api/v1/auth/totp/enable",
			s.enableTOTP,
		)
		protected.With(s.requireCSRF).Post(
			"/api/v1/auth/totp/recovery-codes",
			s.regenerateRecoveryCodes,
		)
		protected.With(s.requireCSRF).Delete(
			"/api/v1/auth/totp",
			s.disableTOTP,
		)
		protected.With(s.requirePermission("settings.read")).Get(
			"/health/database",
			s.databaseHealth,
		)
		protected.With(s.requirePermission("settings.read")).Get(
			"/api/v1/admin/system/info",
			s.adminSystemInfo,
		)
		protected.With(s.requirePermission("settings.read")).Get(
			"/api/v1/admin/settings/keycloak",
			s.getKeycloakSettings,
		)
		protected.With(s.requireCSRF, s.requirePermission("settings.write")).Patch(
			"/api/v1/admin/settings/keycloak",
			s.updateKeycloakSettings,
		)
		protected.With(s.requireCSRF, s.requirePermission("settings.write")).Post(
			"/api/v1/admin/settings/keycloak/test",
			s.testKeycloakSettings,
		)
		protected.With(s.requireCSRF, s.requirePermission("settings.write")).Post(
			"/api/v1/admin/settings/keycloak/auto-configure",
			s.autoConfigureKeycloak,
		)
		protected.With(s.requirePermission("settings.read")).Get(
			"/api/v1/admin/settings/agent-enrollment",
			s.getAgentEnrollmentSettings,
		)
		protected.With(s.requireCSRF, s.requirePermission("settings.write")).Patch(
			"/api/v1/admin/settings/agent-enrollment",
			s.updateAgentEnrollmentSettings,
		)
		protected.With(s.requireCSRF, s.requirePermission("settings.write")).Post(
			"/api/v1/admin/settings/agent-enrollment/token",
			s.issueAgentEnrollmentToken,
		)
		protected.With(s.requireCSRF, s.requirePermission("settings.write")).Delete(
			"/api/v1/admin/settings/agent-enrollment/token",
			s.deleteAgentEnrollmentToken,
		)
		protected.With(s.requirePermission("settings.read")).Get(
			"/api/v1/admin/settings/tracking",
			s.getTrackingSettings,
		)
		protected.With(s.requireCSRF, s.requirePermission("settings.write")).Patch(
			"/api/v1/admin/settings/tracking",
			s.updateTrackingSettings,
		)
		protected.With(s.requirePermission("settings.read")).Get(
			"/api/v1/admin/settings/mail", s.getMailSettings,
		)
		protected.With(s.requireCSRF, s.requirePermission("settings.write")).Patch(
			"/api/v1/admin/settings/mail", s.updateMailSettings,
		)
		protected.With(s.requireCSRF, s.requirePermission("settings.write")).Post(
			"/api/v1/admin/settings/mail/test", s.sendTestMail,
		)
		protected.With(s.requirePermission("settings.read")).Get(
			"/api/v1/admin/mail/deliveries", s.listMailDeliveries,
		)
		protected.With(s.requirePermission("settings.read")).Get(
			"/api/v1/admin/settings/tracking/violations",
			s.listTrackingViolations,
		)
		protected.With(s.requireCSRF, s.requirePermission("settings.write")).Delete(
			"/api/v1/admin/settings/tracking/violations",
			s.clearTrackingViolations,
		)
		protected.With(s.requireCSRF, s.requirePermission("settings.write")).Post(
			"/api/v1/admin/settings/tracking/allowed-hosts",
			s.allowTrackingHost,
		)
		protected.With(s.requirePermission("agents.read")).Get(
			"/api/v1/admin/agents",
			s.listAgents,
		)
		protected.With(s.requireCSRF, s.requirePermission("agents.manage")).Post(
			"/api/v1/admin/agents",
			s.provisionAgent,
		)
		protected.With(s.requireCSRF, s.requirePermission("agents.manage")).Post(
			"/api/v1/admin/agents/{agentID}/tokens/rotate",
			s.rotateAgentToken,
		)
		protected.With(s.requireCSRF, s.requirePermission("agents.manage")).Post(
			"/api/v1/admin/agents/{agentID}/block",
			s.blockAgent,
		)
		protected.With(s.requireCSRF, s.requirePermission("agents.manage")).Post(
			"/api/v1/admin/agents/{agentID}/unblock",
			s.unblockAgent,
		)
		protected.With(s.requireCSRF, s.requirePermission("agents.manage")).Post(
			"/api/v1/admin/agents/{agentID}/certificates",
			s.registerAgentCertificate,
		)
		protected.With(s.requirePermission("assets.read")).Get(
			"/api/v1/assets", s.listAssets,
		)
		protected.With(s.requirePermission("assets.export")).Get(
			"/api/v1/assets.csv", s.exportAssets,
		)
		protected.With(s.requirePermission("assets.read")).Get(
			"/api/v1/assets/software-products", s.listSoftwareProducts,
		)
		protected.With(
			s.requirePermission("assets.read"),
			s.requirePermission("agents.read"),
		).Get(
			"/api/v1/dashboard/statistics", s.dashboardStatistics,
		)
		protected.With(
			s.requirePermission("assets.read"),
			s.requirePermission("relations.read"),
		).Get(
			"/api/v1/assets/visualization", s.assetVisualization,
		)
		protected.With(s.requirePermission("settings.read")).Get(
			"/api/v1/admin/settings/classification",
			s.listClassificationRules,
		)
		protected.With(s.requireCSRF, s.requirePermission("settings.write")).Patch(
			"/api/v1/admin/settings/classification/rules/{ruleID}",
			s.updateClassificationRule,
		)
		protected.With(s.requireCSRF, s.requirePermission("settings.write")).Post(
			"/api/v1/admin/settings/classification/reclassify",
			s.reclassifyAssets,
		)
		protected.With(s.requirePermission("relations.read")).Get(
			"/api/v1/assets/relations/proposed", s.listProposedRelations,
		)
		protected.With(s.requireCSRF, s.requirePermission("relations.write")).Post(
			"/api/v1/assets/relations/{relationID}/{decision}",
			s.reviewProposedRelation,
		)
		protected.With(s.requireCSRF, s.requirePermission("assets.write")).Post(
			"/api/v1/assets", s.createAsset,
		)
		protected.With(s.requirePermission("assets.read")).Get(
			"/api/v1/assets/{assetID}", s.getAsset,
		)
		protected.With(s.requireCSRF, s.requirePermission("assets.write")).Patch(
			"/api/v1/assets/{assetID}", s.updateAsset,
		)
		protected.With(s.requireCSRF, s.requirePermission("assets.delete")).Delete(
			"/api/v1/assets/{assetID}", s.deleteAsset,
		)
		protected.With(s.requireCSRF, s.requirePermission("assets.delete")).Post(
			"/api/v1/assets/{assetID}/restore", s.restoreAsset,
		)
		protected.With(s.requirePermission("assets.read")).Get(
			"/api/v1/assets/{assetID}/history", s.assetHistory,
		)
		protected.With(s.requirePermission("relations.read")).Get(
			"/api/v1/assets/{assetID}/relations", s.assetRelations,
		)
		protected.With(s.requireCSRF, s.requirePermission("relations.write")).Post(
			"/api/v1/assets/{assetID}/relations", s.createAssetRelation,
		)
		protected.With(s.requireCSRF, s.requirePermission("relations.write")).Delete(
			"/api/v1/assets/{assetID}/relations/{relationID}", s.deleteAssetRelation,
		)
		protected.With(s.requireCSRF, s.requirePermission("assets.merge")).Post(
			"/api/v1/assets/merge", s.mergeAssets,
		)
		protected.With(s.requireCSRF, s.requirePermission("assets.merge")).Post(
			"/api/v1/assets/{assetID}/split", s.splitAsset,
		)
		protected.With(s.requirePermission("queries.execute")).Post(
			"/api/v1/query/validate", s.validateQuery,
		)
		protected.With(s.requirePermission("queries.execute")).Get(
			"/api/v1/query/schema", s.queryGrammar,
		)
		protected.With(s.requirePermission("queries.execute")).Post(
			"/api/v1/query/execute", s.executeQuery,
		)
		protected.With(s.requirePermission("settings.read")).Get(
			"/api/v1/admin/settings", s.listSettings,
		)
		protected.With(s.requirePermission("settings.read")).Get(
			"/api/v1/admin/settings/postgresql", s.getPostgresSettings,
		)
		protected.With(s.requireCSRF, s.requirePermission("settings.write")).Post(
			"/api/v1/admin/settings/postgresql/test", s.testPostgresSettings,
		)
		protected.With(s.requireCSRF, s.requirePermission("settings.write")).Patch(
			"/api/v1/admin/settings/postgresql", s.updatePostgresSettings,
		)
		protected.With(s.requireCSRF, s.requirePermission("settings.write")).Patch(
			"/api/v1/admin/settings", s.updateSettings,
		)
		protected.With(s.requirePermission("settings.read")).Get(
			"/api/v1/admin/settings/history", s.settingHistory,
		)
		protected.With(s.requireCSRF, s.requirePermission("settings.write")).Post(
			"/api/v1/admin/settings/rollback", s.rollbackSetting,
		)
		protected.With(s.requirePermission("audit.read")).Get(
			"/api/v1/admin/audit", s.listAudit,
		)
		protected.With(s.requirePermission("audit.read")).Get(
			"/api/v1/admin/audit.csv", s.exportAudit,
		)
		protected.With(s.requirePermission("audit.read")).Get(
			"/api/v1/admin/diagnostics/logs", s.listDiagnosticLogs,
		)
		protected.With(s.requirePermission("agents.read")).Get(
			"/api/v1/admin/diagnostics/enrollment", s.enrollmentDiagnostics,
		)
		protected.With(s.requirePermission("users.read")).Get(
			"/api/v1/admin/users", s.listUsers,
		)
		protected.With(s.requirePermission("users.read")).Get(
			"/api/v1/admin/roles", s.listRoles,
		)
		protected.With(s.requireCSRF, s.requirePermission("users.manage")).Post(
			"/api/v1/admin/users", s.createUser,
		)
		protected.With(s.requireCSRF, s.requirePermission("users.manage")).Patch(
			"/api/v1/admin/users/{userID}", s.updateUser,
		)
		protected.With(s.requireCSRF, s.requirePermission("users.manage")).Post(
			"/api/v1/admin/users/{userID}/password", s.resetUserPassword,
		)
		protected.With(s.requireCSRF, s.requirePermission("users.manage")).Post(
			"/api/v1/admin/users/{userID}/unlock", s.unlockUser,
		)
		protected.With(s.requireCSRF, s.requirePermission("users.manage")).Delete(
			"/api/v1/admin/users/{userID}", s.deleteUser,
		)
		protected.With(s.requirePermission("agents.read")).Get(
			"/api/v1/admin/agent-updates", s.listAgentUpdates,
		)
		protected.With(s.requireCSRF, s.requirePermission("agents.manage")).Post(
			"/api/v1/admin/agent-updates", s.publishAgentUpdate,
		)
		protected.With(s.requireCSRF, s.requirePermission("agents.manage")).Patch(
			"/api/v1/admin/agent-updates/{release}", s.updateAgentUpdateRollout,
		)
		protected.With(s.requireCSRF, s.requirePermission("agents.manage")).Delete(
			"/api/v1/admin/agent-updates/{release}", s.retireAgentUpdate,
		)
		protected.With(s.requirePermission("api_keys.manage")).Get(
			"/api/v1/admin/api-key-scopes", s.apiKeyScopes,
		)
		protected.With(s.requirePermission("api_keys.manage")).Get(
			"/api/v1/admin/api-keys", s.listAPIKeys,
		)
		protected.With(s.requirePermission("api_keys.manage")).Get(
			"/api/v1/admin/api-keys/{keyID}", s.getAPIKey,
		)
		protected.With(s.requireCSRF, s.requirePermission("api_keys.manage")).Post(
			"/api/v1/admin/api-keys", s.createAPIKey,
		)
		protected.With(s.requireCSRF, s.requirePermission("api_keys.manage")).Patch(
			"/api/v1/admin/api-keys/{keyID}", s.updateAPIKey,
		)
		protected.With(s.requireCSRF, s.requirePermission("api_keys.manage")).Post(
			"/api/v1/admin/api-keys/{keyID}/scopes", s.addAPIKeyScopes,
		)
		protected.With(s.requireCSRF, s.requirePermission("api_keys.manage")).Delete(
			"/api/v1/admin/api-keys/{keyID}/scopes/{scope}", s.removeAPIKeyScope,
		)
		protected.With(s.requireCSRF, s.requirePermission("api_keys.manage")).Post(
			"/api/v1/admin/api-keys/{keyID}/rotate", s.rotateAPIKey,
		)
		protected.With(s.requireCSRF, s.requirePermission("api_keys.manage")).Delete(
			"/api/v1/admin/api-keys/{keyID}", s.revokeAPIKey,
		)
	})
	s.router.NotFound(s.notFound)
	s.router.MethodNotAllowed(s.methodNotAllowed)
}

// notFound keeps a misconfigured Agent visible. Without this, an Agent whose
// server.url carries a stray path only ever produces a stdout line on one Pod,
// so the console shows an absent Agent and no reason for it.
func (s *Server) notFound(
	response http.ResponseWriter,
	request *http.Request,
) {
	// A base URL carrying a stray path prefix is the most confusing
	// misconfiguration there is: the console SPA would answer HTTP 200 with
	// HTML and the Agent would report a JSON decode error. Match the Agent
	// endpoints anywhere in the path so the answer names the real problem.
	if strings.Contains(request.URL.Path, "/v1/agent/") {
		s.recordAgentEndpointMisuse(
			request,
			"AGENT_ENDPOINT_NOT_FOUND",
			"",
		)
		writeAPIError(
			response, request, http.StatusNotFound,
			"AGENT_ENDPOINT_NOT_FOUND",
			"This path is not an Agent endpoint. Configure server.url with "+
				"the scheme, host and port only.",
		)
		return
	}
	s.console.ServeHTTP(response, request)
}

func (s *Server) methodNotAllowed(
	response http.ResponseWriter,
	request *http.Request,
) {
	if strings.Contains(request.URL.Path, "/v1/agent/") {
		s.recordAgentEndpointMisuse(
			request,
			"AGENT_ENDPOINT_METHOD_NOT_ALLOWED",
			"",
		)
	}
	writeAPIError(
		response, request, http.StatusMethodNotAllowed,
		"METHOD_NOT_ALLOWED",
		"The requested method is not supported for this resource.",
	)
}

func (s *Server) recordAgentEndpointMisuse(
	request *http.Request,
	code string,
	message string,
) {
	// Same rule as recordEnrollment: "" means "whatever this code means", so
	// the sentence lives in enrollmentGuidance only.
	if message == "" {
		if summary := enrollmentSummary(code); summary != "" {
			message = summary
		} else {
			message = code
		}
	}
	s.recordDiagnostic(request, diagnostics.Event{
		Level:     "warning",
		Component: "agent_transport",
		EventCode: code,
		Message:   message,
		AgentID: strings.TrimSpace(
			request.Header.Get("X-Invenqor-Agent-Id"),
		),
		SourceIP: s.agentDiagnosticSourceIP(request),
		Details: map[string]any{
			"method":        request.Method,
			"path":          request.URL.Path,
			"agent_version": agentVersion(request.UserAgent()),
			"remediation":   enrollmentRemediation(code),
		},
	})
}

func (s *Server) root(response http.ResponseWriter, _ *http.Request) {
	writeJSON(response, http.StatusOK, map[string]any{
		"name":    "Invenqor Server",
		"version": version.Version,
		"status":  "running",
	})
}

func (s *Server) live(response http.ResponseWriter, _ *http.Request) {
	writeJSON(response, http.StatusOK, map[string]any{
		"status": "UP",
	})
}

func (s *Server) ready(response http.ResponseWriter, request *http.Request) {
	context, cancel := context.WithTimeout(request.Context(), time.Second)
	defer cancel()
	if err := s.database.Ping(context); err != nil {
		writeJSON(response, http.StatusServiceUnavailable, map[string]any{
			"status":        "NOT_READY",
			"database_mode": s.database.Mode(),
		})
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{
		"status":        "READY",
		"database_mode": s.database.Mode(),
	})
}

func (s *Server) databaseHealth(response http.ResponseWriter, request *http.Request) {
	context, cancel := context.WithTimeout(request.Context(), time.Second)
	defer cancel()
	status := http.StatusOK
	health := "UP"
	if err := s.database.Ping(context); err != nil {
		status = http.StatusServiceUnavailable
		health = "DOWN"
	}
	payload := map[string]any{
		"status":    health,
		"mode":      s.database.Mode(),
		"opened_at": s.database.OpenedAt(),
	}
	if failure := s.database.PostgresFailure(); failure != nil {
		payload["postgres_startup_failure"] = failure
	}
	writeJSON(response, status, payload)
}

// publicSystemInfo intentionally exposes only what the login screen needs.
// Database topology, bind addresses and enrollment policy belong to the
// authenticated settings view rather than an unauthenticated fingerprinting
// endpoint.
func (s *Server) publicSystemInfo(response http.ResponseWriter, _ *http.Request) {
	writeJSON(response, http.StatusOK, map[string]any{
		"product":        "Invenqor",
		"server_version": version.Version,
	})
}

func (s *Server) adminSystemInfo(response http.ResponseWriter, request *http.Request) {
	policy, _, policyErr := s.loadAgentEnrollmentPolicy(request.Context())
	enrollmentEnabled := s.agentEnrollmentEnabled
	enrollmentMode := s.agentEnrollmentMode()
	enrollmentSource := "startup-environment"
	if policyErr == nil {
		enrollmentEnabled = policy.Enabled
		enrollmentMode = policy.mode()
		enrollmentSource = "database"
	}
	payload := map[string]any{
		"product":                 "Invenqor",
		"server_version":          version.Version,
		"commit":                  version.Commit,
		"build_time":              version.BuildTime,
		"database_mode":           s.database.Mode(),
		"listen_address":          s.listenAddress,
		"port":                    listenPort(s.listenAddress),
		"agent_auto_enrollment":   enrollmentEnabled,
		"agent_enrollment_mode":   enrollmentMode,
		"agent_enrollment_source": enrollmentSource,
	}
	if policyErr != nil {
		payload["agent_enrollment_policy_available"] = false
	}
	if failure := s.database.PostgresFailure(); failure != nil {
		payload["postgres_startup_failure"] = failure
	}
	writeJSON(response, http.StatusOK, payload)
}

// listenPort reports the port an operator can actually reach, so the console's
// runtime panel states a fact rather than a default.
func listenPort(address string) any {
	if address == "" {
		return nil
	}
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil
	}
	number, err := strconv.Atoi(port)
	if err != nil {
		return nil
	}
	return number
}

func (s *Server) agentEnrollmentMode() string {
	if !s.agentEnrollmentEnabled {
		return "disabled"
	}
	if s.agentEnrollmentTokenRequired {
		return "token"
	}
	return "open"
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("X-Content-Type-Options", "nosniff")
		response.Header().Set("X-Frame-Options", "DENY")
		response.Header().Set("Referrer-Policy", "no-referrer")
		// A page gets the console policy - widened only by what an active
		// tracking snippet needs, with a nonce that the console handler
		// writes into the snippet. Everything else is never a document and
		// gets the closed policy.
		policy := apiPolicy
		if strings.HasPrefix(request.URL.Path, "/assets/") {
			policy = basePagePolicy
		} else if isDocumentPath(request.URL.Path) {
			var page *trackingPage
			policy, page = s.pagePolicy(request)
			if page != nil {
				request = request.WithContext(context.WithValue(
					request.Context(), trackingPageKey{}, page,
				))
			}
		}
		response.Header().Set("Content-Security-Policy", policy)
		if request.TLS != nil {
			response.Header().Set(
				"Strict-Transport-Security",
				"max-age=31536000; includeSubDomains",
			)
		}
		next.ServeHTTP(response, request)
	})
}

func discardInboundRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		request.Header.Del(middleware.RequestIDHeader)
		next.ServeHTTP(response, request)
	})
}

func (s *Server) requestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		started := time.Now()
		// The Agent logs this header verbatim, which is what lets an operator
		// paste one identifier into the console and find the server side of a
		// failed registration.
		requestID := middleware.GetReqID(request.Context())
		if requestID != "" {
			response.Header().Set("X-Request-Id", requestID)
		}
		wrapped := middleware.NewWrapResponseWriter(response, request.ProtoMajor)
		next.ServeHTTP(wrapped, request)
		duration := time.Since(started).Milliseconds()
		s.logger.Info(
			"http_request",
			"request_id", middleware.GetReqID(request.Context()),
			"method", request.Method,
			"path", request.URL.Path,
			"status", wrapped.Status(),
			"bytes", wrapped.BytesWritten(),
			"duration_ms", duration,
			"remote_ip", request.RemoteAddr,
		)
		// Persist structured access logs in the shared database so the
		// management console can search requests from every Pod. Successful
		// liveness/readiness probes and static UI assets would consume the
		// bounded retention almost immediately, so retain API/Agent traffic and
		// every failed request instead.
		if shouldPersistRequestLog(request.URL.Path, wrapped.Status()) {
			level := "info"
			if wrapped.Status() >= http.StatusInternalServerError {
				level = "error"
			} else if wrapped.Status() >= http.StatusBadRequest {
				level = "warning"
			}
			s.recordDiagnostic(request, diagnostics.Event{
				Level:     level,
				Component: "http",
				EventCode: "HTTP_REQUEST",
				Message:   "The Server completed an HTTP request.",
				RequestID: requestID,
				SourceIP:  clientIP(request),
				Details: map[string]any{
					"method":      request.Method,
					"path":        request.URL.Path,
					"status":      wrapped.Status(),
					"bytes":       wrapped.BytesWritten(),
					"duration_ms": duration,
				},
			})
		}
	})
}

func shouldPersistRequestLog(path string, status int) bool {
	if status >= http.StatusBadRequest {
		return true
	}
	// Browser policy reports and proxied tracking events arrive on every
	// page view; recording them would fill the retention with noise.
	return path != "/" &&
		!strings.HasPrefix(path, "/assets/") &&
		!strings.HasPrefix(path, tracking.MomentoProxyPath+"/") &&
		path != cspReportPath &&
		path != "/health/live" && path != "/health/ready"
}

func writeJSON(response http.ResponseWriter, status int, payload any) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(status)
	encoder := json.NewEncoder(response)
	encoder.SetEscapeHTML(true)
	_ = encoder.Encode(payload)
}

func bearerToken(request *http.Request) string {
	header := request.Header.Get("Authorization")
	scheme, token, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}
