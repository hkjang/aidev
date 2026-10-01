---
title: "mmcp — 자율 개선 이력"
description: "mmcp: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음."
last_modified_at: 2026-10-02 06:35:04 +0900
---
{% raw %}
<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "SoftwareSourceCode",
 "name": "mmcp",
 "codeRepository": "https://github.com/hkjang/mmcp",
 "url": "https://hkjang.github.io/aidev/projects/mmcp/",
 "description": "mmcp: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음.",
 "inLanguage": "ko",
 "maintainer": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-10-02T06:35:04+09:00"
}
</script>

# mmcp

<p class="tldr"><strong>요약.</strong> mmcp: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음. <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0">건강 B</span> <span class="meta">14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0</span></p>

<ul class="stats"><li><b>1</b><span>회차</span></li><li><b>1</b><span>프로젝트</span></li><li><b>0</b><span>배포 준비 완료</span></li><li><b>0</b><span>릴리즈 진행 중</span></li><li><b>0</b><span>병합 완료</span></li><li><b>0</b><span>검토 대기</span></li><li><b>1</b><span>검증 실패</span></li><li><b>0</b><span>변경 없음</span></li><li><b>0</b><span>실행 오류</span></li><li><b>$3.62</b><span>비용</span></li><li><b>11분</b><span>에이전트 시간</span></li></ul>

## 현황

<dl class="kv">
<dt>저장소</dt><dd><a href="https://github.com/hkjang/mmcp">https://github.com/hkjang/mmcp</a></dd>
<dt>마지막 회차</dt><dd>2026-10-02 05:47 KST — <span class="pill pill-other">• 기타</span> verify failed: 실패한 검증: cd web &amp;&amp; npm test --silent (exit 1)</dd>
<dt>수정 과제</dt><dd>⚠️ 오류 대응(자동 적재): 마지막 회차가 &#x27;verify-failed&#x27; 로 끝났습니다. verify failed: 실패한 검증: cd web &amp;&amp; npm test --silent (exit 1)</dd>
</dl>

## 회차 이력

<div class="table-wrap"><table class="rt"><thead><tr><th>일시</th><th class="primary">프로젝트</th><th>결과</th></tr></thead><tbody><tr data-status="other"><td data-label="일시">2026-10-02 05:47</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/mmcp/">mmcp</a></td><td data-label="결과"><span class="pill pill-failed" title="outcome=verify-failed">검증 실패</span> verify failed: 실패한 검증: cd web &amp;&amp; npm test --silent (exit 1)<div class="meta">3파일 <span style="color:var(--good)">+164</span>/<span style="color:var(--bad)">−3</span> · 테스트 2 — fix: render non-Hangul Mattermost names in given-family order</div></td></tr></tbody></table></div>

## 비용·사용량

<div class="table-wrap"><table class="rt"><caption class="meta">최근 30세션</caption><thead><tr><th>시각</th><th class="primary">프로젝트</th><th>단계</th><th class="num">시간</th><th class="num">턴</th><th class="num">비용</th><th class="num">토큰 입력/출력</th><th>종료</th></tr></thead><tbody><tr data-status="other"><td data-label="시각">05:47</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/mmcp/">mmcp</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">5분</td><td data-label="턴" class="num">33</td><td data-label="비용" class="num">$1.60</td><td data-label="토큰 입력/출력" class="num">1.4M / 15K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">05:42</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/mmcp/">mmcp</a></td><td data-label="단계">정찰</td><td data-label="시간" class="num">5분</td><td data-label="턴" class="num">35</td><td data-label="비용" class="num">$2.02</td><td data-label="토큰 입력/출력" class="num">1.6M / 19K</td><td data-label="종료">error_max_budget_usd</td></tr></tbody></table></div>

## 아이디어 백로그 — 대기 6 / 전체 7

<div class="table-wrap"><table class="rt"><caption class="meta">에이전트가 회차마다 재평가한다. 가치 높고 위험 낮은 대기 항목이 다음 회차 후보다.</caption><thead><tr><th class="primary">아이디어</th><th>가치/위험/크기</th><th>상태</th><th>메모</th><th>갱신</th></tr></thead><tbody><tr data-status="nochange"><td data-label="아이디어" class="primary">internal/mattermost 의 남은 경로에 테스트 없음 — FileContent/PostList.Ordered/CAPEM</td><td data-label="가치/위험/크기">4/1/M</td><td data-label="상태">대기</td><td data-label="메모">2026-10-02 에 client_test.go 가 생겨 DisplayName, send() 의 비2xx→*APIError 와 StatusOf, 빈 본문의 http.StatusText 대체, New() 의 /api/v4 접미사 제거와 비http(s) 거부까지는 덮었다. 남은 것: FileContent 의 *[]byte 우회 경로(JSON 디코드 안 함), PostList.Ordered() 가 Posts 에 없는 id 를 조용히 버리는 것, New() 가 깨진 CAPEM 을 거부하는 것. 전부 httptest + 실제 *Client 로.</td><td data-label="갱신">2026-10-02</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">internal/logbuf 링 버퍼·Subscribe 해제 경로에 테스트 없음</td><td data-label="가치/위험/크기">3/1/S</td><td data-label="상태">대기</td><td data-label="메모">129줄, 테스트 0개. add() 의 용량 초과 폐기, Snapshot() 순서, Subscribe() 가 돌려주는 해제 함수 호출 후 구독자 채널에 더 쓰지 않는지(누수·블로킹)를 go test -race ./internal/logbuf/ 로. slog.Handler 구현이라 WithAttrs/WithGroup 도 대상.</td><td data-label="갱신">2026-10-02</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">ChannelMembers 가 per_page 한 페이지만 받아 멤버 목록을 조용히 잘라낸다</td><td data-label="가치/위험/크기">3/3/M</td><td data-label="상태">대기</td><td data-label="메모">client.go 의 /channels/{id}/members?per_page=N 호출에 page 루프가 없다. 호출자(tools_mm.go)가 애초에 상한을 의도해 넘기는지 미확인 — 호출자를 먼저 읽어야 타당성이 결정된다. 잘못 고치면 큰 채널에서 API 호출이 폭증하므로 위험 3.</td><td data-label="갱신">2026-10-02</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">mattermost.New() 가 x509.SystemCertPool() 에러를 버린다</td><td data-label="가치/위험/크기">2/1/S</td><td data-label="상태">대기</td><td data-label="메모">client.go:48 pool, _ := x509.SystemCertPool(). nil 이면 빈 풀로 대체하므로 동작은 하지만 사내 CA 를 넣은 관리자에게 &#x27;시스템 CA 를 못 읽었다&#x27; 가 조용히 사라진다. New() 가 로거를 받지 않아 시그니처를 건드려야 한다 — 가치 대비 번거로움.</td><td data-label="갱신">2026-10-02</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">mattermost.New() 가 BaseURL 의 query/fragment 를 그대로 보존해 요청 URL 이 깨질 수 있다</td><td data-label="가치/위험/크기">2/2/S</td><td data-label="상태">대기</td><td data-label="메모">client.go:38-43 은 scheme/host 만 검사하고 u.String() 을 base 로 쓴다. &#x27;https://mm.example.com/?x=1&#x27; 이면 base+&#x27;/api/v4&#x27;+path 가 깨진다. 2026-10-02 의 New() 테스트는 비http(s) 거부와 접미사 제거만 덮었고 이 경우는 일부러 비워 뒀다. 설정 저장 시점(settings.go validateSettings)에서 거르는 편이 나을 수 있음 — 그쪽에 이미 URL 정규화가 있다. 운영자 오설정 시나리오라 가치는 낮다.</td><td data-label="갱신">2026-10-02</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">identity_map.mm_display_name 에 남은 옛 &#x27;DoeJohn&#x27; 값이 갱신 전까지 그대로 보인다</td><td data-label="가치/위험/크기">2/2/S</td><td data-label="상태">대기</td><td data-label="메모">identity.go:386-387 이 DisplayName() 을 DB 에 저장하므로 2026-10-02 의 수정 이후에도 기존 행은 다음 갱신 때까지 옛 값을 유지한다. 마이그레이션(internal/store/migrations 는 보호 경로)으로 고치지 말 것 — 관리 콘솔의 &#x27;재검증&#x27;이 이미 갱신 경로이므로, 필요하다면 문서에 한 줄 적는 정도가 적절하다. 실제로 신경 쓸 만한 문제인지 자체가 미확인.</td><td data-label="갱신">2026-10-02</td></tr><tr data-status="released"><td data-label="아이디어" class="primary">mattermost.User.DisplayName() 이 한글 여부를 보지 않고 성+이름을 붙여 라틴 이름을 &#x27;DoeJohn&#x27; 으로 만든다</td><td data-label="가치/위험/크기">4/2/S</td><td data-label="상태">완료</td><td data-label="메모">2026-10-02 수정. client.go:160 이 koreanName() 과 같은 U+AC00-U+D7A3 범위로 분기하고, 아니면 &#x27;given family&#x27;(공백) 로 돌려준다. 두 함수는 입력 계약이 달라 통합하지 않았다. internal/mattermost/client_test.go(신규) 표 테스트 + internal/server/unit_test.go 교차 검증. 실패를 먼저 재현한 뒤 고쳤고 전체 통합 테스트(-race, TEST_POSTGRES_DSN)까지 통과.</td><td data-label="갱신">2026-10-02</td></tr></tbody></table></div>

## 원장 (에이전트가 남긴 기록)

## 2026-10-02
- 선택: mattermost.User.DisplayName() 의 비한국어 이름 깨짐 수정 + internal/mattermost 첫 테스트 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `internal/mattermost/client.go:160` 이 한글 여부를 보지 않고 항상 `LastName+FirstName` 를 붙여 "John Doe" 를 "DoeJohn" 으로 만들던 것을, `server.koreanName()` 과 같은 한글 음절 범위(U+AC00–U+D7A3)로 분기하도록 고쳤다(두 함수는 입력 계약이 달라 통합하지 않고 각자 둠). 신규 `internal/mattermost/client_test.go` 가 DisplayName 표 테스트 + 실제 `httptest.Server` 에 실제 `*Client` 를 물려 비2xx→`*APIError`/`StatusOf`/빈 본문의 `http.StatusText` 대체/`/api/v4` 접미사 제거를 검증하고, `internal/server/unit_test.go` 에 두 경로가 같은 given/family 를 같게 읽는지 교차 검증을 추가했다. 검증: `gofmt -l .`(빈 출력), `go vet ./...`, `./scripts/verify-version.sh`, 그리고 로컬 Postgres(mmcp-pg) 에 스크래치 DB 를 띄워 `TEST_POSTGRES_DSN=... go test -race -count=1 ./...` 전부 통과(`internal/server` 43.5s, `-v` 로 SKIP 0건 확인 — 통합 테스트가 실제로 돌았고 그 안에 `mattermost_send_dm` 승인·발송 경로가 포함된다). 스크래치 DB 는 끝나고 DROP 했다.
- 실패 재현: `--- FAIL: TestUserDisplayName/latin_name_keeps_given_family_order` / `client_test.go:28: DisplayName() = "DoeJohn", want "John Doe"` (교차 검증 쪽도 `unit_test.go:102: User{"John" "Doe"}.DisplayName() = "DoeJohn", want "John Doe"` 로 동시에 실패)
- 보류 아이디어: internal/mattermost 의 남은 커버리지(FileContent 의 *[]byte 우회, PostList.Ordered() 가 없는 id 를 조용히 버리는 것, New() 의 CAPEM 파싱 실패) [4/1/M]; internal/logbuf 링 버퍼·Subscribe 해제 경로 테스트 [3/1/S]; ChannelMembers 가 per_page 한 페이지만 받아 멤버를 조용히 잘라냄 — 호출자 의도 먼저 확인 [3/3/M]; mattermost.New() 가 BaseURL 의 query/fragment 를 보존해 요청 URL 이 깨짐 [2/2/S]; New() 가 x509.SystemCertPool() 에러를 버림 [2/1/S]
- 과제서: 채택 — 과제서의 근거(client.go:163 라틴 분기가 죽은 코드, dm.go:79 매칭이 LastName+FirstName 를 따로 가지고 있어 회귀하지 않음)가 현재 코드와 정확히 일치했다.


[← 대시보드](https://hkjang.github.io/aidev/) · [교훈 모음](https://hkjang.github.io/aidev/lessons/)

{% endraw %}
