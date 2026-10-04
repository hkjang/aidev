# 수리 요약 — moyro PR #33 (CI verify 실패)

- 문제: `TestSettingsUpdatesAreSerializedThroughActivation`(native_settings_test.go:86)이 `completed updates = 1, want 2` 로 실패. 이 PR 의 diff(final.go·request_body.go·새 DB 테스트)와 무관한 **기존 테스트 자체의 경합**이다 — 두 번째 goroutine 이 `completed.Add(1)` **전에** `close(secondEntered)` 를 해서, 본문 goroutine 이 카운터를 올리기 전에 메인이 단언을 읽을 수 있었다.
- 재현: `GOMAXPROCS=2 go test ./internal/httpapi -run TestSettingsUpdatesAreSerializedThroughActivation -count=1500 -cpu=2` + CPU 부하 40개 → CI 와 **같은 줄·같은 메시지**로 실패(4코어 race 러너와 같은 조건).
- 수정: 두 번째 goroutine 에서 `completed.Add(1)` 를 `close(secondEntered)` 앞으로 옮겼다(첫 goroutine 과 같은 순서). 단언은 하나도 느슨해지지 않았다 — 25ms 조기진입 금지 검사, 1초 재개 검사, `completed == 2` 모두 그대로이고 `Add` 는 여전히 락 획득 **뒤**다.
- 검증: 같은 부하 조건 1500회 + `-race` 400회 통과, CI 와 동일한 `go test -race ./...`·`go vet ./...` 전체 통과, `check-source-sizes.sh` 통과, gofmt clean. 실제 DB(55433)로 PR 의 새 테스트 8/8 통과(4.22s).
- PR 의 원래 변경은 한 줄도 건드리지 않았다. 커밋 1개(52a3812), push 없음.
