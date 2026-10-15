# 회차 노트 2026-10-15-102840-irumx-www-improve — irumx-www
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 10:28] base pinned — main@94eb01f
- [러너 10:28] autonomy release — 

## 정찰 노트
- 자동 배정된 수정 과제라 새로 고르지 않았다. 다만 베이스가 main@94eb01f 로 되돌아가 앞 회차의 pretest·serve-static 하드닝이 **하나도 없다**는 것을 직접 확인했다(package.json scripts 9개, serve-static.mjs:52-53·:42 원형, 작업 트리에 node_modules·dist 3벌 전부 없음) — 그래서 red 는 추측이 아니라 저장소 상태 그 자체다.
- 확신 없는 곳: 러너 환경의 node@20 오염. 심링크가 /home/hkjang/node_modules/node/bin/node 로 아직 해석되는 것까지만 봤고, 이 세션은 npm 실행 권한이 없어 **버전을 재확인하지 못했다(미확인)**. 과제서 검증 0단계로 `npm exec -- node -v` 진단을 먼저 넣었다.
- 구현자가 조심할 것: 옛 브랜치 체리픽으로 free-ports·동적 포트까지 끌고 오지 말 것 — 현 config 는 이미 reuseExistingServer:false 이고 기준 2가 playwright.config.ts diff 를 비워 두라고 요구한다. pretest 를 dist 유무로 조건화하는 것도 금지(테스트 Turnstile 키 없는 낡은 dist 로 contact-send 가 엉뚱하게 빨개진다).
- 차선(harfbuzzjs·fontverter)은 devDependencies 8개에 없는 것만 확인했고 verify-build.mjs 의 import 줄은 못 열었다. 1순위가 막힐 경우도 같은 환경 이유라 차선으로 피해지지 않는다는 점을 과제서에 적어 뒀다.
- [러너 10:33] scout done — 수정 과제 — `npm test --silent` 가 깨끗한 체크아웃에서 혼자 통과하게 고침(테스트 준비물 자동 생성 + 정적

## 구현 노트
- 무엇·왜: 커밋 415c9c0, 프로덕션 2 + 문서 1. `package.json` 에 `pretest`(test:build + chromium) 를 넣어 gitignore 된 dist 세 벌을 `npm test` 가 스스로 만들게 하고, `scripts/serve-static.mjs` 를 하드닝했다(기동 전 폴더·index.html 점검 후 exit 1, 404.html 선읽기 후 헤더, 핸들러 전체 catch, 경로 가드 `base + sep`). 테스트·`playwright.config.ts` 는 한 글자도 안 건드렸다(`git diff main` 으로 확인, 새 skip 0건).
- **확신 없는 곳**: `npm test` 가 green 이 된 것은 거의 전부 `pretest` 덕이다. serve-static 의 404·catch·경로 가드는 green 런에서 **실제로 지나가지 않는다**(pretest 가 404.html 을 만들어 주므로) — 그 세 가지는 저장소 밖 하네스(`/tmp/ss-check/check.mjs`, 커밋 안 함, 진짜 프로세스 spawn + 생소켓)로만 red→green 을 증명했다. 비평가는 여기를 먼저 보라.
- 검증 못 한 것: `worker/`·`src/lib/inquiry.ts`·`verify-build.mjs` 는 범위 밖이라 열지 않았다. 외부 Turnstile 의존 3건은 이 기계가 외부 연결이 돼서 통과했을 뿐, 폐쇄망에서는 모른다.
- 일부러 안 한 것: 옛 브랜치 체리픽(free-ports·동적 포트), `pretest` 의 dist 조건화(낡은 dist 엔 테스트 Turnstile 키가 없다), 홈 `node_modules` 손대기(사용자 소유 + 전이 의존이라 지워도 되돌아온다), 저장소 쪽 node 경로 우회(검증 느슨화).
- **다음 역할이 조심할 것**: 이 커밋은 **작업 트리(`/home/hkjang/.cache/auto-improve-wt/irumx-www`)에서 검증되지 않는다.** 측정 확정: 거기 `npm exec -- node -v` → v20.19.2(`/home/hkjang/node_modules/.bin/node` 때문), `npm test` → `Node.js v20.19.2 is not supported by Astro`. 재현하려면 `/tmp` 등 `/home/hkjang` 밖으로 클론해 `npm ci && rm -rf dist dist-draft dist-email && npm test --silent` (→ 77 통과·4 건너뜀·exit 0, 약 2분; 테스트 자체는 28초, 나머지는 pretest 의 이미지 411장 ×3 빌드).
- [러너 10:42] brief accepted — 채택 — 지정한 3개 파일·5개 수용 기준을 그대로 충족했고(선택 항목인 `base + sep` 경로 가드도 같은 파일이라 함께 넣음)
- [러너 10:42] verify failed — 실패한 검증: npm test --silent (exit 1)
