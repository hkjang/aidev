package tracking

import (
	"strings"
	"testing"
	"time"
)

func TestDefaultIsOffAndInjectsNothing(t *testing.T) {
	config := Default()
	if config.Enabled || config.Active() || config.MomentoProxied() {
		t.Fatalf("default configuration is active: %+v", config)
	}
	if err := config.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
	if snippet := config.Snippet("abc"); snippet != "" {
		t.Fatalf("Snippet() = %q, want empty", snippet)
	}
	if sources := config.PolicySources().All(); len(sources) != 0 {
		t.Fatalf("PolicySources() = %v, want none", sources)
	}
}

func TestMomentoProxyNeedsNoExternalOrigin(t *testing.T) {
	config := Default()
	config.Enabled = true
	config.Provider = ProviderMomento
	config.MomentoURL = "https://momento.corp.example/"
	config.MomentoSiteID = "invenqor"
	config = config.Normalize()
	if err := config.Validate(); err != nil {
		t.Fatalf("Validate() = %v", err)
	}
	if !config.MomentoProxied() {
		t.Fatal("proxy is the default for Momento and must be on")
	}
	snippet := config.Snippet("n0nce")
	for _, want := range []string{
		`src="/momento/tracker.js"`, `data-endpoint="/momento"`,
		`data-site-id="invenqor"`, `data-environment="prd"`,
		`data-contract-version="1"`, `nonce="n0nce"`,
	} {
		if !strings.Contains(snippet, want) {
			t.Errorf("snippet %q lacks %q", snippet, want)
		}
	}
	if strings.Contains(snippet, "momento.corp.example") {
		t.Errorf("proxied snippet names the collector: %q", snippet)
	}
	if sources := config.PolicySources().All(); len(sources) != 0 {
		t.Fatalf("proxied Momento added origins to the policy: %v", sources)
	}

	config.MomentoProxy = false
	if config.MomentoProxied() {
		t.Fatal("MomentoProxied() must follow momento_proxy")
	}
	direct := config.Snippet("n0nce")
	if !strings.Contains(direct, `src="https://momento.corp.example/tracker.js"`) ||
		strings.Contains(direct, "data-endpoint") {
		t.Errorf("direct snippet = %q", direct)
	}
	sources := config.PolicySources()
	for _, group := range [][]string{sources.Scripts, sources.Connects, sources.Images} {
		if len(group) != 1 || group[0] != "https://momento.corp.example" {
			t.Errorf("direct Momento sources = %v", group)
		}
	}
}

func TestEveryScriptTagCarriesTheNonce(t *testing.T) {
	config := Default()
	config.Enabled = true
	config.Provider = ProviderCustom
	config.CustomSnippet = `<SCRIPT src="https://t.corp.example/t.js"></SCRIPT>
<script nonce="keep">window.__t=1;</script>
<script>window.__u="https://u.corp.example/collect";</script>`
	snippet := config.Normalize().Snippet("abc")
	if got := strings.Count(snippet, `nonce="abc"`); got != 2 {
		t.Fatalf("nonce added %d times, want 2:\n%s", got, snippet)
	}
	if !strings.Contains(snippet, `nonce="keep"`) {
		t.Fatalf("existing nonce was replaced:\n%s", snippet)
	}
	origins := SnippetOrigins(config.CustomSnippet)
	if len(origins) != 2 || origins[0] != "https://t.corp.example" ||
		origins[1] != "https://u.corp.example" {
		t.Fatalf("SnippetOrigins() = %v", origins)
	}
}

func TestSnippetSizeAndAllowedHostsAreRefused(t *testing.T) {
	config := Default()
	config.CustomSnippet = strings.Repeat("x", MaxSnippetBytes+1)
	if err := config.Validate(); err == nil {
		t.Fatal("a snippet over 8KB was accepted while tracking is off")
	}
	config = Default()
	for _, hostile := range []string{
		"'unsafe-inline'", "https://a.example 'unsafe-inline'", "a.example",
		"https://a.example/path", "https://user@a.example", "data:",
	} {
		config.AllowedHosts = []string{hostile}
		if err := config.Normalize().Validate(); err == nil {
			t.Errorf("allowed_hosts %q was accepted", hostile)
		}
	}
	config.AllowedHosts = []string{
		"https://a.example/", "https://*.b.example", "http://c.example:8080",
		"https://A.example",
	}
	config = config.Normalize()
	if err := config.Validate(); err != nil {
		t.Fatalf("Validate() = %v", err)
	}
	if len(config.AllowedHosts) != 3 {
		t.Fatalf("AllowedHosts = %v, want 3 deduplicated entries", config.AllowedHosts)
	}
}

func TestEnabledProvidersNeedTheirIdentifiers(t *testing.T) {
	cases := map[string]Config{
		"none":    {Enabled: true, Provider: ProviderNone},
		"momento": {Enabled: true, Provider: ProviderMomento, MomentoURL: "https://m.example"},
		"ga4":     {Enabled: true, Provider: ProviderGA4},
		"matomo":  {Enabled: true, Provider: ProviderMatomo, MatomoSiteID: "1"},
		"custom":  {Enabled: true, Provider: ProviderCustom},
		"unknown": {Provider: "piwik"},
	}
	for name, config := range cases {
		if err := config.Normalize().Validate(); err == nil {
			t.Errorf("%s: incomplete configuration was accepted", name)
		}
	}
	matomo := Config{
		Enabled: true, Provider: ProviderMatomo,
		MatomoURL: "https://matomo.corp.example/", MatomoSiteID: "7",
	}.Normalize()
	if err := matomo.Validate(); err != nil {
		t.Fatalf("Matomo Validate() = %v", err)
	}
	if !strings.Contains(matomo.Snippet("n"), `u="https://matomo.corp.example/"`) {
		t.Fatalf("Matomo snippet = %q", matomo.Snippet("n"))
	}
	if sources := matomo.PolicySources().Scripts; len(sources) != 1 ||
		sources[0] != "https://matomo.corp.example" {
		t.Fatalf("Matomo scripts = %v", sources)
	}
}

func TestInjectSnippetHonoursPlacement(t *testing.T) {
	page := []byte("<html><head><title>x</title></head><body><div></div></body></html>")
	head := string(InjectSnippet(page, "<script>1</script>", "head"))
	if !strings.Contains(head, "<script>1</script>\n</head>") {
		t.Fatalf("head placement = %q", head)
	}
	body := string(InjectSnippet(page, "<script>1</script>", "body"))
	if !strings.Contains(body, "<script>1</script>\n</body>") {
		t.Fatalf("body placement = %q", body)
	}
	bare := string(InjectSnippet([]byte("plain"), "<script>1</script>", "head"))
	if !strings.HasSuffix(bare, "<script>1</script>\n") {
		t.Fatalf("fallback placement = %q", bare)
	}
}

func TestRecorderKeepsDistinctOriginsNotCounts(t *testing.T) {
	recorder := NewRecorder()
	clock := time.Date(2026, 9, 12, 8, 0, 0, 0, time.UTC)
	recorder.now = func() time.Time { return clock }
	for range 5 {
		recorder.Record("https://momento.corp.example/collect/v1/events", "connect-src", "/")
	}
	recorder.Record("chrome-extension://abc/x.js", "script-src", "/")
	recorder.Record("data", "img-src", "/")
	items := recorder.List(Default())
	if len(items) != 1 || items[0].Origin != "https://momento.corp.example" ||
		items[0].Count != 5 || items[0].Directive != "connect-src" || items[0].Allowed {
		t.Fatalf("List() = %+v", items)
	}

	// The oldest distinct origin leaves when the buffer is full.
	for index := range MaxViolations {
		clock = clock.Add(time.Second)
		recorder.Record("https://h"+strings.Repeat("x", index%7)+string(rune('a'+index%26))+".example/p", "script-src", "/")
	}
	if got := len(recorder.List(Default())); got != MaxViolations {
		t.Fatalf("recorder holds %d entries, want %d", got, MaxViolations)
	}
	for _, item := range recorder.List(Default()) {
		if item.Origin == "https://momento.corp.example" {
			t.Fatal("the oldest origin survived eviction")
		}
	}

	// An origin the configuration already allows is marked, wildcard included.
	recorder.Forget()
	recorder.Record("https://momento.corp.example/collect", "connect-src", "/")
	recorder.Record("https://region1.google-analytics.com/g/collect", "connect-src", "/")
	recorder.Record("https://other.example/x.js", "script-src", "/")
	config := Config{
		Enabled: true, Provider: ProviderGA4, MeasurementID: "G-1",
		AllowedHosts: []string{"https://momento.corp.example"},
	}.Normalize()
	allowed := map[string]bool{}
	for _, item := range recorder.List(config) {
		allowed[item.Origin] = item.Allowed
	}
	if !allowed["https://momento.corp.example"] ||
		!allowed["https://region1.google-analytics.com"] ||
		allowed["https://other.example"] {
		t.Fatalf("Allowed marks = %v", allowed)
	}
}

func TestAddAllowedHostIsIdempotent(t *testing.T) {
	hosts := AddAllowedHost(nil, "https://a.example/")
	hosts = AddAllowedHost(hosts, "https://A.example")
	hosts = AddAllowedHost(hosts, "https://b.example")
	hosts = AddAllowedHost(hosts, "  ")
	if len(hosts) != 2 || hosts[0] != "https://a.example" || hosts[1] != "https://b.example" {
		t.Fatalf("AddAllowedHost() = %v", hosts)
	}
}
