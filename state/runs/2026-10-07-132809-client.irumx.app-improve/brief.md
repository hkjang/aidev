- 과제: 붙여넣기 분석기의 메시지 4,000자 절단을 알림에 반영 (가치 4 / 위험 1 / 작업량 S)
- 왜: `parseConversation`은 각 메시지를 4,000자로 자르지만 `truncated`에는 전체 입력 길이와 메시지 개수 초과만 반영해, 긴 요청의 뒷부분이 사라져도 사용자가 알 수 없다. 절단 플래그와 화면 안내를 함께 바로잡으면 사용자가 누락을 알고 내용을 나눠 다시 접수할 수 있다.
- 수용 기준:
  1) 전체 입력 200,000자 이하·메시지 300개 이하에서도 반환할 메시지 하나의 정규화·추출 후 본문이 4,000자를 넘으면 `truncated=true`; 본문은 기존처럼 앞 4,000자만 반환한다. `length`·`slice`의 현재 UTF-16 단위를 유지한다.
  2) 카톡·목록·문단 각각 3,999/4,000자 본문은 다른 제한에 걸리지 않을 때 false, 4,001자는 true. 메시지 수·형식·순서·index·author·at·suggested는 기존 규칙과 같다. 전체 입력 초과 및 301개 메시지의 기존 true도 유지한다.
  3) 실제 붙여넣기 화면에서 긴 메시지 하나를 나누면 생략 사실과 나눠 다시 붙여넣는 방법을 안내한다. 한 메시지인데 ‘앞부분 300개 메시지만’이라고 단정하지 않는다. 짧은 입력으로 다시 나누면 경고가 사라지고 기존 ‘고른 메시지만 요청으로’ 흐름은 통과한다.
  4) 단위 시험이 세 형식의 경계값, 전체 입력 한도 초과, 300/301개 메시지 경계를 증명하고, 브라우저 시험이 실제 API 응답을 받은 경고 표시·해제를 증명한다. 미수정 코드에서는 4,001자 플래그 단정이 실패해야 한다.
- 건드릴 파일:
  - `src/shared/kakao.ts:ParseResult, parseConversation` — 자르기 전 본문 길이를 검사해 기존 truncated OR 조건에 반영; 파일 머리 및 truncated 주석에 전체 입력/개수/개별 본문 제한을 설명. 반환 타입에 새 필드는 필요 없다.
  - `src/client/pages/requests.tsx:ImportRequests` — parsed.truncated 경고 문구만 모든 절단 원인에 맞게 변경. 예: ‘내용이 길어 일부 메시지 또는 메시지 뒷부분을 생략했습니다. 나머지는 나눠서 다시 붙여 넣어 주세요.’
  - `tests/kakao.spec.ts` — 경계값·회귀 시험 추가. list fixture는 글머리표 두 개 이상, kakao fixture는 머리줄로 바로 시작해 별도 형식 판별 문제와 섞지 않는다.
  - `tests/ui-flow.spec.ts:카카오톡 대화를 붙여 넣어 고른 메시지만 요청으로` — 기존 시험 앞에서 긴 메시지의 경고 → ‘처음부터 다시 붙여넣기’ → 기존 짧은 fixture에서 경고 없음 확인. 실제 import/parse 응답을 사용하고 기존 접수 검증은 유지한다.
  - 합계 4개 파일, 프로덕션 2개. 서버 라우트 수정 불필요.
- 검증 명령:
  - 준비: Node >=22.12.0, `npm ci`; 브라우저가 없을 때 `npx playwright install chromium`.
  - `npm run check`
  - `npm run build`
  - `npm test -- --project=unit tests/kakao.spec.ts`
  - `npm test -- --project=desktop tests/ui-flow.spec.ts` — 이 파일은 serial이며 앞 시험에서 pm/pid를 만들므로 `-g`로 붙여넣기 시험만 골라 실행하지 않는다.
  - 마지막으로 `npm test` 전체 회귀 한 번. 앞 검증 이후 변경이 없다면 성공한 전체 검사를 반복하지 않는다.
  - 의존성 없이 빠른 회귀 확인(저장소 루트, 현재 Node v22.23.1에서 실행 가능함을 확인):
    `node --experimental-strip-types --input-type=module -e 'import assert from "node:assert/strict"; import {parseConversation} from "./src/shared/kakao.ts"; const r=parseConversation("가".repeat(4001)); assert.equal(r.messages[0].text.length,4000); assert.equal(r.truncated,true);'`
    마지막 true 단정은 현재 코드에서 실패하고 수정 후 통과해야 한다. 이것만으로 정식 시험을 대체하지 않는다.
- 위험과 피할 것:
  - 카톡 감지, 목록의 산문 무시, 단일 번호 항목 처리, HINT/ACK/NOISE, suggested 판단 대상, 4,000/200,000/300 제한값, API의 200,000자 초과 400 응답은 그대로 둔다. 자동 분할·새 API 필드·원문 저장 정책 변경은 제외한다.
  - auth·승인 상태·migrations·workflows·PDF 검사를 건드리지 않는다. 시험 환경 때문에 전역 Node나 프로젝트 실행 스크립트를 함께 고치지 않는다.
  - 이전 회차의 혼합 메일 가로채기 수정과 추가 시험은 현재 base 155a4ee에 없다(EXPORT_HEADER 없음, kakao 시험 9개). 이전 회차가 review-pending이라는 기록과 일치할 수 있으나 병합 상태 원인은 미확인. 그 수정의 재구현/되돌리기/체리픽은 이번 범위가 아니다.
  - Playwright는 unit에도 webServer를 띄우고 dist를 사용한다. 빌드 후 시험 소유의 8798/8799 서버가 옛 번들인지 확인한다. 남의 서버를 종료하지 않는다. npx의 상위 디렉터리 Node v20 선택은 과거 함정이며 현재 재현은 미실시; 실제 선택 버전을 확인해 로컬 실행 환경으로 대응한다.
- 차선 후보: `MAX_INPUT`의 HTTP 거절과 순수 함수 절단 계약을 문서·경계값 시험으로 명확히 하기 — 현재 과제가 이미 다른 변경으로 해결되었을 때만. 두 정책을 억지로 통일하지 말고 200,000자 이하 허용/초과 HTTP 400 및 순수 함수 truncated 동작을 유지한다(가치 2 / 위험 1 / 작업량 S).

확인한 근거와 정찰 검증
- `src/shared/kakao.ts:parseConversation`의 `text: m.text.slice(0, 4000)` 뒤 반환은 `truncated: truncated || cut`뿐이다. `src/worker/routes/requests.ts` 246~253행은 분석 결과를 그대로 반환한다.
- Node 직접 실행으로 세 형식 모두 3,999/4,000자는 같은 길이·false, 4,001자는 4,000자·false임을 관찰했다. 300개는 false, 301개는 300개·true, MAX_INPUT+1은 true임도 확인했다. 브라우저/HTTP 실행은 미확인.
- `npm run check`는 실제 시도했지만 `tsc: not found`(종료 127). node_modules가 없어 빌드·Playwright는 미실행이다. 정찰의 쓰기 경계를 지키기 위해 의존성 설치는 하지 않았다.
- README와 docs/qa.md의 ‘101개 통과’는 개발 당시 기록이며 이번 정찰의 통과 결과가 아니다.

대안 비교와 선택 근거
- 최소안(선택): 기존 boolean에 개별 본문 절단을 합치고 일반적인 경고를 표시. 프로덕션 2파일, API 호환성 유지. 사용자가 잘렸음을 알고 다시 나눌 수 있다는 전제가 핵심이다.
- 확장안: 절단 사유·메시지별 잘린 문자 수를 반환하고 항목마다 강조. 더 정확하지만 API·화면 계약을 늘려 이번 45분 범위를 넘기므로 보류.
- 처리 정책 변경안: 긴 메시지를 거절하거나 자동 분할. 누락 자체는 막지만 선택 수·제목·저장 정책이 달라져 제외.
- 현상 유지: 추가 작업은 없으나 잘린 요청이 완전한 원문처럼 보이는 재현된 문제를 남기므로 선택하지 않는다.

구현 순서와 점검 지점(모두 미착수)
1. 범위 확인·재현: 위 Node 명령의 false/true 단정 실패를 확인. 예상과 다르면 과제서부터 수정한다. 사람 승인 점검 없음.
2. 분석기+단위 시험을 한 묶음으로 수정: 경계값 시험이 수정 전 실패함을 확인한 뒤 구현하고 `npm run check`, `npm run build`, unit 명령으로 증명. 실패 상태로 다음 단계에 넘어가지 않는다. 사람 승인 점검 없음.
3. ImportRequests 안내+기존 UI 시험을 한 묶음으로 수정: 다시 `npm run build`, desktop 파일 전체 실행으로 경고 표시·해제 및 접수를 확인. 사람 승인 점검 없음.
4. `npm test` 전체 실행, `git diff --check`와 파일 범위 점검. 결과를 회차 노트에 기록. 범위 확대 없이 구현자 단계 종료(릴리즈는 후속 역할).

작업량 산정(pmo 절차)
- 근거: 읽은 함수·화면·기존 시험에 대한 bottom-up 추정. 환경 준비/재현 4~6분, 분석기·단위 시험 7~9분, 안내·브라우저 시험 5~7분, 빌드/검증/기록 12~16분 = 기본 28~38분.
- 알려진 불확실성 예비 5분: npx Node 선택, 서버 준비 또는 브라우저 설치 지연. 합계 33~43분은 주관적 중간 신뢰 범위이며 통계적 신뢰구간/완료 보장이 아니다. 네트워크 설치·전체 시험이 16분 이내라는 미확인 가정에 민감하다.
- 유사 추정: 이전 회차도 순수 분석기 1파일+경계 시험으로 끝났으나 실제 소요 시간이 없어 수치 교차 검증은 불가. 이번에는 UI 1파일이 추가된다.
- 관리 예비는 이번 범위에 배정하지 않음(0분); 다른 결함·환경 영구수정은 보류 아이디어로 남긴다. 환경 준비 뒤 시간이 범위를 넘으면 재산정하고 검증 미완료를 명시한다.
- 적용 스킬: 로컬 headcount의 pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration SKILL.md를 읽음. 전용 Skill 도구는 세션에 없어 파일로 읽었다. 외부 원가/편익 수치는 사용하지 않았다.
