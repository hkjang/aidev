- 과제: 저장한 목록 보기가 브라우저 저장소 한도(8192자)를 넘겨 새로고침 후 조용히 사라지는 것을 저장 시점에 거절하기 (가치 3 / 위험 1 / 작업량 M)
- 왜: `web/src/list-tools.tsx` 의 `save()` 는 이름·8개 상한·중복·"q/필터 각 500자 초과" 만 검사하는데, `readListPreferences`(web/src/saved-list-views.ts:37)는 `item.query.length > 8192` 인 보기를 조용히 버린다. 한글은 `URLSearchParams` 직렬화에서 1자→9자라 500자 q + 500자 필터 하나면 약 9,016자가 되어 한도를 넘는다 — 사용자는 "‘X’ 보기를 저장했습니다" 를 보고도 다음 방문에 보기가 사라진다. 저장 시점에 같은 한도로 거절하면 성공 메시지가 실제 보존을 뜻하게 된다.
- 수용 기준:
  1) 직렬화 길이가 8192자를 넘는 목록 상태에서 보기 저장을 누르면 한국어 오류 문구가 나오고 `view.setPreferences` 가 호출되지 않는다(성공 메시지 없음).
  2) 8192자 이하인 기존 보기 저장 동작·문구·스냅샷 문자열은 한 글자도 바뀌지 않는다(기존 `saved-list-views.test.mjs` 단언 그대로 통과).
  3) 테스트가 **프로덕션 함수만으로** 인과를 증명한다: `savedListQuery`(q=가×500 + f_service_id=가×500) → 결과 길이 > 8192 → 그 query 를 담은 JSON 을 `readListPreferences` 에 넣으면 `views` 가 빈 배열(= 조용한 소실 재현) → 새 판정 함수가 그 같은 query 에 대해 참을 돌려준다. 500자 이하 정상 케이스는 거짓.
- 건드릴 파일 (3개):
  - `web/src/saved-list-views.ts` — `readListPreferences` 안에 하드코딩된 `8192` 를 `export const savedListQueryLimit = 8192` 로 뽑고(같은 값 사용), 순수 판정 함수 `export function savedListQueryTooLong(query: string)` (= `query.length > savedListQueryLimit`) 추가. `clip`/`savedListQuery`/`applySavedListQuery` 는 손대지 않는다.
  - `web/src/list-tools.tsx` — `save()`(125~165행)의 기존 500자 검사(144~150행) **뒤에** `if (savedListQueryTooLong(snapshot))` 분기를 추가하고 한국어 오류를 `setError` 로 설정하고 `return`. `snapshot`(96~100행)은 이미 계산돼 있으니 재계산하지 말 것. import 목록에 새 함수 추가.
  - `web/tests/saved-list-views.test.mjs` — 위 수용 기준 3)의 회귀 테스트 추가(기존 파일 스타일: `node:test` + 실제 TS 함수 직접 호출).
- 문구 제안(한국어, Mantine 오류 자리 그대로): `"검색어와 필터가 너무 길어 이 보기를 저장할 수 없습니다. 조건을 줄여 주세요."` — 기존 500자 문구와 구분되게 쓸 것.
- 검증 명령:
  - `npm --prefix web ci`
  - `npm --prefix web test`  ← 기준선을 **먼저 실측해 기록**할 것(2026-09-26 회차 실측은 96통과/0실패/0skip, 정찰 프로필의 93은 오래된 수치다)
  - `npm --prefix web run typecheck`(= `tsc --noEmit`, package.json:8 에 실제로 있음) 그리고 `npm --prefix web run build`(= `tsc --noEmit && vite build`)
  - `git diff --check`
  - Go 무변경이면 Go 스위트·`internal/webassets/dist` 재복사는 불필요(dist 는 `.gitkeep` 만 추적).
- 위험과 피할 것:
  - **`clip(500)` 한도를 건드리지 말 것.** 2026-09-23·2026-09-26 두 회차가 이 상수 주변을 이미 고쳤고, `savedListQuery` 는 저장·복원(`applySavedListQuery`)·주소 복사(`listSharePath`, `{clip:false}`) 세 소비자가 공유한다. 한 곳을 바꾸면 나머지 둘이 어긋난다.
  - `readListPreferences` 의 `> 8192` 판정을 느슨하게 만들지 말 것 — localStorage 실패를 저장 시점으로 옮기는 것이 목적이고, 읽기 쪽 계약은 그대로 둔다.
  - tracking/CSP 경로(`tracking*.ts`, `internal/app/tracking.go`)는 2026-09-25 두 회차가 verify-failed·기각. 이번 변경과 무관하니 손대지 말 것.
  - `save()` 는 컴포넌트 인라인이고 `web/tests` 에 DOM·React 하네스가 없다. **인라인 `save()` 를 위한 가짜 view 객체를 만들어 결함을 증명하지 말 것** — 판정을 순수 함수로 뽑아 `savedListQuery`/`readListPreferences` 실제 함수로 왕복 증명하는 것이 이 저장소에서 통한 방식이다.
  - `localStorage` 전체 한도(보기 8개 합산)까지 검사하려 들지 말 것 — 범위 확대다. 이번엔 개별 보기 query 한도만.
- 이번 정찰이 실제로 확인한 것: `saved-list-views.ts` 전문(114행)·`list-tools.tsx` 1~200행을 열어 `save()` 의 네 검사(이름/8개/중복/500자)와 `snapshot` 계산 위치를 확인했다. `8192` 는 저장소 전체(web/src, web/tests, internal/app)에서 `saved-list-views.ts:36` **한 곳에만** 있다(grep 확인) — 상수 추출이 다른 소비자를 건드리지 않는다. `web/tests` 18파일, `package.json` 에 `typecheck` 스크립트 존재, 테스트 스타일은 `node:test` + `../src/*.ts` 직접 import.
- 미확인(구현자가 실측할 것): 이번 샌드박스에서 `node`/`npm` 실행이 승인 차단되어 **테스트를 한 번도 돌리지 못했다.** 9,016자 계산은 손계산(`가`→`%EA%B0%80` 9자 × 500 = 4,500; `q=` 2 + `&f_service_id=` 14)이며 `URLSearchParams` 실측이 아니다. 착수 첫 단계로 이 길이를 실측해 8192 초과를 확인하고, 아니면 필터 개수를 늘려 넘기는 실제 목록 화면을 찾을 것. 또 한 화면의 필터 개수 상한은 확인하지 않았다.
- 차선 후보: `oidcReturnTo`(Go)와 `safeReturnPath`(TS) 의 `return_to` 규칙을 공유 JSON 벡터로 교차 검증 (3/2/M). 단 auth 보호 경로이므로 "의도된 정규화 차이" 를 먼저 벡터 파일에 문서로 정의하고, Go 쪽 테스트가 `HUNTER_TEST_DSN` 없이 도는 순수 함수인지 먼저 확인할 것.
