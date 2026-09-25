- 과제: `followTopic`이 DB 저장 오류를 404 `not_found`로 감추는 것을 500 `storage_error`로 분리 (가치 2 / 위험 1 / 작업량 S)
- 왜: `backend/internal/httpapi/social.go:281`의 `if err != nil || tag.RowsAffected() == 0`이 INSERT/UPDATE 실패와 "없는 Topic"을 한 덩어리로 404 "Topic을 찾을 수 없습니다"로 보고해, 저장 장애가 사용자에게 "없는 Topic"으로 보이고 운영자가 둘을 구분할 수 없다. 바로 아래 `unfollowTopic`(social.go:289~296)은 같은 테이블 쓰기 실패를 이미 500 `storage_error`로 보고하므로 같은 파일 안에서 계약이 어긋나 있고, 이는 `updatePost`(9d8a6d5)·`joinMoim`(c8d3628)·`leaveMoim`(d329380)에서 세 번 채택된 것과 같은 수정이다.

- 수용 기준:
  1) `Exec` 오류일 때 500 `storage_error`, 메시지는 `unfollowTopic`과 대칭이 되도록 같은 파일의 기존 어휘를 재사용(예: "Topic을 Link할 수 없습니다"). 새 어휘를 발명하지 말 것.
  2) `err == nil && tag.RowsAffected() == 0`(= 없는 slug)은 **기존 404 `not_found` + 기존 문구 "Topic을 찾을 수 없습니다"** 그대로. 성공 200 본문 `{"following":true,"weight":n}`과 400 `invalid_weight` 경로도 무변경.
  3) 대역 없이 실제 `New(repository…)` 배선·세션 쿠키·CSRF·pgx를 지나는 PostgreSQL integration 테스트가, **수정 전 코드에서 저장-오류 케이스만 404로 실패하고 나머지 케이스는 통과**함을 눈으로 확인한 뒤(red) 수정 후 전부 통과할 것. 최소 케이스: 신규 Link 200 / 기존 Link 가중치 변경 200(ON CONFLICT DO UPDATE 경로) / 없는 slug 404 / 저장 오류 500 + `user_topic_follows` 행 0.

- 건드릴 파일:
  - `backend/internal/httpapi/social.go:267 followTopic` — 281줄의 `if err != nil || tag.RowsAffected() == 0`을 `if err != nil { 500 }` 과 `if tag.RowsAffected() == 0 { 404 }` 두 분기로 분리. `errors`·`store` import는 이미 이 파일에 있으므로(`leaveMoim`이 `store.ErrNotFound` 사용) 추가 import 불필요하고, 여기서는 `pgx.ErrNoRows` 판별이 필요 없다(Exec은 0행을 오류로 내지 않음).
  - `backend/internal/httpapi/topic_follow_postgres_integration_test.go` (신규) — `moim_join_postgres_integration_test.go`를 본으로 복사해 쓸 것. 그 파일의 구조(`MOINA_TEST_POSTGRES_DSN` 없으면 `t.Skip`, `store.Open`, `secure.New(bytes.Repeat([]byte{61},32))`, `suffix := time.Now().UnixNano()`로 ID 충돌 회피, `t.Cleanup`에서 트리거·함수 DROP 후 행 삭제)가 그대로 맞는다.
    - 저장 오류는 손으로 만든 대역이 아니라 테스트 전용 `BEFORE INSERT ON user_topic_follows` 트리거가 sentinel user_id만 거부하도록 해서 INSERT 자체에서 낼 것 — `joinMoim` 테스트에서 검증된 방식이다. (확인된 것: `user_topic_follows`는 `001_initial.sql:149`에서 `PRIMARY KEY(user_id,topic_id)`, `weight smallint CHECK(weight BETWEEN 0 AND 100)`. **미확인(추측)**: `ON CONFLICT DO UPDATE`가 붙은 문장에서도 BEFORE INSERT 트리거가 신규 행에 대해 먼저 발화하는지 — red 단계 실행으로 참/거짓을 직접 확인하고, 만약 발화하지 않으면 `BEFORE INSERT OR UPDATE`로 바꿔 볼 것.)
  - `api/openapi.yaml:391 /api/v1/topics/{slug}/follow post` — 관례대로 `responses` 목록은 **늘리지 말고** `description` 한 줄만 추가(현재 이 post에는 `description`이 없고 `responses`는 `'200'` 하나뿐임을 확인). 바로 위 `/api/v1/topics/{slug} get`이 `'404'`를 나열하는 것을 흉내내지 말 것 — `make check`의 route 수(120)는 그대로여야 한다.

- 검증 명령 (backend 디렉터리에서):
  - throwaway DB 띄우고 DSN 주입 — `docker run -d --rm -e POSTGRES_PASSWORD=pw -p 55432:5432 postgres:17-alpine` → `MOINA_TEST_POSTGRES_DSN=postgres://postgres:pw@127.0.0.1:55432/postgres`
  - `go test -race -count=1 ./...` — **출력에서 `--- SKIP` 0줄**을 확인해야 integration이 실제로 돌았다는 증거가 된다.
  - 집중: `go test -race -count=1 -run 'TestPostgreSQL.*TopicFollow|TestPostgreSQL.*Follow' ./internal/httpapi/ -v`
  - `go vet ./...`, 루트에서 `make fmt`(변경 없음 확인), `make check`(OpenAPI route 120개 유지).
  - frontend·e2e는 무변경이므로 생략 가능 — 프런트는 `DiscoveryPages.tsx:16,69`에서 `readableError(error)`로 서버 message를 토스트에 그대로 띄우기만 하므로 코드 변경이 필요 없다(확인함).

- 위험과 피할 것:
  - `unfollowTopic`·`joinMoim`·`leaveMoim`·`getMoim`은 이번에 건드리지 말 것. 최근 3회 릴리즈가 Moim 계약을 연속으로 흔들었으므로 이번은 Topic 경로 하나만.
  - 404 문구 "Topic을 찾을 수 없습니다"는 `getTopic`(social.go:259~265)과 공유하는 사용자 향 문구다. 바꾸지 말 것.
  - 보호 경로(auth.go·oidc.go·mcp_oauth.go·store/migrations·.github/workflows) 무관 — 마이그레이션을 새로 추가할 이유가 전혀 없다. 테스트 트리거는 반드시 `t.Cleanup`에서 DROP할 것(공유 테이블이고 `go test ./...`는 패키지를 병렬 실행한다).
  - grep 결과를 증거로 제출하지 말 것. red 단계 실제 실행 출력이 증거다.
  - `input.Weight` 기본값 50·범위 검사(social.go:274~280)는 그대로 둘 것 — 이번 과제 범위 밖이다.

- 차선 후보: `getMoim`(social.go:557)의 `QueryRow(...).Scan` 오류를 `store.ErrNotFound` 기준으로 404와 500 `storage_error`로 분리. 다만 평 PostgreSQL에서 특정 SELECT만 실패시킬 수단이 없어(트리거는 SELECT에 못 걸고 RLS는 공유 테이블 전역 변경) 500 분기를 실제 DB로 재현할 수 없다는 한계를 미리 알고 고를 것(2026-09-24 회차에서 같은 벽에 부딪혔음).
