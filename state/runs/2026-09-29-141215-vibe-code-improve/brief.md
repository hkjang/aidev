- 과제: 수정 과제 — PR #6의 GitHub Actions 실행 차단 해소와 동일 커밋 검증 완료 (가치 5 / 위험 1 / 작업량 S)
- 왜: CI run 36518589343은 코드 검사 전에 계정 결제/사용 한도 문제로 시작하지 못했으며 check job의 steps=[], runner_id=0이고 Windows package job은 skipped다. 불필요한 코드 변경 없이 기존 수정 PR을 보존하고 실제 검증을 완료해야 같은 실패가 반복되지 않는다.
- 수용 기준: 1) ci-annotations.json의 결제/사용 한도 annotation을 원인으로 기록하고 실행 차단 해소 전에는 완료·릴리즈 성공으로 판정하지 않는다. 2) PR head 1855dcb57ee750819629a7142992f43839c7d233에서 npm ci, npm run check, 두 번들의 node --check를 성공시키고 원장에 커밋·Node 버전·결과를 기록한다. 3) 차단 해소 뒤 같은 PR CI의 check가 실제 실행되어 성공하고, 패키징 단계도 Package VSIX/Verify package/Upload VSIX가 skipped가 아닌 success인지 확인한다; 자산 부족·계정 문제로 불가능하면 정확한 외부 blocker를 남긴다.
- 건드릴 파일: 프로덕션 0개, 저장소 변경 불필요. 이번 회차 journal.md/ledger-entry.md에 수정 과제 및 검증 증거만 기록한다. 읽은 관련 경로: .github/workflows/ci.yml(check/package), scripts/build.mjs(stageCore/warnMissingRuntimeAssets), scripts/package-vsix.ps1(Assert-ChildPath 및 패키징 본문), scripts/verify-package.ps1(Fail 및 필수 파일·번들 검사), scripts/restore-dist-assets.mjs(readCentralDirectory/entryData), tests/unit/markdown.test.ts, tests/unit/plans.test.ts, tests/unit/checkpoints.test.ts. 기능 수정 2파일과 회귀 테스트 2파일은 PR #6에 이미 있으므로 재구현하지 않는다.
- 검증 명령: PR #6의 정확한 head에서 `npm ci`, `npm run check`, `node --check dist/extension.js`, `node --check dist/extension.core.js`. Windows 실제 패키징은 기존 VSIX 자산 확보 후 `node scripts/restore-dist-assets.mjs --from <실제-vsix-경로>`, `npm run build`, `powershell -ExecutionPolicy Bypass -File scripts/package-vsix.ps1`, `powershell -ExecutionPolicy Bypass -File scripts/verify-package.ps1`. 마지막 명령들은 정찰에서 미실행이며 Windows/릴리즈 자산이 필요하다.
- 위험과 피할 것: .github/workflows/ci.yml, 보호 검사, needs/if/continue-on-error, package 검증을 완화하지 않는다. 결제·한도 변경은 외부 계정 권한이 필요한 경계이며 자동으로 지출 한도를 늘리지 않는다. main@43fd7a1에는 PR #6이 아직 없으므로 main만 검사하고 해당 PR이 통과했다고 쓰지 않는다. 무의미한 빈 커밋·동일 수정 중복 PR·메타데이터 재구현 금지. 이번 정찰은 수정·커밋·CI 재실행을 하지 않았다.
- 차선 후보: 같은 PR head의 로컬 검사가 실제 실패하는 경우에만 재현된 실패 한 건의 최소 수정으로 계획을 갱신한다. 결제 문제를 우회할 새 워크플로·러너 전환이나 별도 기능 과제는 선택하지 않는다.

확인한 근거와 설명 정정:
- https://github.com/hkjang/vibe-code/actions/runs/36518589343/job/109246301265 : annotation은 “The job was not started because recent account payments have failed or your spending limit needs to be increased.”라고 명시한다. 어느 결제수단/한도인지는 미확인.
- GitHub API 전체 25 runs 중 failure는 위 1건, run_attempt=1. “같은 이유 두 번 실패”라는 자동 배정 전제는 이번 조회에서 확인되지 않았다. 실패한 스크립트/테스트 단계 자체가 없다.
- 직전 main run 36422197288은 check success이나 Windows Package VSIX/Verify package/SHA256/Upload VSIX가 전부 skipped였다. 녹색 workflow만으로 실제 릴리즈 검증 완료를 주장하지 않는다. 건너뛴 자산 원인은 별도 미확인으로 남긴다.
- 보관 증거: ci-evidence.json, ci-annotations.json. Linux gh는 인증 없음; 연결 도구로 run/jobs 조회, Windows의 기존 gh 인증으로 annotation 읽기 성공. 비밀·전역 설정 변경 없음.

실행 계획 (아직 구현 작업 미착수; 각 단계 상태를 이 파일에 갱신):
1. [대기] `gh api repos/hkjang/vibe-code/check-runs/109246301265/annotations`로 원인을 재확인하고 PR #6 head를 확인한다. 증거가 달라지면 먼저 계획 갱신. 사람 검토 없이 읽기 진행.
2. [대기] PR head에서 위 로컬 check 명령을 실행해 코드 문제와 실행 차단을 분리한다. 실제 fs/Git 및 기존 프로덕션 템플릿 회귀를 유지한다. 실패하면 해당 로그에 근거한 한 건만 재계획. 사람 검토 없이 검증 진행.
3. [외부 blocker] GitHub Billing & plans의 결제/한도 문제는 계정 소유자의 조치가 필요하다. 권한과 해결 증거가 없으면 비용 설정을 변경하거나 재시도를 반복하지 말고 blocked 사유를 원장에 남긴다. 이 단계 완료 전 원격 성공 단계로 진행 금지.
4. [대기] 차단 해소 및 릴리즈 역할의 실행 권한 확인 후 기존 run 재실행(`gh run rerun 36518589343 --repo hkjang/vibe-code`; 관찰 `gh run watch 36518589343 --repo hkjang/vibe-code --exit-status`). check와 실제 Windows package 검증을 확인하고 종료한다. 정찰에는 재실행 권한 없음. 자산 문제가 나타나면 새 증거로 과제 범위를 갱신하며 skip 통과 금지.

대안 비교:
- 선택: 기존 PR 유지 + 로컬 동일 검사 + 계정 차단 해소. 코드 변경 0, 실제 원인과 직접 대응한다.
- 소스/테스트 수정: 재현 실패가 없어 현재는 근거 없음. 로컬 실패가 나타날 때만 차선으로 유효하다.
- 러너/워크플로 재설계: 외부 실행 제한을 우회하고 범위를 늘리므로 제외. 아무 조치 없이 반복 실행도 원인을 해결하지 못하므로 제외.
- 핵심 가정: annotation이 현재 차단 원인을 정확히 반영한다. 소유자 조치 후 새로운 실행 실패가 나오면 다른 사건으로 재진단한다.

추정 근거와 여유:
- bottom-up: 증거 확인 3–5분 + 로컬 검사 5–12분 + 결과 기록 3–5분 = 11–22분, 알려진 설치·파일시스템 지연 contingency 5–10분을 별도로 합쳐 활동 시간 16–32분. 경험적 신뢰도는 중간이며 통계적 확률 추정은 아니다.
- 계정 소유자 조치 대기·원격 큐·실제 패키징 자산 복원 시간은 상한 미확인이므로 45분 내 외부 CI 복구는 보장할 수 없다. 작업량 S는 에이전트의 진단/검증 작업 기준이다. 관리 예비비(새 범위 대응)는 배정하지 않았고 별도 의사결정 사항이다.
- 과거 기록은 검사 성공 수와 결과는 있지만 신뢰할 실행 소요 분포가 없어 유사 추정으로 시간 범위를 독립 검증할 수 없다.
- 적용 스킬: /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md, technology/skills/implementation-planning/SKILL.md, technology/skills/solution-exploration/SKILL.md(뒤 둘도 같은 headcount/plugins 루트). Skill 호출 도구가 없어 실제 SKILL.md 원본을 읽어 적용했다. 추정 스킬의 references/sources.md도 읽었고 외부 비용·확률 주장은 하지 않았다.

정찰 최종 로컬 검증 (코드 변경 없이 실행):
- 1855dcb 원본 src/tests/scripts/vendor와 package/config를 git archive로 run/validation/check에 추출했다. npm ci → npm run check → 두 번들 node --check 모두 exit 0. Node 22.23.1과 CI major 동일 Node 20.20.2 모두 11 files / 146 tests 통과했다. Node 20의 전체 명령 로그는 validation/node20-check.log, 최초 Node 22 로그는 validation/check.log다.
- 임시 fs/Git 테스트가 허용 디렉터리 밖에 쓰거나 상위 aidev 저장소를 발견하지 않도록 TMPDIR=run/validation/tmp, GIT_CEILING_DIRECTORIES=동일 경로를 명령 환경에만 지정했다. 코드/테스트/검증 강도는 변경하지 않았다.
- 실제 사용한 Node 20 명령: validation/check를 cwd로 `npm exec --yes --cache ../npm-cache --package=node@20 -- sh -c 'node --version && npm ci --cache ../npm-cache && npm run check && node --check dist/extension.js && node --check dist/extension.core.js'` (위 TMPDIR/GIT_CEILING_DIRECTORIES와 함께). 실제 실행에서는 cache 절대 경로를 사용했다.
- 원격 CI 차단은 그대로이며 Windows 실제 VSIX 검증은 미실행. 계정 결제 원인은 소스 변경으로 고칠 수 없으므로 수정 성공이나 릴리즈 성공으로 기록하지 않는다.
