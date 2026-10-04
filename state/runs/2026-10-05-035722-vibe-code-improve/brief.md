- 과제: 변경 파일 테스트 추천의 Git 경로 파싱을 NUL 구분 출력으로 고치기 (가치 4 / 위험 1 / 작업량 M)
- 왜: `src/features/verification.ts:gitChangedFiles`는 줄 단위 porcelain 출력의 따옴표만 지워 한글 경로의 Git 이스케이프를 그대로 남기고, 파일명 자체의 ` -> `도 이름 변경으로 오인한다. 실제 경로를 보존하면 한국어 파일의 테스트 추천이 존재하는 파일을 가리키고 staged rename도 새 경로 하나로 정확히 처리된다.
- 수용 기준:
  1) 실제 임시 Git 저장소에서 `core.quotePath=true`일 때 미추적·추적 수정 한글 경로를 원문 그대로 반환한다. 공백, 파일명 앞뒤 공백, ` -> `가 들어간 경로도 trim/분할로 손상시키지 않는다. 일반 ASCII 파일·삭제 파일의 기존 포함 동작을 유지한다.
  2) 실제 `git mv`로 만든 staged rename은 새 경로를 정확히 한 번 반환하며 이전 경로 레코드는 결과에서 제외한다. rename 다음 일반 변경 레코드도 누락하지 않는다. `-z`에서는 첫 경로가 목적지, 다음 NUL 레코드가 원본이다. R/C 상태의 추가 원본 레코드를 건너뛰되 상태 두 칸을 경로로 오인하지 않는다.
  3) 같은 실제 저장소에서 `gitChangedFiles(ws)` → `readProjectInfo(ws, changed)` → `suggestTestCommands(changed, info)`를 연결해 한글 테스트 파일(예: `tests/한글.test.ts`) 및 한글 소스에 대응하는 실제 테스트 파일이 올바른 명령으로 추천됨을 검증한다. package.json과 테스트 파일을 실제 fs에 생성하고, 손으로 만든 ProjectInfo/가짜 execFileSync/Git 출력으로 배선을 대신하지 않는다.
  4) 깨끗한 저장소와 Git 저장소가 아닌 디렉터리는 기존처럼 빈 목록을 반환한다. 실행 전후 Git index·작업 파일·로컬/전역 설정은 그대로다. 기존 85개 테스트와 신규 테스트가 통과한다.
- 건드릴 파일:
  - `src/features/verification.ts:gitChangedFiles`(현재 186행) — git 인자에 `-z`를 추가하고 NUL 레코드를 순회한다. `XY `의 고정 3문자만 제거하고 경로 trim, 따옴표 제거, ` -> ` 분할은 없앤다. 기존 catch의 빈 배열 계약 유지. 테스트를 위해 이 함수와 `readProjectInfo`(현재 200행)의 export만 허용하며 기존 명령 등록부(299~300행)는 이 두 함수를 계속 사용한다.
  - `tests/unit/verification-git.test.ts`(신규) — 실제 fs/Git 기반 회귀 테스트. 기존 `tests/unit/checkpoints.test.ts:tempDir/repo/git/afterEach` 패턴을 참고하되 공유 스텁은 바꾸지 않는다. `git init`, 저장소 로컬 `user.name/user.email/core.quotePath`, 초기 커밋, `git mv`로 fixture를 구성한다. 커밋 서명은 fixture 명령의 `-c commit.gpgsign=false`로만 끈다.
  - 프로덕션 1파일 + 테스트 1파일. 문서·manifest·의존성 변경은 필요 없음. `docs/autonomous-goal-workflow.md:159`의 기존 테스트 추천 약속을 복원하는 작업이다.
- 검증 명령:
  - 구현 환경 준비: `node --version` 확인, package.json의 Node 20.19.2 사용 권장. 대상 node_modules 안에 Node 런타임을 설치하지 않는다. 의존성 없으면 `npm ci`.
  - 집중: `npx vitest run tests/unit/verification-git.test.ts tests/unit/verification.test.ts`
  - 전체: `npm run check` (typecheck → vitest → build), 이어 `node --check dist/extension.js`와 `node --check dist/extension.core.js`.
  - 임시 Git 테스트는 실제 저장소를 만들어야 한다. TMPDIR를 상위 Git 저장소 내부로 지정하면 `GIT_CEILING_DIRECTORIES`도 같은 절대 경로로 지정한다.
- 위험과 피할 것: 이번 범위는 Git 출력 → 경로 → 추천까지다. 공백/셸 메타문자가 들어간 추천 명령의 안전한 인자 quoting은 별도 미해결 문제이며 이 수정으로 해결됐다고 보고하지 않는다. 해당 문자열 명령은 테스트에서 실행하지 않는다. `runShellCommand`, `runAndRecord`, Python 테스트 분류, 공통 Markdown 파서, 공유 `vscode-stub.ts`, 감사 필드, 글로벌 Git 설정은 건드리지 않는다. auth/session·vendor·src/core·scripts·release·.github/workflows도 범위 밖. 지난 PR #6~#15의 날짜/EOL/감사 변경 및 원격 CI 재현 과제를 다시 수행하지 않는다. 원격 CI 실패 원인은 이번에 확인하지 않았으므로 로컬 통과를 원격 실패의 원인 증명으로 쓰지 않는다.
- 차선 후보: 하위 디렉터리의 Python `test_*.py`를 변경된 테스트로 정확히 추천 (가치 3 / 위험 1 / S) — 현재 `TEST_FILE_RE`는 `^test_`만 인정하여 `pkg/test_util.py`가 `pytest pkg/test_util.py` 대신 전체 `pytest`로 추천된다. 1순위가 이미 해결되었거나 중복 제출임이 확인될 때만 별도 채택한다. 둘을 한 회차에 섞지 않는다.

근거와 정찰 검증:
- 기준 HEAD `51148a2`, v1.4.6. 저장소 코드 수정·커밋 없음. 과제서 초안을 먼저 저장하고 실제 재현 결과로 이 최종본을 덮어썼다.
- `validation/repro.cjs`는 원본 TypeScript 전체를 메모리에서 transpile하고 두 private 함수의 export만 덧붙여 실제 함수 본문을 실행했다. 기존 `vscode-stub.ts`는 import 경계로만 사용했다. 저장소 소스는 수정하지 않았고 fixture는 이 회차 validation 디렉터리 안에만 있다.
- `validation/repro.log`: `tests/한글.test.ts` → Git octal escape가 남은 경로 → `npx vitest run tests//355/225/234/352/270/200.test.ts`. `tests/a -> b.test.ts` → `b.test.ts`. 실제 rename에서도 목적지 한글 경로에 따옴표/이스케이프가 남는다. 같은 실제 Git의 `-z` 출력은 목적지 `tests/새파일.test.ts`, 원본 `tests/한글.test.ts`를 별도 NUL 레코드로 제공했다.
- 정찰 기준선: Node 22.23.1 / Vitest 3.2.7에서 11 files / 85 tests 통과(12.06초). 현재 워크트리에 node_modules가 없어 원래 checkout의 설치된 도구를 읽기만 하고 별도 config/cache/TMPDIR를 회차 디렉터리에 두었다. 실행한 정확한 명령은 아래와 같다. 표준 `npm run check`, Node 20, Windows 패키징, VS Code UI 통합은 이번 정찰에서 미실행/미확인이다.
  `TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-10-05-035722-vibe-code-improve/validation/tmp GIT_CEILING_DIRECTORIES=/mnt/c/Users/USER/projects/aidev/state/runs/2026-10-05-035722-vibe-code-improve/validation/tmp node /mnt/c/Users/USER/projects/vibe-code/node_modules/vitest/vitest.mjs run --config /mnt/c/Users/USER/projects/aidev/state/runs/2026-10-05-035722-vibe-code-improve/validation/vitest.config.mjs --reporter=dot`
- 로컬 origin/auto 미머지 ref 10개의 base 대비 변경 경로를 대조했다. verification.ts/verification-git.test.ts는 겹치지 않는다. 현재 GitHub PR 개수/상태 자체는 조회하지 않아 미확인이다.

구현 순서와 검토 지점 (모든 단계 아직 미착수, 사람 승인 지점 없음):
1. 위 두 reader를 동작 변경 없이 export만 한다. `npm run typecheck`와 `npx vitest run tests/unit/verification.test.ts` 통과가 첫 자동 체크포인트다.
2. 신규 회귀 테스트를 실제 fs/Git으로 추가해 구체적인 잘못된 경로/추천 AssertionError를 보관한 뒤, NUL 파싱만 수정한다. 집중 명령 전체 통과가 다음 체크포인트다. 실패 원인은 새 export 누락이나 손조립 대역이어서는 안 된다. 타입체크와 집중 테스트가 통과하기 전 다음 단계로 가지 않는다.
3. `npm run check`와 두 번들 문법 검사를 실행하고 diff가 위 두 파일에 한정되는지 확인한다. 새 사실이 과제 가정을 깨면 brief의 상태/범위를 갱신한 뒤 차선을 판단하고, 조용히 범위를 늘리지 않는다.

대안과 추정 근거 (요청한 세 스킬 적용):
- 선택: Git의 NUL 프로토콜 사용. 새 의존성 없이 경로와 rename의 경계를 보존한다.
- 대안: `core.quotePath=false`만 추가하면 한글은 개선돼도 공백/화살표/줄바꿈 인용과 rename 오인은 남아 수용 기준을 못 채운다. 자체 C escape 해독기는 이스케이프·rename 문법을 계속 관리해야 한다. Git 라이브러리 도입은 같은 목적에 의존성과 변경 범위만 커진다. 일반 테스트만 추천하도록 축소하면 기존 파일별 추천 기능을 잃으므로 선택하지 않았다.
- 핵심 가정: 기능의 입력 계약은 UTF-8 파일명이 있는 실제 Git 저장소이며, Git이 내는 porcelain v1 NUL 레코드를 그대로 해석한다. 셸 명령 인용과 비 UTF-8 바이트 경로까지 이 회차에 해결하려는 과제가 아니다.
- bottom-up 작업량: 경계 export/기준선 4~6분 + Git fixture/회귀 재현 10~14분 + NUL 파싱 수정 5~8분 + 전체 검증/기록 6~9분 = 25~37분. 알려진 Git fixture/환경 차이에 대한 contingency 3~8분을 별도로 두어 총 28~45분, 약 80% 내 완료라는 정찰자의 판단(실측 통계 아님). management reserve는 0분; 새 범위는 다음 회차로 넘긴다.
- 교차 비교: 과거 한 프로덕션 파일+실제 fs 회귀 테스트 과제와 변경 크기는 유사하다. 과거 실제 소요 시간이 기록되지 않아 analogous 방식으로 분 단위 정확도를 검증할 수는 없다. 테스트 기준선 12.06초와 확인한 기존 fixture 패턴에 의존한 추정이며 fixture 구성 뒤 재평가한다.
- 적용 문서: `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md`, `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/implementation-planning/SKILL.md`, `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/solution-exploration/SKILL.md`. 전용 Skill 도구는 제공되지 않아 로컬 파일로 읽었다. 추정 스킬의 sources 목록도 읽었으며, 외부 비용 표준/실측 신뢰구간에 관한 주장은 하지 않는다.

## 구현 중 확인 (2026-10-05)
- 채택 유지. 실제 Git 회귀 테스트에서 기존 경로/추천 AssertionError 8개와 index SHA-256 변경 1개를 재현했다(`validation/git-path-red.log`). `git status` 자체가 index stat cache를 갱신하므로 “-z만 추가해도 index 불변”이라는 가정은 성립하지 않는다.
- 수용 기준 4의 index 불변을 만족하도록 같은 `gitChangedFiles` Git 호출의 `--no-optional-locks` 옵션을 분리 검증한다. 프로덕션 1파일 + 테스트 1파일, 기존 수용 범위 유지. NUL 파싱만 먼저 바꾼 결과와 옵션 적용 결과를 별도 로그로 보관한다.
