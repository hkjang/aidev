# 회차 노트 2026-10-05-211728-umm-improve — umm
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:17] base pinned — main@6aaf940
- [러너 21:17] autonomy release — 

## 정찰 노트
- 최근 네 회차가 Go presentation 패키지와 npm 실행 환경만 파서 verify-failed 가 반복됐으므로, 이번엔 손대지 않은 프런트엔드 i18n 쪽에서 사용자가 바로 보는 결함을 골랐습니다 — `CanvasPage.tsx:2130` 의 영어 리터럴 대체 이름이 머리글과 내려받기 파일 이름 다섯 자리에 퍼집니다(사전에 `'내 공간'` 이 이미 있어 en.ts 수정이 없고, 프로덕션 2파일).
- 제친 후보: make test-go 직렬화(격리 DB 경합 재현이 선행이라 S 가 아님), Node 선언 드리프트(릴리즈·빌드 경로 — 운영자 규칙 4번), outline 이스케이프(렌더러 확인 선행 + escapeLine 반입 금지), 내보내기 이음매(PR #149 사람 반려 접근과 같은 자리 — 규칙 1번).
- 과제서에서 추측으로 남긴 것: ① 서버가 공백뿐인 공간 이름을 허용하는지 **미확인**(그래서 수용 기준을 함수 계약으로만 적었습니다), ② `spaceDisplayName` 이라는 새 모듈 이름·시그니처는 제안이며 구현자가 바꿔도 됩니다, ③ 테스트가 `t('내 공간')` 호출부까지 덮지는 못합니다(한 줄짜리 호출부는 읽어서 확인). 확인한 것: `api.ts:34 export interface Space`, `CanvasPage.tsx:257 useState<Space[]>`, `en.ts:34 '내 공간': 'My space'`, `en.ts:14 'My Space': 'My Space'`(AppLayout 이 아직 쓰는 항등 항목).
- 구현자가 조심할 것: `AppLayout.tsx:61` 을 같이 고치면 `:273` 의 `t(label).split(' ')[0]` 때문에 모바일 탭이 `내` 한 글자가 됩니다 — 이번 범위 밖입니다. `anchor.download` 세 곳의 새니타이즈 부재도 같이 고치지 마세요(별 과제).
- 검증은 반드시 `npm ci --prefix web` 를 먼저 돌리세요. 설치 없는 체크아웃에서는 `npm test` 가 exit 1 이며, 이것은 제품 결함이 아닙니다(지난 두 회차가 이것을 Node 버전 문제로 오진했습니다).
- [러너 21:23] scout done — 이름 없는 공간의 대체 이름이 한국어 UI 에서도 영어 `My Space` 로 뜨고 내려받기 파일 이름에까지 들어간�

## 구현 노트
- `CanvasPage.tsx:2130` 의 대체 이름이 영어 리터럴 `'My Space'` 라서 한국어 UI 의 머리글과 내려받기 파일 이름 다섯 자리에 영어가 샜다. 새 순수 모듈 `web/src/lib/space-name.ts` 의 `spaceDisplayName(spaces, activeSpace, fallback)` 로 선택 로직을 빼고(`trim()` 후 비면 대체), 호출부가 `t('내 공간')` 를 넘긴다 — 프로덕션 2파일, 시험 1파일.
- `t`/`translate` 를 함수 안에서 부르지 **않았다**: 사용자가 지은 이름이 사전 키(`en.ts:158 '생각 공간'`)와 겹쳐도 번역되면 안 되기 때문. 그 경우를 en 로케일 시험으로 못 박았다.
- **확신 없는 곳**: ① 시험이 `CanvasPage.tsx:2131` 호출부 자체를 실행하지 않는다(그 한 줄은 읽어서·typecheck 로만 확인 — `activeName` 을 쓰는 여섯 자리의 렌더 결과를 돌려 본 것이 아니다). ② 서버가 공백뿐인 공간 이름을 실제로 허용하는지 **확인하지 않았다** — 공백 분기는 함수 계약이며 서버 동작 주장이 아니다. ③ Playwright/e2e 는 돌리지 않았다.
- **일부러 안 한 것**: `AppLayout.tsx:61` 의 `'My Space'`(모바일 탭이 `t(label).split(' ')[0]` 로 첫 단어만 쓰므로 `내` 한 글자가 된다), `anchor.download` 세 곳의 새니타이즈 부재, `en.ts` 수정(`'내 공간'` 이 이미 있고 `'My Space'` 항등 항목은 AppLayout 이 쓴다), 버전 올리기.
- **다음 역할이 조심할 것**: 검증 전에 `npm ci --prefix web` 를 먼저 돌려야 한다(설치 없는 체크아웃에서 `npm test` 는 exit 1 이고 제품 결함이 아니다). 기준선은 20파일/218시험 → 이제 **21파일/225시험**. `lint` 경고는 전부 기존 파일의 것이고 새 파일 2개는 무경고. Go 는 0줄이라 돌리지 않았다(DB 통합도 실행하지 않음).
- [러너 21:27] brief accepted — 채택 — 과제서의 근거가 지금 코드와 전부 맞았습니다(`CanvasPage.tsx:2130` 의 영어 리터럴, `en.ts:34 '내 공간': 'My space'` 이미
- [러너 21:27] verify passed — 검증 17개 통과 (auto)

## 비평 노트
- 확인함: diff 3파일 전부, `t === translate`(i18n/index.tsx:69,86)라 시험의 `translate('내 공간')` 단언이 호출부 번역을 대변함, `spaces` 네 갱신 경로가 전부 서버 응답이고 오프라인 큐 합성이 없어 `?.name.trim()` 이 안전함, `activeName` 여섯 사용처(머리글·내려받기 넷·PresentationModal placeholder — 서버로 전송되지 않음), e2e 가 캔버스 머리글을 단언하지 않음. 돌림: test 21파일/225시험, typecheck, lint(경고 50 전부 기존 파일), check-i18n 1060키.
- 못 봄: Playwright e2e, 실제 브라우저에서의 렌더, Go/DB(0줄 변경이라 생략).
- 승인이어도 남는 우려 ①: 시험이 `CanvasPage.tsx:2131` 을 실행하지 않아 그 한 줄만 되돌려도 전부 초록이다. 이 저장소에 `pages/*.test.tsx` 가 아예 없어 거절 사유로 삼지 않았다 — 다음 회차가 캔버스에 시험 발판을 놓으면 이 틈이 닫힌다.
- 승인이어도 남는 우려 ②: 릴리즈 노트에 "한국어 UI 에서 My Space 가 사라졌다" 고 쓰면 거짓이다. `AppLayout.tsx:61`(내비 라벨)과 `internal/store/store.go:628`(서버가 짓는 첫 공간의 실제 이름)은 그대로다. 고친 것은 머리글·내려받기 대체값 한 자리.
- 승인이어도 남는 우려 ③: 커밋 메시지에 없는 동작 변화가 하나 있다 — 앞뒤 공백이 있는 이름이 이제 trim 되어 표시·파일 이름에 들어간다. 그리고 한국어 독자의 기본 파일 이름이 처음으로 비ASCII(`umm-내 공간.md`)가 된다.
- [러너 21:30] review approved — 리뷰 승인 (risk=low)
- [러너 21:31] pr created — https://github.com/hkjang/umm/pull/166
- [러너 21:49] ci passed — 검사 1개 모두 success
- [러너 21:49] merge done — 17e1741
