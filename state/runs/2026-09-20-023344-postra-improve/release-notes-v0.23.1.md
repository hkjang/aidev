# Postra v0.23.1 — gosec 검사 복구·SMTP 어댑터 테스트

v0.23.0의 DCR 호환 OAuth 프록시 커밋이 들여온 gosec 오탐 2건으로 main 의 CI `gosec (medium+)` 잡이 계속 실패하던 것을 바로잡고, 테스트가 없던 SMTP 발송 어댑터에 스크립트된 루프백 릴레이 기반 테스트를 추가한 패치 릴리즈입니다. 실행 동작·설정·환경변수·DB 변경은 없습니다.

## gosec(medium+) 검사 복구

- `gosec@v2.28.0 -severity medium` 이 v0.23.0 이후 두 곳을 지적해 CI 가 실패했습니다: `OAuthProxyTokenPath = "/oauth/token"` 상수(G101, 이름의 "Token" 에 걸린 URL 경로)와 사전 등록 클라이언트의 업스트림 토큰 응답을 그대로 전달하는 `w.Write(body)`(G705, XSS 오염 분석). 둘 다 자격 증명이나 HTML 이 아닙니다.
- ci.yml 이 요구하는 대로 두 곳에 `#nosec Gxxx -- 이유` 주석만 달았습니다. G705 의 근거는 전달 전 `json.Valid` 검사, `Content-Type: application/json`, 핸들러 입구의 `X-Content-Type-Options: nosniff` 세 가지이며, pass-through 토큰 테스트가 이 두 응답 헤더를 단언하도록 보강해 주석의 근거가 유지되도록 했습니다.
- 워크플로 파일과 검사 강도는 바꾸지 않았습니다. 수정 후 같은 명령이 exit 0 · Issues 0 입니다.

## SMTP 발송 어댑터 테스트

- `internal/adapters/smtp` 에 127.0.0.1 로 뜨는 스크립트된 릴레이 픽스처를 두고 실제 `Client` 를 `net/smtp` 경로로 구동합니다. `client.go` 는 변경하지 않았습니다.
- AUTH PLAIN/LOGIN 선택과 정확한 와이어 바이트, 535 → 영구 `AuthError`, AUTH 미광고 시 `auto` 의 비인증 진행과 명시 방식의 실패, MAIL·RCPT 의 4xx 임시/5xx 영구 분류, DATA 응답 유실 → Uncertain, STARTTLS 미광고, 자체 서명 인증서로 실제 STARTTLS 업그레이드와 implicit TLS, Send 후 비밀번호 제로화, `TestConnection` 단계 보고를 검증합니다.

## 검증

- 전체 Go race 테스트, Go vet, `gofmt` 검사, `go mod tidy` 무변경, canonical API 계약 검사
- CI 와 같은 플래그의 `gosec -severity medium -exclude-dir=scripts ./...` 로컬 재현: 수정 전 exit 1 · Issues 2, 수정 후 exit 0 · Issues 0
- `go test -race -covermode=atomic -count=1 ./internal/adapters/smtp/` 통과(86.8%), `GOMAXPROCS=2 -count=20` 반복 통과
- pass-through 토큰 테스트의 `Content-Type` 단언은 해당 헤더 설정을 지우는 변이로 실패를 확인한 뒤 원복
- 프런트엔드 소스 변경이 없어 포함된 `/app` 번들은 v0.23.0 과 동일

테스트는 로컬 루프백 릴레이와 모의 Keycloak·메일·AI 서비스를 사용하며 운영 메일 서버나 운영 데이터에 접근하지 않습니다.

## 업그레이드 및 호환성 주의사항

v0.23.0에서 추가 DB 마이그레이션, 새 설정 키, 새 필수 환경변수는 없습니다. DB·SecretStore·ObjectStore·KEK를 백업하고 유지한 상태에서 실행 파일 또는 컨테이너 이미지를 교체한 뒤 재시작하세요. 프런트엔드 소스 변경은 없으며 Go 단일 실행 파일에 React 자산·글꼴을 포함하므로 폐쇄망 런타임에 Node 서버나 외부 CDN이 필요하지 않습니다.

- 이번 변경은 소스 주석과 테스트뿐이라 실행 파일의 동작은 v0.23.0 과 같습니다. 이미 v0.23.0 을 운영 중이라면 교체를 서두를 이유는 없습니다.
- DCR 호환 OAuth 프록시, Keycloak OAuth MCP 연결, REST OIDC Bearer, API 키 동작은 v0.23.0 과 같습니다. [v0.23.0 릴리즈 내용](v0.23.0.md)과 [Keycloak OAuth MCP 설정](https://github.com/hkjang/postra/blob/v0.23.1/docs/MCP_OAUTH.md)을 참고하세요.

## 배포 파일

- `postra-0.23.1-linux-amd64`: 정적 단일 실행 파일
- `postra-0.23.1.tar.gz`: 폐쇄망 `docker load` 이미지, `postra:0.23.1` (linux/amd64)
- `postra-0.23.1-sbom.cdx.json`: Go CycloneDX SBOM
- `postra-0.23.1-frontend-sbom.cdx.json`: 프런트엔드 CycloneDX SBOM
- `SHA256SUMS.txt`: 배포 파일 SHA-256
