# git-ctx v0.77.8

이번 릴리스는 **경로 아래에 선언된 TOML 의존성을 인벤토리가 통째로 놓치던 문제**를 고칩니다. Cargo와 Poetry는 의존성을 더 긴 경로의 테이블에 적는 형태가 흔한데, 매니페스트 파서가 섹션 이름을 정확 일치로만 봐서 그런 선언이 전부 버려졌습니다. 기본 서비스 포트는 계속 `4747`이며 기존 API·MCP 호환성을 유지합니다.

> **권고 대응 중이라면 이 릴리스가 통보 대상 목록을 늘립니다.** Cargo 워크스페이스의 멤버 크레이트와 Poetry 1.2 이후의 개발·시험 그룹은 `find-dependency-usage`가 보기에 그 라이브러리를 쓰지 않는 저장소와 똑같았습니다. 이번에는 판정이 아니라 **저장되는 내용**이 바뀌므로, 답이 달라지려면 **재색인이 필요합니다.**

## 수정

### `[dependencies.serde]`는 인벤토리에 없었습니다

- `parseTOMLDependencies`는 섹션 이름이 `dependencies`·`dev-dependencies`·`tool.poetry.dependencies`와 **정확히 같을 때만** 그 안을 읽었습니다.
- 그래서 다음 네 가지가 전부 빠졌습니다 — 필드가 여럿인 선언이 흔히 쓰는 **`[dependencies.serde]`**, 모노레포가 버전을 한 곳에 모으는 **`[workspace.dependencies]`**, 플랫폼별 빌드의 **`[target.'cfg(unix)'.dependencies]`**, Poetry 1.2가 `dev-dependencies`를 대체한 **`[tool.poetry.group.dev.dependencies]`**.
- 의존성 테이블보다 경로가 **한 칸 긴 섹션은 그 한 패키지**로 읽습니다. 섹션 이름의 마지막 조각이 패키지 이름이고, 본문의 `version = "1.0"`이 버전입니다.

### `serde.workspace = true`가 `serde.workspace`라는 패키지였습니다

- 버전을 워크스페이스 루트에 모은 모노레포에서, 멤버 크레이트는 전부 `serde.workspace = true`로 선언합니다. 파서는 이 키를 통째로 이름으로 읽어 **이름 `serde.workspace`·버전 없음**인 가짜 패키지를 인벤토리에 넣었습니다.
- 그 저장소는 `serde` 권고 조회에 걸리지 않습니다. 실제로 존재하는 유일한 핀은 아무도 읽지 않는 루트 파일에 있었고, 저장소는 serde를 쓰지 않는 것처럼 보였습니다.
- **점이 든 키는 그 키가 가리키는 패키지의 필드**로 읽습니다. `serde.workspace`·`serde.features`는 이름 `serde`의 선언이 되고, 버전을 정하는 것은 `serde.version`뿐입니다. 같은 패키지가 여러 줄에 나뉘어도 하나로 합칩니다.

### 생태계별 섹션 판정을 분리했습니다

- Cargo는 섹션의 **마지막 조각**으로 판정합니다 — `dependencies`는 `direct`, `dev-dependencies`·`build-dependencies`는 `dev`. `[workspace.dependencies]`도 `[target.'cfg(unix)'.dependencies]`도 저장소의 실제 의존성입니다.
- Poetry는 `[tool.poetry.group.<이름>.dependencies]`를 그룹 이름으로 판정합니다 — `test` 그룹은 `test`, 나머지 그룹은 `dev`. PEP 621 배열과 기존 표기는 그대로입니다.

## 검증

- 새 표 `TestTOMLDependenciesStatedUnderAPath` — Cargo 6건(워크스페이스 dotted key, 서브테이블, `target` 아래 테이블, dev 서브테이블)과 필드가 패키지로 새지 않는지 확인하는 6건, Poetry 4건(서브테이블, dev 그룹, test 그룹). 수정 전에는 실패함
- `gofmt -l`, `go vet ./...`, FTS5 빌드·전체 테스트 통과, manifest·search·indexer는 `-race`도 통과
- 버전 메타데이터 정합성, Kubernetes Kustomize 렌더링, linux/amd64 Docker 이미지 빌드

## 업그레이드 참고

- 마이그레이션은 필요하지 않습니다.
- **재색인이 필요합니다.** 의존성 인벤토리는 색인할 때 만들어지므로, 이미 색인된 저장소의 저장된 선언은 다시 읽기 전까지 그대로입니다. Cargo 워크스페이스나 Poetry 그룹을 쓰는 저장소가 있다면 해당 ref를 다시 색인하세요.
- 재색인 뒤에는 **선언 수가 늘고, 이름 `*.workspace` 같은 가짜 항목이 사라집니다.** 그만큼 권고 조회에 걸리는 저장소도 늘어납니다.

## 오프라인 Docker 이미지

릴리스 자산은 아키텍처 접미사가 없는 다음 두 파일입니다.

- `git-ctx-v0.77.8.tar.gz`
- `git-ctx-v0.77.8.tar.gz.sha256`

```bash
sha256sum -c git-ctx-v0.77.8.tar.gz.sha256
gzip -dc git-ctx-v0.77.8.tar.gz | docker load
docker image inspect git-ctx:v0.77.8 --format '{{.Os}}/{{.Architecture}} {{.Config.User}}'
```

기대 결과는 `linux/amd64 10001`입니다. 아카이브에는 `git-ctx:v0.77.8`과 `git-ctx:0.77.8` 태그가 포함됩니다.

**전체 변경 내역**: https://github.com/hkjang/git-ctx/compare/v0.77.7...v0.77.8
