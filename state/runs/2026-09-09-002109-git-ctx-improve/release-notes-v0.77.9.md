# git-ctx v0.77.9

이번 릴리스는 **pyproject가 선언한 의존성 배열을 인벤토리가 한 줄에서 끊거나 아예 읽지 않던 문제**를 고칩니다. extra를 쓰는 요구사항 하나가 PEP 621 배열을 문자열 한가운데서 잘랐고, 개발·시험 도구를 두는 표준 자리인 나머지 배열은 읽히지 않았습니다. 기본 서비스 포트는 계속 `4747`이며 기존 API·MCP 호환성을 유지합니다.

> **권고 대응 중이라면 이 릴리스가 통보 대상 목록을 늘립니다.** `black[jupyter]>=23.0` 뒤에 적힌 의존성과 `[project.optional-dependencies]`·`[dependency-groups]`·`[tool.pdm.dev-dependencies]`에 핀된 라이브러리는 `find-dependency-usage`가 보기에 그 라이브러리를 쓰지 않는 저장소와 똑같았습니다. 이번에도 판정이 아니라 **저장되는 내용**이 바뀌므로, 답이 달라지려면 **재색인이 필요합니다.**

## 수정

### `black[jupyter]>=23.0` 뒤의 의존성이 전부 사라졌습니다

- PEP 621 배열은 `dependencies\s*=\s*\[(.*?)\]`로 잘라 읽었습니다. **첫 `]`에서 끝나는** 패턴인데, 요구사항은 extra를 적을 때 대괄호를 스스로 가지고 있습니다.
- 그래서 `dependencies = ["black[jupyter]>=23.0", "requests==2.31.0", "urllib3>=2"]`는 `"black[jupyter`까지만 배열로 읽혔고, **그 뒤의 `requests`·`urllib3`는 통째로 없는 것**이 되었습니다.
- 이제 배열은 인용·중첩·주석을 추적하는 스캐너로 읽습니다. 요구사항 안의 대괄호는 요구사항의 일부로 남고, 배열은 자기 짝이 맞는 `]`에서 끝납니다.

### 읽히지 않던 배열이 세 종류 있었습니다

- `[project]`의 PEP 621 배열 하나만 읽었습니다. 나머지는 setuptools·hatch·PDM 프로젝트가 **개발·시험 도구를 두는 표준 자리**입니다 — `[project.optional-dependencies]`, PEP 735의 `[dependency-groups]`, `[tool.pdm.dev-dependencies]`.
- 거기에 라이브러리를 핀한 저장소는 그 라이브러리를 안 쓰는 저장소와 구별되지 않았습니다. 실제로 존재하는 유일한 핀이 아무도 읽지 않는 배열에 있었습니다.
- 이제 **섹션 이름이 말하는 대로 scope를 매깁니다** — extra는 `optional`, `test`라는 이름의 그룹은 `test`, 나머지 그룹은 `dev`. `[project]`에서는 `dependencies` 배열만 `direct`이고, `classifiers`나 `[build-system]`의 `requires` 같은 이웃 배열은 의존성이 아니므로 읽지 않습니다.
- 섹션 순회는 `tomlSections`로 뽑아 `parseTOMLDependencies`와 공유합니다.

### `urllib3!=1.25.0`의 버전이 `"!"`였습니다

- 요구사항 패턴의 연산자 목록에 `!=`와 `===`가 빠져 있었고, 둘 다 조용히 틀렸습니다.
- `urllib3!=1.25.0`은 버전 `"!"`로 기록됐습니다 — 무엇과도 비교할 수 없는 가짜 버전 그룹입니다. **제외는 프로젝트가 받지 않을 릴리스를 말할 뿐 실행 중인 릴리스를 말하지 않으므로**, 핀이 없는 요구사항과 같이 버전을 lock 파일에 맡깁니다.
- `pip===23.3.1`은 버전 없음으로 읽혔습니다. 임의 동등(arbitrary equality)은 정확히 한 릴리스를 고정하므로, **핀 그대로** 기록해 권고를 바로 판정할 수 있게 합니다.

## 검증

- 새 표 `TestPyProjectReadsEveryDependencyArray` — 배열 4종 9개 선언의 이름·버전·scope와, `[build-system] requires`·`classifiers`가 인벤토리로 새지 않는지 확인. 수정 전에는 실패함
- 새 표 `TestRequirementOperatorsThatWereMissing` — `!=`·`===`를 포함한 4개 형태와 `===` 핀의 `Comparable`. 수정 전에는 실패함
- `gofmt -l`, `go vet ./...`, FTS5 빌드·전체 테스트 통과, manifest·search·indexer는 `-race`도 통과
- 버전 메타데이터 정합성, Kubernetes Kustomize 렌더링, linux/amd64 Docker 이미지 빌드

## 업그레이드 참고

- 마이그레이션은 필요하지 않습니다.
- **재색인이 필요합니다.** 의존성 인벤토리는 색인할 때 만들어지므로, 이미 색인된 저장소의 저장된 선언은 다시 읽기 전까지 그대로입니다. `pyproject.toml`을 쓰는 저장소가 있다면 해당 ref를 다시 색인하세요.
- 재색인 뒤에는 **선언 수가 늘고, 버전 `!`인 가짜 항목이 사라집니다.** 그만큼 권고 조회에 걸리는 저장소도 늘어납니다.

## 오프라인 Docker 이미지

릴리스 자산은 아키텍처 접미사가 없는 다음 두 파일입니다.

- `git-ctx-v0.77.9.tar.gz`
- `git-ctx-v0.77.9.tar.gz.sha256`

```bash
sha256sum -c git-ctx-v0.77.9.tar.gz.sha256
gzip -dc git-ctx-v0.77.9.tar.gz | docker load
docker image inspect git-ctx:v0.77.9 --format '{{.Os}}/{{.Architecture}} {{.Config.User}}'
```

기대 결과는 `linux/amd64 10001`입니다. 아카이브에는 `git-ctx:v0.77.9`과 `git-ctx:0.77.9` 태그가 포함됩니다.

**전체 변경 내역**: https://github.com/hkjang/git-ctx/compare/v0.77.8...v0.77.9
