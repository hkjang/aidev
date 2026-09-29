# 소스 분석

## 현재 형태

`vibe-code` 워크스페이스는 두 층으로 구성됩니다.

- **자체 기능 (TypeScript 소스)**: `src/` 아래에 있으며 esbuild로 `dist/extension.js`에 번들됩니다. 목표/계획 관리, 뷰, 상태바, 시드 파일, 감사 로그, `vibe-coders` 프록시 명령, 한국어 기본값 등 Vibe Code가 추가한 모든 런타임 기능이 여기에 있습니다.
- **코어 (vendored 번들)**: `vendor/extension.core.js`는 원본 확장(Roo-Code 계열)의 minified 확장 host 번들입니다. 빌드 시 `dist/extension.core.js`로 복사되고, `dist/extension.js`가 런타임에 로드합니다. 코어는 두 개의 훅(`beforeCore`, `mergeLocaleOverrides`)만 호출하며, 남아 있는 브랜딩/프롬프트 패치 목록은 `vendor/PATCHES.md`에 있습니다.

핵심 구성:

- `package.json`: VS Code 확장 manifest. 확장 ID, command, view, keybinding, setting 정의와 빌드 스크립트.
- `src/extension.ts`: 확장 entry. 훅을 설치한 뒤 `./extension.core.js`를 require해 `activate`/`deactivate`를 위임합니다.
- `src/activation.ts`: 자체 기능의 활성화 순서. 각 단계는 독립적으로 try/catch되어 하나가 실패해도 코어 활성화는 계속됩니다.
- `src/core/host.ts`: 코어가 훅에 넘겨주는 객체(`context`, `output`, `pkg`, `contextProxy`, `changeLanguage`, `getProvider`)의 타입.
- `src/features/*.ts`: 기능 모듈 (`goals`, `goal-loop`, `goal-catalog`, `goal-health`, `goal-lint`, `goal-metrics`, `handoff-on-exit`, `verification`, `checkpoints`, `plans`, `plan-codelens`, `onboarding`, `journal`, `journal-summary`, `command-audit`, `usage`, `usage-dashboard`, `tree`, `seeds`, `defaults`, `status-bar`, `diagnostics`, `team-config`, `update-check`, `welcome`, `vibe-coders-proxy`, `i18n-overrides`, `language`, `network-env`, `workspace`).
- `src/util/markdown.ts`: `.vibe-code/` 마크다운 상태 파일(목표/계획)의 섹션·메타데이터 파싱. 순수 함수이며 `tests/unit/`에서 vitest로 검증합니다.
- `vendor/extension.core.js`: 코어 번들 원본. 기능 추가 목적으로 직접 수정하지 않습니다.
- `scripts/build.mjs`: `src/` 번들 + 코어 복사. `dist/`의 런타임 자산(`node_modules`, `i18n`, `*.wasm`, `workers`)은 지우지 않습니다.
- `dist/`: 빌드 산출물과 VSIX에 실리는 런타임 자산. git에는 포함되지 않습니다.
- `webview-ui/build`: 사이드바/탭 패널에서 로드되는 웹뷰 UI 번들 (여전히 원본 소스 없음).
- `dist/i18n/locales/{ko,en}`: 코어 런타임 번역 리소스. `~/.vibe-code/locales/`의 사용자 오버라이드는 `src/features/i18n-overrides.ts`가 병합합니다.
- `package.nls*.json`: VS Code manifest UI 번역 리소스.
- `assets`: 아이콘, 한국어 데모, 슬래시 명령 템플릿, MCP 추천 설정.
- `scripts/package-vsix.ps1`: VSIX 생성 스크립트.
- `scripts/verify-package.ps1`: 릴리즈 전 정합성 검증 스크립트.
- `scripts/smoke-vscode-cli.ps1`: VSIX 격리 설치 smoke test.
- `scripts/test-extension-host.ps1`: 실제 Extension Host command smoke test.
- `tests/extension-host/index.js`: Extension Host 테스트 엔트리.
- `tests/unit/*.test.ts`: 순수 헬퍼 단위 테스트 (vitest).

## 빌드

```powershell
npm install
npm run check      # typecheck + unit test + build
npm run build      # dist/extension.js + dist/extension.core.js
npm run watch      # src/ 변경 시 자동 빌드
```

## Manifest 요약

현재 `package.json` 기준:

- command: 48개
- keybinding: 8개
- setting: 19개
- devDependency: 5개 (esbuild, typescript, vitest, @types/node, @types/vscode). 런타임 의존성은 코어 번들과 `dist/node_modules`에 포함됩니다.
- activity bar views: 4개 (`vibe-code.SidebarProvider`, `vibe-code.GoalTracker`, `vibe-code.PlanBoard`, `vibe-code.GoalPlanMap`)

중요 설정:

- `vibe-code.language`: 기본값 `ko`
- `vibe-code.tone`: `formal`, `casual`, `terse`
- `vibe-code.autonomy`: `safe`, `assist`, `auto`, `yolo`
- `vibe-code.offlineMode`: `auto`, `online`, `offline`
- `vibe-code.proxyUrl`: 사내 프록시 URL
- `vibe-code.extraCaCertsPath`: 사내 CA 인증서 PEM 경로
- `vibe-code.updateSourcePath`: 사내 공유 폴더 기반 VSIX 업데이트 경로
- `vibe-code.vibeCodersBaseUrl`: Vibe Code 모델 호출이 통과할 `vibe-coders` OpenAI 호환 프록시 URL
- `vibe-code.vibeCodersDefaultModel`: `vibe-coders proxy` provider profile 기본 모델

## 활성화 흐름

코어의 `activate`는 output channel을 만든 직후 `beforeCore` 훅을 await하고, 훅 안에서 `src/activation.ts`의 단계가 다음 순서로 실행됩니다. 이후 코어가 telemetry, sidebar provider, code index를 초기화합니다.

1. `Vibe Code` output channel 생성
2. `vibe-code.language` 설정을 `globalState.language`로 동기화
3. 프록시/CA 환경변수 적용
4. 상태바 항목 생성 및 네트워크 상태 확인
5. `현재 목표` 트리 뷰 등록 및 goal file watcher 연결
6. 데모 시나리오, `.vibeignore`, `.vibemodes`, 슬래시 명령, MCP 추천 파일 seed
7. 텔레메트리 기본 비활성화
8. 기본 모드 `architect` 적용
9. 자율성 preset 적용
10. `.vibe-code/journal/YYYY-MM-DD.md` 작업 일지 초기화
11. 보조 명령 등록:
    - `vibe-code.showContextStats`
    - `vibe-code.openJournal`
    - `vibe-code.openCurrentGoal`
    - `vibe-code.showGoalStatus`
    - `vibe-code.createGoalHandoff`
    - `vibe-code.openLatestPlan`
    - `vibe-code.advanceCurrentPlan`
    - `vibe-code.setCurrentPlanStatus`
    - `vibe-code.setCurrentPlanPriority`
    - `vibe-code.selectHighestPriorityPlan`
    - `vibe-code.showIndexInfo`
    - `vibe-code.healthCheck`
    - `vibe-code.exportTeamConfig`
    - `vibe-code.importTeamConfig`
    - `vibe-code.auditNetwork`
    - `vibe-code.applyVibeCodersProxy`
    - `vibe-code.restorePreviousProviderProfile`
    - `vibe-code.showVibeCodersProxyStatus`
    - `vibe-code.checkVibeCodersProxy`
    - `vibe-code.writeVibeCodersProxyConfig`
12. 기본 i18n 언어를 `ko`로 초기화
13. 기존 sidebar provider, code action provider, terminal action provider 등록

## vibe-coders proxy 연동

`vibe-code`는 `../vibe-coders` 프로젝트를 직접 기동하지 않습니다. 대신 `vibe-coders`가 제공하는 OpenAI 호환 proxy base URL을 Vibe Code provider profile에 저장합니다.

적용 명령은 내부 provider 설정을 다음 형태로 저장합니다.

- `apiProvider`: `openai`
- `openAiBaseUrl`: `vibe-code.vibeCodersBaseUrl`
- `openAiModelId`: `vibe-code.vibeCodersDefaultModel`
- `openAiApiKey`: 명령 실행 시 입력받은 proxy API key
- `openAiHeaders.X-Proxy-Provider`: 선택 설정 값

이후 실제 모델 호출이 `/v1/chat/completions`로 나갈 때 `vibe-coders`가 사용량, 토큰, 비용을 자동 집계합니다.

프록시 적용 전 활성 provider profile 이름은 `vibeCode.previousProviderProfile`에 저장됩니다. `vibe-code.restorePreviousProviderProfile` 명령은 이 이름을 다시 활성화하며, API key나 provider secret을 별도로 복사하지 않습니다.

`vibe-code.showVibeCodersProxyStatus` 명령은 현재 provider profile, base URL, model, `X-Proxy-Provider`, 마지막 적용 시각을 Output 채널에 표시합니다. 이 명령은 네트워크 요청이나 모델 호출을 만들지 않으므로 `vibe-coders` 집계 수치를 변경하지 않습니다.

현재 profile이 `vibe-coders` proxy를 향하고 있거나 마지막 적용 이력이 남아 있으면 상태바에 `VC: ACTIVE` 또는 `VC: READY` 항목이 표시됩니다. 클릭하면 `vibe-code.showVibeCodersProxyStatus` 명령으로 연결됩니다.

## 테스트 구조

현재 테스트는 세 단계입니다.

- VSIX smoke test: `scripts/smoke-vscode-cli.ps1`
  - VSIX를 격리된 VS Code 확장 디렉터리에 설치합니다.
  - 설치 목록, 설치된 package manifest, 필수 파일 포함 여부를 확인합니다.
- 단위 테스트: `npm run test` (`tests/unit/`, vitest)
  - 목표/계획 마크다운 파싱, 섹션 쓰기, 버전 비교 같은 순수 함수를 검증합니다.
- Extension Host test: `scripts/test-extension-host.ps1`
  - `@vscode/test-electron`이 있으면 해당 runner를 사용합니다.
  - 현재 배포물처럼 루트 의존성이 없으면 설치된 VS Code CLI fallback을 사용합니다.
  - `tests/extension-host/index.js`가 실제 extension host 안에서 `activate()`와 `vibe-code.healthCheck` 명령 실행을 검증합니다.

## 한국어 기본값

한국어 기본값은 두 겹으로 적용합니다.

- Manifest 설정: `package.json`의 `vibe-code.language.default = "ko"`
- 런타임 설정: `src/features/language.ts`가 `globalState.language`가 없으면 `vibe-code.language` 또는 `ko`를 저장하고 코어 i18n 언어를 바꿉니다. 코어 자체의 i18next 기본값도 `ko`입니다(`vendor/PATCHES.md`).

이 둘 중 하나만 바꾸면 첫 실행 또는 설정 UI에서 불일치가 생길 수 있으므로 항상 같이 검증해야 합니다.

## Goal Tracker 뷰

`vibe-code.GoalTracker`는 activity bar 안에 추가된 tree view입니다. 현재 `.vibe-code/goals/current.md`를 읽어서 아래를 보여줍니다.

- 목표 제목, 상태, 진행률 요약
- 완료 기준 체크리스트
- `Now` / `Next`
- 최근 검증 로그
- 현재 목표 열기 / 목표 상태 보기 / 목표 핸드오프 생성 바로가기

`current.md`가 없으면 `/goal`로 시작하라는 placeholder item을 보여주며, 10초 polling과 `current.md` watcher로 자동 갱신됩니다.

## Plan Board 뷰

`vibe-code.PlanBoard`는 activity bar 안에 추가된 tree view입니다. 현재 active plan 포인터가 있으면 그 파일을, 없으면 `.vibe-code/plans/*.md` 중 최신 파일을 읽어서 아래를 보여줍니다.

- 계획 제목, 상태, 우선순위, 체크리스트 진행률
- `단계` 체크리스트
- `Now` / `Next`
- `Done`
- `Risks`
- 최신 계획 열기 / 현재 계획 진행 / 계획 상태 변경 / 계획 우선순위 변경 / 현재 계획 목표 연결 / 완료 계획 보관 / 보관 계획 복원 / 현재 계획 선택 / 최고 우선순위 계획 선택 / 계획 목록 보기 / 계획 이력 보기 / 목표-계획 연결 보기 / 현재 목표 열기 / 목표 상태 보기 바로가기

`vibe-code.openLatestPlan` 명령은 active plan 포인터가 있으면 그 계획 파일을 열고, 없으면 최신 계획 파일을 열거나 `.vibe-code/plans/current-plan.md` draft를 생성합니다. `vibe-code.advanceCurrentPlan`은 첫 번째 `Now` 항목을 `Done`으로 이동하고 `Next` 첫 항목을 `Now`로 승격합니다. `vibe-code.setCurrentPlanStatus`는 active plan의 상태를 `draft`, `active`, `blocked`, `done`으로 갱신합니다. `vibe-code.setCurrentPlanPriority`는 active plan의 우선순위를 `P0`, `P1`, `P2`, `P3`로 갱신합니다. `vibe-code.linkCurrentPlanToGoal`은 active plan 파일에 현재 goal 경로와 제목을 기록합니다. `vibe-code.archiveDonePlans`는 `상태: done`인 계획 파일을 `.vibe-code/plans/archive/`로 이동합니다. `vibe-code.restoreArchivedPlan`은 archive에서 계획을 복원하고 active plan으로 지정합니다. `vibe-code.setActivePlan`은 `.vibe-code/plans/.active-plan` 포인터를 갱신합니다. `vibe-code.selectHighestPriorityPlan`은 active plans 중 우선순위가 가장 높은 계획을 active plan으로 지정합니다. `vibe-code.showPlanCatalog`는 active/archive 계획 목록과 상태를 Output에 표시하며 우선순위 `P0 -> P3` 기준으로 정렬합니다. `vibe-code.showPlanHistory`는 audit JSONL의 최근 plan 이벤트를 Output에 표시합니다. `vibe-code.showGoalPlanMap`은 현재 goal과 연결된 active/archive 계획을 Output에 맵 형태로 표시합니다. Plan Board는 10초 polling과 `plans/*.md` watcher로 자동 갱신됩니다.

## Goal Plan Map 뷰

`vibe-code.GoalPlanMap`은 activity bar 안에 추가된 tree view입니다. 현재 goal과 연결된 active/archive plan만 따로 읽어서 아래를 보여줍니다.

- 현재 목표 제목과 상태
- active linked plans
- archived linked plans
- 각 linked plan의 우선순위
- 목표-계획 연결 보기 / 현재 목표 열기 / 최신 계획 열기 바로가기
- linked plan은 우선순위 `P0 -> P3` 순으로 정렬
- active linked plan 클릭 시 `setActivePlan`, archived linked plan 클릭 시 `restoreArchivedPlan` 실행

이 뷰는 `.vibe-code/goals/current.md`, `.vibe-code/plans/*.md`, `.vibe-code/plans/archive/*.md`를 10초 polling과 watcher로 감시하면서 자동 갱신됩니다.

## 감사 로그

현재 배포물은 `.vibe-code/audit/YYYY-MM-DD.jsonl`에 goal/proxy 관련 명령 실행 이력을 append-only JSONL로 남깁니다.

- `goal`
  - `openCurrentGoal`
  - `showGoalStatus`
  - `createGoalHandoff`
- `plan`
  - `openLatestPlan`
  - `advanceCurrentPlan`
  - `setCurrentPlanStatus`
  - `setCurrentPlanPriority`
  - `linkCurrentPlanToGoal`
  - `archiveDonePlans`
  - `restoreArchivedPlan`
  - `setActivePlan`
  - `selectHighestPriorityPlan`
  - `showPlanCatalog`
  - `showPlanHistory`
  - `showGoalPlanMap`
- `command` (코어 `onCommand` 훅)
  - `approved` / `denied` / `exited` (`details.command`, `cwd`, `exitCode`)
- `proxy`
  - `showUsageReport`
  - `openUsageDashboard`
  - `showVibeCodersProxyStatus`
  - `applyVibeCodersProxy`
  - `restorePreviousProviderProfile`
  - `checkVibeCodersProxy`
  - `writeVibeCodersProxyConfig`

각 줄은 `ts`, `kind`, `action`, `details`를 가진 JSON object입니다.

## 웹뷰 UI

웹뷰 UI는 `webview-ui/build/assets/index.js`와 여러 chunk 파일로 구성됩니다. 현재 브랜딩과 로고는 다음 위치를 사용합니다.

- 로고 SVG: `assets/images/vibe-logo.svg`
- 앱명 문자열: `Vibe Code`
- 확장 ID 문자열: `vibe-code`

웹뷰 번들은 minified 상태이므로, 원본 React/TypeScript 소스 없이 직접 큰 변경을 하는 것은 위험합니다. 화면 구조 변경이 필요하면 원본 소스 확보 후 재빌드하는 것이 정석입니다.

## 데이터 디렉터리

워크스페이스에 자동 생성되는 파일:

- `.vibeignore`: 컨텍스트/임베딩 제외 규칙
- `.vibemodes`: 한국어 custom mode 정의
- `.vibe/commands/*.md`: 한국어 슬래시 명령
- `.vibe-code/journal/*.md`: 날짜별 작업 일지
- `.vibe-code/plans/`: 계획 파일 저장 위치
- `.vibe-code/plans/.active-plan`: 현재 작업 기준 plan 포인터
- `.vibe-code/plans/archive/`: 완료된 계획 파일 보관 위치
- `.vibe-code/goals/`: `/goal` 슬래시 커맨드가 관리하는 장기 목표 상태
- `.vibe-code/sessions/`: 긴 세션을 이어받기 위한 handoff 문서
- `.vibe-code/checkpoints/`: 위험 변경 전후의 확인 메모와 롤백 노트
- `.vibe-code/audit/*.jsonl`: goal/proxy 명령 실행 감사 로그
- `.vibe-code/mcp-recommendations.json`: 폐쇄망/사내 환경용 MCP 추천

VS Code global storage에는 대화 기록, API 설정, 임베딩 캐시, 모델 캐시 등이 저장될 수 있습니다.

## 위험 구간

- `dist/extension.js`: 한 줄짜리 번들에 가깝기 때문에 잘못된 문자열 치환으로 문법이 깨질 수 있습니다.
- `package.json` command/view ID: manifest ID와 runtime 등록 command가 다르면 VS Code 메뉴는 보이지만 실행이 실패합니다.
- `webview-ui/build`: chunk 이름이 빌드 결과라 수동 삭제/변경 시 import 경로가 깨질 수 있습니다.
- `dist/node_modules`: VSIX 크기 대부분을 차지합니다. slim 작업은 런타임 의존 파일을 지우지 않도록 검증이 필요합니다.

## 장기 목표 루프

`/goal` 슬래시 명령은 `assets/demo/.vibe/commands/goal.md`에서 seed됩니다. activation 시 기존 `.vibe/commands/` 폴더가 있어도 누락된 명령 파일만 보충하므로, 이미 사용 중인 워크스페이스에도 새 명령이 추가됩니다.

목표 상태는 `.vibe-code/goals/current.md`와 `.vibe-code/goals/<timestamp>-<slug>.md`에 저장합니다. 컨텍스트가 길어지면 `.vibe-code/sessions/<timestamp>-handoff.md`에 이어받기 정보를 남기는 방식으로 오래 이어지는 자율 개발을 지원합니다.
