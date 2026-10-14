# 과제서 — 2026-10-14-150835-workkit.irumx.app-improve (수정 과제)

- 과제: PR #2 의 `check` 잡이 **2초 만에** 실패한 원인(잡 시작/서비스 컨테이너 단계)을 특정해 고치기 (가치 5 / 위험 3 / 작업량 M)

- 왜: 직전 회차(auto/2026-10-14-0958, PR #2)의 변경은 `packages/tools/src/table/csv.ts` 와 `robustness.test.ts` 두 파일뿐이고 `npm run check` 는 로컬에서 통과했는데(원장·비평 노트 123 passed), CI 의 `check` 잡은 **실패했다**. 이 때문에 릴리즈가 막혀 있다.
- **핵심 증거(내가 직접 본 것)**: `/mnt/c/Users/USER/projects/aidev/state/runs/2026-10-14-095843-workkit.irumx.app-improve/ci-ebf4f431dc1e.json` 의 체크런 115625901789 — `"started_at": "2026-10-14T01:09:04Z"`, `"completed_at": "2026-10-14T01:09:06Z"`, `"conclusion": "failure"`, `"output.annotations_count": 2`, `output.title/summary/text` 는 모두 `null`.
  **2초**는 `actions/checkout` 조차 끝나지 않는 시간이다. 즉 실패는 `npm ci`·`npm run check`·`build`·`kits`·`e2e`·Playwright 같은 **스텝 안이 아니라 잡 셋업(서비스 컨테이너 기동 / 워크플로 평가)** 에서 났다. 같은 워크플로 파일로 09:27 에 PR #1 은 통과했다(`.../2026-10-14-090819-.../journal.md:28` "ci passed"). 워크플로 파일은 그 뒤 바뀌지 않았다(`git log .github/workflows/check.yml`).
  → **csv.ts 변경은 원인이 아닐 가능성이 매우 높다.** 코드를 고쳐도 CI 는 다시 같은 2초 실패를 낼 수 있다.

## 0) 먼저 할 일 — 애너테이션 2개를 읽어 원인을 확정 (추측 금지)
정찰 세션에서는 `gh` 호출과 `npm ci` 가 권한으로 거부되어 **실행하지 못했다**(미확인). 구현자는 먼저:

```
gh api repos/hkjang/workkit.irumx.app/check-runs/115625901789/annotations
gh run view 38494124742 --log-failed        # 또는
gh api repos/hkjang/workkit.irumx.app/actions/runs/38494124742/jobs --jq '.jobs[].steps[]|{name,conclusion}'
```
이 출력이 **이번 과제의 원인 진단서**다. 실패 단계 이름과 애너테이션 문구를 journal 구현 노트에 **그대로 붙여** 두세요(다음 회차가 또 이 추적을 반복하지 않게).

진단 결과에 따라 아래 분기 중 하나만 하세요.

### 분기 A — 애너테이션이 서비스 컨테이너/잡 셋업 실패 (가장 유력)
예: `Failed to initialize container postgres:16-alpine`, `Error response from daemon ... toomanyrequests`, `Error: Service container postgres failed`.
- 고칠 것: `.github/workflows/check.yml` 의 `services.postgres` 가 잡 전체를 죽이는 구조. **게이트를 느슨하게 만들지 말 것** — 검사 항목은 하나도 빼지 않는다.
- 허용되는 방향(둘 중 하나, 더 작은 것):
  1. `image: postgres:16-alpine` 을 **다이제스트 고정**(`postgres:16-alpine@sha256:…`)하거나, Docker Hub 레이트리밋을 피하는 레지스트리로 바꾼다. 같은 PostgreSQL 16 이어야 한다(다운그레이드 금지).
  2. 서비스 컨테이너 대신 **스텝 안에서 `docker run` 으로 PostgreSQL 16 을 띄운다** — `Migrate and seed` 스텝 바로 앞에, 같은 포트 `55447:5432`·같은 비밀번호·`pg_isready` 대기 루프로. 이러면 DB 가 필요 없는 `check`/`build`/`kits` 스텝이 DB 기동 실패로 같이 죽지 않는다.
- 어느 쪽이든 **검사 스텝 목록·순서·`npm run check` 내용은 그대로 유지**한다.

### 분기 B — 애너테이션이 워크플로 파일 평가 오류
예: `Invalid workflow file ... .github/workflows/check.yml#L…`. 그 줄을 고친다. (단, 09:27 에 같은 파일로 통과했으므로 가능성은 낮다.)

### 분기 C — 애너테이션이 실제 스텝 실패를 가리킨다(2초 기록이 오해였을 때)
그 스텝의 스크립트를 읽고 **원인**을 고친다. 그 경우 아래 "차선 후보"(csv.ts 경계 결함)가 유력한 후보다 — 로컬에서 `npm ci && npm run check` 로 재현하고, 재현되지 않으면 분기 A 로 돌아간다.

## 수용 기준
1. `gh api .../check-runs/115625901789/annotations` 의 실제 문구가 journal 구현 노트에 인용되어 있고, 고친 원인이 그 문구와 맞는다.
2. 수정 후 PR 의 `check` 잡이 **스텝까지 들어가서** 끝난다(2초 조기 실패가 사라진다). 통과하면 끝, 다른 이유로 실패하면 그 이유가 journal 에 적힌다.
3. 게이트가 느슨해지지 않았다: `.github/workflows/check.yml` 의 diff 에 `npm run check`·`build`·`kits`(2회 빌드 + `cmp` manifest)·`kits:test`·`ci-env`·`migrate`+`seed`·`e2e`·`tools-flows`·`ui-flows`·`a11y` 스텝이 모두 남아 있고, `continue-on-error`·`|| true`·`if: false`·테스트 건너뛰기·PostgreSQL 버전 다운그레이드가 **없다**.
4. 로컬에서 재현 가능한 범위를 실제로 돌려 통과를 확인한다(아래 검증 명령).

## 건드릴 파일 (최대 2개)
- `.github/workflows/check.yml` — `services.postgres` 블록(또는 애너테이션이 가리키는 줄). 분기 A-2 를 고르면 `Migrate and seed` 앞에 기동 스텝 1개 추가.
- (분기 C 일 때만) `packages/tools/src/table/csv.ts:end()` + `packages/tools/src/__tests__/robustness.test.ts`.
- 그 외 프로덕션 파일은 건드리지 말 것.

## 검증 명령 (이 저장소에서 실제로 도는 것)
```
npm ci                       # 이 워크트리에는 node_modules 가 없다. 먼저 해야 vitest 가 돈다
npm run check                # lint + typecheck 4개 패키지 + vitest (DB·Playwright 불필요, 1차 관문)
npm run build
npm run kits && cp dist-kits/manifest.json /tmp/m1.json && npm run kits && cmp /tmp/m1.json dist-kits/manifest.json
npm run kits:test
node scripts/ci-env.mjs
# 워크플로 YAML 을 고쳤으면 문법 확인:
node -e "import('node:fs').then(f=>console.log(f.readFileSync('.github/workflows/check.yml','utf8').length))"   # 최소 확인
# DB·브라우저 스텝(npm run e2e / tools / ui / a11y)은 Docker PostgreSQL 16 + Playwright + 가짜 서버(:18997,:18998,:8797)가 필요하다.
# 분기 A-2 를 골랐다면 그 docker run 명령만은 로컬에서 그대로 한 번 띄워 pg_isready 까지 확인할 것.
```

## 위험과 피할 것
- **워크플로는 보호 경로다.** 검사를 빼거나 느슨하게 해서 통과시키는 것은 금지(운영자 반복 지시). 한 스텝이라도 지우면 이 회차는 기각된다.
- `db/` 마이그레이션과 로그인 역할은 손대지 말 것 — 공유 Supabase 를 쓰고 서비스 역할이 NOLOGIN 이다(커밋 8abcd8a·63dee8d).
- 같은 종류의 CSV 파서가 `table/csv.ts`·`table/xlsx-read.ts`·`apps/web/.../compare.worker.ts` 셋에 있다 — **통합하지 말고** 한쪽만 좁게.
- PR #2 의 csv.ts 변경 자체를 되돌리지 말 것(결함 수정은 타당하다). 분기 C 가 아니면 csv.ts 를 아예 열지 말 것.
- 2초 실패를 "플레이크" 로 단정하고 재실행만 요청하는 것도 금지 — 같은 이유로 두 번 실패했다.

## 차선 후보 (1순위가 성립하지 않을 때 = 애너테이션이 단위 시험을 가리킬 때)
**PR #2 의 행 한도 검사가 파일 끝 빈 줄 때문에 정상 파일을 거부한다 (가치 3 / 위험 2 / 작업량 S)**
PR #2 는 `end()` 에서 `if (rows.length + 1 > LIMITS.rows) throw` 를 `push()` 직후로 올려 빈 줄 분기와 공유하게 했다. 그래서 **내용 행이 정확히 `LIMITS.rows`(50,000)개이고 파일이 빈 줄로 끝나는**(`…\n\n`) CSV 가 이제 `too_many_rows` 로 거부된다 — 고치기 전에는 그 마지막 빈 행이 `csv.ts:126` 의 `rows.filter(...)` 에서 걸러져 정상 처리됐다. 직전 회차 비평 노트도 같은 것을 "우려 ①"(`.../2026-10-14-095843-.../journal.md:24`)로 남겼다. 고칠 방향: 파일 끝의 빈 레코드는 사용자에게 보이지 않으므로 한도 초과 사유가 되지 않게 한다(`end(last)` 플래그로 마지막 빈 레코드만 예외, CRLF 에서 `i` 가 이미 증가한 것 주의). 시험: 내용 50,000행 + 끝 빈 줄 1개가 통과하고, `'\n'.repeat(LIMITS.rows + 2)` 는 계속 `too_many_rows` 로 던지는 것 둘 다. 검증은 `npx vitest run robustness.test.ts compare.test.ts` + `npm run check`.
