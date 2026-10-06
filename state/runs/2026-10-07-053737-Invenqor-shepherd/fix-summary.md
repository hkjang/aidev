# 수리 요약 — Invenqor PR #33

- 문제: CI `server` 잡의 `npm audit --omit=dev --audit-level=high`(ci.yml:30) 가 신규 권고 GHSA-68fv-2mgg-jv7q(source-map-js 1.0.0–1.2.1, high) 로 실패. PR 디프는 Go 2파일뿐이라 이 변경과 무관한, main 에도 있는 의존성 실패다(vite→postcss→source-map-js 가 `dependencies` 경로라 prod 감사에 걸림).
- 재현: 로컬 `web` 에서 `npm ci && npm audit --omit=dev --audit-level=high` → CI 와 같은 출력, exit 1.
- 수정: `npm audit fix` 로 `web/package-lock.json` 의 source-map-js 1.2.1 → 1.2.2(패치 버전, postcss 의 `^1.2.1` 범위 내). 락파일 3줄만 변경, 워크플로·테스트·단언은 손대지 않았다. ci.yml:112 주석이 "신규 권고는 고쳐라" 를 명시한 정책이다.
- 검증(모두 로컬 실측): audit exit 0 "found 0 vulnerabilities" · `npm test` 17파일 150건 통과 · `npm run build` 후 `git diff --exit-code -- server/internal/webui/dist` 무변화 · redocly lint 통과 · `go test ./...` 전 패키지 ok(httpapi 27.1s) · `go vet` 무출력 · `gofmt -l` 빈 출력 · `go build` 성공 후 바이너리 삭제, 트리 clean.
- 남은 것: PostgreSQL(`server-postgres`) 잡은 이번 회차 미실행 — 락파일만 바뀌어 영향 없고 그 잡은 직전 CI 에서 실패하지 않았다. dev 전용 권고는 CI 게이트 밖이라 그대로 뒀다.
