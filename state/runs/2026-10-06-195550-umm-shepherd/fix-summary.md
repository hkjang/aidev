# 수리 요약 — PR #167

- 문제: CI `verify` 의 `Frontend dependency audit`(`npm audit --audit-level=high`, `.github/workflows/ci.yml:118`)가 exit 1. 원인은 이 PR 이 아니라 새로 공개된 권고다 — `source-map-js@1.2.1`(high, GHSA-68fv-2mgg-jv7q), `dompurify@3.4.14`(low ×2). 둘 다 전이 의존(`vite`/`jsdom` → source-map-js, `jspdf` → dompurify)이고 `web/package-lock.json` 은 `origin/main...HEAD` 에서 **한 줄도 바뀌지 않았다**(`git diff --stat -- web/package-lock.json` = 0) — main 에서도 똑같이 빨갛다.
- 재현: `npm ci --prefix web` 후 `npm audit --audit-level=high` → CI 로그와 글자 그대로 같은 출력, EXIT=1.
- 고친 방법: `npm audit fix --package-lock-only` — 락파일만 6줄(`source-map-js 1.2.1→1.2.2`, `dompurify 3.4.14→3.4.16`). `package.json`·소스 0파일. 검증 명령이나 워크플로는 건드리지 않았다.
- 재검증(모두 EXIT=0): `npm ci` → `npm audit --audit-level=high`(found 0 vulnerabilities) · `test:offline-queue` · `typecheck` · `lint`(새 경고 0) · `check-i18n`(1060키) · `npm test`(22파일/230시험) · `build` · `verify:pwa`(150 assets). Go 는 변경 0파일이라 돌리지 않았다.
- 못 돌린 것: Playwright(실브라우저·격리 DB 필요)와 `docker build`. 남는 위험은 `jspdf` 가 쓰는 dompurify 패치 2개가 PDF 내려받기에 미치는 영향인데 build 와 단위시험은 통과했고 e2e `export-pdf.spec.ts` 는 CI 가 확인한다.
