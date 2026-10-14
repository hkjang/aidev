# 수리 요약 — 고칠 결함이 없다 (커밋 없음)

- CI 'check' 는 코드 때문에 실패한 것이 아니다. 잡이 **시작조차 못 했다**(3초, started 01:40:07 → completed 01:40:10). 체크런 주석 2건을 받아 확인한 원문: "The job was not started because recent account payments have failed or your spending limit needs to be increased. Please check the 'Billing & plans' section in your settings" (`gh api repos/hkjang/law.irumx.app/check-runs/115634810274/annotations`). 나머지 1건은 ubuntu-latest 이전 공지(notice)다. 곧 GitHub Actions 결제/지출한도 차단이며, 사람이 Billing & plans 에서 풀어야 한다.
- 로컬에서 CI 와 같은 검증을 처음부터 돌려 전부 통과했다: `npm ci`(239 packages, lock 무변동) → `npm run check` = lint 0 errors/646 warnings(기준선 동일) + typecheck 3개 워크스페이스 + vitest **7 files / 106 tests 통과** → `npm run build` 성공. CI 가 `npm run check`·`npm run build` 까지 갔다면 그 단계는 통과했을 것이다.
- 실패 재현(테스트 유효성): `pipeline.ts` 만 origin/main 으로 되돌리면 `background-redact.test.ts` 3개 중 2개가 실패(300자 경계에서 `@` 가 잘려 `s3cret` 이 로그에 남는 출력을 직접 확인), HEAD 로 되돌리면 전부 통과. 테스트가 실제로 대상을 실행·단언한다.
- DB·브라우저가 필요한 단계(e2e·acceptance·rls-probe)는 이 샌드박스에서 돌리지 못했다 — **미검증**. 그 단계들은 이번 변경(stderr 로그 3줄)과 무관하지만 통과를 주장하지 않는다.
- 따라서 커밋하지 않았다. 워크플로·검증 명령을 손대 통과시키는 것은 금지이고, 그 외에 이쪽에서 할 일이 없다. 결제 차단이 풀린 뒤 CI 재실행이 필요하다.
