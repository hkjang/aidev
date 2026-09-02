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
