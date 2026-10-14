# 과제서 — 2026-10-14-212848-yeopjari-your-improve (수정 과제)

- 과제: PR #4 의 CI 실패를 규명하고(원인은 저장소 밖) 네 커밋을 로컬 전체 게이트로 재검증해 다시 출하한다 (가치 4 / 위험 1 / 작업량 S)

- 왜: PR #4(`auto/2026-10-14-1558`, head `8538b01`)의 `check` 잡은 **2초 만에** `failure` 로 끝났고(`started_at 2026-10-14T07:24:11Z` → `completed_at 07:24:13Z`, `annotations_count: 2`, 러너가 남긴 `ci-failure-ea6beee87790.txt` 의 `실패 단계:` 가 **빈 칸**) — 어떤 단계에도 들어가지 못했다는 뜻이다. `npm ci` 조차 2초에 끝나지 않으므로 워크플로·테스트·코드가 범인일 수 없다. 같은 계정의 다른 저장소(`biz.irumx.app`, `ci-3a84b166d7e9.json`)도 **같은 날 11:48:50Z → 11:48:53Z, 3초 failure** 로 같은 모양을 보이므로 장애는 GitHub Actions 계정 수준(한도 소진/과금 차단 계열)이고 이 저장소에서 고칠 수 없다. 따라서 이번 회차의 실제 일은 **PR #4 가 담은 가치(한도 429 `Retry-After` 수정 + e2e 가짜 서버 주소 충돌 제거)를 잃지 않게 재출하하고, CI 가 못 하는 검증을 로컬에서 전부 재현해 증거를 남기는 것**이다.

- 수용 기준:
  1) 현재 브랜치(`auto/2026-10-14-2128`, 베이스 `main@735609c`)에 PR #4 의 네 커밋이 **순서대로** 올라간다:
     `5f6732c`(한도 초과 응답의 Retry-After 는 다음 한국 자정까지) → `e564e42`(검사 상한을 업로드 전 시각에서 잡아 초 경계 거짓 실패 제거) → `ea6beee`(가짜 서버 주소를 서버에 알려 준 값에서 읽는다) → `8538b01`(가짜 서버 주소는 포트 있는 루프백만 받는다).
     `git cherry-pick 5f6732c e564e42 ea6beee 8538b01` 로 충돌 없이 올라가야 한다(베이스가 PR #4 와 같은 `735609c` 이므로 충돌이 나면 전제가 틀린 것이니 멈추고 원인을 적을 것).
  2) 로컬에서 CI `check.yml` 과 **같은 순서의 게이트**가 전부 통과한다: `npm run check` → `npm run build` → `node scripts/ci-env.mjs` → 마이그레이션 → 서버 기동 → `node scripts/e2e.mjs` 가 **0 failed, exit 0**.
  3) 수정의 인과를 증명한다: `packages/core/src/quota.ts` 의 한국 자정 계산을 되돌려 넣으면 단위 테스트(`ends a daily window at the next Korean midnight`)와 e2e 검사(`하루 한도 429의 Retry-After 는 다음 한국 자정까지`)가 **다시 실패**하고, 되돌림을 풀면 다시 통과한다. 되돌린 코드는 커밋하지 말 것.
  4) 원장(ledger)에 **수정 과제**로 적고, 위 '왜' 의 수치(2초/3초, 빈 실패 단계, 두 저장소)를 근거로 **CI 초록은 이번 회차에도 달성 불가**라는 점과 운영자(hkjang)가 GitHub 과금·스펜딩 한도를 확인해야 한다는 점을 명시한다.

- 건드릴 파일 (프로덕션 파일은 cherry-pick 이 가져오는 2개뿐):
  - `packages/core/src/quota.ts:retryAfterSeconds` — 한국 벽시계(`kstDayStart`/`kstParts`)로 다음 창 시작을 계산. `your_priv.quota_counters` 의 PK 인 `windowStart` 키는 **그대로 둘 것**(옮기면 운영 DB 의 모든 카운터가 한 번에 끊긴다).
  - `scripts/e2e.mjs` — 가짜 서버 주소를 `YOUR_YEOPJARI_URL`(= `scripts/ci-env.mjs` 가 쓰는 변수)에서, 푸시 쪽은 `E2E_PUSH_ORIGIN` 에서 읽는다. 기본값은 옛 하드코딩(`http://127.0.0.1:18998`)과 바이트 동일해야 한다 — 둘 다 미설정으로 돌려 확인할 것.
  - `packages/core/src/__tests__/rules.test.ts` 와 e2e 검사 두 줄 — cherry-pick 이 함께 가져온다.
  - **`.github/workflows/check.yml` 은 한 줄도 건드리지 말 것.** 느슨하게 만드는 것은 금지이고, 조이는 변경조차 지금은 Actions 가 뜨지 않아 검증할 수단이 없다(`git diff main --stat` 에 이 파일이 없어야 한다).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  ```
  npm run check          # lint + typecheck(core→server→web) + vitest run — 단위 53개 이상
  npm run build          # web + server
  bash scripts/dev-reset.sh      # Docker postgres:16-alpine 을 127.0.0.1:55440 로 띄우고 마이그레이션 + :8788 서버
  TZ=UTC node scripts/e2e.mjs    # 실제 HTTP 배선 164검사 — 0 failed / exit 0 이어야 함
  ```
  - 가짜 서버 포트 18998·18997 을 다른 프로젝트의 고아 프로세스가 쥐고 있는 일이 네 회차 연속 있었다. `ss -ltnp | grep 1899` 로 먼저 확인하고, 점유 중이면 이번 수정이 넣는 `YOUR_YEOPJARI_URL=http://127.0.0.1:18996` / `E2E_PUSH_ORIGIN=http://127.0.0.1:18995` 로 **환경변수만 바꿔** 돌릴 것(코드 기본값은 바꾸지 말 것). 이것이 `ea6beee` 가 존재하는 이유다.
  - CI 상태 확인은 하지 말 것(`gh` 가 이 세션에서 막혀 있고, 결과는 위 진단대로 2~3초 failure 다).

- 위험과 피할 것:
  - **CI 를 초록으로 만들려고 워크플로·테스트를 완화하지 말 것.** 원인이 저장소 밖이므로 어떤 완화도 증상만 가리고 게이트를 영구히 약화시킨다.
  - `db/migrations/**`(전진 전용) · `packages/core/src/auth/**`(세션·MFA) · `packages/core/src/vendor/postgres`(패치 사본) · RLS 정책(`your_priv.can_see`) 은 이번 과제와 무관하다 — 열지 말 것.
  - `your_priv.error_alerts.signature` 문자열 모양에 기대는 e2e 검사가 있다(`scripts/e2e.mjs` 의 `signature LIKE 'CLIENT /chat/:id%'`). 서명 조립을 건드리면 운영 DB 의 억제 상태가 풀린다 — 이번 과제 범위 밖.
  - 과거 교훈: 손으로 주입한 의존성이나 소스 문자열 검사로 증명하지 말 것. 이 저장소에서 배선을 증명하는 유일한 수단은 `scripts/e2e.mjs` 다.

- 차선 후보: **`v.int` 가 `'0x10'`·`'1e3'` 같은 비10진 문자열을 정수로 받아들인다** (가치 3 / 위험 1 / 작업량 S — 1순위의 cherry-pick 이 충돌해 성립하지 않을 때).
  `packages/core/src/util/validate.ts:100` 에서 직접 확인했다: `const n = typeof value === 'string' && value.trim() !== '' ? Number(value) : value` — `Number('0x10')`→16, `Number('1e3')`→1000, `Number('  12  ')`→12 이므로 사용자가 보낸 `"0x10"` 이 조용히 16 이 된다. 좁혀 고치는 방향: 10진 정수 모양(`/^-?\d+$/`)만 받고 나머지는 기존 메시지(`${eun(label)} 정수여야 합니다.`)로 거절. 호출부(쿼리스트링 `limit`·`page` 류)가 공백 포함 값을 보내는지 먼저 확인해 `trim()` 허용은 유지할 것. 단위 테스트는 `packages/core/src/__tests__/rules.test.ts` 에 추가하고, **고치기 전에 실패하는 것을 먼저 보일 것**.
