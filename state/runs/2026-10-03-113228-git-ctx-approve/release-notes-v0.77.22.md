# git-ctx v0.77.22

이번 릴리스는 **MCP 응답 예산 계층이 "보여 주는 내용"의 마크다운을 "답변의 구조"로 읽던 문제**를 고치고, OTLP 추적 내보내기의 알려진 취약점(`GO-2026-6505`)을 위해 **OpenTelemetry 를 v1.45.0 으로 올립니다.** `read-file`·`get-symbol-context` 는 파일이나 심볼 본문 전체를 자기 코드 펜스로 감싸는데, 예산 계층은 그 펜스 안의 `### ` 제목과 `- ` 항목을 결과 구역의 경계로 세고 그 자리에서 잘랐습니다. 그래서 자체 제목을 가진 문서는 예산을 다 쓰지 못한 채 짧게 잘리고, 보고되는 결과 개수는 그 문서의 제목 개수였습니다. 기본 서비스 포트는 계속 `4747`이며 기존 API·MCP 호환성을 유지합니다.

> **재색인도 마이그레이션도 필요하지 않습니다.** 저장되는 내용과 검색 결과는 바뀌지 않으며, 바뀌는 것은 예산에 걸린 응답의 절단 자리와 보고되는 결과 개수뿐입니다.

## 수정

### 내용 자신의 제목을 결과 구역의 경계로 셌습니다

- `sectionCount`(`internal/mcp/budget.go`)는 `\n### ` 를 세고, 그런 제목이 없으면 `\n- ` 를 셉니다. 그런데 `formatFileContent`(`read-file`)와 `formatSymbolContext`(`get-symbol-context`)는 보여 주는 내용 **전체**를 답변 자신의 코드 펜스 안에 넣으므로, 그 안의 `### ` 와 `- ` 는 결과 사이의 이음새가 아니라 내용의 일부입니다.
- `cutAtBoundary` 도 같은 제목을 절단 경계로 썼습니다. 그래서 창 중간에 자체 `### ` 제목이 있는 문서를 읽으면 **요청한 예산보다 3분의 1 가까이 적게** 돌려주면서, 절단 공지는 여전히 예산에 맞춰 잘랐다고 말했습니다 — 1만 2천 바이트짜리 `read-file` 이 8천 바이트로 답해졌습니다.
- 결과 개수도 같은 자리에서 틀렸습니다. `mcp_calls.result_count` 에는 그 문서의 제목 개수가 기록되었고, 단계를 `- ` 항목으로 나열하는 심볼 본문은 **단계 하나당 결과 하나**로 집계되었습니다.
- `closeOpenFence` 는 이미 펜스가 어디인지 줄 단위로 알고 있었습니다. 그 주사(走査)를 `lineFence` 로 떼어내 개수 세기와 절단이 함께 쓰도록 했습니다. 펜스를 닫는 계층과 개수를 세고 자르는 계층이 이제 **무엇이 내용인지에 대해 같은 답**을 갖습니다.

### 펜스 인식을 내용을 펜스로 감싸는 도구에만 적용합니다

- 펜스 안을 건너뛰는 규칙을 모든 답변에 적용하면 반대 방향으로 틀립니다. `fencesContent` 는 그 규칙을 받을 도구를 `read-file`·`get-symbol-context` **둘로만** 한정합니다. 이 두 경로만 내용을 답변 자신의 펜스로 감싸고, 둘 다 `contentFence`(`format.go:207`)로 내용 속 가장 긴 백틱 연속보다 긴 펜스를 고르므로 내용이 블록을 먼저 닫을 수 없으며, 인용과 `### Notes` 는 언제나 그 블록 밖에 놓입니다.
- 다른 형식기는 색인된 내용을 **펜스 없는 산문으로** 씁니다 — `formatSemanticSearch`(`format.go:257`)와 `formatRunbooks`(`format.go:485`)는 청크 하나를 `### ` 제목 아래에, `formatChangeRequests`(`format.go:310`)는 머지 리퀘스트 설명을, `formatCodeSearch`(`format.go:118`)는 코드 조각을 씁니다. 청크는 원본 파일의 마크다운을 그대로 담고 청킹은 펜스를 따라가지 않고 제목에서 나누므로(`indexer.go:1110`), 한 청크에 백틱 연속이 짝 없이 남는 것은 **이상한 일이 아닙니다.**
- 그 백틱 연속을 답변 자신의 펜스로 읽으면 그 아래의 적중이 개수와 절단 양쪽에서 전부 사라집니다 — 적중 열한 건짜리 `search-semantic` 응답이 여덟 건으로 감사되었습니다. 코드 검색의 적중 개수를 바이트 주사로 유지하는 것도 같은 이유입니다.

### OpenTelemetry 를 v1.45.0 으로 올렸습니다

- `GO-2026-6505`: OTLP 추적 내보내기가 내보내기 대상 endpoint URL 을 로그에 남길 수 있었습니다. git-ctx 의 OTLP HTTP tracing 은 관리자가 동적으로 켜고 끄며 endpoint 를 설정값으로 받으므로, 이 경로에 해당합니다.
- `go.opentelemetry.io/otel`·`otel/sdk`·`otel/trace`·`otlptracehttp` 를 v1.44.0 에서 v1.45.0 으로, 함께 끌려오는 `go.opentelemetry.io/proto/otlp` 를 v1.11.0 으로, `github.com/go-logr/logr` 를 v1.4.4 로 올렸습니다. 애플리케이션 코드 변경은 없습니다.

## 영향 범위

- 예산에 걸리지 않은 응답은 전과 완전히 동일합니다. 두 수정 모두 예산에 걸린 응답의 경로입니다.
- `read-file`·`get-symbol-context` 는 이제 내용 속 제목에서 멈추지 않고 예산을 끝까지 씁니다. 같은 인자로 같은 파일을 읽으면 종전보다 **더 많은 바이트**를 받을 수 있습니다.
- `read-file`·`get-symbol-context` 의 결과 개수는 이제 내용의 제목·항목 수를 따라 부풀지 않습니다. 이 값을 쓰는 자동화가 있다면 확인하십시오.
- `search-semantic`·`search-runbooks`·`search-change-requests`·`search-code` 의 결과 개수와 절단 자리는 바이트 주사를 그대로 유지하므로 전과 같습니다.
- 도구 이름·인자·응답 필드 구조와 인용 형식, 저장되는 청크와 검색 결과, 접근 판정은 바뀌지 않습니다.

## 검증

- 대역 타입을 쓰지 않고 실제 배선으로 회귀 시험했습니다. 실제 `store.Open("sqlite", …)` fixture 와 실제 `mcp.Server.ServeHTTP` 의 JSON-RPC `tools/call` 로 도구를 호출하고 응답 텍스트를 단언합니다.
- `internal/mcp/fenced_content_test.go` 는 자체 `### ` 제목과 `- ` 항목을 가진 문서를 `read-file`·`get-symbol-context` 로 읽을 때 절단이 내용 속 제목에서 멈추지 않고, 보고되는 개수가 내용의 제목 수를 따라가지 않음을 단언합니다.
- `internal/mcp/unfenced_content_test.go` 는 반대 방향의 대조군입니다. 짝 없는 백틱 연속을 담은 청크가 있어도 `search-semantic` 등 산문 경로의 적중 개수와 절단 자리가 전과 같음을 단언합니다.
- 수정 전 코드에서 해당 케이스들이 실패함을 먼저 확인했습니다.
- 일반·FTS5 빌드와 전체 테스트, 전체 race, `vet`·`gofmt` 통과
- 버전 정합성·회귀 시험, Kubernetes Kustomize 렌더링과 콘솔 구문·계약 시험 통과
- `govulncheck ./...`: `No vulnerabilities found` (로컬 스캐너 v1.6.0, 릴리스 워크플로는 v1.7.0 으로 다시 수행)
- 로컬 검증 결과는 `docs/completion-audit.md`에 기록했습니다.
- PostgreSQL·pgvector·Vault 통합, Docker 이미지 빌드·오프라인 아카이브 검증 및 GitHub 공개는 태그 푸시 후 릴리스 워크플로에서 수행합니다.

## 업그레이드 참고

- 마이그레이션과 재색인은 필요하지 않습니다.
- 설정·환경변수·API 스키마 변경은 없습니다.
- OTLP tracing 을 쓰는 배포는 이 버전으로 올리는 것으로 `GO-2026-6505` 가 해소됩니다. 설정값을 다시 넣을 필요는 없습니다.
- `read-file`·`get-symbol-context` 응답의 결과 개수를 쓰는 자동화가 있다면, 그 값이 더 이상 내용의 제목 개수를 따라가지 않는다는 점을 반영하십시오.

## 오프라인 Docker 이미지

릴리스 워크플로가 다음 두 파일을 만들어 검증 후 첨부합니다.

- `git-ctx-v0.77.22.tar.gz`
- `git-ctx-v0.77.22.tar.gz.sha256`

```bash
sha256sum -c git-ctx-v0.77.22.tar.gz.sha256
gzip -dc git-ctx-v0.77.22.tar.gz | docker load
docker image inspect git-ctx:v0.77.22 --format '{{.Os}}/{{.Architecture}} {{.Config.User}}'
```

기대 결과는 `linux/amd64 10001`입니다. 아카이브에는 `git-ctx:v0.77.22`과 `git-ctx:0.77.22` 태그가 포함됩니다.

**전체 변경 내역**: https://github.com/hkjang/git-ctx/compare/v0.77.21...v0.77.22
