# 과제서 — 2026-10-04-011745-umm-improve (umm)

- 과제: 모델이 번호나 따옴표를 붙여 보낸 제목이 그대로 슬라이드 머리글에 오른다 — 서식만 떼고 라벨을 남기기 (가치 3 / 위험 1 / 작업량 S)

- 왜: `internal/presentation/headings.go:157` `usableHeading` 의 문서 주석은 "a quoted string, a numbered list item" 을 머리글이 아닌 것으로 못 박았고 같은 패키지 시험(`headings_test.go:148` "Quoting and list markers are formatting, not the label.")도 그 계약을 말하는데, 실제 코드는 **따옴표 제거 → 글머리표 제거** 순서로 한 번만 지나가고 번호 목록 표시는 아예 보지 않습니다. 그래서 `- "회고 주기 단축"` 은 `strings.Trim` 이 뒤 따옴표만 먹고 앞 따옴표를 남겨 `"회고 주기 단축` (짝이 안 맞는 따옴표)이 되고, `1. 도입` 은 번호를 이고 그대로 슬라이드 제목이 됩니다. 고치면 모델이 서식을 붙여 답해도 사람 발표 화면에 들어가는 문자열이 사람이 읽을 라벨 하나로 남습니다 — 이 패키지가 "폴리시가 폴리시하던 것을 깨면 안 된다" 고 적어 둔 바로 그 지점입니다.

- 수용 기준:
  1) `usableHeading("- \"회고 주기 단축\"")` 과 `usableHeading("\"- 회고 주기 단축\"")` 가 둘 다 `회고 주기 단축` 을 돌려준다(짝 안 맞는 따옴표·글머리표가 남지 않음).
  2) `usableHeading("1. 도입")` · `usableHeading("2) 도입")` 가 `도입` 을 돌려준다. 반면 번호가 라벨의 내용인 경우는 **건드리지 않는다**: `2026년 계획`, `3.0 릴리스`, `1월 회고` 는 그대로 돌아온다.
  3) 시험은 `usableHeading` 만 부르지 말고 **두 호출 경로가 같은 입력을 같은 값으로 읽는지**를 증명한다 — ① 실제 `Compile`+`nameGroups`(기존 `headings_test.go` 의 Namer 대역 = 모델 경계)로 그룹 슬라이드 `Title` 이 서식 없는 라벨이 되는 것, ② 실제 `sectionDeck`+`Storyline`(기존 `sections_test.go` 의 `fakeSectioner`)로 부 머리글이 서식 없는 라벨이 되고 **나눔이 거절되지 않는 것**(슬라이드 수까지 단언).
  4) 기존 `TestUnusableAnswersAreRefused` · `TestPartsWithDifferentNamesAreStillAccepted` · `go test ./internal/presentation` 전체가 그대로 통과한다.

- 건드릴 파일 (프로덕션 1개):
  - `internal/presentation/headings.go:157` `usableHeading` — 서식 제거를 **거절이 아니라 벗기기**로 고친다. 구체적으로: (a) 따옴표 Trim → 글머리표 제거 → **따옴표 Trim 한 번 더**(순서 때문에 생기는 짝 안 맞는 따옴표를 없애는 유일한 지점, 무한 반복 금지 — 고정 2패스), (b) 선두 번호 표시 `^\d{1,2}[.)]` + **공백 1개 이상**만 제거하고, 제거 결과가 빈 문자열이면 **제거 전 값을 쓴다**. 길이 상한(`maxHeadingRunes` 24)·개행 거절·`""` 반환 계약은 그대로 둔다.
  - `internal/presentation/headings_test.go` — 위 2)·3) 케이스 추가. 기존 단언은 수정하지 말 것.
  - (선택) `internal/presentation/sections_test.go` — 3)② 를 여기에 두는 편이 자연스러우면 여기.
  - 주석에 "왜 2패스인가 / 왜 거절이 아니라 벗기기인가" 를 적는다(주석이 결정 기록인 저장소 관례).

- 검증 명령 (이 저장소에서 실제로 도는 것, 정찰에서 baseline PASS 확인):
  - `go test ./internal/presentation -count=1 -v`  ← 정찰 실행 결과 `ok … 0.042s`. DB 불필요. `-v` 로 새 시험이 SKIP 아님을 확인할 것.
  - `go vet ./...` · `gofmt -l internal/presentation`(무출력)
  - `POSTGRES_DSN=<격리 PostgreSQL 17 DSN> go test -p 1 ./... -count=1` — DSN 이 있으면 전체. **DSN 없이 돌면 HTTP/DB 통합은 SKIP 되므로 "전체 통과" 라고 쓰지 말 것.**
  - 고친 뒤 `git show HEAD:internal/presentation/headings.go` 로 프로덕션 파일만 되돌려 새 시험이 같은 실패를 다시 내는지 확인(이 저장소의 관례).

- 위험과 피할 것:
  - **거절(`return ""`)을 늘리지 말 것.** `sections.go:142-145` 는 제목 하나가 `""` 면 `return nil` 로 **나눔 전체**를 버립니다 — 번호 붙은 제목을 거절하도록 고치면 부 나누기가 조용히 사라집니다. 이번 과제는 벗기기만 합니다.
  - `usableHeading` 은 `nameGroups`(그룹 머리글)와 `usableSections`(부 머리글) **공용**입니다. 한쪽만 보고 고치지 말고 두 경로를 같은 입력으로 함께 확인하세요(운영자 규칙 3번).
  - 2026-10-02 회차가 넣은 중복 부 제목 거절(`sections.go`)은 **건드리지 마세요**. 서식을 벗기면 `1. 도입`·`2. 도입` 이 둘 다 `도입` 이 되어 그 거절에 걸릴 수 있는데, 그것이 옳은 동작입니다(사람 눈에 같은 부). 기존 시험이 그대로 통과하는지만 확인하세요.
  - 정규화를 더 접지 말 것(유니코드 정규화·공백 접기·대소문자). 2026-10-02 회차가 `ToLower` 한 겹을 상한으로 정했고, 그 이상은 서로 다른 제목을 같게 만듭니다.
  - `migrations/`·`internal/auth/`·`.github/workflows/`·`Dockerfile`·루트 `package.json` 은 손대지 마세요(이번 과제와 무관, 최근 3회차가 전부 빌드 경로였음).
  - 미확인: 실패 문구는 정찰이 코드를 읽고 예측한 것이며 실행으로 확인하지 않았습니다(코드를 바꿀 수 없는 역할). `strings.Trim(label, "\"'“”‘’`")` 가 `- "x"` 의 **뒤** 따옴표만 먹는다는 추론은 `strings.Trim` 의 양끝 cutset 의미에서 나온 것입니다 — 구현자는 먼저 그 한 줄을 시험으로 찍어 확인하고, 틀렸으면 과제서를 고쳐 적으세요.

- 차선 후보: `make test-go` 를 CI 와 같은 `-p 1` 직렬 실행으로 맞추기 (가치 3 / 위험 1 / S) — `Makefile:8` 이 `go test ./...`, `ci.yml` 은 `app_settings` 경합 때문에 `-p 1`. 착수한다면 DSN 을 붙여 `-p 1` 없이 3회 돌려 경합 출력을 먼저 증거로 붙일 것(DSN 없이는 DB 통합이 전부 SKIP 되어 경합이 아예 없습니다). 아홉 회차 연속 차선으로만 남은 항목이라 1순위가 성립하면 건드리지 마세요.
