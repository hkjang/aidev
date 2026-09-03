
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

## 2026-09-03
- 선택: 응답 분석기의 구조화 content 파트 누락 및 스트리밍 tool_call 순서 비결정성 수정 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `proxy.contentString`이 문자열 content만 평탄화해, content 파트 배열(`[{"type":"text","text":"..."}]`)로 답하는 업스트림의 `CompletionText`가 빈 문자열이 되던 버그를 고쳤다. `pipeline.go`는 응답에 usage 블록이 없을 때 이 텍스트 기반 추정치로 폴백하므로, 해당 요청들은 completion 토큰 0 · 비용 0으로 기록되고 있었다(요청 측 `flattenContent`는 같은 형태를 이미 평탄화하고 있어 명백한 누락). 텍스트가 없는 파트(이미지 등)는 추정치를 부풀리지 않도록 raw JSON 대신 계속 빈 문자열로 둔다. 함께, 스트리밍 tool_call 이름을 index 키 map을 그대로 range하며 내보내 한 응답의 `tool_invocations` 행 순서가 실행마다 달라지던 것(조회 쿼리는 `ORDER BY created_at, source`뿐이라 표시 순서가 그대로 흔들림)을 index 정렬 방출로 바꿨다. 회귀 테스트 3개를 추가해 수정 전 코드에서 모두 실패함을 확인했고(순서 테스트는 20회 반복), gofmt·go vet·go build·`go test ./...`·`go test -race ./internal/proxy` 모두 통과.
- 보류 아이디어: `POST /admin/pricing`에 음수 단가 검증 부재(음수 비용이 그대로 저장됨) / `audit.InferLanguages`가 동점 신뢰도를 알파벳순으로만 깨서 대표 언어가 자의적으로 결정되는 문제를 근거 개수 기준으로 개선 / redact.go IPv4 규칙의 "사설망 제외" 주석과 실제 동작(전부 마스킹) 불일치 정리 / `EstimateTokens`의 `[]rune(text)` 전체 복사를 `utf8.RuneCountInString`으로 교체 / `databaseConfig`의 DSN 우선순위·`durationEnv`/`floatMapEnv` 등 나머지 config 헬퍼 테스트 보강
- 참고: 2026-09-02 세션의 가격 prefix 수정(`auto/2026-09-02-2250` 브랜치)은 아직 master에 병합되지 않은 상태

## 2026-09-04
- 선택: 모델 단가 음수 검증 부재 수정 (`POST /admin/pricing` + `MODEL_PRICING_KRW_PER_1M`) (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 운영자가 단가를 넣는 두 경로 모두 음수 KRW를 그대로 받아들이고 있었다 — `handlePricing`의 POST는 필드 존재 여부만 검사했고, `config.Load`는 `MODEL_PRICING_KRW_PER_1M`을 범위 검사 없이 unmarshal했다. `audit.EstimateCostKRW`는 단가를 그대로 곱하므로 음수 단가는 음수 비용이 되고, 강제 지점이 전부 `cost > limit` 비교라서(키별 예산 `pipeline.go:417`, 비용 가드 `pipeline.go:427`) 해당 모델의 예산·비용 가드가 무조건 통과하고 쿼터 합계는 요청을 더하는 대신 빼게 된다. `-1`을 "미설정" 뜻으로 넣는 흔한 오타 하나로 강제가 무력화되는 구조라 두 경계에서 각각 400 응답·부팅 거부로 막았다. 0은 "과금하지 않는 모델" 표기 수단이라 계속 허용한다. 나머지 `InsertPricingVersion` 호출부 2곳은 정적 내장 카탈로그 시드라 대상 아님. 회귀 테스트 5개를 추가해 수정 전 코드에서 모두 실패함을 확인했고(잘못된 버전이 DB에 기록되지 않는 것까지 검증), gofmt·go vet·go build·`go test ./...`·`go test -race ./internal/config ./internal/proxy -timeout=30m`(CI와 동일, data race 0)·`cmd/api-surface-audit` 모두 통과.
- 보류 아이디어: OpenAPI `PricingWriteRequest` 스키마에 `minimum: 0` 반영(이번엔 생략 — 재생성에 Node 24·pnpm 11이 필요한데 환경엔 Node 22·pnpm 없음, 손으로 openapi.json을 고치면 `pnpm openapi:check` drift로 CI가 깨짐) / `audit.InferLanguages`가 동점 신뢰도를 알파벳순으로만 깨서 대표 언어가 자의적으로 결정되는 문제를 근거 개수 기준으로 개선 / redact.go IPv4 규칙의 "사설망 제외" 주석과 실제 동작(전부 마스킹) 불일치 정리 / `EstimateTokens`의 `[]rune(text)` 전체 복사를 `utf8.RuneCountInString`으로 교체 / `MODEL_PRICING_KRW_PER_1M` 키가 소문자 정규화되지 않아 `lookupPrice`의 정확 매칭을 항상 놓치고 prefix 루프로만 걸리는 문제
- 참고: 2026-09-02(가격 longest-prefix 매칭)·2026-09-03(PROXY_API_KEYS 트림) 세션 수정 모두 아직 master에 병합되지 않아, master의 `audit.lookupPrice`는 여전히 첫 prefix 일치를 반환한다. 중복 작업하지 말 것.
- 릴리즈: v0.82.2 (2026-09-04)
