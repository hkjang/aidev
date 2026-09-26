- 과제: 사용자 관리 화면이 호출자보다 높은 역할의 계정을 편집하게 두지 않고, 역할 select 도 호출자 역할까지만 나열한다 (가치 3 / 위험 1 / 작업량 S)
- 왜: 서버는 `internal/httpapi/admin.go` 의 `createUser`(899행)·`updateUser`(956행, 973행)에서 `auth.RoleAbove` 로 403 ROLE_ABOVE_CALLER 를 돌려주는데, 화면(`web/src/pages/AdminPage.tsx` `UsersAdmin`)은 모든 행에 편집 버튼을 주고 두 다이얼로그 모두 다섯 역할을 하드코딩해 보여 준다(3336-3345행 추가, 3410-3419행 편집). 그래서 organization_admin 이 super_admin 계정을 열거나 super_admin 역할을 고르면 저장 버튼을 누른 뒤에야 영문 오류를 만난다. 화면이 서버와 같은 서열 규칙을 쓰면 실패할 조작이 애초에 나오지 않는다.
- 수용 기준:
  1) organization_admin 으로 로그인해 보면 super_admin 행의 편집 버튼이 비활성이고, 왜 안 되는지 알려 주는 한국어 툴팁이 붙는다(같은 등급·아래 등급 행은 그대로 편집된다).
  2) 「사용자 추가」와 「사용자 권한 편집」의 권한 select 에 호출자 역할보다 높은 값이 하나도 나오지 않는다(super_admin 호출자에게는 다섯 개 모두 그대로).
  3) 새 순수 모듈의 node:test 가 서열 함수와 목록 함수를 증명한다: 역할 5개 × 호출자 5개 조합에서 `canAdministerRole` 이 서버의 `auth.RoleAbove` 와 정확히 반대(즉 `RoleAbove(target, caller)` 일 때만 false), `assignableRoles` 가 항상 오름차순이고 호출자 역할을 포함하며 그보다 높은 값을 포함하지 않음, 모르는/빈 역할 문자열은 빈 목록과 false(서버 `roleRank` 의 기본값 0 과 같은 취급). 모듈을 되돌리면 이 테스트가 실패해야 한다.
- 건드릴 파일:
  - 새 `web/src/pages/roleScope.ts` — `ROLE_ORDER`(["viewer","analyst","workspace_admin","organization_admin","super_admin"], `internal/auth/auth.go:229` 의 `roleRank` 순서와 같게), `roleRank(role)`(모르는 값 0), `canAdministerRole(callerRole, targetRole)`, `assignableRoles(callerRole)`. 주석에 `auth.RoleAbove`(internal/auth/auth.go:239)가 정본임을 적을 것.
  - 새 `web/test/roleScope.test.mjs` — `import { … } from "../src/pages/roleScope.ts";` (확장자 명시. `web/test/passwordRule.test.mjs` 와 같은 형태, node:test)
  - `web/src/pages/AdminPage.tsx` `UsersAdmin`(3192행~) — 이미 있는 `const { user } = useAuth();`(3193행)을 쓴다. (a) 3278-3288행 편집 버튼: `canAdministerRole(user?.role ?? "", String(row.role))` 이 false 면 `disabled` 로 두고 MUI `Tooltip` 으로 `<span>` 을 감싸 "내 권한보다 높은 계정은 편집할 수 없습니다" 를 보여 준다(비활성 버튼은 title 이 뜨지 않으므로 Tooltip+span 이 필요하다. 감싸는 것이 표 정렬을 망치면 버튼을 렌더하지 않는 쪽도 허용). (b) 3336-3345행과 3410-3419행의 하드코딩된 배열 두 개를 `assignableRoles(user?.role ?? "")` 호출로 바꾼다. Tooltip 이 아직 import 되어 있지 않으면 @mui/material import 에 더한다.
- 검증 명령: `cd web && npm ci && npm run lint && npm test && npm run build`
  (node_modules 가 없는 상태이므로 `npm ci` 가 먼저다. Go 쪽 변경이 없으므로 `go test` 는 필요 없다.)
- 위험과 피할 것:
  - 서버(`internal/httpapi/admin.go`)·`internal/auth` 는 건드리지 말 것. 서버 검사가 정본이고 이미 옳다. 이 과제는 화면을 서버에 맞추는 것이며, 화면 게이팅을 보안 경계로 여겨 서버 검사를 느슨하게 하는 변경은 금지.
  - `internal/database/migrations`, `.github/workflows`, `docs/openapi.yaml` 은 계약이 바뀌지 않으므로 손대지 않는다.
  - 자기 자신의 역할 변경(서버 400 SELF_ROLE)·자기 비밀번호(400 SELF_PASSWORD)의 화면 반영은 **범위 밖**이다. 같은 다이얼로그라 끌려들어가기 쉬우니 별도 회차로 남길 것.
  - 순수 모듈 import 에 `.ts` 확장자를 빼면 node:test 가 모듈을 찾지 못한다(2026-09-20 회차에서 걸린 자리. `tsconfig.app.json` 의 `allowImportingTsExtensions` 는 이미 켜져 있다).
  - `npx prettier` 는 이 저장소의 게이트가 아니다(web 에 prettier 의존성·스크립트 없음, CI 에도 없음). 최신 prettier 기본 설정으로 돌리면 main 의 파일들이 이미 실패하므로 포맷 전체를 손대지 말 것.
- 차선 후보: DataTable 의 페이지 범위 초과 — `web/src/components/DataTable.tsx:87` 이 clamp 하지 않은 `page` 로 slice 하는데 209행은 표시만 clamp 하므로, 검색으로 결과가 줄어든 첫 렌더에서 행도 없고 Empty 도 없는 빈 표가 나온다(86행 useEffect 는 렌더 뒤에 page 를 0 으로 되돌린다). 87·209행이 같은 clamp 값을 쓰게 하고 순수 함수로 뽑아 테스트한다.
