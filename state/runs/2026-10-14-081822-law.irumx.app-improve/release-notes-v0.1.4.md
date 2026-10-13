v0.1.4 모의 도메인 공급자를 운영에서 거부

allowMocks 없이 LAW_DOMAIN_PROVIDER=mock 을 받던 구멍을 막았다. pg·sms·identity·payments
는 전부 `allowMocks` 를 보는데 env.ts 의 domains 만 빠져 있었다.

- env.ts loadEnv: `enumOf` 호출을 `domains` const 로 빼고 `!allowMocks && domains === 'mock'`
  이면 `EnvError` 를 던진다 — `resolveKey` 와 같은 방식으로 기동을 멈춘다. 막지 않으면
  providers/dns.ts 가 DOM-04 소유권 확인(TXT)을 Cloudflare DoH 대신 `mockBase` 의 레코드
  표로 돌려, 타인 도메인의 소유권 증명이 통과할 수 있었다.
- `enumOf` 의 오타 fallback(→manual)과 providers/domains.ts·dns.ts 의 mock 분기는 그대로다
  — 개발·e2e 가 그것들을 쓴다.
- 운영 영향 없음: wrangler.toml 은 pages·allowMocks=false 이고, .env.example 의 mock 은
  LAW_DEV=true 와 함께라 통과한다.
- env-mock-providers.test.ts 7건이 `loadEnv` 를 실제로 호출해 mock×allowMocks 네 조합·
  pages/manual·기본값·오타 fallback 을 고정한다.
- 검증: lint·types(3개 워크스페이스), 단위 103, 웹·서버 빌드, VERSION·wrangler LAW_VERSION
  일치 검사. e2e·acceptance·ui-flows·RLS 프로브는 이 기계에서 돌리지 못했다(다른 개발
  서버가 :8810 을 쓰는 중이고 e2e 의 canonical·sitemap·robots 검사가 그 포트를 박아 둔다)
  — check.yml 이 main 푸시마다 그 전부를 돌린다.
