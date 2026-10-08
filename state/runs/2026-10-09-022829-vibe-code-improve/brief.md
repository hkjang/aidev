- 과제: 공유 폴더 업데이트 탐색에서 VSIX 이름의 디렉터리를 설치 후보에서 제외 (가치 3 / 위험 1 / 작업량 S)
- 왜: `latestVsixVersion`은 디렉터리 엔트리 이름만 비교해 `vibe-code-99.0.0.vsix/`를 진짜 `vibe-code-1.10.1.vsix`보다 최신 설치 파일로 선택하고, `checkForUpdates`는 그 디렉터리 경로를 설치 명령에 넘길 수 있다. 실제 파일만 후보로 남기면 잘못된 업데이트 알림과 유효한 패키지가 가려지는 문제를 막는다.
- 수용 기준:
  1) 실제 `vibe-code-1.10.1.vsix` 파일과 `vibe-code-99.0.0.vsix` 디렉터리가 함께 있으면 `latestVsixVersion`은 `1.10.1`을 반환한다. `checkForUpdates`의 기존 경로 조합과 같은 `path.join(dir, 'vibe-code-' + version + '.vsix')` 결과가 실제 파일임을 검사한다.
  2) VSIX 이름의 디렉터리만 있거나 폴더가 비어 있으면 `undefined`. 파일 후보가 여러 개면 기존 숫자 비교대로 `1.10.0 > 1.9.9 > 1.2.0`이며 무관한 이름·다른 확장자는 무시한다.
  3) 실제 파일을 가리키는 심볼릭 링크는 계속 후보로 허용하고, 디렉터리 링크·깨진 링크는 제외한다. 개별 엔트리 stat 실패가 다른 정상 후보까지 버리게 하지 않는다. 최상위 `readdirSync(dir)` 실패는 기존처럼 호출부 catch로 넘긴다.
  4) 신규 테스트는 실제 임시 디렉터리/파일과 기존 export `latestVsixVersion`을 사용한다. 수정 전에 기준 1·2가 값 불일치로 실패하고, 수정 후 통과하며 프로덕션 변경만 되돌려도 같은 실패가 나야 한다. fs 대역이나 소스 문자열 검사는 쓰지 않는다.
- 건드릴 파일:
  - `src/features/update-check.ts:17 latestVsixVersion` — 기존 파일명 매칭을 통과한 엔트리만 `fs.statSync(path.join(dir, name)).isFile()`로 확인한 뒤 기존 버전 비교에 넣는다. 엔트리별 stat 실패는 제외한다. `statSync`를 택하는 이유는 파일 심볼릭 링크를 기존처럼 지원하기 위해서다. JSDoc에 파일 후보 규칙만 보완한다.
  - `tests/unit/update-check.test.ts` (신규) — 위 실제 fs 회귀 테스트. `afterEach`/finally로 자기 임시 폴더만 정리한다. 기존 `tests/unit/semver.test.ts`와 공유 `vscode-stub.ts`는 변경하지 않는다.
  - 프로덕션 1개 + 테스트 1개. 변경 파일 상한 6개에 여유가 크며 다른 결함은 포함하지 않는다.
- 검증 명령:
  - 구현 워크트리: Node `20.19.2` 확인 → 의존성이 없으면 `npm ci` → `npx vitest run tests/unit/update-check.test.ts tests/unit/semver.test.ts` → `npm run check` → `node --check dist/extension.js` → `node --check dist/extension.core.js`.
  - 이 저장소의 실제 scripts: `check = typecheck && test && build`, `test = vitest run`. 신규 테스트 경로는 구현 후 생긴다. 정찰에서는 저장소 쓰기 금지 때문에 npm ci/build를 실행하지 않았다.
  - 정찰에서 실제 실행한 기존 테스트 명령(쓰기 위치를 회차 validation으로 제한): `/tmp/node2019/bin/node /mnt/c/Users/USER/projects/vibe-code/node_modules/vitest/vitest.mjs run --config /mnt/c/Users/USER/projects/aidev/state/runs/2026-10-09-022829-vibe-code-improve/validation/vitest.config.mjs`. 설정은 현재 워크트리의 원본 테스트와 원본 VS Code 스텁을 읽고 설치된 Vitest만 원 checkout에서 가져온다. Node20.19.2 / Vitest3.2.7에서 19 files / 152 tests 통과(3.50초), `validation/baseline-node20.log`에 보관했다. Node22.23.1에서도 같은 152 tests 통과했다.
  - 결함 재현: `/tmp/node2019/bin/node /mnt/c/Users/USER/projects/aidev/state/runs/2026-10-09-022829-vibe-code-improve/validation/reproduce.cjs`. 현재 exit 1은 의도한 AssertionError(`99.0.0 !== 1.10.1`)다. 이 스크립트는 실제 production 모듈을 esbuild로 번들링해 실행하며 파일/산출물은 회차 validation 안에만 쓴다. `validation/reproduce.log`에 혼합/디렉터리만 있는 경우의 잘못된 반환값과 선택 경로가 디렉터리라는 사실을 남겼다.
- 위험과 피할 것: 패키지 ZIP 내용 검증, SemVer/프리릴리즈 규칙, 업데이트 주기·알림·globalState, 설치 명령 자체, 공유 경로 설정, 대규모 비동기화는 범위 밖. `.github/workflows/`, `vendor/`, `src/core/`, `src/features/mcp/` OAuth·SecretStorage, provider, 패키징 스크립트는 변경하지 않는다. 파일 검사와 실제 설치 사이 TOCTOU를 해결한다고 주장하지 않는다. 파일 링크 보존을 위해 Dirent.isFile만 쓰거나 lstat만 쓰는 접근을 피한다. Windows 링크 생성 권한이 없으면 링크 테스트만 명시적 사유로 skip하고 디렉터리/일반 파일 핵심 테스트는 항상 수행한다. 감사 details에 경로·검출 원문 등 새 정보를 추가하지 않는다.
- 차선 후보: 하위 디렉터리 Python `test_*.py` 직접 추천 (가치 3 / 위험 1 / S) — `src/features/verification.ts:143 TEST_FILE_RE`의 `^test_`가 `pkg/test_util.py`를 인식하지 않아 `suggestTestCommands`는 직접 명령 대신 전체 `pytest`를 추천한다. 경로 시작 또는 `/` 뒤 basename의 `test_`만 좁게 인식하고 `tests/unit/verification.test.ts`에서 최종 명령 문자열을 검증한다. 과거 NUL Git 파서 수정과 별개이며 gitChangedFiles·셸 인용은 절대 함께 고치지 않는다. 1순위 근거가 달라졌을 때만 과제서를 수정하고 이 한 과제로 교체한다.

근거와 한계 (정찰 완료, main@6eaede8 / v1.10.0)

- `update-check.ts:17-24`의 이름 전용 선택 → `:41` 호출 → `:43` 경로 구성 → `:47` 설치 명령, `activation.ts:65` 활성화 배선까지 읽었다. 실제 VS Code 알림/설치 UI는 실행하지 않았다. 결함 재현은 실제 파일시스템을 읽는 프로덕션 선택 함수부터 설치 입력 경로까지이며, 사용자 사고 발생 빈도는 미확인이다.
- `scripts/package-vsix.ps1:184`의 `$package.name-$package.version.vsix`, package.json의 name=`vibe-code`가 탐색 정규식과 일치한다. 기존 보류 항목의 '패키징 파일명 불일치'는 기각했으며 이번 일은 다른 파일 종류 검증 결함이다.
- `git log --all -- src/features/update-check.ts`는 소스 복원 커밋 `2c13ffb` 하나였다. 제공된 과거 개선 기록에도 이 수정은 없으며 재제출 대상과 겹치지 않는다. 원격 PR 상태는 조회하지 않았고 로컬 이력을 원격 머지 판정으로 쓰지 않는다.

대안 비교와 선택

| 접근 | 비용/제약 | 판단 |
|---|---|---|
| 탐색 시 파일 여부 검사 | 프로덕션 1파일, 실제 fs 테스트 가능, 기존 링크 지원 보존 필요 | 선택. 잘못된 후보가 정상 버전을 가리는 원인을 제거 |
| 설치 직전에만 디렉터리 거부 | 알림/설치 UI 테스트 필요, 정상 낮은 버전은 계속 가려짐 | 보류. 선택 결과가 잘못되는 문제를 남김 |
| VSIX manifest까지 검사하는 별도 인덱서 | 압축 처리·의존성/성능·네트워크 공유폴더 계약 필요 | 큰 배포 관리 기능이 요구될 때 재검토, 이번 45분 범위 밖 |
| 폴더 정리를 사용자에게 맡김 | 코드 비용 없음, 같은 상황이 다시 생김 | 재현한 경계 결함을 작은 수정으로 제거할 수 있어 미선택 |

핵심 가정은 공유/로컬 업데이트 디렉터리에 VSIX 외 엔트리가 공존할 수 있다는 것이다. 재현 가능한 입력이지만 실제 사용자 빈도는 미확인이라 가치는 3으로 제한한다.

구현 순서와 체크포인트

1. [pending] 기준선: Node 버전·HEAD 확인, 필요한 의존성 설치, `npx vitest run tests/unit/semver.test.ts` 실행. 성공하면 진행한다. 사람 승인 체크포인트 없음.
2. [pending] 신규 실제 fs 테스트와 `latestVsixVersion`의 파일 판별을 한 단위로 작업한다. 먼저 현재 코드에서 신규 집중 테스트의 AssertionError를 증거로 보관하고, 수정 후 위 집중 명령이 전부 통과하면 단계 완료. 프로덕션 변경만 잠깐 되돌려 같은 값 불일치가 재발하는지 확인 후 복원하고 재검증한다. 깨진 상태를 완료 체크포인트로 남기지 않는다. 사람 승인 없음.
3. [pending] `npm run check`와 두 번들 문법 검사를 실행하고 최종 diff를 확인한다. 기존 기대값·공유 스텁·보호 경로가 바뀌지 않은 상태로 인계한다. 근거가 달라지면 자동 확장하지 말고 brief와 journal에 수정 이유를 먼저 기록한다. 이후 비평/릴리즈 판정은 러너 절차를 따른다.

작업량 추정과 예비 시간

- 상향식: 환경·기준선 5–8분, fs 회귀 테스트 8–10분, 작은 필터 수정 3–5분, 인과 재검증·전체 검사·기록 8–12분 = 기본 24–35분.
- 알려진 변동에 대한 contingency 5–10분은 의존성 준비와 Windows 링크 권한 대응에만 배정한다. 총 29–45분, 주관적 신뢰도 중간(실제 시간 표본이나 통계적 P80이 아님). 추가 기능용 management reserve는 0분으로 이번 범위에 포함하지 않는다.
- 추정 근거는 직접 읽은 단일 함수와 실제 fs 재현이다. 과거 tickCriteria 회차도 1 production+1 test였다는 유사 규모 비교는 가능하지만 소요시간 기록이 없어 별도 시간 추정치로 꾸며내지 않는다. 의존성 준비가 10분을 넘으면 남은 시간을 재산정하고 외부 장애를 코드 수정으로 우회하지 않는다.
- 포함/제외·작업 분해·가정·위험·실측 갱신을 명시하는 방식은 [GAO Cost Estimating and Assessment Guide](https://www.gao.gov/products/gao-20-195g)의 추정 절차를 참고했다. 분 단위 숫자 자체는 이 정찰의 판단이다.

적용 스킬: 전용 Skill 도구가 제공되지 않아 `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/` 아래 `pmo/skills/estimating-and-contingency/SKILL.md`(+ references/sources.md), `technology/skills/implementation-planning/SKILL.md`, `technology/skills/solution-exploration/SKILL.md`를 직접 읽었다. 요청된 비교·실행 순서·검증·체크포인트·추정 근거를 위에 반영했다.
