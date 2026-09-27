# git-ctx v0.77.20

이번 릴리스는 **MCP 응답이 파일 내용을 감싸는 코드 펜스가 그 내용보다 짧아 답변 구조가 무너지던 문제**를 `read-file`·`get-symbol-context` 두 도구에서 고칩니다. 저장소에는 코드 펜스를 그대로 보여 주는 파일이 실제로 있고(사용 예를 담은 파이썬 docstring, 4백틱 안에 3백틱 블록을 보여 주는 README), 그런 내용의 펜스가 답변이 연 블록을 내용 한가운데서 닫아 버리면 뒤따르는 인용·`### Notes`·절단 공지가 파일 본문으로 읽혔습니다. 기본 서비스 포트는 계속 `4747`이며 기존 API·MCP 호환성을 유지합니다.

> **재색인도 마이그레이션도 필요하지 않습니다.** 저장되는 내용과 검색 결과는 바뀌지 않으며, 바뀌는 것은 응답이 내용을 감쌀 때 쓰는 백틱 개수뿐입니다.

## 수정

### 코드 펜스가 감싸는 내용보다 짧았습니다

- `internal/mcp/format.go`의 두 경로가 같은 입력을 다르게 읽었습니다. `formatFileContent` 는 내용에 백틱 세 개가 있으면 네 개로만 올렸고, `formatSymbolContext` 는 아예 올리지 않고 언제나 백틱 세 개로 감쌌습니다.
- 그래서 4백틱 블록 안에 3백틱 블록을 보여 주는 README 는 `read-file` 에서도 무너졌고, 사용 예에 펜스를 담은 docstring 을 가진 심볼은 `read-file` 로는 살아남아도 `get-symbol-context` 로 읽으면 무너졌습니다.
- 무너진 뒤의 결과는 답변 전체에 걸칩니다. 내용의 펜스가 블록을 닫으면 그 뒤의 `Source:` 인용과 `### Notes`, 그리고 `finishCall`(`internal/mcp/dispatch.go:376-382`)이 예산 적용 뒤에 덧붙이는 인자 안내·신선도 안내가 파일 본문으로 읽히고, 답변의 마지막 펜스는 끝내 닫히지 않는 블록을 하나 더 엽니다.
- 두 경로가 새 헬퍼 `contentFence` 하나를 함께 쓰도록 고쳤습니다 — 내용 안의 가장 긴 백틱 연속보다 한 개 긴 펜스이며, 최소 세 개입니다. 백틱 세 개를 담은 내용은 종전과 똑같이 네 개가 되므로 `read-file` 의 기존 계약은 그대로입니다.

## 영향 범위

- 백틱을 담지 않거나 두 개 이하만 담은 내용은 전과 완전히 동일한 백틱 세 개로 감싸입니다.
- 백틱 세 개를 담은 내용은 전과 동일하게 네 개로 감싸입니다. 달라지는 것은 백틱 네 개 이상을 담은 내용(다섯 개 이상으로 감싸임)과, 지금까지 전혀 올리지 않던 `get-symbol-context` 입니다.
- 도구 이름·인자·응답 필드 구조와 인용 형식, 저장되는 청크와 검색 결과, 접근 판정은 바뀌지 않습니다.
- 알려진 남은 제약: `closeOpenFence`(`internal/mcp/budget.go:111-119`)는 답변이 예산에 걸려 **절단될 때** 여는 펜스 길이를 모른 채 언제나 백틱 세 개를 붙입니다. 이 릴리스 이전에도 `read-file` 은 네 개를 내보냈으므로 선존 제약이며, 이제 `get-symbol-context` 도 같은 경로에 들어갑니다. 별도 회차에서 다룹니다.

## 검증

- 대역 타입을 쓰지 않고 실제 배선으로 회귀 시험했습니다. 실제 `store.Open("sqlite", …)` fixture 와 실제 `mcp.Server.ServeHTTP` 의 JSON-RPC `tools/call` 로 `get-symbol-context`(실제 `search.SymbolContext` → `document_chunks` 재조립)와 `read-file`(실제 `search.ReadFile` → `repository_files` 청크 재조립) 두 경로를 각각 호출하고 응답 텍스트를 단언합니다.
- 응답을 CommonMark 방식으로 읽는 시험 헬퍼(백틱 세 개 이상인 줄이 블록을 열고, 같은 길이 이상의 백틱만 있는 줄이 닫음)로 ① 끝에 열린 채 남은 블록이 없음 ② 블록이 정확히 하나 ③ 본문 마지막 줄이 그 블록 안에 있음을 단언합니다.
- 수정 전 코드에서 두 케이스가 모두 실패함을 먼저 확인했습니다 — `the answer ends inside a code fence that never closes`, `the content was split across 2 code blocks, want 1`, `"def probe():" is outside the code block the answer opened`.
- 일반·FTS5 빌드와 전체 테스트, 전체 race, `vet`·`gofmt` 통과
- 버전 정합성·회귀 시험, Kubernetes Kustomize 렌더링과 콘솔 구문·계약 시험 통과
- `govulncheck ./...`: `No vulnerabilities found` (로컬 스캐너 v1.6.0, 릴리스 워크플로는 v1.7.0 으로 다시 수행)
- 로컬 검증 결과는 `docs/completion-audit.md`에 기록했습니다.
- PostgreSQL·pgvector·Vault 통합, Docker 이미지 빌드·오프라인 아카이브 검증 및 GitHub 공개는 태그 푸시 후 릴리스 워크플로에서 수행합니다.

## 업그레이드 참고

- 마이그레이션과 재색인은 필요하지 않습니다.
- 설정·환경변수·API 스키마 변경은 없습니다.
- MCP 응답 본문을 고정된 백틱 세 개 또는 네 개로 파싱하는 클라이언트 측 스크립트가 있다면, 여는 줄의 백틱 개수를 읽어 같은 길이 이상으로 닫는 줄을 찾도록 바꾸십시오. 표준 마크다운 파서를 쓰는 클라이언트는 손댈 것이 없습니다.

## 오프라인 Docker 이미지

릴리스 워크플로가 다음 두 파일을 만들어 검증 후 첨부합니다.

- `git-ctx-v0.77.20.tar.gz`
- `git-ctx-v0.77.20.tar.gz.sha256`

```bash
sha256sum -c git-ctx-v0.77.20.tar.gz.sha256
gzip -dc git-ctx-v0.77.20.tar.gz | docker load
docker image inspect git-ctx:v0.77.20 --format '{{.Os}}/{{.Architecture}} {{.Config.User}}'
```

기대 결과는 `linux/amd64 10001`입니다. 아카이브에는 `git-ctx:v0.77.20`과 `git-ctx:0.77.20` 태그가 포함됩니다.

**전체 변경 내역**: https://github.com/hkjang/git-ctx/compare/v0.77.19...v0.77.20
