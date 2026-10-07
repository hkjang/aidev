- 과제: DST shift 예약의 다음 실행 순서와 조회 개수 일관성 수정 (가치 4 / 위험 2 / 작업량 M)
- 왜: `src/shared/cron.ts:nextRuns`는 벽시계 순서로 후보 n개를 모으자마자 중단해서 30분 서머타임 전환 때 더 이른 UTC 후보를 놓친다. 실제 예약 갱신과 상세 화면의 `nextAfter`는 n=1을 사용하므로, 미리보기와 실행 순서를 일치시켜 예정된 회차 누락을 막아야 한다.
- 수용 기준:
  1) cron `15,30 2 * * *`, timezone `Australia/Lord_Howe`, after `2026-10-03T12:00:00Z`, gapPolicy `shift`에서 n=1/2/5의 첫 실행은 모두 `2026-10-03T15:30:00.000Z`(현지 02:30)이다. n=2는 15:30Z, 15:45Z 순서이고 15:45Z(없는 02:15를 02:45로 이동)에만 shifted 표시가 있다.
  2) 같은 조건에서 n=1 반복 호출 결과가 한 번에 n=5를 얻은 결과와 같고, UTC는 엄격히 증가하며 중복이 없다. after와 같은 순간은 제외한다. 기존 KST, New York skip/shift, 가을 overlap_once 테스트가 그대로 통과한다.
  3) 실제 로컬 Worker의 `GET /api/w/:w/schedules/:id` 응답 upcoming에서 아래 고정 날짜 회귀를 증명한다. 소스 문자열 검사나 nextAfter 대역으로 대신하지 않는다.
- 건드릴 파일:
  - `src/shared/cron.ts:nextRuns` — shift로 UTC 순서가 뒤집힐 수 있는 날에는 더 이른 후보를 검토하기 전에 n개 제한으로 중단하지 않게 한다. 일치 날짜의 후보를 UTC 정렬·중복 제거한 뒤 필요한 개수를 선택하는 좁은 수정이 기본안이다. skip의 기존 빠른 경로는 가능한 유지한다.
  - `tests/unit-core.spec.ts:test.describe('시각·예약')` — 위 고정 날짜와 n별 접두 일치, 순차 조회, after 경계, 중복 제거를 추가한다.
  - `tests/api-schedule.spec.ts` — 기존 person/addModel/publishAgent/sql 헬퍼를 써 실제 HTTP 상세 응답을 검증한다.
  - 프로덕션 1개 + 기존 테스트 2개로 제한. 아래 참조 파일들은 읽기 전용이다.
- 검증 명령: 저장소 루트에서 아래 명령을 사용한다. Node 최소 재현은 실제 실행했고 현재 실패한다. npm 검사 및 Playwright 전체 실행은 환경 제약을 아래에 별도 기록했다.
- 위험과 피할 것: cron 문법·dayMatches·localToUtc·타임존 정책을 함께 바꾸거나 파서를 통합하지 않는다. auth, migrations, workflows, 위임·권한·실행 엔진·API 계약은 변경하지 않는다. 전체 5년의 분 단위 후보를 한 번에 만들지 말고 하루 안에서 처리하며 기존 5년 탐색 상한을 유지한다. 감사 details에 입력 원문을 추가하지 않는다. LIVE=1, 배포, 원격 D1, 운영 메일은 쓰지 않는다.
- 차선 후보: 예약 repeat 입력 검증 — undefined/null/빈 객체/숫자 time에서 repeatToCron이 TypeError를 던지는 것을 Node로 확인했다. `src/worker/domain/schedules.ts:planSchedule`의 입력 경계에서 400으로 거절하고 API 테스트를 붙이는 별도 과제로 한정한다. HTTP 500 응답 자체는 미확인이다. 1순위가 이미 수정되어 재현되지 않을 때만 전환한다.

확인한 근거와 범위

- 기준 커밋 main@0e65235(v0.8.1). 과거 개선 기록·보류 항목·기각 기록은 제공되지 않았다.
- 실제 nextRuns 출력: n=1은 15:45Z, n=2/5는 15:30Z부터 시작한다. 동일 함수의 반환 계약 위반이라 외부 cron 규격 해석에 의존하지 않는다.
- `src/worker/domain/schedules.ts:planSchedule`는 n=6을 받아 5회를 노출한다. `nextAfter`는 n=1을 사용하고 `tickOne`은 이를 반복해 실제 다음 회차를 갱신한다.
- `src/worker/api/schedules.ts`의 상세 GET은 저장된 next_run_at부터 nextAfter를 반복하여 upcoming을 만든다. 따라서 상세 HTTP 테스트로 배선과 최종 출력까지 검증할 수 있다. 실제 큐 실행까지의 새 회귀 시나리오는 이번 범위에 넣지 않는다.
- 대안 비교: 하루 내 UTC 후보 정렬은 기존 시각 변환 계약을 보존하고 1개 파일로 끝난다(선택). UTC 타임라인을 전면 열거하는 재설계는 비용·DST 회귀 위험이 커 제외한다. shift를 거절하거나 skip으로 바꾸는 방법은 기존 사용자 선택을 깨므로 제외한다. 현상 유지는 실제 회차 누락 때문에 선택하지 않는다.
- 핵심 가정: 기존 localToUtc의 30분 전환 변환값은 맞고, 제한을 적용하는 순서가 결함이다. 실제 출력과 n=2 결과로 확인했다. 다른 역사적 시간대의 날짜 단위 전환까지 정합성을 증명한 것은 아니다.

구현 순서와 검증 지점 (모두 미착수, 사람 확인 대기 없음)

1. [ ] 아래 Node 재현으로 결함을 확인하고 의존성을 준비한다. 정찰에서는 저장소 쓰기가 금지되어 설치하지 않았다. 구현 단계에서는 `npm ci` 후 `npm run check`로 기준 상태를 확인한다. 설치/기준 검사 실패는 기능 결함과 구분해 기록하며 lockfile 갱신이나 버전 업그레이드로 우회하지 않는다.
2. [ ] nextRuns의 좁은 수정과 unit 회귀 테스트를 함께 작성한다. 아래 Node 명령 통과 후 `npm run build`, `npx playwright test --project=unit tests/unit-core.spec.ts`로 검증한다. 실패하면 다음 단계로 넘어가지 않고 원인을 기록한다.
3. [ ] API 회귀를 추가한다. person → addModel → publishAgent → POST schedules로 `{spaceId, agentId, input:'DST ordering regression', kind:'recurring', repeat:{every:'cron',expr:'15,30 2 * * *'}, timezone:'Australia/Lord_Howe', gapPolicy:'shift'}` 예약을 만든다. POST pause로 자동 Cron 실행을 막은 뒤 기존 sql 헬퍼로 해당 예약의 next_run_at만 `Date.parse('2026-10-02T16:00:00Z')`로 고정한다. GET 상세의 upcoming 첫 3개 utcIso가 `2026-10-02T16:00:00.000Z`, `2026-10-03T15:30:00.000Z`, `2026-10-03T15:45:00.000Z`인지 확인한다. 현재 코드는 두 번째가 15:45Z여서 회차 하나가 빠진다. 이 방법은 서버 시계 조작이나 계산 함수 대역 없이 날짜와 동작을 고정한다.
4. [ ] `npm run check` 및 `npx playwright test --project=unit --project=api tests/unit-core.spec.ts tests/api-schedule.spec.ts`를 통과시키고 변경 파일 수를 확인한다. 증거와 실제 실행한 명령을 구현 노트에 남긴다. 현실이 계획과 다르면 범위를 늘리지 말고 이 계획과 기록을 먼저 수정한다.

실행 가능한 최소 재현 (Node v22.23.1에서 실행 확인)

```bash
node --experimental-strip-types --input-type=module <<'JS'
import assert from 'node:assert/strict';
import {nextRuns, parseCron} from './src/shared/cron.ts';
const spec = parseCron('15,30 2 * * *');
const after = Date.parse('2026-10-03T12:00:00Z');
const one = nextRuns(spec, 'Australia/Lord_Howe', after, 1, 'shift');
const five = nextRuns(spec, 'Australia/Lord_Howe', after, 5, 'shift');
assert.equal(new Date(five[0].utc).toISOString(), '2026-10-03T15:30:00.000Z');
assert.equal(one[0].utc, five[0].utc);
JS
```

정찰 검증 결과와 환경 함정

- 실제 src/shared/cron.ts를 import한 Node test 검사: KST 1개, New York gap(skip/shift) 1개, overlap 1개 통과; Lord Howe 접두 일치 1개 실패(기대한 결함 재현). 3 pass / 1 fail. 저장소 Playwright 통과 수로 해석하지 않는다.
- `npm run check`: exit 127, `tsc: not found`. `npm test -- --list`: @playwright/test 모듈 없음. 이 환경에서 빌드/API 통과 여부는 미확인이다. 사용자 지정 쓰기 범위를 지키려고 의존성 설치와 dist 생성은 하지 않았다.
- `playwright.config.ts`는 unit만 선택해도 공통 webServer 5개를 시작한다. dist가 필요하므로 먼저 npm run build가 필요하다. 로컬 포트 8870~8875, 테스트 DB .wrangler/test-state 사용. db-local.mjs는 지정 로컬 상태를 삭제하고 마이그레이션을 적용하므로 개인 개발 상태를 넘기지 않는다.
- reuseExistingServer=true이므로 기존 8870 서버가 살아 있으면 구 빌드를 테스트할 수 있다. 구현자가 직접 시작한 테스트 서버인지 확인하여 새 빌드로 실행할 것. 무관한 프로세스를 종료하지 않는다.

추정 근거 (estimating-and-contingency 적용)

- 상향식 기준 33분: 환경 준비·재현 5 + 계산/단위 회귀 10 + HTTP 회귀 10 + 빌드/검증/노트 8. 알려진 변수인 테스트 기동·시간대 경계 확인에 예비 7분을 별도로 두어 목표 40분, 예상 범위 30~45분으로 잡는다.
- 범위의 신뢰도는 중간(주관적 판단이며 통계적 보장 아님). 작은 함수 1개와 이미 있는 헬퍼가 근거다. 비교 가능한 과거 회차/분당 생산성 데이터는 없어 유사·모수 추정값은 만들지 않았다.
- 전제: npm 의존성을 정상 설치할 수 있고 로컬 Worker가 기존 구성으로 기동한다. 첫 환경 확인 후 남은 시간을 재추정한다. 장시간 설치 장애나 별도 기존 실패 수리는 포함하지 않는다. 관리 예비는 이번 회차에 배정하지 않으며 새 범위에 쓰지 않는다.
- 스킬 적용: 로컬 headcount의 pmo/estimating-and-contingency, technology/implementation-planning, technology/solution-exploration SKILL.md를 직접 읽음. 이 세션에는 Skill 호출 도구가 없어 파일로 확인했다. 추정의 외부 권위 수치나 통계는 인용하지 않았다.
