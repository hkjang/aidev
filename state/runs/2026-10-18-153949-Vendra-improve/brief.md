# 과제서 — 2026-10-18 (base: main@dfbe947, v0.7.73)

- 과제: 저장된 보기 조회가 실패하면 목록을 비우지 말고 **서버가 말한 이유**를 말하고 다시 시도할 수 있게 하기 (가치 3 / 위험 2 / 작업량 S)

- 왜: `web/src/pages/Objects.tsx` 의 `ObjectList.loadViews`(:266-275)가 거부를 `.catch(() => setViews([]))` 로 삼켜, 서버가 **문장으로 설명한** 실패(`productivity.go:463`·`:475` 의 `500 database_error "저장된 보기를 조회하지 못했습니다"`, 세션 만료 401 등)가 「저장된 보기가 하나도 없다」와 화면상 **글자 하나 다르지 않게** 보인다. 더 나쁜 것은 `saveView`(:301-319)·`removeView`(:320-330)가 성공한 뒤 `await loadViews()` 를 부르는 것이다 — POST 는 201 로 저장됐는데 뒤따르는 GET 이 실패하면 `setViews([])` 가 방금 저장한 보기를 지우고 모달은 이미 닫혔으며 `viewError` 는 비어 있어, 사용자는 **아무 메시지 없이** 드롭다운에 「기본 보기」만 남은 것을 보고 저장이 안 된 줄 알고 같은 보기를 다시 저장한다(서버에는 중복이 쌓인다). 같은 파일의 목록 조회는 이미 28572eb 가 이 거짓을 고쳤다(:235-256 의 `.catch` 와 :437-453 의 배너·재시도) — 바로 옆 요청만 아직 옛 모양이다.

- 수용 기준:
  1) `GET /api/v1/me/saved-views` 가 거부되면 **서버가 보낸 `error.message` 가 그대로** 화면에 뜬다(메시지가 없으면 대체 문구). 지금은 아무 것도 뜨지 않는다.
  2) 조회가 실패해도 **이미 떠 있던 저장된 보기 목록이 지워지지 않는다** — 마지막으로 서버가 실제로 말한 것이 남는다. 특히 「보기 저장 성공 → 새로고침 GET 실패」에서 방금 저장한 보기가 화면에서 사라지지 않는다.
  3) 실패 안내에 전용 「다시 시도」가 있고, 그것이 `window.location.reload()` 가 아니라 `loadViews()` 를 **다시 호출**한다(재호출이 성공하면 보기 목록이 채워진다).
  4) 재조회가 성공하면 그 안내가 사라진다.
  5) 기존 테스트 파일(`objects-error.test.tsx`, `objects-order.test.tsx`, `objects-draft-submit.test.tsx`, `object-status.test.tsx`)은 **한 글자도 고치지 않는다** — 대조군이다. 특히 `objects-error.test.tsx:24-25` 주석이 "saved-views 는 자체 `.catch` 를 들고 있어 여기서는 정상 응답한다" 고 적고 있으므로, 그 파일의 두 헬퍼는 saved-views 를 성공으로 답해 계속 통과해야 한다.

- 건드릴 파일 (프로덕션 1파일 + 신규 테스트 1파일):
  - `web/src/pages/Objects.tsx` — `ObjectList` 안에서만:
    - `loadViews`(:266-275): `.catch(() => setViews([]))` 를 **`setViews` 를 부르지 않는** catch 로 바꾸고 `cause instanceof Error && cause.message ? cause.message : "저장된 보기를 조회하지 못했습니다"` 를 새 실패 state 에 담는다. `.then` 에서는 그 state 를 비운다(성공이 실패를 대체한다 — :230-233 이 `load` 에서 쓴 함수형 갱신 패턴을 그대로 베끼면 성공 경로의 렌더가 늘지 않는다).
    - 새 state 를 **`viewError`(:212) 와 따로** 둘 것(예: `const [viewsFailure, setViewsFailure] = useState("")`). `viewError` 는 저장·삭제 실패의 자리이고 `saveView` 가 시작할 때마다 `setViewError("")` 로 비우며(:303) `!viewModal` 일 때만 렌더된다(:432) — 거기에 조회 실패를 섞으면 모달이 열린 동안 조회 실패가 숨고 저장 시도마다 지워진다.
    - 배너: `.saved-view-tools`(:406-430) 바로 아래, `viewError` 배너(:432)와 **같은 모양**(`<div className="form-error"><AlertCircle />…</div>`)으로 한 줄 + 「다시 시도」 버튼(`onClick={() => void loadViews()}`). 목록 실패 배너(:437-442)가 쓰는 `role="status"` 를 따라도 좋다. 전용 className 을 하나 붙여 테스트가 집어낼 수 있게 할 것(목록 쪽 `list-load-failed` 와 같은 결).
  - 신규 `web/src/pages/objects-saved-views.test.tsx` — **`web/src/pages/objects-draft-submit.test.tsx:22-49` 의 `network()` 하네스를 베껴 `fetch` 만 `vi.stubGlobal` 로 대체**하고 Objects·React 이벤트·실제 `api`/`post`/`del` 을 통과시킬 것. 이유: 기준 2 의 핵심 시나리오가 `post("/api/v1/me/saved-views", …)` 성공 뒤의 GET 실패인데, `post` 는 `api.ts` 모듈 **안쪽**의 `api` 를 호출하므로 `vi.mock("../api", { api })` 로는 쓰기 경로가 대체되지 않는다(`objects-error.test.tsx` 의 `api` mock 방식은 이 시나리오에 쓰지 말 것). 저장 모달 열기는 `보기 저장` 버튼(:419-421), 삭제는 `aria-label="현재 보기 삭제"`(:426), 드롭다운은 `aria-label="저장된 보기"`(:410).

- 검증 명령 (이 워크트리는 리눅스 파일시스템이라 Vitest 가 돈다 — `/mnt/c` 사본은 쓰지 말 것):
  ```
  npm ci --ignore-scripts --prefix web          # 이 워크트리에 web/node_modules 없음
  npm test --prefix web -- src/pages/objects-saved-views.test.tsx   # 고치기 전에 먼저 돌려 빨강 확인
  npm test --prefix web                          # 전체
  cd web && npx tsc -b --noEmit && npx eslint src --max-warnings 0 && npm run build
  ```
  고친 뒤 인과 확인: 새 catch 안의 `setViewsFailure` 한 줄만 중성화(파일은 계속 컴파일되게)하고 새 테스트만 빨강·기존 파일은 초록인지 볼 것. 파일 통째 되돌리기는 빌드 에러가 되어 아무것도 증명하지 못한다. 커밋 전 `gate.py secrets` 를 diff 에 돌릴 것.

- 위험과 피할 것:
  - **`NewObject`(≈:721- 이하 전체)는 손대지 말 것.** 011c5a3(중복 제출 가드)이 아직 HEAD 조상이 아님을 이번에 확인했다 — 같은 함수를 건드리면 충돌한다. 이 과제는 `ObjectList` 안에서 끝난다.
  - Go·마이그레이션·권한·`.github/workflows` 무변경. `productivity.go` 는 **읽기만** 했고 고칠 것이 없다.
  - `setCanShareViews` 는 실패 시 그대로 두는 것이 맞다(마지막으로 서버가 말한 값). 새로 `false` 로 떨어뜨리지 말 것 — 보기 저장 모달의 공유 체크박스가 이유 없이 사라진다.
  - 안내 문구를 `<Empty title>`·목록 실패 배너와 **같은 문장으로 쓰지 말 것** — 2026-10-08 회차가 같은 문장을 두 번 찍어 `Found multiple elements with the text` 로 터졌다.
  - 로그인 잠금 Go 테스트는 플래키하다(이 과제는 Go 를 돌릴 필요가 없다).

- 차선 후보: **공급업체 드롭다운 조회 실패를 `.catch(() => {})` 로 삼키는 것**(Objects.tsx:261-265) — 드롭다운이 비어 새 레코드를 만들 수 없는데 화면이 이유를 말하지 않는다. 1순위와 **같은 파일·같은 컴포넌트**이므로 **둘 중 하나만** 하고 한 회차에 묶지 말 것(파일 수가 아니라 리뷰 면적이 커진다).
