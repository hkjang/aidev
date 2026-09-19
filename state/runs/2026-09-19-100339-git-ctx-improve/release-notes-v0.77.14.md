# git-ctx v0.77.14

이번 릴리스는 **릴리스 워크플로가 코드 변경 없이 같은 태그에서 반복 실패하던 문제**를 고칩니다. 도달 가능한 취약 의존성을 막는 `govulncheck` 단계가 `google.golang.org/grpc` v1.82.1의 GO-2026-6348을 보고했고, 취약점 데이터베이스는 매일 갱신되므로 v0.77.13 태그는 손대지 않았는데도 다시 실패했습니다. 애플리케이션 코드는 바뀌지 않았으며, 기본 서비스 포트는 계속 `4747`이고 기존 API·MCP 호환성을 유지합니다.

> **이 릴리스는 의존성만 올립니다.** 저장되는 내용도 판정도 바뀌지 않으므로 재색인은 필요하지 않습니다. v0.77.13 오프라인 이미지가 배포되지 않은 환경이라면 이 이미지가 그 내용을 그대로 담고 있습니다.

## 수정

### 같은 태그가 두 번, 같은 이유로 실패했습니다

- `release.yml`의 `Verify reachable dependencies` 단계는 `govulncheck ./...`를 실행하고, 도달 가능한 취약점이 하나라도 있으면 종료 코드 3으로 빌드를 멈춥니다. v0.77.13 트리에서 이 단계는 **GO-2026-6348 — HTTP/2 DATA 프레임 조각화로 인한 힙 메모리 소진(OOM)** — 을 `google.golang.org/grpc@v1.82.1`에서 찾았습니다. grpc는 직접 의존성이 아니라 OTLP 추적 내보내기(`otlptracehttp`)가 끌고 오는 간접 의존성이고, `search.Query`에서 `transport.NewHTTP2Client`까지 호출 경로가 이어져 있어 도달 가능으로 판정됐습니다.
- 취약점 데이터베이스는 매일 갱신됩니다. 태그가 만들어질 때 통과한 검사가 다음 날 같은 커밋에서 실패할 수 있고, 그래서 **코드가 안 바뀐 태그가 반복해서 같은 단계에서 멈췄습니다.** 워크플로나 검사를 완화하는 대신 취약점을 닫습니다.
- `google.golang.org/grpc`를 **v1.83.2**로 올립니다. GO-2026-6348과 함께 GO-2026-6443도 닫힙니다. grpc가 요구하는 `golang.org/x/net` v0.58.0(GO-2026-5942 닫힘)·`x/sync` v0.22.0·`x/sys` v0.47.0·`x/text` v0.41.0이 따라 올라갑니다.
- 바뀐 파일은 `go.mod`·`go.sum`뿐입니다. 전부 간접 의존성의 패치·마이너 갱신이고, `go 1.25.0`과 `toolchain` 줄은 그대로입니다.

## 검증

- `govulncheck ./...` — 수정 전 종료 코드 3(GO-2026-6348 도달 가능), 수정 후 `No vulnerabilities found`. 모듈 수준까지 0건
- `gofmt -l`, `go vet ./...`, FTS5 빌드·전체 테스트 통과, `-race`도 통과
- 버전 메타데이터 정합성, Kubernetes Kustomize 렌더링, linux/amd64 Docker 이미지 빌드·오프라인 아카이브 재적재 검증

## 업그레이드 참고

- 마이그레이션은 필요하지 않습니다.
- **재색인은 필요하지 않습니다.** 마스킹·인벤토리·검색 코드는 v0.77.13과 동일합니다.
- v0.77.13에서 올린 파이썬 락파일 인벤토리 변경을 아직 받지 않았다면, 그 릴리스의 재색인 안내가 이 이미지에도 그대로 적용됩니다.

## 오프라인 Docker 이미지

릴리스 자산은 아키텍처 접미사가 없는 다음 두 파일입니다.

- `git-ctx-v0.77.14.tar.gz`
- `git-ctx-v0.77.14.tar.gz.sha256`

```bash
sha256sum -c git-ctx-v0.77.14.tar.gz.sha256
gzip -dc git-ctx-v0.77.14.tar.gz | docker load
docker image inspect git-ctx:v0.77.14 --format '{{.Os}}/{{.Architecture}} {{.Config.User}}'
```

기대 결과는 `linux/amd64 10001`입니다. 아카이브에는 `git-ctx:v0.77.14`과 `git-ctx:0.77.14` 태그가 포함됩니다.

**전체 변경 내역**: https://github.com/hkjang/git-ctx/compare/v0.77.13...v0.77.14
