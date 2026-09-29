# 자율 목표 개발 워크플로

## 목적

`/goal` 슬래시 커맨드는 긴 프로젝트형 개발을 한 번의 대화에 가두지 않기 위한 운영 체계입니다. 사용자가 `/goal <원하는 결과>`를 지정하면 에이전트는 목표 파일을 만들고, 작은 작업 단위로 구현/검증/기록을 반복하며, 세션이 길어져도 다음 대화에서 이어받을 수 있게 handoff를 남깁니다.

## 생성되는 파일

- `.vibe-code/goals/current.md`
  - 현재 active 목표의 최신 상태입니다.
  - 심볼릭 링크 대신 일반 Markdown 파일로 유지해 Windows와 zip 공유에 안전하게 둡니다.
- `.vibe-code/goals/<YYYY-MM-DD-HHmm>-<slug>.md`
  - 목표별 원본 기록입니다.
  - 완료 기준, 작업 큐, 결정 로그, 변경 로그, 검증 로그를 누적합니다.
- `.vibe-code/sessions/<YYYY-MM-DD-HHmm>-handoff.md`
  - 컨텍스트가 길어지거나 세션을 넘길 때 다음 작업자가 바로 이어받기 위한 요약입니다.
- `.vibe-code/checkpoints/`
  - 위험한 변경 전후의 확인 메모, 스냅샷 지시, 롤백 노트를 저장하는 위치입니다.

## VS Code 명령

- `Vibe Code: 현재 목표 열기`
  - `.vibe-code/goals/current.md`를 엽니다.
  - 파일이 없으면 draft 목표 파일을 생성합니다.
- `Vibe Code: 목표 상태 보기`
  - 현재 목표의 제목, 상태, 체크리스트 진행률, 현재 단계, 다음 행동, Now/Next, 최근 검증 로그를 Output 채널에 표시합니다.
  - 네트워크 요청이나 모델 호출 없이 로컬 목표 파일만 읽습니다.
- `Vibe Code: 목표 핸드오프 생성`
  - 현재 목표 파일을 읽어 `.vibe-code/sessions/<YYYY-MM-DD-HHmmss>-handoff.md`를 생성합니다.
  - handoff에는 다음 세션 시작 문장, 바로 할 일, 현재 목표 스냅샷이 포함됩니다.
- `Vibe Code: 최신 계획 열기`
  - `.vibe-code/plans/*.md` 중 최신 계획 파일을 엽니다.
  - 계획 파일이 없으면 `.vibe-code/plans/current-plan.md` draft를 생성합니다.
- `Vibe Code: 현재 계획 진행`
  - 최신 계획의 첫 번째 `Now` 항목을 `Done`으로 이동합니다.
  - `Next`에 항목이 있으면 첫 번째 항목을 `Now`로 올립니다.
- `Vibe Code: 현재 계획 상태 변경`
  - 최신 계획의 `상태:` 값을 `draft`, `active`, `blocked`, `done` 중 하나로 갱신합니다.
- `Vibe Code: 현재 계획 우선순위 변경`
  - 최신 계획의 `우선순위:` 값을 `P0`, `P1`, `P2`, `P3` 중 하나로 갱신합니다.
  - `우선순위:` 줄이 없으면 추가하고 마지막 갱신 시각도 함께 갱신합니다.
- `Vibe Code: 현재 계획 목표 연결`
  - 최신 계획 파일에 `연결 목표: .vibe-code/goals/current.md`와 `연결 목표 제목:`을 기록합니다.
  - 현재 작업 계획이 어떤 goal을 향하는지 명시해 handoff와 유지보수 시점을 안정화합니다.
- `Vibe Code: 완료 계획 보관`
  - `.vibe-code/plans/*.md` 중 `상태: done`인 계획 파일을 `.vibe-code/plans/archive/`로 이동합니다.
  - 완료 계획은 보존하면서 활성 계획 목록은 정리합니다.
  - 같은 이름의 보관 계획이 이미 있으면 덮어쓰지 않고 새 파일에 `-<timestamp>` suffix를 붙여 둘 다 남깁니다.
- `Vibe Code: 보관 계획 복원`
  - `.vibe-code/plans/archive/`의 완료 계획을 active plans 영역으로 되돌립니다.
  - 같은 이름의 계획이 이미 있으면 `-restored-<timestamp>` suffix를 붙여 충돌을 피합니다.
- `Vibe Code: 현재 계획 선택`
  - `.vibe-code/plans/.active-plan` 포인터 파일을 갱신해 현재 작업 기준 계획을 고릅니다.
  - 이후 계획 진행, 상태 변경, goal 연결, plan board는 이 active plan을 기준으로 동작합니다.
- `Vibe Code: 최고 우선순위 계획 선택`
  - active plans 중 우선순위가 가장 높은 계획을 현재 계획으로 선택합니다.
  - 우선순위는 `P0 -> P1 -> P2 -> P3` 순이며, 같은 우선순위에서는 파일명 순으로 결정합니다.
- `Vibe Code: 계획 목록 보기`
  - active plans와 archived plans를 Output 채널에 요약합니다.
  - 현재 active plan 포인터와 각 계획의 제목/상태/우선순위를 함께 보여줍니다.
  - 목록은 우선순위 `P0 -> P3` 순으로 정렬됩니다.
- `Vibe Code: 계획 이력 보기`
  - `.vibe-code/audit/*.jsonl`의 최근 `plan` 이벤트를 Output 채널에 출력합니다.
  - 최근 plan 수명주기 이동과 상태 변경 흐름을 빠르게 점검할 때 사용합니다.
- `Vibe Code: 목표-계획 연결 보기`
  - 현재 목표와 연결된 active/archive 계획을 Output 채널에 출력합니다.
  - 각 계획의 파일명, 상태, 우선순위, 제목, 연결 목표 제목을 함께 보여줍니다.

## 상태바

`.vibe-code/goals/current.md`가 있으면 VS Code 상태바에 `$(target) 목표: <이름> <완료>/<전체>` 항목이 표시됩니다. 이 항목은 목표 제목, 상태, 체크리스트 진행률을 10초마다 다시 읽고, 클릭하면 `Vibe Code: 현재 목표 열기` 명령으로 연결됩니다.

## Goal Tracker 뷰

Activity Bar의 `현재 목표` 뷰는 `current.md`를 읽어서 목표 요약, 완료 기준, `Now`, `Next`, 최근 검증 로그, 바로 실행 명령을 tree 형태로 보여줍니다. `current.md`가 없으면 `/goal`로 시작하라는 placeholder를 표시하고, 목표 파일 생성/수정/삭제에 따라 자동으로 갱신됩니다.

## Plan Board 뷰

Activity Bar의 `현재 계획` 뷰는 `.vibe-code/plans/.active-plan` 포인터가 있으면 그 파일을, 없으면 최신 계획 파일을 읽어서 계획 제목, 상태, 우선순위, 단계 체크리스트, `Now`, `Next`, `Done`, `Risks`, 바로 실행 명령을 tree 형태로 보여줍니다. 계획 파일이 없으면 `최신 계획 열기` 명령으로 draft를 만들라는 placeholder를 표시하고, `plans/*.md` 생성/수정/삭제에 따라 자동으로 갱신됩니다. 바로 실행에는 계획 진행, 상태 변경, 우선순위 변경, 현재 목표 연결, 완료 계획 보관, 보관 계획 복원, 현재 계획 선택, 최고 우선순위 계획 선택, 계획 목록 보기, 계획 이력 보기, 목표-계획 연결 보기가 포함됩니다.

## Goal Plan Map 뷰

Activity Bar의 `목표-계획 맵` 뷰는 현재 목표와 연결된 plan만 따로 모아 보여줍니다. active/archive 구분, 연결된 계획 수, 각 계획의 상태, 우선순위, 제목을 tree 형태로 보여주며, `current.md`, `plans/*.md`, `plans/archive/*.md` 변경에 따라 자동 갱신됩니다. linked plan은 우선순위 `P0 -> P3` 순으로 정렬되며, 항목을 클릭하면 active plan은 바로 현재 계획으로 선택되고 archive plan은 복원 후 active plan으로 지정됩니다. 바로 실행에는 `목표-계획 연결 보기`, `현재 목표 열기`, `최신 계획 열기`가 포함됩니다.

## Audit

목표/계획 관련 명령은 `.vibe-code/audit/YYYY-MM-DD.jsonl`에 자동 기록됩니다. 현재는 `openCurrentGoal`, `showGoalStatus`, `createGoalHandoff`, `openLatestPlan`, `advanceCurrentPlan`, `setCurrentPlanStatus`, `setCurrentPlanPriority`, `linkCurrentPlanToGoal`, `archiveDonePlans`, `restoreArchivedPlan`, `setActivePlan`, `selectHighestPriorityPlan`, `showPlanCatalog`, `showPlanHistory`, `showGoalPlanMap`와 `vibe-coders` 프록시 상태/설정 계열 명령이 append-only JSONL로 남습니다.

## 루프

1. 목표 확인: 새 목표인지, `.vibe-code/goals/current.md`를 이어갈지 결정합니다.
2. 완료 기준 작성: 검증 가능한 체크박스로 바꿉니다.
3. 작은 작업 선택: `Now` 항목 하나를 고릅니다.
4. 컨텍스트 수집: 필요한 파일과 문서만 읽습니다.
5. 구현: 기존 코드 패턴을 우선해 작게 수정합니다.
6. 검증: 문법 검사, 타입체크, lint, focused test, 패키징 등 가능한 검증을 실행합니다.
7. 기록: 목표 파일의 현재 상태, 작업 큐, 변경 로그, 검증 로그를 갱신합니다.
8. 반복: 완료 기준이 남아 있으면 다음 작은 작업으로 이어갑니다.

## 중단 기준

다음 상황에서는 자율 진행을 멈추고 사용자 확인을 받습니다.

- 목표가 너무 모호해 완료 기준을 만들 수 없음
- destructive command, force push, DB 삭제, 대규모 파일 삭제가 필요함
- API key, 비밀번호, 유료 결제, 외부 배포 권한이 필요함
- 같은 blocker가 세 번 반복됨
- 사용자가 중지, 보류, 방향 전환을 요청함

## 유지보수 포인트

- 템플릿 원본: `assets/demo/.vibe/commands/goal.md`
- 현재 워크스페이스 seed 결과: `.vibe/commands/goal.md`
- 런타임 seed: `dist/extension.js`
  - 기존 `.vibe/commands/` 폴더가 있어도 누락된 `.md` 명령 파일만 보충합니다.
  - legacy `.vibe/commands/목표.md`가 남아 있으면 `goal.md`로 마이그레이션하고, 이미 `goal.md`가 있으면 legacy 파일을 제거합니다.
  - `.vibe-code/goals`, `.vibe-code/sessions`, `.vibe-code/checkpoints`, `.vibe-code/audit` 디렉터리를 필요 시 생성하고, 완료 계획은 `.vibe-code/plans/archive/`로 분리합니다.
  - `.vibe-code/plans/.active-plan` 포인터 파일을 사용해 active plan을 결정합니다.
  - `vibe-code.openCurrentGoal`, `vibe-code.showGoalStatus`, `vibe-code.createGoalHandoff`, `vibe-code.openLatestPlan`, `vibe-code.advanceCurrentPlan`, `vibe-code.setCurrentPlanStatus`, `vibe-code.setCurrentPlanPriority`, `vibe-code.linkCurrentPlanToGoal`, `vibe-code.archiveDonePlans`, `vibe-code.restoreArchivedPlan`, `vibe-code.setActivePlan`, `vibe-code.selectHighestPriorityPlan`, `vibe-code.showPlanCatalog`, `vibe-code.showPlanHistory`, `vibe-code.showGoalPlanMap` 명령을 등록합니다.
  - `vibe-code.GoalTracker` tree view를 등록하고 `current.md` watcher와 polling으로 자동 갱신합니다.
  - `vibe-code.PlanBoard` tree view를 등록하고 `plans/*.md` watcher와 polling으로 자동 갱신합니다.
  - goal/proxy 이벤트를 `YYYY-MM-DD.jsonl`에 append합니다.
  - active goal 상태바 항목을 생성해 현재 목표와 진행률을 빠르게 볼 수 있게 합니다.
- 검증:
  - `scripts/verify-package.ps1`가 VSIX 안에 `/goal` 템플릿과 이 문서가 포함되는지 확인합니다.
  - `tests/extension-host/index.js`가 실제 Extension Host activation 후 워크스페이스에 `/goal` 파일, 목표 디렉터리, 현재 목표 파일, handoff 파일이 생성되는지 확인합니다.

## 확장이 루프를 굴리는 방식

- **프롬프트 컨텍스트**: 코어가 시스템 프롬프트를 만들 때마다 `promptContext` 훅으로 `current.md` 요약(목표, 상태, 단계, 다음 행동, Now/Next, 완료 기준 진행, 루프 규칙)이 OBJECTIVE 섹션 뒤에 붙습니다. 모델이 매 턴 파일을 다시 읽지 않아도 목표를 기억합니다. 상태가 `active`/`draft`일 때만 붙습니다.
- **자동 재개**: 활성화 4초 후 활성 목표에 Now 항목이 남아 있고 `마지막 갱신`이 `vibe-code.goalResumeWithinHours`(기본 72시간) 안이면 "이어서 진행할까요?"를 묻습니다(`vibe-code.goalAutoResume`: `ask`/`always`/`never`). 승인하면 사이드바에 `/goal 이어서` 작업을 시작합니다. `Vibe Code: 목표 이어서 진행` 명령으로 언제든 같은 동작을 합니다.
- **작업 완료 확인**: 코어의 `onTask` 훅으로 작업 완료를 받으면 목표 파일이 그 작업 중 갱신됐는지 확인합니다. 갱신되지 않았으면 경고(목표 열기 / 이어서 갱신 요청), Now 항목이 남았으면 "계속 진행"을 제안합니다. `goal/taskCompleted` 감사 이벤트에 토큰·비용·소요 시간이 남습니다.
- **자동 handoff**: 코어가 컨텍스트를 압축(condense)하면 `.vibe-code/sessions/`에 handoff를 자동 생성하고 목표 파일에 기록합니다.

## 여러 목표 다루기

- `목표 목록` 뷰: `.vibe-code/goals/*.md`를 현재 목표 우선, 최근 갱신순으로 보여줍니다. 항목을 클릭하면 그 목표로 전환합니다.
- `Vibe Code: 현재 목표 전환`: 전환 전에 `current.md`를 `목표 파일:` 줄이 가리키는 원본 파일에 저장합니다(줄이 없으면 `<시각>-<slug>.md`를 만들고 줄을 추가). 편집이 유실되지 않습니다.
- `Vibe Code: 새 목표`: 제목 → 유형 프리셋(기능 개발/버그 수정/리팩터링/문서화/자유 형식, 각각 다른 완료 기준) → 설명 순으로 묻고 목표 파일을 만들어 현재 목표로 둡니다. "바로 시작"을 누르면 사이드바에 `/goal <제목>` 작업을 시작합니다.
- `Vibe Code: 계획을 완료 기준과 연결`: 활성 계획에 `연결 완료 기준: 1, 3`을 기록합니다. 그 계획이 `done`이 되면 목표의 해당 완료 기준이 자동 체크됩니다.

## 지표, 회고, 안전장치

- `Vibe Code: 목표 진행 지표 (7일)`: audit JSONL을 집계해 하루 완료 항목 수, 검증 통과율과 실패 명령, 명령 승인/거부/실패, 목표 미갱신 작업 수, 토큰·비용, 핸드오프·정체 경고 횟수를 Output에 보여줍니다.
- `Vibe Code: 주간 회고 생성`: 같은 집계에 vibe-coders 주간 사용량, 완료 목록, 반복 실패(병목), 다음 주 항목(목표의 미완료 Now·완료 기준)을 붙여 `.vibe-code/retro/<날짜>-weekly.md`를 만듭니다.
- 체크포인트: 파괴적 명령(`rm -rf`, `git reset --hard`, force push, `DROP TABLE` 등)이 승인되면 실행 직전에 추적 파일의 미커밋 변경과 미추적 파일(.gitignore 제외분 제외)을 임시 index로 함께 스냅샷해 `refs/vibe-checkpoints/<시각>`에 고정한 뒤 복원 명령이 적힌 메모를 `.vibe-code/checkpoints/`에 남깁니다(`vibe-code.checkpointBeforeDestructive`). 수동 생성과 목록 명령도 있습니다.
- 예산 가드: `vibe-code.weeklyBudgetKrw`를 넘으면 하루 한 번 auto/yolo 자율성을 assist로 낮추고 경고합니다. 상태바 프록시 항목이 경고색으로 바뀝니다.

## 정체 감지와 파일 린트

- 상태바 목표 항목은 활성 목표가 `vibe-code.goalStaleHours`(기본 48시간) 동안 갱신되지 않았거나, 같은 첫 Now 항목이 3세션째 남아 있으면 경고색으로 바뀌고 툴팁에 이유를 보여줍니다. 활성화 시 한 번 경고 토스트도 뜹니다.
- 목표/계획 파일을 열면 구조 검사가 Problems 패널과 밑줄로 표시됩니다: 제목 줄 형식, 필수 섹션 누락, 잘못된 `상태:`/`우선순위:`, Now 미완료 항목 과다(목표 4개, 계획 3개 초과), `done`인데 미체크 항목, `연결 목표:` 누락.
- 확장이 비활성화될 때(창 닫기·리로드) 활성 목표에 마지막 handoff 이후 audit 이벤트가 있으면 handoff를 자동 생성합니다.

## 검증 실행과 완료 게이트

- 계획의 `## 검증 계획`(또는 목표의 `## 완료 기준`) 항목이 명령이면(백틱으로 감싸거나 `npm`, `go`, `pytest` 같은 러너로 시작) `검증 실행` CodeLens가 붙습니다. 워크스페이스 루트에서 실행하고 출력은 Vibe Code Output 채널에 흐르며, 결과는 계획과 목표 파일의 `## 검증 로그`에 `- <시각> - \`명령\` → OK/FAIL exit N (초)` 형태로 남습니다. 통과하면 항목이 체크됩니다.
- `Vibe Code: 검증 실행`: 활성 계획의 검증 명령 목록에서 고르거나 직접 입력합니다.
- `Vibe Code: 변경 파일 테스트 추천`: `git status` 변경 파일에 대응하는 테스트 명령을 추천하고 여러 개를 골라 순서대로 실행합니다. 실패하면 거기서 멈춥니다.
- 완료 게이트: 계획을 `done`으로 바꿀 때 `단계`/`Now`/`검증 계획`에 미체크 항목이 있으면, 목표를 `done`으로 바꿀 때(`Vibe Code: 목표 상태 변경`) `완료 기준`에 미체크 항목이 있으면 목록을 보여주고 확인을 받습니다. 강행하면 audit에 `completionGateOverridden`이 남습니다.

## 계획 파일 CodeLens

`.vibe-code/plans/*.md`를 편집기에서 열면 체크리스트 줄 위에 CodeLens가 붙습니다.

- `## Now` 항목: `완료로 이동` (Done으로 옮기고 `[x]` 처리), `Next로 되돌리기`
- `## Next` 항목: `Now로 승격`
- 그 외 체크리스트(`## 단계`, `## 검증 계획` 등): `체크` / `체크 해제`
- 제목 줄(`# 계획:`): `현재 계획으로 선택` (`.active-plan` 포인터 갱신)

각 동작은 열린 문서를 편집·저장하고 `마지막 갱신:`을 갱신하며 `.vibe-code/audit/`에 `completePlanItem`, `deferPlanItem`, `promotePlanItem`, `togglePlanCheckbox` 이벤트를 남깁니다. `Vibe Code: 현재 계획 진행` 명령과 같은 규칙으로 섹션을 다시 씁니다.

## 세션 요약

확장이 비활성화될 때(창 닫기, 리로드) 오늘 audit JSONL에서 마지막 요약 이후의 이벤트를 모아 `.vibe-code/journal/<날짜>.md` 끝에 `## 세션 요약 (HH:MM KST)` 블록을 추가합니다. 완료한 계획 항목, 계획 상태 변경, 보관/핸드오프 횟수, 프록시 적용/복원, 실패 이벤트, 현재 목표의 `다음 행동`과 남은 Now 항목이 들어갑니다. 새 이벤트가 없으면 아무것도 쓰지 않습니다. 블록 끝의 `<!-- vibe-code:summary-until ... -->` 주석이 다음 요약의 기준점입니다.
