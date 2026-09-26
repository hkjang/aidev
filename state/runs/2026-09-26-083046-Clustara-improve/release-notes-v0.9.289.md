## 잘못 입력한 조용한 시간과 팀 채널이 "저장됨" 으로 보이고도 아무 일도 하지 않았습니다

`POST /admin/k8s/notify/config` 는 `quiet_hours` 를 TrimSpace 만 하고, `team_channels` 는
`json.Valid` 만 확인하고 저장했습니다. 읽는 쪽은 그보다 좁은 계약을 요구하기 때문에, 저장에
성공한 값이 실제로는 조용한 시간을 만들지도 담당팀 채널을 라우팅하지도 않았습니다. 이번
릴리즈는 쓰기 검증을 읽는 쪽과 같은 계약으로 맞춥니다.

### 알림 설정 저장 · `quiet_hours` 와 `team_channels` 를 읽는 쪽과 같은 계약으로 검증합니다

`inQuietHours` 는 형식이 깨졌거나 숫자가 아니거나 범위를 벗어난 `quiet_hours` 를 "조용한 시간
없음" 으로 삼킵니다. 그래서 운영자가 `25-30`·`22`·`abc-8`·`-5`·`22-`·`22-22` 를 입력해도 응답은
`200 {"ok":true}` 였고 GET 이 그 값을 그대로 돌려줘 설정된 것처럼 보였지만, 심야 알림은 계속
나갔습니다. `team_channels` 도 `json.Valid` 만 통과하면 저장돼 `[1,2]`·`"x"`·`3`·`null`·
`{"core":3}` 가 들어갔고, 이후 `resolveTeamChannel` 의 `map[string]string` Unmarshal 에서 조용히
실패해 모든 팀의 알림이 기본 채널로 떨어졌습니다.

이제 파싱은 `parseQuietHours` 한 곳에 모으고, 쓰기 경로에만 `validateQuietHours` 를 적용합니다 —
`HH-HH` 형식, 두 시각 모두 0-23, `start != end`, 빈 값은 해제. `team_channels` 는 읽는 쪽과 같은
`map[string]string` 로 Unmarshal 되어야만 저장합니다. 두 필드를 모두 검증한 뒤에 플래그를 쓰므로
거절된 POST 는 이전 설정을 그대로 남기며 반쯤 적용되지 않습니다. 관리 UI 는 400 을 그대로
표시합니다.

읽기 경로(`inQuietHours`)는 의도적으로 관용적인 동작을 그대로 둡니다. 이미 플래그에 저장된 값의
의미가 바뀌지 않으므로, 이번 릴리즈로 없던 억제 창이 갑자기 생기거나 기존 억제가 풀리는 일은
없습니다. 전달 가능한 스캔의 중복 제거·조용한 시간 판정·담당팀 채널 라우팅·딥링크 동작은
그대로이며, DB 스키마 변경은 없습니다.

### 검증

신규 테스트 5개 — 단위 `TestParseQuietHours`·`TestValidateQuietHours`, `TestInQuietHours` 와
`TestResolveTeamChannel` 의 케이스 추가, 그리고 실제 SQLite·`Server.Routes` 종단 3개(잘못된 POST 가
400 이고 플래그가 바뀌지 않음, 현재 시각을 덮는 창을 저장하면 scan 이 `suppressed:"quiet_hours"`
이고 거절이 그 창을 지우지 않음, 해제 후 webhook 수신). 수정 전 코드에 먼저 붙여 모든 잘못된
값이 `200 {"ok":true}` 로 저장되고 GET 이 그대로 돌려주는 것을 확인했습니다.

`go build ./...`, `go vet ./...`, `go test ./...`(20개 패키지 전부 통과) 및 버전 일치·산출물 이름
릴리즈 게이트를 실행했습니다. 실 Kubernetes·PostgreSQL·실 Mattermost·브라우저 렌더링은
미검증입니다.

### 배포 파일

| 파일 | 설명 |
|------|------|
| clustara-v0.9.289.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| clustara-v0.9.289.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.9.289.md | 오프라인 배포 가이드 |

### 빠른 시작

```bash
# 이미지 로드
gunzip -c clustara-v0.9.289.tar.gz | docker load

# 실행
docker run -d --name clustara --restart=always \
  -p 9090:9090 \
  -v /opt/clustara/data:/data \
  -e UPSTREAM_BASE_URL=https://api.openai.com \
  -e UPSTREAM_API_KEY=sk-... \
  -e ADMIN_TOKEN=change-me \
  clustara:v0.9.289
```
