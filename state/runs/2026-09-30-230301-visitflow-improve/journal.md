# 회차 노트 2026-09-30-230301-visitflow-improve — visitflow
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 23:03] base pinned — main@cee8ae9
- [러너 23:03] autonomy release — 

## 정찰 노트
- 보류 아이디어 중 유일하게 **원인이 코드로 끝까지 확정된** 항목이라 골랐다: `upsertVisitor` 의 phone_hash 단독 조회 → 같은 visitor_id → `visitor_visits` UNIQUE 위반 → `writeVisitError` fallback 500. 로비 응답 역전(M)·가져오기 역전(S)은 브라우저 재현이 아직 없어 제쳤고, 로컬 E2E 진입점(M)은 CI 에 이미 잡이 있어 가치가 겹친다.
- 추측으로 남긴 것: 500 자체는 지난 회차 실측 기록이고 이번 세션에서 DB 를 띄워 재현하지는 않았다(DSN 미실행). 코드 경로만 읽어 확정했으니 구현자는 **수정 전 바이너리로 500 을 먼저 재현**하고 시작할 것. 프런트도 미실행(node_modules 없음) — "vitest 69개" 는 회차 기록에서 가져온 숫자다.
- 조심할 것: `upsertVisitor` 의 조회 키와 `normalizePhone` 은 건드리지 말 것(watchlist 해시·재방문 식별·마스킹이 딸려 있다). 화면과 서버가 하이픈·공백 섞인 같은 번호를 같게 보는지 end-to-end 로 확인할 것.
- 프로필은 1일 전(2026-09-29) 것이고 이번에 읽은 범위와 어긋나는 데가 없어 다시 쓰지 않았다.
- [러너 23:06] scout done — 한 방문 안에 같은 휴대전화를 두 번 넣으면 나는 이유 없는 500 을 화면 사전 검사 + 서버 400 으로 닫기 (가

## 구현 노트
- 무엇을 왜: `createVisitRecord` 의 기존 방문자 루프가 이미 본 `normalizePhone` 을 `firstByPhone` 에 추적해 트랜잭션 시작 전에 `400 duplicate_visitor` 로 답한다(전엔 `visitor_visits` UNIQUE 위반이 fallback 500 `visit_failed` 로 나갔다). 화면은 `visitors.ts` 의 새 순수 함수 `duplicatePhoneErrors`/`duplicatePhoneError` 가 기존 `phoneDigits` 로 같은 경계를 먼저 잡고, 칸 안내·버튼 `disabled`·`submit()` 가드가 그 한 계산만 읽는다. `upsertVisitor` 조회 키·`normalizePhone`·스키마는 읽기만 했다.
- 확신 없는 곳·검증 못 한 것: **MCP 경로는 실행하지 않았다** — `mcp.go:166` 이 방문자 **1명** 인 `VisitInput` 만 만드므로 중복이 구조적으로 불가능해 새 검사가 그 경로에서 도는 것을 확인할 방법이 없었다(코드로만 확인). Go 테스트는 REST 와 현장 등록(`/api/v1/lobby/walk-ins`) 두 경로만 실제로 지난다. 서버 400 본문 문구(`방문자 1 과 방문자 2 의 …`)와 화면 문구(`방문자 2: 방문자 1 과 …`)는 **일부러 다르다** — 화면은 칸 자체가 이미 그 방문자의 것이라 접두를 한 줄 안내에만 붙였다. PDF 가이드는 재생성하지 않았다.
- 일부러 하지 않은 것: `upsertVisitor` 의 조회 키를 "이름+전화" 로 넓히지 않았다(재방문 식별·watchlist·`masked_at`·`frequent_visitors` 까지 의미가 번지는 데이터 변경). 같은 번호를 **의도적으로** 넣는 경우(대표 한 명이 여러 명을 대리)는 지원하지 않는다 — 스키마가 방문 하나에 같은 방문자를 두 번 담을 수 없으므로 화면·서버가 거절하는 것이 현재 계약이다.
- 다음 역할이 조심할 것: 새 Go 테스트 `TestDuplicateVisitorPhoneRejected` 는 **`VISITFLOW_TEST_DSN` 이 있어야 돈다**(없으면 조용히 SKIP 이라 통과처럼 보인다). 이 환경은 5432·8080 이 다른 프로젝트 컨테이너에 잡혀 있어 Postgres 는 15450, 서버는 전용 도커 네트워크 + 18099 로 띄웠고 Playwright 는 번들 chromium 이 없어 `channel: "chrome"` 이 필요하다. 저장소 e2e 중 `admin console` 1개는 **이 변경과 무관하게 실패한다** — 새 DB + 수정 전 번들로도 동일하게 실패하는 것을 확인했고 보류 아이디어로 넘겼다(나머지 9개 통과). `cmd/visitflow/webdist` 는 스텁만 남게 되돌렸다.
- [러너 23:22] brief accepted — 채택 — 지정한 파일 5개·근거·수용 기준 5개가 지금 코드와 정확히 맞았다. 과제서가 미확인으로 남긴 500 을 실제 Postgre
- [러너 23:22] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 판정 approve. 새 Go 통합 테스트를 **main 임시 worktree + HEAD 테스트 파일**로 실제 PostgreSQL 16 에 돌려 수정 전 FAIL(`500 visit_failed`)·수정 후 PASS 를 직접 재현했다 — 원장의 `- 실패 재현:` 은 사실이고 증상도 일치한다. DSN 지정 `go test ./...`(app 47.5s, SKIP 아님)·`go vet`·`gofmt`·`tsc -b`·vitest 79·`npm run build` 모두 통과, 작업 트리 깨끗, webdist 스텁만 남음.
- 코드로 확인: `normalizePhone`(ASCII 숫자만)과 `phoneDigits`(`/[^0-9]/g`)가 같은 규칙, `UNIQUE(visit_id,visitor_id)`(0005_baseline.sql:198) 진단 정확, `upsertVisitor` 조회 키·스키마·인증·권한 미변경. 세 진입점이 모두 `createVisitRecord` 하나를 지나는 것도 grep 으로 확인했다(MCP 는 방문자 1명이라 구조적으로 무해).
- **못 본 것**: 브라우저 재현. `.tsx` 는 Vitest 가 수집하지 않으므로 `disabled`·`submit()` 가드 배선은 읽기로만 확인했다(둘 다 `duplicateMessage` 를 읽는 것 확인). 구현자의 실제 Chromium 변이 검증은 재실행하지 않았다 — 순수 파생값이라 위험은 낮다고 봤다.
- 승인이어도 남는 우려: (1) 중복 검사가 같은 루프의 watchlist 조회보다 앞서 있어, 번호가 겹치면서 *회사*가 watchlist 인 방문자는 403 + `watchlist.match` 감사 대신 400 을 받는다 — 방문 생성은 막히므로 우회는 없고 탐지 신호만 지연된다. (2) Alert 가 `duplicateMessage || blockReason` 이라 `submitBlockReason` 의 '위에서 아래로' 순서 계약을 한 자리 앞지른다(취향 수준). (3) `USER_GUIDE.pdf` 미재생성은 b06dd21 이후 7커밋째인 기존 drift — 릴리즈 노트에서 언급만.
- 다음 회차가 알 것: `TestDuplicateVisitorPhoneRejected` 는 DSN 없으면 조용히 SKIP 한다(CI ci.yml:43 은 설정되어 있어 실제로 돈다). 같은 번호를 의도적으로 두 번 넣는 사용은 이제 명시적 거절이며, 이전에도 500 이었으므로 잃은 동작은 없다.
- [러너 23:27] review approved — 리뷰 승인 (risk=low)
- [러너 23:28] pr created — https://github.com/hkjang/visitflow/pull/30
- [러너 23:32] ci passed — 검사 2개 모두 success
- [러너 23:32] merge done — bc02eb5
- [러너 23:41] release ci-blocked — 릴리즈 커밋 CI: failed — 성공이 아닌 검사: test=failure (태그 보류)
