# PR 처리기 노트 2026-10-09-044328-umm-shepherd — umm PR #170
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-09-015837-umm-improve)
# 회차 노트 2026-10-09-015837-umm-improve — umm
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 01:58] base pinned — main@e54875c
- [러너 01:58] autonomy low-risk — 회귀(reverted) 2026-10-06T22:38:39+09:00

## 정찰 노트
- dev·preview·e2e 설치 진단을 선택: 실제 npm 실패를 재현했고 기존 가드 재사용으로 프로덕션 1파일·시험 1파일에 한정; 번역 시험·문서 정리보다 관찰 가능한 개선이다.
- e2e는 과거 추측인 exit 127이 아니라 상위 Playwright 경유 @playwright/test 누락/exit 1. 가드 직접 실행은 올바른 패키지와 설치 명령을 안내했다.
- 미확인: 설치 후 CLI --help/--list 및 Vitest 성공. web/node_modules가 없어 전체 시험 미실행; 설치·실제 npm 배선 검증을 구현자가 수행해야 한다.
- Makefile/outline 현재 상태는 과거 성공 기록과 다름. #167 파일명 회귀·Go 게이트·outline 과제를 다시 제출하지 말고 지정 두 파일만 수정할 것. 세 스킬은 Skill 도구 부재로 로컬 SKILL.md를 읽어 적용했다.
- [러너 02:04] scout done — dev·preview·e2e에서도 설치 누락을 기존 가드로 설명하기 (가치 2 / 위험 1 / 작업량 S)

## 구현 노트
- c719700: package.json에 predev/prepreview(vite)·pree2e(@playwright/test) 3줄 추가; 기존 명령 본문·가드·락파일 불변, 프로덕션 1파일+시험 1파일.
- 기준선 10개 PASS → 새 시험 7 FAIL/15 PASS → 배선 후 22 PASS; 훅만 원복해 같은 7 FAIL 재현 후 복구. 실제 미설치 체크아웃도 세 진입점 안내·exit 1 확인 후 설치 디렉터리 복원.
- make test-web 8게이트 exit 0: 23파일/252시험, audit 0건, i18n 1060키, PWA 150 assets; lint의 미변경 src 경고 50개는 남음. 버전 검사·시험 포맷 검사도 PASS.
- 과제서 Prettier 명령은 루트 기준 상대경로로 exit 2; web에서 npm exec -- prettier --check scripts/require-installed.test.mjs 실행, 포맷 수정 후 PASS.
- 검증 못 한 것: 실제 브라우저/서버/DB 제품 E2E·Go·Docker. dev/preview --help 및 e2e --list(129개/38파일)는 CLI 배선 검증만 의미함.
- 일부러 제외: format 가드·파서·파일명·Go 게이트·보호 파일·버전·릴리즈·원격 작업. CI의 npx playwright test 직접 호출에는 새 pree2e가 적용되지 않음.
- 다음 역할: npm ci --prefix web 선행; 세 technology 스킬은 Skill 도구 부재로 로컬 SKILL.md를 읽어 적용. red/reverted-red/test-web/CLI 로그와 원장·ideas.json은 이 회차 디렉터리에 보존.
- [러너 02:08] brief accepted — 채택 — 세 훅 누락과 기존 가드 재사용 가능성을 현재 코드 및 실제 npm 실행으로 확인했고 지정한 두 파일만 수정했으며
- [러너 02:09] verify passed — 검증 17개 통과 (auto)

## 비평 노트
- approve / low / blocking 없음: 두 파일 diff·커밋·기존 가드·npm 환경 격리·원장의 7개 실패 재현을 확인; 범위 이탈·보안/개인정보 변경·비가역 변경 없음.
- 관련 Vitest 22개 직접 재실행 PASS, git diff --check PASS; 저장소 소스 수정 없음.
- 실제 서버·DB·브라우저 E2E, Go, Docker, 전체 웹 게이트는 재실행하지 않음; --help/--list는 CLI 배선 검증으로만 해석할 것.
- 다음 회차·릴리즈 참고: CI의 npx playwright test에는 pree2e가 적용되지 않음. Skill 도구 부재로 지정 세 로컬 SKILL.md를 읽어 적용.
- [러너 02:10] review approved — 리뷰 승인 (risk=low)
- [러너 02:10] pr created — https://github.com/hkjang/umm/pull/170
- [러너 02:14] ci failed — 성공이 아닌 검사: verify=failure · 실패한 검사: ? 잡: verify 

## 수리 노트
- 맞았던 지적: CI의 GO-2026-6629 실패를 그대로 재현; 틀린 지적은 발견되지 않았으며 웹 설치 가드와 무관한 Go 의존성 문제임을 확인.
- 수정: c53a678에서 go.mod/go.sum의 x/text만 v0.41.0으로 갱신; 세 technology 스킬은 Skill 도구 부재로 로컬 SKILL.md를 읽어 적용, push 없음.
- 검증: govulncheck FAIL→PASS→구버전 재복원 FAIL→최종 PASS; DB 연결 전체 Go·vet·race·빌드·모듈 검사 및 make test-web 252시험/PWA PASS. 기존 CI 검사를 활용하여 새 테스트는 추가하지 않음.
- 확신 없는 곳: 실제 브라우저 E2E·Docker 이미지·외부 AI·설정 조건부 시험/전체 CI는 미검증; 기존 lint 경고 50개와 미호출 취약점 보고는 남음. 상세 증거는 fix-summary.md와 fix-*.log.

## 심사 노트
- approve / merge / low, blocking 없음: HEAD c53a678 네 파일·두 커밋과 지정 세 로컬 스킬 검토; 현재 origin/main에는 동일 x/text 수리가 이미 반영되어 실질 병합 변경은 웹 두 파일이며 충돌 없음.
- 확인: 실제 npm 회귀 22시험과 기준/HEAD 미설치 차이 재현, 전체 웹 252시험·빌드·PWA, govulncheck·vet·모듈 검증·PRECIS 시험 및 격리 PostgreSQL 연결 전체 Go 시험 PASS.
- 보안·개인정보·라이선스 차단 결함 없음; 가역적인 기존 훅 확장으로 목적과 구현이 일치해 머지 권고. 소스 수정 없음, 심사용 DB 폐기.
- 못 본 것: 브라우저 제품 E2E·Docker·외부 AI/설정 조건부 경로·race·원격 전체 CI 재실행; 기존 lint 경고와 미호출 취약점 보고는 남고 --help/--list는 CLI 검증만 의미함.
