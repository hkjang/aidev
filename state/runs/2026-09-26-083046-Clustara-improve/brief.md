# 과제서 — 2026-09-26-083046-Clustara-improve (base main@adc3294)

- 과제: 알림 설정 저장 시 `quiet_hours`·`team_channels` 를 읽는 쪽과 같은 계약으로 검증하기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `POST /admin/k8s/notify/config`(`internal/proxy/k8s_notify.go:176-206`)는 `quiet_hours` 를 `strings.TrimSpace` 만 하고 플래그에 저장하는데, 그 값을 읽는 `inQuietHours`(같은 파일 27-45행)는 형식 불일치·비숫자·범위 밖(`25-30`) 을 전부 "조용한 시간 없음" 으로 삼켜서 운영자가 오타를 내면 UI 는 "알림 설정 저장됨" 을 띄우고 GET 도 그 값을 되돌려 주지만 심야 알림이 계속 나간다. 같은 핸들러의 `team_channels` 는 `json.Valid` 만 보므로 `[1,2]` 같은 배열이 "must be JSON object" 검사를 통과한 뒤 `resolveTeamChannel`(47-57행)의 `Unmarshal(map)` 에서 실패해 모든 팀 라우팅이 조용히 기본 채널로 떨어진다 — 둘 다 쓰기 검증이 읽기 파서보다 느슨해서 실패가 보이지 않는 결함이다.
- 수용 기준:
  1) 잘못된 `quiet_hours`(`25-30`, `22`, `abc-8`, `-5`, `22-`)를 POST 하면 400 + 명확한 에러 코드(예: `invalid_quiet_hours`)로 거절되고, 저장돼 있던 이전 플래그 값이 그대로 남는다(거절 후 GET 이 이전 값을 돌려준다).
  2) 유효한 값(`22-08`, `0-6`, `23-0`, 빈 문자열=해제)은 지금과 똑같이 저장되고, `inQuietHours` 의 기존 동작(midnight wrap, `start==end` → 조용하지 않음, 이미 저장된 잘못된 값은 여전히 "조용하지 않음")은 바뀌지 않는다 — 읽기는 관용적으로, 쓰기만 엄격하게.
  3) `team_channels` 가 JSON 이지만 객체가 아닐 때(`[1,2]`, `"x"`, `3`) 400 으로 거절된다. 값이 객체면 지금처럼 통과한다.
  4) 테스트가 증명할 것: (a) 잘못된 quiet_hours POST 가 400 이고 플래그 미변경 (b) 유효 값 POST 후 그 값이 실제로 스캔을 억제한다 — `POST /admin/k8s/notify/config` 로 현재 시각을 덮는 창(`h:=time.Now().Hour(); fmt.Sprintf("%d-%d", h, (h+1)%24)`; h=23 이면 `23-0` 이 wrap 으로 참)을 저장하고 `POST /admin/k8s/notify/scan` 이 `suppressed:"quiet_hours"` 를 돌려주는 것을 실제 `Server.Routes()` 로 확인 (c) JSON 배열 team_channels 가 400 이고 `resolveTeamChannel` 이 그런 값에서 "" 를 주는 것.
- 건드릴 파일:
  - `internal/proxy/k8s_notify.go` — `parseQuietHours(spec string) (start, end int, ok bool)` 를 새로 만들고 `inQuietHours` 가 그것을 호출하도록 바꿔(동작 동일) `handleK8sNotifyConfig` 의 POST 가 같은 함수로 검증해 400 반환. 같은 POST 블록의 `team_channels` 검사를 `json.Valid` → `json.Unmarshal` into `map[string]string`(또는 `map[string]any` 로 객체 여부 확인) 로 바꿈. 에러 응답은 옆줄과 동일하게 `writeOpenAIError(w, http.StatusBadRequest, ..., "invalid_request_error", ...)`.
  - `internal/proxy/k8s_notify_test.go` — 기존 `TestInQuietHours`(5행)·`TestResolveTeamChannel`(28행) 표에 새 케이스를 붙이고 `parseQuietHours` 단위 테스트 추가.
  - `internal/proxy/k8s_notify_scan_test.go` 또는 새 테스트 파일 — 종단 테스트. 배선은 이 파일 37-47행 그대로: `openTestStore(t)` + `store.NewAsyncLogger(db, 32, filepath.Join(t.TempDir(), "fallback.ndjson"))` + `NewServer(testConfig("http://upstream.invalid","secret"), db, logger, nil)` + `httptest.NewServer(server.Routes())` + `postJSON(t, proxy.URL+"/admin/...", "", body)`(bearer 는 빈 문자열로 통과함 — 같은 파일 101행에서 확인).
  - (선택) `docs/K8S_OPERATIONS_HUB.md:873` 의 `/admin/k8s/notify/config` 줄에 "`HH-HH`(0-23), 잘못된 값은 400" 을 덧붙이는 것까지만. 버전 마커·changelog·다른 문서는 손대지 말 것.
- 검증 명령 (이 저장소에서 실제로 도는 것):
  1) `go test ./internal/proxy -run 'K8sNotify|QuietHours|TeamChannel' -count=1`
  2) `go build ./...` → `go vet ./...` → `go test ./...`(전체 약 80s)
  3) 수정한 파일만 `gofmt -l`
  - 고치기 전 코드에 새 테스트를 붙여 (a) 는 200(저장 성공)으로, (b) 는 잘못된 값 저장 후에도 스캔이 억제되지 않는 것으로, (c) 는 200 으로 실패하는 것을 먼저 확인할 것.
- 위험과 피할 것:
  - **읽기 경로를 엄격하게 만들지 말 것.** 이미 DB 에 잘못된 `k8s_quiet_hours` 가 저장돼 있을 수 있고, 그때 지금 동작은 "조용한 시간 없음"(알림 나감)이다. `inQuietHours` 가 에러를 내거나 기본 창을 만들면 운영 중 알림이 사라진다.
  - `mattermost_team_channels` 플래그는 notify 스캔 밖(다른 Mattermost 경로)에서도 읽힐 수 있다 — 쓰기 검증만 바꾸고 읽는 쪽 `resolveTeamChannel` 의 관용적 실패(빈 문자열 → 기본 채널)는 그대로 둘 것.
  - `mattermostConfig` 스냅샷은 15초 캐시다. 테스트에서 플래그를 바꾸면 `server.invalidateMattermostCache()` 필요(quiet_hours 는 캐시 대상이 아니라 `s.flagValue` 로 매번 읽지만, 같은 테스트에서 Mattermost 플래그를 만지면 해당됨).
  - `internal/proxy/docs_reference_test.go`·`release_gate_test.go` 가 docs 의 API 경로·버전 마커를 검사한다 — 문서는 위에 적은 한 줄 외에 건드리지 말 것.
  - 인증(`evaluateAdminAccess`)·`store/sqlstore.go` DDL·`.github/` 는 이번 범위 밖.
- 차선 후보: notify scan 의 podsec dedup 키에 Kind 가 없어 같은 클러스터의 동명 Pod/Deployment 가 한 알림으로 합쳐지는 것 수정 (`internal/proxy/k8s_notify.go:153` 의 `"podsec/"+p.Namespace+"/"+p.Name` → Kind 포함; 145행 rbac 키와 대비). 기존 6시간 dedup 기록과 일회성 불일치가 생기는 점을 커밋 메시지에 적을 것.

## 미확인 (구현자가 확인해야 할 것)
- `quiet_hours` 를 이 핸들러 밖에서 쓰는 경로는 grep 으로 없음을 봤지만(`k8s_quiet_hours` 는 `k8s_notify.go` 3곳뿐), 스케줄러가 스캔을 주기 실행하는 경로는 이번에 읽지 않았다.
- `start==end`(`22-22`) 를 400 으로 거절할지는 판단에 맡긴다. 해제는 빈 문자열이 정본이므로 거절을 권하지만, 관용적으로 통과시키고 지금처럼 "조용하지 않음" 으로 두어도 수용 기준을 깨지 않는다.
- 실 Kubernetes·실 PostgreSQL·브라우저 렌더링은 이번 정찰에서 미확인(SQLite + httptest 만).
