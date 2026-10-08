// Package analytics lets an administrator attach a visitor tracking snippet to
// the pages umm serves — and, which is the harder half, keeps the content
// security policy strict while it runs.
//
// umm pins script execution to a per-response nonce, so a snippet pasted into
// the shell document would simply be refused, silently, and the administrator
// would be left looking at an empty dashboard with no idea why. This package
// therefore produces both halves of the answer at once: the markup to inject,
// with the response nonce on every script tag, and the origins the policy must
// allow for it, read out of the snippet itself. 'unsafe-inline' is never the
// answer — once a policy is loosened that way it stays loose after tracking is
// turned off again.
//
// Momento comes first. It is the in-house collector, the one provider for which
// nothing leaves the network, and through the same-origin proxy under
// ProxyPath no external origin has to appear in the policy at all.
package analytics

import (
	"errors"
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

	// SettingKey names the app_settings row this configuration lives in.
	SettingKey = "analytics"

	// MaxSnippetBytes bounds a pasted snippet. Every real tracker loader fits
	// in a fraction of this; anything larger is a page, not a snippet.
	MaxSnippetBytes = 8 * 1024

	// ProxyPath is where umm forwards Momento traffic on its own origin, so the
	// tracker and the collector both look like umm to the browser's policy.
	ProxyPath = "/momento"

	// ReportPath receives the browser's policy violation reports while
	// tracking is on. It is only named in the policy while tracking is on.
	ReportPath = "/csp-report"

	PlacementHead = "head"
	PlacementBody = "body"
)

// Providers lists the choices in the order the settings screen offers them.
// Momento is first because it is the one that keeps the data inside.
var Providers = []string{ProviderMomento, ProviderGA4, ProviderGTM, ProviderMatomo, ProviderCustom, ProviderNone}

// Config is the stored setting. Field names match the JSON the settings screen
// saves, so a row reads straight into it.
type Config struct {
	Enabled       bool   `json:"enabled"`
	Provider      string `json:"provider"`
	MomentoURL    string `json:"momento_url"`
	MomentoSiteID string `json:"momento_site_id"`
	// MomentoProxy sends the tracker and its events through ProxyPath instead of
	// straight to the collector. On by default: it is the path on which the
	// policy never has to name anything outside umm.
	MomentoProxy  bool   `json:"momento_proxy"`
	MeasurementID string `json:"measurement_id"`
	MatomoURL     string `json:"matomo_url"`
	MatomoSiteID  string `json:"matomo_site_id"`
	CustomSnippet string `json:"custom_snippet"`
	// AllowedHosts is where an administrator adds an origin the snippet did not
	// name itself — usually one the violation list has just shown them.
	AllowedHosts string `json:"allowed_hosts"`
	IncludeAdmin bool   `json:"include_admin"`
	Placement    string `json:"placement"`
}

// Normalized fills the defaults a partially written row would otherwise lack.
func (c Config) Normalized() Config {
	c.Provider = strings.ToLower(strings.TrimSpace(c.Provider))
	if c.Provider == "" {
		c.Provider = ProviderNone
	}
	c.Placement = strings.ToLower(strings.TrimSpace(c.Placement))
	if c.Placement != PlacementBody {
		c.Placement = PlacementHead
	}
	c.MomentoURL = strings.TrimSpace(c.MomentoURL)
	c.MomentoSiteID = strings.TrimSpace(c.MomentoSiteID)
	c.MeasurementID = strings.TrimSpace(c.MeasurementID)
	c.MatomoURL = strings.TrimSpace(c.MatomoURL)
	c.MatomoSiteID = strings.TrimSpace(c.MatomoSiteID)
	c.CustomSnippet = strings.TrimSpace(c.CustomSnippet)
	c.AllowedHosts = strings.TrimSpace(c.AllowedHosts)
	return c
}

// Active reports whether a page at path should carry the snippet.
//
// The administration screens are left out unless asked for: the person
// configuring umm is rarely the visitor anyone wants to count. Paths that are
// not pages are the caller's concern — the policy for those is narrower still.
func (c Config) Active(path string) bool {
	c = c.Normalized()
	if !c.Enabled || c.Provider == ProviderNone {
		return false
	}
	if !c.IncludeAdmin && (path == "/admin" || strings.HasPrefix(path, "/admin/")) {
		return false
	}
	return strings.TrimSpace(c.Snippet("")) != ""
}

// ProxyActive reports whether ProxyPath should forward to Momento.
//
// It does not depend on the page: the tracker a page has already loaded keeps
// sending events after the administrator flips include_admin, and those must
// still land somewhere. It does depend on the feature being on — an off switch
// that left an open proxy behind would not be an off switch.
func (c Config) ProxyActive() bool {
	c = c.Normalized()
	return c.Enabled && c.Provider == ProviderMomento && c.MomentoProxy && originOf(c.MomentoURL) != ""
}

// Validate says what is missing for the chosen provider. A snippet saved with
// a hole in it looks exactly like one that works, until nothing arrives.
func (c Config) Validate() error {
	c = c.Normalized()
	if len(c.CustomSnippet) > MaxSnippetBytes {
		return fmt.Errorf("추적 스니펫은 %dKB를 넘을 수 없습니다", MaxSnippetBytes/1024)
	}
	for _, host := range splitHosts(c.AllowedHosts) {
		if origin := originOf(host); origin == "" || !hasPrefixFold(host, "http") {
			return fmt.Errorf("허용 출처 %q는 https://host 형태의 출처여야 합니다", host)
		}
	}
	switch c.Provider {
	case ProviderNone, ProviderMomento, ProviderGA4, ProviderGTM, ProviderMatomo, ProviderCustom:
	default:
		return errors.New("추적 도구는 momento, ga4, gtm, matomo, custom 중 하나여야 합니다")
	}
	if !c.Enabled {
		return nil
	}
	switch c.Provider {
	case ProviderMomento:
		if !isHTTPURL(c.MomentoURL) {
			return errors.New("Momento 수집기 주소는 http(s) 전체 주소여야 합니다")
		}
		if c.MomentoSiteID == "" {
			return errors.New("Momento 사이트 ID가 필요합니다")
		}
	case ProviderGA4, ProviderGTM:
		if c.MeasurementID == "" {
			return errors.New("측정 ID가 필요합니다")
		}
	case ProviderMatomo:
		if !isHTTPURL(c.MatomoURL) {
			return errors.New("Matomo 주소는 http(s) 전체 주소여야 합니다")
		}
		if c.MatomoSiteID == "" {
			return errors.New("Matomo 사이트 ID가 필요합니다")
		}
	case ProviderCustom:
		if c.CustomSnippet == "" {
			return errors.New("붙여 넣을 추적 스니펫이 비어 있습니다")
		}
		if !containsFold(c.CustomSnippet, "<script") {
			return errors.New("추적 스니펫에 <script> 태그가 없습니다")
		}
	}
	return nil
}

// Snippet renders the markup to inject, with the nonce on every script tag —
// which is the whole of what lets it run under the strict policy.
func (c Config) Snippet(nonce string) string {
	c = c.Normalized()
	switch c.Provider {
	case ProviderMomento:
		site := html.EscapeString(c.MomentoSiteID)
		if site == "" {
			return ""
		}
		if c.MomentoProxy {
			if originOf(c.MomentoURL) == "" {
				return ""
			}
			return withNonce(fmt.Sprintf(`<script async src="%s/tracker.js" data-site-id="%s" data-endpoint="%s" data-environment="prd" data-contract-version="1"></script>`, ProxyPath, site, ProxyPath), nonce)
		}
		base := strings.TrimRight(c.MomentoURL, "/")
		if originOf(base) == "" {
			return ""
		}
		return withNonce(fmt.Sprintf(`<script async src="%s/tracker.js" data-site-id="%s" data-environment="prd" data-contract-version="1"></script>`, html.EscapeString(base), site), nonce)
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
		base := strings.TrimRight(c.MatomoURL, "/")
		site := html.EscapeString(c.MatomoSiteID)
		if originOf(base) == "" || site == "" {
			return ""
		}
		return withNonce(fmt.Sprintf(`<script>var _paq=window._paq=window._paq||[];_paq.push(['trackPageView']);_paq.push(['enableLinkTracking']);(function(){var u="%s/";_paq.push(['setTrackerUrl',u+'matomo.php']);_paq.push(['setSiteId','%s']);var d=document,g=d.createElement('script'),s=d.getElementsByTagName('script')[0];g.async=true;g.src=u+'matomo.js';s.parentNode.insertBefore(g,s);})();</script>`, html.EscapeString(base), site), nonce)
	case ProviderCustom:
		return withNonce(c.CustomSnippet, nonce)
	}
	return ""
}

// PolicySources lists the origins the snippet needs beyond umm's own, per
// directive. A common setup needs no policy knowledge at all: the provider
// says where it loads from and where it reports to.
func (c Config) PolicySources() (scripts, connects, images []string) {
	c = c.Normalized()
	add := func(origin string) {
		scripts = append(scripts, origin)
		connects = append(connects, origin)
		images = append(images, origin)
	}
	switch c.Provider {
	case ProviderMomento:
		// Through the proxy the collector is umm itself, already 'self'.
		if !c.MomentoProxy {
			if origin := originOf(c.MomentoURL); origin != "" {
				add(origin)
			}
		}
	case ProviderGA4, ProviderGTM:
		scripts = append(scripts, "https://www.googletagmanager.com")
		connects = append(connects, "https://www.google-analytics.com", "https://analytics.google.com", "https://*.google-analytics.com")
		images = append(images, "https://www.google-analytics.com", "https://www.googletagmanager.com")
	case ProviderMatomo:
		if origin := originOf(c.MatomoURL); origin != "" {
			add(origin)
		}
	case ProviderCustom:
		// A pasted snippet writes down the addresses it loads and reports to, so
		// those are allowed without anyone reading a console error first.
		for _, origin := range SnippetOrigins(c.CustomSnippet) {
			add(origin)
		}
	}
	for _, host := range splitHosts(c.AllowedHosts) {
		add(strings.TrimSuffix(host, "/"))
	}
	return dedupe(scripts), dedupe(connects), dedupe(images)
}

// SnippetOrigins lists every http(s) origin written into a snippet: the script
// it loads, the endpoint it posts to, the pixel it requests. Trackers almost
// always write their own address into their loader, so reading it here is what
// keeps a pasted snippet working without the administrator translating a
// policy error into a host name.
func SnippetOrigins(snippet string) []string {
	origins := make([]string, 0, 2)
	seen := make(map[string]struct{}, 2)
	for index := 0; index < len(snippet); {
		start := indexFold(snippet[index:], "http")
		if start < 0 {
			break
		}
		start += index
		end := start
		for end < len(snippet) && !isURLBoundary(snippet[end]) {
			end++
		}
		index = end
		candidate := snippet[start:end]
		if !hasPrefixFold(candidate, "http://") && !hasPrefixFold(candidate, "https://") {
			continue
		}
		origin := originOf(candidate)
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

// AddAllowedHost appends an origin to the allow list, leaving what is already
// there — and its order — alone. This is the one click behind "allow this".
func AddAllowedHost(existing, origin string) string {
	origin = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(origin), "/"))
	if origin == "" {
		return existing
	}
	for _, host := range splitHosts(existing) {
		if strings.EqualFold(strings.TrimSuffix(host, "/"), origin) {
			return existing
		}
	}
	if strings.TrimSpace(existing) == "" {
		return origin
	}
	return strings.TrimSpace(existing) + ", " + origin
}

// withNonce adds the nonce to every script tag that does not already carry
// one. The snippet is otherwise left exactly as pasted.
func withNonce(snippet, nonce string) string {
	if nonce == "" || snippet == "" {
		return snippet
	}
	var builder strings.Builder
	remaining := snippet
	for {
		index := indexFold(remaining, "<script")
		if index < 0 {
			builder.WriteString(remaining)
			return builder.String()
		}
		end := index + len("<script")
		builder.WriteString(remaining[:end])
		tag := remaining[end:]
		if closing := strings.IndexByte(tag, '>'); closing >= 0 {
			tag = tag[:closing]
		}
		if !containsFold(tag, "nonce=") {
			builder.WriteString(` nonce="` + html.EscapeString(nonce) + `"`)
		}
		remaining = remaining[end:]
	}
}

// indexFold finds sub in s ignoring ASCII case and returns an index into s.
//
// strings.ToLower is the obvious way and the wrong one: it changes byte lengths
// for some runes — U+212A KELVIN SIGN is three bytes and folds to a one-byte
// 'k', U+0130 'İ' is two and folds to three — so an index taken from the folded
// copy lands somewhere else in the original. One such letter anywhere in a
// snippet would put the nonce into the middle of the tag name, and the tag
// would stop being a script tag without a word said. Every needle here is
// ASCII, and folding only ASCII keeps every byte where it is.
func indexFold(s, sub string) int {
	if len(sub) == 0 {
		return 0
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		match := true
		for j := 0; j < len(sub); j++ {
			if foldASCII(s[i+j]) != foldASCII(sub[j]) {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func containsFold(s, sub string) bool { return indexFold(s, sub) >= 0 }

func hasPrefixFold(s, prefix string) bool {
	return len(s) >= len(prefix) && indexFold(s[:len(prefix)], prefix) == 0
}

func foldASCII(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

// isURLBoundary names the bytes that cannot be part of an address written
// inside HTML or JavaScript, which is where each address ends.
func isURLBoundary(letter byte) bool {
	switch letter {
	case '"', '\'', '`', '<', '>', ' ', '\t', '\n', '\r', ')', ',', ';', '\\', '+':
		return true
	}
	return false
}

func originOf(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || parsed.Scheme == "" {
		return ""
	}
	return strings.ToLower(parsed.Scheme) + "://" + strings.ToLower(parsed.Host)
}

func isHTTPURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func splitHosts(list string) []string {
	fields := strings.FieldsFunc(list, func(letter rune) bool {
		return letter == ',' || letter == ' ' || letter == '\n' || letter == '\t' || letter == '\r'
	})
	out := fields[:0]
	for _, field := range fields {
		if trimmed := strings.TrimSpace(field); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func dedupe(list []string) []string {
	if len(list) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(list))
	out := list[:0]
	for _, item := range list {
		key := strings.ToLower(item)
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}
