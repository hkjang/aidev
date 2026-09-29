# PR 처리기 노트 2026-09-30-010956-nexabuilder-shepherd — nexabuilder PR #39
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-29-003529-nexabuilder-improve)
# 회차 노트 2026-09-29-003529-nexabuilder-improve — nexabuilder
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:35] base pinned — master@a3ca143
- [러너 00:35] autonomy release — 

## 구현 노트
- 수정 과제 판정: PR #38 의 CI 실패는 코드 원인이 **아니다**(3회차째 같은 결론). 다만 이전 회차의 논증 하나는 틀렸다 — #38 은 #37 의 `ListExportController` 변경을 cherry-pick 으로 품고 있어 "트리가 다른 두 브랜치" 가 아니다. 결정적 근거는 시간: 성공 22건이 470~611초인데 실패 2건은 39초·4초이고, 두 PR 모두 `build.gradle.kts`·`gradle/`·`.github/` 를 건드리지 않았다. 워크플로는 한 줄도 손대지 않았다(완화 금지).
- 그 자리를 실재 결함으로 채웠다: `text`(H2 CLOB) 값이 Excel 셀 한도 32,767자를 넘으면 POI 가 IllegalArgumentException → 400 이라 CSV·PDF 는 되는데 XLSX 만 실패했다. `exportXlsx` 만 `cellText()` 로 자르고 `stringify`(3형식 공유)는 그대로 뒀다. 프로덕션 1파일, 커밋 297a248.
- **확신 없는 곳**: (1) CI 실패의 정체는 **여전히 모른다** — 로그를 볼 수단이 없다(`gh` 미인증, `git ls-remote origin` 도 자격증명 없음, 비공개 저장소). 과금/러너 할당 같은 가설은 세웠으나 증명하지 않았고 코드에 반영하지도 않았다. (2) 잘린 셀을 사용자에게 알리는 것은 서버 WARN 로그뿐이다 — 파일 안에서는 말줄임표가 유일한 표시다. (3) 말줄임표 자리를 1자로 셌다(`…` 는 BMP 1 char). 서로게이트 페어가 `max-1` 경계에 걸리면 마지막 코드포인트가 반으로 잘릴 수 있다 — 테스트하지 않았고 한글·ASCII 데이터에서는 발생하지 않는다.
- 일부러 안 한 것: 워크플로 수정(금지), #37/#38 의 세 번째 cherry-pick 재제출(같은 접근 반복 금지 — ideas.json 에 사람 판단 항목으로 남겼다), `stringify` 에 자르기 넣기(CSV·PDF 를 같이 망가뜨린다), PDF 쪽 길이 처리(OpenPDF 에는 셀 한도가 없다).
- 다음 역할 주의: 신규 3건은 실제 H2 CLOB·POI·MockMvc 로 돌고 `@DirtiesContext` 없이 컨텍스트를 공유한다. `seedWithLongText` 는 32,767자짜리 값을 두 번 넣으므로 이 클래스만 몇 초 더 걸린다. 검증: 단일 클래스 16건 통과, `clean test` 612건(609+3)·0 fail·0 error·0 skip(5분59초), `bootJar -x test` 성공(6초).
- [러너 00:49] verify passed — 검증 1개 통과 (auto)

## 비평 노트
- 판정 approve(risk low, blocking 없음). 결함 실재를 POI 바이트코드로 확인했다 — `CellBase.setCellValue(String)` 에 `SpreadsheetVersion.getMaxTextLength()` 검사와 IllegalArgumentException 이 있고 `SXSSFCell.getSpreadsheetVersion()` 은 EXCEL2007, GlobalExceptionHandler:92-95 가 400 으로 바꾼다.
- 원장에 `- 실패 재현:` 줄이 없어 테스트 유효성을 직접 봤다. `xlsxExportCutsCellsToTheExcelLengthLimit` 는 `cellText()` 만이 만들 수 있는 결과(`hasSize(max)`+`endsWith("…")`)를 단언해 수정 전 반드시 실패한다. 나머지 2건은 수정 전에도 통과하는 가드다. 재실행 16건/0 fail(1.8초).
- 구현자의 미검증 항목 중 서로게이트 페어를 추적했다: 짝 없는 서로게이트는 UTF-8 인코더가 `?` 로 바꿔 쓰므로 예외·파일 손상이 아니다 — 차단 아님.
- 승인이어도 남는 우려: XLSX 만 값이 **조용히 잘린다**(사용자 표시는 말줄임표 하나, 나머지는 서버 WARN). 릴리즈 노트에 "전체 값이 필요하면 CSV" 한 줄이 필요하다.
- 못 본 것: 주석·커밋이 주장하는 "PDF 는 성공" 은 32k 초과 값으로 검증되지 않았다(PDF 테스트 없음 — 주장이 검증보다 넓다). PR #38 CI 실패 원인도 여전히 미해명이며 이 PR 은 그것을 건드리지 않았다.
- [러너 00:54] review approved — 리뷰 승인 (risk=low)
- [러너 00:54] pr created — https://github.com/hkjang/nexabuilder/pull/39
- [러너 00:54] ci failed — 성공이 아닌 검사: test + bootJar=failure
