# 수리 요약 — PR #13 (커밋 없음)

**지적이 틀렸다고 판단했다. 고칠 코드 결함이 없어 커밋하지 않았다.**

- **재현 실패(= 결함 없음).** HEAD `1dd2151` 에서 CI `check` job 을 그대로 돌려 전부 exit 0: `npm ci` → `npm run check`(typecheck · **87/87 tests** · build) → `node --check` 두 번들. CI 러너와 같은 **Node 20.20.2**(`node-version: 20` 이 해석되는 값) + **TZ=UTC** 로 확인했고, `engines.node` 핀 20.19.2 및 TZ=KST 에서도 같다. 테스트 5회 반복 무결(플레이크 아님). 증거: `validation/pr13-ci-check-reproduction.log`.
- **diff 가 CI 를 깨뜨릴 수 없다.** 이 변경은 `src/features/goal-metrics.ts` · `tests/unit/goal-metrics.test.ts` **2개 파일뿐**이다. `ci.yml` · `package.json` · `package-lock.json` 미접촉이므로 install·workflow·startup 실패의 경로가 없고, 남는 경로(typecheck/test/build)는 위에서 전부 녹색이다.
- **실패는 인프라 쪽 징후다.** 보고된 실패는 **4초**에 끝났고 첨부된 실패 로그는 **비어 있다** — `npm ci` 한 단계도 4초에 못 끝나므로 job 이 스텝을 실행하지 못한 것이다. 프로필에 기록된 이 저장소의 선례(run 36518589343: `steps=[]`, `runner_id=0`, "recent account payments have failed or your spending limit needs to be increased")와 같은 모양이다.
- **테스트는 대상을 실제로 검증한다.** `src/features/goal-metrics.ts` 만 `git checkout origin/main --` 로 되돌리면 새 테스트 3개가 정확히 빨개지고(`completedItems` 5개, `activeDays 2`, 회고에 `두 달 전 항목`), 복원하면 9/9 녹색 — 인과를 직접 확인했다.
- **확인 못 한 것(정직하게):** `gh` 미인증이고 push·네트워크가 없어 run 36781871266 의 annotation 을 **직접 읽지 못했다** — 결제/한도 원인은 정황 증거(4초 + 빈 로그 + 선례)로만 판정했다. windows-latest `package` job 도 여전히 미재현이다. 러너가 재실행(re-run)하면 그대로 통과할 것으로 본다.
