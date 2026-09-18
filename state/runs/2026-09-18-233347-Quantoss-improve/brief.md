# 과제서 (2026-09-18, 수정 과제 — 오류 대응)

## 상황 판단 (정찰 결과)
- 직전 회차 종료 사유는 `hold: budget` — 러너 예산 소진. 이 세션에서는 이전 회차의 로그·원장을 읽을 권한이 없어 어느 단계에서 얼마나 썼는지는 **미확인**.
- 저장소에는 `.github/workflows/` 가 **없다**(확인). "릴리즈 워크플로" 는 러너의 검증 단계(`gofmt -l` / `go vet ./...` / `go build ./...` / `go test -count=1 ./...`)를 뜻한다.
- 검증 경로에서 재현되는 유일한 이상: **`go test ./...` 가 약 3분 걸리며, 그 중 172.6초가 `internal/zz_dbg` 한 패키지**다(이 세션에서 실측: 나머지 25개 패키지 합계 < 5초).
- `internal/zz_dbg/main_test.go`(55줄, 커밋 3da3780c 에 대형 기능 커밋과 함께 섞여 들어옴)는 테스트가 아니라 디버그 스크래치다:
  - `config.LoadDotEnv("/mnt/c/Users/USER/projects/Quantoss/.env")` — 운영자 PC 절대 경로 하드코딩
  - `toss.NewClient(...)` + `c.TokenCache = "/mnt/c/Users/USER/projects/Quantoss/data/.token.json"` — **실거래 토큰 캐시 파일을 그대로 씀**. README:41 "client 당 유효 토큰 1개, 두 프로세스가 같은 키를 쓰면 서로 토큰 무효화" 에 따라 라이브 에이전트 실행 중 `go test ./...` 를 돌리면 에이전트 토큰을 깨뜨릴 수 있다(충돌 실제 발생 여부는 미확인, 경로가 같다는 것만 확인).
  - `universe.FetchDaily`(토스 REST), `catalyst.DARTList`(DART API 200일치) — 외부 네트워크 호출, 단언(`t.Error`/`if`) 0건, 전부 `fmt.Println`.
  - `.env` 가 없는 환경(CI·다른 머신)에서는 `config.FromEnv()` 가 키 없이 어떻게 되는지 미확인 — 구현자가 확인할 것.
- 러너가 검증·수리 루프에서 `go test ./...` 를 여러 번 돌리면 매번 3분 + 외부 API 응답을 기다리므로 회차 시간·예산을 갉아먹는다. 이것이 예산 소진의 원인이라고 단정할 수는 없지만(미확인), 검증 단계에서 고칠 수 있는 유일한 실체적 결함이며 워크플로를 느슨하게 하는 것이 아니라 **검증이 아닌 것을 검증 경로에서 빼는 것**이다.

## 과제
- 과제: `internal/zz_dbg` 디버그 스크래치를 `go test ./...` 경로에서 분리 (가치 4 / 위험 1 / 작업량 S)
- 왜: `go test ./...` 의 98% 시간이 단언 없는 네트워크 디버그 스크립트에 쓰이고, 그 스크립트가 실거래 토큰 캐시를 공유해 라이브 에이전트를 깨뜨릴 수 있다. 빼면 전체 검증이 5초 안에 끝나고 외부 서비스·운영자 PC 경로 의존이 검증 경로에서 사라진다.
- 수용 기준:
  1) `go test -count=1 ./...` 가 `internal/zz_dbg` 를 실행하지 않고 전체가 10초 안에 끝난다(`time go test -count=1 ./...` 로 확인, 결과를 원장에 기록).
  2) 스크래치 코드는 운영자가 쓰던 용도로 남긴다: 파일 맨 위에 `//go:build zzdbg` 태그를 붙여 `go test -tags zzdbg ./internal/zz_dbg` 로만 돈다(삭제하지 말 것 — 운영자가 직접 넣은 파일). 파일 상단 주석에 "디버그 스크래치, 실거래 토큰 캐시를 공유하므로 에이전트 실행 중 돌리지 말 것" 한 줄.
  3) 태그 없이도 `go vet ./...`·`go build ./...` 가 통과하고, 태그를 붙이면 컴파일이 되는지(`go vet -tags zzdbg ./internal/zz_dbg`) 확인. 디렉터리에 빌드 대상 파일이 하나도 없을 때 `go test ./...` 가 경고("build constraints exclude all Go files")를 내는지 확인하고, 낸다면 `doc.go`(패키지 선언 + 한 줄 주석, 태그 없음)를 추가해 조용히 통과시킨다.
  4) README `## 빌드` 절(README.md:33~37) 바로 아래에 한 줄: `go test -tags zzdbg ./internal/zz_dbg` 는 실API·`.env` 를 쓰는 디버그 스크래치라 기본 테스트에서 제외됨.
  5) 원장에는 '수정 과제' 로 기록하고, 실측한 전/후 `go test` 소요 시간(전: ~175s, 후: 실측값)을 적는다.
- 건드릴 파일:
  - `internal/zz_dbg/main_test.go` — 1행에 `//go:build zzdbg` + 빈 줄, 상단 설명 주석. 본문 로직은 손대지 않음.
  - `internal/zz_dbg/doc.go` — 3) 에서 필요할 때만 신규.
  - `README.md:33~37` — 한 줄 추가.
- 검증 명령(이 저장소에서 실제로 도는 것, 이 순서로):
  - `gofmt -l .` (출력 없어야 함)
  - `go vet ./...`
  - `go build ./...`
  - `time go test -count=1 ./...` (10초 이내, zz_dbg 미실행)
  - `go vet -tags zzdbg ./internal/zz_dbg` (태그 경로가 여전히 컴파일되는지 — **실행하지 말 것**, 실토큰을 건드린다)
- 위험과 피할 것:
  - `main_test.go` 의 로직·경로를 "고치려" 하지 말 것(운영자 스크래치, 효과 없는 변경은 반려 사유). 태그와 주석만.
  - 태그 붙인 테스트를 실제로 실행하지 말 것 — 실거래 토큰 캐시(`data/.token.json`)를 공유한다.
  - `.github/workflows` 를 새로 만들지 말 것(푸시 토큰 `workflow` 스코프 미확인 → 회차 전체 유실 위험, 보류 아이디어 기록 참조).
  - 다른 테스트 타임아웃·`-short` 도입 등 검증을 느슨하게 만드는 변경 금지.
  - 이번 회차는 예산이 빠듯하다(정찰 시점 잔여 ≈ $1.2). 전체 테스트를 두 번 이상 돌리지 말 것: 수정 전 실측은 이 과제서의 수치(172.6s)를 쓰고, 수정 후 1회만 측정.
- 차선 후보: `Config.Validate` 가 gap_reclaim 파라미터(GapMin/GapMax/GapPullMin/GapVolMult)를 검사하지 않음 — `0 < GapMin < GapMax`, `GapPullMin >= 0`, `GapVolMult > 0` 검사 + `internal/config` 테스트 (가치 3 / 위험 1 / S). 1순위가 이미 해결돼 있거나(파일에 이미 빌드 태그가 있으면) 성립하지 않을 때만.
