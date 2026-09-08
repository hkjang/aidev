## `r"m" -rf /` 라고 적힌 루트 삭제가 모든 게이트를 통과하고 실행됐습니다

하나의 명령 문자열을 두 파서가 읽습니다 — **게이트**(Command Risk Parser·터미널 denylist·access mode 분류기)와 Kubernetes exec argv 를 조립하는 **executor** 입니다. 게이트는 원본 바이트를 보면서 토큰의 *바깥쪽 따옴표만* 떼어냈고, executor 는 따옴표·백슬래시를 제대로 해석해 실제 프로그램을 실행했습니다. 그래서 셸에서 흔한 표기 하나로 프로그램 이름이 모든 게이트에서 동시에 가려졌습니다. 이번 릴리즈는 그 불일치 다섯 지점을 고칩니다.

### ① `critical` 이어야 할 명령이 승인조차 필요 없는 `low` 로 채점됐습니다

`r"m" -rf /`, `\rm -rf /`, `re"boot"`, `shut'down' -h now`, `mkfs".ext4" /dev/sda` 가 전부 `critical` 이 아니라 `low` 로 채점됐습니다. `critical` 은 정책을 한 줄도 읽기 전에 걸리는 하드 블록(실행 핸들러에서 한 번 더)이고 `low` 는 승인이 필요 없는 read_only 티어입니다. 즉 첫 토큰을 받아주는 allowlist 하나만 있으면 `evaluateTerminalPolicy` 가 루트 삭제에 대해 `Allowed=true, RequireApproval=false, access_mode=read_only` 를 돌려줬습니다. 신규 종단 테스트가 이 응답을 그대로 재현합니다.

### ② denylist 의 `"rm -rf"` 는 원본 문자열 substring 검사였습니다

`r"m" -rf /data` 에는 그 부분문자열이 없으므로 차단 목록도 같은 표기를 놓쳤습니다. 이제 **deny 쪽은 executor 가 해석한 형태로도 대조**합니다. allow 쪽은 종전대로 원본 문자열만 읽습니다 — 정규화하면 allowlist 를 더 쉽게 만족시키는 반대 방향이 되기 때문입니다.

### ③ 항상 승인을 강제하는 full TTY 티어를 `"bash"` 가 빠져나갔습니다

`isInteractiveShell` 이 원본 `Fields` 를 써서 `"bash"`·`\bash`·`'sh'` 를 인터랙티브 셸로 보지 못하고 read_only 로 분류했습니다. 이제 같은 분해기를 쓰므로 표기와 무관하게 전체 TTY 로 분류되어 승인을 거칩니다.

### ④ executor 의 분해기가 승인된 텍스트와 두 곳에서 어긋났습니다

명시적으로 빈 인자(`sh -c "" ls`)를 버려 뒤 인자가 한 칸씩 당겨졌고 — 실제로 실행된 것은 `sh -c ls` 입니다 — 작은따옴표 안의 백슬래시를 이스케이프로 처리해 `grep 'a\.b' f` 가 `a.b` 를 찾았습니다(셸은 문자 그대로 둡니다).

### ⑤ `podExecArgs` 가 호출자 argv 의 빈 요소를 버리고 각 요소를 trim 했습니다

`sh -c <script> <argv0> <path> <query> <n>` 처럼 위치 파라미터를 넘기는 호출(증적 검색 경로)에서 인자가 밀렸습니다. 이제 호출자가 준 argv 를 그대로 전달합니다.

---

따옴표 해석은 이제 `analyzer.ShellWords` 한 곳에 POSIX 규칙으로 문서화되어 있고, executor 의 분해기가 같은 규칙을 따릅니다.

**검증**: 신규 테스트 8개(analyzer 5 + kube 2 + proxy 3)를 고치기 전 코드에 되돌려 붙여 각 결함을 지목하며 실패하는 것을 확인했습니다(analyzer 16건·kube 5건·proxy 8건 실패). 정상 조회가 계속 `low`·미차단으로 남는지 지키는 오탐 회귀와, allow 쪽이 넓어지지 않았는지 지키는 테스트를 포함합니다. `go build ./...`, `go vet ./...`, `go test ./...` 전부 통과.

### 배포 파일

| 파일 | 설명 |
|------|------|
| clustara-v0.9.275.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| clustara-v0.9.275.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.9.275.md | 오프라인 배포 가이드 |

### 빠른 시작

```bash
# 이미지 로드
gunzip -c clustara-v0.9.275.tar.gz | docker load

# 실행
docker run -d --name clustara --restart=always \
  -p 9090:9090 \
  -v /opt/clustara/data:/data \
  -e UPSTREAM_BASE_URL=https://api.openai.com \
  -e UPSTREAM_API_KEY=sk-... \
  -e ADMIN_TOKEN=change-me \
  clustara:v0.9.275
```
