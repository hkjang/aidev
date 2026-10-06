## 조용한 시간에 억제된 notify scan 이, 아무 문제도 없는 스캔과 한 글자도 다르지 않게 응답했습니다

`POST /admin/k8s/notify/scan` 을 `cluster_id` 없이 돌리는 것이 함대 전체를 커버하는 사용법이고,
`docs/ADMIN_GUIDE.md` 도 그 한 줄을 cron 에 걸어 한 작업으로 모든 클러스터를 덮으라고 권합니다.
그 cron 은 대개 **야간** 에 돕니다 — 그리고 야간은 운영자가 설정한 조용한 시간
(`k8s_quiet_hours`) 창이 열려 있는 시간입니다.

스캔이 그 창에 걸리면 알림을 보내지 않고 `{"sent": 0, "suppressed": "quiet_hours", ...}` 로
답합니다. 그런데 그 조기 반환은 진단 키를 담는 `window` 맵이 **만들어지기 전에** 일어났습니다.
바로 위의 팬아웃 루프는 이미 클러스터 목록·인벤토리·이벤트·리비전 조회를 전부 실행해 **대가를
치른 뒤** 였으므로, 거기서 나온 `clusters_error`·`events_error`·`revisions_error`·`truncated`·
`clusters_truncated` 는 계산되자마자 한 글자도 나가지 못하고 버려졌습니다.

그래서 **클러스터 레지스트리를 읽지 못한 설치** 의 조용한 시간 스캔이 완전히 건강한 설치의
스캔과 **바이트 단위로 같은** 응답을 냈습니다:

```json
{"sent": 0, "suppressed": "quiet_hours", "quiet_hours": "1-4", "timezone": "Asia/Seoul"}
```

이 엔드포인트의 출력은 사람이 읽지 않습니다. cron 이 돌리고, 사람은 알림이 오는지로만 결과를
봅니다. 조용한 시간에는 알림이 오지 않는 것이 정상이므로, 운영자가 볼 수 있는 증상은 **아무것도
없었습니다.** v0.9.293(이벤트·리비전 창)·v0.9.295(`clusters_truncated`)·v0.9.297
(`clusters_error`) 세 회차가 쌓아 올린 진단이, 하필 그 진단이 가장 필요한 시간대에서 통째로
사라져 있었습니다.

### notify scan · 억제된 응답도 조회 진단을 그대로 싣습니다

`window` 맵 조립 블록을 quiet 판정 **위로 그대로 옮기고**, quiet 분기의 응답에 `clock` 과 같은
방식으로 병합합니다. `truncated` 와 `truncation_notice` 도 함께 실립니다. 옮긴 것은 **순수한 맵
조립뿐** 이며 조회는 한 줄도 움직이지 않았으므로, I/O 의 순서와 횟수는 이전과 같습니다.

억제된 응답이 이제 담는 것:

| 키 | 의미 |
|------|------|
| `clusters_error` · `clusters_notice` | 클러스터 목록 조회가 실패해 전 클러스터 스캔이 단일 공유 창으로 퇴행했습니다 |
| `events_error` · `revisions_error` · `window_notice` | 장애 상관분석에 쓰는 창이 실패했거나 상한에 찼습니다 |
| `truncated` · `clusters_truncated` · `truncation_notice` | 검사 대상이 상한을 넘어 일부만 점검했습니다 |

`sent: 0`·`suppressed`·`quiet_hours`·`timezone` 의 의미와 키 이름은 **그대로** 입니다. 조회가
전부 정상이면 `*_error`·`*_notice` 키는 **붙지 않으므로**, 건강한 설치의 조용한 시간 응답은
`truncated: false` 가 더해지는 것 외에 지금까지와 같습니다.

### 같은 고장은 몇 시에 읽든 같은 문구로 말합니다

세 안내 문구를 패키지 상수(`notifyClustersNotice`·`notifyScanWindowNotice`·
`notifyScanTruncationNotice`)로 뽑았습니다. 같은 스캔이 **억제되거나 끝까지 실행되거나** 두 경로
중 하나로 끝나는데, 03시에 읽은 운영자와 10시에 읽은 운영자가 하나의 고장을 서로 다른 두 이야기로
읽으면 안 되기 때문입니다. 문구 값 자체는 이전 릴리즈와 동일합니다.

### 그대로인 것

조용한 시간 판정 규칙, 타임존 해석(`k8s_notify_timezone`), 알림 전송 경로, 중복 제거 키,
담당팀 라우팅, `cluster_id` 지정 실행 경로, DB 스키마와 설정은 모두 바뀌지 않았습니다.

**호환성 참고**: 억제 응답의 키가 늘어나므로, 이 JSON 을 소비하는 스크립트는 알려지지 않은 키를
무시해야 합니다.

### 검증

신규 종단 테스트 2개(`internal/proxy/k8s_notify_quiet_window_test.go`)를 수정 **전** 코드에 먼저
붙여 빨강을 확인했습니다. 표준 배선만 씁니다 — 실제 SQLite + `store.Open`/`Migrate` +
`NewServer` + `httptest.NewServer(server.Routes())` + 실제 `kube.InventoryFromObject` privileged
Pod, 손으로 만든 대역·주입 객체는 없습니다. 조회 실패는 같은 dsn 의 두 번째 커넥션에서
`DROP TABLE k8s_clusters` 로 재현하고, 정상 조회의 조용한 시간 응답이 진단을 붙이지 않는 것도
함께 봅니다. 기존 quiet 테스트 3곳은 무수정 통과합니다.

`gofmt -l`(변경 파일 출력 없음), `go build ./...`, `go vet ./...`, `go test ./... -count=1`
(20개 패키지 전부 ok, 실패 0) 및 버전 일치·산출물 이름 릴리즈 게이트를 통과했습니다. 실
Kubernetes·PostgreSQL·브라우저 렌더링은 미검증이며, 프로덕션 파일은 `internal/proxy/k8s_notify.go`
1개입니다.

### 배포 파일

| 파일 | 설명 |
|------|------|
| clustara-v0.9.298.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| clustara-v0.9.298.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.9.298.md | 오프라인 배포 가이드 |

### 빠른 시작

```bash
# 이미지 로드
gunzip -c clustara-v0.9.298.tar.gz | docker load

# 실행
docker run -d --name clustara --restart=always \
  -p 9090:9090 \
  -v /opt/clustara/data:/data \
  -e UPSTREAM_BASE_URL=https://api.openai.com \
  -e UPSTREAM_API_KEY=sk-... \
  -e ADMIN_TOKEN=change-me \
  clustara:v0.9.298
```
