# 회차 노트 2026-09-23-210455-ai-admin-improve — ai-admin
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:05] base pinned — main@a7b64fe
- [러너 21:05] autonomy release — 

## 정찰 노트
- 역할 이름 검증을 고른 이유: 채택된 최근 4회차와 같은 유형(요청 검증 ↔ DB 계약 불일치, users.go, 통합 테스트 재현 가능)이면서 보호 경로를 하나도 열지 않는다. loadGrants 정렬은 internal/auth, decideApproval은 workflow.go라 위험 구역이어서 각각 3·2순위로 내렸고, 레거시 시드 SQL(M)은 45분과 Pages CI 44개 캡처 요건에 걸려 제외했다.
- 확신 없는 곳: 121자 → 500이 나는 것은 스키마(varchar(120))와 코드 경로로 추론했을 뿐 실제 DB로 재현하지 않았다(정찰은 DB를 띄우지 않음). `openapi.go`에 역할 수정 요청 스키마가 있는지도 확인하지 않았다.
- 구현자가 조심할 것: 검증 블록을 지웠을 때 실제로 500이 나는지 먼저 확인하고, 재현되지 않으면 차선 후보로 갈 것(효과 없는 변경은 반려 사유다).
- `request.Name`은 `*string`이다 — nil(미전송)과 ""(빈 값 전송)을 구분하지 않으면 설명·권한만 바꾸는 기존 호출이 깨진다.
- 이 머신의 포트 55432/55433/55439/15434는 다른 세션의 postgres가 점유 중이므로 새 포트로 전용 DB를 띄울 것.
- [러너 21:09] scout failed — 과제서 없음 — 구현자가 직접 고른다

## 구현 노트
- `updateRole`이 `name`을 검증 없이 `varchar(120)`에 넣어 121자는 500 `role_update_failed`, 빈 값·공백만은 그대로 저장됐다. `updateUser`의 표시 이름 검증과 같은 형태(trim → 빈 값/120 rune 초과면 400 `name_invalid`)를 DB 조회 전에 넣었다.
- 확신 없는 곳: 커밋된 SPA(`internal/ui/dist`)는 재빌드하지 않았고 `npm test`도 돌리지 않았다(웹 소스 무변경). 화면에서 400 `name_invalid`가 어떤 문구로 보이는지는 실제 브라우저로 확인하지 못했다. `openapi.go`에는 역할 수정 요청 본문 스키마가 없어 손대지 않았다.
- 일부러 안 한 것: `description`은 `text` 컬럼이라 상한이 없어 검증을 넣지 않았다(효과 없는 변경 회피). `RolesPage.tsx`에 `maxLength={120}`을 걸면 dist 재빌드까지 필요해 별도 아이디어로 남겼다.
- 다음 역할이 조심할 것: `internal/server/role_name_integration_test.go`는 `TEST_POSTGRES_DSN`이 없으면 SKIP이고 두 스키마를 `DROP SCHEMA ... CASCADE` 하므로 전용 폐기 DB에서만 돌릴 것(이번엔 포트 55444에 `postgres:16-alpine`을 새로 띄웠다 — 55432/55433/55439/15434는 다른 세션이 점유 중).
- 검증: `go test -race -count=1 ./...`(internal/server 91.0s) 전체 PASS, `make lint`, `go build ./...` 통과. red→green과 rune→byte 되돌림 실패를 모두 확인했다.
- [러너 21:14] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 구현자가 "재현 안 함"이라 남긴 자리를 직접 확인했다: 새 postgres(55451, 검토 후 삭제)에서 검증 블록 제거 시 121자 500 `role_update_failed`, 빈 값·공백만 200 저장, trim 미적용까지 red가 나고 HEAD는 6/6 PASS. rune→byte 되돌림도 'exactly the column width'에서 깨진다.
- 전체 `go test -count=1 ./...`를 DSN 걸고 재실행해 전부 PASS(internal/server 14.2s, SKIP 아님), `make lint`·`go build ./...` 통과. 작업 트리 clean 복구 확인.
- 보안·법무 차단 없음: 새 경로·식별자·비밀값·의존성·개인정보 없음, `roles.manage` 인가 그대로, 입력을 좁히기만 한다. revert로 완전히 되돌아온다(마이그레이션 없음).
- 못 본 것: 실제 브라우저 화면, `npm test`/웹 빌드(웹 소스 무변경이라 dist 불일치는 없음), Pages CI.
- 릴리즈 노트가 알아야 할 것: `openapi.go:203`에는 여전히 역할 수정 요청 스키마가 없고 `RolesPage.tsx:73`에 `maxLength={120}`이 없다(121자는 이제 500 대신 400 `name_invalid`). 다음 회차 아이디어로 남길 것.
- [러너 21:17] review approved — 리뷰 승인 (risk=low)
- [러너 21:17] pr created — https://github.com/hkjang/ai-admin/pull/30
- [러너 21:24] ci passed — 검사 2개 모두 success
- [러너 21:24] merge done — c111ad1
- [러너 21:34] release published — v1.2.25
- [러너 21:36] assets verified — v1.2.25 자산 2개 (이전 v1.2.24: 2)
