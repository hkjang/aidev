- 과제: 영문 랜딩 페이지의 `<head>`(메타·canonical·JSON-LD)를 영문으로 복구한다 — JS 가 고쳐 주지 못하는 부분만 (가치 3 / 위험 1 / 작업량 S)

- 왜: `docs/index_en.html` 전체에 `Enterprise Telemetry &amp; System Platform` 이라는 문자열이 **145 곳** 박혀 있다(이번 회차 실측: `grep -c "Enterprise Telemetry" docs/index_en.html` → 145, `docs/index.html` 과 `docs/script.js` 는 0). 원래 한국어였던 자리에 이 구절이 일괄 치환된 사고로 보인다. 본문 대부분은 `data-i18n` 속성이 있어 `docs/script.js:413-417` 이 로드 시 `el.textContent = translations[lang][key]` 로 덮어쓰므로 브라우저에서는 보이지 않지만, **`<head>` 안의 `<title>`·meta·JSON-LD 는 그 코드가 건드리지 않는다** — 크롤러·소셜 미리보기·AI 답변 엔진이 읽는 바로 그 부분만 영구히 깨진 채 남는다. 덤으로 이 영문 페이지의 `canonical`/`og:url`/`twitter:url` 이 전부 한국어 루트 `https://hkjang.github.io/goalforge/` 를 가리키고 `og:locale` 이 `ko_KR` 이라, 영문 페이지가 스스로를 "한국어 페이지의 중복" 으로 선언하고 있다. head 한 덩어리만 고치면 색인·공유 링크가 정상화된다.

- 수용 기준:
  1) `sed -n '1,134p' docs/index_en.html | grep -c "Enterprise Telemetry"` → **0** (현재 **10** — 실측: 6, 66, 69, 74, 77, 82, 85, 90, 93, 127행). 파일 전체 수치는 줄어들기만 하면 되고 0 일 필요 없다 — 본문은 이번 범위가 아니다.
  2) `sed -n '1,134p' docs/index_en.html | grep -nP '[\x{AC00}-\x{D7A3}]'` → **출력 없음** (현재 **4줄** — line 7 description, 8 keywords 끝의 `AI 에이전트 오케스트레이터`, 17 og:title, 18 og:description).
  3) `canonical`(11행)·`og:url`(14행)·`twitter:url`(25행)이 `https://hkjang.github.io/goalforge/index_en.html` 를 가리키고, `og:locale`(22행)이 `en_US` 다.
  4) head 안의 `<script type="application/ld+json">` 블록이 **유효한 JSON 으로 파싱되고** FAQPage 의 4개 Question `name`/`text` 가 영문이다. 증명:
     `python3 -I -c "import re,json,sys; s=open('docs/index_en.html',encoding='utf-8').read(); b=re.search(r'application/ld\+json\">(.*?)</script>',s,re.S).group(1); d=json.loads(b); print(json.dumps(d,ensure_ascii=False)[:400])"`
     → 예외 없이 출력되고, 출력에 한글과 `Enterprise Telemetry` 가 없어야 한다.
  5) `git diff --stat` 이 `docs/index_en.html` **한 파일만** 보여 주고, `git diff` 의 모든 hunk 가 head(1~134행) 안에 있다. 본문·`styles.css`·`script.js`·`docs/index.html` 은 건드리지 않는다.

- 건드릴 파일: `docs/index_en.html` 의 1~134행만. 구체적으로:
  - 6행 `<title>` — 지금 `GoalForge | Enterprise Telemetry &amp; System Platform` 을 4번 반복한 손상 문자열. **같은 파일 27행 `twitter:title` 의 `GoalForge | Goal-First AI Development Orchestrator` 를 그대로 쓸 것** (이미 올바른 영문이며 검증 불필요).
  - 7행 `meta name="description"` — 한국어. **같은 파일 28행 `twitter:description`** 또는 **52행 JSON-LD `SoftwareApplication.description`**(둘 다 이미 올바른 영문)을 재사용할 것.
  - 8행 `meta name="keywords"` — 앞부분은 영문이고 맨 끝 `AI 에이전트 오케스트레이터` 하나만 한글. 그 항목만 빼거나 `AI Agent Orchestrator` 로 바꿀 것. 나머지 키워드 목록은 손대지 말 것.
  - 11행 `canonical`, 14행 `og:url`, 25행 `twitter:url` — 끝에 `index_en.html` 을 붙일 것.
  - 17행 `og:title`, 18행 `og:description` — 한국어. 각각 27·28행 영문과 같게 맞출 것.
  - 22행 `og:locale` — `ko_KR` → `en_US`.
  - JSON-LD `@graph` 안 FAQPage(대략 60~95행)의 손상된 **8곳**(66, 69, 74, 77, 82, 85, 90, 93행 — `name` 4개 + `acceptedAnswer.text` 4개). **영문 원문을 지어내지 말 것**: `docs/script.js` 의 **en 사전**(파일 내 두 번째 `translations` 블록, 약 211행부터)에 `faq_1_q`/`faq_1_a` ~ `faq_4_q`/`faq_4_a` 가 이미 있고, `docs/index.html` 의 같은 위치 JSON-LD 가 깨지지 않은 한국어 원문이다. 두 쪽을 대조해 Q1=`faq_1_q`(What is GoalForge?), Q2=`faq_2_q`(How is GoalForge different from raw AI coding agents?), Q3=프로바이더, Q4=MCP 서버 순서가 맞는지 확인한 뒤 en 사전 문자열을 옮길 것. 본문 FAQ 섹션(745~810행)은 이번 범위가 아니다.
  - 127행 — `<script id="auto-lang-detector">` 안 `lang.indexOf('Enterprise Telemetry &amp; System Platform') !== -1` 비교. **원본은 `docs/index.html:127` 의 `lang.indexOf('한') !== -1` 이다(이번 회차에 직접 읽어 확인함).** 그대로 되돌려 두 파일을 일치시킬 것. 영원히 거짓인 죽은 비교일 뿐이고 앞의 `data-lang` 경로가 실제 동작을 담당하므로 **동작은 바뀌지 않는다**. 자신이 없으면 이 줄은 건드리지 말 것 — 그 경우 수용 기준 1)의 기대값은 0 이 아니라 1 이다.

- 검증 명령 (이 저장소에서 실제로 도는 것, 이번 회차에 실행해 확인함):
  - `grep -c "Enterprise Telemetry" docs/index_en.html` (전/후 비교. 현재 145)
  - `sed -n '1,134p' docs/index_en.html | grep -c "Enterprise Telemetry"` (현재 9 → 0)
  - `sed -n '1,134p' docs/index_en.html | grep -nP '[\x{AC00}-\x{D7A3}]'` (현재 7·8·17·18행 → 무출력). **주의: `grep -P` 를 쓸 것.** 이 환경의 로캘에서 `grep '[가-힣]'` 은 `Invalid collation character` 로 죽는다(실측).
  - 위 수용 기준 4)의 `python3 -I` JSON 파싱 한 줄. `python3`·`xmllint` 가 PATH 에 있다는 것은 확인했으나 **이 명령 자체는 이번 회차에 실행하지 못했다(샌드박스 승인 거부) — 미확인.** 구현자가 수정 전에 한 번 돌려 "지금도 파싱은 된다(내용만 쓰레기다)" 를 먼저 확인하고 기준점으로 삼을 것.
  - `xmllint --html --noout docs/index_en.html 2>&1 | head` — HTML 구조가 더 나빠지지 않았는지. **주의: 수정 전에도 경고가 나올 수 있으므로 전/후 출력을 비교할 것이지 "무출력" 을 기준으로 삼지 말 것.**
  - `git diff --stat` → `docs/index_en.html` 한 줄만
  - Go 쪽은 이 변경과 무관하다. `go:embed` 검색 결과 **이 저장소에 `go:embed` 가 하나도 없어** `docs/` 는 바이너리에 들어가지 않는다 — `go test ./...` 를 통과 근거로 내세우지 말 것. 그래도 커밋 전 `go build ./...` 가 깨지지 않았음은 한 번 보는 게 좋다.

- 위험과 피할 것:
  - **본문(135~874행)으로 번지지 말 것.** 손상 145곳 중 대부분이 본문이지만 거의 모두 `data-i18n` 속성이 있어 `docs/script.js:413-417` 이 런타임에 덮어쓴다. 전체 복구는 별도 회차 과제이고, 이번에 같이 하면 "건드릴 파일" 이 커져 사람 손을 다시 타게 된다.
  - **`docs/script.js` 와 `docs/index.html` 은 읽기만 할 것.** 둘 다 깨지지 않은 정본이다. 특히 script.js 의 사전을 "정리" 하지 말 것.
  - **영문 문구를 새로 지어내지 말 것.** 필요한 영문은 전부 같은 파일 head(27·28·52행) 아니면 `docs/script.js` 의 en 사전에 이미 있다. 운영자 교훈("가이드 문서의 API 메서드는 소스에서 확인할 것")과 같은 취지다 — 없는 주장을 메타데이터에 넣지 말 것.
  - 브라우저 실행 검증은 이번에도 하지 않는다(미확인). 수용 기준은 전부 파일 내용으로 판정 가능하게 썼다.
  - `hreflang` `<link rel="alternate">` 쌍을 추가하는 것이 canonical 수정과 함께라면 더 옳지만, `docs/index.html` 까지 건드려야 하므로 **이번 범위 밖**으로 둔다. canonical 만 고쳐도 지금보다 엄격히 낫다.
  - 보호 경로(auth/migrations/workflows)는 전혀 닿지 않는다. 정적 HTML 1개 파일이다.

- 차선 후보: **MCP `activity_report` 의 `since` 설명을 실제 지원 문법과 맞추기** (가치 2 / 위험 1 / 작업량 S). 이번 회차에 실측 확인함: `internal/mcp/tools.go:73` 이 `"Duration such as 24h or 7d (default 24h)."` 라고 광고하지만 `internal/mcp/tools.go:657` 의 `activityReport` 는 `time.ParseDuration(args.Since)` 를 쓰므로 `"7d"` 는 `tools/call` 에서 `since must be a positive duration such as 24h, got "7d"` 로 실패한다. **고치는 방향은 설명 쪽(`7d` → `168h`)으로만 할 것** — 운영자 교훈("같은 종류의 파서가 여러 경로에 있으면 통합하지 말고 좁히는 방향으로만")대로 `cmd/goalforge/main.go:4400` 의 `d`/`w` 파서를 끌어오지 말 것(애초에 main 패키지라 import 도 안 된다). 다만 2026-10-09 에 같은 과제가 과제서로 올라갔다가 회차 결과 `no-change` 로 끝난 전력이 있으니, 고를 거면 `tools/list` → `tools/call` 계약 테스트(`internal/mcp/server_test.go:fixture` 재사용)까지 붙여 실제로 증명할 것.
