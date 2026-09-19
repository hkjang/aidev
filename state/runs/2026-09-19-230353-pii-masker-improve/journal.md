# 회차 노트 2026-09-19-230353-pii-masker-improve — pii-masker
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 23:04] base pinned — main@25c25a8
- [러너 23:04] autonomy release — 

## 정찰 노트
- 선택 이유: 2026-09-17의 "고아 디렉터리 삭제" 변경으로 잘린 `job.json`이 곧 job 전체(결과 파일 포함) 삭제가 되어, 이전 회차가 스스로 남긴 후속 과제(원자적 쓰기)의 가치가 올라갔다. `persistLocked` 한 함수 + 테스트 3개로 끝나는 S 규모라 HEAD 허용(가치 2)·config 테스트(순수 테스트 추가)를 제쳤다. 동기 대기열 메모리 상한·job 재개는 M/위험 3이라 제외.
- 확신 없는 곳: rename 대상이 디렉터리일 때 기존 `TestCreateRemovesTheDirectoryWhenTheRecordCannotBePersisted`가 그대로 통과한다는 것은 Linux EISDIR 동작에 근거한 추론이며 직접 실행해 보지 않았다(구현자가 먼저 돌려 볼 것). `internal/app` 통합 테스트가 job 디렉터리 항목 수를 세는지는 미확인.
- 구현자 주의: fsync·백업 파일·재시도 같은 "보험" 코드는 넣지 말 것(효과 없는 변경 금지 지침). 판별력 있는 테스트는 `job.json.tmp` 자리에 디렉터리를 두어 tmp 쓰기를 실패시키고 이전 `job.json`이 남는지 보는 것 — 이전 구현은 이 경우 `job.json`을 그대로 덮어써 실패한다. `runJob`의 `_ = Save` 무시는 별도 아이디어로 남겼으니 이번에 같이 고치지 말 것.
- 검증 기준선: `go test -count=1 ./...` 전 패키지 통과(약 2초), 작업 트리 clean, main@25c25a8.
- [러너 23:08] scout done — `job.json`을 임시 파일 + `os.Rename`으로 원자적으로 기록 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- 무엇/왜: `internal/jobs/store.go:persistLocked`가 `job.json`을 제자리 덮어쓰기 대신 `job.json.tmp`(상수 `recordTempName`)에 쓴 뒤 `os.Rename`으로 교체하도록 바꿈(rename 실패 시 tmp best-effort 제거). 2026-09-17 이후 잘린 기록은 결과 파일까지 디렉터리째 삭제되므로, 완료 직전 크래시 job이 404로 사라지는 대신 직전 `running` 기록이 남아 `job_interrupted`로 표시되게 함. 커밋 d6d93af, 변경 파일: store.go, store_test.go, README.md 한 문장.
- 검증: `gofmt -l`(무출력)·`go vet`·`go build`·`go test -count=1 ./...`·`go test -race -count=3 ./internal/{jobs,service,app}/...` 전부 통과. 이전 구현으로 임시 되돌려 `TestSaveKeepsThePreviousRecordWhenTheWriteFails`·`TestLoadIgnoresALeftoverTemporaryRecord`가 실제로 실패함을 확인(`TestSaveLeavesNoTemporaryFileBehind`는 예상대로 이전 구현에서도 통과 — 판별력은 나머지 둘에 있음). 정찰이 확신 없다던 기존 `TestCreateRemovesTheDirectoryWhenTheRecordCannotBePersisted`는 수정 없이 통과(rename 대상이 디렉터리 → EISDIR). `internal/httpapi` 통합 테스트 2곳이 job 디렉터리를 `os.ReadDir`하지만 항목 수를 세지 않고 이름만 검사·접두어 필터하므로 영향 없음(열람으로 확인).
- 확신 없는 곳·검증 못 한 것: Windows에서의 테스트 통과 여부(WSL Linux에서만 실행). `os.Rename`이 대상 `job.json`을 다른 프로세스가 열고 있으면 Windows에서 공유 위반으로 실패할 수 있으나 런타임에 `job.json`을 읽는 곳은 기동 시 `load()`뿐이라 실제 충돌 경로는 없다고 판단 — 실행으로 증명한 것은 아님.
- 일부러 하지 않은 것: fsync·백업 파일·재시도 루프(과제서·운영자 지침대로 프로세스 크래시 원자성만 목표, 전원 차단 내구성은 범위 밖 — 주석·README도 그렇게만 적음). `runJob`의 `_ = s.jobStore.Save` 무시는 별도 아이디어로 남김. `readJobRecord`·`firstExistingFile`·`load()`의 고아 판정 로직은 건드리지 않았고 주석만 보강.
- 다음 역할이 조심할 것: 세 번째 테스트는 `completed` job으로 심어 `load()`가 스스로 persist하지 않게 했으므로, 잔재 tmp를 지우는 주체가 이어지는 `Save`임이 분명함(`running`으로 심으면 `load()`의 interrupted 전환 persist가 먼저 tmp를 덮어써 Save 단계의 판별력이 사라짐). 테스트는 DB·네트워크 없이 `t.TempDir()`만으로 돈다.
- [러너 23:11] brief accepted — 채택 — 과제서의 근거(`persistLocked` 제자리 쓰기 + 2026-09-17 이후 잘린 기록은 디렉터리째 삭제)가 코드와 정확히 일치했고
- [러너 23:11] verify passed — 검증 3개 통과 (policy)

## 비평 노트
- 판정 approve / risk low / blocking 없음: diff·커밋·문서·테스트·잠금·실패·재시작·삭제 경로 확인, 실제 회귀나 범위 이탈 없음.
- build·vet·전체 test·전체 race -count=3 통과, gofmt·diff --check 정상; 저장소 코드 수정 없음.
- Windows 구현은 Go 소스로 확인했으나 실행 미검증; 비Unix rename 원자성은 보장되지 않으므로 README의 보장을 일반화하지 말 것. Go 1.25/Docker·강제 종료·전원 차단 실험도 미수행.
- 새 테스트의 실제 호출·단언과 이전 코드 차이를 확인; 이전 구현 복원 실험은 재실행하지 않음. 정상 저장 테스트는 보완 검증이고 나머지 둘에 판별력이 있음.
- [러너 23:13] review approved — 리뷰 승인 (risk=low)
- [러너 23:13] pr created — https://github.com/hkjang/pii-masker/pull/20
- [러너 23:13] ci passed — 검사 없음 — 정책으로 허용
- [러너 23:13] merge done — d6d93af
- [러너 23:16] release published — v1.0.23
- [러너 23:16] gh-release created — GitHub Release v1.0.23
- [러너 23:16] manifest ok — pii-masker-image.tar.gz 
- [러너 23:16] assets uploaded — 1개
- [러너 23:16] assets verified — v1.0.23 자산 1개 (이전 v1.0.22: 1)
