- 과제: 사용자 편집 다이얼로그가 자기 자신의 역할 변경(SELF_ROLE)·계정 중지(SELF_DISABLE)를 애초에 제안하지 않게 한다 (가치 3 / 위험 1 / 작업량 S)
- 왜: 서버 `updateUser`(internal/httpapi/admin.go:949 SELF_DISABLE, 977 SELF_ROLE)는 호출자가 자기 계정의 역할을 바꾸거나 자기 계정을 비활성화하면 400 을 돌려주는데, `UsersAdmin` 편집 다이얼로그(web/src/pages/AdminPage.tsx:3416-3438)는 자기 행을 열어도 권한 select 와 「계정 활성화」 체크박스를 그대로 열어 둔다. 그래서 super_admin 이 자기 권한을 viewer 로 고르거나 자기 계정을 중지로 바꾼 뒤 저장해야 영문 메시지("you cannot change your own role")를 만난다. 같은 다이얼로그의 비밀번호 재설정만은 이미 자기 자신이면 숨겨져 있어(3439행 `edit.id !== user?.id`) 세 제약 중 둘만 거울이 빠진 상태다.
- 수용 기준:
  1) 자기 계정 행을 편집할 때 권한 select 가 disabled 이고 왜 바꿀 수 없는지 한국어로 보인다(예: helperText "자기 계정의 권한은 바꿀 수 없습니다"). 다른 계정을 편집할 때는 지금과 글자 그대로 동일하게 동작한다(`assignableRoles` 목록·onChange 그대로).
  2) 자기 계정이 활성 상태일 때 「계정 활성화」 체크박스가 disabled 여서 중지로 바꿀 수 없다. 서버 조건(`in.Active != nil && !*in.Active`)과 같은 방향으로 — **끄는 것만** 막고, 남의 계정 체크박스는 켜고 끄는 것 모두 그대로.
  3) 새 순수 함수 테스트가 (a) 자기 자신 / 남 두 경우에 세 제약(role·active·password)의 허용 여부가 서버 분기와 일치함을 증명하고, (b) 돌연변이 — 자기/남 판정을 뒤집기, 항상 허용 반환 — 에서 실패한다. 기존 roleScope 9건은 그대로 통과.
- 건드릴 파일 (프로덕션 2개 + 테스트 1개):
  - `web/src/pages/roleScope.ts` — 파일 머리 주석이 이미 "서버가 거절할 것을 화면이 제안하지 않게 한다"라고 적고 있으니 그 자리에 순수 함수 하나를 더한다. 예: `selfAccountLimits(callerId: string | undefined, targetId: string)` → `{ canChangeRole: boolean; canDeactivate: boolean; canResetPassword: boolean }`. 전부 `callerId !== undefined && callerId === targetId` 일 때만 false. callerId 가 없으면(로그인 정보 미확정) 서버가 주체 없이는 이 라우트에 닿지 않으므로 제약 없음(true)으로 두고 그 판단을 주석에 적을 것. 역할 서열 함수(`ROLE_ORDER`/`roleRank`/`canAdministerRole`/`assignableRoles`)는 건드리지 말 것.
  - `web/src/pages/AdminPage.tsx` — `UsersAdmin`(3194) 안에서 `edit` 이 있을 때 한 번 `const limits = selfAccountLimits(user?.id, edit.id)` 를 구하고: 권한 `TextField select`(3416-3427)에 `disabled={!limits.canChangeRole}` + helperText, 「계정 활성화」 `Checkbox`(3428-3438)에 `disabled={edit.active && !limits.canDeactivate}`, 그리고 **기존 3439행 조건을 `!edit.oidc && limits.canResetPassword` 로 바꿔** "나인지" 판정이 한 곳에서만 나오게 한다(지금은 id 비교가 인라인으로 흩어져 있다). 편집 버튼의 `canAdministerRole` 게이트(3285)와 생성 다이얼로그(3311~3380)는 손대지 말 것.
  - `web/test/roleScope.test.mjs` — 기존 파일에 새 describe/test 를 덧붙인다. import 는 이 저장소 관례대로 확장자까지: `from "../src/pages/roleScope.ts"`. 기존 파일 방식대로 서버 분기를 옮긴 복제본(`serverRefuses(kind, callerId, targetId, wantActive, newRole, currentRole)`)을 두고 순회로 비교하면 갈라짐을 잡는다.
- 검증 명령:
  - `cd web && npm ci && npm run lint && npm test && npm run build` (worktree 에 node_modules 가 없으므로 `npm ci` 선행. `npm test` = `node --test test/*.test.mjs`, 현재 130건 통과가 기준선. build = `tsc -b && vite build`)
  - 테스트 먼저 실패시키기: 새 함수를 추가하기 전 테스트를 돌려 `ERR_MODULE_NOT_FOUND`/`is not a function` 을 확인하고, 그 뒤 "항상 허용" 스텁으로 값 불일치 실패를 받아낼 것.
  - 프로덕션 배선 확인(2026-09-27·28 회차가 쓴 방법이 그대로 통함): vite dev 로 진짜 `UsersAdmin` 이 아니라 최소 하네스에 진짜 dist 를 띄우기는 users API 모킹이 필요하다. 더 싼 길은 `npm run build` 뒤 dist 를 임시 서버에 올려 `/api/v1/users` 와 `/api/v1/me`(useAuth 가 읽는 것) 를 흉내 내고 headless Chrome(`google-chrome` 있음, `puppeteer-core` 는 /tmp 에 설치)으로 자기 행/남의 행 두 경우의 select·checkbox `disabled` 속성을 읽는 것. 하네스는 커밋 전에 지우고 `git status` 로 3파일만 담긴 것을 확인할 것.
- 위험과 피할 것:
  - **서버를 손대지 말 것.** internal/httpapi/admin.go·internal/auth 가 권한의 정본이고 화면은 거울일 뿐이다(roleScope.ts 머리 주석이 같은 말을 한다). 화면에서 막았다고 서버 검사를 느슨하게 하는 방향은 금지.
  - `useAuth()` 의 `user` 는 확인함: `web/src/contexts/AuthContext.tsx:11-12` 의 `AuthValue.user: User | null`, `User` 는 `web/src/api/client.ts:1-8` 에서 `{ id: string; email; display_name; department; organization_name; role }`. 따라서 `user?.id` 는 `string | undefined` 이고 새 함수의 첫 인자 타입은 `string | undefined` 여야 한다. 세션 미확정 구간에서는 `user === null` 이라 `undefined` 가 들어온다(`refresh()` 가 `/api/v1/me` 로 채운다).
  - MUI `TextField select` 의 `disabled` 와 `helperText` 를 같이 줄 때 helperText 도 회색으로 흐려진다 — 읽히는지 브라우저에서 확인할 것. 흐려서 안 읽히면 helperText 대신 select 위/아래 `Typography variant="caption"` 로 두는 편이 낫다(판단은 구현자에게).
  - `npx prettier --check` 를 게이트로 쓰지 말 것(저장소에 prettier 의존성·CI 단계 없고 main 의 20여 파일이 이미 실패).
  - `edit.active` 체크박스를 자기 계정에서 통째로 disabled 하면 서버보다 더 좁아진다(서버는 끄는 것만 막는다). 기준 2 대로 `edit.active && !limits.canDeactivate` 로 조건을 걸 것.
  - .github/workflows·migrations·docs/openapi.yaml 은 이번 과제에서 건드릴 이유가 없다.
- 차선 후보: DataTable 기본 rowKey 가 동명이인·같은 인덱스에서 중복될 수 있는 것을 고친다 (web/src/components/DataTable.tsx:100-111 이 row.name 등을 먼저 쓰고 없을 때만 `${safePage}-${index}`; 값이 중복되면 React key 가 겹친다). 1순위의 근거는 모두 확인되었으므로(서버 분기·다이얼로그 위치·`User.id`) 차선으로 넘어갈 이유는 예상 못 한 충돌이 있을 때뿐이다.
