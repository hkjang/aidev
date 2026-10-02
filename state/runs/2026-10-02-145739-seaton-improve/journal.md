## 구현 노트
- 바꾼 것: `web/src/pages/EmployeesPage.tsx` 의 `load` 하나. `HistoryPage.load` 와 같은 꼴로 요청 순번(`requestRef`)과 조회 시작 시 `setError("")` 를 넣었다. 새 규칙을 만들지 않고 저장소에 이미 있는 꼴을 옮긴 것이다. 프로덕션 1파일, 새 E2E 1파일(`web/e2e/employee-reload.spec.ts`), 문서 변경 없음.
- 확신 없는 곳: (1) 두 번째 E2E 는 낡은 응답이 반영될 틈을 주려고 `page.waitForTimeout(500)` 을 쓴다 — 느린 러너에서 React 반영이 500ms 를 넘으면 버그가 있어도 초록이 될 수 있다(거짓 통과 방향이고 거짓 실패는 아니다). (2) `load` 가 조회마다 `error` 를 지우므로, 가져오기 실패 배너도 그다음 조회에 사라진다. 그게 맞는 동작이라고 판단했지만 사용자 확인은 받지 못했다 — `upload`/`assignFromFile` 의 catch 는 `load` 를 부르지 않으니 가져오기 직후에는 그대로 남는다. (3) 전체 E2E 는 수정본에서 78건 중 76건 통과, 실패 2건은 mcp-oauth·tracking 뿐이다. 둘은 `E2E_COLLECTOR_HOST` 가 기본 127.0.0.1 이라 bridge 네트워크의 앱 컨테이너가 호스트의 가짜 IdP·수집기에 닿지 못해 실패한다(spec 주석이 그 전제를 적어 둠). 변경 전 이미지로 전체 스위트를 다시 돌려 같은 2건임을 대조하지는 않았다 — 내 변경은 직원 화면 컴포넌트 한 곳이라 두 spec 의 경로와 겹치지 않는다.
- 일부러 안 한 것: 같은 순번 패턴이 이제 두 화면에 생겼지만 lib 헬퍼로 뽑지 않았다 — 순수 헬퍼 단위테스트는 배선 결함을 증명하지 못하고 파일만 늘기 때문. `rows.Err()` 삼킴(Go)은 붉은-먼저를 만들 하네스가 없어 이번에도 제외하고, 대신 '하네스 조사' 과제를 ideas.json 에 새로 넣었다.
- 다음 역할이 조심할 것: 새 spec 은 실서버 + PostgreSQL 이 있어야 돈다. 이 셸에서는 `PLAYWRIGHT_BROWSERS_PATH=/home/hkjang/.cache/ms-playwright` 가 필요하고, 포트 8080 은 이전 회차의 `seaton-app` 컨테이너가 쥐고 있어 이번엔 18781 을 썼다. `seaton:e2e-before` 라는 태그는 9월 29일자 낡은 이미지가 남아 있어 역검증 전에 반드시 다시 빌드해야 한다 — 그대로 쓰면 엉뚱한 코드로 붉은/초록을 판정한다.
- [러너 15:16] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: diff 전체, `EmployeesPage.load` 와 `HistoryPage.load` 대조(같은 꼴, 조기 return 도 finally 를 지나 loading 누수 없음), 새 spec 두 개가 실제로 바뀐 경로를 지나는지(시드 영업팀 2 vs 전체 10, `api.ts` 의 error.message 계약), 원장의 '실패 재현' 두 출력이 이번 증상과 일치하는지. 직접 재실행: `npm run lint`(e2e 포함, exit 0)·`npm test`(159건)·`playwright --list`(2건 등록).
- 못 본 것: 실서버 E2E 를 이 세션에서 돌리지 않았고(원장 역검증 기록을 근거로 수용), 변경 전 이미지로 전체 스위트를 다시 돌려 실패 2건이 동일한지 대조하지 않았다.
- 승인이어도 남는 우려: `employee-reload.spec.ts:92` 의 `waitForTimeout(500)` 은 느린 러너에서 거짓 통과 방향으로 샌다. 그리고 `load` 가 매 조회마다 `error` 를 지우므로 가져오기 실패 배너가 다음 필터 변경에 사라진다 — 릴리즈 노트에 적을 사용자 체감 변화는 이것뿐이고, 옳은 동작인지는 ideas.json 과제로 남아 있다.
- 다음 회차: mcp-oauth·tracking 두 실패를 '정상 기준'으로 고정하지 말 것. 환경 전제(`E2E_COLLECTOR_HOST`) 때문이라는 설명은 이번에도 재확인되지 않았다.
- 차단 소견: security·legal 모두 없음. 새 엔드포인트·식별자·비밀값·암호 비교·신뢰 못 할 입력의 싱크가 없고, 개인정보를 새로 수집·저장·전송하지 않는다.
- [러너 15:19] review approved — 리뷰 승인 (risk=low)
- [러너 15:19] pr created — https://github.com/hkjang/seaton/pull/40
- [러너 15:24] ci passed — 검사 2개 모두 success
- [러너 15:24] merge done — cd7d20e
