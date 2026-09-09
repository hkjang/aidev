## 레지스트리 포트가 태그로 읽혀, 태그 없는 이미지가 "고정됨" 으로 통과했습니다

하나의 이미지 참조를 네 곳이 읽습니다 — SEC-02 포스처(이미지 태그 정책), SEC-10 정책 게이트, Dockerfile 빌드 게이트, 이미지 사용 인벤토리·원장. 그런데 올바른 파서를 쓰는 곳은 **정책 게이트 하나뿐**이었습니다. 이번 릴리즈는 그 파싱 결함 다섯 지점을 고칩니다. **있는 위험이 포스처 리포트에서 사라졌고, 없는 위험은 정상 구성에 붙었습니다.**

### ① `registry.corp.local:5000/team/app` 이 태그가 없는데도 고정된 것으로 통과했습니다

`imageTagAndDigest` 는 **마지막 `/` 뒤에 오는 `:` 만** 태그로 읽습니다. 그런데 SEC-02 `imageFindings` 와 Dockerfile 게이트는 문자열에 `:` 가 있는지만 보는 substring 검사였습니다. 그래서 `registry.corp.local:5000/team/app` — 태그가 없으므로 풀 시점에 `:latest` 가 되는 참조 — 가 **레지스트리 포트의 콜론 덕분에** 고정된 것으로 통과했습니다.

포트를 명시한 사설 레지스트리는 이 제품이 겨냥하는 폐쇄망의 기본형입니다. 그 결과 같은 이미지를 두고 포스처 리포트는 "정상", 정책 게이트는 "위반" 이라고 서로 다르게 판정하고 있었습니다.

이제 네 곳 모두 정책 게이트와 같은 파서를 씁니다. 같은 레지스트리 포트 뒤에 있는 태그 있는 참조와 digest 고정 참조는 종전대로 깨끗하게 남습니다.

### ② `FROM --platform=linux/amd64 base:1.0` 에서 진짜 base 가 검사되지 않았습니다

Dockerfile 게이트는 `FROM [--flag...] <ref> [AS <stage>]` 에서 `Fields()[1]` 을 이미지 참조로 썼습니다. 그래서 플래그가 붙으면 **"--platform=linux/amd64" 라는 이름의 이미지가 태그 미고정** 이라고 보고되고, 정작 검사해야 할 `base:1.0` 은 아예 검사되지 않았습니다.

같은 이유로 반대 방향의 오탐도 있었습니다 — 앞선 stage 를 가리키는 `FROM build` 는 고정할 태그가 존재하지 않는데도 mutable base 로 잡혔습니다. 이제 플래그를 문법대로 건너뛰고, stage 이름은 base 이미지로 보지 않습니다.

### ③ 고정되지 않은 debug 이미지를 돌리는 Pod 가 "고정된 이미지만 쓴다" 로 표시됐습니다

이미지 사용 인벤토리가 `ephemeralContainers` 를 건너뛰었습니다. `ExtractImages` 는 정책 룰이 debug 이미지를 봐야 한다고 이미 주석으로 명시하고 있었는데, 인벤토리 쪽만 빠져 있었습니다. 이제 포함합니다.

### ④ 미러 레지스트리를 두면 멀쩡한 두 이미지가 "태그가 옮겨졌다" 로 보고됐습니다

원장의 tag drift 키가 registry 를 빼고 repository 만 썼습니다. 그래서 `harbor.corp/app:1.0` 과 `docker.io/app:1.0` 이 **한 항목으로 합쳐지고**, 서로 다른 두 digest 가 "같은 태그가 다른 digest 를 가리킨다" 로 잡혔습니다. 업스트림 옆에 미러 레지스트리를 두는 것은 정확히 이 제품이 보는 폐쇄망 구성입니다.

이제 registry 를 포함한 참조로 묶습니다. 겸사겸사 init 컨테이너 이미지도 spec·status 양쪽에서 원장에 담습니다.

### 검증

신규 테스트 4개를 **고치기 전 코드에 되돌려 붙여** 네 개가 각 결함을 지목하며 실패하는 것을 확인했습니다. 여기에는 같은 레지스트리 포트 뒤의 태그 있는·digest 고정 참조가 계속 깨끗한지 지키는 오탐 회귀와, 포스처와 정책 게이트의 판정이 일치하는지 대조하는 검사가 포함됩니다. `go build ./...`, `go vet ./...`, `go test ./...` 전부 통과(20 패키지).

### 배포 파일

| 파일 | 설명 |
|------|------|
| clustara-v0.9.279.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| clustara-v0.9.279.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.9.279.md | 오프라인 배포 가이드 |

### 빠른 시작

```bash
# 무결성 확인
sha256sum -c clustara-v0.9.279.tar.gz.sha256

# 이미지 로드
gunzip -c clustara-v0.9.279.tar.gz | docker load

# 실행
docker run -d --name clustara --restart=always \
  -p 9090:9090 \
  -v /opt/clustara/data:/data \
  -e UPSTREAM_BASE_URL=https://api.openai.com \
  -e UPSTREAM_API_KEY=sk-... \
  -e ADMIN_TOKEN=change-me \
  clustara:v0.9.279
```
