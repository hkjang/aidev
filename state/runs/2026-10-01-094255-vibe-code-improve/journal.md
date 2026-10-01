# 회차 노트 2026-10-01-094255-vibe-code-improve — vibe-code
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 09:43] base pinned — main@43fd7a1
- [러너 09:43] autonomy release — 
- [러너 09:48] scout done — 수정 과제 — PR #15 head `a1e3c94` 로 CI check job 로컬 재현 판정 + `enforceBudget` 의

## 구현 노트
- 0단계: PR #15 head `a1e3c94` 를 `/tmp/pr15` 로 추출해 CI check job 네 단계를 Node 20.19.2 에서 재현 — 전부 exit 0(`validation/pr15-node20-check.log`). 원격 CI 실패는 코드 결함이 아니다(열 번째 같은 판정). `ci.yml` 은 읽기만 했다.
- 1단계: `enforceBudget` 의 "오늘" 이 UTC 날짜(`toISOString().slice(0,10)`)라 09:00 KST 경계에서 한 KST 날이 두 키로 쪼개져 경고·자율성 강등·감사가 하루 두 번 났다. 순수 export `budgetNoticeDay` = `kstDate(now)` 로 감사 파일명과 같은 경계를 쓰게 했다. 프로덕션 1파일 · 테스트 1파일 · 문서 0줄.
- **확신 없는 곳·검증 못 한 것**: (a) `enforceBudget` 전체 흐름은 재현하지 않았다 — 대역 금지 때문에 순수 경계의 날짜 키 계약만 증명했다. "하루 두 번 발동" 자체는 추론이다(코드 읽기 + 키 비교). (b) 원격 CI 상태·annotation 은 읽지 못했다(`gh` 미인증, push·WebFetch 불가). PR 번호↔브랜치 대응은 추정. (c) CI `package` job(windows-latest)은 이번에도 미확인. (d) `npm run build` 의 "dist/ is missing runtime assets" 경고는 PR #15 기준선 실행에도 똑같이 나오는 기존 상태다.
- 일부러 하지 않은 것: 문서는 고치지 않았다 — `docs/autonomous-goal-workflow.md:147`·`package.nls*.json:109` 의 "하루 한 번"/"once per day" 는 **고친 뒤에야 사실이 되는** 약속이지 대체된 옛 가이드가 아니다. 차선 후보(`goal-health.ts` 의 `since`)는 grep 결과 읽는 경로가 없어 기각했다.
- 다음 역할이 조심할 것: 이 워크트리는 `npm ci` 가 선행돼야 테스트가 돈다. 인과 확인 로그(`validation/causation-red.log`)는 `budgetNoticeDay` export 가 HEAD 에 없어 `TypeError` 로 오염돼 있다 — **깨끗한 결함 증거는 `validation/usage-kst-red.log`**(추출만 한 상태의 `AssertionError` 3건)다.
- [러너 09:55] brief accepted — 채택 — 0단계 게이트를 지시대로 Node 20.19.2 에서 네 단계 모두 재현해 "코드 결함 아님" 을 증거 로그로 판정하고, 멈추지
- [러너 09:55] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인: KST/UTC 경계를 Node 20.19.2 로 직접 계산(08:00 KST → UTC 는 전날, kstDate 는 같은 날)해 결함과 수정 방향이 둘 다 사실임을 봤고, `writeAudit`(workspace.ts:64)이 정말 `kstDate()` 파일명임을 열어 확인했다. `budgetNotifiedDate` 읽기/쓰기 경로는 usage.ts:134-135 둘뿐이고 `src/` 에 남은 UTC 날짜 키는 0건. 로컬 4단계 재실측: typecheck 0 · 11 files 85 tests 통과 · build 0 · `node --check` 두 번들 0.
- 테스트 유효성: `usage-kst-red.log` 가 "동작 변경 없이 추출만 한 상태" 의 `AssertionError` 3건이어서 심볼 누락이 아닌 진짜 결함 증거다. 단 `usage.test.ts:46` 은 머지되는 한 줄 구현 기준 동어반복이고, 경계를 실제로 못박는 것은 :31-39 두 테스트다.
- 못 본 것: 원격 CI 상태·annotation(gh 미인증), CI `package` job(windows) — 열한 회차째 미확인. 코드 결함이 아니라 운영자 조치 항목.
- 승인이어도 남는 우려(릴리즈 노트에 적을 것): `vibeCode.budgetNotifiedDate` 는 마이그레이션 없이 의미만 UTC→KST 로 바뀌므로, 00:00~08:59 KST 에 이미 알림을 받은 사용자는 업그레이드 직후 **1회** 경고와 자율성 강등이 더 발동한다. 하루 뒤 수렴하고 revert 한 번으로 되돌아온다.
- 보안·법무 차단 없음: 인가·식별자·비밀값·암호 비교·권한 확대 변경이 없고 감사 필드와 개인정보 수집은 그대로다. 기존 사항 두 개는 notes 로만 남겼다(감사 디렉터리 보존 집행 경로 없음, 기본 `vibeCodersBaseUrl` 이 http — 기본값이 localhost 라 공격 경로 아님).
- [러너 09:58] review approved — 리뷰 승인 (risk=low)
- [러너 09:58] pr created — https://github.com/hkjang/vibe-code/pull/16
- [러너 10:00] ci passed — 검사 2개 모두 success
- [러너 10:00] merge done — ad0254c
- [러너 10:10] release published — v1.4.6
- [러너 10:10] gh-release created — GitHub Release v1.4.6
- [러너 10:25] assets missing — 이전 v1.4.0 엔 2개, v1.4.6 엔 0개 — 워크플로: null: null/null
