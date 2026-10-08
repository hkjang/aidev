- 과제: 자동 저장 중 신규 업무를 제출하면 폼 이벤트가 비워져 저장되지 않는 문제 수정 (가치 4 / 위험 1 / 작업량 S)
- 왜: `NewObject.save`는 진행 중인 초안 PUT을 기다린 뒤 `new FormData(e.currentTarget)`를 실행하는데, React는 이벤트 콜백 반환 직후 currentTarget을 null로 만든다. 자동 저장과 사용자의 제출이 겹칠 때 업무 POST가 시작되지 않는 결함을 고쳐 입력한 내용을 정상 저장할 수 있게 한다.
- 수용 기준:
  1) 실제 `<Objects type="contract" />`에서 「새 계약」을 열고 제목을 입력해 700ms 자동 저장 PUT이 진행 중인 상태로 만든 뒤 「초안 저장」을 누른다. PUT 완료 전 업무 POST는 없어야 하고, PUT 완료 뒤 정확히 한 번 `/api/v1/contracts` POST가 제출 시점의 제목·입력값으로 전송되어야 한다. POST 성공 → 초안 DELETE → 모달 닫기 → 목록 재조회가 이어지며 unhandled rejection이 없어야 한다.
  2) 초안 PUT이 없는 즉시 제출도 정상 동작한다. 초안 PUT 완료를 기다리는 순서는 유지한다(이를 빼면 삭제된 초안이 늦은 PUT으로 되살아날 수 있음).
  3) PUT이 500으로 실패해도 기존 saveDraft의 catch 정책대로 기다림을 마치고 명시적 업무 저장은 계속 가능해야 한다. 업무 POST가 실패하면 서버 메시지가 모달에 보이고 다시 저장할 수 있어야 한다.
  4) 신규 테스트는 실제 React DOM 이벤트·진짜 HTMLFormElement/FormData와 원본 api/post/put/del 경로를 지난다. 구현 전 위 1번이 실제 저장 호출 부재와 FormData 오류로 실패하고, 구현 후 통과해야 한다. 기존 목록 정렬·목록 오류 테스트는 그대로 통과한다.
- 건드릴 파일:
  - `web/src/pages/Objects.tsx:NewObject.save`(:780-834) — 첫 await 전에 제출 시점의 FormData를 확보하고 이후에는 그 스냅샷을 사용한다. 지금 :784가 await, :787이 FormData, :788이 try다(좌표보다 함수와 문장으로 확인). DOM 참조만 미리 잡아 나중에 다시 읽는 방식은 기다리는 동안 입력이 바뀔 수 있어 제출 시점 값 보존을 못 한다.
  - `web/src/pages/objects-draft-submit.test.tsx`(신규) — deferred fetch로 진행 중 PUT을 통제하는 실제 컴포넌트 회귀 테스트. 프로덕션 1파일 + 테스트 1파일.
- 검증 명령:
  - 저장소 루트 기준 `cd web && npm ci --ignore-scripts`(현재 워크트리 node_modules 없음; 구현자 설치 단계).
  - `cd web && npm test -- src/pages/objects-draft-submit.test.tsx src/pages/objects-error.test.tsx src/pages/objects-order.test.tsx`
  - `cd web && npm test`
  - `cd web && npx tsc -b --noEmit && npx eslint src --max-warnings 0 && npm run build`
  - 정찰 진단: 아래 실행 기록 참조. Go/DB 통합은 정찰에서 실행하지 않았고 이 웹 전용 수정의 동작 증거로 대신 삼지 않는다.
- 위험과 피할 것: Go·auth·migrations·workflows·의존성·통화·공용 Modal/Loading을 건드리지 않는다. 자동 저장 구조, debounce 700ms, 초안 DELETE 실패 정책, 저장된 보기/정렬 정책은 그대로 둔다. 중복 제출 방지 전반은 별도 후보이며 이번 과제의 필수 변경에 섞지 않는다. FormData나 React 이벤트를 가짜 객체로 만들어 원인을 숨기지 말 것. api export만 mock하면 같은 모듈의 post/put/del이 원본 api를 닫아 잡는 함정이 있으므로 `vi.stubGlobal("fetch", ...)`를 사용한다. 보호할 원문이나 입력값을 감사로그에 추가하지 않는다.
- 차선 후보: 업무 목록의 저장된 보기 조회 실패 표시와 전용 재시도 (가치 3 / 위험 2 / 작업량 M) — 본 과제의 실제 이벤트 재현이 성립하지 않을 때만 전환. `ObjectList.loadViews`(:267-278)의 `.catch(() => setViews([]))`를 별도 조회 실패 상태로 처리하고 같은 context 재조회 버튼을 제공. 서버 오류 또는 한국어 기본문구·성공 시 오류 제거·업무 행 유지·정상 빈 보기에서는 오류 없음의 실행 테스트를 추가. 저장/삭제용 viewError와 구별하고 다른 화면으로 확장하지 않는다.

근거와 실행 순서 (구현 담당: 다음 구현 에이전트, 모든 단계 아직 미완료):
1. 환경 준비 및 재현(8~12분): 위 npm ci 후 실제 Objects를 MemoryRouter 안에 렌더. GET suppliers→items:[], saved-views→items:[],canShare:false, GET drafts/new-object:contract→draft:null, GET contracts→items:[]를 fetch에서 경로·method로 분기한다. 「새 계약」 클릭 → placeholder 「계약 제목」 입력 → PUT 시작을 기다림 → 「초안 저장」 클릭 → deferred PUT 해제. `npm test -- src/pages/objects-draft-submit.test.tsx`로 원인과 실패를 확인. 검토 지점: 사람 승인 없음; 빌드/환경 오류라면 결함 재현으로 세지 말고 하네스를 먼저 수정.
2. 국소 수정(3~5분): Objects.tsx save의 FormData 생성만 첫 await 앞에 옮기는 최소안을 우선 적용. `npm test -- src/pages/objects-draft-submit.test.tsx`로 통과 확인. 검토 지점: 사람 승인 없음; await 순서나 입력 변환식까지 바꾸게 되면 범위를 재평가.
3. 회귀·검증(10~15분): 정상 즉시 제출, PUT 실패 후 명시적 저장, POST 오류 후 재시도 케이스를 완성하고 위 전체 웹 테스트·tsc·eslint·build 실행. 검토 지점: 사람 승인 없음; 테스트 완료 전 구현 완료로 표시하지 말 것.

탐색한 해법: (A) 제출 직후 FormData 스냅샷 확보 — 새 의존성 없이 한 함수에서 해결하므로 선택. (B) formRef.current를 await 뒤에 읽기 — null 이벤트는 피하지만 대기 중 변경/언마운트 의미가 달라져 제외. (C) 자동 저장 중 제출을 막기 — UI 지연과 상태 추가가 필요하므로 제외. (D) 현상 유지 — 정상 구매 업무의 명시적 저장이 중단되어 수용 불가. 핵심 가정은 실제 이벤트에서 PUT 대기 경로가 발생한다는 것이며, 단계 1의 실행 테스트가 이를 판정한다.

추정 근거: 위 분해의 합은 기본 21~32분. npm 설치 및 비동기 테스트 타이밍이라는 알려진 변동에 예비 5~8분을 별도로 두어 총 26~40분, 신뢰도 중간(통계적 확률을 뜻하지 않음). 과거 같은 컴포넌트 1파일+신규 테스트 회차들과 규모를 비교했으나 과거 실소요 시간은 없어 시간으로 환산하지 않았다. 관리 예비시간은 0분이며 새 범위는 다음 회차로 보낸다. 금액 추정 없음.

스킬 적용: Skill 도구는 이 세션에 없었으므로 아래 로컬 SKILL.md를 직접 읽었다. solution-exploration의 대안·선택 근거, implementation-planning의 단계별 증명·검토 지점, estimating-and-contingency의 범위·가정·분해·별도 예비를 위에 반영했다.
- /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/solution-exploration/SKILL.md
- /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/implementation-planning/SKILL.md
- /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md

정찰 검증 기록: main@d8591b9(v0.7.72), 작업 트리 무변경. web/node_modules가 없어서 허용된 run/assets/scout-web 아래에 필요한 원본 6파일을 그대로 복사하고, lockfile이 같은 기존 설치의 node_modules를 참조하는 별도 진단 하네스를 만들었다. 최초 symlink 소스 하네스는 setup import 단계에서 실패(0 tests)하여 결함 근거에서 제외했다. 복사본 하네스 실행 결과는 아래와 같다. 정찰 전체 웹 스위트 및 수정 후 통과는 미확인이다.

- 재현 명령(실행 확인): `cd /mnt/c/Users/USER/projects/aidev/state/runs/2026-10-08-190833-Vendra-improve/assets/scout-web && node node_modules/vitest/vitest.mjs run --config vitest.config.mts`
- 결과: exit 1, `Test Files 1 failed (1) / Tests 1 failed (1) / Errors 1 error`, 51.20초. `repro.test.tsx:22`의 계약 POST 관찰이 false이고, `TypeError: Failed to construct 'FormData': parameter 1 is not of type 'HTMLFormElement'`가 `save src/pages/Objects.tsx:787:15`에서 발생. DOM 출력에도 disabled 버튼 「저장 중…」이 남음.
- 원본 `Objects.tsx` 등 6개 파일과 진단 복사본은 바이트 동일함을 확인했다. 외부 설치를 참조했지만 web/package-lock.json은 현재 저장소와 동일(cmp exit 0). 실험은 fetch만 대체했으며 api/post/put/del, React 이벤트, FormData는 원본이다. 수정한 복사본으로 통과시킨 실험은 하지 않았고 구현자에게 맡긴다.
- 로그: `assets/scout-web/reproduction.log`, 재현 코드: `assets/scout-web/repro.test.tsx`. 진단 전용 파일이므로 그대로 저장소에 복사하는 대신 repo의 afterEach cleanup과 `vi.unstubAllGlobals()`를 실패 여부와 무관하게 실행하는 정식 테스트로 다듬는다. 임의의 sleep보다 deferred PUT과 waitFor로 순서를 관찰한다.
