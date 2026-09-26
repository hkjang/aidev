# 회차 노트 2026-09-26-220237-orbit-improve — orbit
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 22:02] base pinned — main@81222bf
- [러너 22:02] autonomy release — 

## 정찰 노트
- 골랐다: 과거 Orbit 응답의 links 가 nodes 밖 사람을 가리키는 끊긴 참조. 지난 회차가 맞춘 "현재/과거 같은 모양" 계약의 남은 구멍이고 프로덕션 1파일(writeOrbitAt)로 닫힌다. orbitAt DB 테스트 재작성(2026-09-24 verify 실패)과 성능 항목(LIMIT·GROUP BY)은 회귀 기준이 main 에 없어 제쳤고, openapi 렌더 테스트는 가치가 낮아 차선으로 돌렸다.
- 직접 열어 확인: timetravel.go:147~184 가 orbitLinks 결과를 거르지 않음, OrbitCanvas.tsx:425 가 끝점 없는 링크를 continue, person_links 의 CHECK(person_a<person_b), relationships 의 기본값 없는 NOT NULL 은 id/user_id/person_id 뿐, main 에 seedRelationship·seedPersonLink 헬퍼 없음, go test ./... 초록.
- 추측(미확인): docker 로 postgres 를 새 포트(55491)에 띄울 수 있다는 가정 — 못 띄우면 수용 기준 3 을 증명할 수 없으니 차선 후보로 갈아탈 것. 새 아이디어 "createPersonLink 가 비-uuid person_id 에 500" 도 DSN 으로 확인하지 않았다.
- 조심할 것: 거르는 자리는 writeOrbitAt 안이지 orbitLinks 안이 아니다(현재 경로가 같은 함수를 쓴다). person_links 에 시간 필터를 거는 쪽으로 범위를 넓히지 말 것 — 이 모듈은 사용자가 정한 값은 오늘 값을 쓴다고 선언해 뒀다.
- 프로필은 0일 전 것이고 이번에 읽은 코드와 어긋나는 데가 없어 다시 쓰지 않았다.
- [러너 22:07] scout done — 과거 Orbit(`GET /orbit?at=`) 응답의 `links` 가 그날 없던 사람을 가리키는 끊긴 참조를 담는 것을 고치기 (가치 3

## 구현 노트
- `writeOrbitAt` 이 `orbitLinks` 의 오늘 전체 목록을 그대로 담아 `?at=` 응답의 `links` 가 `nodes` 밖 id 를 가리켰다. `nodes` 의 id 집합으로 두 끝점이 모두 있는 링크만 남겼다(거르는 자리는 `orbitLinks` 가 아니라 `writeOrbitAt` — 현재 경로가 같은 함수를 쓴다). 프로덕션 1파일 + 테스트 + docs/API.md 한 단락.
- 확신 없는 곳: 없음에 가깝지만 굳이 적자면 — 링크 항목의 키 수를 `len(l) != 4` 로 고정했으니 앞으로 `orbitLinks` 가 키를 늘리면 이 시험이 먼저 빨개진다(의도한 것이지만 남의 변경을 막는 자리다). `nodes` 의 `id` 가 문자열이 아닌 경우는 타입 단정 실패 시 그 노드를 집합에 넣지 않게만 해 뒀다 — `orbitAt` 이 항상 문자열을 넣으므로 실제로는 일어나지 않는다.
- 일부러 하지 않은 것: `person_links` 에 시점 필터(`created_at<=at`)를 걸지 않았다. 이 모듈은 사용자가 손으로 정하는 값은 오늘의 값을 쓴다고 선언해 뒀고(timetravel.go:17~19) 이번 범위는 "응답 안에서 닫히게" 만드는 것이다. `orbitLinks`·`getOrbit`·`orbitAt` 의 SQL 과 성능(LIMIT·GROUP BY)도 건드리지 않았다.
- 다음 역할이 조심할 것: 새 시험 `TestOrbitAtLinksStayWithinNodes` 는 **실제 postgres 가 있어야 돈다**. `ORBIT_TEST_DATABASE_URL` 이 없으면 SKIP 이고 CI 는 postgres 없이 돌므로 CI 초록만으로는 이 시험이 돌았다는 뜻이 아니다. 로컬 확인: docker `postgres:16-alpine` 을 빈 포트에 띄우고 `ORBIT_TEST_DATABASE_URL=… go test -race -count=1 -v ./internal/server -run TestOrbit`(3 테스트 10 하위 시험 PASS). 이번에 쓴 포트 55491 컨테이너는 지웠다. 웹은 건드리지 않아 `npm` 을 돌리지 않았다 — verify 가 `npm ci` 를 돈다는 점만 기억할 것.
- [러너 22:12] brief accepted — 채택 — 근거(`writeOrbitAt` 이 `orbitLinks` 결과를 거르지 않음, `orbitAt` 의 `JOIN relationships`, `person_links` 의 `CHECK (person_a < person_b
- [러너 22:12] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 판정 approve / risk low / blocking 없음. 새 시험이 정말 바뀐 경로를 지나는지 직접 재현했다 — docker postgres(포트 55493) 위 HEAD 에서 TestOrbit 3개 10 하위 시험 PASS, `internal/server/timetravel.go` 만 main 판으로 되돌리니 하위 시험 2개가 원장의 `- 실패 재현:` 줄과 같은 메시지로 빨개졌다. 확인 뒤 되돌리고 컨테이너도 지웠다(worktree clean). gofmt·vet·`go test ./...` 초록.
- 못 본 것: 웹(`npm ci`/vitest/build)은 diff 에 웹 파일이 없어 돌리지 않았고 verify 기록(7개 통과)만 믿었다. `orbitAt`·`orbitLinks` 의 성능(LIMIT 1000/5000)과 `createPersonLink` 의 비-uuid 500 의혹은 이번 범위 밖으로 두고 보지 않았다.
- 승인이어도 남는 우려 ①: **릴리즈 노트에 동작 변경으로 적을 것** — `?at=` 의 `links` 가 이제 그 응답의 `nodes` 로 좁아진다. 웹(OrbitPage→eclipse/gravityAssist)은 이미 끝점 없는 링크를 버려 영향이 없지만 API·MCP 호출자에게는 보이는 변화다. `docs/API.md:48` 의 "every other key … is the same in both cases" 는 새 단락(58~62)과 문자 그대로는 어긋난다(키 모양 이야기로 읽혀 결함으로는 보지 않았다).
- 승인이어도 남는 우려 ②: 같은 끊긴 참조가 **현재 경로**에도 원리상 남아 있다 — `getOrbit` 도 `JOIN relationships` 라 relationships 행 없는 사람은 `nodes` 에 없는데 링크는 그대로 실린다. 오늘은 `data.go:298` 이 같은 트랜잭션에서 그 행을 넣어 도달 불가이고, 새 하위 시험이 현재 경로를 "전체 목록" 으로 못 박았다. 그 가정이 깨지는 변경이 오면 이 시험이 먼저 막는다.
- 다음 회차가 알 것: `timetravel_db_test.go` 의 `len(l) != 4` 가 링크 키 수를 고정하므로 `orbitLinks` 에 키를 더하면 무관한 변경이 여기서 먼저 빨개진다. 그리고 이 시험은 CI(postgres 없음)에서 늘 SKIP — 초록은 돌았다는 뜻이 아니다.
- [러너 22:16] review approved — 리뷰 승인 (risk=low)
- [러너 22:16] pr created — https://github.com/hkjang/orbit/pull/12
- [러너 22:18] ci passed — 검사 1개 모두 success
- [러너 22:18] merge done — c279ef8
