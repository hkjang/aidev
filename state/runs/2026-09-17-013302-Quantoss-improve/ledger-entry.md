## 2026-09-17
- 선택: `notify.Notifier` 리터럴 생성 시 nil http 패닉 방지(지연 초기화) + `internal/journal`·`internal/notify` 첫 단위 테스트 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 두 패키지 모두 `_test.go` 가 0건이었고, `notify.Send` 는 백그라운드 고루틴에서 `n.http.PostForm` 을 부르므로 `New()` 를 거치지 않은 `Notifier` 는 첫 알림에서 프로세스를 통째로 죽였다(현재 호출부 3곳은 모두 `New()` 를 쓰지만 방어가 없었다 — `toss.Client` limiter nil 패닉과 같은 계열). `client()` 로 `sync.Once` 지연 초기화하고, 전송 본체를 동기 `send()` 로 분리하고 텔레그램 API 베이스를 비공개 필드로 빼서 `httptest` 로 검증 가능하게 했다(기본값·운영 경로 동작 불변). 테스트 작성 중 `journal.Load` 가 "읽기 전용" 이라며 `Path`·`EquityPath` 만 비우고 `FlowsPath` 는 남겨 두어, 읽어 온 저널에 `RecordFlow` 하면 원본 파일에 덧붙여지는 불일치를 발견해 함께 비웠다(호출부 중 Load→RecordFlow 조합은 없음). 신규 테스트 12개: Summary(승률·PF·평균 손익·MDD·무손실 시 +Inf), KeyTag/Key, Record·Mark(1분 1회 기록)·RecordFlow → Load 전체/날짜 필터 왕복, 손상 줄·파일 없음, LastErr/Log.Write, ReturnPct; Notifier Enabled 5케이스, 텔레그램 경로·폼, 슬랙 JSON·`**`→`*` 변환, 리터럴 무패닉, 4xx·연결 실패 무시. 두 수정을 각각 되돌리면 리터럴 테스트가 실제 nil 패닉으로, 왕복 테스트가 "read-only" 단언으로 실패함을 확인. `gofmt -l`(clean)·`go vet ./...`·`go build ./...`·`go test -count=1 ./...`(신규 2패키지는 `-race` 도) 전부 통과. 커밋 39d7a120.
- 보류 아이디어:
  - 스윙 실행기 마지막 봉 판정 — 재평가: 진입은 이제 "신호일 다음 거래일에만 체결, 놓치면 만료"(runner.go:274) 로 설계가 바뀌어 PR #13 의 지연 체결 문제는 해소됨. 청산(runner.go:339~)만 여전히 `last[p.Symbol]` 한 봉으로 판정 — 데몬이 하루 통째로 죽으면 그날의 손절·목표를 놓친다. 범위를 "진입 다음 봉부터 순회" 로 좁혀서 할 것. runner.go 가 최근 5회 재작성돼 충돌 위험 (가치 3 / 위험 3 / M)
  - GitHub Actions CI 없음 — 푸시 토큰 `workflow` 스코프 미확인이라 회차 전체가 날아갈 수 있음. `scripts/check.sh`(gofmt/vet/build/test) 로컬 스크립트부터 (가치 4 / 위험 3 / S)
  - `Config.Validate` 가 gap_reclaim 파라미터(GapMin/GapMax/GapPullMin/GapVolMult, config.go:517~)를 검사하지 않음 — `GapMin >= GapMax` 면 신호가 영영 안 나오는데 조용히 통과 (가치 3 / 위험 1 / S)
  - `swing.Runner` 동시 보유 한도 `MaxPositions*5`(runner.go:461), 예약 상한 `*3`(:528) — 배수 근거가 없고 서로 다름. 의도 확정 후 테스트로 고정 (가치 3 / 위험 2 / S)
  - `journal.Summary`·`swing.Evaluate`·`report/weekly`·`main.go:459` 가 PnL==0 을 패배로 집계 — 4곳 함께 (가치 2 / 위험 1 / S)
