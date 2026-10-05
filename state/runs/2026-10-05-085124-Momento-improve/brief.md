- 과제: Event Schema 저장 실패를 한국어로 안내한다 — 손으로 적는 JSON 이 깨지면 지금은 V8 의 영문 `SyntaxError` 가 Alert 에 그대로 뜬다 (가치 3 / 위험 1 / 작업량 S)

- 왜: `web/src/pages/AdminPage.tsx:3703-3715` 의 `save` 뮤테이션이 `schema: JSON.parse(form.schemaText)` 를 `mutationFn` 안에서 평가하므로, 5줄짜리 자유 입력 칸(`label="JSON Schema"` — 3768행, `multiline minRows={5}`)에 적은 JSON 이 한 글자라도 깨지면 요청이 서버에 가기도 전에 자바스크립트 엔진의 영문 문장(`Unexpected token '}' ... is not valid JSON` 류)이 만들어지고, 3775행의 `<Alert severity="error">{save.error.message}</Alert>` 가 그것을 그대로 띄운다. 서버까지 간 실패도 마찬가지로 영문·pgx 원문이다 — `upsertEventDefinition`(internal/httpapi/admin.go:1395-1446)이 `INVALID_MODE`("validation mode must be allow, warn, or reject", 1410행)와 `DEFINITION_SAVE_FAILED`(1416·1423·1428·1433·1437·1441행, 모두 `err.Error()`)를 돌려주므로 v0.34.54~56 이 사용자·사이트·망 구분에서 고친 것과 똑같은 누출이 여기 하나 남아 있다.

- 수용 기준:
  1) 「JSON Schema」 칸에 깨진 JSON(예: `{"properties": }`)을 적고 「저장」을 누르면 Alert 본문이 한국어로, **어느 칸을 고쳐야 하는지** 말한다. 엔진 원문의 표지(`Unexpected token`, `is not valid JSON`, `JSON.parse`, `position`)가 본문에 섞이지 않는다. 원문은 지우지 말고 기존 선례대로 `detail` caption 으로만 남긴다.
  2) 서버가 돌려주는 코드 전부가 한국어 안내를 갖는다 — `INVALID_PAYLOAD`·`INVALID_MODE`·`DEFINITION_SAVE_FAILED`·`REQUEST_FAILED`, 그리고 모르는 코드는 **서버 메시지로 되돌림**(`describeUserError`/`describeSiteError`/`describeNetworkError` 의 `default` 분기와 같은 계약). `DEFINITION_SAVE_FAILED` 의 문구는 **중립으로 둘 것** — 원인 미확정(아래 "위험과 피할 것").
  3) 테스트가 증명할 것: (a) 깨진 JSON 에서 나오는 실제 `SyntaxError`(손으로 만든 가짜 객체가 아니라 `try { JSON.parse('{"properties": }') } catch (e) { e }` 로 얻은 진짜 에러)의 안내에 엔진 원문 표지가 없다, (b) 아는 코드 네 개의 안내가 서버 영문이 아니다, (c) 모르는 코드는 서버 메시지 그대로, (d) 500 의 pgx 원문이 본문이 아니라 `detail` 로만 남는다. 기존 `adminErrors.test.mjs` 의 사용자·사이트·망 구분 단언은 **글자 그대로 그대로 통과**해야 한다(문장 상수를 건드리지 않았다는 증거).

- 건드릴 파일 (프로덕션 2 + 테스트 1):
  - `web/src/pages/adminErrors.ts` — `describeEventDefinitionError(error): UserErrorNotice` 를 **새로** 추가. 기존 `refusal(error)`(65-75행)로 code·message 를 읽고 `switch`, 상수 `UNREADABLE_PAYLOAD`·`LOST_REQUEST`·`PASS_TO_ADMIN`(78-83행)을 재사용. **`describeUserError`/`describeSiteError`/`describeNetworkError` 와 합치지 말 것** — 세 함수를 따로 둔 이유(핸들러마다 코드 집합·고칠 것이 다름)가 그대로 적용되고, 합치려 한 시도는 운영자 지침("계약이 다른 파서를 통합하려 하지 말 것")에 걸린다. 공유는 `refusal` 과 문장 상수까지만.
    - JSON 구문 오류 분기는 `switch` **앞**에 둔다. 이 에러에는 `code` 가 없어(`refusal` 이 `""` 를 돌려줌) 그냥 두면 `default` 로 떨어져 영문이 그대로 나온다. 판정은 `error instanceof SyntaxError` 로 하고, 한 겹 더 쓰고 싶으면 메시지 패턴(`/JSON/`)을 **보조로만** 더할 것(instanceof 는 같은 realm 에서 던진 에러라 성립한다).
  - `web/src/pages/AdminPage.tsx` — (1) 78-80행 import 목록에 `describeEventDefinitionError` 추가, (2) 3350-3352행의 `NetworkErrorAlert` 바로 옆에 `EventDefinitionErrorAlert({ error })` 를 같은 모양으로 추가(`<AdminNoticeAlert notice={describeEventDefinitionError(error)} />`), (3) 3775행의 `{save.error && <Alert severity="error">{save.error.message}</Alert>}` 를 `{save.error && <EventDefinitionErrorAlert error={save.error} />}` 로 교체. **다른 Alert 은 손대지 말 것** — 2411(설정 저장)·2508(rekey)·2955(보존정책)·3110(dimensions)·2759·1370 은 각각 다른 핸들러이고 다음 회차의 조각이다.
  - `web/test/adminErrors.test.mjs` — 새 `describeEventDefinitionError` 를 import 목록에 더하고(`../src/pages/adminErrors.ts`, **`.ts` 확장자 필수**), 기존 `apiError(status, code, message)` 헬퍼(22-27행)를 그대로 써서 위 (a)~(d) 를 단언. 기존 단언은 한 줄도 고치지 말 것.
  - Go 파일은 **건드리지 않는다**. 서버 검사가 권한·검증의 정본이고 화면은 거울이다.

- 검증 명령:
  - `cd web && npm ci && npm run lint && npm test && npm run build` (worktree 에 `web/node_modules` 가 없을 수 있어 `npm ci` 선행 — 수 분. 현재 테스트 수는 미확인이나 v0.34.56 시점 191건 이상이고, 새 단언만큼 늘어야 한다)
  - `npm test` 는 `web/package.json` 이 `"${npm_node_execpath:-node}" --test test/*.test.mjs` 로 고정돼 있다(v0.34.53). `node --test test/adminErrors.test.mjs` 로 한 파일만 돌릴 때는 Node 22.18+ / 24+ 여야 `.ts` 가 스트리핑된다.
  - **프로덕션 배선 확인(이 과제의 핵심)**: 앞 네 회차와 같은 방식 — `npm run build` 의 dist 를 `/api/v1/me`·`/api/v1/sites`·`/api/v1/event-definitions` 를 흉내 낸 임시 서버에 올리고 headless Chrome(google-chrome 설치돼 있음, `puppeteer-core` 는 없어 /tmp 에 따로 설치)으로 Event Schema 화면을 열어 「JSON Schema」 칸에 `{"properties": }` 를 적고 「저장」을 눌러 Alert DOM 을 읽을 것. 임시 하네스와 `web/dist` 는 커밋 전에 지우고 `git status` 로 3파일만 담긴 것을 확인.
  - `npx prettier --check` 는 게이트로 쓰지 말 것(저장소에 prettier 의존성·CI 단계가 없다).

- 위험과 피할 것:
  - **먼저 재현부터 하라 — 여기에 하나 미확인이 있다.** `JSON.parse` 는 `mutationFn` **안에서 동기적으로** 던져진다. @tanstack/react-query 5.85.5 의 retryer 가 그 동기 throw 를 잡아 `save.error` 에 넣는다고 읽었지만 라이브러리 소스를 이 세션에서 열어 확인하지 못했다(`web/node_modules` 없음). 브라우저에서 깨진 JSON 으로 「저장」을 눌러 **Alert 에 영문 SyntaxError 가 실제로 뜨는 것**을 먼저 보고 시작할 것. 뜨지 않고 아무 일도 일어나지 않으면(= 에러가 삼켜진다면) 이 과제는 성립하지 않으니 아래 차선 후보로 갈 것. grep 으로 문자열이 있다는 것은 증거가 아니다.
  - `DEFINITION_SAVE_FAILED` **원인을 특정하지 말 것**. 이 500 은 서로 다른 여섯 자리가 함께 쓴다(Begin 실패, INSERT, version 조회, 두 Exec, Commit). 특히 1421행 INSERT 는 `SELECT id,... FROM sites WHERE site_key=$1` 이라 site_key 가 맞지 않으면 0행 삽입 → `pgx.ErrNoRows`(`no rows in result set`)로 같은 500 이 된다. 이름 중복은 `ON CONFLICT(site_id,name) DO UPDATE` 로 흡수되므로 **"이미 등록된 이벤트" 라고 쓰면 거짓**이다. v0.34.55(`SITE_CREATE_FAILED`)·v0.34.56(`NETWORK_CREATE_FAILED`) 선례대로 중립 문구 + `detail` 로 둘 것. DB 재현 없이 500 문구를 특정하면 과거 두 회차가 피한 함정에 들어간다(`MOMENTO_TEST_POSTGRES_DSN` 이 없으면 통합 테스트는 조용히 skip 된다).
  - 저장 버튼은 `disabled={!site || !form.name}`(3779행)이므로 빈 이름은 오지 않는다 — 서버 쪽 이름 비어 있음 분기를 상상해서 넣지 말 것.
  - **범위를 넓히지 말 것.** 「JSON Schema」 칸에 실시간 helperText 검증(입력 중 파싱)을 붙이는 것은 다른 과제다(보류 아이디어의 CIDR 화면 검증과 같은 범주). 이번엔 실패 안내만.
  - 보호 경로(internal/auth, internal/database/migrations, .github/workflows, docs/openapi.yaml)는 건드리지 않는다. `AdminNoticeAlert`(3357행, 세 Alert 이 공유하는 레이아웃)와 세 기존 `describe*` 문장 상수도 그대로 둔다 — 기존 테스트가 그 문장을 글자 그대로 단언한다.

- 차선 후보: **설정 저장(2411행) 실패를 한국어로 안내한다** — `putSetting`(internal/httpapi/admin.go:551~)이 `UNKNOWN_SETTING`(555)·`INVALID_PAYLOAD`(560)·`SECRET_SEAL_FAILED`(574)·`INVALID_SETTING`(581, `validateAdminSetting` 의 영문 문장들)·mcp.oauth 거절(588행 이하)을 돌려주는데 2411행이 `{save.error.message}` 다. 1순위보다 큰 이유: 한 Alert 이 9개 설정 그룹을 덮고 `validateAdminSetting` 의 하위 문장을 모두 읽어야 하므로 M 에 가깝다. 집으려면 `validateAdminSetting` 을 먼저 열어 코드별 문장 집합을 확정하고, `SettingsAdmin.save`(2168-2174행)가 그룹마다 순차 `put` 을 돌려 **중간 실패 시 일부만 저장된다**는 점을 안내 문구에 반영할 것(이것은 문구만으로는 못 고치는 별개 결함이므로, 손대려면 과제를 쪼갤 것).
