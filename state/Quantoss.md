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
