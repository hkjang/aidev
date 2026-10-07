# 회차 노트 2026-10-07-161806-eval.irumx.app-improve — eval.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:18] base pinned — main@202c416
- [러너 16:18] autonomy release — 

## 정찰 노트
- 왜 이것: 설계 수용 시험 12번(예산 동시 예약)은 두 회차 연속 실패해 세 번째로 올리지 않았다. 10번(큐 중복 전달)은 시험에서 큐로 메시지를 보내는 경로가 아예 없어(grep 확인) 프로덕션에 훅을 더해야 하므로 제쳤다. D1 바인딩 쪼개기 시험은 계산이 이미 안전해(96·90 ≤ 100) 기각했다. 남은 것 중 실제로 회차를 헛돌게 만들던 것 — 시험 세션 캐시와 로컬 D1 비우기의 어긋남 — 을 골랐다. 프로덕션 코드 0개, 시험 보조 2개.
- 확인한 것: `scripts/db-local.mjs` 가 `.wrangler/test-state` 만 rmSync · `tests/helpers.ts:71` 이 `.wrangler/test-sessions` 에 storageState 저장 · `src/worker/auth.ts:38` cookieCache 300초 · `src/worker/routes/me.ts:19` 가 workspaces 를 DB 에서 따로 조회 · `tests/helpers.ts:92-95` ownerApi 의 `me.workspaces[0].id`. 이 다섯 줄이 합쳐져 TypeError 가 된다.
- 추측으로 적은 것(구현자가 확인할 것): 실패를 재현하려면 쿠키 캐시 5분 창 안에 두 번 돌려야 한다는 타이밍 — 직접 재현해 보지 않았다(빌드가 수 분이라 예산 안에서 못 했다). 재현이 안 되면 차선(문서 한 줄)으로 돌릴 것. `--test-scheduled` 가 큐 소비자를 부르지 않는다는 것도 미확인.
- 조심할 것: `auth.ts` 의 cookieCache 를 끄는 방향으로 고치지 말 것(운영 설정). 세션 캐시 자체를 없애지도 말 것(로그인 코드 15분 5번 제한 회피용). `reuseExistingServer: true` 라 서버를 끄지 않으면 재현도 검증도 둘 다 거짓이 된다.
- 프로필은 0일 전 것이 지금 코드와 맞아 새로 쓰지 않았다.
- [러너 16:22] scout done — 시험 세션 캐시와 로컬 D1 비우기가 어긋나 두 번째 실행이 깨지는 것 고치기 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- 무엇을 왜: `scripts/db-local.mjs` 가 `.wrangler/test-sessions`(로그인 쿠키 보관)도 지우게 한 줄 더하고, `tests/helpers.ts:login` 의 캐시 재사용 조건에 `me.workspaces?.length || me.operator` 를 더했고, `ownerApi` 는 업체 공간이 없으면 한국어 안내로 throw. 프로덕션 코드 0개.
- 재현/검증(실제 실행): 고치기 전 — run1 통과 → `wrangler dev`(:8860) 끄고 3초 뒤 같은 명령 → `TypeError: Cannot read properties of undefined (reading 'id')` at `helpers.ts:94`. 고친 뒤 같은 절차 → 통과. 세션 캐시만 남고 DB 는 새로 만든 상태를 억지로 만든 실행(수동으로 띄운 서버 + 옛 storageState 복원)에서도 통과 — 캐시를 버리고 재로그인했다. 전체 `npm run build && npm test` → `50 passed (1.3m)`, exit 0.
- 확신 없는 곳: (1) 캐시를 버리면 로그인 코드를 다시 받는다 — 같은 트리에서 아주 짧은 간격으로 여러 번 돌리면 "15분에 5번" 제한에 더 가까워질 수 있다. 전체 시험 한 번에서는 로그인이 사람당 1~2번뿐이라 문제없었지만 한계는 재 보지 않았다. (2) `ownerApi` 의 한국어 안내는 `ops@eval.test`(업체 공간 없음 + `operator:true`)로만 확인했다 — 그 확인용 임시 시험(`tests/api-tmpcache.spec.ts`)은 통과를 보고 지웠으므로 저장소에는 없다.
- 일부러 안 한 것: `auth.ts` 의 `cookieCache` 는 건드리지 않았다(운영 성능 설정). 세션 캐시 자체도 없애지 않았다(로그인 코드 제한 회피용). mjs 스크립트와 ts 시험 사이에 `.wrangler/test-sessions` 상수를 공유하지 않고 양쪽에 고정값으로 두었다(mjs 가 ts 를 import 하지 않는다). `tests/` 에 영구 시험을 더하지 않았다 — 고친 것이 시험 보조 자체라 증거는 실제 두 번째 실행 기록이다.
- 다음 역할이 조심할 것: 이 워크트리는 `npm ci` 가 안 된 상태였다(`npm run build` 가 `subset-font` ERR_MODULE_NOT_FOUND 로 죽는다) — 먼저 `npm ci`, 그 다음 `npx playwright install chromium`. 재현·검증은 반드시 :8860 의 `wrangler dev` 를 끈 뒤에 해야 한다(`reuseExistingServer: true`). `pkill -f "port 8860"` 은 자기 셸의 명령줄까지 잡아 자살하므로 스크립트 파일 안에서 돌릴 것.
- [러너 16:38] brief accepted — 채택 — 과제서가 지목한 다섯 줄(db-local 의 rmSync · helpers 의 storageState 경로 · cookieCache 300초 · me.ts 의 workspaces 별도 조회 
- [러너 16:40] verify passed — 검증 4개 통과 (auto)

## 비평 노트
- 확인한 것: 두 파일 diff 전부 · `src/worker/routes/me.ts`(workspaces 는 DB, operator 는 env) · `src/worker/access.ts`·`auth.ts`(cookieCache 300초, OTP 15분 5번) · `scripts/admin.mjs`(사용자 id 가 난수라 DB 를 비우면 옛 세션의 userId 가 안 맞는다 — 가드가 소유자에게는 실제로 듣는다) · `playwright.config.ts`(db-local 은 webServer 안, `reuseExistingServer:true` 면 안 돈다) · `scripts/backup-drill.mjs`(db-local 을 부르지 않아 시험 중간에 세션 폴더가 지워질 일은 없다) · `npx tsc -p tsconfig.node.json` exit 0. 구현자가 보고한 실패(`helpers.ts:94` 의 `reading 'id'`)는 고치기 전 파일의 그 줄(`me.workspaces[0].id`)·증상과 정확히 맞는다 — 다른 이유의 출력을 붙인 것이 아니다.
- 못 본 것: `npm run build && npm test` 를 다시 돌리지 않았다(워크트리에 `npm ci` 가 안 돼 있어 build 가 `subset-font` 로 죽는다 — 예산으로 끊었다). "50 passed" 는 구현자 기록을 믿었다. OTP 15분 5번 한계도 직접 재지 않았다.
- 승인이어도 남는 우려 1: 가드가 운영자(`ops@eval.test`)에게는 듣지 않는다 — `me.operator` 는 env 에서 나와 DB 세션의 증거가 아니다. `tests/api-targets.spec.ts:161` 은 지금 159줄이 먼저 터져 가려져 있다.
- 승인이어도 남는 우려 2: '세션 없음' 과 '활성 업체 공간 없음' 을 같게 본다. 업체 공간을 suspended 로 바꾸고 되돌리지 않는 시험이 생기면 `ownerApi` 마다 재로그인 → 15분 5번에 걸려 `새 메일 없음` 으로 엉뚱하게 깨지고, 96줄 안내문이 원인을 잘못 지목한다. 다음 회차가 알 것: `.wrangler/test-sessions` 경로가 mjs(뿌리 기준)·ts(cwd 기준) 두 곳에 따로 박혀 있다.
- 릴리즈 노트용: 프로덕션 코드 0줄, 시험 보조만. 되돌리기는 두 파일 revert.
- [러너 16:43] review approved — 리뷰 승인 (risk=low)
- [러너 16:43] pr created — https://github.com/hkjang/eval.irumx.app/pull/1
- [러너 16:48] ci no-ci — 이 커밋에 검사가 없음 (정책 allow_merge_without_ci 가 없으면 차단)
