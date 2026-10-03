# 회차 노트 2026-10-03-181738-ox-arena-improve — ox-arena
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 18:17] base pinned — main@f68a6fa
- [러너 18:17] autonomy release — 

## 정찰 노트
- base(main@f68a6fa)에는 2026-10-02 의 vitest/phase.ts 가 **없다**(package.json·src/lib 직접 확인, review-pending). 그래서 "테스트를 써서 증명" 이 필요한 후보(탈락 분모, counts.none, 폴링 간격)는 전부 vitest 재추가 → 충돌을 부른다. 그걸 피하면서 가치가 가장 큰 1파일 과제가 ci.yml 이라 골랐다.
- 탈락 모드 분모 불일치(가치 4)를 제친 이유: 조사해 보니 미응답자도 그 라운드에 탈락해 '참가 자격자' 집합이 클라이언트에 남지 않는다 → 클라이언트만으로는 못 고치고 admin_reveal(보호 경로, 로컬 SQL 검증 수단 없음)을 건드려야 한다.
- 추측으로 적은 것: `npm ci && npm run build` 기준선이 녹색인지 **확인하지 못했다**(샌드박스에서 npm 승인 거부, node_modules 없음). 구현자는 반드시 먼저 기준선을 돌릴 것. `npm run test --if-present` 의 무스크립트 동작도 이 환경에서 실행 검증은 못 했다.
- 구현자가 조심할 것: pages.yml 은 0줄 diff 유지(유일한 릴리즈 경로), permissions 는 contents:read 로만, vitest/eslint/의존성 업그레이드를 끼워 넣지 말 것.
- 이 과제의 진짜 증거는 PR 에서 ci.yml 이 실제로 녹색으로 도는 것이다 — `gh pr checks` 결론을 반드시 남길 것(YAML 문자열 검사는 증거가 아니다).
- [러너 18:22] scout done — PR 검증용 CI 워크플로 추가 (`.github/workflows/ci.yml` 신규 1파일) (가치 4 / 위험 1 / 작업량 S)

## 구현 노트
- 무엇/왜: 신규 `.github/workflows/ci.yml` 1파일(17줄)만 추가. `pull_request` + `push:[main]` 에서 `npm ci` → `npm run test --if-present` → `npm run build`. PR 에 검증이 0 이라 타입 오류가 머지된 뒤 유일한 릴리즈 경로(pages.yml)에서야 빨개지던 문제를 머지 전으로 당긴다. `pages.yml` 은 한 글자도 안 건드렸다(`git diff --stat origin/main -- .github/workflows/pages.yml` 출력 없음), `permissions: contents: read` 만.
- 정찰이 미확인으로 남긴 기준선을 먼저 확인했다: `npm ci`(82 packages) + `npm run build` → EXIT=0 녹색. 그 뒤 `node_modules` 를 지우고 CI 와 동일한 3단계를 그대로 실행 → EXIT=0. `test` 스크립트 없는 상태의 `npm run test --if-present` 도 EXIT=0 으로 건너뜀을 실행 확인. env 없이 빌드 통과도 실측됨(그래서 워크플로에 `vars.VITE_*` 를 넣지 않았다).
- **확신 없는 곳**: 수용 기준 4(PR 에서 ci.yml 녹색)를 **확인하지 못했다** — `gh auth status` 미인증, `git remote -v` push 가 `DISABLED`. 워크플로는 로컬 실행 불가라 이 한 가지는 구조적으로 이 세션에서 증명 못 한다. 대신 ⓐ GitHub 워크플로 스키마 검증기(`npx @action-validator/cli`)로 ci.yml 통과 + `pull_request`→`pull_requests` 로 한 글자 바꾼 사본이 EXIT=1 로 거부되는 음성 대조(검증기가 무의미하지 않음), ⓑ `src/lib/util.ts` 에 타입 오류 한 줄을 넣어 CI 명령이 실제로 EXIT=2 로 빨개지고 되돌리면 EXIT=0 이 되는 재현을 남겼다. **다음 역할(비평/릴리즈)은 PR 이 열린 뒤 `gh pr checks` / `gh run list --workflow=ci.yml` 결론(success)을 반드시 확인할 것** — 특히 `actions/setup-node` 의 `cache: npm` 은 러너에서 처음 돌아간다(`package-lock.json` 은 git 에 추적됨을 확인).
- 일부러 안 한 것: README 배지(1파일 유지), vitest/eslint/의존성 업그레이드(심사 중 PR 과 충돌 — `--if-present` 가 둘을 양립시킨다), 차선 후보인 countdown 버튼 수정(1순위가 막히지 않았으므로 ideas.json 에 pending 으로 남김). `npm ci` 가 보고한 취약점 4건(moderate 3/high 1)은 상세를 읽지 않고 ideas.json 신규 항목으로만 적었다 — 스코프 밖.
- [러너 18:26] brief accepted — 채택 — 1순위를 과제서 권장 YAML 그대로 구현했고, 수용 기준 1·2·3·5 를 충족했다. 4(PR 녹색)만 이 세션의 GitHub 권한 부�
- [러너 18:26] verify passed — 검증 2개 통과 (auto)
- [러너 18:26] pr created — https://github.com/hkjang/ox-arena/pull/2
- [러너 18:26] guard held — .github/workflows/ci.yml 
