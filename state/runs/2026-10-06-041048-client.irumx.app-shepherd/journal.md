# PR 처리기 노트 2026-10-06-041048-client.irumx.app-shepherd — client.irumx.app PR #1
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-06-033752-client.irumx.app-improve)
# 회차 노트 2026-10-06-033752-client.irumx.app-improve — client.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:37] base pinned — main@e541757
- [러너 03:37] autonomy release — 

## 정찰 노트
- 저장소는 커밋 3개짜리 신생이지만 완성도가 높다(시험 46개·보안 시험 15개·캡처 115장·문서 6종). 그래서 "공백 메우기" 보다 **읽어서 찾은 실제 결함** 을 골랐다: `parseKakao` 가 카톡 머리줄 한 줄만 보고 형식을 확정한 뒤 그 앞의 줄을 말없이 버린다 — 요청 접수의 유일한 입구에서 생기는 조용한 자료 손실이고, 순수 함수라 실제 프로덕션 함수로 테스트가 된다(대역 불필요).
- 제친 후보: CI 추가(가치는 높지만 `.github/workflows` 보호 경로 + 단위 시험이 wrangler dev 를 요구해 CI 에서 도는지 확인 불가), PDF 구조 검사 오탐(보안 검사를 느슨하게 하는 방향 — 위험 4), PNG 짧은 파일(읽어 보니 이미 올바르게 거절됨 → rejected).
- 추측으로 적은 것: 수용 기준 1 의 "지금 코드는 format==='kakao', messages.length===1" 은 **정규식을 눈으로 따라간 결과이지 실행해서 본 것이 아니다**. 구현자는 먼저 그 입력으로 실패하는 테스트를 만들어 현재 동작을 확인하고, 다르면 입력을 조정해 같은 결함을 재현할 것.
- 검증 미확인: 이 워크트리는 `node_modules` 가 비어 있고 이 세션에서 `npm ci` 실행이 허용되지 않아 어떤 명령도 돌려 보지 못했다. 명령은 `package.json`·`playwright.config.ts`·README 에서 확인한 것이다. `--project=unit` 만 골라도 `webServer`(wrangler dev + dist)가 필요하다는 점을 과제서에 적어 두었다.
- 조심할 것: `HINT`·`ACK`·머리줄 정규식을 함께 건드리면 `suggested` 단정 9개가 같이 흔들려 원인 분리가 안 된다. 이번 변경은 **형식 판정만**, 파일 2개(`src/shared/kakao.ts`, `tests/kakao.spec.ts`)로 끝낼 것. 파일 머리 주석(135줄 "내보내기 머리말은 버린다")도 같이 고쳐야 한다 — 이 저장소는 주석이 규칙 문서 역할을 한다.
- [러너 03:43] scout done — 붙여넣기 분석기 — 메일 안에 섞인 카톡 한 줄이 입력 전체를 가로채 앞부분을 통째로 버리는 문제 (가치

## 구현 노트
- 무엇을 왜: `parseKakao` 가 첫 머리줄 앞의 줄을 말없이 버리므로, 그 앞에 내보내기 머리말(`EXPORT_HEADER`)·`PC_DATE`·`MOBILE_DATE`·`MOBILE_SYSTEM` 이 아닌 글이 있으면 `null` 을 돌려 목록·문단으로 읽게 했다(프로덕션 파일 1개, 형식 판정만). 커밋 7716c3b.
- **중요 — 다음 역할이 꼭 알 것**: `npm test` 의 api·desktop·mobile 묶음은 **`dist/` 의 빌드 결과를 wrangler dev 로 띄워** 돈다(unit 만 TS 를 직접 import). 소스를 고친 뒤 `npm run build` 를 다시 하지 않으면 라우트는 **옛 코드를 그대로 돌린다**. 임시 탐침 시험으로 실제 HTTP 를 때려 보고 이것을 잡았다(고친 뒤에도 라우트가 `format:"kakao"` 를 돌려줬다). 다시 빌드한 뒤 라우트 응답이 `format:"list"` + 메일 항목 2개로 바뀌었고, 진짜 내보내기는 `kakao` 그대로였다. 탐침 파일은 커밋 전에 지웠다.
- 확신 없는 곳: ① 수용 기준 3 을 과제서의 입력(글머리표 2개)으로는 끝까지 증명하지 못했다 — `list` 로 떨어지면 `parseList` 가 글머리표 없는 산문(섞여 들어온 그 카톡 한 줄)을 설계대로 버린다. 그래서 "모든 줄이 남는다" 는 글머리표 없는 입력(→`paragraphs`)으로 증명했고, `list` 쪽 손실은 ideas.json 1번으로 넘겼다. ② 알아보는 머리말이 EXPORT_HEADER·날짜·모바일 알림뿐이라, PC 내보내기 맨 앞에 평문 알림 줄('…님이 …님을 초대하였습니다.')이 있는 진짜 내보내기는 이제 `list`/`paragraphs` 로 떨어진다 — 자료 손실은 없고 `author`·`at` 만 비는 열화다. 실제 내보내기 파일이 없어 그 줄 형태를 확인하지 못했고, 느슨하게 하면 결함이 되돌아오므로 일부러 엄격을 골랐다.
- 일부러 하지 않은 것: `PC_LINE`·`MOBILE_LINE`·`MOBILE_SYSTEM`·`HINT`·`ACK` 와 `MAX_INPUT`/`MAX_MESSAGES`·`truncated` 는 손대지 않았다(기존 단정 9개의 원인 분리를 위해). `requests.tsx` 는 빈 `author`/`at` 을 이미 `{m.author && …}` 로 다루므로 바꿀 것이 없음을 읽어서 확인만 했다.
- 환경 함정 둘: ① 이 머신의 `npx` 는 `/home/hkjang/node_modules/.bin/node`(v20.19.2)를 먼저 집어 wrangler 가 뜨지 않는다. `scripts/db-local.mjs` 가 조용히 실패해 빈 D1 로 서버가 떠 api 시험이 `no such table: verification` 500 으로 깨진다. 워크트리의 `node_modules/.bin/node` 심링크(gitignore 대상, 커밋 안 됨)로 우회했다 — 전역 설정은 건드리지 않았다. ② `npx playwright install chromium` 을 해야 desktop·mobile 12개가 돈다.
- 커밋에 글꼴 파일 3개(`public/fonts/*.woff2`, `src/client/styles/fonts.css`)가 함께 들어갔다: `scripts/subset-font.mjs` 가 `src/shared` 의 **주석 글자까지** 훑어 서브셋을 만들기 때문이다. 재실행해도 바이트가 같고(idempotent) HEAD 의 "글꼴은 소스와 같은 커밋에서 맞춘다" 관례를 지킨 것이다.
- [러너 03:57] brief accepted — 채택 — 결함·파일·수용 기준 1·2 가 코드와 정확히 맞았다. 다만 수용 기준 3("버려지는 줄이 없다")은 글머리표가 있는 
- [러너 03:58] verify passed — 검증 4개 통과 (auto)

## 비평 노트
- 구현 노트의 "확신 없는 곳 ②"(평문 알림이 앞에 있는 PC 내보내기)를 HEAD·main 양쪽을 node 로 직접 돌려 확인했고, "자료 손실은 없다" 가 틀렸다 — 긴 내보내기는 paragraphs 로 떨어져 문단마다 4000자로 잘리고(11,405자 → 4,044자) truncated=false 라 경고조차 없다.
- 더 흔한 입력에서 더 나쁘다: 소개 한 줄 + 글머리표 두 줄이 섞인 카톡 붙여넣기는 list 로 떨어져 **카톡 메시지 4개 전부**가 사라지고 글머리표 2개만 남는다(main 은 4개 모두 보존). 고치려던 손실이 같은 입구에서 커졌으므로 reject.
- 수리가 먼저 볼 파일: `src/shared/kakao.ts:110-115`(포기 대신 앞 블록을 index 0 메시지로 내보내거나, 무손실인 paragraphs 경로로만 넘길 것)과 `src/shared/kakao.ts:51`(EXPORT_HEADER 앵커 없음 — 원래 결함이 한 줄 입력에서 그대로 남음). 시험은 `tests/kakao.spec.ts:97-113` 에 "버려진 줄 없음" 단정을 추가해야 한다.
- 못 본 것: `npm run check`·`npm test`·`npm run build` 는 돌리지 않았다(결함이 순수 함수에서 재현되어 불필요). 범위·되돌리기·보안·법무는 깨끗하다 — 글꼴 3개 파일은 subset-font 재생성이고 revert 는 간단하다.
- [러너 04:03] review rejected — 리뷰 거절: src/shared/kakao.ts:110-115 첫 머리줄 앞에 알아보지 못하는 줄이 하나라도 있으면 카톡 분석을 통째로 포기하는데, 그 입력이 parseList 로 떨어지�
- [러너 04:03] pr created — https://github.com/hkjang/client.irumx.app/pull/1

## 수리 노트
- 지적 4개 전부 맞았다 — node 로 HEAD·main 을 직접 돌려 재현(list 2개 vs kakao 4개, paragraphs 4,044자 vs kakao 9,130자, EXPORT_HEADER 부분 일치로 안내 한 줄 소실, 시험이 손실을 안 잡음). 틀린 지적은 없었다.
- 고친 방법은 비평가가 제시한 쪽(A): `return null` 포기를 없애고 첫 머리줄 앞 블록을 author·at 빈 **머리 메시지 index 0** 으로 남긴다. 카톡 파싱을 유지하니 세 입력 모두 main 보다 더 보존한다. `EXPORT_HEADER` 는 줄 전체 앵커 + 맨 앞 두 줄 한정. 덤으로 4000자 컷에도 `truncated` 를 켰다.
- **방향이 바뀐 점**: 이 변경은 이제 "카톡으로 읽지 않는다" 가 아니라 "카톡으로 읽되 앞의 글을 버리지 않는다" 다. 그래서 시험 2개의 `format` 기대를 바꿨다(paragraphs/list → kakao). 단언을 느슨하게 한 게 아니라 `droppedLines()` 로 **더 센** 단언(버려진 줄 0개)으로 바꿨다 — 이전 시험은 format 과 글머리표 2개만 봤다.
- 확신 없는 곳: ① 메일 본문이 한 덩어리 머리 메시지가 되므로 글머리표별로 쪼개지지 않는다(손실은 없고 사람이 고른다). 쪼개려면 parseList 의 "산문은 버린다" 설계를 바꿔야 해 범위 밖으로 뒀다. ② `목록`(list) 경로 자체의 산문 손실은 그대로다 — 기존 시험 `tests/kakao.spec.ts:91` 이 그 동작을 고정하고 있어 손대지 않았고, 파일 머리 주석에 그 한계를 명시했다.
- 검증: `npm run build` 통과, `npm test` 52 passed(exit 0). 글꼴 3개 파일은 주석 글자가 바뀌어 subset-font 가 재생성한 것(관례대로 같은 커밋). 커밋 46bc6d2.

## 심사 노트
- 확인한 것: 새 시험 5개를 main·7716c3b(반려본)·HEAD 세 버전에 node 로 직접 돌렸다 — main 5개 전부 실패, 반려본 4개 실패, HEAD 15/15 통과. 거절 사유 4개가 모두 실행으로 재현되고 HEAD 에서 사라진다. 기존 단정 10개는 그대로 통과(회귀 없음). 경계도 돌려 봤다: 머리말 4000자 초과→`truncated=true`(main 은 조용히 소실), 빈 입력·공백만→무사, 날짜 줄만 앞에 있을 때 빈 머리 메시지 안 생김, MAX_MESSAGES 초과 시 경고 켜짐.
- 확인한 것: 호출자 2곳(`src/worker/routes/requests.ts:170`, `src/client/pages/requests.tsx:212·257`)이 빈 `author`/`at` 을 이미 다루고, 저장 경로 `cleanText(it.text, 4000)` 가 새 `MAX_TEXT` 와 맞아 2차 무언 절단이 없다. 글꼴 3개는 `npm run build` 첫 단계가 재생성하고 `verify-build.mjs` 는 바이트를 비교하지 않아 릴리즈 경로 위험 없음. 문서 수치(52개·unit·kakao 15개)를 시험 파일에서 세어 일치 확인. 보호 경로·마이그레이션·인증·권한·의존성 변경 없음, revert 는 파일 7개 2커밋으로 깨끗.
- 못 본 것: `npm run check`(tsc)·`npm run build`·api·ui 묶음을 돌리지 못했다(워크트리에 node_modules 없음, 전역 설치는 운영자 금지 사항). 대신 `/requests/import/parse` 를 때리는 api·ui 시험이 없고 라우트가 `c.json(parseConversation(text))` 한 줄임을 읽어 확인해 파급 범위를 닫았다. 타입은 추가가 `MAX_TEXT` 상수와 형이 정확히 맞는 객체 push 뿐이어서 눈으로만 봤다.
- 권고의 근거(merge): 거절 사유 4개가 전부 실행으로 고쳐졌고, 시험이 대상을 실제로 돌려 main 에서 실패하므로 약속을 진짜로 잰다. 범위는 형식 판정 1개 파일로 좁고, 세 입력 모두 main 보다 더 보존한다 — 같은 입구에서 손실이 커지던 반려 사유가 역전됐다.
- 남긴 참고(차단 아님): `src/client/pages/requests.tsx:268` 경고 문구가 "앞부분 300개 메시지만" 한 종류뿐인데 이제 `truncated` 가 메시지 하나의 4000자 컷에도 켜져 원인을 잘못 설명한다(main 은 그 경우 경고가 아예 없었으므로 열화는 아니다). `kakao.ts:123` 은 머리말 앞에 한 줄이 더 있으면 `seen<=2` 창이 밀려 '저장한 날짜 : …' 가 머리 메시지에 섞인다(소음, 손실 아님).
