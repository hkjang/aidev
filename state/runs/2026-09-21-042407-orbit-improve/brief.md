- 과제: Time Travel 시작일에 첫 만남(first_met)을 반영해 등록 전 관계도 돌아볼 수 있게 하기 (가치 3 / 위험 1 / 작업량 M)
- 왜: `orbitAt`은 `people.first_met`까지 보고 과거 사람을 포함하지만 `orbitRange`는 `created_at`과 교류 시각만 보므로, 2020년에 처음 만난 사람을 오늘 등록하고 교류를 넣지 않으면 슬라이더로 2020년까지 돌아갈 수 없다. 현재 API의 `earliest_at`을 사람 포함 규칙에 맞추면 이미 구현된 과거 조회를 화면에서도 이용할 수 있다.
- 수용 기준: 1) 내 사람의 created_at·first_met·내 교류 occurred_at 중 가장 이른 시점이 현재 GET /api/v1/orbit의 earliest_at으로 반환되고, 빈 사용자는 null이며 다른 사용자 기록은 섞이지 않는다. 2) 오늘 등록/과거 first_met/교류 없음인 사람의 경우 응답 earliest_at을 그대로 URL 인코딩해 GET /api/v1/orbit?at=...에 보내면 historical=true이고 해당 사람이 nodes에 포함된다; 그 첫 만남보다 앞선 날에는 제외된다. 3) 실제 PostgreSQL과 `server.New`의 라우팅·인증·getOrbit→orbitRange / getOrbit→writeOrbitAt→orbitAt을 통과하는 회귀 테스트가 1~2를 증명하고, 기존 first_met 없는 4개 legacy 비교 테스트가 유지된다. first_met이 등록일보다 늦으면 등록일, 교류가 first_met보다 이르면 교류가 시작점이어야 한다.
- 건드릴 파일: `internal/server/timetravel.go:orbitRange` — 기존 독립 집계를 유지하면서 first_met 최솟값을 포함하고 주석을 실제 시작일 규칙에 맞춘다; `internal/server/timetravel_db_test.go:openTestStore/seedUser/seedPerson/seedInteraction/TestOrbitRangeMatchesLegacyJoin` — 기존 헬퍼를 재사용하고 새 first_met 범위 및 HTTP 왕복 테스트를 추가한다(관계·세션 시드 헬퍼는 이 파일에만); `docs/guide.md:2.2 나의 Orbit 우주 캔버스` — 시작일이 등록·첫 만남·교류 중 가장 앞선 기록이고 변경 이력 없는 중요도·소속·고정은 현재 값임을 짧게 설명한다.
- 검증 명령: 저장소 루트에서 `go test ./...`; 격리된 PostgreSQL의 DSN을 ORBIT_TEST_DATABASE_URL에 설정하고 `go test -count=1 -v ./internal/server -run 'TestOrbit(Range|FirstMet)'`, `go test -race -count=1 ./...`. 새 HTTP 테스트는 `TestOrbitFirstMet...`로 이름 짓는다. DB 없는 초록 또는 SKIP을 SQL 검증 완료로 취급하지 않는다.
- 위험과 피할 것: auth.go·internal/store/migrations·.github/workflows·VERSION·웹 구현·의존성은 변경하지 않는다. 지난 회차가 제거한 사람×교류 조인을 되살리지 않는다. legacyOrbitRange는 first_met 없는 입력의 기준일 뿐이므로 새 first_met 케이스를 옛 질의와 같게 강요하거나 옛 4개 테스트를 삭제하지 않는다. first_met은 date, 다른 두 값은 timestamptz이므로 현재 DB 날짜 변환 규칙과 orbitAt의 `$2::date`가 왕복에서 같은 날을 가리키게 한다; UTC와 Asia/Seoul DB 세션에서 확인하고, 제품 전체의 시간대 정책 변경은 별도 과제로 남긴다. 시드/정리는 고유 사용자와 FK cascade를 쓰며 운영 DB에 테스트하지 않는다.
- 차선 후보: orbitAt 사람 포함·기억·마지막 교류의 시점 경계를 실제 PostgreSQL 테스트로 고정 (가치 3 / 위험 1 / M) — 1순위 가정이 재현되지 않을 때만 채택. 같은 파일의 헬퍼로 created_at/first_met/과거 교류 각각으로 포함, 미래 기록만 있으면 제외, 승인된 기억의 coalesce(occurred_at,created_at)<=at만 집계, 1년 넘은 마지막 교류는 유지하되 미래 교류는 제외를 실제 orbitAt 호출로 증명한다. 프로덕션 질의 변경 없이 `go test -count=1 -v ./internal/server -run TestOrbitAt`과 DB 포함 전체 Go 테스트로 검증한다.

실행 순서 및 검토 지점 (모두 구현자 자체 검토, 사람 대기 없음; 현재 상태는 전부 미착수):
1. 기존 테스트와 DB 환경 확인: `go test -count=1 -v ./internal/server -run TestOrbitRange`를 DB 포함 실행한다. 신규 fixture는 `seedPerson` 반환 id에 first_met을 UPDATE하고 relationships에 id/user_id/person_id를 INSERT하면 된다(001_init.sql 확인; 나머지는 기본값, anchored는 후속 마이그레이션). 필요 세션은 sessions(token_hash,user_id,expires_at)에 secure.SHA256(고유 토큰)을 넣고 orbit_session 쿠키로 `New(st,"test","","")`를 호출한다. 인증 우회 context나 SQL mock을 증거로 쓰지 않는다. 검토: 기존 4개 케이스 PASS, SKIP 없음.
2. 새 `TestOrbitFirstMet...`에 위 반례와 HTTP 왕복을 추가해 현 코드에서 earliest_at 기대값 실패를 확인한 뒤, 같은 작업 단위에서 orbitRange와 주석을 수정한다. 검증: `go test -count=1 -v ./internal/server -run 'TestOrbit(Range|FirstMet)'`. 검토: 구/신 케이스가 모두 PASS하고 first_met 없는 기존 출력 불변. 날짜 경계는 DB 연결 전체의 TimeZone을 DSN에서 고정해 별도 UTC/Asia/Seoul 실행으로 확인한다(풀의 한 연결에만 SET 하지 않음).
3. guide.md 2.2를 갱신하고 `go test -race -count=1 ./...`를 DB 포함 실행한다. 검토: diff가 위 세 파일 안이고 보호 파일 없음; 새 계약은 명시적 기대값으로 검증. DB를 사용할 수 없으면 성공으로 기록하지 말고 검증 제한과 차선 판단을 남긴다.

DB 준비 명령(구현자용, 정찰에서 실행하지 않음):
```sh
docker run -d --name orbit-firstmet-test -e POSTGRES_PASSWORD=orbit -e POSTGRES_DB=orbit -p 127.0.0.1:55471:5432 postgres:16-alpine
docker exec orbit-firstmet-test pg_isready -U postgres -d orbit
ORBIT_TEST_DATABASE_URL='postgres://postgres:orbit@127.0.0.1:55471/orbit?sslmode=disable&timezone=UTC' go test -count=1 -v ./internal/server -run 'TestOrbit(Range|FirstMet)'
ORBIT_TEST_DATABASE_URL='postgres://postgres:orbit@127.0.0.1:55471/orbit?sslmode=disable&timezone=Asia%2FSeoul' go test -count=1 -v ./internal/server -run 'TestOrbit(Range|FirstMet)'
ORBIT_TEST_DATABASE_URL='postgres://postgres:orbit@127.0.0.1:55471/orbit?sslmode=disable' go test -race -count=1 ./...
docker rm -f orbit-firstmet-test
```
컨테이너 이름/포트가 이미 있으면 다른 빈 이름/포트를 택하고 이 작업이 만든 컨테이너만 정리한다. pg_isready 성공 후 테스트한다.

선택 검토: API 범위만 넓히는 방법을 채택한다(기존 UI가 earliest_at을 그대로 소비하므로 UI 변경 불필요). UI에서 임의의 고정 과거 연도로 늘리는 방법은 실제 관계 시작점을 모르며 REST 소비자 문제도 남겨 제외했다. 관계 이력 테이블 도입은 중요도 등 복원까지 요구될 때의 대안이며 이번 45분 범위를 넘긴다. 아무것도 하지 않고 API 직접 호출만 문서화하는 방법은 UI에서 접근 불가능한 문제를 남긴다. CI postgres는 가치가 같지만 보호 경로를 건드리고, 순수 테스트 보강은 이번 사용자 동작 개선보다 뒤로 뒀다.

견적 근거: bottom-up 기본 30~35분(환경·시드/반례 10분, SQL·HTTP 왕복 12~15분, 시간대 검증·문서·전체 테스트 8~10분), 알려진 불확실성인 날짜 형변환/DB 기동에 contingency 5~10분을 별도로 둔 총 35~45분이다. 통계적 80% 달성을 보장할 표본은 없으며 확신은 중간; 관리 예비비는 배정하지 않고 새 범위 발견 시 계획을 수정한다. 유사 추정은 09-20의 동일 파일·DB 헬퍼 기반 작은 SQL 개선이 한 회차에 완료된 기록과 비교했으며, 이번에는 HTTP 왕복이 추가돼 S 대신 M이다(과거 실제 소요 분은 미확인이라 수치 교차검증은 불가). Docker 사용 가능·이미지 확보 가능 및 현재 기본 TimeZone이 일관적이라는 전제이며, 이미지 다운로드와 날짜 경계 확인 후 재추정한다.

정찰 확인: main@e4a2ca0, 최근 git log -30, README·문서 5종·Makefile·go.mod·package.json·Vite/Vitest·두 CI workflow·위 함수/스키마를 읽었다. 저장소 내 CLAUDE.md·AGENTS.md·별도 로드맵·실제 TODO/FIXME 항목은 검색에서 발견하지 못했다. `go test ./...` 통과(캐시), Docker daemon 29.7.2 응답; DB 포함 테스트·브라우저 재현·웹 테스트는 정찰에서 미실행이며 web/node_modules 없음. 이전 프로필의 'main에 DB 테스트가 없다'는 현재와 달라 profile.md를 갱신한다.

적용 스킬: Skill 전용 호출 도구는 노출되지 않아 로컬 원문으로 읽음 — /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md (references/sources.md 포함), /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md, /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md. 위 견적은 저장소/회차 기록에 근거한 작업 추정이며 외부 비용 산정 기준의 준수나 통계적 신뢰도를 주장하지 않는다.
