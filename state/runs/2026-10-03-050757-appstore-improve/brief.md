# 과제서 (2026-10-03, base main@17be5ee / v2.11.11)

- 과제: 앱 상세 헤더가 상태 배지에 영문 내부값 `published` 를 그대로 노출한다 (가치 3 / 위험 1 / 작업량 S)

- 왜: 공개 앱 상세(`/apps/:slug`)는 카탈로그에서 가장 많이 열리는 화면인데, 헤더의 배지 줄이 `{item.status}` 를 그대로 그려 모든 사용자에게 한국어 UI 한가운데서 영문 내부 enum `published` 를 보여 준다(`web/src/pages/public-pages.tsx:469-475`, 직접 확인). 바로 두 파일 옆에 같은 상태값을 한국어로 옮기는 공용 표 `APP_STATUSES`/`AppStatusBadge`(`web/src/features/apps/app-status.tsx`, 14줄 전체 확인)가 이미 있고 내 앱 목록(`personal-pages.tsx:88`·`:126` 의 `<AppCard app={app} showStatus />`)은 그것을 써서 "게시됨" 으로 보여 주므로, 같은 제품 안에서 같은 값이 화면에 따라 `published` 와 `게시됨` 으로 갈린다. 공용 배지를 재사용하면 노출 문구가 한국어로 바뀌고 상태 라벨의 정본이 한 곳으로 모인다.

- 수용 기준:
  1) `/apps/:slug` 를 열면 헤더 배지 줄에 `published` 가 아니라 `게시됨` 이 보인다. 배지의 tone 은 지금과 같은 positive 를 유지한다(현재도 `item.status === "published" ? "positive" : "warning"` 이라 게시된 앱의 색은 변하지 않는다).
  2) `status` 가 없는 응답에서는 지금처럼 배지가 아예 렌더되지 않는다 — `{item.status && ...}` 가드를 그대로 남길 것. (`AppStatusBadge` 는 `status` 가 undefined 면 `—` 를 그리므로 가드를 지우면 빈 배지가 새로 생긴다. 이 동작은 `app-status.tsx:12-13` 에서 확인했다.)
  3) 같은 화면의 다른 배지(`추천`, `SecurityVerifiedBadge`)와 아래 `정보` 카드·`서비스 열기`·즐겨찾기 버튼·`AppAdminLink` 는 그대로다.
  4) 테스트가 증명할 것: 수정 전에 실패하고 수정 후 통과하는 Vitest 1건 이상 — 공개 상세 응답에 `status: "published"` 를 주고 화면에 `게시됨` 이 있고 `published` 라는 텍스트는 없음을 단언한다. `web/src/pages/public-pages.test.tsx` 에 이미 공개 상세를 띄우는 테스트(파일 내 `status: "published"` fixture 가 line 28·112·209·319·396 에 있음)가 있으므로 그 패턴(실제 `AuthProvider`·`FavoritesProvider`·TanStack Query·라우터를 쓰고 HTTP 경계만 스텁)을 그대로 따를 것. 기존 테스트 전부가 계속 통과해야 한다(기존 동작 보존 기준선).

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `web/src/pages/public-pages.tsx:469-475` — `AppDetailPage` 의 `badge-row` 안 인라인 블록
    ```
    {item.status && (
      <Badge tone={item.status === "published" ? "positive" : "warning"}>
        {item.status}
      </Badge>
    )}
    ```
    을 `{item.status && <AppStatusBadge status={item.status} />}` 로 교체. import 는 `../features/apps/app-status` 에서 새로 추가할 것(현재 이 파일의 line 30 은 `import { AppCard } from "../features/apps/app-card";` 뿐이고 `app-status` 를 가져오지 않는다 — 직접 확인). **`Badge` import 는 지우지 말 것**: 이 파일의 line 92·419·465·603 에서 계속 쓴다(직접 grep 확인).
  - `web/src/pages/public-pages.test.tsx` — 위 기준 4의 테스트 추가.

- 검증 명령 (이 순서대로, 워크트리에 `web/node_modules` 가 **없으므로** 1번이 첫 단계):
  1) `npm --prefix web ci --no-audit --no-fund`  (복합 bash 한 줄로 묶지 말고 단일 명령으로 — 과거 회차에서 권한 승인에 막힌 이력 있음)
  2) 수정 전에 새 테스트만 돌려 red 확인: `npm --prefix web test -- src/pages/public-pages.test.tsx`
  3) `npm --prefix web test`  — 기준선 수는 **직접 실행해 확인할 것**. 직전 회차 기록은 95건이지만 미확인이다. `it(` grep 으로 세지 말 것(실제 통과 수와 다름).
  4) `npm --prefix web run lint` / `npx --prefix web prettier --check <수정한 파일>` / `npm --prefix web run build` / `./scripts/check-offline-assets.sh web/dist`
  5) `go build ./cmd/server`  — Go 코드는 안 건드리지만 빌드가 깨지지 않는지. 산출물 `server` 는 커밋 전에 삭제할 것(과거 회차에서 매번 지웠다).
  6) 여유가 있으면 `CI=true npm --prefix web run test:e2e` — 이 과제는 E2E 없이도 성립한다. 돌릴 경우 Chromium 설치가 선행이고 오래 걸리며, 기존 skip 1건(모바일 전용 테스트의 desktop 제외)은 정상이다.
  7) 수정만 되돌려(`git stash push -u -m "<unique-tag>"` 후 `git stash apply <sha>`, 또는 임시 WIP 커밋) 새 테스트가 다시 실패하는 것을 확인.

- 위험과 피할 것:
  - **배지를 지우지 말 것.** `app-card.tsx:19-22` 주석은 "공개 카탈로그는 게시된 앱만 나열하므로 카드에서 상태는 noise" 라고 적고 있어 공개 상세에서도 배지를 **삭제**하는 선택지가 있다. 이번 회차의 범위는 라벨 교체뿐이다 — 정보를 없애는 변경은 사람의 판단이 필요하고 과거에 "출력이 실제로 변하지 않거나 정보를 줄이는 수정" 이 반려된 이력이 있다.
  - `app-status.tsx` 의 `APP_STATUSES` 표는 건드리지 말 것 — 관리자 화면·내 앱 목록이 같은 표를 쓴다. 라벨 문구를 "바꾸는" 과제가 아니다.
  - `publiclyViewable`·`AppCard`·`showStatus` 계약을 함께 바꾸지 말 것(2026-10-01 회차 `4ffaa90` 가 막 정리한 자리다).
  - 보호 경로 금지: `internal/auth`, `migrations`, `.github/workflows`, 세션/CSRF/rate limit, `internal/webui/dist`. 이 과제는 `web/src` 두 파일로 끝난다.
  - **즐겨찾기 개수·페이지 nav(`public-pages.tsx:361`·`:410`)는 건드리지 말 것.** 2026-09-23 회차의 같은 수정(`0898f7a`)이 아직 main 에 없는 상태로 review-pending 이며(이번 회차에서도 line 361 이 `${apps.data.total}개 앱`, line 410 에 `!favoritesOnly` 없음을 재확인), 다시 고치면 중복 제출이 된다.
  - `tone` 계산을 "개선" 하려 들지 말 것. `AppStatusBadge` 가 이미 `positive` 를 주므로 기존 인라인 삼항을 그냥 지우면 된다.

- 차선 후보: **`/my/apps` 의 빈 상태가 그 화면과 맞지 않는 일반 문구를 보여 준다** (가치 2 / 위험 1 / 작업량 S). `personal-pages.tsx:120` 의 `<EmptyState actions={<ButtonLink to="/submit">앱 등록</ButtonLink>} />` 가 title·description 없이 렌더되어 `ui.tsx:258-260` 의 기본값 "표시할 항목이 없습니다 / 조건을 바꾸거나 새 항목을 등록해 보세요." 가 나온다. 그 화면에는 필터가 없어 "조건을 바꾸어" 볼 것이 없고, 같은 상황의 대시보드(`personal-pages.tsx:79-83`)는 "등록한 앱이 없습니다" 를 쓴다. 대시보드와 같은 문구로 맞추고 Vitest 1건을 더하면 끝난다(프로덕션 파일 1개).
