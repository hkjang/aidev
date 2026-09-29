# 유지보수 가이드

## 기본 원칙

Vibe Code가 추가한 기능은 모두 `src/`의 TypeScript 소스입니다. 기능 변경은 `src/`에서 하고 `npm run build`로 `dist/extension.js`를 다시 만듭니다. 코어 번들 `vendor/extension.core.js`는 원본 확장의 빌드 산출물이므로 기능 추가 목적으로 손대지 않습니다(허용되는 예외는 `vendor/PATCHES.md` 참고).

변경 전에는 항상 다음 파일을 먼저 확인합니다.

- `package.json`: command, view, setting, keybinding, 확장 ID, 빌드 스크립트
- `package.nls.json`, `package.nls.ko.json`: VS Code 설정/명령 UI 번역
- `src/activation.ts`: 자체 기능 활성화 순서
- `src/features/*.ts`: 기능별 구현
- `src/util/markdown.ts` + `tests/unit/`: 목표/계획 파일 파싱 규칙과 테스트
- `vendor/PATCHES.md`: 코어 번들에 남아 있는 패치와 훅 계약
- `webview-ui/build/assets/index.js`: 웹뷰 UI 번들 (원본 소스 없음)
- `scripts/package-vsix.ps1`: 릴리즈 패키징
- `scripts/verify-package.ps1`: 릴리즈 전 검증

## 개발 환경

```powershell
npm install          # esbuild, typescript, vitest, 타입 정의만 설치됩니다
npm run watch        # src/ 변경 시 dist/extension.js 자동 빌드
npm run check        # typecheck + unit test + build
```

`dist/`는 git에 포함되지 않습니다. 새로 clone한 환경에서는 release VSIX의 런타임 자산(`node_modules`, `i18n`, `*.wasm`, `workers`)을 먼저 복원합니다.

```powershell
gh release download --repo hkjang/vibe-code --pattern '*.vsix' --dir release   # 또는 기존 VSIX 사용
node scripts/restore-dist-assets.mjs            # release/ 의 최신 VSIX에서 dist/ 복원
node scripts/restore-dist-assets.mjs --from path\to\vibe-code-1.4.5.vsix
npm run build
```

`scripts/build.mjs`가 빠진 자산을 경고로 알려줍니다.

## CI

`.github/workflows/ci.yml`이 push/PR마다 실행됩니다.

1. `check` (ubuntu): `npm ci`, `npm run check`, `node --check`.
2. `package` (windows): 빌드 후 최신 GitHub release의 VSIX에서 런타임 자산을 복원하고 `package-vsix.ps1`, `verify-package.ps1`을 실행한 뒤 VSIX와 SHA256을 artifact로 올립니다. release가 없으면 패키징 단계는 건너뜁니다.

새 버전을 배포할 때는 VSIX를 GitHub release 자산으로 올려야 다음 CI가 그 안의 런타임 자산을 다시 쓸 수 있습니다.

VS Code에서 디버그하려면 이 폴더를 열고 `F5`(Extension Development Host) 대신 `code --extensionDevelopmentPath=<repo>`를 사용하거나, `npm run test:extension-host`로 실제 Extension Host 테스트를 실행합니다.

## 변경 절차

1. `package.json`에서 manifest ID와 설정 기본값을 먼저 정리합니다.
2. `package.nls*.json`에 새 command/setting의 표시 문자열을 추가합니다.
3. 런타임 동작은 `src/features/`에 구현하고 `src/activation.ts`의 단계 목록에 등록합니다. 새 명령은 `package.json`의 `contributes.commands`와 ID가 일치해야 합니다.
4. 마크다운 파일 형식이 바뀌면 `src/util/markdown.ts`와 `tests/unit/`을 함께 갱신합니다.
5. 사용자 안내가 필요한 기능은 `readme.ko.md`와 `docs/`를 함께 갱신합니다.
6. `scripts/verify-package.ps1`와 `tests/extension-host/index.js`에 회귀 검증 조건을 추가합니다.
7. `npm run check` 후 VSIX를 다시 만들고 설치 가능 여부를 확인합니다.

## vibe-coders 프록시 변경 시

- 확장은 `vibe-coders`를 기동하지 않습니다.
- 모델 호출이 `vibe-code.vibeCodersBaseUrl`로 가도록 provider profile만 적용합니다.
- `vibe-code.applyVibeCodersProxy` 명령은 `apiProvider=openai`와 `openAiBaseUrl`을 저장합니다.
- 적용 전 활성 provider profile 이름은 복원용으로만 저장합니다.
- `vibe-code.restorePreviousProviderProfile` 명령은 저장된 profile 이름을 다시 활성화합니다.
- `vibe-code.showVibeCodersProxyStatus` 명령은 네트워크 호출 없이 현재 라우팅 상태와 마지막 적용 이력을 Output에 표시합니다.
- `vibe-coders` proxy 상태바 항목은 routed 상태면 `VC: ACTIVE`, 마지막 적용 이력만 있으면 `VC: READY`로 보여야 하며 클릭 시 `vibe-code.showVibeCodersProxyStatus`로 연결되어야 합니다.
- 사용량 집계는 `vibe-coders`가 프록시 호출을 받으면서 자동 처리합니다.
- API key는 문서/설정 파일에 쓰지 않고 적용 명령에서 Vibe Code의 내부 provider 저장 흐름에 맡깁니다.

## 한국어 기본값 체크리스트

- `package.json`의 `vibe-code.language.default`가 `ko`인지 확인합니다.
- `src/features/language.ts`가 첫 실행 시 `globalState.language`를 `ko`로 초기화하는지 확인합니다.
- `package.nls.ko.json`의 command/setting 번역이 누락되지 않았는지 확인합니다.
- `readme.ko.md`가 VSIX 안에 포함되는지 확인합니다.
- `Vibe Code: 환경 점검` 명령에서 settings/globalState 언어가 모두 `ko`로 보이는지 확인합니다.

## `/goal` 슬래시 명령 변경 시

- 템플릿 원본은 `assets/demo/.vibe/commands/goal.md`입니다.
- 현재 개발 워크스페이스 확인용으로 `.vibe/commands/goal.md`도 함께 갱신합니다.
- 기존 워크스페이스를 위해 `src/features/seeds.ts`의 slash command seed 로직은 누락 파일만 보충해야 하며, 사용자가 수정한 기존 명령 파일은 덮어쓰지 않아야 합니다.
- legacy `.vibe/commands/목표.md`는 activation 시 `goal.md`로 마이그레이션되거나 정리되어 `/목표`가 남지 않아야 합니다.
- 장기 상태 디렉터리 `.vibe-code/goals`, `.vibe-code/sessions`, `.vibe-code/checkpoints`가 activation 시 생성되는지 Extension Host 테스트로 확인합니다.
- `.vibe-code/audit/*.jsonl`에 goal/proxy 이벤트가 append되는지 Extension Host 테스트로 확인합니다.
- `vibe-code.openCurrentGoal`, `vibe-code.showGoalStatus`, `vibe-code.createGoalHandoff`, `vibe-code.openLatestPlan`, `vibe-code.advanceCurrentPlan`, `vibe-code.setCurrentPlanStatus`, `vibe-code.setCurrentPlanPriority`, `vibe-code.linkCurrentPlanToGoal`, `vibe-code.archiveDonePlans`, `vibe-code.restoreArchivedPlan`, `vibe-code.setActivePlan`, `vibe-code.selectHighestPriorityPlan`, `vibe-code.showPlanCatalog`, `vibe-code.showPlanHistory`, `vibe-code.showGoalPlanMap` manifest/runtime 등록이 함께 맞는지 확인합니다.
- `vibe-code.GoalTracker` view manifest 등록과 runtime registration marker(`goal tracker view registered`)가 함께 맞는지 확인합니다.
- `vibe-code.PlanBoard` view manifest 등록과 runtime registration marker(`plan board view registered`)가 함께 맞는지 확인합니다.
- `vibe-code.GoalPlanMap` view manifest 등록과 runtime registration marker(`goal plan map view registered`)가 함께 맞는지 확인합니다.
- active goal 상태바 항목은 `current.md`가 있을 때만 표시되어야 하며, 체크리스트 진행률을 함께 보여주고 클릭 시 `vibe-code.openCurrentGoal`로 연결되어야 합니다.
- `vibe-coders` proxy 상태바 항목은 runtime registration marker(`proxy status bar item created`)가 남아야 합니다.
- 계획 draft는 `.vibe-code/plans/current-plan.md`로 생성되어야 하며 `## Now` 섹션이 포함되어야 합니다.
- `vibe-code.advanceCurrentPlan` 실행 후 첫 번째 `Now` 항목이 `Done`으로 이동하고, `vibe-code.setCurrentPlanStatus`는 `상태:` 줄을 직접 갱신해야 합니다.
- `vibe-code.setCurrentPlanPriority`는 `우선순위:` 줄을 직접 갱신하거나 없으면 추가해야 합니다.
- `vibe-code.linkCurrentPlanToGoal`는 최신 계획에 `연결 목표:`와 `연결 목표 제목:` 줄을 기록해야 합니다.
- `vibe-code.archiveDonePlans`는 완료된 계획을 `.vibe-code/plans/archive/`로 이동하고 audit에 `archiveDonePlans`를 남겨야 합니다.
- `vibe-code.restoreArchivedPlan`는 archive의 계획을 active plans 영역으로 복원하고 audit에 `restoreArchivedPlan`을 남겨야 합니다.
- `vibe-code.setActivePlan`는 `.vibe-code/plans/.active-plan`를 갱신하고 이후 plan 명령이 해당 파일을 기준으로 동작하게 해야 합니다.
- `vibe-code.selectHighestPriorityPlan`은 active plans를 `P0 -> P3` 순으로 정렬해 `.active-plan`을 갱신하고 audit에 `selectHighestPriorityPlan`을 남겨야 합니다.
- `vibe-code.showPlanCatalog`는 active/archive 계획의 제목과 상태를 출력하고 audit에 `showPlanCatalog`를 남겨야 합니다.
- `vibe-code.showPlanHistory`는 최근 plan audit 이력을 출력하고 audit에 `showPlanHistory`를 남겨야 합니다.
- `vibe-code.showGoalPlanMap`은 현재 goal과 연결된 active/archive 계획을 출력하고 audit에 `showGoalPlanMap`을 남겨야 합니다.
- 목표 루프 문서는 `docs/autonomous-goal-workflow.md`에 유지합니다.

## 검증 명령

```powershell
npm run check
powershell -ExecutionPolicy Bypass -File scripts/verify-package.ps1
powershell -ExecutionPolicy Bypass -File scripts/package-vsix.ps1
powershell -ExecutionPolicy Bypass -File scripts/verify-package.ps1
powershell -ExecutionPolicy Bypass -File scripts/smoke-vscode-cli.ps1
powershell -ExecutionPolicy Bypass -File scripts/test-extension-host.ps1
```

검증 스크립트는 다음을 확인합니다.

- 확장 ID가 `vibe-code.vibe-code`로 유지되는지
- 모든 command/setting prefix가 `vibe-code.`인지
- 기본 언어가 `ko`인지
- 오래된 브랜딩 문자열이 다시 들어오지 않았는지
- `dist/extension.js`, `dist/extension.core.js`, 웹뷰 번들이 Node 문법 검사를 통과하는지
- `dist/extension.js`가 코어를 로드하고, `dist/extension.core.js`가 `vendor/`와 같고 훅 호출을 포함하는지
- 각 goal/plan/proxy 명령의 `writeAudit` 호출이 번들에 남아 있는지
- VSIX에 필수 파일이 포함되는지

`smoke-vscode-cli.ps1`는 VS Code CLI로 VSIX를 격리 설치한 뒤 다음을 확인합니다.

- 설치 목록에 `vibe-code.vibe-code@<package.json version>`이 보이는지
- 설치된 package manifest가 `vibe-code` ID와 한국어 기본값을 유지하는지
- 런타임 번들, 웹뷰 번들, 한국어 README, 유지보수 문서가 설치 경로에 있는지

`test-extension-host.ps1`는 실제 Extension Host에서 테스트 엔트리를 실행합니다.

- `@vscode/test-electron`이 설치되어 있으면 정식 test-electron runner를 사용합니다.
- 현재 패키지처럼 루트 `node_modules`가 없으면 설치된 VS Code CLI의 `--extensionDevelopmentPath` / `--extensionTestsPath` 경로로 fallback합니다.
- 테스트는 `vibe-code.vibe-code` 확장 발견, 한국어 기본값, 필수 command 등록, `activate()`, `vibe-code.healthCheck` 실행을 확인합니다.

## 릴리즈 절차

1. `changelog.md`에 변경 내역을 기록하고 `package.json` 버전을 올립니다.
2. `npm run check`를 실행합니다.
3. `scripts/verify-package.ps1`를 실행합니다.
4. `scripts/package-vsix.ps1`를 실행합니다.
5. `scripts/verify-package.ps1`를 다시 실행합니다.
6. `scripts/smoke-vscode-cli.ps1`로 격리 설치 smoke test를 실행합니다.
7. `scripts/test-extension-host.ps1`로 Extension Host 테스트를 실행합니다.
8. VS Code에서 VSIX를 설치해 사이드바, 명령 팔레트, `Vibe Code: 환경 점검`을 실행합니다.

## 주의사항

- `dist/extension.js`는 빌드 산출물입니다. 직접 고치지 말고 `src/`를 고친 뒤 빌드합니다.
- `vendor/extension.core.js`는 minified 코어이므로 훅 계약 외의 수정을 피합니다.
- command ID를 바꾸면 `package.json`, runtime 등록, webview 호출이 모두 맞아야 합니다.
- `webview-ui/build`의 chunk 파일은 서로 import 관계가 있으므로 개별 삭제를 피합니다.
- `dist/node_modules` slimming은 VSIX 크기를 줄이지만 런타임 의존 파일을 지울 수 있습니다.
- 웹뷰 UI(`webview-ui/build`)는 아직 원본 소스가 없습니다. 화면 구조 변경은 원본 소스 확보 후 재빌드하는 것이 정석입니다.
