# 회차 노트 2026-09-24-020423-kanpic-improve — kanpic
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 02:04] base pinned — main@0f9f926
- [러너 02:04] autonomy release — 

## 구현 노트
- 과제서의 15자리 공용 픽스처를 만들다 보니 그 경계는 파일 문과 클립보드 문이 이미 일치했고, 대신 앞자리 0 이 갈렸다(`00123` 이 업로드로는 글자, 붙여넣기로는 123). `decomposeNumberText` 에 이미 있던 "열여섯 자리 넘는 번호" 가드 옆에 같은 꼴의 가드 한 줄(`/^[+-]?0\d+$/`)을 얹고, `testdata/incoming-number.json` 으로 두 문을 붙들었다.
- **확신 없는 곳**: (1) 이것은 제품 판단이 섞여 있다 — 엑셀·구글 시트는 붙여넣기의 `00123` 을 123 으로 바꾼다. kanpic 은 파일 가져오기에서 이미 반대로 정해 두었고 그쪽에 맞췄다. (2) 데이터 정리도 함께 좁아졌다(`looksLikeNumberStoredAsText('00123')` 이 이제 false) — 붙여넣기로 지킨 것을 정리가 도로 123 으로 고치면 뜻이 없어서인데, `007` 을 정말 7 로 바꾸고 싶던 사람은 이제 손으로 쳐야 한다. (3) 브라우저 실제 붙여넣기(E2E)는 돌리지 않았다 — jsdom 의 `materializePaste`·`parseClipboardHtml` 까지만 통과시켰다.
- **일부러 하지 않은 것**: `CanvasGrid.parsedValue` 의 `Number()` 빠른 길은 그대로 두었다(친 `007` 은 7). 사람이 그 자리에서 적는 값은 다른 문이고, 건드리면 오래된 입력 동작이 통째로 바뀐다. `compareLists.looksLikeIdentifier` 도 두었다 — `007.5` 를 번호로 보아 두 문과 갈리지만 목록 키 맞추기는 다른 계약이고 증거가 없다. 두 가지 모두 ideas.json 에 남겼다.
- **다음 역할이 조심할 것**: Go 쪽 픽스처 테스트는 `internal/importexport` 에 있고 `../../testdata/incoming-number.json` 을, 웹 쪽은 `web/` 기준 `../testdata/...` 를 읽는다 — 파일을 옮기면 양쪽 경로를 함께 고쳐야 한다. 픽스처는 **두 문이 겹치는 평문 정수·소수만** 담는다(지수는 파일 쪽만, 자릿점·통화·백분율은 클립보드 쪽만 수로 읽는다). 사례를 더할 때 그 밖의 것을 넣으면 두 파서를 합치라는 압력이 생긴다 — 합치지 말 것. USER_GUIDE.pdf 는 다시 구웠다(1.83MB, 이전 1.85MB).
- [러너 02:15] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인: 구현자가 의심한 세 자리를 다 열었다 — LEADING_ZERO_NUMBER 와 서버 HasSignificantLeadingZero 의 겹치는 범위 값 일치, 정리(convertTextNumbers) 축소가 numberEntry.agreement.test.ts 로 붙들려 있는 것, 새 테스트가 가드 없이는 실제로 깨지는 것(PLAIN_INTEGER 가 `00123`→123). web 512테스트·관련 Go 4패키지 통과, 새 Go 픽스처는 -count=1 로 직접 실행.
- 못 본 것: E2E(실제 브라우저 붙여넣기), DB 통합, `npm run build`, USER_GUIDE.pdf 안의 글자(바이너리라 markdown 과 맞는지 확인 못 함).
- 거절 사유는 코드가 아니라 docs/USER_GUIDE.md:384 한 문장이다. (1) "열여섯 자리가 넘는" — 실제 가드는 15자리 초과라 열여섯 자리는 이미 글자다. 같은 문서 254·350·529줄은 "열다섯 자리를 넘는" 으로 맞게 적혀 있다. (2) "CSV·XLSX로 가져올 때와 같은 규칙" — XLSX 는 service.go:1134-1156 에서 자릿수 가드를 일부러 대지 않아 긴 번호가 수로 뭉개진다. 앞자리 0 절만 XLSX 에 맞다.
- 수리가 먼저 볼 파일: docs/USER_GUIDE.md:384 (문장 두 곳 수정) → docs/USER_GUIDE.pdf 재생성(scripts/generate_pdf.js, web/node_modules 있음).
- 승인해도 남을 우려: 정리에서 `007`→7 이 사라진 것은 사용자에게 보이는 기능 축소다. 릴리즈 노트에 적고, USER_GUIDE.md:529 의 정리 설명에도 앞자리 0 이야기를 한 줄 넣는 편이 좋다.
- [러너 02:19] review rejected — 리뷰 거절: docs/USER_GUIDE.md:384 새로 적은 문장이 자릿수 한도를 한 자리 틀리게 말한다. 실제 가드는 web/src/lib/spreadsheetNumber.ts:132 의 significantDigits(digits)>15

## 수리 노트
- 지적 두 개 다 맞았다. 틀린 것은 없었다 — 384줄의 "열여섯 자리" 는 실제 가드(>15)보다 한 자리 느슨하게 적혀 있었고, "CSV·XLSX 와 같은 규칙" 은 XLSX 에 거짓이다. 실제로 돌려 확인했다: `parseXLSXValue("12345678901234567890", CellTypeUnset)` → `float64 1.23...e+19`, 같은 함수로 `"007"` → `"007"`.
- 고친 방법: 384줄에서 "열다섯 자리를 넘는" 으로 바로잡고, 같은 규칙인 곳(CSV·TSV)과 다른 곳(XLSX 는 앞자리 0만 같고 형식 없이 적힌 긴 번호는 수로 들어올 수 있다)을 갈라 적었다. 긴 번호가 걱정되면 CSV 로 올리라는 안내를 한 줄 붙였다.
- 비평이 남긴 우려도 받았다: 529줄 정리 설명에 앞자리 0 정수는 바꾸지 않는다는 한 줄을 넣었다. 이 변경이 실제로 좁힌 동작인데 문서에 없었다. 코드는 건드리지 않았다.
- 여전히 확신 없는 곳: (1) USER_GUIDE.pdf 를 다시 구웠지만(1.858MB) 이 기계에 pdftotext 류가 없어 **PDF 안의 글자**가 markdown 과 맞는지 확인하지 못했다 — 이전 회차와 같은 공백이다. (2) E2E·DB 통합은 돌리지 않았다. (3) 529줄에 붙인 "정말 수로 만들려면 그 칸에 직접 쳐 넣으세요" 는 `CanvasGrid.parsedValue` 의 `Number()` 빠른 길에 기대는데, 그 경로는 브라우저에서 재현하지 않았다(384줄이 이미 같은 말을 하고 있어 새 약속은 아니다).
- [러너 02:24] repair done — # 수리 요약 (시도 1)  지적 두 개 다 맞았다. (1) `docs/USER_GUIDE.md:384` 이 "열여섯 자리가 넘는" 이라 적었으나 가드는 `spreadsheetNumber.ts:135` 의 `significantDigits(

## 비평 노트 (머지 심사)
- 확인: `LEADING_ZERO_NUMBER=/^[+-]?0\d+$/`(spreadsheetNumber.ts:86)를 `HasSignificantLeadingZero`(number.go:37)와 값별로 맞춰 픽스처 범위에서 일치함을 봤다. 새 테스트는 진짜 문을 지난다(Go 는 `Parse("번호.csv")`, 웹은 `materializePaste`·`parseClipboardHtml`) — 가드를 지우면 `00123` 이 123 이 되어 둘 다 깨진다. 돌린 것: go build/vet/gofmt, go test 3패키지 -count=1, web 512테스트, check-release-docs(v0.252.0)·check-commit-identities 모두 통과.
- 수리가 고친 문서 두 줄도 코드로 재확인했다 — 가드는 `significantDigits>15` 라 "열다섯 자리를 넘는" 이 맞고, `parseXLSXValue`(service.go:1134-1156)는 앞자리 0 만 거르고 자릿수 가드를 대지 않으므로 XLSX 를 갈라 적은 것이 맞다.
- 못 본 것: E2E·DB 통합 미실행, USER_GUIDE.pdf 안의 글자(서브셋 폰트라 스트림에서 한글이 안 뽑힌다 — 다만 markdown 을 바로잡은 e689082 에서 함께 재생성됨). `web/e2e/flash-fill.spec.ts:127` 이 `'007'` 을 쓰지만 문자열 보존만 기대하므로 이번 축소로 깨질 방향이 아니다.
- 승인해도 남는 우려(릴리즈 노트): ① 붙여넣기 `00123` 이 엑셀·구글 시트와 반대로 글자로 남는다 — 제품 판단이니 사용자에게 알린다. ② 데이터 정리가 `007`→7 을 더 이상 하지 않는다(보이는 기능 축소, 대체는 직접 입력). 가져오기 미리보기 경고 수는 어긋나지 않는다 — service.go:1114 가 `00123` 을 애초에 세지 않는다.
- 작은 것(막지 않음): `clipboardNumber.fixture.test.ts:17` 이 CWD 기준 `../testdata/...` 를 읽어 vitest 를 `web/` 밖에서 돌리면 깨진다. 픽스처 최소 개수가 Go 25 이상 / 웹 26 이상으로 어긋나 있다. 보안·법무 차단 사유는 없다(인증·비밀값·외부 요청·의존성 무관, 개인정보 신규 수집 없음).
- [러너 02:29] review approved — 리뷰 승인 (risk=low)
- [러너 02:29] pr created — https://github.com/hkjang/kanpic/pull/32
- [러너 02:38] ci passed — 검사 2개 모두 success
- [러너 02:38] merge done — e689082
- [러너 02:50] release published — v0.253.0
- [러너 02:52] assets verified — v0.253.0 자산 2개 (이전 v0.252.0: 2)
