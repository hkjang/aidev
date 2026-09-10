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

- 릴리즈: v1.2.10 (2026-09-07, run 2026-09-07-102543-ai-admin-approve)
- 릴리즈: v1.2.11 (2026-09-08, run 2026-09-08-150055-ai-admin-improve)
## 2026-09-08
- 선택: 확인하지 못한 자격 증명을 세션 만료로 보고하던 인증 미들웨어 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `auth.Authenticate`가 session·API key 조회 실패를 자격 증명 오류와 같은 값으로 반환하고 미들웨어가 모두 401 `unauthenticated`로 답해, DB 장애가 나면 아직 유효한 session이 만료된 것으로 보여 모든 운영자가 로그인 화면으로 밀려나고 API key client는 키가 폐기된 것으로 읽었다(프런트엔드는 이미 401이 아닌 인증 오류에 "서비스에 연결할 수 없습니다"+재시도를 띄우도록 되어 있었지만 서버가 그 상태를 만들지 않았다). v1.2.10 로그인 분류·990c7fb의 DB 오류 분류와 같은 원칙으로 `ErrUnauthenticated` sentinel을 도입해 "확인해서 거부한" 경우만 401로, 조회 실패는 `Retry-After`와 함께 503 `auth_unavailable`로 답하고 MCP의 `WWW-Authenticate` challenge도 실제 거부에만 붙게 했다. 연결 불가 pool을 향한 요청의 401/503 분기와, 정상 DB에서 폐기 session·비활성 계정·없는 key는 401이고 session·role table이 안 읽힐 때만 503인지를 통합 테스트로 검증했다(수정 전 코드에서는 모두 401로 실패). Docker PostgreSQL 16에 `TEST_POSTGRES_DSN`을 걸어 `go test -race -count=1 ./...`를 통과시켰고 `gofmt -l`·`go vet ./...`·`go build ./...`·`scripts/verify-version.sh`·`npm ci && npm test`(62개)·`npm run build`도 통과했다. 이번 세션 규칙대로 버전·CHANGELOG는 건드리지 않았고 `docs/api.md`에만 401/503 계약을 명시했다.
- 보류 아이디어: CI에 정적 분석 단계(`gofmt -l`, `go vet`) 추가 — eslint는 설정 자체가 없어 축소 범위 권장 (가치 3 / 위험 1 / M) · `loadGrants`가 map 순회로 roles·permissions 순서를 무작위화해 `/api/v1/auth/me` 응답 순서가 요청마다 뒤바뀜 (가치 2 / 위험 1 / S) · `safeCSVCell`이 OWASP가 함께 권고하는 tab(0x09)·CR(0x0D) 선행 문자를 중화하지 않음 (가치 2 / 위험 1 / S) · `listUsers`의 `q`에만 길이 상한이 없어 매우 긴 검색어가 세 컬럼 ILIKE 스캔으로 들어감 (가치 2 / 위험 1 / S) · `decideApproval`이 `approval_action.comment`에는 trim한 값을, `approval_request.decision_comment`에는 원문을 저장해 같은 결정의 두 기록이 달라짐 (가치 2 / 위험 1 / S)

## 2026-09-08
- 선택: 프로필·환경 설정 저장이 DB 장애를 이메일 중복으로 보고하고 저장하지 못한 설정을 성공으로 답하던 문제 수정 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `updateProfile`은 모든 DB 오류를 409 "이메일 중복 여부를 확인해 주세요."로 답해 연결 끊김·제약 위반 같은 백엔드 장애가 이메일 중복으로 보였고, `updatePreferences`는 `user_preference` 행이 있다고 가정하고 읽은 뒤 UPDATE해서 app_user에 직접 추가된 계정(행 없음)은 DB가 정상인데도 500이 되고 저장하려던 설정이 사라졌다. 990c7fb의 원칙대로 unique 위반(23505)만 409 `email_conflict`로 구분하고 나머지는 500 `profile_update_failed`, 대상 계정이 없으면 404 `user_not_found`를 반환하게 했으며, 없는 설정 행은 실패가 아니라 column 기본값으로 읽고(`loadPreferences`) 저장은 `ON CONFLICT` upsert로 첫 저장이 행을 만들도록 했다(GET이 행을 몰래 INSERT하던 lazy 생성은 제거). Docker로 PostgreSQL 16을 띄워 `TEST_POSTGRES_DSN`을 걸고 새 통합 테스트가 수정 전 코드에서 409·409·500으로 실패하는 것을 확인한 뒤 `go test -race -count=1 ./...`·`gofmt -l`·`go vet ./...`·`go build ./...`·`scripts/verify-version.sh`·`npm ci && npm test`(62개)·`npm run build`를 모두 통과시켰다. 이번 세션 규칙대로 버전·CHANGELOG는 건드리지 않고 `docs/api.md`에만 계약을 명시했다.
- 보류 아이디어: CI에 정적 분석 단계(`gofmt -l`, `go vet`) 추가 — eslint는 설정 자체가 없어 축소 범위 권장 (가치 3 / 위험 1 / M) · `loadGrants`가 map 순회로 roles·permissions 순서를 무작위화해 `/api/v1/auth/me` 응답 순서가 요청마다 뒤바뀜 (가치 2 / 위험 1 / S) · `updatePreferences`가 `locale`(varchar 20)·`timezone`(varchar 80)에 길이·형식 검증 없이 저장해 긴 값이 400 대신 500이 됨 (가치 2 / 위험 1 / S) · `listUsers`의 `q`에만 길이 상한이 없어 매우 긴 검색어가 세 컬럼 ILIKE 스캔으로 들어감 (가치 2 / 위험 1 / S) · `safeCSVCell`이 OWASP가 함께 권고하는 tab(0x09)·CR(0x0D) 선행 문자를 중화하지 않음 (가치 2 / 위험 1 / S)

- 릴리즈: v1.2.12 (2026-09-08, run 2026-09-08-183103-ai-admin-improve)
- 릴리즈: v1.2.13 (2026-09-08, run 2026-09-08-205400-ai-admin-approve)
## 2026-09-08
- 선택: 중간에 끊긴 AI 채팅 답변을 완결된 답변처럼 끝내던 문제 수정 (PR #11 재시도) (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `chatCompletions`는 공급자 응답의 200과 header를 먼저 내보낸 뒤 본문을 중계하는데, 연결이 답변 도중 끊기거나(`upstream_read_failed`) 공급자 제한 시간을 넘겨도(`upstream_timeout`) 중계 loop만 빠져나와 응답을 정상적으로 끝냈다. 스트리밍에서는 잘린 답변이 모델이 스스로 멈춘 완전한 답변처럼 보이고, 비스트리밍에서는 잘린 JSON이 200과 함께 전달되어 실패가 감사 이벤트에만 남았다. 중계 loop를 `relayChatBody`로 분리해 전달 완료 여부와 중단 사유(`upstream_read_failed`·`upstream_timeout`·`request_cancelled`·`client_write_failed`·`client_flush_failed`)를 반환하게 하고, 끝까지 전달하지 못하면 감사에 `failure`+`reason`을 남긴 뒤 `panic(http.ErrAbortHandler)`로 응답을 중단하도록 했다(v1.2.5 감사 CSV 내보내기와 같은 방식이며 `recoverer`·`accessLog`가 이미 이 신호를 처리한다). 프런트엔드도 함께 고쳐 `streamJson`이 스트림 read 실패를 조용히 종료하지 않고 "AI 응답이 완료되기 전에 연결이 끊어졌습니다."로 실패시키며, 플레이그라운드는 이미 받은 부분 답변을 지우지 않고 오류와 함께 남긴다. 일곱 가지 중계 결과를 Go 단위 테스트로, 답변 도중 연결을 끊는 합성 공급자를 향한 요청이 완전한 응답으로 전달되지 않고 감사에 `upstream_read_failed`로 남는지를 PostgreSQL 통합 테스트로, 끊긴 스트림이 오류가 되는지를 web 테스트로 검증했다(수정 전 코드로 되돌려 Go 통합 테스트와 web 테스트가 실제로 실패하는 것을 확인했다). Docker로 PostgreSQL 16을 띄워 `TEST_POSTGRES_DSN`을 설정한 뒤 `go test -race -count=1 ./...`(통합 테스트 포함)을 통과시켰고 `gofmt -l`·`go vet ./...`·`go build ./...`·`scripts/verify-version.sh`·`npm ci && npm test`(63개)·`npm run build`도 모두 통과했다. 이번 세션 규칙대로 버전·CHANGELOG는 건드리지 않았고 `docs/api.md`에만 중단 계약을 명시했다.
- 보류 아이디어: CI에 정적 분석 단계(`gofmt -l`, `go vet`) 추가 — eslint는 설정 자체가 없어 축소 범위 권장 (가치 3 / 위험 1 / M) · `auth.User`가 app_user 조회 실패를 '없는 사용자'·'비활성 계정'과 같은 오류로 돌려줌 — `ErrUnauthenticated` sentinel이 71122c3으로 병합되어 이제 착수 가능 (가치 3 / 위험 2 / S) · `loadGrants`가 map 순회로 roles·permissions 순서를 무작위화해 `/api/v1/auth/me` 응답 순서가 요청마다 뒤바뀜 (가치 2 / 위험 1 / S) · `updatePreferences`가 `locale`(varchar 20)·`timezone`(varchar 80)에 길이·형식 검증 없이 저장해 긴 값이 400 대신 500이 됨 (가치 2 / 위험 1 / S) · `relayChatBody`가 `client_write_failed`(사용자 이탈)도 공급자 장애와 같은 `result=failure`로 감사해 공급자 실패율 집계가 부풀려짐 (가치 2 / 위험 1 / S)

- 릴리즈: v1.2.14 (2026-09-08, run 2026-09-08-233109-ai-admin-improve)
## 2026-09-09
- 선택: SSO 로그인 마지막 두 단계가 백엔드 장애를 계정 문제로 보고하던 문제 수정 (가치 3 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: OIDC callback은 계정 준비(`provisionOIDCUser`)와 세션 생성(`auth.CreateSession`)의 모든 실패를 403 `oidc_user_unavailable`("관리자에게 문의해 주세요")·403 `oidc_session_failed`("계정 상태를 확인해 주세요")로 답했다. 두 단계 모두 `ai_admin` 자기 table에 읽고 쓰므로 DB 장애가 나면 멀쩡한 계정을 고치라고 관리자에게 보내고, 다시 시도하면 되는 SSO 로그인이 영구 거부처럼 보였다. 71122c3이 `Authenticate`에 도입한 구분을 이어받아 `auth.User`는 실제로 읽어서 거부한 경우(없는 사용자·비활성 계정)에만 `ErrUnauthenticated`를 반환하고 조회 실패는 감싸 올리며, `CreateSession`은 session INSERT 실패를 `Login`과 같은 방식으로 감싼다. `provisionOIDCUser`는 확인된 거부(비활성 계정·매핑 역할 없음·고유 username 확보 실패)를 `errOIDCUserUnavailable`로 표시하고, 역할 배정 확인 조회 실패를 더는 "역할 없음"으로 보고하지 않는다. callback은 이 오류로 분기해 확인된 거부는 403을 유지하고 나머지는 `Retry-After`와 함께 503 `oidc_provisioning_unavailable`·`oidc_session_unavailable`(기존 provisioning·session 단계 공유)을 반환하며, 로그인 화면에 두 코드의 재시도 문구를 넣었다. 분류 함수 표 테스트와, 비활성 계정은 403·`app_user`/`session` table을 숨기면 503이 되는 PostgreSQL 통합 테스트를 추가해 수정 전 오류 값으로 되돌리면 실제로 실패하는 것을 확인했다. Docker로 PostgreSQL 16을 띄워 `TEST_POSTGRES_DSN`을 걸고 `go test -race -count=1 ./...`(통합 테스트 포함)·`gofmt -l`·`go vet ./...`·`go build ./...`·`scripts/verify-version.sh`·`npm ci && npm test`(64개)·`npm run build`를 모두 통과시켰다. 이번 세션 규칙대로 버전·CHANGELOG는 건드리지 않고 `docs/api.md`에만 계약을 명시했다.
- 보류 아이디어: CI에 정적 분석 단계(`gofmt -l`, `go vet`) 추가 — eslint는 설정 자체가 없어 축소 범위 권장 (가치 3 / 위험 1 / M) · `loadGrants`가 map 순회로 roles·permissions 순서를 무작위화해 `/api/v1/auth/me` 응답 순서가 요청마다 뒤바뀜 (가치 2 / 위험 1 / S) · `updatePreferences`가 `locale`(varchar 20)·`timezone`(varchar 80)에 길이·형식 검증 없이 저장해 긴 값이 400 대신 500이 됨 (가치 2 / 위험 1 / S) · `listUsers`의 `q`에만 길이 상한이 없어 매우 긴 검색어가 세 컬럼 ILIKE 스캔으로 들어감 (가치 2 / 위험 1 / S) · `safeCSVCell`이 OWASP가 함께 권고하는 tab(0x09)·CR(0x0D) 선행 문자를 중화하지 않음 (가치 2 / 위험 1 / S)

- 릴리즈: v1.2.15 (2026-09-09, run 2026-09-09-064014-ai-admin-approve)
## 2026-09-09
- 선택: CI에 정적 분석 lint 단계(gofmt -l, go vet, verify-version.sh) 추가 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: CI가 `go test` / `npm test` / `npm run build` / `docker build`만 실행해, 테스트는 통과하지만 잘못된 상태 세 가지를 아무도 막지 않았다 — gofmt로 정리되지 않은 서식, `go vet`이 잡는 진단, 그리고 릴리즈 메타데이터 불일치다. 특히 `scripts/verify-version.sh`는 VERSION·CHANGELOG·README·docs·compose.offline.yml·web 버전 표기가 서로 맞는지 검사하는 이 저장소의 릴리즈 안전장치인데도 CI가 전혀 실행하지 않아, 어긋난 버전 표기는 사람이 손으로 스크립트를 돌릴 때만 드러났다(직전 6회차가 매번 손으로 실행해 온 검사다). 데이터베이스도 컨테이너도 필요 없는 `lint` job을 추가해 기존 `test` job과 나란히 돌리고, 같은 세 명령을 `make lint`(= `make all`의 첫 단계)로 묶어 로컬에서 CI와 동일하게 확인할 수 있게 한 뒤 README 개발 절에 적었다. 보류 목록의 권고대로 web eslint는 범위에서 제외했다(설정 자체가 없고 오프라인 빌드 정책상 의존성 추가를 따로 검토해야 함). 검증은 세 검사가 지금 코드에서 통과하는지 확인하는 데 그치지 않고, 일부러 서식이 어긋난 `.go` 파일을 넣어 `make lint`와 CI job의 셸 조각이 실제로 파일명을 출력하며 실패(exit 1)하는지 확인한 뒤 되돌렸고, `python3 -c "yaml.safe_load(...)"`로 워크플로 YAML이 파싱되어 `lint`·`test` 두 job이 나오는지 확인했다. Docker로 PostgreSQL 16을 띄워 `TEST_POSTGRES_DSN`을 걸고 `go test -race -count=1 ./...`(통합 테스트 포함, `internal/server` 46s)·`go build ./...`·`gofmt -l`·`go vet ./...`·`scripts/verify-version.sh`·`npm ci && npm test`(64개)·`npm run build`를 모두 통과시켰다. 이번 세션 규칙대로 버전·CHANGELOG는 건드리지 않았고, 커밋 전 `git status`로 빌드 산출물이 섞이지 않았음을 확인했다(변경 파일 3개).
- 보류 아이디어: 대시보드의 승인 대기 건수(`legacy.go:51`)·공급자 목록(`dashboardProviders`)이 여전히 조회 실패를 0건·빈 목록으로 감춤 — 2026-09-06 회차의 수정 `df278de`는 branch `auto/2026-09-06-2320`에 남아 main에 병합되지 않았고 v1.2.12에는 다른 변경이 들어가, 버그는 main에 그대로 있다(재착수 전 미병합 사유 확인 필요) (가치 3 / 위험 1 / S) · `loadGrants`가 map 순회로 roles·permissions 순서를 무작위화해 `/api/v1/auth/me` 응답 순서가 요청마다 뒤바뀜 (가치 2 / 위험 1 / S) · `updatePreferences`가 `locale`(varchar 20)·`timezone`(varchar 80)에 길이 검증 없이 upsert해 긴 값이 400 대신 500이 되고 같은 요청의 다른 설정 변경도 함께 사라짐 (가치 2 / 위험 1 / S) · `listUsers`의 `q`에만 200자 상한이 없어(감사·레거시·MCP는 모두 있음) 매우 긴 검색어가 세 컬럼 ILIKE 스캔으로 들어감 (가치 2 / 위험 1 / S) · `safeCSVCell`이 `value[0]` 한 byte에서 `"=+-@"`만 검사해 OWASP가 함께 권고하는 tab(0x09)·CR(0x0D) 선행 문자를 중화하지 않음 (가치 2 / 위험 1 / S)

- 릴리즈: v1.2.16 (2026-09-09, run 2026-09-09-145444-ai-admin-approve)
## 2026-09-10
- 선택: 대시보드가 조회 실패를 "승인 대기 0건"·"공급자 없음"으로 감추던 문제 수정 (main 재착수) (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 보류 목록 1번의 재착수 전제였던 "df278de가 왜 병합되지 않았는지"를 먼저 확인했다 — 그 commit이 담긴 `auto/2026-09-06-2320`은 origin에 아예 push되지 않았고(원격 branch 목록에 없음) `git merge-base --is-ancestor df278de main`도 NO다. 즉 반려가 아니라 단순 미push였고, 버그는 main에 그대로 남아 있었다: `dashboard`는 `_ = s.db.Pool.QueryRow(...).Scan(&pending)`로 승인 대기 건수 조회 오류를 버리고 0을 내려보냈고, `dashboardProviders`는 조회 실패를 빈 목록으로, scan 실패 행을 건너뛴 짧은 목록으로 반환해 화면의 "등록된 AI 공급자가 없습니다"와 구별되지 않았다. 두 값은 레거시 스키마가 아니라 `ai_admin` 자기 테이블에서 오므로 degradation으로 볼 이유가 없어, 조회가 실패하면 500 `dashboard_unavailable` 오류 봉투를 반환하고(`DashboardPage.tsx:60`이 이미 `ErrorState`+재시도를 표시한다) 공급자 목록은 v1.2.9의 공통 `collectRows` 경로를 쓰도록 했다. 레거시 스키마 기반 지표(`metrics[].available`)의 degradation 계약은 그대로 두었다. 검증은 통과 확인에 그치지 않고, 공급자 상태 매핑·scan 실패 시 부분 목록 없음·연결 불가 pool을 향한 500 응답을 테스트로 덮은 뒤 `dashboard` 핸들러만 수정 전 형태로 되돌려 `TestDashboardReportsApplicationQueryFailure`가 실제로 200과 `"pendingApprovals":0`, `"providerHealth":null`로 실패하는 것을 확인하고 되돌렸다. Docker로 PostgreSQL 16을 띄워 `TEST_POSTGRES_DSN`을 걸고 `go test -race -count=1 ./...`(통합 테스트 포함, `internal/server` 47s)·`gofmt -l`·`go vet ./...`·`go build ./...`·`scripts/verify-version.sh`·`npm ci && npm test`(64개)·`npm run build`를 모두 통과시켰다. 이번 세션 규칙대로 VERSION·CHANGELOG·릴리즈 메타데이터는 건드리지 않았고(df278de에 있던 버전 bump는 의도적으로 제외했다) `docs/api.md`에만 대시보드 실패 계약을 명시했으며, 커밋 전 `git status`로 빌드 산출물이 섞이지 않았음을 확인했다(변경 4개).
- 보류 아이디어: `loadGrants`가 map 순회로 roles·permissions 순서를 무작위화해 `/api/v1/auth/me` 응답 순서가 요청마다 뒤바뀜 — 지금 코드(auth.go:250-256)에서 재확인 (가치 2 / 위험 1 / S) · `updatePreferences`가 `locale`(varchar 20)·`timezone`(varchar 80)에 길이 검증 없이 upsert해 긴 값이 400 대신 500이 되고 같은 요청의 다른 설정 변경도 함께 사라짐 (가치 2 / 위험 1 / S) · `listUsers`의 `q`에만 200자 상한이 없어(감사·레거시·MCP는 모두 있음) 매우 긴 검색어가 세 컬럼 ILIKE 스캔으로 들어감 (가치 2 / 위험 1 / S) · `safeCSVCell`이 `value[0]` 한 byte에서 `"=+-@"`만 검사해 OWASP가 함께 권고하는 tab(0x09)·CR(0x0D) 선행 문자를 중화하지 않음 (가치 2 / 위험 1 / S) · 새 아이디어: `aiModels`·`aiCatalog`가 `available_models`의 JSON 파싱 실패를 `_ = json.Unmarshal`로 무시해 모델 목록이 기본 모델 하나로 조용히 줄어듦 (가치 2 / 위험 1 / S) · 새 아이디어: `dashboardProviders`의 `LIMIT 10`이 목록이 잘렸음을 알리지 않아 11개 이상 등록 시 일부 공급자가 미등록처럼 보임 (가치 2 / 위험 1 / S)

- 릴리즈: v1.2.17 (2026-09-10, run 2026-09-10-111300-ai-admin-improve)
## 2026-09-10
- 선택: 저장할 수 없는 locale·timezone이 개인 설정 저장 전체를 날리던 문제 수정 (가치 2 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `updatePreferences`는 `theme`(CHECK 제약)과 `fontScale`은 검증하면서 `locale`(varchar 20)·`timezone`(varchar 80)은 `TrimSpace`만 하고 그대로 upsert했다. column이 담을 수 없는 값이 오면 PostgreSQL이 INSERT를 거부해 호출자 실수가 400이 아니라 500 `preferences_update_failed`가 되고, 이 handler는 모든 설정을 한 statement로 쓰므로 같은 요청에 담긴 `theme`·알림 설정 변경까지 함께 사라졌다. 저장소가 이미 쓰는 `utf8.RuneCountInString` 상한 방식으로 언어 태그(20자 이하, `ko`/`ko-KR`/`zh-Hans-CN` 형식)와 IANA 시간대 이름(80자 이하, `UTC`/`Asia/Seoul` 형식)만 받도록 하고, 아무것도 쓰기 전에 검사해 거부된 요청이 다른 설정을 건드리지 않게 했다(중복돼 있던 `TrimSpace` 분기도 `optionalTrimmed`로 통일). 검증은 통과 확인에 그치지 않고, 형식·길이 경계 22가지를 덮는 표 테스트와 "거부된 요청 뒤에도 저장된 theme이 그대로인지" 확인하는 PostgreSQL 통합 테스트를 추가한 뒤 검사 블록만 제거해 되돌려 실제로 500 `preferences_update_failed`로 실패하는 것을 확인하고 복구했다. Docker로 PostgreSQL 16을 띄워 `TEST_POSTGRES_DSN`을 걸고 `go test -race -count=1 ./...`(통합 테스트 포함, `internal/server` 48s)·`make lint`(`gofmt -l`·`go vet ./...`·`scripts/verify-version.sh`)·`go build ./...`·`npm ci && npm test`(64개)·`npm run build`를 모두 통과시켰다. 이번 세션 규칙대로 VERSION·CHANGELOG·릴리즈 메타데이터는 건드리지 않고 `docs/api.md`에만 계약을 명시했으며, 커밋 전 `git status`로 빌드 산출물이 섞이지 않았음을 확인했다(변경 4개).
- 보류 아이디어: `loadGrants`가 map 순회로 roles·permissions 순서를 무작위화해 `/api/v1/auth/me` 응답 순서가 요청마다 뒤바뀜 — SQL에 `ORDER BY r.code,p.code`가 있는데 map이 그 순서를 버린다(auth.go:250-256에서 재확인) (가치 2 / 위험 1 / S) · `listUsers`의 `q`에만 200자 상한이 없어(감사·레거시·MCP는 모두 있음) 매우 긴 검색어가 세 컬럼 ILIKE 스캔으로 들어감 (가치 2 / 위험 1 / S) · `safeCSVCell`이 `value[0]` 한 byte에서 `"=+-@"`만 검사해 OWASP가 함께 권고하는 tab(0x09)·CR(0x0D) 선행 문자를 중화하지 않음 (가치 2 / 위험 1 / S) · `decideApproval`이 `approval_action.comment`에는 trim한 값을, `approval_request.decision_comment`에는 원문을 길이 제한 없이 저장해 같은 결정의 두 기록이 달라짐 (가치 2 / 위험 1 / S) · 새 아이디어: `loadProvider`가 `available_models` JSON 파싱 실패를 `_ = json.Unmarshal`로 무시해 `chatCompletions`의 모델 허용 목록이 기본 모델 하나로 조용히 줄고 정상 모델 요청이 400 `model_not_allowed`로 거부됨 — `aiModels`·`aiCatalog`의 같은 결함과 한 번에 고칠 수 있다 (가치 2 / 위험 1 / S)

- 릴리즈: v1.2.18 (2026-09-10, run 2026-09-10-182121-ai-admin-improve)
