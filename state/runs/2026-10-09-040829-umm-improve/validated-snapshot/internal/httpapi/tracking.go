package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/hkjang/umm/internal/analytics"
)

// Visitor tracking, as an administrator turns it on from the settings screen.
//
// The snippet itself is the small part. umm's policy pins scripts to a
// per-response nonce, so the work here is to put that nonce on the snippet, to
// widen the policy by exactly the origins the snippet needs while it is on,
// and to keep what the browser refused where the administrator can see it.
// Everything below reads one cached setting; when tracking is off — the
// default — every request takes the same path it always did.

// trackingTTL bounds how long an instance keeps serving a setting another
// instance has changed. Saving on this instance invalidates at once.
const trackingTTL = 30 * time.Second

// trackingConfig returns the current setting, cached briefly. Without a store
// — which is how the unit tests build a Server — there is nothing to read and
// tracking is simply off.
func (s *Server) trackingConfig(ctx context.Context) analytics.Config {
	if s == nil || s.Store == nil {
		return analytics.Config{}
	}
	s.trackingMu.Lock()
	defer s.trackingMu.Unlock()
	if !s.trackingLoadedAt.IsZero() && time.Since(s.trackingLoadedAt) < trackingTTL {
		return s.tracking
	}
	var config analytics.Config
	_ = s.Store.GetSetting(ctx, analytics.SettingKey, &config)
	s.tracking = config.Normalized()
	s.trackingLoadedAt = time.Now()
	return s.tracking
}

func (s *Server) invalidateTracking() {
	s.trackingMu.Lock()
	s.trackingLoadedAt = time.Time{}
	s.trackingMu.Unlock()
}

// violations is the recorder for refused origins, created on first use so a
// Server literal in a test needs nothing extra.
func (s *Server) violations() *analytics.Recorder {
	s.violationsOnce.Do(func() { s.violationRecorder = analytics.NewRecorder() })
	return s.violationRecorder
}

// contentSecurityPolicy pins script execution to the exact bundle umm served.
//
// script-src uses a per-response nonce with 'strict-dynamic', so an injected
// <script src="..."> is refused even when it points at an allowed origin.
// style-src deliberately keeps 'unsafe-inline': Mantine writes its theme
// variables into a runtime <style> element and React Flow positions every node
// with a style attribute, so removing it would break the canvas without closing
// a comparable hole. Everything else is denied outright.
func contentSecurityPolicy(nonce string) string {
	return trackedContentSecurityPolicy(nonce, analytics.Config{}, false)
}

// trackedContentSecurityPolicy is the same policy widened for the page that
// carries the tracking snippet: the snippet's origins are added to the three
// directives a tracker uses, and the browser is asked to report what it still
// refuses. Nothing is loosened — never 'unsafe-inline' — so turning tracking
// off returns the policy to exactly what it was.
func trackedContentSecurityPolicy(nonce string, config analytics.Config, tracked bool) string {
	scripts := []string{"'nonce-" + nonce + "'", "'strict-dynamic'", "'self'"}
	images := []string{"'self'", "data:", "blob:"}
	connects := []string{"'self'"}
	if tracked {
		extraScripts, extraConnects, extraImages := config.PolicySources()
		scripts = append(scripts, extraScripts...)
		connects = append(connects, extraConnects...)
		images = append(images, extraImages...)
	}
	directives := []string{
		"default-src 'self'",
		"img-src " + strings.Join(images, " "),
		"font-src 'self' data:",
		"style-src 'self' 'unsafe-inline'",
		"style-src-attr 'unsafe-inline'",
		"script-src " + strings.Join(scripts, " "),
		"connect-src " + strings.Join(connects, " "),
		"worker-src 'self'",
		"manifest-src 'self'",
		"object-src 'none'",
		"frame-src 'none'",
		"frame-ancestors 'none'",
		"base-uri 'self'",
		"form-action 'self'",
	}
	if tracked {
		directives = append(directives, "report-uri "+analytics.ReportPath)
	}
	return strings.Join(directives, "; ")
}

// injectSnippet places the snippet where the setting says, after the shell
// document has already had its own tags labelled — the snippet arrives with
// its nonce already on, so it must not pass through injectNonce a second time.
func injectSnippet(document []byte, snippet, placement string) []byte {
	if strings.TrimSpace(snippet) == "" {
		return document
	}
	payload := []byte("\n" + snippet + "\n")
	marker, index := []byte("</head>"), -1
	if placement == analytics.PlacementBody {
		marker = []byte("</body>")
	}
	index = bytes.LastIndex(bytes.ToLower(document), marker)
	if index < 0 {
		// A shell without the closing tag is not one Vite produced; append
		// rather than drop, so the administrator's setting still does something.
		return append(document, payload...)
	}
	out := make([]byte, 0, len(document)+len(payload))
	out = append(out, document[:index]...)
	out = append(out, payload...)
	return append(out, document[index:]...)
}

// cspReport receives the browser's account of what the policy refused.
//
// It is only ever named in the policy while tracking is on, and it answers 204
// to everything: a report is a courtesy from the browser, never something to
// argue with. Reports are read only while tracking is on, so an idle install
// does not keep a list anybody could fill from outside.
func (s *Server) cspReport(w http.ResponseWriter, r *http.Request) {
	defer func() { w.WriteHeader(http.StatusNoContent) }()
	config := s.trackingConfig(r.Context())
	if !config.Enabled || config.Provider == analytics.ProviderNone {
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 16<<10))
	if err != nil {
		return
	}
	for _, report := range parseCSPReports(body) {
		s.violations().Record(report.blocked, report.directive, report.page)
	}
}

type cspViolation struct{ blocked, directive, page string }

// parseCSPReports reads both shapes a browser sends: the report-uri body, one
// object under "csp-report", and the Reporting API's list of typed reports.
func parseCSPReports(body []byte) []cspViolation {
	var legacy struct {
		Report struct {
			BlockedURI         string `json:"blocked-uri"`
			EffectiveDirective string `json:"effective-directive"`
			ViolatedDirective  string `json:"violated-directive"`
			DocumentURI        string `json:"document-uri"`
		} `json:"csp-report"`
	}
	if json.Unmarshal(body, &legacy) == nil && legacy.Report.BlockedURI != "" {
		directive := legacy.Report.EffectiveDirective
		if directive == "" {
			directive = legacy.Report.ViolatedDirective
		}
		return []cspViolation{{legacy.Report.BlockedURI, directive, pagePathOf(legacy.Report.DocumentURI)}}
	}
	var modern []struct {
		Type string `json:"type"`
		Body struct {
			BlockedURL         string `json:"blockedURL"`
			EffectiveDirective string `json:"effectiveDirective"`
			DocumentURL        string `json:"documentURL"`
		} `json:"body"`
	}
	if json.Unmarshal(body, &modern) != nil {
		return nil
	}
	out := make([]cspViolation, 0, len(modern))
	for _, report := range modern {
		if report.Type == "csp-violation" && report.Body.BlockedURL != "" {
			out = append(out, cspViolation{report.Body.BlockedURL, report.Body.EffectiveDirective, pagePathOf(report.Body.DocumentURL)})
		}
	}
	return out
}

// pagePathOf keeps the path of the page that reported, and nothing else: the
// query string of a document URL is where a return_to or a search lives.
func pagePathOf(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Path == "" {
		return ""
	}
	return parsed.Path
}

// analyticsViolations lists what the policy has refused since the recorder
// was last emptied, with the origins the current setting already covers
// marked so a fixed snippet stops nagging.
func (s *Server) analyticsViolations(w http.ResponseWriter, r *http.Request) {
	config := s.trackingConfig(r.Context())
	writeJSON(w, 200, map[string]any{
		"violations": s.violations().List(config),
		"reportPath": analytics.ReportPath,
		"active":     config.Enabled && config.Provider != analytics.ProviderNone,
	})
}

func (s *Server) forgetAnalyticsViolations(w http.ResponseWriter, r *http.Request) {
	s.violations().Forget()
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// momentoProxy forwards ProxyPath to the Momento collector on umm's origin.
//
// This is the path on which tracking never names an outside origin in the
// policy: the browser loads the tracker from umm and posts events to umm, and
// umm hands both to the collector. It answers only while Momento is the
// configured provider with the proxy switched on — a proxy that stayed open
// after the setting went off would be an open relay to wherever the address
// last pointed. umm's own cookie is stripped on the way out: the collector is
// told about a visit, never handed the session that made it.
func (s *Server) momentoProxy(w http.ResponseWriter, r *http.Request) {
	config := s.trackingConfig(r.Context())
	if !config.ProxyActive() {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodOptions:
	default:
		w.Header().Set("Allow", "GET, HEAD, POST, OPTIONS")
		writeError(w, http.StatusMethodNotAllowed, "허용되지 않는 메서드입니다.")
		return
	}
	target, err := url.Parse(strings.TrimSpace(config.MomentoURL))
	if err != nil || target.Host == "" {
		http.NotFound(w, r)
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, analytics.ProxyPath)
	if rest == "" {
		rest = "/"
	}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(request *httputil.ProxyRequest) {
			request.SetURL(target)
			request.Out.URL.Path = strings.TrimRight(target.Path, "/") + rest
			request.Out.URL.RawPath = ""
			request.Out.Host = target.Host
			// The visitor's address is what a collector counts, so it is
			// forwarded; the visitor's session with umm is not its business.
			request.SetXForwarded()
			request.Out.Header.Del("Cookie")
			request.Out.Header.Del("Authorization")
		},
		Transport: s.momentoTransport(),
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			writeError(w, http.StatusBadGateway, "Momento 수집기에 닿지 못했습니다.")
		},
	}
	proxy.ServeHTTP(w, r)
}

// momentoTransport is shared across proxied requests so connections to the
// collector are reused, and bounded so a collector that stops answering does
// not hold umm's handlers with it.
func (s *Server) momentoTransport() http.RoundTripper {
	s.momentoOnce.Do(func() {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.ResponseHeaderTimeout = 10 * time.Second
		transport.MaxIdleConnsPerHost = 8
		s.momentoRoundTripper = transport
	})
	return s.momentoRoundTripper
}
