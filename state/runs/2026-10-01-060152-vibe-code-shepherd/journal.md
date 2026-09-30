# PR 처리기 노트 2026-10-01-060152-vibe-code-shepherd — vibe-code PR #11
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-01-033250-vibe-code-improve)
# 회차 노트 2026-10-01-033250-vibe-code-improve — vibe-code
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:33] base pinned — main@43fd7a1
- [러너 03:33] autonomy release — 

## 정찰 노트
- PR #10 = `auto/2026-10-01-0103` head `c3d1ddb` (추정: 브랜치 시간순 + 변경 내용). diff 는 `journal.ts`+`journal.test.ts` 뿐이라 코드 결함 가능성이 좁다 — 0단계는 timebox 하고 남는 시간을 실제 결함에 쓰라고 썼다.
- 1단계로 `goals.ts:123`/`handoff-on-exit.ts:48` 을 고른 이유: 같은 파일(`goals/current.md`)에 쓰는 `appendVerificationLog` 만 EOL 을 보존하는 비대칭을 소스에서 직접 확인했고, 두 파일이 열린 5개 브랜치 어디에도 없어 충돌 0 이다. 차선이던 `journal-summary.ts` 를 제친 것은 같은 일지 파일의 다른 writer(`journal.ts:35`)를 PR #10 이 지금 고치고 있어 '한쪽만 고치기' 가 되기 때문이다.
- 확신 없는 것(과제서에 추측으로 적음): PR 번호↔브랜치 대응, `git archive -o` + `tar -xf` 가 샌드박스를 통과하는지(정찰의 파이프 방식은 막혔다), `handoff-on-exit.ts → goals.ts` import 의 순환 여부. 원격 CI 상태는 이번에도 **미확인** — WebFetch 권한 거부, gh 미인증.
- 구현자가 조심할 것: `ci.yml` 금지, `markdown.ts`/`plans.ts`/`goal-lint.ts`/`journal.ts` 는 읽기만(미머지 PR 이 물고 있음), LF 파일의 출력 바이트 무회귀를 기준 2로 반드시 증명할 것, `\s*` 정규식 금지.
- 5회차 연속 verify-failed 의 공통 원인은 코드가 아니라 원격 판정 수단 부재로 보인다. ideas.json 에 운영 항목으로 남겼다.
- [러너 03:38] scout done — 수정 과제 — PR #10(`origin/auto/2026-10-01-0103`, head `c3d1ddb`) CI 실패를 Node 20 로컬 재현�

## 구현 노트
- 0단계: PR #10 head `c3d1ddb` 를 `/tmp/pr10` 에 추출해 ci.yml check job 4단계를 Node 20.19.2 로 재현 — 전부 exit 0(`validation/pr10-node20-check.log`). 원격 실패는 코드 결함이 아니다. `ci.yml` 은 읽기만 했다.
- 1단계: `.vibe-code/goals/current.md` 에 쓰는 세 경로 중 두 개(`goals.ts createHandoffFile`, `handoff-on-exit.ts` 세션 종료)가 `appendFileSync` 로 LF 만 써서 CRLF 파일을 깨뜨렸다. `goals.ts` 에 순수 export `appendGoalNote` + `handoffNote` 를 두고 둘이 같이 쓰게 했다(`appendVerificationLog` 와 같은 계약).
- 확신 없는 것: (a) PR #10 ↔ 브랜치 대응은 정찰의 **추정**이고 원격 CI 상태는 이번에도 **미확인**(gh 미인증·WebFetch 거부) — "원격이 괜찮다" 고 읽지 말 것. (b) `createHandoffFile` 테스트만 `CoreHost` 를 `{ output: { appendLine(){} } }` 로 넘긴다 — `log()` 싱크 전용이고 목표 텍스트·fs·append 경로는 전부 프로덕션이지만, 대역 금지 원칙의 경계선이니 비평가가 여기를 먼저 볼 것. (c) 기존 CRLF 파일에 이미 섞여 있던 LF 줄은 `restoreEol` 이 전체를 CRLF 로 정규화하면서 함께 바뀐다 — `appendVerificationLog` 와 같은 동작이지만 append 이외의 바이트가 움직이는 유일한 경우다.
- 일부러 하지 않은 것: `journal-summary.ts:85·101` 의 같은 계열 LF-only append(같은 일지 파일의 다른 writer `journal.ts:35` 를 PR #10 이 고치는 중 — 한쪽만 고치는 꼴이 된다), `createHandoffFile` 의 `host` 인자 제거(goal-loop.ts 까지 번져 프로덕션 3파일이 된다), 문서(이 append 경로를 단언하는 줄이 docs/·readme 에 없다), 버전·CHANGELOG.
- 다음 역할이 조심할 것: 신규 `tests/unit/goal-handoff.test.ts` 는 실제 임시 fs 에 쓴다 — `TMPDIR=/tmp/vibe-tmp GIT_CEILING_DIRECTORIES=/tmp/vibe-tmp` 를 명령 범위에 함께 주고 돌릴 것. `npm run check` 는 88 tests(기준선 82 + 신규 6). Node 20.19.2 는 대상 디렉터리 **밖**(`/tmp/node2019`)에 설치해야 한다 — 안에 설치하면 `npm ci` 가 지워 `node --check` 가 exit 127 이 된다.
- [러너 03:45] brief accepted — 채택 — 0단계 게이트를 지시대로 Node 20.19.2 에서 4단계 모두 재현해 "코드 결함 아님" 을 증거 로그로 판정하고, 멈추지 �
- [러너 03:45] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인: diff 3파일 전부, goals.ts/handoff-on-exit.ts 소스, markdown.ts 의 detectEol/normalizeEol/restoreEol, src 전체 appendFileSync 잔여(모두 다른 파일 — current.md writer 는 이제 일관), 그리고 validation/goal-eol-red.log 의 수정 전 실패 출력(증상 일치). 로컬 Node 22.23.1 로 88/88 직접 재실행했다.
- 못 본 것: 원격 CI(gh 미인증·WebFetch 거부, 5회차 연속 미확인), Node 20 재실행·dist node --check 는 구현자 로그에만 의존, 실제 Extension Host/VSIX.
- 판정 approve. 구현자가 의심한 세 곳 모두 결함 아님 — CoreHost 대역은 log 싱크 전용이고 텍스트·fs·append 는 전부 프로덕션, import 간선은 원래 있던 것, CRLF 재정규화는 appendVerificationLog 와 같은 기존 계약이다.
- 남는 우려: appendGoalNote 는 append 가 아니라 전체 재작성이라 혼합 EOL 파일에서 메모 밖 바이트도 통일된다 — 릴리즈 노트에 한 줄 필요. goal-handoff.test.ts:96 은 수정 전에도 통과하므로 이번 변경을 고정하지 않는다(다른 5개가 한다).
- 다음 회차: journal-summary.ts:101 의 같은 계열 LF-only append 가 남아 있다. PR #10 머지 후 journal.ts:35 와 함께 한 번에 볼 것.
- [러너 03:48] review approved — 리뷰 승인 (risk=low)
- [러너 03:48] pr created — https://github.com/hkjang/vibe-code/pull/11
- [러너 03:49] ci failed — 성공이 아닌 검사: typecheck + unit tests + build=failure

## 수리 노트
- 틀린 지적: "CI 실패 = 코드 결함". pristine `git archive HEAD` 에서 CI check job 4단계를 Node **20.19.2 와 20.20.2**(= `node-version: 20` 의 실제값) 양쪽으로 돌려 전부 exit 0 / 88·88 tests. TZ=UTC·LC_ALL=C 에서도 88/88 — 신규 테스트의 한국어·KST 의존을 의심해 일부러 지운 변수다.
- 맞은 지적: 없음. 러너가 준 실패 로그는 본문이 **0줄**이고 소요 4s — 프로필의 2026-09-29 run 36518589343(`steps=[]`, `runner_id=0`, 결제/지출한도 안내)과 같은 job-미시작 신호다.
- 고친 방법: **고치지 않았다.** 결함이 없으므로 규칙대로 커밋 없이 끝냈다. 워크트리 clean, tracked 파일 무변경, `ci.yml`·테스트 무수정.
- 확신 없는 곳: "job 미시작" 은 추론이다 — `gh` 미로그인, GitHub API 는 저장소부터 익명 404(비공개)라 run 36761215096 annotation 을 직접 읽지 못했다. 6회차 연속 이 벽에 막혔고, 결제 해소 후 `workflow_dispatch` 재실행만이 확정 수단이다.
- 다음 역할에게: 코드를 다시 의심하기 전에 위 재현 결과를 먼저 반박할 것. 재현 로그는 `/tmp/pr11/{ci,check,ci20,check20,check-locale}.log`(휘발성).
