- 과제: 오류 stderr 로그 세 경로를 redactSecrets 로 통과시키기 — after-commit 실패 로그가 운영에서 원문을 찍는다 (가치 4 / 위험 1 / 작업량 S)

- 왜: `packages/core/src/http/pipeline.ts` 에서 오류를 stderr 로 쓰는 곳은 네 군데인데 `redactSecrets` 를 거치는 것은 `logFailure` 의 JSON 한 줄(:487)뿐이다. 나머지 세 곳 중 **:54 `Background.run` 의 catch 는 `env.dev` 게이트가 없어 운영에서 그대로 돈다** — `detail: String(err).slice(0, 300)`. 이 경로에는 after-commit 작업(`:208`, `:411` → `jobs/kick.ts:kickOutbox` → `outbox.ts:processOutbox` → `jobs/handlers.ts`: 문서 스캔·내보내기 빌드·알림 발송)의 실패가 전부 모이므로, 공급자 응답이나 DB 연결 오류에 섞인 postgres DSN·Bearer 토큰이 운영 로그에 원문으로 남는다. `redactSecrets` 가 DSN 규칙(:501)을 갖고 있는 이유가 바로 오류 문구에 DSN 이 섞여 나오기 때문이다. 나머지 둘(:111 `[dev]` 한 줄, :489 원본 스택)은 `env.dev` 전용이지만 같은 1줄 수정이고 개발·스테이징 로그에 남는 문제라 함께 닫는다.

- 수용 기준:
  1) `Background.run` 의 catch 가 `redactSecrets(String(err))` 를 먼저 적용한 뒤 `.slice(0, 300)` 한다 — **순서가 중요하다**. 300자에서 먼저 자르면 DSN·Bearer 가 중간에서 끊겨 패턴이 어긋나 치환을 빠져나간다. `:487` 의 `redactSecrets(message).slice(0, 300)` 와 같은 순서로 맞출 것.
  2) `:111` 의 `[dev] ...` 한 줄과 `:489` 의 `err.stack` 도 `redactSecrets` 를 통과한다.
  3) 네 경로 모두 출력 형식(JSON 필드 이름·순서, `[dev] ${method} ${path}: ` 접두사)은 그대로 — 로그 파서가 있다면 깨지지 않는다.
  4) 테스트가 **운영 배선을 실제로 지나는 실행 테스트**로 증명한다: `new Background()`(파이프라인이 `:208`/`:411` 에서 쓰는 바로 그 export 된 클래스, 대역 주입 없음) 를 만들고 `vi.spyOn(console, 'error')` 로 출력을 받은 뒤, `bg.run('테스트', async () => { throw new Error('...postgres://law_app:s3cret@db.example/law... Bearer abcdefgh12345678') })` → `await bg.settled()` → 캡처된 모든 줄에 `s3cret` 과 토큰 원문이 **없고** `[숨김]` 이 있음을 단정한다. 수정 전에는 실패해야 한다(실패 재현을 기록에 남길 것).
  5) `npm run check` 가 통과하고, 기존 vitest 파일 수·통과 수가 줄지 않는다.

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `packages/core/src/http/pipeline.ts:54` — `Background.run` 의 catch: `detail: redactSecrets(String(err)).slice(0, 300)`
  - `packages/core/src/http/pipeline.ts:111` — `dispatchRequest` 의 dev 로그: 템플릿 문자열 전체 또는 메시지 부분을 `redactSecrets` 로 감쌈
  - `packages/core/src/http/pipeline.ts:489` — `logFailure` 의 스택 출력: `console.error(redactSecrets(err.stack))`
  - `packages/core/src/__tests__/background-redact.test.ts` — **신규. 이 파일명을 쓸 것.** `redact-secrets.test.ts` 라는 이름은 쓰지 말 것: 아직 머지되지 않은 선행 회차(review-pending)가 그 경로에 파일을 추가한다 — 같은 이름이면 충돌한다.

- 검증 명령:
  - `npm run check` (= lint + typecheck + vitest). 단위 시험만으로 충분하다 — DB 불필요.
  - 좁게 돌릴 때: `npx vitest run packages/core/src/__tests__/background-redact.test.ts`
  - 주의: 이 정찰 세션은 샌드박스 제한으로 vitest 를 실행하지 못했다 — **기준선 통과 여부는 미확인**. 구현자는 수정 전에 `npm run check` 를 한 번 돌려 기준선(직전 기록: vitest 6 files 103 tests, lint 0 errors / 646 warnings)을 먼저 확인할 것.

- 위험과 피할 것:
  - **`redactSecrets` 의 정규식을 넓히지 말 것** (`:496-504`). 운영자 지시: 같은 종류의 검사는 좁히는 방향으로만. 이번 과제는 호출 지점을 늘리는 것이지 패턴을 바꾸는 것이 아니다. 특히 주민번호 패턴(`:503`)의 과잉 차단은 알려진 사안이고 사람 판단 대기 중이다 — 손대지 말 것.
  - `alertOperator`(`:137-165`)는 이미 `redactSecrets` 를 통과한다(`:145`). 여기서 DB `record_error_alert` 로 넘기는 `message` 는 **이미 가려진 값**이다. 건드리지 말 것 — 운영자 지시(검출 대상 원문을 감사 details 로 넘기지 말 것)가 이미 지켜져 있다.
  - `auth/*`, `db/migrations/*`, `.github/workflows/*` 는 건드리지 않는다. 이 과제는 그럴 필요가 없다.
  - `Background` 클래스의 `pending`/`keepAlive`/`settled` 동작은 바꾸지 말 것 — 셧다운 대기에 쓰이고 공유 DB 연결 문제와 얽힌 곳이다. catch 안의 `detail` 한 조각만 바꾼다.
  - 테스트에서 `console.error` 를 spy 로 바꿀 때 `afterEach` 로 반드시 복원할 것(`vi.restoreAllMocks()`). 복원하지 않으면 뒤에 도는 파일의 로그가 삼켜져 진단이 어려워진다.
  - 운영자 지시대로 **손으로 만든 대역이 아니라 진짜 객체**로 증명한다. `logFailure` 는 비공개 함수라 `handleRequest` 전체 deps(Database·Settings·Router)가 필요하다 — 가짜 deps 를 조립해 `:111`/`:489` 를 시험하려 들지 말 것. 그 둘은 dev 전용·1줄 변경이므로 `Background` 경로의 실행 테스트 + 코드 검토로 닫고, 그 사실을 PR 설명에 적을 것.

- 차선 후보: **`util/validate.ts` 경계 테스트** (가치 3 / 위험 1 / 작업량 M, 프로덕션 0파일) — 모든 라우트 입력이 지나가는 검증기인데 테스트가 0건이다. `optional` 의 `''` 취급, `text` 의 `min:0`, `array` 의 인덱스별 오류 경로, `int` 의 문자열 강제변환, `koreanParticle` 조사 메시지를 현재 동작 그대로 고정한다. 주의: 이 정찰은 `validate.ts` 내부를 열지 않았다 — 함수 목록은 이전 회차 기록 기준이며 **미확인**이다. 구현자가 먼저 파일을 열어 실제 export 를 확인하고 시작할 것.
