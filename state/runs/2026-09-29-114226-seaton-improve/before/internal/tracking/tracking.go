// Package tracking 은 관리자가 화면에서 붙인 방문 추적 스니펫을 서비스되는
// 문서에 넣고, 그 스니펫이 돌 수 있게 콘텐츠 보안 정책(CSP)에 필요한 출처를
// 계산한다.
//
// SeatOn 의 정책은 script-src 'self' 로 잠겨 있어 스니펫을 그냥 붙이면 조용히
// 차단된다. 이 패키지는 답의 두 반쪽 — 넣을 마크업과 정책에 더할 출처 — 을
// 함께 만들고, 요청마다 nonce 를 달아 'unsafe-inline' 없이 인라인 코드를 허용한다.
package tracking

import (
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

	// MaxSnippetBytes 는 붙여넣은 스니펫의 상한이다. 추적 로더는 몇백 바이트면
	// 충분하고, 그보다 훨씬 큰 것은 스니펫이 아니라 다른 무엇이다.
	MaxSnippetBytes = 8 * 1024

	// ProxyPrefix 는 같은 오리진 프록시 경로다. Momento 스니펫에
	// data-endpoint 로 주면 외부 출처가 정책에 등장하지 않는다.
	ProxyPrefix = "/momento"

	// KeyPrefix 는 settings 테이블에서 이 기능의 키가 시작하는 문자열이다.
	KeyPrefix = "tracking."
)

// Config 는 settings 테이블의 tracking.* 키를 읽은 결과다.
type Config struct {
	Enabled       bool
	Provider      string
	MomentoURL    string
	MomentoSiteID string
	MomentoProxy  bool
	MeasurementID string
	MatomoURL     string
	MatomoSiteID  string
	CustomSnippet string
	AllowedHosts  string
	IncludeAdmin  bool
	Placement     string
}

// ReadConfig 는 설정 키·값 맵을 Config 로 옮긴다. 모든 값은 문자열로 저장되므로
// 불리언은 "true" 만 참으로 본다.
func ReadConfig(values map[string]string) Config {
	get := func(key, fallback string) string {
		if value := strings.TrimSpace(values[KeyPrefix+key]); value != "" {
			return value
		}
		return fallback
	}
	config := Config{
		Enabled:       get("enabled", "false") == "true",
		Provider:      strings.ToLower(get("provider", ProviderMomento)),
		MomentoURL:    get("momento_url", ""),
		MomentoSiteID: get("momento_site_id", ""),
		MomentoProxy:  get("momento_proxy", "true") == "true",
		MeasurementID: get("measurement_id", ""),
		MatomoURL:     get("matomo_url", ""),
		MatomoSiteID:  get("matomo_site_id", ""),
		CustomSnippet: strings.TrimSpace(values[KeyPrefix+"custom_snippet"]),
		AllowedHosts:  get("allowed_hosts", ""),
		IncludeAdmin:  get("include_admin", "false") == "true",
		Placement:     strings.ToLower(get("placement", "head")),
	}
	if config.Placement != "body" {
		config.Placement = "head"
	}
	return config
}

// Active 는 이 경로의 문서에 스니펫을 실을지 답한다. 관리 화면과 개인 설정은
// 관리자가 명시적으로 켜지 않는 한 뺀다 — 관리자 몇 사람의 조작은 누구도
// 보고 싶은 방문 데이터가 아니다.
func (c Config) Active(path string) bool {
	if !c.Enabled || c.Provider == ProviderNone || c.Provider == "" {
		return false
	}
	if !c.IncludeAdmin && (strings.HasPrefix(path, "/admin") || strings.HasPrefix(path, "/profile")) {
		return false
	}
	return strings.TrimSpace(c.Snippet("")) != ""
}

// Validate 는 고른 provider 에 빠진 것을 알려 준다. 꺼져 있으면 무엇이 비어 있든
// 저장을 막지 않는다 — 켜기 전에 채워 가는 것이 자연스러운 순서다.
func (c Config) Validate() error {
	if len(c.CustomSnippet) > MaxSnippetBytes {
		return fmt.Errorf("추적 스니펫은 %d바이트를 넘을 수 없습니다", MaxSnippetBytes)
	}
	switch c.Provider {
	case ProviderNone, ProviderMomento, ProviderGA4, ProviderGTM, ProviderMatomo, ProviderCustom:
	default:
		return fmt.Errorf("tracking.provider 는 momento, ga4, gtm, matomo, custom, none 중 하나여야 합니다")
	}
	if !c.Enabled {
		return nil
	}
	switch c.Provider {
	case ProviderMomento:
		if c.MomentoSiteID == "" {
			return fmt.Errorf("tracking.momento_site_id 가 필요합니다")
		}
		if originOf(c.MomentoURL) == "" {
			return fmt.Errorf("tracking.momento_url 은 http(s) 주소여야 합니다")
		}
	case ProviderGA4, ProviderGTM:
		if c.MeasurementID == "" {
			return fmt.Errorf("tracking.measurement_id 가 필요합니다")
		}
	case ProviderMatomo:
		if c.MatomoSiteID == "" {
			return fmt.Errorf("tracking.matomo_site_id 가 필요합니다")
		}
		if originOf(c.MatomoURL) == "" {
			return fmt.Errorf("tracking.matomo_url 은 http(s) 주소여야 합니다")
		}
	case ProviderCustom:
		if c.CustomSnippet == "" {
			return fmt.Errorf("tracking.custom_snippet 이 비어 있습니다")
		}
	}
	return nil
}

// ProxyTarget 은 같은 오리진 프록시가 넘겨야 할 Momento 수집기 주소다. 프록시를
// 쓰지 않는 구성이면 nil — 그때 /momento/* 는 존재하지 않는 경로다.
func (c Config) ProxyTarget() *url.URL {
	if !c.Enabled || c.Provider != ProviderMomento || !c.MomentoProxy {
		return nil
	}
	parsed, err := url.Parse(strings.TrimSpace(c.MomentoURL))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawQuery, parsed.Fragment = "", ""
	return parsed
}

// Snippet 은 문서에 넣을 마크업이다. 스니펫의 모든 script 태그에 nonce 가 붙어
// 정책은 그대로 엄격하게 남는다.
func (c Config) Snippet(nonce string) string {
	switch c.Provider {
	case ProviderMomento:
		site := html.EscapeString(strings.TrimSpace(c.MomentoSiteID))
		base := strings.TrimRight(strings.TrimSpace(c.MomentoURL), "/")
		if site == "" || originOf(base) == "" {
			return ""
		}
		if c.MomentoProxy {
			// 로더와 수집 요청이 모두 이 앱의 오리진을 지나므로 정책에 더할
			// 출처가 없다. 정책을 바꿀 수 없는 설치에서도 그대로 동작한다.
			return withNonce(fmt.Sprintf(`<script async src="%s/tracker.js" data-site-id="%s" data-environment="prd" data-contract-version="1" data-endpoint="%s"></script>`, ProxyPrefix, site, ProxyPrefix), nonce)
		}
		return withNonce(fmt.Sprintf(`<script async src="%s/tracker.js" data-site-id="%s" data-environment="prd" data-contract-version="1"></script>`, html.EscapeString(base), site), nonce)
	case ProviderGA4:
		id := html.EscapeString(strings.TrimSpace(c.MeasurementID))
		if id == "" {
			return ""
		}
		return withNonce(fmt.Sprintf(`<script async src="https://www.googletagmanager.com/gtag/js?id=%s"></script>
<script>window.dataLayer=window.dataLayer||[];function gtag(){dataLayer.push(arguments);}gtag('js',new Date());gtag('config','%s');</script>`, id, id), nonce)
	case ProviderGTM:
		id := html.EscapeString(strings.TrimSpace(c.MeasurementID))
		if id == "" {
			return ""
		}
		return withNonce(fmt.Sprintf(`<script>(function(w,d,s,l,i){w[l]=w[l]||[];w[l].push({'gtm.start':new Date().getTime(),event:'gtm.js'});var f=d.getElementsByTagName(s)[0],j=d.createElement(s),dl=l!='dataLayer'?'&l='+l:'';j.async=true;j.src='https://www.googletagmanager.com/gtm.js?id='+i+dl;f.parentNode.insertBefore(j,f);})(window,document,'script','dataLayer','%s');</script>`, id), nonce)
	case ProviderMatomo:
		base := strings.TrimRight(strings.TrimSpace(c.MatomoURL), "/")
		site := html.EscapeString(strings.TrimSpace(c.MatomoSiteID))
		if originOf(base) == "" || site == "" {
			return ""
		}
		return withNonce(fmt.Sprintf(`<script>var _paq=window._paq=window._paq||[];_paq.push(['trackPageView']);_paq.push(['enableLinkTracking']);(function(){var u="%s/";_paq.push(['setTrackerUrl',u+'matomo.php']);_paq.push(['setSiteId','%s']);var d=document,g=d.createElement('script'),s=d.getElementsByTagName('script')[0];g.async=true;g.src=u+'matomo.js';s.parentNode.insertBefore(g,s);})();</script>`, html.EscapeString(base), site), nonce)
	case ProviderCustom:
		return withNonce(strings.TrimSpace(c.CustomSnippet), nonce)
	}
	return ""
}

// PolicySources 는 스니펫이 돌기 위해 정책에 더해야 할 출처다. provider 에서
// 아는 것은 자동으로, 붙여넣은 스니펫은 그 안의 주소를 읽어서, 나머지는
// 관리자가 적은 allowed_hosts 에서 온다.
func (c Config) PolicySources() (scripts, connects, images []string) {
	add := func(origin string) {
		scripts = append(scripts, origin)
		connects = append(connects, origin)
		images = append(images, origin)
	}
	switch c.Provider {
	case ProviderMomento:
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
		for _, origin := range SnippetOrigins(c.CustomSnippet) {
			add(origin)
		}
	}
	for _, host := range splitHosts(c.AllowedHosts) {
		add(host)
	}
	return scripts, connects, images
}

// SnippetOrigins 는 스니펫에 적힌 모든 http(s) 출처다 — 로더가 읽는 스크립트,
// 보고를 보내는 끝점, 픽셀. 추적 도구는 거의 항상 자기 주소를 로더 안에 적어
// 두므로 여기서 읽으면 관리자가 정책 오류를 호스트 이름으로 번역할 일이 없다.
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

// AddAllowedHost 는 쉼표로 구분한 허용 목록에 출처 하나를 더한다. 이미 있으면
// 그대로 두고, 기존 항목의 순서는 바꾸지 않는다.
func AddAllowedHost(existing, origin string) string {
	origin = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(origin), "/"))
	if origin == "" {
		return existing
	}
	for _, host := range splitHosts(existing) {
		if strings.EqualFold(host, origin) {
			return existing
		}
	}
	if strings.TrimSpace(existing) == "" {
		return origin
	}
	return strings.TrimSpace(existing) + ", " + origin
}

func splitHosts(list string) []string {
	fields := strings.FieldsFunc(list, func(letter rune) bool {
		return letter == ',' || letter == ' ' || letter == '\n' || letter == '\r' || letter == '\t'
	})
	hosts := fields[:0]
	for _, field := range fields {
		if trimmed := strings.TrimSpace(field); trimmed != "" {
			hosts = append(hosts, trimmed)
		}
	}
	return hosts
}

// Inject 는 문서의 placement 자리에 스니펫을 넣는다. head 는 </head> 바로 앞,
// body 는 </body> 바로 앞이다. 그 태그가 없으면(HTML 이 아니거나 잘린 문서)
// 끝에 붙인다 — 붙이지 않는 것보다는 눈에 띄는 쪽이 낫다.
func Inject(document []byte, snippet, placement string) []byte {
	if strings.TrimSpace(snippet) == "" {
		return document
	}
	closing := "</head>"
	if placement == "body" {
		closing = "</body>"
	}
	text := string(document)
	index := indexFold(text, closing)
	if index < 0 {
		index = len(text)
	}
	var builder strings.Builder
	builder.Grow(len(text) + len(snippet) + 2)
	builder.WriteString(text[:index])
	builder.WriteString(snippet)
	builder.WriteString("\n")
	builder.WriteString(text[index:])
	return []byte(builder.String())
}

// withNonce 는 아직 nonce 가 없는 모든 script 태그에 nonce 를 붙인다. 붙여넣은
// 스니펫이 엄격한 정책 아래에서 손대지 않고 돌게 하는 것이 이것이다.
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

// indexFold 는 ASCII 대소문자만 무시하고 sub 를 찾아 s 의 인덱스를 돌려준다.
//
// strings.ToLower 에서 얻은 인덱스를 원본에 쓰면 안 된다. 접을 때 바이트 길이가
// 달라지는 글자가 있다 — U+212A 켈빈 기호는 3바이트가 1바이트 'k' 로, U+0130 'İ'
// 는 2바이트가 3바이트로. 그런 글자가 스니펫에 하나만 있어도 그 뒤의 nonce 가
// 태그 이름 한가운데 박혀 추적이 조용히 멎는다. 찾는 문자열은 모두 ASCII 이므로
// ASCII 만 접으면 모든 바이트가 제자리에 있다.
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

func foldASCII(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

// isURLBoundary 는 HTML 이나 JavaScript 안에 적힌 URL 이 끝나는 글자다.
func isURLBoundary(letter byte) bool {
	switch letter {
	case '"', '\'', '`', '<', '>', ' ', '\t', '\n', '\r', ')', ',', ';', '\\', '+':
		return true
	}
	return false
}

// originOf 는 http(s) 주소의 scheme://host[:port] 를 돌려주고, 그 밖의 것은
// 빈 문자열이다. 정책에 넣을 수 있는 출처는 그것뿐이다.
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
