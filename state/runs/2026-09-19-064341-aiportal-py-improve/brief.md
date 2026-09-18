# 과제서 (2026-09-19, aiportal-py)

## 우선 과제 판정 — "hold: budget" 은 저장소 결함이 아니다

- 러너가 보고한 마지막 실패 사유는 `hold: budget` 이다. 이 저장소에는 GitHub Actions 워크플로가 없고(`.github/` 없음), 유일한 CI 파일 `.gitlab-ci.yml` 은 `deploy` stage 하나뿐이며(main/develop 브랜치에서 `git reset --hard` + `kubectl rollout restart`/`podman restart`) 테스트·빌드 단계가 없다. 즉 "릴리즈 워크플로가 같은 이유로 두 번 실패" 한 원인이 저장소 안의 스크립트·테스트일 수 없고, 러너(aidev) 쪽 세션 예산 상한에서 회차가 중단된 것이다.
- 이전 회차 로그 디렉터리는 `/mnt/c/Users/USER/projects/aidev/state/runs/` 아래에서 찾지 못했다(이번 run 디렉터리만 존재, 실패 로그 미확인). 저장소 작업 트리는 clean, `main` 최신 커밋 `6729686`(PR #18 머지)까지 모두 정상 머지됐다.
- 따라서 "고칠 워크플로 결함" 은 없다. 이번 회차는 **예산 안에 확실히 끝나는 S 과제**를 골라 구현하고, 원장에는 '수정 과제' 로 "budget hold 재발 방지 — 과제 크기를 S 로 제한, 문서 갱신 범위 최소화" 라고 기록한다. 워크플로(`.gitlab-ci.yml`)는 건드리지 않는다.
- 이번 정찰에서 `python3 -m pytest` 는 실행 승인이 나지 않아 **돌리지 못했다(미확인)**. 직전 회차 기록상 1025 passed. 구현자는 시작 시 먼저 한 번 돌려 기준선을 확인할 것.

## 과제: feedback_batch_loop 의 동기 배치 호출을 run_in_executor 로 옮겨 이벤트 루프 블로킹 제거 (감사 A-115) (가치 4 / 위험 2 / 작업량 S)

- 왜: `service/feedbackservice.py:236 feedback_batch_loop` 는 `async` 함수인데 동기 `process_feedback_batch(10)` 을 두 곳(라인 247 초기 실행, 269 주기 실행)에서 직접 호출한다. 배치는 행마다 `requests.get`(DEFAULT_TIMEOUT)·`requests.post`(LLM_TIMEOUT 300초)와 DB 갱신을 동기로 수행하므로 피드백 N 건이면 그 시간 동안 uvicorn 이벤트 루프가 멈춰 `/health` 를 포함한 모든 요청이 응답하지 않는다(readiness probe 실패 → 재시작 위험). 같은 배치를 부르는 `api.py:1006` 의 `POST /feedback_batch` 는 이미 `await loop.run_in_executor(None, feedbackservice.process_feedback_batch, minutes)` 를 쓰므로 대조군이 있다.
- 수용 기준:
  1) `feedback_batch_loop` 안의 두 호출이 모두 `await asyncio.get_running_loop().run_in_executor(None, process_feedback_batch, 10)` 형태가 되고, 루프 본문 어디에도 `process_feedback_batch(...)` 직접 호출이 남지 않는다.
  2) A-114 계약 유지: `FeedbackFetchError` 등 배치 예외는 `run_in_executor` 의 await 에서 그대로 올라와 기존 `except Exception` 이 잡고, `background_tasks.heartbeat(TASK_FEEDBACK_BATCH)` 는 성공한 주기에만 남는다(순서: 배치 await → heartbeat, 같은 try 본문).
  3) 테스트가 증명할 것: (a) AST 정적 검사 — `feedback_batch_loop` 안에 `process_feedback_batch` 를 **직접 호출하는** `ast.Call` 이 없고, `run_in_executor` 호출의 두 번째 위치 인자가 `Name('process_feedback_batch')` 인 `Await` 문이 초기 실행·주기 실행 두 try 본문에 각각 있음. (b) 동작 검사 — 진짜 `asyncio` 루프에서 helper 형태로 뽑은 "블로킹 함수를 executor 로 넘기는 한 주기" 를 실행하는 동안 다른 코루틴(예: 50ms 주기 카운터)이 계속 진행함을 확인(대역 FakeTask 금지 — 운영자 원칙). (c) 기존 `tests/unit/test_feedback_batch.py::test_loop_heartbeats_only_after_batch_in_same_try` 가 **계속 통과**해야 한다 — 단, 그 테스트의 `_statement_calls(stmt, "process_feedback_batch")` 는 `ast.Call` 의 func 이름만 보므로 `run_in_executor(None, process_feedback_batch, 10)` 을 "배치 호출" 로 인식하지 못해 `batch_index is None` 으로 **실패한다**. 검사를 느슨하게 하지 말고, "직접 호출 **또는** `run_in_executor` 의 두 번째 인자로 넘김" 을 배치 호출로 인정하는 helper(`_statement_runs_batch`)로 바꾸되 직접 호출은 별도 테스트에서 금지한다. 옛 코드(직접 호출)를 되돌려 넣어 새 정적 검사가 실제로 실패하는지 확인할 것.
- 건드릴 파일:
  - `service/feedbackservice.py:feedback_batch_loop` — 라인 247, 269 두 곳을 `await asyncio.get_running_loop().run_in_executor(None, process_feedback_batch, 10)` 으로. `asyncio` 는 이미 import 되어 있음(라인 266 `asyncio.sleep` 사용). 주석에 A-115 표기.
  - `tests/unit/test_feedback_batch.py` — `_statement_calls` 기반 순서 검사를 executor 경유도 인식하도록 확장 + 직접 호출 금지 검사 + executor 인자 검사 추가. 동작 검사는 같은 파일 또는 `tests/unit/test_task_supervisor.py` 의 진짜 `asyncio.Task` 방식(`asyncio.run`, 실제 `create_task`)을 따라 새 테스트로 추가.
  - `docs/CURRENT_STATE_AUDIT.md`(A-115 항목 한 줄)·`docs/OPERATIONS.md`(피드백 배치가 executor 스레드에서 돎) — **이 두 파일만**, 각 5줄 이내. 예산 상한 때문에 docs 5종 전체 갱신은 하지 말 것.
- 검증 명령: `python3 -m pytest -q tests/unit/test_feedback_batch.py tests/unit/test_task_supervisor.py` → 통과 확인 후 `python3 -m pytest -q` 전체(직전 기준 1025 passed, 이번엔 +2~4). `python3 -m pyflakes service/feedbackservice.py tests/unit/test_feedback_batch.py` undefined name 0건.
- 위험과 피할 것:
  - `CancelledError` 가 executor 실행 도중 도착하면 await 에서 올라와 기존 `except asyncio.CancelledError: break` 가 동작한다. 단 executor 스레드는 끝까지 돈다(최대 330초×N) — shutdown 이 그만큼 늦어질 수 있음을 docs/OPERATIONS.md 에 한 줄 적되, 스레드 강제 중단 같은 추가 장치는 넣지 말 것(범위 밖).
  - `api.py`, `util/task_supervisor.py`, `util/feedback_batch.py` 는 건드리지 않는다. `feedback_batch_loop` 의 heartbeat 위치·`_shutting_down` 판정(교훈 2026-09-09)도 그대로.
  - `api.py`/`service/feedbackservice.py` 는 psycopg2·kiwipiepy 미설치로 import 불가 → 정적 검사는 AST, 동작 검사는 helper 수준의 진짜 asyncio 로. 소스 문자열 검사(`"run_in_executor" in src`)는 금지.
  - 예산: 이 회차가 `hold: budget` 으로 두 번 끊겼으므로 탐색·문서 갱신을 최소화하고 코드→테스트→검증→커밋 순으로 곧장 진행할 것. 커밋 메시지는 기존 관례대로 한국어 `fix: … (감사 A-115)`.
- 차선 후보: Milvus 삭제 경로 예외화 (A-113 후속) — `util/milvus_confluence.delete_to_milvus_space_file`(except 에서 logger.error 후 None 반환)·`delete_to_milvus`(try 없음)를 `MilvusInsertError` 옆의 `MilvusDeleteError` 로 통일하고 `api.py` `/run_confluence_pipeline_delete_spacefile` 이 실패를 응답에 노출. 가치 3 / 위험 2 / S. 1순위가 성립하지 않을 때(예: 기존 순서 검사 수정이 "워크플로 느슨화" 로 판정될 우려)만 선택.
