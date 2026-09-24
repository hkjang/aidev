# 회차 노트 2026-09-24-044412-aiportal-front-improve — aiportal-front
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 04:44] base pinned — main@fb38bdd
- [러너 04:44] autonomy release — 

## 정찰 노트
- 게시판 상세 3형제를 골랐다: 지난 회차에 이 항목을 막던 '빈 상태 기대 계약 미확인' 이 이번에 해소됐다 — 같은 저장소의 상세 조회 표준형 `SupportOcrDetail.getOcrs`(:39-42, :55-58)가 openAlert 계약을 이미 확정해 두고 있어 문구를 새로 발명할 필요가 없다. 목록 3형제(NoticeList/LibraryList/RequestList)도 같은 결함이지만 참고할 형제 계약이 없어 뒤로 미뤘고, 즐겨찾기 ButtonSetting 은 근거가 틀려 rejected 로 내렸다.
- 추측으로 적은 것: ①`npm test` 기준선 32파일 516테스트는 직전 회차 기록이고 정찰이 직접 돌리지 않았다(node_modules 비어 있음) ②신규 스펙의 라우터 구성은 `sidemenuHistoryFailure.spec.js` 하네스를 본뜬 설계일 뿐 실제로 작성해 보지는 않았다.
- 조심할 것: `toast` 가 세 파일에 이미 import 돼 있어 손이 가지만 상세 화면 계약은 `openAlert` 다. 목록 화면의 현 동작(안내 없음)을 근거로 삼지 말 것. 세 파일은 문자 단위로 거의 같으니 한쪽만 고치지 말 것 — 이 저장소 결함 대부분이 형제 비대칭이었다.
- 라우터 params 가 비면 `boardId=undefined` 로 요청이 다른 모양으로 나가 Red 가 조용히 사라진다. 첫 케이스에서 `/board/{id}` 요청이 실제로 나갔음을 단정해 고정할 것.
- A(릴리즈)는 20회째 무변경이 정답이다. 재조사하지 말고 버전 파일·태그·노트·커밋을 일체 만들지 말 것.
- [러너 04:48] scout done — 릴리즈 버전 결정 입력 부재 (가치 5 / 위험 2 / 작업량 S — 단, **저장소 안에서 수행 불가**)

## 구현 노트
- 무엇을 왜: 상세 3형제(`NoticeDetail`/`LibraryDetail`/`RequestDetail`)의 `fetchBoardDetail` 에 `isSuccess` 분기와 `catch` 의 `'COM'` 가드·`openAlert` 를 넣었다. 실패가 자리표시자 제목으로 정상 렌더돼 사용자가 실패를 알 수 없었다. 문구·수단은 형제 `SupportOcrDetail.getOcrs` 계약 그대로(새로 발명 없음).
- **확신 없는 곳**: ①`catch` 문구만 마침표가 없다(`'…실패하였습니다'`) — 형제 `SupportOcrDetail`/`SupportSttDetail` 이 업무 실패엔 마침표, `catch` 엔 없는 형태를 쓰고 있어 그 불일치까지 그대로 따랐다. 의도적 모방이지 오타가 아니다. ②`RequestDetail` 의 기본 문구 '요청 상세 조회에 실패하였습니다' 는 이 저장소에 전례가 없는 새 문장이다(과제서 지정값). ③전역 Alert '1회' 는 실제 싱글턴 `showAlert`/`alertText` ref 를 `watch(flush:'sync')` 로 관찰해 셌다 — **같은 문구로 연속 2회 열리면 ref 값이 바뀌지 않아 1회로 집계된다**. COM 중복 케이스는 문구가 다르므로 검출되며(변이 3으로 확인) 실제로 그 형태의 중복만 가능하다.
- 일부러 하지 않은 것: 업무 실패 분기의 `notice/library = null` 초기화 — 변이 4로 **18건 전부 통과**(효과 0)임을 확인했다. `fetchBoardDetail` 은 `onMounted` 에서 한 번만 돌고 초기값이 이미 `null` 이라 무효 변경이다. 자리표시자 computed 제거·빈 상태 UI·`toast` 전환·목록 3형제·`RequestDetail` 의 `inf.library.detail` 엔드포인트는 범위 밖으로 두었다.
- 다음 역할이 조심할 것: `boardDetailFailure.spec.js` 는 DB·네트워크 없이 돌지만 `vi.stubEnv('VITE_BACKEND_API_TARGET')` 를 **동적 import 앞**에 두는 순서가 필수다(`api/index.js` 가 모듈 로드 시 target 을 읽는다). `describe.each` 로 3화면을 돌리므로 한 화면만 고치면 즉시 드러난다. 전역 Alert 은 모듈 싱글턴이라 `beforeEach/afterEach` 에서 초기화한다.
- 릴리즈(A): 진입 조건 미충족으로 무변경(20회째). 재조사하지 않았고 버전 파일·태그·노트·릴리즈 커밋을 일체 만들지 않았다. 사람 입력 4가지 필요 — ①첫 릴리즈 버전 ②태그 형식·주석 태그 여부 ③릴리즈 노트 위치·양식·언어 ④GitHub Release 사용 여부.
- [러너 04:55] brief accepted — 채택 — A(릴리즈)는 지시대로 재조사 없이 무변경, B 를 수용 기준 1~6 그대로 구현·검증했다. 다만 과제서가 "실패 시 기�
- [러너 04:55] verify passed — 검증 2개 통과 (auto)

## 비평 노트
- 확인한 것: 전체 `npx vitest run` 33파일/534테스트 통과. main 의 상세 3형제 소스를 되돌려 신규 스펙만 돌려 **9건 Red**(3화면 × 업무실패/기본문구/catch)를 직접 확인하고 즉시 복원, 작업 트리 clean. `isSuccess` 도입의 오탐 위험은 같은 엔드포인트 선례(`RequestWrite.vue:83-86`)로 닫았다.
- 보안·법무 차단 없음: 인가·식별자·비밀값 변경 없고, 서버 메시지는 `utils/alertText.js` 의 `escapeHtml` 을 거쳐 v-html 로 가므로 XSS 경로가 없다. 개인정보 신규 처리도 없다.
- 못 본 것: 실제 백엔드 응답 형태(저장소 밖)와 브라우저 실행. 판단은 전부 저장소 내부 근거와 테스트 실행 결과다.
- 승인이어도 남는 우려: 목록 3형제(`NoticeList:60`/`LibraryList:58`/`RequestList:132`)는 여전히 실패를 빈 목록으로 감춰 형제 비대칭이 목록 층에 남았다 — 다음 회차 1순위 후보. `RequestDetail:69` 의 `inf.library.detail` 재사용과 라우트 파라미터 변경 시 재조회 부재는 선재 결함으로 이번 변경이 악화시키지 않았다.
- 릴리즈: 커밋 1개·4파일뿐으로 버전 파일·태그·릴리즈 커밋이 만들어지지 않았음을 확인했다(20회째 무변경 유지가 맞다).
- [러너 04:58] review approved — 리뷰 승인 (risk=low)
- [러너 04:58] pr created — https://github.com/hkjang/aiportal-front/pull/33
- [러너 04:59] ci passed — 검사 없음 — 정책으로 허용
- [러너 04:59] merge done — a6ed70e
- [러너 05:00] release failed — 릴리즈 안 함: 다음 버전을 결정할 입력이 저장소에 없어 릴리즈 진입 조건을 충족하지 못했다(21회째 동일 원인). 이번 회차 워크트리 HEAD 8f11ecb(git statu
