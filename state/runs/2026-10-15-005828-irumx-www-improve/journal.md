# 회차 노트 2026-10-15-005828-irumx-www-improve — irumx-www
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:58] base pinned — main@94eb01f
- [러너 00:58] autonomy release — 
- [러너 00:58] scout skipped — 요청된 작업이 있어 정찰을 건너뛴다 (과제는 이미 정해져 있다)

## 구현 노트
- 무엇/왜: `.github/` 가 없어 PR 이 전부 "CI 검사 없음" 으로 막혔다. README '명령' 의 검증 순서를 그대로 `.github/workflows/ci.yml` 한 파일로 옮겼다(커밋 2ef209f, 1 file changed). 명령·기대값·테스트는 하나도 손대지 않았다.
- 검증: ci.yml 의 run 단계를 파싱해 **깨끗한 클론**(`git clone` → /tmp/ci-sim, node_modules·캐시 없음, `CI=true`)에서 그 순서대로 실행 — npm ci 0 / npm run build 0(`✓ 모두 통과`) / playwright install 0 / test:build 0 / npm test 0(77 통과·기존 폭 조건 4 건너뜀). actionlint 1.7.7 exit 0, PyYAML 파싱 확인.
- 확신 없는 곳: ① `npx playwright install --with-deps chromium` 의 `--with-deps` 는 이 샌드박스에서 `sudo: a password is required` 로 못 돌렸다(GitHub 호스티드 러너는 무비밀번호 sudo). 로컬 시뮬레이션은 `--with-deps` 없이 돌렸다 ② `node-version: '22'`(.nvmrc 가 없고 engines 는 >=22.12 — 로컬은 22.23.1) ③ GitHub 러너에서의 `wrangler dev` 는 실제로 돌려 보지 못했다. 로컬은 Cloudflare 인증 없이 동작했다(HOME 에 토큰 없음) ④ 워크플로가 실제로 GitHub 에서 녹색인지는 push 권한이 없어 확인 불가.
- 일부러 안 한 것: 테스트 추가·수정 금지(명세), 스크립트·문서·소스 수정 금지(명세) — 그래서 `npm run build` 가 `test:build` 안에서 또 돌는 중복, Playwright 브라우저 캐시, serve-static 하드닝, harfbuzzjs 미선언은 ideas.json 으로 넘겼다. CHANGELOG 는 저장소에 없어 더하지 않았다(릴리즈 작업 금지 규칙도 있음).
- 다음 역할 주의: `npm test` 는 반드시 `npm run test:build` 뒤에 돌려야 한다(pretest 없음, dist·dist-draft·dist-email 필요). 8820~8823 포트를 쓰고 서버 재사용을 끄므로 다른 서버가 점유하면 즉시 실패. 이 PR 은 보호 경로(.github/workflows) 라 사람 승인이 필요하다.
- 환경 메모(저장소와 무관): 이 임시 HOME 의 `/home/hkjang/node_modules/.bin/node` 가 node 20 을 가리켜 `npm run` 이 "Node.js v20.19.2 is not supported by Astro" 로 죽었다. 검증 동안만 치워 두고 끝에 되돌렸다 — 저장소 문제가 아니다.
- [러너 01:09] verify failed — 실패한 검증: npm test --silent (exit 1)
