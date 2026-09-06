# ai-admin 자율 개선 기록

## 2026-09-02
- 선택: 내장 SPA 번들 캐시 헤더 적용과 디렉터리 목록 노출 차단 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `internal/server/server.go`의 `spa` 핸들러가 `embed.FS` 파일을 아무 캐시 헤더 없이 제공해(embed는 ModTime이 0이라 Last-Modified/ETag도 없음) 화면을 열 때마다 약 1MB 번들을 전부 다시 내려받았고, `/assets`·`/fonts` 같은 디렉터리 경로는 `http.FileServer`가 파일 목록을 노출했다. 내용 해시가 붙은 `assets/`는 `max-age=31536000, immutable`, 이름이 고정된 `fonts/`는 `max-age=604800`, `index.html`과 SPA fallback은 `no-cache`로 나누고 `fs.Stat`로 일반 파일만 서빙하도록 고쳤다. 실제 내장 번들을 대상으로 하는 회귀 테스트를 추가해 수정 전 코드에서 실패하는 것까지 확인했고, `go vet`·`go test ./...`·`scripts/verify-version.sh`·`npm test`(55개)·`npm run build`를 모두 통과시켰다. 저장소 관례에 따라 VERSION을 1.2.1로 올리고 CHANGELOG·README·docs·web 버전 메타데이터를 함께 맞췄다.
- 보류 아이디어:
  - `safeCSVCell`이 OWASP가 함께 권고하는 tab(0x09)·CR(0x0D) 선행 문자를 중화하지 않음 (가치 2 / 위험 1 / S)
  - 여러 핸들러의 길이 검증이 rune이 아닌 byte 기준이라 한글 입력이 의도한 한도의 약 1/3에서 거부됨(예: 공급자 이름 160) (가치 3 / 위험 2 / M)
  - `clearSessionCookies`가 `setSessionCookies`와 달리 `Secure` 플래그를 설정하지 않아 `security.cookie_secure` 활성 배포에서 비대칭 (가치 2 / 위험 2 / S)
  - CI에 정적 분석 단계(`go vet`, `golangci-lint`, eslint)가 없어 회귀를 테스트로만 잡고 있음 (가치 3 / 위험 1 / M)
  - `r.NotFound(s.spa)`가 GET 외 메서드도 받아 알 수 없는 경로로의 POST가 200 + index.html을 반환 (가치 2 / 위험 2 / S)
- 릴리즈: v1.2.1 (2026-09-02, 태그 사후 푸시)

## 2026-09-02
- 선택: 길이 검증을 바이트가 아닌 문자 수 기준으로 통일 (가치 3 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `keys.go`/`providers.go`/`users.go`/`workflow.go`/`mcp.go`의 사용자 입력 길이 검증이 `len()`으로 바이트를 세어, PostgreSQL `varchar(n)`(문자 단위)과 "1~190자여야 합니다" 같은 한국어 오류 안내가 모두 문자를 세는데 서버만 바이트를 세는 불일치가 있었다. 그 결과 한글 표시 이름은 190자가 아니라 63자에서, 공급자·API 키 이름은 160자가 아니라 53자에서 거부되었다(프런트엔드에 maxLength가 없어 사용자는 입력 후에야 400을 받음). 이미 `resource_mutation.go`·`settingscatalog`가 쓰던 `utf8.RuneCountInString`으로 8곳(키 이름 160, 공급자 이름 160, 기본 모델 240 2곳, 표시 이름 190 2곳, 이메일 320, 승인 제목 300 2곳)을 바꿨고, 역할 코드·OIDC state/code 같은 프로토콜·식별자 값은 의도적으로 바이트 기준을 유지했다. 한글 경계값(한도 정확히 통과/한도+1 거부) 단위 테스트를 추가해 수정 전 코드에서 실패하는 것을 실제로 확인했고, PostgreSQL 통합 테스트에도 키 이름·표시 이름·승인 제목 경계 케이스를 넣었다. `go vet`·`go test -count=1 ./...`·`go build ./...`·`scripts/verify-version.sh`·`npm test`(55개)·`npm run build`를 모두 통과시켰다. 저장소 관례에 따라 VERSION을 1.2.2로 올리고 CHANGELOG·README·docs·web 버전 메타데이터를 함께 맞췄다(직전 릴리즈 커밋과 동일하게 `internal/ui/dist`는 재빌드하지 않음).
- 보류 아이디어:
  - `safeCSVCell`이 OWASP가 함께 권고하는 tab(0x09)·CR(0x0D) 선행 문자를 중화하지 않음 (가치 2 / 위험 1 / S)
  - `clearSessionCookies`가 `setSessionCookies`와 달리 `Secure` 플래그를 설정하지 않아 `security.cookie_secure` 활성 배포에서 비대칭 (가치 2 / 위험 2 / S)
  - CI에 정적 분석 단계(`go vet`, `golangci-lint`, eslint)가 없어 회귀를 테스트로만 잡고 있음 (가치 3 / 위험 1 / M)
  - `r.NotFound(s.spa)`가 GET 외 메서드도 받아 알 수 없는 경로로의 POST가 200 + index.html을 반환 (가치 2 / 위험 2 / S)
  - 공급자 `availableModels` 배열이 항목 수·항목 길이를 전혀 검증하지 않아 2MB 본문 한도까지 임의 크기 JSON이 저장됨 (가치 3 / 위험 2 / S)
- 릴리즈: v1.2.2 (2026-09-02, 태그 사후 푸시)
- 릴리즈: v1.2.2 (2026-09-02)

## 2026-09-03
- 선택: AI 공급자 `availableModels` 배열 검증·정규화 추가 (가치 3 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `providers.go`의 `availableModels`는 항목 수도 항목 길이도 전혀 검증하지 않아 2MB 본문 한도까지 임의 크기 JSON이 `ai_admin.ai_provider.available_models` jsonb에 저장되었고, 저장된 값은 `/api/v1/models`·`/api/v1/ai/catalog`·chat 허용 모델 검사가 호출될 때마다 언마샬+정렬되었다. `chatCompletions`가 240자를 넘는 모델 이름을 이미 거부하므로 그보다 긴 이름은 저장되어도 쓸 수 없는 값이었다. `validateProviderRequest`에 항목 수 200개·이름 240자 한도를 넣고, `normalizeProviderRequest`에서 읽는 쪽이 이미 쓰던 `uniqueStrings`(트림·빈 항목 제거·중복 제거·정렬)를 저장 전에도 적용했다. 세 경로(생성, 수정, 승인 적용 `workflow.go:649`)가 모두 이 두 함수를 거치므로 한곳 수정으로 전부 덮인다. 정규화 결과와 200개/240자 경계값 단위 테스트를 추가해 수정 전 코드에서 실패하는 것을 실제로 확인했고, `go vet`·`go build ./...`·`go test -count=1 ./...`·`scripts/verify-version.sh`·`npm test`(55개)·`npm run build`를 모두 통과시켰다. 저장소 관례에 따라 VERSION을 1.2.3으로 올리고 CHANGELOG·README·docs·web 버전 메타데이터를 함께 맞췄으며, `docs/api.md`에 새 한도를 명시했다(직전 릴리즈 커밋들과 동일하게 `internal/ui/dist`는 재빌드하지 않음).
- 보류 아이디어:
  - `safeCSVCell`이 OWASP가 함께 권고하는 tab(0x09)·CR(0x0D) 선행 문자를 중화하지 않아, `\t=cmd|...` 같은 값이 기존 방어를 우회함(Go `csv.Writer`는 tab만 있는 필드를 인용하지 않음) (가치 3 / 위험 1 / S)
  - `clearSessionCookies`가 `setSessionCookies`와 달리 `Secure` 플래그를 설정하지 않아 `security.cookie_secure` 활성 배포에서 비대칭 (가치 2 / 위험 2 / S)
  - CI에 정적 분석 단계(`go vet`, `golangci-lint`, eslint)가 없어 회귀를 테스트로만 잡고 있음 (가치 3 / 위험 1 / M)
  - `r.NotFound(s.spa)`가 GET 외 메서드도 받아 알 수 없는 경로로의 POST가 200 + index.html을 반환 (가치 2 / 위험 2 / S)
  - 공급자 HTTP 통합 테스트가 없어 `POST/PATCH /api/v1/ai/providers`의 낙관적 잠금·검증 경로가 단위 테스트로만 검증됨 (가치 3 / 위험 1 / M)
- 릴리즈: v1.2.3 (2026-09-03, 태그 사후 푸시)
- 릴리즈: v1.2.3 (2026-09-03)

## 2026-09-03
- 선택: 지원하지 않는 HTTP 메서드 응답을 JSON 오류 봉투로 통일하고 SPA fallback의 메서드 제한 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `Handler()`가 chi 기본 methodNotAllowed responder를 그대로 써서 `PUT /api/v1/settings`나 `DELETE /health/live`가 본문·Content-Type 없는 405를 반환했다. 저장소가 `docs/api.md`에 문서화한 `{"error":{"code","message"}}` 봉투를 파싱하는 클라이언트(프런트엔드 `parseResponse` 포함)는 이 응답을 읽지 못했고, chi가 내려주던 `Allow` header는 같은 경로의 다른 메서드를 빠뜨려 `PUT /api/v1/settings`에 `Allow: GET`만 주고 `PATCH`를 누락했다. 또 `r.NotFound(s.spa)`가 메서드를 가리지 않아 `POST /dashboard` 같은 알 수 없는 경로로의 쓰기 요청이 200 + `index.html`을 받아 성공한 것처럼 보였다. chi는 커스텀 handler 등록 시 `Allow`를 직접 설정하지 않고 `methodsAllowed`를 export하지 않으므로, `chi.Routes.Match`로 7개 메서드를 라우터에 직접 질의해 `Allow`를 계산하는 `s.methodNotAllowed`를 등록했고, `spa`는 GET·HEAD만 화면을 제공하고 나머지에 405를 반환하도록 했다(`/api/`·`/v1/` 알 수 없는 경로는 기존대로 메서드 무관 404 `not_found` 유지). 실제 라우터를 세워 405 상태·`Allow`·JSON 봉투·SPA 화면 제공을 검증하는 테이블 테스트를 추가하고, 수정 전 코드에서 4개 케이스가 실패하는 것을 직접 확인했다. `go vet`·`go build ./...`·`go test -count=1 ./...`·`go test -race`·`scripts/verify-version.sh`·`npm ci && npm test`(55개)·`npm run build`를 모두 통과시켰다. 저장소 관례에 따라 VERSION을 1.2.4로 올리고 CHANGELOG·README·docs·web 버전 메타데이터를 맞췄으며 `docs/api.md`에 405 계약을 명시했다(직전 릴리즈 커밋들과 동일하게 `internal/ui/dist`는 재빌드하지 않음).
- 보류 아이디어:
  - `safeCSVCell`이 OWASP가 함께 권고하는 tab(0x09)·CR(0x0D) 선행 문자를 중화하지 않음. `docs/api.md`는 "formula injection 위험 문자를 중화합니다"라고 명시하고 있어 문서와 구현이 어긋남 (가치 3 / 위험 1 / S)
  - HEAD 요청이 모든 GET 라우트에서 405. `apiScopeForPermission`이 `http.MethodHead` 분기를 갖고 있으나 chi가 GET에서 HEAD를 유도하지 않아 사문화됨. 미들웨어로 HEAD→GET 재라우팅 시 OIDC callback 등 부작용 검토 필요 (가치 3 / 위험 3 / M)
  - `clearSessionCookies`가 `setSessionCookies`와 달리 `Secure`를 설정하지 않고 CSRF 쿠키의 `SameSite`도 Strict가 아닌 Lax로 지움 (가치 2 / 위험 2 / S)
  - 공급자 HTTP 통합 테스트가 없어 `POST/PATCH /api/v1/ai/providers`의 낙관적 잠금·검증 경로가 단위 테스트로만 검증됨 (가치 3 / 위험 1 / M)
  - CI에 정적 분석 단계(`go vet`, `golangci-lint`, eslint)가 없어 회귀를 테스트로만 잡고 있음 (가치 3 / 위험 1 / M)
- 릴리즈: v1.2.4 (2026-09-03)

## 2026-09-03
- 선택: 감사 CSV export가 중간에 실패해도 200 OK로 잘린 파일을 내려주던 문제 수정 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `exportAuditEvents`는 200과 BOM·header를 먼저 내보낸 뒤 최대 50,000행을 스트리밍하는데, 30초 조회 제한 시간 초과나 `rows.Scan` 실패가 나면 `return`으로 조용히 끝나 운영자에게 완전한 파일처럼 보이는 잘린 감사 기록을 남겼다(성공 감사 이벤트만 생략되고 클라이언트는 정상 다운로드로 판단). 행 쓰기 루프를 `streamAuditCSV`로 분리해 기록한 행 수와 중단 사유(`row_scan_failed`, `row_stream_failed`→제한 시간 초과 시 `export_timeout`, `client_write_failed`)를 반환하게 하고, 중단 시 `audit.export`를 `failure`+`reason`+`rows`로 남긴 뒤(스트리밍 실패를 감사에 남기는 `chatCompletions`의 기존 관례와 동일) `panic(http.ErrAbortHandler)`로 응답을 끊어 브라우저 다운로드가 실패하도록 했다. 이를 위해 `recoverer`가 chi Recoverer처럼 `http.ErrAbortHandler`는 다시 panic하도록 하고, `accessLog`의 로그 기록을 defer로 옮겨 panic·중단으로 끝난 요청도 접근 로그에 남게 했다. pgx.Rows를 흉내내는 stub으로 DB 없이 잘림 보고를 검증하는 단위 테스트와 중단 신호 전달·중단 요청 접근 로그 테스트를 추가했고(수정 전 recoverer/accessLog 동작에서는 실패), `go vet`·`go build ./...`·`go test -count=1 ./...`·`go test -race ./internal/server`·`scripts/verify-version.sh`·`npm ci && npm test`(55개)·`npm run build`를 모두 통과시켰다. 저장소 관례에 따라 VERSION을 1.2.5로 올리고 CHANGELOG·README·docs·web 버전 메타데이터를 맞췄으며 `docs/api.md`에 중단 계약을 명시했다(직전 릴리즈 커밋들과 동일하게 `internal/ui/dist`는 재빌드하지 않음).
- 보류 아이디어:
  - `safeCSVCell`이 OWASP가 함께 권고하는 tab(0x09)·CR(0x0D) 선행 문자를 중화하지 않음. 다만 표시 이름 등 주요 필드가 이미 TrimSpace되어 실제 도달 경로는 좁음 (가치 2 / 위험 1 / S)
  - HEAD 요청이 모든 GET 라우트에서 405. 헬스 체크·업타임 모니터가 흔히 HEAD를 쓰므로 최소한 `/health/*`, `/api/v1/meta`에는 HEAD를 등록할 가치가 있음. 전체 GET에 HEAD→GET 재라우팅은 OIDC callback·감사 export 부작용 때문에 위험 (가치 3 / 위험 3 / M)
  - `clearSessionCookies`가 `setSessionCookies`와 달리 `Secure`를 설정하지 않고 CSRF 쿠키의 `SameSite`도 Strict가 아닌 Lax로 지움 (가치 2 / 위험 2 / S)
  - 공급자 HTTP 통합 테스트가 없어 `POST/PATCH /api/v1/ai/providers`의 낙관적 잠금·검증 경로가 단위 테스트로만 검증됨 (가치 3 / 위험 1 / M)
  - CI에 정적 분석 단계(`go vet`, `golangci-lint`, eslint)가 없어 회귀를 테스트로만 잡고 있음 (가치 3 / 위험 1 / M)
- 릴리즈: v1.2.5 (2026-09-03)
- 릴리즈: v1.2.5 (2026-09-03)

## 2026-09-04
- 선택: 상태 확인·메타 endpoint의 HEAD 요청 지원 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: chi는 GET 라우트에서 HEAD를 유도하지 않으므로 `/health/live`·`/health/ready`·두 API alias·`/api/v1/meta`·`/api/v1/openapi.json`이 HEAD 요청에 405 `method_not_allowed`를 반환했다. 로드 밸런서·가동 감시 도구는 흔히 HEAD로 확인하므로, 실제로 존재하고 부작용도 없는 경로가 없는 것처럼 보였다. `getWithHead` 헬퍼로 이 여섯 경로를 GET·HEAD에 함께 등록해 같은 상태 코드와 header를 반환하게 했고(본문은 net/http가 HTTP 규격대로 비운다), OpenAPI 문서에 `head` operation을 추가해 라우터-문서 양방향 일치를 검사하는 기존 `TestOpenAPIMatchesImplementedRoutesAndPathParameters`를 통과시켰다. 인증이 필요한 나머지 GET 경로는 감사 CSV export·OIDC callback 등 부작용 때문에 기존 계약을 유지했다. HEAD 응답 상태·Content-Type과 `Allow: GET, HEAD`를 검증하는 테스트를 추가했고, 기존 테이블 테스트의 `DELETE /health/live` 기대값이 `GET`에서 `GET, HEAD`로 바뀌는 것으로 수정 전 동작을 확인했다. `scripts/verify-version.sh`·`go vet`·`go build ./...`·`go test -count=1 ./...`·`go test -race ./internal/server`·`npm ci && npm test`(55개)·`npm run build`를 모두 통과시켰다. 저장소 관례에 따라 VERSION을 1.2.6으로 올리고 CHANGELOG·README·docs·web 버전 메타데이터를 맞췄으며 `docs/api.md`·`docs/operations.md`에 HEAD 계약을 명시했다(직전 릴리즈 커밋들과 동일하게 `internal/ui/dist`는 재빌드하지 않음).
- 보류 아이디어:
  - `safeCSVCell`이 OWASP가 함께 권고하는 tab(0x09)·CR(0x0D) 선행 문자를 중화하지 않음. 다만 표시 이름 등 주요 필드가 이미 TrimSpace되어 실제 도달 경로는 좁음 (가치 2 / 위험 1 / S)
  - `auth.truncate`가 user agent를 1000바이트로 자르면서 UTF-8 경계를 지키지 않아, 다국어 UA가 잘린 자리에서 깨지면 PostgreSQL이 세션 INSERT를 거부해 로그인이 401로 실패할 수 있음 (가치 2 / 위험 1 / S)
  - `listKeys`·`listKeyScopes`가 `rows.Scan` 실패한 행을 조용히 건너뛰고 200을 반환해, 관리자가 불완전한 API 키 목록을 완전한 목록으로 오인할 수 있음(감사 CSV 잘림과 같은 부류) (가치 3 / 위험 2 / S)
  - `clearSessionCookies`가 `setSessionCookies`와 달리 `Secure`를 설정하지 않고 CSRF 쿠키의 `SameSite`도 Strict가 아닌 Lax로 지움 (가치 2 / 위험 2 / S)
  - CI에 정적 분석 단계(`go vet`, `golangci-lint`, eslint)가 없어 회귀를 테스트로만 잡고 있음. web에는 eslint 설정 자체가 없음 (가치 3 / 위험 1 / M)
- 릴리즈: v1.2.6 (2026-09-04)
- 릴리즈: v1.2.6 (2026-09-04)

## 2026-09-04
- 선택: 목록 조회가 읽지 못한 행을 건너뛰고 200을 반환하던 문제 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `internal/server`의 29개 `rows.Next()` 루프 중 절반 이상이 `rows.Scan` 실패 행을 `continue`로 넘기고 `rows.Err()`도 확인하지 않아, API 키·키 권한·사용자·역할·권한·세션·AI 공급자·모델·승인 정책/요청·레거시 테이블/컬럼 목록이 짧아진 결과를 200으로 반환했다. 관리자는 짧은 목록을 완전한 목록과 구별할 수 없어 실제로 남아 있는 고권한 API 키나 대기 중인 승인 요청을 없는 것으로 판단할 수 있었고, 이는 v1.2.5에서 고친 감사 CSV 잘림과 같은 부류다. 행 단위 목록 생성을 제네릭 헬퍼 `collectRows`(scan 실패 즉시 중단 + `rows.Err()` 확인)로 모아 13개 지점을 옮기고, 1행→N항목이라 헬퍼가 맞지 않는 `/v1/models`는 저장소가 이미 쓰던 명시적 형태로 고쳤다. MCP 도구 3곳도 `toolFailure`로 실패를 알린다. 대시보드 위젯 3곳은 조회 실패 자체를 빈 값으로 처리하는 기존 degradation 계약이라 그대로 두었다. `collectRows`의 정상·scan 실패·stream 실패 동작을 기존 `stubAuditRows`를 재사용한 단위 테스트로 검증(수정 전 동작에서는 실패)하고, 열 개 목록 endpoint 응답을 통합 테스트에 추가했다. Docker로 PostgreSQL 16을 띄워 `TEST_POSTGRES_DSN`을 설정한 뒤 `go test -race -count=1 ./...`(통합 테스트 포함)을 실제로 통과시켰고 `gofmt -l`·`go vet`·`go build ./...`·`scripts/verify-version.sh`·`npm ci && npm test`(62개)·`npm run build`도 모두 통과했다. 저장소 관례에 따라 VERSION을 1.2.9로 올리고 CHANGELOG·README·docs·web 버전 메타데이터를 맞췄으며 `docs/api.md`에 목록 완전성 계약을 명시했다(직전 릴리즈 커밋들과 동일하게 `internal/ui/dist`는 재빌드하지 않음).
- 보류 아이디어:
  - `safeCSVCell`이 OWASP가 함께 권고하는 tab(0x09)·CR(0x0D) 선행 문자를 중화하지 않음. `docs/api.md`의 "formula injection 위험 문자를 중화합니다"와 어긋나지만 주요 필드가 TrimSpace되어 실제 도달 경로는 좁음 (가치 2 / 위험 1 / S)
  - `auth.truncate`가 user agent를 1000바이트로 자르면서 UTF-8 경계를 지키지 않아, 다국어 UA가 잘린 자리에서 깨지면 PostgreSQL이 세션 INSERT를 거부해 로그인이 401로 실패할 수 있음 (가치 2 / 위험 1 / S)
  - `clearSessionCookies`가 `setSessionCookies`와 달리 `Secure`를 설정하지 않고 CSRF 쿠키의 `SameSite`도 Strict가 아닌 Lax로 지움 (가치 2 / 위험 2 / S)
  - 대시보드 위젯 3개 조회가 실패를 빈 배열로 감춰, 통계·최근 활동이 0건인지 조회 실패인지 화면에서 구분할 수 없음 (가치 2 / 위험 2 / S)
  - CI에 정적 분석 단계(`go vet`, `golangci-lint`, eslint)가 없어 회귀를 테스트로만 잡고 있음. web에는 eslint 설정 자체가 없음 (가치 3 / 위험 1 / M)
- 릴리즈: v1.2.9 (2026-09-04)
- 릴리즈: v1.2.9 (2026-09-04)
## 2026-09-06
- 선택: 로그인 실패 원인을 자격 증명 오류와 백엔드 장애로 구분 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `POST /api/v1/auth/login`이 `s.auth.Login`의 모든 오류를 401 `invalid_credentials`로 답해, PostgreSQL 연결 실패·session 행 기록 실패·읽을 수 없는 비밀번호 해시 같은 장애가 "비밀번호가 틀렸다"로 보였고 아이디별 backoff에 누적되어 DB가 복구된 뒤에도 정상 계정이 최대 60초 더 차단될 수 있었으며 감사 기록도 `reason: invalid_credentials`로 남아 장애를 자격 증명 공격으로 오인하게 했다(추가로 `auth.Login`이 `pgx.ErrNoRows`가 아닌 조회 오류를 `passwordHash == nil` 분기로 흘려보내 DB 장애와 없는 아이디를 구별하지 못했다). `auth.Login`이 `ErrInvalidCredentials`/`ErrAccountInactive`를 구분해 반환하고 나머지는 감싸 올리도록 고친 뒤, 서버는 자격 증명 오류에만 401+backoff를, 그 밖의 장애에는 `Retry-After`와 함께 503 `login_unavailable`을 반환하고 감사 `reason`을 세 가지로 나눈다(정지 계정의 상태 코드·응답 본문은 비밀번호 오류와 동일하게 유지해 계정 열거 불가). 순수 분류 함수의 테이블 테스트와 연결 불가 pool을 향한 로그인이 503을 내고 backoff에 누적되지 않는지 확인하는 핸들러 테스트를 추가해 수정 전 동작에서 4개 케이스가 실패하는 것을 실제로 확인했고, 정지 계정의 401 본문 동일성과 `account_inactive` 감사 기록을 통합 테스트에 추가했다. Docker로 PostgreSQL 16을 띄워 `TEST_POSTGRES_DSN`을 설정한 뒤 `go test -race -count=1 ./...`(통합 테스트 포함)을 통과시켰고 `gofmt -l`·`go vet`·`go build ./...`·`scripts/verify-version.sh`·`npm ci && npm test`(62개)·`npm run build`도 모두 통과했다. 저장소 관례에 따라 VERSION을 1.2.10으로 올리고 CHANGELOG·README·docs·web 버전 메타데이터를 맞췄으며 `docs/api.md`에 로그인 실패 계약을 명시했다(직전 릴리즈 커밋들과 동일하게 `internal/ui/dist`는 재빌드하지 않음).
- 보류 아이디어: `auth.truncate`가 user agent를 UTF-8 경계 무시하고 1000바이트로 잘라 세션 INSERT가 거부될 수 있음(이제 401이 아니라 503으로 보고되지만 로그인 실패 자체는 남음) (가치 3 / 위험 1 / S) · `loadGrants`가 map 순회로 roles·permissions를 만들어 `/api/v1/auth/me` 응답 순서가 요청마다 뒤바뀌고 프로필 화면의 태그가 섞임 (가치 2 / 위험 1 / S) · `safeCSVCell`이 OWASP가 함께 권고하는 tab(0x09)·CR(0x0D) 선행 문자를 중화하지 않음 (가치 2 / 위험 1 / S) · 대시보드 위젯 3개 조회가 실패를 빈 배열·0으로 감춰 대기 중인 승인이 0건인지 조회 실패인지 구분 불가 (가치 2 / 위험 2 / S) · CI에 정적 분석 단계(`gofmt -l`, `go vet`, eslint)가 없어 회귀를 테스트로만 잡고 있음 (가치 3 / 위험 1 / M)

## 2026-09-06
- 선택: 저장할 수 없는 요청 header가 감사 기록을 지우거나 로그인을 막던 문제 수정 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: chi의 `middleware.RequestID`는 신뢰할 수 없는 `X-Request-Id` header를 그대로 추적 ID로 쓰는데, 모든 감사 이벤트가 이 값을 `ai_admin.audit_event.request_id`(`varchar(80)`)에 기록한다. 따라서 80자를 넘거나 UTF-8이 아닌 header를 보내면 PostgreSQL이 감사 INSERT를 거부해 **호출자가 자기 요청의 감사 기록만 골라 없앨 수 있었고**(`s.audit`은 실패를 warn 로그로만 남긴다), 감사를 트랜잭션에 포함하는 변경(`auditInTx`)은 통째로 500으로 실패했다. host 이름이 긴 서버에서는 chi가 생성한 ID도 같은 증상을 일으켰다. 저장·검색 가능한 형식(80자 이하, 영숫자와 `-._:/`)만 그대로 쓰고 나머지는 서버가 만든 `req-<hex>`로 대체하는 `boundedRequestID` 미들웨어를 넣었으며, 이미 같은 문자 규칙으로 추적 ID를 redirect에 넣던 `safeOIDCFailureReference`가 그 검사를 공유하도록 정리했다. 같은 부류인 User-Agent도 함께 고쳤다: `auth.truncate`가 1000 **byte**로 잘라 다국어 UA가 경계에서 깨지거나 UA에 UTF-8이 아닌 byte가 있으면 session INSERT가 거부되어 자격 증명이 맞는데도 로그인이 실패했으므로, 유효하지 않은 byte를 걸러내고 column과 같은 1000 **자** 기준으로 rune 경계에서 자르도록 했다. 미들웨어 표 테스트(7개 케이스)와 rune 절단 표 테스트, 그리고 적대적 header로 실패한 로그인이 감사 기록을 남기는지·적대적 UA로도 로그인이 성공하는지 확인하는 PostgreSQL 통합 테스트를 추가해 수정 전 코드에서 실제로 실패하는 것을 확인했다(감사 기록 0건, 로그인 401). Docker로 PostgreSQL 16을 띄워 `TEST_POSTGRES_DSN`을 설정한 뒤 `go test -race -count=1 ./...`(통합 테스트 포함)을 통과시켰고 `gofmt -l`·`go vet`·`go build ./...`·`scripts/verify-version.sh`·`npm ci && npm test`(62개)·`npm run build`도 모두 통과했다. 저장소 관례에 따라 VERSION을 1.2.11로 올리고(1.2.10은 아직 병합되지 않은 형제 branch가 사용) CHANGELOG·README·docs·web 버전 메타데이터를 맞췄으며 `docs/api.md`·`docs/security.md`에 추적 ID 정규화 계약을 명시했다(직전 릴리즈 커밋들과 동일하게 `internal/ui/dist`는 재빌드하지 않음).
- 보류 아이디어: CI에 정적 분석 단계(`gofmt -l`, `go vet`, eslint)가 없어 회귀를 테스트로만 잡고 있음 (가치 3 / 위험 1 / M) · `loadGrants`가 map 순회로 roles·permissions를 만들어 `/api/v1/auth/me` 응답 순서가 요청마다 뒤바뀌고 프로필 화면의 태그가 섞임 (가치 2 / 위험 1 / S) · `safeCSVCell`이 OWASP가 함께 권고하는 tab(0x09)·CR(0x0D) 선행 문자를 중화하지 않음 (가치 2 / 위험 1 / S) · 대시보드 위젯 3개 조회가 실패를 빈 배열·0으로 감춰 대기 중인 승인이 0건인지 조회 실패인지 구분 불가 (가치 2 / 위험 2 / S) · `listUsers`의 `q` 파라미터에만 길이 상한이 없어 매우 긴 검색어가 세 컬럼 ILIKE 스캔으로 들어감 (가치 2 / 위험 1 / S)

## 2026-09-06
- 선택: 대시보드가 조회 실패를 "승인 대기 0건"·"공급자 없음"으로 감추던 문제 수정 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `dashboard` 핸들러가 `_ = s.db.Pool.QueryRow(...).Scan(&pending)`으로 승인 대기 건수 조회 오류를 버리고 0을 내려보냈고, `dashboardProviders`도 조회 실패를 빈 목록으로, scan 실패 행을 건너뛴 짧은 목록으로 반환해 화면의 "등록된 AI 공급자가 없습니다"와 구별되지 않았다. 두 값은 레거시 스키마가 아니라 `ai_admin` 자기 테이블에서 오므로 degradation으로 볼 이유가 없어, 조회가 실패하면 500 `dashboard_unavailable` 오류 봉투를 반환하고(프런트엔드는 이미 `ErrorState`+재시도를 표시) 공급자 목록은 v1.2.9에서 도입한 공통 `collectRows` 경로를 쓰도록 했다. 레거시 스키마 기반 지표는 `metrics[].available` degradation 계약을 그대로 두었다. 공급자 상태 매핑·scan 실패 시 부분 목록 없음·연결 불가 pool을 향한 대시보드 요청의 500 응답을 테스트로 검증했고, 수정 전 코드에서는 같은 요청이 200과 `"pendingApprovals":0`, `"providerHealth":[]`를 반환하는 것을 실제로 확인했다. Docker로 PostgreSQL 16을 띄워 `TEST_POSTGRES_DSN`을 설정한 뒤 `go test -race -count=1 ./...`(통합 테스트 포함)을 통과시켰고 `gofmt -l`·`go vet`·`go build ./...`·`scripts/verify-version.sh`·`npm ci && npm test`(62개)·`npm run build`도 모두 통과했다. 저장소 관례에 따라 VERSION을 1.2.12로 올리고(1.2.10·1.2.11은 아직 병합되지 않은 형제 branch가 사용) CHANGELOG·README·docs·web 버전 메타데이터를 맞췄으며 `docs/api.md`에 대시보드 실패 계약을 명시했다(직전 릴리즈 커밋들과 동일하게 `internal/ui/dist`는 재빌드하지 않음).
- 보류 아이디어: CI에 정적 분석 단계(`gofmt -l`, `go vet`) 추가 — eslint는 설정 자체가 없어 축소 범위 권장 (가치 3 / 위험 1 / M) · `loadGrants`가 map 순회로 roles·permissions를 만들어 `/api/v1/auth/me` 응답 순서가 요청마다 뒤바뀌고 프로필 화면의 태그가 섞임 (가치 2 / 위험 1 / S) · `safeCSVCell`이 OWASP가 함께 권고하는 tab(0x09)·CR(0x0D) 선행 문자를 중화하지 않음 (가치 2 / 위험 1 / S) · `listUsers`의 `q` 파라미터에만 길이 상한이 없어 매우 긴 검색어가 세 컬럼 ILIKE 스캔으로 들어감 (가치 2 / 위험 1 / S) · `decideApproval`이 `approval_action.comment`에는 trim한 값을, `approval_request.decision_comment`에는 원문을 저장해 같은 결정의 두 기록이 달라짐 (가치 2 / 위험 1 / S)

## 2026-09-07
- 선택: 중간에 끊긴 AI 채팅 응답을 완결된 답변처럼 끝내던 문제 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `chatCompletions`는 공급자 응답의 200과 header를 먼저 내보낸 뒤 본문을 중계하는데, 연결이 답변 도중 끊기거나(`upstream_read_failed`) 공급자 제한 시간을 넘겨도(`upstream_timeout`) 중계 loop만 빠져나와 응답을 정상적으로 끝냈다. 스트리밍에서는 잘린 답변이 모델이 스스로 멈춘 완전한 답변처럼 보이고, 비스트리밍에서는 잘린 JSON이 200과 함께 전달되어 감사 이벤트에만 실패가 남았다. 중계 loop를 `relayChatBody`로 분리해 전달 완료 여부와 중단 사유를 반환하게 하고, 끝까지 전달하지 못하면 감사에 `failure`+`reason`을 남긴 뒤 `panic(http.ErrAbortHandler)`로 응답을 중단하도록 했다(v1.2.5 감사 CSV 내보내기와 같은 방식이며 `recoverer`·`accessLog`가 이미 이 신호를 처리한다). 프런트엔드도 함께 고쳐 `streamJson`이 스트림 read 실패를 조용히 종료하지 않고 "AI 응답이 완료되기 전에 연결이 끊어졌습니다."로 실패시키며, 플레이그라운드는 이미 받은 부분 답변을 지우지 않고 오류와 함께 남긴다. 다섯 가지 중계 결과(정상·upstream 읽기 실패·제한 시간 초과·요청 취소·클라이언트 쓰기 실패)를 단위 테스트로, 답변 도중 끊는 합성 공급자를 향한 요청이 완전한 응답으로 전달되지 않고 감사에 `upstream_read_failed`로 남는지를 PostgreSQL 통합 테스트로 검증했다(수정 전 코드에서는 잘린 본문이 오류 없이 그대로 수신되어 실패). Docker로 PostgreSQL 16을 띄워 `TEST_POSTGRES_DSN`을 설정한 뒤 `go test -race -count=1 ./...`(통합 테스트 포함)을 통과시켰고 `gofmt -l`·`go vet ./...`·`go build ./...`·`scripts/verify-version.sh`·`npm ci && npm test`(63개)·`npm run build`도 모두 통과했다. 저장소 관례에 따라 VERSION을 1.2.13으로 올리고(1.2.10~1.2.12는 아직 병합되지 않은 형제 branch가 사용) CHANGELOG·README·docs·web 버전 메타데이터를 맞췄으며 `docs/api.md`에 중단 계약을 명시했다(직전 릴리즈 커밋들과 동일하게 `internal/ui/dist`는 재빌드하지 않음).
- 보류 아이디어: DB 오류를 업무 규칙 충돌로 잘못 보고하는 handler들(`rotateKey`의 409 `key_not_active`, `decideApproval`의 409 `approval_not_pending`, `updateUser`의 404 `user_not_found`·400 `roles_invalid`)을 로그인 분류(v1.2.10)와 같은 방식으로 구분 (가치 3 / 위험 2 / M) · CI에 정적 분석 단계(`gofmt -l`, `go vet`) 추가 — eslint는 설정 자체가 없어 축소 범위 권장 (가치 3 / 위험 1 / M) · `loadGrants`가 map 순회로 roles·permissions를 만들어 `/api/v1/auth/me` 응답 순서가 요청마다 뒤바뀜 (가치 2 / 위험 1 / S) · `safeCSVCell`이 OWASP가 함께 권고하는 tab(0x09)·CR(0x0D) 선행 문자를 중화하지 않음 (가치 2 / 위험 1 / S) · `decideApproval`이 `approval_action.comment`에는 trim한 값을, `approval_request.decision_comment`에는 원문을 저장해 같은 결정의 두 기록이 달라짐 (가치 2 / 위험 1 / S)

