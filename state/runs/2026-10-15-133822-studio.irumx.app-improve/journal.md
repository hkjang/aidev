# 회차 노트 2026-10-15-133822-studio.irumx.app-improve — studio.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 13:38] base pinned — main@b00d53c
- [러너 13:38] autonomy release — 
- [러너 13:38] scout skipped — 요청된 작업이 있어 정찰을 건너뛴다 (과제는 이미 정해져 있다)

## 구현 노트
- 바꾼 것: `.github/workflows/ci.yml` 하나뿐(43줄, 신규). push(main)·pull_request 에서 `npm ci` → `npm run build` → `npx playwright install --with-deps chromium` → `npm test`. 저장소에 워크플로가 없어 PR 이 'CI 검사 없음' 으로 막히던 것을 푼다. 프로덕션 코드 0개.
- 확신 없는 곳: **GitHub Actions 에서 실제로 돌려 보지 못했다**(이 세션은 push·gh 권한이 없다). 로컬 워크트리에서 같은 명령을 같은 순서로 돌려 통과만 확인했다 — `npm run build` "빌드 점검 통과", `npm test` "76 passed (57.7s)", YAML 은 `yaml.safe_load` 파싱 확인. ubuntu-latest 에서 `wrangler dev`(로컬 D1·R2·Queues, `services` 바인딩 `mail-gateway` 는 로컬에 없어 경고만)와 가짜 Resend(:8849)가 뜨는지는 첫 실행에서 봐야 한다. 포트 8848·8849 충돌, 네트워크 없이 workerd 가 받아지는지도 미확인.
- 시간대: 러너는 보통 UTC 인데 로컬은 KST 다. 브라우저 쪽은 `playwright.config.ts` 가 `Asia/Seoul` 을 고정하지만 unit·api 는 Node 시간대를 쓴다. 지금 시험에 TZ 의존 단정은 못 찾았으나(`toLocaleString('ko-KR')` 은 통화·숫자뿐) 확신하진 않는다 — 첫 CI 실행에서 unit·api 가 깨지면 여기부터 보라.
- 일부러 안 한 것: `npm run check` 분리 단계, `npm audit`, 캐시된 Playwright 브라우저, 배포 자동화. 명세가 파일 하나로 못 박았고 배포는 사람 손으로 둔다(scripts/deploy.mjs). 아이디어 파일에 적어 뒀다.
- 다음 역할 주의: `npm test` 는 **`npm run build` 가 먼저 돌아 `dist` 가 있어야** 한다. 또 이 샌드박스에는 `/home/hkjang/node_modules/.bin/node`(v20) 가 있어 `npx` 가 그걸 집어 wrangler 가 "Node.js v22 필요" 로 죽는다 — 워크트리의 `node_modules/.bin/node` 를 nvm v22 로 심링크해 두었다(gitignore 대상, 커밋 아님). CI 에는 없는 문제다.
- [러너 13:56] verify passed — 검증 4개 통과 (auto)
- [러너 13:56] pr created — https://github.com/hkjang/studio.irumx.app/pull/2
- [러너 13:56] guard held — .github/workflows/ci.yml 
