# 회차 노트 2026-10-01-214225-igame-improve — igame
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:42] base pinned — main@4933595
- [러너 21:42] autonomy release — 

## 구현 노트
- 왜: 개인 API 키의 실제 권한은 요청마다 다시 읽는 `api_keys` 정책 + 소유자 role 이 정하는데, 기존 테스트는 Principal/정책을 손으로 만들어 `(&Server{}).enforceAPIKeyPermissions` 에 주입해 `authenticateAPIKey` 를 아예 지나지 않았다. 신규 `internal/api/apikey_scope_pg_test.go` 가 실제 `Router()`+실제 PG17+실제 bearer 로 정책 축소/복원·소유자 강등·키로 키 관리 금지·회전/폐기/만료를 고정한다. 제품 코드 0개(테스트 1 + README 1구절).
- 확신 없는 곳: (a) 수정 전 Red 가 없다 — 결함 수정이 아니라 공백 보강이라 인과는 변이 M1~M5 로만 증명했다. (b) 소유자 강등 테스트는 가드가 둘(`requireRole` 의 role 검사, `availableKeyPermissions` 의 role 필터)이라 한쪽만 제거하면 녹색이고 M1+M6 동시 제거에서만 Red — 특정 한 줄을 고정하지 못한다. (c) 정책 복원을 DB 직접 쓰기가 아니라 실제 PUT 으로 하므로 설정 캐시(TTL 5s) 타이밍에 의존한다. 같은 Server 인스턴스라 `invalidateSetting` 으로 즉시 반영되지만, 다중 인스턴스 가정에서는 성립하지 않는다.
- 일부러 안 한 것: 조사 중 찾은 둘을 고치지 않았다 — `loadAPIKeyPolicyContext` 가 `api_keys` 설정 부재 시 자기 기본값을 버리고 ErrNoRows 를 올려 모든 키 인증이 401 이 되는 문제(도달 경로 미증명, ideas.json 에 pending), 그리고 `touched` CTE 가 만료 키에도 `last_used_at` 을 올리는 것(계약 변경 근거 없음, rejected). 프런트·SDK 는 범위 밖이라 npm 을 돌리지 않았다.
- 다음 역할 주의: 새 테스트 4개는 **DB 가 있어야 돈다**. DSN 없는 `make test`·`-race` 는 skip 이므로 검증 증거가 아니다. 실행 절차는 README 110~114행 그대로(일회용 PG17 + `igame_test_extensions` 에 pgcrypto 선설치). 각 테스트 함수가 `migratedPool` 을 따로 불러 전용 스키마에 migration 을 적용하므로 api 패키지 소요가 24.9s → 27.4s 로 늘었다.
- [러너 21:54] verify passed — 검증 4개 통과 (policy)
