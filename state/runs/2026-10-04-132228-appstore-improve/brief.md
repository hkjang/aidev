- 과제: `/my/apps` 의 빈 상태가 필터 없는 화면에서 "조건을 바꾸어" 보라고 안내한다 (가치 2 / 위험 1 / 작업량 S)

- 왜: `MyAppsPage`(`web/src/pages/personal-pages.tsx:120`)의 `EmptyState` 가 `actions` 만 주고 title/description 없이 렌더되어, `ui.tsx:258-260` 의 기본값 **"표시할 항목이 없습니다 / 조건을 바꾸거나 새 항목을 등록해 보세요."** 가 그대로 나온다. 그런데 이 화면에는 검색창도 필터도 정렬도 없어(`PageHeader` + `card-grid` 뿐 — 직접 확인) 바꿀 "조건" 자체가 존재하지 않고, 앱을 한 번도 등록하지 않은 신규 사용자가 내 앱 화면에서 처음 읽는 문장이 이것이다. 같은 `myApps` 데이터를 읽는 대시보드(`personal-pages.tsx:79-83`)는 같은 상황에서 "등록한 앱이 없습니다" 를 쓰므로, 같은 제품 안에서 같은 상태의 안내가 화면마다 갈린다.

- 수용 기준:
  1) `api.myApps` 가 빈 배열을 돌려줄 때 `/my/apps` 의 빈 상태 제목이 "등록한 앱이 없습니다" 이고, 설명은 이 화면에 맞는 문장(예: "앱을 등록하면 상태와 검토 결과를 여기에서 확인할 수 있습니다.")이며, "조건을 바꾸거나" 문구가 화면에 **없다**.
  2) 기존 `actions` 의 `/submit` 링크("앱 등록")는 그대로 남는다.
  3) 대시보드(`/my`)의 빈 상태는 지금 그대로 "등록한 앱이 없습니다" 를 유지한다(회귀 없음).
  4) Vitest 가 수정 전에는 "등록한 앱이 없습니다" 를 찾지 못해 실패하고, 수정 후 통과한다. 프로덕션 파일만 되돌리면 같은 테스트가 다시 실패하는 것을 확인할 것.

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `web/src/pages/personal-pages.tsx` : `MyAppsPage` 의 `<EmptyState actions={...} />`(line 120 부근)에 `title`·`description` 을 전달. 그 외 아무것도 바꾸지 말 것.
  - `web/src/pages/personal-pages.test.tsx` : 기존 파일 맨 아래에 테스트 1건 추가. **이 파일에는 이미 `/my/apps` 를 띄우는 하네스가 있다** — line 52 `describe("My applications")` 의 `it`(53·78·126·150)과 line 170 `describe.each(["/my/apps", "/my"])` 가 프로덕션 라우터·`AuthProvider`·`FavoritesProvider`·TanStack Query·실제 `api` 클라이언트를 쓰고 HTTP 경계만 스텁한다. 그 하네스를 그대로 재사용해 `/api/v1/me/apps` 가 `[]` 를 돌려주게만 하면 된다(새 렌더 유틸을 직접 만들지 말 것).

- 검증 명령 (워크트리에 `web/node_modules` 가 없으므로 첫 단계가 설치다):
  ```
  npm --prefix web ci --no-audit --no-fund
  npm --prefix web test
  npm --prefix web run lint
  npx --prefix web prettier --check src/pages/personal-pages.tsx src/pages/personal-pages.test.tsx
  npm --prefix web run build
  ```
  기준선 테스트 수는 직전 회차 기록이 97건이나 **미확인** — 반드시 직접 실행해 확인하고 +1 을 기록할 것.

- 위험과 피할 것:
  - `ui.tsx` 의 `EmptyState` 기본값(`title`/`description`)은 **바꾸지 말 것**. 그 기본값은 `admin-pages.tsx:1205·1395·3557·3632`, `public-pages.tsx:639`, `personal-pages.tsx:527` 의 bare `<EmptyState />` 6곳이 공유하므로, 기본값을 고치면 이번 과제와 무관한 화면 6개의 문구가 같이 바뀌어 리뷰 범위를 벗어난다(이 6곳은 각각 필터·검색이 있는 표라 기본 문구가 틀리지 않다).
  - 이번 회차에 bare `<EmptyState />` 를 쓰는 다른 화면까지 같이 고치지 말 것 — 파일 수가 늘면 사람 손을 다시 타는 비율이 올라간다.
  - `public-pages.tsx` 는 **열지도 말 것**. 그 파일의 즐겨찾기 개수(`:362` `${apps.data.total}개 앱`)와 페이지 nav(`:411`)는 2026-09-23 회차 커밋 `0898f7a` 가 고쳤으나 아직 main 에 없는 review-pending 상태이고, 같은 줄을 다시 건드리면 중복 제출이다(2026-10-04 재확인: 여전히 main 에 없음).
  - 보호 경로(`internal/auth`·`migrations`·`.github/workflows`·`internal/webui/dist`)와 Go 쪽은 이번 과제와 무관하다. 빌드 산출물 `server`·`web/dist` 는 커밋 전에 지울 것.
  - 문구는 한국어, 커밋 메시지는 영어 `fix(web):` 관례를 따를 것.

- 차선 후보: **`/my/apps/:id/edit` 에서 앱을 찾지 못하면 영문 라이브러리 오류가 그대로 보인다 (가치 3 / 위험 1 / 작업량 S) — 전제 미확인**
  `AppFormPage`(`web/src/pages/app-form-page.tsx:69-73`)의 `existing` 쿼리는 `queryFn: async ({signal}) => (await api.myApps(signal)).find((app) => app.id === id)` 라서, 목록에 그 id 가 없으면 **`undefined` 를 반환**한다. 설치된 `@tanstack/react-query`는 `^5.87.1`(`web/package.json:20`). TanStack Query v5 는 `queryFn` 이 `undefined` 를 반환하면 쿼리를 실패시키며 `Query data cannot be undefined for query key ...` 를 던지는 것으로 알려져 있고, 그렇다면 `app-form-page.tsx:133-141` 의 `existing.error` 분기가 한국어 UI 한가운데에 그 영문 개발자 메시지를 `ErrorState` 로 그대로 띄운다(다른 탭에서 앱을 지웠거나 소유권을 잃은 뒤 수정 링크를 누른 경우). **이 v5 동작은 이번 정찰에서 실행해 확인하지 못했다(워크트리에 node_modules 가 없음).** 구현자는 먼저 `/api/v1/me/apps` 가 `[]` 를 돌려주는 상태로 `/my/apps/<id>/edit` 를 렌더하는 테스트를 써서 실제로 어떤 화면이 나오는지 관측하고, 영문 원문이 노출되는 것이 재현될 때만 착수할 것(재현되지 않으면 결함이 아니므로 1순위로 돌아갈 것). 수정 방향은 `queryFn` 에서 `find` 결과가 없을 때 한국어 메시지로 `throw` 하는 것 — 프로덕션 파일 1개.
