# 수리 결과: 고칠 것이 없다 — 커밋 없음

- **원인 확정.** check-run 115699045578 의 annotations 2건을 드디어 읽었다(`GH_CONFIG_DIR=/home/hkjang/.config/gh gh api .../annotations`). 첫 건이 `annotation_level: failure` 로 **"The job was not started because recent account payments have failed or your spending limit needs to be increased."** — 둘째 건은 ubuntu-latest 가 Ubuntu 26 으로 바뀐다는 `notice`. 잡이 **시작조차 안 했다**(05:34:27→05:34:30, 3초). 코드·테스트·워크플로의 결함이 아니다.
- **브랜치는 실제로 녹색.** 이 HEAD(62b3471)에서 CI 'check' 의 모든 단계를 로컬에서 그대로 돌렸다 — `npm ci` ok · `npm run check` **126 passed (7 files)** · `npm run build` ok · `node scripts/perf.mjs` **223.7 KB / 예산 230 KB** · `ci-env` ok · `migrate` 10건 ok · 서버 `:8796` 기동 ok · `node scripts/e2e.mjs` **220 passed / 0 failed**('하루 한도' 2건 PASS) · `node scripts/a11y.mjs` **심각한 접근성 위반 없음**.
- **그러므로 고치지 않았다.** 지출 한도는 저장소 코드로 풀 수 없고, 통과시킬 유일한 수단은 금지된 것(워크플로 수정·검증 느슨하게)뿐이다. 과제서가 미리 적어 둔 분기("지출 한도면 코드로 고칠 것이 없다") 그대로다.
- **사람이 할 일 하나.** GitHub 설정 → Billing & plans 에서 결제 수단·지출 한도를 풀면 재실행만으로 통과한다. 그 전에는 몇 번 재시도해도 같은 3초 실패가 반복된다.
