# 수리 요약 (수리 시도 2)

- 문제: `db-breaker.test.ts` 에서 `expect(dropped).toBe(tried)` 가 `again.healthy()` 뒤에서 앞으로 옮겨져, "health 도 차단기를 거쳐 연결 없이 실패한다"(db.ts:340-345) 를 아무 단정도 검사하지 않았다. 비평 지적은 맞았다.
- 고친 것: `:153 expect(await again.healthy()).toBe(false)` **뒤에** `expect(dropped).toBe(tried)` 를 한 줄 추가(앞의 것은 그대로 둠, 옮기지 않음) + 왜 두 번인지 주석 3줄. 테스트 파일 1개 +4줄, 프로덕션 0개.
- 변이 검증: `db.ts:345` 를 `await this.retrying(() => this.sql\`SELECT 1\`)` → `await this.sql\`SELECT 1\`` 로 바꾸면 이제 **1 failed / 10 passed** (`expected 15 to be 12`, 옛 파일과 같은 자리) — 고치기 전 이 변이는 11개 전부 통과했다. 검증 후 `db.ts` 는 원상 복구(`git diff` 로 확인).
- 재검증: `npm run check` 통과 — eslint + core/server/web typecheck + vitest **109 passed (6 files)**.
- 남은 우려(이번 범위 밖, 비평가가 제기): 벽시계 3곳이 `BUDGET_CEILING_MS`(10s) 하나로 느슨해진 것은 그대로 뒀다 — 지적 사유가 아니었고 단정을 조이는 변경은 이 수리 범위를 넘는다.
