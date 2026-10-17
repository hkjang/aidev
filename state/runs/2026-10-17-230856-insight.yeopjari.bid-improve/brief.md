- 과제: PR #4 의 check 워크플로가 3초 만에 실패하는 원인을 확정해 고치기 (수정 과제, 가치 4 / 위험 2 / 작업량 M)
- 왜: PR #4(`auto/2026-10-17-1523`, 커밋 `b55ab2d`)의 `check` 잡이 같은 이유로 두 번 실패해 머지되지 못했는데, 같은 커밋의 `npm run check`(7파일 113개)·`npm run build` 는 로컬에서 전부 통과한 기록이 남아 있다(이전 회차 `verify.txt`). 원인을 확정하지 못하면 다음 회차도 같은 자리에서 멈춘다.
- 수용 기준: 1) 실패한 체크런의 **주석(annotation) 원문과 실패 단계 이름**을 받아 과제 기록에 그대로 남긴다. 2) 원인이 저장소 쪽(코드·테스트·스크립트·워크플로 설정 오류)이면 고치고, 같은 검증을 로컬에서 재현해 통과를 보인다. 3) 원인이 GitHub 쪽(과금·러너·서비스 컨테이너 기동 실패 등 저장소가 못 고치는 것)이면 **check.yml 을 건드리지 말고** 그 근거(주석 원문·잡 소요 3초)를 적고 재실행으로 끝낸다. 4) 어느 경우든 게이트를 약하게 만드는 변경(단계 삭제·`continue-on-error`·타임아웃 늘리기·테스트 skip)은 넣지 않는다.

- 확정된 사실 (이번 정찰이 직접 읽은 것)
  - 실패한 체크런: `https://github.com/hkjang/insight.yeopjari.bid/actions/runs/39010546706/job/117331978771`, check_run id `117331978771`, head_sha `b55ab2d`, base `main@601659d`.
    (출처: `/mnt/c/Users/USER/projects/aidev/state/runs/2026-10-17-152316-insight.yeopjari.bid-improve/ci-b55ab2de5485.json`)
  - `status: completed`, `conclusion: failure`, `started_at 2026-10-17T06:36:37Z` → `completed_at 2026-10-17T06:36:40Z` — **잡 전체가 3초**, `annotations_count: 2`.
  - 3초는 `actions/checkout` + `setup-node` + `npm ci` 가 도는 시간이 아니다. 즉 **실패는 단계 스크립트 안이 아니라 잡이 단계에 들어가기 전/직후**다. 같은 파일의 `ci-failure-b55ab2de5485.txt` 에 실패 단계가 비어 있는 것은 러너가 단계 정보를 받아오지 않은 것이지, 단계가 없다는 뜻이 아니다.
  - 같은 커밋의 로컬 검증은 통과했다: `verify.txt` 에 `Test Files 7 passed (7) / Tests 113 passed (113)` 과 web+server 빌드 성공이 그대로 남아 있다.
  - 변경 범위는 2개 파일뿐이다: `apps/web/src/api/client.ts`(`download()` 의 fetch try/catch + `db_unavailable` 일 때 `reportDegraded()`), 신규 `apps/web/src/__tests__/client-download.test.ts`. 워크플로·서버·DB 코드는 건드리지 않았다(`git show --stat b55ab2d` 로 확인).
  - `.github/workflows/check.yml` 은 잡 1개(`check`), `services.postgres: postgres:16-alpine`, 포트 `55442:5432`, 단계 순서 `npm ci → npm run check → npm run build → scripts/ci-env.mjs → scripts/migrate.mjs → 서버 기동 → scripts/e2e.mjs → playwright install → scripts/screenshots.mjs`. 느슨하게 만들 자리는 없고, 이번에도 만들지 않는다.
  - 미확인: 주석 2건의 **원문**. 이번 정찰 환경에서는 `gh` 호출이 승인되지 않아 받지 못했다. 이것이 원인 확정의 유일한 결정적 증거다.

- 순서 (이 순서대로, 각 단계의 결과를 보고 다음으로)
  1. 주석과 로그부터 받는다 — 추측으로 코드를 고치지 말 것:
     - `gh api repos/hkjang/insight.yeopjari.bid/check-runs/117331978771/annotations`
     - `gh run view 39010546706 --log-failed`  (안 되면 `gh run view 39010546706`)
     - `gh api repos/hkjang/insight.yeopjari.bid/actions/runs/39010546706/jobs --jq '.jobs[] | {name, conclusion, steps: [.steps[] | {name, conclusion}]}'`
     - `gh pr checks 4` 로 두 번째 실패(재실행 또는 두 번째 커밋)가 같은 메시지인지 확인한다.
  2. 분기 A — 주석이 **저장소 원인**을 가리키면(예: 워크플로 YAML 파싱 오류, 참조 액션 없음, 스크립트 오류, lockfile 불일치, 테스트 실패) 그 원인만 고친다. 고친 뒤 아래 "검증 명령" 의 로컬 재현을 돌려 통과를 보인다.
  3. 분기 B — 주석이 **GitHub 쪽 원인**을 가리키면(예: `The job was not started because recent account payments have failed…`, 스펜딩 리밋, 러너 할당 실패, `Failed to initialize container postgres:16-alpine`) 코드를 고치지 않는다. 주석 원문을 원장에 적고, `gh run rerun 39010546706 --failed` 로 재실행한 결과를 기록한다. 서비스 컨테이너 기동 실패가 반복되면 그때만 `check.yml` 의 서비스 설정을 **동등하거나 더 엄격하게** 바꾸는 것까지 검토하되(예: health 재시도 유지한 채 이미지 태그 고정), 단계 삭제·`continue-on-error` 는 금지다.
  4. 어느 분기든 확인한 주석 원문과 결론을 회차 노트/원장에 '수정 과제' 로 남긴다. 원인을 못 밝히면 "밝히지 못했다" 고 적는다 — 추정으로 단정하지 말 것.

- 건드릴 파일 (분기 A 에서만, 주석이 가리키는 곳으로 좁혀서. 아래는 후보이지 예정이 아니다)
  - `.github/workflows/check.yml` — 설정 오류가 확인된 경우에만, 엄격함을 유지한 수정.
  - `apps/web/src/__tests__/client-download.test.ts` — 테스트가 CI 에서만 깨진 것으로 드러난 경우. 알려진 환경 의존 두 군데: (a) `http://127.0.0.1:1` 로 연결 거절을 만드는 네트워크 실패 테스트, (b) `globalThis.window`/`document`/`URL.createObjectURL` 주입(`afterEach` 복원). 둘 다 **삭제하지 말고** 환경에 안 기대는 방식으로 좁힐 것.
  - `apps/web/src/api/client.ts` — 프로덕션 결함이 CI 에서만 드러난 경우에만. `request()`/`download()` 통합은 금지(성공 경로 계약이 blob 저장 vs JSON 파싱으로 다름).
  - 합계 3개를 넘기지 말 것. 넘어가면 과제를 쪼개고 이번 회차엔 한 조각만.

- 검증 명령 (이 저장소에서 실제로 도는 것, 워크트리 안에서)
  - `npm ci --no-audit --fund=false` — 끝나고 `git status --short` 에 lockfile 변동이 없어야 한다.
  - `npm run check` — eslint + core/server/web 타입 검사 + vitest. 기준 기대값: `Test Files 7 passed (7) / Tests 113 passed (113)`.
  - `npm run build`
  - 단일 파일: `npm run test -- apps/web/src/__tests__/client-download.test.ts`
  - DB 가 필요한 뒷단(`scripts/migrate.mjs` → 서버 → `node scripts/e2e.mjs`)은 2026-10-08 회차와 같은 **폐기용 Postgres 16 컨테이너 + 분리된 포트**로만 돌릴 것:
    `BASE=http://localhost:18791 DATABASE_URL=postgres://postgres@127.0.0.1:55484/insight APP_DATABASE_URL=postgres://insight_app@127.0.0.1:55484/insight node scripts/e2e.mjs`
    (역할 이름은 `irumx_gateway` → `insight_app`. `insight_gateway` 로 추측하면 실패한다 — 과거에 걸린 자리다.)

- 위험과 피할 것
  - **`node scripts/ci-env.mjs` 와 `scripts/dev-reset.sh` 를 기존 개발 환경에 돌리지 말 것** — 설정(.env.dev)을 덮어쓰고 DB 를 초기화한다. 꼭 필요하면 워크트리 안에서, 기존 파일이 없음을 먼저 확인하고.
  - 워크플로를 통과시키려고 느슨하게 만드는 변경 금지(명시적 금지 사항).
  - `db.ts`·`errors.ts`·`migrations`·`auth/*`·`routes/sso.ts`·`http/pipeline.ts` 는 이번 과제와 무관하다 — 열어 보기만 하고 바꾸지 말 것.
  - 원인을 못 본 채 "아마 이것이겠지" 로 테스트를 바꾸면 다음 회차에 같은 실패가 또 난다. 주석 원문을 못 받으면 분기 B 의 기록만 남기고 코드는 그대로 두는 것이 옳다.
  - 가짜 OAuth 는 `:18999` 고정이고 기존 서비스를 끄지 말 것.

- 차선 후보: apps/web client 의 나머지 가드 테스트 보강 (가치 3 / 위험 1 / S) — `client-download.test.ts` 가 만든 node 환경 브라우저 전역 부트스트랩을 그대로 써서, `isAdminSurface()` 에 따른 `yi_acsrf`/`yi_csrf` 선택과 상태 코드별 `defaultMessage` 를 실제 HTTP 응답으로 못 박는다. jsdom 추가 금지. (1순위가 "저장소 원인 없음" 으로 끝나 시간이 남을 때만.)
