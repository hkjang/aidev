# PR 처리기 노트 2026-10-14-235738-law.irumx.app-shepherd — law.irumx.app PR #7
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-14-221836-law.irumx.app-improve)
# 회차 노트 2026-10-14-221836-law.irumx.app-improve — law.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 22:18] base pinned — main@c62b3a0
- [러너 22:18] autonomy release — 

## 정찰 노트
- 우선 과제(PR #6 CI 실패)는 다섯 회차 연속 같은 신호다 — 잡 3초·빈 로그에 보존된 체크런 주석("account payments have failed or your spending limit…")이 일치하고, 175134 수리 회차가 CI 순서를 로컬에서 끝까지 돌려(e2e 27/27·acceptance 26/26) 코드 무결을 증명했다. `.github/workflows/check.yml` 을 직접 읽어 3초·빈 로그를 만들 저장소 쪽 원인이 없음을 재확인했으므로(첫 실행 step 이 apt-get 설치라 어느 step 이든 들어갔으면 3초를 넘는다) 전수 재현을 네 번째로 돌리는 대신 **축소 확인 + 차선 구현**으로 과제서를 짰다.
- 차선은 `env.ts loadEnv` 특성화 테스트로 골랐다 — 프로덕션 0파일, DB 불필요, 미머지 PR 3개와 파일 충돌 없음. AT-26 양성 대조 조이기(3/3/S)는 CI 결제가 막힌 지금 검증이 불가능해 제쳤고, Background requestId 는 미머지 background-redact.test.ts 와 충돌해 제쳤다.
- 추측으로 적은 것: 기준선 숫자(6 files 103 tests, lint 646 warnings)는 원장 인용이다. 이 세션은 샌드박스 승인 제한으로 `npx vitest run` 과 `gh` 를 **실행하지 못했다** — 구현자가 손대기 전에 `npm run test` 로 실제 기준선을 먼저 적어야 한다.
- 구현자가 조심할 것: `env.ts` 는 던지면 워커가 기동 못 하는 자리다 — :116 의 `new URL()` 이 `TypeError` 로 터지는 것은 **기록만** 하고 고치지 말 것(별도 아이디어로 남겼다). 기존 `env-mock-providers.test.ts` 는 수정 금지, 헬퍼는 복사해 쓸 것.
- [러너 22:23] scout done — 수정 과제 — PR #6 CI 실패를 **축소 경로로 확정**하고(전수 재현 금지), 그 뒤 `env.ts loadEnv` 전반 특성화 테�

## 구현 노트
- 우선 과제(PR #6 CI)는 축소 경로로 확정: `gh` 미인증이라 주석을 직접 못 받았지만 `check.yml` 첫 `run` step 이 `apt-get install postgresql-client-17` 이라 3초·빈 로그를 만들 저장소 쪽 원인이 없다 → 저장소 밖(Actions 결제). 고친 코드 없음, 워크플로도 안 건드렸다.
- 본 작업은 신규 `packages/core/src/__tests__/env-loading.test.ts` 104건뿐 — **프로덕션 0파일**, `env.ts` 는 읽기만. `npm run check` 통과(lint 0 errors / 646 warnings = 기준선, 신규 파일 경고 0 / typecheck 3 / vitest 7 files 207 tests, 이 세션에서 측정한 기준선 6 files 103).
- **과제서가 틀린 곳(비평가가 먼저 볼 자리)**: 과제서는 dev 파생 키 세 개가 "서로 다름" 이라 했으나 **바이트까지 동일**하다 — 시드 접두사가 45자인데 `:212` 루프가 0..31 만 읽어 `purpose` 가 출력에 닿지 못한다. 실제 동작(동일함)을 기록하고 주석에 이유를 달았다. dev 한정이고 운영은 키 없으면 기동 거부라 취약점은 아니지만, "purpose-separated keys" 라는 타입 주석은 dev 에서 거짓이다 — 아이디어로 남겼다.
- 확신 없는 것: 돌연변이 20건 중 **1건이 살아남았다** — `:119` 의 `parsed.hostname.toLowerCase()`. URL 파서가 이미 호스트를 소문자화하므로 중복 호출이고, 내 테스트는 그 줄이 아니라 파서를 재고 있다. 제거하면 동작 변경이라 그냥 주석에 적었다.
- 일부러 안 한 것: `:116` `new URL` 을 `EnvError` 로 감싸기(동작 변경, 범위 밖 — 현재 `TypeError` 동작을 테스트로 고정만 해 뒀다), `env.ts` 의 어떤 조건도 넓히거나 좁히지 않음, `env-mock-providers.test.ts` 미수정(헬퍼는 복사).
- 다음 역할이 조심할 것: 이 테스트는 **DB·서버 불필요**(`npx vitest run packages/core/src/__tests__/env-loading.test.ts`, 0.2초). `e2e`/`acceptance`/`test:rls` 는 돌리지 않았다 — 이 변경과 무관하다. 파일명은 미머지 PR 3개와 충돌을 피하려 고른 것이니 바꾸지 말 것.
- [러너 22:31] brief accepted — 채택 — 수용 기준 1)의 축소 경로 조건이 그대로 성립했고(`gh` 미인증, 워크플로에 3초·빈 로그를 만들 원인 없음), 2)~5)�
- [러너 22:31] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인한 것: diff 는 `packages/core/src/__tests__/env-loading.test.ts` 신규 1파일(427줄)뿐, 프로덕션 0파일. 직접 돌려 전부 통과 — 신규 104건, 전체 `vitest run` 7 files **207 tests pass**, `typecheck` 3개 모두 깨끗, `lint` **0 errors / 646 warnings**(기준선 유지, 207-104=103 으로 구현 노트의 기준선과도 일치).
- 구현자가 의심한 두 자리를 코드로 검증: ① dev 키 3종 동일 주장은 **사실** — 시드 접두사 `law-development-key-do-not-use-in-production:` 가 정확히 45자이고 `env.ts:212` 루프가 `i<32` 라 `i % seed.length === i` 로 `purpose` 가 출력에 닿지 못한다(운영은 키 없으면 `:206` 에서 기동 거부이므로 공격 경로 없음). ② `:119 toLowerCase()` 는 WHATWG URL 이 ASCII 호스트를 이미 소문자화하므로 실제로 중복이며, 제거해도 동작 불변 — 테스트가 못 잡는 것이 맞고 손대지 않은 판단도 맞다. 주석의 사실 주장(`??` 가 빈 문자열을 통과시켜 별칭을 가린다, `fromHex` 가 trim 한다, `cnameTarget` 만 소문자화 안 됨, scheme 이 화이트리스트가 아님)도 전부 코드와 일치.
- 승인이어도 남는 우려: 특성화 테스트가 **개선이면 깨지는 기대 두 개**를 고정했다 — `:157`(dev 키 purpose 미분리)과 `:290`(`new URL` 미래핑 → `EnvError` 아닌 `TypeError`). 둘 다 "이 기대를 뒤집으라" 는 주석이 달려 있으니, 나중에 `env.ts` 를 고치는 회차는 테스트를 버그로 오해하지 말고 주석을 먼저 읽을 것. `env.ts:5` 의 "purpose-separated keys" 문구는 dev 에서 거짓인 채로 남아 있다(수정 범위 밖).
- 못 본 것: `e2e`/`acceptance`/`test:rls`/`ui-flows`(DB·브라우저 필요) — 테스트 전용 변경이라 무관. PR #6 CI 실패의 결제 원인도 독립 확인 못 함(이 세션도 `gh` 미인증) — 다만 저장소를 건드리지 않은 선택은 옳다. `prodRaw` 헬퍼가 `env-mock-providers.test.ts` 와 중복 복사된 상태로 남는다(미머지 PR 충돌 회피 의도, 드리프트 위험은 사소).
- 릴리즈 노트용: 사용자에게 보이는 변화 없음 — 테스트 커버리지만 103 → 207 건.
- [러너 22:34] review approved — 리뷰 승인 (risk=low)
- [러너 22:34] pr created — https://github.com/hkjang/law.irumx.app/pull/7
- [러너 22:35] ci failed — 성공이 아닌 검사: check=failure · 실패한 검사: ? 잡: check 

## 수리 노트
- 틀린 지적: "PR #7 의 CI check 실패" 는 이 변경의 결함이 아니다 — 깨끗한 워크트리에 `npm ci` 후 CI 와 같은 명령을 돌려 전부 통과(`npm run check`: lint 0 errors/646 warnings, typecheck 3개 깨끗, 207/207 tests; `npm run build` 성공, 트리 clean). 재현되지 않는다.
- 맞는 판단은 앞선 회차 쪽: 잡 3초·빈 로그는 `check.yml` 첫 `run` step(apt-get 설치)조차 들어가지 못했다는 뜻이고, diff 는 테스트 파일 1개 추가(프로덕션 0파일)라 러너 할당에 영향을 줄 수 없다 → 저장소 밖(Actions 결제/한도).
- 고친 것 없음 — **커밋 없음**. 워크플로 수정·단언 완화·테스트 삭제는 전부 금지 사항이라 손대지 않았다. 근거는 `fix-summary.md`.
- 확신 없는 곳: 결제 원인을 GitHub 에서 직접 못 봤다(`gh` 미인증 + 비공개 저장소라 비인증 API 404) — 전부 간접 증거다. `e2e`/`acceptance`/`test:rls` 는 미실행(DB 필요, 이 변경과 무관).
- 다음 역할에게: 비평가가 지적한 "개선이면 깨지는 기대 2건"(`:157` dev 키 purpose 미분리, `:290` `new URL` 미래핑)은 그대로 남겨 뒀다 — 의도된 특성화이고 주석에 뒤집는 조건이 적혀 있다.
