# PR 처리기 노트 2026-10-15-101316-workkit.irumx.app-shepherd — workkit.irumx.app PR #5
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-15-093843-workkit.irumx.app-improve)
# 회차 노트 2026-10-15-093843-workkit.irumx.app-improve — workkit.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 09:38] base pinned — main@4213cea
- [러너 09:38] autonomy release — 

## 정찰 노트
- 고른 이유: 1순위(저장한 비교 조건이 중복 헤더에 잘못 붙는다)는 프로덕션 파일 **1개**로 끝나고, UI 배선(`CompareTool.tsx:326` applySaved 가 `settings === null` 이면 적용을 멈춘다)을 직접 읽어 확인했으므로 빨간 테스트 → green 이 production 배선을 통과한다.
- 제친 후보: csv.ts 계열은 코드 결함이 이 워크트리에 아직 살아 있어도 열린 PR #2·#3·#4 가 같은 `end()`/`push()` 줄을 고쳐 두어 네 번째 갈래가 된다 — 제외했다. vendor/postgres PATCH 5 가설은 `errored()`(connection.js:410)가 `initial = null` 하는 것을 읽어 **기각**했다. CI 2초 실패는 10/14 세 회차가 이미 소진한 길(저장소 밖 결제 차단)이라 손대지 않았다.
- 추측으로 적은 것: 중복 헤더가 실무 파일에 흔하다는 전제는 경험적 판단이고 이 저장소 fixture 로 재현해 보지는 않았다(샘플에는 중복이 없음을 확인했다). 또 비교 열(pairs)까지 `problems` 로 막는 것이 과한지 아닌지는 구현자가 테스트로 판단하도록 완화 조건을 과제서에 적어 뒀다.
- 구현자가 조심할 것: `oldKeys`/`newKeys` 는 `preset.keys` 와 길이·순서가 1:1 이어야 한다(`presetFrom` 과 `CompareTool.tsx:336` 이 그 짝을 믿는다) — 모호할 때도 배열에서 빼지 말고 `problems` 로만 막을 것. `npm ci` 를 먼저 돌릴 것(워크트리에 node_modules 없음). e2e/a11y 는 돌리지 말 것.
- [러너 09:43] scout done — 저장한 비교 조건이 **이름이 겹치는 열**에 조용히 잘못 붙는 것 막기 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- `applyPreset`(packages/tools/src/compare/preset.ts)의 열 찾기를 `findIndex`(처음 일치) → `flatMap`(일치 개수)으로 바꿨다. 0개면 기존 "없습니다", 2개 이상이면 새 problem 한 줄(`새 파일에 'X' 열이 2개 있어 어느 것인지 알 수 없습니다.`) → `settings: null`. 프로덕션 1파일, UI 무수정.
- **과제서의 근거는 틀렸다**: `toTable`(types.ts:97~105)이 헤더를 고유화해 `비고,비고` 는 `['비고','비고 (2)']` 로 들어오고, 두 번째를 저장해도 재적용이 정확히 붙는다(probe 확인). 결함이 실제로 닿는 경로는 고유화 접미사 `X (2)` 가 **시트의 실제 열 이름과 충돌**할 때다 — 그때 저장한 비교 열 하나가 비교에서 조용히 빠진다. 테스트는 그 경로를 실제 CSV 문자열로 지나간다.
- **확신 없는 곳**: ① 이 충돌이 실무에서 얼마나 자주 나는지 — 사람이 손수 `X (2)` 로 이름 붙인 시트 + 같은 이름 중복이 동시에 필요하다. 가치를 과제서의 3 에서 2 로 내린 이유. ② `reported` Set 으로 같은 모호 문장을 기준 열·비교 열에서 한 번만 내는데, 이 중복 억제는 **새 문장에만** 걸고 기존 "없습니다" 문장은 전처럼 중복 가능하게 남겼다(기존 동작 보존 우선). ③ `-1` 이 `oldKeys`/`newKeys` 에 들어가는 것은 과제서 지시대로이며 `problems.length` 때문에 밖으로 못 나가지만, 장래에 `problems` 를 경고로 낮추면 `-1` 이 새는 구조다 — 그 가정을 코드 주석으로 적어 뒀다.
- **일부러 안 한 것**: `toTable` 의 접미사 생성 자체를 고치지 않았다(이름 충돌의 뿌리지만 열 이름이 바뀌면 저장된 조건·보고서 라벨까지 파급 — ideas.json 에 3/2/S 로 올렸다). `CompareTool.tsx` 의 이름 키 `keyMap` 충돌도 같은 뿌리지만 통합 금지 원칙상 분리했다. `csv.ts`·`.github/workflows/**`·`sanitize.ts` 는 열지 않았다.
- **다음 역할 주의**: 검증은 전부 단위 시험이다 — `npm ci` → `npx vitest run compare.test.ts` (고치기 전 1 failed | 27 passed, 뒤 28 passed) → `npm run check` (**11 파일 136 passed**, 과제서가 적은 123 은 낡은 숫자다) → `npm run build`. `e2e`·`tools`·`ui`·`a11y` 는 지시대로 돌리지 않았다(공유 dev DB :55447 때문에 이 워크트리에서 환경 탓 실패).
- [러너 09:50] brief accepted — 채택 — 권장한 수정(일치 개수 세기, `-1` push 로 배열 길이 유지, UI 무수정)과 수용 기준 1~4 를 모두 그대로 구현했지만, �
- [러너 09:50] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인한 것: eb20989 만이 이 회차의 신규 변경이다(origin/main 이 이미 4213cea 를 담아 DB 차단기 커밋은 머지됨, 로컬 main 레프만 낡음). 새 테스트가 수정 전 `findIndex` 경로에서 실제로 깨지는 것을 헤더 인덱스로 추적해 확인했고 원장의 실패 재현 출력과 증상이 일치한다. `-1` 이 `oldKeys`/`newKeys` 로 들어가는 모든 경로에서 problems 가 비지 않음을 손으로 따라 preset.ts:78 과 CompareTool.tsx:328 이 두 겹으로 막는 것을 확인했다. 승인.
- 직접 돌린 것: `npx vitest run compare.test.ts`(28 passed) → `npm run check`(136 passed) → `npm run build` 통과, 트리 clean.
- 못 본 것: `e2e`·`tools`·`ui`·`a11y`(지시대로 미실행 — 공유 dev DB :55447·Playwright). 순수 함수 1개 변경이라 위험은 낮다고 봤다.
- 승인이어도 남는 우려: ① docs/tools.md:39 가 멈추는 조건을 "열이 없으면" 으로만 적어 새 "이름이 두 열을 가리키면" 이 문서에 없다 — 릴리즈 노트에 한 줄. ② 접미사 충돌이 있는 파일의 저장 조건은 같은 파일에 재적용해도 이제 멈춘다(의도된 fail-closed, 사용자 체감 변화). ③ 뿌리인 `toTable` 고유화(types.ts:98~105)는 그대로 — 원장 보류 ④.
- 다음 회차가 알 것: problems 를 경고로 낮추는 변경을 하면 CompareTool.tsx:336·338 이 곧바로 `headers[-1]` 로 깨진다. 그 가정이 preset.ts:61~62 주석에만 있다.
- [러너 09:54] review approved — 리뷰 승인 (risk=low)
- [러너 09:54] pr created — https://github.com/hkjang/workkit.irumx.app/pull/5
- [러너 09:55] ci failed — 성공이 아닌 검사: check=failure · 실패한 검사: ? 잡: check 

## 수리 노트
- **틀린 지적**: `check` 실패는 diff 탓이 아니다. 3초·로그 0줄은 잡 미시작(Actions 결제·쿼터 차단)이고, 이 워크플로는 정상 실행 시 10분대다. diff 는 `preset.ts`+`compare.test.ts` 2파일이며 `.github/**` 무수정(YAML 파싱 확인).
- **맞았던 것**: 앞선 비평의 승인 판단. 되돌림 검증으로 새 테스트가 대상을 실제로 지나감을 확인했다 — origin/main 의 preset.ts 로 교체 시 `1 failed | 27 passed`, HEAD 로 복원 시 `28 passed`.
- **고친 방법**: 없음. 저장소 쪽 결함이 없어 커밋하지 않았다(워크플로·검증 명령 수정은 금지 경로).
- **증명한 green**: `npm ci` → `npm run check`(136 passed) → `npm run build` → `npm run kits`×2 + `cmp`(manifest 동일) → `kits:test`(3종 OK) → `scripts/ci-env.mjs`. 트리 clean.
- **확신 없는 곳**: `e2e`·`tools`·`ui`·`a11y` 는 미실행(Docker·Playwright·가짜 서버, 공유 dev DB :55447) — 순수 함수 1개 변경이라 위험은 낮다고 보지만 CI 가 실제로 돌아 본 적은 없다. `gh` 미인증·비공개 저장소라 잡 로그를 직접 열어 결제 차단을 문자로 확인하지는 못했다.
