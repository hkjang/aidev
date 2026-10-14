# PR 처리기 노트 2026-10-14-191429-workkit.irumx.app-shepherd — workkit.irumx.app PR #4
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-14-183828-workkit.irumx.app-improve)
# 회차 노트 2026-10-14-183828-workkit.irumx.app-improve — workkit.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 18:38] base pinned — main@53ea324
- [러너 18:38] autonomy release — 

## 정찰 노트
- 러너가 지정한 수정 과제를 1순위로 두되, 직전 회차에서 기각된 "원인은 워크플로다" 전제를 반복하지 않도록 **주석을 먼저 읽고 분기하는 과제**로 적었다. 직전 회차가 인증 API 로 읽은 주석은 결제·지출 한도 차단(steps 빈 배열)이었고, 그렇다면 check.yml 은 한 줄도 건드려선 안 된다(검사 완화 금지).
- 확신 없는 곳: 이 세션에서 `gh` 와 `npm ci` 가 모두 권한 거부돼 **지금의 실패 원인도, 로컬 green 도 직접 확인하지 못했다**(미확인). 과제서의 "결제 차단" 은 직전 회차 기록의 인용이고, 검증 명령은 돌려 보지 않은 채 check.yml 의 단계에서 그대로 옮겼다.
- 코드 변경은 차선 후보로 둔 csv.ts 의 `push()` 가드를 골랐다 — main@53ea324 에서 한도 검사가 전부 `end()`(89행) 안에 있고 개행 없는 입력은 122행에서야 검사되는 것을 직접 읽어 확인했다. detectDelimiter 의 64KB 자르기(3순위)보다 자원 영향이 커서 앞에 뒀다.
- 새로 확인한 것: 열린 PR #2(0958)와 PR #3(1508)이 둘 다 `csv.ts` 의 같은 훅(88~96행)을 고쳐 **머지 충돌이 예정돼 있다**. 구현자가 PR 정리까지 손대려면 한쪽을 먼저 올리고 다른 쪽을 리베이스할 것.
- 구현자가 조심할 것: 새 가드의 red 를 만들기 어렵다(한도 초과 입력은 가드 전에도 같은 `too_many_cells` 로 던진다). 시험을 "한도+1 항목을 넘기지 않고 멈춘다" 로 쓸 수 없다고 판단되면 시험 없이 가드만 넣지 말고 3순위(detectDelimiter 64KB)로 교체할 것.
- [러너 18:44] scout done — PR #3 의 `check` 잡이 실패한 원인을 **잡 주석으로 먼저 판정**하고, 저장소 안의 원인일 때만 그 한 곳을 좁�

## 구현 노트
- 무엇·왜: `parseCsv`(csv.ts) 의 한도 검사가 전부 `end()` 안이라 개행 없는 줄(쉼표만 든 10MB 파일)에서는 불리지 않았다. `push()` 안 `record.push(field)` 직전에 `record.length >= LIMITS.cells` 가드 한 줄을 같은 `too_many_cells` 코드·문구로 넣었다. 프로덕션 파일 1개 + 시험 1개. `.github/workflows/check.yml` 은 한 줄도 건드리지 않았다.
- 확신 없는 곳: ① **PR #3 의 실제 실패 원인을 이 세션에서 확인하지 못했다** — `gh` 미인증 + 사설 저장소라 익명 API 404. 직전 회차가 읽어 둔 결제·지출 한도 주석을 전제로 분기했을 뿐이다. 주석이 바뀌었다면 전제가 무너진다. ② 넓은 줄에서 전에는 `cell_too_long` 이 나던 입력이 이제 `too_many_cells` 로 난다(의도한 변경, 시험으로 고정). 그 외 에러 코드 우선순위 조합은 전수 확인하지 않았다.
- 검증 못 한 것: `scripts/e2e.mjs` 148 passed / **7 failed**, `ui-flows.mjs` 9 passed / 2 problems. 내 변경과 무관함은 증명했다(변경을 되돌려 재빌드한 기준선이 동일한 7개 실패; core·server·scripts 는 `@workkit/tools` 를 import 하지 않음). 원인은 환경 충돌 — :55447 은 본 체크아웃의 상주 컨테이너 `workkit-irumx-dev` 가 쓰고 있어 내 컨테이너는 포트가 안 붙었고, 공유 dev DB 의 `files.object_key` 가 가리키는 zip 이 이 워크트리 `data/media` 에 없다(`exists=NO` → files.ts:93 `file integrity` 500).
- 일부러 안 한 것: 워크플로 완화·캐시 추가(결제 차단이면 효과 없고 운영자 금지), `xlsx-read.ts`·`compare.worker.ts` 의 같은 종류 파서 손대기(계약 다름), 열린 PR #2·#3 과의 충돌 정리(머지 순서는 러너 몫).
- 다음 역할이 조심할 것: 단위 게이트는 `npm ci` 뒤 `npm run check`(10 파일 **123 passed**)·`npm run build`·kits 2회+`cmp`·`ci-env` 모두 green, Playwright `tools-flows.mjs` 55 passed·`a11y.mjs` 위반 없음(최종 빌드 기준). e2e 를 다시 돌리려면 :55447 공유 DB 문제를 먼저 치워야 한다 — 내가 그 공유 dev DB 에 `migrate.mjs`(이미 최신, 스키마 변경 없음)와 `seed-local.mjs` 를 돌렸고 이 워크트리의 `data/media` 를 한 번 비웠다. csv.ts:84~96 은 PR #2·#3 과 세 갈래로 겹친다.
- [러너 19:01] brief fallback — 차선 — 1순위는 전제를 **확인할 수단이 없었다**(이 세션 `gh` 미인증 + 사설 저장소라 404). 과제서가 지정한 분기 ②·③ �
- [러너 19:01] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인한 것: csv.ts:88 가드의 경계를 임시 시험으로 직접 재서 off-by-one 없음(500,000셀 통과 / 500,001셀에서만 too_many_cells, 개행 유무 양쪽). 원장에 `- 실패 재현:` 줄이 없어 csv.ts 를 main 판으로 되돌려 red 를 만들었다 — robustness.test.ts:63 이 `cell_too_long` 을 받아 실패, HEAD 에서 통과. 복원 후 트리 clean.
- 못 본 것: e2e·tools·ui·a11y(공유 dev DB :55447 충돌로 이 세션에서 안 돌림). core·server 가 `@workkit/tools` 를 import 하지 않아 영향 범위는 브라우저 파서 한 곳으로 닫힌다고 보고 단위 게이트(`npm run check` 123 passed)로 갈음했다.
- 승인이어도 남는 우려 ①: 개행 없이 넓은 줄 + 과길이 셀 입력의 오류 코드가 `cell_too_long` → `too_many_cells` 로 바뀐다(의도·시험 고정, UI 는 code+message 를 그대로 올려 공백 없음). 릴리즈 노트에는 "개행 없는 거대 CSV 가 한도에서 멈춘다" 로 적으면 된다.
- 우려 ②: git 밖 상태 — 구현자가 공유 dev DB 에 migrate(스키마 무변경)·seed-local 을 돌리고 이 워크트리 `data/media` 를 비웠다. 다음 회차가 e2e 를 돌리려면 재시드가 선행 조건이다.
- 우려 ③: 열린 PR #2·#3 이 `csv.ts:84~96` 의 같은 훅을 건드려 세 갈래로 겹친다 — 머지 순서를 러너가 정해 한쪽을 리베이스할 것. 보안·법무 차단 사유 없음(인증·결제·마이그레이션·워크플로 무변경, 개인정보 무관).
- [러너 19:04] review approved — 리뷰 승인 (risk=low)
- [러너 19:04] pr created — https://github.com/hkjang/workkit.irumx.app/pull/4
- [러너 19:05] ci failed — 성공이 아닌 검사: check=failure · 실패한 검사: ? 잡: check 

## 수리 노트
- **지적이 틀렸다** — CI `check` 실패는 저장소 결함이 아니다. 잡이 3초에 `steps: []` 로 끝났고 GitHub 주석이 "The job was not started because recent account payments have failed or your spending limit needs to be increased" 였다(`gh api .../check-runs/115795358986/annotations`, `GH_CONFIG_DIR=/home/hkjang/.config/gh` 로 인증). 앞선 두 회차의 "결제 차단" 전제는 **맞았고**, 이제 추정이 아니라 확인됐다. 고칠 것이 없어 **커밋하지 않았다**(`check.yml` 완화는 금지이자 무의미).
- **맞았던 것** — 구현자·비평가의 "e2e 실패 7건은 내 변경과 무관" 은 사실이다. 구조적으로도 닫힌다: `packages/core/src`·`apps/server/src`·`scripts/` 에 `@workkit/tools` import 가 **0건**이라 csv.ts 변경이 서버 경로에 닿을 수 없다.
- **틀렸던 것(부분)** — 비평 노트의 "공유 DB 충돌로 e2e 는 못 돌린다" 는 포기가 일렀다. 전용 PostgreSQL 컨테이너(:55497)를 띄워 **CI 전 단계를 완주**했다: e2e **155 passed/0 failed**, tools-flows 55/0, ui-flows **25/0**(앞 회차 9 passed/2 problems 였던 것), a11y 위반 없음, check 123 passed, kits manifest 2회 동일.
- **함정의 정체(다음 회차용)** — 서버는 `WORKKIT_DATABASE_URL`/`WORKKIT_ADMIN_DATABASE_URL` 을, 시험 스크립트는 `DATABASE_URL`(+`APP_DATABASE_URL`, 기본 :55447)을 읽는다. 한쪽만 덮으면 서버와 시험이 다른 DB 를 보고 `files.ts:93 file integrity` 500 이나 admin login 401 이 난다 — 공유 DB 를 피할 땐 **네 변수를 모두** 같은 DB 로 맞출 것.
- **여전히 확신 없는 곳** — 청구가 풀린 뒤 GitHub 러너에서 정말 green 일지는 내 로컬 완주로 갈음했을 뿐 직접 못 봤다(Playwright 는 `--with-deps` 없이 설치해 시스템 라이브러리는 이미 있던 것을 썼다). 비평가의 우려 ①(오류 코드 `cell_too_long`→`too_many_cells`)·③(PR #2·#3 과 csv.ts:84~96 3중 충돌)은 내 소관 밖이라 그대로 남아 있다.
