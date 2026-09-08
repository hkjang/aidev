# git-ctx v0.77.10

이번 릴리스는 **프로젝트가 실제로 쓰는 이름의 pip 요구사항 파일을 인벤토리가 읽지 않던 문제**를 고칩니다. 인식 대상이 `requirements.txt`·`requirements-dev.txt` 두 이름의 정확 일치였기 때문에, 파이썬 프로젝트가 보통 갖는 형태가 통째로 빠져 있었습니다. 기본 서비스 포트는 계속 `4747`이며 기존 API·MCP 호환성을 유지합니다.

> **권고 대응 중이라면 이 릴리스가 통보 대상 목록을 늘립니다.** pip-tools 쌍, 접두사·접미사로 갈라 쓴 이름, `requirements/` 디렉터리 레이아웃에 핀을 둔 저장소는 `find-dependency-usage`가 보기에 그 라이브러리를 쓰지 않는 저장소와 똑같았습니다. 이번에도 판정이 아니라 **저장되는 내용**이 바뀌므로, 답이 달라지려면 **재색인이 필요합니다.**

## 수정

### `requirements.in`도 `requirements/base.txt`도 인벤토리에 없었습니다

- `Recognize`는 `requirements.txt`와 `requirements-dev.txt`를 **정확히 그 이름일 때만** pip 요구사항 파일로 봤습니다.
- 그래서 파이썬 프로젝트가 보통 갖는 다음 형태가 전부 읽히지 않았습니다 — pip-tools 쌍(`requirements.in`을 컴파일해 `requirements.txt`를 만드는 형태), 한 벌을 갈라 쓴 이름(`requirements_test.txt`, `dev-requirements.txt`), 그리고 대다수 Django 프로젝트가 출발점으로 삼는 환경별 한 파일 레이아웃(`requirements/base.txt`, `requirements/prod.in`).
- 핀이 한 번도 읽히지 않은 저장소는 **그 라이브러리를 쓰지 않는 저장소와 구별되지 않습니다.** 그런 저장소를 포함한 권고 조회는 무사 통보처럼 읽혔습니다.
- 이제 이름 규칙으로 인식합니다 — `requirements` 접두사·접미사(`-`·`_`·`.` 구분), 확장자 `.txt`·`.in`, 그리고 `requirements/` 디렉터리 아래의 `.txt`·`.in` 파일.

### 파일 이름이 말하는 scope를 버리고 전부 `direct`로 적었습니다

- 요구사항 파일에는 `// indirect` 같은 표시가 없어서, **한 벌이 무엇에 쓰이는지 말하는 것은 파일 이름뿐**입니다.
- 그런데 모든 선언이 `direct`로 들어갔고, 저장소가 시험에만 쓰는 도구와 서비스가 실제로 싣는 의존성이 같은 자리에 섞였습니다.
- 이제 이름이 `dev`·`develop`·`development`·`local`을 말하면 `dev`, `test`·`tests`·`testing`을 말하면 `test`로 기록합니다. 그 밖은 그대로 `direct`입니다.

### `This directory contains`가 패키지 "This" 버전 "directory"였습니다

- 요구사항이 사는 자리의 `.txt`를 읽게 되면 README 같은 산문도 파서에 닿습니다.
- 요구사항 패턴은 연산자를 선택으로 두고 버전 그룹은 따로 받았기 때문에, **이름 뒤에 맨 단어가 오면 그것을 버전으로 읽었습니다** — 수정 전 실제 출력에서 `This directory contains`는 패키지 `This`의 버전 `directory`가 되었습니다.
- 어떤 요구사항도 연산자 없이 버전을 적지 않습니다. 그런 줄은 이제 건너뜁니다.

## 검증

- 새 표 `TestRequirementsFilesTheWayProjectsNameThem` — 인식해야 하는 이름 10건과 인식하면 안 되는 이름 4건. 수정 전에는 실패함
- 새 표 `TestRequirementsScopeComesFromTheFileName` — 이름에서 읽어야 하는 scope 7건. 수정 전에는 실패함
- 새 시험 `TestProseIsNotARequirement` — 산문 한 줄이 선언으로 새지 않는지 확인. 수정 전에는 실패함
- `gofmt -l`, `go vet ./...`, FTS5 빌드·전체 테스트 통과, manifest·indexer·search는 `-race`도 통과
- 버전 메타데이터 정합성, Kubernetes Kustomize 렌더링, linux/amd64 Docker 이미지 빌드

## 업그레이드 참고

- 마이그레이션은 필요하지 않습니다.
- **재색인이 필요합니다.** 의존성 인벤토리는 색인할 때 만들어지므로, 이미 색인된 저장소의 저장된 선언은 다시 읽기 전까지 그대로입니다. 요구사항 파일을 `requirements.txt` 이외의 이름으로 두는 저장소가 있다면 해당 ref를 다시 색인하세요.
- 재색인 뒤에는 **선언 수가 늘고, `dev`·`test` scope로 갈리는 선언이 생깁니다.** 그만큼 권고 조회에 걸리는 저장소도 늘어납니다.
- `docs/configuration.md`의 지원 매니페스트 설명에도 읽는 이름 규칙을 적었습니다.

## 오프라인 Docker 이미지

릴리스 자산은 아키텍처 접미사가 없는 다음 두 파일입니다.

- `git-ctx-v0.77.10.tar.gz`
- `git-ctx-v0.77.10.tar.gz.sha256`

```bash
sha256sum -c git-ctx-v0.77.10.tar.gz.sha256
gzip -dc git-ctx-v0.77.10.tar.gz | docker load
docker image inspect git-ctx:v0.77.10 --format '{{.Os}}/{{.Architecture}} {{.Config.User}}'
```

기대 결과는 `linux/amd64 10001`입니다. 아카이브에는 `git-ctx:v0.77.10`과 `git-ctx:0.77.10` 태그가 포함됩니다.

**전체 변경 내역**: https://github.com/hkjang/git-ctx/compare/v0.77.9...v0.77.10
