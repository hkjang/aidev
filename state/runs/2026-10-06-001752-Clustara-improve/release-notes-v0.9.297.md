## 클러스터 목록을 읽지 못한 notify scan 이, 등록된 클러스터가 없는 설치와 한 글자도 다르지 않게 응답했습니다

`POST /admin/k8s/notify/scan` 을 `cluster_id` 없이 돌리는 것이 함대 전체를 커버하는 사용법이고,
`docs/ADMIN_GUIDE.md` 도 그 한 줄을 cron 에 걸어 한 작업으로 모든 클러스터를 덮으라고 권합니다.
그 실행은 대상 클러스터 목록을 `ListK8sClusters` 로 정합니다.

그런데 `notifyScanTargets` 는 그 조회가 실패하면 `[]string{""}` 을 돌려주고 **오류를 버렸습니다**.
그것은 등록된 클러스터가 하나도 없는 정상 설치가 쓰는 단일 공유 창 폴백과 **바이트 단위로 같은**
대상 목록입니다. 그래서 응답은 `clusters_scanned: 1` 에 오류 키 하나 없이, 건강한 미등록 설치의
응답과 한 글자도 다르지 않았습니다.

이 엔드포인트의 출력은 사람이 읽지 않습니다. cron 이 돌리고, 사람은 알림이 오는지로만 결과를
봅니다. 그래서 DB 가 고장 나 함대를 열거하지 못한 실행이 v0.9.295 가 없앤 동작 — 등록된 모든
클러스터가 **조회 창 하나를 나눠 써** 조용한 클러스터의 privileged 워크로드는 평가 자체가 되지
않는 상태 — 로 조용히 퇴행해도, 운영자가 볼 수 있는 증상은 아무것도 없었습니다. 이번 릴리즈는
열거에 실패한 함대를 존재하지 않는 함대와 구별해 보고합니다.

### `POST /admin/k8s/notify/scan` · 레지스트리 조회 실패를 `clusters_error` 로 보고합니다

`notifyScanTargets` 의 반환을 `([]string, error)` 로 바꿔, 폴백 대상 목록과 **함께** 오류를
호출자(`handleK8sNotifyScan` 한 곳)에게 올립니다. 응답과 감사 로그에 두 키가 실립니다.

- `clusters_error` — `ListK8sClusters` 가 돌려준 오류 문자열. 기존 `events_error`·`revisions_error`
  와 같은 결이고, 같은 자리에 실립니다.
- `clusters_notice` — "클러스터 목록을 읽지 못해 전 클러스터 스캔이 단일 공유 창으로
  퇴행했습니다. 클러스터마다 예산을 따로 쓰지 못했으므로 조용한 클러스터는 평가되지 않았을 수
  있습니다. `cluster_id` 로 클러스터별로 다시 실행하고 DB 상태를 확인하세요." 라는 운영자 안내.

`clusters_notice` 는 두 상관 창(이벤트·리비전)의 포화를 말하는 기존 `window_notice` 와 **별도 키**
입니다. 한 문장에 섞으면 그 문구를 고정한 계약이 흔들리고, 두 가지는 운영자가 해야 할 일도 서로
다릅니다.

### 요청을 깨지 않습니다

오류는 **500 으로 올리지 않습니다.** 요청을 깨면 cron 스캔이 아예 돌지 않아 알림이 전부 끊기고,
그것은 지금 고치려는 결함보다 나쁩니다. 스캔은 전과 같이 단일 공유 창 폴백으로 끝까지 돌되,
그 실행이 퇴행한 실행이라는 사실을 응답에 적습니다.

폴백 동작 자체와 주석의 계약은 그대로입니다. 읽기는 성공했지만 **비어 있는** 레지스트리는 두 키
없이 지금까지와 완전히 같게 응답하므로, 등록 전 설치나 `events`·`clusters_scanned` 만 읽던 호출자는
영향이 없습니다. `clusters_scanned`·`clusters_truncated` 의 의미, `cluster_id` 를 지정한 실행 경로,
인벤토리 조회 실패의 500, 분석 호출(`AnalyzeRCA`→`EnrichWithConfigChanges`→`AnalyzeSecurity`)·중복
제거 키·담당팀 라우팅·딥링크·조용한 시간 판정, DB 스키마·설정은 모두 그대로입니다.

### 검증

신규 종단 테스트 2개(`internal/proxy/k8s_notify_registry_test.go`)를 수정 전 코드에 먼저 붙여
빨강을 확인했습니다. 실제 SQLite + `store.Open`/`Migrate` + `store.NewAsyncLogger` + `NewServer` +
`httptest.NewServer(server.Routes())` + httptest Mattermost webhook + 실제
`kube.InventoryFromObject` privileged Pod 로 돌리며, 손으로 만든 대역이나 주입 객체는 없습니다.
조회 실패는 두 번째 커넥션에서 `DROP TABLE k8s_clusters` 로 재현하고, 비어 있는 레지스트리가 두 키
없이 응답하는 것을 같은 파일에서 함께 봅니다. 감사 로그가 핸들러 끝의 기존 `window` 병합으로 두
키를 그대로 받는 것도 테스트로 확인했습니다.

`gofmt -l`(해당 파일 출력 없음) · `go build ./...` · `go vet ./...` · `go test ./... -count=1`(20개
패키지 전부 통과) 과 버전 일치·산출물 이름 릴리즈 게이트를 실행했습니다. 실 Kubernetes·
PostgreSQL·브라우저 렌더링은 미검증이며, 프로덕션 파일은 `internal/proxy/k8s_notify.go` 1개입니다.
DB 스키마·설정 변경은 없으므로 이미지만 교체하면 되고, 롤백도 이전 이미지로 되돌리는 것으로
끝납니다.

### 오프라인 배포 패키지

- 이미지: `clustara:v0.9.297` (linux/amd64)
- 파일: `clustara-v0.9.297.tar.gz` · `clustara-v0.9.297.tar.gz.sha256` ·
  `README-offline-v0.9.297.md`

```bash
sha256sum -c clustara-v0.9.297.tar.gz.sha256
gunzip -c clustara-v0.9.297.tar.gz | docker load
```
