## 알림을 켜기 전에 돈 notify scan 이 알림을 6시간 삼키고도 보냈다고 보고했습니다

`POST /admin/k8s/notify/scan` 은 Mattermost 가 꺼져 있거나 webhook 이 없거나 카테고리가 mute 인
동안에도 finding 의 6시간 중복 제거 윈도우를 먼저 써 버렸습니다. 이번 릴리즈는 전달 가능한
알림만 윈도우를 쓰도록 순서를 고칩니다.

### notify scan · 전달할 수 없는 알림은 6시간 중복 제거 윈도우를 쓰지 않습니다

`handleK8sNotifyScan` 의 notify 클로저가 기록까지 하는 `ShouldSendK8sNotification` 을 먼저 부르고
나서야 `notifyMattermostTo` 가 Mattermost 비활성·webhook 미설정·카테고리 mute 일 때 조용히
반환했습니다. 알림을 켜기 전에 cron 으로 돈 스캔이 같은 finding 을 6시간 동안 억제했고, 그 사이
응답과 감사 로그에는 `sent` 로 적혀 운영자는 알림이 나간 줄 알았습니다.

전달 가능 판정을 `mattermostSnapshot.canNotify` 한 함수로 모아 `notifyMattermostTo` 와 스캔이 같은
조건을 읽습니다. 스캔은 윈도우를 claim 하기 전에 물어보고, 보내지 못한 건수를 응답·감사 로그의
새 필드 `undeliverable` 로 보고합니다. 알림을 나중에 켜면 그 시점의 스캔에서 바로 통지됩니다.

전달 가능한 스캔의 중복 제거(6h)·조용한 시간·담당팀 채널 라우팅·리소스 딥링크는 기존 동작을
유지합니다. DB 스키마·설정 변경은 없습니다.

### 검증

신규 end-to-end 회귀 2개(실제 SQLite·`Server.Routes`·httptest webhook — Mattermost 비활성과
카테고리 mute 각각)는 수정 전 `sent=1`(미전달인데 전송으로 집계)로 실패하고, 첫 단언을 일시
완화하면 2단계가 `sent=0`(윈도우 소모로 알림을 켠 뒤에도 미통지)로 실패해 증상 전체를
재현했습니다. `go build ./...`, `go vet ./...`, `go test ./...`(19 패키지 전부 ok)와 버전 일치·
산출물 이름 릴리즈 게이트를 실행했습니다. 실 Kubernetes·PostgreSQL·실 Mattermost·브라우저
렌더링은 미검증입니다.

### 배포 파일

| 파일 | 설명 |
|------|------|
| clustara-v0.9.288.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| clustara-v0.9.288.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.9.288.md | 오프라인 배포 가이드 |

### 빠른 시작

```bash
# 이미지 로드
gunzip -c clustara-v0.9.288.tar.gz | docker load

# 실행
docker run -d --name clustara --restart=always \
  -p 9090:9090 \
  -v /opt/clustara/data:/data \
  -e UPSTREAM_BASE_URL=https://api.openai.com \
  -e UPSTREAM_API_KEY=sk-... \
  -e ADMIN_TOKEN=change-me \
  clustara:v0.9.288
```
