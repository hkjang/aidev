# 수리 요약 — 고치지 않았다 (지적이 이 diff 의 결함을 가리키지 않는다)

- 재현 결과: diff 는 `packages/core/src/__tests__/domain.test.ts` +58행뿐(프로덕션·워크플로·설정·lockfile 무접촉)이고, 로컬에서 CI 와 같은 게이트가 전부 green 이다 — `npm ci`(exit 0), `npm run check`(eslint + 3×tsc + **vitest 164 passed**), `npm run build`(exit 0), `node scripts/ci-env.mjs`(exit 0).
- CI 는 `3s`·failure 로 끝났다. 이 머신에서 캐시된 `npm ci` 만 13초다 — checkout→setup-node→npm ci 중 어느 것도 3초 안에 끝나지 않으므로 **diff 를 읽는 단계가 하나도 실행되지 않았다**. 로그·annotation 은 비공개 저장소 + 이 세션 토큰 없음(`gh`: 미로그인, 무인증 API: repo/run/joblog 전부 404)으로 인용 불가.
- diff 자체의 유효성은 변이로 재확인했다: `validate.ts` 의 ISO 왕복 가드를 제거 → 같은 시험 **5 failed**(`2026-02-30` 통과), `git checkout --` 복원 후 **85 passed**. 단언은 프로덕션 `v.day` 에 실제로 물린다. 작업 트리 깨끗(`git status` 빈 값).
- 그래서 커밋 없이 끝낸다. 남은 유일한 길은 `.github/workflows/check.yml` 을 느슨하게 고치는 것뿐인데 그것은 절대 규칙 위반이고 원인도 미확정이다(3회차 연속 추정만 가능).
- 비평가에게: 이 PR 은 저장소 안에서 green 이다. 다음 조치는 코드 수정이 아니라 **인증된 세션에서 run 38506072123 의 annotation 2건을 읽는 것**이다.
