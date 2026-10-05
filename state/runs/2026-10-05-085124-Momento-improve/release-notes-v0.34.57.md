# Momento v0.34.57 — Event Schema 를 저장하다 실패하면 한국어로 무엇을 고칠지 말해 줍니다

콘솔 **관리 → Event Schema** 의 「저장」 이 실패했을 때 띄우는 문장이 달라집니다. 지난 세 릴리스가 사용자 다이얼로그(v0.34.54)·사이트 다이얼로그(v0.34.55)·망 구분 폼(v0.34.56)에 한 것을 이 카드에 합니다. 서버·API·데이터베이스는 그대로입니다.

## 이번 것은 서버 영문이 아니라 **브라우저 엔진**의 문장이었습니다

앞의 세 릴리스가 고친 것은 서버가 보낸 영문이었습니다. 이 카드는 다릅니다 — **네 폼 중 유일하게 가장 흔한 실패가 서버에 닿지조차 않습니다.**

「JSON Schema」 칸은 다섯 줄짜리 자유 입력이고, `save` 뮤테이션이 `mutationFn` **안에서** `JSON.parse(form.schemaText)` 를 평가합니다. 그래서 한 글자만 깨져도 요청이 만들어지기 전에 V8 이 던지고, react-query 가 그 **동기** throw 를 `save.error` 에 담아 Alert 이 엔진 문장을 그대로 띄웠습니다.

```
Unexpected token '}', "{"properties": }" is not valid JSON
```

어느 칸을 어떻게 고쳐야 하는지는 **그 어디에도 없습니다.** 화면에는 자유 입력 칸이 셋인데 엔진의 문장은 그중 어느 것도 가리키지 않습니다.

서버까지 간 실패도 같았습니다. `internal/httpapi/admin.go` 의 `upsertEventDefinition`(`:1395-1446`)이 `INVALID_MODE` 의 영문과 `DEFINITION_SAVE_FAILED` 의 `err.Error()` 를 돌려주므로 pgx 원문이 브라우저까지 흘렀습니다.

```
ERROR: ... (SQLSTATE 23505)
no rows in result set
```

## 이제 무엇을 고칠지 말합니다

| 실패 | 화면에 뜨던 것 | 이제 뜨는 것 |
|---|---|---|
| 깨진 JSON (서버에 닿지 않음) | `Unexpected token '}', … is not valid JSON` | 「JSON Schema」 칸의 내용이 올바른 JSON 이 아닙니다 + 짝·쉼표·겹따옴표를 확인하라는 말, 비울 때는 `{}` + 작은 글씨로 **엔진 원문** |
| 「정책」 값 | `mode must be one of …` | 「정책」 값이 올바르지 않습니다. 화면을 새로 고친 뒤 allow·warn·reject 중에서 다시 고르세요 |
| 본문을 읽지 못함 | 영문 한 문장 | 보낸 내용을 읽지 못했습니다 + 작은 글씨로 **서버 원문** |
| 저장 500 | pgx 원문 | 이벤트 규격을 저장하지 못했습니다 + 작은 글씨로 **서버 원문** |
| 요청이 닿지 않음 | 영문 한 문장 | 요청이 서버에 닿지 못했습니다 |

안내는 **어느 칸인지를 이름으로 말합니다** — 엔진의 문장이 끝내 말하지 못한 것이 그것이기 때문입니다.

### JSON 구문 분기는 `switch` 앞에 둡니다

이 에러에는 `code` 가 **없습니다.** 코드만 보고 분기하면 `default` 로 떨어져 엔진의 영문이 그대로 나갑니다. 그래서 구문 분기가 `switch` **앞**에 섭니다.

판정은 `error instanceof SyntaxError` 로 합니다. 메시지 패턴 `/JSON/` 은 보조로만 두고, **`code` 가 없을 때만** 봅니다 — 코드가 붙은 서버 거절의 메시지에 'JSON' 이라는 낱말이 들어 있어도 구문 분기가 가로채지 않습니다. 그것까지 테스트가 단언합니다.

## 500 은 원인을 말하지 않습니다

`DEFINITION_SAVE_FAILED` 의 문구는 일부러 중립으로 두었습니다. 이 500 은 **서로 다른 여섯 자리**가 함께 씁니다(Begin, INSERT, 버전 조회, 두 Exec, Commit).

게다가 그 INSERT 는 `site_key` 로 사이트를 고르므로, 맞는 사이트가 없으면 0행이 삽입되어 `pgx.ErrNoRows` 로 **같은 500** 이 됩니다 — 데이터베이스 장애와 구별할 표지가 없습니다.

특히 **'이미 등록된 이벤트' 라고 설명하지 않습니다.** 그 INSERT 에는 `ON CONFLICT(site_id,name) DO UPDATE` 가 붙어 있어 이름이 겹치면 **거절이 아니라 갱신으로 흡수됩니다.** 중복이라고 말했다면 그것은 그냥 거짓입니다. 원인은 이 기계에서 데이터베이스로 재현하지 않았으므로 특정하지 않았습니다.

서버 원문은 지우지 않고 앞의 세 Alert 과 같은 모양의 작은 글씨로 함께 둡니다 — 관리자에게 전달할 유일한 단서입니다.

## 앞의 세 함수와 합치지 않았습니다

`web/src/pages/adminErrors.ts` 에 `describeEventDefinitionError` 를 **따로** 두었습니다. `upsertEventDefinition` 은 자기 코드 셋을 답하고 고쳐야 할 칸이 다릅니다. 하나의 `switch` 가 네 계약을 대신 답하게 만들지 않았습니다.

대신 **에러에서 코드와 메시지를 읽어내는 `refusal` 하나만** 네 경로가 공유합니다. 공용 상수 `UNREADABLE_PAYLOAD`·`LOST_REQUEST`·`PASS_TO_ADMIN` 은 **재사용만 하고 문장은 그대로 두었습니다** — 기존 단언들이 글자 그대로 묶고 있어, 통과한다는 사실이 곧 바뀌지 않았다는 증명입니다. 화면 쪽도 같은 모양입니다: `AdminNoticeAlert` 이 배치를 맡고 `EventDefinitionErrorAlert` 은 어느 `describe*` 에 물을지만 정합니다. 모르는 코드는 **서버 메시지로 되돌아갑니다.**

## 확인

`web/test/adminErrors.test.mjs` 가 이 성질을 고정합니다(199 → **211**건).

순수 함수 테스트에 더해 **실제 프로덕션 배선**을 확인했습니다 — `npm run build` 의 dist 를 `/api/v1/me`·`/api/v1/sites`·`/api/v1/event-definitions` 를 흉내 낸 임시 서버에 올리고 headless Chrome 으로 `/admin?section=schemas` 의 「JSON Schema」 칸을 실제로 채우고 「저장」 을 눌러 Alert DOM 을 읽었습니다. 에러 객체는 손으로 만든 대역이 아니라 앱 자신이 만든 진짜 `SyntaxError` 와 `api()` 가 HTTP 응답에서 만든 진짜 `APIError` 입니다. 19개 확인 전부 기대대로였고, 고치기 전 main 그대로의 dist 로 같은 하네스를 돌리면 **17건이 실패**했습니다.

그 과정에서 '요청이 가기 전에 던져진다' 는 진단도 함께 증명했습니다 — 깨진 JSON 으로 「저장」 을 누른 뒤 **서버에 도달한 POST 가 0건**이었습니다.

릴리스 검증은 이전과 같습니다 — `go vet`·`go test` 전 패키지 통과, sdk `npm audit`(`found 0 vulnerabilities`)·typecheck·테스트 27건·build, 콘솔 `npm audit`(`found 0 vulnerabilities`)·`eslint`·테스트 **211건**·`npm run build` 모두 통과합니다.

## 범위

바꾼 것은 프로덕션 2파일(`web/src/pages/adminErrors.ts`, `AdminPage.tsx` 의 Alert 한 곳)과 테스트 1파일입니다. **서버는 손대지 않았습니다** — `admin.go` 가 검증의 정본으로 남고 화면은 거울입니다. `describeUserError`·`describeSiteError`·`describeNetworkError` 와 나머지 Alert 세 곳(설정 저장·보존 정책·dimensions)은 각각 다른 핸들러의 코드 집합이라 이번에 포함하지 않았습니다. Go·API 변경과 데이터베이스 마이그레이션은 없습니다.
