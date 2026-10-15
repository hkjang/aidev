# 회차 노트 2026-10-15-205838-studio.irumx.app-improve — studio.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 20:58] base pinned — main@b00d53c
- [러너 20:58] autonomy release — 

## 정찰 노트
- 우선 과제(verify-failed) 를 추적했다: 원인은 코드가 아니라 **시험 하네스가 PATH 의 `node` 를 믿는 것**이다 — `playwright.config.ts:26` 의 `npx wrangler dev` 가, npm 이 PATH 앞에 붙인 저장소 밖 `/home/hkjang/node_modules/.bin/node`(v20)로 떠서 죽고 webServer 가 안 올라온다. 1858 회차(심링크 복구 → verify failed)와 1958 회차(심링크 이동 → verify passed)가 같은 base 에서 그 차이뿐이었다. 두 회차가 저장소 밖에서 손으로 비켜 간 것이 "같은 이유로 두 번 실패" 의 실체다.
- 정정 2건을 과제서에 적었다: ① 이 base 에는 `.github/workflows/` 가 **없다**(CI 는 머지 안 된 브랜치 2cdba01 에만) — 실패한 "워크플로 단계" 는 러너의 `npm test --silent` 다 ② `process.execPath` 로 Node 를 고정하는 흔한 처방은 여기서 안 통한다(`playwright` 셰방도 같은 PATH 를 탄다) — 고칠 자리는 PATH.
- 추측/미확인: `/home/hkjang/node_modules/.bin/node` 가 **지금도** 있는지 확인 못 했다(샌드박스가 저장소 밖 읽기·복합 명령을 막았고 예산도 걸렸다). 그래서 과제서의 재현 절차를 `/tmp` 가짜 스텁으로 적었다 — 심링크 유무와 무관하게 결함을 드러낸다. 시험도 돌리지 못했다(이 워크트리에 `node_modules` 가 없다 — `npm ci` 부터 필요).
- 제친 후보: cron.ts·files.ts·mp4.ts 관련 아이디어는 전부 머지 대기 브랜치 5개가 들고 있어 충돌한다. CI 단계 추가 2건은 워크플로 파일이 base 에 없어 `rejected` 로 바꿨다.
- 구현자가 조심할 것: 통과시키려고 timeout·retries·project 제외·심링크 이동 같은 **느슨하게 하는 길로 가지 말 것**(그게 금지 사항이고 반복 실패의 원인). 종전 기준선은 **76개**(b00d53c). `npm test` 전에 `npm run build` 필수.
- [러너 21:04] scout done — `npm test` 가 PATH 의 엉뚱한 Node 로 `wrangler dev` 를 띄우려다 webServer 가 죽는다 — 시험 하네스를 저장소 밖 `no

## 구현 노트
- 무엇을 왜: PATH 에서 **저장소 밖** `node_modules/.bin` 만 덜어내는 순수 함수(`tests/lib/harness-path.ts`)를 만들어 시험 서버 2개에 주고, 시험 경로의 `npx` 홉 넷(playwright.config.ts·db-local.mjs·admin.mjs·unit-db.spec.ts)을 지금 돌고 있는 node 로 로컬 설치본 직접 호출로 바꿨다. `db-local.mjs` 앞단에 Node 버전 가드. 프로덕션 파일 3개(설정·스크립트 2) + 시험 2개.
- 과제서와 다르게 한 곳(실증 근거 있음): PATH 걸러내기만으로는 안 된다 — `npm exec` 가 조상의 `.bin` 을 PATH 앞에 **다시** 붙인다(PATH 에 낡은 node 가 전혀 없어도 `npm exec -- wrangler --version` 이 Node 20 으로 떴다). 그래서 `command` 의 `npx wrangler dev` → `node node_modules/wrangler/bin/wrangler.js dev`. 공용 함수는 `.mjs` 대신 `.ts`(`tsconfig.node.json` 이 `allowJs` 없이 `playwright.config.ts` 를 검사한다).
- 확신 없는 곳·검증 못 한 것: ① **Windows 미검증** — `spawnSync` 에서 `shell: process.platform === 'win32'` 를 뺐다(절대 경로 node + 인자 배열이면 셸이 필요 없다는 판단이지만 Windows 에서 돌려보지 못했다). `scripts/admin.mjs` 는 운영자가 직접 쓰는 스크립트라 이 변경이 닿는다 ② `db-local.mjs` 의 버전 비교(`older`)는 프리릴리스 꼬리표(`23.0.0-nightly`)를 NaN 으로 읽는다 — 오탐은 안 나지만 단위 시험은 없다(가드는 Node 20 으로 직접 돌려 0.018초 exit 1 을 확인했다) ③ `getPlatformProxy`(unit-db)는 여전히 Playwright 를 띄운 node 로 in-process 로 돈다 — Node 20 에서도 통과했지만 더 낡은 node 에서는 모른다.
- 일부러 하지 않은 것: `scripts/deploy.mjs`·`portfolio-import.mjs` 의 npx 홉 4개(시험 경로가 아니고 자동 시험이 없다 — 아이디어로 적었다), `reuseExistingServer`(차선 후보, 원인 분리), timeout·retries·project 제외 같은 느슨하게 하는 변경(금지), 저장소 밖 심링크 손대기(앞 두 회차의 실패 원인).
- 다음 역할이 조심할 것: 검증은 `npm run build` → `npm test` 순서(webServer 가 `dist` 를 띄운다). 이 환경은 `$HOME` 이 임시라 **`PLAYWRIGHT_BROWSERS_PATH=/home/hkjang/.cache/ms-playwright`** 없이는 desktop·mobile 7개가 브라우저 없음으로 깨진다(저장소 문제 아님). 재현하려면 저장소의 **조상** 디렉터리에 Node 20 을 놓아야 한다 — `/tmp` 스텁은 `npx` 가 다시 붙이지 않아 결함을 드러내지 못한다.
- [러너 21:18] brief accepted — 채택 — 과제서의 진단(원인은 코드가 아니라 하네스가 PATH 를 믿는 것)과 파일·행 번호가 정확했고 수용 기준 1·2·4·5 �
- [러너 21:19] verify passed — 검증 4개 통과 (auto)

## 비평 노트
- 확인한 것: diff 6파일 전부 정독. 진단을 독립 재현 — `npm exec` 가 조상 `.bin` 5개(`~/.cache/auto-improve-wt`, `~/.cache`, `~`, `/home`, `/`)를 PATH 앞에 붙이는 것을 확인했고 `repoBinOnlyPath` 가 저장소 것만 남기는 것을 확인. `spawnSync` 가 **자식 env 의 PATH** 로 명령을 찾는다는 것을 스텁으로 실증(unit-db 의 `spawnSync('node', …)` 전제가 성립). `older()` 를 10개 값으로 돌려 봄 — `process.versions.node` 형태(`x.y.z`)에서는 오탐·누락 없음(`23.0.0-nightly`→통과, `22.11.0-nightly`→차단). `npm run check` exit 0 직접 확인. verify.txt 에 신규 단위시험 4개 통과 + 80 passed(76+4) 확인. 원장의 `- 실패 재현:` 은 증상과 일치하는 실제 출력(그림자 있음/없음 양쪽 검증)이라 신뢰함.
- 못 본 것: Windows 에서 실제 실행(구현자 자가의심 ①). 다만 `shell` 제거는 절대경로 node.exe + 인자배열이므로 오히려 안전하고, `{...process.env, PATH}` 의 `Path`/`PATH` 중복은 Node 가 대문자 우선으로 중복 제거해 걸러진 쪽이 이긴다 — 위험은 낮다고 봄. Node 20 그림자를 직접 놓은 재현은 하지 않음(저장소 밖 쓰기를 피했다 — 앞선 두 회차의 실패 원인).
- 승인이어도 남는 우려(릴리즈 노트에): ① `npm run build` 는 **여전히 노출**돼 있다 — `npm run check`(`.bin/tsc`)·`subset-font.mjs` 가 조상의 낡은 node 로 뜰 수 있다. `npm test` 전에 build 가 필수이므로 같은 증상이 build 단계에서 재발할 수 있다(원장 보류 ③, 다음 회차 최우선 후보) ② `deploy.mjs`·`portfolio-import.mjs` 의 npx 홉 4개는 그대로(보류 ②) ③ `node_modules/wrangler/bin/wrangler.js` 를 3곳에서 직접 가리킨다 — wrangler 가 bin 배치를 바꾸면 깨진다(package.json `bin` 과 일치하므로 지금은 정당, 깨지면 시끄럽게 깨짐) ④ `scripts/admin.mjs:30` 의 "Windows 셸에서…" 주석은 셸을 뺀 뒤로 근거가 낡았다(동작은 무해).
- 보안·법무: 인증·권한·결제·마이그레이션·비밀값·개인정보에 닿지 않는다. 오히려 공격면이 줄었다 — Windows `shell: true` 제거(인자 경유 셸 메타문자), PATH 의 빈 항목(POSIX 에서 CWD 조회) 제거. 차단 소견 없음.
- [러너 21:24] review approved — 리뷰 승인 (risk=low)
- [러너 21:24] pr created — https://github.com/hkjang/studio.irumx.app/pull/5
- [러너 21:29] ci no-ci — 이 커밋에 검사가 없음 (정책 allow_merge_without_ci 가 없으면 차단)
