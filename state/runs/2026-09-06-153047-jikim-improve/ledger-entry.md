## 2026-09-06
- 선택: 잘못된 타입의 security·service 설정 값을 fail-closed로 거부 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `validateSetting`이 `security`·`service` 설정의 타입을 확인하지 않아 `allow_local_login: "false"`나 `session_timeout_minutes: "60"` 같은 값이 200으로 저장되고, `SecurityConfig`가 읽을 때 타입 단언에 실패해 조용히 버려졌습니다. 관리자에게는 로컬 로그인이 꺼진 것처럼 보이지만 실제로는 기본값(허용)이 유지되는 무증상 보안 오설정이라 `optionalString`·`optionalIntInRange` 헬퍼를 추가해 security의 boolean·정수·문자열 필드, service의 문자열 필드, AI의 `max_tokens`·`timeout_seconds`를 타입까지 검증하고 어긋나면 400으로 거부하게 했습니다. 검증은 `settings_validation_test.go`에 거부/수용 케이스 단위 테스트를 추가하고 `./scripts/verify.sh` 전체(Go test·vet·gofmt, React test·lint·build, docs, compose)를 통과시켜 확인했습니다. 커밋 `dbf7b03`.
- 보류 아이디어:
  - 신뢰 프록시 목록 기반 `X-Forwarded-For` 처리로 감사 로그·로그인 rate limit의 IP가 리버스 프록시 주소로 고정되는 문제 해결 (가치 4 / 위험 3 / M)
  - Transit `batch_input`/`batch_results` 지원 추가로 OpenBao 호환 범위 확대 (가치 3 / 위험 3 / M)
  - 로그인 성공 판정 전에 rate limiter를 `succeeded`로 초기화하는 순서 정리 (`login`, `baoUserpassLogin` 둘 다 해당) (가치 2 / 위험 1 / S)
  - `settings` GET이 주입하는 `four_eyes`·`required_approvals`·`supported_targets`가 PUT 왕복 시 `workflow` 설정에 그대로 저장되는 문제 정리 (가치 2 / 위험 1 / S)
  - `server.go`·`auth_handlers.go`·`settings.go`의 `var _ = ...` 데드 코드 제거와 관리 화면의 "v0.2.0 프리뷰" 문구 최신화 (가치 1 / 위험 1 / S)
