## 2026-10-08
- (원장 항목에 비밀/내부 정보 의심 문자열이 있어 비공개 기록으로 옮김 — run 2026-10-08-013826-law.irumx.app-improve)

- 릴리즈: 0.1.2 (2026-10-08, run 2026-10-08-030708-law.irumx.app-approve)
## 2026-10-14
- 선택: 모의 도메인 공급자를 운영에서 거부 — loadEnv 가 allowMocks 없이 LAW_DOMAIN_PROVIDER=mock 을 받던 구멍 (가치 3 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `env.ts:loadEnv` 에서 `enumOf` 호출을 `domains` const 로 빼고, `!allowMocks && domains === 'mock'` 이면 `EnvError` 를 던지게 했다 — pg·sms·identity·payments 가 이미 `allowMocks` 를 보는 것과 같은 방식이다. 막지 않으면 `providers/dns.ts:10` 이 DOM-04 소유권 확인(TXT)을 Cloudflare DoH 대신 `mockBase` 의 레코드 표로 돌려 타인 도메인 소유권 증명이 통과할 수 있었다. 신규 테스트 7건이 `loadEnv` 를 실제로 호출해 mock×allowMocks 네 조합·pages/manual·기본값·오타 fallback 을 모두 고정했고, `npm run check`(lint 0 errors / typecheck 3개 워크스페이스 / vitest 6 files 103 tests) 가 통과했다. 프로덕션 파일 1개 + 테스트 1개.
- 실패 재현: `AssertionError: expected function to throw an error, but it didn't` — `env-mock-providers.test.ts:27` ("refuses LAW_DOMAIN_PROVIDER=mock when mock providers are not allowed") 와 `:33` 2건 실패, 나머지 5건은 수정 전에도 통과(주변 계약이 안 바뀜을 확인). 수정 후 7/7 통과.
- 보류 아이디어: ① redactSecrets 계약을 표 테스트로 고정(4/1/S, 프로덕션 0파일 — 다음 회차 1순위) ② util/validate.ts 경계 테스트(3/1/M) ③ loadEnv 전반 단위 테스트 집합 — 이번 파일이 씨앗(3/1/M) ④ isSafeLink 를 실제 링크 렌더 경로에 배선할지 조사(3/3/M) ⑤ 모듈 계층(forms·workflow·engagements) 순수 함수 테스트 공백 — forms.ts 단계·조건 평가부터(3/2/L)
- 과제서: 채택 — 과제서의 근거가 현재 코드와 정확히 일치했고(env.ts:163, dns.ts:10, wrangler.toml:46/57), 지정한 수용 기준·파일 범위·금지 사항을 그대로 구현했다.

- 릴리즈: 0.1.4 (2026-10-14, run 2026-10-14-081822-law.irumx.app-improve)
