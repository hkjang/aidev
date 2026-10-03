# 회차 노트 2026-10-04-070319-GoalForge-improve — GoalForge
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:03] base pinned — main@dd6dcb0
- [러너 07:03] autonomy release — 

## 구현 노트
- 무엇을 왜: `gateKindStrength` 가 `review:1` 을 `build:1` 과 같은 칸에 두어 `@build` 조건이 `review` 게이트 통과만으로 MET 이 됐다(반대 방향도). 리뷰는 보통 그 코드를 쓴 세션의 판단이라 런이 자기 작업을 스스로 인증하는 경로다. `security`·`performance` 의 기존 특별 분기를 `offLadderKinds` 집합으로 일반화하고 `review` 를 넣었다. 프로덕션 파일 1개(`internal/policy/gatekind.go`) + 테스트 3개 + `docs/GUIDE.md` 한 줄. 커밋 0eb0e17.
- 확신 없는 곳: ① **행동을 좁히는 변경이다.** `@build`/`@test` 조건을 `review` 게이트로 측정해 온 기존 프로젝트가 있다면 MET → WRONG_KIND 로 바뀐다. 의도한 조임이고 repo 의 방향과 맞지만, 저장소 내 그런 사용례는 못 찾았을 뿐 외부 DB 에 있는지는 알 수 없다(종류 미요구 조건은 전혀 영향 없음 — 기존 테스트가 고정). ② `internal/api/setup.go:177` 이 JSON `required_kind` 를 소문자화 없이 넘기는데 저장 시 `store.go:663` 이 lower 하므로 괜찮다고 판단했으나 API 경로로 직접 끝까지 돌려 보지는 않았다.
- 일부러 안 한 것: `KnownGateKinds` 를 집합 둘 순회로 바꾸면서 종류가 빠질 위험이 생겼는데, 리팩터를 더 하지 않고 일곱 종류 전부를 `ValidGateKind`+`KnownGateKinds` 로 단정하는 테스트를 넣어 닫았다. `internal/observer/tick.go` 가 WRONG_KIND 를 알아차리는지는 확인하지 않았고 ideas.json 에 후보로 남겼다 — 범위를 넘고 파일 수가 늘어서다.
- 다음 역할이 조심할 것: `internal/store/sqlite/proofkind_test.go` 의 새 테스트 둘은 실제 SQLite 파일(`t.TempDir()`)을 만들고 `GoalProgressDetail` 프로덕션 배선을 지난다 — 대역 없음, DB 필요, 패키지 전체는 ~42s. 판정 규칙은 `EvidenceSatisfies` 한 곳에만 있으니 소비처(`diagnostics.kindChecks`, `sqlite.criterionStatus`)에 규칙을 복제하지 말 것. 릴리즈·빌드 경로는 건드리지 않았고 `scripts/build-release.sh` 는 이번에 돌리지 않았다.
- [러너 07:16] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 확인한 것: `main` 의 `gatekind.go` 만 되돌려 신규 테스트 4개를 돌려 전부 빨강임을 직접 봤다(`EvidenceSatisfies("build","review")=true want false`, readiness `proof kind`=LevelOK, 실제 SQLite 에서 `Status:MET RequiredKind:build EvidenceKind:review Satisfied:true`) — `ledger-entry.md:5` 의 출력과 일치하고, 되돌린 파일은 원복해 트리는 깨끗하다. `go build`/`go vet`/`gofmt`/`go test ./...` 전부 통과. 판정: **approve**, risk low, 차단 없음.
- 구현자 우려 ② 해소: `ValidGateKind` 가 내부에서 `ToLower`, `store.go:663`·`gates.go:52` 가 저장 시 정규화, `EvidenceSatisfies` 가 양쪽을 또 `ToLower` 한다 — `setup.go:177` 의 비정규화 전달은 끝까지 안전하다. 우려 ①(행동 좁히기)은 실재하나 의도된 조임이고 순수 함수 변경이라 revert 로 완전히 되돌아온다.
- 커밋이 이름 대지 않은 세 번째 소비 경로도 닫혔다 — `internal/observer/verify.go:79`(review 게이트가 `build_log` 를 주장하던 경로). 방향은 맞지만 그 경로에만 테스트가 없다. `kindForEvidence` 가 review 를 돌려주지 않고 `ExecutedEvidence` 가 앞을 막아 빈 문자열로 새는 구멍은 없음을 확인했다.
- 못 본 것: 외부 기존 프로젝트 DB(저장소 안에는 review 게이트를 객관적 조건에 묶은 생성 경로가 없다), `scripts/build-release.sh`, `internal/observer/tick.go` 가 WRONG_KIND 를 어떻게 다루는지(구현자가 남긴 후보 그대로).
- 승인이어도 남는 우려 — 릴리즈 노트가 알아야 할 것: readiness 의 `proof kind` FAIL 은 런을 막지 않는다(`internal/app/plan.go:192-200` 이 의도적으로 WARN 으로 낮춤). review 게이트를 `@build` 에 묶어 둔 프로젝트는 계속 실행되며 영원히 완료되지 않는다 — 거짓 완료보다 낫지만 경고를 무시하면 예산 소모로 나타난다. 다음 회차 후보: `cmd/goalforge/main.go:3475` 가 일곱 종류를 `KnownGateKinds()` 대신 하드코딩, `readiness.go:154` 가 사다리 밖 security·performance 를 아직 '동작 확인' 으로 센다.
- [러너 07:24] review approved — 리뷰 승인 (risk=low)
- [러너 07:24] pr created — https://github.com/hkjang/goalforge/pull/77
- [러너 07:30] ci passed — 검사 5개 모두 success
- [러너 07:30] merge done — 0eb0e17
