# 과제서 — 2026-09-22 19:04 회차 (aiportal-front-admin)

- 과제: 수정 과제 — 릴리즈 실패의 저장소 측 원인 제거: 릴리즈 관례 문서·CHANGELOG·버전 단일화·버전 일치 검사 추가 (가치 4 / 위험 2 / 작업량 M)
- 왜: 릴리즈 에이전트는 `release-prompt.md` 절차 1~5에 따라 태그·릴리즈 커밋·CHANGELOG·릴리즈 노트 관례를 **저장소에서** 찾는데, 이 저장소에는 그중 어느 것도 없고 버전 파일만 둘(루트 `package.json` 0.0.0, `upgrade/admin-v2/package.json` 0.1.0, 각 lock 포함)이라 "관례 없음 → skipped" 조건에도 못 들어가 두 번 연속 failed 로 끝났다. 저장소 안에 관례를 한 번 문서로 확정하고 버전 값을 한 곳으로 맞추면, 다음 릴리즈 실행이 절차 1·2·3의 입력을 모두 얻어 진행할 수 있다(스킬 도구 미전달 문제는 외부 러너 소유이며 이 과제 범위 밖 — 아래 "위험" 참고).
- 실패 단계 확인: 이 저장소에는 `.github/workflows` 가 없고(`.gitlab-ci.yml` 은 배포 전용) 실패한 "워크플로"는 러너의 릴리즈 단계(`/mnt/c/Users/USER/projects/aidev/release-prompt.md`)다. 실패 사유 원문 중 저장소가 고칠 수 있는 부분은 "태그·릴리즈 커밋·버전 증가 이력·CHANGELOG/릴리즈 노트 없음, 버전 파일은 있어 skipped 조건 불성립, 다음 버전·태그·노트 관례를 정할 근거 없음" 이다. 이전 6회차 과제서가 모두 외부 러너 수정(외부 편집 금지로 no-change)을 겨냥했으므로 이번에는 **저장소 안에서 끝나는 쪽만** 다룬다.

## 수용 기준
1) `docs/RELEASE.md` 가 존재하고 다음을 한국어로 명시한다: 릴리즈 단위는 저장소 전체 1개 버전; 버전 원본은 `upgrade/admin-v2/package.json` 이며 루트 `package.json`·두 `package-lock.json`(최상위 `version` 과 `packages[""].version`)이 같은 값을 가져야 함; 태그 형식 `v<version>` 주석 태그; 릴리즈 커밋 메시지 `chore(release): v<version>`; 증가 규칙은 patch(현재 이력이 fix 만이므로), 첫 릴리즈는 `v0.1.1`; 릴리즈 노트는 `CHANGELOG.md` 의 해당 버전 절을 그대로 GitHub Release 본문으로 사용(`github_release: true`, 자산 없음); 릴리즈 전 검사는 `node upgrade/admin-v2/scripts/check-version.mjs` 와 `upgrade/admin-v2` 의 `npm run verify`.
2) 루트 `CHANGELOG.md` 가 Keep a Changelog 양식(한국어)으로 존재하고 `## [Unreleased]` 아래에 최근 병합 PR #18~#21 의 `fix(admin-v2)` 4건이 `### Fixed` 항목으로 적혀 있다(커밋 제목 그대로, `git log --oneline -8` 로 확인 가능).
3) 루트 `package.json` 과 `package-lock.json` 의 버전이 `0.1.0` 으로 맞춰지고(두 lock 의 `version` 두 곳 모두), `node upgrade/admin-v2/scripts/check-version.mjs` 가 4개 파일 6개 값을 읽어 모두 같으면 exit 0·다르면 파일과 값을 찍고 exit 1 한다. 일부러 한 곳만 바꿔 exit 1 이 나는 것도 확인해 기록한다(같은 값을 읽는 경로 전부를 한 검사가 본다는 점이 핵심 — 운영자 규칙).
4) `upgrade/admin-v2/package.json` scripts 에 `"check:version": "node scripts/check-version.mjs"` 가 추가되고, 기존 `verify`/`build` 스크립트는 바꾸지 않는다.
5) `upgrade/admin-v2` 에서 `npm ci && npm run verify` 가 기존과 같이 통과한다(변경 전에도 통과해야 하므로 먼저 한 번 돌려 기준선을 남길 것; node_modules 는 현재 비어 있음 — 이번 정찰에서 `ls upgrade/admin-v2/node_modules | wc -l` = 0 확인).

## 건드릴 파일
- `docs/RELEASE.md` (신규) — 위 관례. `docs/INDEX.md` 목록에 한 줄 추가.
- `CHANGELOG.md` (신규, 루트) — `[Unreleased]` 절만. 과거 버전 절을 지어내지 말 것.
- `package.json`, `package-lock.json` (루트) — `"version": "0.0.0"` → `"0.1.0"` (lock 은 3행·9행 두 곳). 다른 필드는 손대지 말 것(특히 `crypto-js` file: 의존성 — 별개 보류 아이디어).
- `upgrade/admin-v2/scripts/check-version.mjs` (신규) — `validate-runtime-config.mjs` 와 같은 스타일(node:fs/path, `fail()` 로 `[check-version]` 접두 메시지 후 exit 1, 한국어 주석). 경로는 `import.meta.url` 기준으로 잡아 cwd 무관하게 동작.
- `upgrade/admin-v2/package.json` — scripts 에 `check:version` 추가만.
- `README.md` 루트 — "릴리즈" 소절 2~3줄로 `docs/RELEASE.md` 링크(선택, 짧게).

## 검증 명령 (이 저장소에서 실제로 도는 것)
```
node upgrade/admin-v2/scripts/check-version.mjs            # exit 0 기대 (의존성 설치 불필요)
node -e "const p=require('./package.json');console.log(p.version)"   # 0.1.0
cd upgrade/admin-v2 && npm ci && npm run verify              # typecheck + vitest, 변경 전후 동일 통과
git diff --stat                                              # 위 목록 밖 파일이 없어야 함
```
- 수용 기준 3의 음성 확인: 임시로 루트 package.json 버전을 0.1.9 로 바꿔 `node upgrade/admin-v2/scripts/check-version.mjs` 가 exit 1 인지 보고 되돌린다(커밋에 포함 금지).

## 위험과 피할 것
- **외부 러너(`/mnt/c/Users/USER/projects/aidev/bin/run.sh`, `release-prompt.md`, headcount plugins)는 절대 편집하지 말 것.** 스킬 도구 미전달(`marketing:product-launch`, `technology:release-and-deployment` 를 찾지 못함) 은 러너 소유 결함으로 이 과제로 없어지지 않는다. 과제 완료 후에도 다음 릴리즈가 그 이유로 실패할 수 있음을 원장에 그대로 적을 것 — 저장소 측 원인 제거만이 이번 성과다.
- 태그를 만들거나 버전을 0.1.1 로 올리지 말 것. 태그·릴리즈 커밋은 릴리즈 단계가 만든다. 이번 PR 은 관례 확정 + 버전 값 정합만 한다.
- `.gitlab-ci.yml`, `upgrade/admin-v2/deploy/`, `src/` 인증·라우터, `verify`/`build` 스크립트 본문은 건드리지 말 것.
- 루트 앱은 테스트가 없고 `crypto-js` file: 의존성 때문에 `npm ci` 가 실패할 수 있다(tarball 미존재, 이전 프로필). 루트에서 npm 설치를 시도하지 말 것 — 검증은 `check-version.mjs` 와 admin-v2 `verify` 로 충분하다.
- 커밋 메시지는 기존 관례대로 한국어 `chore(release-docs): 릴리즈 관례와 버전 일치 검사를 추가한다` 류. 여러 커밋으로 나누지 말 것.
- 문서에 확인하지 않은 사실(원격 태그·GitHub Release 존재 여부)을 쓰지 말 것 — 이번 정찰에서 로컬 `git tag` 는 비어 있음을 확인했고 원격은 미확인.

## 산정(기초)
- 분해: RELEASE.md 15분 / CHANGELOG 10분 / 버전 동기화 5분 / check-version.mjs+스크립트 15분 / npm ci+verify 기준선·재검증 10~20분. 합 55~65분, 8/10 확률로 45~75분 안. 상방 위험은 `npm ci` 네트워크(레지스트리 접근 불가 시 수용 기준 5 를 "미실행·사유" 로 기록하고 나머지는 완료).
- 제외: 러너 수정, 실제 릴리즈 수행, 원격 조회, 루트 앱 빌드.

## 차선 후보
- 「AdvancedPolicyView 저장·새로고침이 미저장 편집을 덮어씀」(3/2/M) — 1순위가 성립하지 않을 때만. 단 1순위가 성립하지 않는 경우는 "저장소에 이미 릴리즈 관례가 있음" 뿐이며 이번 정찰에서 없음을 확인했다.
