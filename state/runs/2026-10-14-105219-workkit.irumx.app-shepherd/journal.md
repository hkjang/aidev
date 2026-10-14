# PR 처리기 노트 2026-10-14-105219-workkit.irumx.app-shepherd — workkit.irumx.app PR #2
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-14-095843-workkit.irumx.app-improve)
# 회차 노트 2026-10-14-095843-workkit.irumx.app-improve — workkit.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 09:58] base pinned — main@53ea324
- [러너 09:58] autonomy release — 

## 정찰 노트
- 직전 회차가 차선으로 남긴 1순위(빈 줄만 든 CSV 가 `LIMITS.rows` 검사를 우회)를 지금 코드에서 다시 확인해 그대로 골랐다 — csv.ts:91~96 의 한도 검사가 "내용 있는 레코드" 분기 안에만 있는 것, csv.ts:99 가 빈 행을 무조건 push 하는 것, `LIMITS.fileBytes`=10MB 를 직접 봤다. XLSX 병합 제목 줄·serialToIso 1900 윤년은 재현 시험이 없어 한 세션에 안 끝날 위험이 있어 제쳤고, 구분자 선택 UI 는 Playwright 가 필요해 제쳤다.
- 추측으로 적은 것: ① "브라우저가 멈춘다"는 1,000만 개 빈 행 객체 할당을 근거로 한 추론이고 실제로 재현해 보지는 못했다 ② 이 워크트리에 `node_modules` 가 없어(확인함) 시험을 한 번도 돌리지 못했다 — 과제서의 기대 실패 메시지 문구는 추정이다.
- 확인한 것: `LIMITS` 는 `packages/tools/src/index.js` 에서 나온다(compare.test.ts:11 이 이미 import). `robustness.test.ts` 의 import 목록에는 아직 없으므로 시험을 추가할 때 넣어야 한다.
- 구현자가 조심할 것: 행 번호(`line: rows.length + 1`)가 빈 행이 `rows` 에 들어 있다는 전제에 의존한다 — 메모리를 더 줄이려고 빈 행 push 를 없애면 사용자에게 보이는 행 번호가 전부 밀린다. 한도 검사 추가만으로 끝낼 것. `detectDelimiter`(직전 커밋 53ea324) 와 다른 경로 파서(xlsx-read, compare.worker)는 손대지 말 것.
- 프로필은 0일 전 것이고 이번에 읽은 코드(csv.ts, types.ts, 시험 배치, node_modules 없음)와 어긋나지 않아 새로 쓰지 않았다.
- [러너 10:01] scout done — 빈 줄만 가득한 CSV 가 행 한도 검사를 우회해 브라우저를 멈추는 것 막기 (가치 3 / 위험 2 / 작업량 S)

## 구현 노트
- 바꾼 것: `packages/tools/src/table/csv.ts` 의 `end()` 에서 `LIMITS.rows` 검사를 `push()` 직후 한 줄로 올려 "내용 있는 레코드" 분기와 "빈 줄" 분기가 공유하게 했다. 개행만 든 10MB CSV 가 약 1,000만 개 빈 행 객체를 쌓은 뒤 `empty` 로 끝나던 것을 50,001번째 줄에서 `too_many_rows` 로 즉시 끝내려는 것. 프로덕션 파일 1개 + 시험 1개.
- 확신 없는 곳·검증 못 한 것: (1) 실제 10MB 빈 줄 파일로 브라우저 워커가 멈추는 것을 **직접 재현하지 않았다** — 단위 시험(`LIMITS.rows + 2` 줄)으로 코드 경로만 증명했다. 메모리 폭주 주장은 코드 읽기 기반이다. (2) `cell_too_long`(push 안) → `too_many_rows` 순서를 유지했다고 판단하지만 그 순서를 고정하는 시험은 없다. (3) `apps/web` 워커 경로(compare.worker.ts:89)를 지나는 end-to-end 확인은 못 했다 — Playwright 가 필요해 일부러 안 돌렸다.
- 일부러 하지 않은 것: 빈 행 push 제거(행 번호 `line: rows.length + 1` 이 밀려 대조 보고서가 깨진다), `detectDelimiter` 의 split 범위 축소(별건, 다음 회차 후보), `xlsx-read.ts`·`compare.worker.ts` 파서 통합(운영자 지시로 금지).
- 다음 역할이 조심할 것: `npm ci` 먼저 — 워크트리에 node_modules 가 없다. 검증은 `npm run check` 로 끝난다(DB·Playwright 불필요). `npm run e2e`·`tools`·`ui`·`a11y` 는 이 변경과 무관하고 외부 의존이 있어 돌리지 않았다.
- [러너 10:04] brief accepted — 채택 — 근거(csv.ts:93 의 행 한도 검사가 내용 분기 안에만 있음, 빈 줄 분기 csv.ts:99)가 지금 코드와 정확히 맞았고, 권장�
- [러너 10:04] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인: diff 2파일 전부 읽고, parseCsv 를 JS 로 그대로 옮겨 고치기 전/후를 돌렸다 — 고치기 전 '넓은 행 1 + 빈 줄 1,000만' 은 `empty` 가 아니라 **힙 OOM**(line 127 패딩이 1,000만 x 200셀)이었고 고친 후 11ms 에 too_many_rows. 새 시험은 고치기 전 `code:'empty'` 로 실제로 실패한다(원장 실패 출력과 일치). `npm run check` 직접 통과(123 passed).
- 승인이어도 남는 우려 ①: 데이터 행이 **정확히 50,000**이고 파일이 빈 줄로 끝나면(끝 `\n\n`) 이제 거부된다 — 릴리즈 노트에 "빈 줄도 행 수에 포함" 을 한 줄 적으면 문의를 줄인다. 한도 검사는 고치기 전에도 빈 줄을 세고 있었으므로 새 불일치는 아니다.
- 우려 ②(다음 회차): 개행 없이 구분자만 든 파일은 `end()` 가 안 불려 행·셀 한도를 여전히 우회한다(10MB 쉼표 = record 1,000만). 이번 범위 밖이지만 같은 계열의 남은 구멍이다.
- 못 본 것: compare.worker.ts end-to-end, `npm run e2e`·`tools`·`ui`·`a11y`(외부 의존, 이 순수 로직 변경과 무관해 생략).
- 보안·법무 차단 사유 없음(인증·결제·마이그레이션·개인정보·의존성 무관, 자원 한도를 좁히는 방향).
- [러너 10:08] review approved — 리뷰 승인 (risk=low)
- [러너 10:09] pr created — https://github.com/hkjang/workkit.irumx.app/pull/2
- [러너 10:09] ci failed — 성공이 아닌 검사: check=failure · 실패한 검사: ? 잡: check 

## 수리 노트
- 지적이 틀렸다: CI `check` 실패는 코드와 무관한 **GitHub Actions 결제 차단**이었다 — 잡 주석 "The job was not started because recent account payments have failed or your spending limit needs to be increased", API 로 본 `steps` 가 빈 배열(2초 failure, checkout·`npm ci` 도 안 돌았다). 러너가 로그 끝부분을 비워 보낸 이유도 이것(로그가 없다).
- 맞았던 것: 비평 노트의 검증 결론은 재현됐다. 로컬 `npm run check` = lint·typecheck 통과 + vitest **123 passed**, 그리고 CI 가 못 간 단계도 `npm run build` 성공 · `npm run kits` 2회 빌드 manifest `cmp` 동일 · `npm run kits:test` 키트 3개 통과.
- 고친 방법: 없음 — 소스 무수정, 커밋 없음. 규칙상 CI 워크플로를 고쳐 통과시키는 것은 금지이고, 애초에 워크플로 내용 문제가 아니다. 대신 `csv.ts` 만 `origin/main` 으로 되돌려 새 시험이 `code:'empty'` 로 실패하는 것(red)과 복구 후 통과(green, 5 passed)를 직접 확인해 회귀 시험이 대상을 잡고 있음을 증명했다.
- 여전히 확신 없는 곳: 외부 의존 단계(`e2e`·`tools`·`ui`·`a11y`)는 DB·Playwright·가짜 서버가 없어 **미검증**이다. 이 변경은 `packages/tools` 순수 로직이라 영향이 없다고 보지만 돌려 본 것은 아니다. 비평가가 전에 적은 우려 ②(개행 없이 구분자만 든 파일은 `end()` 가 안 불려 한도 우회)는 이번 범위 밖으로 그대로 남아 있다.
- 다음 역할이 할 일: 코드가 아니라 계정 쪽이다 — Billing & plans 에서 결제·지출 한도를 풀고 run 38494124742 재실행. 그때까지 CI 는 무슨 커밋을 올려도 같은 자리에서 실패한다.
