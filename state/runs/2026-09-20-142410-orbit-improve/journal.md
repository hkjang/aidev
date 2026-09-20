# 회차 노트 2026-09-20-142410-orbit-improve — orbit
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:24] base pinned — main@3ba13ad
- [러너 14:24] autonomy release — 

## 정찰 노트
- [정찰 14:3x] `orbitRange` 데카르트 곱(timetravel.go:140)을 골랐다. 두 회차 연속 차선이었던 이유는 "DB 없어 증명 불가"였는데, 이 환경에 docker 와 postgres:16-alpine 이미지가 있음을 확인해 `ORBIT_TEST_DATABASE_URL` opt-in 테스트로 실제 `store.Open`+`orbitRange` 를 통과시킬 수 있다. Time Travel 문서화는 안전하지만 동작 변화가 없어 차선으로 내렸고, 미래 `at` 거부는 슬라이더가 못 만드는 경우라 가치가 낮다.
- 추측으로 적은 것: 시드에 필요한 `users`/`people`/`interactions` 의 NOT NULL 열과 id 형식(uuid 여부)은 001_init.sql 을 끝까지 읽지 않아 미확인 — 구현자가 확인할 것. 미머지 09-18 브랜치의 `newDBServer` 파일명도 미확인.
- 조심할 것: `store.Bootstrap` 을 테스트 시드로 쓰지 말 것(관리자 계정·설정 행 부작용). 테스트 helper 이름은 `openTestStore` 로 두어 09-18 브랜치와 텍스트 충돌을 피하되 환경변수 이름은 같게. SQL 문자열을 소스에서 검사하는 테스트는 운영자 규칙 위반.
- 프로필은 어제(09-19) 것이 지금 코드와 일치해 다시 쓰지 않았다. 덧붙일 사실 하나: 이 실행 환경에는 docker 가 있고 postgres 16/17 이미지가 이미 있다(CI 에는 여전히 없음). `go test ./...` 는 지금 초록.
- [러너 14:28] scout done — `orbitRange` 의 사람×교류 데카르트 곱을 두 스칼라 서브쿼리 `least` 로 바꾸고, 실제 postgres 를 끼는 opt-in 테�

## 구현 노트
- [구현 14:33] `orbitRange`(timetravel.go:140) 의 `people LEFT JOIN interactions ON user_id`(사람×교류 곱)를 `least((SELECT min(created_at) FROM people WHERE user_id=$1),(SELECT min(occurred_at) FROM interactions WHERE user_id=$1))` 로 바꿨다 — `/orbit` 첫 로드마다 도는 질의라 O(N×M)→O(N+M). 시그니처·호출부·`earliest_at` 은 그대로. 커밋 1ab369a(perf(orbit)).
- 증거: docker postgres:16-alpine(55470, 끝나고 `docker rm -f` 함) 위에서 신규 `timetravel_db_test.go` 가 진짜 `store.Open`+`*Server{store}`.`orbitRange` 를 부른다. 옛 질의로 4/4 통과 → 일부러 교류 쪽 user_id 필터를 뺀 질의로 4/4 실패(테스트가 잘못된 질의를 잡음) → 새 질의로 4/4 통과. EXPLAIN ANALYZE(사람 3·교류 4): 옛 `Nested Loop Left Join (actual rows=12)`, 교류 스캔 loops=3 / 새 계획은 조인 노드 없이 InitPlan 두 개(3행+4행). 테스트 뒤 users/people/interactions 모두 0행(정리 확인).
- 확신 없는 곳: 없음에 가깝다. 굳이 적으면 — 테스트의 시각 비교는 `time.Equal` 이라 pgx 가 KST 로 돌려줘도 통과하지만, 서브테스트 이름의 한글이 `-run` 패턴에서 `_` 로 치환되는 것은 Go 표준 동작이다. CI 에는 postgres 가 없어 이 테스트는 SKIP 으로만 돈다(수용 기준대로).
- 일부러 하지 않은 것: `orbitAt` 포함 규칙(first_met)·마이그레이션·CI 워크플로는 안 건드렸다(범위 밖). 벤치마크 없음(정확성 동등 + EXPLAIN 조인 노드 제거로 충분). `relationships` 행은 FK 가 요구하지 않아 시드하지 않았다. `Bootstrap` 은 부작용 때문에 안 썼고 users 에 직접 INSERT.
- 다음 역할이 조심할 것: `TestOrbitRangeMatchesLegacyJoin` 은 `ORBIT_TEST_DATABASE_URL` 이 있어야 실제로 돈다 — 재검증하려면 `docker run -d --name orbit-recon-pg -e POSTGRES_PASSWORD=orbit -e POSTGRES_DB=orbit -p 127.0.0.1:55470:5432 postgres:16-alpine` 뒤 `ORBIT_TEST_DATABASE_URL='postgres://postgres:orbit@127.0.0.1:55470/orbit?sslmode=disable' go test -race -run TestOrbitRange -v ./internal/server/`. 헬퍼 이름(`openTestStore`)은 미머지 09-18 브랜치의 `newDBServer` 와 다르게 두었고 환경변수 이름은 같다. web 은 손대지 않았다(vitest 109·build 통과만 확인).
- [러너 14:33] brief accepted — 채택 — 근거(timetravel.go:140 의 user_id 만으로 조인, data.go:740 에서 /orbit 마다 호출, FK cascade)가 코드와 정확히 일치했고 수�
- [러너 14:33] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- [비평 14:4x] approve, risk low, blocking 없음. 확인한 것: 001_init.sql 로 새 질의와 옛 질의의 동치(interactions.person_id NOT NULL FK → 사람 0이면 교류도 0), $1 바인딩 유지, FK cascade 로 테스트 정리; 별도 postgres:16-alpine(55471, 삭제함)에서 새 테스트 4/4 통과 → 교류 서브쿼리의 user_id 필터를 지운 변이로 4/4 실패 → 원복. 끝난 뒤 users/people/interactions 0행. DB 없이 `go test -race` 는 SKIP 으로 통과.
- 못 본 것: 실제 규모의 실행 시간 차이(EXPLAIN 은 구현자 기록만 신뢰). data.go:740 호출부는 시그니처가 같아 읽기만 했다.
- 남는 우려: CI 에는 postgres 가 없어 이 테스트는 항상 SKIP — 방어는 사람이 직접 돌릴 때만 산다. ORBIT_TEST_DATABASE_URL 이 운영 DB 를 가리키면 마이그레이션과 INSERT/DELETE 가 실행되니 릴리즈 노트에 경고 한 줄.
- 다음 회차: 미머지 09-18 브랜치의 `newDBServer` 와 같은 환경변수·다른 헬퍼(openTestStore)가 공존하게 된다 — 머지 시 한쪽으로 합칠 것.
- [러너 14:35] review approved — 리뷰 승인 (risk=low)
- [러너 14:35] pr created — https://github.com/hkjang/orbit/pull/8
