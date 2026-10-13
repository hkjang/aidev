# v0.7.4 — 리뷰 쓰기의 파트너 이름, 띄어쓰기를 그대로 받는다

리뷰 쓰기 화면의 "파트너 이름" 입력란이 공백을 삼키던 것을 고쳤다. 고친 파일은 프로덕션 1개
(`apps/web/src/routes/Write.tsx`).

- 입력란이 `value={form.partner_name.trim()}` 로 제어돼, 한 글자씩 치면 공백을 넣는 순간
  React 가 DOM 값을 되돌려 "메가존 클라우드" 가 "메가존클라우드" 로 붙었다. 값을 다듬지 않고
  그대로 보여 준다.
- 같은 칸을 다 지우면 표시 조건(`form.partner_name ?`)이 거짓이 되어 입력란 자체가 사라졌다.
  화면 전용 `partner_by_name` 플래그로 표시 여부를 "이름으로 적기" 선택에 묶는다. 서버 `parse`
  는 shape 키만 읽어 이 화면 전용 키를 무시한다.
- `scripts/ui-flows.mjs` 에 검사 2건 추가(띄어쓰기가 있는 이름을 그대로 입력 / 다 지워도 입력란이
  남아 다시 쓸 수 있음). 고치기 전 2건 실패.

## 검증 (0.7.4 버전 올림 뒤 이 기계에서 다시 수행)

| 검사 | 결과 |
| --- | --- |
| `npm run check` (lint·타입·단위) | 118 passed |
| `npm run build` | 성공 |
| `node scripts/perf.mjs` | 첫 방문 JS 223.7 KB (예산 230 KB) |
| `node scripts/ci-env.mjs` | 성공 |
| `node scripts/e2e.mjs` (dev-reset 직후) | 218 passed / 0 failed |
| `node scripts/ui-flows.mjs` (dev-reset + seed-demo 직후) | 19 passed / 0 problem |
| `node scripts/a11y.mjs` (라이트·다크 60면) | 심각한 접근성 위반 없음 |
| VERSION ↔ wrangler.toml `REVIEW_VERSION` | 일치 (`scripts/deploy-pages.sh` 의 검사) |

커밋 `807ab20` 에 대한 GitHub `check` 워크플로도 success 였다.

## 올린 곳

`VERSION`, `package.json`, `package-lock.json`, `apps/web/package.json`,
`apps/server/package.json`, `packages/core/package.json`, `wrangler.toml` (`REVIEW_VERSION`).

배포는 이 저장소의 관례대로 수동이다 — `bash scripts/deploy-pages.sh`,
`npx wrangler deploy --config workers/scheduler/wrangler.toml`.
