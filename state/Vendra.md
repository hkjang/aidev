# Vendra 자율 개선 기록

## 2026-09-02
- 선택: 요청 본문의 날짜를 검증 없이 `$n::date`로 캐스팅하던 네 개의 쓰기 경로 수정 (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공 (commit e5cef01)
- 요약: `validDateFields`는 business object의 세 날짜에만 적용돼 있었고, 같은 결함이 견적 유효일(포털 입찰), 예정일(포털 납품/인보이스/문의), 거래 시작일(공급업체 등록), 거래일(구매 원장) 네 곳에 남아 있었다. 잘못된 날짜가 PostgreSQL 캐스트까지 도달해 "…저장하지 못했습니다"라는 필드를 지목하지 않는 오류로 돌아왔고, 특히 포털 입찰은 마감 직전에 견적 전체(금액·품목·조건)가 저장되지 않으면서 공급업체에게 아무 단서도 주지 못했다. 철자가 아니라 연산(요청 파라미터의 date 캐스트)을 기준으로 훑어 찾았고, 같은 기준으로 패키지를 파싱해 검증 누락을 잡는 `TestEveryDateCastIsGuarded`와 네 엔드포인트를 실제로 호출하는 통합 테스트를 추가했다. 검증: docker postgres:16-alpine을 띄워 CI와 동일한 세 DSN(VENDRA_TEST_DSN / MIGRATE / UPGRADE)으로 `go test ./internal/... ./cmd/...` 전체 통과, `gofmt -l`·`go vet` 무결. 가드 테스트는 수정 하나를 되돌리면 실패하는 것까지 확인했다.
- 보류 아이디어:
  - CI의 go job이 `go test ./internal/...`만 돌려 `./cmd/...`를 빼놓음 — Makefile/README와 불일치 (가치 2 / 위험 1 / S)
  - `web` 프론트엔드 테스트 커버리지가 9개 파일뿐 — Sourcing/Objects/Admin 페이지에 테스트 없음 (가치 3 / 위험 1 / L)
  - 공급업체 수정(PATCH /suppliers/{id})은 tradingSince·annualSpend를 아예 갱신하지 않음 — 의도인지 누락인지 확인 필요 (가치 3 / 위험 2 / S)
  - `POST /spend/transactions`의 quantity·unitPrice·amount에 상한이 없음 — 폼에만 존재하는 제약 (가치 3 / 위험 2 / M)
  - `Makefile`의 VERSION(0.6.21)이 README의 릴리스 예시(0.7.26)와 어긋남 — 릴리스 문서 정합성 (가치 2 / 위험 1 / S)
- 릴리즈: v0.7.35 (2026-09-02, 태그 사후 푸시)

## 2026-09-02
- 선택: 요청 본문의 숫자를 범위 검사 없이 numeric 컬럼에 넣던 쓰기 경로 전부 수정 (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공 (commit 1be68e5)
- 요약: 날짜 스윕과 같은 기준(철자가 아니라 연산 — 요청 본문의 숫자가 numeric 컬럼에 도달)으로 훑어, 범위 검사가 있는 곳이 리스크 핸들러 하나뿐임을 확인하고 나머지를 채웠다. 업무 객체의 금액·점수, 공급업체 연간 거래금액, 구매 원장의 금액·수량·단가, 포털 입찰의 총 견적금액·납기, 포털 제출 금액이 대상이다. 범위를 넘으면 numeric(20,2)/integer가 거부해 "저장하지 못했습니다"로 필드 없이 돌아왔고, 범위 안이면 더 나빴다 — 폼은 전부 min="0"인데 API는 음수를 받아 연간 지출 합계와 집중도 분모를 깎고 승인 라우팅의 minAmount 아래로 빠졌으며, 음수 입찰은 200을 받고도 가격 비교에서 제외됐고, 99,999점 객체는 Supplier 360 품질 평균에 섞였다. 상한은 10^15 — numeric(20,4)의 10^16과 float64의 2^53 둘 다 아래라 받은 값이 그대로 저장된다. 검증: docker postgres:16-alpine으로 CI와 동일한 세 DSN을 걸고 `go test ./internal/... ./cmd/...` 전체 통과, gofmt·go vet 무결. 구매 원장 가드를 되돌리면 통합 테스트와 파싱 가드 테스트가 모두 실패하는 것까지 확인했다.
- 보류 아이디어:
  - CI의 go job이 `go test ./internal/...`만 돌려 `./cmd/...`를 빼놓음 — Makefile/README와 불일치 (가치 2 / 위험 1 / S)
  - `web` 프론트엔드 테스트 커버리지가 9개 파일뿐 — Sourcing/Objects/Admin 페이지에 테스트 없음 (가치 3 / 위험 1 / L)
  - 공급업체 수정(PATCH /suppliers/{id})은 tradingSince·annualSpend·businessNumber를 갱신하지 않음 — 편집 폼이 보내지 않으므로 의도로 보이나 API 전용 클라이언트에는 무응답 (가치 2 / 위험 2 / S)
  - `Makefile`의 VERSION(0.6.21)이 README의 릴리스 예시(0.7.26)와 어긋남 — 릴리스 문서 정합성 (가치 2 / 위험 1 / S)
  - 문자열 길이 상한(maxIdentifierLen)이 공급업체·리스크·문서에만 적용됨 — 업무 객체 제목·통화 등은 미적용 (가치 3 / 위험 1 / M)
- 릴리즈: v0.7.36 (2026-09-02)

## 2026-09-03
- 선택: 요청 본문의 짧은 자유 텍스트(레코드 이름·제목·코드)를 길이 검사 없이 text 컬럼에 넣던 쓰기 경로 전부 수정 (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공 (commit ba6e366)
- 요약: 날짜·숫자 스윕과 같은 기준(철자가 아니라 연산 — 요청 본문의 짧은 문자열이 레코드를 표시하는 text 컬럼에 도달)으로 훑었다. maxIdentifierLen은 공급업체 10개 필드, 리스크 유형, 문서 이름에만 걸려 있었고 나머지는 전부 무방비였다. text에는 길이가 없어 20,000자 계약 제목이 그대로 저장된 뒤 레코드를 열지도 않은 사람의 목록·드롭다운·내보내기·감사 로그를 전부 밀어냈고, 구매 원장의 품목명·분류는 Spend 차트의 그룹 키라 범례가 그 길이에 맞춰졌으며, 표시 이름은 그 계정이 남기는 모든 감사 줄과 결재 단계의 서명이었다. 특히 포털 자가등록은 createSupplier가 처음부터 검사해 온 suppliers.name 컬럼을 아무도 재지 않은 두 번째 문으로 썼는데, 그 문이 바로 외부인이 지나는 문이다. rows.go에 validDateFields·validNumberFields 옆에 validTextFields를 두어 거절이 고칠 입력란을 지목하게 했고, 호출자가 하나뿐이던 overlongField를 대체했다. 검증: docker postgres:16-alpine으로 CI와 동일한 세 DSN을 걸고 `go test ./internal/... ./cmd/...` 전체 통과, gofmt·go vet 무결. 패키지를 파싱하는 TestEveryRequestLabelIsBounded(검사를 뺀 필드의 사유는 unboundedByDesign에 명시)와 17개 엔드포인트를 실제로 호출하는 TestEveryLabelOnAWriteIsBounded를 추가했고, 포털 프로필 가드를 되돌리면 둘 다 실패하는 것까지 확인했다.
- 보류 아이디어:
  - CI의 go job이 `go test ./internal/...`만 돌려 `./cmd/...`를 빼놓음 — Makefile/README와 불일치 (가치 2 / 위험 1 / S)
  - `web` 프론트엔드 테스트 커버리지가 9개 파일뿐 — Sourcing/Objects/Admin 페이지에 테스트 없음 (가치 3 / 위험 1 / L)
  - 업무 객체의 data jsonb 블롭에는 어떤 검증도 없음 — 폼이 쓰는 키(품목·수량·단가)가 API로는 무제한 (가치 3 / 위험 2 / M)
  - `Makefile`의 VERSION(0.6.21)이 README의 릴리스 예시(0.7.26)와 어긋남 — 릴리스 문서 정합성 (가치 2 / 위험 1 / S)
  - 공급업체 수정(PATCH /suppliers/{id})은 tradingSince·annualSpend·businessNumber를 갱신하지 않음 — 편집 폼이 보내지 않으므로 의도로 보이나 API 전용 클라이언트에는 무응답 (가치 2 / 위험 2 / S)
- 릴리즈: v0.7.37 (2026-09-03)

## 2026-09-03
- 선택: 요청 값이 리스크 등급(LOW/MEDIUM/HIGH/CRITICAL)으로 들어가는 모든 쓰기 경로에 어휘 검사 추가 (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공 (commit d36cc9f)
- 요약: 날짜·숫자·문자열 스윕과 같은 기준(철자가 아니라 연산 — 요청 값이 리스크 등급 컬럼이나 그 등급으로 라우팅하는 결재 규칙에 도달)으로 훑었다. 여섯 개 문 중 검사하던 곳은 createRisk의 severity 하나뿐이었다. 등급은 자유 텍스트가 아니라 질의가 분기하는 단어다 — 낙찰 계산은 `CASE risk_level WHEN 'LOW' THEN 100 … ELSE 0`이라 "high"로 저장된 업체는 고위험이 아니라 CRITICAL보다 나쁜 값으로 읽혀 입찰에서 지고, 대시보드의 `IN('HIGH','CRITICAL')` 집계에서는 빠지며, `NOT IN('CRITICAL')` 추천 목록에는 남는다. 업무 객체 쪽이 더 나쁘다: "High"는 HIGH를 요구하는 결재 규칙과 매칭되지 않아 matchingWorkflow가 아무것도 찾지 못하고, 제출은 결재자 없이 그 자리에서 승인된다(no_matching_workflow). 워크플로 조건은 등급과 함께 형태도 검사한다 — workflowConditions로 파싱되지 않는 조건 하나가 저장되면 그 객체 유형의 모든 제출이 영구히 500이 된다. rows.go에 riskGrades/enumField/validEnumFields를 두고 createRisk의 switch를 대체했다. 검증: docker postgres:16-alpine으로 CI와 동일한 세 DSN을 걸고 `go test ./internal/... ./cmd/...` 전체 통과, gofmt·go vet 무결. 패키지를 파싱하는 TestEveryRiskGradeIsInTheVocabulary와 여섯 엔드포인트를 실제로 호출하는 TestEveryRiskGradeOnAWriteIsInTheVocabulary를 추가했고, updateSupplier 가드와 updateWorkflow 가드를 각각 되돌리면 둘 다 실패하는 것까지 확인했다.
- 보류 아이디어:
  - CI의 go job이 `go test ./internal/...`만 돌려 `./cmd/...`를 빼놓음 — Makefile/README와 불일치 (가치 2 / 위험 1 / S)
  - 업무 객체의 status는 여전히 임의 문자열 — `status IN('completed','accepted','closed')` 같은 집계가 오타 하나로 조용히 빠짐(유형별 어휘가 어디에도 정의돼 있지 않아 위험은 큼) (가치 3 / 위험 3 / M)
  - 업무 객체의 data jsonb 블롭에는 어떤 검증도 없음 — 폼이 쓰는 키(품목·수량·단가)가 API로는 무제한이고 목록 응답이 블롭 전체를 반환 (가치 3 / 위험 2 / M)
  - 워크플로 조건 폼 라벨은 "공급업체 Risk 조건"인데 실제로는 업무 객체의 risk_level과 매칭됨 — 라벨과 동작 불일치 (가치 3 / 위험 2 / S)
  - `Makefile`의 VERSION(0.6.21)이 README의 릴리스 예시(0.7.26)와 어긋남 — 릴리스 문서 정합성 (가치 2 / 위험 1 / S)
- 릴리즈: v0.7.38 (2026-09-03)
