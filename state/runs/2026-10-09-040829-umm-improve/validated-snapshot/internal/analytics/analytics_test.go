package analytics

import (
	"strings"
	"testing"
	"time"
)

func momento(proxy bool) Config {
	return Config{Enabled: true, Provider: ProviderMomento, MomentoURL: "https://momento.internal/", MomentoSiteID: "umm-prd", MomentoProxy: proxy}
}

func TestOffByDefaultMeansNoSnippetAnywhere(t *testing.T) {
	var zero Config
	for _, path := range []string{"/", "/today", "/space/abc", "/admin", "/login"} {
		if zero.Active(path) {
			t.Fatalf("a zero configuration is active on %s", path)
		}
	}
	// Filled in but not switched on is still off: the switch is the decision.
	configured := momento(true)
	configured.Enabled = false
	if configured.Active("/today") || configured.ProxyActive() {
		t.Fatal("a configured but disabled tracker is active")
	}
	if snippet := zero.Snippet("n"); snippet != "" {
		t.Fatalf("a zero configuration renders %q", snippet)
	}
	scripts, connects, images := zero.PolicySources()
	if len(scripts)+len(connects)+len(images) != 0 {
		t.Fatalf("a zero configuration widens the policy: %v %v %v", scripts, connects, images)
	}
}

func TestMomentoThroughTheProxyNamesNoOutsideOrigin(t *testing.T) {
	config := momento(true)
	snippet := config.Snippet("n0nce")
	for _, want := range []string{`src="/momento/tracker.js"`, `data-endpoint="/momento"`, `data-site-id="umm-prd"`, `nonce="n0nce"`, `data-contract-version="1"`} {
		if !strings.Contains(snippet, want) {
			t.Fatalf("proxy snippet lacks %s: %s", want, snippet)
		}
	}
	if strings.Contains(snippet, "momento.internal") {
		t.Fatalf("the proxied snippet must not point the browser outside umm: %s", snippet)
	}
	scripts, connects, images := config.PolicySources()
	if len(scripts)+len(connects)+len(images) != 0 {
		t.Fatalf("the proxied collector is 'self' already; got %v %v %v", scripts, connects, images)
	}
	if !config.ProxyActive() {
		t.Fatal("the proxy must be on when Momento is proxied")
	}

	direct := momento(false)
	snippet = direct.Snippet("n0nce")
	if !strings.Contains(snippet, `src="https://momento.internal/tracker.js"`) || strings.Contains(snippet, "data-endpoint") {
		t.Fatalf("direct snippet: %s", snippet)
	}
	scripts, connects, _ = direct.PolicySources()
	if len(scripts) != 1 || scripts[0] != "https://momento.internal" || len(connects) != 1 {
		t.Fatalf("the direct collector must be allowed for scripts and connections: %v %v", scripts, connects)
	}
	if direct.ProxyActive() {
		t.Fatal("no proxy without the proxy switch")
	}
}

func TestAdminScreensAreLeftOutUnlessAsked(t *testing.T) {
	config := momento(true)
	if config.Active("/admin") || config.Active("/admin/analytics") {
		t.Fatal("administration screens are tracked without include_admin")
	}
	if !config.Active("/administer") {
		t.Fatal("a page that merely starts with the letters is not the admin screen")
	}
	if !config.Active("/today") || !config.Active("/") || !config.Active("/login") {
		t.Fatal("ordinary pages are not tracked")
	}
	config.IncludeAdmin = true
	if !config.Active("/admin/analytics") {
		t.Fatal("include_admin does not include the admin screens")
	}
}

// The standard's own regression: a letter whose lower-case form has a
// different byte length must not move the nonce into the tag name.
func TestNonceLandsInsideEveryScriptTagWhateverTheLettersAround(t *testing.T) {
	cases := map[string]string{
		"plain":         `<script src="https://t.example/a.js"></script><script>go()</script>`,
		"upper case":    `<SCRIPT src="https://t.example/a.js"></SCRIPT>`,
		"kelvin sign":   "KKKK<script>1</script>",
		"dotted I":      "İİİİ<script>1</script><script>2</script>",
		"already there": `<script nonce="theirs">1</script><script>2</script>`,
	}
	for name, snippet := range cases {
		config := Config{Enabled: true, Provider: ProviderCustom, CustomSnippet: snippet}
		out := config.Snippet("abc")
		opened := strings.Count(strings.ToLower(out), "<script")
		nonced := strings.Count(out, `nonce="abc"`)
		theirs := strings.Count(out, `nonce="theirs"`)
		if opened == 0 || nonced+theirs != opened {
			t.Fatalf("%s: %d script tags, %d carry our nonce, %d theirs: %s", name, opened, nonced, theirs, out)
		}
		if strings.Contains(out, "<sc nonce") || strings.Contains(out, "<SC nonce") {
			t.Fatalf("%s: the nonce landed inside the tag name: %s", name, out)
		}
	}
}

func TestSnippetOriginsAreReadOutOfThePastedCode(t *testing.T) {
	snippet := `<script async src="HTTPS://cdn.tracker.example/loader.js"></script>
<script>fetch('https://collect.tracker.example/e',{method:'POST'});new Image().src="https://cdn.tracker.example/p.gif?x=1"+n;</script>`
	got := SnippetOrigins(snippet)
	want := []string{"https://cdn.tracker.example", "https://collect.tracker.example"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("origins = %v, want %v", got, want)
	}
	config := Config{Enabled: true, Provider: ProviderCustom, CustomSnippet: snippet, AllowedHosts: "https://extra.example/, https://cdn.tracker.example"}
	scripts, connects, images := config.PolicySources()
	for _, group := range [][]string{scripts, connects, images} {
		if strings.Join(group, " ") != "https://cdn.tracker.example https://collect.tracker.example https://extra.example" {
			t.Fatalf("policy sources = %v", group)
		}
	}
}

func TestValidateRefusesWhatWouldSaveAsNothing(t *testing.T) {
	tooLong := Config{Enabled: true, Provider: ProviderCustom, CustomSnippet: "<script>" + strings.Repeat("x", MaxSnippetBytes) + "</script>"}
	if tooLong.Validate() == nil {
		t.Fatal("a snippet over the limit is accepted")
	}
	noScript := Config{Enabled: true, Provider: ProviderCustom, CustomSnippet: "<img src=https://p.example/x.gif>"}
	if noScript.Validate() == nil {
		t.Fatal("a snippet with no script tag is accepted")
	}
	noSite := momento(true)
	noSite.MomentoSiteID = ""
	if noSite.Validate() == nil {
		t.Fatal("Momento without a site id is accepted")
	}
	badHost := momento(true)
	badHost.AllowedHosts = "cdn.example"
	if badHost.Validate() == nil {
		t.Fatal("an allowed host without a scheme is accepted; the policy would not match it")
	}
	unknown := Config{Enabled: true, Provider: "piwik"}
	if unknown.Validate() == nil {
		t.Fatal("an unknown provider is accepted")
	}
	// Off, anything goes but the hard limits — the administrator may be
	// filling the form in before switching it on.
	off := Config{Enabled: false, Provider: ProviderMomento}
	if err := off.Validate(); err != nil {
		t.Fatalf("a disabled, half-filled configuration is refused: %v", err)
	}
	if err := momento(true).Validate(); err != nil {
		t.Fatalf("a complete Momento configuration is refused: %v", err)
	}
}

func TestRecorderKeepsDistinctOriginsNotCounts(t *testing.T) {
	recorder := NewRecorder()
	moment := time.Date(2026, 9, 13, 21, 0, 0, 0, time.UTC)
	recorder.now = func() time.Time { return moment }
	for i := 0; i < 5; i++ {
		recorder.Record("https://cdn.tracker.example/loader.js", "script-src-elem", "/today")
	}
	recorder.Record("https://cdn.tracker.example/other.js", "script-src-elem 'self'", "/today")
	moment = moment.Add(time.Minute)
	recorder.Record("https://collect.tracker.example/e", "connect-src", "/space/x")
	recorder.Record("inline", "script-src", "/today")
	recorder.Record("chrome-extension://abc/x.js", "script-src", "/today")
	recorder.Record("data:image/png;base64,AAAA", "img-src", "/today")

	list := recorder.List(Config{Enabled: true, Provider: ProviderCustom, CustomSnippet: `<script src="https://cdn.tracker.example/loader.js"></script>`})
	if len(list) != 2 {
		t.Fatalf("want two distinct http origins, got %+v", list)
	}
	if list[0].Origin != "https://collect.tracker.example" || list[0].Directive != "connect-src" || list[0].Allowed {
		t.Fatalf("most recent first, and not yet allowed: %+v", list[0])
	}
	if list[1].Origin != "https://cdn.tracker.example" || list[1].Count != 6 || list[1].Directive != "script-src-elem" || !list[1].Allowed {
		t.Fatalf("the repeated origin folds into one entry that the snippet already allows: %+v", list[1])
	}

	for i := 0; i < MaxViolations+10; i++ {
		moment = moment.Add(time.Second)
		recorder.Record("https://h"+strings.Repeat("x", i%7)+string(rune('a'+i%26))+".example/"+strings.Repeat("y", i), "img-src", "/")
	}
	if got := len(recorder.List(Config{})); got > MaxViolations {
		t.Fatalf("the recorder holds %d entries, more than %d", got, MaxViolations)
	}
	recorder.Forget()
	if len(recorder.List(Config{})) != 0 {
		t.Fatal("Forget left entries behind")
	}
}

func TestAddAllowedHostLeavesTheListAlone(t *testing.T) {
	if got := AddAllowedHost("", "https://a.example/"); got != "https://a.example" {
		t.Fatalf("first entry: %q", got)
	}
	if got := AddAllowedHost("https://a.example", "https://b.example"); got != "https://a.example, https://b.example" {
		t.Fatalf("second entry: %q", got)
	}
	if got := AddAllowedHost("https://a.example, https://b.example", "HTTPS://A.example/"); got != "https://a.example, https://b.example" {
		t.Fatalf("a duplicate changed the list: %q", got)
	}
}
