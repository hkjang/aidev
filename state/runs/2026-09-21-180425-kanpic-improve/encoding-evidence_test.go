package external

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"kanpic/internal/formula"
	"kanpic/internal/importexport"
)

type fixedSettings map[string]any

func (f fixedSettings) Values(context.Context) (map[string]any, error) { return f, nil }

// httptest 서버는 루프백에 있다. 시험만 그것을 허용하고, 정책이 루프백을 막는
// 것은 따로 본다.
func testFetcher(settings fixedSettings) *Fetcher {
	fetcher := New(settings, nil)
	fetcher.allowPrivate = true
	return fetcher
}

func serve(t *testing.T, handler http.HandlerFunc) (*httptest.Server, string) {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)
	host, _, _ := net.SplitHostPort(strings.TrimPrefix(server.URL, "https://"))
	return server, host
}

func withInsecureTLS(fetcher *Fetcher, server *httptest.Server) *Fetcher {
	// 시험 서버의 자체 서명 인증서를 받아들인다. 운영에서는 시스템 신뢰 저장소를 쓴다.
	fetcher.tlsClient = server.Client()
	return fetcher
}

func TestOffByDefaultAndOnlyAllowedHosts(t *testing.T) {
	server, host := serve(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("hello")) })
	request := formula.ExternalRequest{Function: "WEBSERVICE", URL: server.URL + "/x"}
	key := formula.ExternalKey(request.Function, request.URL)

	off := withInsecureTLS(testFetcher(fixedSettings{"external.enabled": false, "external.allowed_hosts": []any{host}}), server)
	if got := off.Resolve(context.Background(), []formula.ExternalRequest{request})[key]; got.Err == nil || !strings.Contains(got.Err.Message, "꺼져") {
		t.Fatalf("꺼져 있으면 거절해야 한다: %+v", got)
	}
	notListed := withInsecureTLS(testFetcher(fixedSettings{"external.enabled": true, "external.allowed_hosts": []any{"api.example.com"}}), server)
	if got := notListed.Resolve(context.Background(), []formula.ExternalRequest{request})[key]; got.Err == nil || !strings.Contains(got.Err.Message, "허용되지 않은 호스트") {
		t.Fatalf("목록에 없는 호스트는 거절해야 한다: %+v", got)
	}
	listed := withInsecureTLS(testFetcher(fixedSettings{"external.enabled": true, "external.allowed_hosts": []any{host}}), server)
	if got := listed.Resolve(context.Background(), []formula.ExternalRequest{request})[key]; got.Err != nil || got.Text != "hello" {
		t.Fatalf("허용된 호스트는 가져와야 한다: %+v", got)
	}
}

func TestPolicyRefusesWhatMustNeverBeReached(t *testing.T) {
	for _, ip := range []string{"127.0.0.1", "10.1.2.3", "192.168.0.1", "172.16.5.5", "169.254.169.254", "100.64.0.1", "::1", "fe80::1", "0.0.0.0"} {
		if PublicAddress(net.ParseIP(ip)) {
			t.Errorf("%s 은 닿으면 안 된다", ip)
		}
	}
	for _, ip := range []string{"93.184.216.34", "2606:2800:220:1:248:1893:25c8:1946", "8.8.8.8"} {
		if !PublicAddress(net.ParseIP(ip)) {
			t.Errorf("%s 은 공개 주소다", ip)
		}
	}
	if _, err := CheckURL("http://api.example.com/x", []string{"api.example.com"}); err == nil {
		t.Error("http 는 거절해야 한다")
	}
	if _, err := CheckURL("https://user:pw@api.example.com/x", []string{"api.example.com"}); err == nil {
		t.Error("자격 증명이 든 주소는 거절해야 한다")
	}
	if !HostAllowed("data.example.com", []string{"*.example.com"}) || HostAllowed("example.com", []string{"*.example.com"}) || HostAllowed("evil-example.com", []string{"*.example.com"}) {
		t.Error("와일드카드는 아래 호스트만 허용해야 한다")
	}
	if !HostAllowed("API.Example.com.", []string{"api.example.com"}) {
		t.Error("대소문자와 끝점은 무시해야 한다")
	}
	// 루프백을 허용하지 않는 진짜 정책으로 루프백 서버를 부르면 거절돼야 한다.
	server, host := serve(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("secret")) })
	strict := withInsecureTLS(New(fixedSettings{"external.enabled": true, "external.allowed_hosts": []any{host}}, nil), server)
	request := formula.ExternalRequest{Function: "WEBSERVICE", URL: server.URL}
	if got := strict.Resolve(context.Background(), []formula.ExternalRequest{request})[formula.ExternalKey(request.Function, request.URL)]; got.Err == nil || !strings.Contains(got.Err.Message, "사설망") {
		t.Fatalf("루프백은 허용 목록에 있어도 거절해야 한다: %+v", got)
	}
}

func TestRedirectsAndOversizedBodiesAreRefused(t *testing.T) {
	server, host := serve(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect":
			http.Redirect(w, r, "https://elsewhere.example.com/", http.StatusFound)
		case "/big":
			_, _ = w.Write([]byte(strings.Repeat("x", 3*1024)))
		default:
			_, _ = w.Write([]byte("ok"))
		}
	})
	fetcher := withInsecureTLS(testFetcher(fixedSettings{"external.enabled": true, "external.allowed_hosts": []any{host}, "external.max_kb": float64(2)}), server)
	resolve := func(path string) formula.ExternalResult {
		request := formula.ExternalRequest{Function: "WEBSERVICE", URL: server.URL + path}
		return fetcher.Resolve(context.Background(), []formula.ExternalRequest{request})[formula.ExternalKey(request.Function, request.URL)]
	}
	if got := resolve("/redirect"); got.Err == nil || !strings.Contains(got.Err.Message, "넘기는") {
		t.Errorf("리다이렉트는 따라가지 않아야 한다: %+v", got)
	}
	if got := resolve("/big"); got.Err == nil || !strings.Contains(got.Err.Message, "크기") {
		t.Errorf("큰 응답은 거절해야 한다: %+v", got)
	}
	if got := resolve("/ok"); got.Err != nil || got.Text != "ok" {
		t.Errorf("보통 응답은 와야 한다: %+v", got)
	}
}

func TestImportDataParsesATableAndCachesIt(t *testing.T) {
	hits := 0
	server, host := serve(t, func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write([]byte("\uFEFF품목,단가,비고\n연필,1200,\n공책,3.5e3,\"a,b\"\n"))
	})
	fetcher := withInsecureTLS(testFetcher(fixedSettings{"external.enabled": true, "external.allowed_hosts": []any{host}, "external.cache_seconds": float64(60)}), server)
	now := time.Now()
	fetcher.now = func() time.Time { return now }
	request := formula.ExternalRequest{Function: "IMPORTDATA", URL: server.URL + "/t.csv"}
	key := formula.ExternalKey(request.Function, request.URL)
	got := fetcher.Resolve(context.Background(), []formula.ExternalRequest{request})[key]
	if got.Err != nil || got.Rows != 3 || got.Columns != 3 {
		t.Fatalf("표를 읽지 못했다: %+v", got)
	}
	if got.Values[0] != "품목" || got.Values[4] != 1200.0 || got.Values[7] != 3500.0 || got.Values[8] != "a,b" {
		t.Fatalf("칸이 다르다: %#v", got.Values)
	}
	fetcher.Resolve(context.Background(), []formula.ExternalRequest{request})
	if hits != 1 {
		t.Fatalf("캐시가 있으면 다시 부르지 않아야 한다: %d번", hits)
	}
	now = now.Add(2 * time.Minute)
	fetcher.Resolve(context.Background(), []formula.ExternalRequest{request})
	if hits != 2 {
		t.Fatalf("캐시가 지나면 다시 불러야 한다: %d번", hits)
	}
}

// 소수점을 쉼표로 적는 로케일의 엑셀은 CSV 를 세미콜론으로 가른다. 업로드로 들어온
// 그런 파일은 이미 열로 갈라 읽으므로, 같은 파일을 IMPORTDATA 로 가져와도 같은 표가
// 나와야 한다 — 아니면 시트 하나가 A 열의 글자 덩어리로 들어앉는다.
func TestImportDataReadsWhicheverSeparatorCutsTheFile(t *testing.T) {
	got := parseCSV("품목;단가;비고\n연필;1200;\"가,나\"\n")
	if got.Err != nil || got.Rows != 2 || got.Columns != 3 {
		t.Fatalf("세미콜론으로 가른 표를 읽지 못했다: %+v", got)
	}
	if got.Values[3] != "연필" || got.Values[4] != 1200.0 || got.Values[5] != "가,나" {
		t.Fatalf("칸이 다르다: %#v", got.Values)
	}
	// 따옴표 안의 세미콜론은 가르는 것이 아니라 글자다.
	quoted := parseCSV("품목,비고\n연필,\"가;나;다;라\"\n")
	if quoted.Err != nil || quoted.Columns != 2 || quoted.Values[3] != "가;나;다;라" {
		t.Fatalf("따옴표 안의 세미콜론에 속았다: %+v", quoted)
	}
	// 표 위의 제목 한 줄은 아무것으로도 갈리지 않으므로 아무것도 정하지 않는다.
	titled := parseCSV("2026년 1분기 보고서\n품목;단가\n연필;1200\n공책;3500\n")
	if titled.Err != nil || titled.Columns != 2 || titled.Values[4] != "연필" || titled.Values[5] != 1200.0 {
		t.Fatalf("제목 줄에 속았다: %+v", titled)
	}
}

// 무엇으로 읽어야 하는지 밝히고 오는 파일은 밝힌 대로 읽는다. 업로드로 들어온 같은
// 파일을 가져오기가 이제 그렇게 읽으므로, 문이 어느 쪽이냐에 따라 같은 주소가 다른
// 표가 되어서는 안 된다 — 윈도우 도구가 올려 둔 UTF-16LE CSV 가 그런 파일이다.
func TestImportDataReadsACSVThatSaysItIsUTF16(t *testing.T) {
	body := []byte{0xFF, 0xFE}
	for _, unit := range utf16.Encode([]rune("지점,매출\r\n서울,1200\r\n")) {
		body = append(body, byte(unit), byte(unit>>8))
	}
	got := parseCSV(string(body))
	if got.Err != nil || got.Rows != 2 || got.Columns != 2 {
		t.Fatalf("UTF-16 로 적힌 표를 읽지 못했다: %+v", got)
	}
	if got.Values[0] != "지점" || got.Values[2] != "서울" || got.Values[3] != 1200.0 {
		t.Fatalf("칸이 다르다: %#v", got.Values)
	}
}

// 관리자가 allowed_hosts 를 고치면 곧바로 통해야 한다. 정책은 아무 데도 닿지 않고
// 설정만 읽으므로 캐시의 바깥에 있다.
func TestPolicyIsNotCachedInEitherDirection(t *testing.T) {
	server, host := serve(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) })
	settings := fixedSettings{"external.enabled": true, "external.allowed_hosts": []any{"api.example.com"}, "external.cache_seconds": float64(300)}
	fetcher := withInsecureTLS(testFetcher(settings), server)
	now := time.Now()
	fetcher.now = func() time.Time { return now }
	request := formula.ExternalRequest{Function: "WEBSERVICE", URL: server.URL + "/x"}
	key := formula.ExternalKey(request.Function, request.URL)
	resolve := func() formula.ExternalResult {
		return fetcher.Resolve(context.Background(), []formula.ExternalRequest{request})[key]
	}
	if got := resolve(); got.Err == nil || !strings.Contains(got.Err.Message, "허용되지 않은 호스트") {
		t.Fatalf("목록에 없으면 거절해야 한다: %+v", got)
	}
	settings["external.allowed_hosts"] = []any{host}
	if got := resolve(); got.Err != nil || got.Text != "ok" {
		t.Fatalf("목록에 넣으면 캐시가 살아 있어도 곧바로 가져와야 한다: %+v", got)
	}
	settings["external.allowed_hosts"] = []any{"api.example.com"}
	if got := resolve(); got.Err == nil || !strings.Contains(got.Err.Message, "허용되지 않은 호스트") {
		t.Fatalf("목록에서 빼면 캐시에 든 답도 내주면 안 된다: %+v", got)
	}
}

// 캐시를 지우는 것은 새 답이 들어올 때뿐이다. 상한이 없으면 매번 다른 주소를 부르는
// 워크북 하나가 지도를 끝없이 키운다.
func TestCacheSweepsExpiredAndStaysBounded(t *testing.T) {
	fetcher := New(nil, nil)
	now := time.Now()
	fetcher.now = func() time.Time { return now }
	put := func(key string, live time.Duration) {
		fetcher.store(key, formula.ExternalResult{Text: key}, now.Add(live))
	}
	for index := 0; index < maxCacheEntries; index++ {
		put("old"+strconv.Itoa(index), time.Minute)
	}
	if len(fetcher.cache) != maxCacheEntries {
		t.Fatalf("가득 차야 한다: %d", len(fetcher.cache))
	}
	now = now.Add(2 * time.Minute)
	put("fresh", time.Minute)
	if len(fetcher.cache) != 1 {
		t.Fatalf("지난 항목은 새 답이 들어올 때 쓸려 나가야 한다: %d", len(fetcher.cache))
	}
	for index := 0; index < maxCacheEntries+50; index++ {
		put("live"+strconv.Itoa(index), time.Hour+time.Duration(index)*time.Second)
	}
	if len(fetcher.cache) > maxCacheEntries {
		t.Fatalf("살아 있는 항목만 있어도 상한을 넘으면 안 된다: %d", len(fetcher.cache))
	}
	if _, ok := fetcher.cache["live"+strconv.Itoa(maxCacheEntries+49)]; !ok {
		t.Error("가장 나중에 들어온 답은 남아 있어야 한다")
	}
	if _, ok := fetcher.cache["live0"]; ok {
		t.Error("먼저 만료될 답부터 나가야 한다")
	}
}

// 관리자가 external.max_kb 를 올리면 그 자리에서 통해야 한다. 크기 상한은 부르는
// 방법을 정하는 정책이므로, 옛 상한에서 받은 거절은 새 상한에 대한 답이 아니다.
func TestRaisingTheSizeCeilingIsBelievedAtOnce(t *testing.T) {
	hits := 0
	server, host := serve(t, func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write([]byte(strings.Repeat("x", 3*1024)))
	})
	settings := fixedSettings{"external.enabled": true, "external.allowed_hosts": []any{host}, "external.cache_seconds": float64(300), "external.max_kb": float64(2)}
	fetcher := withInsecureTLS(testFetcher(settings), server)
	now := time.Now()
	fetcher.now = func() time.Time { return now }
	request := formula.ExternalRequest{Function: "WEBSERVICE", URL: server.URL + "/big"}
	key := formula.ExternalKey(request.Function, request.URL)
	resolve := func() formula.ExternalResult {
		return fetcher.Resolve(context.Background(), []formula.ExternalRequest{request})[key]
	}
	if got := resolve(); got.Err == nil || !strings.Contains(got.Err.Message, "크기") {
		t.Fatalf("상한을 넘으면 거절해야 한다: %+v", got)
	}
	if got := resolve(); got.Err == nil || hits != 1 {
		t.Fatalf("정책이 그대로면 다시 부르지 않아야 한다: %+v, %d번", got, hits)
	}
	settings["external.max_kb"] = float64(8)
	if got := resolve(); got.Err != nil || len(got.Text) != 3*1024 {
		t.Fatalf("상한을 올리면 캐시에 든 거절을 내주면 안 된다: %+v", got)
	}
	settings["external.max_kb"] = float64(2)
	if got := resolve(); got.Err == nil || !strings.Contains(got.Err.Message, "크기") {
		t.Fatalf("상한을 내리면 캐시에 든 답도 내주면 안 된다: %+v", got)
	}
}

// 원격이 500 으로 답한 것은 정책이 정한 일이 아니라 저쪽 사정이다. 배포가 끝나 원격이
// 다시 멀쩡해졌는데도 cache_seconds(최대 하루) 동안 그 거절이 답인 척하면 안 된다.
// 그렇다고 아예 담지 않으면 시간 제한을 매 다시 계산마다 처음부터 다시 기다리므로
// 잠깐만 담는다. 이 응답의 성질이 정한 거절(크기 상한)은 그것과 달라 그대로 담긴다.
func TestRemoteFailuresAreKeptOnlyBriefly(t *testing.T) {
	broken := true
	hits := 0
	server, host := serve(t, func(w http.ResponseWriter, r *http.Request) {
		hits++
		switch {
		case r.URL.Path == "/big":
			_, _ = w.Write([]byte(strings.Repeat("x", 3*1024)))
		case broken:
			http.Error(w, "deploying", http.StatusInternalServerError)
		default:
			_, _ = w.Write([]byte("ok"))
		}
	})
	fetcher := withInsecureTLS(testFetcher(fixedSettings{"external.enabled": true, "external.allowed_hosts": []any{host}, "external.cache_seconds": float64(3600), "external.max_kb": float64(2)}), server)
	now := time.Now()
	fetcher.now = func() time.Time { return now }
	resolve := func(path string) formula.ExternalResult {
		request := formula.ExternalRequest{Function: "WEBSERVICE", URL: server.URL + path}
		return fetcher.Resolve(context.Background(), []formula.ExternalRequest{request})[formula.ExternalKey(request.Function, request.URL)]
	}
	if got := resolve("/x"); got.Err == nil || !strings.Contains(got.Err.Message, "500") {
		t.Fatalf("500 은 거절로 와야 한다: %+v", got)
	}
	if got := resolve("/x"); got.Err == nil || hits != 1 {
		t.Fatalf("잠깐은 담아 두어 되풀이해 부르지 않아야 한다: %+v, %d번", got, hits)
	}
	broken = false
	now = now.Add(failureCacheFor / 2)
	if got := resolve("/x"); got.Err == nil || hits != 1 {
		t.Fatalf("아직 담긴 동안은 그 거절이 나와야 한다: %+v, %d번", got, hits)
	}
	now = now.Add(failureCacheFor)
	if got := resolve("/x"); got.Err != nil || got.Text != "ok" {
		t.Fatalf("원격이 나으면 cache_seconds 를 기다리지 않고 통해야 한다: %+v", got)
	}
	if hits != 2 {
		t.Fatalf("다시 한 번만 불러야 한다: %d번", hits)
	}
	// 성공한 답은 예전대로 cache_seconds 만큼 산다.
	now = now.Add(10 * time.Minute)
	if got := resolve("/x"); got.Err != nil || got.Text != "ok" || hits != 2 {
		t.Fatalf("성공한 답의 수명은 그대로여야 한다: %+v, %d번", got, hits)
	}
	// 크기를 넘긴다는 거절은 같은 응답에서 언제나 같은 답이므로 짧게 담을 까닭이 없다.
	if got := resolve("/big"); got.Err == nil || !strings.Contains(got.Err.Message, "크기") {
		t.Fatalf("상한을 넘으면 거절해야 한다: %+v", got)
	}
	now = now.Add(2 * failureCacheFor)
	if got := resolve("/big"); got.Err == nil || hits != 3 {
		t.Fatalf("정책이 정한 거절은 cache_seconds 만큼 담겨야 한다: %+v, %d번", got, hits)
	}
}

// 한 번의 다시 계산이 여러 주소를 부르고, 여러 사람이 같은 워크북을 함께 고친다.
// Fetcher 는 함께 써도 안전하다고 적혀 있으므로 -race 로 그것을 못 박는다.
func TestResolveIsSafeToShare(t *testing.T) {
	server, host := serve(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok" + r.URL.Path))
	})
	fetcher := withInsecureTLS(testFetcher(fixedSettings{"external.enabled": true, "external.allowed_hosts": []any{host}, "external.cache_seconds": float64(300)}), server)
	var waiting sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		waiting.Add(1)
		go func(worker int) {
			defer waiting.Done()
			requests := make([]formula.ExternalRequest, 0, 4)
			for index := 0; index < 4; index++ {
				requests = append(requests, formula.ExternalRequest{Function: "WEBSERVICE", URL: server.URL + "/p" + strconv.Itoa((worker+index)%5)})
			}
			results := fetcher.Resolve(context.Background(), requests)
			for _, request := range requests {
				got := results[formula.ExternalKey(request.Function, request.URL)]
				path := strings.TrimPrefix(request.URL, server.URL)
				if got.Err != nil || got.Text != "ok"+path {
					t.Errorf("%s 의 답이 다르다: %+v", path, got)
				}
			}
		}(worker)
	}
	waiting.Wait()
	if len(fetcher.cache) != 5 {
		t.Fatalf("주소마다 한 자리씩만 써야 한다: %d", len(fetcher.cache))
	}
}

func TestShortKeepsKoreanLettersWhole(t *testing.T) {
	trimmed := short(strings.Repeat("한", 200))
	if !utf8.ValidString(trimmed) || strings.ContainsRune(trimmed, utf8.RuneError) {
		t.Fatalf("글자를 쪼개면 안 된다: %q", trimmed)
	}
	if got := utf8.RuneCountInString(trimmed); got != maxMessageRunes+1 {
		t.Fatalf("%d글자 + 말줄임표여야 한다: %d", maxMessageRunes, got)
	}
	if short("짧은 메시지") != "짧은 메시지" {
		t.Error("짧은 메시지는 그대로 두어야 한다")
	}
}

// 업로드 가져오기(internal/importexport 의 parseScalar)는 우편번호·사번의
// "00123" 과 스무 자리 계좌번호를 글자로 둔다 — 수로 읽으면 123 과
// 1.2345678901234567e19 가 되어 파일에 적힌 것과 다른 값이 칸에 남고 되돌릴
// 길이 없다. IMPORTDATA 는 수식 엔진의 자(formula.DecimalNumber)만 불러 그
// 두 가드가 없었으므로 같은 CSV 가 어느 문으로 들어오느냐에 따라 다른 표가
// 되었다. 같은 파일은 어느 문으로 들어오든 같은 표여야 한다.
func TestImportDataKeepsNumbersThatUploadKeepsAsText(t *testing.T) {
	body := "사번,계좌,금액,비율,영\n00123,12345678901234567890,1200,0.5,0\n007.5,1234567890123456.5,3.5e3,-0,+0\n"
	server, host := serve(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	})
	fetcher := withInsecureTLS(testFetcher(fixedSettings{"external.enabled": true, "external.allowed_hosts": []any{host}}), server)
	request := formula.ExternalRequest{Function: "IMPORTDATA", URL: server.URL + "/t.csv"}
	got := fetcher.Resolve(context.Background(), []formula.ExternalRequest{request})[formula.ExternalKey(request.Function, request.URL)]
	if got.Err != nil || got.Rows != 3 || got.Columns != 5 {
		t.Fatalf("표를 읽지 못했다: %+v", got)
	}
	for index, want := range []any{
		"사번", "계좌", "금액", "비율", "영",
		// 앞자리 0 과 열여섯 자리 넘는 번호는 글자다. 나머지는 예전대로 수.
		"00123", "12345678901234567890", 1200.0, 0.5, 0.0,
		// 소수점이 있으면 앞의 0 은 자리이지 번호가 아니다. 지수로 적은 것은 수다.
		7.5, "1234567890123456.5", 3500.0, -0.0, 0.0,
	} {
		if got.Values[index] != want {
			t.Errorf("%d번째 칸 = %#v, want %#v", index, got.Values[index], want)
		}
	}
	// 같은 바이트를 업로드 문으로 넣어 칸마다 같은 타입·같은 값인지 본다.
	// 한쪽의 규칙만 고쳐서는 두 문이 다시 갈린다.
	uploaded, err := importexport.Parse("t.csv", []byte(body), 0)
	if err != nil {
		t.Fatalf("업로드로 읽지 못했다: %v", err)
	}
	seen := 0
	for _, cell := range uploaded.Sheets[0].Cells {
		var value any
		if err := json.Unmarshal(cell.Value, &value); err != nil {
			t.Fatalf("%d:%d 칸을 읽지 못했다: %v", cell.Row, cell.Column, err)
		}
		fetched := got.Values[(cell.Row-1)*got.Columns+cell.Column-1]
		if fetched != value {
			t.Errorf("%d:%d 칸이 문에 따라 다르다: 업로드 %#v, IMPORTDATA %#v", cell.Row, cell.Column, value, fetched)
		}
		seen++
	}
	if seen != len(got.Values) {
		t.Errorf("업로드는 %d칸, IMPORTDATA 는 %d칸", seen, len(got.Values))
	}
}

// 같은 바이트의 논리값은 두 문에서 Go 타입과 JSON 타입까지 같아야 한다.
// 공백·빈칸·아포스트로피의 기존 계약 차이는 별도 기대값으로 지킨다.
func TestImportDataReadsBooleansLikeUpload(t *testing.T) {
	for _, tc := range []struct {
		name    string
		body    string
		columns int
		remote  []any
		upload  []any
	}{
		{
			name:    "booleans and lookalikes",
			body:    "true,TRUE,True,false,FALSE,FaLsE\ntruex,falsehood,t,f,1,0\n00123,12345678901234567890,'true, true,false ,text\n",
			columns: 6,
			remote:  []any{true, true, true, false, false, false, "truex", "falsehood", "t", "f", 1.0, 0.0, "00123", "12345678901234567890", "'true", " true", "false ", "text"},
		},
		{
			name:    "existing whitespace apostrophe and empty cells",
			body:    " 12,12 ,'true,'+12\n,false , true\n",
			columns: 4,
			remote:  []any{12.0, 12.0, "'true", "'+12", "", "false ", " true", nil},
			upload:  []any{" 12", "12 ", "'true", 12.0, nil, "false ", " true", nil},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server, host := serve(t, func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(tc.body))
			})
			fetcher := withInsecureTLS(testFetcher(fixedSettings{"external.enabled": true, "external.allowed_hosts": []any{host}}), server)
			request := formula.ExternalRequest{Function: "IMPORTDATA", URL: server.URL + "/booleans.csv"}
			got := fetcher.Resolve(context.Background(), []formula.ExternalRequest{request})[formula.ExternalKey(request.Function, request.URL)]
			if got.Err != nil || got.Columns != tc.columns || got.Rows*got.Columns != len(tc.remote) || len(got.Values) != len(tc.remote) {
				t.Fatalf("표를 읽지 못했다: %+v", got)
			}
			uploaded, err := importexport.Parse("booleans.csv", []byte(tc.body), 0)
			if err != nil {
				t.Fatal(err)
			}
			cells := make(map[int]json.RawMessage)
			for _, cell := range uploaded.Sheets[0].Cells {
				cells[(cell.Row-1)*tc.columns+cell.Column-1] = cell.Value
			}
			wantUpload := tc.upload
			if wantUpload == nil {
				wantUpload = tc.remote
			}
			for index, want := range tc.remote {
				row, column := index/tc.columns+1, index%tc.columns+1
				fetched := got.Values[index]
				if fetched != want {
					t.Errorf("%d:%d IMPORTDATA = %T(%v), want %T(%v)", row, column, fetched, fetched, want, want)
				}
				raw, present := cells[index]
				if wantUpload[index] == nil {
					if present {
						t.Errorf("%d:%d 업로드 빈칸이 저장됨: %s", row, column, raw)
					}
					continue
				}
				var value any
				if !present {
					t.Fatalf("%d:%d 업로드 칸 누락", row, column)
				}
				if err := json.Unmarshal(raw, &value); err != nil {
					t.Fatal(err)
				}
				if value != wantUpload[index] {
					t.Errorf("%d:%d 업로드 = %T(%v), want %T(%v)", row, column, value, value, wantUpload[index], wantUpload[index])
				}
				if tc.upload == nil {
					remoteJSON, err := json.Marshal(fetched)
					if err != nil {
						t.Fatal(err)
					}
					if fetched != value || string(remoteJSON) != string(raw) {
						t.Errorf("%d:%d 두 경로 타입/값 불일치: 원격 %T(%s), 업로드 %T(%s)", row, column, fetched, remoteJSON, value, raw)
					}
				}
			}
		})
	}
}

func TestScoutEncodingEvidence(t *testing.T) {
 for _, tc := range []struct{name string; body []byte}{
  {"invalid-utf8", []byte{'i','d',',','v','\n','1',',',0xff,'\n'}},
  {"utf32le", []byte{0xff,0xfe,0,0,'A',0,0,0}},
  {"utf32be", []byte{0,0,0xfe,0xff,0,0,0,'A'}},
 } {
  t.Run(tc.name,func(t *testing.T){
   server,host:=serve(t,func(w http.ResponseWriter,r *http.Request){_,_=w.Write(tc.body)})
   f:=withInsecureTLS(testFetcher(fixedSettings{"external.enabled":true,"external.allowed_hosts":[]any{host}}),server)
   req:=formula.ExternalRequest{Function:"IMPORTDATA",URL:server.URL+"/input.csv"}
   got:=f.Resolve(context.Background(),[]formula.ExternalRequest{req})[formula.ExternalKey(req.Function,req.URL)]
   _,uploadErr:=importexport.Parse("input.csv",tc.body,0)
   raw,_:=json.Marshal(got.Values)
   t.Logf("remoteErr=%v rows=%d columns=%d JSON=%s uploadErr=%v",got.Err,got.Rows,got.Columns,raw,uploadErr)
   if got.Err!=nil || uploadErr==nil {t.Fatal("scout premise differs")}
  })
 }
}
