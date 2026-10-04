## 이벤트 조회가 창에 잘려 나갔는데도, 전부 보여준 응답과 한 글자도 다르지 않았습니다

`GET /admin/k8s/events` 는 `{"events": [...]}` 하나만 돌려주었습니다. 그런데 스토어는 `limit` 을
기본 100·최대 500 으로 **조용히 조정**하고(`boundedLimit`), 응답에는 실제로 적용된 상한도 그
상한에 닿았는지도 들어 있지 않았습니다. `limit=1000` 을 보낸 호출자는 1000행을 요청했다고 믿은
채 500행을 받았고, 받은 배열이 그 클러스터의 이벤트 전부인지 `last_seen` 내림차순으로 잘린 머리
부분인지 알 방법이 없었습니다.

장애 조사 중에 이것은 조용한 실패입니다. 원인이 된 더 오래된 이벤트가 창 밖으로 밀려나도 응답은
정말 이벤트가 그만큼뿐인 클러스터의 것과 똑같이 보였으므로, 운영자가 볼 수 있는 증상은 "그 이벤트가
없다" 뿐이었습니다. 이번 릴리즈는 잘려 나간 조회를 온전한 조회와 구별해 보고합니다.

### `/admin/k8s/events` · 실효 조회 상한과 창 포화를 보고합니다

핸들러가 조회 **전에** `store.K8sEventQueryLimit` 으로 실효 상한을 확정해 그 값으로 조회하고,
응답에 두 필드를 **항상** 싣습니다.

- `limit` — 이번 조회에 실제로 적용한 정수 상한.
- `window_full` — 반환된 이벤트 수가 그 상한 이상인지(`len(events) >= limit`).

`window_full` 이 true 일 때만 `window_notice` 로 "조회 상한에 도달했으며 더 오래된 이벤트가 있을
수 있습니다." 를 덧붙이고, 상한 미만이면 `window_full: false` 에 `window_notice` 는 생략됩니다.

상한 조정 규칙은 바뀌지 않았습니다 — 생략·빈 값·공백·정수로 해석할 수 없는 값·0·음수는 기본
100, 1~500 은 그대로, 500 초과는 500 으로 제한됩니다. 스토어가 쓰던 `boundedLimit(limit, 100, 500)`
호출을 같은 값을 돌려주는 신규 공개 헬퍼 `K8sEventQueryLimit` 로 바꿔, 핸들러와 스토어가 **한
군데서** 상한을 정하게 했습니다.

`window_full` 은 조회 창의 포화 표시이며 이벤트 누락 확정이나 정확한 전체 건수·`has_more` 가
아닙니다 — 저장된 행이 정확히 500개여도 501개여도 `limit=500` 에서는 true 입니다. 기존 `events`
키는 `last_seen` 내림차순·빈 결과 `[]` 그대로이므로 `events` 만 읽던 호출자는 영향이 없습니다.
인증(401 `authentication_required`)·메서드(405 `method_not_allowed`)·DB 실패(500
`k8s_events_failed`) 응답과 `cluster_id` 필터의 의미, DB 스키마·설정은 모두 그대로입니다.

### 검증

신규 종단 테스트 2개(`TestK8sEventsReportsEffectiveWindow`·`TestK8sEventsPreservesErrorResponses`)는
실제 SQLite·`store.NewAsyncLogger`·`NewServer`·`httptest.NewServer(server.Routes())` 로 돌며 손으로
만든 대역이 없습니다. 501행 클러스터·정확히 500행 클러스터·상한 미만 클러스터를 한 DB 에 함께
넣어, 다른 클러스터의 더 최신 이벤트가 대상 클러스터의 창 예산을 쓰지 않는 것, `limit=1000` 이
500 으로 보고되는 것, 포화일 때만 `window_notice` 가 비어 있지 않게 실리는 것을 봅니다.

`go build ./...`, `go vet ./...`, `go test ./...` 와 버전 일치·산출물 이름 릴리즈 게이트를 통과
했습니다. 실 Kubernetes·PostgreSQL·브라우저 렌더링은 미검증이며, 프로덕션 파일은
`internal/proxy/admin_k8s.go`·`internal/store/k8s.go` 2개입니다. DB 스키마·설정 변경은 없습니다.

### 배포 파일

| 파일 | 설명 |
|------|------|
| clustara-v0.9.296.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| clustara-v0.9.296.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.9.296.md | 오프라인 배포 가이드 |

### 빠른 시작

```bash
# 무결성 확인
sha256sum -c clustara-v0.9.296.tar.gz.sha256

# 이미지 로드
gunzip -c clustara-v0.9.296.tar.gz | docker load

# 실행
docker run -d --name clustara --restart=always \
  -p 9090:9090 \
  -v /opt/clustara/data:/data \
  -e UPSTREAM_BASE_URL=https://api.openai.com \
  -e UPSTREAM_API_KEY=sk-... \
  -e ADMIN_TOKEN=change-me \
  clustara:v0.9.296
```
