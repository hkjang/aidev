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
