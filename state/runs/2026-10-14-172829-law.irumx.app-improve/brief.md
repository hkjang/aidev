- 과제: PR #5 (auto/2026-10-14-1238 / ffd2bf6, AT-26 양성 대조) 의 CI 실패 원인을 확정하고, 저장소 안의 결함이면 고친다 (가치 4 / 위험 2 / 작업량 S)
- 왜: 러너가 이번 회차를 'verify-failed → fix-round: CI failed, PR open #5' 로 적어 두었다. 정찰 세션에서는 `gh` 가 권한으로 막혀(아래 "확인 못 한 것") CI 로그를 읽지 못했으므로, 실패가 저장소 안(코드/스크립트)인지 밖(Actions 결제 차단)인지 **아직 확정되지 않았다**. 확정하지 않으면 다음 회차도 같은 자리에 멈춘다.
- 수용 기준:
  1) PR #5 의 실패한 **단계 이름과 로그 원문 한 줄**을 journal.md 에 인용한다 (`gh run list --branch auto/2026-10-14-1238`, `gh run view <id> --log-failed`).
  2) 실패가 저장소 안이면: 원인을 고치고, **CI 와 같은 순서로 로컬에서 재현해 고치기 전 실패 / 고친 뒤 통과**를 둘 다 출력으로 남긴다. 워크플로(`.github/workflows/check.yml`)나 검사 기준을 느슨하게 해서 통과시키는 것은 금지 — 조이는 방향만.
  3) 실패가 저장소 밖(결제/할당량으로 잡이 시작조차 못 함: 잡 소요 ~3초, `The job was not started because recent account payments have failed or your spending limit needs to be increased.`)이면: 그 로그 인용을 증거로 "코드 결함 없음" 을 원장에 적고, **차선 후보로 넘어가 회차를 채운다**. 이미 두 회차(2026-10-14 1028·1238)가 같은 결론에 도달했으므로 세 번째 전수 재현은 하지 말고, 아래 2)의 축소판(단위+빌드만)으로 끝낸다.
  4) `npm run check` 가 통과한다(기준선: lint 0 errors / 646 warnings / typecheck 3개 워크스페이스 / vitest).

- 건드릴 파일 (실패가 저장소 안일 때만, 그리고 원인이 가리키는 파일만):
  - `scripts/acceptance.mjs` — PR #5 가 바꾼 유일한 파일(7+/1-). AT-26 구간 `main()` 안, 현재 main 기준 959행 근처. 바뀐 내용: `logPath = process.env.LAW_SERVER_LOG ?? new URL('../data/server.log', …)`, `existsSync(logPath) ? readFileSync(…) : ''`, 그리고 양성 대조 `ok('서버 로그가 이 실행의 것임(가려진 메일 기록 존재)', /"channel":"mail"[^\n]*\*+@example\.test/.test(log), …)`.
  - 그 외 파일은 CI 로그가 직접 가리킬 때만 연다. **프로덕션 코드는 건드리지 않는 것이 기본값** — PR #5 는 scripts 1파일짜리 변경이다.

- 정찰이 미리 확인해 둔 것 (추측 아님, 실제로 열어 봄):
  - `.github/workflows/check.yml` 은 87줄, **릴리즈 워크플로는 없다**(워크플로 파일이 이 하나뿐). 단계 순서: apt postgresql-client-17 → `npm ci` → `npm run check` → `npm run build` → `node scripts/ci-env.mjs` → migrate → RLS 프로브(`db/tests/rls-probe.sql`) → mock-providers+server 기동(`nohup … > server.log`) → `node scripts/e2e.mjs` → DROP/CREATE DATABASE + migrate + `bootstrap.js` + `rm -rf data/media` + `node scripts/acceptance.mjs`. 잡 env 에 `LAW_SERVER_LOG: server.log`, `LAW_LOAD_P95_MS: '3000'`, `timeout-minutes: 40`.
  - **`existsSync` 는 이미 import 되어 있다** — `scripts/acceptance.mjs:15` (`import { existsSync, readFileSync, writeFileSync } from 'node:fs'`), 751행에서도 이미 쓴다. 따라서 PR #5 의 `existsSync` 사용은 ReferenceError 를 내지 않는다. (이 가설은 정찰이 확인해 **배제**했다.)
  - 양성 대조 정규식은 실제 로그 형식과 맞는다 — `packages/core/src/mail/mailer.ts:71` 이 `console.log(JSON.stringify({ level:'info', channel:'mail', driver:'console', to: maskAddress(message.to), … }))` 를 찍고, `maskAddress`(:146-150)는 `ab@example.test` → `ab***@example.test` 를 만들어 `\*+@example\.test` 에 걸린다. (이 가설도 **배제**.)
  - 단, 대조의 **강도는 약하다**: CI 에서 `server.log` 는 서버 기동 단계부터 누적되고 e2e 단계가 이미 메일을 보내므로, acceptance 가 메일을 한 통도 안 보내도 e2e 가 남긴 줄로 대조가 통과한다. 이것은 CI 실패 원인은 **아니고**(통과하는 방향), 조이고 싶으면 별도 회차 거리다.
  - 작업 트리는 `main@c62b3a0` 기준이고 PR #5 의 ffd2bf6 은 **머지되어 있지 않다**. 로컬 재현 전에 `git cherry-pick ffd2bf6` 또는 `git checkout origin/auto/2026-10-14-1238` 로 PR 내용을 먼저 올려야 한다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `npm run check`  (= `npm run lint` + `npm run typecheck` + `npm run test`; DB 불필요)
  - `npm run test`   (vitest 단위만, 빠름)
  - `npm run build`
  - CI 후반을 재현할 때만(오래 걸림, docker 필요): postgres:17-alpine 을 55460 에 띄우고 `LAW_PG_ADMIN_URL=postgres://postgres:devpass@127.0.0.1:55460/law node scripts/ci-env.mjs` → `node scripts/migrate.mjs` → `psql … -f db/tests/rls-probe.sql` → 서버 기동 → `node scripts/e2e.mjs` → `LAW_SERVER_LOG=server.log node scripts/acceptance.mjs`. **check.yml 의 단계를 그대로 베껴 쓸 것** — 순서를 바꾸면 빈 DB 전제가 깨져 실패한다.

- 위험과 피할 것:
  - **워크플로 파일을 고쳐 통과시키지 말 것** (지시에 명시). 기준·정규식·타임아웃을 넓히는 변경 금지.
  - `db/migrations/*`, `packages/core/src/auth/*`, `modules/payments.ts` 는 이번 과제 범위 밖 — 열지 말 것.
  - 두 회차가 이미 "CI 실패 = Actions 결제 차단(저장소 밖)" 이라는 증거를 모았다. 같은 전수 로컬 재현을 세 번째로 돌리는 것은 시간만 쓰고 판정이 '차선' 으로 끝났다 — 수용 기준 3) 의 축소 경로를 쓸 것.
  - `scripts/acceptance.mjs` 는 AT-26 뿐 아니라 뒤의 네 검사와 `data/acceptance.json` 출력 형식을 공유한다. 출력 형식을 바꾸면 CI 의 업로드 단계가 아니라 다음 회차의 비교가 깨진다.

- 차선 후보: **util/validate.ts 경계 테스트** (가치 3 / 위험 1 / 작업량 M, 프로덕션 0파일). 모든 라우트 입력이 지나가는 검증기인데 테스트가 0건이다. 신규 `packages/core/src/__tests__/validate-boundaries.test.ts` 에 현재 동작을 그대로 고정하는 특성화 테스트를 쓴다 — 검증기를 넓히거나 고치지 말고 **기록만** 할 것.
  - 정찰이 **실제로 확인한 export**(`packages/core/src/util/validate.ts`): `Validator<T>` 타입(:13, `run(value, path, errors)`), `parse(shape, input)`(:23), `parseOne(validator, value, path)`(:36), 그리고 `v` 객체(:52)의 멤버 13개 — `string`(:54) `text`(:72) `int`(:97) `bool`(:115) `uuid`(:126) `email`(:137) `instant`(:153) `oneOf`(:164) `array`(:181) `object`(:198) `json`(:216) `optional`(:221).
  - 한 회차에 13개 전부는 M 을 넘는다. **`string`·`text`·`int`·`optional` 네 개만** 고정하는 것으로 쪼개라. 이미 읽어 확인한 경계: `string` 은 `trim()` 후 길이를 재고 `min` 기본값이 1이라 공백만 입력은 "입력해 주세요", `text` 는 `\r\n?`→`\n` 정규화 + 제어문자(`\u0000-\u0008\u000b\u000c\u000e-\u001f`) 제거 후 `min:0` 이면 공백 입력을 `''` 로 통과시키고 길이는 `[...text].length`(코드포인트)로 재며, `int` 는 공백 아닌 문자열을 `Number()` 로 강제변환한 뒤 `Number.isInteger` 를 본다. 오류 메시지는 `eul`/`eun` 조사 헬퍼를 거치므로 **메시지 문자열까지 그대로 고정**할 것.
  - 성공 판정은 `npm run check` 통과 + 조건 한 줄씩 지워 보는 돌연변이로 테스트가 프로덕션 경로를 실제로 지난다는 증명(2026-10-14 redactSecrets 회차와 같은 방식).

- 확인 못 한 것 (미확인 — 구현자가 직접 확인할 것):
  - **PR #5 의 CI 실패 로그 자체**. 이 정찰 세션에서는 `gh pr view 5` / `gh pr checks 5` / `gh run list` 가 모두 "requires approval" 로 거부되어 실패 단계와 원문을 읽지 못했다. 수용 기준 1) 이 이것이다.
  - **기준선 테스트 통과 여부**. `npm run test` 도 샌드박스에서 실행 거부되었다(직전 회차와 같은 제한). 기준선 수치(vitest 7 files 106 tests 등)는 원장 기록이고 이번 세션에서 재확인하지 못했다.
  - PR #5 가 `origin/auto/2026-10-14-1238` 의 ffd2bf6 **한 커밋뿐**이라는 것은 `git log main..origin/auto/2026-10-14-1238` 로 확인했다. 다만 PR 번호 ↔ 브랜치 대응은 `gh` 를 못 써서 **추정**이다 — 다른 후보는 `auto/2026-10-14-1028`(0804a58)·`auto/2026-10-14-0958`(19bf148). 구현자가 `gh pr view 5` 로 먼저 확인할 것.
