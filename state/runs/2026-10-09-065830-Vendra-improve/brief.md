- 과제: 자동 저장 응답 대기 중 신규 업무 중복 제출 방지 (가치 4 / 위험 2 / 작업량 S)
- 왜: `NewObject.save`는 초안 PUT을 기다린 뒤에야 제출 버튼을 잠그며, 이미 제출 중인지 검사하지 않아 실제 두 번 클릭에서 `POST /api/v1/contracts`가 2회 발생한다. 첫 제출부터 완료까지 중복 진입을 막으면 느린 자동 저장 중에도 사용자의 한 등록 의도가 여러 생성 요청으로 바뀌지 않는다.
- 수용 기준:
  1) 진행 중인 초안 PUT이 있을 때 첫 제출 즉시 버튼은 disabled이고 `저장 중…`을 표시하며, PUT 응답 전 POST는 0회다.
  2) PUT 대기 중 버튼 재클릭 및 실제 form에 연속 submit 이벤트를 보내도 첫 제출만 처리한다. PUT 완료 후 POST는 정확히 1회이고 첫 제출 당시 제목·금액·설명·체크박스 스냅샷을 사용한다.
  3) POST와 후속 초안 DELETE가 각각 대기하는 동안 반복 submit도 추가 POST를 만들지 않는다. 정상 완료 시 DELETE 1회, 모달 닫힘, 목록 GET 총 2회(최초 1회 + 완료 후 1회)로 끝난다.
  4) POST가 실패하면 기존 서버 오류 문구가 보이고 버튼이 다시 활성화되며 수정 후 재시도가 실제 새 POST 1회를 보낸다. 초안 PUT 실패 뒤 직접 저장을 계속하는 정책, DELETE 실패를 저장 실패로 뒤집지 않는 정책은 유지한다.
  5) 실제 `Objects` 렌더·React DOM 이벤트·진짜 HTMLFormElement/FormData·원본 api/post/put/del을 통과하고 fetch만 대체하는 테스트로 1~4를 증명한다. 기존 스냅샷·즉시 제출·PUT 500·POST 재시도 4개 테스트를 약화 없이 통과시키고 전체 웹 검증을 완료한다.
- 건드릴 파일:
  - `web/src/pages/Objects.tsx:NewObject.save`(:780-833) — preventDefault 직후 동기적 중복 진입 검사, 첫 await 전 제출 잠금과 busy 설정, 실패 시 재시도 허용. 이미 있는 submitted ref를 재사용할 수 있다. 제출 버튼(:1020 부근)의 기존 disabled={busy}를 활용한다. 프로덕션 변경은 이 파일 하나로 제한한다.
  - `web/src/pages/objects-draft-submit.test.tsx:network/openContract/submit/finishDeletion` — 기존 fetch 경계 도우미로 중복 클릭·submit 및 POST/DELETE 대기 중 중복 진입 회귀 추가. 기존 단정은 유지한다. 새 테스트 전용 파일을 선호하면 1개 추가까지 허용하지만 도우미 추출 리팩터는 하지 않는다.
- 검증 명령(저장소 루트 기준; 의존성 설치 뒤 실행):
  ```bash
  npm ci --ignore-scripts --prefix web
  npm test --prefix web -- src/pages/objects-draft-submit.test.tsx src/pages/objects-error.test.tsx src/pages/objects-order.test.tsx
  npm test --prefix web
  cd web
  npx tsc -b --noEmit
  npx eslint src --max-warnings 0
  npm run build
  ```
  새 파일을 추가하면 집중 실행 대상에 그 경로도 넣는다. test 래퍼·인수 전달은 아래 정찰 실행으로 확인했고 타입·린트·빌드는 package.json/CI에서 확인한 명령이다. 이번 정찰에서 전체 웹·tsc·lint·build·Go/DB 통합은 실행하지 않았다.
- 위험과 피할 것: `FormData`를 첫 await 뒤로 옮기거나 DOM 참조만 보관하지 말 것(fd56d79 회귀). busy state 검사만으로 동기적 submit 중복을 막았다고 판단하지 말고 ref로 진입을 차단할 것. submitted는 saveDraft와 늦은 복원도 막으므로 성공 경로 finally에서 무조건 false로 되돌리지 말 것; POST 실패 때만 기존 재시도 의미를 유지한다. 입력 전체 비활성화·취소/모달 닫기 정책·중첩 자동 저장 PUT 직렬화·서버 idempotency·공용 Modal·금액/통화·auth/session/migrations/.github/workflows는 제외한다. 실제 DB 중복 행 생성은 미확인이고 이번에 증명한 것은 생성 POST 중복 전송이다.
- 차선 후보: 저장된 보기 조회 실패 표시와 전용 재시도 (3/2/M) — `ObjectList.loadViews`(:267)의 `.catch(() => setViews([]))`를 좁혀 실패 이유와 전용 재시도를 제공한다. 주 과제는 이미 재현됐으므로 구현 시 해당 결함이 선행 수정되어 더 이상 재현되지 않는 경우에만 선택한다. 공급업체 조회 실패와 함께 확대하지 않는다.

실행 근거와 재현:
- 기준 HEAD `e030b63`, 작업 트리 변경 없음. 최근 30개 로그에서 `fd56d79`의 폼 스냅샷 수정 병합을 확인했다. 현재 :783 FormData → :785 submitted=true → :786 await draftRequest → :787 busy=true 순서이며 submitted를 읽는 진입 가드는 없다.
- 지정된 기록 경로의 `assets/probe/web`에 웹 사본과 진단 테스트만 만들었다. 프로덕션 소스는 수정하지 않았다. Node v22.23.1 / npm 10.9.8, 사본에서 `npm ci --ignore-scripts` 성공(255 packages).
- `assets/probe/web`에서 실행한 실제 명령: `npm test -- src/pages/objects-duplicate-probe.test.tsx src/pages/objects-draft-submit.test.tsx src/pages/objects-error.test.tsx src/pages/objects-order.test.tsx`.
- 결과: 1 failed / 15 passed, 4 files, 30.82초. 실패는 새 진단의 `expected ... to have a length of 1 but got 2`; 콘솔에 `submit button disabled while PUT pending: false`, `POST requests after PUT completes: 2`. 미처리 rejection 보고 없음. 기존 3파일 15개는 모두 통과해 이전 폼 이벤트 결함과 구분된다.
- 로그: `assets/duplicate-probe.log`. 진단 소스: `assets/probe/web/src/pages/objects-duplicate-probe.test.tsx`. 새 테스트는 기존 network의 미완료 PUT을 만든 후 정상 버튼을 두 번 클릭하고 PUT을 완료시킨다. React 이벤트나 FormData를 직접 만든 대역은 사용하지 않는다.
- 서버 `internal/httpapi/objects.go:createObject`(:219-293)도 읽었다. 매 호출의 INSERT 경로는 확인했으나 DB 실측은 하지 않았고 서버 변경은 요구하지 않는다.

구현 순서와 검토 지점(모두 구현자의 자동 검증, 사람 승인 대기 없음):
1. [미착수] 기존 집중 15개를 기준선으로 확인하고, 새 중복 클릭 재현 및 실제 form submit 경로를 테스트로 고정한다. 위 집중 명령에서 새 테스트만 의도한 횟수/버튼 상태로 실패하는지 확인한다. 컴파일 실패·0 tests·mock 설정 오류는 재현 증거로 인정하지 않는다.
2. [미착수] NewObject.save에서 preventDefault → 이미 submitted면 return → 첫 FormData 스냅샷 → 제출 잠금/버튼 busy → PUT 대기 → POST → DELETE → onSaved 순서를 보장한다. 비동기 대기를 try 내부에 두어 해제·오류 처리가 분명하게 하되 PUT 오류 허용은 기존 saveDraft에서 처리되는 의미를 보존한다. 집중 테스트를 통과시키고 POST 오류 후 수정·재시도 및 DELETE 대기 중 잠금이 유지되는지 확인한다.
3. [미착수] 전체 npm test, tsc, CI 범위 eslint src, build를 위 명령으로 실행한다. 실패하면 이번 변경 원인인지 판별하고 과제 범위를 확장하지 않는다. 가능하면 진입 가드만 잠깐 되돌려 실제 submit 중복 테스트가 실패하고, busy 이동만 되돌리면 버튼 상태 테스트가 실패함을 확인한 뒤 복원한다. 검증 결과와 남은 한계를 journal에 기록한다.

대안 비교 및 추정 근거:
- 선택: 기존 submitted ref 검사 + busy 설정 시점 이동. UI 클릭과 submit 함수 양쪽을 막고 기존 4개 행동 테스트를 그대로 재사용할 수 있다.
- busy 이동만: 변경량은 가장 작지만 렌더/버튼 비활성화에 의존하고 submit 핸들러의 중복 진입은 남는다. 단독 대안으로 채택하지 않는다.
- 서버 idempotency 도입: 재전송 전반을 다룰 수 있지만 요청 식별자·저장 계약·DB 테스트가 필요해 이번 45분 범위를 넘는다. 더 큰 범위의 별도 과제로만 검토한다.
- 현상 유지: 생성 POST 2회를 실제 확인했으므로 문서 경고나 테스트만 추가하는 것보다 작은 코드 수정이 가치가 높다. 대안 중 핵심 가정은 existing submitted의 실패 해제가 재시도를 계속 허용한다는 것; 기존 POST 오류 테스트로 확인한다.
- 작업분해 추정: 재현/추가 테스트 10~14분, 1파일 수정 4~6분, 전체 검증/기록 8~12분 = 기본 22~32분. 알려진 변동(느린 설치·DOM 비동기 테스트 조정) 대응 5~8분을 별도로 두어 합계 27~40분. 신뢰도 중간의 작업 추정이며 통계적 확률/보장치가 아니다. 관리 예비분은 0분으로 책정하고 예기치 않은 범위는 보류 목록에 넘긴다.
- 유사 비교: 직전 fd56d79는 같은 프로덕션 1파일·동일 fetch 하네스의 4개 회귀 테스트였고 이번도 같은 크기의 수정이다. 과거 실제 소요시간 자료가 없어 유사법으로 별도 분 단위 견적을 꾸미지는 않았으며 환경/DB 변경이 없다는 조건에서 S로 판단했다.
- 추정 방법의 참고: 범위·작업분해·가정·위험을 기록하고 실행 뒤 갱신하는 원칙은 [GAO Cost Estimating and Assessment Guide](https://www.gao.gov/products/gao-20-195g)를 확인했다. 위 시간 수치는 저장소 작업에 대한 정찰자의 판단이다.
- 요청된 스킬은 전용 Skill 도구가 노출되지 않아 `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/{pmo,technology}/skills/` 아래 estimating-and-contingency, implementation-planning, solution-exploration의 SKILL.md를 직접 읽어 적용했다.
