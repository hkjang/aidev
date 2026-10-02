- 과제: 자동 병합 승인이 같은 커밋에 승인을 매 순회마다 다시 발급하는 것을 멈춘다 (가치 4 / 위험 2 / 작업량 S)

- 왜: `AutoApproveMerges`(internal/observer/automerge.go:27)는 `Status=="DONE"` + 공급된 발견 + 게이트 통과만 보고 승인을 발급하는데, 승인을 발급해도 그 세 조건이 하나도 변하지 않는다. `RequestScopedApproval`(internal/store/sqlite/approval.go:91)은 매 호출마다 `NewID("APR")` 로 새 행을 넣고 중복을 전혀 보지 않으므로, 병합이 아직 안 된 작업 하나에 대해 15분 순회마다 `MERGE_BRANCH` 승인이 하나씩 새로 APPROVED 상태로 쌓인다(감사 체인 링크도 발급당 2개씩). 게다가 `TickResult.Acted()`(tick.go:40)가 `len(project.Merged) > 0` 을 활동으로 세므로, 2026-10-01 회차가 `Err` 축에서 복구한 "조용한 스윕" 설계가 `--merge` 를 켠 프로젝트에서 다시 무력화되어 같은 줄이 영원히 찍힌다. CLI 쪽도 `printDecisions("병합", …)` + "`goalforge merge --work-item ID` 로 반영합니다"(cmd/goalforge/standards.go:485-492)를 매번 다시 안내한다.

- 수용 기준:
  1) 아무것도 변하지 않은 상태에서 `AutoApproveMerges` 를 두 번 호출하면, 두 번째 호출의 `decisions.Approved` 에 그 작업이 들어가지 않고 `approvals` 테이블의 그 작업·그 커밋에 대한 `MERGE_BRANCH` 행이 **1개로 유지**된다.
  2) 두 번째 호출이 침묵하지 않는다 — 이미 승인이 있다는 사실이 `decisions.Refused[workID]`(또는 `Detail`)로 사람이 읽을 수 있게 남는다. "계산해 두고 아무도 읽지 않는 값" 을 또 만들지 말 것.
  3) 이미 발급된 승인을 **소비한 뒤**(`ConsumeScopedApproval` → status `CONSUMED`)에는 다시 승인받을 수 있는 경로가 막히지 않는다. 즉 건너뛰기 조건은 `status='APPROVED'`(미소비)에만 걸린다.
  4) 테스트가 증명해야 하는 것: (1) 반복 호출의 멱등성 — 실제 `*store.Store`(SQLite)로 승인 행 수를 직접 센다, (2) 기존 통과 경로가 그대로 통과 — 첫 호출은 여전히 승인한다, (3) `VerifyIntegrity(ctx).Intact()` 가 여전히 참(`TestAnAutomaticMergeApprovalIsChained` 와 같은 방식).

- 건드릴 파일 (프로덕션 2개 + 테스트 2개):
  - `internal/store/sqlite/approval.go` — 새 읽기 메서드 하나. 예: `func (s *Store) ScopedApprovalGranted(ctx, projectID, actionType string, scope ApprovalScope) (bool, error)` — `SELECT 1 FROM approvals WHERE project_id=? AND action_type=? AND status='APPROVED' AND work_item_id=? AND commit_sha=? AND (target_ref='' OR target_ref=?) LIMIT 1`. `ConsumeScopedApproval`(:266)의 정확일치 쿼리와 **같은 조건**을 쓸 것 — 조건이 어긋나면 "발급은 건너뛰는데 소비는 못 찾는" 교착이 생긴다. `ConsumeScopedApproval` 자체는 건드리지 말 것.
  - `internal/observer/automerge.go:108` — `scope` 를 만든 직후, `RequestScopedApproval` 앞에서 `ScopedApprovalGranted` 로 확인하고 이미 있으면 `decisions.Refused[workID] = "이미 승인되어 있습니다 — \`goalforge merge --work-item …\` 로 반영하면 됩니다"` 후 `continue`.
  - `internal/store/sqlite/approval_test.go`(또는 기존 승인 테스트 파일) — 새 메서드의 APPROVED/CONSUMED/다른 커밋 구분.
  - `internal/observer/automerge_test.go` — 멱등성 테스트. 기존 `autonomyFixture`/`verifiedWork`/`mergePolicy`/`suppliedItem` 헬퍼를 그대로 재사용할 것(automerge_test.go 안에 이미 있다). **먼저 실패하는 테스트를 쓰고 빨강을 눈으로 확인한 뒤** 프로덕션을 고칠 것 — 지금 코드에서는 승인 행이 2개가 되어 반드시 실패한다.
  - 끝내고도 시간이 남으면 `automerge.go:168` 의 `var _ = time.Now` 를 정리할지 판단하라(시간 기반 검사를 추가하지 않는다면 `time` import 와 함께 지우는 것이 맞다). 같은 줄이 `tick.go` 끝에도 있으나 **그건 건드리지 말 것** — 범위가 번진다.

- 검증 명령 (이 저장소에서 실제로 도는 것, 이번 회차에 실행 확인):
  - `go test ./internal/observer/ -run TestAutoApproveMerges -count=1 -v` (새 테스트 이름에 맞춰 `-run` 조정)
  - `go test ./internal/observer/ ./internal/store/sqlite/ -count=1` — **이번 회차 기준선: 둘 다 `ok`**(observer 14.4s, store/sqlite 34.7s). 새로 깨지는 것이 있으면 이 변경 탓이다.
  - `gofmt -l ./cmd ./internal`(무출력) / `go vet ./...` / `go build ./...`
  - 마지막에 `go test ./... -count=1`(약 60s, 2026-10-01 기준 전부 통과).

- 위험과 피할 것:
  - `internal/observer/automerge.go` 는 위험 구역이다. 단, 이 변경은 봉투를 **좁히지도 넓히지도 않는다** — 발급 조건은 그대로 두고 "이미 발급된 것을 또 발급하지 않는다" 만 더한다. `admits()`·`gatesPassedOn()`·`AutoApprove` 의 어떤 규칙도 복제하지 말 것("봉투가 무엇을 허용하는지 판단하는 규칙은 `AutoApprove` 한 곳에만" 이 이 저장소의 명시적 결정이다). 멱등성 가드는 봉투 규칙이 아니라 중복 발급 방지이므로 그 결정과 충돌하지 않는다 — 커밋 본문에 이 구분을 적을 것.
  - **범위 밖(이번에 하지 말 것)**: 같은 작업이 새 커밋을 얻었을 때 옛 커밋의 APPROVED 행이 영구히 살아남아 그 커밋이 계속 릴리즈 가능해지는 문제. 실재하는 별개 결함으로 보이지만(미검증) 설계 판단이 필요하고 범위를 M 이상으로 키운다. 아이디어 파일에 남기고 넘어갈 것.
  - **`AutoApproveMerges` 에 `DailyLimit` 을 끼워 넣지 말 것.** 그것도 실재하는 공백이지만 기록 경로(`auto_approvals` 에 병합 승인이 기록되지 않는다)를 먼저 정해야 하는 단독 회차 과제다. 두 개를 한 회차에 담으면 파일 수와 위험이 함께 커진다.
  - 스키마·마이그레이션(`internal/store/sqlite/store.go`)을 건드릴 필요가 없다 — 쿼리는 기존 `approvals` 컬럼만 읽는다. 마이그레이션을 추가하려 하고 있다면 설계가 어긋난 것이다.
  - `.github/workflows/*`, `internal/policy/*`, `internal/gitops/*` 는 이 과제와 무관하다.
  - 샌드박스가 복합 셸 명령(`&&`, `;`, 파이프)과 `gh` 를 승인 없이 막는다 — 명령을 하나씩 쪼개 실행할 것.
  - 커밋 제목은 한국어 한 줄로 "무엇이 달라지는지". 예: `자동 병합 승인이 같은 커밋에 승인을 다시 발급하지 않게 한다`.

- 추정 근거 (S = 한 세션 45분 안): 프로덕션 변경은 쿼리 1개 + 분기 1개로 20줄 안쪽. 비용의 대부분은 테스트와 검증이고, 느린 패키지 두 개(observer 14s + store/sqlite 35s)를 두세 번 돌리는 시간이 실질 하한이다. 범위 25~40분, 가장 그럴듯한 값 30분. 흔들릴 수 있는 가정: 새 메서드 테스트를 둘 중 어느 파일에 둘지(기존 승인 테스트 파일 이름 미확인 — `ls internal/store/sqlite/ | grep approval` 로 먼저 확인할 것). 전체 `go test ./...` 가 예상 밖으로 깨지면 +15분.

- 차선 후보: `internal/model` 의 `ParseCriterion` 계약 테스트 (가치 3 / 위험 1 / 작업량 S) — `internal/model/` 에는 지금도 `model.go` 하나뿐이고 `[no test files]` 다(이번 회차에 `ls` 로 확인). `ParseCriterion`(model.go:98)의 `type=value` / `type@kind=value`(kind 소문자화) / 공백 트림 / `=` 없음·빈 type·빈 value·`@` 한쪽만 빈 경우의 거절을 표 기반 테스트로 고정한다. 프로덕션 코드 변경 없음, 테스트 파일 1개(`internal/model/model_test.go` 신규). `a@b@c=v` 처럼 `@` 가 두 번 나오는 입력의 현재 동작은 의도된 것인지 미확인이므로 단정하지 말 것.
