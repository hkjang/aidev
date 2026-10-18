# 과제서 — 2026-10-18 (GoalForge, base main@f746b00)

- 과제: 영문 랜딩 페이지에서 **JS 가 절대 덮어쓰지 않는** 9 줄을 영문으로 복구한다 — 언어 버튼 라벨 1 줄 + `pre.cli-code` 주석 8 줄 (가치 3 / 위험 1 / 작업량 S)

- 왜: `docs/index_en.html` 에는 손상 문자열 `Enterprise Telemetry &amp; System Platform` 이 들어간 줄이 **136 줄**(`grep -c` 실측) 남아 있는데, 그 대부분은 `data-i18n` 속성이 붙어 있어 `docs/script.js` 가 런타임에 en 사전 값으로 덮어쓴다. 그러나 `data-i18n` 이 **어느 조상에도 없는** 9 줄은 JS 를 켜도 그대로 보인다 — 특히 166 행의 한국어 전환 버튼은 `한` 대신 `Enterprise Telemetry & System Platform` 이라는 문장 전체를 버튼 안에 렌더링해 네비게이션 레이아웃을 깨뜨리고, 553~602 행의 CLI 예시 주석 8 줄은 영문 방문자에게 뜻 없는 문장으로 보인다. 이 9 줄만 고치면 JS 유무와 무관하게 보이는 손상이 사라지고, 남은 127 줄은 전부 `data-i18n` 이 책임지는 별개(L) 과제로 깔끔히 분리된다.

## 수용 기준

1) `docs/index_en.html:166` 의 버튼 텍스트가 `docs/index.html:166` 과 **바이트 단위로 같다** (즉 `한`). `KO`·`Korean` 같은 새 라벨을 지어내지 말 것 — 같은 저장소의 한국어 페이지가 정본이고, 짝 버튼은 이미 `EN` 이다.
2) 아래 8 줄의 `<span class="term-comment">` 안에 `Enterprise Telemetry` 가 하나도 없고, 각 줄이 **같은 행 번호의 `docs/index.html` 한국어 원문과 뜻이 같은 영문**이다 (아래 "원문 대조표" 참조). 명령줄(`goalforge …`)·들여쓰기·`<pre>`/`<code>`/`<span>` 태그 구조는 한 글자도 바뀌지 않는다.
3) `grep -c "Enterprise Telemetry" docs/index_en.html` 이 **136 → 127** 로 정확히 9 줄 줄어든다. 그 이상 줄어들면 `data-i18n` 줄을 건드린 것이고(범위 밖), 덜 줄어들면 빠뜨린 줄이 있다.
4) `git diff --stat` 이 **파일 1 개 / 9 insertions(+) / 9 deletions(-)** 이고 모든 hunk 가 160~610 행 안에 있다. `docs/index.html`·`docs/script.js`·Go 코드에는 변경이 없다 (`git status --porcelain` 으로 확인).
5) 체크 스크립트(수용 기준 1~3 을 그대로 단언하는 것, **저장소 밖**에 두어 커밋하지 말 것)를 수정 전 파일(`git show HEAD:docs/index_en.html`)에 돌려 **9 건 FAIL** → 수정 후 **0 FAIL** 로 인과를 고정한다. 2026-10-17 회차가 같은 방식(`check_head.py`, 11 FAIL → 0 FAIL)으로 증명했으니 그 방식을 재사용할 것.
6) `xmllint --html --noout docs/index_en.html` 경고 **집합이 수정 전·후 완전히 동일**하다(2026-10-17 실측으로 61 건이었으나 이번에 수를 다시 측정할 것 — 중요한 것은 diff 가 비는 것이다). **`xmllint` 의 존재는 이번 세션에서 확인하지 못했다(미확인 — 샌드박스가 `command -v` 를 거부했고, 직전 회차가 썼다는 기록만 있다).** 없으면 이 기준은 건너뛰고 "xmllint 미설치" 라고 적을 것 — 기준 1~5 만으로도 증명은 성립한다.

## 원문 대조표 (실측, 2026-10-18)

좌: 손상된 `docs/index_en.html` 행 번호 — 우: 같은 행의 `docs/index.html` 한국어 원문. 영문 표현은 구현자가 정하되, 아래 괄호 안 제안은 `docs/script.js:322-325` 의 en 사전(`cli_1_title`="1. Diagnostics & Init", `cli_2_title`="2. Goal & Gate Setup", `cli_3_title`="3. Discovery & Autonomous Run", `cli_4_title`="4. Approval, Merge & Rollback")과 용어를 맞춘 것이다.

| 행 | 한국어 원문 (`docs/index.html`) | 제안 영문 |
|---|---|---|
| 553 | `# 환경 검증 (Git, CLI, 권한)` | `# Verify the environment (Git, CLI, permissions)` |
| 556 | `# 프로젝트 초기화 (워크트리, 자동 커밋)` | `# Initialize the project (worktree, auto-commit)` |
| 568 | `# 버전에 맞는 목표 생성` | `# Create a goal for the version` |
| 573 | `# 결정론적 검증 게이트 추가` | `# Add a deterministic verification gate` |
| 584 | `# 작업 탐색 & 가치 점수 측정` | `# Discover tasks & score their value` |
| 588 | `# 연속 자율 실행 (Quota까지)` | `# Continuous autonomous run (until quota)` |
| 598 | `# 병합 승인 요청` | `# Request merge approval` |
| 602 | `# 메인 브랜치 병합 또는 롤백` | `# Merge to the main branch, or roll back` |

(166 행은 표 밖: `한` 으로 복구.)

## 건드릴 파일

- `docs/index_en.html:166` — `<button class="lang-btn active" data-lang="ko">` 의 텍스트 노드를 `한` 으로.
- `docs/index_en.html:553,556,568,573,584,588,598,602` — `<span class="term-comment">` 내부 텍스트만 영문으로. **8 줄 모두 `pre class="cli-code"` 블록 안이고 `data-i18n` 이 없다**(실측).
- 프로덕션 파일 **1 개**. 테스트 파일 없음(저장소에 docs 를 검증하는 Go 테스트가 없다 — 아래 검증 명령 참조).

## 검증 명령

이 저장소에서 실제로 도는 것:

```
# 1) 핵심: 손상 줄 수 (136 → 127 이어야 한다)
grep -c "Enterprise Telemetry" docs/index_en.html

# 2) 고친 9 줄에 손상이 남지 않았는지 (출력이 비어야 한다)
sed -n '166p;553p;556p;568p;573p;584p;588p;598p;602p' docs/index_en.html | grep "Enterprise Telemetry"

# 3) data-i18n 줄을 건드리지 않았는지 — 아래 목록이 수정 전과 같아야 한다 (127 줄)
grep -n "Enterprise Telemetry" docs/index_en.html

# 4) 한국어 페이지는 그대로 (0 이어야 한다)
grep -c "Enterprise Telemetry" docs/index.html

# 5) 변경 범위
git diff --stat && git status --porcelain && git diff --check

# 6) HTML 구조가 상하지 않았는지 — 경고 diff 가 비어야 한다
xmllint --html --noout docs/index_en.html

# 7) 저장소에 go:embed 가 없어 docs 는 바이너리와 무관하지만, 회차 관례상 돌릴 것
go build ./... && go vet ./...
```

`go test ./...` 는 이 변경과 무관하다(docs 를 읽는 테스트가 없음). 다만 **9 분 가까이 걸리므로**(sqlite ~46s, observer ~21s 포함) 시간이 남을 때만 돌리고, 돌렸다면 결과를 그대로 적을 것. 돌리지 않았다면 "미실행" 이라고 적을 것 — "무관하므로 통과" 라고 쓰지 말 것.

## 위험과 피할 것

- **127 행을 건드리지 말 것.** `} else if (lang.indexOf('ko') !== -1 || lang.indexOf('Enterprise Telemetry &amp; System Platform') !== -1) {` 는 `<head>` 안 인라인 스크립트의 `navigator.language` 비교다. 원본은 `'한'` 이었겠지만 `navigator.language` 에 `한` 이 들어올 일이 없어 **어느 쪽이든 도달 불가**이고, 되돌리면 2026-10-17 회차가 세운 "head 에 한글 금지" 기준과 정면으로 충돌한다. 그 회차도 같은 이유로 손대지 않았다.
- **233 행(`#terminal-body` 안 `term-comment`)을 이번 범위에 넣지 말 것.** `data-i18n` 은 없지만 `docs/script.js:525` 가 `document.getElementById('terminal-body')` 를 잡아 터미널 탭 데이터로 내용을 교체한다 — 런타임에 덮어쓰는지 **이번에 끝까지 확인하지 못했다(미확인)**. 확인 없이 고치면 "보이는 것만 고친다" 는 이 과제의 경계가 흐려지고, `terminalTabsData` 의 도구명 불일치(별개 보류 아이디어)와 얽힌다.
- **`data-i18n` 이 붙은 127 줄을 건드리지 말 것.** 특히 **직전 회차 정찰 노트의 서술은 틀렸다**: FAQ 답변(749·759·769·779·789·799·809)·히어로 서브타이틀(195)·GitHub 버튼(209)은 "`data-i18n` 이 없어 보인다" 고 기록돼 있었지만, 이번에 **부모 요소에 `data-i18n` 이 있음을 실측으로 확인**했다 (`<div class="faq-answer" data-i18n="faq_1_a">` 748 행, `<p class="hero-subtitle" data-i18n="hero_subtitle">` 193 행, `<a … data-i18n="btn_github">` 208 행). `grep -v data-i18n` 이 **줄 단위**라서 생긴 오판이다. 이 줄들은 script.js 가 덮어쓰므로 별개 L 과제다.
- **`docs/script.js` 를 건드리지 말 것.** 남은 127 줄 복구의 정본 사전이고, 터미널 데모 도구명 교정도 그 과제 뒤에 할 일로 분리돼 있다.
- 영문을 **지어내지 말 것**: 166 행은 `docs/index.html` 의 `한` 을 그대로 쓰고, 8 줄은 같은 행 한국어 원문의 **직역**에 머물 것. 2026-10-17 회차가 영문 원문이 없는 자리에서 임의 창작을 피하려 화면 표시 내용에 맞춘 선례가 있다.
- 보호 경로(auth·session·migrations·`.github/workflows`·Go 코드) 는 전혀 건드리지 않는다. 이 과제는 정적 문서 1 파일에 갇혀 있다.
- `&amp;` 처리 주의: 손상 문자열 자체가 `&amp;` 를 포함하므로 치환 시 엔티티가 깨지지 않게 할 것. 584 행 한국어 원문에는 `&` 가 있고(`작업 탐색 & 가치 점수`) 그 줄의 현재 손상 텍스트에도 **이스케이프되지 않은 생 `&`** 가 섞여 있다(실측: `… Platform & Enterprise …`) — 새로 쓰는 영문에서 `&` 를 쓸 경우 `&amp;` 로 적어 기준 6)의 xmllint 경고 diff 가 비도록 할 것.

## 차선 후보

**`docs/index.html` 과 `docs/index_en.html` 에 `hreflang` 짝 추가 (가치 3 / 위험 1 / 작업량 S).** 실측: 두 파일 모두 `rel="canonical"` 만 있고(`index.html:11` → 한국어 루트, `index_en.html:11` → `index_en.html`) **`rel="alternate"`/`hreflang` 은 양쪽에 하나도 없다**. 각 파일 `<head>` 에 `ko`/`en`/`x-default` 3 줄씩 넣으면 끝이고, 2026-10-17 의 self-referential canonical 수정의 자연스러운 후속이다. 파일 2 개·6 줄. 1 순위가 성립하지 않을 때(예: 166 행 라벨을 `한` 으로 두는 것에 이견이 있을 때) 이쪽을 고를 것.
