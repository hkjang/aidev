# 과제서 — 2026-10-03 (hunter)

- 과제: 발견 건 일괄 변경의 조치 기한 폼이 서버가 거절하는 **확장 연도(±YYYYYY) RFC3339** 를 보내지 않게 제출 직전 값으로 막기 (가치 3 / 위험 1 / 작업량 S)

- 왜: `web/src/finding-bulk-state.ts` 의 `findingBulkPatch` 는 `new Date(input.due)` 가 유한하기만 하면 `patch.due_date = date.toISOString()` 을 그대로 wire 로 보낸다(38-43행). ECMA-262 의 `Date.prototype.toISOString` 은 연도가 0000~9999 를 벗어나면 `+010000-01-01T00:00:00.000Z` 형태의 확장 연도를 만들고, 서버 `internal/app/finding_ops.go:67` `validateFindingOpsResource` 는 `time.Parse(time.RFC3339, s)` (레이아웃 `2006-01-02T15:04:05Z07:00`, 4자리 연도 고정)로만 받으므로 그 문자열을 400 으로 거절한다. `internal/app/finding_bulk.go:67-69` 가 그 오류를 `findingBulkError{400, …}` 로 감싸므로 **담당자·상태를 같이 바꾸던 일괄 변경 전체가 실패**하고, 사용자는 입력란 옆 안내가 아니라 왕복 후 폼 상단 오류(`finding-bulk.tsx:90-120` 의 `setError`)를 본다. 이 저장소에서 2026-09-28·10-01 에 반복 성공한 "폼이 검사하는 값 = 서버가 받는 값" 패턴의 남은 구멍이다.

## 선행 확인 (가장 먼저, 실패하면 차선으로)
정찰 환경에서 `node`·`go run`·`gh` 실행 권한을 받지 못해 **런타임 재현을 하지 못했다(미확인)**. 아래 두 가지를 먼저 확인하고, 둘 중 하나라도 어긋나면 1순위를 버리고 차선 후보로 가라.
1. `node -e 'const d=new Date("10000-01-01T00:00");console.log(Number.isFinite(d.getTime()), d.toISOString())'`
   → `true` 와 `+010000-…Z`(또는 로컬 TZ 보정된 확장 연도)가 나와야 한다.
2. 임시 Go 테스트로 `validateFindingOpsResource(map[string]any{"due_date": "+010000-01-01T00:00:00.000Z"})` 가 **non-nil** 인지 확인(= 서버가 거절).
   `go test -run '^TestFindingOps' -count=1 ./internal/app` 하네스는 DSN 없이 돈다(`internal/app/finding_ops_test.go:57-61` 이 이미 DSN 없이 이 함수를 직접 호출한다).
3. 도달 경로도 미확인이다: `web/src/finding-bulk.tsx:266-274` 의 `TextInput type="datetime-local"` 에 `min`/`max` 가 없어 브라우저 연도 칸이 5~6자리를 허용하면(Chrome 은 275760 까지) 숫자 오타로 들어온다. 브라우저 확인이 어렵다면 **"폼은 서버가 거절할 문자열을 보내지 않는다"는 계약 결함**으로 좁혀서 수정·테스트하고, 도달 가능성은 "미확인" 으로 적어라. 추측으로 단정하지 말 것.

## 수용 기준
1. `findingBulkPatch({changeDue:true, clearDue:false, due:"10000-01-01T00:00", …})` 가 **서버로 보내기 전에** 한국어 오류를 throw 한다. 기존 유효 입력(`"2026-10-03T12:00"`)의 반환 wire 문자열은 **한 글자도 바뀌지 않는다**.
2. `clearDue` 의 `due_date: null`, 빈 문자열·비유한 날짜 거절, 담당자·상태 분기는 현재 동작 그대로다(기존 web 테스트 전부 유지).
3. 공유 벡터가 두 경로를 함께 고정한다: 새 `internal/app/testdata/finding-bulk-due-date.json` 에 **wire 문자열 + 서버 수락 여부**를 적고 Go 쪽은 `validateFindingOpsResource` 로 그 판정을 단언, TS 쪽은 로컬 입력 집합에 대해 `findingBulkPatch` 가 **거절(throw) 하거나 벡터에서 accepted 로 표시된 문자열만** 만들어 내는지 단언한다. 경계로 최소 `9999-12-31T23:59`(수락 쪽)와 5자리 연도(거절 쪽), `+010000-…`·`-000001-…` wire 문자열을 포함한다.
4. 고친 뒤 검사를 되돌리면(예: 새 판정만 무력화) **그 테스트 하나만** 다시 실패하는 것으로 인과를 확정하고, 그 red 출력을 보고에 그대로 적는다.

## 건드릴 파일 (프로덕션 1~2개)
- `web/src/finding-bulk-state.ts` : `findingBulkPatch` 의 `input.changeDue && !input.clearDue` 분기 — `date.toISOString()` 결과를 변수에 받아 `/^\d{4}-/` 같은 4자리 연도 형태인지 **보내는 값 그대로** 검사하고 아니면 한국어 오류 throw. 담당자 바이트 검사(26-30행)와 `clearDue` 경로는 건드리지 말 것.
- (선택, 2번째 프로덕션 파일) `web/src/finding-bulk.tsx:266-274` : `datetime-local` 에 `max="9999-12-31T23:59"` 추가. 브라우저 동작을 실제로 확인할 수 없으면 넣지 말고 후속 아이디어로 남겨라.
- `web/tests/finding-bulk.test.mjs` : 현재 `due` 는 28행의 빈 문자열 기준선뿐 — 기한 분기 단언이 사실상 없다. 위 3)·4) 추가.
- `internal/app/testdata/finding-bulk-due-date.json` : 새 공유 벡터(기존 `finding-bulk-assignee.json`·`csv-safety.json` 의 형식을 따를 것).
- `internal/app/finding_bulk_validate_test.go` (또는 새 `finding_ops_due_test.go`) : 벡터를 `validateFindingOpsResource` 로 돌리는 Go 테스트. 기존 `TestFindingBulkAssigneeSharedVectors`·`TestFindingBulkValidationContract` 는 보존.

## 검증 명령 (이 저장소에서 실제로 도는 것)
```sh
npm --prefix web ci
npm --prefix web test            # pass 개수를 실측해 기준선과 대조 (10-02 회차 실측 104). 0건 실행을 통과로 읽지 말 것
npm --prefix web run typecheck
npm --prefix web run build
go test -run '^TestFinding' -count=1 -v ./internal/app   # SKIP(DSN 없음)을 통과로 세지 말 것
go vet ./internal/app
gofmt -l <바꾼 Go 테스트 파일>
git diff --check
```
Go 프로덕션 코드를 바꾸지 않으므로 `internal/webassets/dist` 재복사와 `go test -race ./...`(HUNTER_TEST_DSN 필요)는 불필요하다.

## 위험과 피할 것
- **서버 계약을 고치지 말 것.** `validateFindingOpsResource`·`finding_bulk.go` 는 무변경이다. 폼만 서버 계약에 맞춘다(09-28 회차에서 서버를 건드리지 않은 것이 통한 이유).
- `patch.due_date` 에 실제로 보내는 값 외의 다른 문자열로 검사하지 말 것. 이 저장소는 "검사한 값 ≠ 보낸 값" 으로 두 번 깨졌다(`finding-bulk-state.ts` trim, `sbomLabelError`).
- `web/src/resources.tsx:1110` 에 `new Date(values[f.key]).toISOString()` 로 **같은 모양의 문제**가 있다. 공통 폼 파급이 커서 이번 범위에서 제외한다 — 고치지 말고 후속 아이디어로만 적어라.
- auth/OIDC·migrations·`.github/workflows`·`Dockerfile`·`scripts/release.sh`·`third_party/pentagi` 는 건드리지 않는다(이번 과제와 무관).
- `web/node_modules` 는 새 워크트리에 없다 → `npm ci` 먼저. `npm test` 는 `${npm_node_execpath:-node}` 를 쓰는 현재 형태를 맨 `node` 로 되돌리지 말 것(AGENTS.md 7장).
- 순수 테스트만 늘리는 과제로 축소하지 말 것 — 이 저장소에서 테스트 전용 PR(#15, finding bulk validation contract)은 통과하지 못하고 닫혔다. 이번 변경의 중심은 **프로덕션 거절 추가**다.
- 2026-09-29·10-01 회차 산출물(`sbom-label.json` 등)은 **main 에 없다**. 그 주제를 다시 제출하지 말고 현재 main 기준으로만 작업하라.

## 차선 후보
`web/src/automation-state.ts:196-209` — `localDateTime` 이 `.slice(0, 16)` 으로 초를 잘라 `datetime-local` 에 넣고 `isoDateTime` 이 다시 `toISOString()` 으로 올리므로, 당직표에서 무관한 필드만 고쳐 저장해도 교대 시작·종료가 최대 59.999초 이동한다(2/2/S). 같은 두 함수에 왕복 보존 단언을 `web/tests/automation-state.test.mjs` 에 추가하고 초를 보존하도록 고친다. 서버 검사(`Date.parse(starts_at) >= Date.parse(ends_at)`)와 `expected_updated_at` 계약은 건드리지 않는다.
