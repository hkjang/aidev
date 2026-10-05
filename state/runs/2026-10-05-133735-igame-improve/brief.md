# 과제서 (2026-10-05, base main@b07a745, VERSION 0.7.28)

- 과제: OIDC·AI 설정 저장이 행이 없으면 아무것도 쓰지 않고 200 과 감사 기록을 남기는 것을 고치기 (가치 3 / 위험 2 / 작업량 S)

- 왜: `putOIDCSetting`(internal/api/admin.go:404)과 `putAISetting`(admin.go:509)은 `UPDATE system_settings SET … WHERE key='oidc'` / `WHERE key='ai'` 로만 저장한다. 행이 없으면 `Exec` 는 **에러 없이 0행**을 돌려주고 두 핸들러는 그대로 `200 {"setting":…, "client_secret_configured"/"api_key_configured":…}` 를 답하며, OIDC 쪽은 `s.audit(r,"oidc.update",…)` 로 "누가 어느 그룹에 admin 을 줬는지" 까지 **일어나지 않은 변경**을 감사 테이블에 남긴다. 같은 파일의 일반 설정 저장 `putSetting`(admin.go:158-161)은 이미 `INSERT … ON CONFLICT(key) DO UPDATE … RETURNING` 으로 쓰고 있어 같은 테이블에 두 개의 쓰기 계약이 갈라져 있고, 두 핸들러는 바로 앞에서 이전 값을 읽을 때 `pgx.ErrNoRows` 를 **의도적으로 허용**한다(admin.go:390 부근 / 493) — 행이 없을 수 있다는 것을 코드가 이미 전제하면서 쓰기만 조용히 버린다. 고치면 설정 저장이 성공을 주장할 때 실제로 저장돼 있고, 감사 기록이 실제 변경만 담는다.

- 도달성(정직하게): `migrations/001_initial.sql:237-245` 가 `oidc`·`ai` 행을 `ON CONFLICT (key) DO NOTHING` 으로 seed 하고, 제품 코드에 `system_settings` DELETE 경로는 없다(`rg 'DELETE FROM system_settings' internal` 결과 테스트 fixture 뿐 — 미확인 아님, 확인함). 따라서 **정상적으로 마이그레이션된 설치에서는 행이 존재**하며 이것은 지금 터지고 있는 장애가 아니라 "조용한 쓰기 유실 + 거짓 감사" 를 막는 견고성·감사 정직성 수정이다. PR 설명에 이 한계를 그대로 쓸 것 — 운영 장애로 과장하지 말 것.

- 수용 기준:
  1) `ai` 행이 없는 상태에서 `PUT /api/v1/admin/settings/ai` 를 보내면, 응답이 성공(200)일 때 `system_settings` 에 `key='ai'` 행이 실제로 존재하고 `value` 가 보낸 설정이며 `secret=true`·`updated_by` 가 호출한 관리자다. (현재는 200 이지만 행 수가 0 그대로)
  2) `oidc` 행이 없는 상태에서 `PUT /api/v1/admin/settings/oidc` 도 같다. 그리고 **쓰기가 실제로 반영되지 않았는데 `oidc.update` 감사 행만 남는 상태가 더는 생기지 않는다** — 성공 응답이면 설정 행과 감사 행이 함께 있고, 그 뒤 `GET /api/v1/admin/settings/oidc` 가 방금 보낸 `issuer`/`client_id`/`enabled` 를 돌려준다.
  3) 행이 **있는** 기존 경로의 동작이 그대로다: 기존 값이 갱신되고, 빈/`********` `client_secret`·`api_key` 는 저장된 값을 그대로 이어받고(새로 암호화하지 않음), `secret` 이 `true` 로 남고, `s.invalidateSetting`·`invalidateOIDCProviders` 와 감사 payload 필드가 변하지 않는다. 테스트가 이것을 회귀 가드로 증명해야 한다.
  4) 변이로 인과 확인: 고친 쓰기를 원래 `UPDATE … WHERE key=…` 로 되돌리면 1)·2) 가 다시 Red 가 되고, 되돌리면 녹색이다.

- 건드릴 파일 (프로덕션 1개):
  - `internal/api/admin.go:putOIDCSetting` — `UPDATE system_settings SET value=$1,secret=true,updated_by=$2,updated_at=now() WHERE key='oidc'` 를 `putSetting` 과 같은 upsert 로: `INSERT INTO system_settings(key,value,secret,updated_by) VALUES('oidc',$1,true,$2) ON CONFLICT(key) DO UPDATE SET value=excluded.value,secret=true,updated_by=excluded.updated_by,updated_at=now()`. `ON CONFLICT(key)` 는 `putSetting` 이 이미 쓰고 있으므로 제약은 있다. `secret=true` 를 INSERT 와 DO UPDATE **양쪽에** 명시할 것 — 한쪽만 쓰면 새로 만든 행이나 갱신된 행 중 하나가 `secret` 을 잃는다.
  - `internal/api/admin.go:putAISetting` — 같은 모양으로 `key='ai'`.
  - (선택, 범위를 넓히지 말 것) 두 곳 모두 `raw, _ := encodeSetting(in)` 로 마샬 오류를 버린다. 같은 줄을 지나가며 고치는 것은 허용하지만, 이것만으로 별도 테스트를 만들지 말고 과제의 중심을 옮기지 말 것.
  - `internal/api/<신규>_pg_test.go` — 테스트 1개 파일. 기존 헬퍼 재사용: `migratedPool`(fixture_pg_test.go), `insertTestUser`, 관리자 세션 쿠키로 실제 `Router()` 를 `httptest` 로 띄우는 기존 admin 테스트 방식(`admin_total_pg_test.go` 를 본보기로 그대로 따를 것 — 2026-10-04 에 추가된 가장 최신 선례다).
  - `docs/api.md` — 두 설정 PUT 행에 "행이 없으면 새로 만든다" 한 줄. 한 줄 넘기지 말 것.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `gofmt -l .` (무출력), `go vet ./...`, `go build ./...`
  - `go test ./cmd/... ./internal/... ./migrations/...` (DSN 없으면 PG 회귀는 skip 됨 — 이것만으로는 증명이 아니다)
  - 실DB 필수: README 절차로 버려도 되는 PostgreSQL 17 을 띄우고 `pgcrypto` 를 `igame_test_extensions` 스키마에 설치한 뒤
    `make test-db DSN='postgres://igame:igame@127.0.0.1:15432/igame?sslmode=disable&search_path=public,igame_test_extensions'`
  - 신규 테스트만: `go test ./internal/api/ -run <새TestName> -race -count=3`
  - `bash scripts/check-release-contract.sh` (v0.7.28 확인)
  - 실행 뒤 `api_test_`/`migrate_` 잔여 스키마 0개와 `pgcrypto` 보존을 확인할 것.
  - 프런트·SDK 를 건드리지 않으면 `make lint`/`make test`(vitest)/`make web-build` 는 생략 가능. `docs/api.md` 만 고치면 PDF 생성도 불필요.

- 위험과 피할 것:
  - **마이그레이션을 고치지 말 것.** seed 를 바꾸거나 새 마이그레이션을 추가하면 적용된 SQL 체크섬 계약과 충돌한다. 행이 없는 상태는 테스트 안에서 `DELETE FROM system_settings WHERE key='ai'` 로 만들고, fixture 가 호출별 전용 스키마라 다른 테스트에 번지지 않는다(2026-09-29 회차에서 격리 확인).
  - **auth 흐름 자체를 건드리지 말 것.** `oidcLogin`/`oidcCallback`/`loadOIDCSetting`/`loadAISetting` 과 2026-10-03 에 들어간 `oidcUnreadable` 503 분기는 그대로 둔다. 이번 변경은 **쓰기 한 문장**이다.
  - 봉인·검증 로직을 바꾸지 말 것: `in.APIKey == "" || in.APIKey == "********"` 캐리오버와 `s.Secrets.Seal`, `validateSetting`, 400 `invalid_ai`/`invalid_oidc` 응답은 그대로.
  - 감사 payload 필드를 늘리거나 줄이지 말 것(`enabled`/`base_url`/`model`, OIDC 의 `admin_groups` 등). 오류 **코드** 문자열(`ai_setting_unavailable`, `oidc_*`)은 프런트·테스트 계약이므로 재사용만 할 것.
  - 손으로 만든 대역·직접 주입한 `Server{}` 로 증명하지 말 것. 실제 `Router()` + 실제 PG + 실제 관리자 세션 쿠키로 왕복할 것. grep 결과를 증거로 제출하지 말 것.
  - `putSetting` 의 upsert 문을 공통 헬퍼로 묶으려 하지 말 것 — 그쪽은 `RETURNING` 으로 이전 값을 가져오고 `secret` 을 건드리지 않는 다른 계약이다. 세 곳을 통합하려다 범위가 터진다.
  - 2026-09-10 교훈: 릴리즈 워크플로를 건드리지 않으므로 릴리즈 재현은 불필요하다. `.github/workflows` 를 열지 말 것.

- 차선 후보: **secretbox `Open` 의 거부 경로와 AAD 결합 회귀 보강** (가치 2 / 위험 1 / S, 프로덕션 파일 0개). `internal/secretbox/secretbox_test.go` 는 왕복 + nonce 차이 한 테스트뿐이어서, `Open`(secretbox.go:34-48)의 `v1:` 접두사 없음 / base64 깨짐 / `len(data) < NonceSize` / ciphertext 1비트 변조 / 다른 키 거부가 전부 미검증이고, 무엇보다 **AAD `[]byte("igame:v1")` 를 Seal·Open 양쪽에서 지워도 현재 테스트가 통과한다**(컨텍스트 결합 상실이 조용히 넘어간다). `New([]byte)` 와 실제 `Seal`/`Open` 만 쓰고 DB 가 필요 없으며 `go test ./internal/secretbox/... -count=1` 로 끝난다. 1순위가 성립하지 않을 때(예: 실DB 에서 행 삭제 후 PUT 이 이미 500 을 답한다면) 이것을 고를 것.
