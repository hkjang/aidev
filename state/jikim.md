# jikim 자율 개선 기록

## 2026-09-05
- 선택: SQL LIKE 와일드카드 이스케이프로 OpenBao LIST prefix 범위 이탈 수정 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `ListSecretChildren`이 caller prefix를 그대로 `LIKE` 패턴에 넣어 `_`, `%`가 와일드카드로 해석됐고, `kvlike_x` prefix LIST가 `kvlikeXx` 하위 key 이름까지 반환했습니다(감사 로그·Secret 검색어도 동일 문제). `internal/store/store.go`에 `escapeLike`, `secrets.go`에 `secretChildrenPattern`을 추가해 세 호출지점 모두 문자 그대로 대조하도록 고쳤습니다. 검증은 `logic_test.go` 단위 테스트 추가 + 일회용 postgres:17-alpine 컨테이너에 `JIKIM_TEST_POSTGRES_DSN`을 걸어 통합 테스트를 실행해 수정 전 FAIL / 수정 후 PASS를 확인했고, `./scripts/verify.sh` 전체(Go test·vet·gofmt, React test·lint·build, docs, compose)가 통과했습니다. 커밋 `6f66501`.
- 보류 아이디어:
  - `security` 설정의 잘못된 타입 값(예: `allow_local_login: "false"`)이 오류 없이 저장되고 조용히 무시되는 문제를 검증으로 막기 (가치 3 / 위험 2 / S)
  - 신뢰 프록시 목록 기반 `X-Forwarded-For` 처리로 감사 로그·로그인 rate limit의 IP가 리버스 프록시 주소로 고정되는 문제 해결 (가치 4 / 위험 3 / M)
  - Transit `batch_input`/`batch_results` 지원 추가로 OpenBao 호환 범위 확대 (가치 3 / 위험 3 / M)
  - `baoUserpassLogin`에서 `AllowLocalLogin` 확인 전에 rate limiter를 `succeeded`로 초기화하는 순서 정리 (가치 2 / 위험 1 / S)
  - `server.go`의 `var _ = fmt.Sprintf`, `auth_handlers.go`의 `var _ = store.ErrUnauthorized` 같은 데드 코드 제거 (가치 1 / 위험 1 / S)
- 릴리즈: v0.2.2 (2026-09-05)
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

- 릴리즈: v0.2.3 (2026-09-06, run 2026-09-06-153047-jikim-improve)
## 2026-09-07
- 선택: 신뢰 Reverse Proxy 설정 기반 X-Forwarded-For 처리로 감사 로그·로그인 rate limit IP 고정 문제 해결 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `remoteIP`가 `RemoteAddr`만 사용해 운영 권장 구성인 TLS Reverse Proxy 뒤에서는 감사 로그의 IP가 항상 Proxy 주소로 기록되고 로그인 실패 제한 키의 IP 성분도 모든 클라이언트에서 동일해졌습니다. `X-Forwarded-For`를 무조건 신뢰하면 누구나 IP를 위조할 수 있어 환경변수 대신 `security` 설정에 `trusted_proxies`(CIDR·IP 목록)를 추가하고, 접속 주소가 등록 대역일 때만 전달 체인을 오른쪽에서 왼쪽으로 훑어 신뢰 대역 밖 첫 주소를 클라이언트로 판정하도록 `clientIP`를 새로 만들었습니다(목록이 비면 기존 동작 유지, 30초 TTL 캐시). 저장소의 "환경변수 네 개" 제품 계약을 깨지 않으려고 설정 값으로 두었고 관리 화면 보안 탭에 입력 필드와 안내를 추가했습니다. 검증은 `clientIP` 판별 10개 케이스·rate key 분리·`trusted_proxies` 검증/파싱 단위 테스트를 추가하고 `./scripts/verify.sh` 전체(Go test·vet·gofmt, React test·lint·build, docs, compose)를 통과시켜 확인했습니다. 커밋 `aa0eccf`, 릴리스 커밋 `a20a367`(v0.2.4).
- 보류 아이디어: Transit batch_input/batch_results 지원으로 OpenBao 호환 범위 확대 (3/3/M) / 로그인 성공 판정 전에 rate limiter를 succeeded로 초기화하는 순서 정리 (2/1/S) / settings GET이 주입하는 파생 필드가 PUT 왕복 시 workflow 설정에 저장되는 문제 정리 (2/1/S) / 감사 로그 보존(audit_retention_days) 자동 정리 구현 (3/3/M) / requestIsHTTPS가 신뢰 Proxy 여부와 무관하게 X-Forwarded-Proto를 신뢰하는 부분을 trusted_proxies와 일관되게 정리 (2/3/S)

