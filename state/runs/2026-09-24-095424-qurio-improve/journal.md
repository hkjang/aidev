# 회차 노트 2026-09-24-095424-qurio-improve — qurio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 09:54] base pinned — main@88a90a2
- [러너 09:54] autonomy release —

## 정찰 노트
- 고른 이유: `defer pool.Close()` 가 `t.Cleanup(DELETE…)` 보다 먼저 돌아 정리 코드가 죽어 있는 것을 최소 재현 테스트(`seq=[body defer cleanup]`)로 직접 확인했다. gofmt/딥링크/OIDC 감사 후보는 각각 a432f3f·79ce48f·98602d8 로 이미 구현돼 review-pending 이라 중복이고, mail·handoff·tracking 계열은 패키지가 main 에 아예 없어 성립하지 않는다.
- 제친 것: store 의 DSN 통일(차선)은 33개 통합 테스트 파일이 같은 `POSTGRES_DSN` 관례를 쓰는데 한 패키지만 바꾸면 dbexec·store·나머지 세 관례가 되어 운영자 규칙("모든 경로가 같게 읽는지")에 걸린다고 봤다.
- 추측으로 적은 것: `api_key_rate` 테스트가 쓰는 nonce 접두사를 확인하지 못해 psql 잔존 확인 쿼리의 `ratekey-%` 는 미확인이다 — 구현자가 소스에서 읽고 맞춰야 한다. 55432 포트 점유 가능성도 미확인.
- 조심할 것: `api_keys/approvals/auth_oidc/passwords` 네 파일은 DELETE 를 `defer` 로 나중에 등록해 LIFO 로 이미 올바르다. 고치면 회귀다.
- 증명 방식: 반드시 도커 PostgreSQL 에 붙여 red → green → 되돌려 red 를 확인할 것. 소스 문자열 검사는 증거로 인정되지 않는다. 
- [러너 09:58] scout done — `internal/store` 통합 테스트의 `defer pool.Close()` → `t.Cleanup(DELETE…)` 순서 역전 바로잡기 (가치 3 / 위험 1 / 작�

## 구현 노트
- 무엇/왜: `internal/store` 통합 테스트 네 곳의 `defer pool.Close()` 를 `t.Cleanup(pool.Close)` 로 바꿨다(커밋 522f0e5). Go 가 본문 defer 를 t.Cleanup 보다 먼저 돌려 픽스처 DELETE 가 닫힌 풀에서 조용히 실패하고 있었다 — 실제 PostgreSQL 17 에서 수정 전 잔존(users 4/api_keys 5/authentications 1) → 수정 후 0 → 되돌려 2 로 인과까지 확인했다.
- 확신 없는 곳: 잔존 카운트는 `user_id>1`(부트스트랩 시스템 계정 id=1 제외) 기준으로 셌다. `profile-%` 접두사로만 세면 `UpdateUserProfile` 이 이름을 '저장된 표시 이름' 으로 바꿔 놓아 놓친다 — 과제서의 psql 쿼리를 그대로 쓰면 거짓 0 이 나온다. 다른 세션이 같은 DB 를 쓰지 않는 전용 컨테이너 전제다.
- 일부러 안 한 것: `t.Cleanup` 안 `_, _ =` 의 오류 무시는 그대로 뒀다(수용 기준 3 — 판정 흔들지 않기). DSN env 이름·`t.Skip` 분기·다른 패키지의 같은 순서 역전(cmd/qurio, platformapi, intelligenceapi, agentapi, runtimeapi)은 범위 밖으로 두고 ideas 에 남겼다. `defer func(){DELETE}()` 패턴 네 파일은 이미 올바르므로 손대지 않았다.
- 다음 역할 주의: 이 네 테스트는 `-tags=integration` + 실제 PostgreSQL 이 있어야 돈다(없으면 여전히 Skip). 검증은 도커 postgres:17-alpine 을 127.0.0.1:55432 에 띄우고 `POSTGRES_DSN`(+ 전체 스위트용 `QURIO_TEST_POSTGRES_DSN`, `QURIO_INTEGRATION_DSN`) 을 준 뒤 `cmd/qurio` 마이그레이션 테스트로 부트스트랩해서 했다. 컨테이너는 세션 끝에 제거했다.
- [러너 10:04] brief accepted — 채택 — 과제서가 지목한 네 지점(28/119/226행, 31행)이 코드와 정확히 일치했고 수용 기준 1~4 를 실제 PostgreSQL 로 red→green�
- [러너 10:07] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인함: 최소 재현으로 order=[body-defer cleanup-delete cleanup-close] 를 직접 증명했고, 도커 PostgreSQL 17 전용 DB 에서 main 잔존 4/5/1 → 브랜치 0 으로 인과를 재현했다. CI 형태(3개 DSN, -race -p=1 -tags=integration ./internal/store)도 통과하고 패키지 전체 실행 후 잔존 0.
- 구현자가 의심한 자리는 모두 무해했다: 남는 users 1행은 마이그레이션이 심는 id=1 Qurio System 계정이라 user_id>1 기준이 옳고, cleanup 의 DELETE 가 전부 context.Background() 라 본문 defer cancel() 에 취소되지 않는다(이번 변경의 유일한 함정을 비껴갔다).
- 못 본 것: 전체 ./... 통합·web·e2e·Oracle 경로는 돌리지 않았다(변경이 닿지 않음). 스카웃이 추측으로 남긴 ratekey-% 접두사는 실제로 durable-rate-/concurrent-rate-/rotation-rate- 였고, 나는 접두사 무관 전수 카운트로 우회했다.
- 승인이어도 남는 우려: 같은 순서 역전이 cmd/qurio·httpapi·agentapi·runtimeapi·intelligenceapi·platformapi·integrations·domain/legacy 의 통합 테스트 25곳 남아 있어 CI DB 는 여전히 쌓인다(다음 회차 1순위). 또 cleanup 의 _, _ = 오류 무시가 그대로라 누가 되돌려도 테스트는 조용히 통과한다 — 회귀 방지 장치 없음.
- 릴리즈 노트: 테스트 전용 변경으로 사용자 영향 없음. revert 로 완전 복구 가능.
- [러너 10:10] review approved — 리뷰 승인 (risk=low)
- [러너 10:10] pr created — https://github.com/hkjang/qurio/pull/21
- [러너 10:29] ci timeout — 제한 시간 안에 CI 완료를 확인하지 못함
