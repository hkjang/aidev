- 과제: `/my/apps/:id/edit` 에서 앱을 못 찾으면 영문 라이브러리 오류가 한국어 UI 에 그대로 보인다 (가치 3 / 위험 1 / 작업량 S)
- 왜: `app-form-page.tsx:71-75` 의 `existing` 쿼리는 `(await api.myApps(signal)).find((app) => app.id === id)` 로, 내 앱 목록에 그 id 가 없으면 `undefined` 를 반환한다(코드 직접 확인). TanStack Query v5 는 queryFn 의 `undefined` 반환을 오류로 바꾸므로 `:134-141` 의 `ErrorState` 가 `error.message` 를 그대로 description 에 넣어(`ui.tsx:292`) "Query data cannot be undefined…" 라는 영문 내부 메시지를 보여 준다. 삭제된 앱의 수정 주소를 북마크했거나 남의 앱 id 를 입력한 소유자가 "앱을 찾을 수 없습니다" 가 아니라 라이브러리 내부 문구를 읽게 되고, 돌아갈 링크도 없다.
- 수용 기준:
  1) `/api/v1/me/apps` 가 그 id 를 포함하지 않는 상태에서 `/my/apps/<id>/edit` 를 렌더하면 한국어 안내(예: "앱을 찾을 수 없습니다")가 보이고 화면에 `Query data cannot be undefined`·`queryFn`·`undefined` 같은 영문 내부 문구가 **전혀 없다**.
  2) 그 화면에서 `/my/apps` 로 돌아가는 링크(또는 버튼)가 있어 사용자가 막히지 않는다.
  3) 기존 동작 보존: 그 id 를 포함하는 응답에서는 수정 폼이 그대로 뜨고 기존 값이 채워진다(`name`·`slug` 입력값 확인). `/api/v1/me/apps` 가 **401/500 등 실제 HTTP 오류**를 주면 지금처럼 `ErrorState`("화면을 불러오지 못했습니다" + 다시 시도)가 뜬다 — 404 성격의 "없음" 과 네트워크 오류를 구분해야 한다.
  4) 테스트: 위 1)·2)·3)을 Vitest 로 증명한다. 수정 전에 1) 이 실패(영문 메시지 관측)하는 것을 먼저 확인하고, 수정 후 프로덕션 파일만 `git checkout` 으로 되돌려 같은 테스트가 다시 실패하는 것까지 확인할 것.
- 건드릴 파일:
  - `web/src/pages/app-form-page.tsx:71-75` — `existing` 의 `queryFn` 이 못 찾았을 때 `undefined` 대신 `null` 을 반환하게 한다(`?? null`). v5 는 `null` 은 정상 데이터로 받는다.
  - `web/src/pages/app-form-page.tsx:128-141` 뒤 — `isPending`·`error` 분기 **다음에** `if (edit && !existing.data)` 분기를 추가해 `EmptyState`(`ui.tsx:258`)로 `title="앱을 찾을 수 없습니다"` + 설명 + `actions={<ButtonLink to="/my/apps">내 앱 목록</ButtonLink>}` 를 렌더한다. 분기 순서를 지켜야 로딩 중에 "없음" 이 번쩍이지 않는다.
  - `web/src/pages/app-form-page.tsx:12-22` 의 import 블록에 `EmptyState`·`ButtonLink` 추가(현재 `Button, Card, ErrorState, Field, Input, LoadingState, PageHeader, Select, Switch, Textarea` 만 import — 직접 확인). `ButtonLink` 가 `ui.tsx` 에 있음은 `NotFoundState`(`ui.tsx:454-463`)가 쓰는 것으로 확인했다.
  - `web/src/pages/app-form-page.test.tsx` — **신규 파일**(`web/src/pages/` 에 `admin/auth/personal/public/review-pages.test.tsx` 만 있고 `app-form-page` 테스트는 없음, 이번 회차 `ls` 로 확인). 프로덕션 라우터·`AuthProvider`·`FavoritesProvider`·TanStack Query·실제 `api` 클라이언트를 쓰고 HTTP 경계만 스텁하는 하네스는 `web/src/pages/personal-pages.test.tsx` 의 것을 그대로 따라 쓸 것(손으로 만든 대역 금지).
  - `:81`·`:172` 의 `if (!existing.data) return;` / `{edit && existing.data && …}` 는 `null` 에도 그대로 동작하니 **고치지 말 것**.
  - 프로덕션 파일은 **1개**(`app-form-page.tsx`)로 끝낼 것. `ui.tsx` 의 공용 기본값은 bare `<EmptyState />` 를 쓰는 여러 화면이 공유하므로 건드리지 말 것.
- 검증 명령 (이 워크트리에는 `web/node_modules` 가 **없다** — 첫 단계로 설치):
  - `npm --prefix web ci --no-audit --no-fund` (단일 명령으로, 복합 bash 는 권한에 막힐 수 있음)
  - `npm --prefix web test` — 기준선 개수를 **먼저 실행해 기록**할 것(직전 회차 기록 99건, 이번 회차에서 미실행·미확인).
  - `npm --prefix web run lint` (`--max-warnings 0`), 수정한 파일만 `npx --prefix web prettier --check <경로>`
  - `npm --prefix web run build`, `./scripts/check-offline-assets.sh web/dist`
  - `./scripts/check-env-contract.sh`, `./scripts/check-docs.sh`, `go build ./cmd/server`
  - E2E 는 이 과제에 불필요(공개 화면 변화 없음). 돌린다면 `CI=true npm --prefix web run test:e2e` 는 오래 걸리고 desktop 통과를 전체 통과로 기록하지 말 것.
- 위험과 피할 것:
  - **TanStack Query v5 가 `undefined` 를 오류로 바꾼다는 전제는 이번 정찰에서 실행해 확인하지 못했다(미확인 — `web/node_modules` 부재).** 구현자는 가장 먼저 테스트 1건으로 실제 화면을 관측할 것. 만약 v5 가 오류를 던지지 않고 `data===undefined` 로 남으면 증상은 "빈 폼이 뜨고 저장하면 엉뚱한 일이 일어남" 이 되는데, 그래도 수용 기준 1)·2)는 같은 수정(`?? null` + not-found 분기)으로 성립한다. 관측한 실제 증상을 커밋 메시지·PR 본문에 적고 과제서의 추측을 그대로 베끼지 말 것.
  - `public-pages.tsx` 는 건드리지 말 것 — 2026-09-23 회차의 `0898f7a`(즐겨찾기 개수·페이지 nav)가 **네 회차 연속 main 에 없고** 여전히 review-pending 이다(이번 회차에 `git log` 로 재확인).
  - 입력 길이·제어문자 하드닝처럼 **출력이 실제로 변하지 않는 수정은 사람이 반려한 이력**(2026-09-08·09-10)이 있다. 이번 과제는 화면 문구가 눈에 보이게 바뀌는 것이 핵심이니 거기서 벗어나지 말 것.
  - 보호 경로 `internal/auth`·`migrations`·`.github/workflows`·`internal/webui/dist` 와 공개 API 계약(`pagination` limit 등)은 건드리지 말 것. 이 과제는 서버를 전혀 건드리지 않는다.
  - `grep` 으로 문자열이 있다는 것을 증거로 제출하지 말 것 — 실제 렌더 결과로 증명할 것.
  - 빌드 산출물 `server`(go build)는 커밋 전에 삭제할 것.
- 차선 후보: `/favorites` 에서 검색하면 즐겨찾기가 있는데도 "즐겨찾기한 앱이 없습니다 / 앱 상세 또는 카드에서 하트를 눌러 보관하세요." 가 뜬다 (3/2/S — `public-pages.tsx:385-403` 의 `EmptyState` 가 `favoritesOnly` 면 검색어·카테고리 유무와 무관하게 같은 문구를 쓰고 "필터 초기화" 액션도 `!favoritesOnly` 로 막혀 있다. 이번 회차에 해당 블록을 직접 열어 확인). 단, 위 "피할 것" 의 `0898f7a` 와 같은 파일이므로 1순위가 성립하지 않을 때만 고르고, 그때도 `:361`·`:410` 은 손대지 말 것.
