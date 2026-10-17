# 회차 노트 2026-10-17-191337-moyro-improve — moyro
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:13] base pinned — main@a7b87e6
- [러너 19:13] autonomy release — 

## 정찰 노트
- 적재된 오류는 `interrupted before PR`(러너 중단)이지 릴리즈 빨간불이 아니다. 이번 세션은 `curl`·`WebFetch`·`node`/`bash scripts/*` 가 모두 권한 거부되어 빨간 런의 유무를 **확인하지 못했다(미확인)** — 그래서 과제서 맨 앞에 구현자가 10분 안에 돌릴 공개 Actions API 확인 절차를 박아 두었다(10-16 은 이 확인을 안 해 예산 절반을 날렸다).
- 그 위에서, 전제가 거짓이어도 성립하는 **진짜 반복 릴리즈 실패 구조**를 읽기만으로 찾아 1순위로 삼았다: `scripts/verify-pages.mjs:266` 이 기대 버전을 리터럴로 들고 있고, `git show --stat a7b87e6` 이 그 동기화에 21개 파일(게이트 자신 포함)이 손으로 들어갔음을 보여 준다. 한 군데만 어긋나면 ci·release 가 재실행해도 똑같이 깨진다.
- 다른 후보를 제친 이유: httpapi 상태코드 재분류(listReactions)는 네 회차 연속 계열이라 방향 중복, `NewRouter` 라우팅 테스트는 M 이고 '오류 대응' 우선 과제와 무관, `verify-product-ui.sh` 상대 경로는 CI 가 항상 루트에서 부르므로 실패 원인이 아니다.
- 확신 없는 곳(추측): ① 지금 `node scripts/verify-pages.mjs` 가 실제로 exit 0 인지 **못 돌려 봤다** — 구현자가 먼저 확인할 것. ② 두 openapi YAML 의 `  version:` 한 줄 모양은 grep 출력으로만 확인했고 파일 전체 구조는 안 봤다.
- 구현자가 조심할 것: `screens.html` 의 `v0.2.1` 과 sitemap 의 `2026-08-30`/`2026-09-07` 은 과거 캡처 자산의 **의도적 고정값**이다. 일관성을 이유로 유도 대상에 끌어들이면 다음 릴리즈가 거짓 실패한다. `.github/` 와 스크립트 시그니처는 그대로 둘 것.
- [러너 19:18] scout done — 릴리즈 게이트 `scripts/verify-pages.mjs` 가 기대 버전을 **문자열 리터럴로 박아 두어** 매 릴리즈마다 손으로 �

## 구현 노트
- 전제 먼저 확정: 공개 Actions API 1회 호출로 최근 15런 중 실패는 `38859433565`(10-16, 이미 수리·머지) 단 1건이고 base `a7b87e6` 에서 CI·Pages·Release 세 워크플로가 전부 success — "릴리즈 두 번 실패" 는 거짓. 과제서가 지시한 "빨간 런 없으면" 경로로 1순위 과제를 구현했다.
- 무엇을: `scripts/verify-pages.mjs` 의 기대 버전 리터럴 `"v0.2.51"`(`:266` 조건식+메시지, `:286` 메시지)을 `webapp/package.json` 의 `version` 유도로 바꾸고, 선언 출처 네 곳(webapp/plugin-sdk package.json, openapi-v4/openapi-moyro yaml)의 상호 drift 검사를 **추가**했다. 종전에는 네 출처 불일치가 전혀 검출되지 않았으므로 완화가 아니라 강화다. 프로덕션 1파일, 의존성 0, 인자 없는 시그니처 불변.
- 확신 없는 곳: openapi 버전을 `/^\s{2}version:\s*(\S+)\s*$/m` 로 뽑는다. 두 파일 모두 `  version:` 2-space 줄이 **정확히 1개**임을 grep 으로 확인했지만, 나중에 다른 2-space `version:` 줄이 `info:` **앞에** 추가되면 첫 매치가 그쪽으로 옮겨간다(뒤에 추가되는 건 무해). YAML 파서 금지 제약 하의 의도적 트레이드오프다 — 비평가는 여기를 먼저 볼 것.
- 일부러 안 한 것: `screens.html` 의 `v0.2.1` 과 sitemap 의 `2026-08-30`/`2026-09-07` 은 과거 캡처 자산의 고정값이라 리터럴로 유지(유도하면 다음 릴리즈가 거짓 실패). a7b87e6 이 손으로 고친 나머지 복사본(docs 본문, SystemInfoContext.tsx, USER_GUIDE.md)은 범위를 넓히지 않으려 손대지 않았고 ideas.json 에 M 후보로 적었다.
- 검증: 깨끗한 트리 exit 0(수정 전과 동일 출력), `grep -n '0\.2\.5'` 0줄, 네 출처 각각 `0.2.52` 변조 → 네 경우 전부 exit 1(어긋난 파일을 이름으로), 되돌리면 exit 0. 기존 단언 5종 + 리더 오류 2종을 **실제 변조로 발화 확인**했다(지우기 아님). size ratchet·`git diff --check`·`node --check`·`go build ./...` exit 0, `.github/`·`webapp/`·`server/`·`docs/`·`plugin-sdk/` diff 0줄.
- 다음 역할이 조심할 것: 이 게이트를 부르는 워크플로는 **세 곳**이다 — `ci.yml:151`, `release.yml:151`, **`pages.yml:33`**(정찰 과제서는 두 곳으로 적었다). 릴리즈 세션은 이제 버전을 `webapp/package.json` 에 올리면 네 선언 출처를 **모두** 함께 올려야 게이트가 통과한다(그게 이 변경의 목적). DB·서버 테스트는 불필요(Go 파일 무수정).
- [러너 19:23] brief accepted — 채택 — 과제서 스스로 지시한 전제 확인 절차를 먼저 실행해 "릴리즈 두 번 실패" 가 거짓임을 1회 curl 로 확정했고(빨간 
- [러너 19:23] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- 판정 approve (risk low, blocking 없음). 확인한 것: main 판 게이트로 실패 재현을 **독립 재현**했다 — `webapp/package.json` 만 0.2.52 로 변조 시 구 게이트는 `Pages verification passed` / EXIT=0(드리프트에 눈이 멀었음), HEAD 판은 EXIT=1 로 7개 파일을 이름으로 보고. 원장의 RED/GREEN 출력과 증상이 일치한다.
- 약화 아님을 변조로 확인: 네 선언 출처를 모두 올리고 docs HTML 만 남기면 docs 4페이지 마커 누락으로 EXIT=1, 단독 드리프트 3건도 전부 EXIT=1, 깨끗한 트리 EXIT=0. `plugin-sdk/js` 는 최근 3릴리즈 lockstep 이라 정당한 분기를 막지 않는다. size ratchet EXIT=0.
- 구현자가 의심한 자리(openapi 정규식 첫 매치)를 직접 봤다: 두 스펙 모두 `info:` 가 2번째 줄이고 2-space `version:` 줄이 각 1개이므로 오늘은 안전하며, 깨지는 방향도 **시끄러운 거짓 드리프트**지 조용한 통과가 아니다. 다만 openapi 버전을 인용부호로 감싸면 따옴표까지 캡처해 거짓 실패가 난다.
- 못 본 것: 서버/웹 테스트는 돌리지 않았다(Go·TS 무수정). `pages.yml` 의 실제 실행 결과와 GitHub 측 런 상태는 이 세션에 권한이 없어 미확인 — 로컬 재현만으로 판단했다.
- 승인이어도 남는 우려(릴리즈 노트·다음 회차): ① `docs/openapi-v4.yaml:6` 설명문·docs 본문·`SystemInfoContext.tsx`·`USER_GUIDE.md` 의 버전 리터럴은 여전히 수동 동기화이며 게이트가 보지 않는다. ② 릴리즈 **태그**와 저장소 내 네 선언을 잇는 검사는 없다(태그 v0.2.52 를 선언 0.2.51 커밋에 달면 통과) — 선행 구멍이지만 다음 회차 1순위 후보. ③ `verify-pages.mjs:262` 주석이 호출처를 두 곳으로만 적었다(실제 `pages.yml:33` 포함 세 곳).
- [러너 19:28] review approved — 리뷰 승인 (risk=low)
- [러너 19:28] pr created — https://github.com/hkjang/moyro/pull/41
- [러너 19:39] ci passed — 검사 3개 모두 success
- [러너 19:39] merge done — ba28762
- [러너 20:09] release published — v0.2.52
- [러너 20:26] assets verified — v0.2.52 자산 1개 (이전 v0.2.51: 1)
