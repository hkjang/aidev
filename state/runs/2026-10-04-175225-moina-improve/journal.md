# 회차 노트 2026-10-04-175225-moina-improve — moina
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 17:52] base pinned — main@1f1336e
- [러너 17:52] autonomy release — 
- [러너 17:57] scout done — `storage_error` 500 출구가 pg 오류를 버려 운영자 로그에 원인이 남지 않는 것을 `writeStorageError` 한 곳으로 모�

## 구현 노트
- 무엇/왜: `storage_error` 500 출구가 `err`를 버려 운영자에게 원인이 안 남던 것을 `writeStorageError`(신규 `httpapi/storage_error.go`)로 모았다 — 응답은 `writeError(w,500,"storage_error",msg)`와 바이트 단위로 동일하고, `error_code`·`handler`·`cause_type`·`pg_code`(SQLSTATE)를 `request_id`와 함께 한 줄 더 남긴다. 적용은 일곱 출구(resolveReport 4 + adminResetPassword + changePassword + followTopic).
- 확신 없는 곳: (1) `adminResetPassword`·`changePassword`·`followTopic` 세 출구의 **로그**는 integration으로 확인하지 않았다 — 응답 불변만 기존 테스트가 지킨다(헬퍼 자체는 단위 테스트가, resolveReport는 실제 경로 red→green으로 증명). (2) `oidc_discovery_error.go`의 `deepestOIDCDiscoveryErrorType`을 새 `deepestErrorType`에 위임시켰다(과제서는 복사를 권했다) — 같은 코드라 동작은 동일하고 OIDC 테스트가 `cause_type`을 통과로 지켰지만, 비평가가 먼저 볼 곳은 여기다. (3) integration 테스트가 `slog.SetDefault`로 전역 default logger를 바꾼다 — `t.Cleanup`으로 복원하고 `httpapi`에 `t.Parallel()`이 0건임을 확인했으나 앞으로 병렬 테스트가 추가되면 깨질 수 있다. 버퍼는 mutex로 감쌌고 `-race` 통과.
- 일부러 안 한 것: 나머지 약 150개 `storage_error` 출구(파일 15개 = 재작업 위험 구간이라 후속 과제로 ideas.json에 남김), OpenAPI(새 status code 없음), 프런트·e2e(응답 불변이라 `readableError` 경로 무변경), `store`에 SQLSTATE 헬퍼 추가(프로덕션 파일 수 억제 — `pgSQLState`는 `httpapi`에 비공개).
- 다음 역할 주의: `TestStorageError*` 3건은 DB 없이 돈다. `TestPostgreSQLAdminResolveReport…`의 새 로그 단언은 **DSN이 있어야** 돈다(없으면 `t.Skip`). 전체 검증은 `postgres:16-alpine` DSN으로 한 번 돌렸다 — 0 FAIL / 0 SKIP / 최상위 `TestPostgreSQL*` 44건 유지 / `make check` route 120개 유지.
- [러너 18:04] brief accepted — 채택 — 과제서의 근거가 현재 코드와 정확히 일치했습니다(`oidc_discovery_error.go:97`·`:116`의 관례, 일곱 출구의 줄 번호와 m
- [러너 18:04] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인함: 헬퍼를 로그 없는 passthrough로 되돌려 integration red를 직접 재현(저장 실패 2케이스만 FAIL, 복원 후 6/6 PASS) — 새 단언이 바뀐 경로를 실제로 지납니다. DSN(postgres:16-alpine)으로 `go test -race -count=1 ./... -v` 전량: 0 FAIL / SKIP 0줄 / `TestPostgreSQL*` 44건 PASS, build·vet 통과. 워크트리 clean 복원.
- 구현자의 불확실 3건 모두 해소: (1) 세 출구의 로그는 integration 미검증이지만 일곱 호출부 전부 `err != nil` 분기 안이고 `handler` 문자열이 함수명과 일치, 헬퍼는 단위 테스트가 지킴. (2) OIDC 위임은 코드 동일·OIDC 테스트 전량 PASS. (3) 전역 `slog.SetDefault`는 `t.Parallel()` 0건·Cleanup LIFO 순서(logger 복원→트리거 DROP→pool Close) 확인으로 현재 안전.
- 못 본 것: `make lint`(staticcheck 미설치, 오프라인)·프런트·e2e. 이 diff가 Go 7파일에 한정되고 프런트/OpenAPI/migration 무변경이라 CI gate에 맡겼습니다.
- 승인이어도 남는 우려: ~160개 중 7개만 변환됐으므로 **릴리즈 노트에 "일부 출구"임을 명시**할 것. 새 줄이 `ErrorContext`라 500마다 level=ERROR 줄이 새로 생깁니다(관례 원본은 WarnContext) — 알림 규칙이 있으면 영향.
- 다음 회차: `httpapi`에 `t.Parallel()`을 추가하면 `admin_report_resolve_postgres_integration_test.go:127`의 전역 logger 교체가 깨집니다. resolveReport 네 출구는 handler 값이 같아 실패 지점 구분이 안 되니 후속 이관 때 구분자를 고려.
- [러너 18:08] review approved — 리뷰 승인 (risk=low)
- [러너 18:08] pr created — https://github.com/hkjang/moina/pull/40
