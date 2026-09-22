# 과제서 — 2026-09-22

- 과제: 미래 시각으로 들어온 교류 기록을 거부하고, 교류 입력 검증을 이름 있는 순수 함수로 묶기 (가치 3 / 위험 2 / 작업량 S)

- 왜: `POST /api/v1/people/{personID}/interactions`(`internal/server/data.go:402 createInteraction`)는 `occurred_at` 에 상한이 없어 오타로 들어온 2225년 값이 그대로 저장된다. 그러면 같은 한 건을 경로마다 다르게 읽는다 — `relationshipMetrics`(data.go:459)는 `days<0` 을 `math.Max(0,days)` 로 0으로 눌러 오늘 만난 것처럼 만점 가중치를 주고, `recalculateRelationship`(data.go:478,503)은 `max(occurred_at)` 을 그대로 써서 `last_interaction_at` 을 미래로 박아 궤도 문법이 그 관계를 영원히 "방금 연락함"으로 읽으며, Time Travel 의 과거 화면은 `occurred_at<=at`(timetravel.go:63,93)이라 같은 기록을 아예 보지 못한다. 막으면 잘못된 한 번의 입력이 친밀도·흐름·마지막 접촉을 영구히 왜곡하는 길이 닫힌다.

- 수용 기준:
  1) `occurred_at` 이 허용 오차(권장 5분, 시계 어긋남 몫)를 넘어 미래이면 `400 validation_error` 와 한국어 메시지를 돌려주고 INSERT·`recalculateRelationship`·감사 로그 어느 것도 일어나지 않는다.
  2) 지금 통과하던 입력은 그대로 통과한다 — 과거 시각, 생략(zero → `time.Now()`), 허용 오차 안의 살짝 미래. `kind` 화이트리스트와 `weight` 보정(`<=0 || >10 → 1`)의 동작은 바뀌지 않는다.
  3) 테스트가 (a) **실제 핸들러** `(&Server{}).createInteraction` 을 `store` 가 nil 인 서버로 불러(= throttle_test 와 같은 수법. DB 에 닿으면 nil 포인터로 죽으므로 "검증이 DB 앞에서 끝난다"가 구조적으로 증명된다) 미래 값에 400·`validation_error`·비어 있지 않은 메시지가 나옴을 확인하고, (b) 순수 검증 함수 단위로 과거·zero·오차 안 미래가 통과하고 zero 가 현재로 채워짐을 확인한다. 통과 경로를 nil store 핸들러로 부르지 말 것(패닉이 난다 — 그건 증거가 아니다).

- 건드릴 파일:
  - `internal/server/data.go:402 createInteraction` — 지금 인라인으로 흩어진 검증(kind·zero 보정·weight 보정)을 바로 위에 있는 `validatePersonInput`(data.go:230) 관례대로 `validateInteractionInput(in *interactionInput, now time.Time) error` 로 빼고, 여기에 미래 상한을 더한다. 핸들러 안 익명 구조체(data.go:406~411)를 파일 수준의 `interactionInput` 타입으로 올린다(`personInput` 과 같은 자리). 검증 호출은 지금처럼 `decodeJSON` 직후, 사람 존재 확인(data.go:425)·`activeDataKey` 앞에 둔다.
  - 허용 오차는 `const interactionFutureSkew = 5 * time.Minute` 처럼 이름 있는 상수로. `now` 를 인자로 받아야 테스트가 시계를 고정할 수 있다.
  - `internal/server/data_test.go` (신규) — 위 3)의 테스트. 핸들러 테스트는 `httptest.NewRequest` + `context.WithValue(ctx, userContextKey, User{ID: ...})`(auth.go:24,32) 로 사용자를 넣는다. chi 라우트 컨텍스트가 없어 `chi.URLParam` 이 빈 문자열이지만 검증이 그 앞에서 끝나므로 상관없다. 응답 본문은 server 패키지의 `apiError` 타입으로 디코드한다(throttle_test.go:110 과 같은 방식).
  - (선택) `web/src/pages/PersonPage.tsx:InteractionDialog`(422~) — `type="datetime-local"` 입력에 `max` 를 두어 왕복 전에 막기. 서버 오류는 이미 같은 다이얼로그의 `Alert` 로 보이므로 필수는 아니다. 넣으면 `npm run build` 까지 확인하고, 이것 때문에 필수 범위를 미루지 말 것.
  - (선택) `docs/API.md` 에 교류 기록의 `occurred_at` 제약 한 줄. 확인함: `docs/*.md` 에 `occurred_at` 을 설명한 자리는 현재 없다.

- 검증 명령 (이 저장소에서 실제로 도는 것, 작업 트리 루트에서):
  - `gofmt -l internal/server` — 출력이 없어야 한다
  - `go vet ./...`
  - `go test -race -count=1 -v ./internal/server -run 'Interaction'`
  - `go test -race ./...` — 이번 정찰에서 초록 확인(캐시). DB 테스트(`timetravel_db_test.go`)는 `ORBIT_TEST_DATABASE_URL` 이 없으면 skip 된다
  - (선택 프런트를 건드렸을 때만) `cd web && npm ci --no-audit --no-fund && npm run build` — node_modules 가 없어 몇 분 걸린다. `npm run lint` 는 eslint 가 설치돼 있지 않아 실패한다(기존 문제, 이번 과제 아님)

- 추정: 25~45분(10번 중 8번). 근거 — 검증 함수 추출·상한 추가 15줄 남짓, 새 테스트 60줄 남짓, 실행 경로는 하나. 늘어나는 경우: 선택 항목(프런트 `max`)을 포함하면 `npm ci` 로 +10분. 포함하지 않는 것: 다른 핸들러의 검증 정리, weight 정책 변경, 프런트 리팩터.

- 위험과 피할 것:
  - `weight` 의 묵음 보정(`<=0 || >10 → 1`)은 이번에 바꾸지 말 것. 계약 변경이고 이번 근거와 무관하다.
  - 과거 쪽 하한은 두지 말 것 — 1970년대의 첫 만남을 적는 것은 정당한 사용이다.
  - `parseOrbitAt`(timetravel.go:20)의 미래 거부는 별건(보류 아이디어에 있음). 같이 손대면 과제가 둘이 되고 계약도 둘이 바뀐다.
  - 교류를 INSERT 하는 자리는 `data.go:441` 하나뿐임을 grep 으로 확인했다(`mcp.go`·`ai.go`·`workflow.go`·`export.go` 에는 없다 — MCP 도구는 읽기와 기억 쪽뿐). 새 입력 경로를 만들지 말 것.
  - 보호 경로(`internal/server/auth.go`·`internal/store/migrations`·`.github/workflows`)는 이 과제에 필요 없다. 마이그레이션을 새로 만들 이유도 없다(스키마 변경 없음).
  - 최근 두 회차(2026-09-21)가 OrbitPage·Time Travel 프런트 과제에서 no-change 로 끝났다. 이번 필수 범위는 Go 서버 쪽에 두고, 프런트는 선택으로만 둔다.
  - 미확인: 실제 postgres 로 "미래 기록이 지표를 망가뜨린다"를 재현해 보지는 않았다(이 세션에 DB 없음). 근거는 위에 적은 코드 경로 읽기다. 구현자가 DB 를 띄울 수 있으면 고치기 전 상태에서 한 번 재현해 두면 더 좋다.

- 차선 후보: `orbitAt` 의 포함 규칙(`p.created_at<=at` / `first_met<=at` / 그 이전 교류)과 기억 수 시점 필터를 `internal/server/timetravel_db_test.go` 의 `openTestStore`·`seedUser`·`seedPerson`·`seedInteraction` 을 늘려 실제 postgres 로 고정하기 (가치 3 / 위험 1 / M). 주의: `orbitAt` 은 `JOIN relationships` 라 사람만 시드하면 한 명도 안 나온다 — `relationships` 행 시드 헬퍼가 먼저 필요하고, `memories` 도 시드해야 기억 수 필터를 볼 수 있다.
