---
title: "Vendra — 자율 개선 이력"
description: "Vendra: 자율 개선 회차 7회, 릴리즈 6건. 최근 릴리즈 v0.7.41."
last_modified_at: 2026-09-05 10:00:32 +0900
---
{% raw %}
<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "SoftwareSourceCode",
 "name": "Vendra",
 "codeRepository": "https://github.com/hkjang/Vendra",
 "url": "https://hkjang.github.io/aidev/projects/Vendra/",
 "description": "Vendra: 자율 개선 회차 7회, 릴리즈 6건. 최근 릴리즈 v0.7.41.",
 "inLanguage": "ko",
 "maintainer": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-09-05T10:00:32+09:00",
 "version": "0.7.41"
}
</script>

# Vendra

<p class="tldr"><strong>요약.</strong> Vendra: 자율 개선 회차 7회, 릴리즈 6건. 최근 릴리즈 v0.7.41.</p>

<ul class="stats"><li><b>7</b><span>회차</span></li><li><b>1</b><span>프로젝트</span></li><li><b>6</b><span>릴리즈</span></li><li><b>1</b><span>머지(릴리즈 없음)</span></li><li><b>0</b><span>변경 없음</span></li><li><b>0</b><span>실패</span></li></ul>

## 현황

<dl class="kv">
<dt>저장소</dt><dd><a href="https://github.com/hkjang/Vendra">https://github.com/hkjang/Vendra</a></dd>
<dt>마지막 회차</dt><dd>2026-09-04 23:39 KST — <span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/Vendra/pull/107">PR #107</a>, released <a href="https://github.com/hkjang/Vendra/releases/tag/v0.7.41">v0.7.41</a></dd>
<dt>최근 릴리즈</dt><dd><a href="https://github.com/hkjang/Vendra/releases/tag/v0.7.41">v0.7.41</a> — released <a href="https://github.com/hkjang/Vendra/releases">전체 릴리즈 →</a></dd>
</dl>

## 회차 이력

<div class="table-wrap"><table class="rt"><thead><tr><th class="primary">일시</th><th>결과</th></tr></thead><tbody><tr data-status="released"><td data-label="일시" class="primary">2026-09-04 23:39</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/Vendra/pull/107">PR #107</a>, released <a href="https://github.com/hkjang/Vendra/releases/tag/v0.7.41">v0.7.41</a></td></tr><tr data-status="released"><td data-label="일시" class="primary">2026-09-03 21:14</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/Vendra/pull/106">PR #106</a>, released <a href="https://github.com/hkjang/Vendra/releases/tag/v0.7.40">v0.7.40</a></td></tr><tr data-status="released"><td data-label="일시" class="primary">2026-09-03 13:03</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/Vendra/pull/105">PR #105</a>, released <a href="https://github.com/hkjang/Vendra/releases/tag/v0.7.39">v0.7.39</a></td></tr><tr data-status="released"><td data-label="일시" class="primary">2026-09-03 07:21</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/Vendra/pull/104">PR #104</a>, released <a href="https://github.com/hkjang/Vendra/releases/tag/v0.7.38">v0.7.38</a></td></tr><tr data-status="released"><td data-label="일시" class="primary">2026-09-03 01:22</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/Vendra/pull/103">PR #103</a>, released <a href="https://github.com/hkjang/Vendra/releases/tag/v0.7.37">v0.7.37</a></td></tr><tr data-status="released"><td data-label="일시" class="primary">2026-09-02 19:01</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/Vendra/pull/102">PR #102</a>, released <a href="https://github.com/hkjang/Vendra/releases/tag/v0.7.36">v0.7.36</a></td></tr><tr data-status="merged"><td data-label="일시" class="primary">2026-09-02 13:08</td><td data-label="결과"><span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/Vendra/pull/101">PR #101</a>, release released</td></tr></tbody></table></div>

## 원장 (에이전트가 남긴 기록)

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

## 2026-09-03
- 선택: 요청 본문이 실어 나르는 레코드 id를 형식 검사 없이 `$n::uuid`로 넘기던 쓰기 경로 전부 수정 (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공 (commit 5cdf29e)
- 요약: 날짜·숫자·문자열·어휘 스윕과 같은 기준(철자가 아니라 연산 — 호출자가 준 id가 uuid 컬럼에 쓰이거나 그 id로 레코드를 조회)으로 훑었다. id가 들어오는 문은 셋인데 둘만 지키고 있었다 — 경로 id는 app.go의 라우터가, 질의 필터는 uuidParam이. 본문이 실어 나르는 id는 그냥 통과했고 열여섯 개 핸들러가 무방비였다. 문제는 메시지만이 아니라 답이 사람마다 달랐다는 것이다: 조회를 먼저 하는 핸들러는 실패한 질의를 거부로 읽어(supplierScopeAllowed는 어떤 오류든 false) 공급업체 번호를 붙여넣은 요청에 403 "데이터 접근 범위를 벗어났습니다"라고 답했고 — 존재하지도 않는 레코드를 볼 권한이 없다고 말한 셈 — company 스코프 계정은 그 조회를 건너뛰고 캐스트에 닿아 필드를 지목하지 않는 400을 받았다. 문서 업로드는 파일을 이미 디스크에 스트리밍·해싱한 뒤 insert가 도는 자리라 재시도마다 업로드를 다시 쓰게 했고, 포털의 납품·인보이스 등록(공급업체가 메일에서 id를 복사해 붙이는 곳)이 외부인이 지나는 문이다. rows.go에 validDateFields·validNumberFields·validTextFields·validEnumFields 옆에 validUUIDFields/validRecordID를 두고 스코프 검사보다 앞에 놓았다. MCP의 mcpObjects와 compare_suppliers도 캐스트 실패를 로그로 남기고 "도구를 실행하지 못했습니다"를 모델에 돌려주는 대신 도구 오류로 답한다. 검증: docker postgres:16-alpine으로 CI와 동일한 세 DSN을 걸고 `go test ./internal/... ./cmd/...` 전체 통과, gofmt·go vet 무결. 패키지를 파싱하는 TestEveryRequestRecordIDIsChecked(검사에서 뺀 erpVendorId의 사유는 notARecordID에 명시)와 열여섯 엔드포인트를 실제로 호출하는 TestEveryRecordIDOnAWriteIsChecked, 포털 쪽 TestAPortalWriteNamesAMalformedRecordID를 추가했고, createObject 가드와 uploadDocument 가드를 각각 되돌리면 셋 다 실패하는 것까지 확인했다. 덤으로 newPortalFixture가 감사 로그보다 계정을 먼저 지워 정리가 조용히 실패하고 다음 실행이 공급업체 번호 중복으로 죽던 것을 고쳤다(같은 정리 순서 함정이 세 번째다).
- 보류 아이디어:
  - CI의 go job이 `go test ./internal/...`만 돌려 `./cmd/...`를 빼놓음 — Makefile/README와 불일치 (가치 2 / 위험 1 / S)
  - 업무 객체의 status는 여전히 임의 문자열 — `status IN('completed','accepted','closed')` 같은 집계가 오타 하나로 조용히 빠짐(유형별 어휘가 어디에도 정의돼 있지 않아 위험은 큼) (가치 3 / 위험 3 / M)
  - 업무 객체의 data jsonb 블롭에는 어떤 검증도 없음 — 폼이 쓰는 키(품목·수량·단가)가 API로는 무제한이고 목록 응답이 블롭 전체를 반환 (가치 3 / 위험 2 / M)
  - 이메일 형식은 어디서도 검사되지 않음 — 초대·사용자 생성·공급업체 담당자가 "김구매"를 이메일로 저장하고, 그 주소로 알림이 나감 (가치 3 / 위험 1 / M)
  - `Makefile`의 VERSION(0.6.21)이 README의 릴리스 예시(0.7.26)와 어긋남 — 릴리스 문서 정합성 (가치 2 / 위험 1 / S)
- 릴리즈: v0.7.39 (2026-09-03)

## 2026-09-03
- 선택: 요청이 실어 나르는 이메일 주소를 형식·정규화 없이 email 컬럼에 넣던 쓰기 경로 전부 수정 (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공 (commit 6e88674)
- 요약: 날짜·숫자·문자열·어휘·레코드id 스윕과 같은 기준(철자가 아니라 연산 — 요청 값이 email 컬럼에 도달)으로 훑었고, 이메일은 애플리케이션 어디에서도 검사된 적이 없었다. 일곱 개 문이 무방비였다: 공급업체 등록·수정(대표 이메일과 taxInfo에 중첩된 세금계산서 이메일), 담당자 두 문, 포털 프로필·담당자, 관리자 계정 생성, 초대. 결함은 두 갈래다. (1) 형식 미검사 — "김구매"가 모든 표면에 주소로 저장됐고 폼의 type="email" 말고는 아무도 막지 않아, 포털·API·붙여넣기는 그냥 통과했다. 거절이 없으니 보고도 없고, 메일이 안 온 뒤에야 알게 된다. (2) 정규화 미적용 — users.email은 계정의 필드가 아니라 정체성인데 INSERT는 lower만 하고 trim은 하지 않고 login은 `WHERE email=$1`을 자기가 trim한 값으로 조회한다. 스프레드시트 셀에서 공백째 붙여넣은 주소는 어떤 비밀번호로도 열리지 않는 계정이 되고 증상은 영원히 "자격 증명이 올바르지 않습니다"뿐이며, 같은 공백이 ON CONFLICT(email)을 빗나가게 해 Keycloak 로그인이 두 번째 계정을 조용히 만든다. 초대 주소는 가입이 공급업체 레코드와 포털 계정 양쪽에 복사하는 값이라 오타 하나가 연락 안 되는 업체와 못 들어가는 계정 둘 다가 된다. rows.go에 validEmailFields/validEmail/isEmailAddress를 기존 다섯 검사 옆에 두었고, 로컬 파트는 의도적으로 ASCII로 제한했다(한글 로컬 파트는 한 칸 아래 상자에 들어간 이름이다). 검증: docker postgres:16-alpine으로 CI와 동일한 세 DSN을 걸고 `go test ./internal/... ./cmd/...` 전체 통과, gofmt·go vet 무결. 패키지를 파싱하되 요청 필드명이 아니라 statement의 컬럼 목록을 읽는 TestEveryStoredEmailIsAnAddress(주소 출처가 설정·로그인 잠금 키·OIDC 클레임·담당자 레코드·초대장인 다섯 문은 notFromTheCaller에 사유 명시)와, 엔드포인트를 실제로 호출하고 붙여넣은 주소로 만든 계정으로 로그인까지 해보는 TestEveryEmailOnAWriteIsAnAddress·TestAPortalWriteNamesAMalformedEmail을 추가했다. updateSupplier와 portalCreateContact 가드를 각각 되돌리면 셋 다 실패하는 것까지 확인했다. 덤으로 기존 TestInvitationExpiryStaysWithinTheWindowTheFormOffers가 케이스 라벨(공백 포함)을 그대로 로컬 파트에 넣던 것을 슬러그로 바꿨다.
- 보류 아이디어:
  - CI의 go job이 `go test ./internal/...`만 돌려 `./cmd/...`를 빼놓음 — Makefile/README와 불일치 (가치 2 / 위험 1 / S)
  - 업무 객체의 status는 여전히 임의 문자열 — `status IN('completed','accepted','closed')` 같은 집계가 오타 하나로 조용히 빠짐(유형별 어휘가 어디에도 정의돼 있지 않아 위험은 큼) (가치 3 / 위험 3 / M)
  - 업무 객체의 data jsonb 블롭에는 어떤 검증도 없음 — 폼이 쓰는 키(품목·수량·단가)가 API로는 무제한이고 목록 응답이 블롭 전체를 반환 (가치 3 / 위험 2 / M)
  - 전화번호·웹사이트도 형식 검사가 없음 — 길이만 볼 뿐이라 "내선 3번"이 전화번호로, 사내 위키 제목이 웹사이트로 저장됨 (가치 3 / 위험 1 / M)
  - `Makefile`의 VERSION(0.6.21)이 README의 릴리스 예시(0.7.26)와 어긋남 — 릴리스 문서 정합성 (가치 2 / 위험 1 / S)
- 릴리즈: v0.7.40 (2026-09-03)

## 2026-09-04
- 선택: 요청 값이 공급업체 거래 상태로 들어가는 쓰기 경로에 어휘 검사 추가, 그리고 폼과 API가 서로 다른 단어를 쓰던 것을 하나로 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공 (commit 71e618a)
- 요약: 날짜·숫자·문자열·리스크등급·레코드id·이메일 스윕과 같은 기준(철자가 아니라 연산 — 요청 값이 질의가 분기하는 컬럼에 도달)으로, 리스크 등급 스윕이 riskLevel만 가져가고 옆에 두고 온 status를 훑었다. 거래 상태는 라벨이 아니라 질의가 읽는 단어다 — 대시보드의 거래 가능 타일은 status='active'를, 심사 대기 타일은 status='screening'을 세고, MCP 추천 도구는 status IN('active','approved')만 추리며, 목록 필터는 철자를 그대로 비교한다. createSupplier·updateSupplier 둘 다 들어온 status를 길이 말고는 보지 않았다. 그리고 그 결함은 이미 제품 안에서, 가장 눈에 안 띄는 자리에서 일어나 있었다: 웹앱이 어휘를 세 번(목록 필터, 배지 라벨 맵, 편집 폼 드롭다운) 따로 적어 두었고, 편집 폼만 "registered"라고 적혀 있었다 — 저장소 어디에도 없는 철자다. 결과는 둘이고 두 번째가 나쁘다. (1) 편집 폼에서 등록을 고르면 "registered"가 저장돼 등록 필터에 영영 안 잡히고, 라벨 맵에 한국어가 없어 영문이 그대로 배지에 뜨며, 색도 warning이 아닌 기본색이 된다. (2) 포털 자가등록이 넣는 "registration"이 옵션에 없어서 select가 첫 옵션으로 떨어진다 — 자가등록 업체를 열어 전화번호만 고치고 저장하면 상태 칸을 건드린 적도 없이 후보로 되돌아간다. 누가 보고 있던 등록 큐에서 빠져 아무도 안 보는 목록으로 가고, 화면에는 아무 흔적도 없다. rows.go의 riskGrades 옆에 supplierStatuses를, web/src/status.ts에 같은 목록을 두고 필터·드롭다운·라벨 맵이 모두 그것을 읽게 했다. 검증: docker postgres:16-alpine으로 CI와 동일한 세 DSN을 걸고 `go test ./internal/... ./cmd/...` 전체 통과, gofmt·go vet 무결, 웹 테스트·빌드·린트 통과. 가드는 넷 — 패키지를 파싱하는 TestEverySupplierStatusIsInTheVocabulary, 요청 필드가 아니라 statement를 읽는 TestEverySupplierStatusInTheSQLIsInTheVocabulary, status.ts와 rows.go를 서로에게 묶는 TestTheStatusListTheFormOffersIsTheOneTheAPIAccepts, 엔드포인트를 실제로 호출하는 TestEverySupplierStatusOnAWriteIsInTheVocabulary — 여기에 웹 쪽 supplier-status.test.tsx가 "registration" 업체를 편집 폼에 띄워 그대로 저장한다. updateSupplier 가드를 되돌리면 파싱·통합 테스트가, status.ts에 "registered"를 되돌리면 교차 검사와 웹 테스트가 실패하는 것까지 확인했다.
- 보류 아이디어:
  - CI의 go job이 `go test ./internal/...`만 돌려 `./cmd/...`를 빼놓음 — Makefile/README와 불일치 (가치 2 / 위험 1 / S)
  - 업무 객체의 status는 여전히 임의 문자열 — `status IN('completed','accepted','closed')` 같은 집계가 오타 하나로 조용히 빠짐(유형별 어휘가 어디에도 정의돼 있지 않아 위험은 큼) (가치 3 / 위험 3 / M)
  - 전화번호·웹사이트는 길이만 볼 뿐 형식 검사가 없음 — "내선 3번"이 전화번호로, 사내 위키 제목이 웹사이트로 저장됨 (가치 3 / 위험 1 / M)
  - 업무 객체의 data jsonb 블롭에는 어떤 검증도 없음 — 폼이 쓰는 키(품목·수량·단가)가 API로는 무제한이고 목록 응답이 블롭 전체를 반환 (가치 3 / 위험 2 / M)
  - `Makefile`의 VERSION(0.6.21)이 README의 릴리스 예시(0.7.26)와 어긋남 — 릴리스 문서 정합성 (가치 2 / 위험 1 / S)
- 릴리즈: v0.7.41 (2026-09-04)


[← 대시보드](https://hkjang.github.io/aidev/)

{% endraw %}
