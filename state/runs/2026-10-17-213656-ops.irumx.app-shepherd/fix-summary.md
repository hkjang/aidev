# 수리 요약 — ops.irumx.app PR #3

- 문제(지적이 맞았다): `resourceLabel` 이 `'do:*'` 만 바꿔 `run.ts:174` 의 cost_jump 자원 키 `'account'` 를 놓쳤다. 접두사가 없어 `resources.key`(worker:·d1:·r2:·kv:·queue:·do:·aigw:·vectorize:·ai: — `collect/resources.ts:111-132`, `analytics.ts:100·197` 로 전부 확인)와 안 맞고, 고친 폴백(run.ts:326·327)·화면(anomalies.tsx:299)을 그대로 지나가 `[이룸옵스] 예상 비용 급등 — account` / `자원: account` 로 나갔다.
- 고친 것: `src/shared/policy.ts:362` `resourceLabel` 에 `'account'` → `'계정 전체'`(저장소 기존 표현 — services.tsx:721·budgets.tsx:289) 분기 한 줄 + 주석에 경로·날짜, `tests/unit-policy.spec.ts:258` 에 단언 한 줄(시험 제목도 `do:*·account` 로). 프로덕션 1 + 시험 1 파일, 커밋 d771143.
- red 확인: 고치기 전 `npm run test:unit` → `1 failed … Expected: "계정 전체" / Received: "account"`, 49 passed. 고친 뒤 **50 passed (782ms)**.
- 재검증: `npm run check` exit 0, `npm run test:unit` 50 passed, `npm run build && npm test` **91 passed (14.4s)** exit 0. 빌드가 다시 쓴 추적 파일 `public/fonts/*.woff2`·`src/client/styles/fonts.css` 3개는 커밋 전에 되돌렸다(diff 2파일만).
- 남은 한계: cost_jump 을 실제로 통과시키는 실행 시험이 없어(`tests/` 에 cost_jump 사례 0건) 표시 지점 3곳의 실제 출력은 순수 함수 시험 + 코드 읽기로만 확인했다. `collect/analytics.ts:135` 의 `'r2:*'` 는 비평가가 "차단 사유 아님" 으로 적고 표시 경로에 닿지 않음을 확인했으므로 범위 밖으로 두었다.
