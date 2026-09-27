- 과제: 퍼블리시 게이트가 승인 이력의 `expires_at` 을 저장소·런타임과 같은 트림 규칙으로 읽도록 수정 (가치 3 / 위험 1 / 작업량 S)
- 왜: `internal/dataworks/domain.go:668` `bestApprovalStatus` 는 같은 함수 안에서 `trace.Step`·`trace.Status` 는 `strings.TrimSpace` 후 비교하는데 `trace.ExpiresAt` 만 원문으로 `!= ""` 검사하고 원문을 `time.Parse(time.RFC3339Nano, …)` 에 넘긴다(679~684행). 그래서 공백이 섞인 `" 2030-01-01T00:00:00Z "` 처럼 **먼 미래의 유효한 승인 만료일**을 가진 행은 파싱 실패로 status 가 `expired` 로 덮여 `MissingApprovals`·`BlockedReasons` 에 올라가고(`domain.go:90~95`) 퍼블리시가 영구히 막힌다 — 저장소의 `store.EntitlementActive`(dataworks_operations.go:541)·런타임 `contractScopeActive`(dataworks_runtime.go:317)·액션 센터는 모두 트림 후 파싱한다. 고치면 레거시 승인 행을 가진 상품이 실제 승인 상태대로 게이트를 통과한다.
- 수용 기준:
  1) `bestApprovalStatus` 가 `expires_at` 을 한 번 `TrimSpace` 한 지역 값으로 빈 값 검사와 파싱에 함께 쓴다. 트림 후 비어 있으면 만료 없음(종전과 같이 status 유지), 트림 후에도 해석 불가면 종전대로 `expired`(닫히는 방향 유지).
  2) 공백이 든 미래 만료일을 가진 `approved` 승인 행이 `EvaluatePublishGate`·`EvaluatePublishGateV2` 에서 `ApprovalStatus[step] == "approved"` 가 되고 `MissingApprovals`/`BlockedReasons` 에 그 step 이 없다. 공백이 든 **과거** 만료일은 여전히 `expired` + 차단.
  3) 저장·API 응답의 `expires_at` 원문은 바뀌지 않는다(판정에만 트림). `POST …/approvals` 는 이미 트림해 저장하므로 쓰기 경로는 손대지 않는다(`admin_dataworks.go:1116`).
  4) 표 테스트가 프로덕션 수정을 되돌리면 실패하고, 기존 `internal/dataworks` 테스트는 그대로 통과한다 — 특히 `TestEvaluatePublishGateHonorsCustomRequirementForStandardProduct`(domain_test.go:69, `ExpiresAt: "invalid-time"` → `ApprovalStatus["security"] == "expired"` 이고 `!gate.Allowed`)가 깨지면 안 된다.

  확인한 사실: `EvaluatePublishGateV2`(domain.go:166 부근)는 내부에서 `EvaluatePublishGate(p, readiness, approvals, pack, now)`(domain.go:178)를 호출하므로 한 곳만 고치면 두 게이트가 함께 낫는다. `bestApprovalStatus` 의 호출부는 `domain.go:90` 한 곳뿐이다(`grep -rn bestApprovalStatus` 로 확인).
- 건드릴 파일:
  - `internal/dataworks/domain.go:668 bestApprovalStatus` — `raw := strings.TrimSpace(trace.ExpiresAt); if raw != "" { … time.Parse(time.RFC3339Nano, raw) … }` 로 바꾼다. 다른 분기·필드·JSON 이름은 그대로.
  - `internal/dataworks/domain_test.go` — 표 테스트 추가: 공백 미래 만료(approved 유지) / 공백 과거 만료(expired) / 공백만(만료 없음) / 빈 값 / 해석 불가(`" invalid-time "` → expired) / 만료 == now(expired, `!After` 유지) 를 `EvaluatePublishGate` 또는 `EvaluatePublishGateV2` 결과의 `ApprovalStatus`·`MissingApprovals` 로 단언.
  - (선택) `docs/OPERATIONS.md` — 승인 만료일 판정 규칙 2~3줄. OPERATIONS 는 PDF 정본이 없으므로 PDF 재생성 불필요(`ls docs/*.pdf` 로 확인할 것).
  프로덕션 파일 1개 + 테스트 1개.
- 검증 명령:
  - `go test ./internal/dataworks/ -run 'PublishGate|Approval' -v`
  - `go build ./... && go vet ./... && go test ./...`
  - `go run ./cmd/api-surface-audit` (gap 0 이어야 함)
  - `gofmt -l internal/dataworks/domain.go internal/dataworks/domain_test.go` (출력 없어야 함. `domain.go` 에 CRLF 줄이 0 개임을 확인했으므로 `dataworks_runtime.go` 류의 CRLF 함정은 없다. 단, 이번 정찰 세션에서는 권한 때문에 `gofmt` 자체를 실행하지 못했다 — 2026-09-20 회차가 이 패키지에서 출력 없음을 확인한 바 있다)
  - web 변경이 없으면 웹 체크 불필요.
- 위험과 피할 것:
  - `requiredApprovalSteps`·`RequiresStrictPublishGate`·masking 증거 판정(`admin_dataworks.go:1738 부근`)은 건드리지 말 것. 게이트가 **느슨해지는** 다른 변경을 같이 넣지 말 것 — 이번 변경은 "공백 때문에 잘못 막힌 것" 만 푼다.
  - 해석 불가 값은 계속 `expired` 로 닫아야 한다(주석에 이유를 남길 것).
  - `internal/store` 의 `UpsertApprovalTrace`(dataworks_governance.go:180)는 `ExpiresAt` 을 트림하지 않는다 — 저장 원문을 바꾸는 방향(마이그레이션·백필)으로 가지 말 것. 판정만 고친다.
  - `gofmt -l internal/proxy/dataworks_runtime.go` 는 저장소 기존 CRLF 때문에 HEAD 에서도 파일명을 출력한다. 이번 과제는 그 파일을 건드리지 않는다.
  - 보호 경로(auth/keycloak/mcp_oauth, store 마이그레이션, .github/workflows)는 손대지 않는다.
- 차선 후보: `dataWorksTimestampOK`(admin_dataworks.go:1766)가 받아들이는 `valid_from`/`valid_to`/`expires_at` 값을 런타임 파서(`contractScopeActive`)·액션 센터·`store.parseStoredTime` 이 모두 같게 읽는지 표 테스트로 고정(파서 통합은 하지 말고 동치만 단언, 테스트 전용 변경 / 가치 2 / 위험 1 / 작업량 S).
