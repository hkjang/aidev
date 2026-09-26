- 과제: 「목록 주소 복사」에서 저장용 500자 잘림을 분리해 복사한 주소가 주소창과 같은 결과를 내게 하기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `listSharePath`(web/src/list-export.ts:68)가 `savedListQuery`(web/src/saved-list-views.ts:67)를 그대로 재사용하는데, 그 안의 `clip(value, 500)`(saved-list-views.ts:55)은 localStorage 저장용 한도다. 주소창의 `q`·`f_*`는 `patchListParams`(web/src/list-view.ts:116)가 자르지 않으므로, 500 코드유닛을 넘는 검색어·필터를 걸고 보던 화면에서 「목록 주소 복사」(web/src/list-tools.tsx:213)를 누르면 **사용자가 보고 있는 것과 다른 결과를 내는 주소**가 조용히 복사되고, 받는 사람은 잘린 조건으로 다른 목록을 본다.
- 수용 기준:
  1) 501자 이상의 `q`와 `f_<key>`가 들어 있는 `URLSearchParams`로 `listSharePath`를 부르면, 돌려준 주소를 `new URL(...).searchParams`로 다시 읽었을 때 `q`·`f_*`가 **원본 문자열과 완전히 같다**(길이·내용 모두, U+FFFD 없음).
  2) `savedListQuery`를 직접 부르는 기존 두 경로 — `web/src/list-tools.tsx:96`의 저장 스냅샷과 `applySavedListQuery`(saved-list-views.ts:92) — 는 **동작이 그대로**다. 즉 500자 잘림과 서로게이트 쌍 보정이 유지되고, 기존 테스트 "unknown and oversized controls cannot override accepted list configuration" 과 "clipping a long search or filter never splits a surrogate pair into replacement characters"(web/tests/saved-list-views.test.mjs)가 수정 없이 통과한다.
  3) 주소 복사의 기존 허용 계약이 그대로다: 정렬 키는 `columns`에 있을 때만, `size`는 `10/50/100`만, `tab`은 기존 허용 목록만, `baseline`은 `tab=compare` + `/software|/campaigns/<uuid>` 경로일 때만, `page`는 2 이상일 때만 남고 `item`/`token`/`draft` 같은 임의 매개변수는 사라진다(web/tests/convenience.test.mjs:34 "list sharing preserves the active query and comparison but strips arbitrary data" 가 그대로 통과).
  4) 테스트가 증명할 것: (a) 잘림 없는 주소 복사, (b) 저장한 보기 쪽 500자 잘림은 살아 있음 — 같은 입력을 두 경로에 넣어 **한쪽만 잘리는 것**을 한 테스트 안에서 대조할 것.
- 건드릴 파일 (프로덕션 2개 + 테스트 1~2개):
  - `web/src/saved-list-views.ts:67 savedListQuery` — 네 번째 인자로 잘림 여부를 받는다(예: `limit = 500` 또는 `options: { clip?: boolean }`). **기본값은 현재 동작(500자 잘림)** 으로 두어 `list-tools.tsx:96`·`applySavedListQuery`의 호출이 한 글자도 바뀌지 않게 할 것. `clip` 헬퍼(55~64행)와 그 주석은 저장 경로 설명이므로 그대로 둔다.
  - `web/src/list-export.ts:68 listSharePath` — 75행의 `savedListQuery(params, columns, filters)` 호출에만 "자르지 않음"을 넘긴다. 이 함수의 다른 부분(tab/baseline/page)은 손대지 않는다.
  - `web/tests/convenience.test.mjs` — "list sharing" 테스트 아래에 회귀 테스트 1개 추가. TDD: 먼저 501자 `q`(예: `"가".repeat(400) + "a".repeat(200)`)와 501자 `f_service_id`로 실패를 확인한 뒤 고칠 것.
  - (선택) `web/tests/saved-list-views.test.mjs` — 새 인자의 기본값이 잘림을 유지한다는 단언 1줄.
- 검증 명령 (이 저장소에서 실제로 도는 것):
  ```sh
  npm --prefix web ci
  npm --prefix web test          # 현재 기준선 93 통과 / 0 실패 / 0 skip (이번 회차 미실행 — 착수 시 먼저 확인할 것)
  npm --prefix web exec -- tsc -p tsconfig.json --noEmit
  npm --prefix web run build
  git diff --check
  ```
  Go 코드는 바뀌지 않으므로 `go test`·`internal/webassets/dist` 재복사는 불필요하다. 다만 `npm run build` 는 `tsc --noEmit && vite build` 라 타입 오류를 여기서 잡는다.
- 위험과 피할 것:
  - **저장 경로의 한도를 같이 풀지 말 것.** `readListPreferences`(saved-list-views.ts:17)는 `item.query.length > 8192` 인 보기를 조용히 버린다. 주소 복사만 풀고 저장은 500자 유지가 이번 과제의 전부다.
  - `savedListQuery` 의 기본 인자를 바꾸면 `list-tools.tsx:96`의 `snapshot` 과 저장된 `view.query` 비교(`selected = prefs.views.find(item => item.query === snapshot)`)가 어긋나 "현재 조건 = 저장한 보기" 배지가 사라진다. 기본값 유지 여부를 이 비교로 직접 확인할 것.
  - 서버 URL 길이 걱정은 불필요하다 — 확인함: `use-list-view.tsx`(40~95행)는 `rows.filter(...)`로 **클라이언트에서만** 검색·필터하고 `q`/`f_*` 를 API로 보내지 않는다. 그래도 새 네트워크 호출을 끼워 넣지 말 것.
  - tracking·auth·OIDC·Go 쪽은 건드리지 말 것. 최근 회차에서 tracking 경로는 verify-failed(2026-09-25)·기각(2026-09-25)이 연속으로 났다.
  - 과거 교훈: 계약이 다른 두 소비자(저장 / 주소)를 **통합하지 말고 분리**하는 방향이 맞다. 단, 분리한 뒤 두 경로를 같은 입력으로 한 테스트에서 대조해 증명할 것.
  - 미확인: 이 세션에서 `node`·`npm` 실행이 샌드박스에 막혀 **실제로 재현 실행을 하지 못했다.** 위 결함은 코드 읽기(`patchListParams` 무잘림 → `listSharePath` → `savedListQuery` → `clip`)로만 확인했다. 구현자는 반드시 실패하는 테스트부터 써서 재현을 실증할 것. 기준선 93 통과도 이전 회차 기록이며 이번에 재확인하지 않았다.
- 차선 후보: **저장한 보기가 8192자 한도를 넘겨 새로고침 후 조용히 사라지는 것 막기** — `list-tools.tsx`의 `save()`(123행 부근)는 이름 공백·8개 상한·이름 중복만 검사하고 `savedListQuery` 스냅샷 길이는 검사하지 않는데, `readListPreferences`는 `query.length > 8192`인 보기를 버린다. 한글은 `URLSearchParams.toString()`에서 문자당 9자로 퍼센트 인코딩되므로 500자 `q` + 500자 필터 하나면 9,000자를 넘어 한도를 초과한다. 저장은 성공한 것처럼 보이고 다음 방문에 사라진다. 다만 `save()`가 컴포넌트 안 인라인 함수라 순수 함수로 뽑아내야 테스트할 수 있어(web/tests에는 DOM·React 하네스가 없다) 1순위보다 작업량이 크다.
