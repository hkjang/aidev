# 과제서 — 2026-10-06-045755-hunter-improve (hunter, base main@398cd71 / VERSION 1.22.0)

- 과제: 진단 범위·실행 정책 폼의 숫자 상·하한을 서버가 실제로 수락하는 범위로 맞추고 공유 벡터로 두 쪽을 함께 고정 (가치 4 / 위험 2 / 작업량 M)

- 왜: 공통 자원 폼이 `scopes`·`policies` 의 숫자 7개에 선언한 `min`/`max` 가 서버 `validateScope`(internal/app/policy.go:79)·`validatePolicy`(policy.go:96)의 수락 범위와 **전부** 다르다 — 컨트롤은 `max_concurrency` 를 20 까지 올리게 해 주지만 서버는 1 만 받고, `max_requests` 는 1 부터 받는 것처럼 보이지만 서버 최소는 10 이다. Mantine `NumberInput` 은 선언된 `min`/`max` 로 blur 때 **클램프까지 하므로** 폼이 운영자를 서버가 거절하는 값으로 적극적으로 안내하고, 저장 버튼을 누른 뒤에야 `max_requests 값은 10~100 범위입니다` 같은 서버 문구를 본다. 게다가 서버는 `number()`(domain.go:115)로 읽어 **소수를 정수로 잘라 그 값을 그대로 덮어쓰므로**(policy.go:84·101 `m[b.k] = n`) 폼이 허용하는 `max_rps: 2.7` 은 조용히 2 로 저장된다.

## 확인한 사실 (실제로 열어 본 것)

폼 선언 — `web/src/resources.tsx`:

| kind | 필드 | 폼 선언 (min–max, 행) | 서버 수락 (정수) |
| --- | --- | --- | --- |
| scopes | `max_rps` | 0.1–100 (483-489) | 1–5 |
| scopes | `max_requests` | 1–1000 (490-496) | 10–100 |
| scopes | `timeout_seconds` | 1–600 (497-503) | 30–300 |
| policies | `max_rps` | 0.1–100 (352-358) | 1–5 |
| policies | `max_concurrency` | 1–20 (359-367) | 1–1 |
| policies | `max_requests` | 1–1000 (368-374) | 10–100 |
| policies | `timeout_seconds` | 1–600 (375-381) | 30–300 |

- 서버 진실의 단일 출처는 `internal/app/policy.go` 의 두 리터럴 테이블:
  - `validateScope` 79행: `{{"max_rps", 1, 5}, {"max_requests", 10, 100}, {"timeout_seconds", 30, 300}}` — 구조체는 `{k string; def, max int}`, 검사는 `n := number(m, b.k, b.def); if n < 1 || n > b.max` → **하한은 항상 1**, `def` 는 값이 없거나 문자열일 때의 폴백이다(최소값이 아님). 그래서 `max_requests: 5` 는 `5 < 1` 이 아니고 `5 > 100` 도 아니라 **통과한다**. 즉 폼의 하한 1 은 400 을 만들지 않는다 — 고쳐야 할 것은 **상한 3개와 소수 허용**이고, 하한은 `def` 와 맞추는 편이 운영 의도에 맞는다(아래 수용 기준 2 참고).
  - `validatePolicy` 96행: `{{"max_rps", 1, 5}, {"max_concurrency", 1, 1}, {"max_requests", 10, 100}, {"timeout_seconds", 30, 300}}` — 같은 계약.
- `number()`(`internal/app/domain.go:115-128)는 `float64`/`int`/`json.Number` 만 숫자로 읽고 그 외(문자열 포함)는 `fallback` 을 돌려준다. `float64` 는 `int(n)` 으로 **잘린다** → `2.7` → `2`, `0.5` → `0` → `0 < 1` → 400.
- 렌더러는 `resources.tsx:1019-1028`: `<NumberInput {...common} value={values[f.key]} min={f.min ?? 0} max={f.max} onChange={(v) => change(f.key, v)} />`. `step`·`allowDecimal`·`decimalScale` 는 **없다**(코드로 확인) → 소수 입력이 그대로 통과한다.
- 제출 경로는 `resourceSubmitBody`(`web/src/resource-form-state.ts:125`) → `resources.tsx` 의 `api(...)`. `resourceSubmitBody` 에는 **`number` 분기가 없다**(파일 전체 확인) — 즉 NumberInput 이 준 값이 그대로 wire 에 간다. Mantine `NumberInput` 의 `onChange` 는 `number | string` 이므로 입력을 비우면 `""` 가 간다 — 그때 `number()` 는 `def` 를 돌려주고 `m[b.k] = def` 로 조용히 기본값이 저장된다(이 부분은 코드 추론이며 브라우저 실측은 **미확인**).

## 수용 기준

1. `scopes`·`policies` 의 위 7개 필드 선언이 서버 테이블과 같은 상한을 쓴다: `max_rps` 5, `max_concurrency` 1, `max_requests` 100, `timeout_seconds` 300. 운영자가 컨트롤에서 올릴 수 있는 최대값이 곧 서버가 받는 최대값이다.
2. 하한도 서버의 `def` 와 맞춘다: `max_rps` 1, `max_concurrency` 1, `max_requests` 10, `timeout_seconds` 30. (서버 하한 자체는 1 이므로 이것은 거절을 막는 변경이 아니라 "운영 기준 밖 값을 권하지 않는다" 는 변경이다 — 커밋 메시지에서 두 근거를 구분해 적을 것. `max_rps` 의 `min: 0.1` 은 서버가 `int(0.x)=0` 으로 잘라 400 을 내므로 **이것만은 거절을 막는 변경**이다.)
3. 소수가 입력 단계에서 막힌다: 네 필드 모두 정수만 받는다(`allowDecimal={false}` 또는 `step={1}` + `decimalScale={0}` 중 Mantine 8 에서 실제로 동작하는 쪽을 실측해 고를 것). `2.7` 이 조용히 2 로 저장되는 경로가 사라진다.
4. 새 공유 벡터가 **서버 테이블이 진실**임을 Go 쪽에서 먼저 증명한다: 각 (kind, 필드)에 대해 상한값은 수락, 상한+1 은 거절, 소수(`x.5`)는 거절(또는 잘림 후 판정)을 `validateScope`/`validatePolicy` 로 단언한다. DSN 불필요.
5. 같은 벡터를 TS 쪽에서 돌려 폼 선언의 `min`/`max` 가 벡터와 **정확히 일치**함을 단언한다. 선언이 다시 벌어지면 이 테스트 하나만 실패한다.
6. 유효 입력의 wire 값은 한 글자도 바뀌지 않는다. 서버 계약(`policy.go` 의 두 테이블, `number()`)은 **손대지 않는다**.

## 건드릴 파일 (프로덕션 2 + 테스트 3)

1. `web/src/resource-form-state.ts` — 새 export `resourceNumberBounds`(또는 유사명): `{ scopes: { max_rps: {min:1,max:5,integer:true}, … }, policies: { … } }` 형태의 순수 리터럴 테이블. **`.tsx` 는 Node 타입 스트리핑으로 import 되지 않으므로**(프로필 확인됨) 테스트가 읽을 수 있는 곳은 이 `.ts` 모듈뿐이다. 10-04 회차가 `formBody` → `resourceSubmitBody` 로 옮긴 것과 같은 이유·같은 방식.
2. `web/src/resources.tsx` — `scopes`·`policies` 의 7개 `number` 필드 선언이 1의 테이블을 펼쳐 쓰게 바꾼다(`...resourceNumberBounds.policies.max_rps`). 렌더러 `1019-1028` 의 `NumberInput` 에 정수 잠금을 추가한다 — `Field` 타입(98행 근처 `type` union 과 `min`/`max` 선언부)에 `integer?: boolean` 또는 `step?: number` 를 더해 선언에서 읽고, 선언에 없는 기존 number 필드(findings `contribution_points`, schedules `interval_minutes`, scenarios 쪽)의 동작은 **바꾸지 않는다**.
3. `internal/app/testdata/resource-number-bounds.json`(신규, 테스트) — 기존 공유 벡터 4개(`csv-safety.json`, `finding-bulk-assignee.json`, `tracking-origins.json`, `finding-bulk-due-date.json`, `resource-datetime.json`)의 구조를 복제. 각 항목에 `kind`, `field`, `min`, `max`, `integer`, 그리고 Go 가 돌릴 `accepted`/`rejected` 샘플값을 적는다.
4. `internal/app/resource_number_bounds_test.go`(신규) — 3의 벡터를 읽어 `validateScope`/`validatePolicy` 에 돌린다. **주의**: `validateScope` 는 숫자 외에 `name`·`service_id`·`allowed_hosts`·`allowed_paths`·`expires_at`(미래여야 함, `expires.After(time.Now())`)도 요구한다. 숫자 사례마다 나머지를 유효하게 채운 최소 맵을 만들고, `expires_at` 은 `time.Now().Add(time.Hour)` 로 둘 것(과거 값은 형식과 무관한 이유로 거절되어 판정을 오염시킨다 — 10-04 회차가 같은 함정을 기록했다).
5. `web/tests/resource-number-bounds.test.mjs`(신규) — `readFileSync` 로 3의 JSON 을 읽고 `../src/resource-form-state.ts` 의 테이블과 `deepStrictEqual` 로 맞춘다. 기존 `web/tests/resource-form-state.test.mjs` 의 import 스타일을 그대로 따를 것(번들 없음, `node --test`).

## 검증 명령 (이 저장소에서 실제로 도는 것)

```sh
npm --prefix web ci
npm --prefix web test              # 기준선을 먼저 기록할 것 (v1.22.0 시점 110통과 로 기록됨 — 실측 필수)
npm --prefix web run typecheck
npm --prefix web run build
npm --prefix web exec -- prettier --check src/resources.tsx src/resource-form-state.ts
go test -run '^TestResourceNumberBounds' -count=1 -v ./internal/app    # DSN 불필요
go test -count=1 ./internal/app
go vet ./...
gofmt -l internal/app/resource_number_bounds_test.go                   # 빈 출력이어야 함
git diff --check
```

Go 프로덕션 코드를 바꾸지 않으므로 `internal/webassets/dist` 재복사와 `go test -race ./...`(HUNTER_TEST_DSN 필요)는 생략해도 된다. 프런트만 바꿀 경우 `go build ./cmd/hunter` 도 불필요하다 — 다만 `npm run build` 는 반드시 통과시킬 것.

**인과 확정(이 저장소의 관례)**: 새 테스트를 먼저 red 로 만들고(수정 전 폼 선언으로 5번 테스트가 실패), 고친 뒤 green 을 확인하고, 그 다음 상한 하나만 옛 값(예: `max_concurrency: 20`)으로 되돌려 **그 테스트 하나만** 다시 실패하는 것을 확인할 것.

## 위험과 피할 것

- **서버 범위를 넓히지 마세요.** `policy.go:79`·`96` 의 두 테이블은 진단 안전 통제다(AGENTS.md 6장·8장: "느슨하게 만들어 통과시키기 금지"). 이번 과제는 **폼을 서버에 맞추는** 것이고 반대 방향은 기각이다.
- 이미 저장된 행이 지금 범위 밖 값을 갖고 있을 수 있다(예: 과거에 `max_requests: 5` 로 저장된 scope — 서버 하한이 1 이라 통과했다). 폼 `min` 을 10 으로 올리면 **그 행을 열었을 때 Mantine 이 blur 에서 10 으로 클램프**해 운영자가 의도하지 않은 값 변경을 저장할 수 있다. 수용 기준 2 를 구현할 때 이 경로를 확인하고, 위험하다고 판단되면 **하한은 그대로 두고 상한·소수만 고치는 축소 범위**로 내려올 것(그래도 수용 기준 1·3·4·5 는 전부 달성된다). 이 판단을 커밋 메시지에 남길 것.
- `number` 필드의 렌더러는 **모든 kind 가 공유**한다(`resources.tsx:1019`). 정수 잠금을 선언에서 읽지 않고 렌더러에 무조건 넣으면 `contribution_points`·`interval_minutes` 등 손대지 않기로 한 필드까지 바뀐다.
- `settings.tsx:66` 이 `resources.tsx` 에서 `FieldForm`·`initialValues` **둘만** import 한다. `Field` 타입에 선택 속성을 더하는 것은 안전하지만 `FieldForm` 의 기존 시그니처·렌더 결과를 바꾸지 말 것(설정 화면 전체가 같은 렌더러를 쓴다). `settings.tsx:514` 의 save 는 `resourceSubmitBody` 를 **쓰지 않으므로** 이번 변경이 설정 제출 경로에 영향을 주지 않는다(코드로 확인).
- 보호 경로(auth/OIDC/migrations/`.github/workflows`/`third_party/pentagi`/`scripts/release.sh`)는 건드리지 않는다. 이번 과제는 그 어느 것에도 닿지 않는다.
- 테스트만 있는 PR 은 이 저장소에서 닫힌 이력이 있다(PR #15). 프로덕션 수정(1·2)과 반드시 함께 낼 것.

## 차선 후보

**web 테스트 러너의 Node 하한을 `engines`(>=22.18)로 선언** (가치 3 / 위험 1 / 작업량 S). `web/package.json` 에 `engines.node` 가 없어 툴체인 드리프트가 조용한 0건 실행으로 나타난 이력이 있다(2026-10-02 회차). CI `node-version: '26'` 와 Dockerfile `node:26` digest 가 이미 있으니 **단일 출처를 어디로 둘지 먼저 정하고**, `engine-strict` 는 켜지 말 것(기여자 환경을 막는다). 릴리즈 경로 인접이라 1순위가 성립하지 않을 때만.

3순위: `list-export.ts:39` 의 `new Date(value).toISOString()` 이 `|value| > 8.64e15` 에서 `RangeError` 를 던져 CSV 내보내기 전체를 깨뜨리는 것 방어 (가치 2 / 위험 1 / 작업량 S). 도달 가능성이 낮아 가치가 낮다.
