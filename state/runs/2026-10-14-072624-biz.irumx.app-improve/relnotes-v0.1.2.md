v0.1.2 공유 DB 가드·세션 확인 속도·짧은 금액 표기

v0.1.1 뒤 다섯 갈래가 쌓였다. 공유 Supabase 를 여러 서비스가 함께 쓰는 구성을
배포·마이그레이션 쪽에서 마저 굳혔고, 요청당 서울 왕복을 줄였으며, 금액을 짧게
적을 때 1억 경계가 틀리던 것을 고쳤다.

- 공유 DB 가드: 배포가 공유 Hyperdrive 만 쓰고 서비스 역할로 로그인하지 않는다.
  운영 마이그레이션은 로그인 역할을 건드리지 않는다 — 공유 DB 에서 서비스 역할은
  NOLOGIN 이라 운영 마이그레이션에 앱 비밀번호를 넘기지 않는다(docs 반영).
- 속도: 세션·소속 확인을 기다리지 않게 바꿔 요청당 서울 왕복이 3번 줄었다.
- format.ts krwShort: 억/만 분기를 반올림 전 원값으로 판정해 99,995,000~99,999,999원이
  "10,000만" 으로 나오던 것을 반올림된 만 값 기준으로 바꿨다. 음수는 Math.round 가
  동점을 +∞ 로 깨뜨려 비대칭이 남으므로 크기를 반올림하고 부호를 다시 붙였다.
  시험 19개 추가.
- 버전: VERSION, 워크스페이스 4개와 package-lock, scripts/ci-env.mjs 와 wrangler.toml
  의 BIZ_VERSION, docs/openapi.yaml(npm run routes).
- 검증: 단위 82, e2e 300, ui-flows 16. npm run check(ESLint + tsc 3개)와 npm run build.
