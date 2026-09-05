## 2026-09-05
- 선택: 미국 종목 가격이 `%.0f` 로 뭉개져 표시되던 곳 일괄 수정 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 사람이 읽는 가격 문자열을 만드는 곳 9군데가 시장과 무관하게 `%.0f` 를 쓰고 있어, 미국 종목은 $150.32 가 "150", $0.90 이 "1" 로 찍혔다 — (a) 6개 전략의 신호 사유(저널 `reason_in` 에 영구 기록, 사후 분석의 유일한 단서), (b) 엔진 진입 이벤트의 `손절/목표`(텔레그램·이슈 알림), (c) GitHub 상태 표의 평균가·현재가·손절·목표·평가손익. 운영자가 상태 표에서 보호선을 확인할 수 없고, 저널만 봐서는 어느 가격에서 신호가 났는지 복원할 수 없는 문제다. 이미 있던 `market.FormatPrice(Country, ...)`(KR 정수 / US 2자리, $1 미만은 4자리)로 통일하고, 평가손익은 통화 기호가 붙는 `market.Money(Currency, ...)` 로 바꿨다. 테스트를 위해 `engine.entryReason`·`agent.positionRow` 를 순수 함수로 분리했다(동작 동일). 검증: 신규 테스트 3개(전략 사유 US/KR/1달러 미만, 진입 이벤트 사유, 상태 표 3케이스 — `internal/agent` 첫 테스트 파일) 추가 후 세 곳을 모두 옛 `%.0f` 로 되돌리면 실제로 "신고가 103"·"손절 148"·"| 150 | 152 | +18 |" 를 내며 실패함을 확인했다. `gofmt -l`(clean)·`go vet ./...`·`go build ./...`·`go test -count=1 ./...` 전부 통과. 커밋 212afd4.
- 보류 아이디어:
  - GitHub Actions CI 없음 — 워크플로 추가 (위험 재평가: 푸시 토큰에 `workflow` 스코프가 없으면 외부 푸시 스크립트가 거부당해 회차 전체가 날아간다 → 위험 1→3)
  - `internal/journal`·`internal/notify` 테스트 0건 — Summary/MaxDrawdown, Load 날짜 필터, Notifier httptest 테스트 (가치 3 / 위험 1 / S)
  - `notify.Notifier` 를 구조체 리터럴로 만들면 `http` 가 nil → 패닉, `Send` 안에서 지연 초기화 (가치 3 / 위험 1 / S)
  - `Config.Validate` 가 gap_reclaim 파라미터(GapMin/GapMax/GapPullMin/GapVolMult)를 검사하지 않음 — `GapMin >= GapMax` 면 신호가 영영 안 나오는데 조용히 통과 (가치 3 / 위험 1 / S)
  - `journal.Summary` 가 PnL==0 인 거래를 패배로 집계 (`t.PnL > 0` else) — 승률·평균손실 왜곡 (가치 2 / 위험 1 / S)
