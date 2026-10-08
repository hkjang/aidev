package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/hkjang/invenqor/server/internal/tracking"
)

// trackingPolicyKey is the server_metadata row that holds the visitor
// tracking configuration, shared by every Server pod like the enrollment
// policy is.
const trackingPolicyKey = "tracking_policy"

// cspReportPath is where browsers post the requests the content security
// policy refused. It is unauthenticated because the browser sends the report
// without credentials, and it stores nothing but a bounded list of origins in
// memory.
const cspReportPath = "/api/v1/tracking/csp-report"

// maxCSPReportBytes keeps an unauthenticated endpoint from being used to push
// large bodies at the Server.
const maxCSPReportBytes = 8 * 1024

// maxMomentoProxyBytes bounds one event batch forwarded to the collector.
const maxMomentoProxyBytes = 256 * 1024

// basePagePolicy is the policy the console has always shipped with. While
// tracking is off it is emitted byte for byte, so a fresh installation sees
// no change at all.
const basePagePolicy = "default-src 'self'; frame-ancestors 'none'; " +
	"object-src 'none'; base-uri 'self'"

// apiPolicy is for responses that are never a document. Nothing there needs
// to load anything, so the policy says so.
const apiPolicy = "default-src 'none'; frame-ancestors 'none'"

var errTrackingPolicyConflict = errors.New(
	"tracking policy changed concurrently",
)

// trackingValidationError carries what the administrator got wrong, as
// opposed to a storage failure, so the handler can answer 400 rather than 500.
type trackingValidationError struct{ err error }

func (e trackingValidationError) Error() string { return e.err.Error() }

type trackingPolicy struct {
	tracking.Config
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy string    `json:"updated_by"`
}

func (policy trackingPolicy) publicValue() map[string]any {
	config := policy.Config
	return map[string]any{
		"enabled":             config.Enabled,
		"active":              config.Active(),
		"provider":            config.Provider,
		"providers":           tracking.Providers,
		"momento_url":         config.MomentoURL,
		"momento_site_id":     config.MomentoSiteID,
		"momento_environment": config.MomentoEnvironment,
		"momento_proxy":       config.MomentoProxy,
		"momento_proxy_path":  tracking.MomentoProxyPath,
		"measurement_id":      config.MeasurementID,
		"matomo_url":          config.MatomoURL,
		"matomo_site_id":      config.MatomoSiteID,
		"custom_snippet":      config.CustomSnippet,
		"allowed_hosts":       config.AllowedHosts,
		"placement":           config.Placement,
		"policy_sources":      config.PolicySources().All(),
		"version":             policy.Version,
		"updated_at":          policy.UpdatedAt,
		"updated_by":          policy.UpdatedBy,
		"source":              "database",
	}
}

func initialTrackingPolicy() trackingPolicy {
	return trackingPolicy{Config: tracking.Default(), UpdatedBy: "default"}
}

// loadTrackingPolicy reads the stored policy. A missing row is the default,
// off, and is not written: a fresh installation leaves no trace until an
// administrator changes something. The value is not validated here so a row
// an older or newer version wrote can still be read and corrected from the
// console; trackingConfig is where an invalid value falls back to off.
func (s *Server) loadTrackingPolicy(
	ctx context.Context,
) (trackingPolicy, string, error) {
	var raw string
	err := s.database.DB().QueryRowContext(
		ctx,
		`SELECT value FROM server_metadata WHERE key=$1`,
		trackingPolicyKey,
	).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return initialTrackingPolicy(), "", nil
	}
	if err != nil {
		return trackingPolicy{}, "", fmt.Errorf("load tracking policy: %w", err)
	}
	policy := initialTrackingPolicy()
	if err := json.Unmarshal([]byte(raw), &policy); err != nil {
		return trackingPolicy{}, "", fmt.Errorf("decode tracking policy: %w", err)
	}
	policy.Config = policy.Config.Normalize()
	return policy, raw, nil
}

// trackingConfig is the read used on every page request. A storage failure
// is treated as "no tracking" so a database outage never breaks the console.
func (s *Server) trackingConfig(ctx context.Context) tracking.Config {
	policy, _, err := s.loadTrackingPolicy(ctx)
	if err == nil {
		err = policy.Config.Validate()
	}
	if err != nil {
		s.logger.Warn("tracking_policy_unavailable", "error", err.Error())
		return tracking.Default()
	}
	return policy.Config
}

func (s *Server) updateTrackingPolicy(
	ctx context.Context,
	updatedBy string,
	mutate func(*tracking.Config) error,
) (trackingPolicy, trackingPolicy, error) {
	for range 5 {
		before, raw, err := s.loadTrackingPolicy(ctx)
		if err != nil {
			return trackingPolicy{}, trackingPolicy{}, err
		}
		after := before
		if err := mutate(&after.Config); err != nil {
			return before, trackingPolicy{}, trackingValidationError{err}
		}
		after.Config = after.Config.Normalize()
		if err := after.Config.Validate(); err != nil {
			return before, trackingPolicy{}, trackingValidationError{err}
		}
		after.Version = before.Version + 1
		after.UpdatedAt = time.Now().UTC()
		after.UpdatedBy = updatedBy
		encoded, err := json.Marshal(after)
		if err != nil {
			return before, trackingPolicy{}, err
		}
		var result sql.Result
		if raw == "" {
			result, err = s.database.DB().ExecContext(
				ctx,
				`INSERT INTO server_metadata(key,value,updated_at)
				 VALUES($1,$2,$3) ON CONFLICT(key) DO NOTHING`,
				trackingPolicyKey, string(encoded), after.UpdatedAt,
			)
		} else {
			result, err = s.database.DB().ExecContext(
				ctx,
				`UPDATE server_metadata SET value=$1,updated_at=$2
				 WHERE key=$3 AND value=$4`,
				string(encoded), after.UpdatedAt, trackingPolicyKey, raw,
			)
		}
		if err != nil {
			return before, trackingPolicy{}, fmt.Errorf("save tracking policy: %w", err)
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return before, trackingPolicy{}, err
		}
		if changed == 1 {
			return before, after, nil
		}
	}
	return trackingPolicy{}, trackingPolicy{}, errTrackingPolicyConflict
}

func (s *Server) getTrackingSettings(
	response http.ResponseWriter,
	request *http.Request,
) {
	policy, _, err := s.loadTrackingPolicy(request.Context())
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	writeJSON(response, http.StatusOK, policy.publicValue())
}

func (s *Server) updateTrackingSettings(
	response http.ResponseWriter,
	request *http.Request,
) {
	var input struct {
		Enabled            *bool     `json:"enabled"`
		Provider           *string   `json:"provider"`
		MomentoURL         *string   `json:"momento_url"`
		MomentoSiteID      *string   `json:"momento_site_id"`
		MomentoEnvironment *string   `json:"momento_environment"`
		MomentoProxy       *bool     `json:"momento_proxy"`
		MeasurementID      *string   `json:"measurement_id"`
		MatomoURL          *string   `json:"matomo_url"`
		MatomoSiteID       *string   `json:"matomo_site_id"`
		CustomSnippet      *string   `json:"custom_snippet"`
		AllowedHosts       *[]string `json:"allowed_hosts"`
		Placement          *string   `json:"placement"`
		Reason             string    `json:"reason"`
	}
	if decodeJSON(request, &input) != nil {
		writeAPIError(
			response, request, http.StatusBadRequest,
			"INVALID_TRACKING_POLICY", "The request body is invalid.",
		)
		return
	}
	principal := principalFromContext(request.Context())
	before, after, err := s.updateTrackingPolicy(
		request.Context(),
		principal.User.ID,
		func(config *tracking.Config) error {
			assign := func(target *string, value *string) {
				if value != nil {
					*target = *value
				}
			}
			if input.Enabled != nil {
				config.Enabled = *input.Enabled
			}
			assign(&config.Provider, input.Provider)
			assign(&config.MomentoURL, input.MomentoURL)
			assign(&config.MomentoSiteID, input.MomentoSiteID)
			assign(&config.MomentoEnvironment, input.MomentoEnvironment)
			if input.MomentoProxy != nil {
				config.MomentoProxy = *input.MomentoProxy
			}
			assign(&config.MeasurementID, input.MeasurementID)
			assign(&config.MatomoURL, input.MatomoURL)
			assign(&config.MatomoSiteID, input.MatomoSiteID)
			assign(&config.CustomSnippet, input.CustomSnippet)
			if input.AllowedHosts != nil {
				config.AllowedHosts = *input.AllowedHosts
			}
			assign(&config.Placement, input.Placement)
			return nil
		},
	)
	if err != nil {
		s.writeTrackingPolicyError(response, request, err)
		return
	}
	s.recordAdminAudit(
		request, "tracking.policy.update", "tracking_policy",
		trackingPolicyKey, before.publicValue(), after.publicValue(),
		input.Reason,
	)
	writeJSON(response, http.StatusOK, after.publicValue())
}

func (s *Server) writeTrackingPolicyError(
	response http.ResponseWriter,
	request *http.Request,
	err error,
) {
	if errors.Is(err, errTrackingPolicyConflict) {
		writeAPIError(
			response, request, http.StatusConflict,
			"TRACKING_POLICY_CONFLICT",
			"The policy changed concurrently. Reload and try again.",
		)
		return
	}
	var invalid trackingValidationError
	if errors.As(err, &invalid) {
		writeAPIError(
			response, request, http.StatusBadRequest,
			"INVALID_TRACKING_POLICY", invalid.Error(),
		)
		return
	}
	s.internalError(response, request, err)
}

// listTrackingViolations shows the administrator which addresses the policy
// is blocking, so a snippet can be fixed without reading the browser console.
func (s *Server) listTrackingViolations(
	response http.ResponseWriter,
	request *http.Request,
) {
	config := s.trackingConfig(request.Context())
	writeJSON(response, http.StatusOK, map[string]any{
		"items":   s.violations.List(config),
		"active":  config.Active(),
		"maximum": tracking.MaxViolations,
	})
}

// clearTrackingViolations forgets the recorded reports, which is how an
// administrator checks whether a change actually fixed the snippet.
func (s *Server) clearTrackingViolations(
	response http.ResponseWriter,
	request *http.Request,
) {
	s.violations.Forget()
	s.recordAdminAudit(
		request, "tracking.violations.clear", "tracking_policy",
		trackingPolicyKey, nil, nil, "",
	)
	response.WriteHeader(http.StatusNoContent)
}

// allowTrackingHost adds one blocked origin to the allow list. It is the
// one-click fix for the reports listed above.
func (s *Server) allowTrackingHost(
	response http.ResponseWriter,
	request *http.Request,
) {
	var input struct {
		Origin string `json:"origin"`
		Reason string `json:"reason"`
	}
	if decodeJSON(request, &input) != nil {
		writeAPIError(
			response, request, http.StatusBadRequest,
			"INVALID_TRACKING_POLICY", "The request body is invalid.",
		)
		return
	}
	origin := strings.TrimSpace(input.Origin)
	if origin == "" {
		writeAPIError(
			response, request, http.StatusBadRequest,
			"INVALID_TRACKING_POLICY", "origin is required.",
		)
		return
	}
	principal := principalFromContext(request.Context())
	before, after, err := s.updateTrackingPolicy(
		request.Context(),
		principal.User.ID,
		func(config *tracking.Config) error {
			config.AllowedHosts = tracking.AddAllowedHost(
				config.AllowedHosts, origin,
			)
			return nil
		},
	)
	if err != nil {
		s.writeTrackingPolicyError(response, request, err)
		return
	}
	s.recordAdminAudit(
		request, "tracking.allowed_host.add", "tracking_policy",
		trackingPolicyKey, before.publicValue(), after.publicValue(),
		input.Reason,
	)
	writeJSON(response, http.StatusOK, after.publicValue())
}

type cspReport struct {
	Report struct {
		BlockedURI         string `json:"blocked-uri"`
		ViolatedDirective  string `json:"violated-directive"`
		EffectiveDirective string `json:"effective-directive"`
		DocumentURI        string `json:"document-uri"`
	} `json:"csp-report"`
}

// receiveCSPReport records what a browser refused to load. Reports are always
// answered with 204 so a misbehaving page never sees an error from us.
func (s *Server) receiveCSPReport(
	response http.ResponseWriter,
	request *http.Request,
) {
	defer response.WriteHeader(http.StatusNoContent)
	body, err := io.ReadAll(io.LimitReader(request.Body, maxCSPReportBytes))
	if err != nil || len(body) == 0 {
		return
	}
	var report cspReport
	if json.Unmarshal(body, &report) != nil {
		return
	}
	directive := report.Report.EffectiveDirective
	if directive == "" {
		directive = report.Report.ViolatedDirective
	}
	s.violations.Record(
		report.Report.BlockedURI, directive, report.Report.DocumentURI,
	)
}

// trackingPage is what the security middleware hands the console handler:
// the nonce it wrote into the policy and the snippet that carries it.
type trackingPage struct {
	Snippet   string
	Placement string
}

type trackingPageKey struct{}

// pagePolicy keeps the strict page policy and adds only what the configured
// snippet needs, including a nonce for its inline code. The returned page is
// nil when nothing is to be injected.
func (s *Server) pagePolicy(
	request *http.Request,
) (string, *trackingPage) {
	config := s.trackingConfig(request.Context())
	if !config.Active() {
		return basePagePolicy, nil
	}
	nonce, err := tracking.NewNonce()
	if err != nil {
		s.logger.Error("tracking_nonce_failed", "error", err.Error())
		return basePagePolicy, nil
	}
	return policyFor(config, nonce), &trackingPage{
		Snippet:   config.Snippet(nonce),
		Placement: config.Placement,
	}
}

// policyFor assembles the page policy for an active configuration. The base
// directives stay as they are; script-src, connect-src and img-src are spelled
// out only to add the nonce and the snippet's origins. 'unsafe-inline' is
// never emitted: the nonce is what lets the snippet run.
func policyFor(config tracking.Config, nonce string) string {
	sources := config.PolicySources()
	scripts := append([]string{"'self'", "'nonce-" + nonce + "'"}, sources.Scripts...)
	connects := append([]string{"'self'"}, sources.Connects...)
	images := append([]string{"'self'", "data:"}, sources.Images...)
	return basePagePolicy +
		"; script-src " + strings.Join(scripts, " ") +
		"; connect-src " + strings.Join(connects, " ") +
		"; img-src " + strings.Join(images, " ") +
		// While tracking is on, ask the browser to say what it refused. That
		// report is what turns a console error into a one-click fix.
		"; report-uri " + cspReportPath
}

// decorateConsole injects the snippet the middleware prepared for this
// request into the console shell.
func decorateConsole(request *http.Request, page []byte) []byte {
	prepared, _ := request.Context().Value(trackingPageKey{}).(*trackingPage)
	if prepared == nil || prepared.Snippet == "" {
		return page
	}
	return tracking.InjectSnippet(page, prepared.Snippet, prepared.Placement)
}

// isDocumentPath reports whether a request can be a page. Everything else
// gets the closed policy and never a snippet.
func isDocumentPath(path string) bool {
	return !strings.HasPrefix(path, "/api/") &&
		!strings.HasPrefix(path, "/v1/") &&
		!strings.HasPrefix(path, "/health/") &&
		!strings.HasPrefix(path, "/assets/") &&
		!strings.HasPrefix(path, tracking.MomentoProxyPath+"/") &&
		path != "/mcp"
}

// momentoProxy forwards /momento/* to the collector while Momento tracking is
// on. The browser then talks to this Server only, so no external origin has to
// enter the policy and a closed network needs no policy change at all.
func (s *Server) momentoProxy(
	response http.ResponseWriter,
	request *http.Request,
) {
	config := s.trackingConfig(request.Context())
	if !config.MomentoProxied() {
		writeAPIError(
			response, request, http.StatusNotFound,
			"TRACKING_PROXY_DISABLED",
			"The Momento proxy is off. Enable Momento tracking with the same-origin proxy first.",
		)
		return
	}
	target, err := url.Parse(config.MomentoURL)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	suffix := strings.TrimPrefix(request.URL.Path, tracking.MomentoProxyPath)
	proxy := &httputil.ReverseProxy{
		Rewrite: func(proxied *httputil.ProxyRequest) {
			proxied.SetURL(target)
			proxied.Out.URL.Path = strings.TrimSuffix(target.Path, "/") + suffix
			proxied.Out.URL.RawPath = ""
			proxied.Out.Host = target.Host
			// The visitor's session is for this Server, not the collector.
			proxied.Out.Header.Del("Cookie")
			proxied.Out.Header.Del("Authorization")
			proxied.SetXForwarded()
		},
		ModifyResponse: func(upstream *http.Response) error {
			// The collector must not plant cookies on this origin.
			upstream.Header.Del("Set-Cookie")
			return nil
		},
		Transport: s.momentoTransport,
		ErrorHandler: func(
			response http.ResponseWriter, request *http.Request, err error,
		) {
			s.logger.Warn(
				"tracking_proxy_failed", "error", err.Error(),
				"path", request.URL.Path,
			)
			writeAPIError(
				response, request, http.StatusBadGateway,
				"TRACKING_PROXY_UNAVAILABLE",
				"The Momento collector did not answer.",
			)
		},
	}
	request.Body = http.MaxBytesReader(response, request.Body, maxMomentoProxyBytes)
	proxy.ServeHTTP(response, request)
}
