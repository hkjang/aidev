# 과제서 — 2026-10-06 정찰

- 과제: 릴리즈 게이트가 깨져 있던 동안 main 에 올라가지 못한 **검증 완료된 Go 버그 수정 3건**을 되살리기 (가치 5 / 위험 1 / 작업량 S)
- 왜: 2026-10-01·10-03·10-04 회차는 각각 "DB 조회 실패를 데이터 없음으로 숨기는" 결함 하나를 고치고 전 검증을 통과했는데, 같은 기간 릴리즈 **검증 배선**이 깨져 있어 회차 결과가 `verify-failed` 였고 그 커밋들이 머지되지 않았다. 배선은 2026-10-05 회차(a67c06a·9d43941·cf3131c)가 고쳐 v0.9.67 이 나갔으므로 지금은 그대로 올릴 수 있다. **세 결함이 base(31f420d)에 그대로 남아 있음을 실행으로 확인했다** — 아래 좌표는 이번 세션에 `grep -n` 으로 직접 본 현재 줄 번호다.

## 0단계 — 먼저 확인할 것 (실행 2분)

```
git log --oneline -1 HEAD                      # 31f420d chore: release v0.9.67 이어야 함
git merge-base --is-ancestor 4700560 HEAD; echo $?   # 1 (= 아직 안 들어옴) 이어야 함
git merge-base --is-ancestor 5fcc7a3 HEAD; echo $?   # 1
git merge-base --is-ancestor e64199a HEAD; echo $?   # 1
```
세 개가 모두 `1` 이면 본 과제 진행. 하나라도 `0` 이면 그건 이미 들어온 것이니 **빼고** 나머지만 하고, 셋 다 `0` 이면 차선 후보로 넘어갈 것.

> **함정 (이번 세션에 실제로 걸렸다)**: 이 워크트리의 로컬 `main` ref 는 **baa3415 = v0.9.53 에 멈춰 있다**(`git log --oneline -1 main` 으로 확인). `git diff main`·`git checkout main -- <file>` 를 쓰면 13개 릴리즈 분량을 되돌려 버린다. **항상 `HEAD` 또는 `31f420d` 를 기준으로 쓸 것.**

## 수용 기준

1) 세 커밋이 각자의 원래 메시지로 브랜치에 올라간다(한 커밋으로 묶지 말 것 — 비평·수리가 건별로 판정할 수 있어야 한다). 각 프로덕션 파일의 변경이 원 커밋과 동일: `git diff 4700560 HEAD -- internal/proxy/admin_dataworks.go` 가 빈 출력(같은 방식으로 나머지 둘).
2) 새·추가된 테스트가 전부 통과:
   `go test ./internal/proxy -run 'ActionCenterRejectsUnavailablePublishGate|EvidenceRejectsUnavailableSources|UnreadableContractScope' -count=1`
3) **각 수정의 프로덕션 파일만** `git checkout 31f420d -- <그 파일>` 로 되돌리면 해당 테스트(만)가 실패함을 확인한다 = 테스트가 결함을 실제로 증명한다. 세 건 각각 확인하고 매번 복원할 것. 기대 실패 메시지(원장 기록):
   - A: `status = 200, want 500` + 응답에 `"blocked_launches":0`
   - B: `status = 200, want 500 (risk_basis would be dropped silently)` + cleanup 단언에서 `GET evidence types` 가 한 종류 부족
   - C: `status = 403, want 500 (a failed contract read is not an inactive contract)` + `"code":"contract_scope_inactive"`
4) `go build ./...` 0 · `go vet ./...` 0 · `go test ./... -count=1` 전체 통과 · `go run ./cmd/api-surface-audit` 누락 0(550 routes / 612 OpenAPI paths, 감사 4항목 전부 `[]`).
5) `docs/OPERATIONS.md` 에 세 운영 문단이 **모두** 들어가고 충돌 잔해(`<<<<<<<`)가 없다. `git diff --check` 0.
6) 보호 파일 무변경 — `git diff --name-only 31f420d` 에 다음이 **없어야** 한다: `.github/workflows/ci.yml`, `web/**`, `scripts/web-run.mjs`, 루트 `package.json`, `internal/store/**`, `internal/proxy/keycloak*`, `internal/proxy/mcp_oauth.go`.

## 건드릴 파일 (프로덕션 3 + 테스트 3 + 문서 1)

- `internal/proxy/admin_dataworks.go:253` — 액션 센터 상품 루프의 `if gate, err := s.dataWorksPublishGate(r.Context(), product); err == nil && !gate.Allowed {` 가 평가 오류를 버려, 자산 준비도·승인 이력·Evidence Pack 조회가 실패하면 `approved`/`review`/`risk_review` 상품이 `blocked_launches` 에서 조용히 빠지고 200·0건으로 보고된다. → 오류를 `writeOpenAIError(…, 500, err.Error(), "server_error", "publish_gate_failed")` 로 반환. 같은 실패에 `:877`(GET …/publish-gate)·`:1840`(POST …/publish)은 이미 500 을 낸다. **원 커밋 4700560 의 hunk 가 현재 코드의 context 와 그대로 맞는 것을 이번 세션에 확인했다.**
- `internal/proxy/admin_dataworks_ops.go:547-550` — `buildProductEvidencePack` 이 `LatestProductDefinition`/`LatestProductRiskReview`/`LatestProductPOCPlan` 의 오류를 `_` 로 버린다(현재 코드 그대로 확인). 짧아진 목록이 `ReplaceProductEvidencePack`(= `DELETE FROM dw_product_evidence WHERE product_key = ?` 후 삽입)에 넘어가 **저장된 증거가 영구 삭제**되는데 응답은 200 + `"persisted":true` 다. → 오류를 반환해 `Replace` 전에 끊고 500 `evidence_refresh_failed`(저장 행이 없어 즉석 생성하는 GET 경로는 바로 위 `ListProductEvidence` 와 같은 코드 `evidence_failed`). 원 커밋 5fcc7a3.
- `internal/proxy/dataworks_runtime.go:246,252` — `usableEntitlement` 의 `if err != nil || !found || contract.ProductKey != product.ProductKey { continue }` 가 조회 실패를 "계약 없음" 과 묶는다. 한 API 키가 한 상품에 엔타이틀먼트를 여럿 가질 수 있고 아무도 사용 가능하지 않으면 호출부(`:70`)가 `candidates[0]` 로 되돌아가므로, **다른 후보의 DB 읽기 실패가 403 `contract_scope_inactive`** 로 보고된다. 단일 후보 경로(`:100`)는 같은 실패에 이미 500 `contract_lookup_failed` 다. → 시그니처를 `(store.APIEntitlement, bool, error)` 로 바꾸고 **첫 오류를 지역 변수에 보관하되 루프는 끝까지 돌려**(사용 가능한 후보를 찾으면 오류를 버리고 성공), 끝까지 못 찾았을 때만 오류를 반환해 호출부가 500 `contract_lookup_failed` 를 낸다(새 code 없음). `!found`·ProductKey 불일치·`contractScopeActive`·`purpose` 판정은 그대로 `continue`. 원 커밋 e64199a.
- 테스트 3파일(원 커밋에서 그대로 가져옴): `internal/proxy/admin_dataworks_action_center_test.go`(**이미 있는 파일에 약 92줄 추가**), `internal/proxy/admin_dataworks_evidence_test.go`(신규 — HEAD 에 없음을 확인), `internal/proxy/dataworks_runtime_entitlement_test.go`(신규 — HEAD 에 없음을 확인). 세 테스트는 모두 실제 `store.SQLStore`(SQLite) + `NewServer(...).Routes()` HTTP 경로만 쓰고, 조회 실패는 테이블 rename 또는 같은 이름의 뷰 치환으로 만든다(store 코드 변경 없음).
- `docs/OPERATIONS.md` — 세 커밋이 각각 6·10·7줄을 추가한다. HEAD 의 OPERATIONS.md 는 v0.9.66 대비 **+25줄 달라졌다**(2026-10-05 의 "웹 워크벤치 검증" 절 등) → **cherry-pick 충돌이 날 수 있다**(미확인: 이 세션에서는 `git merge-tree`·`cherry-pick` 실행 권한이 없어 충돌 여부를 확정하지 못했다). 충돌 시 "하나만 남기기" 로 해결하지 말고 **세 문단 모두 살려** 수동 병합하고, 2026-10-05 가 추가한 절은 건드리지 말 것. `ls docs/*.pdf` — OPERATIONS 정본 PDF 없음 → PDF 재생성 불필요.

## 작업 순서

1. 0단계 확인.
2. `git cherry-pick 4700560` → 충돌은 `docs/OPERATIONS.md` 에서만 기대. 해결 후 `go build ./...`·대상 테스트.
3. `git cherry-pick 5fcc7a3` → 같은 방식.
4. `git cherry-pick e64199a` → 같은 방식.
5. 기준 3(프로덕션 파일만 되돌려 red 확인)을 건별로 수행.
6. 기준 4 전체 검증.

cherry-pick 이 Go 파일에서 깨지면 **손으로 고치지 말고** 해당 커밋의 diff(`git show <sha> -- <file>`)를 보고 같은 변경을 다시 적용할 것 — 위의 "건드릴 파일" 설명이 의도를 그대로 담고 있다.

## 검증 명령 (이 저장소에서 실제로 도는 것)

```
go build ./...
go vet ./...
go test ./internal/proxy -run 'ActionCenterRejectsUnavailablePublishGate|EvidenceRejectsUnavailableSources|UnreadableContractScope' -count=1
go test ./... -count=1          # internal/proxy 36~40s, internal/store 14s, 전체 분 단위
go run ./cmd/api-surface-audit  # 누락 0 (550 routes / 612 OpenAPI paths)
git diff --check
```
**미확인**: 이번 정찰 세션에서는 `go` 실행이 승인되지 않아 base 에서 직접 돌려 보지 못했다. 위 명령·수치는 원장(2026-10-01~10-05 회차)이 같은 base 계열에서 기록한 값이다. 구현자는 0단계 직후 `go build ./...` 로 base 가 깨끗한지 먼저 확인할 것.

## 위험과 피할 것

- **로컬 `main` ref 가 v0.9.53(baa3415)에 멈춰 있다.** `git diff main` / `git checkout main -- …` 금지. `HEAD` 또는 `31f420d` 를 쓸 것.
- `gofmt -l internal/proxy/dataworks_runtime.go` 는 **HEAD 에서도** 파일명을 출력한다(기존 CRLF). 줄 끝을 건드리지 말고 gofmt 출력을 수정 신호로 오해하지 말 것.
- `web/node_modules` 가 이 워크트리에 **없다**(확인). web 변경이 없으므로 `npm` 계열을 돌리지 말 것 — 돌리면 긴 설치 + 이 과제와 무관한 실패를 끌어온다.
- 세 커밋은 모두 부모가 `98c5081`(v0.9.66, HEAD 의 조상)이고 서로 독립이다. 프로덕션·테스트 파일이 겹치지 않으므로 Go 쪽 충돌은 기대하지 않는다. 겹치는 것은 `docs/OPERATIONS.md` 뿐.
- 과거 교훈: 조회 오류를 "정상 0건" 으로 숨기는 경로가 이 저장소의 반복 결함 계열이다. 세 수정 모두 **게이트를 닫는/오류를 드러내는 방향**이며, 반대로 느슨하게 만드는 변경(`continue` 로 되돌리기, 500→200)은 넣지 말 것.
- `internal/store` 는 **읽기만**. 마이그레이션·스키마는 건드리지 않는다.
- 새 action type·새 summary 키·새 API 경로가 없으므로 `api-surface-audit` 수치(550/612)는 변하지 않아야 한다. 변하면 무언가를 잘못 가져왔다는 신호다.

## 차선 후보

1. **e64199a 1건만** 되살리기 — `internal/proxy/dataworks_runtime.go` + 테스트 1파일 + 문서. cherry-pick 이 예상보다 크게 깨지거나 전체 `go test` 가 번지면 가장 작은 조각만 담는다(런타임 접근 게이트라 운영 영향이 가장 직접적이다).
2. `web/eslint.config.js` 에 `**/*.{js,mjs}` 설정 객체를 추가해 릴리즈 헬퍼 스크립트(`web/scripts/run-with-supported-node.mjs`, `scripts/web-run.mjs`)가 **규칙 0개로 린트되는** 사각지대를 없애기(원본 5ed6876 — 이것도 main 에 없다). web 1파일. `globals` 패키지가 의존성에 없으니 lock 재생성 없이 Node 전역만 인라인 선언할 것.
