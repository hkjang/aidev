## 2026-09-06
- 선택: 스윙 실행기가 마지막 봉만 보고 체결·청산하던 버그 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 직전 커밋(9077310)이 넣은 일봉 스윙 모의 실행기 `swing.Runner` 는 테스트가 0건이었고, 예약 진입을 **"오늘 시가"** 에, 청산 판정을 **마지막 봉 하나**에만 적용했다 — 매일 거르지 않고 실행할 때만 성립하는 가정이다. 하루라도 거르면(당일 일봉 지연으로 `!anyToday` 조기 반환, 프로세스 중단 등) 신호일 다음 봉이 아니라 며칠 뒤 시가로 체결되고, 거른 날에 났던 손절·목표는 통째로 무시된 채 마지막 봉으로만 판정되어 전진 검증 기록이 백테스트(`Simulate`)와 조용히 어긋났다. 이 모듈의 존재 이유가 "백테스트와 같은 규칙으로 앞으로 돌려보는 것" 이라 기록이 거짓이 되는 결함이다. 진입은 `SignalDate` 보다 큰 첫 봉을 찾아 그 봉 시가로 체결하고 `EntryDate` 도 그 날짜로 남기게, 청산은 진입 다음 봉부터 마지막 봉까지 `Simulate` 와 같은 우선순위(갭 손절 → 손절 → 목표 → 기간 만료)로 순회하게 바꿨다. 청산일과 RFC3339 오프셋도 실제 청산 봉 기준으로 기록한다(과거 봉의 서머타임도 맞음). 테스트를 위해 데이터 원본 주입용 `PoolFn`/`DailyFn` 필드를 추가했다(nil 이면 기존대로 `universe` 패키지 — 운영 경로 동작 불변). 검증: `internal/swing/runner_test.go` 신규 3테스트(하루 거른 경로·매일 실행 경로가 `Simulate` 와 같은 체결가·청산일·손익을 내는지, 같은 날 재실행 멱등)를 추가하고, 러너를 옛 로직으로 되돌리면 하루 거른 테스트가 실제로 "청산 기록 0건" 으로 실패함을 확인했다. `gofmt -l`(clean)·`go vet ./...`·`go build ./...`·`go test -count=1 ./...` 전부 통과. 커밋 e63952b.
- 보류 아이디어:
  - `swing.Runner` 가 `MaxPositions` 를 동시 보유에 전혀 적용하지 않음 — 현금만 한도라 MaxPositions=1 설정에서 8종목까지 동시 보유 가능 (가치 3 / 위험 2 / S)
  - `internal/journal`·`internal/notify` 테스트 0건 — Summary/MaxDrawdown, Load 날짜 필터, Notifier httptest 테스트 (가치 3 / 위험 1 / S)
  - `notify.Notifier` 를 구조체 리터럴로 만들면 `http` 가 nil → 패닉, `Send` 안에서 지연 초기화 (가치 3 / 위험 1 / S)
  - `Config.Validate` 가 gap_reclaim 파라미터(GapMin/GapMax/GapPullMin/GapVolMult)를 검사하지 않음 — `GapMin >= GapMax` 면 신호가 영영 안 나오는데 조용히 통과 (가치 3 / 위험 1 / S)
  - `journal.Summary`·`swing.Evaluate` 가 PnL==0 인 거래를 패배로 집계 (`PnL > 0` else) — 승률·평균손실 왜곡 (가치 2 / 위험 1 / S)
