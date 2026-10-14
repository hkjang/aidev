# PR 처리기 노트 2026-10-14-131017-law.irumx.app-shepherd — law.irumx.app PR #5
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
- 맞은 지적: CI `check` 가 빨간 것은 사실. **틀린 것은 원인 귀속** — 코드가 아니라 Actions 결제/지출한도 차단이다(잡 3초 + 실패 로그 공백 = steps 미실행; 이 워크플로는 apt-get + npm ci 만으로도 3초 불가). 같은 저장소 110037 회차가 받아 둔 체크런 주석 원문("The job was not started because recent account payments have failed or your spending limit needs to be increased")과 신호가 일치한다. 이번에도 gh 미인증이라 **주석 직접 재조회는 못 했다**.
- 그래서 **고치지 않았고 커밋도 없다**(절대 규칙: 워크플로를 고쳐 통과시키는 것 금지). 워킹트리는 ffd2bf6 그대로.
- 대신 앞 두 역할이 "못 본 것"으로 남긴 것을 메웠다 — CI 전 단계를 로컬 docker(55460, CI 동일)에서 완주: check 통과(lint 0 errors/646 warnings, vitest 6 files/103) · build · migrate · **rls-probe** · **e2e 27/27** · 빈 DB 후 **acceptance 26/26**(AT-26 포함). 비평이 신뢰로 넘긴 수치를 이제 실측으로 바꿨다.
- 단언 유효성도 대조로 못 박았다(빈 DB + AT-26 단독): **빈 로그 파일 → main PASS / HEAD FAIL**, 없는 경로 → main 은 뒤 네 검사를 건너뛴 채 FAIL / HEAD 는 그 자리에서 FAIL, 진짜 server.log → PASS. 비평의 우려 ②(LAW_LIVE 경로)는 이번에 바뀐 게 없어 그대로 유효하다.
- 확신 없는 곳: ① 로컬에 psql/pg_dump 가 없어(sudo 불가) `LAW_PG_ADMIN_URL` 을 비우고 harness 의 docker 분기로 돌렸다 — CI 가 타는 psql 바이너리 분기는 미검증(이 diff 밖). ② 첫 실행 24/26 의 AT-17·AT-19 실패는 모의 공급자 한 프로세스에 acceptance 를 두 번 돌려 남은 장애 주입 잔재로 판단했다(재기동 후 1회 = 26/26). ③ 작업 중 삭제된 워크트리에서 돌던 유령 서버(:8810)를 하나 죽였다 — 다음 역할은 서버·모의공급자가 **이 워크트리**에서 돌고 있음을 전제할 것.
