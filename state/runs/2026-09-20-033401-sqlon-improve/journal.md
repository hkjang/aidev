# 회차 노트 2026-09-20-033401-sqlon-improve — sqlon
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:34] base pinned — main@57f99b7
- [러너 03:34] autonomy release — 

## 정찰 노트
- 선택 이유: main(57f99b7) 에는 이전 캠페인 커밋(silent SSO·tracking·handoff·mail·mcp-oauth)이 하나도 없어 그와 얽힌 보류 아이디어는 대상 코드가 없다. `/api/*` 에 사용자 컨텍스트가 안 실려 critical 계획을 REST 로 승인할 수 없는 결함은 main 에서 코드로 재확인했고(withUser 호출 2곳뿐, approve 의 userFrom→nil), 로컬 브랜치 `auto/2026-09-17-0933` 의 a47973b(부모가 main)가 같은 수정을 담고 있어 cherry-pick + adminAudit actor 로 한 세션에 끝난다. docs 3종 채우기는 코드 변경이 없어 차선으로 뒀다.
- 확신 없는 곳: a47973b 의 cherry-pick 을 실제로 해 보지 않았다(부모가 main 이라 깨끗할 것으로 추정). 감사 JSONL 의 파일 경로와 MCP 감사가 쓰는 actor 필드명은 audit.go 에서 확인하지 않았다. `go test ./...` 전체 소요 시간 미확인.
- 구현자가 조심할 것: requireAdmin 호출부가 29곳이라 다 바꾸면 M 으로 부풀 수 있다 — requireDBA(12곳) + adminAudit 만으로 수용 기준을 채우고 requireAdmin 은 남겨도 된다고 과제서에 적었다. 사용자를 컨텍스트에 싣기 시작하면 `canUseProfileID(nil→실제 사용자)` 가 권한 검사를 시작해 기존 테스트가 깨질 수 있으니 깨지면 원인을 노트에 적을 것. 테스트는 실제 mux+세션 쿠키 경로로만(대역·문자열 검사 금지).
- 러너에게: 로컬 `auto/2026-09-1x-*` 브랜치 9개가 main 에 미머지 상태로 쌓여 있다. 이 회차 결과가 a47973b 와 중복되므로 머지 시 둘 중 하나만 택할 것.
- [러너 03:38] scout done — REST 변경 관리·DBA 콘솔의 행위자가 항상 "dba" 로 기록되는 결함 수정 + `adminAudit` 에 actor 필드 추가 (가치 4 

## 구현 노트
- 과제서 지름길대로 a47973b 를 cherry-pick(충돌 없음, requireDBA 가 인증 사용자를 실은 요청을 돌려주고 호출부 12곳 갱신)하고, 그 위에 `adminAudit` 이 컨텍스트 사용자가 있으면 `actor`(이름만) 를 넣도록 했다. 커밋 b14b245(cherry-pick; CHANGELOG 가 CRLF 로 통째 재작성됐던 것을 main 원본 + Unreleased 항목만으로 amend) 와 f727973(adminAudit actor + 감사 JSONL 을 실제로 읽는 테스트 2개).
- 확신 없는 곳·검증 못 한 것: (1) b14b245 의 CHANGELOG 문구가 f727973 의 감사 actor 까지 함께 적혀 있어 커밋 단위로 보면 한 커밋 앞서 있다 — 최종 트리는 정확함. (2) 단독 모드의 actor 부재는 `PUT /api/datasets/glossary`(requireAdmin 경로) 로 검증했고 requireDBA 단독 경로는 프로파일 저장소가 없어 generate 를 못 돌려 따로 단언하지 않았다(단독 모드 requireDBA 는 `return r, s.requireAdmin(...)` 로 원본 r 을 돌려주므로 사용자가 실릴 길이 없음). (3) 실제 DB 가 필요한 execute/rollback 경로의 actor 는 통합 테스트가 없어 미실행 — 코드상 approve 와 같은 `userFrom` 분기.
- 일부러 하지 않은 것: requireAdmin(29곳)은 과제서 허용대로 남김 → 다음 회차. `canUseProfileID` 의 u==nil 신뢰 규칙은 이번 변경의 영향 밖(세 핸들러가 requireDBA 를 거치지 않음)이지만 프로브로 **미인증 GET /api/profile-catalogs/{profile} 가 메타 모드에서 200** 임을 확인해 ideas.json 1순위(5/2/S)로 올렸다 — 게이트 선택이 UI 흐름 판단을 요구해 이번 범위에 넣지 않음.
- 다음 역할이 조심할 것: 감사 파일은 `s.opDir()/audit/audit-YYYYMMDD.jsonl`(테스트는 `auditFilePath` 헬퍼 재사용). `go test ./... -count=1` 전체가 약 5초라 부담 없음. 이 브랜치 결과는 로컬 `auto/2026-09-17-0933` 의 a47973b 와 겹치므로 머지 시 둘 중 하나만(이쪽이 상위집합). 저장소 CHANGELOG.md 는 CRLF/LF 혼재 — 편집 시 바이너리로 다뤄 줄바꿈을 보존할 것.
- [러너 03:44] brief accepted — 채택 — 지름길(cherry-pick)이 실제로 깨끗이 적용됐고 수용 기준 1)·2)·3) 을 모두 프로덕션 배선 테스트로 채웠다; 과제서�
- [러너 03:44] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 확인: admin.go·dbaapi.go 를 main 판으로 되돌려 새 테스트 2개가 실제로 실패(actor="dba" / actor 없음)함을 봤고 HEAD 에서 통과; vet·gofmt(3파일)·`go test ./...` 통과. actor 는 감사 해시 체인(전체 엔트리 해시)에 포함되어 변조 보호됨. requireDBA 가 실은 사용자는 changeexecutor/dbconn 에서 권한 검사에 쓰이지 않아 기존 동작 변화 없음.
- 못 본 것: execute/rollback 의 actor(실제 DB 필요) — approve 와 같은 분기라 읽기로만 확인.
- 승인이지만 릴리즈 전 고칠 것: CHANGELOG 의 "감사 항목(`admin:*`)에도 actor" 는 과장 — 메타 모드에서도 actor 가 붙는 adminAudit 액션은 `admin:change_generate` 하나뿐(requireAdmin 경로 32개는 여전히 없음). '변경 관리 API 감사 항목' 으로 좁혀 쓸 것.
- 다음 회차: 미인증 GET /api/profile-catalogs/{profile} 200(구현자 프로브, 이번 diff 밖) 을 security 1순위로; adminAudit ts(로컬)와 change_execute ts(UTC) 시간대 혼재도 정리 대상.
- 머지 시 a47973b(auto/2026-09-17-0933)와 중복 — 이 브랜치가 상위집합.
- [러너 03:47] review approved — 리뷰 승인 (risk=low)
- [러너 03:47] pr created — https://github.com/hkjang/sqlon/pull/8
- [러너 03:51] ci no-ci — 이 커밋에 검사가 없음 (정책 allow_merge_without_ci 가 없으면 차단)
