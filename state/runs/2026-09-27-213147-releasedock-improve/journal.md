# 회차 노트 2026-09-27-213147-releasedock-improve — releasedock
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:31] base pinned — main@0c4a5f8
- [러너 21:31] autonomy release — 

## 정찰 노트
- 고른 이유: 커서 우선순위(Last-Event-ID 대 `?after`)는 v0.5.21 의 수동 재연결과 브라우저 자동 재연결이 **서로 다른 쪽**을 채우는 실제 사용자 경로인데 테스트가 전무하다(기존 3건 모두 `after=0`). 프로덕션 0 파일·테스트 1 파일로 끝나 사람 손을 다시 탈 확률이 가장 낮다.
- 제친 후보: 스트림 한도 429 테스트(동시성 타이밍 위험이 더 큼, 차선으로 남김), SimpleDeployPage error 안내(렌더 테스트 도달 비용이 세 회차째 걸림돌), rAF 배치 setLogs(성능 측정 근거 없음), make vet·MCP OAuth(미머지 브랜치 의존 — 후자 2건은 rejected 로 내렸다).
- 확신 없는 곳: 전체 모드 테스트를 핸들러 직접 호출로 두는 것이 최선인지는 추측이다(저장소 관례를 따랐을 뿐, 라우터 경유가 더 낫다면 그렇게 해도 범위 내). SimpleDeployPage 본문과 migrations 개수는 이번에 다시 열지 않았다 — 미확인.
- 구현자가 조심할 것: DSN 없이 `go test` 는 종료 0 으로 SKIP 하니 반드시 `-v` 로 SKIP 없음을 확인할 것. `strings.Contains(body,"line 1")` 이 `line 10` 에 걸리는 함정과, log id 를 1..10 으로 가정하지 말고 DB 에서 읽는 것. 수용 기준 4 의 되돌림 실험 뒤 프로덕션 코드를 반드시 원복할 것.
- 저장소 상태: 작업 트리 깨끗, 코드 변경 없음. 샌드박스 Bash 가 `${VAR}` 확장을 거부해 검증 명령에서 셸 변수를 뺐다(프로필에 기록).
- [러너 21:36] scout done — SSE 재연결 커서 우선순위(Last-Event-ID 대 `?after` 중 큰 값)의 실제 HTTP 회귀 테스트 (가치 3 / 위험 1 / 작업량 M

## 구현 노트
- 무엇을 왜: 두 SSE 스트림의 재연결 커서 계약(Last-Event-ID 와 `?after` 중 큰 값)을 고정하는 통합 테스트 5건을 `backend/internal/server/simple_stream_cursor_test.go` 에 추가했다. 프로덕션 코드는 한 줄도 바뀌지 않았다(커밋 7b1fea6, 파일 1개).
- 확신 없는 곳: 전체 모드 1건은 저장소 관례대로 `streamReleaseLogs` 를 **직접** 호출하므로 인증·라우팅을 지나지 않는다 — 커서 파싱만 덮는다. 또 클라이언트(웹)가 실제로 그 id 를 이어받는지는 이번에 보지 않았다(서버 계약만 고정). 음수 `Last-Event-ID`("-5") 는 파싱에 성공해 전체 로그를 다시 내려보내는데, 의도인지 확인하지 않았고 테스트도 없다(ideas.json 에 후보로 남김).
- 일부러 안 한 것: 두 파서를 공용 헬퍼로 합치지 않았다 — 계약이 다른 두 경로를 묶지 말라는 과거 교훈. 스트림 한도(3/user) 테스트는 동시성 타이밍 위험 때문에 다음 회차로 미뤘다.
- 다음 역할이 조심할 것: **이 테스트 5건은 `TEST_POSTGRES_DSN` 이 없으면 전부 SKIP 된다**(종료 0 이라 통과로 오해하기 쉬움). 도커 PostgreSQL 16 으로 채워 `-v` 로 SKIP 없음을 확인할 것. 검증에 쓴 컨테이너 `releasedock-pg-2131`(포트 55471)은 이 세션이 띄운 것이니 필요 없으면 지워도 된다.
- [러너 21:42] brief accepted — 채택 — 과제서의 근거(두 핸들러의 max 선택, 기존 3건이 모두 `after=0`, 헤더를 지나는 테스트 부재)가 지금 코드와 그대로
- [러너 21:43] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인함: 돌연변이 양방향을 직접 재현했다 — 헤더 파싱을 죽이면 3건(:143 :158 :222), `queryID > lastID` 를 죽이면 2건(:128 :173) FAIL. 원장의 실패 재현과 줄번호·기대값까지 일치. 프로덕션 2파일은 원복하고 트리 clean 확인. 도커 PG16 으로 SKIP 없이 새 5건+기존 3건 PASS, `go test ./...` 전 패키지 ok, vet·gofmt 깨끗. 판정 approve.
- 못 본 것: 웹(`SimpleRunDetailPage`)이 서버가 준 id 를 실제로 이어받는지 — 서버 계약만 고정됐다. 전체 모드 1건은 `withPermission("releases.read")`(server.go:301)를 지나지 않아 릴리즈 스트림의 인가 회귀는 여전히 미덮임.
- 남는 우려: verify.json 의 `go test ./...` 1초 종료 0 은 DSN 없는 전량 SKIP 이다 — 이 회차 산출물을 그 게이트는 검증하지 않는다. 릴리즈·다음 회차는 TEST_POSTGRES_DSN 을 채우고 `-v` 로 확인할 것.
- 결함 아님으로 정리: 음수 `Last-Event-ID` 는 id 가 양수라 lastID=0 과 동일해 동작 차이가 없다. `…PrefersTheQueryWhenItIsAheadOfTheHeader` 주석의 "프록시/재생 연결이 헤더를 붙인다" 는 과장(수동 재연결은 헤더를 안 보낸다) — 문구만 과하고 테스트 값은 유효하다.
- 릴리즈 노트: 프로덕션 0 파일, 테스트 1 파일(+223줄), server 패키지 실행시간 ~7초 증가. 사용자 가시 변화 없음.
- [러너 21:47] review approved — 리뷰 승인 (risk=low)
- [러너 21:47] pr created — https://github.com/hkjang/releasedock/pull/26
- [러너 21:50] ci passed — 검사 1개 모두 success
- [러너 21:50] merge done — 7b1fea6
