# 회차 노트 2026-10-05-202728-releasedock-improve — releasedock
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 20:27] base pinned — main@dd9fce1
- [러너 20:27] autonomy release — 

## 정찰 노트
- 전체 모드 저장 로그 백엔드 조각을 골랐다: 최근 다섯 회차가 전부 프런트 로그 안내였고 그 안내들이 모두 "되찾을 수단이 없다" 로 끝난다 — `server.go:287-301` 을 다 읽어 `/releases/{id}/logs` 가 정말 없음을 확인했고, `simple.go:1351-1400` 이 `logRowScanner`·`writeSimpleRunLog`·`runLogDisposition` 을 이미 DB 없이 시험 가능한 모양으로 일반화해 둬서 L 이던 과제의 risk 가 2 로 내려갔다.
- 제친 후보: `make vet` 는 **기각**(브랜치 `auto/2026-09-17-0853` 커밋 `f700e87` 이 같은 변경을 이미 담고 사람이 머지하지 않고 있다 — 우회 금지). 단순 모드 `stopped` 가드는 차선으로 내렸다(EventSource 스펙상 `close()` 뒤 error 는 안 뜨므로 도달 가능성 자체가 의심스럽다). 스트림 한도 통합 테스트는 DSN 없이 SKIP 되는 것을 이번에 픽스처로 확인해 또 밀었다.
- 추측으로 적은 것(미확인): 내려받기 파일 이름에 넣을 릴리즈 식별 정보 — `releaseSelect` 가 name/version 을 주는지 열어 보지 않았다. `release_job_logs` 스키마도 `streamReleaseLogs` 의 SELECT 로만 알고 마이그레이션 파일은 보지 않았다(필요한 네 컬럼은 그 SELECT 가 증명한다).
- 구현자가 조심할 것: 네 공용 헬퍼를 **복제하지 말고 호출**할 것(두 경로가 갈라진다 — 개명·이동은 범위 밖, 별도 아이디어로 남겼다). 릴리즈 DB 픽스처는 이 저장소에 전례가 없어(`releases_*_test.go` 0개) 세션을 먹을 수 있다 — 그때는 DB 없는 렌더 테스트로 마치고 라우트 테스트를 뺐다고 **명시**할 것. DSN 없는 SKIP 을 PASS 로 보고하지 말 것.
- 세션이 모자라면 `?format=text` 내려받기만 끝내도 목적은 달성된다. JSON 페이지네이션은 화면 조각이 붙을 때 더 필요해진다.
- [러너 20:32] scout done — 전체 모드 릴리즈의 저장 로그 조회·내려받기 백엔드 엔드포인트 (가치 3 / 위험 2 / 작업량 M)

## 구현 노트
- 무엇·왜: 전체 모드에 `GET /api/v1/releases/{id}/logs` 를 추가했다(`releases.read`). 기본 JSON 은 단순 모드와 같은 필드·커서 계약, `?format=text` 는 헤더·200 을 먼저 보낸 뒤 행을 스트리밍하는 평문 내려받기. 표시 한도(4999줄)로 밀려난 줄과 끊김 이후의 줄을 되찾을 서버 경로가 없던 것을 메운다. 프로덕션 2파일 +108줄 / 삭제 0줄.
- 재사용: 렌더링·잘림 공지·파일 이름은 `logRowScanner`·`writeSimpleRunLog`·`simpleRunLogTruncatedNotice`·`runLogDisposition` 을 **호출**했다(복제·개명·이동 없음). 새 프로덕션 로직은 `releaseLogLabel(appName, version)` 과 `writeReleaseLogDownload` 둘뿐. SQL 은 `streamReleaseLogs` 와 같은 JOIN·같은 정렬이고 스트림은 한 줄도 바꾸지 않았다.
- **확신 없는 곳·검증 못 한 것**: ① 권한 문자열이 `releases.read` 라는 것을 테스트로 고정하지 못했다 — 인증된 principal 이 `store` 를 요구하고 DSN 이 없다. 라우트 테스트는 401 까지만 간다(라우트 등록·패턴 비충돌·인증 게이트는 고정됨). ② JSON 경로의 두 쿼리와 404/500 매핑은 DB 가 필요해 단위 테스트가 지나지 않는다 — `go build`/`vet` 과 코드 검토만. ③ `?limit` 의 `requested < limit` 경계(2000 을 그대로 요청하면 무시됨)는 단순 모드 `listSimpleRunLogs` 를 글자대로 따랐고 의도된 동작인지는 확인하지 않았다.
- 일부러 하지 않은 것: 릴리즈 DB 픽스처(전례 0개 — `releases_*_test.go` 없음, 세션을 먹고 DSN 없으면 SKIP 된다 → 아이디어로 남겼다). 프런트(`api.ts`·`ReleaseDetailPage`)는 범위 밖. 네 공용 헬퍼의 개명·이동도 범위 밖(이제 이름이 실제로 거짓이 됐으니 별도 회차 후보).
- 다음 역할이 조심할 것: 신규 7건은 **DB 없이** 돈다 — `go test ./internal/server/ -run ReleaseLog -count=1`. 같은 패키지의 45건 SKIP 은 전부 기존 DB 테스트이고 PASS 가 아니다. 웹 0 파일 변경이라 웹 테스트는 돌리지 않았다. `fakeLogRows` 대역은 `simple_log_test.go` 에 있는 것을 그대로 썼으니 그 파일을 옮기면 새 테스트가 깨진다.
- [러너 20:38] brief accepted — 채택 — 과제서가 지목한 모든 근거(`server.go:301` 에 `/logs/stream` 만 등록, `/logs` 부재, `releases.go:1507` 의 JOIN·정렬, `simple.go:1
- [러너 20:39] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인함: 라우트 한 줄을 지워 테스트가 실제로 빨강이 되는 것(404 catch-all)을 재현했고, 구현자가 "DB 없이 못 했다" 던 JSON 경로를 PostgreSQL 16 컨테이너(마이그레이션 024까지)로 직접 돌려 페이징·과거끝 커서 보존·404·`?format=text` 본문·헤더가 주석대로임을 확인했다. 실제 DB 로 backend 전 패키지 통과(internal/server 144 PASS / 1 SKIP), gofmt·vet 무출력. 리뷰용 임시 파일은 전부 제거, worktree clean.
- 못 본 것: 권한 문자열 `releases.read` 자체(미들웨어를 지나는 RBAC 시딩 테스트 없음 — 한 줄 diff 로만 확인), 로그가 매우 큰 릴리즈의 실측 성능, 프런트(변경 0파일).
- 승인이어도 남는 우려: ① `{id}` 가 UUID 가 아니면 404 가 아니라 500 database_error (기존 getReleaseByID 와 동일한 기존 패턴 — 이번 결함 아님) ② `?format=text` 는 행 상한이 없고 계획이 `Sort on l.id` + `Seq Scan on release_jobs`(release_id 인덱스가 비종료 상태 전용 부분 인덱스뿐), SSE 의 사용자당 한도에 해당하는 보호가 없다 ③ 엔드포인트만 들어가 UI 에서 아직 쓰이지 않는다 — 릴리즈 노트에 명시할 것.
- 다음 회차가 알 것: 구현 노트의 "릴리즈 DB 픽스처 전례 없음" 은 틀렸다 — `server_test.go:446 newRollbackRetryFixture` + `simple_stream_cursor_test.go:211-216` 의 `SetPathValue`+`principalKey` 주입 패턴이 선례다. 이번 리뷰가 그 패턴으로 JSON 경로를 검증했으니 영구 테스트화는 S 규모다.
- [러너 20:46] review approved — 리뷰 승인 (risk=low)
- [러너 20:47] pr created — https://github.com/hkjang/releasedock/pull/34
- [러너 20:49] ci passed — 검사 1개 모두 success
- [러너 20:49] merge done — 18a53ce
