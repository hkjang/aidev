# 과제서 — releasedock 2026-09-20-162406 (main@ed71528, v0.5.16)

- 과제: 진행 중 실행의 SSE 로그 중복 제거를 전체 배열 탐색에서 마지막 id 비교로 바꾸기 (가치 2 / 위험 1 / 작업량 S)

- 왜: `web/src/pages/simple/SimpleRunDetailPage.tsx:191` 의 `receive` 가 스트림 프레임 하나마다 `current.some((line) => line.id === parsed.id)` 로 지금까지 쌓인 배열 전체를 훑어, 긴 실행(수천~수만 줄)에서 프레임당 비용이 O(n), 실행 전체가 O(n²) 입니다. 서버 `backend/internal/server/simple.go:1188` 은 `WHERE run_id=$1 AND id>$2 ORDER BY id LIMIT 500` 으로만 보내고 프레임마다 `id: %d` 를 붙여(`simple.go:1202`) 재연결 시 `Last-Event-ID` 와 `?after=` 중 큰 값에서 다시 시작하므로(`simple.go:1171-1174`), 클라이언트가 이미 가진 마지막 id 보다 크지 않은 프레임만 버리면 같은 안전망을 O(1) 로 얻습니다. 순수 헬퍼로 빼면 지금 `SimpleRunDetailPage.test.ts` 가 하는 것과 같은 방식(순수 함수 export 를 vitest 로 검사)으로 규칙을 고정할 수 있습니다.

- 수용 기준:
  1) `SimpleRunDetailPage.tsx` 의 `receive` 가 배열 전체를 훑지 않는다 — `current.some(...)` 이 사라지고, 새 순수 함수(예: `appendStreamedLine(current: SimpleLogLine[], line: SimpleLogLine): SimpleLogLine[]`)가 "`line.id` 가 `current` 의 마지막 id 보다 크면 덧붙이고, 아니면 `current` 를 그대로(같은 참조) 반환" 한다. 빈 배열에는 항상 덧붙인다.
  2) 실제 화면 동작이 바뀌지 않는다 — 정상 스트림(id 오름차순)은 전부 표시되고, 재연결로 같은 프레임이 다시 오면 한 번만 표시되며, `lastIdRef.current = parsed.id` 갱신·`end` 처리·자동 스크롤은 그대로다.
  3) `SimpleRunDetailPage.test.ts` 에 새 헬퍼의 테스트가 증명할 것: (a) 오름차순 id 3개를 차례로 넣으면 3줄이 순서대로 쌓인다, (b) 이미 있는 마지막 id 와 같은 줄을 다시 넣으면 배열이 같은 참조로 돌아온다(`toBe`), (c) 마지막 id 보다 작은 id(재연결로 되돌아온 옛 프레임)도 버린다, (d) 빈 배열에는 어떤 id 든 덧붙인다. 기존 테스트 90건은 그대로 통과.

- 건드릴 파일:
  - `web/src/pages/simple/SimpleRunDetailPage.tsx:186-195` `receive` — `setLogs((current) => appendStreamedLine(current, parsed))` 로 교체. 헬퍼는 같은 파일의 `collectStoredLogs`/`nextLogCursor` 옆에 `export function` 으로 두고, 왜 마지막 id 비교로 충분한지(서버가 id 오름차순으로만 보내고 재연결은 Last-Event-ID 로 이어짐) 주석 두세 줄로 적을 것 — 기존 주석 밀도에 맞춤.
  - `web/src/pages/simple/SimpleRunDetailPage.test.ts` — `describe('appending a streamed line', …)` 블록 추가. 기존 `line(id)` 헬퍼 재사용.
  - 문서·백엔드·VERSION 은 손대지 않음(사용자에게 보이는 약속이 바뀌지 않음).

- 검증 명령 (저장소 루트에서):
  - `cd web && npm ci && npm test -- --run` (vitest, 약 90건 + 새 4건) — 이 워크트리에는 `node_modules` 가 없으므로 `npm ci` 가 먼저 필요.
  - `cd web && npx tsc -b --noEmit` — 타입 검사(`npm test` 는 타입 오류를 못 잡음). 또는 `npm run build` 후 `web/dist` 를 지울 것.
  - 백엔드는 손대지 않으니 `go test` 는 필수 아님. 돌린다면 `cd backend && go test ./...` 은 `TEST_POSTGRES_DSN` 없이는 통합 테스트가 조용히 skip 됨(`-v` 로 확인).
  - 고치기 전 코드에 새 테스트 (b)/(c) 를 먼저 돌려 볼 것: 기존 `some` 도 (b) 는 통과하지만 (c) 는 통과하지 못하는 것이 아니라 **둘 다 통과**할 수 있음 — 그러니 "실패 → 성공" 증명 대신, 새 헬퍼가 `current` 를 같은 참조로 돌려주는 (b) 의 `toBe` 와 `some` 이 없어진 것(grep)으로 변화를 보일 것. 테스트 이름은 문장형(기존 관례).

- 위험과 피할 것:
  - 서버 정렬 가정을 넓히지 말 것 — 서버가 id 오름차순 외의 순서로 보낼 일은 없고(`ORDER BY id`), 클라이언트에서 정렬/병합 로직을 새로 만들지 말 것. 헬퍼는 "마지막 id 이하 버림" 한 줄 규칙만.
  - `lastIdRef.current = parsed.id` 는 그대로 둘 것 — 재연결 URL 의 `?after=` 가 이 값을 읽음. 헬퍼 안으로 옮기면 상태 갱신 함수가 두 번 불릴 때(StrictMode) 부작용이 생김.
  - `useEffect` 의 deps(`[live, loadingLogs, id, run]`) 나 `run.reload()`/`end` 처리는 손대지 말 것 — 이번 과제 범위 밖이고, `run` 객체 변화로 EventSource 가 다시 열리는 동작이 재연결 안전망의 일부임.
  - 렌더링 자체(프레임마다 `setLogs` → 전체 리스트 재렌더) 는 이번에 건드리지 말 것 — 별도 아이디어(rAF 배치)로 ideas.json 에 둠. 여기서 같이 하면 M 이 되고 jsdom 타이머 테스트가 필요해짐.
  - 보호 경로(auth/migrations/workflows) 는 전혀 건드리지 않음. `web/dist` 는 커밋하지 말 것. `simple.go` 의 `logBudget`/`append`(세 회차 연속 손댄 자리) 는 손대지 않음.
  - `make vet`(auto/2026-09-17-0853) 는 아직 main 에 없음 — Makefile 의 `test` 는 `go test` 두 번 + `npm test -- --run` 뿐이니 gofmt/vet 은 이번엔 무관.

- 차선 후보: `make test` 가 `TEST_POSTGRES_DSN` 없이 돌 때 통합 테스트가 통째로 skip 됐음을 눈에 띄게 알리기 (가치 2 / 위험 1 / S) — Makefile `test` 타깃 앞에 DSN 이 비어 있으면 `echo "WARN: TEST_POSTGRES_DSN unset — backend integration tests will be skipped"` 한 줄(실패시키지 않음), README 개발 절에 한 문장. 스킵 지점은 `backend/internal/server/server_test.go` 와 `simple_batch_test.go` 의 `t.Skip`(프로필 기준 448행·23행 — 이번 정찰에서 행 번호 미확인). 1순위가 성립하지 않을 때(예: 이미 누가 고쳤을 때)만.
