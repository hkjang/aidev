package tracking

import (
	"strings"
	"testing"
	"time"
)

func settings(extra map[string]string) map[string]string {
	values := map[string]string{"tracking.enabled": "true", "tracking.momento_url": "https://momento.intra", "tracking.momento_site_id": "SITE_1"}
	for key, value := range extra {
		values[key] = value
	}
	return values
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func TestDefaultsAreOff(t *testing.T) {
	config := ReadConfig(map[string]string{})
	if config.Enabled || config.Active("/") {
		t.Fatal("아무 설정도 없으면 꺼져 있어야 한다")
	}
	if config.Provider != ProviderMomento || !config.MomentoProxy || config.Placement != "head" {
		t.Fatalf("기본 provider 는 momento·프록시 켬·head 여야 한다: %+v", config)
	}
	if err := config.Validate(); err != nil {
		t.Fatalf("꺼진 설정은 비어 있어도 저장할 수 있어야 한다: %v", err)
	}
	// 켜기만 하고 provider 를 none 으로 두면 여전히 아무것도 붙지 않는다.
	if ReadConfig(map[string]string{"tracking.enabled": "true", "tracking.provider": "none"}).Active("/") {
		t.Fatal("provider none 은 켜도 붙지 않아야 한다")
	}
}

func TestMomentoProxySnippetHasNoExternalOrigin(t *testing.T) {
	config := ReadConfig(settings(nil))
	snippet := config.Snippet("n0nce")
	for _, want := range []string{`src="/momento/tracker.js"`, `data-endpoint="/momento"`, `data-site-id="SITE_1"`, `nonce="n0nce"`} {
		if !strings.Contains(snippet, want) {
			t.Fatalf("프록시 스니펫에 %s 가 없다: %s", want, snippet)
		}
	}
	if strings.Contains(snippet, "momento.intra") {
		t.Fatalf("프록시를 쓰면 스니펫에 수집기 주소가 나오지 않아야 한다: %s", snippet)
	}
	scripts, connects, images := config.PolicySources()
	if len(scripts)+len(connects)+len(images) != 0 {
		t.Fatalf("프록시를 쓰면 정책에 더할 출처가 없어야 한다: %v %v %v", scripts, connects, images)
	}
	if target := config.ProxyTarget(); target == nil || target.String() != "https://momento.intra" {
		t.Fatalf("프록시 대상은 수집기 주소여야 한다: %v", target)
	}
}

func TestMomentoDirectSnippetAddsOrigin(t *testing.T) {
	config := ReadConfig(settings(map[string]string{"tracking.momento_proxy": "false", "tracking.momento_url": "https://Momento.intra:8443/"}))
	snippet := config.Snippet("n")
	if !strings.Contains(snippet, `src="https://Momento.intra:8443/tracker.js"`) || strings.Contains(snippet, "data-endpoint") {
		t.Fatalf("직접 연결 스니펫은 수집기의 tracker.js 를 읽고 endpoint 를 덮어쓰지 않아야 한다: %s", snippet)
	}
	scripts, connects, _ := config.PolicySources()
	if !contains(scripts, "https://momento.intra:8443") || !contains(connects, "https://momento.intra:8443") {
		t.Fatalf("직접 연결이면 수집기 출처가 script-src·connect-src 에 들어가야 한다: %v %v", scripts, connects)
	}
	if config.ProxyTarget() != nil {
		t.Fatal("프록시를 끄면 /momento/* 는 없어야 한다")
	}
}

func TestValidateRequiresProviderFields(t *testing.T) {
	cases := map[string]map[string]string{
		"momento 사이트 없음": settings(map[string]string{"tracking.momento_site_id": ""}),
		"momento 주소 없음":  settings(map[string]string{"tracking.momento_url": ""}),
		"momento 주소 오류":  settings(map[string]string{"tracking.momento_url": "momento.intra"}),
		"ga4 id 없음":      {"tracking.enabled": "true", "tracking.provider": "ga4"},
		"matomo 주소 없음":   {"tracking.enabled": "true", "tracking.provider": "matomo", "tracking.matomo_site_id": "1"},
		"custom 비어 있음":   {"tracking.enabled": "true", "tracking.provider": "custom"},
		"모르는 provider":   {"tracking.enabled": "false", "tracking.provider": "piwik"},
	}
	for name, values := range cases {
		if err := ReadConfig(values).Validate(); err == nil {
			t.Errorf("%s: 저장이 거절되어야 한다", name)
		}
	}
	if err := ReadConfig(settings(nil)).Validate(); err != nil {
		t.Fatalf("갖춘 momento 설정은 통과해야 한다: %v", err)
	}
}

func TestOversizedSnippetIsRejectedEvenWhenOff(t *testing.T) {
	big := strings.Repeat("a", MaxSnippetBytes+1)
	config := ReadConfig(map[string]string{"tracking.provider": "custom", "tracking.custom_snippet": big})
	if err := config.Validate(); err == nil {
		t.Fatal("8KB 를 넘는 스니펫은 저장되지 않아야 한다")
	}
	config = ReadConfig(map[string]string{"tracking.provider": "custom", "tracking.custom_snippet": big[:MaxSnippetBytes]})
	if err := config.Validate(); err != nil {
		t.Fatalf("딱 8KB 는 허용해야 한다: %v", err)
	}
}

func TestNonceOnEveryScriptTag(t *testing.T) {
	config := ReadConfig(map[string]string{"tracking.enabled": "true", "tracking.provider": "custom", "tracking.custom_snippet": `<SCRIPT src="https://t.intra/a.js"></SCRIPT>
<script nonce="keep">1</script>
<script>2</script>`})
	got := config.Snippet("abc")
	if strings.Count(got, `nonce="abc"`) != 2 || !strings.Contains(got, `nonce="keep"`) {
		t.Fatalf("nonce 가 없는 태그 둘에만 붙고 이미 있는 것은 그대로여야 한다: %s", got)
	}
	if !strings.HasPrefix(got, `<SCRIPT nonce="abc" src=`) {
		t.Fatalf("대문자 태그도 찾아야 한다: %s", got)
	}
}

func TestNonceSurvivesCaseFoldingRunes(t *testing.T) {
	// U+0130 은 접으면 길어지고 U+212A 는 짧아진다. ToLower 의 인덱스를 원본에
	// 쓰면 nonce 가 태그 이름 한가운데 박힌다.
	for _, prefix := range []string{"İİİİ", "KKKK", "KK"} {
		config := ReadConfig(map[string]string{"tracking.enabled": "true", "tracking.provider": "custom", "tracking.custom_snippet": prefix + `<script>1</script>`})
		got := config.Snippet("n")
		if got != prefix+`<script nonce="n">1</script>` {
			t.Fatalf("%q 뒤의 태그가 깨졌다: %s", prefix, got)
		}
	}
}

func TestSnippetOriginsAndPolicySources(t *testing.T) {
	snippet := `<script src="HTTPS://Tracker.intra/t.js"></script>
<script>window.__t={endpoint:"https://tracker.intra/collect",pixel:'http://pixel.intra:8080/p.gif?id=1'};fetch("data:text/plain,x")</script>`
	origins := SnippetOrigins(snippet)
	if len(origins) != 2 || origins[0] != "https://tracker.intra" || origins[1] != "http://pixel.intra:8080" {
		t.Fatalf("스니펫의 http(s) 출처를 중복 없이 순서대로 읽어야 한다: %v", origins)
	}
	config := ReadConfig(map[string]string{"tracking.enabled": "true", "tracking.provider": "custom", "tracking.custom_snippet": snippet, "tracking.allowed_hosts": "https://extra.intra,\n https://tracker.intra"})
	scripts, connects, images := config.PolicySources()
	for _, group := range [][]string{scripts, connects, images} {
		if !contains(group, "https://tracker.intra") || !contains(group, "http://pixel.intra:8080") || !contains(group, "https://extra.intra") {
			t.Fatalf("스니펫 출처와 allowed_hosts 가 모두 정책에 들어가야 한다: %v", group)
		}
	}
}

func TestAdminPathsExcludedUnlessIncluded(t *testing.T) {
	config := ReadConfig(settings(nil))
	if !config.Active("/") || !config.Active("/login") {
		t.Fatal("일반 화면에는 붙어야 한다")
	}
	if config.Active("/admin/settings") || config.Active("/profile/keys") {
		t.Fatal("include_admin 이 꺼져 있으면 관리 화면에 붙지 않아야 한다")
	}
	config = ReadConfig(settings(map[string]string{"tracking.include_admin": "true"}))
	if !config.Active("/admin/settings") {
		t.Fatal("include_admin 을 켜면 관리 화면에도 붙어야 한다")
	}
}

func TestInjectPlacement(t *testing.T) {
	document := []byte("<!doctype html><html><HEAD><title>x</title></HEAD><body><div id=\"root\"></div></body></html>")
	head := string(Inject(document, `<script nonce="n">1</script>`, "head"))
	if !strings.Contains(head, `<script nonce="n">1</script>`+"\n</HEAD>") {
		t.Fatalf("head 는 </head> 바로 앞(대소문자 무시)이어야 한다: %s", head)
	}
	body := string(Inject(document, `<script>2</script>`, "body"))
	if !strings.Contains(body, `</div><script>2</script>`+"\n</body>") {
		t.Fatalf("body 는 </body> 바로 앞이어야 한다: %s", body)
	}
	if got := string(Inject([]byte("plain"), "<script>3</script>", "head")); got != "plain<script>3</script>\n" {
		t.Fatalf("닫는 태그가 없으면 끝에 붙인다: %q", got)
	}
	if got := Inject(document, "  ", "head"); string(got) != string(document) {
		t.Fatal("빈 스니펫은 문서를 바꾸지 않아야 한다")
	}
}

func TestAddAllowedHost(t *testing.T) {
	if got := AddAllowedHost("", "https://a.intra/"); got != "https://a.intra" {
		t.Fatalf("빈 목록: %q", got)
	}
	if got := AddAllowedHost("https://a.intra", "https://b.intra"); got != "https://a.intra, https://b.intra" {
		t.Fatalf("이어 붙이기: %q", got)
	}
	if got := AddAllowedHost("https://a.intra, https://b.intra", "HTTPS://A.intra"); got != "https://a.intra, https://b.intra" {
		t.Fatalf("이미 있으면 그대로: %q", got)
	}
}

func TestRecorderKeepsDistinctOrigins(t *testing.T) {
	recorder := NewRecorder()
	clock := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	recorder.now = func() time.Time { return clock }
	for i := 0; i < 5; i++ {
		recorder.Record("https://momento.intra/collect/v1/events", "connect-src", "/")
	}
	recorder.Record("chrome-extension://abc/inject.js", "script-src-elem", "/")
	recorder.Record("data:text/plain,x", "img-src", "/")
	clock = clock.Add(time.Minute)
	recorder.Record("https://momento.intra/tracker.js", "script-src-elem 'self'", "/login")
	items := recorder.List(ReadConfig(map[string]string{}))
	if len(items) != 2 {
		t.Fatalf("http 출처 둘만 남아야 한다: %+v", items)
	}
	if items[0].Origin != "https://momento.intra" || items[0].Directive != "script-src-elem" || items[0].Page != "/login" {
		t.Fatalf("최근 것이 먼저이고 지시어는 첫 단어만: %+v", items[0])
	}
	if items[1].Count != 5 || items[1].Directive != "connect-src" {
		t.Fatalf("같은 출처는 횟수만 늘어야 한다: %+v", items[1])
	}
}

func TestRecorderMarksAllowedAndEvicts(t *testing.T) {
	recorder := NewRecorder()
	recorder.Record("https://momento.intra/collect", "connect-src", "/")
	recorder.Record("https://www.google-analytics.com/g/collect", "connect-src", "/")
	config := ReadConfig(settings(map[string]string{"tracking.momento_proxy": "false", "tracking.allowed_hosts": "https://www.googletagmanager.com"}))
	items := recorder.List(config)
	for _, item := range items {
		if item.Origin == "https://momento.intra" && !item.Allowed {
			t.Fatal("설정이 이미 허용하는 출처는 allowed 로 표시되어야 한다")
		}
		if item.Origin == "https://www.google-analytics.com" && item.Allowed {
			t.Fatal("허용하지 않은 출처가 allowed 로 표시되면 안 된다")
		}
	}
	ga := ReadConfig(map[string]string{"tracking.enabled": "true", "tracking.provider": "ga4", "tracking.measurement_id": "G-1"})
	recorder.Record("https://region1.google-analytics.com/g/collect", "connect-src", "/")
	for _, item := range recorder.List(ga) {
		if strings.HasSuffix(item.Origin, "google-analytics.com") && !item.Allowed {
			t.Fatalf("와일드카드 항목이 덮는 출처는 allowed 여야 한다: %+v", item)
		}
	}
	recorder.Forget()
	for i := 0; i < MaxViolations+10; i++ {
		recorder.Record("https://h"+strings.Repeat("x", i%7)+strings.Repeat("y", i/7)+".intra/p", "img-src", "/")
	}
	if got := len(recorder.List(ga)); got != MaxViolations {
		t.Fatalf("기록은 %d개를 넘지 않아야 한다: %d", MaxViolations, got)
	}
}
