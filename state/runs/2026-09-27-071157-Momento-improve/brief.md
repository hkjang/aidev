- 과제: DataTable 이 clamp 하지 않은 page 로 slice 해 행도 Empty 도 없는 빈 표를 보여주는 것을 고친다 (가치 3 / 위험 1 / 작업량 S)
- 왜: `DataTable.tsx:87` 이 raw `page` 로 `filtered.slice(page*pageSize, …)` 를 하는데, 표시용 페이지 번호만 `209-212행`에서 `Math.min(page, ceil(len/pageSize)-1)` 로 clamp 하고 slice 는 clamp 하지 않는다. 그래서 뒤쪽 페이지에 있다가 결과가 줄면 `paged` 가 빈 배열이 되고, `194행`의 Empty 는 `!filtered.length` 일 때만 나오므로 **행도 안내문도 없는 빈 표**가 나온다 — 검색을 치는 순간(`86행 useEffect(setPage(0))` 는 렌더·페인트 뒤에 실행되므로 한 프레임) 깜빡이고, `rows.length` 가 그대로인 채 내용만 바뀐 재조회(기간 변경 등)에서 검색어가 걸려 있으면 effect 자체가 안 돌아 **빠져나올 수 없는 상태**가 된다(결과가 10개 이하면 `205행` 조건 때문에 페이지 이동 컨트롤도 사라진다). DataTable 은 콘솔의 거의 모든 표가 쓰므로(`web/src` 안 `<DataTable` 사용 65곳) 한 곳을 고치면 전부 나아진다.
- 수용 기준:
  1) 필터 결과가 현재 페이지보다 적어지면 그 렌더에서 곧바로 **마지막 유효 페이지의 행**이 보인다(빈 표 아님). 즉 `paged` 와 `TablePagination` 의 `page` 가 같은 clamp 값을 쓴다.
  2) `rows` 의 길이는 그대로인 채 내용만 바뀌어 검색 일치가 줄어드는 경우(예: 100행 → 같은 길이 100행, 일치 80 → 5)에도 표가 비지 않는다. `useEffect(…, [query, rows.length])` 의 의존성은 **건드리지 않고** clamp 만으로 해결할 것.
  3) node:test 가 순수 함수로 이를 증명한다: page 가 범위 안일 때는 그대로, 범위를 넘으면 마지막 페이지로 내려가고, total=0 이면 0, pageSize 보다 total 이 작으면 0. 그리고 "clamp 된 page 로 slice 하면 total>0 인 어떤 (page, pageSize, total) 조합에서도 결과가 비지 않는다"를 순회로 확인해, clamp 를 빼면(raw page 로 되돌리면) 실패하는 것을 실제로 확인할 것.
- 건드릴 파일 (3개):
  - `web/src/components/tablePaging.ts` (신규) — `clampPage(page, pageSize, total): number` 하나. 이미 같은 자리에 `csvExport.ts` 라는 컴포넌트용 순수 모듈 선례가 있고 `web/test/csvExport.test.mjs` 가 `"../src/components/csvExport.ts"` 로 **확장자까지 적어** import 한다. 같은 방식을 따를 것(`.ts` 확장자 누락은 이 저장소에서 실제로 걸린 적 있음). pageSize<=0 같은 방어는 넣되 주석으로 왜 0 을 돌려주는지 남길 것.
  - `web/src/components/DataTable.tsx` — `87행` 위에 `const safePage = clampPage(page, pageSize, filtered.length)` 를 두고, `87행` slice 와 `209-212행` `TablePagination.page` 가 **둘 다** `safePage` 를 읽게 한다. `103행`의 기본 rowKey fallback `` `${page}-${index}` `` 도 같은 값을 쓰도록 `safePage` 로 바꿀 것(같은 값을 두 곳이 다르게 읽지 않도록). `86행 useEffect`, `205행 filtered.length > 10` 조건, `194행 Empty` 분기, `onPageChange`/`onRowsPerPageChange` 는 그대로 둔다.
  - `web/test/tablePaging.test.mjs` (신규) — 위 수용 기준 3.
- 검증 명령 (worktree 에 `web/node_modules` 가 없으므로 `npm ci` 선행):
  ```
  cd web && npm ci && npm run lint && npm test && npm run build
  ```
  (`npm test` = `node --test test/*.test.mjs`, 현재 main 기준 115 통과. `npm run build` = `tsc -b && vite build`.)
  Go 쪽은 이 변경이 닿지 않으므로 돌리지 않아도 된다.
  더 강한 증거를 원하면 이전 두 회차가 한 대로 `npm run build` 산출물(`web/dist`)을 임시 서버에 올려 headless Chrome(puppeteer-core, `scripts/guide` 의존성 재사용)으로 실제 DOM 을 확인할 것 — 행이 25개 넘는 표에서 2~3페이지로 간 뒤 검색어를 쳤을 때 빈 표가 안 보이는지. **미확인:** 이 worktree 에서 puppeteer-core/Chrome 이 실제로 되는지는 이번 정찰에서 확인하지 않았다(node_modules 없음, 테스트 미실행).
- 위험과 피할 것:
  - `npx prettier --check` 를 게이트로 쓰지 말 것 — web 에 prettier 의존성·스크립트가 없고 CI 에도 없으며, npx 최신판 기본 설정으로는 main 의 20여 개 파일이 이미 실패한다.
  - `useEffect` 의존성 배열을 "고치려고" 건드리지 말 것(예: `rows` 자체나 `filtered.length` 추가). 렌더 중 setState 루프·불필요한 페이지 리셋을 부르고, clamp 만으로 수용 기준 1·2 가 충족된다. 범위를 넓히지 말고 이번에는 clamp 한 조각만 담을 것.
  - 보류 목록의 "기본 rowKey 가 동명이인에서 중복될 수 있다"(`93-104행`)는 **이번 과제가 아니다**. `103행`은 `page`→`safePage` 치환만 하고 fallback 우선순위 자체는 건드리지 말 것.
  - CSV 내보내기(`149행`)는 `filtered` 전체를 넘기므로 페이지와 무관하다 — 같이 손대지 말 것.
  - `internal/auth`, `internal/httpapi/admin.go`·`server.go`, `internal/database/migrations`, `.github/workflows`, `docs/openapi.yaml` 은 이번 과제에서 열 이유가 없다.
  - 서버는 이 표에 대해 아무 계약도 두지 않으므로 서버·openapi 변경은 없다.
- 차선 후보: 자기 자신의 역할 변경(SELF_ROLE) 제약을 사용자 편집 다이얼로그에 반영 (가치 2 / 위험 1 / S). 서버는 `internal/httpapi/admin.go:978` 에서 400 `SELF_ROLE`("you cannot change your own role")로 막는데 화면은 자기 행의 권한 select 를 그대로 열어 둔다(비밀번호 쪽은 `admin.go:986` 400 `SELF_PASSWORD` 이고 화면이 이미 `edit.id !== user?.id` 로 필드를 숨긴다). 지난 회차가 만든 `web/src/pages/roleScope.ts`(`ROLE_ORDER`/`roleRank`/`canAdministerRole`/`assignableRoles`) 바로 옆에 순수 함수로 얹고 `AdminPage.tsx` 의 `UsersAdmin` 편집 다이얼로그가 그것을 읽게 하면 된다(파일 3개). **미확인:** `AdminPage.tsx` 의 해당 다이얼로그 행 번호와 `edit.id !== user?.id` 분기의 현재 위치는 이번 정찰에서 다시 열어 확인하지 않았고 2026-09-26 회차 기록에 의존한다 — 구현 전에 직접 확인할 것.
