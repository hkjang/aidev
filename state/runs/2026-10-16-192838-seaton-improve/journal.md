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
- [러너 19:51] merge done — 3a191ad

## 릴리즈 노트
- 버전·태그: 1.4.18 / `v1.4.18`(주석 태그, 커밋 `1c8421a` 위). 최근 다섯 회차가 모두 패치만 올렸고 이번 변경도 내부 수정 한 파일이라 패치로 올렸다. detached HEAD 에서 커밋·태그했고 브랜치는 만들거나 옮기지 않았다(main 은 b377293 그대로).
- 방식은 v1.4.15~v1.4.17 과 동일하게 맞췄다: 버전 표기는 코드가 아니라 문서에만 있고(앱 버전은 Docker `--build-arg VERSION`), README·docs/ADMIN_GUIDE.md·docs/USER_GUIDE.md·docs/ROADMAP_PLAN.md 를 1.4.18 로 고친 뒤 `python3 scripts/build-docs.py` 로 HTML 을, 공용 `md2pdf.mjs` 로 USER_GUIDE.pdf·ADMIN_GUIDE.pdf 를 다시 구웠다. 커밋 제목 `docs: v1.4.18 기준으로 문서 정비`, 태그 주석은 커밋 메시지와 동일 — 모두 직전 세 태그와 같은 꼴이다. 바뀐 파일 10개가 v1.4.17 커밋과 같은 목록이다.
- 손대지 않은 것 하나를 적어 둔다: `build-docs.py` 전체 실행은 `EXECUTIVE_REPORT`·`USER_GROUPS_ANALYSIS` 의 HTML·PDF 도 건드린다(생성기가 1487c1d 에서 `figure`/`figcaption` CSS 를 더한 뒤 그 둘만 재생성되지 않아 v1.4.0 이후로 표류 중). 버전 표기와 무관한 표류이고 v1.4.13~v1.4.17 다섯 회차가 모두 커밋에서 뺐으므로 이번에도 `git checkout --` 으로 되돌렸다. 고치려면 별도 문서 회차가 맞다.
- 검증(모두 통과): `gofmt -l .`(출력 없음) · `go build ./...` · `go vet ./...` · `go test ./...`(internal/app 11.589s) · `web` 에서 `npm ci && npm test && npm run build` — CI 의 `test` 잡과 같은 순서다. 그 뒤 README 가 적어 둔 로컬 릴리즈 검증 `scripts/release-image.sh 1.4.18`(스크립트에 실행 권한이 없어 `bash` 로 호출) → `seaton:v1.4.18` 빌드 성공, `gzip -t SeatOn-v1.4.18.tar.gz` 통과(47,646,130바이트). 버전 일치는 추측하지 않고 확인했다 — 빌드된 이미지에서 바이너리를 꺼내 박힌 버전 문자열이 `1.4.18` 하나임을 봤다. `/api/v1/version` 호출은 DB 없이 기동이 안 되어 생략했다.
- GitHub Release 는 만들지 않았다. `release.yml` 이 `v*.*.*` 태그 푸시에 반응해 이미지를 빌드하고 `SeatOn-v1.4.18.tar.gz` 를 `--generate-notes --title "SeatOn v1.4.18"` 로 첨부한다 — 그래서 `release.json` 의 `github_release` 는 false, `assets` 는 빈 배열이다. 로컬 검증용으로 만든 tar.gz 는 지웠다(러너가 올릴 자산이 아니다). 원격에는 아무것도 보내지 않았다: 커밋·태그 푸시는 러너의 일이다.
- 다음 회차가 알아 둘 것: 릴리즈 노트 본문은 PR 제목에서 자동 생성되므로, 이번처럼 "오류 시 200 대신 500 database_error" 같은 동작 변경을 사람에게 알리려면 PR 제목이나 문서 커밋 본문이 유일한 자리다. 이번에는 문서 커밋 본문에 적었다.
- [러너 20:02] release published — v1.4.18
- [러너 20:04] assets verified — v1.4.18 자산 1개 (이전 v1.4.17: 1)
