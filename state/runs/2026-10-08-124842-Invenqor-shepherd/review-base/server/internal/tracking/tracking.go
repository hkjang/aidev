// Package tracking injects a visitor tracking snippet into the management
// console shell.
//
// The console's content security policy allows scripts from the Server origin
// only, so a tracking snippet cannot simply be pasted into the page: the
// browser drops it without a word and the administrator sees nothing arrive.
// This package produces both halves of the answer - the markup to inject and
// the policy sources it needs - with a per-request nonce so the pasted inline
// code runs while the policy stays as strict as it is for everything else.
package tracking

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html"
	"net/url"
	"strings"
)

const (
	ProviderNone    = "none"
	ProviderMomento = "momento"
	ProviderGA4     = "ga4"
	ProviderGTM     = "gtm"
	ProviderMatomo  = "matomo"
	ProviderCustom  = "custom"

	// MaxSnippetBytes bounds a pasted snippet. A tracker loader is a few
	// hundred bytes; anything larger is a page, not a snippet.
	MaxSnippetBytes = 8 * 1024
	// MaxAllowedHosts bounds the administrator's manual allow list.
	MaxAllowedHosts = 64

	// MomentoProxyPath is the same-origin prefix the Server forwards to the
	// Momento collector, so no external origin has to enter the policy.
	MomentoProxyPath = "/momento"
)

// Providers lists the accepted provider names. Momento comes first: it is the
// self-hosted collector, the one choice that keeps the data inside.
var Providers = []string{
	ProviderNone, ProviderMomento, ProviderGA4, ProviderGTM, ProviderMatomo,
	ProviderCustom,
}

// Config is the administrator's tracking configuration. It lives in the
// database rather than in a build-time environment variable because on a
// closed network the collector address differs per installation and changes
// while the Server runs; a setting that needs a redeploy stays off.
type Config struct {
	Enabled            bool     `json:"enabled"`
	Provider           string   `json:"provider"`
	MomentoURL         string   `json:"momento_url"`
	MomentoSiteID      string   `json:"momento_site_id"`
	MomentoEnvironment string   `json:"momento_environment"`
	MomentoProxy       bool     `json:"momento_proxy"`
	MeasurementID      string   `json:"measurement_id"`
	MatomoURL          string   `json:"matomo_url"`
	MatomoSiteID       string   `json:"matomo_site_id"`
	CustomSnippet      string   `json:"custom_snippet"`
	AllowedHosts       []string `json:"allowed_hosts"`
	Placement          string   `json:"placement"`
}

// Default is what a fresh installation runs with: nothing is injected and the
// policy is exactly what it was before tracking existed.
func Default() Config {
	return Config{
		Provider:           ProviderNone,
		MomentoEnvironment: "prd",
		MomentoProxy:       true,
		AllowedHosts:       []string{},
		Placement:          "head",
	}
}

// Normalize trims the free-text fields and fills the defaults a stored value
// written by an older version may lack.
func (c Config) Normalize() Config {
	c.Provider = strings.ToLower(strings.TrimSpace(c.Provider))
	if c.Provider == "" {
		c.Provider = ProviderNone
	}
	c.MomentoURL = strings.TrimRight(strings.TrimSpace(c.MomentoURL), "/")
	c.MomentoSiteID = strings.TrimSpace(c.MomentoSiteID)
	c.MomentoEnvironment = strings.TrimSpace(c.MomentoEnvironment)
	if c.MomentoEnvironment == "" {
		c.MomentoEnvironment = "prd"
	}
	c.MeasurementID = strings.TrimSpace(c.MeasurementID)
	c.MatomoURL = strings.TrimRight(strings.TrimSpace(c.MatomoURL), "/")
	c.MatomoSiteID = strings.TrimSpace(c.MatomoSiteID)
	c.CustomSnippet = strings.TrimSpace(c.CustomSnippet)
	c.AllowedHosts = normalizeHosts(c.AllowedHosts)
	c.Placement = strings.ToLower(strings.TrimSpace(c.Placement))
	if c.Placement != "body" {
		c.Placement = "head"
	}
	return c
}

func normalizeHosts(hosts []string) []string {
	unique := make(map[string]struct{}, len(hosts))
	result := make([]string, 0, len(hosts))
	for _, host := range hosts {
		host = strings.TrimSuffix(strings.TrimSpace(host), "/")
		if host == "" {
			continue
		}
		key := strings.ToLower(host)
		if _, seen := unique[key]; seen {
			continue
		}
		unique[key] = struct{}{}
		result = append(result, host)
	}
	return result
}

// Validate reports what is wrong with the configuration. The snippet size and
// the allow list are checked even while tracking is off, so a value that could
// never be switched on is refused at the moment it is typed.
func (c Config) Validate() error {
	known := false
	for _, provider := range Providers {
		if c.Provider == provider {
			known = true
		}
	}
	if !known {
		return fmt.Errorf(
			"provider must be one of %s", strings.Join(Providers, ", "),
		)
	}
	if len(c.CustomSnippet) > MaxSnippetBytes {
		return fmt.Errorf(
			"custom_snippet must not exceed %d bytes", MaxSnippetBytes,
		)
	}
	if len(c.AllowedHosts) > MaxAllowedHosts {
		return fmt.Errorf(
			"allowed_hosts must not exceed %d entries", MaxAllowedHosts,
		)
	}
	for _, host := range c.AllowedHosts {
		if !isPolicyOrigin(host) {
			return fmt.Errorf(
				"allowed_hosts entry %q is not an http(s) origin", host,
			)
		}
	}
	if c.MomentoURL != "" && originOf(c.MomentoURL) == "" {
		return fmt.Errorf("momento_url is not a valid http(s) address")
	}
	if c.MatomoURL != "" && originOf(c.MatomoURL) == "" {
		return fmt.Errorf("matomo_url is not a valid http(s) address")
	}
	if !c.Enabled {
		return nil
	}
	switch c.Provider {
	case ProviderNone:
		return fmt.Errorf("choose a provider before enabling tracking")
	case ProviderMomento:
		if c.MomentoURL == "" || c.MomentoSiteID == "" {
			return fmt.Errorf("momento_url and momento_site_id are required")
		}
	case ProviderGA4, ProviderGTM:
		if c.MeasurementID == "" {
			return fmt.Errorf("measurement_id is required")
		}
	case ProviderMatomo:
		if c.MatomoURL == "" || c.MatomoSiteID == "" {
			return fmt.Errorf("matomo_url and matomo_site_id are required")
		}
	case ProviderCustom:
		if c.CustomSnippet == "" {
			return fmt.Errorf("custom_snippet is empty")
		}
	}
	return nil
}

// isPolicyOrigin accepts what a CSP source list accepts from an
// administrator: an http(s) origin, optionally with a leading wildcard
// label. Anything else - and in particular a quoted keyword such as
// 'unsafe-inline' - is refused, so the allow list can never loosen the policy
// beyond the origins it names.
func isPolicyOrigin(value string) bool {
	lower := strings.ToLower(value)
	if !strings.HasPrefix(lower, "http://") &&
		!strings.HasPrefix(lower, "https://") {
		return false
	}
	parsed, err := url.Parse(strings.Replace(value, "*.", "wildcard.", 1))
	if err != nil || parsed.Host == "" || parsed.User != nil {
		return false
	}
	if parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	for _, letter := range value {
		if letter <= ' ' || letter == '\'' || letter == '"' || letter == ';' ||
			letter == ',' {
			return false
		}
	}
	return true
}

// Active reports whether the console shell should carry the snippet.
func (c Config) Active() bool {
	if !c.Enabled || c.Provider == ProviderNone {
		return false
	}
	return strings.TrimSpace(c.Snippet("")) != ""
}

// MomentoProxied reports whether /momento/* should be forwarded to the
// collector, which is the case only while Momento tracking is on.
func (c Config) MomentoProxied() bool {
	return c.Active() && c.Provider == ProviderMomento && c.MomentoProxy &&
		originOf(c.MomentoURL) != ""
}

// Snippet renders the markup to inject. The nonce is applied to every script
// tag so the policy can stay strict.
func (c Config) Snippet(nonce string) string {
	switch c.Provider {
	case ProviderMomento:
		site := html.EscapeString(c.MomentoSiteID)
		environment := html.EscapeString(c.MomentoEnvironment)
		if site == "" {
			return ""
		}
		if c.MomentoProxy {
			// Through the same-origin proxy the loader and the endpoint are
			// both served by this Server, so the policy needs no new origin.
			return withNonce(fmt.Sprintf(
				`<script async src="%s/tracker.js" data-site-id="%s" data-environment="%s" data-contract-version="1" data-endpoint="%s"></script>`,
				MomentoProxyPath, site, environment, MomentoProxyPath,
			), nonce)
		}
		base := html.EscapeString(c.MomentoURL)
		if base == "" {
			return ""
		}
		return withNonce(fmt.Sprintf(
			`<script async src="%s/tracker.js" data-site-id="%s" data-environment="%s" data-contract-version="1"></script>`,
			base, site, environment,
		), nonce)
	case ProviderGA4:
		id := html.EscapeString(c.MeasurementID)
		if id == "" {
			return ""
		}
		return withNonce(fmt.Sprintf(`<script async src="https://www.googletagmanager.com/gtag/js?id=%s"></script>
<script>window.dataLayer=window.dataLayer||[];function gtag(){dataLayer.push(arguments);}gtag('js',new Date());gtag('config','%s');</script>`, id, id), nonce)
	case ProviderGTM:
		id := html.EscapeString(c.MeasurementID)
		if id == "" {
			return ""
		}
		return withNonce(fmt.Sprintf(`<script>(function(w,d,s,l,i){w[l]=w[l]||[];w[l].push({'gtm.start':new Date().getTime(),event:'gtm.js'});var f=d.getElementsByTagName(s)[0],j=d.createElement(s),dl=l!='dataLayer'?'&l='+l:'';j.async=true;j.src='https://www.googletagmanager.com/gtm.js?id='+i+dl;f.parentNode.insertBefore(j,f);})(window,document,'script','dataLayer','%s');</script>`, id), nonce)
	case ProviderMatomo:
		base := html.EscapeString(c.MatomoURL)
		site := html.EscapeString(c.MatomoSiteID)
		if base == "" || site == "" {
			return ""
		}
		return withNonce(fmt.Sprintf(`<script>var _paq=window._paq=window._paq||[];_paq.push(['trackPageView']);_paq.push(['enableLinkTracking']);(function(){var u="%s/";_paq.push(['setTrackerUrl',u+'matomo.php']);_paq.push(['setSiteId','%s']);var d=document,g=d.createElement('script'),s=d.getElementsByTagName('script')[0];g.async=true;g.src=u+'matomo.js';s.parentNode.insertBefore(g,s);})();</script>`, base, site), nonce)
	case ProviderCustom:
		return withNonce(c.CustomSnippet, nonce)
	}
	return ""
}

// withNonce adds the nonce to every script tag that does not already carry
// one, which is what lets a pasted snippet run under a strict policy unchanged.
func withNonce(snippet, nonce string) string {
	if nonce == "" || snippet == "" {
		return snippet
	}
	var builder strings.Builder
	remaining := snippet
	for {
		index := strings.Index(strings.ToLower(remaining), "<script")
		if index < 0 {
			builder.WriteString(remaining)
			return builder.String()
		}
		end := index + len("<script")
		builder.WriteString(remaining[:end])
		tag := remaining[end:]
		if closing := strings.Index(tag, ">"); closing >= 0 {
			tag = tag[:closing]
		}
		if !strings.Contains(strings.ToLower(tag), "nonce=") {
			builder.WriteString(` nonce="` + html.EscapeString(nonce) + `"`)
		}
		remaining = remaining[end:]
	}
}

// Sources lists the extra origins the snippet needs, grouped by directive,
// derived from the provider so a common setup needs no policy knowledge.
type Sources struct {
	Scripts  []string
	Connects []string
	Images   []string
}

// All returns every origin in the three groups, deduplicated, for the
// violation list to mark what the current configuration already allows.
func (s Sources) All() []string {
	seen := make(map[string]struct{})
	result := make([]string, 0)
	for _, group := range [][]string{s.Scripts, s.Connects, s.Images} {
		for _, origin := range group {
			key := strings.ToLower(origin)
			if _, duplicate := seen[key]; duplicate {
				continue
			}
			seen[key] = struct{}{}
			result = append(result, origin)
		}
	}
	return result
}

// PolicySources lists what the configured snippet loads from and reports to.
func (c Config) PolicySources() Sources {
	var sources Sources
	everywhere := func(origin string) {
		sources.Scripts = append(sources.Scripts, origin)
		sources.Connects = append(sources.Connects, origin)
		sources.Images = append(sources.Images, origin)
	}
	switch c.Provider {
	case ProviderMomento:
		if !c.MomentoProxy {
			if origin := originOf(c.MomentoURL); origin != "" {
				everywhere(origin)
			}
		}
	case ProviderGA4, ProviderGTM:
		sources.Scripts = append(sources.Scripts, "https://www.googletagmanager.com")
		sources.Connects = append(sources.Connects,
			"https://www.google-analytics.com",
			"https://analytics.google.com",
			"https://*.google-analytics.com",
		)
		sources.Images = append(sources.Images,
			"https://www.google-analytics.com",
			"https://www.googletagmanager.com",
		)
	case ProviderMatomo:
		if origin := originOf(c.MatomoURL); origin != "" {
			everywhere(origin)
		}
	case ProviderCustom:
		// A pasted snippet names the addresses it loads and reports to, so
		// those origins are allowed without anybody reading a policy error.
		for _, origin := range SnippetOrigins(c.CustomSnippet) {
			everywhere(origin)
		}
	}
	for _, host := range c.AllowedHosts {
		everywhere(host)
	}
	return sources
}

// SnippetOrigins lists every http(s) origin written into a tracking snippet:
// the script it loads, the endpoint it posts to, the pixel it requests. A
// tracker almost always writes its own address somewhere in its loader.
func SnippetOrigins(snippet string) []string {
	origins := make([]string, 0, 2)
	seen := make(map[string]struct{}, 2)
	for index := 0; index < len(snippet); {
		start := strings.Index(strings.ToLower(snippet[index:]), "http")
		if start < 0 {
			break
		}
		start += index
		end := start
		for end < len(snippet) && !isURLBoundary(snippet[end]) {
			end++
		}
		index = end
		origin := originOf(snippet[start:end])
		if origin == "" {
			continue
		}
		if _, duplicate := seen[origin]; duplicate {
			continue
		}
		seen[origin] = struct{}{}
		origins = append(origins, origin)
	}
	return origins
}

// isURLBoundary reports the characters that cannot appear in a URL written
// inside HTML or JavaScript, which is where each address ends.
func isURLBoundary(letter byte) bool {
	switch letter {
	case '"', '\'', '`', '<', '>', ' ', '\t', '\n', '\r', ')', ',', ';', '\\',
		'+':
		return true
	}
	return false
}

// originOf reduces an address to scheme://host, which is the form a policy
// source list takes. Anything that is not http(s) yields "".
func originOf(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" {
		return ""
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return ""
	}
	return scheme + "://" + strings.ToLower(parsed.Host)
}

// NewNonce returns a fresh per-request script nonce.
func NewNonce() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate script nonce: %w", err)
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}

// AddAllowedHost appends an origin to the allow list, leaving the existing
// entries and their order alone.
func AddAllowedHost(existing []string, origin string) []string {
	origin = strings.TrimSuffix(strings.TrimSpace(origin), "/")
	if origin == "" {
		return existing
	}
	for _, host := range existing {
		if strings.EqualFold(host, origin) {
			return existing
		}
	}
	return append(append([]string{}, existing...), origin)
}

// InjectSnippet places the markup just before the closing tag it belongs to,
// falling back to the end of the document when the tag is missing.
func InjectSnippet(page []byte, snippet, placement string) []byte {
	marker := "</head>"
	if placement == "body" {
		marker = "</body>"
	}
	text := string(page)
	index := strings.LastIndex(strings.ToLower(text), marker)
	if index < 0 {
		return []byte(text + "\n" + snippet + "\n")
	}
	return []byte(text[:index] + snippet + "\n" + text[index:])
}
