# 수리 요약 — PR #162

- 문제: CI `verify` 의 "Frontend dependency audit"(`npm audit --audit-level=high`, web)이 undici 8.10.0 의 high 권고 11건으로 exit 1. 이 PR 의 Go 1줄 변경과 무관한 전이 개발 의존성(jsdom → undici) 문제지만 검증을 막고 있었다.
- 재현: worktree 에서 `cd web && npm audit --audit-level=high` → CI 와 같은 목록, 같은 `EXIT=1`.
- 고침: `npm audit fix --package-lock-only` 로 web/package-lock.json 의 undici 만 8.10.0 → 8.11.2 (`^8.9.0` 범위 안, dev). 다른 패키지·package.json·워크플로·시험은 건드리지 않았다(diff 3줄).
- 검증: `npm ci` 재현성 확인 후 audit=0 취약점, test:offline-queue·typecheck·lint(기존 경고만)·check-i18n·vitest 186 통과·build 통과. PR 본 변경은 `POSTGRES_DSN`(umm-test-pg:15433, user/pw `umm`)로 3개 시험 `-v` PASS(SKIP 아님).
- 미확인: Playwright 와 `go test -p 1 ./...` 전체는 안 돌렸다(변경이 개발 전용 lockfile 한 줄).
