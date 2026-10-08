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
