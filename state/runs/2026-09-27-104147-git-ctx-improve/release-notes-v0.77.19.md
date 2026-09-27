# git-ctx v0.77.19

이번 릴리스는 **진단용 오류 문자열을 바이트 예산에 맞춰 자를 때 마지막 UTF-8 문자가 깨지던 문제**를 임베딩 런타임과 Milvus 벡터 저장소에서 고칩니다. 한글은 한 글자가 3바이트라 예산이 글자 중간에 떨어지면 invalid UTF-8 이 남았고, `encoding/json` 이 그 꼬리를 `U+FFFD` 로 바꿔 내보내므로 관리 콘솔의 상태 JSON 을 읽는 운영자는 깨진 글자를 받았습니다. 기본 서비스 포트는 계속 `4747`이며 기존 API·MCP 호환성을 유지합니다.

> **재색인도 마이그레이션도 필요하지 않습니다.** 저장되는 내용과 검색 결과는 바뀌지 않으며, 바뀌는 것은 진단문을 자르는 위치뿐입니다.

## 수정

### 진단문을 자른 자리가 글자 경계가 아니었습니다

- `internal/embedding/runtime.go`의 `clipRuntimeError` 는 `value[:limit] + "…"` 로, `internal/vectorstore/milvus.go`의 `truncate` 는 `value[:limit]` 로 바이트 경계에서 잘랐습니다. 저장소에 남아 있던 마지막 두 자리였습니다.
- 두 결과 모두 운영자에게 도달합니다. `clipRuntimeError` 는 `Runtime.Snapshot().LastError`(`json:"lastError,omitempty"`)를 거쳐 관리 콘솔 상태 JSON(`internal/app/health.go:325`·`339`)에 실립니다. Milvus 쪽은 `milvusStore.Status`(`milvus.go:251-253`) → `vectorstore.TestConnection`(`vectorstore.go:178-180`) → `app.searchBackendHealth`(`health.go:478-480`) → `adminHealth` 의 `jsonOut`(`health.go:507`) 경로로 같은 JSON 에 실립니다.
- 두 자리 모두 연속 바이트에서 물러나 자르도록 고쳤습니다 — `for cut > 0 && value[cut]&0xC0 == 0x80 { cut-- }`. 이미 옳게 동작하던 형제 절단기 `source.truncateError`·`mcp.runeSafeCut`·`search.cutAtRuneBoundary` 와 같은 관용구이며, 순환 임포트를 만들지 않도록 이번에도 각 패키지에 두고 그 사유를 계약 주석에 적었습니다.
- 예산 상수(`300`·`400`), 임베딩의 `…` 접미사, Milvus 의 `"milvus %s: %s"` 문구는 한 글자도 바꾸지 않았습니다.

## 영향 범위

- 잘린 진단문의 끝에서 **깨진 글자 대신 온전한 글자까지만** 남습니다. 비ASCII 문자가 예산 경계에 걸칠 때만 결과가 달라지며, 그 경우 최대 3바이트가 덜 실립니다.
- ASCII 진단문과 예산보다 짧은 진단문은 전과 완전히 동일합니다.
- 오류의 발생 여부, 상태 판정(`ok`/실패), 검색 결과와 접근 판정은 바뀌지 않습니다. 관리 콘솔 JSON 의 필드 이름과 구조도 그대로입니다.
- 이미 기록된 `LastError` 스냅샷은 프로세스 메모리의 값이므로 재기동과 함께 새 규칙으로 다시 채워집니다.

## 검증

- 대역 타입을 쓰지 않고 실제 배선으로 회귀 시험했습니다. 임베딩은 실제 `NewRuntime()` 과 실제 `Runtime.Guard("model", RuntimePolicy{}, factory)` 가 `finish` 를 거쳐 기록한 `Snapshot().LastError` 를, 벡터 저장소는 실제 `Open(FromMap{…milvus…})` 와 `httptest` 503 응답, 430바이트 한국어 본문으로 얻은 실제 `Status(ctx)` 의 오류 문자열을 단언합니다.
- 두 시험 모두 3바이트 글자가 예산에 걸치는 패딩(`limit-2`·`limit-1`)과 걸치지 않는 패딩(`limit-3`·`limit`)을 함께 도는 경계 표입니다. 수정 전 코드에서 걸치는 두 패딩만 실패함을 먼저 확인했습니다 — `pad=298`·`pad=299`, `pad=398`·`pad=399` 가 `is not valid UTF-8` 로 실패했고, `pad=297`·`pad=300`·`pad=397`·`pad=400` 은 수정 전에도 통과하는 동작 무변경 대조군입니다.
- UTF-8 단언과 함께 관리 콘솔 JSON 에 `U+FFFD` 이스케이프가 실리지 않음을 직렬화 결과의 바이트로 단언합니다.
- 일반·FTS5 빌드와 전체 테스트, 전체 race, `vet`·`gofmt` 통과
- 버전 정합성·회귀 시험, Kubernetes Kustomize 렌더링과 콘솔 구문·계약 시험 통과
- `govulncheck ./...` (v1.7.0): `No vulnerabilities found`
- 로컬 검증 결과는 `docs/completion-audit.md`에 기록했습니다.
- PostgreSQL·pgvector·Vault 통합, Docker 이미지 빌드·오프라인 아카이브 검증 및 GitHub 공개는 태그 푸시 후 릴리스 워크플로에서 수행합니다.

## 업그레이드 참고

- 마이그레이션과 재색인은 필요하지 않습니다.
- 설정·환경변수·API 스키마 변경은 없습니다.
- 잘린 진단문을 바이트 길이로 비교하는 스크립트가 있다면, 비ASCII 문자가 경계에 걸친 경우에 한해 길이가 최대 3바이트 짧아질 수 있습니다.

## 오프라인 Docker 이미지

릴리스 워크플로가 다음 두 파일을 만들어 검증 후 첨부합니다.

- `git-ctx-v0.77.19.tar.gz`
- `git-ctx-v0.77.19.tar.gz.sha256`

```bash
sha256sum -c git-ctx-v0.77.19.tar.gz.sha256
gzip -dc git-ctx-v0.77.19.tar.gz | docker load
docker image inspect git-ctx:v0.77.19 --format '{{.Os}}/{{.Architecture}} {{.Config.User}}'
```

기대 결과는 `linux/amd64 10001`입니다. 아카이브에는 `git-ctx:v0.77.19`과 `git-ctx:0.77.19` 태그가 포함됩니다.

**전체 변경 내역**: https://github.com/hkjang/git-ctx/compare/v0.77.18...v0.77.19
