- 과제: 발견 건 일괄 변경의 조치 담당자 200바이트 검사를 서버와 같은 문자열에 대해 세고, 공유 벡터로 두 파서를 함께 검증 (가치 3 / 위험 1 / 작업량 S)

- 왜: 같은 200바이트 한도를 웹과 서버가 **서로 다른 문자열**에 대해 센다. `web/src/finding-bulk-state.ts:26` 은 `new TextEncoder().encode(input.assignee.trim()).length > 200` 로 **trim 한 뒤** 세고, `internal/app/finding_bulk.go:63` 은 `len(name) > 200` 으로 **trim 하기 전** 원문 바이트를 센 다음 `in.Patch[field] = strings.TrimSpace(name)`(66행) 로 trim 해서 저장한다. 그래서 내용이 정확히 200바이트이고 앞뒤에 공백이 붙은 담당자는 웹 검사를 통과해 요청이 나가고 서버가 400 "조치 담당자는 줄바꿈 없이 UTF-8 200바이트 이하로 입력하세요" 로 거절한다 — 사용자는 폼 안의 한국어 안내 대신 왕복 후 서버 오류를 본다. 덧붙여 `validateFindingBulk` 는 DB 없이 돌 수 있는 순수 함수인데 현재 순수 단위 테스트가 **하나도 없다**(`grep validateFindingBulk internal/app/*_test.go` 결과 0건 — DB 필요한 `finding_bulk_test.go` HTTP 경로로만 간접 검증). 이 회차에 웹 검사를 서버 계약에 맞추고 두 파서를 하나의 공유 벡터로 함께 돌리면 불일치가 닫히고 순수 테스트 공백도 함께 메워진다.

- 방향(중요): **서버 계약을 바꾸지 말고 웹을 서버에 맞춘다.** 서버가 받아들이는 집합을 넓히는 것은 `openapi.json` 계약 변경이고 보호 경로(`finding_bulk.go` 는 원자적 일괄 변경·전체 롤백 경로)를 건드린다. 이번 과제의 프로덕션 변경은 **웹 1파일**뿐이다.

- 수용 기준:
  1) `findingBulkPatch({changeAssignee: true, assignee: <앞뒤 공백 + 200바이트 내용>})` 가 `web/src/finding-bulk-state.ts` 의 한국어 오류("담당자는 UTF-8 기준 200바이트까지 입력할 수 있습니다.")를 던진다 — 즉 웹이 서버가 400 을 낼 입력을 서버에 보내지 않는다.
  2) 저장되는 값은 그대로다: 검사를 통과한 입력에 대해 `patch.assignee` 는 여전히 `input.assignee.trim()` 이다(바이트 **검사**만 원문 기준으로 바뀌고, **보내는 값**은 trim 된 값 그대로 — 한 글자도 바뀌지 않아야 한다).
  3) 새 공유 벡터 `internal/app/testdata/finding-bulk-assignee.json` 의 모든 사례에서 Go `validateFindingBulk` 와 TS `findingBulkPatch` 가 **같은 수락/거절 판정**을 낸다. 수정 전에는 최소 1개 사례가 TS 쪽에서 실패해야 한다(먼저 실패를 확인할 것).
  4) 제어 문자 판정은 양쪽 모두 원문 기준·같은 집합임을 벡터가 단언한다(C0 `\u0000-\u001f`, DEL·C1 `\u007f-\u009f`).

- 건드릴 파일 (프로덕션 1 + 테스트 3):
  - `web/src/finding-bulk-state.ts:26` — `findingBulkPatch` 의 길이 검사를 `new TextEncoder().encode(input.assignee).length > 200`(원문 기준)으로 바꾼다. 24~25행의 제어 문자 검사와 28행의 `patch.assignee = input.assignee.trim()` 은 **그대로 둔다**. 왜 원문 기준인지 한 줄 주석으로 남길 것(서버 `finding_bulk.go` 가 trim 전 `len` 으로 재는 계약).
  - `internal/app/testdata/finding-bulk-assignee.json` (신규) — 기존 `testdata/csv-safety.json`·`testdata/tracking-origins.json` 과 같은 모양의 사례 배열. 각 사례: `{"name":"…","assignee":"…","accepted":true|false,"reason":"length|control|ok"}`. 최소 포함: (a) 200바이트 한글 내용 + 앞뒤 공백 1칸씩(거절 — 이번 회차의 실패 사례), (b) 정확히 200바이트 공백 없음(수락), (c) 201바이트(거절), (d) 앞뒤 공백만 있는 값(수락, trim 후 빈 문자열 = 담당자 해제), (e) `\n` 포함(거절·control), (f) `\u007f` 포함(거절·control), (g) `\u009f` 포함(거절·control), (h) 한글·이모지 섞인 멀티바이트 경계값.
  - `internal/app/finding_bulk_validate_test.go` (신규) — `os.ReadFile("testdata/finding-bulk-assignee.json")` → 각 사례로 `findingBulkRequest{Items: []findingBulkItem{{ID:"f1", UpdatedAt:"2026-09-28T00:00:00Z"}}, Patch: map[string]any{"assignee": c.Assignee}}` 를 만들어 `validateFindingBulk(&in)` 을 호출하고 `err == nil` 이 `accepted` 와 같은지 단언. 수락 사례에서는 `in.Patch["assignee"] == strings.TrimSpace(c.Assignee)` 도 단언(서버가 trim 해서 저장한다는 계약 고정). **모델 코드: `internal/app/report_csv_test.go:22`(`os.ReadFile` + 순수 함수, DSN 불필요).** `testApp`/`request` 헬퍼를 쓰지 말 것 — 쓰면 `HUNTER_TEST_DSN` 없이 skip 된다.
  - `web/tests/finding-bulk.test.mjs` (신규, 또는 `web/tests/convenience.test.mjs` 의 `finding-bulk-state.ts` 블록 확장) — 같은 JSON 을 `new URL("../../internal/app/testdata/finding-bulk-assignee.json", import.meta.url)` 로 읽어 `findingBulkPatch` 판정을 대조. **모델 코드: `web/tests/convenience.test.mjs:138`, `web/tests/tracking-state.test.mjs:212-218`.**

- 검증 명령 (이 저장소에서 실제로 도는 것):
  ```sh
  # Go 순수 테스트 — DB 불필요. skip 되면 testApp 을 썼다는 뜻이니 고칠 것.
  go test -run 'FindingBulk' -count=1 ./internal/app
  go vet ./internal/app
  gofmt -l internal/app/finding_bulk_validate_test.go   # 빈 출력이어야 함 (gofmt -w . 금지)

  # 웹
  npm --prefix web ci
  npm --prefix web test          # node --experimental-strip-types --test tests/*.test.mjs
  npm --prefix web run typecheck # tsc --noEmit
  npm --prefix web run build
  git diff --check
  ```
  - **기준선을 먼저 실측하고 기록할 것**: `npm --prefix web test` 의 통과/실패/skip 수. 직전 회차(2026-09-27) 실측은 **97 통과 / 0 실패 / 0 skip** 이었다. 이 값을 그대로 믿지 말고 착수 시점에 직접 돌려 확인하고 보고에 적을 것.
  - 프로덕션 변경이 웹 1파일이고 Go 프로덕션 코드는 무변경이므로 `internal/webassets/dist` 재복사와 전체 `go test -race ./...` 는 필요 없다. 하지 않았다면 "하지 않음" 으로 적을 것(통과로 보고하지 말 것).

- 위험과 피할 것:
  - **서버(`internal/app/finding_bulk.go`)의 프로덕션 코드를 고치지 말 것.** 63행을 `len(strings.TrimSpace(name))` 로 바꾸는 것도 가능한 수정이지만 그것은 서버가 받는 집합을 넓히는 **계약 변경**이고, 이 파일은 원자적 일괄 변경·전체 롤백·현재 권한 재검사 경로다. 이번 회차 범위 밖.
  - **trim 집합이 Go 와 JS 에서 완전히 같지 않다(미확인 아님 — 코드 읽기로 확인된 차이).** JS `String.prototype.trim` 은 U+FEFF 를 제거하지만 Go `strings.TrimSpace`(=`unicode.IsSpace`)는 **U+FEFF 를 제거하지 않는다**. 그래서 "trim 후 값이 같다" 를 벡터로 단언하려면 U+FEFF·U+2000~U+200A·U+3000 을 **벡터에 넣지 말거나**, 넣는다면 "의도된 차이" 로 벡터 파일에 주석/필드로 먼저 문서화할 것. 수용 기준 3)은 **수락/거절 판정 일치**만 요구한다 — trim 결과 문자열 일치까지 양쪽에서 대조하려 들지 말 것.
  - 제어 문자 집합은 실제로 같다: Go `unicode.IsControl` 은 Unicode 카테고리 Cc = U+0000–U+001F + U+007F–U+009F 이고 TS 정규식 `[\u0000-\u001f\u007f-\u009f]` 와 동일하다(코드 읽기로 확인, 런타임 대조는 이번 벡터가 처음으로 증명함).
  - **`web/src/list-*.ts`·`saved-list-views.ts` 는 건드리지 말 것.** 2026-09-23·09-26·09-27 세 회차가 연속으로 그 축을 고쳤다. 이번 과제와 무관하다.
  - `tracking*`/CSP 경로는 2026-09-25 두 회차가 verify-failed·기각. 접근 금지.
  - 손으로 만든 대역이 아니라 실제 `validateFindingBulk`·실제 `findingBulkPatch` 를 호출할 것. `findingBulkRequest` 는 unexported 이므로 테스트는 반드시 `package app` 내부 테스트로 둘 것.
  - 수정 전 실패를 먼저 확인할 것(TDD): 벡터 사례 (a) 가 수정 전 TS 테스트에서 실패하는 것을 눈으로 본 뒤 구현하고, 구현 후 검사를 `.trim()` 으로 되돌려 **그 사례 하나만** 다시 실패하는지로 인과를 확정할 것.

- 이번 회차에 **하지 말 것**(범위 밖으로 확인만 해 둔 것들):
  - `web/src/saved-list-views.ts:139` 의 `["10","50","100"]` 은 `list-view.ts:15` 의 `pageSizes = [10,25,50,100]` 와 달라 보이지만 **버그가 아니다**: `use-list-view.tsx` 의 `setPageSize` 가 `size: value === 25 ? null : String(value)` 로 25 를 URL 에 절대 쓰지 않고 `readListState` 의 기본값도 25 다. 고치지 말 것(무동작 변경).
  - `savedViewSaveError` 에 이름 길이(60자) 검사가 없지만 `list-tools.tsx:306` 의 `maxLength={60}` 이 UTF-16 코드유닛으로 같은 한도를 걸어 `readListPreferences` 의 `name.length > 60` 폐기에 도달할 수 없다. 버그 아님.
  - `resources.tsx:1109` 의 `type: "datetime"` 직렬화가 빈 값을 `""` 로 보내지만 `finding_ops.go:141` 의 `has_manual_due = coalesce(data->>'due_date','')<>''` 가 이를 "기한 없음" 으로 정확히 처리한다. 버그 아님.

- 차선 후보: **`internal/app/validateFindingBulk` 의 나머지 계약(items 1~100개, ID 200자, 중복 ID 거절, `updated_at` RFC3339Nano 필수, 미지원 필드 거절, status 5종)을 DB 없는 순수 테스트로 고정.** 1순위가 성립하지 않으면(예: 웹 쪽 실패 사례를 실제로 재현하지 못하면) 같은 신규 파일 `internal/app/finding_bulk_validate_test.go` 하나만으로 이 테스트 공백을 메운다 — 프로덕션 코드 0파일, `go test -run 'FindingBulk' -count=1 ./internal/app`, DSN 불필요. AGENTS.md §4 의 "일괄 변경은 최대 100개·각 updated_at 검사" 계약이 현재 순수 테스트로 전혀 고정돼 있지 않다.
