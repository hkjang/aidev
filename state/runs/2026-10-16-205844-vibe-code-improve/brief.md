# 과제서 — 2026-10-16 (base main@6eaede8, v1.10.0)

- 과제: `suggestTestCommands` 의 "변경된 테스트 파일" 분기가 하위 디렉터리 Python 테스트와 저장소 루트 Go 테스트를 잘못 처리하는 것 고치기 (가치 3 / 위험 1 / 작업량 S)

- 왜: `src/features/verification.ts` 의 `TEST_FILE_RE`(:143)가 `^test_.*\.py$` 로 **경로 문자열 전체의 시작**에 앵커돼 있어 루트의 `test_x.py` 만 테스트로 인식하고 `pkg/test_util.py` 는 인식하지 못한다 — 그러면 `.py && hasPytest` 분기(:170-174)로 떨어져 `base="test_util"` 로 `pkg/test_test_util.py`·`tests/test_test_util.py`·`test/test_test_util.py` 를 찾고 전부 실패해 `pytest`(**전체 스위트**)를 추천한다. 방금 고친 테스트 파일 하나만 돌리면 되는데 전체를 돌리라고 하는 것이다. 같은 분기의 Go 줄(:160)은 `go test ./${dir}/...` 라 루트의 `a_test.go`(`dir === "."`)에서 `go test .//...` 라는 이상한 경로를 만든다 — 바로 아래 비테스트 Go 줄(:169)은 `dir === "." ? "" : dir + "/"` 로 그 경우를 **이미 명시적으로 처리**하고 있어, 한 함수 안에서 같은 값을 두 방식으로 쓰는 불일치다. 고치면 (a) 하위 디렉터리 파이썬 테스트가 직접 추천되고 (b) 루트 Go 테스트 추천이 `go test ./...` 로 정규화돼 `seen` 중복 제거에 걸려 거의 같은 명령 두 줄이 아니라 한 줄로 나온다.

- 수용 기준:
  1) 변경 목록에 `pkg/test_util.py` 가 있으면 추천이 `pytest pkg/test_util.py` 를 포함하고, 맨손 `pytest`(전체 파이썬 테스트)는 **포함하지 않는다**.
  2) 기존 동작 불변: 루트 `test_x.py` → `pytest test_x.py`; `tests/` · `__tests__/` · `.test.` · `.spec.` 매칭 그대로; `notatest_util.py` 같은 비테스트 이름은 테스트로 인식되지 **않는다**(세그먼트 경계가 지켜지는지의 증거).
  3) 변경 목록에 루트 `a_test.go` 가 있으면 추천에 `go test ./...` 가 나오고 `go test .//...` 는 나오지 않는다. 하위 디렉터리 `internal/proxy/server_test.go` 는 기존대로 `go test ./internal/proxy/...` 다.
  4) 기존 `tests/unit/verification.test.ts:82-86` 의 `describe("suggestTestCommands")` 기대 배열(6개 명령, 순서 포함)은 **한 글자도 바꾸지 않고** 그대로 통과한다.

- 건드릴 파일 (프로덕션 1 + 테스트 1):
  - `src/features/verification.ts:143 TEST_FILE_RE` — `^test_.*\.py$` 를 경로 세그먼트 시작으로 좁힌다. 제안: `(^|\/)test_[^/]*\.py$`. `[^/]*` 를 쓰는 이유는 `.*` 가 `/` 를 넘어가 `test_a/b.py` 까지 잡는 것을 막기 위함이다. 다른 대안(`\/test_` 추가)은 루트 케이스를 못 잡으니 쓰지 말 것.
  - `src/features/verification.ts:160` — `go test ./${dir}/...` 를 바로 아래 :169 와 **같은 형태**(`./${dir === "." ? "" : dir + "/"}...`)로 맞춘다. 두 줄이 같은 표현을 쓰게만 하고, 공통 헬퍼를 새로 만들거나 함수 구조를 바꾸지 말 것.
  - `tests/unit/verification.test.ts` — 기존 `describe("suggestTestCommands")` **블록 끝에 `it` 를 추가**하거나 파일 끝에 새 `describe` 를 붙인다. 기존 `const info` 를 재사용하되 `existingFiles` 를 바꿔야 하면 테스트 안에서 **새 객체를 만들어** 쓰고 공유 `info` 는 손대지 말 것(기준 4).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - 집중: `npx vitest run tests/unit/verification.test.ts`
  - 전체: `npm run check` (= typecheck + vitest + esbuild 빌드)
  - 번들 문법: `node --check dist/extension.js` 와 `node --check dist/extension.core.js`
  - Node 는 `engines.node` 핀인 **20.19.2** 로 돌릴 것. 기준선은 추정하지 말고 수정 전에 `npx vitest run` 을 한 번 돌려 files/tests 수를 **실측**해 기록할 것(2026-10-09 회차는 20 files / 162 tests 였다 — 이번에 다시 재어 확인하라).
  - 실패 먼저: 수정 전에 신규 테스트를 돌려 `AssertionError`(기대 `pytest pkg/test_util.py`, 실제 `pytest`)를 로그로 남기고, 고친 뒤 통과시키고, 마지막에 `src/features/verification.ts` **프로덕션 한 파일만** `git checkout HEAD --` 로 되돌려 같은 테스트가 같은 `AssertionError` 로 재발하는지 확인할 것. 이번 과제는 새 export 를 만들지 않으므로(기존 `suggestTestCommands` 만 호출) 되돌린 상태에서 `TypeError: not a function` 류로 오염되지 않는다 — 깨끗한 인과 증거가 나온다.

- 위험과 피할 것:
  - **`gitChangedFiles`(:188-199, porcelain 줄 단위 파싱 / NUL)는 절대 건드리지 말 것.** 2026-10-05 회차가 정확히 그 수정을 제출했고 main 에 머지되지 않았다 — 같은 접근 재제출 금지 규칙에 걸린다. 이번 과제는 그 아래 **순수 함수**만 바꾼다.
  - 정규식에 `\s*` 를 새로 쓰지 말 것(이 저장소가 두 번 데인 자리). 세그먼트 경계는 `(^|\/)` 로만 표현하고 `\b` 는 `_` 때문에 쓰지 말 것.
  - `readProjectInfo`(:201-230)의 `probe` 후보 목록은 **바꾸지 말 것.** 여기 손대면 실제 fs 접근이 섞여 순수 경계 증명이 흐려지고 기준 4 가 깨진다. 이번 수정은 `existingFiles` 에 이미 들어오는 값만으로 성립한다(`pkg/test_util.py` 는 `pkg/util.py` 변경 시 이미 probe 되고, 테스트는 `info` 를 직접 넘긴다).
  - `suggestTestCommands` 의 **추천 순서**를 바꾸지 말 것 — 기존 테스트가 배열 순서를 그대로 고정한다.
  - 추천 명령의 공백·셸 메타문자 quoting 은 **범위 밖**이다(별도 아이디어, 위험 3). 이번에 섞지 말 것.
  - 보호 경로 금지: `vendor/`, `src/core/`, `src/features/mcp/`, `src/features/checkpoints.ts`, `src/features/vibe-coders-proxy.ts`, `.github/workflows/`, `scripts/*.ps1`, `release/`. 공유 `tests/unit/vscode-stub.ts` 도 늘리지 말 것.
  - 버전·CHANGELOG·package.nls* 는 이 과제에서 바꿀 것이 없다(사용자에게 보이는 문구가 아니라 추천 명령 문자열이다).
  - 체크포인트 테스트가 상위 `aidev` 저장소를 탐색해 전체 검사가 멈춘 전례가 있다 — 전체 검사 시 `GIT_CEILING_DIRECTORIES` 를 임시 디렉터리로 지정할 것(2026-10-09 회차 기록).

- 미확인 (추측으로 적지 말 것):
  - 정규식 동작은 **코드 읽기로만** 확인했다. 이 세션에서 `node -e`/스크립트 실행이 권한으로 막혀 실제 `RegExp.test` 결과를 찍어 보지 못했다. 구현자는 첫 단계에서 신규 테스트로 이것을 **실측**할 것.
  - `go test .//...` 가 Go 툴체인에서 실제로 실패하는지는 **미확인**(이 환경에 Go 없음). 그래서 수용 기준 3 은 "Go 가 깨진다" 가 아니라 **추천 문자열이 :169 와 같은 형태인지**로만 적었다 — 그 선을 넘어 주장하지 말 것.
  - VS Code UI 흐름(실제 추천 퀵픽·실행), Windows 패키징(`npm run vsix`/`verify`/`smoke:vscode`), 원격 CI/PR 상태는 이번 정찰에서 미확인이다.

- 차선 후보: **로드맵의 릴리즈 자동화 현황을 현재 workflow 와 맞추기 (가치 2 / 위험 1 / 작업량 S)** — `docs/improvement-roadmap.md:9` 는 VSIX 검증 자동화의 "다음 단계" 를 적지만 저장소에는 이미 `release-vsix.yml`(published release 패키징/첨부 + 수동 태그 입력)과 `ui-snapshots.yml` 이 있다. 구현된 단계와 남은 단계(태그 push 로 release 생성은 미구현)를 분리해 적되, `docs/maintenance-guide.md` 와 중복 정본을 만들지 말고 워크플로 파일 자체는 읽기만 할 것.
