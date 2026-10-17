# 회차 노트 2026-10-17-220917-postra-improve — postra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 22:09] base pinned — main@23479bc
- [러너 22:09] autonomy release — 
- [러너 22:14] scout done — 릴리즈 워크플로(`.github/workflows/release.yml`)가 v0.25.9 태그에서 두 번 연속 실패한 **실제 단계**를 로그로 고�

## 구현 노트
- 1순위는 기각했다. `gh` 미인증이라 공개 REST API 로 봤더니 `release.yml` 실행 **88건 전부 success**(v0.25.9 = run 39008042493)이고 v0.25.9 릴리즈는 자산 5개가 이미 공개돼 있다. 실제로 **세 번** 연속 죽은 것은 `ci.yml` 의 `govulncheck` 잡 → `Scan` 단계(run 37743324018 / 37807055974 / 39001934751)이고, `3a10d84` 로 이미 해결돼 로컬 재현도 `govulncheck ./...` exit 0 이다. 고칠 결함이 없고 완화는 금지라 **워크플로 파일은 한 줄도 건드리지 않았다.**
- 그래서 차선을 구현했다: `syncHint` 에 `stage=greeting`+`protocol=imap`+`class=other` 분기(`greetingNotIMAPHint`) 1개. 프로덕션 1파일, 테스트 3파일, 커밋 ae1b4d0.
- **확신 없는 곳 / 검증 못 한 것**: ① 힌트를 기존 greeting 힌트 **뒤**에 뒀으므로 `connecting && AccountSessions > 1`(동시 접속) 분기가 shadowing 한다 — IDLE 연결이 떠 있는 동안 포트 오설정이면 덜 정확한 동시접속 힌트가 나간다. 과제서 지시대로 둔 것이고 하위 테스트로 그 우선순위를 고정해 뒀지만, 더 정확하게 하려면 순서를 바꿔야 한다(이번엔 범위 밖). ② `class=other` 에 도달하는 **제3의** 경로가 있다면(분류 불가 read 오류) 힌트가 틀릴 수 있다 — 어댑터 테스트로 확인한 것은 두 경로(isGreeting 거부, errUnframed)뿐이다. ③ 프런트·전체 저장소 테스트·PostgreSQL·브라우저 e2e 미실행(Go 진단 전용 변경).
- 일부러 하지 않은 것: 워크플로/README/docs 수정 0건(실패하지 않는 단계를 고치는 셈), POP3 쪽 동일 힌트(어휘가 달라 문구를 재사용할 수 없다), `domain.SyncDiagnostic` 필드 추가(api/web/spa/assets 로 번진다 — `contracts -check` exit 0 확인).
- 다음 역할이 조심할 것: 어댑터 테스트의 과대 길이 greeting 픽스처는 `maxLineBytes+(64<<10)` 이어야 한다. `maxLineBytes+1` 로 줄이면 상한이 트립하지 못해 마지막 1바이트에서 멈추고 **15초 걸려 `class=timeout` 으로 FAIL** 한다(실측). 이 크기는 리더 버퍼 한 개 이상 초과가 필요하다는 뜻이고, 느려 보여도 줄이지 말 것.
- [러너 22:25] brief rejected — 기각(1순위) → 차선 채택 — 1순위의 전제("릴리즈 워크플로가 v0.25.9 에서 두 번 실패했고 산출물이 배포되지 않았다")가 
- [러너 22:26] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인함: 전체 diff 정독, `syncHint` 분기 순서, `ClassifyInbound`/imap `Dial` greeting 경로, 읽기 경계(`safeSyncDiagnostic`). 실행 — build·vet·`contracts -check`(0)·`git diff --check`, `go test -race -count=1` 로 application 85.3s / imap 4.7s / pop3 5.4s / httpapi 23.5s 전부 PASS. 판정 approve, risk low, 차단 없음.
- 테스트 효력: application 쪽 두 단언은 새 상수 `greetingNotIMAPHint` 에 걸려 있어 수정 전 코드에서는 반드시 실패하고, 실제 루프백 서버+프로덕션 Dialer 로 e2e 경로를 지난다. 어댑터 테스트는 main 에서도 통과하는 특성 테스트이나 패키지 경계 결합 고정이 목적이라 수용. 회차 노트가 경고한 15초 지연은 재현되지 않았다(0.03s).
- 구현자의 자기 의심 둘 중: ① shadowing 은 영향 좁음(포트 오설정이면 IDLE 도 못 붙어 AccountSessions 가 1 을 안 넘음) ② 제3의 other 경로는 실재(핸드셰이크 이후 TLS 읽기 오류) — 문구가 한정돼 있어 오도 위험 낮음. 둘 다 퇴행이 아니라 거절하지 않았다.
- 남는 우려(릴리즈가 볼 것): `docs/SYNC_DIAGNOSTICS.md:100` 의 "자주 보는 조합" 표에 새 `greeting`+`imap`+`other` 행이 없다. main 의 623dee1 도 이 문서를 갱신하지 않아 이번 PR 만의 퇴행은 아니니, 릴리즈 노트와 함께 한 줄 추가를 권한다.
- 못 본 것: 프런트엔드(미변경이라 생략), PostgreSQL·브라우저 e2e, `make lint-security`/`lint-format`, 저장소 전체 `go test`. Go 진단 문자열 한 줄 추가라 범위를 이렇게 잡았다.
- [러너 22:31] review approved — 리뷰 승인 (risk=low)
- [러너 22:32] pr created — https://github.com/hkjang/postra/pull/41
- [러너 22:37] ci passed — 검사 10개 모두 success
- [러너 22:37] merge done — ae1b4d0
- [러너 22:51] release published — v0.25.10
- [러너 22:53] assets verified — v0.25.10 자산 5개 (이전 v0.25.9: 5)
