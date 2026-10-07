## 2026-10-06
- 선택: 붙여넣기 분석기 — 메일에 섞인 카톡 한 줄이 입력 전체를 가로채 앞부분을 통째로 버리는 문제 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `parseKakao`(`src/shared/kakao.ts`)가 카톡 머리줄 한 줄만 보고 형식을 `kakao` 로 확정한 뒤 첫 머리줄 앞의 줄을 말없이 버려, "메일 본문 + 고객이 카톡으로 보낸 한 줄" 을 붙여 넣으면 메일의 요청 목록이 요청 접수 입구에서 사라졌다. 이제 첫 머리줄 앞의 비어 있지 않은 줄을 훑어 내보내기 머리말(새 상수 `EXPORT_HEADER`)·`PC_DATE`·`MOBILE_DATE`·`MOBILE_SYSTEM` 이 아닌 글이 있으면 "깨끗한 내보내기가 아님" 으로 `null` 을 돌려 `stripMail`→`parseList`→`parseParagraphs` 로 떨어진다(프로덕션 코드 1개 파일, 정규식 `PC_LINE`·`MOBILE_LINE`·`HINT`·`ACK` 는 손대지 않았다). 검증: `npm run check`(tsc 3개) 통과, `npm run build` 뒤 `npm test` 로 Playwright 49개(unit 13·api 24·desktop 9·mobile 3) 전부 통과 — 기존 kakao 9개와 api-security 15개를 포함해 수정한 단정은 없다. 임시 탐침 시험으로 실제 HTTP(`POST /api/p/:pid/requests/import/parse`)를 때려 라우트 응답이 `format:"list"` + 메일 항목 2개(전에는 `kakao` + 1개)이고 진짜 내보내기는 `kakao` 그대로임을 확인한 뒤 탐침 파일은 지웠다. 그 탐침이 함정 하나를 잡았다: **api·desktop·mobile 묶음은 `dist/` 빌드 결과를 wrangler dev 로 띄워 돌기 때문에, 소스만 고치고 `npm run build` 를 다시 하지 않으면 라우트는 옛 코드를 돌린다**(unit 만 TS 를 직접 import 한다) — 처음 통과한 49개는 그래서 옛 번들을 본 것이었고, 다시 빌드해 49개를 다시 돌렸다.
- 실패 재현: `✘ 9 [unit] › 메일·그 밖의 글 › 카톡 머리줄 한 줄이 섞여도 메일 본문을 가로채지 못한다` → `Error: expect(received).not.toBe(expected) // Expected: not "kakao"` / `✘ 10 › 카톡 한 줄이 섞인 문단 글은 어느 줄도 버리지 않는다` → `Expected: "paragraphs" Received: "kakao"` (고치기 전 11 passed / 2 failed. 고친 뒤 13 passed.)
- 보류 아이디어: ① `list` 형식이 글머리표 아닌 줄을 말없이 버린다 — 섞인 메일에서 그 카톡 한 줄이 여전히 사라진다(의도된 설계·기존 테스트가 고정, 버린 줄 수를 알려 주는 쪽이 저위험) ② `MAX_INPUT` 자르기가 두 경로에서 다르게 읽힌다(라우트 400 거절 vs `normalize` 의 `truncated` — API 경로에서는 죽은 분기) ③ `parseList` 가 `bulletCount>=2` 를 요구해 항목 하나짜리 메일이 문단으로 떨어진다 ④ PDF 구조 검사 오탐(압축 스트림의 우연한 `/JS` 바이트, 위험 4) ⑤ 상태 전이 '무엇으로 대체되었는지' 연결을 api 시험으로 좁혀 확인.
- 과제서: 채택 — 결함·파일·수용 기준 1·2 가 코드와 정확히 맞았다. 다만 수용 기준 3("버려지는 줄이 없다")은 글머리표가 있는 그 입력으로는 끝까지 증명할 수 없다: 섞인 입력이 `list` 로 떨어지면 `parseList` 가 글머리표 없는 산문(= 섞여 들어온 카톡 한 줄)을 설계대로 버리고, 그 설계는 기존 테스트가 항목 3개로 고정하고 있다. 그래서 기준 3 은 글머리표 없는 섞인 입력(→`paragraphs`)으로 "모든 줄이 남는다" 를 증명하고, `list` 쪽 손실은 보류 아이디어 ①로 넘겼다.

## 2026-10-07
- 선택: 붙여넣기 분석기의 메시지 4,000자 절단을 알림에 반영 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 정규화·추출된 개별 메시지 본문이 4,000자를 넘는 경우를 기존 truncated 조건에 합치고, 화면에는 일부 메시지 또는 뒷부분 생략과 나눠 다시 붙여넣는 방법을 안내하도록 고쳤다(프로덕션 2파일, 시험 2파일, 커밋 09e7133). 세 형식의 3,999/4,000/4,001자·300/301개, 전체 입력 200,000/200,001자, 정규화·UTF-16·메타데이터·suggested 회귀를 검증했고 실제 import/parse 응답을 받은 경고 표시·해제와 기존 선택 접수도 확인했다. npm run check 및 npm run build 통과, npm test -- --project=unit tests/kakao.spec.ts 30 passed, npm test -- --project=desktop tests/ui-flow.spec.ts 9 passed, 마지막 npm test 122 passed (4.2m), git diff --check 통과 후 hkjang 작성자로 커밋했다.
- 실패 재현: 수정 전 새 단위 시험에서 `Expected: true` / `Received: false` (세 형식 4,001자 및 UTF-16 시험 4 failed / 26 passed); 수정 조건만 되돌린 재검증에서도 4,001자 3개가 다시 실패했다. UI 수정 전 실제 API는 truncated=true였지만 `Expected substring: "일부 메시지 또는 메시지 뒷부분을 생략했습니다"` / `Received string:    "내용이 길어 앞부분 300개 메시지만 보여 줍니다. 나머지는 나눠서 다시 붙여 넣어 주세요."`로 실패했다(1 failed / 7 passed / serial 후속 1 did not run).
- 보류 아이디어:
  - 목록의 글머리표 없는 산문 누락 안내 (가치 3 / 위험 3 / 작업량 M): 기존 정책을 유지하고 후속 회차에서 손실 안내 검토.
  - MAX_INPUT의 HTTP 거절·순수 함수 절단 계약 문서화 (가치 2 / 위험 1 / 작업량 S): 순수 함수 경계 시험만 이번에 추가, HTTP 경계와 문서화는 남음.
  - 긴 선행 공백 입력의 분석 지연 조사 (가치 3 / 위험 2 / 작업량 M): 초기 경계 fixture에서 13.8초/14.8초를 관찰했지만 정확한 병목은 미확정.
  - 붙여넣기 입력창의 200,000자 상한 안내 (가치 3 / 위험 1 / 작업량 S): 현재 maxLength만 있으며 실제 클립보드 초과 입력 동작은 후속 재현 필요.
- 과제서: 채택 — 개별 본문 slice 후 truncated 누락과 단정적인 300개 안내가 현재 코드 및 실패 시험에서 확인되어 지정한 4파일 범위로 수용 기준을 충족했다.

