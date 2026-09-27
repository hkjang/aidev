- 과제: 규제 추적 재생성이 승인 이력의 `expires_at` 을 지워 만료 승인을 영구 유효로 만드는 문제 수정 (가치 3 / 위험 1 / 작업량 S)

- 왜: `POST /admin/dataworks/products/{key}/regulatory-trace`(internal/proxy/admin_dataworks_ops.go:148, 162)가 호출하는 `syncApprovalTracesFromRegulatoryTrace`(같은 파일 583행)는 `ExpiresAt` 을 채우지 않은 `store.ApprovalTrace` 를 결정적 id `"appr_" + p.ProductKey + "_" + step`(step ∈ data_owner|legal|compliance)으로 UPSERT 하고, `store.SQLStore.UpsertApprovalTrace`(internal/store/dataworks_governance.go:180, UPSERT 는 191행)의 `ON CONFLICT(id) DO UPDATE SET … expires_at = excluded.expires_at` 이 기존 값을 무조건 빈 문자열로 덮는다. 그래서 관리자가 `POST …/products/{key}/approvals`(admin_dataworks.go:1116 근처, 워크벤치 "승인 결정 수정" → "만료 시각" 필드가 그 id 를 그대로 실어 보낸다 — web/src/pages/products/product-workspace-page.tsx:396 `id: approval.id || undefined`)로 기록해 둔 만료일이, OpenAPI 가 "Generate or **replace** the regulatory trace matrix"(admin_openapi.go:322)라고 계약한 재생성 한 번에 사라진다. `bestApprovalStatus`(internal/dataworks/domain.go:668~683)는 빈 `expires_at` 을 "만료 없음" 으로 읽어 `approved` 를 그대로 돌려주므로, 만료되어 차단돼야 할 승인으로 퍼블리시 게이트가 다시 열린다(게이트가 느슨해지는 방향의 결함이고, 영향 단계는 엄격 게이트의 필수 3단계 전부다).

- 수용 기준:
  1) 실제 `store.SQLStore`(SQLite) + `NewServer(...).Routes()` HTTP 경로에서: 규제 추적 생성(3단계 approved) → `POST …/products/{key}/approvals` 로 `{"id":"appr_<product_key>_legal","step":"legal","status":"approved","required":true,"expires_at":"<과거 시각>"}` 기록 → 규제 추적 **재생성** 이후에도 `GET …/products/{key}/approvals` 의 그 행 `expires_at` 이 관리자가 넣은 값 그대로 남는다(저장 원문 불변, 트림·재포맷 없음).
  2) 같은 흐름에서 `POST /admin/factory/products/{key}/publish` 가 재생성 전·후 모두 `409`(`"error":"publish gate blocked"`, admin_dataworks.go:1808)를 유지하고, `GET …/products/{key}/publish-gate`(admin_dataworks.go:834)의 `publish_gate.missing_approvals`·`blocked_reasons` 에 그 단계가 계속 남는다. 미래 만료일로 넣은 대조 사례는 재생성 전·후 모두 `200` 이어야 한다(닫히는 방향만 고치고 열리는 방향은 건드리지 않았음을 확인).
  3) 규제 추적이 실제로 갱신해야 하는 값은 그대로 갱신된다: 재생성에서 `decision` 을 `approved` → `rejected` 로 바꾼 사례가 `status="rejected"` 로 반영되고 `decided_by`·`notes`(row.Evidence)·`evidence_ref`(row.ID)도 새 값으로 바뀐다 — 즉 "덮어쓰지 않는 필드" 는 `expires_at` 하나뿐임을 단언.
  4) 프로덕션 변경(admin_dataworks_ops.go)만 되돌린 상태에서 1)·2) 의 과거 만료 사례만 실패하고 나머지 사례와 기존 테스트는 전부 통과함을 실행으로 확인하고, 실패 출력을 회차 노트에 남긴다.

- 건드릴 파일 (프로덕션 1 + 테스트 1 + 문서 1):
  - `internal/proxy/admin_dataworks_ops.go:583` `syncApprovalTracesFromRegulatoryTrace` — `for _, row := range rows` 루프 **전에** `s.db.ListApprovalTraces(ctx, p.ProductKey)`(프록시에서 이미 쓰는 메서드: admin_dataworks.go:1084·1699·1846)를 한 번 호출해 `id → ExpiresAt` 맵을 만들고, 새로 만드는 `store.ApprovalTrace` 의 `ExpiresAt` 에 그 값을 실어 준다. 조회 오류는 종전처럼 조용히 넘기되(이 함수는 `_ =` 로 오류를 무시하는 best-effort 경로다) 그 경우 만료일을 지우지 않는 쪽이 안전하므로 판단을 주석에 남길 것. 규제 추적 입력에는 만료일 개념이 아예 없으므로 "지우지 않는다" 가 유일하게 맞는 동작이다. 새 store 메서드나 인터페이스 변경은 필요 없다.
  - `internal/proxy/admin_dataworks_test.go` — 위 HTTP 표 테스트 추가. 기존 `postJSON`(server_test.go:416)·`requireStatus`(admin_dataworks_test.go:427) 헬퍼와, 이미 `regulatory-trace` 두 번 POST → `publish` 409 → 200 을 검증하는 기존 테스트(admin_dataworks_test.go:190~212)의 픽스처 구성 방식을 그대로 재사용할 것. 손으로 만든 대역·직접 주입 없이 실제 라우트로만 재현한다.
  - `docs/OPERATIONS.md` 4절(2026-09-27 회차가 승인 만료일 판정 규칙을 적어 둔 절)에 "규제 추적 재생성은 승인 만료일을 지우지 않는다" 2~3줄. `ls docs/*.pdf` 로 OPERATIONS 정본 PDF 가 없음을 확인한 뒤 md 만 고칠 것.

- 검증 명령:
  - `go test ./internal/proxy/ -run 'RegulatoryTrace|Approval|DataWorks' -v`
  - `go build ./...` · `go vet ./...` · `go test ./...` (internal/proxy 만 30~40초)
  - `go run ./cmd/api-surface-audit` — gap 0 이어야 한다(라우트·OpenAPI 를 안 건드리므로 값은 그대로)
  - `gofmt -l internal/proxy/admin_dataworks_ops.go internal/proxy/admin_dataworks_test.go` — 출력 없어야 한다(이 두 파일은 CRLF 문제 없음. `dataworks_runtime.go` 는 기존 CRLF 때문에 HEAD 에서도 파일명을 출력하니 그 파일은 건드리지 말 것)

- 위험과 피할 것:
  - `UpsertApprovalTrace` 의 UPSERT 컬럼 목록(internal/store/dataworks_governance.go:191)을 고치지 말 것 — 관리자 `POST …/approvals` 는 만료일을 **빈 값으로 지우는** 정상 경로이고 그 동작이 이 컬럼에 의존한다. 보존 판단은 호출자(sync) 쪽에만 둔다.
  - `bestApprovalStatus`(internal/dataworks/domain.go:668)는 2026-09-27 회차(67aa619)에서 방금 트림 규칙을 고친 자리다. 이번 회차에서 다시 손대지 말 것.
  - `Required: true`·`Notes`·`EvidenceRef`·`DecidedBy` 를 규제 추적이 덮는 것은 의도된 동작이다. "관리자 입력 보존" 을 이 필드들로 확대하지 말 것(범위가 커지고 3) 과 충돌한다).
  - 보호 경로 없음: 인증(keycloak*.go·mcp_oauth.go), store 마이그레이션, `.github/workflows/ci.yml`, `web/embed.go` 를 건드릴 필요가 전혀 없다. web 변경도 불필요하므로 웹 체크는 실행하지 않아도 된다(실행하지 않았음을 회차 노트에 적을 것).
  - 미확인: 이 결함을 코드 실행으로 재현하지는 않았다(정찰은 코드를 바꾸지 않는다). 근거는 위 세 지점의 소스 독해다 — 구현자는 먼저 수정 전 상태에서 재생성이 `expires_at` 을 비우고 `publish` 가 409→200 으로 열리는 것을 실제로 확인한 뒤 고칠 것. 만약 재현되지 않으면(예: sync 가 실제로는 다른 id 를 쓰거나 워크벤치가 id 를 보내지 않아 행이 겹치지 않는 경우) 억지로 진행하지 말고 차선 후보로 넘어갈 것.

- 차선 후보: `dataWorksTimestampOK`(admin_dataworks.go:1766)를 통과해 저장된 `valid_from`/`valid_to`/`expires_at` 을 런타임(`contractScopeActive`)·액션 센터(admin_dataworks.go:252~295)·`store.parseStoredTime`·`bestApprovalStatus` 네 경로가 같은 입력에서 같게 읽는지 표 테스트로 고정(파서 통합은 하지 말고 동치만 단언). 테스트 전용이라 프로덕션 파일 0개.
