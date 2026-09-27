# 회차 노트 2026-09-27-000251-weekly-improve — weekly
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:02] base pinned — main@8701146
- [러너 00:02] autonomy release — 

## 정찰 노트
- 고른 이유: 문서가 이미 약속한 화면 요소가 실제로 없는 자리를 찾았습니다 — ADMIN_GUIDE.md:150 은 Confluence 탭이 "실패 수" 를 보여 준다고 적었지만 AdminPage.tsx 에 `pagesFailed` 는 0회 등장하고, 같은 카드의 진단 표는 최신 500건 보관 중 `LIMIT 20` 만 읽으며 잘림 표시가 없습니다. 지난 두 회차에 채택된 "잘렸다고 말한다" 와 같은 모양이라 경로가 이미 검증돼 있고(불리언 + paging-check 통과 전례), 제품 파일 3개·마이그레이션 0개로 끝납니다. 다섯 회차 연속 차선으로 밀린 MCP 문서 정정은 값이 낮아 이번에도 제쳤고, "실패 발송 다시 시도" 는 세 큐의 재시도 계약을 먼저 정해야 해 한 세션에 맞지 않습니다.
- 확신 없는 곳(과제서의 추측): `confluence_sync_state` 의 `system_type='CONFLUENCE'` 행이 마이그레이션에서 자동으로 생기는지 확인하지 않았습니다 — 없으면 핸들러가 500 `QUERY_FAILED` 로 끝나므로 시험이 그 행을 먼저 채워야 합니다. `confluence_test.go` 의 기존 하네스 사용 방식도 열어 보지 않았습니다(파일 존재만 확인). 이번 회차에 테스트를 실행하지 않았습니다.
- 구현자가 조심할 것: 280행의 `errorRows, _ :=` 가 질의 오류를 버리는 것은 일부러 범위 밖으로 뺐습니다 — 시험으로 증명할 수 없으니 계약을 늘리지 마세요. 같은 표에 오류가 아닌 진단(`recordConfluenceNotice`)도 들어가니 화면 문구는 "진단" 으로 쓰고, `pages_failed`(마지막 1회 카운터)와 표(누적 500건)를 같은 숫자처럼 말하지 마세요. 진단 행은 직접 INSERT 하지 말고 `recordConfluenceNotice`/`recordConfluenceError` 로 만드세요.
- 환경: `weekly-test-pg`(pgvector/pgvector:pg16, 0.0.0.0:15434, Up 4 days)가 떠 있어 실제 DB 전체 시험이 가능합니다 — 프로필의 "docker 컨테이너 0개" 기록은 낡아 profile.md 를 새로 썼습니다.
- [러너 00:12] scout timeout — 단계 제한 시간 초과
- [러너 00:12] scout done — Confluence 자동 수집 카드가 실패 수를 보여 주고, 진단 표가 20건에서 잘렸다고 말한다 (가치 3 / 위험 1 / 작�

## 구현 노트
- Confluence 상태 응답이 진단 표의 잘림을 말하게 했습니다: 질의를 `LIMIT confluenceRecentErrorLimit+1`(=21) 로 바꿔 21번째 행의 존재만으로 `recentErrorsTruncated` 를 세우고 그 행은 응답에 넣지 않습니다. COUNT 질의를 더하지 않았고 `total` 도 싣지 않았습니다. 화면에는 `pagesFailed` 칸과 조건부 한 줄을 더했습니다.
- 확신 없는 곳: **수용 기준 3(화면)은 렌더링 시험으로 증명하지 않았습니다** — 이 저장소에 AdminPage 를 렌더하는 시험이 없어 `tsc -b`·`vite build`·JSX 위치 확인까지입니다. 비평가는 AdminPage.tsx 의 `sync-overview` 새 칸과 `최근 동기화 진단` Card 안 표 뒤 한 줄을 눈으로 확인해 주세요.
- 확신 없는 곳: `ORDER BY created_at DESC, id DESC` 로 동률을 깼지만, 같은 밀리초에 21건이 들어가는 경우를 시험이 만들지는 않습니다(시험의 21건은 순차 INSERT 라 created_at 이 대부분 다릅니다).
- 일부러 하지 않은 것: `errorRows, _ :=` 가 질의 오류를 버리는 것(같은 함수)은 범위 밖입니다 — 실패를 재현할 방법이 없어 시험으로 증명할 수 없고, 과제서도 계약을 늘리지 말라고 못박았습니다. ideas.json 에 pending 으로 남겼습니다.
- 일부러 하지 않은 것: `mutation-check`·`authz-check` 는 돌리지 않았습니다(두 회차 연속 출력 없이 timeout, 소스를 제자리에서 고쳐 씁니다). a11y·failstate 는 배포와 브라우저가 필요합니다.
- 다음 역할이 조심할 것: 새 시험은 **실제 DB 가 있어야 돕니다**(`weekly-test-pg`, pgvector/pgvector:pg16, 15434). 진단 행은 `recordConfluenceNotice`/`recordConfluenceError` 로만 만들었고 직접 INSERT 하지 않았습니다 — 고치실 때도 그 경로를 유지해 주세요.
- 다음 역할이 조심할 것: `pages_failed`(마지막 동기화 1회)와 진단 표(누적 500건)는 다른 숫자입니다. 가이드에 그렇게 적었으니 화면 문구를 바꿀 때 둘을 같은 것처럼 말하지 마세요.
- [러너 00:21] brief accepted — 채택 — 과제서가 지목한 `confluence_handlers.go:280` 의 `LIMIT 20`, `AdminPage.tsx` 의 `pagesFailed` 0회 등장, `types.ts:229` 의 미사용 필�
- [러너 00:24] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인: 새 시험을 main 워크트리(detached)로 옮겨 돌려 `recentErrorsTruncated=<nil>, want true` 실패를 직접 봤습니다 — 시험이 바뀐 경로를 정말 지납니다. 실DB 로 `go test ./...` 전부, guard-check --changed(6개), openapi-check(119), paging-check, tsc·vitest(160)·vite build 모두 통과. LIMIT+1 경계는 20건/21건 두 상태를 시험이 다 밟고, 21번째 행은 응답에 새지 않습니다. 인가는 requireRole("ADMIN") 뒤, 마이그레이션 0개라 revert 로 완전 복구됩니다.
- 못 본 것: 화면을 실제로 렌더한 확인은 여전히 없습니다(이 저장소에 AdminPage 렌더 시험이 없음). a11y·failstate·mutation·authz 검사는 이 환경에서 돌리지 않았습니다.
- 승인이어도 남는 우려 ①: `.sync-overview` 가 `repeat(4,1fr)`(styles.css:21)인데 칸이 5개가 됐습니다 — `실패` 타일이 둘째 줄에 혼자 내려앉습니다. 보기 문제라 차단하지 않았으나 이 카드를 다시 만지면 `repeat(5,1fr)` 로 고치세요.
- 승인이어도 남는 우려 ②: 20 이 Go 상수·화면 문구·ADMIN_GUIDE 세 곳에 따로 박혀 묶이지 않았고, `pages_failed` 는 RUNNING 중 0 을 보여 줍니다(시작에 리셋, 완료에만 기록). 릴리즈 노트에 '실패 칸은 마지막 1회' 를 한 줄 남기면 좋겠습니다.
- 원장에 `- 실패 재현:` 줄이 없습니다 — 위에 적은 실패 출력이 그 자리를 대신합니다.
- [러너 00:29] review approved — 리뷰 승인 (risk=low)
- [러너 00:29] pr created — https://github.com/hkjang/weekly/pull/24
- [러너 00:54] ci passed — 검사 1개 모두 success
- [러너 00:54] merge done — e141627
- [러너 01:25] release published — v0.310.0
- [러너 01:27] assets verified — v0.310.0 자산 1개 (이전 v0.309.0: 1)
