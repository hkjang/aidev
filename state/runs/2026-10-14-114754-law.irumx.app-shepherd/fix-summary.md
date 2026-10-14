# 수리 요약 — 고칠 결함이 없다 (커밋 없음, 2회차 확인)

- CI 'check' 실패는 코드 탓이 아니다. 잡이 **3초**에 끝났는데(과제서의 `check fail 3s`), 이 워크플로는 `apt-get install postgresql-client-17` + `npm ci`(여기선 15초)만으로도 3초를 넘는다 — 즉 잡이 **시작조차 못 했다**. 직전 수리 회차(2026-10-14-110037)가 같은 잡 115634810274 의 체크런 주석을 `gh api` 로 받아 원문을 확인했다: "The job was not started because recent account payments have failed or your spending limit needs to be increased." GitHub Actions 결제/지출한도 차단이며 사람이 Billing & plans 에서 풀어야 한다. (이번 회차엔 `gh` 자격증명이 없어 주석을 **직접 재조회하지는 못했다** — 근거는 직전 회차 인용 + 3초 소요시간이다.)
- 로컬에서 CI 와 같은 검증을 처음부터 돌려 전부 통과: `npm ci`(239 packages, package-lock.json 무변동) → `npm run check` = lint **0 errors / 646 warnings**(기준선 동일) + typecheck 3개 워크스페이스 + vitest **7 files / 106 tests 통과** → `npm run build` 성공.
- 실패 재현(테스트 유효성 직접 확인): `pipeline.ts` 만 `origin/main` 으로 되돌리면 `background-redact.test.ts` 3개 중 **2개 실패** — 로그에 `postgres://law_app:s3cret` 이 그대로 찍히는 출력을 눈으로 확인했다. HEAD 로 되돌리니 106/106 통과. 테스트가 실제로 대상을 실행·단언한다.
- DB·브라우저가 필요한 CI 후반 단계(e2e·acceptance·rls-probe)는 이 샌드박스에서 못 돌렸다 — **미검증**. 이번 변경은 stderr 로그 3줄이라 무관하다고 보지만 통과를 주장하지 않는다.
- 따라서 커밋하지 않았다. 워크플로·검증 명령을 고쳐 통과시키는 것은 금지이고 그 외에 이쪽에서 할 일이 없다. **결제 차단 해제 후 같은 커밋 0804a58 로 CI 재실행**만 하면 된다.
