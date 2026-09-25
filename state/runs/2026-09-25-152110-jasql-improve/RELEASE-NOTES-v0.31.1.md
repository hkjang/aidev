# JASQL v0.31.1

문서 패치 릴리즈입니다. **서버 동작은 v0.31.0과 같습니다** — 코드 변경은 버전
상수와 새 테스트뿐이며, 기능·API·설정은 그대로입니다.

## 개발자 가이드를 소스와 다시 맞췄습니다

`docs/development.md`가 소스와 어긋나 있어 항목을 하나씩 소스에서 확인해
고쳤습니다.

- **Go 버전** `1.24+` → go.mod 의 `go 1.25.0`
- **의존성** "외부 의존성 0 · go.sum 없음" → godror / pgx / x-crypto 와 go.sum
  존재. godror 는 `oracle` 빌드 태그 전용이고 cgo 가 필요하므로 기본 빌드,
  크로스 빌드, CI 는 태그 없이 돈다는 점까지 명시
- **전체 테스트 시간** `~17초` → 실측 약 70초(catalog 약 56초 + mcp 약 10초,
  측정 머신 기준 대략치)
- **MCP 도구 수** 24 → 실제 `tools/list` 응답 기준 42
- **코드 트리**에 빠져 있던 `internal/meta`, `internal/oracle`, `scripts`,
  `.github` 추가, 테스트 배치 표 갱신
- 새로 들어온 **CI 절**(`.github/workflows/ci.yml`: push·PR 마다 build / vet /
  test) 추가
- `jasql-goldgen` 의 기본 `-out` 이 `data/kcb/golden_queries.json` 을
  덮어쓴다는 **경고** 추가

## 다시 어긋나지 않게 테스트로 고정했습니다

`internal/mcp/docs_test.go` 가 문서에 적힌 도구 수를 뽑아, 실제 `ServeStdio`
`tools/list` 왕복으로 센 수와 비교합니다. 대역이 아니라 프로덕션 전송 경로를
쓰므로 도구가 늘거나 줄면 문서를 고치기 전까지 테스트가 빨개집니다. 문서에서
도구 수 표기를 지워도 실패합니다.

## 검증

- `go build ./...` / `go vet ./...` / `go test ./...` 전부 통과
  (catalog 56.978s, mcp 10.018s, meta 0.493s, oracle 0.011s)
- 문서에 적힌 명령을 실제로 실행해 확인: `jasql-eval -verbose`,
  `jasql-goldgen -data <tmp> -n 80 -out <tmp>`(실데이터 무변경), Windows
  크로스 빌드
- 릴리즈 이미지 `jasql-mcp:v0.31.1` 을 띄워 `/healthz`(카탈로그 190테이블
  컴파일)와 `/openapi.json`(`"version": "0.31.1"`) 응답 확인

## 자산

| 파일 | 설명 |
| --- | --- |
| `jasql-mcp-0.31.1-docker.tar.gz` | 순수 Go 도커 이미지 `jasql-mcp:v0.31.1` (DB 실행 없음) |
| `jasql-mcp-linux-amd64` / `jasql-mcp-linux-arm64` / `jasql-mcp-windows-amd64.exe` | 단독 실행 바이너리 |
| `DEPLOY-OFFLINE.md` | 오프라인망 배포 가이드 |
| `SHA256SUMS-v0.31.1.txt` | 위 자산의 SHA256 |

> Oracle Instant Client 내장 이미지(`jasql-mcp-v0.31.1.tar.gz`)는 이번 자산에
> 포함되지 않았습니다. Instant Client 는 재배포 조건이 따로 있어 저장소에 없으며,
> 빌드 머신에 `instantclient/` 를 준비한 뒤 `scripts/release-image.sh` 로 동일한
> 이름 규칙(`dist/jasql-mcp-v0.31.1.tar.gz`)으로 만들 수 있습니다. 동작은
> v0.31.0 이미지와 같으므로, 폐쇄망에 이미 반입한 v0.31.0 이미지를 계속 쓰셔도
> 됩니다.

## 업그레이드

```sh
docker load -i jasql-mcp-0.31.1-docker.tar.gz
docker rm -f jasql-mcp
docker run -d --name jasql-mcp --restart unless-stopped \
  -p 8787:8787 -v jasql-data:/app/data/kcb \
  -e JASQL_META_DB='...' -e JASQL_ADMIN_TOKEN='...' \
  jasql-mcp:v0.31.1
```

볼륨과 메타 DB는 그대로 둡니다. 스키마 변경이 없어 v0.31.0 이미지로의 롤백도
데이터 손실 없이 가능합니다.
