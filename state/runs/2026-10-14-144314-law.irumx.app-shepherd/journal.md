# PR 처리기 노트 2026-10-14-144314-law.irumx.app-shepherd — law.irumx.app PR #5
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-14-123832-law.irumx.app-improve)
# 회차 노트 2026-10-14-123832-law.irumx.app-improve — law.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:38] base pinned — main@c62b3a0
- [러너 12:38] autonomy release — 

## 구현 노트
- 우선 과제(PR #4 CI 실패)는 코드가 아니라 **Actions 결제 차단**이다(직전 두 회차의 체크런 주석 인용 + 잡 3초). 그래서 두 회차가 미검증으로 남긴 CI 후반 단계를 로컬 docker `law-irumx-dev`(postgres:17-alpine, 55460 — CI 와 같은 포트·비밀번호)에서 끝까지 재현했고, 그 과정에서 발견한 **검증 쪽 결함 하나**를 고쳤다: `scripts/acceptance.mjs` AT-26 의 로그 검사가 빈 로그·틀린 경로에서도 통과(없는 경로는 ENOENT 로 뒤 네 검사를 건너뜀). 양성 대조 1줄 + `existsSync` 로 조였다. scripts 1파일 7줄, 프로덕션 0파일.
- **확신 없는 곳**: ① 양성 대조가 `"channel":"mail"` 콘솔 로그 한 줄의 형식(mailer.ts:71)에 의존한다 — 메일 드라이버가 console 이 아니거나 그 로그 모양이 바뀌면 CI 가 빨개진다. 다만 console 드라이버는 `mails()`(dev 메일함)를 쓰는 acceptance 전체의 전제이고 AT-26 자신도 쓰므로 새 의존은 아니라고 판단했다. ② CI 실패 주석 원문은 **이번 세션에서 직접 재조회하지 못했다**(gh 미인증, 저장소 비공개 → 익명 API 404). 근거는 직전 회차 인용이다.
- **일부러 하지 않은 것**: 워크플로(check.yml)는 한 줄도 건드리지 않았다(느슨하게 만드는 것 금지, 조이는 변경만). AT-26 의 RRN 정규식·`redactSecrets` 정규식도 손대지 않았다(두 파서를 통합하지 말라는 지시). PR #4 의 커밋을 이 브랜치로 끌어오지 않았다 — 러너가 base 를 main@c62b3a0 로 고정했다.
- **다음 역할이 조심할 것**: 이 변경을 검증하려면 DB·서버가 필요하다(`npm run dev:reset` 뒤 `LAW_SERVER_LOG=... node scripts/acceptance.mjs`). 단위 시험(vitest)은 scripts/ 를 포함하지 않으므로 `npm run check` 로는 이 변경이 전혀 증명되지 않는다. 그리고 PR #4 내용(0804a58 + 미머지 19bf148)을 워킹트리에 얹어 전 단계를 돌려 **e2e 27/27, acceptance 26/26, 단위 171건 통과**를 확인했다 — PR #4 는 결제 차단만 풀리면 통과한다.
- [러너 12:57] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인한 것: 단언이 정말 바뀐 경로를 지나는가 — mailer.ts:71 의 로그 줄 모양(JSON 리터럴 순서)과 maskAddress(mailer.ts:146) 출력을 노드로 재현해 양성 대조가 `rr***@example.test` 줄에는 매치, 빈 문자열에는 불일치임을 실측. 아래 음성 `email` 정규식이 가려진 주소에 걸리지 않는 것(= AT-26 영구 실패 아님)도 실측. 기본 경로는 dev-reset.sh:17, CI 는 check.yml:36 과 일치. eslint 통과.
- 양성 대조의 재료는 AT-26 자신의 `newClient('rrn@example.test')` 로그인 메일이고, 그 sendConsole 이 dev 메일함과 로그를 같은 호출에서 만든다 — 구현자의 ①(새 의존 아님) 판단은 타당. 단독 `AT-26` 실행도 성립.
- 못 본 것: DB·서버를 띄운 실제 acceptance 실행(25/26·26/26 수치는 구현자 보고 신뢰). PR #4 의 CI 주석 원문도 재확인 불가.
- 승인이어도 남는 우려(릴리즈 노트용): ① 주석이 "이 실행의 로그"라 하지만 CI server.log 는 e2e 부터 누적 — 증명 범위는 "살아 있는 서버의 로그"까지. ② `LAW_LIVE=1` 경로에서는 양성 대조 실패 + 뒤 4검사가 빈 로그를 공허 통과(AT-26 판정은 전과 같이 실패라 회귀 아님).
- 판정: approve / risk low / blocking 없음. 프로덕션·마이그레이션·워크플로 무변경, revert 로 완전 복구.
- [러너 13:00] review approved — 리뷰 승인 (risk=low)
- [러너 13:00] pr created — https://github.com/hkjang/law.irumx.app/pull/5
- [러너 13:01] ci failed — 성공이 아닌 검사: check=failure · 실패한 검사: ? 잡: check 

## 수리 노트
- 지적(CI check 실패)은 **코드 결함이 아니다**: 잡이 3초에 죽고 로그가 비었다 — `npm ci` 전에 끝난 것이므로 유일한 변경(acceptance.mjs, 잡 후반에만 실행)은 돌지도 않았다. PR #4 를 막았던 Actions 결제 차단과 같은 모양. 그래서 **커밋 없음**.
- 대신 CI 순서를 로컬(docker law-irumx-dev, 55460)에서 끝까지 재현: check(eslint 0 error·tsc·vitest 103/103) → build → migrate → test:rls → e2e 27/27 → 빈 DB 후 acceptance **26/26**(새 단언 통과).
- 변경의 효력도 실측: 빈 로그를 가리키면 새 단언만 `✗ 0B` 로 AT-26 FAIL(변경 전엔 통과). 구현자의 ①(기존 의존) 판단과 비평가의 실측과 일치.
- 확신 없는 곳: CI 실패 주석 **원문은 이번에도 못 봤다**(gh 미인증, 비공개 저장소). 3초·빈 로그라는 간접 증거뿐이다. 비평가의 잔여 우려(누적 server.log, `LAW_LIVE=1` 경로)는 회귀가 아니라 판단해 손대지 않았다.
- 다음 단계: Actions 결제 복구 뒤 `check` 재실행. 로컬에서 할 수 있는 검증은 더 없다.
