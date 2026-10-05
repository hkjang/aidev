# 회차 노트 2026-10-05-085124-Momento-improve — Momento
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 08:51] base pinned — main@f25577c
- [러너 08:51] autonomy release — 

## 정찰 노트
- 앞 회차가 '다음 1순위' 로 지목한 설정 저장(AdminPage.tsx:2411)을 열어 보니 putSetting(admin.go:551~)이 코드 다섯 갈래에 validateAdminSetting 의 영문 문장 묶음까지 달려 M 이었다. 그래서 같은 계열에서 더 작은 조각인 Event Schema(3775행 ← upsertEventDefinition, 코드 세 개)를 골랐다 — 쪼갠 네 회차가 모두 통과한 패턴 그대로다.
- 이 조각이 앞 네 회차보다 나은 점: 가장 흔한 실패가 서버 영문이 아니라 **클라이언트의 V8 SyntaxError** 다(3709행 JSON.parse 가 mutationFn 안에 있다). 번역이 아니라 실제 누출을 막는 일이다.
- 추측으로 적은 것 하나: react-query 5.85.5 retryer 가 mutationFn 의 **동기** throw 를 save.error 로 잡는지 라이브러리 소스로 확인하지 못했다(worktree 에 web/node_modules 없음). 과제서가 브라우저 재현을 선행 조건으로 걸어 뒀다 — 영문 SyntaxError 가 Alert 에 실제로 뜨지 않으면 차선(설정 저장)으로 갈 것.
- 테스트는 돌리지 않았다(node_modules·MOMENTO_TEST_POSTGRES_DSN 없음). DEFINITION_SAVE_FAILED 500 은 여섯 자리가 공유하고 site_key 불일치로도 나므로 원인을 특정하지 말 것 — v0.34.55·56 이 피한 함정이다.
- 프로필(2026-10-01)은 다시 쓰지 않았다 — 4일 전이고 이번에 확인한 것(.ts 확장자 import, DataTable 단일 구현, npm_node_execpath test 명령, prettier 금지)이 모두 지금 코드와 일치했다.
- [러너 08:56] scout done — Event Schema 저장 실패를 한국어로 안내한다 — 손으로 적는 JSON 이 깨지면 지금은 V8 의 영문 `SyntaxError` 가 Al

## 구현 노트
- 「JSON Schema」 칸의 JSON 이 깨지면 `mutationFn` 안의 `JSON.parse` 가 먼저 던져 V8 영문이 Alert 에 그대로 떴다. `adminErrors.ts` 에 `describeEventDefinitionError` 를 새로 두고 Event Schema Alert 하나만 바꿨다(프로덕션 2파일 + 테스트 1파일, 테스트 199 → 211).
- **확신 없는 곳**: `DEFINITION_SAVE_FAILED` 를 DB 로 재현하지 않았다 — 여섯 자리가 함께 쓰는 500 이고 `site_key` 불일치가 `pgx.ErrNoRows` 로 같은 코드가 되므로 문구를 중립으로 두고 원인을 특정하지 않았다(`MOMENTO_TEST_POSTGRES_DSN` 없으면 통합 테스트는 조용히 skip). `INVALID_MODE` 는 「정책」 이 세 값짜리 select 라 화면에서 도달할 길이 없어 **실제 서버가 그것을 내는 것을 못 봤다** — 임시 서버로 흉내 낸 응답으로만 확인했다. 보조 패턴 `/JSON/` 은 `!code` 로 가뒀지만 cross-realm 상황 자체를 만들어 보지는 않았다(같은 realm 이라 `instanceof` 가 성립하는 것은 브라우저에서 확인).
- **일부러 하지 않은 것**: 입력 중 helperText 실시간 검증(범위 밖 — ideas.json 에 새 항목으로 적었다. 타이핑 중에는 거의 항상 깨진 JSON 이라 디바운스 설계가 필요하다). 다른 Alert 세 곳(2411 설정 저장·2955 보존 정책·3110 dimensions)과 Go, 세 기존 `describe*` 문장 상수는 손대지 않았다 — 기존 21건이 그 문장을 글자 그대로 단언한다.
- **다음 역할이 조심할 것**: `web/test/*.test.mjs` 는 `.ts` 확장자까지 적어 import 하므로 Node 22.18+/24+ 가 필요하다(이 환경 v22.23.1, `npm test` 는 `npm_node_execpath` 로 PATH 가림을 피한다). DB 는 필요 없다. `npx prettier --check` 는 게이트가 아니다(저장소에 prettier 의존성·CI 단계 없음). 브라우저 확인에 쓴 임시 하네스는 `/tmp/evh` 에만 두었고 `web/dist` 는 지웠다 — 커밋은 3파일뿐이다.
- [러너 09:04] brief accepted — 채택 — 인용한 행 번호(AdminPage.tsx 3709 `JSON.parse(form.schemaText)`·3768 `label="JSON Schema"`·3775 Alert·3350 `NetworkErrorAlert`·78-80 import
- [러너 09:05] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 직접 돌린 것: `npm test`(211/211), `npm run lint`, `npm run build` 모두 통과. **원장의 "항등 스텁" 재현을 /tmp 에 복제해 직접 재실행** — 새 12건 중 **10건 실패**, 기존 21건은 전부 통과. 테스트가 바뀐 경로를 실제로 지난다(`realParseError` 가 진짜 `JSON.parse` 를 호출). 만든 `web/dist`·/tmp 산출물은 지웠고 작업 트리는 깨끗하다.
- 코드 대조: `upsertEventDefinition`(admin.go:1395~)이 내는 코드가 정확히 `INVALID_PAYLOAD`·`INVALID_MODE`·`DEFINITION_SAVE_FAILED` 셋뿐이고, 주석이 주장한 `ON CONFLICT(site_id,name) DO UPDATE`(1421)·`site_key` 로 고르는 INSERT·500 여섯 자리가 모두 사실이었다. `client.ts` 가 `REQUEST_FAILED`/`RESPONSE_NOT_JSON` 으로 code 를 **항상** 채우므로 보조 패턴을 `!code` 로 가둔 것이 실제로 성립한다. 범위 이탈·보안·개인정보 쟁점 없음(새 엔드포인트·권한·의존성·비밀값 없음, MUI Typography 가 텍스트로 이스케이프).
- 못 본 것: 브라우저 DOM 배선(`EventDefinitionErrorAlert`, AdminPage.tsx:3782)은 node:test 범위 밖이라 **원장의 headless Chrome 기록을 신뢰**했고 재현하지 않았다. Go·Postgres 는 돌리지 않았다(DSN 없음 — 이 변경은 Go 를 건드리지 않는다).
- 승인이어도 남는 우려(다음 회차용): ① 오프라인 등 **전송 실패(`TypeError: Failed to fetch`)는 여전히 영문**으로 뜬다 — 네 `describe*` 가 공통으로 비어 있는 자리이고 이번 변경의 회귀는 아니다(변경 전과 동일). ② 모르는 코드(403·401 등)도 서버 영문 그대로 — 셋과 같은 **의도된 계약**이다. ③ `err.Error()` 로 pgx 원문(SQLSTATE·제약 이름)을 브라우저까지 보내는 것은 **서버 쪽에 그대로 남아 있다**(admin.go) — 이번엔 본문에서 caption 으로 내려간 것뿐. 릴리즈 노트는 "깨진 JSON 안내" 로 적으면 충분하다.
- [러너 09:09] review approved — 리뷰 승인 (risk=low)
- [러너 09:09] pr created — https://github.com/hkjang/Momento/pull/25
- [러너 09:15] ci passed — 검사 1개 모두 success
- [러너 09:15] merge done — 010499c
- [러너 09:26] release published — v0.34.57
- [러너 09:27] assets verified — v0.34.57 자산 1개 (이전 v0.34.56: 2)
