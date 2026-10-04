# 과제서 (2026-10-04, base main@db87cfc / VERSION 1.20.0)

- **과제**: 일반 자원 공통 폼(`formBody`)의 `datetime` 필드도 **보내는 wire 문자열**을 검사해, 서버가 `time.Parse(time.RFC3339, …)` 로 거절하는 확장 연도(`+YYYYYY`)와 Invalid Date 를 제출 전에 한국어 오류로 막기 (가치 4 / 위험 2 / 작업량 S~M)

## 왜

v1.20.0 이 발견 건 **일괄** 변경(`findingBulkPatch`)에서 "보내는 값 그대로 4자리 연도 검사"를 닫았지만, 같은 결함이 **단일 자원 공통 폼**에 그대로 남아 있다. `web/src/resources.tsx:1110` 의 `formBody` 는

```ts
if (f.type === "datetime") {
  out[f.key] = values[f.key] ? new Date(values[f.key]).toISOString() : "";
}
```

로 **아무 검사 없이** 직렬화한다 — 일괄 폼에 있던 `Number.isFinite` 유한성 검사조차 없다. 네 개의 `datetime` 필드(아래 "확인한 사실" 참조)가 모두 Go `time.Parse(time.RFC3339, s)`(연도 정확히 4자리·부호 없음)로 읽히므로, UTC 서쪽 시간대에서 `9999-12-31T23:59` 을 입력하면 `+010000-01-01T04:59:00.000Z` 가 나가 **같은 폼의 다른 모든 필드까지 함께 400** 으로 되돌아온다(브라우저 시간대에 따라 수락/거절이 갈린다). 고치면 사용자가 왕복 후 서버 오류 대신 해당 입력란 맥락의 한국어 안내를 받고, 일괄/단일 두 경로가 같은 계약을 쓴다.

## 확인한 사실 (이 세션에서 직접 열어 봄)

- `web/src/resources.tsx:1099-1119` `formBody(fields, values)` — **export 되어 있지 않음**. 호출자는 `resources.tsx:1477` `withEditRevision(formBody(cfg.fields, values), edit)` 하나뿐이고, 그 결과가 POST/PUT 본문 전체가 된다(변경하지 않은 필드도 함께 나간다).
- `datetime` 타입 필드는 정확히 4개: `resources.tsx:269` findings `due_date`, `:276` findings `expires_at`(위험 수용 만료), `:436` schedules `next_run_at`, `:470` scopes `expires_at`(승인 만료).
- 렌더러는 `resources.tsx:1066-1080` 의 Mantine `TextInput type="datetime-local"` 이며 **`max` 속성이 없다** → 9999 년 입력 가능.
- 서버 측 판정자는 모두 `time.Parse(time.RFC3339, …)`:
  - `internal/app/finding_ops.go:67` `validateFindingOpsResource(values map[string]any) error` — due_date, 빈 문자열 허용, **DB 불필요(순수)**
  - `internal/app/domain_schedules.go:14` `validateSchedule(m map[string]any) error` (35행에서 `next_run_at` 파싱) — **순수**
  - `internal/app/policy.go:45` `validateScope(m map[string]any) error` (67행) — **순수이지만 `expires.After(time.Now())` 도 함께 요구**한다. 과거 연도는 "형식 오류"가 아니라 "미래 아님"으로 거절되므로 벡터에서 이유를 섞지 말 것.
  - findings `expires_at` 은 `internal/app/domain.go:487`·`:704` 와 `policy.go:232`, `notifications_queue.go:478` 가 `time.Parse(time.RFC3339, …)` 로 읽는다(집계·만료 판정).
- `web/src/resource-form-state.ts`(79행, export 3개: `changeResourceField`, `withEditRevision`, `resourceDetailPath`)가 `resources.tsx:83` 에서 import 되는 **순수 모듈**이고, `web/tests/resource-form-state.test.mjs` 가 `../src/resource-form-state.ts` 를 직접 import 한다. **`.tsx` 는 Node 타입 스트리핑으로 import 할 수 없으므로 순수 로직은 반드시 `.ts` 쪽으로 빼야 테스트할 수 있다.**
- 기존 선례: `web/src/finding-bulk-state.ts:36-56` 의 `/^\d{4}-/` 검사와 주석, 공유 벡터 `internal/app/testdata/finding-bulk-due-date.json`(form.tz / form.due / wire / accepted 열). 같은 모양을 그대로 따를 것.

## 수용 기준

1. UTC 서쪽 시간대(예: `TZ=America/New_York`)에서 `datetime` 필드에 `9999-12-31T23:59` 을 넣고 저장하면, 요청이 나가기 **전에** 해당 필드 라벨이 들어간 한국어 오류가 던져진다(기존 `formBody` 의 다른 오류와 같은 `throw new Error(...)` 경로). 동 시간대에서 서버로 가는 문자열은 더 이상 `+`/`-` 로 시작하지 않는다.
2. 빈 값(`""` → `""`)과 평범한 유효 값(예: 아시아/서울의 `2026-12-31T23:59`)의 **wire 문자열이 한 글자도 바뀌지 않는다**. 즉 기존 정상 경로 회귀 없음.
3. 유효하지 않은 날짜 문자열(예: 서버 행에서 온 파싱 불가 문자열)이 들어와도 `RangeError: Invalid time value` 가 아니라 한국어 `Error` 가 난다.
4. 새 공유 벡터 JSON 하나를 Go 와 TS 양쪽에서 돌려 **같은 수락/거절 판정**을 단언한다. Go 쪽은 `validateFindingOpsResource` + `validateSchedule` 로(DSN 불필요), TS 쪽은 `process.env.TZ` 를 바꿔 가며 새 순수 헬퍼로 판정한다. 수정 전 Go 쪽은 벡터 전부 통과해야 하고(벡터가 서버 계약을 정확히 인코딩했다는 증거), TS 쪽만 실패해야 한다.
5. 인과 확정: 새 검사를 무력화하면 **그 테스트 하나만** 다시 실패한다. 그리고 검사 대상을 `values[f.key]`(입력 원문)로 바꾼 "그럴듯한 오답"으로도 같은 테스트가 실패해야 한다(= 검사한 값 = 보낸 값).

## 건드릴 파일 (프로덕션 2 / 테스트 3)

- `web/src/resource-form-state.ts` — **신규 export 순수 함수** 추가. 예: `export function datetimeFieldWire(label: string, value: string): string` — 빈 값은 `""` 반환, `new Date(value)` 가 유한하지 않으면 한국어 오류, `toISOString()` 결과가 `/^\d{4}-/` 가 아니면 한국어 오류, 아니면 그 문자열 반환. `finding-bulk-state.ts:42-52` 의 주석 수준으로 근거(ECMA-262 확장 연도 ↔ Go RFC3339 4자리)를 남길 것.
- `web/src/resources.tsx:1109-1111` `formBody` — `datetime` 분기를 위 헬퍼 호출로 교체(`out[f.key] = datetimeFieldWire(f.label, values[f.key])`). **그 외 분기(json·required 검사)와 `initialValues`(800-824행)는 손대지 말 것.**
- `internal/app/testdata/resource-datetime.json`(신규 공유 벡터) — `finding-bulk-due-date.json` 의 열 구조를 따르고 `note` 에 두 읽는 쪽과 `validateScope` 의 "미래" 추가 조건을 명시.
- `internal/app/resource_datetime_test.go`(신규) — 벡터를 `validateFindingOpsResource`(due_date)와 `validateSchedule`(next_run_at)에 먹여 `accepted` 열을 단언. `validateSchedule` 은 `interval_minutes`·`profile` 등 다른 필수 값을 함께 채워야 next_run_at 까지 도달한다(14-40행 확인).
- `web/tests/resource-form-state.test.mjs` — 기존 3개 테스트 보존하고 벡터 기반 테스트 1개 추가(`TZ` 변경 방식은 `web/tests/finding-bulk.test.mjs` 의 due date 테스트를 그대로 참고).

## 검증 명령 (이 저장소에서 실제로 도는 것)

```sh
npm --prefix web ci
npm --prefix web test            # 기준선 실측부터: v1.20.0 시점 106통과/0실패/0skip 로 추정(반드시 실측, 0건 실행을 통과로 읽지 말 것)
npm --prefix web run typecheck
npm --prefix web run build
npx --prefix web prettier --check web/src/resource-form-state.ts web/src/resources.tsx
go test -run '^TestResourceDatetime' -count=1 -v ./internal/app
go vet ./internal/app
gofmt -l internal/app/resource_datetime_test.go
git diff --check
```

Go 프로덕션 코드 무변경이면 `internal/webassets/dist` 재복사와 `go test -race ./...`(HUNTER_TEST_DSN 필요)는 생략 가능. `-run` 에 걸리는 testApp 기반 테스트가 SKIP 되면 **통과로 세지 말 것**.

## 위험과 피할 것

- `formBody` 는 **모든** 자원의 공통 제출 경로다. `datetime` 분기 **밖**은 한 줄도 바꾸지 말 것. `settings.tsx:66` 은 `FieldForm`/`initialValues` 만 import 하므로 `formBody` 변경이 설정 화면에 번지지 않는다(확인함) — `initialValues` 를 건드리면 그 보장이 깨진다.
- `validateScope` 는 형식 외에 "미래" 조건이 있다. 과거 연도 벡터를 "형식 거절"로 적으면 틀린 증거가 된다. 이번 회차는 `validateScope` 를 Go 단언에서 **빼고** 벡터 `note` 에만 차이를 기록하는 쪽이 안전하다.
- 서버 계약을 **느슨하게 만들지 말 것**. Go 쪽은 무변경이 목표다.
- `third_party/pentagi`, `.github/workflows`, `Dockerfile`, `scripts/release.sh`, auth/OIDC/MCP 경로는 건드리지 않는다.
- 과거 교훈: 회차 요약보다 main 을 믿을 것. 기준선 테스트 수는 과거 과제서 숫자를 베끼지 말고 **실측**할 것(09-28·10-01 에서 반복해 틀렸다).
- 테스트만 있는 PR 은 이 저장소에서 닫힌 이력이 있다(PR #15). 프로덕션 거절 수정과 함께 낼 것.
- **미확인**: 이 세션에서 `node`/`go` 를 실행하지 않았다(읽기 전용 정찰, 예산). 위 기준선 106 과 "9999-12-31T23:59 이 `datetime-local` 에 실제로 타이핑된다"는 브라우저 동작은 10-03 회차 기록에 근거한 **추정**이다. 구현자는 먼저 수정 전 상태에서 실패 테스트로 재현할 것.

## 차선 후보

**`datetime` 필드 저장 시 초·밀리초가 조용히 잘리는 것** — `resources.tsx:815-821` `initialValues` 가 `.toISOString().slice(0, 16)`(분 단위)로 로컬 값을 만들고, `formBody` 가 **사용자가 그 필드를 건드리지 않아도** 그 값을 다시 직렬화해 보낸다(PUT 본문은 전체 필드). 그래서 일괄 변경이 `…T12:34:56.789Z` 로 넣은 `due_date` 가 있는 발견 건의 제목만 고쳐 저장하면 `…T12:34:00.000Z` 로 바뀐다. 도달성은 1순위보다 높지만 올바른 수정이 "건드리지 않은 필드의 원본 보존" 설계 변경이라 위험·작업량이 크다(3/3/M). 1순위가 성립하지 않을 때만, 그리고 먼저 재현을 확정한 뒤 고를 것.
