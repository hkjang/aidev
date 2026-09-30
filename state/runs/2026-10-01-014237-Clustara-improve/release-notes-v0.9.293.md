## notify scan 의 이벤트·리비전 조회가 실패해도, 창이 가득 차도 "이상 없음" 과 똑같이 보고했습니다

`POST /admin/k8s/notify/scan` 은 장애 상관분석에 쓰는 두 창을
`events, _ := ListK8sEvents(ctx, clusterID, 500)` 와
`revisions, _ := ListK8sRevisions(..., Limit: 1000)` 로 읽으면서 **두 오류를 모두 `_` 로 버리고**
결과가 상한에 찼는지도 보지 않았습니다. DB 조회가 깨져도, 창이 클러스터의 행을 담기에 작아도
`AnalyzeRCA` 는 비었거나 부분적인 입력으로 상관분석을 돌렸고, 응답은 정말 아무 문제 없는
클러스터의 것과 한 글자도 다르지 않았습니다(`sent: 0`, `evaluated_rca: 0`, `truncated: false`).
이 엔드포인트의 출력은 사람이 읽지 않으므로 증상은 "알림이 그냥 오지 않는다" 로만 나타났습니다.
이번 릴리즈는 저하된 스캔을 깨끗한 스캔과 구별해 보고합니다.

### notify scan · 이벤트·리비전 조회 실패와 창 포화를 보고합니다

v0.9.290 이 인벤토리 조회에 `truncated`·`truncation_notice` 를 붙였지만, 상관분석이 읽는 이 두
창은 그 계약 밖에 남아 있었습니다.

이제 두 조회의 오류를 받아 응답과 감사 로그에 `events_error`·`revisions_error` 로 적고, 실제로
분석에 들어간 개수 `events`·`revisions` 와 포화 여부 `events_truncated`·`revisions_truncated` 를
함께 보고합니다. 넷 중 하나라도 걸리면 인벤토리의 `truncation_notice` 와 같은 결의
`window_notice` 를 덧붙입니다. 기존 `truncated` 는 인벤토리 잘림이라는 의미를 그대로 유지합니다.

조회가 실패해도 스캔은 계속합니다. 인벤토리만 읽는 보안 분석은 여전히 유효하므로, 알림을 멈추는
대신 **저하된 실행을 깨끗한 실행으로 보고하지 않는** 쪽을 택했습니다.

### 포화 판정을 "상한+1 요청" 이 아니라 "창이 가득 찼다" 로 합니다

인벤토리가 쓰는 "상한보다 한 행 더 요청해 잘림을 감지" 트릭은 여기서는 쓸 수 없습니다.
스토어가 이 두 조회에 하드 상한을 겁니다 — `ListK8sEvents` 는 `boundedLimit(limit, 100, 500)`,
`ListK8sRevisions` 는 `boundedLimit(f.Limit, 100, 1000)` — 그래서 501 을 요청해도 조용히 500 으로
깎이고, 그 응답은 마침 500행이 들어찬 창과 구별되지 않습니다. 포화는 따라서 "요청한 상한과 같은
개수가 돌아왔다"(`len >= budget`)로 판정합니다. 스토어 상한과 다른 호출자의 요청 상한은 이번
변경에서 건드리지 않았습니다.

### 운영에 미치는 영향

**알림 동작은 바뀌지 않습니다.** 알림 개수·중복 제거 윈도우·조용한 시간·담당팀 채널 라우팅·
딥링크는 그대로이고, DB 스키마·설정 변경도 없습니다. 이번 변경은 **보고만** 추가합니다.

스캔 응답이나 감사 로그에 `window_notice` 가 보이면 그 스캔의 상관분석은 창 밖 이벤트·설정
변경을 반영하지 못한 것입니다 — 알림이 없다고 해서 이상이 없다는 뜻은 아닙니다. `cluster_id` 로
범위를 좁혀 다시 실행하고, `events_error`·`revisions_error` 가 있으면 수집기·DB 상태를
확인하십시오.

### 검증

신규 테스트 4개(실제 SQLite·`store.Open`+`Migrate`·`Server.Routes`·httptest webhook — 손으로
만든 대역 없음)를 수정 전 코드에 먼저 붙여 `TestNotifyScanReportsAFailedEventLookup` 과
`TestNotifyScanReportsASaturatedEventWindow` 가 실패하는 것을 확인했습니다. 조회 실패는 같은
SQLite 파일에 두 번째 연결로 붙어 `k8s_events` 테이블만 떨어뜨려 재현합니다 — 보안 분석이 읽는
인벤토리는 그대로 두는, 버려진 오류가 감추던 바로 그 부분 실패입니다.

`go build ./...`, `go vet ./...`, `go test ./... -count=1`(20개 패키지 전부 ok) 및 버전 일치·
산출물 이름 릴리즈 게이트를 통과했습니다. 실 Kubernetes·PostgreSQL·실 Mattermost·브라우저
렌더링은 미검증이며, 프로덕션 파일은 `internal/proxy/k8s_notify.go` 1개입니다.

### 배포 파일

| 파일 | 설명 |
|------|------|
| clustara-v0.9.293.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| clustara-v0.9.293.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.9.293.md | 오프라인 배포 가이드 |

### 빠른 시작

```bash
# 무결성 확인
sha256sum -c clustara-v0.9.293.tar.gz.sha256

# 이미지 로드
gunzip -c clustara-v0.9.293.tar.gz | docker load

# 실행
docker run -d --name clustara --restart=always \
  -p 9090:9090 \
  -v /opt/clustara/data:/data \
  -e UPSTREAM_BASE_URL=https://api.openai.com \
  -e UPSTREAM_API_KEY=sk-... \
  -e ADMIN_TOKEN=change-me \
  clustara:v0.9.293
```
