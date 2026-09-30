# PR 처리기 노트 2026-10-01-070412-vibe-code-shepherd — vibe-code PR #13
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-01-063238-vibe-code-improve)
# 회차 노트 2026-10-01-063238-vibe-code-improve — vibe-code
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 06:32] base pinned — main@43fd7a1
- [러너 06:32] autonomy release — 

## 정찰 노트
- 배정이 fix-round 라 0단계는 정해져 있었다. PR #12 = `origin/auto/2026-10-01-0535` head `2a264c8`(직전 회차의 journal EOL 커밋)임을 `git log -1`·`git diff --stat main...` 로 확인했다. 1단계는 미머지 PR 여섯 건이 물고 있는 7개 파일을 전부 피할 수 있는 곳에서 골랐다 — `goal-metrics.ts` 의 감사 읽기 창(`slice(-days)` = 파일 개수)이 `formatMetrics`/`buildRetro` 가 단언하는 날짜 창과 어긋나는 것. 운영자가 되풀이한 "같은 값을 쓰는 쪽과 읽는 쪽이 같게 보는지 end-to-end 확인" 에 정확히 맞고 프로덕션 1파일로 끝난다.
- 제친 후보: markdown.ts/plans.ts 계열(#6~#9 4중 충돌), ci.yml package job(보호 경로 + 워크플로 완화 금지 + Windows 필요), vibe-coders-proxy(전역 설정 보호 경로).
- 추측으로 적은 것: PR 번호 ↔ 브랜치 대응(이 세션에 GitHub 접근 없음). `activeDays` 를 KST 로 바꿔도 기존 기대값 3 이 유지된다는 판단은 픽스처 ts 가 01~05Z 인 것을 읽고 계산한 것이며 **실행으로 확인하지 못했다** — 이 세션은 `npx vitest run` 조차 권한 거부됐다. 테스트 기준선(82/11 files)도 실측이 아니라 이전 회차 값이다.
- 구현자가 조심할 것: `tests/unit/vscode-stub.ts:4` 의 `workspaceFolders: undefined as undefined` 타입이 좁아 대입에 국소 캐스트가 필요하다(공유 스텁을 고치면 미머지 PR #12 테스트와 충돌). `journal-summary.ts` 의 `parseAuditLines` 는 import 만, 파서 통합 금지. 출력 문구는 기존 테스트 6개가 고정하므로 창만 고칠 것.
- [러너 06:37] scout done — 수정 과제 — PR #12 head `2a264c8` 로 CI check job 로컬 재현 판정 + `readRecentAudit` 이 "최근 N일" 을 "최근 N개 파일"

## 구현 노트
- 0단계 통과(PR #12 head `2a264c8`, Node 20.19.2, `npm ci`/`npm run check`/`node --check` 두 번들 전부 exit 0 — `validation/pr12-node20-check.log`). 일곱 회차 연속 "로컬 전부 통과 = 코드 결함 아님". 그래서 1단계: `goal-metrics.ts` 의 감사 읽기 창을 파일 개수(`slice(-days)`) → KST 날짜(`n.slice(0,10) >= from`)로, `activeDays` 를 UTC → `kstDate` 로 고쳤다. 프로덕션 1파일 · 테스트 1파일. 커밋 `1dd2151`.
- **확신 없는 곳**: (1) PR 번호 ↔ 브랜치 대응은 정찰의 추정을 그대로 썼다 — 이 세션에 GitHub 접근이 없어 PR #12 의 원격 실패 사유·"같은 이유로 두 번 실패" 를 **확인하지 못했다**. (2) CI `package` job(windows-latest)은 이번에도 미재현 — Windows 수단이 없다. (3) 과제서가 지정한 `fs.copyFileSync` 대신 `vi.setSystemTime({toFake:["Date"]})` 으로 시계를 옮겨 프로덕션 `writeAudit` 이 과거 날짜 파일을 스스로 만들게 했다. 복사는 파일명만 과거가 되고 `ts` 는 오늘로 남아 수용 기준 3(activeDays 의 KST 경계)을 증명할 수 없기 때문인데, **fake timer 를 이 저장소에서 쓴 것은 이번이 처음**이다(기존 테스트에 선례 없음) — 비평가가 먼저 볼 곳. `writeAudit` 의 `host` 는 catch 경로의 `log` 에만 닿아 최소 캐스트로 넘겼고, 대신 로그가 비었음을 단언해 모든 쓰기가 실제로 착지한 것을 확인한다.
- **일부러 하지 않은 것**: 차선 후보(`exitCode` 누락을 통과로 세는 `:51-52`)를 같은 커밋에 얹지 않았다 — 같은 파일 한 줄이지만 창 수정과 섞으면 인과 귀속이 흐려진다. 범위는 이미 검증했으니 다음 회차가 바로 집을 수 있다(`ideas.json` 에 계약 결정 메모까지 남겼다). `formatMetrics`/`buildRetro` 의 출력 문구·형식, `days` 호출부(둘 다 7), 공유 `vscode-stub.ts`, `ci.yml`, 미머지 PR 이 물고 있는 7개 파일은 건드리지 않았다.
- **다음 역할이 조심할 것**: 새 `describe("audit window")` 는 fake timer 로 시스템 시각을 2026-09-22 로 고정하고 `vscode-stub` 의 `workspace.workspaceFolders` 를 임시 디렉터리로 **대입**한다 — `afterEach` 가 `useRealTimers()` 와 `workspaceFolders = undefined` 로 되돌리므로 이 파일 안에서는 격리되지만, 같은 스텁을 대입하는 테스트를 새로 더하면 파일 간 실행 순서에 의존하지 않는지 확인할 것(vitest 는 파일별 모듈 격리라 현재는 안전). 기존 기대값 4개는 한 줄도 바꾸지 않았고 `activeDays` 3 은 픽스처 ts 가 01~05Z(KST 10~14시)여서 KST 로 바꿔도 그대로다 — 실행으로 확인했다. 기준선은 실측: main@43fd7a1 = 11 files / 82 tests → 지금 11 files / 87 tests.
- [러너 06:45] brief accepted — 채택 — 0단계 게이트를 지시대로 Node 20.19.2 에서 4단계 모두 재현해 "코드 결함 아님" 을 증거 로그로 판정하고, 멈추지 �
- [러너 06:45] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 인과를 직접 재현했다: Node 20.19.2 에서 HEAD 는 typecheck·87/87 tests·build·`node --check` 두 번들 전부 exit 0, `src/features/goal-metrics.ts` **만** `git checkout main --` 로 되돌리면 정확히 3개(`completedItems` 5개, `activeDays 2`, 회고에 `두 달 전 항목`)가 빨개진다 — 원장의 실패 재현이 사실이고 테스트가 바뀐 경로를 지난다. 경계(`2026-09-16` 포함/`2026-09-15` 제외)도 테스트가 고정하고, KST 는 DST 가 없어 `- (days-1)*864e5` 산술이 정확하다.
- 못 본 것: 원격 CI(gh 미인증·push 불가)와 windows-latest `package` job — 여덟 회차째 미확인이며 이 diff 의 결함이 아니다. 실제 Extension Host·VSIX 도 확인 못 했다.
- 승인이어도 남는 우려 ①: 창의 **상한이 없다**. 시계가 뒤로 간 워크스페이스나 손으로 만든 `.vibe-code/audit/2027-01-01.jsonl` 은 여전히 읽혀 `from ~ to` 제목 밖의 항목이 회고에 섞인다 — 이번에 고친 라벨/데이터 불일치의 반대쪽 끝이다(main 도 같았으므로 회귀는 아님). `&& n.slice(0,10) <= kstDate()` 로 닫을 수 있다.
- 우려 ②: `days=0` 의 의미가 뒤집혔다 — `slice(-0)` 은 **전체** 파일이었고 지금은 0개다. 호출부 둘 다 7 이라 현재 영향은 없지만 새 호출자가 0 을 넘기면 정반대 결과를 얻는다. 우려 ③(**릴리즈 노트에 넣을 것**): 최근 7일에 활동이 없는 사용자는 이제 지표·회고가 비어 나온다(전에는 마지막 활동 7개 파일을 보여줬다). 제목이 이미 달력 창을 주장했으므로 의도된 수정이지만 사용자 눈에 보이는 변화다.
- 보안·법무 차단 없음: 새로 수집·전송·보존하는 개인정보가 없고 오히려 읽는 감사 파일 범위를 좁힌다. 파일명은 regex 검증 후 사용, `from` 은 시계에서 파생(사용자 입력 아님), 인증·비밀값·의존성·워크플로·`workspace.ts:66` 미접촉. 다음 회차는 원장의 보류 아이디어(`goal-metrics.ts:55-56` 의 `exitCode` 누락을 통과로 세는 것)를 먼저 볼 것.
- [러너 06:49] review approved — 리뷰 승인 (risk=low)
- [러너 06:49] pr created — https://github.com/hkjang/vibe-code/pull/13
- [러너 06:50] ci failed — 성공이 아닌 검사: typecheck + unit tests + build=failure

## 수리 노트
- **지적이 틀렸다 — 코드 결함이 아니라 인프라다. 커밋하지 않았다.** CI `check` 네 단계를 CI 와 같은 Node 20.19.2 로 HEAD `1dd2151` 에서 재현했고 전부 exit 0 이었다(`npm ci` · `npm run check` = 11 files / 87 tests 통과 · `node --check` 두 번들). 실패한 검사는 **4초**에 끝났고 과제서의 실패 로그 꼬리가 **비어 있다** — 보여줄 step 로그가 없었다는 뜻이고, 프로필에 기록된 선례(run 36518589343, `steps=[]`, `runner_id=0`, "job was not started … recent account payments have failed or your spending limit needs to be increased")와 같은 모양이다.
- **고친 방법: 없음.** 프로덕션·테스트 모두 무수정. 절대 규칙(테스트 약화 금지, 워크플로 완화 금지)에 따라 근거만 `fix-summary.md` 에 남겼다. 러너는 결제·지출 한도를 확인한 뒤 이 커밋 그대로 CI 를 재실행하면 된다.
- **환경 가설을 추측이 아니라 실행으로 배제했다:** TZ(`UTC`/`America/Los_Angeles`/`Pacific/Kiritimati` 전부 87/87 — `kstDate` 는 `Intl` `timeZone:"Asia/Seoul"` 로 기계 TZ 무관), Node 드리프트(25.9.0 에서도 87/87), 머지 충돌(`origin/main` == `HEAD^` == `43fd7a1` 이라 fast-forward, `refs/pull/13/merge` 충돌 불가), 락파일(깨끗한 `node_modules` 에서 `npm ci` 0).
- **테스트가 비어 있지 않음도 직접 확인했다:** `goal-metrics.ts` 만 main 으로 되돌리면 `TZ=UTC` 에서 정확히 3개가 빨개지고(`completedItems`·`activeDays 2`·회고의 `두 달 전 항목`) 복원하면 녹색이다. 앞선 비평의 재현과 일치한다.
- **여전히 확신 없는 곳:** 원격 annotation 원문을 이 회차에서 못 읽었다 — `gh` 미인증, 공개 API 는 private 저장소라 404. 4초·빈 로그·선례로부터의 추론이다. windows-latest `package` job 은 아홉 회차째 미재현(`needs: check` 로 skip). 비평의 잔여 우려 ①창 상한 없음 ②`days=0` 의미 반전 ③활동 없는 사용자에게 빈 지표는 **범위 밖이라 손대지 않았다** — 승인된 diff 의 의도이고 CI 실패와 무관하다.
