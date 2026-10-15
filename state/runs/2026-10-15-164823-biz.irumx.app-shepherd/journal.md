# PR 처리기 노트 2026-10-15-164823-biz.irumx.app-shepherd — biz.irumx.app PR #8
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-15-141846-biz.irumx.app-improve)
# 회차 노트 2026-10-15-141846-biz.irumx.app-improve — biz.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:18] base pinned — main@714a660
- [러너 14:18] autonomy release — 

## 정찰 노트
- 보류 1순위였던 쿼터 Retry-After 를 골랐다. 직전 회차가 "day/month 429 를 노출하는 경로가 거의 없다" 며 보류했는데, 이번에 `helpers.ts:131` 의 `limit()` 이 주석 그대로 "the common case" 이고 day 창을 쓰면서 `rateLimited` 를 삼키지 않고 `respond.ts:69` 의 `retry-after` 헤더까지 올려보내는 것을 확인해 전제가 깨졌다. 차선이던 finance from/to 는 DB e2e 가 필요하고 finance 경로에 가까워 제쳤고, CI 원인 확정은 저장소 밖 문제로 네 회차째라 `rejected` 로 내렸다.
- 추측으로 적은 것: 기준 1·2 의 기대 숫자는 **실행값이 아니라 손계산**이다 — 이 세션에서 `node` 실행이 권한에 막혔다(`node -e`·`node <파일>` 둘 다 "requires approval"). 실제로 재계산하다 월 값을 한 번 틀려(2678400 → 2624400) 고쳤으니 구현자는 표를 믿지 말고 먼저 돌려 보고, 어긋나면 불변식(기준 4)을 정본으로 삼아야 한다. 재현 스크립트를 `check-retry.mjs` 로 남겼지만 역시 실행하지 못했다.
- 구현자가 조심할 것: `windowStart` 는 `quota_counters` 의 기본키라 **절대 건드리지 말 것**(라벨을 옮기면 카운터가 고아가 되고 한도가 한 번 리셋된다). 고치는 것은 `retryAfterSeconds` 의 산술뿐이고 `minute` 분기는 이미 맞으니 손대지 않는다.
- 충돌 회피: 미머지 PR #5·#6 이 `domain.test.ts`(+58행)와 새 `time.test.ts`(15개)를 들고 있다. 이 체크아웃 `main@714a660` 의 `__tests__/` 에는 둘 다 없으므로 새 파일 `quota.test.ts` 로 가도록 과제서에 못 박았다.
- 증거 수준의 한계: `consume()` 은 실제 `Sql` 을 요구하고 vitest 에 DB 가 없어 순수 함수(`retryAfterSeconds` 를 export)까지만 증명한다. 운영자 지시대로 가짜 `sql` 주입은 금지했고, DB 의존 vitest 를 새로 만드는 것(CI 모양 변경)도 금지했다.
- [러너 14:24] scout done — 쿼터 429 의 Retry-After 가 하루·한 달 창에서 실제 초기화 시각보다 정확히 9시간 길다 (가치 4 / 위험 2 / 작�

## 구현 노트
- 무엇·왜: `quota.ts` 에 `windowOpensAt` 을 추가하고 `retryAfterSeconds` 의 day·month 분기만 그쪽을 세게 했다. 카운터 열쇠는 KST 날짜를 UTC 자정으로 라벨링한 값이라 창은 라벨보다 9시간 이른 UTC 15:00 에 열리는데, 헤더는 라벨까지를 세어 항상 32,400초 과대였다. `retryAfterSeconds` 를 export 했다(시험용, 한 줄 주석으로 이유 명시).
- 확신 없는 곳·검증 못 한 것: (1) **실제 HTTP 429 응답을 관측하지 못했다** — DB 가 필요한 `consume()` 를 돌릴 수 없어 증거는 순수 함수 반환값과 불변식까지다. `rateLimited` → `respond.ts:69` 의 헤더 배선은 소스 읽기로만 확인했다. (2) `npm run e2e`/`ui`/`a11y` 미실행(과제서가 범위 밖으로 지정). (3) 프로덕션에서 day/month 429 가 실제로 얼마나 나는지 모른다 — 가치 추정의 근거는 `helpers.ts:131` 의 `limit()` 주석("the common case")뿐이다.
- 일부러 안 한 것: `windowStart`(DB 기본키라 옮기면 카운터 고아화), `util/time.ts`, `minute` 분기, `agent.ts:97` 이 `retryAfter` 를 버리는 것(별개 사안 — ideas.json 에 신규 항목으로 올렸다), `.github/workflows`.
- 다음 역할이 조심할 것: 새 시험은 DB·브라우저 없이 돈다(`npm test -- packages/core/src/__tests__/quota.test.ts`). 과제서 기대값은 손계산이었으나 `verify-retry.mjs` 를 실제로 돌려 9개 값 전부 일치를 확인했다. 되돌림 재실패를 sed 로 할 때 `while` 조건(51행)이 먼저 매치되니 주의 — 그 변이는 9개만 실패시킨다(의도한 변이는 54·56행, 23개 실패).
- [러너 14:28] brief accepted — 채택 — 지정된 권장 형태(`windowOpensAt` 한 함수 추가, day·month 분기 끝줄만 교체, minute·`windowStart`·`time.ts` 무변경)를 그대�
- [러너 14:28] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인한 것: quota.ts:54,56 을 수정 전 형태로 되돌려 실제로 돌렸고 35개 중 23개 실패 → 복원 후 35개 통과(원장에 `- 실패 재현:` 줄이 없어 직접 재현). `npm run check` 전체 통과(eslint + tsc 3개 + 173개 시험). 9개 기대값을 KST 경계로 독립 재계산해 전부 일치, `Math.max(1,…)` 하한도 음수·0 불가 확인. 범위는 `windowStart`·minute·time.ts·마이그레이션 무변경으로 깨끗하고 순수 함수라 revert 로 완전히 돌아온다. 판정 approve / risk low / blocking 없음.
- 못 본 것: 실제 HTTP 429 응답(DB 필요). 시험이 치는 `retryAfterSeconds` 는 quota.ts:103 이 쓰는 그 함수이므로 errors.ts:97 → respond.ts:68-69 의 헤더 설정 한 단계만 소스 읽기로 남았다. e2e/ui/a11y 미실행(과제서 범위 밖).
- 승인이어도 남는 우려 — 릴리즈 노트: 체감 범위가 커밋 메시지보다 좁다. `apps/web/src/api/client.ts:181` 이 `retry-after` 를 파싱해 `ApiError.retryAfter` 에 담지만 apps/web 안에서 읽는 곳이 없다. 수혜자는 헤더를 보는 API·MCP 클라이언트이고 화면이 아니다 — UI 개선으로 읽히게 쓰지 말 것.
- 다음 회차: `retryAfterSeconds` 가 시험용으로 export 되어 core 공개 표면이 한 함수 늘었다. agent.ts:97 이 `retryAfter` 를 버리는 경로에서는 이번 수정 효과가 여전히 소실된다(구현자가 ideas.json 신규 항목으로 올림). VERSION 은 0.1.4 그대로다.
- [러너 14:32] review approved — 리뷰 승인 (risk=low)
- [러너 14:32] pr created — https://github.com/hkjang/biz.irumx.app/pull/8
- [러너 14:33] ci failed — 성공이 아닌 검사: check=failure · 실패한 검사: ? 잡: check 

## 수리 노트
- 맞았던 것: `check=failure` 는 사실. **틀렸던 것은 원인 귀속** — 잡 116265181090 은 steps=0·3초이고 annotation 이 `The job was not started because recent account payments have failed or your spending limit needs to be increased`(Actions 결제·지출한도 차단). 저장소가 체크아웃조차 안 됐다.
- 결정적 증거: 이 PR 커밋이 없는 `main@714a660` 의 run 38491249147 도 3초·동일 annotation. 10-14 00:30 이후 전 run 이 3~5초, 그 직전 성공 run 은 9~11분. 이 diff 와 무관한 계정 차단이다.
- 고친 방법: 없음(커밋 없음). 로컬에서 `npm ci`·`npm run check`(173 시험)·`npm run build`·`ci-env.mjs` 전부 통과해 같은 자리 실패가 재현되지 않았다. `quota.ts:54,56` 되돌림 → 23/35 실패, 복원 → 35 통과로 인과만 독립 재확인했고 트리는 clean.
- 여전히 확신 없는 곳: 실제 HTTP 429 응답은 아직 관측 못 했다(DB 필요) — `errors.ts:97` → `respond.ts:68-69` 헤더 배선은 소스 읽기까지. CI 가 다시 돌면 e2e 단계가 이 공백을 처음으로 덮는다.
- 다음 조치는 저장소 밖: hkjang 의 GitHub Billing & plans 복구 후 `gh run rerun 38687364903`. 그 전에는 이 PR 을 CI 로 평가할 수 없다(5회차째 같은 상태).
