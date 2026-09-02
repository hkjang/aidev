
## 2026-09-02
- 선택: 모델 가격 prefix 매칭 비결정성 수정 (가치 5 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `audit.lookupPrice`가 가격표 map을 range하며 첫 prefix 일치를 반환해, 내장 카탈로그에 서로의 prefix인 항목(gpt-4o vs gpt-4o-mini, claude-sonnet-4 vs claude-sonnet-4-5)이 많은 탓에 버전 접미사가 붙은 실제 모델 ID(`gpt-4o-mini-2026-06-01`)가 호출마다 최대 16.7배 다른 단가로 계산되던 버그를 고쳤다. 가장 긴(구체적인) prefix를 채택하도록 바꾸고, 같은 규칙을 손으로 복제해 동일 버그를 갖고 있던 `admin_explain.lookupModelPrice`는 새로 export한 `audit.LookupPrice`에 위임시켰다. 회귀 테스트를 추가해 옛 구현에서 실패(16250 vs 975)함을 확인했고, gofmt·go vet·go build·`go test ./...`·`go test -race ./internal/audit ./internal/proxy`·`cmd/api-surface-audit` 모두 통과했다.
- 보류 아이디어: redact.go IPv4 규칙의 "사설망 제외" 주석과 실제 동작(전부 마스킹) 불일치 정리 / internal/config 패키지 테스트 부재 보강 / `EstimateTokens`의 `[]rune(text)` 전체 복사를 `utf8.RuneCountInString`으로 교체 / 가격표 키 정규화(소문자·trim)를 적재 시점에 일원화해 조회마다 재정규화 제거 / prefix 매칭에 경계 검사 추가로 `gpt-4`가 `gpt-45`에 잘못 매칭되는 것 방지

## 2026-09-03
- 선택: `PROXY_API_KEYS` 필드 트림 및 빈 시크릿 검증 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `config.parseProxyKeys`가 CSV 항목 전체만 TrimSpace하고 `:` 분리 후 개별 필드는 트림하지 않아, `PROXY_API_KEYS="dev: dev-proxy-key: alice"`처럼 자연스럽게 띄어쓴 설정이 `" dev-proxy-key"`의 해시로 저장되던 버그를 고쳤다. 인증 경로의 `bearerToken`은 토큰을 트림해서 넘기므로 이 키는 어떤 요청으로도 절대 매칭되지 않는 무성 인증 실패였다. 아울러 시크릿이 빈 항목(`dev:`, `dev::alice:team`)이 `sha256("")`을 active 키 해시로 등록해 게이트웨이를 키 필수 모드로 뒤집던 것도 건너뛰도록 했다. 테스트가 전무하던 `internal/config`에 첫 테스트 파일을 추가해 문서화된 `name:key:owner:team` 형식과 두 회귀를 덮었고, 수정 전 코드에서 실패함을 확인했다. gofmt·go vet·go build·`go test ./...`·`go test -race ./internal/config` 모두 통과.
- 보류 아이디어: redact.go IPv4 규칙의 "사설망 제외" 주석과 실제 동작(전부 마스킹) 불일치 정리 / `EstimateTokens`의 `[]rune(text)` 전체 복사를 `utf8.RuneCountInString`으로 교체 / 가격표 키 정규화(소문자·trim)를 적재 시점에 일원화해 조회마다 재정규화 제거 / `databaseConfig`의 DSN 우선순위·`durationEnv`/`floatMapEnv` 등 나머지 config 헬퍼 테스트 보강 / `parseProxyKeys`에서 중복 키가 같은 ID로 서로를 덮어쓰는 문제 경고 처리
- 릴리즈: v0.82.0 (2026-09-03)
