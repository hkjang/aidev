# 회차 노트 2026-10-14-090819-workkit.irumx.app-improve — workkit.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 09:08] base pinned — main@8bffd36
- [러너 09:08] autonomy release — 

## 정찰 노트
- 고른 이유: `detectDelimiter` 의 첫-줄 고정 점수는 직전 커밋(74458bf "제목 줄 위 표")이 공식 지원하기로 한 파일 모양과 정면으로 어긋난다 — 읽어서 확인한 결함이고, 생산 코드 1파일·시험 1파일로 끝난다. 날짜 직렬값(1900 윤년)·docx CRLF 는 가치가 낮아 제쳤고, XLSX zip/매크로 가드는 이미 촘촘해 손댈 데가 없었다.
- 추측으로 적은 것: 제목 줄에 쉼표가 든 세미콜론 CSV 를 **실제로 돌려 보지는 못했다**(워크트리에 node_modules 가 없어 vitest 가 없고 예산상 npm ci 를 생략). 코드를 손으로 따라가 score 1001 vs 0 을 계산한 결과다. 구현자는 먼저 실패하는 시험부터 쓸 것.
- 조심할 것: `parseCsv` 와 `compare.worker.ts` 는 열지 말 것(파서 통합 금지 교훈). 기존 단언 compare.test.ts:46-47 의 기대값을 바꿔야 한다면 구현이 틀린 것이다.
- [러너 09:12] scout done — CSV 구분자 자동 감지가 "제목 줄"에 끌려가지 않게 (가치 4 / 위험 2 / 작업량 S)

## 구현 노트
- 무엇/왜: `detectDelimiter`(packages/tools/src/table/csv.ts) 의 점수 기준을 "첫 줄의 구분자 개수" 에서 "줄들 사이 최빈 개수" 로 바꿨다 — `score = hits * 1000 + modal`. 제목 줄의 쉼표 하나가 본문 구분자를 이기는 결함이 사라진다. 생산 코드 1파일 + 시험 1파일.
- 확신 없는 곳: (1) 빈도 동점 시 **더 큰 개수를 고르도록** 했다(`n === hits && count > modal`). 과제서가 지정하지 않은 부분이라 내 판단이다 — counts 가 [1,2] 처럼 각 1회씩인 2줄 파일에서만 갈리고, 후보 간 `score` 의 끝자리에만 영향한다. 틀렸다고 보면 `n > hits` 만 남기면 된다(그 경우 Map 삽입 순서 = 줄 순서로 결정).
- 확신 없는 곳: (2) `npm run tools`(5만 행 Playwright)·`npm run e2e` 는 돌리지 않았다 — Playwright·PostgreSQL·가짜 서버가 없다. 감지 함수만 바뀌었고 `compare.worker.ts:88` 은 호출부라 배선 변화는 없지만, 실제 브라우저 흐름은 미검증이다.
- 일부러 안 한 것: 같은 함수 안의 `text.split(/\r?\n/)` 전체 쪼개기 성능(별건, ideas.json pending), `parseCsv`·`xlsx-read.ts`·`compare.worker.ts` 는 열지 않음(파서 통합 금지).
- 다음 역할이 조심할 것: 새 시험 4건은 전부 순수 단위/인메모리라 DB·브라우저 없이 돈다. 검증은 `npx vitest run packages/tools/src/__tests__/compare.test.ts packages/tools/src/__tests__/robustness.test.ts` (31 passed) 와 `npm run check` (122 passed) — 둘 다 통과 확인했다. 워크트리에 `npm ci` 가 필요하다.
- [러너 09:15] brief accepted — 채택 — 근거(`counts[0]` 고정, 직전 커밋 74458bf 의 제목 줄 지원, 권장 구현·검증 명령)가 지금 코드와 정확히 맞았고 제안�
- [러너 09:15] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인함: 두 구현을 떼어 내 나란히 돌려 실패 재현을 검증했다 — 새 시험 2건은 main 에서 정말 갈린다(제목 줄+세미콜론 `,`→`;`, 제목 줄+탭 `,`→탭), 원장의 재현 출력과 증상이 맞는다. `npm run check` 직접 실행(122 passed), 워크트리 무변경, 범위 이탈·위험 구역 접촉 없음. 보안·법무 차단 사유 없음(순수 브라우저 측 파싱, 새 의존성·개인정보·외부 상태 없음). 판정 approve / risk low.
- 승인이어도 남는 우려(릴리즈 노트에 "구분자 자동 감지 기준 변경" 한 줄 권장): 헤더 고정을 버린 대가로 **표 아래 주석·합계 줄**이 짧거나 들쭉날쭉한 표를 이길 수 있다 — `id,name,dept` + 3행 뒤에 `note;1`…`note;5` 가 오면 `;` 로 감지된다(main 은 `,` 로 옳게 봤다). CompareTool.tsx:636-649 의 수동 구분자·머리글 행 선택으로 한 번에 복구되므로 차단하지 않았다.
- 다음 회차가 알 것: 구현자가 자기 의심으로 적은 동점 처리(`n === hits && count > modal`)는 구현 노트가 말한 것보다 넓게 작동해 hits 동점인 **모든 짧은 파일에서 승자를 결정**한다(`a,b` / `1,2` / `x;y;z;w` / `p;q;r;s` → `;`). 더 손댈 거면 `guessHeaderLine`(types.ts:82)처럼 "표처럼 생긴 줄" 에 가중치를 주는 방향.
- 못 본 것: `npm run tools`·`e2e`·`ui`·`a11y`(Playwright·PostgreSQL·가짜 서버 필요) 미실행 — 호출부 `compare.worker.ts:88` 은 diff 밖이고 배선이 그대로여서 실제 브라우저 흐름은 코드 독해로만 확인했다.
- [러너 09:21] review approved — 리뷰 승인 (risk=low)
- [러너 09:21] pr created — https://github.com/hkjang/workkit.irumx.app/pull/1
- [러너 09:27] ci passed — 검사 1개 모두 success
- [러너 09:27] merge done — 9b35b5f
- [러너 09:30] release published — 0.3.1
