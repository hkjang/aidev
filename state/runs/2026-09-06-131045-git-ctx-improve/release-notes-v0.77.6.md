# git-ctx v0.77.6

이번 릴리스는 **이름이 부분만 겹치는 이웃 패키지 때문에 저장소가 "영향"으로 보고되던 문제**를 고칩니다. 인벤토리 질의는 이름을 일부러 부분 일치로 찾는데, 권고 판정이 그렇게 걸려 온 선언까지 질의한 패키지의 버전인 양 읽고 있었습니다. 기본 서비스 포트는 계속 `4747`이며 기존 API·MCP 호환성을 유지합니다.

> **권고 대응 중이라면 이 릴리스가 통보 대상 목록을 바꿉니다.** "영향"으로 지목돼 CODEOWNERS 소유자까지 조회됐던 저장소 일부는 그 라이브러리를 쓰지 않습니다. 저장된 내용이 아니라 판정 방식이 바뀐 것이라 재색인 없이 답이 달라집니다.

## 수정

### `requests` 권고가 `requests-toolbelt`을 쓰는 저장소를 지목했습니다

- 인벤토리 질의는 이름을 **부분 일치**로 찾습니다. 운영자가 손에 든 것은 권고가 부르는 라이브러리 통칭이지 매니페스트가 적는 좌표가 아니라서, `log4j`가 `org.apache.logging.log4j:log4j-core`를 찾아 주어야 하기 때문입니다.
- 그런데 같은 규칙이 `requests`에 **`requests-toolbelt`을 물어 왔고**, 수정 버전 판정은 걸려 온 모든 선언의 버전을 질의한 패키지의 버전으로 읽었습니다.
- 그래서 requests 2.31.0 권고에 `requests-toolbelt 0.10.1`을 쓰는 저장소가 **"영향"으로 보고되고**, 그 저장소의 소유자가 CODEOWNERS에서 조회되어 통보 대상에 올랐습니다. 쓰지도 않는 라이브러리의 취약점 때문입니다.

### 판정은 질의한 패키지의 선언만 봅니다

- 이름 **전체가 같거나**, 이름의 **온전한 한 조각**일 때만 같은 패키지로 봅니다 — maven group·artifact, 모듈 경로나 scoped 패키지의 경로 요소, group의 마지막 점 구분 조각.
- 좌표는 그대로 걸립니다 — `log4j` → `org.apache.logging.log4j:log4j-core`, `gin-gonic/gin` → `github.com/gin-gonic/gin`.
- **접두사나 하이픈 한 단어만 겹치는** 이름은 판정에서 빠집니다 — `requests` 대 `requests-toolbelt`.
- 제외한 선언은 응답에서 사라지지 않습니다. **Declarations에는 그대로 남고**, 어떤 패키지를 왜 뺐는지 notes에 이름과 함께 적습니다(이름이 많으면 5개까지, 나머지는 개수로).
- **권고 없는 질의는 종전 그대로** 부분 일치 전체를 묶어 보여 줍니다. 거기서 넓은 매칭은 주장이 아니라 탐색을 돕는 장치이기 때문입니다.

### limit을 넘는 저장소가 판정에서 말없이 빠졌습니다

- 판정용 버전 그룹을 **limit으로 잘린 목록이 아니라 조회한 선언 전체**에서 만듭니다.
- 락파일 우선 재그룹이 잘린 목록을 읽고 있어서, 상위 N건 밖의 저장소는 아무 말 없이 판정 대상에서 빠질 수 있었습니다.

## 검증

- 새 표 `TestAdvisoryJudgesOnlyTheQueriedPackage` — 이웃 패키지가 "영향"을 만들지 않고, Declarations에는 남으며, notes가 제외한 이름을 말하는지
- 새 표 `TestAdvisoryStillReachesTheCoordinate` — `log4j`·`gin-gonic/gin` 같은 통칭이 여전히 좌표에 닿는지
- `gofmt -l`, `go vet ./...`, FTS5 빌드·전체 테스트 통과, search·mcp는 `-race`도 통과
- FTS5 빌드·태그 없는 빌드·PostgreSQL 3가지 조합 전체 단위·통합·race 테스트, 빌드 모드 교차 시험, 릴리스 데이터베이스 업그레이드 시험, 콘솔 시험

## 업그레이드 참고

- 마이그레이션은 필요하지 않습니다.
- **재색인은 필요하지 않습니다.** 인벤토리에 저장된 선언은 그대로이고, 그 선언을 판정에 쓰는 범위만 달라집니다.
- 이전에 "영향"으로 돌아오던 저장소 일부가 이제 목록에서 빠집니다. 그것이 이번 수정의 목적입니다.

## 오프라인 Docker 이미지

릴리스 자산은 아키텍처 접미사가 없는 다음 두 파일입니다.

- `git-ctx-v0.77.6.tar.gz`
- `git-ctx-v0.77.6.tar.gz.sha256`

```bash
sha256sum -c git-ctx-v0.77.6.tar.gz.sha256
gzip -dc git-ctx-v0.77.6.tar.gz | docker load
docker image inspect git-ctx:v0.77.6 --format '{{.Os}}/{{.Architecture}} {{.Config.User}}'
```

기대 결과는 `linux/amd64 10001`입니다. 아카이브에는 `git-ctx:v0.77.6`과 `git-ctx:0.77.6` 태그가 포함됩니다.

**전체 변경 내역**: https://github.com/hkjang/git-ctx/compare/v0.77.5...v0.77.6
