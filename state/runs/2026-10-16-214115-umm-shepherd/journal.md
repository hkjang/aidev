# PR 처리기 노트 2026-10-16-214115-umm-shepherd — umm PR #172
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-16-205839-umm-improve)
# 회차 노트 2026-10-16-205839-umm-improve — umm
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 20:58] base pinned — main@745dcf3
- [러너 20:58] autonomy low-risk — 회귀(reverted) 2026-10-06T22:38:39+09:00

## 정찰 노트
- 설치 가드·x/text 계열은 PR #170/#171 이 베이스에 들어와 done 처리했고, 남은 후보 중 AppLayout 의 'My Space'(가치 3)는 AppLayout.tsx:273 의 `t(label).split(' ')[0]` 때문에 '내 공간'→'내' 가 되어 짧은 라벨 설계가 선행돼야 해 M·위험 3 으로 밀었다. test-go 확장은 main 에 미반영이지만 verify-failed 전력이라 제외했다.
- 그래서 web/src/lib 에서 유일하게 테스트가 없는 edge-vocabulary.ts 를 골랐다. 단순 커버리지가 아니라 relationOptions(:31)가 EdgeRelation(api.ts:86)과 컴파일 시점 연결이 없는 실제 드리프트 구멍을 시험 파일의 exhaustive Record 로 닫는 것이 핵심이다.
- 추측으로 적은 것: ① api.ts 에 멤버를 더하면 typecheck 가 깨진다는 것은 tsconfig.app.json include=["src"] 로부터의 추론이고 실행 확인은 안 했다(과제서가 구현자에게 임시 멤버로 찍고 되돌리라고 지시함). ② 차선 후보의 미설치 format 실패 문구/종료코드도 미확인. ③ migrations/010 의 CHECK 제약은 이번에 재확인하지 않고 과거 기록에 근거했다.
- 이번 정찰은 web/node_modules 가 없어 어떤 npm 명령도 돌리지 않았다. 기준선 시험 수는 미측정이므로 구현자가 시험 추가 전에 한 번 돌려 기준선을 찍어야 한다.
- 구현자 주의: setLocale 은 전역이라 afterEach 복원 필수, en.ts 문구는 고정 대상이지 개선 대상이 아니며, 폴백(?? relation) 시험은 넣지 말 것. 프로덕션 파일 0개로 끝나야 한다.
- [러너 21:04] scout done — `edge-vocabulary.ts` 의 연결 어휘 12개와 `relationOptions` 를 실제 타입·실제 `translate`·실제 `en.ts` 로 못 박기 (가�

## 구현 노트
- 무엇을 왜: `web/src/lib/edge-vocabulary.test.ts` 한 파일만 추가(커밋 a5959a5, **프로덕션 0파일**). 라벨 12개를 실제 `setLocale`/`translate`+실제 `en.ts` 로 ko·en 양쪽 고정하고, `Record<EdgeRelation|EdgeOrigin, …>` **타입 애너테이션**으로 유니온 확장을 컴파일 오류로 만든 뒤 `relationOptions` 가 같은 멤버임을 실행으로 단언했습니다. 24파일/260시험(기준선 23/252).
- 확신 없는 곳: 없는 쪽에 가깝지만 두 가지만 비평가가 보세요. ① `new Set(...)` 비교는 **중복은 못 잡습니다** — 중복은 바로 아래 정확한 배열 순서 시험이 잡는다는 전제에 의존합니다. ② `migrations/010` 의 CHECK 제약은 저도 **열어 보지 않았습니다**(과제서 지시대로 폴백 시험을 넣지 않았으니 결론에 영향은 없습니다).
- 일부러 하지 않은 것: `?? relation` 폴백 시험(수용기준 2번의 exhaustive 망과 의미가 충돌), `en.ts` 문구 개선, 사전/파서 통합(운영자 규칙 3번), `isInferred` 삭제(프로덕션 0파일 제약 — 지금 `web/src` 어디서도 호출되지 않는 export 임은 `grep` 으로 확인, ideas.json 에 남겼습니다), `tsconfig` 의 `lib` 올리기.
- 과제서 대비 바뀐 한 줄: 집합 비교를 `[...].sort()` 로 쓰면 oxlint 가 새 파일에 `unicorn(no-array-sort)` 2건을 내고 권고대로 `toSorted()` 를 쓰면 `error TS2550`(lib < es2023) 이라, `new Set(...)` 비교로 바꾼 뒤 드릴을 **그 최종 형태로 다시** 돌려 빨개지는 것을 확인했습니다.
- 다음 역할 주의: `npm ci --prefix web` 가 **반드시 선행**해야 합니다(이 회차 시작 때도 `web/node_modules` 가 없었습니다). DB·Go·브라우저는 필요 없고 `make test-web` 8게이트로 전부 검증됩니다. `setLocale` 은 모듈 전역이라 이 시험의 `afterEach(() => setLocale('ko'))` 를 지우면 뒤 시험이 영어로 돕니다.
- [러너 21:10] brief accepted — 채택 — 과제서의 근거가 지금 코드와 전부 맞았습니다(`edge-vocabulary.ts` 만 인접 테스트 없음, `relationOptions:31` 의 배열 타�
- [러너 21:11] verify passed — 검증 17개 통과 (auto)

## 비평 노트
- 승인. 드릴을 직접 재현했다: `api.ts:86` 에 `'blocks'` 를 임시로 더하니 `edge-vocabulary.test.ts:20` 이 TS2741 로 빨개졌고 api.ts 는 즉시 원복(worktree clean). 시험은 실제 `setLocale`/`translate`/`en.ts` 를 지나가며 '항상 참' 단언이 없다. 게이트: test 24/260·typecheck·lint(신규 파일 경고 0)·check-i18n 1060키·build 전부 통과. 안 돌린 것: audit(네트워크)·verify:pwa·offline-queue·e2e·Go.
- 같은 드릴에서 `edge-vocabulary.ts:8` 도 함께 TS2741 을 냈다 — 라벨 표는 프로덕션이 이미 지키고 있었고, 이번에 새로 닫힌 구멍은 `relationOptions`(:31) 하나다. 그래서 머리 주석의 "모듈 자신은 할 수 없는 단 하나"는 문자 그대로는 과장이다(콜론 뒤 설명은 맞다). 결함이 아니라 문구 정밀도 — 다음에 그 한 줄만 좁혀 쓰면 된다.
- 구현자가 의심한 두 곳 다 괜찮다: `new Set` 의 중복 맹점은 바로 아래 배열 전체 고정 시험이 덮고, `migrations/010` 는 이번 변경이 DB 에 닿지 않아 무관하다.
- 남는 우려 둘(릴리즈 노트·다음 회차용): ① dream 시험은 "en.ts 에 'Dream' 키가 없다"는 상태에 묶여 있다(키 부재는 직접 확인). ② `isInferred` 는 `web/src` 어디서도 호출되지 않는 export 인데 이 시험이 고정하므로, 지울 때 마지막 `it` 도 같이 지워야 한다.
- 보안·법무 차단 없음: 프로덕션 0파일, 신규 의존성·엔드포인트·개인정보·비밀값 없음. revert 는 파일 1개 삭제. 커밋 트레일러 없음, VERSION 미변경 — 관례 일치.
- [러너 21:14] review approved — 리뷰 승인 (risk=low)
- [러너 21:14] pr created — https://github.com/hkjang/umm/pull/172
- [러너 21:18] ci failed — 성공이 아닌 검사: verify=failure · 실패한 검사: ? 잡: verify 

## 수리 노트
- 맞았던 것: verify 실패는 실재한다 — 로컬에서 CI 와 같은 govulncheck 명령으로 동일한 stdlib 12건·exit 1 을 재현했다. 틀렸던 것: 그 실패를 이 PR 의 결함으로 보는 것 — `origin/main`(745dcf3)을 별도 worktree 로 꺼내 돌려도 같은 12건이 나왔고, diff 는 `web/src/lib/edge-vocabulary.test.ts` 한 파일(Go 파일·go.mod·go.sum 0개)이라 Go 스캔 입력이 베이스와 같다.
- 고친 방법: **없음, 커밋 0개.** 원인은 go.mod 의 `toolchain go1.26.7` 이 go1.26.9 보안 패치보다 뒤진 것이고, 손대면 무관한 go.mod·go.sum·Dockerfile 을 건드리게 돼 절대 규칙 2번에 걸린다. 처방은 fix-summary.md 에 실측과 함께 적었다(toolchain 1.26.9 + x/net v0.60.0 → `No vulnerabilities found.` exit 0, toolchain 만으론 4건 잔존).
- 확신 있는 것: 이 변경 자체는 `make test-web` 8게이트 exit 0(vitest 24/260, audit 0건, build, verify:pwa 150자산). CI 는 Go 절에서 멈춰 프런트 게이트를 **한 번도 실행하지 않았다**(ci.yml 의 프런트엔드 절이 govulncheck 뒤에 있다).
- 확신 없는 곳: ① 처방의 Dockerfile `golang:1.26.9-alpine` 은 태그 존재만 확인했고 `docker build` 는 돌리지 않았다 ② x/crypto v0.54.0→v0.57.0 이 internal/auth 에 미치는 영향은 `go test` 로 확인하지 못했다(이 worktree 에 POSTGRES_DSN 없음) — 처방을 집행하는 회차가 둘 다 실제로 돌려야 한다.
- 다음 역할에게: 이 베이스 결함은 PR #172 뿐 아니라 **지금 열리는 모든 PR 의 verify 를 막는다**. 중재가 이 PR 을 그대로 통과시킬 수 없다면, 의존성·툴체인 전용 PR 을 먼저 머지한 뒤 이 브랜치를 재실행하는 순서가 맞다.
