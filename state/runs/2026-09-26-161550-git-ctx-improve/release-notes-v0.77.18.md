# git-ctx v0.77.18

이번 릴리스는 **감싸인(wrapped) 데드라인 오류가 호출 추적에 `timeout` 이 아니라 `error` 로 기록되던 문제**를 고칩니다. 같은 오류를 두고 사용자에게 보이는 진단문은 "tool timeout" 이라고 말하는데 감사 행에는 `error` 가 남아, 운영자가 상태 값으로 타임아웃을 셀 수 없었습니다. 기본 서비스 포트는 계속 `4747`이며 기존 API·MCP 호환성을 유지합니다.

> **재색인도 마이그레이션도 필요하지 않습니다.** 저장되는 내용과 검색 결과는 바뀌지 않으며, 바뀌는 것은 새로 기록되는 단계의 `status` 값뿐입니다.

## 수정

### 감싸인 데드라인이 타임아웃으로 분류되지 않았습니다

- `internal/calltrace/calltrace.go`의 `Span.Fail` 은 `err == context.DeadlineExceeded` 로 값 비교를 했습니다. 저장소에서 데드라인을 값으로 비교하던 유일한 자리였고, 다른 12곳은 이미 `errors.Is` 를 씁니다.
- 원격 소스의 데드라인은 감싸여서 도착합니다. `net/http` 가 `*url.Error` 로 감싸고 호출자가 다시 단계 이름을 덧붙이므로, 값 비교는 실제 타임아웃을 평범한 오류로 분류했습니다. bare `context.DeadlineExceeded` 만 `timeout` 으로 통과했습니다.
- 같은 `sourceErr` 를 읽는 두 자리가 어긋나 있던 것이 근거였습니다 — `internal/search/service.go:3326`(→`Fail`)은 `error` 로 기록하고, 3415(`errors.Is`)는 같은 오류를 타임아웃으로 읽어 진단문을 썼습니다. 값 비교를 `errors.Is(err, context.DeadlineExceeded)` 로 바꿔 한 줄 고쳤고, 감싸인 데드라인이 도착하는 이유를 주석에 적었습니다.
- 상태 문자열 상수의 값과 호출부(`internal/search`·`internal/mcp`)는 한 줄도 건드리지 않았습니다.

## 영향 범위

- `mcp_call_steps.status` 와 `mcp_calls.trace_summary`(`internal/mcp/dispatch.go:387-404`)에 **새로 기록되는** 행만 달라집니다. 원격 소스가 데드라인으로 실패한 단계가 `error` 대신 `timeout` 으로 저장되어, 진단문과 감사 행이 같은 말을 합니다.
- **이미 저장된 행은 바뀌지 않습니다.** 과거 기간을 상태 값으로 집계하면 그 구간의 타임아웃은 여전히 `error` 로 남아 있습니다.
- 검색 결과, 접근 판정, 오류 발생 여부 자체는 바뀌지 않습니다. 실패한 호출은 전과 똑같이 실패하며 분류만 정확해집니다.
- `context.Canceled` 는 이번에도 별도 상태로 나누지 않았습니다. 새 상태 상수는 `mcp_call_steps.status` 계약과 관리 콘솔을 함께 건드리는 변경이라 분리해 뒀습니다.

## 검증

- 실제 `Recorder`·`Span` 과 실제 `fmt.Errorf("%w", …)`·`&url.Error{Err: context.DeadlineExceeded}` 를 쓰는 5케이스 표 회귀 시험(`TestFailClassifiesWrappedDeadlinesAsTimeout`). 대역 타입을 쓰지 않습니다.
- 수정 전 코드에서 예측한 두 케이스만 실패함을 먼저 확인했습니다 — `wrapped_by_the_caller` 와 `wrapped_by_net/http` 가 `status="error", want "timeout"`. bare 데드라인·일반 오류·`nil` 세 케이스는 수정 전에도 통과하는 동작 무변경 대조군입니다.
- `Step.Detail` 이 감싼 오류의 전문과 같음을, `Summary()` 가 `"source-query gitlab: timeout"` 과 정확히 일치함을 함께 단언합니다.
- 일반·FTS5 빌드와 전체 테스트, 전체 race, `vet`·`gofmt` 통과
- 버전 정합성·회귀 시험, Kubernetes Kustomize 렌더링과 콘솔 구문·계약 시험 통과
- `govulncheck ./...` (v1.7.0): `No vulnerabilities found`
- 로컬 검증 결과는 `docs/completion-audit.md`에 기록했습니다.
- PostgreSQL·pgvector·Vault 통합, Docker 이미지 빌드·오프라인 아카이브 검증 및 GitHub 공개는 태그 푸시 후 릴리스 워크플로에서 수행합니다.

## 업그레이드 참고

- 마이그레이션과 재색인은 필요하지 않습니다.
- 설정·환경변수·API 스키마 변경은 없습니다.
- 호출 추적 상태로 타임아웃을 집계하는 대시보드가 있다면, 이 릴리스를 배포한 시점을 경계로 같은 오류가 `error` 에서 `timeout` 으로 옮겨 갑니다. 두 상태의 합은 바뀌지 않습니다.

## 오프라인 Docker 이미지

릴리스 워크플로가 다음 두 파일을 만들어 검증 후 첨부합니다.

- `git-ctx-v0.77.18.tar.gz`
- `git-ctx-v0.77.18.tar.gz.sha256`

```bash
sha256sum -c git-ctx-v0.77.18.tar.gz.sha256
gzip -dc git-ctx-v0.77.18.tar.gz | docker load
docker image inspect git-ctx:v0.77.18 --format '{{.Os}}/{{.Architecture}} {{.Config.User}}'
```

기대 결과는 `linux/amd64 10001`입니다. 아카이브에는 `git-ctx:v0.77.18`과 `git-ctx:0.77.18` 태그가 포함됩니다.

**전체 변경 내역**: https://github.com/hkjang/git-ctx/compare/v0.77.17...v0.77.18
