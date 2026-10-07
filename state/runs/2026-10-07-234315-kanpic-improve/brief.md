- 과제: 분수 서식의 분모 탐색 상한을 좁혀, 자리 기호 일곱 개 이상인 서식이 칸을 그릴 때마다 최대 10억 번 돌지 않게 한다 (가치 3 / 위험 2 / 작업량 S)
- 왜: `internal/formula/format_fraction.go:parseFractionFormat` 이 분모 자리 기호 개수를 **9 까지** 받아(`if places > 9 { places = 9 }` → `maxDenominator = 999,999,999`), `bestFraction` 이 분모를 1 부터 그 수까지 **하나씩** 재므로 `?????????/?????????` 가 붙은 칸 하나에 10억 번 반복이 걸린다. 거울인 `web/src/lib/cellFormat.ts` 도 같은 규칙(`maxDenominator=10**Math.min(places,9)-1`, `for(let candidate=2;candidate<=spec.maxDenominator;candidate+=1)`)이라 브라우저 메인 스레드에서 그리드가 멎고, 서버 `TEXT()`·XLSX 가져온 서식도 같은 값을 그릴 때마다 같은 비용을 낸다. 상한을 좁히면 비용이 1000배 줄고, `bestFraction` 이 "1e-12 보다 더 나아져야 갈아탄다" 는 규칙을 쓰므로 실제 값에서 결과가 바뀌는 자리는 거의 없다.
- 수용 기준:
  1) `parseFractionFormat` 의 자리 기호 상한이 Go·TS 양쪽에서 같은 수(권장 **6**, `maxDenominator = 999,999`)로 내려가고, 왜 그 수인지("배정밀도 값에서 1e-12 안으로 들어오는 분수는 여섯 자리 분모 안에 거의 다 있고, 그보다 넓히면 칸 하나가 그리드를 멈춘다")가 한국어 주석으로 남는다.
  2) `testdata/cell-formats.json` 의 분수 사례(4433~4508행, `# ?/?`·`# ??/??`·`# ???/???`·`0 ?/?`·`#,##0 ?/?`·`# ?/?" 개"`)와 `internal/formula/cell_formats_test.go:TestFractionFormatsFollowExcel` 의 18행이 **한 줄도 바뀌지 않고** 그대로 통과한다. 픽스처는 고치지 않는다 — 이것이 "출력이 안 바뀌었다" 의 증거다.
  3) 테스트가 (a) 수정 전에는 `?????????/?????????` 한 번 그리기가 수 초를 먹는 것을, (b) 수정 후에는 같은 서식이 `??????/??????` 와 같은 글자를 내면서 짧은 시간 안에 끝나는 것을 못 박는다. 측정은 `time.Since` 로 상한을 두는 쪽이 `testing.B` 보다 CI 에서 흔들리지 않는다(상한은 넉넉히 — 1초 정도).
  4) 웹 쪽도 같은 상한을 받았음을 `web/src/lib/cellFormat.test.ts` 의 한 사례로 못 박는다(`formatCellValue(1/3,{number_format:'?????????/?????????'})` 가 여섯 자리 상한의 결과와 같다).
- 건드릴 파일 (프로덕션 2개):
  - `internal/formula/format_fraction.go:parseFractionFormat` — `places > 9` 상한 한 줄과 주석. `bestFraction`·`renderFraction`·`fractionText` 의 셈은 건드리지 않는다.
  - `web/src/lib/cellFormat.ts:parseFractionFormat` (279행 `maxDenominator=10**Math.min(places,9)-1`) — 같은 수로. `renderFraction`(295~)·루프(320행)는 그대로.
  - `internal/formula/cell_formats_test.go` — `TestFractionFormatsFollowExcel` 아래에 새 테스트 하나(상한·시간). 기존 18행은 그대로 둔다.
  - `web/src/lib/cellFormat.test.ts` — 사례 한 개 추가.
- 검증 명령:
  - `go test ./internal/formula -run TestFraction -v` (빨강 → 초록을 먼저 보일 것)
  - `go test ./internal/formula -count=1` (이번 세션 기준 `ok 0.141s`)
  - `go test ./... && go vet ./... && go build ./... && gofmt -l ./cmd ./internal ./pkg` (gofmt 는 출력이 없어야 한다)
  - `cd web && npm ci && npm test -- cellFormat` (설치에 시간이 걸린다. `cellFormats.fixture.test.ts` 가 `../testdata/cell-formats.json` 을 읽으므로 `web` 에서 돌려야 한다)
  - `./scripts/check-release-docs.sh`, `./scripts/check-commit-identities.sh HEAD`
- 위험과 피할 것:
  - **Go 와 TS 를 반드시 같이, 같은 수로.** 한쪽만 고치면 격자와 서버가 같은 칸을 다르게 그려 `cellFormats.fixture.test.ts` 가 걸린다. 두 파서를 **통합하려 하지 말 것** — 거울로 두는 것이 이 저장소의 관례다(두 파일 머리의 주석이 그렇게 적어 두었다).
  - 이것은 의도한 **좁히기**다. `bestFraction` 의 근사 알고리즘을 연분수·Stern–Brocot 로 바꾸려 하지 말 것 — 같은 세션에 끝나지 않고, "같은 만큼 어긋나면 분모가 작은 쪽" 과 `bestError-1e-12` 여유의 동치 증명이 필요해진다.
  - `maxFractionValue`(1e15, `functions_text.go:215` 에서 쓴다)·`spec.denominator` 못 박은 경로(`?/8`)·`renderFraction` 의 올림·부호 처리는 손대지 않는다.
  - `migrations`·`.github/workflows`·`internal/auth`·`apikey`·`handoff` 는 이 과제와 무관하다.
  - 상한을 6 보다 더 내리면(예: 4) `?????/?????` 같은 서식의 출력이 눈에 보이게 바뀐다. 6 을 권하는 근거는 위 1)의 주석 문구다 — 더 내리고 싶으면 사례로 증명할 것.
- 미확인(추정으로 적은 것):
  - "10억 번 반복이 수 초" 는 **실행으로 재지 않았다**(코드를 바꾸지 않는 정찰이라 타이밍 테스트를 넣지 못했다). 자리 3개(999 회)가 기존 테스트에서 즉시 끝나는 것만 확인했다. 구현자는 먼저 (a) 의 빨강으로 실제 시간을 재서 과제의 전제를 확인하고, 수 초가 아니라 수십 밀리초라면 상한 숫자를 다시 정하거나 차선 후보로 넘어갈 것.
  - 7~9 자리 서식에서 출력이 실제로 바뀌는 값이 있는지 전수로 보지 않았다. 바뀌는 사례를 찾으면 주석에 적고, 바뀌어도 되는 좁히기라는 점을 커밋 메시지에 남길 것.
- 차선 후보: cron step 자리가 부호를 받아들인다 (`*/+2`) — `internal/automation/schedule.go:parseCronField` 의 `strconv.Atoi(stepRaw)` 는 2026-10-06 수정(값 자리 `isDecimalDigits`) 범위 밖이라 그대로다. 같은 `isDecimalDigits` 를 step 자리에도 쓰면 되고, 빨강으로 `*/+2 0 * * *` 가 지금 통과하는지 먼저 확인할 것. 다만 `internal/automation` 은 10-02·10-03·10-06 세 회차가 이미 고른 패키지라 회귀 원인 구분이 어려워진다 — 1순위가 성립하면 쓰지 말 것.
