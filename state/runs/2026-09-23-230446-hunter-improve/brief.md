# 과제서 — 2026-09-23-230446-hunter-improve (hunter, base main@96f7414)

- 과제: 저장한 목록 보기·목록 주소 복사의 검색어/필터 500자 잘림을 코드포인트 단위로 고쳐 문자가 깨지지 않게 하기 (가치 3 / 위험 1 / 작업량 S)

- 왜: `web/src/saved-list-views.ts`의 `savedListQuery`는 `q`와 각 `f_<key>`를 `value.slice(0, 500)`으로 자르는데, 이는 UTF-16 코드유닛 절단이라 500번째 경계가 서로게이트 쌍(이모지·보조평면 문자) 가운데에 떨어지면 외톨이 서로게이트가 남는다. 이 값은 곧바로 `new URLSearchParams(...).toString()`으로 직렬화되어(같은 파일 `savedListQuery` 반환, `list-export.ts:listSharePath`의 URL 생성) U+FFFD로 치환되므로, 저장한 보기를 다시 적용하거나 복사한 주소를 연 사람은 **원래 검색어와 다른 문자열**로 검색하게 된다.

- 수용 기준:
  1) 500번째 코드유닛이 서로게이트 쌍 가운데인 검색어를 `savedListQuery`에 넣으면 반환 질의를 `new URLSearchParams(...).get("q")`로 되읽었을 때 U+FFFD가 없고, 잘린 결과가 원본의 접두사(prefix)여야 한다(`original.startsWith(result)` 참).
  2) 같은 보장이 `f_<key>` 필터 값에도 적용된다(현재 같은 `slice(0,500)` 사용).
  3) 기존 동작은 그대로다 — 500코드유닛 이하 값은 바이트 단위로 동일하게 반환되고, 허용 키 목록(`q`/`sort`/`dir`/`size`/`f_*`)·필터 정렬 순서·`sort`가 `columns`에 없을 때 `dir`도 빠지는 규칙은 변하지 않는다. `web/tests/saved-list-views.test.mjs`·`convenience.test.mjs`의 기존 케이스가 그대로 통과한다.
  4) 테스트는 **수정 전에 실패**해야 한다(TDD). 먼저 실패를 눈으로 확인한 뒤 구현한다.

- 건드릴 파일:
  - `web/src/saved-list-views.ts:savedListQuery` — `q`와 `f_<key>`의 `.slice(0, 500)` 두 군데를 코드포인트 안전 절단으로 교체. 권장 구현: 작은 로컬 헬퍼 `function clip(value: string, limit = 500)` 를 같은 파일에 두고, `value.length <= limit`이면 그대로 반환, 아니면 `limit` 위치가 low surrogate(`0xDC00–0xDFFF`)인지 보고 한 칸 줄여 자른다. (`Array.from(value).slice(0,500).join("")`은 한도의 의미를 코드유닛→코드포인트로 바꿔 3)의 기존 동작 보존을 깨므로 쓰지 말 것 — 한도는 코드유닛 500 그대로 두고 경계만 보정한다.)
  - `web/tests/saved-list-views.test.mjs` — `savedListQuery`를 직접 부르는 회귀 테스트 1개 추가. 실제 `URLSearchParams` 왕복으로 증명하고(소스 문자열 검사 금지), `q`와 `f_*` 둘 다 덮는다. 문자열 구성 예: `"가".repeat(499) + "🚀" + "나"`.
  - (문서 변경 불필요 — 사용자에게 보이는 문구·API 계약 변화 없음. `openapi.json`·가이드는 건드리지 말 것.)

- 검증 명령 (이 저장소에서 실제로 도는 것):
  ```sh
  npm --prefix web test          # 이번 정찰 시점 기준선: 이전 회차 기록 93 통과 / 0 실패 (이번 회차에서 직접 재실행은 하지 못함 — 구현자가 먼저 기준선을 찍을 것)
  npm --prefix web exec -- tsc --noEmit
  npm --prefix web run build
  ```
  Go 코드는 바뀌지 않으므로 Go 테스트 스위트(수백 초, DB 필요)는 돌리지 않는다. `internal/webassets/dist` 재복사도 이 과제에는 불필요하다(Go 실행 파일 검증을 하지 않으므로). 마지막에 `git diff --check`.

- 위험과 피할 것:
  - **500이라는 한도 자체를 바꾸지 말 것.** `readListPreferences`의 `item.query.length > 8192` 저장 한도와 짝이 맞춰져 있다.
  - **`listSharePath`(주소 복사)에 500자 한도를 적용할지 말지의 정책 논쟁으로 범위를 넓히지 말 것.** 그건 별도 보류 아이디어다. 이번에는 "자르는 위치만 올바르게"에 한정한다.
  - `applySavedListQuery`의 키 삭제 목록(`q`,`sort`,`dir`,`page`,`size`,`f_*`)을 건드리지 말 것 — 저장한 보기가 내비게이션·상세 ID·승인 매개변수를 못 바꾸게 하는 보안 경계다.
  - 보호 경로 접근 금지: `internal/app/auth*.go`, `mcp_oauth.go`, `third_party/pentagi`, `.github/workflows`, `scripts/release.sh`. 이 과제는 `web/` 안에서 끝난다.
  - **과거 교훈**: 이전 회차들의 성공 기록(메일 모듈, `trackingSnippetOrigins`, `handoffClaimsPerUser`, Go `csvSafe` 정렬)은 **main@96f7414에 없다** — 이번 정찰에서 grep으로 확인했다. 그 PR들이 아직 리뷰 대기이므로 같은 것을 다시 구현하지 말 것.
  - 운영자 규칙: 실제 출력이 바뀌지 않는 수정은 넣지 말 것. 이 과제는 1)의 왕복 테스트로 출력 변화를 증명할 수 있어야 하며, 만약 구현 전 테스트가 **통과해 버리면**(즉 Node/브라우저가 외톨이 서로게이트를 U+FFFD로 바꾸지 않으면) 이 과제는 성립하지 않으니 즉시 차선 후보로 넘어갈 것. (나는 이 U+FFFD 치환을 URL 직렬화 규격에서 추론했고, 이번 회차에서 `node -e` 실행 권한을 얻지 못해 **직접 실행으로 확인하지 못했다 — 미확인**.)

- 차선 후보: **`savedListQuery`의 허용 페이지 크기 목록에서 빠진 항목 정리 대신** → `web/src/list-export.ts:listSharePath`가 주소 복사에도 저장소용 500자 한도를 적용해 긴 검색어를 조용히 자르는 문제를, "주소 복사는 자르지 않고 저장한 보기만 자른다"로 분리하기(가치 3 / 위험 1 / 작업량 S). `savedListQuery`에 절단 여부 플래그를 넘기는 방식이 가장 작다. 기존 `convenience.test.mjs:"list sharing preserves the active query..."` 테스트가 깨지지 않는지 반드시 확인할 것.
