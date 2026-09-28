- 과제: ESLint에서 Playwright 생성 보고서·실행 결과 제외 (가치 2 / 위험 1 / 작업량 S)
- 왜: `eslint.config.mjs`의 `globalIgnores`에 Playwright 생성물 경로가 없어 실패 trace의 JavaScript까지 소스로 검사하며, 최근 두 회차는 보고서를 삭제하고 lint를 실행했다. 생성 디렉터리 두 곳만 제외하면 실패 증거를 보존하면서 실제 제품·테스트 소스의 린트를 계속 실행할 수 있다.
- 수용 기준: 1) `playwright-report/**`와 `test-results/**` 아래 중첩 JavaScript 경로를 실제 ESLint가 무시한다. 2) `app/admin/page.tsx`, `e2e/tree-ops.spec.ts`, `scripts/guide-screenshots/playwright.config.ts`는 무시하지 않고 기존 규칙으로 검사한다. 3) 아래 실제 ESLint API 검증과 `npm run lint`가 통과하며, 검증을 위해 기존 보고서·trace를 삭제하지 않는다.
- 건드릴 파일: `eslint.config.mjs:eslintConfig/globalIgnores` — 기존 항목을 유지하고 `playwright-report/**`, `test-results/**`만 추가. 설정 1파일, 제품 코드 0파일, 영구 테스트 추가 불필요.
- 검증 명령: 의존성 없을 때 `npm ci --legacy-peer-deps`; 아래 ESLint API 검증; `npm run lint`; `git diff --check`. 단위 회귀 확인은 `npm run test:unit`(이번 정찰 95건/30 suites 통과). E2E·build는 설정 두 줄 변경의 필수 증거가 아니며 이번 범위에서는 요구하지 않는다.
- 위험과 피할 것: `e2e/**`, `scripts/**`, `**/*.js` 전체 제외 금지. auth·data I/O·proxy·Dockerfile·workflows·package.json·lockfile·Playwright 설정 변경 금지. 보고서 삭제나 lint 명령 축소로 통과시키지 않는다. 생성물 외 기존 lint 오류가 보이면 별도 기록하고 범위를 넓히지 않는다. AGENTS.md 블록을 지우지 않는다.
- 차선 후보: `validateActivityImport` 자기참조 오류 중복 제거(가치 2 / 위험 1 / S) — 1순위의 두 경로가 이미 실제 ESLint에서 제외되어 과제가 성립하지 않을 때만. `lib/activityData.ts:validateActivityImport`의 자기참조 전용 진단을 유지하며 별도 `visit` 순환 패스의 같은 오류만 억제하고, `lib/activityData.test.ts`의 자기참조 2건 기대를 1건으로 변경. 2노드 이상 순환·자기참조 노드에 연결된 자식 검증은 보존. `node --test lib/activityData.test.ts`와 `npm run test:unit`로 검증. 저장 I/O 함수는 수정하지 않는다.

## 범위 및 확인한 근거
- 대상 사용자는 로컬 E2E 실패를 조사하면서 린트를 실행하는 개발자·구현 러너다. 릴리즈 파이프라인 교체나 테스트 실행 정책 변경은 범위 밖이다.
- `eslint.config.mjs` 전체 확인: 기존 globalIgnores는 `.next/**`, `out/**`, `build/**`, `next-env.d.ts`뿐.
- `package.json`의 lint는 `eslint`(별도 대상 제한 없음). `playwright.config.ts`는 HTML reporter와 `trace: 'retain-on-failure'`, `screenshot: 'only-on-failure'` 사용.
- `e2e/globalSetup`은 `test-results/e2e-data/activity.json`을 만들고 `globalTeardown`은 e2e-data만 지운다. 전체 test-results 삭제를 해결책으로 삼으면 진단 산출물이 소실된다.
- 현재 워크트리에 node_modules가 없어서 `npm run lint`는 `eslint: not found`, exit 127. 워크트리 정식 lint 통과는 **미확인**이다.
- 출력 폴더에 현재 설정을 그대로 복사하고 기존 `/mnt/c/Users/USER/projects/cutover/node_modules`의 실제 ESLint를 연결한 읽기 실험을 했다. cwd는 이 저장소, overrideConfigFile은 복사본이다. `isPathIgnored`는 두 생성물 경로와 세 소스 경로 모두 false; `lintText('const = ;', {filePath})`는 두 생성물 경로 모두 실제 파싱 오류 1건을 반환했다. 문자열 검색으로 동작을 증명한 것이 아니며, 원본 저장소는 수정하지 않았다. 잠재 경로에 대한 실제 엔진 검증이며, 이번에 실제 실패 trace를 새로 생성한 것은 아니다.

## 실행 계획 (모든 단계 pending; 사람 승인 체크포인트 없음)
1. 환경 및 기준선: 의존성을 준비하고 아래 API 검증을 수정 전에 실행한다. 현재 기대는 첫 생성 경로 assertion 실패이며, 의존성 로드 실패와 구분한다. 예상과 다르면 원인을 기록하고 과제서를 갱신한다. 증거: 아래 명령의 종료 상태와 실제 경로 판정.
2. 변경: `eslint.config.mjs`의 globalIgnores에 두 경로만 추가한다. 증거: 아래 동일 검증이 통과하고 `npm run lint`가 통과한다. 새 테스트 프레임워크·스크립트 파일을 도입하지 않는다.
3. 마무리: `git diff --check` 및 `git diff --stat`로 설정 1파일 범위를 확인한다. 필요 시 `npm run test:unit`을 한 번 실행한다. 구현 완료 기록에 명령/결과를 남긴다. 실패 보고서가 이미 있으면 보존한다.

### 구현자가 저장소 루트에서 실행할 검증

```bash
node --input-type=module <<'JS'
import assert from 'node:assert/strict';
import { ESLint } from 'eslint';
const eslint = new ESLint();
for (const filePath of [
  'playwright-report/trace/index.js',
  'test-results/trace/index.js',
]) {
  assert.equal(await eslint.isPathIgnored(filePath), true, filePath);
  const [result] = await eslint.lintText('const = ;', { filePath });
  assert.equal(result.errorCount, 0, filePath);
}
for (const filePath of [
  'app/admin/page.tsx',
  'e2e/tree-ops.spec.ts',
  'scripts/guide-screenshots/playwright.config.ts',
]) {
  assert.equal(await eslint.isPathIgnored(filePath), false, filePath);
  const [result] = await eslint.lintText('const = ;', { filePath });
  assert.ok(result.errorCount > 0, `소스 검사가 비활성화됨: ${filePath}`);
}
console.log('생성물 제외 및 실제 소스 검사 확인 완료');
JS
npm run lint
git diff --check
```

`lintText`는 파일을 쓰지 않으며 경로별 실제 parser/ignore 배선을 검사한다. 생성물의 ignored 경고는 허용하되 errorCount는 0이어야 한다. 위 검증은 현재 설정에서 실패하고 수정 뒤 성공해야 하며, 고의 오류를 제품 코드에 넣을 필요가 없다. Next.js 코드를 추가해야 할 상황이 생기면 이번 범위에서 벗어난 것이다; AGENTS.md에 따라 설치된 Next 가이드를 먼저 읽고 별도 과제로 남긴다.

## 접근 비교 및 추정 근거
- 선택: 기존 globalIgnores에 두 경로 추가. 이미 사용하는 설정 표면만 활용하며 생성 경로가 기본값이라는 전제가 핵심이다(현재 Playwright 설정에서 별도 outputDir/outputFolder 없음).
- 대안: lint 대상을 소스 디렉터리 목록으로 바꾸기. 신규 소스 디렉터리 누락 위험이 있어 선택하지 않았다.
- 대안: Playwright 출력 위치를 이미 제외된 디렉터리로 옮기기. global-setup/teardown 데이터 경로와 보고서 이용 방식까지 바뀌어 선택하지 않았다.
- 현행 유지/실행 전 삭제: 변경 비용은 없지만 진단 증거가 사라지고 반복 작업이 남아 선택하지 않았다.
- 추정 방법: bottom-up, 준비·기준선 5–10분 + 설정 변경/API 검증 5–8분 + 전체 lint/마무리 5–7분 = 기본 15–25분. 알려진 불확실성(의존성 설치·설정 로딩 지연)에 contingency 5–10분, 총 20–35분. 설치 가능한 환경에서 45분 이내라는 판단의 신뢰는 중간이며 통계적 백분위가 아니다.
- management reserve는 배정하지 않음(새 범위에 자동 사용 금지). 설치 실패·관련 없는 lint 오류는 재추정 사유이지 보호 경로 수정 허가가 아니다. 과거 회차의 보고서 삭제 우회가 문제 근거이며 작업시간 실측은 없어 analogous 수치 교차검증은 불가하다.
- 추정의 범위·가정·작업 분해 기록 방식 참고: [GAO Cost Estimating and Assessment Guide](https://www.gao.gov/products/gao-20-195g). 분 단위 숫자는 GAO 수치가 아닌 정찰자의 작업별 판단이다.
- 적용 스킬: 로컬 headcount의 `pmo:estimating-and-contingency`, `technology:implementation-planning`, `technology:solution-exploration` SKILL.md를 직접 읽음. 전용 Skill/skills.list/skills.read 호출 도구는 노출되지 않았으며, 정본 경로 `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/{pmo,technology}/skills/`를 이용했다.
