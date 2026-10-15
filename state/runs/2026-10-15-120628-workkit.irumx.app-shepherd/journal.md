# PR 처리기 노트 2026-10-15-120628-workkit.irumx.app-shepherd — workkit.irumx.app PR #6
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-15-113832-workkit.irumx.app-improve)
# 회차 노트 2026-10-15-113832-workkit.irumx.app-improve — workkit.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:38] base pinned — main@4213cea
- [러너 11:38] autonomy release — 

## 정찰 노트
- CI 포렌식을 또 1순위로 올리지 않았다 — 이 세션은 `gh` 미인증 + 사설 저장소 404 + 다른 회차 디렉터리 읽기 차단으로 PR #5 의 잡 주석을 볼 수단이 아예 없고, 같은 유형의 과제서가 이미 기각·차선·기각으로 끝났다. 대신 **저장소 안에서 실제로 고칠 수 있는 것**을 골랐다: `scripts/ci-env.mjs`(17·19줄)만 :55447 을 상수로 박아, `DATABASE_URL` 로 전용 DB 를 줘도 서버만 공유 dev DB 로 가는 반쪽 상태가 된다. 다섯 회차가 e2e·tools·ui·a11y 를 완주하지 못한 직접 원인이다.
- 차선(`toTable` 고유화 접미사 충돌)을 제친 이유: 가치는 비슷하지만 검증 신뢰도를 복구하는 쪽이 이 캠페인에서 급하다. Docker 를 못 쓰면 차선으로 내려가면 된다.
- 추측으로 적은 것(미확인): ① PR #5 의 실제 실패 사유 — 결제 차단이라는 1차 증거는 2026-10-14 수리 회차의 주석뿐이다 ② `.env.dev` 가 `.gitignore` 에 있는지 ③ `scripts` 를 실행하는 단위 시험이 기존에 있는지(없다고 보고 새로 만들라고 적었다). `npm ci` 를 돌리지 않아 테스트는 한 번도 실행하지 않았다.
- 구현자가 조심할 것: 수용 기준 1(환경변수 없을 때 `.env.dev` 바이트 동일)을 `cmp` 로 먼저 증명할 것 — 이것이 워크플로를 느슨하게 하지 않았다는 증거다. 상주 컨테이너 `workkit-irumx-dev`(:55447)는 본 체크아웃의 개발 DB 이니 절대 건드리지 말 것.
- [러너 11:43] scout done — `.env.dev` 생성기(`scripts/ci-env.mjs`)가 DB 주소 재지정을 무시해 **검증 도구와 서버가 서로 다른 DB 를 보는 것*

## 구현 노트
- 바꾼 것: `scripts/ci-env.mjs` 가 두 DB URL 의 호스트:포트를 상수 하나(`WORKKIT_DEV_DB_HOSTPORT ?? '127.0.0.1:55447'`)에서 끌어오게. 프로덕션 1파일 + 새 시험 1파일. 이유는 서버와 검증 스크립트가 서로 다른 DB 를 보던 반쪽 상태를 없애 CI 와 같은 검증을 워크트리에서 완주시키는 것 — e2e 가 148/7 에서 **155 passed, 0 failed** 로 바뀌었다.
- 확신 없는 곳·검증 못 한 것: ① **PR #5 의 실제 실패 사유는 이번에도 확인 못 했다** — `gh` 미인증 + 사설 404. 결제 차단은 2026-10-14 수리 회차의 주석이 유일한 1차 증거이고 내 변경이 그것을 고치지는 못한다. 이 회차는 "로컬에서 green 증명" 까지다. ② 새 시험은 `/tmp` 에 디렉터리를 만들어 `node scripts/ci-env.mjs` 를 자식 프로세스로 돌린다 — `tmpdir()` 를 못 쓰거나 `process.execPath` 가 다른 샌드박스에서는 깨질 수 있다(이 환경에서는 통과). ③ `WORKKIT_DEV_DB_HOSTPORT` 는 값 검증을 하지 않는다(빈 문자열을 주면 URL 이 망가진다) — 개발 전용 변수라 일부러 두지 않았다.
- 일부러 하지 않은 것: `scripts/dev-reset.sh`(상단 주석이 "never points at anything but workkit-irumx-dev" 라고 자기 계약을 적어 둔다 — 과제서의 선택 조항대로 뒀다), `.github/workflows/check.yml`·`packages/core/src/env.ts`·`packages/tools/**`(금지), `scripts/e2e.mjs:28` 의 `APP_DATABASE_URL`(이미 재지정 가능하므로 결함이 아니고, 운영자의 '통합 금지·좁게만' 규칙에 걸린다).
- 다음 역할이 조심할 것: 새 시험 `packages/core/src/__tests__/ci-env.test.ts` 는 **DB 가 필요 없다**(`npm run check` 안에서 돈다, 12 파일 138 passed). 반면 전용 DB 로 CI 순서를 완주하려면 `WORKKIT_DEV_DB_HOSTPORT` 와 `DATABASE_URL` 말고 **`APP_DATABASE_URL` 까지** 같은 주소로 줘야 한다 — 빼먹으면 RLS 3개 + 트리거 5개가 코드와 무관하게 깨진다(처음 완주에서 직접 겪었다). 수용 기준 1 은 `cmp` 로 증명했다: 환경변수 없는 `.env.dev` 가 고치기 전후 980B·sha256 `60be5f56…` 동일.
- [러너 11:55] brief accepted — 채택 — 근거(`ci-env.mjs:17,19` 만 호스트:포트를 상수로 박아 서버와 검증 도구가 갈린다)가 지금 코드와 정확히 맞았고, 권
- [러너 11:55] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인한 것: `scripts/ci-env.mjs` 와 새 시험을 읽고, `main` 판 생성기를 /tmp 에 꺼내 직접 돌려 **기본값 출력이 980B 바이트 동일**함을 `cmp` 로 재확인했다(원장의 수용 기준 1 이 사실). 고치기 전 판은 `WORKKIT_DEV_DB_HOSTPORT=…:55448` 을 줘도 55448 이 0회 나와, 새 시험 2·3 이 수정 전엔 정말 실패한다. `npm run check` 재실행 green(12 파일 138 passed). 비밀값 추가 없음(`.gitignore` 가 `.env.*`), CI 는 변수를 안 줘 동작 불변 — 보안·법무 차단 사유 없음.
- 못 본 것: `npm run e2e`·`tools`·`ui`·`a11y` 는 돌리지 않았다(Docker·Playwright 미준비). 구현자의 155/0 주장은 재현하지 않았고, PR #5 의 실제 실패 사유도 이 세션에선 확인 불가(`gh` 권한 없음).
- 승인 후 남는 우려 ①: `ci-env.mjs:12` 새 주석이 "옆 스크립트들은 DATABASE_URL 에서 같은 DB 를 받는다" 고 적지만 `e2e.mjs:28` 은 `APP_DATABASE_URL` 이 따로 필요하고 `dev-reset.sh:9,11` 은 :55447 을 박아 둔다 — 전용 DB 격리 절차(세 변수 + dev-reset.sh 우회)가 저장소 어디에도 없어 다음 회차가 같은 반쪽 상태에 또 빠질 수 있다. 원장 보류 ① 대로 README '검증' 에 한 번 적는 것이 맞다.
- 승인 후 남는 우려 ②: `WORKKIT_DEV_DB_HOSTPORT=''`(빈 문자열)은 `??` 를 통과해 `postgres://…@/workkit` 로 망가진다(개발 전용·의도된 선택). 릴리즈 노트는 이 회차를 **검증 환경 정비**로만 적어야 한다 — CI 결제 차단을 고치는 변경이 아니고 사용자 동작도 바뀌지 않는다(VERSION 은 앞 커밋의 0.3.2 그대로).
- [러너 12:00] review approved — 리뷰 승인 (risk=low)
- [러너 12:00] pr created — https://github.com/hkjang/workkit.irumx.app/pull/6
- [러너 12:00] ci failed — 성공이 아닌 검사: check=failure · 실패한 검사: ? 잡: check 

## 수리 노트
- 지적 중 **틀린 것**: "CI check 실패 = 이 변경의 결함". 3초·무로그 실패는 잡 미시작(계정 쪽 Actions 차단) 신호이고, 워크플로는 YAML 유효 + v0.2.1 이후 무변경이며 diff 는 2파일뿐이다. 그래서 **커밋하지 않았다**(근거는 fix-summary.md).
- 지적 중 **맞았던 것**: 아무도 CI 와 같은 검증을 끝까지 돌려 보이지 않았다는 점. 이번에 전용 DB(:55497 컨테이너, 상주 dev DB 는 무접촉)로 CI 15단계를 모두 완주했다 — check 138 passed · build · kits manifest `cmp` 일치 · e2e 155/0 · tools 55/0 · ui 25/0 · a11y 위반 없음. 환경변수 없는 `.env.dev` 는 main 판과 `cmp` 바이트 동일(980B).
- 고친 방법: 없음(코드 무변경). 끝나고 fake 서버·앱 서버·전용 컨테이너를 내리고 `.env.dev` 를 기본값으로 되돌렸다 — 워크트리 트리 clean.
- 여전히 확신 없는 곳: ① 결제 차단은 **직접 확인하지 못했다**(`gh` 미인증·사설 404). 3초 실패의 다른 원인(러너 미할당 등)도 원리상 배제 못 하지만, 어느 쪽이든 저장소 코드로는 고칠 수 없다. ② a11y·tools·ui 는 `--with-deps` 없이 받은 chromium 으로 돌렸다(이 환경에선 통과).
- 다음 역할에게: 비평가 우려 ①(전용 DB 격리 절차를 README '검증' 에 기록)·②(`WORKKIT_DEV_DB_HOSTPORT=''` 방어)는 CI 차단과 무관해 손대지 않았다 — 별도 회차 감.
