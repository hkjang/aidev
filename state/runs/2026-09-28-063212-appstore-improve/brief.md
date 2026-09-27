- 과제: 빠른 이동 팔레트에서 앱 검색 결과가 늦게 도착하면 골라 둔 항목이 밀려 Enter 가 엉뚱한 화면으로 이동한다 (가치 3 / 위험 2 / 작업량 S)
- 왜: `CommandPalette` 는 선택 위치를 배열 인덱스 `active` 로만 들고 있는데(`command-palette.tsx:109`), 검색 결과 배열은 앱 응답이 도착하면 앞쪽에 항목이 끼어드는 구조다(`results` = `appDestinations` 뒤에 매칭 메뉴, `:185-188`). 느린 연결에서 검색어를 넣고 ↓ 로 메뉴 항목을 골라 둔 사용자는 앱 응답이 도착하는 순간 하이라이트가 다른 항목으로 옮겨가고, 그 타이밍에 Enter 를 누르면 의도하지 않은 앱 상세로 이동한다. 선택을 항목 id 로 추적하면 결과가 늘어나도 고른 항목이 그대로 남고, 결과가 줄어 `active` 가 범위를 벗어나 Enter 가 아무 일도 하지 않던 죽은 상태(`go(results[active])` 가 undefined, `:230`)도 함께 사라진다.
- 수용 기준:
  1) 검색어를 넣고 앱 응답이 아직 안 온 상태에서 ↓ 로 메뉴 항목을 고른 뒤 앱 응답이 도착하면, 같은 메뉴 항목이 계속 `aria-selected="true"` 로 남고 Enter 가 그 메뉴 경로로 이동한다(이동 확인은 `rememberRecent` 가 localStorage `appstore.recentDestinations` 첫 항목에 그 id 를 쓰는 것으로 관측 가능 — `:59-86`).
  2) 검색어가 바뀌면 선택이 지금처럼 목록 첫 항목으로 돌아간다(`:204` 의 리셋 동작 보존). 빈 검색어의 "최근 이동" + 메뉴 목록, 앱 결과를 메뉴보다 앞에 두는 순서, ↑↓ 순환(양 끝에서 감싸기), `onMouseMove` 로 선택 이동, `aria-activedescendant` 가 현재 선택 항목의 `id` 를 가리키는 것 모두 그대로.
  3) 새 Vitest 테스트가 수정 전에는 실패하고(앱 응답 도착 후 하이라이트가 인접 항목으로 밀림) 수정 후 통과한다. 수정만 되돌리면 그 테스트가 다시 실패하는 것까지 확인한다.
  4) 기존 Vitest 전체와 lint·build 가 통과한다. 프로덕션 파일은 `command-palette.tsx` 한 개만 바꾼다.
- 건드릴 파일:
  - `web/src/features/navigation/command-palette.tsx:109,204,215-232,258-260,279-281` — `active` 인덱스 상태를 선택 항목 id 상태(예: `activeId: string | null`)로 바꾸고, 렌더·키보드 처리에서 쓰는 인덱스는 `results.findIndex((d) => d.id === activeId)` 가 -1 이면 0 으로 떨어지게 파생시킬 것. ↑↓ 는 파생 인덱스 ± 1 을 `results.length` 로 감싼 뒤 그 위치 항목의 id 를 저장. `onMouseMove` 는 `destination.id` 를 저장. 검색어 변경 effect 는 `setActiveId(null)`(= 첫 항목).
  - `web/src/features/navigation/command-palette.test.tsx` — 회귀 테스트 1건 추가. 기존 파일(88줄)의 `renderPalette` 헬퍼는 렌더 직후 `client.isFetching()` 이 0 이 될 때까지 기다리므로, 앱 응답을 붙잡는 새 테스트에서는 **입력 전에** 그 대기가 끝나게 두고 타이핑 뒤에는 그 헬퍼를 다시 쓰지 말 것.
- 테스트 설계(익명 세션이면 `canManage` 가 false 여서 `app-admin:` 항목이 안 생겨 단순하다):
  - 검색어 `앱` 은 `matches()`(label+hint+to 를 이어 부분일치) 기준으로 스토어 메뉴 중 `전체 앱`(`/apps`)·`MCP 앱`(`/apps?mcp=true`) 두 개와 맞는다(`nav-items.ts:43-54` 확인). 앱 응답은 `/api/v1/apps?...`(`api.ts:223-231`, 경로는 `/api/v1/public/apps` 가 **아님**) 을 붙잡아 두었다가 이름 하나짜리 페이지로 풀어 준다.
  - 절차: 팔레트 렌더 → 입력에 `앱` 타이핑(`@testing-library/user-event` 사용 가능, devDependency 에 있음) → 앱 요청이 붙잡힌 동안 ↓ 1회 → 두 번째 항목(`MCP 앱`)이 선택된 것 확인 → 앱 응답 풀기 → 선택이 여전히 `MCP 앱` 임을 단언(수정 전 실패: `전체 앱` 로 밀림) → Enter → localStorage 첫 항목 id 가 `menu:/apps?mcp=true`.
- 검증 명령 (워크트리에 `web/node_modules` 가 없으므로 첫 단계로 설치 필요, 복합 bash 는 승인에 막힐 수 있으니 한 줄씩):
  - `npm --prefix web ci --no-audit --no-fund`
  - `npm --prefix web test` — 먼저 수정 전에 한 번 돌려 **기준선 개수를 직접 확인**할 것(직전 회차 기록은 82건이지만 이번 정찰은 실행하지 못했다 — 미확인. `it(` grep 은 68건이라 grep 수를 기준선으로 쓰지 말 것)
  - `npm --prefix web run lint`, `(cd web && npx prettier --check src/features/navigation/command-palette.tsx src/features/navigation/command-palette.test.tsx)`
  - `npm --prefix web run build`, `./scripts/check-offline-assets.sh web/dist`
  - 선택: `CI=true npm --prefix web run test:e2e`(Chromium 설치 필요, 오래 걸림). 이 과제는 네트워크 타이밍 의존이라 E2E 로 red 를 잡기 어렵다 — Vitest 로 증명하고 E2E 는 회귀 확인용으로만.
- 위험과 피할 것:
  - 결과 **순서** 를 바꾸는 방식(앱 결과를 메뉴 뒤로 보내기)으로 해결하지 말 것 — 보이는 랭킹이 바뀌는 제품 결정이고 수용 기준 2 를 깬다. 디바운스·`placeholderData` 도 쓰지 말 것: 타이머는 테스트를 불안정하게 만들고, 이전 검색어의 앱 결과를 남기는 것은 더 나쁜 UX 다.
  - `queryKey ["command-apps", query]` 와 `enabled`·`staleTime` 은 건드리지 말 것(`:115-120`).
  - `public-pages.tsx:361` 의 `${apps.data.total}개 앱` 과 `:410` 의 페이지 nav 는 2026-09-23 회차(`0898f7a`, review-pending)가 이미 고치려던 자리다. 이번 정찰에서 main 에 여전히 없음을 확인했지만 **중복 제출이므로 손대지 말 것.**
  - 보호 경로(`internal/auth`·`migrations`·`.github/workflows`·`internal/webui/dist`)와 공개 API 계약은 무관하니 건드리지 말 것.
  - 손으로 만든 대역 대신 지금 테스트처럼 프로덕션 `AuthProvider`·`api.apps` 를 통과하는 fetch 스텁으로 검증할 것.
- 차선 후보: `/favorites` 의 검색·정렬이 즐겨찾기가 아니라 전체 카탈로그를 대상으로 동작해 검색어를 넣으면 즐겨찾기가 사라진 것처럼 보이는 문제(가치 2 / 위험 2 / 작업량 M, `public-pages.tsx:226-242` 의 서버 쿼리 후 slug 필터). 다만 100개 초과 설치 문제와 뿌리가 같아 한 회차로 깔끔히 끝나지 않을 수 있으니, 1순위가 성립하지 않을 때만.
