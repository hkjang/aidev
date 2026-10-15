# 수리 요약 — 커밋 없음 (지적이 이 변경의 결함이 아님)

- 지적 원인 확정: `check` 잡(116265181090)은 **0단계·3초**로 끝났고 annotation 원문이 `The job was not started because recent account payments have failed or your spending limit needs to be increased` 다 — GitHub Actions 결제·지출한도 차단이라 `actions/checkout` 조차 돌지 않았다. 저장소 밖 문제이고 diff 를 읽은 단계가 0개다. (`GH_CONFIG_DIR=/home/hkjang/.config/gh` 로 읽음 — 앞선 회차들이 못 읽었던 그 경로.)
- 로컬 재현 결과 같은 자리 실패가 없다: `npm ci` → `npm run check` 통과(eslint + tsc 3개 + 9파일 173 시험), `npm run build` 통과, `node scripts/ci-env.mjs` 통과. CI 가 diff 에 닿는 단계는 전부 녹색이다.
- 인과 확인: `quota.ts:54,56` 을 수정 전(`windowStart`)으로 되돌리면 35개 중 23개 실패, 복원하면 35개 통과 — 새 시험은 실제로 대상을 돌리고 단언한다. TZ=UTC·America/New_York·Pacific/Kiritimati 에서도 35개 전부 통과(`kstDateKey` 가 TZ 무관이라 CI 환경차 아님).
- 고치지 않은 이유: 통과시키려면 워크플로나 계정 과금을 손대야 하고 둘 다 금지·범위 밖이다. 코드는 이미 맞다 — 결제 복구 후 재실행(`gh run rerun 38687364903`)이 유일한 조치다.
