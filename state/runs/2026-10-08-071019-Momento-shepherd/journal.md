# PR 처리기 노트 2026-10-08-071019-Momento-shepherd — Momento PR #29
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-08-053931-Momento-improve)
# 회차 노트 2026-10-08-053931-Momento-improve — Momento
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 05:39] base pinned — main@992bb75
- [러너 05:39] autonomy release — 

## 정찰 노트
- 네 회차 연속 차선이던 「JSON Schema」 helperText 를 1순위로 올렸다 — 코드로 확인하니 「저장」 이 `disabled={!site || !form.name}`(AdminPage.tsx:3815) 로 schemaText 를 아예 안 보고 helperText 도 없다. 다른 후보를 제친 이유: 남은 Alert 네 곳(차선)은 같은 패턴의 여섯 번째 반복이라 값이 낮고, 설정 저장·순차 PUT·createNetwork 는 각각 M 이거나 Postgres 가 없어 재현 불가, 표 검색·CSV format·rowKey 는 '의도일 수 있음' 을 가리지 못해 또 미뤘다.
- 이 과제가 선례보다 안전한 이유를 과제서에 적었다 — 판정자가 서버가 아니라 같은 `JSON.parse` 라서 '화면이 서버보다 좁아짐' 이 구조적으로 불가능하다. 그래서 「저장」 을 닫는 것이 이 칸에서만 정당하다는 경계도 함께 적었다.
- 확신 없는 곳(과제서에 미확인으로 적음): web 테스트 총수(이전 회차 기록 248 을 그대로 옮겼고 직접 돌리지 않았다 — `npm ci` 가 수 분이라 생략). 구현자는 변경 전 숫자를 직접 찍고 시작할 것.
- 확인한 것 하나를 과제서의 '피할 것' 에 넣었다 — 서버는 `Schema map[string]any`(admin.go:1401)로 받으므로 `[1,2]`·`"hi"` 는 400 이지만 **`null` 은 통과한다**. 그래서 '객체가 아니면 막는다' 는 규칙은 화면을 서버보다 좁게 만든다. 이번엔 **구문만** 판정한다.
- 구현자가 조심할 것: adminErrors.ts:352 의 문장을 상수로 뽑을 때 **글자를 바꾸지 말 것**(adminErrors.test.mjs 가 `/JSON Schema/` 로 단언하고 다른 자리들은 문장을 글자 그대로 본다), `.ts` 확장자 붙인 값 import, 그리고 브라우저 하네스는 사례마다 새 탭(useUnsavedWarning 의 beforeunload).
- [러너 05:45] scout done — 「JSON Schema」 칸이 저장 전에 JSON 구문 오류를 말한다 (순수 모듈 `schemaTextRule.ts`) (가치 3 / 위험 1 / 작업량 

## 구현 노트
- 무엇을 왜: 「JSON Schema」 칸의 깨진 JSON 은 뮤테이션 안 `JSON.parse` 가 던져 **요청이 서버에 가지도 않는데** 「저장」 이 열려 있어 사용자가 헛클릭 왕복을 한 번 치렀다. 순수 모듈 `schemaTextRule.ts` 를 두어 `adminErrors.ts` 의 안내와 칸 아래 helperText 가 같은 상수(`JSON_SCHEMA_RULE`)를 쓰게 하고 「저장」 에 `!!blocking` 을 더했다. 문장 글자는 그대로다.
- 확신 없는 곳: (1) 서버가 `null` 을 통과시킨다는 것은 **코드 독해로만** 확인했다(`Schema map[string]any` → nil map → `json.Marshal` 이 `null`). 이 환경에 Postgres 가 없어 실제 200 을 받아 보지는 못했다. 이 전제가 틀리면 '구문만 판정' 이라는 선택의 근거 하나가 약해지지만, 판정이 더 좁아지지는 않는다. (2) `judgeSchemaText` 의 `detail`(엔진 원문)은 **아무도 렌더링하지 않는다** — 타입 계약으로만 두고 캡션을 만들지 않았다(이유는 아래).
- 일부러 하지 않은 것: `detail` 을 helperText 안에 캡션으로 넣지 않았다 — 넣으면 V8 영문이 helperText DOM 본문에 섞여 수용 기준 3 과 충돌한다. 저장 실패 Alert 이 이미 같은 값을 캡션으로 보인다. 타입 판정(객체 아님)도 하지 않았다(`null` 이 서버를 통과하므로 막으면 화면이 좁아진다 — ideas.json 에 별도 회차로 적었다). 서버·`admin.go`·Go·다른 Alert·다른 저장 버튼은 손대지 않았다.
- adminErrors 쪽 단언 1건은 **변경 전후 모두 통과하는 드리프트 가드**다(두 문장이 아직 같으므로). red 가 된 것은 새 모듈 테스트 쪽 5건이다 — 비평가가 "red 를 못 봤다" 로 읽지 않도록 적어 둔다.
- 다음 역할이 조심할 것: 브라우저 하네스는 /tmp/jsonharness 에만 있고 커밋에 없다(`web/dist` 도 지웠다). 폼을 채운 탭은 `useUnsavedWarning` 의 `beforeunload` 로 재이동이 30초 타임아웃에 걸리니 사례마다 새 탭을 쓸 것. `AdminPage.tsx` 의 import 는 확장자 없이(`./schemaTextRule`), `adminErrors.ts`·테스트 쪽은 `.ts` 를 붙여야 한다 — node --test 가 후자를 직접 읽는다.
- [러너 06:03] brief accepted — 채택 — 인용한 행 번호(AdminPage.tsx 3745 `JSON.parse(form.schemaText)`·3804 `label="JSON Schema"`·3815 `disabled={!site || !form.name}`·80 `describ
- [러너 06:03] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: 항등 스텁(`judgeSchemaText`→`{}`, `SCHEMA_TEXT_HELP`→`""`)으로 바꿔 직접 red 를 재현했다 — 5건 실패로 원장과 일치. 복원 후 `npm test` 259/259·`lint`·`build` green. `adminErrors.ts:352` 의 문장을 상수로 뺀 것은 main 의 그 줄과 `cmp` 바이트 비교로 동일(IDENTICAL). 서버 `null` 통과도 `admin.go:1401`(map[string]any→nil)·`1413`(json.Marshal→`null`) 로 확인.
- '화면이 서버보다 좁아짐' 을 가장 엄격히 봤고 성립한다 — 판정자가 `AdminPage.tsx:3746` 과 같은 렌더·같은 값·같은 `JSON.parse` 라서 막히는 값은 요청이 만들어지지 않는 값뿐이다. 범위 이탈·보안·법무 차단 사유 없음, revert 로 완전히 복구된다. → **approve / risk low / blocking 없음**.
- 못 본 것: 커밋되지 않은 브라우저 DOM 하네스는 재실행하지 않았다(배선은 `3756`·`3816-3817`·`3822` 독해 + tsc 통과로만 확인). Postgres 가 없어 `null` 의 실제 200 도 받아 보지 못했다 — 틀려도 판정이 좁아지지는 않는다.
- 남는 우려(다음 회차 1순위 후보): **`PlatformAdminPage.tsx:264` 의 「JSON Schema」 칸이 똑같이 무방비다** — `124` 행이 같은 `JSON.parse(eventSchema)` 를 뮤테이션 안에서 돌리고 버튼은 `!eventName` 만 본다. 이번 변경은 정당하게 손대지 않았고, 그래서 같은 이름의 칸이 두 화면에서 지금 다르게 행동한다.
- 릴리즈 노트에 적을 동작 변경 한 줄: 「JSON Schema」 칸을 완전히 비우면 이제 「저장」 이 닫힌다(전에는 눌리고 뮤테이션이 던졌다). 나머지 `SchemaTextJudgement.detail` 의 프로덕션 소비자 부재는 결함이 아니라 취향 문제로 둔다.
- [러너 06:07] review approved — 리뷰 승인 (risk=low)
- [러너 06:08] pr created — https://github.com/hkjang/Momento/pull/29
- [러너 06:11] ci failed — 성공이 아닌 검사: test=failure · 실패한 검사: ? 잡: test 

## 수리 노트
- 지적은 맞았다 — govulncheck 실패를 로컬에서 그대로 재현했다(exit 3, GO-2026-6629). 다만 **이 PR 이 만든 결함이 아니다**: 브랜치 diff 는 web 파일 5개뿐이고 Go·go.mod 를 건드리지 않는다. 새 권고가 main 의 x/text v0.39.0 핀을 빨간불로 만든 것이라, 수리 대상은 PR 의 로직이 아니라 의존성 핀이었다.
- 고친 방법: `go get golang.org/x/text@v0.41.0` + `go mod tidy` → go.mod/go.sum 만 변경(x/text v0.39.0→v0.41.0, 그 모듈 요구로 x/sync v0.21.0→v0.22.0). 코드·테스트·CI 워크플로는 무수정. 「JSON Schema」 변경 자체는 비평가가 approve 한 그대로 둔다.
- 재검증: govulncheck exit 0(`No vulnerabilities found`), `go vet` 통과, `go test -race ./cmd/... ./internal/...` 전부 ok, web `npm ci && lint && test 259/259 && build` 통과. `web/dist` 는 지웠고 커밋은 go.mod/go.sum 2파일.
- CI 는 govulncheck(52행)가 sdk(53)·web(55) 보다 먼저라 5bf68a6 회차에서 **sdk·web 단계는 실행조차 안 됐다** — web 은 위처럼 로컬로 돌려 메웠지만 **sdk 게이트와 `npm audit`·Docker build 는 돌리지 않았다**(sdk 는 이 PR·이 커밋이 손대지 않은 영역).
- 확신 없는 곳: x/sync 동반 상승이 pgx 런타임에 주는 영향은 -race 단위 테스트까지만 확인했고, Postgres 가 없어 통합 테스트는 조용히 skip 됐다 — 실제 DB 연결 경로(`database.Open`)는 로컬에서 실행해 보지 못했다.

## 심사 노트
- 확인한 것: 항등 스텁 사본으로 red 5건을 직접 재현했고(저장소 무수정) 원본은 55/55 green. CASES 표가 같은 전역 `JSON.parse` 에 실제로 넣어 등가성을 못 박으므로 화면이 뮤테이션보다 좁아질 수 없다 — 막히는 값은 요청이 만들어지지 않는 값뿐이다.
- 확인한 것: Docker build 를 뺀 CI 전 단계를 로컬에서 재현해 모두 통과(go test -race·vet·govulncheck `No vulnerabilities found`·sdk audit/typecheck/29건/build·web audit/lint/259건/build). 수리 노트가 못 돌렸다고 적은 sdk 게이트와 `npm audit` 이 이 심사에서 메워졌다.
- 확인한 것: 보호 파일 무수정, 새 경로·권한·개인정보 처리 없음. `detail`(V8 원문)은 helperText 가 렌더하지 않고 저장·전송·로그되지 않아 '감사 details' 규칙에 걸리지 않는다. 의존성 상승은 간접 2개(BSD-3-Clause), revert 로 완전 복구.
- 못 본 것: Docker build(환경에 docker 없음), headless Chrome DOM 하네스(배선은 AdminPage.tsx:3756·3815-3816·3822 독해 + tsc + eslint 로만), Postgres 부재로 Go 통합 테스트 skip — 서버의 `schema=null` 통과는 admin.go:1400·1413 독해로만.
- 권고 `merge`: 결함 없음, risk low. 범위 밖 참고로 PlatformAdminPage.tsx:259-265 의 같은 칸이 아직 무방비임을 남겼다(다음 회차 후보).
