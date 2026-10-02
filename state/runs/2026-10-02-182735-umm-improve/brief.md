- 과제: 같은 이름의 부를 둘 이상 제안받으면 그 나눔을 받지 않기 (가치 3 / 위험 1 / 작업량 S)
- 확인된 현행 동작(정찰이 실제로 돌려 봄, 임시 시험은 삭제했고 작업 트리는 깨끗함):
  ```
  go test ./internal/presentation -count=1 -run TestReconDuplicateSectionTitles -v
    duplicate titles: added=2 slides=22
    section slide titles: ["도입" "도입"]
    trim-equal titles: added=2       // {0,"도입"}, {8," 도입 "}
    case-equal titles: added=2       // {0,"Intro"}, {8,"intro"}
  ```
  즉 `flatStory(20)` + `fakeSectioner{{0,"도입"},{8,"도입"}}` 로 `sectionDeck` 을 지나면 지금은 **글자 그대로 같은 이름의 구분 슬라이드 두 장**이 덱에 들어간다. 구현자는 이 세 경우를 그대로 시험으로 옮기면 된다.
- 왜: `internal/presentation/sections.go:120` 의 `usableSections` 는 모델이 돌려준 부 경계를 "전부 아니면 전무" 로 검사하는데(순서 뒤바뀜·범위 밖·너무 촘촘함·제목 아님) **같은 제목이 두 번 오는 것**은 통과시킨다. 그러면 `sectionDeck` 이 제목만 있고 본문이 없는 구분 슬라이드를 글자 그대로 같은 이름으로 두 장 끼워 넣어, 발표 중에 "아까 그 부로 돌아왔나?" 로 보이는 덱이 나온다 — 함수 주석이 스스로 "나누지 않은 쪽이 나은 나눔" 이라고 적어 둔 바로 그 범주다.
- 수용 기준: 1) 제안된 부 중 두 개가 (usableHeading 을 지난 뒤) 같은 제목이면 `usableSections` 가 `nil` 을 돌려주고 `sectionDeck` 이 0 을 돌려주며 `story.Slides` 가 한 장도 늘지 않는다(기존 all-or-nothing 과 같은 처리). 2) 제목이 서로 다른 정상 제안은 지금과 똑같이 통과한다(기존 시험 전부 통과). 3) 시험이 증명할 것: 고치기 전에는 같은 제목 두 장이 덱에 들어간다는 것(= 새 시험이 수정 전 코드에서 실패) · 고친 뒤에는 0 장 · 정상 제안은 영향 없음.
- 건드릴 파일:
  - `internal/presentation/sections.go:usableSections` — 루프 안에서 이미 쓰고 있는 `title := usableHeading(section.Title)` 결과를 작은 `map[string]bool` 에 담아 중복이면 `return nil`. 비교는 `usableHeading` 을 지난 값에 최소 정규화만 더할 것 — 권장은 `strings.ToLower` 하나뿐이다(`usableHeading`(`headings.go:157-176`)이 이미 TrimSpace·따옴표 3종·목록 표시를 떼므로 `" 도입 "` 과 `도입` 은 그 단계에서 이미 같은 값이 된다. 위 확인 출력의 trim-equal 경우가 그 증거다). 거절 이유를 주석 한두 줄로 적을 것(주석이 결정 기록인 저장소 관례).
  - `internal/presentation/sections_test.go` — 기존 `fakeSectioner`·`flatStory(20)` 하네스를 그대로 써서 시험 1~2개 추가: ① 같은 제목 두 개 → `sectionDeck` 이 0, 슬라이드 수 불변 ② 대소문자만 다른 같은 제목(정규화를 넣었다면) 또는 세 부 중 두 부가 겹침 → 0. 정상 경로 시험은 이미 있으니 새로 쓰지 말 것.
  - 그 밖의 파일은 건드리지 않는다(프로덕션 파일 1개).
- 검증 명령:
  - `go test ./internal/presentation -count=1 -run TestSection -v` (DB 불필요, 정찰 실행 0.03초대)
  - `go test ./internal/presentation -count=1` 전체
  - `go vet ./internal/presentation` · `gofmt -l internal/presentation`
  - 마무리: `POSTGRES_DSN=<격리 PostgreSQL 17 DSN> go test -p 1 ./... -count=1` (DSN 없으면 HTTP 통합 시험이 SKIP 되니 `-v` 로 SKIP 여부를 확인하고 노트에 적을 것). Go 만 건드리므로 web 쪽 명령은 불필요.
  - 수정 전 실패 재현을 반드시 남길 것: 새 시험을 먼저 쓰고 수정 없이 돌려 "2 sections inserted, want 0" 류의 실패 출력을 회차 노트에 붙인 뒤, 고친 다음 `git show HEAD:internal/presentation/sections.go` 로 프로덕션 파일만 되돌려 같은 실패가 다시 나는지 확인.
- 위험과 피할 것:
  - `sectionDeck` 의 삽입 루프(원본 리스트를 걸으며 만드는 부분)와 `SlideSources`/`story.Sections` 계산은 건드리지 말 것 — 인덱스 계산이 여기에 묶여 있고, 이번 변경은 **검사만** 추가하는 것이다.
  - `usableHeading`(`headings.go:157`)을 고치지 말 것 — 그룹 머리글도 같은 함수를 쓰므로 범위가 번진다.
  - 정규화를 과하게 넣지 말 것(유니코드 정규화·공백 접기·동의어). 지나치면 정상적인 서로 다른 제목까지 거절해 부가 사라지고, 그건 조용한 기능 퇴행이다. 대소문자 접기까지가 상한.
  - 보호 경로(auth/migrations/.github/workflows)와 무관한 과제다 — 그쪽은 열지 말 것. 마이그레이션 없음, 버전 올리지 않음(릴리스는 별도 회차).
  - 손으로 만든 대역 주의사항: `fakeSectioner` 는 **모델 경계**의 대역이고 기존 시험이 이미 쓰는 것이다(프로덕션의 실제 구현은 `GatewaySectioner` 로 AI 게이트웨이가 필요). 우리 쪽 배선(`sectionDeck`·`usableSections`)은 실제 함수를 그대로 지나야 하며, `usableSections` 만 직접 호출해 끝내지 말고 `sectionDeck`+실제 `Storyline` 으로 슬라이드 수까지 단언할 것.
- 차선 후보: 캔버스의 공간 이름 대체값이 한국어 UI 에 영어 `'My Space'` 로 뜬다 — `web/src/pages/CanvasPage.tsx:2130` 의 `spaces.find(...)?.name || 'My Space'` 가 머리글(:2276)·내려받기 이름(:1966, :1990, :2024)·자식 컴포넌트(:3524)로 그대로 흐르고, 공간 목록이 아직 안 온 첫 렌더에서도 보인다. `t()` 로 바꾸고 i18n 키를 더하는 S 짜리지만 증명이 Playwright 쪽이라 더 무겁다.
