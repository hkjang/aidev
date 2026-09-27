# 회차 노트 2026-09-27-155208-orbit-improve — orbit
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 15:52] base pinned — main@65c6ae9
- [러너 15:52] autonomy release — 

## 정찰 노트
- 골랐다: data.go 의 사람 id uuid 가드(500→404/400). 보류 목록의 좁은 항목("createPersonLink 만 500")을 열어 보니 같은 결함이 getPerson·updatePerson·deletePerson·listPersonLinks·createPersonLink·deletePersonLink 여섯 곳에 공통이었다 — 한 곳만 고치면 반쪽 수정이라 범위를 여섯으로 넓혔다. 제친 이유: openapi 테스트(세 회차째 차선)는 값이 낮고, orbitAt contexts/LIMIT/memories-count·first_met TimeZone 은 전부 DSN 이 있어야 증명되어 CI 에서 SKIP 으로 숨는다. 이번 과제는 `&Server{store:nil}` 로 DB 없이 증명되어 CI 에서 실제로 돈다.
- 미확인(과제서에도 적었다): "postgres 가 uuid 컬럼에 잘못된 문자열을 받으면 22P02 오류를 내고 0행이 아니다" 는 스키마(001_init.sql:74 `id uuid`)와 `internalError` 배선을 읽은 추론이다. docker 실행이 승인되지 않아 실제 500 을 재현하지 못했다. 수용 기준 1~4 는 이 추론과 무관하게 증명되므로 구현이 막히지는 않는다. `deletePersonLink` 의 `linkID` 가 uuid 컬럼인지도 004_person_links.sql 을 열지 못해 구현자에게 확인을 맡겼다.
- 조심할 것: `createInteraction`(data.go:453)은 `err != nil || !exists` 라 이미 404 로 떨어진다 — 손대면 멀쩡한 경로를 흔든다. `mcp.go` 의 `args.PersonID` 는 봉투가 달라 범위 밖(후속 아이디어로 남겼다). `normalizeLink`·`orbitLinks` 는 현재/과거 두 경로가 공유하니 건드리지 말 것.
- 프로필은 1일 전 것이 지금 코드와 맞아 다시 쓰지 않았다(data.go·timetravel.go 구조, DB 테스트 3개, 포트 회피 목록 모두 확인됨). `go test ./...` 는 이 회차 시작 시점에 초록이다.
- [러너 15:56] scout done — URL/본문으로 들어온 사람 id 가 uuid 모양이 아닐 때 `data.go` 의 여섯 핸들러가 500 을 내는 것을 DB 앞에서 404/

## 구현 노트
- 무엇을/왜: `internal/server/data.go` 에 순수 함수 `looksLikeUUID` 하나를 더해 여섯 핸들러(getPerson·updatePerson·deletePerson·listPersonLinks·createPersonLink·deletePersonLink)가 uuid 컬럼에 모양 어긋난 문자열을 넘겨 500(postgres 22P02)을 내던 것을 DB 앞에서 404 로, `createPersonLink` 본문 `person_id` 는 400 으로 끝내게 했다. 프로덕션 1파일, 커밋 1개.
- 확신 없는 곳: 없다시피 하지만 비평가가 먼저 볼 곳은 **대문자 uuid 를 통과시키기로 한 판단**이다 — postgres 는 대문자를 같은 id 로 읽으므로 거부하면 지금 찾아지던 요청이 404 가 된다(수용 기준 3 위반). 반대로 중괄호·하이픈 없는 형태는 postgres 가 받아 주지만 Orbit 이 발급한 적 없는 모양이라 거부한다. 이 비대칭은 의도적이고 `looksLikeUUID` 주석에 이유를 적었다.
- 정찰이 "미확인" 으로 남긴 22P02→500 추론은 이번에 실제 postgres(docker `postgres:16-alpine`, 포트 55521)로 확정했다. `looksLikeUUID` 를 `return true` 한 줄로 되돌린 빌드에서 실제 핸들러가 500 internal_error, 되살리면 404/400; 같은 실행에서 정상 uuid(없는 uuid 404 / 200 / 201 / 204)는 양쪽 동일. `psql` 로 `ERROR: 22P02: invalid input syntax for type uuid` 도 확인했다. **이 DB 탐침 파일은 일회용이라 커밋하지 않았다** — 커밋된 테스트는 전부 `&Server{store: nil}` 과 순수 함수라 DSN 없이 CI 에서 실제로 돈다.
- 일부러 안 한 것: `mcp.go` 의 `args.PersonID`(JSON-RPC 봉투라 오류 매핑이 REST 와 다름, 확인 안 함 — 후속 아이디어로 남김), `createInteraction`(`err != nil || !exists` 라 이미 404), `setAnchor`(과제 범위 밖), `normalizeLink`·`orbitLinks`(현재/과거 공유 경로), 문서 갱신(응답 코드가 정상 경로에서 바뀌지 않아 고칠 문장이 없다).
- 다음 역할이 조심할 것: 새 테스트는 DB 가 필요 없다(`go test -race ./...` 로 그대로 돈다). 기존 `timetravel_db_test.go` 는 여전히 DSN 이 있어야 돌고 DSN 을 줬을 때도 전부 통과하는 것을 확인했다. 컨테이너는 정리했고 작업 트리에 남은 임시 파일은 없다.
- [러너 16:03] brief accepted — 채택 — 근거(여섯 핸들러가 uuid 컬럼에 무검사 전달, `createInteraction` 은 이미 404, `person_links.id` 도 uuid)가 지금 코드와 정�
- [러너 16:03] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인했다: diff 전체(프로덕션 `data.go` 7군데 + 순수 함수 1개, 테스트 20 하위 시험). 테스트는 `&Server{store:nil}` 로 핸들러를 직접 부르고 상태·`apiError.code`·빈 메시지 아님을 단언하므로, 가드가 없으면 `s.store.DB` nil 역참조로 반드시 패닉한다 — 원장의 `실패 재현`(data.go:697 nil pointer)이 이번 변경이 고치는 증상과 정확히 맞는다. `gofmt -l internal/server`(무출력)·`go vet ./...`·`go test -count=1 ./internal/server` 직접 돌려 초록.
- 구현자가 의심한 자리(대문자 허용)를 따로 뒤졌다 — 판단이 맞다. people 행은 `createPerson`(data.go:332)의 `id.New()` 로만 생기고 소문자 하이픈 형태뿐이라, 가드가 기존에 찾아지던 id 를 죽이는 경우가 없음을 확인했다(`INSERT INTO people` 전수 grep).
- 못 본 것: 실제 postgres 로 재현하지 않았다(원장의 DSN 탐침 기록과 스키마 `uuid` 컬럼 확인으로 대신했다). 웹·MCP 클라이언트가 이 엔드포인트에 어떤 모양의 id 를 보내는지도 코드 밖에서는 확인하지 않았다.
- 승인이어도 남는 우려(릴리즈 노트·다음 회차): ① `setAnchor`(data.go:618)는 같은 결함이 그대로다 — `relationships.person_id` 도 uuid 라 `/people/{bad}/anchor` 만 형제 경로와 달리 아직 500 이다. 최우선 후속. ② `mcp.go:170·202·219` 도 동일. ③ 대문자 허용의 대가로 `createPersonLink` 의 대문자 입력은 자기연결 400 을 비껴가고(404 로 떨어짐), `normalizeLink` 의 Go 문자열 순서가 postgres uuid 순서와 어긋나 `CHECK (person_a < person_b)` 를 어겨 500 이 될 수 있다 — 둘 다 전부터 있던 것이고 정답은 거부가 아니라 소문자 정규화지만, 공유 경로라 이번 범위 밖이 맞다.
- 원장 문구 하나만 부정확: "새 오류 코드·메시지는 만들지 않았다" 인데 data.go:715 의 "이을 사람을 확인해 주세요." 는 새 메시지다(코드는 기존 `validation_error`). 문서 미갱신 판단은 옳다 — `openapi.go` 는 404 를 아예 열거하지 않고 400 은 모든 작업에 이미 붙어 있다.
- [러너 16:06] review approved — 리뷰 승인 (risk=low)
- [러너 16:06] pr created — https://github.com/hkjang/orbit/pull/13
- [러너 16:09] ci passed — 검사 1개 모두 success
- [러너 16:09] merge done — cd73845
- [러너 16:15] release published — v0.7.3
- [러너 16:16] assets verified — v0.7.3 자산 1개 (이전 v0.7.2: 1)
