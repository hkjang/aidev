# 과제서 (2026-09-28, run 2026-09-28-211204-umm-improve)

- 과제: 되감은 채 다른 공간으로 옮기면 그 공간이 거짓 날짜를 달고 얼어붙는다 (가치 4 / 위험 1 / 작업량 S)

- 왜: `web/src/pages/CanvasPage.tsx:512-519` 의 `params.spaceId` effect 는 공간이 바뀔 때 `setActiveSpace` 와 `setViewportKnown(false)` 만 하고 `rewind` 상태를 비우지 않습니다. 그래서 되감기를 켠 채 공간 메뉴(`openSpace`, 같은 파일 1761행 — 클라이언트 라우팅이라 React 상태가 살아 있음)로 다른 공간을 열면, `activeSpace` effect(675행)가 새 공간의 **현재** 노트를 불러오는데 `rewind` 는 **옛 공간의** 스냅샷 메타데이터(`at`·`removedEdges`·`earliest`)를 그대로 들고 있습니다. 결과는 네 가지가 동시에 틀립니다 — ① 배너(2900-2905행)가 현재 데이터를 보여 주면서 "〈옛 시각〉의 공간입니다" 라고 거짓을 말함, ② `readOnly`(323행 `rewind !== undefined`)가 켜져 새 공간을 **편집할 수 없음**(이유는 그 거짓 배너뿐), ③ 이벤트 스트림이 `rewindRef.current` 로 막혀(692행) 협업자의 변경이 영영 안 들어옴, ④ 자리 복원이 건너뛰어짐(1079행). 빠져나오는 길은 "지금으로" 메뉴 하나뿐이고, 사용자는 자기가 아직 되감겨 있다는 것을 배너의 옛 시각으로만 추측해야 합니다. 고치면 공간 전환이 언제나 현재의 편집 가능한 공간을 엽니다.

- 수용 기준:
  1) 되감기를 켠 상태에서 공간 메뉴로 다른 공간을 열면 되감기 배너(`…의 공간입니다`)가 사라지고, 되감기 단추가 눌린 상태(`aria-pressed`)가 아니며, 새 공간의 생각을 편집할 수 있다(읽기 전용 아님).
  2) 같은 공간에 머무르는 동안에는 되감기가 유지된다 — 기존 `web/e2e/space-rewind.spec.ts` 의 단언이 그대로 통과해야 한다(되감기를 아무 렌더에서나 지워 버리는 해법은 이 시험이 잡아냄).
  3) 새 시험이 **고치기 전 코드에서 실패**하는 것을 확인하고 그 사실을 적을 것. 시험은 실제 바이너리 + 실제 PostgreSQL 를 지나는 Playwright e2e 로 쓸 것 — `CanvasPage` 에는 단위 시험 파일이 없고(확인: `web/src` 의 `*.test.*` 17개에 CanvasPage 없음) 손으로 만든 대역으로는 이 배선 결함을 증명할 수 없다.

- 건드릴 파일 (프로덕션 2개 + 시험 1개):
  - `web/src/pages/CanvasPage.tsx:512-519` — `params.spaceId` effect 의 `if (params.spaceId !== activeSpace) { … }` 블록 안에 `setRewind(undefined)` 를 더한다. 이미 있는 주석("A different space opens fresh…")과 같은 뜻이므로 그 주석을 되감기까지 포함하도록 한 줄 늘릴 것. 추가 재조회는 필요 없다 — 675행 effect 가 새 `activeSpace` 로 `loadCanvas()`·`loadAttachments()`·`loadBranches()` 를 이미 다시 돈다.
  - `web/e2e/space-rewind.spec.ts` — 새 test 1개 추가. 기존 파일의 방식을 그대로 쓸 것: `signIn(page)`, `unique('…')`, `page.evaluate` 안에서 `fetch('/api/v1/spaces', 'POST')` 로 공간 두 개와 각각의 노트를 만들고, 첫 공간을 `page.goto('/space/<A>')` 로 연 뒤 `getByRole('button', {name:'되감기'})` → `getByRole('menuitem', {name:'하루 전'})` 로 되감고 `getByText(/의 공간입니다$/)` 가 보이는 것을 확인, 그 다음 **반드시 클라이언트 라우팅으로** 공간 B 로 옮길 것(공간 메뉴: 2275행 `Menu.Label`("공간 {count}개")이 있는 메뉴를 열고 B 의 이름을 클릭). `page.goto` 는 전체 새로고침이라 React 상태가 초기화되어 **결함이 재현되지 않는다** — 이 함정이 이 과제의 핵심이다.
  - (선택, 문구를 더한다면) `web/src/i18n/*` — 새 문자열을 쓰지 않는 것이 낫다. 이 과제는 상태 해제만으로 끝난다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd web && npx tsc --noEmit`
  - `cd web && npx vitest run` (기존 182개 전후, 회귀 없음 확인)
  - `cd web && npx oxlint && npx prettier --check src e2e`
  - e2e: 실제 바이너리 + PostgreSQL 17 도커가 필요하다. 과거 회차가 쓴 방식 — 도커 `umm-e2e-pg` 를 띄우고 `go build ./cmd/...` 한 바이너리를 올린 뒤 `cd web && npx playwright test e2e/space-rewind.spec.ts`. (`web/package.json` 의 `"e2e": "playwright test"`; Playwright webServer 설정은 **미확인** — `web/playwright.config.ts` 를 먼저 읽고 그 설정이 요구하는 준비를 따를 것.)
  - Go 코드는 건드리지 않으므로 `go test` 는 필요 없다. 혹시 건드렸다면 `go test -p 1 ./...` 를 실제 PG 에 대해 돌릴 것.

- 위험과 피할 것:
  - **`setRewind(undefined)` 를 effect 밖이나 렌더 본문에 두지 말 것** — 되감기 자체가 즉시 해제되어 기능이 죽고, 기존 `space-rewind.spec.ts` 가 잡는다. 해제는 반드시 `params.spaceId !== activeSpace` 가드 **안**에서만.
  - `rewindTo` (566행)·`rewindRef` 동기화 effect(312-314행)·`readOnly` memo(322-324행)는 건드리지 말 것. `rewindRef` 는 effect 로 따라오므로 별도 갱신이 필요 없다.
  - 보호 경로(`internal/auth`, `migrations/`, `.github/workflows/`)는 이 과제와 무관하다 — 열지 말 것. 릴리즈·버전은 올리지 말 것(릴리즈는 별도 세션이며, 과거에 릴리즈 경로를 건드린 머지가 되돌림으로 이어졌다).
  - 과거 교훈: `page.goto` 로 "공간 전환"을 흉내내면 시험이 초록인데 결함이 남는다(위 시험 항목). 그리고 e2e 는 느린 CI 에서 타이밍에 약하다 — 노트는 화면 조작이 아니라 API 로 먼저 만들고(2026-09-13 회차가 `canvas.spec.ts` 를 그렇게 고쳤다) 단언은 `toBeVisible`/`toHaveCount(0)` 로 기다리게 할 것.
  - 이 고침은 `web/src/pages/CanvasPage.tsx` 한 파일이다. 같은 보류 목록에 있는 "공백 한 칸 발표 제목"(`internal/presentation/service.go:336` 부근 — **미확인**, `title := req.Title` 한 줄만 봤음)은 **이번 회차에 넣지 말 것**. 별 과제다.

- 차선 후보: **`exportOutline` 의 내려받기 이름에 공간 이름을 담기** (가치 2 / 위험 1 / 작업량 S) — `internal/httpapi/export_handlers.go:101` 이 언제나 리터럴 `attachment; filename="umm-outline.md"` 를 쓰므로 여러 공간의 개요를 받으면 구분할 수 없다. 같은 패키지에 이미 쓸 도구가 다 있다: `internal/httpapi/content_disposition.go` 의 `dispositionSafe`/`maxDispositionStemBytes` 와 `handoff_handlers.go:159` 의 `handoffFilename(spaceName)` (`oneLine` 까지 지나감). `issueHandoffClaim` 이 공간 이름을 한 번 더 조회해 이름을 만드는 방식을 그대로 따르면 되고, 시험은 `handoff_handlers_test.go:83` 의 표 시험과 같은 모양으로 쓸 수 있다. 확장자만 `.md` 로 맞출 것.
