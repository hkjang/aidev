---
title: "Quantoss — 자율 개선 이력"
description: "Quantoss: 자율 개선 회차 5회, 릴리즈 0건. 최근 릴리즈 없음."
last_modified_at: 2026-09-05 10:00:32 +0900
---
{% raw %}
<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "SoftwareSourceCode",
 "name": "Quantoss",
 "codeRepository": "https://github.com/hkjang/Quantoss",
 "url": "https://hkjang.github.io/aidev/projects/Quantoss/",
 "description": "Quantoss: 자율 개선 회차 5회, 릴리즈 0건. 최근 릴리즈 없음.",
 "inLanguage": "ko",
 "maintainer": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-09-05T10:00:32+09:00"
}
</script>

# Quantoss

<p class="tldr"><strong>요약.</strong> Quantoss: 자율 개선 회차 5회, 릴리즈 0건. 최근 릴리즈 없음.</p>

<ul class="stats"><li><b>5</b><span>회차</span></li><li><b>1</b><span>프로젝트</span></li><li><b>0</b><span>릴리즈</span></li><li><b>5</b><span>머지(릴리즈 없음)</span></li><li><b>0</b><span>변경 없음</span></li><li><b>0</b><span>실패</span></li></ul>

## 현황

<dl class="kv">
<dt>저장소</dt><dd><a href="https://github.com/hkjang/Quantoss">https://github.com/hkjang/Quantoss</a></dd>
<dt>마지막 회차</dt><dd>2026-09-04 22:49 KST — <span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/Quantoss/pull/9">PR #9</a>, release skipped</dd>
<dt>최근 릴리즈</dt><dd>skipped — skipped</dd>
<dt>사유</dt><dd>No release history exists in this repository: 0 git tags, 0 GitHub releases (gh authenticated as hkjang), no version file (no VERSION/package.json/version.go/version const in any Go source), no CHANGELOG.md or docs/RELEASE*.md, no .github/workflows, no Makefile or scripts/release*.sh, and no release/versioning section in README.md. The only Markdown under reports/ and docs/ is auto-generated trading report output, not release notes. Establishing a new versioning convention is a human decision, so nothing was created, committed, or tagged.</dd>
</dl>

## 회차 이력

<div class="table-wrap"><table class="rt"><thead><tr><th class="primary">일시</th><th>결과</th></tr></thead><tbody><tr data-status="merged"><td data-label="일시" class="primary">2026-09-04 22:49</td><td data-label="결과"><span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/Quantoss/pull/9">PR #9</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시" class="primary">2026-09-04 04:14</td><td data-label="결과"><span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/Quantoss/pull/6">PR #6</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시" class="primary">2026-09-03 20:39</td><td data-label="결과"><span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/Quantoss/pull/4">PR #4</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시" class="primary">2026-09-03 06:43</td><td data-label="결과"><span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/Quantoss/pull/2">PR #2</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시" class="primary">2026-09-03 00:46</td><td data-label="결과"><span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/Quantoss/pull/1">PR #1</a>, release skipped</td></tr></tbody></table></div>

## 원장 (에이전트가 남긴 기록)

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


[← 대시보드](https://hkjang.github.io/aidev/)

{% endraw %}
