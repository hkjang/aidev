# PR 처리기 노트 2026-09-30-020049-nexabuilder-shepherd — nexabuilder PR #39
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

## 수리 노트
- **지적(CI 실패)은 틀렸다 — 재현되지 않는다.** CI 가 죽은 바로 그 커밋 `297a248`(트리 clean)에서 CI 와 동일한 두 명령이 모두 통과했다: `test` → 612건/0 fail/0 error/0 skip(5분54초), `bootJar -x test` → 성공(6초, jar 생성). 고칠 코드 결함을 찾지 못했으므로 **커밋하지 않았다**.
- 코드 밖이라는 결정적 근거 두 가지를 새로 확보했다: (1) 잡이 **7초**에 죽는데 이 프로젝트는 Gradle 콜드 스타트만 12초다(이번 세션 실측) — `gradlew test` 는 시작도 못 했다. (2) 넘겨받은 실패 로그가 **비어 있다** — 스텝 출력이 아예 없다. 덤으로 `git diff origin/master...HEAD -- .github/` 는 빈 diff 이고 워크플로는 09-13 이후 불변이다(그 뒤 CI 다수 성공). 워크플로는 한 줄도 안 건드렸다.
- 비평이 추론으로만 확인했던 "테스트가 수정 전 반드시 실패한다" 를 **실제로 돌려 증명**했다: `cellText()` 본문을 `return value;` 로 되돌리니 `xlsxExportCutsCellsToTheExcelLengthLimit` 가 `16 tests completed, 1 failed`, 복원 후 16건 전부 통과. 단언을 느슨하게 하거나 테스트를 지운 곳 없음.
- NPE 가능성도 닫았다: `resolveColumns` 의 `label` 은 `String.valueOf(...)` 라 항상 non-null, `stringify` 는 null 에 `""` 를 준다 — 새 `cellText()` 호출부 둘 다 null 을 받을 수 없다.
- **여전히 확신 없는 곳**: CI 가 왜 7초에 죽는지는 끝내 모른다(`gh` 미인증·비공개 저장소로 로그 접근 불가). 인프라(과금/러너) 가설은 미증명이며 사람이 Actions 로그를 직접 열어야 한다. 앞 회차가 남긴 "XLSX 만 조용히 잘림", "PDF 는 32k 초과로 미검증" 은 이번에도 그대로다.
