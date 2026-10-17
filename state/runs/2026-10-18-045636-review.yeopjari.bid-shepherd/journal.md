# PR 처리기 노트 2026-10-18-045636-review.yeopjari.bid-shepherd — review.yeopjari.bid PR #10
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-18-042907-review.yeopjari.bid-improve)
# 회차 노트 2026-10-18-042907-review.yeopjari.bid-improve — review.yeopjari.bid
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 04:29] base pinned — main@b4874e8
- [러너 04:29] autonomy release — 
- [러너 04:29] scout skipped — 요청된 작업이 있어 정찰을 건너뛴다 (과제는 이미 정해져 있다)

## 구현 노트
- 왜: main CI 'check' 의 원인을 네 회차째 가설로 두지 않고 닫았다. 이 세션은 `gh` 가 인증돼 있어(`GH_CONFIG_DIR=/home/hkjang/.config/gh`) check-run annotation 원문을 읽었다 — "The job was not started because recent account payments have failed or your spending limit needs to be increased". 실패 두 건(v0.7.5·v0.7.6) 모두 `steps` 0개 / 3~4초. 저장소 결함이 아니라 프로덕션 코드 0줄로 끝내고, 그 진단 절차만 README 「검증」 아래 한 절(21줄)로 남겼다. 커밋 05e1fd7, 변경 파일 1개.
- 확신 없는 곳: README 에 적은 `gh api .../jobs --jq` 한 줄은 이 세션에서 실제로 돌려 `steps: 0` 을 받은 형태 그대로지만, 나머지 두 줄(`check-runs` → `annotations`)은 `--jq '.[].message'` 로 축약해 적었고 내가 돌린 것은 `--jq '.[]|{level,message}'` 였다 — 출력 모양만 다르고 경로는 같다. 이전 회차들이 `gh` 미인증으로 막혔으므로 토큰 없는 환경에서는 이 절의 명령이 그대로 안 돈다(README 에 그 전제는 안 적었다).
- 일부러 안 한 것: ① 러너 라벨 고정 — 같은 annotation 의 notice 가 `ubuntu-latest` 가 **2026-10-19(내일)** 부터 Ubuntu 26 으로 이전한다고 알린다. 이번 명세가 "이것만 고치세요" 이고 `.github/workflows/check.yml` 은 릴리즈 경로라 손대지 않았다. ideas.json 에 [4/2/S] 로 올렸다 — 다음 역할이 가장 먼저 볼 것. ② 워크플로 수정 일절 없음(단계 삭제·continue-on-error·타임아웃 완화 0건, `git show --stat` = README.md 1개). ③ 열린 PR 내용 재게시 없음.
- 다음 역할이 조심할 것: 로컬 전체 게이트를 main@b4874e8 에서 9단계 그대로 통과시켰다 — `130 passed (7 files)` / `224.1 KB of 230 KB` / `applied 10 migrations` / e2e `218 passed, 0 failed` / a11y `심각한 접근성 위반 없음`. e2e·a11y 는 `127.0.0.1:55443` 의 docker postgres(yeopjari-review-dev)와 8796 서버가 있어야 돌고, `bash scripts/dev-reset.sh` 로 DB 를 비운 직후여야 218/0 이 나온다. 8796 에 고아 서버가 떠 있으면 남의 서버를 검사해 가짜 통과가 난다 — 돌리기 전에 확인할 것. a11y 는 `npx playwright install chromium`(`--with-deps` 없이; 붙이면 sudo 를 요구한다)이 필요하다.
- 기준선 갱신: 단위 테스트가 118/6 → **130 tests / 7 files** 로 올라갔고(v0.7.5 의 `db-breaker.test.ts`), 첫 방문 JS 가 223.7 → **224.1 KB**(여유 5.9 KB)로 줄었다. 옛 회차 기록의 118/223.7 을 그대로 기대값으로 쓰면 어긋난다.
- CI 는 여전히 빨갛다. 이 회차로 고쳐지지 않는다 — 조치는 사람의 GitHub Billing & plans 확인뿐이다.
- [러너 04:38] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인한 것: 실제 base 는 origin/main@b4874e8 이라 심사 대상은 05e1fd7 의 README.md +21 줄뿐(로컬 `main` ref 는 v0.7.4 에 멈춰 있어 diff 가 825줄로 보인다). 구현자가 의심한 gh 명령 4줄을 전부 실제로 실행 — jq 경로·필드·출력이 문서와 일치했고, annotations 는 billing 문구를, jobs 는 `steps: 0` 을 그대로 돌려줬다. 커밋 날짜 2026-10-15 도 맞다.
- 거절 사유: README.md:105 가 check.yml 이 「검증」 "같은 목록" 을 밟는다고 쓰지만 ui-flows.mjs·screenshots.mjs 는 .github/ 어디에도 없다(4/6). README.md:123 의 "「검증」 전부 로컬 통과" 도 구현 노트가 남긴 출력 5건에 그 둘이 빠져 있다. 수리가 가장 먼저 볼 파일은 README.md 105·123 두 줄 — CI 가 실제로 밟는 4단계로 한정하거나 '브라우저 흐름·스크린샷은 로컬 전용' 을 명시할 것.
- 못 본 것: `BlobNotFound` XML 응답은 재현하지 않았다(로그 다운로드 미실행). 로컬 게이트 9단계도 재실행하지 않고 구현 노트의 출력을 그대로 읽었다.
- 승인했어도 남을 우려: 이 절의 명령은 `gh` 인증을 전제하는데 그 전제가 문서에 없다(이전 세 회차가 막힌 지점). 그리고 같은 annotation 의 notice 대로 2026-10-19 부터 ubuntu-latest 가 Ubuntu 26 으로 바뀌면 다음 CI 빨강은 billing 이 아니며 'steps 0' 판정이 통하지 않는다.
- 보안·법무: 차단 없음. 코드 0줄·권한·비밀값·개인정보 변화 없음. billing 문구를 공개 README 에 남기는 것은 소유자=운영자이므로 참고 수준.
- [러너 04:41] review rejected — 리뷰 거절: README.md:105 `.github/workflows/check.yml 이 같은 목록을 … 밟습니다` 는 사실과 다릅니다. 바로 위 「검증」 블록의 6개 명령 중 check.yml 에 있는 것�

## 수리 노트
- 지적 두 건 다 맞았다. check.yml 의 단계는 `npm run check`·build·`perf.mjs`·ci-env·migrate·서버 기동·`e2e.mjs`·playwright·`a11y.mjs` 로, 「검증」 6개 중 ui-flows.mjs·screenshots.mjs 는 없다. 틀린 지적은 없었다.
- 고친 방법: README.md:105 를 "이 중 네 개를 밟습니다 + ui-flows·screenshots 는 CI 에 없으니 로컬에서 직접 돌려야 증명되고 초록 CI 를 근거로 쓸 수 없다" 로, :123 을 "CI 가 밟는 네 단계가 로컬 통과(130 · 224.1 KB · 218 · 위반 0)" 로 좁혔다. 다음 회차가 이 줄을 읽고 유일한 UI 증명 게이트를 건너뛸 길을 막는 것이 목적이다.
- 재검증은 구현 노트를 믿지 않고 이 HEAD 에서 직접 돌렸다: `npm run check` → 130 passed (7 files), `npm run build` + `perf.mjs` → 224.1 KB / 230 KB, `dev-reset.sh`(고아 8796 서버를 내 빌드로 교체) → e2e 218 passed / 0 failed, `seed-demo.mjs` + `a11y.mjs` → 심각한 접근성 위반 없음. a11y 는 `npx playwright install chromium` 이 먼저 필요했다.
- 여전히 확신 없는 곳: ui-flows.mjs·screenshots.mjs 는 이번에도 돌리지 않았다(이 회차가 UI 를 건드리지 않으므로 의도적). 그리고 이 절의 `gh` 명령이 인증을 전제한다는 점은 비평이 지적한 '남을 우려' 였을 뿐 거절 사유가 아니어서 README 에 추가하지 않았다 — 범위 밖으로 판단했다.
- [러너 04:46] repair done — # 수리 요약 (커밋 177ec51)  - 문제: README.md:105 가 check.yml 이 「검증」 6개 "같은 목록" 을 밟는다고 썼지만 실제로는 4개(`npm run check`·`perf.mjs`·`e2e.mjs`·`a11y

## 비평 노트
- 수리 후 재심사(approve). README.md:105-107 을 check.yml 과 한 줄씩 대조 — CI 9단계에 ui-flows.mjs·screenshots.mjs 가 없다는 것, 「검증」 6개 중 4개만 밟는다는 것 모두 사실. 거절 사유 두 줄은 해소됐다.
- gh 명령 4줄을 그대로 실행해 확인: jobs `{"name":"check","conclusion":"failure","steps":0}`, annotations 가 인용된 billing 문구 그대로, run list 가 2026-10-15 실패 2건(5초·6초, README 의 "3~6초" 범위 안). 숫자는 이 HEAD 에서 직접 재실행 — `npm run check` 130 passed (7 files), `perf.mjs` 224.1 KB / 230 KB.
- 못 본 것: e2e 218 · a11y 위반 0 은 재실행하지 않았다(공유 dev DB 리셋이 필요하고 README-only diff 가 바꿀 수 없는 값). ui-flows.mjs·screenshots.mjs 도 안 돌렸다. BlobNotFound XML 도 재현하지 않았다.
- 승인해도 남는 우려 — 다음 회차가 먼저 볼 곳은 `.github/workflows/check.yml:3`: 주석이 아직 `the same gate as a laptop (README "검증")` 라고 말해 방금 고친 README:105-107 과 어긋난다(v0.7.2 bc3088d 부터 있던 줄, 이번 diff 밖이라 거절하지 않았다). 그리고 이 절의 `gh` 명령은 인증(GH_CONFIG_DIR)을 전제하는데 README 에 안 적혀 있다.
- 릴리즈 노트용: 2026-10-19 부터 `ubuntu-latest` 가 Ubuntu 26 으로 이전한다(같은 annotation 의 notice). 내일 이후의 CI 빨강은 billing 이 아닐 수 있으니 'steps 0' 판정을 자동 적용하지 말 것. 보안·법무 차단 없음(코드 0줄·개인정보·권한·비밀값 변화 없음).
- [러너 04:50] review approved — 리뷰 승인 (risk=low)
- [러너 04:50] pr created — https://github.com/hkjang/review.yeopjari.bid/pull/10
- [러너 04:50] ci failed — 성공이 아닌 검사: check=failure · 실패한 검사: ? 잡: check 

## 수리 노트
- 맞은 지적: 없다 — 이번에 넘어온 '결함' 은 비평 지적이 아니라 CI 빨강 한 건이고, 그건 저장소 결함이 아니었다. `gh api .../jobs` 가 `steps: 0` · 2초(19:50:26→28), annotation 이 `The job was not started because recent account payments have failed or your spending limit needs to be increased`. 잡이 시작도 안 했으므로 재현할 로컬 실패가 존재하지 않는다(앞 회차 05e1fd7 이 README 에 적어둔 판정 절차를 그대로 따라 확인).
- 이 PR 의 diff 는 `origin/main`(b4874e8) 대비 README.md 1파일 +24줄뿐이다. 린트·타입·테스트·번들이 읽는 파일이 하나도 안 바뀌었으니 검증을 깨뜨릴 수단 자체가 없다.
- 그래도 구현/수리 노트의 숫자를 믿지 않고 이 HEAD 에서 재실행: `npm run check` → 130 passed (7 files), `npm run build` + `perf.mjs` → 224.1 KB / 230 KB. 둘 다 CI 1·3단계와 같은 명령이다.
- 고친 것 없음 · 커밋 없음. 워크플로를 손대거나 단계를 지워 초록을 만드는 길밖에 없었고 그건 금지 규칙이라, 절대 규칙대로 fix-summary.md 에 근거만 남겼다. 조치는 사람의 GitHub Billing & plans 확인뿐이다.
- 확신 없는 곳: e2e(218) · a11y(위반 0) · ui-flows · screenshots 는 이번에도 안 돌렸다(docker PostgreSQL · chromium 필요, README-only diff 가 바꿀 수 없는 값). 그리고 **2026-10-19 부터 `ubuntu-latest` 가 Ubuntu 26 으로 이전**하므로 내일부터는 'steps 0 = billing' 을 자동 적용하면 틀릴 수 있다.
