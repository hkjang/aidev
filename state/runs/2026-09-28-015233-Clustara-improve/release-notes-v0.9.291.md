## 조용한 시간이 게이트웨이 컨테이너의 시계로 판정돼 서울 운영자가 창을 9시간 어긋나게 받았습니다

`POST /admin/k8s/notify/scan` 은 조용한 시간을 `time.Now().Hour()` — 게이트웨이 컨테이너의 로컬
시각 — 으로 판정했습니다. 폐쇄망 배포 이미지에는 `/usr/share/zoneinfo` 가 없는 경우가 많아 그
시계는 사실상 UTC 였고, 서울에서 `22-08` 을 저장한 운영자는 요청한 것의 반대를 얻었습니다.
이번 릴리즈는 알림 설정에 `timezone` 을 추가해, 조용한 시간을 운영자가 지정한 타임존으로
판정하고 어느 시계를 썼는지 응답에 적습니다.

### 조용한 시간 · 운영자가 지정한 타임존으로 판정합니다

`handleK8sNotifyScan` 이 컨테이너 로컬 시각으로 창을 판정했기 때문에, zoneinfo 가 없는 이미지에서
`22-08` 은 KST 로 07:00~17:00 이 되었습니다 — 밤새 알림이 나가고 업무시간이 조용해집니다. 같은
제품의 노드 추세 "업무시간" 필터는 브라우저 타임존을 쓰고 그 사실을 화면에 적는데, 조용한 시간만
어느 시계로 판정하는지 말하지 않았습니다.

`GET/POST /admin/k8s/notify/config` 가 `timezone` 을 함께 다룹니다. IANA 이름(`Asia/Seoul`, `UTC`
등)만 저장하며, 쓰기 검증은 읽는 쪽과 같은 호출(`time.LoadLocation`)로 합니다. 스캔이 불러올 수
없는 값은 400 `invalid_timezone` 으로 거절하고 이전 설정을 그대로 남깁니다(부분 적용 없음 —
`quiet_hours` 와 같은 순서). 스캔 응답은 실제로 판정에 쓰인 `timezone` 을 보고하고, 저장된
타임존을 불러올 수 없어 로컬 시각으로 되돌아간 경우에는 `timezone_notice` 를 덧붙입니다.
이미지에 타임존 데이터가 없어도 이름이 해석되도록 게이트웨이는 Go 의 임베디드 타임존
DB(`time/tzdata`)를 포함합니다.

### 이미 저장된 설정의 의미는 바뀌지 않습니다

`timezone` 의 빈 값은 서버 로컬 시각을 쓰는 기존 동작이고, 지금까지 저장된 모든 창이 판정받아 온
시계가 바로 그것입니다. 이번 릴리즈로 갑자기 새 억제 창이 생겨 알림이 사라지거나, 반대로 기존
억제가 풀리는 일은 없습니다. 창을 의도한 시계로 옮기려면 타임존을 명시적으로 저장하십시오.
DB 스키마 변경은 없으며 중복 제거 윈도우·담당팀 채널 라우팅·딥링크 동작은 그대로입니다.

### 검증

신규 종단 테스트 3개(실제 SQLite 스토어 · `Server.Routes` · httptest Mattermost webhook)를 수정
전 코드에 먼저 붙였습니다. 테스트는 실행 호스트의 로컬 시각과 3시간 이상 떨어진 실제 IANA 존을
골라 양방향으로 단언합니다 — 지정한 존에서만 조용한 창은 억제되고, 호스트 존에서만 조용한 창은
억제되지 않고 finding 이 webhook 에 도달합니다. `.In(loc)` 만 되돌리면 같은 테스트가 다시
실패합니다. `go build ./...`, `go vet ./...`, `go test ./... -count=1`(19 패키지 전부 ok) 및 버전
일치·산출물 이름 릴리즈 게이트를 통과했습니다.

실 Kubernetes·PostgreSQL·실 Mattermost·브라우저 렌더링은 미검증이며, DST 경계 동작은
`time.Time.In` 에 맡겼습니다.

### 배포 파일

| 파일 | 설명 |
|------|------|
| clustara-v0.9.291.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| clustara-v0.9.291.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.9.291.md | 오프라인 배포 가이드 |

### 빠른 시작

```bash
# 무결성 확인
sha256sum -c clustara-v0.9.291.tar.gz.sha256

# 이미지 로드
gunzip -c clustara-v0.9.291.tar.gz | docker load

# 실행
docker run -d --name clustara --restart=always \
  -p 9090:9090 \
  -v /opt/clustara/data:/data \
  -e UPSTREAM_BASE_URL=https://api.openai.com \
  -e UPSTREAM_API_KEY=sk-... \
  -e ADMIN_TOKEN=change-me \
  clustara:v0.9.291
```
