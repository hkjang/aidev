# 수리 요약 — 커밋 없음 (지적이 이 변경의 결함이 아님)

- 원인 확정: `check` 잡 116265181090 은 **steps=0 · 3초**(API 로 확인)이고 annotation 원문이 `The job was not started because recent account payments have failed or your spending limit needs to be increased` — GitHub Actions 결제·지출한도 차단으로 `actions/checkout` 조차 돌지 않았다. 이 PR 코드는 CI 에서 평가된 적이 없다. (`GH_CONFIG_DIR=/home/hkjang/.config/gh gh run view 38687364903`)
- 이 diff 무관함의 결정적 증거: 같은 차단이 **main 푸시에서도** 같다 — 이 PR 커밋이 하나도 없는 `main@714a660` 의 run 38491249147 이 3초·동일 annotation. 2026-10-14 00:30 이후 모든 run 이 3~5초로 끝나고, 그 직전 성공 run 들은 9~11분이었다.
- 로컬 재현 없음(같은 자리 실패가 안 난다): `npm ci` → `npm run check` 통과(eslint + tsc 3개 + 9파일 **173 시험**), `npm run build` 통과, `node scripts/ci-env.mjs` 통과. CI 가 diff 에 닿는 단계는 전부 녹색이다.
- 인과 독립 확인: `quota.ts:54,56` 만 수정 전(`windowStart`)으로 되돌리면 **23/35 실패**, 복원하면 **35 통과** — 새 시험은 실제로 대상을 실행하고 단언한다. 이후 트리는 clean(커밋·수정 없음).
- 고치지 않은 이유: 통과시킬 방법이 워크플로 수정이나 계정 과금 복구뿐이고 둘 다 금지·범위 밖이다. 결제 복구 후 `gh run rerun 38687364903` 이 유일한 조치다.
