# 수리 요약 (커밋 177ec51)

- 문제: README.md:105 가 check.yml 이 「검증」 6개 "같은 목록" 을 밟는다고 썼지만 실제로는 4개(`npm run check`·`perf.mjs`·`e2e.mjs`·`a11y.mjs`)뿐이고, :123 의 "「검증」 전부 로컬 통과" 도 ui-flows·screenshots 를 돌리지 않은 상태라 근거보다 넓었다. 둘 다 비평이 맞았다 — check.yml 을 읽어 단계 목록을 확인했다.
- 고침: :105 를 "이 중 네 개(…)를 밟습니다 / ui-flows·screenshots 는 CI 에 없어 브라우저 흐름·스크린샷은 로컬에서 직접 돌려야 증명되고 초록 CI 를 근거로 쓸 수 없다" 로 바꿨다. :123 은 "CI 가 밟는 네 단계가 로컬 통과(단위 130 · JS 224.1 KB · e2e 218 · 심각 위반 0)" 로 좁혔다.
- 재검증: 그 네 단계를 이 HEAD 에서 직접 돌렸다 — `130 passed (7 files)`, `224.1 KB of 230 KB`, `applied 10 migrations` 뒤 e2e `218 passed, 0 failed`, a11y `심각한 접근성 위반 없음`. 적어 둔 숫자는 전부 실행 출력이다.
- 범위: README.md 1파일 5+/2-. 워크플로·테스트·프로덕션 코드 0줄. CI 빨강은 여전히 billing 문제로 이 회차에서 고쳐지지 않는다.
