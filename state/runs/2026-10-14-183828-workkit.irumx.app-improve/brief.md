# 과제서 (2026-10-14 정찰, 수정 과제 회차)

- 과제: PR #3 의 `check` 잡이 실패한 원인을 **잡 주석으로 먼저 판정**하고, 저장소 안의 원인일 때만 그 한 곳을 좁게 고치기 (가치 4 / 위험 2 / 작업량 S)
- 왜: 러너가 "릴리즈 워크플로가 같은 이유로 두 번 실패" 로 이 회차를 수정 과제로 지정했는데, 직전 회차(10:52 수리)가 **인증된 GitHub API 로 잡 주석을 읽어** 원인을 저장소 밖으로 특정해 두었다: "The job was not started because recent account payments have failed or your spending limit needs to be increased." (`steps: []`, 소요 2초). 이 세션에서는 `gh` 호출이 권한 거부되어 **지금도 같은 주석인지는 미확인**이다 — 그래서 구현자가 가장 먼저 할 일은 추측으로 워크플로를 고치는 것이 아니라 주석을 한 번 읽어 분기하는 것이다. 같은 전제(원인이 워크플로/서비스 컨테이너)로 쓴 과제서가 직전 회차에 기각됐다.
- 수용 기준:
  1) PR #3(브랜치 `auto/2026-10-14-1508`, 커밋 46b7130) 의 최신 `check` 실행에 대해 **원인 문장을 하나 인용**해 회차 기록에 남긴다. 명령: `gh run list --branch auto/2026-10-14-1508 --limit 5` → `gh api repos/hkjang/workkit.irumx.app/actions/runs/<id>/jobs` (`steps` 배열과 주석 메시지를 본다). `gh run view <id> --log-failed` 로 보강.
  2) 주석이 결제·지출 한도 차단(= `steps` 가 빈 배열, 소요 수 초)이면 **`.github/workflows/check.yml` 을 한 줄도 바꾸지 않는다**. 잡이 시작조차 하지 않으므로 워크플로 수정은 효과가 없고, 검사를 빼거나 조건을 느슨하게 만드는 것은 금지다. 대신 ③ 을 하고 아래 **차선 후보**를 이번 회차의 코드 변경으로 삼는다.
  3) 로컬에서 check.yml 의 "DB 없이 도는 구간" 을 그 순서대로 재현해 green 을 기록한다(실제 명령은 아래 검증 명령). PostgreSQL·Playwright 가 필요한 뒤쪽 단계(`Migrate and seed` 이후)는 환경이 없으면 "미실행" 으로 적고 green 이라고 쓰지 않는다.
  4) 주석이 **실제 단계 실패**(`steps` 에 `conclusion: failure` 인 단계가 있음)로 나오면, 그 단계의 스크립트·테스트만 읽고 그 한 곳을 고친다. 그 단계를 로컬에서 같은 명령으로 재현해 고치기 전 red → 고친 뒤 green 을 보여야 한다.
- 건드릴 파일(차단 확인 시 — 즉 차선 후보를 수행):
  - `packages/tools/src/table/csv.ts:84 push()` — 한 레코드가 끝없이 자라는 것을 막는 가드 한 줄 추가(아래 차선 후보 상세).
  - `packages/tools/src/__tests__/robustness.test.ts` — 회귀 시험 1개 추가.
  - (실제 단계 실패로 나왔다면) 그 단계가 부르는 `scripts/*.mjs` 또는 해당 테스트 파일 **하나만**. 워크플로 YAML 은 피할 것.
- 검증 명령 (이 저장소에서 실제로 도는 것, 이 순서로):
  - `npm ci --no-audit --fund=false` — 자동 개선 워크트리에는 `node_modules` 가 없다(이 세션에서 `ls node_modules` 로 확인). 안 하면 `vitest: not found`.
  - `npx vitest run robustness.test.ts compare.test.ts`
  - `npm run check` (= `eslint .` + 4개 패키지 typecheck + `vitest run`; 직전 회차 기준 10 파일 124 passed)
  - `npm run build`
  - `npm run kits && cp dist-kits/manifest.json /tmp/m1.json && npm run kits && cmp /tmp/m1.json dist-kits/manifest.json && npm run kits:test`
  - `node scripts/ci-env.mjs`
  - 뒤쪽 단계(`node scripts/migrate.mjs`·`scripts/e2e.mjs`·`tools-flows.mjs`·`ui-flows.mjs`·`a11y.mjs`)는 Docker PostgreSQL 16(:55447)·가짜 옆자리(:18997)·가짜 토스(:18998)·Playwright Chromium 이 떠 있어야 돈다. 준비 못 하면 "미실행" 으로 적는다.
- 위험과 피할 것:
  - **워크플로를 느슨하게 만들지 말 것**(검사 제거·`continue-on-error`·`if:` 완화 전부 금지, 운영자 반복 지시). 서비스 컨테이너 다이제스트 고정이나 `docker run` 이관 같은 산탄총 수정도 직전 회차에서 원인과 무관하다고 판정됐다.
  - 결제 차단은 저장소에서 고칠 수 없다. "고칠 것이 없다" 는 결론도 정당한 결과다 — 억지로 코드를 바꾸지 말고 ③ + 차선 후보로 회차를 채울 것.
  - `csv.ts` 외의 다른 파서(`xlsx-read.ts`, `apps/web/src/tools/compare.worker.ts`)는 건드리지 말 것 — 계약이 다르다(운영자 지시).
  - 보호 경로: `packages/core/src/auth/*`, `domain/payments.ts`·`refunds.ts`, `db/` 마이그레이션 — 이번 과제와 무관하므로 열지 말 것.
  - PR #3 은 **main 기준**이라 PR #2(행 한도를 `push()` 직후로 올린 변경)를 포함하지 않는다. 이 워크트리(main@53ea324)의 `csv.ts:93` 은 아직 행 한도를 내용 분기 안에서 본다 — 직접 확인함.

## 차선 후보 (차단 확인 시 이번 회차의 코드 변경 = 1순위로 올려서 할 것)

- 과제: 개행 없이 구분자만 든 CSV 가 한 레코드를 1,000만 항목까지 키운 뒤에야 한도를 보는 것 막기 (가치 3 / 위험 2 / 작업량 S)
- 지금 코드(확인함): `parseCsv`(`packages/tools/src/table/csv.ts:77`)의 한도 검사는 전부 `end()`(89행) 안에 있고, `end()` 는 `'\n'`/`'\r'`(116~119행)이나 루프 뒤 122행에서만 불린다. 10MB 가 전부 쉼표인 파일은 `push()`(84행)만 약 1,000만 번 돌아 `record` 배열이 1,000만 항목이 된 뒤 122행에서야 `too_many_cells` 로 던진다. 오류 코드는 맞고 **자원이 문제**다(브라우저 워커에서 수백 MB 할당). `LIMITS.cells` = 500,000, `LIMITS.fileBytes` = 10MB (`packages/tools/src/table/types.ts:41~53`).
- 권장 구현: `push()` 안에서 `record.push(field)` **전에** 레코드 폭을 보고 같은 코드로 던진다 —
  `if (record.length >= LIMITS.cells) throw new TableError('too_many_cells', ...)` (문구는 94행의 기존 메시지를 그대로 재사용).
  이유: 한 레코드가 셀 한도보다 넓으면 패딩 뒤 격자는 무조건 한도를 넘으므로 새 가드는 **기존 검사보다 항상 같거나 더 엄격하지 않다** — 한 행뿐이어도 `width > LIMITS.cells` 라 `end()` 가 어차피 던질 입력만 더 일찍 잡는다. 정상 파일(열 수백 개)은 영향 없음.
- 수용 기준: 1) `','.repeat(2_000_000)` 같은 입력이 `code: 'too_many_cells'` 로 던진다 2) 던지기 전 `record` 가 `LIMITS.cells` 보다 커지지 않는다(시험에서 폭을 직접 세기 어렵다면 입력 크기를 키워도 시간이 선형에 머무는 것으로 갈음) 3) 기존 단언 유지: `robustness.test.ts` 의 행 한도/셀 한도/`empty` 시험, `compare.test.ts` 의 CSV 읽기·구분자 감지(43~85행).
- 검증 명령: `npx vitest run robustness.test.ts compare.test.ts` → `npm run check`.
- 주의: 시험 입력을 1,000만 항목으로 만들면 **고치기 전** 실행이 느리거나 힙을 먹는다. red 를 보일 때는 200만 항목 정도로 줄이고, `expect(...).toThrow(objectContaining({ code: 'too_many_cells' }))` 는 고치기 전에도 통과할 수 있으므로 red 의 근거는 "가드 추가 전/후 시간·메모리" 보다 **가드가 없으면 `push()` 가 한도 초과 폭까지 쌓인다는 것을 직접 단언**하는 쪽이 낫다(예: 폭이 한도를 넘는 한 줄 입력이 `too_many_cells` 로 던지는지 — 이 단언은 지금도 통과하므로, 새 시험은 "한도 초과 입력이 한도+1 항목을 넘기지 않고 멈춘다" 를 볼 수 있는 형태로 쓸 것. 쓸 수 없다고 판단하면 시험 없이 가드만 넣지 말고 구현자 판단으로 과제를 3순위로 교체).
- 3순위: `detectDelimiter`(`csv.ts:41`)가 10MB 텍스트 전체를 `split('\n')` 한 뒤 앞 30줄만 쓴다 — 앞 64KB 만 잘라 쪼개고 마지막 잘린 줄은 버린다(가치 2 / 위험 1 / 작업량 S). `compare.test.ts` 의 구분자 감지 단언 3개가 그대로 통과해야 한다.
