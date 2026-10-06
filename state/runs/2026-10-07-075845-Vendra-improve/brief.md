# 과제서 — 2026-10-07 (base main@9f50392)

- 과제: 업무 목록 화면이 **서버가 실제로 적용한 정렬**을 읽고 사용자에게 말하게 하기 — 지금 드롭다운은 떨어진 `amount_desc` 를 계속 적용된 것처럼 보여 준다 (가치 3 / 위험 2 / 작업량 S)

- 왜: 지난 회차(fe3f618)가 API 를 고쳐 목록 응답에 **실제 적용된** 정렬을 담게 했다 — `objects.go:93` 이 `applied := objectOrderApplied(order, hasPermission(p, objectType+".amount.read"))` 를 한 번 계산해 `:118` 의 `writeJSON(w, 200, map[string]any{"items":…, "count":…, "limit":…, "truncated":…, "order": applied})` 로 내보낸다. 웹은 그 키를 **읽지 않는다**: `ObjectList.load`(`web/src/pages/Objects.tsx:196-207`)의 응답 타입이 `{ items: BusinessObject[]; truncated?: boolean }` 이고 `setResult` 에 그 둘만 담으며, 정렬 `<select>`(`:329-334`)의 `value` 는 URL 파라미터(`:176` `params.get("order") || "updated_desc"`)다. 그래서 `<type>.amount.read` 가 없는 사용자는 「금액 높은순」이 선택된 드롭다운 아래에 최신순 행을 받고(금액 칸은 `redactObject` 가 떨어뜨려 비어 있다) 맨 위 행을 「가장 큰 계약」으로 읽는다. 고치면 응답이 이미 들고 있는 사실이 화면에 도달한다 — 새 API·새 권한·새 쿼리 없이.

- 수용 기준:
  1) URL 이 `?order=amount_desc` 인데 응답이 `"order":"updated_desc"` 로 오면 화면에 그 사실을 말하는 안내 한 줄이 나오고, 정렬 `<select>` 는 **적용된** 값(최근 수정순)을 보여 준다.
  2) 요청한 정렬과 응답의 `order` 가 같으면 안내는 **나오지 않는다**(기존 화면과 한 픽셀도 다르지 않음).
  3) 응답에 `order` 키가 아예 없으면(구버전 응답) 아무 일도 일어나지 않고 드롭다운은 지금처럼 URL 값을 보여 준다 — 즉 `order` 는 선택적으로 읽는다.
  4) 로딩 중(응답 전)에는 안내를 내지 않는다 — `loading` 은 `result.key !== requestKey`(`:282`)로 이미 판정되어 있으니 그것을 쓸 것.
  5) 새 vitest 테스트가 ①의 안내 문구를 **문구로** 확인하고, ②·③을 각각 확인한다(세 케이스).

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `web/src/pages/Objects.tsx`
    - `ObjectList.load`(`:196`) — `api<{ items: BusinessObject[]; truncated?: boolean; order?: string }>` 로 넓히고, `:178-182` 의 `result` state 타입/초기값에 `order?: string` 추가해 `setResult` 에 `response.order` 를 담는다.
    - 렌더 구역(`:282-303`) — `loading` 아래에서 `const appliedOrder = !loading && result.order ? result.order : order;` 를 만들고, `appliedOrder !== order` 일 때 `truncated` 배너와 **같은 모양**으로 한 줄(`<p className="form-error warning" role="status"><AlertCircle />…</p>` — `:298-303` 이 본보기, `AlertCircle` 은 이미 import 되어 있다). 문구는 한국어로 「요청한 정렬을 적용할 수 없어 <적용된 정렬 이름>으로 표시했습니다」 류 + 금액 정렬인 경우 권한 때문임을 암시하되 권한 이름을 노출하지 말 것.
    - `<select value={order}>`(`:329`) → `value={appliedOrder}`. **네 값이 `<option value>` 네 개와 글자까지 일치하므로 변환 표가 필요 없다**: `updated_desc`/`due_asc`/`amount_desc`/`title_asc` (`:330-333`), 기본값도 `updated_desc`(`:176`). `onChange` 의 `set("order", …)` 는 그대로 — URL 이 요청의 출처라는 것은 바꾸지 않는다.
  - `web/src/pages/objects-order.test.tsx`(신규) — 하네스는 `web/src/pages/work-inbox.test.tsx:1-19` 를 그대로 베낄 것:
    `vi.mock("../api", async (importOriginal) => ({ ...(await importOriginal<typeof import("../api")>()), api: vi.fn() }))` + `const call = vi.mocked(api)` + `render(<MemoryRouter initialEntries={["/contracts?order=amount_desc"]}><Objects type="contract" /></MemoryRouter>)` (`Objects` 는 `./Objects` 의 default export `:163`, `type="contract"` 면 `ObjectList` 로 간다 — `ObjectList` 는 export 되어 있지 않으니 default 를 쓸 것).
    **mock 은 URL 로 분기해야 한다** — `ObjectList` 가 `api` 를 세 번 부른다: ① `${c.endpoint}?q=&status=&order=…`(계약은 `/api/v1/contracts`), ② `/api/v1/suppliers?limit=500`(`:212`), ③ `/api/v1/me/saved-views?context=object:contract`(`:217-219`, `{items, canShare}` 를 기대). ②③은 `{items:[],canShare:false}` 로 답하면 된다(둘 다 `.catch` 가 있어 실패해도 조용하지만, 거부로 답하면 unhandled rejection 이 다른 테스트에 붙을 수 있다 — 반드시 resolve 할 것).

- 검증 명령 (이 워크트리에 `web/node_modules` 가 **없다** — 설치가 먼저):
  ```
  cd web && npm ci --ignore-scripts
  npm test                       # node scripts/run-vitest.mjs run — 기준선 22 files / 98 tests
  npx tsc -b --noEmit
  npx eslint src --max-warnings 0
  npm run build
  ```
  고치기 전에 새 테스트를 **먼저 돌려 실패를 실제 출력으로** 남길 것. Go 쪽은 한 글자도 바뀌지 않으므로 회귀 확인용으로만: `gofmt -l internal cmd`(무출력), `go vet ./internal/... ./cmd/...`.

- 위험과 피할 것:
  - **Go·권한·SQL 무변경.** `objectOrderApplied`/`objectOrderBy`(`objects.go:122-162`)는 지난 회차가 「적용된 정렬의 유일한 출처」로 정리하고 주석으로 못 박았다 — 다시 판단하는 코드를 웹에 심지 말 것(권한을 웹에서 추측해 `amount_desc` 를 미리 숨기는 식). 웹은 **응답이 말한 것만** 믿는다.
  - 잘못된 `order` 를 400 으로 거절하지 말 것 — `objects.go:124-130` 주석이 「웹이 URL 값을 그대로 되보내므로 거절하면 컨트롤 하나를 바로잡는 대신 화면이 통째로 빈다」고 이미 결정했다.
  - `<select value={appliedOrder}>` 로 바꾸면, 권한 없는 사용자가 「금액 높은순」을 고르면 URL 은 `amount_desc` 가 되고 드롭다운은 다시 「최근 수정순」으로 **되돌아오며** 안내가 뜬다. 이것이 의도한 동작이다(거짓말 대신 거절을 보여 준다) — 되돌아옴을 막으려고 URL 을 서버 값으로 덮어쓰지 말 것(무한 루프와 뒤로가기 깨짐).
  - `vitest.config` 가 `restoreMocks` 를 켜 두었다 — `work-inbox.test.tsx:29-31` 주석대로 `beforeEach` 에서 mock 을 reset 하지 말 것(교훈 「beforeEach that returns a mock」과 같은 자리).
  - 기존 테스트 파일은 고치지 말고 **추가만**. `object-status.test.tsx` 는 `ObjectTable` 만 직접 렌더하므로 이 변경에 닿지 않는다(대조군).
  - `web/node_modules` 설치는 몇 분 걸린다 — 먼저 시작해 둘 것. 이 워크트리는 Linux 파일시스템(`/home/hkjang/.cache/...`)이라 vitest 가 돈다(`/mnt/c` 였다면 안 됐다).

- 차선 후보: **목록 조회가 실패하면 `<Loading />` 이 영원히 돌고 이유도 재시도 버튼도 없는 것** (가치 3 / 위험 2 / S). `ObjectList.load`(`Objects.tsx:196-207`)에 `.catch` 가 없어 `api` 거부가 unhandled rejection 이 되고 `result.key` 가 `requestKey` 와 영원히 어긋나 `:282` 의 `loading` 이 true 로 고정된다 — `:359` 가 스피너만 그린다. 이 저장소가 `WorkInbox` 에서 이미 같은 계열을 고쳤고(`work-inbox.test.tsx` 가 그 증인) 실패 상태 + 재시도 버튼의 본보기가 거기 있다. 1순위와 **같은 함수**를 건드리므로 한 회차에 둘을 섞지 말 것.
