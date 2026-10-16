# 회차 노트 2026-10-16-192838-seaton-improve — seaton
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:28] base pinned — main@026d7a7
- [러너 19:28] autonomy release — 

## 정찰 노트
- maps.go 를 골랐다: 두 회차(a3cc4a5·2b81dec)가 이미 깔아 둔 rowScanner+fakeRows 꼴을 그대로 쓰는 프로덕션 1파일 과제이고, 건물·층 목록은 좌석맵 선택의 기준이라 행이 조용히 빠지면 사용자가 바로 오판한다. mcp.go(2026-10-08)·bulkAssignments(2026-10-09)는 가치가 비슷하지만 그 두 회차가 no-change 로 끝난 원인이 기록되지 않아 제쳤다 — 원인 모르는 과제를 재선택하지 않는다.
- 확인한 것: maps.go:17-31·59-77·105-131 의 옛 루프 3개, employees.go:20-33 의 rowScanner 와 규약 주석, employees_test.go:186-253 의 fakeRows 가 *int·**int·*bool·*[]byte 를 받지 못하는 것, `go test ./internal/app` 베이스라인 통과(11.478s).
- 추측으로 적은 것: 실서버에서 이 세 경로를 깨뜨려 본 적은 없다. `ALTER TABLE ... TYPE integer` 수법이 buildings·floors·floor_maps 에도 통한다는 것은 직전 회차에서의 유추이며 과제서에 미확인으로 표시했다.
- 구현자가 조심할 것: Scan 인수와 map 리터럴을 바이트 단위로 옮겨 열 짝을 보존할 것(fname/bname, seatCount/reviewCount 는 같은 타입이라 뒤바꿔도 컴파일된다). rowScanner 를 maps.go 에 재선언하지 말 것. 다른 파일의 같은 패턴은 손대지 말 것.
- 단위테스트만으로 수용 기준이 다 차므로 Docker·Playwright 에 시간을 쓰다 회차를 비우지 말 것.
- [러너 19:33] scout done — 건물·층·도면 목록(`internal/app/maps.go`)이 행 읽기 실패를 삼키고 부분 목록을 200 으로 돌려주는 것을 멈춘�

## 구현 노트
- 무엇·왜: maps.go 의 세 목록 루프(`if rows.Scan(...) == nil`)를 순수 함수 scanBuildings/scanFloors/scanFloorMaps 로 뽑아 Scan 오류와 rows.Err() 를 notFoundOrServer(500 database_error)로 올린다. 프로덕션 1파일, 응답 계약 무변경(Scan 인수·map 리터럴은 들여쓰기 외 바이트 동일).
- 확신 없는 곳: `listFloorMaps` 는 실서버 대조를 **못 했다** — 행을 만들려면 실제 도면 업로드가 필요하고 POST 가 CSRF 토큰을 요구한다. 단위테스트로만 덮였다(구조는 다른 둘과 동일). `listFloors`·`listBuildings` 는 실서버로 변경 전 200 `{"items":[]}` ↔ 수정본 500 을 대조했고 DB 로그에 `cached plan must not change result type` 2건을 확인했다.
- 과제서·프로필의 오류 하나: `ALTER TABLE <t> ALTER COLUMN <int열> TYPE integer` 는 **듣지 않는다**(이미 integer 면 결과 타입 불변 → 12회 모두 200). 결과 타입이 바뀌어야 한다 — sort_order→bigint, code→varchar(40) 로 바꿔서야 재현됐다. ideas.json 에 프로필 수정 후보로 적었다.
- 일부러 안 한 것: mcp.go·keys.go·detection.go·settings.go·grid.go·dashboard.go:317 의 같은 패턴(과제서 지시, 재작업률 규칙). 프런트·문서·E2E(응답 계약 무변경).
- 다음 역할이 조심할 것: 새 테스트는 DB 없이 돈다(`go test ./internal/app`, 11.5초). employees_test.go 의 fakeRows 에 *int·**int·*bool·*[]byte case 를 더했으므로 다른 파일 테스트도 이 네 타입을 쓸 수 있다. 실서버 대조는 Docker 이미지 빌드 없이 `CGO_ENABLED=0` 정적 바이너리를 alpine:3 에 ro 마운트해 돌렸다(이 셸은 sudo 가 없어 네이티브 기동이 /var/lib/seaton 에서 막힌다). 컨테이너·네트워크·임시파일은 모두 정리했다.
- [러너 19:42] brief accepted — 채택 — 과제서가 지목한 자리와 수단이 모두 지금 코드와 맞았다. 세 루프의 줄 번호, `rowScanner`(employees.go:20-24)와 규약 �
- [러너 19:42] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인: maps.go 의 세 scanX 를 임시 복제본에서 변경 전 꼴로 되돌려 새 테스트 6건이 실제로 실패하는 것(원장에 `- 실패 재현:` 줄이 없어 직접 재현), 열 짝을 뒤바꿔 ReturnsEveryRow 2건이 실패하는 것, Scan 인수·map 리터럴·SQL 이 main 과 바이트 동일한 것, notFoundOrServer 가 DB 원문을 숨기는 것, gofmt·vet·`go test ./... -count=1` 전부 통과.
- 못 본 것: 실서버·Docker·Playwright 로 돌려보지 않았다. listFloorMaps 는 읽기(maps.go:146-152)로만 배선을 확인했다. 보안·법무 관점에서 차단 사유 없음 — 새 엔드포인트·권한 변경·개인정보 수집·의존성 추가가 없다.
- 승인이어도 남는 우려: 이 세 엔드포인트가 이제 오류 시 200 대신 500 database_error 를 낸다(릴리즈 노트에 적을 것). 프런트 MapsPage·SeatMapPage 는 try/catch 로 받아 배너를 띄우므로 화면은 안전하다. 테스트의 `gridRaw == nil` 둘째 행은 프로덕션에 없는 합성 상태다(grid 는 NOT NULL DEFAULT '{}').
- 다음 회차: 프로필의 `ALTER COLUMN ... TYPE integer` 재현 수법은 틀렸다 — 결과 타입이 바뀌는 ALTER(sort_order→bigint, code→varchar(40))로 고쳐 적을 것. 남은 같은 패턴은 mcp.go(3)·keys.go:28·detection.go:468·settings.go:35·grid.go:150·dashboard.go:317.
- [러너 19:46] review approved — 리뷰 승인 (risk=low)
- [러너 19:46] pr created — https://github.com/hkjang/seaton/pull/46
- [러너 19:51] ci passed — 검사 2개 모두 success
