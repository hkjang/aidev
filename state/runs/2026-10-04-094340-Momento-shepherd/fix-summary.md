# 수리 요약 — Momento PR #23

- 실패 재현: `cd web && npm ci && npm audit` → exit 1, `9 high severity vulnerabilities`(braces GHSA-vfj7-8cjw-p6xm). CI 로그와 같은 자리다.
- 원인: PR 의 코드와 무관하다. 새로 공개된 braces 권고가 `braces *`(패치 버전 없음)여서 override 로는 못 막고, 트리에 braces 를 끌어오는 유일한 경로가 `typescript-eslint 8.42.0 → …/typescript-estree → fast-glob → micromatch → braces` 였다. 의존성 변경이 PR diff 에 없으므로 main 도 같은 실패다.
- 고친 방법: `web/package.json` 의 typescript-eslint 를 **fast-glob 을 tinyglobby 로 교체한 최초 버전인 8.48.0** 으로만 올렸다(8.47.1 은 미발행, latest 8.71.0 대신 최소 범프). 다른 패키지·CI·테스트·단언은 손대지 않았다.
- 검증: `rm -rf node_modules && npm ci && npm audit && npm run lint && npm test && npm run build` 전부 통과 — audit `found 0 vulnerabilities`, lint 무출력, 테스트 **191/191**, build 성공. sdk 는 lock 에 fast-glob 이 없어 무관(CI 에서도 통과했다).
