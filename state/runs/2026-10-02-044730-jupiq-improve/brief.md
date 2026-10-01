- 과제: `ResourceListPage.test.tsx` 정렬 테스트의 두 번째 클릭이 **로딩 중 테이블**을 눌러 `pointer-events: none`으로 깨지는 경쟁 상태를 없앤다 (가치 4 / 위험 1 / 작업량 S)

- 왜: 지난 회차 검증이 `cd web && npm test --silent` 에서 exit 1 로 끝났고, 실제 실패는 **딱 한 개**다 — `src/components/ResourceListPage.test.tsx > ResourceListPage 작업 컬럼 > 서버 정렬 컬럼을 누르면 sort·order를 붙여 첫 페이지부터 다시 조회한다` (`Error: Unable to perform pointer interaction as the element has 'pointer-events: none': TH(label=작업 유형)`, `ResourceListPage.test.tsx:109`). 테스트가 첫 클릭 뒤 **요청이 나갔다는 것만** 기다리고 **응답이 끝났다는 것은 기다리지 않은 채** 같은 헤더를 다시 누르기 때문에, antd `Spin`이 `ant-spin-blur`(= `pointer-events: none`)를 씌워 둔 사이에 포인터가 떨어지면 깨진다. 타이밍에 따라 통과·실패가 갈리는 플래키라 릴리즈 게이트를 막고 있고, 기다림 한 줄을 제대로 넣으면 같은 사용자 동작(정렬 토글)을 더 정확하게 검증하면서 게이트가 다시 열린다.

- 수용 기준:
  1) 수정 전 상태에서 실패를 재현한다 — `npm --prefix web test -- src/components/ResourceListPage.test.tsx` 를 **반복 실행**(예: `--repeat 5` 또는 수동 5회)해 위 `pointer-events: none` 오류를 최소 1회 관찰한다. 재현되지 않으면 "재현 불가" 를 기록하고, 아래 2)의 논거(두 번째 클릭이 `refreshing=true` 구간에 떨어질 수 있다)를 DOM으로 직접 증명한다 — 첫 `waitFor` 직후 `document.querySelector('.ant-spin-blur')` 가 `null`이 아닌 순간이 존재함을 임시 로그로 확인.
  2) 두 번째 `user.click(sortHeader())` **앞에**, 테이블이 더 이상 로딩 중이 아님을 기다리는 단계가 있다. 구체적으로는 `await waitFor(() => expect(document.querySelector('.ant-spin-blur')).toBeNull())` 수준의 명시적 대기(또는 동등하게 `refreshing` 종료를 증명하는 대기). **`userEvent.setup({ pointerEventsCheck: 0 })` 로 검사를 끄거나, `fireEvent.click` 으로 바꿔치기하거나, 테스트를 `skip` 하는 것은 금지** — 그것은 로딩 중 헤더가 실제로 눌리지 않는다는 사실을 숨기는 것이고 "검증을 느슨하게 해서 통과시키기" 에 해당한다.
  3) 테스트가 증명하는 것은 그대로다: 첫 클릭 → `sort=action` + `order=asc` + `page=1`, 두 번째 클릭 → `order=desc`. 단언 문구를 약화시키지 않는다.
  4) 같은 파일의 나머지 서브테스트 3개와 다른 21개 테스트 파일은 무변경으로 통과한다 — `npm --prefix web test` 전체가 `22 passed (22)` / `144 passed (144)` 이상.
  5) 변경 후 `npm --prefix web test -- src/components/ResourceListPage.test.tsx` 를 연속 5회 돌려 5회 모두 통과한다(플래키가 사라졌다는 증거).

- 건드릴 파일 (프로덕션 0개 — 테스트 1개):
  - `web/src/components/ResourceListPage.test.tsx:93-111` — `'서버 정렬 컬럼을 누르면 sort·order를 붙여 첫 페이지부터 다시 조회한다'` 안, 102-107행 `waitFor` 와 109행 두 번째 `user.click(sortHeader())` **사이에** 로딩 종료 대기를 넣는다. 가능하면 같은 대기를 101행 첫 클릭 앞에도 넣어 대칭을 맞춘다(첫 클릭은 `findAllByLabelText('수정')` 덕에 이미 안전하지만, 같은 경쟁이 재발하지 않게).
  - (선택, 같은 파일 안) 반복되는 대기를 `const settled = () => waitFor(() => expect(document.querySelector('.ant-spin-blur')).toBeNull())` 같은 작은 헬퍼로 묶고, **왜** 기다리는지(= `ResourceListPage.tsx:416` 의 `<Table loading={refreshing}>` 가 antd `Spin`으로 `pointer-events: none` 을 씌운다)를 한국어 주석으로 남긴다 — 이 저장소 관례.
  - 프로덕션 코드는 건드리지 않는다. `web/src/components/ResourceListPage.tsx` 의 `loading={refreshing}`(416행)은 의도된 동작이며 이번 과제 범위 밖이다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `npm --prefix web ci` (이 정찰 세션에서는 샌드박스가 네트워크를 막아 **실행하지 못했다**; 구현자는 먼저 이것부터 돌릴 것)
  - 집중: `npm --prefix web test -- src/components/ResourceListPage.test.tsx` (수정 전 반복 → 수정 후 5회)
  - 전체(= 실패한 게이트와 같은 명령): `cd web && npm test --silent`
  - `npm --prefix web run lint`, `npm --prefix web run build`, `git diff --check`
  - Go 쪽은 이번 변경과 무관하므로 필수는 아니지만, 확인하려면 `go test -count=1 ./...`

- 위험과 피할 것:
  - **`pointerEventsCheck: 0` 로 끄지 말 것.** 이 테스트가 깨진 이유는 antd/jsdom 버그가 아니라 테스트가 비동기 상태를 덜 기다린 것이다. 검사를 끄면 "로딩 중에는 정렬 헤더가 눌리지 않는다" 는 실제 사실이 영원히 가려진다.
  - `INTERACTION_TIMEOUT = 30_000`(64행)은 이미 넉넉하다. **타임아웃을 더 늘리는 것으로 때우지 말 것** — 이 실패는 느려서가 아니라 잘못된 시점에 눌러서 난다(실패한 서브테스트는 1596ms 만에 끝났다).
  - `vitest.config.ts` 의 `environment: 'jsdom'`·`setupFiles`·`css: true` 를 건드리지 말 것. `css: true` 를 끄면 antd 스타일이 안 깔려 이 테스트가 "우연히" 통과하지만, 다른 스타일 기반 단언(`src/styles.contract.test.ts`)이 무너질 수 있다.
  - 같은 로그에 `Error: Not implemented: window.getComputedStyle(elt, pseudoElt)` 가 5회 찍히지만(rc-util `getScrollBarSize` → jsdom 미구현) **이것은 경고일 뿐 실패 원인이 아니다**(2026-09-29 회차에서도 같은 경고가 나면서 144개 전부 통과했다). 같이 고치려 들지 말 것 — 범위가 번진다.
  - 보호 경로(`internal/auth`, `migrations/`, `.github/workflows/`)는 건드리지 않는다. 이번 과제는 워크플로 파일을 고칠 필요가 전혀 없다.
  - 프로필의 함정 그대로: `client.ts` 는 모듈 로드 시 `VITE_API_BASE_URL` 을 읽는다. 이 과제는 `api/` 테스트를 건드리지 않으므로 해당 없음.

- 미확인 (추측으로 적은 것을 명시):
  - 이 정찰 세션에서는 `npm ci` 가 샌드박스에 막혀 **테스트를 한 번도 실행하지 못했다.** 진단의 근거는 전적으로 이전 회차의 실제 검증 로그 `state/runs/2026-10-02-013739-jupiq-improve/verify.txt:105-138` 이다(스택트레이스·파일·행 번호·오류 문구 모두 그 로그에서 그대로 옮겼다).
  - `pointer-events: none` 의 출처가 antd `Spin` 의 `ant-spin-blur` 라는 것은 `ResourceListPage.tsx:416` 의 `loading={refreshing}` 과 antd 관례로부터의 **추론**이다. 구현자는 실패 시점의 `TH` 조상 클래스를 실제로 찍어 확인한 뒤 대기 셀렉터를 확정할 것(`ant-spin-blur` 가 아니면 실제로 관찰된 클래스를 쓸 것).
  - "릴리즈 워크플로가 같은 이유로 두 번 실패했다" 는 적재 문구는 **확인되지 않았다**: `2026-09-29-191711` 과 `2026-09-29-172225` 의 `verify.txt` 는 각각 `22 passed (22)` / `21 passed (21)` 로 web 테스트가 통과했다. web 테스트 실패는 `2026-10-02-013739` **한 번**이다. 따라서 이 과제는 "반복 실패한 고질" 이 아니라 "새로 드러난 플래키" 로 다룰 것.

- 차선 후보: **`internal/store` 순수 헬퍼 5개(`firstLabel`·`latestMetricSample`·`countBool`·`countRuntime`·`filterBool`) 표 기반 테스트** (가치 2 / 위험 1 / 작업량 S). 다섯 회차 연속 차선에 머물렀다. 1순위가 성립하지 않으면(예: 반복 실행해도 실패가 전혀 재현되지 않고, 로딩 중 `ant-spin-blur` 구간도 관찰되지 않아 고칠 것이 없을 때) 이것을 하고, 그 경우 1순위 아이디어는 `rejected` 로 내릴 것.
