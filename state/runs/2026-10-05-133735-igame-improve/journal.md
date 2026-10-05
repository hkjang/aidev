# 회차 노트 2026-10-05-133735-igame-improve — igame
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 13:37] base pinned — main@b07a745
- [러너 13:37] autonomy release — 
- [러너 13:43] scout done — OIDC·AI 설정 저장이 행이 없으면 아무것도 쓰지 않고 200 과 감사 기록을 남기는 것을 고치기 (가치 3 / 위험

## 구현 노트
- `putOIDCSetting`·`putAISetting` 의 bare `UPDATE … WHERE key=…` 를 `putSetting` 과 같은 `INSERT … ON CONFLICT(key) DO UPDATE` 로 바꿨다. 0행 UPDATE 는 err=nil 이라 두 핸들러가 200 을 답하고 OIDC 는 "어느 그룹에 admin 을 줬는지" 까지 감사 행에 남겼다 — 일어나지 않은 변경이다. `secret=true` 는 양쪽 절반에 뒀다. 프로덕션 1파일(internal/api/admin.go) + 테스트 1 + docs/api.md 한 줄.
- **확신 없는 곳·검증 못 한 것**: (1) `scripts/smoke-settings.sh` 는 돌리지 못했다 — 살아 있는 서버와 관리자 자격이 필요하다. 그 스크립트는 **행이 있는** 왕복(저장→비운 secret 재저장→복원)만 지나가고, 그 경로가 그대로임은 `TestPutSettingsKeepTheirBehaviourWhenTheRowExists` 가 실DB 로 지킨다. (2) 행이 **없는** 상태는 테스트 안에서 `DELETE FROM system_settings WHERE key=…` 로 만든 것이고, 실제 설치에서 행이 사라지는 경로는 제품 코드에 없다 — 이 변경은 장애 수정이 아니라 견고성·감사 정직성 수정이다. PR 설명에서 운영 장애로 과장하지 말 것. (3) 두 upsert 는 `s.DB.Exec` 한 문장이라 `putSetting` 의 `RETURNING` 계약과 섞지 않았다 — 공통 헬퍼로 묶지 않은 것은 의도다.
- 일부러 하지 않은 것: 마이그레이션·seed 미변경(적용된 SQL 체크섬 계약), auth 흐름(`oidcLogin`/`oidcCallback`/`loadOIDCSetting`/`oidcUnreadable` 503)·봉인·`validateSetting`·감사 payload 필드·오류 코드 문자열 전부 미변경. `.github/workflows` 열지 않음. 차선 후보(secretbox AAD 회귀)는 1순위가 성립해 다음 회차로 넘겼다 — ideas.json 에 pending.
- 다음 역할이 조심할 것: **신규 `internal/api/setting_upsert_pg_test.go` 는 `IGAME_TEST_DSN` 이 없으면 skip 된다** — DSN 없는 `go test` 의 녹색은 이 변경의 증거가 아니다. README 절차로 PG17 과 `igame_test_extensions` 의 pgcrypto 를 준비하고 `make test-db DSN=…` 로 돌려야 한다. 프런트·SDK 미변경이라 `make lint`/vitest/web-build 는 돌리지 않았다.
- [러너 13:50] brief accepted — 채택 — 근거가 지금 코드와 정확히 일치했고(admin.go:404/509 의 bare UPDATE, 바로 앞의 의도적 `ErrNoRows` 허용, `putSetting` 의 기�
- [러너 13:50] verify passed — 검증 4개 통과 (policy)

## 비평 노트
- 확인함: 버려도 되는 PG17 을 직접 띄워 새 테스트를 **main 의 admin.go 에 얹어** 돌렸고 FAIL 출력이 주장한 증상(200 + 저장 없음 + admin_groups 를 적은 oidc.update 감사 1건)과 일치, HEAD 에서는 PASS. 러너가 비운 Go 검증도 메움 — go build ./..., gofmt, go test ./... , DSN 붙인 api+database, -race 전부 통과. 스키마(001_initial.sql:42)로 secret/updated_at 기본값과 PK 충돌 대상 유효성, secret=true 양쪽 배치 확인. 경로는 requireRole("admin") 그대로.
- 못 본 것: smoke-settings.sh 는 **실행하지 않음**(읽어서만 해소 — 행이 있는 왕복만 지나가고 그 단언 전부가 TestPutSettingsKeepTheirBehaviourWhenTheRowExists 로 실DB 고정됨). 프런트·SDK 미변경이라 lint/vitest/web-build 미실행.
- 승인이어도 남는 우려: 릴리즈 노트에 운영 장애로 쓰지 말 것 — 001_initial.sql:237 이 oidc·ai 행을 seed 하고 삭제 경로가 없어 '행 없음' 은 정상 설치에서 도달 불가. 견고성·감사 정직성 수정으로 적어야 함.
- 다음 회차 후보(이 diff 전후 동일, 이번 결함 아님): oidc/ai PUT 이 본문 `null` 을 통과시켜 issuer/client_id/admin_groups 를 조용히 비움(decodeJSON api.go:740 vs 일반 경로 admin.go:136); secret 읽기→쓰기 비트랜잭션 경합으로 방금 회전한 secret 덮어쓰기 가능; 행 없음 경로 감사 from 이 제로값으로 보임.
- [러너 13:56] review approved — 리뷰 승인 (risk=low)
- [러너 13:56] pr created — https://github.com/hkjang/igame/pull/35
- [러너 14:02] ci passed — 검사 1개 모두 success
- [러너 14:02] merge done — 1b98e36
- [러너 14:13] release published — v0.7.29
- [러너 14:30] assets verified — v0.7.29 자산 1개 (이전 v0.7.28: 1)
