- 과제: 액션 센터가 공백 포함 계약 만료일을 런타임과 같게 읽도록 수정 (가치 3 / 위험 1 / 작업량 S)
- 왜: `handleDataWorksActionCenter`는 `scope.ValidTo` 원문을 파싱하지만 런타임의 `contractScopeCanServe`는 앞뒤 공백을 제거하므로, 미래의 정상 계약도 액션 센터에서는 해석 불가로 처리되어 긴급 갱신(`high`) 경고가 뜬다. 같은 저장 행의 만료일을 두 경로가 동일하게 읽게 하면 실제 서비스 중인 계약의 잘못된 갱신 경고와 조회 창 밖 계약의 집계를 없앨 수 있다.
- 수용 기준: 1) 앞뒤 공백이 있는 `valid_to`가 20일 뒤이면 기본 30일 창에서 `contract_expiring` 1건·severity `medium`, 7일 창에서는 0건이고, 60일 뒤이면 기본 창에서 0건·13주 창에서 `medium` 1건이다. 2) 빈 값·공백만인 만료일은 경고 없음, 공백 포함 과거 날짜와 해석 불가 날짜는 기존대로 `high`, draft/suspended/revoked 계약은 제외하며 JSON 필드와 저장된 원문은 유지한다. 3) 실제 SQLite store + `NewServer(...).Routes()` HTTP 테스트에서 동일한 공백 포함 미래 계약과 API 키·접근권으로 `POST /v1/data-products/dw_credit_score/query` 200을 확인하고 액션 센터 GET의 summary와 actions를 함께 단언한다; 변경을 되돌리면 미래 계약의 경고 수준/조회 창 테스트가 실패해야 한다.
- 건드릴 파일: `internal/proxy/admin_dataworks.go:handleDataWorksActionCenter` — 계약 루프의 만료일을 한 번 TrimSpace한 지역 값으로 빈 값 검사와 RFC3339Nano 파싱; `internal/proxy/admin_dataworks_action_center_test.go` — 기존 `getActionCenter`, `actionsOfType`와 실제 서버 헬퍼를 이용한 회귀 사례; `docs/OPERATIONS.md:만료 예정 계약·권한 확인` — 계약 valid_to 역시 앞뒤 공백을 무시한다고 짧게 명시.
- 검증 명령: 저장소 루트에서 `go test ./internal/proxy -run 'TestDataWorksActionCenter|TestDataWorksContractScopeRefusesKeyOfAnotherProduct' -count=1`; `go build ./...`; `go vet ./...`; `go test ./...`; `go run ./cmd/api-surface-audit`. 포맷 확인은 `gofmt -l internal/proxy/admin_dataworks.go internal/proxy/admin_dataworks_action_center_test.go`.
- 위험과 피할 것: auth/session/migrations/.github/workflows는 수정하지 않는다. 런타임의 허용 날짜 문법·만료 경계·valid_from 판정·상태 판정은 확대하지 않는다. `contractScopeActive`를 액션 센터의 필터로 대체하면 이미 만료된 계약 경고까지 사라지므로 금지한다. 날짜 저장값을 일괄 정규화하는 마이그레이션도 하지 않는다. 이전에 고친 Entitlement·retirement·pricing·dist 문제는 다시 구현하지 않는다. 소스 문자열 검사나 판정 함수만의 대역으로 HTTP 증명을 대체하지 않는다.
- 차선 후보: `parseExpiryHorizon`의 d/w 정수 곱셈 오버플로를 400으로 거부 — `internal/proxy/admin_dataworks.go:parseExpiryHorizon`에서 `count*unit` 전 최대 duration/unit 검사, 같은 테스트 파일에서 큰 양수 d/w가 `invalid_expiring_within`인지 확인. 1순위 불성립 시에만 선택하고 두 과제를 묶지 않는다.

범위와 근거
- 기준 HEAD `88f853a`(v0.9.58). 위 세 수정 대상 및 `internal/proxy/dataworks_runtime.go:contractScopeCanServe/contractScopeActive/usableEntitlement`, `internal/store/dataworks_operations.go:UpsertContractScope`, `internal/proxy/admin_dataworks_access_window_test.go:newAccessWindowTestServer`, `internal/proxy/admin_dataworks_cross_product_test.go:TestDataWorksContractScopeRefusesKeyOfAnotherProduct`를 실제 읽었다.
- Store는 valid_to를 그대로 저장하고 runtime은 trim 후 읽는다. 관리자 POST는 이미 trim하므로 레거시 행 재현은 실제 `db.UpsertContractScope`로 넣어야 한다. API 키 설정과 실제 query 요청 방식은 cross_product_test의 기존 사례를 따른다. `ValidFrom`은 빈 값으로 두어 별도 문제와 혼합하지 않는다.
- 코드 비교로 불일치는 확인했다. 이번 정찰에서는 새 HTTP 재현 테스트를 작성하지 않았으므로 해당 입력의 실제 HTTP 응답은 미확인이다. 운영 DB에 해당 레거시 행이 얼마나 있는지도 미확인이다.

대안 비교 (solution-exploration)
- 선택: 표시 경로의 지역 값만 trim. 런타임이 이미 허용하는 입력을 그대로 따르는 최소 변경이고 외부 의존성이 없다.
- 공통 날짜 파서 추출: 향후 여러 경로 통일에는 유리하지만 이번 수정에 필요하지 않으며 valid_from·만료 경계 계약까지 흔들 위험이 있다.
- 저장 행 정규화: 모든 읽기에 효과가 있으나 DB 쓰기·마이그레이션과 기존 감사 원문 변경이 필요해 이번 범위에 부적절하다.
- 변경 보류/문서만 안내: 해당 행이 존재하지 않으면 비용을 아끼지만 코드상 오경보는 남는다. 가장 중요한 가정은 레거시 공백 행을 런타임의 정상 입력으로 계속 지원한다는 것이다.

실행 순서 (implementation-planning; 구현자가 상태를 갱신)
1. [pending] 위 세 파일에 국소 수정·HTTP 회귀·운영 문서를 한 묶음으로 적용. 증명: 위 특정 proxy 테스트 명령 및 gofmt. 체크포인트: 구현자 자동 검토, 별도 사람 승인 없음. 두 경로의 같은 행을 검증하고 summary/actions를 모두 확인한다.
2. [pending] 작업 패치 중 trim 변경만 잠시 되돌려 새 HTTP 회귀가 실패하는지 확인 후 즉시 복구. 증명: 같은 특정 proxy 테스트 명령이 복구 후 성공. 체크포인트: 실패 이유가 예상 경고 수준/조회 창인지 확인; 다르면 과제서를 수정하고 범위를 넓히지 않는다.
3. [pending] 전체 build/vet/test/API 감사 실행 및 diff 확인. 증명: 위 전체 검증 명령. 체크포인트: 다음 비평 에이전트에 인계; 실행하지 않은 검증은 미확인으로 남긴다.

작업량 근거 (estimating-and-contingency)
- Bottom-up 추정: 국소 수정 3–5분, 실제 서버 회귀 구성 10–16분, 문서·전체 검증·변이 확인 8–12분. 기본 21–33분, 중간 규모 HTTP 픽스처 조정과 검사 지연에 한해 contingency 5분을 별도 배정하여 26–38분을 계획한다.
- 범위는 통계적 신뢰구간이 아닌 정찰자의 중간 확신 추정이다. 이전 액션 센터 정합성 수정은 비교 가능한 사례이나 실제 소요 시간이 없어 독립 정량 추정으로 쓰지 않았다. 신규 외부 서비스·웹 빌드·DB 이행·보호 경로 수정은 포함하지 않는다.
- Management reserve는 0분(추가 범위 승인 없음). 첫 HTTP 검증에서 예상보다 큰 문제가 나오면 남은 작업을 재추정하고 차선 여부를 기록한다. 45분에 맞추려고 검증 범위를 조용히 빼지 않는다.
- 요청된 스킬은 전용 Skill 도구가 노출되지 않아 `/mnt/c/Users/USER/projects/headcount/plugins/{pmo,technology}/skills/`의 세 SKILL.md를 직접 읽어 적용했다. PMO references/sources.md도 읽었으며 외부 비용 모델·금전 가치·통계 신뢰도는 주장하지 않았다.

정찰 검증 결과
- 현재 HEAD에서 `go build ./...`, `go vet ./...`, `go test ./...` 모두 성공(exit 0). proxy 테스트 32.718초, store 테스트 13.482초; 일부 패키지는 캐시 결과다.
- `go run ./cmd/api-surface-audit` 성공: server routes 550 / OpenAPI paths 612 / CLI·SDK 각 5, 네 gap 목록 모두 비어 있음.
- `git status --short` 출력 없음. 웹 검증은 이번 정찰에서 실행하지 않았다. 위 성공은 기존 기준선이며 제안 수정의 효과는 구현자가 새 HTTP 회귀로 증명해야 한다.
