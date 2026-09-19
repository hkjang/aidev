# 회차 노트 2026-09-20-033356-seaton-improve — seaton
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:33] base pinned — main@3f3c8ab
- [러너 03:33] autonomy release — 

## 정찰 노트
- 고른 이유: main(3f3c8ab)에 메일·MCP OAuth 두 브랜치가 아직 미머지라 settings/SettingsPage/seats/auth.authenticate 를 만지는 후보는 충돌 위험이 커서 뺐고, self_demotion 은 09-17 의 self_deactivation 과 같은 틀(validateUserPatch·users.ts·admin.spec)에 케이스 하나 얹는 일이라 45분 안에 e2e 까지 끝난다. 두 브랜치 diff 로 이 자리가 안 겹치는 것을 확인했다.
- 확신 없는 곳: MUI Select 의 disabled 를 Playwright 가 어느 role(combobox div vs 숨은 input)로 잡는지, aria-label 을 어디에 붙여야 `getByRole("combobox",{name})` 이 맞는지는 추측 — 실제 렌더로 확인할 것. vitest 는 이 세션에서 npm 권한이 없어 돌리지 못했다(go test 는 통과).
- 조심할 것: 자기에게 system_admin 을 다시 보내는 것과 role 없는 자기 PATCH(메일 주소 편집)는 통과해야 기존 e2e 가 안 깨진다. openapi.go/API_AND_MCP.md 는 기존 줄만 고치고 PDF 는 굽지 말 것.
- [러너 03:37] scout done — 사용자 권한 화면에서 자기 계정의 권한 강등 막기 (`400 self_demotion`) (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- 바꾼 것: `validateUserPatch` 에 `targetID==actorID && role!=system_admin → 400 self_demotion`, UsersPage 자기 행 권한 Select 를 `canChangeRole`(lib/users.ts) 로 disabled + Tooltip, aria-label `<이름> 권한`. 문서 3곳 기존 줄만·HTML 재생성·admin-users.png 교체. 커밋 1a64524. 이유: v1.4.4 에서 관리자가 자기 role 을 낮추면 PATCH 204 뒤 바로 GET /users 403(curl 로 재현) — 되돌릴 길이 없음.
- 검증: go test ./... (TestValidateUserPatch 14건)·vet·gofmt, tsc, vitest 102건, docker 이미지 + postgres:16 실서버로 Playwright 47건 전부 통과. 새 e2e 단언은 v1.4.4 이미지에 돌리면 실패함(combobox 없음)을 확인했고, curl 로 자기 seat_manager 400·system_admin 204·email/active 만 204·GET /users 여전히 200 을 봤다.
- 확신 없는 곳: MUI Select 의 disabled 는 combobox div 의 `aria-disabled="true"` 로 확인했고 `toBeDisabled()` 는 쓰지 않았다(div 라 disabled 속성이 없음). Tooltip 이 활성 행에서 title="" 로 렌더되는 것은 MUI 가 빈 title 이면 툴팁을 안 띄우는 동작에 기댄 것 — 화면 캡처에선 문제 없었다. 실제 Keycloak 은 없어 SSO 관리자 시나리오는 미확인(로직은 actor/target id 비교뿐이라 source 와 무관).
- 일부러 안 한 것: ADMIN_GUIDE.pdf 굽기(미머지 브랜치와 이진 충돌), openapi/API_AND_MCP 에 새 줄 삽입(기존 줄만 수정), `authenticate` 근처 수정(OAuth 브랜치 자리). 없는 id 에 role 만 보내면 204 인 기존 동작(Exec 영향 행 미확인)은 범위 밖으로 두고 ideas 에 적었다.
- 다음 역할이 조심할 것: Playwright 는 실제 docker 이미지 + PostgreSQL 이 있어야 돈다. 호스트 8080 이 막혀 있으면 브리지 + `-p 127.0.0.1:18080:8080 --add-host=host.docker.internal:host-gateway` 로 띄우고 tracking.spec 은 `E2E_COLLECTOR_HOST=host.docker.internal` 이 없으면 1건 실패한다(환경 문제, 코드 무관). guide-shots.mjs 는 PNG 22장을 전부 덮어쓰므로 admin-users.png 만 남기고 나머지는 git checkout 으로 되돌렸다.
- [러너 03:47] brief accepted — 채택 — 과제서의 근거(라우트 그룹·SSO upsert 의 role 보존 조건·미머지 브랜치와의 비겹침)가 코드와 모두 맞았고 수용 �
- [러너 03:47] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: diff 전체(auth.go·auth_test.go·openapi.go·users.ts/test·UsersPage·admin.spec·문서 3곳) 읽고, auth.go 를 main 으로 되돌려 TestValidateUserPatch 가 실패하는지(실패함 → 테스트가 변경을 진짜 검증), go test/vet·vitest users.test.ts·tsc -b 통과를 직접 돌렸다. MUI SelectInput 소스에서 combobox div 의 aria-disabled/aria-label 을 확인해 구현자의 "확신 없는 곳"은 해소.
- 못 본 것: Playwright 는 docker 이미지를 다시 굽지 않아 재실행하지 않음 — 구현 노트의 47건 통과에 기댐. 실제 Keycloak SSO 관리자 시나리오도 미확인(로직은 id 비교뿐이라 source 무관).
- 판정 approve, risk low, blocking 없음. 권한을 좁히는 변경이라 보안·법무 소견 없음.
- 릴리즈 노트: `PATCH /users/{id}` 에 `400 self_demotion` 추가(자기 role 을 system_admin 이 아닌 값으로). 화면은 자기 행 권한 Select 비활성.
- 다음 회차: admin-users.png 는 이진 교체라 미머지 브랜치가 같은 PNG 를 굽으면 충돌 — 릴리즈 문서 정비 회차에서 PDF 와 함께 한 번에 굽는 편이 안전. A 가 마지막 다른 관리자 B 를 강등하는 것은 여전히 허용(범위 밖).
- [러너 03:49] review approved — 리뷰 승인 (risk=low)
- [러너 03:49] pr created — https://github.com/hkjang/seaton/pull/31
