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

## 2026-09-08
- 선택: OpenBao Transit batch_input/batch_results 지원 추가 (가치 3 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `/v1/transit/{encrypt,decrypt}/{key}`가 단건만 처리해 `bao` CLI와 OpenBao SDK가 흔히 쓰는 batch 호출을 사용할 수 없었습니다. `batch_input`이 있을 때만 동작하는 추가 경로로 구현해 기존 단건 계약은 그대로 두고, 입력 순서를 유지한 `batch_results`·`reference` 반향·항목별 `error`·전부 실패 시에만 400이라는 OpenBao 의미를 맞췄으며, 미구현 파라미터(`context`, `nonce`, `associated_data`, `key_version`)는 조용히 무시하지 않고 항목 오류로 처리했습니다. batch decrypt는 감사 기록에 실패하면 평문을 전부 withhold하는 기존 보안 불변식을 유지합니다. 검증은 순서·항목 오류·빈 배열·전부 실패·감사 실패·권한 거부를 덮는 단위 테스트 5개(hook 기반, DB 불필요)를 추가하고 `./scripts/verify.sh` 전체(Go test·vet·gofmt, React test·lint·build, docs, compose)를 통과시켜 확인했습니다. 커밋 `29788a6`.
- 보류 아이디어: 로그인 성공 판정 전에 rate limiter를 succeeded로 초기화하는 순서 정리 (2/1/S) / settings GET이 주입하는 파생 필드가 PUT 왕복 시 workflow 설정에 저장되는 문제 정리 (2/1/S) / 감사 로그 보존(audit_retention_days) 자동 정리 구현 (3/3/M) / requestedOpenBaoVersion이 음수 version 쿼리를 오류 대신 latest로 처리하는 동작 정리 (2/2/S) / 데드 코드(var _ = ...) 제거와 관리 화면의 'v0.2.0 프리뷰' 문구 최신화 (1/1/S)

- 릴리즈: v0.2.4 (2026-09-08, run 2026-09-08-202134-jikim-improve)
## 2026-09-09
- 선택: Transit 핸들러의 store 오류를 종류별 상태 코드로 분리 (가치 2 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `/v1/transit/{encrypt,decrypt}/{key}`가 store 오류를 종류와 무관하게 `400 + err.Error()`로 반환해 DB 장애가 "잘못된 요청"으로 보고되고 pgx·crypto 내부 오류 문자열(SQLSTATE, DSN host 등)이 클라이언트에 노출됐습니다. KV 핸들러와 같은 방식으로 `baoTransitFailure` 헬퍼를 추가해 `ErrInvalid`는 400+이유, `ErrNotFound`는 OpenBao와 동일한 400 `encryption key not found`, 나머지는 500+일반 메시지로 나누고 batch 경로도 같은 판정을 항목별 `error`에 적용했으며(전부 실패인데 서버 장애가 섞이면 400 대신 500), AEAD 인증 실패는 잘못된 입력이므로 `TransitDecrypt`에서 `ErrInvalid`로 감싸 400을 유지했습니다. 검증은 단건 4케이스·batch 서버 장애·부분 성공을 덮는 hook 기반 단위 테스트 3개를 추가하고 기존 batch 테스트의 hook 오류를 실제 store 오류 타입으로 바로잡은 뒤 `./scripts/verify.sh` 전체(Go test·vet·gofmt, React test·lint·build, docs, compose)를 통과시켜 확인했습니다. 커밋 `ab3f6b1`.
- 보류 아이디어: 로그인 성공 판정 전에 rate limiter를 succeeded로 초기화하는 순서 정리 (2/1/S) / settings GET이 주입하는 파생 필드가 PUT 왕복 시 workflow 설정에 저장되는 문제 정리 (2/1/S) / 감사 로그 보존(audit_retention_days) 자동 정리 구현 (3/3/M) / MCP tool 오류가 모든 store 오류를 err.Error() 그대로 반환하는 문제를 Transit과 같은 방식으로 정리 (2/1/S) / Transit rewrap 엔드포인트 추가 (2/3/M)

- 릴리즈: v0.2.5 (2026-09-09, run 2026-09-09-011104-jikim-improve)
## 2026-09-09
- 선택: MCP 도구 오류에서 store 내부 정보 노출 차단 (가치 2 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `mcpToolCall`이 모든 도구 실패를 `err.Error()` 그대로 `isError` 본문에 실어 보내, DB 장애 시 pgx 오류의 DSN host·SQLSTATE가 MCP 클라이언트에 노출되고 `transit.decrypt`는 서버 장애까지 감사 기록에 400으로 남겼습니다. Transit 핸들러의 `baoTransitFailure`와 같은 방식으로 `mcpToolFailure`를 추가해 store sentinel(ErrInvalid/ErrNotFound/ErrForbidden/ErrUnauthorized/ErrConflict)과 디스패치가 직접 만든 오류만 사유를 전달하고 나머지는 "도구를 실행할 수 없습니다"로 접었으며, 디스패치 sentinel이 자기 상태 코드를 들고 다니게 해 `transit.decrypt` 감사 StatusCode가 실제 실패 종류를 따르도록 했습니다. 검증은 누출 차단·감사 상태 500·ErrInvalid 사유 보존·sentinel 매핑을 덮는 hook 기반 단위 테스트 4개를 추가하고(전용 `mcp_tool_errors_test.go`) `./scripts/verify.sh` 전체(Go test·vet·gofmt, React test·lint·build, docs, compose)를 통과시켜 확인했습니다. 곁들여 `transit.encrypt`가 테스트 seam을 우회해 `s.store`를 직접 부르던 것을 `s.encryptTransit`로 맞추고 데드 코드 `var _ model.User`를 제거했으며, api-guide에 도구 실패 텍스트 계약을 문서화했습니다. 커밋 `c5926b5`.
- 보류 아이디어: 로그인 성공 판정 전에 rate limiter를 succeeded로 초기화하는 순서 정리 (2/1/S) / settings GET이 주입하는 파생 필드가 PUT 왕복 시 workflow 설정에 저장되는 문제 정리 (2/1/S) / 감사 로그 보존(audit_retention_days) 자동 정리 구현 (3/3/M) / MCP secrets.metadata가 SecretVersions 실패 시에도 result를 채워 부분 결과와 오류가 섞이는 경로 정리 (2/1/S) / Transit rewrap 엔드포인트 추가 (2/3/M)

- 릴리즈: v0.2.6 (2026-09-09, run 2026-09-09-082103-jikim-improve)
## 2026-09-09
- 선택: 실패한 capability 확인을 permission denied가 아닌 서버 장애로 보고 (가치 3 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: OpenBao KV·Transit 핸들러 9곳과 MCP 도구 3곳이 `allowed, err := CanAccess(...)` 결과를 `err != nil || !allowed`로 접어 DB 장애로 policy 조회가 실패해도 403 `permission denied`를 돌려줬습니다. 호출자는 정책 오설정을 쫓게 되고 재시도 가능한 서버 장애가 종결 오류로 보이므로, 앞선 v0.2.5·v0.2.6의 오류 매핑 정리와 같은 방식으로 `baoAllow`·`baoAccessFailure`와 `mcpAccessFailure`를 추가해 판정 불가와 거부를 나눴습니다(sentinel은 기존대로 403, ErrInvalid는 400+이유, 나머지는 500 `failed to check permissions`이며 driver 문자열 미노출, MCP transit.decrypt 감사 StatusCode도 실제 실패 종류를 따름). 곁들여 userpass login이 SecurityConfig 조회 실패를 "local login is disabled"로 보고하던 것을 500으로 분리하고 secrets.metadata의 부분 결과 경로를 막았으며, compatibility.md에 판정 계약을 문서화했습니다. 검증은 hook 기반 단위 테스트 6개(`openbao_access_errors_test.go`) 추가 후 `./scripts/verify.sh` 전체(Go test·vet·gofmt, React test·lint·build, docs, compose) 통과로 확인했습니다. 커밋 `c143d70`.
- 보류 아이디어: 로그인 성공 판정 전에 rate limiter를 succeeded로 초기화하는 순서 정리 (2/1/S) / settings GET이 주입하는 파생 필드가 PUT 왕복 시 workflow 설정에 저장되는 문제 정리 (2/1/S) / 감사 로그 보존(audit_retention_days) 자동 정리 구현 (3/3/M) / SessionByToken이 DB 장애까지 ErrUnauthorized로 접어 인증 미들웨어가 장애를 403으로 보고하는 문제 정리 (3/2/M) / Transit rewrap 엔드포인트 추가 (2/3/M)

- 릴리즈: v0.2.7 (2026-09-10, run 2026-09-10-105020-jikim-approve)
## 2026-09-10
- 선택: SessionByToken·Authenticate가 DB 장애를 인증 실패로 접는 문제 수정 (가치 3 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `users.go`의 `Authenticate`와 `SessionByToken`이 `QueryRow` 실패를 원인과 무관하게 `ErrUnauthorized`로 바꿔, DB 장애가 "자격증명 거부"·"세션 만료"로 보고되고 인증 미들웨어는 401/403을, 로그인 rate limiter는 장애를 실패 시도로 계산해 5분 창 동안 계정을 잠갔습니다. v0.2.5~v0.2.7의 오류 매핑 정리와 같은 방식으로 `lookupFailed`로 `pgx.ErrNoRows`와 그 밖의 오류를 나눠 행 없음만 sentinel로 남기고, `withAuth`·`sessionStatus`는 `storeError`로 500을, `withBaoAuth`는 새 `baoSessionFailure`로 만료 토큰(403 `permission denied`)과 조회 불가(500 `failed to look up token`)를 갈랐으며, `login`·`baoUserpassLogin`은 장애를 실패 시도로 계산하지 않고 500을 반환하고 driver 문자열을 응답에 담지 않습니다. 검증은 hook 기반 단위 테스트 7개(`auth_outage_test.go` 6개 + store `lookupFailed` 1개)를 추가하고 `./scripts/verify.sh` 전체(Go test·vet·gofmt, React test·lint·build, docs, compose) 통과로 확인했습니다. 커밋 `d79383b`.
- 보류 아이디어: 로그인 성공 판정 전에 rate limiter를 succeeded로 초기화하는 순서 정리 (2/1/S) / settings GET이 주입하는 파생 필드가 PUT 왕복 시 workflow 설정에 저장되는 문제 정리 (2/1/S) / 감사 로그 보존(audit_retention_days) 자동 정리 구현 (3/3/M) / baoKVWrite가 SecretExistsByPath로 create·update capability를 고르면서 생기는 TOCTOU 정리 (2/2/S) / Transit rewrap 엔드포인트 추가 (2/3/M)

- 릴리즈: v0.2.8 (2026-09-10, run 2026-09-10-121123-jikim-improve)
