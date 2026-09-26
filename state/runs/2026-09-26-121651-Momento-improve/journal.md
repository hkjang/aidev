# 회차 노트 2026-09-26-121651-Momento-improve — Momento
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:16] base pinned — main@715e60b
- [러너 12:16] autonomy release — 

## 정찰 노트
- 왜 이것: 최근 세 회차가 스킬 부재로 변경 없이 끝났으니 확실히 끝나는 web 전용 S 과제가 필요했다. 서버 403 근거(admin.go 899/956/973행)와 화면 하드코딩(AdminPage.tsx 3278/3336/3410행)을 직접 열어 확인했고, 순수 모듈 + node:test 라는 이 저장소의 기존 패턴에 그대로 얹힌다. MCP·CI·guide 캡처 후보는 외부 의존(Keycloak·Postgres·워크플로)이라 제쳤다.
- 추측으로 적은 것: Tooltip 이 AdminPage 의 @mui/material import 에 이미 있는지 확인하지 않았다(없으면 한 줄 추가). 차선 후보의 "빈 표" 는 코드 읽기로 추론한 것이고 브라우저로 보지는 않았다.
- 돌리지 못한 것: web/node_modules 가 없어 npm ci/lint/test/build 를 실행하지 않았다. 구현자는 npm ci 부터 시작할 것.
- 조심할 것: 서버 검사(internal/auth, admin.go)는 권한의 정본이니 손대지 말 것. 자기 자신 제약(SELF_ROLE/SELF_PASSWORD)의 화면 반영은 같은 다이얼로그라 끌려들기 쉬운데 범위 밖이며 ideas.json 에 별 항목으로 남겼다.
- [러너 12:21] scout done — 사용자 관리 화면이 호출자보다 높은 역할의 계정을 편집하게 두지 않고, 역할 select 도 호출자 역할까지�

## 구현 노트
- 서버가 403 ROLE_ABOVE_CALLER 로 막는 조작을 화면이 먼저 내주지 않게 했다: 순수 모듈 `web/src/pages/roleScope.ts` 를 만들고 `UsersAdmin` 의 편집 버튼(비활성+Tooltip)과 두 권한 select(`assignableRoles`)가 그것을 읽는다. 서버·internal/auth 는 무변경.
- 과제서와 의도적으로 다른 점 한 곳: `canAdministerRole` 은 **호출자** 역할을 모를 때(빈 문자열 등) 서버를 그대로 흉내 내지 않고 false 로 닫는다. 서버 `RoleAbove("", "")` 는 false(=허용)지만 화면이 편집을 안 내주는 쪽이 무해하다. 대상 역할이 모르는 값이면 서버처럼 등급 0 으로 다룬다. 수용 기준의 5×5 순회는 그대로 만족한다(알려진 역할은 등급≥1).
- 확신 없는 곳: (a) 호출자 역할이 알려지지 않은 상태에서 「사용자 추가」를 열면 select 옵션이 0개인데 `form.role` 기본값은 "viewer" 라 MUI out-of-range 경고가 날 수 있다 — 실제로는 users 섹션이 orgOnly 라 도달 불가라고 보고 방치했다(미검증). (b) 표 정렬: Tooltip 의 `<span>` 이 들어간 행과 안 들어간 행의 폭 차이를 스크린샷으로만 봤고 픽셀 단위로 비교하지는 않았다.
- 일부러 하지 않은 것: 자기 자신 제약(SELF_ROLE·SELF_PASSWORD)의 화면 반영 — 범위 밖이라 ideas.json 에 별 항목으로 남겼다. 차선 후보(DataTable page clamp)도 손대지 않았다. docs/openapi.yaml·가이드는 계약이 안 바뀌어 미변경.
- 검증: node:test 9건을 먼저 실패시키고 통과시킨 뒤 돌연변이 3종(항상 true / 한 단계 위 허용 / ROLE_ORDER 뒤집기)으로 각각 5·3·4건 실패를 확인했다(복원 후 9/9). `npm ci && npm run lint && npm test`(115 통과) `&& npm run build` 통과.
- 다음 역할이 조심할 것: 화면 DOM 확인은 저장소 밖 임시 하네스(/tmp/rolecheck, dist + API 를 흉내 낸 stub 서버 + headless Chrome)로 했고 커밋하지 않았다 — 재현하려면 다시 만들어야 한다. Go 쪽 변경이 없어 `go test` 는 돌리지 않았다(Postgres 없음). `npx prettier` 는 이 저장소의 게이트가 아니므로 쓰지 말 것.
- [러너 12:28] brief accepted — 채택 — 근거로 든 행 번호(auth.go:229·239, admin.go:899·956·973, AdminPage.tsx 3278·3336·3410)와 `useAuth` 위치가 모두 현재 코드와 일
- [러너 12:28] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: 실제 대상은 715e60b...HEAD 3개 파일 185줄(로컬 main 은 d466ee9 로 낡아 diff 가 머지된 20여 커밋을 섞어 보여준다 — SHA 로 고정해 읽었다). auth.go:229·239 와 admin.go:899·956·973 을 열어 ROLE_ORDER·canAdministerRole·assignableRoles 가 서버 RoleAbove 의 정확한 반대임을 대조했고, Go 변경이 0줄이라 서버 경계는 그대로다.
- 구현자가 의심한 (a) 옵션 0개 → 도달 불가로 확정: users 는 orgOnly 이고 AdminPage.tsx:181-192 가 user null·analyst·viewer 를 걸러내므로 호출자 등급은 항상 4·5 다. (b) 표 정렬은 <span> 이 패딩 없는 인라인이라 문제 없다고 봤고 픽셀 비교는 안 했다.
- 돌린 것: npm run lint 무경고 / npm test 115건 전부 통과(roleScope 9건) / npm run build 성공, 이후 git status 깨끗. web/node_modules 는 이미 있었다(프로필의 "없음" 기록은 낡았으니 갱신할 것). Go 는 변경이 없어 미실행.
- 승인이어도 남는 우려: ROLE_ORDER 와 테스트의 SERVER_RANK 가 둘 다 web 트리 안의 손복사본이라 테스트는 Go 정본과의 일치를 보지 못한다. organization_admin 위·super_admin 아래 역할이 추가되면 TS 는 등급 0 으로 보아 편집 버튼을 다시 내주고 테스트는 통과한다(실패 방향은 오늘 동작으로의 회귀라 무해). 역할을 늘리는 회차는 roleScope.ts 를 같이 고쳐야 한다 — 릴리즈 노트에 넣을 값이 있다.
- 판정: approve, risk low, blocking 없음(새 개인정보·비밀값·권한 확대 없음). 자기 자신 제약(SELF_ROLE·SELF_PASSWORD)의 화면 반영은 여전히 미해결이며 ideas.json 에 남아 있다.
- [러너 12:32] review approved — 리뷰 승인 (risk=low)
- [러너 12:32] pr created — https://github.com/hkjang/Momento/pull/17
- [러너 12:38] ci passed — 검사 1개 모두 success
- [러너 12:38] merge done — 0d20425
- [러너 12:48] release published — v0.34.45
- [러너 12:50] assets verified — v0.34.45 자산 2개 (이전 v0.34.44: 2)
