- 과제: 빈 메타데이터의 줄 경계를 지켜 인접 정보 삭제와 잘못된 완료 보관을 막기 (가치 4 / 위험 2 / 작업량 M)
- 왜: `matchLine`·`setLine`·`touchPlan`의 `\s*`가 줄바꿈을 소비해 빈 `상태:` 다음의 `작성일:`을 상태로 읽거나 상태 변경 때 삭제하며, `archiveDonePlans`도 빈 상태 다음 줄의 `done`을 완료 상태로 오인한다. 메타데이터 읽기·편집·보관에 같은 한 줄 계약을 적용하면 사용자가 직접 편집한 목표/계획 파일의 정보를 보존하고 잘못된 보관을 막는다.
- 수용 기준:
  1) `matchLine`은 라벨이 없거나 값이 빈 문자열/공백/탭뿐이면 호출자가 준 fallback을 반환한다. 값은 해당 물리적 줄에서만 읽어 trim하며, 바로 다음 메타데이터·빈 줄·섹션 제목을 값으로 읽지 않는다. 기존 정상 값과 라벨 이스케이프 테스트는 유지한다.
  2) `setLine`은 빈 라벨도 기존 줄로 인식해 그 자리에서 한 번만 교체한다. 없는 라벨은 기존 우선순위(마지막 갱신 → 작성일)로 삽입하되 앵커 값이 비어 있어도 그 줄 바로 뒤에 삽입한다. 앵커 둘 다 없는 입력의 기존 무변경 계약은 유지한다.
  3) `touchPlan`은 빈 상태/마지막 갱신 줄을 제자리에서 갱신하고 인접 작성일·연결 목표·계획 파일·빈 줄·본문을 보존한다. status=null이면 상태는 변경하지 않는다. 마지막 갱신이 없고 작성일만 빈 값으로 존재할 때도 작성일 바로 뒤에 갱신 줄을 추가한다. 기존 누락 상태 삽입 정책은 유지한다.
  4) 실제 `planTemplate`/`goalTemplate`에서 라벨 값만 비운 입력에 편집 → `planMeta`/`parseGoal`/`lintPlan`/`lintGoal`을 적용한다. 빈 상태의 읽기 결과는 각 호출자의 fallback(draft, (unknown), 빈 값)에 맞고, 상태를 active로 갱신한 후 파서가 active로 읽으며 작성일과 다른 필드는 그대로여야 한다. LF/CRLF, 공백뿐인 값, 파일 끝 빈 라벨을 포함한다. 문자열 전체 비교 또는 변경 대상 외 줄 비교로 데이터 보존을 입증한다.
  5) 실제 임시 디렉터리에 `planTemplate` 기반 파일을 만들고 `archiveDonePlans`를 실행한다. `상태:\ndone` 및 CRLF 대응 입력은 보관하지 않고 원본 바이트를 보존하며, 실제 `상태: done`만 보관한다. `planMeta`가 읽는 상태와 보관 판정이 일치해야 한다. 기존 이름 충돌 보관 테스트는 계속 통과한다.
- 건드릴 파일:
  - `src/util/markdown.ts:matchLine,setLine,touchPlan` — 정규화 후 한 줄 안에서만 라벨/값을 읽고 바꾼다. 빈 값도 매칭하는 편집 패턴과 trim 후 fallback을 명시한다. setLine/touchPlan의 작성일·마지막 갱신 앵커도 함께 처리한다.
  - `src/features/plans.ts:archiveDonePlans` — 독립적인 `/^상태:\s*done$/m` 판정을 수정된 `matchLine(text, "상태") === "done"`으로 맞춘다. 대소문자 변환/상태 집합 확대/보관 경로 변경은 하지 않는다.
  - `tests/unit/markdown.test.ts` — 실제 템플릿 및 실제 소비 함수로 위 읽기·편집·린트 회귀 테스트를 추가한다. 필요하면 기존 plans import에 planMeta만 더한다.
  - `tests/unit/plans.test.ts` — 기존 실제 fs 임시 디렉터리 패턴으로 빈 상태 보관 방지 테스트를 추가한다.
- 검증 명령: 구현 워크트리에서 `npm ci`, `npx vitest run tests/unit/markdown.test.ts tests/unit/plans.test.ts`, 마지막으로 `npm run check`(typecheck + 전체 vitest + esbuild). package.json과 CI에 정의된 실제 명령이다. 정찰에서는 아래 별도 설치 경로로 원본 tests/unit 전체 실행: Git 탐색 경계 지정 후 11개 파일/82개 테스트 통과(11.31초); 최초 환경 실패와 원인은 profile.md에 기록한다. 루트 npm run check/Windows VSIX 검증은 정찰에서 실행하지 않았다.
- 위험과 피할 것: 프로덕션 파일 2개로 제한한다. 제목 headingTitle/제목 린트는 별도 계약이므로 이번에 합치지 않는다. 섹션 재조립·체크리스트 들여쓰기·advance 동작·정렬·활성 계획 포인터·provider·checkpoints·vendor·릴리즈/워크플로·버전 파일은 건드리지 않는다. 정규식의 `.+`를 유지한 채 `\s`만 바꾸면 빈 기존 줄을 인식하지 못한다. fallback은 값을 trim한 뒤 적용한다. 동일 필드의 별도 파서인 archiveDonePlans를 빠뜨리지 않는다. 가짜 Task/문서/fs를 만들지 말고 기존 vscode import 스텁 아래 실제 함수·실제 템플릿·실제 fs로 검증한다.
- 차선 후보: 빈 제목이 다음 상태 줄을 제목으로 읽고 제목 린트도 통과하는 문제 — `markdown.ts:headingTitle`과 `goal-lint.ts:lintGoal,lintPlan`을 함께 수정하는 별도 과제(가치 3 / 위험 1 / 작업량 S). 첫 과제의 실제 재현이 구현 환경에서 성립하지 않을 때만 택한다.

확인한 근거 (main@43fd7a1, 1.4.5)
- Node v22.23.1에서 실제 markdown.ts를 직접 import: `상태:\n작성일: original\n마지막 갱신: old\n`의 matchLine 결과는 `작성일: original`; setLine/touchPlan 결과에는 작성일 줄이 없다.
- esbuild로 실제 plans/goals/goal-lint와 저장소의 기존 vscode-stub만 연결해 실행: 실제 planTemplate/goalTemplate에서 상태를 비우면 LF/CRLF 모두 planMeta.status와 parseGoal.status가 `작성일: 2026-09-29 12:00:00 KST`가 된다. setLine/touchPlan 뒤 작성일 존재 검사는 양쪽 모두 false.
- 같은 실행에서 실제 디렉터리에 `done.md`(정상 done), `empty.md`(빈 상태 다음 독립 done 줄)를 저장하고 archiveDonePlans 호출: LF/CRLF 모두 두 파일을 보관했다. 정규식 복제나 fs 대역으로 얻은 결과가 아니다.
- 실제 UI 명령 배선: plans.ts의 상태 변경은 touchPlan, 목표 연결은 setLine 두 번 → touchPlan, 완료 보관 명령은 archiveDonePlans를 호출한다. goals.ts의 목표 상태 변경도 touchPlan을 쓴다. VS Code Extension Host 자체는 미실행.

대안과 선택
- 선택: 기존 한 줄 메타데이터 도우미와 보관 소비 경로만 수정. 저장 형식/명령 API를 유지하며 재현된 삭제를 막는다.
- 전체 라인 AST/새 파서 도입: 향후 확장에는 유리하나 섹션·제목 계약까지 넓어져 45분 과제에 맞지 않는다.
- 템플릿에서 빈 값 금지 또는 린트 경고만 추가: 사용자 수동 편집을 막지 못하고 기존 파일을 수정할 때 삭제가 남으므로 부족하다.
- 아무것도 하지 않기: 비용은 없지만 실제 데이터 삭제/오보관 재현이 있어 채택하지 않는다.
- 가장 큰 가정: 이 저장소의 `label: value`는 한 물리적 줄이라는 계약. 템플릿과 소비 코드가 이를 지지한다. 여러 줄 메타데이터 지원 요구는 확인되지 않았다.

실행 단계 (구현자 진행 상태를 각 항목에 기록; 사람 승인 체크포인트 없음)
1. [대기] 기존 집중 테스트 기준선과 위 재현을 확인한다. 증거: `npx vitest run tests/unit/markdown.test.ts tests/unit/plans.test.ts` 및 새 회귀 단정의 수정 전 실패. 재현이 다르면 계획을 먼저 갱신한다.
2. [대기] markdown.ts 수정과 metadata 회귀 테스트를 한 단위로 완료한다. 증거: `npx vitest run tests/unit/markdown.test.ts`. 통과 확인 후 다음 단계.
3. [대기] archiveDonePlans를 같은 상태 읽기에 연결하고 실제 파일 테스트 추가. 증거: `npx vitest run tests/unit/markdown.test.ts tests/unit/plans.test.ts`.
4. [대기] `npm run check`로 전체 회귀·타입·빌드를 확인한다. 외부 VSIX 자산 경고와 실제 실패를 구분하고 새 범위를 끌어들이지 않는다.

작업량 추정 (pmo:estimating-and-contingency 적용)
- 바텀업: 재현/테스트 설계 5–7분, metadata 수정·테스트 10–13분, 보관 연결·실제 fs 테스트 5–7분, 전체 검증/리뷰 5–8분 = 기본 25–35분.
- 알려진 불확실성 예비 5–8분: 빈 앵커·CRLF 회귀 또는 의존성 설치 변동. 합계 30–43분, 45분 안 완료에 중간 정도 확신(측정된 확률 아님).
- 별도 관리 예비: 0분 배정; 새로운 상태 형식/제목 처리 요구는 다음 과제로 보류한다. 기본 추정에 예비를 중복 가산하지 않았다.
- 유사 사례 교차확인: 이전 1.4.5도 공용 markdown 도우미+소비 경로 2파일의 국소 수정으로 채택됐다. 과거 실제 소요시간 자료가 없으므로 정량 유사 추정은 미확인이고 이번 범위의 크기만 비교했다.
- 전제: 기존 npm lock 설치 가능, 외부 모델/VS Code 호스트 없이 단위 테스트 가능. 범위에서 UI·마이그레이션·릴리즈 수정 제외. 추정 입력은 정찰의 코드 관찰과 제공된 회차 기록이다.

정찰에서 실제 실행한 전체 테스트 명령 (출력 폴더에만 설치·캐시·임시 파일 생성)
```bash
GIT_CEILING_DIRECTORIES=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-29-123226-vibe-code-improve/validation/tmp TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-29-123226-vibe-code-improve/validation/tmp node /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-29-123226-vibe-code-improve/validation/node_modules/vitest/vitest.mjs run --config /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-29-123226-vibe-code-improve/validation/vitest.config.mjs
```
적용 스킬: `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md`, 같은 headcount의 `plugins/technology/skills/implementation-planning/SKILL.md`, `plugins/technology/skills/solution-exploration/SKILL.md`. 비용 추정은 외부 기준 수치를 인용하지 않은 이 과제의 작업 분해 추정이다.
