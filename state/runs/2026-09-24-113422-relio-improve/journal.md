# 회차 노트 2026-09-24-113422-relio-improve — relio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:34] base pinned — main@54e94f3
- [러너 11:34] autonomy low-risk — 롤백 PR 
- [러너 11:38] scout failed — 과제서 없음 — 구현자가 직접 고른다

## 구현 노트
- 무엇을 왜: `intelligence` 의 `contactRoles`·`stageLimits` 가 조회 실패를 빈 map 으로 삼켜, contacts 를 못 읽으면 그 세트의 모든 딜에 NO_DECISION_MAKER·NO_CHAMPION 이 켜지고 `opportunity_health_snapshots` 에 기록됐다. 두 조회를 같은 함수의 `rules`·`changesFor` 와 같은 `(map, error)` 로 맞췄다. 겸해서 `rows_err_test.go` 의 손수 갱신하던 패키지 열거를 internal 전체 순회로 바꿨다(머지로 들어온 `internal/mail` 이 빠져 있었다 — 다만 mail 의 루프 3곳은 이미 전부 `Err()` 를 본다).
- 확신 없는 곳·검증 못 한 것: (1) **`stageLimits` 의 오류 경로는 end-to-end 로 고립시키지 못했다.** 딜 조회가 `pipeline_stages` 를 먼저 조인해 그쪽에서 먼저 실패하므로, 실제 서버로는 `contactRoles` 쪽만 갈라 보였다. stageLimits 는 실제 `*pgxpool.Pool`(응답 없는 포트) 단위 테스트와 AST 불변식으로만 증명했다. (2) **rows.Err() 가 잡는 '스트림 중간 절단'** 자체는 재현하지 못했다 — 실제로 본 것은 Query 단계 실패다. (3) 이 변경으로 오류가 올라가면 `serviceError` 가 **400 `invalid_request` + pgx 원문**(`relation "contacts" does not exist (SQLSTATE 42P01)`)을 내보낸다. 이 저장소의 모든 DB 오류가 원래 그렇고 내가 만든 것은 아니지만 노출 면이 조금 넓어졌다 — 보류 아이디어에 별도로 적었다.
- 일부러 하지 않은 것: `serviceError` 의 400 기본값·메시지 스크럽 (전 엔드포인트 상태 코드가 바뀌어 계약 테스트·프런트까지 번짐), `intelligence` 의 나머지 조회·Limit:200 상한, 문서 갱신(사용자에게 보이는 설정·기능이 늘지 않아 ADMIN_GUIDE 에 적을 것이 없다).
- 다음 역할이 조심할 것: 새 `health_lookup_test.go` 는 DB 가 필요 없다 — 127.0.0.1 의 빈 포트를 잡았다 놓고 그리로 붙으므로 네트워크가 막힌 샌드박스에서도 돈다(연결 거부가 곧 기대하는 오류다). `rows_err_test.go` 는 이제 internal 아래 **모든** 패키지를 본다: 새 `for rows.Next()` 루프를 추가하면 어디든 걸린다. 프런트(web/)는 이번에 건드리지 않아 npm 검증은 돌리지 않았다.
- [러너 11:47] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인한 것: diff 3파일 전부 읽고, 두 새 테스트를 **변이 시험**으로 검증했다 — `contactRoles` 의 `return nil, err` 를 되돌리면 `health_lookup_test.go` 가 실패하고, `internal/mail` 에 Err() 없는 루프를 넣으면 `rows_err_test.go` 가 그 줄을 지목한다(둘 다 원복, 트리 깨끗). build·vet·gofmt·`go test -count=1 ./...` 전부 통과, 변경 두 패키지는 -race 도 통과. 오류 전파는 `rules`/`changesFor` 가 이미 쓰던 경로라 새 실패 부류가 아니다.
- 못 본 것: 프런트(web/, 이 diff 가 건드리지 않음), 실제 PostgreSQL 을 띄운 end-to-end. `stageLimits` 의 Scan/`rows.Err()` 분기는 구현자 말대로 여전히 미실행이다(Query 단계까지만 증명).
- 판정: **approve**, risk low, blocking 없음. security·legal 모두 차단 사유를 찾지 못했다 — 인가·비밀값·개인정보 수집/보존이 바뀌지 않고, `serviceError` 의 400+pgx 원문은 정적 SQL + 서버측 UUID 라 공격자가 조종할 수 없는 기존 부류다(별도 하드닝 회차 권고).
- 릴리즈 노트에 넣을 것: 이 수정은 **앞으로의** 기록만 막는다. 삼킴 기간에 `opportunity_health_snapshots` 에 이미 적힌 허위 NO_DECISION_MAKER·NO_CHAMPION 행은 백필되지 않아 그 구간 추이·코칭 지표는 계속 왜곡된다.
- 다음 회차가 알 것: 일시적 DB 실패 시 DealsAtRisk·CoachingDashboard·DealHealth 가 열화 대신 통째 오류가 된다 — 설정이 안 늘어 ADMIN_GUIDE 를 건너뛰었지만 **장애 대응 행 한 줄**은 값어치가 있다. 테스트에 `postgres://relio:relio@`(닫힌 포트용 가짜) 가 하드코딩돼 시크릿 스캐너가 걸 수 있다.
- [러너 11:50] review approved — 리뷰 승인 (risk=low)
- [러너 11:50] pr created — https://github.com/hkjang/relio/pull/33
- [러너 11:54] ci passed — 검사 2개 모두 success
- [러너 11:54] merge done — 9cd53bd
- [러너 11:54] release skipped — 자율화 단계 low-risk — 릴리즈는 사람이
