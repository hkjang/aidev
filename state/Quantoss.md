# Quantoss 자율 개선 기록

## 2026-09-03
- 선택: 설정 검증(`config.Validate`) 추가 — .env 오타를 시작 시 차단 (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 실거래 봇인데 `.env` 값에 대한 범위 검증이 전혀 없어, 오타 하나가 조용히 사고로 이어질 수 있었다(`QUANTOSS_POLL_SECONDS=0` → 에이전트 티커 패닉, `QUANTOSS_FLATTEN_AT=25:15` → 강제 청산이 영영 발동 안 함, `QUANTOSS_DAILY_LOSS_LIMIT_PCT=-3` → 첫 거래부터 진입 차단, 수수료/거래세를 비율 아닌 %로 입력 → 손익 왜곡). `Config.Validate()` 로 리스크%·비중·시각 순서(개장 ≤ 진입 시작 < 진입 종료 ≤ 강제 청산)·가격대·비용 단위·상관 임계값·시세 장애 임계값을 검사하고 문제를 **모두 모아 한 번에** 보고하도록 했고, `ParseClock` 에 HH 0~23 / MM 0~59 범위 검사를 넣었다. 호출 지점은 `FromEnv()` 끝과 `optimize` 파라미터 자동 적용 직후(손상된 params.json 방어). 검증: `gofmt -l`(clean) · `go vet ./...` · `go test ./...` 전부 통과, 신규 테스트 5개(기본 설정 통과 / 위험값 33종 거부 / 다중 문제 동시 보고 / ParseClock 범위 / FromEnv 거부) 추가. 커밋 110975b.
- 보류 아이디어:
  - GitHub Actions CI 없음 — `go build`/`go vet`/`go test`/`gofmt -l` 워크플로 추가 (가치 4 / 위험 1 / S)
  - `internal/journal`·`internal/notify` 테스트 0건 — Summary/MaxDrawdown, Load 날짜 필터, Notifier httptest 테스트 (가치 3 / 위험 1 / S)
  - README "알려진 한계" 가 stale — WebSocket 미사용이라 적혀 있으나 이미 구현됨 (가치 2 / 위험 1 / S)
  - `risk.RecordPartial` 은 DailyPnL 만 더하고 일일 손실 한도를 재평가하지 않음 — 절반 익절 손실이 한도를 넘겨도 halt 안 됨 (가치 3 / 위험 2 / S)
  - `market.FetchCandles` 종료 조건 `page[0].TS.Before(cutoff) || !page[0].TS.After(cutoff)` 가 중복 — 단순화 + 페이지네이션 테스트 (가치 2 / 위험 1 / S)

## 2026-09-03
- 선택: 부분 청산 손실이 일일 손실 한도를 우회하던 버그 수정 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `risk.RecordPartial` 이 `DailyPnL` 만 더하고 한도를 재평가하지 않아, 손실인 부분 청산(서버 조건주문 부분 체결 → `engine.ExternalExit` 경로)이 일일 손실 한도를 넘겨도 `HaltedReason` 이 비어 있어 신규 진입이 계속 허용됐다 — 실거래에서 손실이 한도 너머로 누적될 수 있는 경로다. 한도 판정을 `evaluateHalt()` 로 분리해 `RecordClose`/`RecordPartial` 이 공유하게 했고, 겸사겸사 (a) `dayStartEquity` 가 0 일 때(NewDay 미호출) 한도가 0 이 되어 손익 -1 에도 중단되던 오판을 막고, (b) 먼저 정해진 중단 사유가 이후 판정으로 덮어써지고 매 청산마다 경고 로그가 반복되던 것을 최초 1회로 고정했다. 검증: 신규 테스트 4개(부분 청산 손실 한도 발동/거래·연승 카운트 제외, 이익 부분 청산은 무중단, 사유 고정, NewDay 전 무중단) 추가 후 `gofmt -l`(clean)·`go vet ./...`·`go build ./...`·`go test ./...` 전부 통과. 수정을 되돌리면 새 테스트가 실제로 실패함을 확인해 회귀 방지력을 검증했다. 커밋 add77ac.
- 보류 아이디어:
  - GitHub Actions CI 없음 — `go build`/`go vet`/`go test`/`gofmt -l` 워크플로 추가 (가치 4 / 위험 1 / S)
  - `internal/journal`·`internal/notify` 테스트 0건 — Summary/MaxDrawdown, Load 날짜 필터, Notifier httptest 테스트 (가치 3 / 위험 1 / S)
  - `journal.Summary` 가 PnL==0 인 거래를 패배로 집계 (`t.PnL > 0` else) — 승률이 미세하게 왜곡 (가치 2 / 위험 1 / S)
  - README "알려진 한계" 가 stale — WebSocket 미사용이라 적혀 있으나 이미 구현됨 (가치 2 / 위험 1 / S)
  - `market.FetchCandles` 종료 조건 `page[0].TS.Before(cutoff) || !page[0].TS.After(cutoff)` 가 중복 — 단순화 + 페이지네이션 테스트 (가치 2 / 위험 1 / S)

## 2026-09-03
- 선택: 청산 실패 시 포지션이 보호 없이 남던 버그 수정 (서버 손절 예약 복구 + 절반 익절 재시도) (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `Live.Sell` 은 이중 매도를 막으려 서버 손절 예약(조건주문)을 먼저 취소하는데, 그 뒤 주문 접수 실패·판매가능수량 0·미체결로 빠져나가는 경로에서 예약을 되살리지 않았다. `GuardID` 가 비어 `UpdateStop` 도 갱신을 건너뛰므로 실거래 포지션이 장 끝까지 **서버측 손절 없이** 남는 안전 결함이다. 실패 경로마다 `restoreGuard` 로 복구하고, 살아 있는 미체결 매도 주문은 매수와 동일하게 취소 후 최종 상태를 재확인하도록 했다(취소 후에도 주문이 살아 있으면 이중 매도 위험이라 예약은 복구하지 않고 ERROR 로그). `cancelGuard` 는 취소가 확인된 경우에만 `GuardID` 를 비우고 성공 여부를 반환하게 바꿔, 취소 실패 시 살아 있는 예약을 잊고 `placeGuard` 가 하나 더 만드는 이중 예약도 막았다(`placeGuard` 에도 `GuardID != ""` 조기 반환 추가, `UpdateStop` 은 취소 실패 시 교체 보류). 함께 엔진의 같은 계열 버그도 고쳤다 — 절반 익절 매도가 실패해도 `pos.ScaledOut = true` 를 먼저 세워, 절반 익절이 영영 재시도되지 않고 `structureExit`(잔량 청산)·`timeStop` 이 아직 전량인 포지션에 잘못 적용됐다. 이제 체결된 뒤에만 세운다(목표 도달 시 본전 손절 상향은 실패해도 유지). 검증: httptest 기반 `internal/broker/live_test.go` 신규(매도 실패 4종 + `UpdateStop` 취소 실패)와 엔진 절반 익절 재시도 테스트를 추가하고, 수정을 각각 되돌리면 새 테스트가 실제로 실패함을 확인했다. `gofmt -l`(clean)·`go vet ./...`·`go build ./...`·`go test -count=1 ./...` 전부 통과. 커밋 e0d05fa.
- 보류 아이디어:
  - GitHub Actions CI 없음 — `go build`/`go vet`/`go test`/`gofmt -l` 워크플로 추가 (가치 4 / 위험 1 / S)
  - `internal/journal`·`internal/notify` 테스트 0건 — Summary/MaxDrawdown, Load 날짜 필터, Notifier httptest 테스트 (가치 3 / 위험 1 / S)
  - `toss.Client` 를 구조체 리터럴로 만들면 `limiter.tokens` 가 nil 이라 첫 요청에서 패닉 — `acquire` 에 지연 초기화 (가치 3 / 위험 1 / S)
  - `journal.Summary` 가 PnL==0 인 거래를 패배로 집계 (`t.PnL > 0` else) — 승률이 미세하게 왜곡 (가치 2 / 위험 1 / S)
  - README "알려진 한계" 가 stale — WebSocket 미사용이라 적혀 있으나 이미 구현됨 (가치 2 / 위험 1 / S)

## 2026-09-04
- 선택: 미국 종목 트레일링 손절이 KRX 호가단위로 뭉개지던 버그 수정 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `engine.manage` 의 트레일링 손절 한 줄만 국내 전용 `market.RoundToTick`(KRX 호가 계단)을 쓰고 있었고, 나머지 가격 계산은 모두 시장별 `market.RoundTick(Country, ...)` 를 쓰고 있었다. 미국 종목은 전 가격대가 KRX 기준 "2,000 미만 → 호가 1" 에 걸려 트레일 손절이 **달러 단위로 내림**됐다 — $150 종목이면 의도보다 최대 $1(0.7%) 느슨해지고, $1 미만 종목은 내림 결과가 0 이라 트레일링이 아예 발동하지 않았다. `RoundTick(e.Cfg.Country, ...)` 로 교체하고, 같은 실수가 반복되지 않도록 이제 자기 테스트에서만 쓰이던 국내 전용 `RoundToTick` 을 제거(호출부는 `RoundTick("KR", ...)` 로 대체)했다. 검증: `internal/engine/trail_test.go` 신규 3케이스(US $0.01 호가 → 154.37, US $1 미만 → 0.9032, KR 계단 호가 → 10,350) 추가 후 수정을 되돌리면 US 두 케이스가 실제로 실패(154, 0.8=미갱신)함을 확인. `gofmt -l`(clean)·`go vet ./...`·`go build ./...`·`go test -count=1 ./...` 전부 통과. 커밋 3f86050.
- 보류 아이디어:
  - GitHub Actions CI 없음 — `go build`/`go vet`/`go test`/`gofmt -l` 워크플로 추가 (가치 4 / 위험 1 / S)
  - `internal/journal`·`internal/notify` 테스트 0건 — Summary/MaxDrawdown, Load 날짜 필터, Notifier httptest 테스트 (가치 3 / 위험 1 / S)
  - `notify.Notifier` 를 구조체 리터럴로 만들면 `http` 가 nil (toss.Client 의 limiter nil 패닉과 같은 계열) — 지연 초기화 (가치 3 / 위험 1 / S)
  - `journal.Summary` 가 PnL==0 인 거래를 패배로 집계 (`t.PnL > 0` else) — 승률이 미세하게 왜곡 (가치 2 / 위험 1 / S)
  - 진입 알림/로그가 손절·목표가를 `%.0f` 로 출력 — 미국 종목은 $3.45 가 "3" 으로 보임, `market.FormatPrice(Country, ...)` 사용 (가치 2 / 위험 1 / S)

## 2026-09-04
- 선택: 주간·누적 수익률이 외부 입출금을 수익률로 잡던 버그 수정 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 직전 커밋 038e412 가 **일일** 수익률만 `실현손익/시작자산` 으로 고쳤고, 주간 리뷰(`weekly.go`)·리포트 인덱스(`IndexMarkdown`)·대시보드 타일(`docs/index.html` 누적 수익률)은 그대로 자산 곡선 양 끝 `(EndEquity/StartEquity - 1)` 로 계산하고 있었다. 입금·배당세·환전이 그대로 수익률이 되어, 100,000원 입금이 낀 주는 `+10101.00%` 로 표시된다(실제 +2.01%) — 봇이 되는지 판단하는 헤드라인 숫자가 망가지는 문제. 일별 `ReturnPct` 를 복리로 합성하는 `report.CumulativeReturnPct(days)` 를 추가해 세 곳을 통일했다(곱셈이라 정렬 순서 무관 — 인덱스는 내림차순, 주간은 오름차순이라 중요). 검증: `internal/report/cumulative_test.go` 신규 3테스트(복리/순서무관/빈 슬라이스 + 입금 주의 주간·인덱스 마크다운) 추가 후, 두 호출부를 옛 계산식으로 되돌리면 실제로 `+10101.00%` 를 출력하며 실패함을 확인. `gofmt -l`(clean)·`go vet ./...`·`go build ./...`·`go test -count=1 ./...` 전부 통과. 커밋 967e9b8.
- 보류 아이디어:
  - GitHub Actions CI 없음 — `go build`/`go vet`/`go test`/`gofmt -l` 워크플로 추가 (가치 4 / 위험 1 / S)
  - `internal/journal`·`internal/notify` 테스트 0건 — Summary/MaxDrawdown, Load 날짜 필터, Notifier httptest 테스트 (가치 3 / 위험 1 / S)
  - `notify.Notifier` 를 구조체 리터럴로 만들면 `http` 가 nil (toss.Client 의 limiter nil 패닉과 같은 계열) — 지연 초기화 (가치 3 / 위험 1 / S)
  - `agent/github.go` 상태 표(평균가·현재가·손절·목표)가 `%.0f` — 미국 종목 $150.32 가 "150", $0.90 이 "1" 로 보임, `market.FormatPrice(Country, ...)` 사용 (가치 3 / 위험 1 / S)
  - `journal.Summary` 가 PnL==0 인 거래를 패배로 집계 (`t.PnL > 0` else) — 승률·평균손실이 미세하게 왜곡 (가치 2 / 위험 1 / S)

## 2026-09-05
- 선택: 전략 override(QUANTOSS_US_STRATEGY·QUANTOSS_PAPER_STRATEGY)가 실제 실행에 반영되지 않던 버그 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `runAgent` 가 전략을 `QUANTOSS_STRATEGY` 하나로 **미리 한 번** 해석해(`mk`) 모든 시장에 재사용하고 있었다. `ForMarket("US")` 이 `m.Strategy = USStrategy` 를 세워도 그 값은 로그·리포트·알림 **라벨에만** 쓰여서, `QUANTOSS_MARKET=BOTH` + `QUANTOSS_US_STRATEGY=vwap_reclaim` 은 미국 세션이 국내 전략으로 돌면서 기록만 `vwap_reclaim` 으로 남았다 — 이 프로젝트의 핵심인 전진 검증 기록이 통째로 거짓이 되는 결함이다. 직전 커밋(93b0b10)이 넣은 `QUANTOSS_PAPER_STRATEGY` 도 같은 이유로 무력했다: `FromEnv` 가 `Mode=="paper"` 일 때만 `Strategy` 를 덮어쓰는데, 모의 병행 실행은 실거래와 같은 `.env`(`QUANTOSS_MODE=live`)를 쓰고 `runAgent` 가 `cfg.Mode="paper"` 로 바꾸는 것은 그보다 뒤라 조건이 성립하지 않는다. 수정: `ForMarket` 이 override 를 `Strategy` 필드에 확정(우선순위 PaperStrategy > USStrategy > Strategy, 멱등)해 실행 전략과 라벨이 늘 같은 값을 보게 하고, `runAgent` 는 시장별로 `Registry` 를 조회하되 **API 접속 전에** 모든 시장의 전략 이름을 검증한다(몇 시간 뒤 시작하는 US 세션이 오타로 그제서야 죽지 않도록). `backtest`/`optimize` 의 `-strategy` 기본값도 `marketCfg` 이후의 `cfg.Strategy` 로 바꿔 `-market US` 백테스트가 override 를 무시하던 문제를 함께 고쳤고, `-strategy` 를 명시하면 override 보다 우선하도록 했다. 검증: `internal/config/market_test.go` 에 우선순위·멱등·런타임 모드 전환 테스트 2개를 추가하고 `ForMarket` 을 옛 동작으로 되돌리면 4개 단언이 실제로 실패함을 확인. 바이너리 스모크로 `QUANTOSS_US_STRATEGY=nope_typo` → "알 수 없는 전략: nope_typo (시장 US)" 즉시 종료, `QUANTOSS_MODE=live` + `QUANTOSS_PAPER_STRATEGY=nope2` + `paper` → "(시장 KR)" 로 잡히는 것, `-strategy combo` 가 오타 override 를 이기는 것까지 확인. `gofmt -l`(clean)·`go vet ./...`·`go build ./...`·`go test -count=1 ./...` 전부 통과. 커밋 0e5821c.
- 보류 아이디어:
  - GitHub Actions CI 없음 — `go build`/`go vet`/`go test`/`gofmt -l` 워크플로 추가 (가치 4 / 위험 1 / S)
  - `internal/journal`·`internal/notify` 테스트 0건 — Summary/MaxDrawdown, Load 날짜 필터, Notifier httptest 테스트 (가치 3 / 위험 1 / S)
  - `notify.Notifier` 를 구조체 리터럴로 만들면 `http` 가 nil (toss.Client 의 limiter nil 패닉과 같은 계열) — 지연 초기화 (가치 3 / 위험 1 / S)
  - `agent/github.go` 상태 표(평균가·현재가·손절·목표)가 `%.0f` — 미국 종목 $150.32 가 "150", $0.90 이 "1" 로 보임, `market.FormatPrice(Country, ...)` 사용 (가치 3 / 위험 1 / S)
  - `Config.Validate` 가 신규 gap_reclaim 파라미터를 검사하지 않음 — `GapMin >= GapMax` 면 신호가 영영 안 나오는데 조용히 통과 (가치 3 / 위험 1 / S)
  - `journal.Summary` 가 PnL==0 인 거래를 패배로 집계 (`t.PnL > 0` else) — 승률·평균손실이 미세하게 왜곡 (가치 2 / 위험 1 / S)
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

