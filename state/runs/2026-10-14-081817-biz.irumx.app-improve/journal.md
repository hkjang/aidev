# 회차 노트 2026-10-14-081817-biz.irumx.app-improve — biz.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 08:18] base pinned — main@346eb3c
- [러너 08:18] autonomy release — 

## 정찰 노트
- safeNext 시험을 골랐다: 로그인 복귀 경로 가드가 sso.ts:57 과 apps/web/src/lib/nav.ts:2 에 같은 한 줄로 중복돼 있는데 양쪽 다 시험이 0개다. 차선인 v.day 시험(validate.ts:216)보다 보안 접점이 크고 비용은 같다. krwShort 10억 표기·relative 경계는 제품 의도가 미확인이라 또 미뤘고, download.ts textarea 누수는 node 환경에 DOM 이 없어 증거를 만들 수 없다.
- 추측으로 적은 것: `'/\t/evil.com'`(실제 탭) 이 safeNext 를 통과한다는 것은 소스 판독이고, 서버 최종 복귀(sso.ts:302 `new URL(next_path, publicUrl)` → pathname+search 만 사용)가 이를 '/' 로 접는다고 읽었지만 브라우저/실행 확인은 하지 않았다. 그래서 탭 케이스는 시험에서 제외하라고 적었다 — 지금 동작을 계약으로 못 박으면 거짓이 된다.
- 구현자 주의: 프로덕션 코드는 건드리지 말 것(두 사본 통합도 금지 — 운영자 지시). 워크스페이스를 넘는 import 로 케이스 표를 공유하면 tsc 가 깨지니 표는 각 파일에 복제. 길이 경계는 `<300` 이라 299 통과/300 거절.
- 이 체크아웃에는 node_modules 가 없어 시험을 실제로 돌려 보지 못했다(`npm ci` 를 돌리면 작업 트리가 변해 읽기 전용 원칙에 걸린다). 명령은 package.json·vitest.config.ts 로 확인했다.
- [러너 08:23] scout done — 로그인 복귀 경로 가드 safeNext 의 회귀 시험과 두 사본 일치 고정 (가치 4 / 위험 1 / 작업량 S)

## 구현 노트
- 과제서대로 시험만 추가했다(프로덕션 0파일). `safeNext` 의 거절 11건·통과 4건을 15개 케이스 표로 적고, 같은 표를 `packages/core/src/__tests__/domain.test.ts` 와 신규 `apps/web/src/lib/nav.test.ts` 에 복제했다. 화면 사본은 시그니처가 `string | null | undefined` 라 `undefined` 1건만 더 넣었다. 커밋 7a92754.
- 확신 없는 곳: 두 사본 일치는 "같은 표에 같은 기대값" 으로만 증명된다 — 시험이 두 함수를 직접 비교하지는 않는다(워크스페이스 경계 때문에 불가). 한쪽 구현이 바뀌면 그 파일 시험이 깨지는 것은 변이로 확인했지만, 두 표 자체가 미래에 따로 편집되는 것은 시험으로 막지 못한다(주석으로만 경고).
- 검증 못 한 것: 로그인 왕복의 실제 브라우저 동작은 확인하지 않았다(DB·OAuth·Playwright 필요). sso.ts:302 이후 경로가 어떻게 쓰이는지는 읽기만 했다.
- 일부러 하지 않은 것: 탭/제어문자 케이스(지금 통과를 계약으로 못 박으면 거짓, 가드 좁히기는 범위 밖 — ideas.json 에 신규 후보로 올림), 두 사본 통합(운영자 지시), 표를 공용 모듈로 빼기(tsc 가 워크스페이스별로 돌아 깨짐).
- 다음 역할이 조심할 것: 이 시험들은 DB·브라우저 없이 돈다(`npm test -- packages/core/src/__tests__/domain.test.ts apps/web/src/lib/nav.test.ts`). `npm run check` 는 113개 전원 통과, `npm run build` 통과. 변이 증거는 원장 '실패 재현' 에 세 건 모두 적었고 변이는 전부 되돌렸다 — 작업 트리에 프로덕션 변경이 없음을 `git status` 로 확인했다.
- [러너 08:27] brief accepted — 채택 — 지정된 두 파일에 지정된 케이스 표를 그대로 넣고 변이 시험으로 가드를 보고 있음을 증명했다. 탭 문자 케이스
- [러너 08:27] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인한 것: 구현자 주장을 손으로 재현했다 — 두 사본에서 `//` 절 제거 4건 실패, `/\` 절 제거 4건 실패, `.trim()` 제거 2건 실패. `npm run check` 113/113 통과, diff 는 시험 2파일 +72줄뿐이며 `safeNext` 두 export 는 main 에 이미 있었다(프로덕션 변경 0).
- 남는 우려 1 — 길이 경계 미고정: 표에 299자 통과·321자 거절만 있어 `< 300` → `<= 300` 변이가 113개 전원 통과로 살아남는다. 과제서가 요구한 '300 거절' 케이스(`'/' + 'a'.repeat(299)`)를 다음 회차에 두 표에 추가할 것.
- 남는 우려 2 — 주석 과장: nav.test.ts:8·domain.test.ts:217 의 '한쪽만 건드리면 다른 파일이 깨진다' 는 거짓(자기 파일 시험이 깨지고, 두 표의 분기는 아무것도 잡지 못한다). 구현 노트는 맞게 적었으니 주석만 좁히면 된다.
- 남는 우려 3 — 잠복 위험(릴리즈 노트에 적을 것): `'/\t/evil.com'` 은 양쪽 가드를 통과하고 `new URL('/\t/evil.com', publicUrl).host` 는 `evil.com` 이다(node 로 실행 확인). 지금 안전한 유일한 이유는 sso.ts:304 가 `pathname+search` 만 쓰기 때문 — 이 줄이 `target.href` 로 바뀌면 오픈 리다이렉트다. 새 시험은 이 자리를 덮지 않는다.
- 못 본 것: 실제 브라우저 로그인 왕복(Login.tsx:20 `<Navigate to={next}>`) — DB·OAuth·Playwright 필요. 보안·법무 차단 사유 없음(새 공격면·비밀값·권한 확대·개인정보 변경·새 의존성 없음). 판정 approve / risk low.
- [러너 08:31] review approved — 리뷰 승인 (risk=low)
- [러너 08:31] pr created — https://github.com/hkjang/biz.irumx.app/pull/3
- [러너 08:41] ci passed — 검사 1개 모두 success
- [러너 08:41] merge done — 7a92754
