## 2026-10-05
- 선택: 이벤트 목록 API의 실효 조회 상한과 창 포화 여부 명시 (가치 2 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: GET /admin/k8s/events의 스토어 상한 500이 응답에 드러나지 않아 일부 조회를 전체로 오인할 수 있던 문제를 고쳤다. 공유 K8sEventQueryLimit으로 스토어·핸들러의 정규화를 맞추고, 기존 events 배열에 limit/window_full 및 포화 시에만 보수적인 window_notice를 추가했으며 기존 운영 가이드를 갱신했다. 실 SQLite+Server.Routes HTTP 테스트 2개(성공 창 18케이스, 오류 3케이스)로 경계·기본값·클러스터 격리·최신순·빈 배열·오류 계약을 검증하고, 수정 전 실패 및 응답 메타데이터 제거 재실패를 확인한 뒤 gofmt(출력 없음), go test ./internal/proxy -run 'K8sEvents|NotifyScan' -count=1(ok 2.314s), go build ./..., go vet ./..., go test ./... -count=1(실패 0, proxy 67.557s, store 16.214s)을 통과했다; 프로덕션 2파일, 커밋 0c79c53.
- 실패 재현: `--- FAIL: TestK8sEventsReportsEffectiveWindow/over_cap (0.01s)` / `admin_k8s_events_window_test.go:141: response is missing effective limit; requested "&limit=1000", store returned 500 rows`
- 보류 아이디어: /admin/k8s/inventory의 실효 상한과 창 포화 미표시 (가치 2 / 위험 1 / 작업량 S) — 별도 엔드포인트로 분리.
- 보류 아이디어: events/revisions 스토어 상한이 다른 호출자 요청을 조용히 깎는 계약 불일치 (가치 4 / 위험 3 / 작업량 L) — 이번은 events GET만 완료.
- 보류 아이디어: notifyScanTargets의 클러스터 목록 조회 오류가 빈 레지스트리 폴백과 구별되지 않음 (가치 3 / 위험 2 / 작업량 S) — 직전 변경과 같은 경로이므로 보류.
- 보류 아이디어: notify scan이 조용한 시간에도 팬아웃 조회를 먼저 실행 (가치 2 / 위험 2 / 작업량 S) — 오류·suppressed 우선순위 변경은 별도 과제.
- 과제서: 채택 — 핸들러의 무메타데이터 응답과 스토어 500 제한을 현 코드 및 수정 전 실제 HTTP 실패로 확인했고 지정한 4파일 범위를 지켰다.
