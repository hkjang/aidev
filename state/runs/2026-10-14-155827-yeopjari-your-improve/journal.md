# 회차 노트 2026-10-14-155827-yeopjari-your-improve — yeopjari-your
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 15:58] base pinned — main@735609c
- [러너 15:58] autonomy release — 
- [러너 16:04] scout done — PR #3 의 CI 실패를 고쳐 한도 429 의 `Retry-After` 수정을 다시 출하한다 (가치 4 / 위험 2 / 작업량 S)

## 구현 노트
- PR #3 커밋 2개를 cherry-pick 해 한도 429 의 `Retry-After` 수정을 재출하하고, `scripts/e2e.mjs` 의 가짜 서버 주소를 `YOUR_YEOPJARI_URL`/`E2E_PUSH_ORIGIN` 에서 읽게 했다(기본값은 옛 하드코딩과 바이트 동일 → CI 무변경).
- **CI 실패는 이 저장소 탓이 아니다.** 러너의 `state/runs/*/ci-*.json` 에 증거가 있었다: 성공한 main 실행 98초 vs 실패한 두 실행 2초·3초, 그리고 `2026-10-14T01:09:04Z` 이후 같은 계정 **다섯 저장소**의 `check` 가 전부 2~3초 failure(연속 10건). Actions 계정 수준 장애다. 과제서 1·2순위는 실험/코드로 반증했다(ideas.json 에 rejected 로 기록).
- **확신 없는 곳**: ① 계정 장애의 정확한 사유(한도 소진인지 과금 차단인지)는 `gh`·웹 접근이 없어 **미확인** — 체크런 수명과 전 저장소 동시성으로 추론한 것이다. ② `18998` 이 다른 프로젝트 고아 프로세스에 묶여 있어 **기본 포트로 e2e 를 끝까지 돌려 보지 못했다**; 기본값 해석(origin·listen 포트·prefix 문자열)만 미설정 상태로 확인했다. CI 는 두 변수를 안 쓰므로 기본 경로를 탄다.
- **일부러 안 한 것**: `check.yml` 무수정(보호 경로, 느슨하게 만들기 금지). `windowStart()` 무수정(`quota_counters` PK). `BASE`·`DATABASE_URL` 의 기본값 의존은 남겨 뒀다(이번 실패와 무관, 범위 초과).
- **다음 역할이 조심할 것**: e2e 는 Docker PostgreSQL 16(127.0.0.1:55440) + `bash scripts/dev-reset.sh` 로 **매번 새 DB** 가 필요하다 — 초기화 없이 재실행하면 관리자 MFA 등이 이미 등록돼 여러 검사가 거짓 실패한다(실제로 겪었다). 18998 이 막혀 있으면 `YOUR_YEOPJARI_URL=http://127.0.0.1:18996 E2E_PUSH_ORIGIN=http://127.0.0.1:18995` 로 돌릴 것. 이 워크트리의 `.env.dev`(gitignore 대상)는 `YOUR_YEOPJARI_URL` 을 18996 으로 바꿔 둔 상태라 서버도 그 값으로 떠 있다; `node scripts/ci-env.mjs` 로 되돌릴 수 있다. **CI 가 초록이 되기 전에는 머지하지 말 것 — 다만 그것은 코드가 아니라 운영자의 GitHub 과금 확인에 달려 있다.**
- [러너 16:18] brief fallback — 차선 — 1순위 과제(“CI 에서만 깨지는 검증 쪽 원인”)는 전제가 틀렸다. 러너 자신의 `ci-*.json` 이 잡 수명 2~3초와 5개 저
- [러너 16:18] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인한 것: 옛 `retryAfterSeconds` 를 따로 재구성해 돌려 새 단언 4개가 전부 실패함을 봤다(36000/118740 vs 3600/86340) — 테스트가 진짜 바뀐 경로를 지난다. `kstParts().m` 1-based × `Date.UTC` 0-based 조합이 "다음 한국 달 1일" 로 맞고 12월 넘김도 맞다. TZ 3종(UTC·Asia/Seoul·America/New_York)에서 53/53, `npm run check` 초록. `e2e.mjs` 기본값이 옛 하드코딩과 바이트 동일임을 실행으로 확인(18998/18997·prefix·aud 전부). `window_start` 키 무변경 → 운영 카운터 무영향, 세 커밋 revert 로 완전 복귀.
- 못 본 것: **e2e 를 돌리지 못했다**(18998 이 고아 프로세스 점유, 공유 dev DB 초기화는 병행 세션과 충돌). 구현자가 돌린 164/0 은 18996/18995 오버라이드 경로이고, CI 가 타는 기본 경로는 바이트 동일성으로만 입증했다.
- 승인이어도 남는 우려: ① `e2e.mjs:34` `Number(new URL(...).port)` 는 포트 없는 URL 에서 0 → 셸에 운영값(`https://yeopjari.bid`)이 떠 있으면 가짜 서버가 임의 포트에 붙고 `fetch(authorize)` 가 **실제 운영 SSO 로 나간다**(옛 하드코딩은 그 전에 실패했다). 루프백·포트 단언 한 줄로 닫힌다. ② `e2e.mjs:534` `?? null` 은 위 신고 검사가 깨지면 NOT NULL 위반으로 e2e 전체를 불명확하게 중단시킨다. ③ e2e 단언은 상한만 본다(옛 버그는 항상 +9h 라 걸리지만 과소값은 통과 — 정확값은 단위 테스트가 고정). ④ `e2e.mjs:24-28` 주석의 "한 곳에서 읽는다" 는 같은 줄 폴백이 `ci-env.mjs:35` 를 다시 적은 것과 어긋난다. ⑤ `discover.ts:166` "내일 낮에 다시 채워져요" 는 자정 리셋과 불일치(기존 문구).
- 머지 게이트: CI 빨간불이 계정 수준 Actions 장애라는 진단을 `ci-*.json` 으로 독립 확인했다(같은 날 세 저장소 모두 3초 failure, 9월 실행은 수 분대 정상). 그래도 **운영자가 과금·한도를 확인해 check 가 초록이 된 뒤 머지할 것** — 이 저장소의 유일한 배선 증명이 e2e 다.
- 릴리즈 노트에 쓸 것: 하루/한 달 한도 429 의 `Retry-After` 가 한국 자정 기준으로 바뀌었다(이전에는 최대 9시간, 한국 새벽에는 최대 33시간까지 과대 안내). 사용자에게 보이는 문구·스키마·권한은 무변경.
- [러너 16:24] review approved — 리뷰 승인 (risk=low)
- [러너 16:24] pr created — https://github.com/hkjang/yeopjari-your/pull/4
- [러너 16:24] ci failed — 성공이 아닌 검사: check=failure · 실패한 검사: ? 잡: check 
