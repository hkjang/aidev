# 과제서 (2026-09-25 정찰, base main@234387a / VERSION 1.8.1)

- 과제: API 키 만료 안내가 **받을 주소가 없는 소유자의 키까지 "안내함"으로 표시해** 그 키를 영구히 삼키는 것을 막는다 (가치 3 / 위험 2 / 작업량 S)

- 왜: `internal/store/mail.go:ExpiringAPIKeys`(159행)는 `UPDATE api_keys SET expiry_notified_at=now() … RETURNING`으로 **보내기 전에 표시부터** 한다. 그런데 수신자 해석은 그 뒤 `internal/mail/service.go:resolve`(238행)가 `Store.UserEmails`(`internal/store/mail.go:30`, `WHERE id=ANY($1) AND active AND email<>''`)로 하므로, 소유자가 비활성이거나 `users.email`이 빈 문자열(`migrations/001_initial.sql:10` — `email text NOT NULL DEFAULT ''`)이면 주소가 0개가 되고 `Notify`는 `mail_deliveries` 기록도 경고 로그도 없이 그냥 돌아간다(`service.go:122-124`). 키 행에는 `expiry_notified_at`이 이미 박혀 있으므로 나중에 관리자가 그 사용자의 메일 주소를 채워 넣어도 다음 시간별 스캔에서 다시 뽑히지 않고, 키는 조용히 만료돼 자동화가 401을 받기 시작한다. 릴레이 장애로 실패한 경우에는 최소한 `mail_deliveries`에 failed 행이 남지만(주석이 근거로 삼는 보상), **주소 없는 소유자는 어디에도 흔적이 남지 않는 것**이 이 결함의 핵심이다.

- 수용 기준:
  1) 만료가 창 안이고 소유자의 `users.email`이 `''`인(또는 `active=false`인) 활성 키는 `ExpiringAPIKeys`가 돌려주지 않고 **`expiry_notified_at`도 NULL로 남는다**.
  2) 그 뒤 소유자의 `email`을 채우고 같은 창으로 다시 부르면 그 키가 **그때 한 번 나온다**(안내 기회가 보존됨).
  3) 종전 동작 무변경: 주소가 있는 소유자의 창 안 키는 여전히 한 번만 나오고(두 번째 호출은 0건), 창 밖(30일 뒤 만료)·비활성 키·`expires_at IS NULL` 키는 나오지 않는다.
  4) 테스트는 실제 PostgreSQL에서 돈다(`JUPIQ_INTEGRATION_TEST_DSN`). 수정을 되돌리면(= WHERE 절의 소유자 조건 제거) 1)과 2)가 실제로 **실패**하는 것을 확인하고 그 사실을 요약에 적을 것.
  5) `go vet ./...`, `gofmt -l .` 무출력, `make test-integration` 통과. store 통합 테스트의 SKIP 수가 0인지 `-v`로 확인할 것(프로필의 검증 함정).

- 건드릴 파일:
  - `internal/store/mail.go:ExpiringAPIKeys` — `UPDATE api_keys … WHERE` 에 소유자 조건 한 줄을 더한다. 권장 형태: `AND user_id IN (SELECT id FROM users WHERE active AND email<>'')`. 인자 번호(`$1`)는 그대로 두어 호출부가 바뀌지 않게 한다. 함수 주석(156-158행)의 "돌려준 키는 안내한 것으로 표시하므로…"에 "보낼 주소가 있는 소유자의 키만 표시한다"는 조건을 덧붙인다.
  - `internal/store/mail_integration_test.go:TestMailStoreContractsIntegration` (148-170행의 "만료 임박 키" 구획) — **여기가 함정이다.** `database.Seed`(`internal/store/store.go:124`)는 `email`을 넣지 않으므로 이 테스트의 `adminID`는 지금 `email=''`이다. 즉 수정 후에는 **기존 단언(161행 `len(keys) != 1`)이 그대로 깨진다**. 먼저 `UPDATE users SET email=$2 WHERE id=$1`로 주소를 채워 기존 계약(한 번만)을 되살리고, 그다음 주소 없는 두 번째 사용자(또는 주소를 비운 상태에서 만든 키)로 기준 1)·2)를 새로 단언할 것. 정리(`defer`)는 47-49행 패턴을 따라 `marker` 접두사로 지울 것.
  - 선택: `internal/collector/collector.go:notifyExpiringKeys`(217-234행)는 **바꾸지 않는다.** 이 과제는 store 질의 한 곳으로 끝난다.

- 검증 명령:
  ```
  docker run -d --rm --name jupiq-brief -e POSTGRES_PASSWORD=postgres -p 55432:5432 postgres:16-alpine
  export JUPIQ_INTEGRATION_TEST_DSN='postgres://postgres:postgres@127.0.0.1:55432/postgres?sslmode=disable'
  go test -count=1 -race -v -run TestMailStoreContractsIntegration ./internal/store/
  make test-integration        # = go test -count=1 -p=1 -run Integration ./internal/store ./internal/api
  go vet ./... && gofmt -l . && go test -count=1 ./...
  ```
  (도커 실행 권한이 없으면 기존 DSN을 쓸 것. DSN 없이 `go test ./...`는 이 결함의 증거가 될 수 없다 — 통합이 전부 skip된다.)

- 위험과 피할 것:
  - `internal/auth/`, `migrations/`, `.github/workflows/`는 건드리지 말 것. **마이그레이션을 새로 만들 필요가 없다** — 스키마 변경 없이 WHERE 절만으로 끝난다.
  - `mail_deliveries`·`FinishMailDelivery`·`ListMailDeliveries`·`PruneMailDeliveries`는 이번 범위 밖이다.
  - 운영자 규칙: 손으로 만든 대역이 아니라 **실제 `Store`와 실제 PostgreSQL**로 증명할 것. `internal/mail/mail_test.go`의 가짜 Sender로 이 결함을 재현하려 하지 말 것(결함은 SQL에 있고 대역은 그것을 보지 못한다).
  - `expiry_notified_at`의 "한 번만" 보장을 **약화시키지 말 것**: 표시를 발송 성공 뒤로 미루는 큰 재설계(두 단계 질의)로 확대하면 릴레이 장애 시 되풀이 발송이 생긴다. 이번에는 "주소가 있는 소유자만 표시"라는 좁은 조건만 더한다.
  - 하위 질의 대신 `JOIN users`로 쓰면 `RETURNING`의 열 이름이 모호해질 수 있다(`api_keys.id` vs `users.id`). `IN (SELECT …)` 형태를 권한다.
  - 프런트·OpenAPI·가이드 문서는 이 과제로 바뀌지 않는다(API 응답 모양 무변경).

- 차선 후보: **`ListMailDeliveries`의 limit 상한 처리를 다른 목록과 맞춘다** — `internal/store/mail.go:111`이 `if limit < 1 || limit > 200 { limit = 50 }`이라 `?limit=500`이 상한 200이 아니라 기본값 50으로 **줄어든다**(`store.go:pageBounds`가 상한으로 자르는 관례와 어긋나고, `internal/api/mail_handlers.go:20`의 `queryIntOrReject`는 음수·비정수만 막는다). `limit > 200 → 200`으로 고치고 store 단위/통합 테스트로 0·1·50·200·500·201 경계를 단언. (가치 2 / 위험 1 / 작업량 S)
