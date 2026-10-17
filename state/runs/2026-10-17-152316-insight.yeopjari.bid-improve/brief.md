- 이번 회차 성격: 자동 배정된 「오류 대응」이지만, 직전 회차의 error 는 `interrupted before PR`(러너가 PR 전에 끊김)이었고 **CI 워크플로가 깨진 것이 아니다.** `.github/workflows/check.yml` 을 읽어 확인했고(ci→check→build→ci-env→migrate→서버→e2e→Chromium→screenshots, 느슨하게 만들 자리 없음), main@601659d 작업 트리는 깨끗하다. 따라서 직전 회차가 끝내지 못한 같은 과제를 **파일 2개로 좁혀** 다시 싣는다. 워크플로 파일은 건드리지 말 것.

- 과제: 지급 목록 CSV 내려받기의 DB 장애·네트워크 실패를 공통 점검 안내로 전달하기 (가치 3 / 위험 1 / 작업량 M)
- 왜: `apps/web/src/api/client.ts:234 download()` 는 `request()` 와 달리 `db_unavailable`(503) 에서 `reportDegraded()` 를 부르지 않고, fetch 자체가 터지면 `ApiError` 가 아닌 날 `TypeError` 를 던진다. 그래서 DB 가 닿지 않는 동안 운영자가 「지급 목록 내려받기」를 누르면 공통 점검 배너가 뜨지 않고 `Admin.tsx:1041 exportList()` 의 `failed(err, '내려받지 못했어요')` 만 떠서, 같은 장애가 다른 화면에서는 안내되고 이 화면에서는 안내되지 않는다.
- 수용 기준:
  1) `download()` 가 503 `db_unavailable` 응답을 받으면 `onServiceDegraded` 리스너가 정확히 1회 호출되고, 기존대로 같은 `ApiError`(status 503 / code `db_unavailable` / 서버 메시지)를 그대로 던진다.
  2) `download()` 가 fetch 단계에서 실패하면 `request()` 와 같은 `ApiError`(status 0, code `network`, 같은 한국어 메시지)를 던진다. `AbortError` 처리를 새로 넣을 필요는 없다(`download` 는 `signal` 을 받지 않는다) — 넣는다면 `request()` 와 동일하게 그대로 다시 던질 것.
  3) 성공 경로는 바뀌지 않는다: 200 + `content-disposition: attachment; filename="…"` 이면 그 파일 이름으로 저장하고, 헤더가 없으면 `fallbackName` 을 쓴다. 응답 헤더 `x-insight-stale: 1` 을 받은 경우도 `request()` 와 같이 `reportDegraded()` 1회(선택 — 2)·1) 만으로도 수용하되, 넣으면 성공 저장은 그대로여야 한다).
  4) 테스트는 손으로 주입한 가짜 의존성이 아니라 **실제 HTTP 응답**으로 위 세 경우를 증명한다: node `http` 서버를 테스트 안에서 띄우고(포트 0), 프로덕션 `api.download` 를 그대로 호출해 리스너 호출 횟수·던진 `ApiError` 의 status/code·저장된 파일 이름을 단정한다.
- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `apps/web/src/api/client.ts:234 download()` — fetch 를 try/catch 로 감싸 `network` `ApiError` 로 바꾸고, `!response.ok` 분기에서 만든 error 의 `code === 'db_unavailable'` 일 때 `reportDegraded()` 를 부른다(`request()` 의 line 207 과 같은 자리·같은 순서: 리스너 먼저, 그다음 throw). 성공 분기의 blob/링크 코드는 건드리지 않는다.
  - `apps/web/src/__tests__/client-download.test.ts` (신규) — 위 수용 기준 4) 의 실제 HTTP 테스트. 현재 `apps/web` 에는 테스트가 하나도 없고, `vitest.config.ts` 의 `include` 는 이미 `apps/**/*.test.ts` 를 잡으므로 설정 변경은 필요 없다.
- 테스트 작성 주의 (vitest `environment: 'node'`):
  - `client.ts` 는 모듈 평가 시점에는 브라우저 전역을 읽지 않지만, 호출 시 `window.location.pathname`(`isAdminSurface`), `document.cookie`(`csrfToken`), `document.createElement`/`document.body`, `URL.createObjectURL`/`revokeObjectURL`, `window.setTimeout` 을 쓴다. 테스트에서 이들을 최소한으로 세우고 **끝나면 원래 값으로 복원**하라(과거 교훈: 브라우저 global 은 복원할 것).
  - `path` 는 `fetch(path)` 에 그대로 넘어가므로 상대 경로가 아니라 테스트 서버의 절대 URL(`http://127.0.0.1:${port}/api/v1/admin/rewards/export`)을 넘기면 node fetch 로 실제 요청이 간다.
  - `Response`/`fetch` 는 Node 22 내장을 쓰고 모킹하지 말 것. 503 응답 본문은 서버가 실제로 내보내는 모양(`{"error":{"code":"db_unavailable","message":"…"},"request_id":"…"}`)으로 쓸 것.
  - `onServiceDegraded` 가 돌려주는 해제 함수를 `finally` 에서 반드시 불러 다른 테스트로 리스너가 새지 않게 할 것.
- 검증 명령:
  - `npm ci --no-audit --fund=false` (최초 1회. 이 worktree 에는 `node_modules` 가 없고, `npm install` 은 lockfile 을 흔들 수 있으니 쓰지 말 것)
  - 실패 재현 → 수정 → 통과: `npm run test -- apps/web/src/__tests__/client-download.test.ts`
  - 전체: `npm run check` (eslint + core/server/web typecheck + vitest 전체), `npm run build`
  - e2e(`node scripts/e2e.mjs`)·`scripts/screenshots.mjs` 는 실 Postgres·Chromium 이 필요하고 이번 변경은 브라우저 측 알림 배선이라 **필수 아님**. 돌릴 경우 반드시 격리 인스턴스를 쓰고 기존 개발 DB/서버를 건드리지 말 것(`scripts/dev-reset.sh`·`scripts/ci-env.mjs` 실행 금지).
- 위험과 피할 것:
  - `request()` 본문·`invalidateAll()` 호출 순서·`api.cached/prefetch` 는 이번 범위가 아니다. 204 캐시 무효화 문제(보류 아이디어)와 묶지 말 것.
  - 서버 쪽(`packages/core/src/db.ts`, `errors.ts:isDatabaseUnreachable`, `http/pipeline.ts`, `routes/admin.ts`)은 **읽기만** 하고 바꾸지 말 것. 차단기·503 분류·감사 정책에 걸린다.
  - `routes/admin.ts` 의 지급 목록 내보내기는 수령 정보 원문을 다루는 경로다. 테스트든 로그든 전화번호·계좌 같은 값을 만들어 넣지 말고(감사 details 에 원문 금지), 테스트 응답 본문은 더미 CSV 한 줄로 충분하다.
  - 새 테스트 프레임워크·jsdom·DOM 라이브러리 의존성을 추가하지 말 것(기존은 vitest node 환경 + Playwright 스크립트뿐이다).
  - 배너 UI(`App.tsx` 의 `.in-degraded`, `AppProvider` 의 30초 health 확인)는 이미 `onServiceDegraded` 를 소비하고 있다. 이번 회차에 UI 를 손보지 말고 알림이 나가는 것까지만 증명하라(브라우저에서 배너가 실제로 뜨는지는 이번 정찰 미확인).
- 실행 순서 (각 단계 끝에서 저장소가 통과 상태여야 한다):
  1) `npm ci --no-audit --fund=false` → 증명: 종료 코드 0, `git status --short` 에 lockfile 변동 없음.
  2) 테스트 먼저. `apps/web/src/__tests__/client-download.test.ts` 를 수정 없는 `download()` 에 대고 쓴다 → 증명: `npm run test -- apps/web/src/__tests__/client-download.test.ts` 가 **db_unavailable 알림 0회 / network 변환 없음**으로 실패하고, 성공 저장 테스트는 이미 통과(회귀 가드). 이 실패 출력을 회차 기록에 남길 것.
  3) `client.ts:234 download()` 최소 수정 → 증명: 같은 명령이 전부 통과.
  4) 수정만 되돌려 2) 의 실패가 그대로 재현되는지 확인하고 복구 → 증명: 되돌린 상태에서 같은 실패, 복구 후 통과.
  5) 전체 검증 → 증명: `npm run check` 와 `npm run build` 통과, `git diff --check` 통과, `git status --short` 에 지정 2개 파일만.
- 범위 밖(명시): 서버 코드·워크플로 파일·배너 UI·`request()`/`download()` 통합·204 캐시 무효화·e2e/스크린샷 스크립트 수정·새 의존성.
- 작업량 근거: 비슷한 과거 회차(2026-10-08 두 건, 프로덕션 1개 + 테스트 2개, 단위+실HTTP 검증)를 기준으로 한 유추 추정이며 **한 세션(45분) 안 80% 신뢰**로 본다. 가장 불확실한 조각은 node 환경에서 브라우저 전역(document/URL/window)을 세우는 첫 web 테스트 부트스트랩이고, 이것이 25분을 넘기면 성공 저장 테스트를 빼고 수용 기준 1)·2) 만으로 줄여 마감할 것(여유분은 이 한 가지에만 쓴다). `npm ci` 가 느릴 수 있어 1) 에 별도로 5~10분을 잡아 둘 것.

- 차선 후보: `apps/web/src/api/client.ts:170` 의 204 조기 반환이 쓰기 성공 시 `invalidateAll()` 보다 앞서는 문제 — 단, 현재 `api.cached`/`prefetch` 를 쓰는 화면 호출처가 검색되지 않아 사용자 피해가 미확인이므로, 먼저 실제 소비 경로를 찾고 없으면 이 후보도 접고 `util/text.ts:isSafeLink()` 죽은 코드·주석 정리(가치 3 / 위험 2 / S)로 내려갈 것.
