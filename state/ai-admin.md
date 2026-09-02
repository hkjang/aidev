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
