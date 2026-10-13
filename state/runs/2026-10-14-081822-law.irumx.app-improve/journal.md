# 회차 노트 2026-10-14-081822-law.irumx.app-improve — law.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 08:18] base pinned — main@316cf8d
- [러너 08:18] autonomy release — 

## 정찰 노트
- 모의 도메인 공급자 가드를 골랐다: pg·sms·identity·payments 는 전부 `allowMocks` 를 보는데 `env.ts:163` 의 domains 만 빠져 있고, 그 결과 `dns.ts:10` 이 DOM-04 소유권 확인을 mock 레코드 표로 돌린다 — 프로덕션 1파일로 끝나는 보안 구멍이다. redactSemrets 표 테스트(가치 4)는 동작 변화가 없어 차선으로 돌렸다.
- 운영 영향 없음을 확인했다: `wrangler.toml:46` = pages, `:57` allowMocks=false. `.env.example:18` 의 mock 은 `LAW_DEV=true` 와 함께라 통과한다.
- 추측으로 적은 것: `loadEnv` 가 던지는 쪽과 `manual` 로 떨어뜨리는 쪽 중 **던지기**를 택한 것은 `resolveKey`(env.ts:196)의 선례에 맞춘 판단이고, 운영 문서에 명시된 정책을 본 것은 아니다. 구현자가 기동 실패를 더 무섭게 본다면 `manual` 강등 + 경고 로그도 수용 기준 1을 만족한다고 본다.
- 미확인: 샌드박스 제한으로 `npx vitest run` 을 실행하지 못해 기준선 통과 여부를 못 봤다. 구현자는 먼저 `npm run test` 를 한 번 돌려 기준선을 확인할 것.
- 조심할 것: 조건을 `!allowMocks && domains === 'mock'` 정확히 이것만으로 두고, `enumOf` 의 오타 fallback(→manual)은 건드리지 말 것. `providers/domains.ts`·`dns.ts` 의 mock 분기를 지우면 개발·e2e 가 전부 깨진다.
- [러너 08:22] scout done — 모의 도메인 공급자를 운영에서 거부 — `loadEnv` 가 `LAW_DOMAIN_PROVIDER=mock` 을 `allowMocks` 없이 받아들이는 구�

## 구현 노트
- 과제서대로 구현: `env.ts:loadEnv` 에서 `enumOf` 를 `domains` const 로 빼고 `!allowMocks && domains === 'mock'` 이면 `EnvError`. 막지 않으면 `dns.ts:10` 이 DOM-04 소유권 확인을 `mockBase` 레코드 표로 돌린다. 커밋 da035e0, 프로덕션 1파일 + 테스트 1파일.
- 확신 없는 곳·검증 못 한 것: `npm run e2e`·`acceptance`·`ui-flows`·`test:rls` 는 DB 가 필요해 **실행하지 못했다**. 다만 `scripts/ci-env.mjs` 가 `.env.example` 을 그대로 복사하고 거기 `LAW_DEV=true` 가 있어 `allowMocks` 가 기본 true 이므로 영향 없을 것으로 본다 — 추론이지 실행 증거는 아니다. 저장소 전체에서 `LAW_DOMAIN_PROVIDER`/`LAW_ALLOW_MOCK_PROVIDERS` 를 쓰는 자리는 `wrangler.toml:46,57` 과 `.env.example:18` 뿐임은 grep 으로 확인했다.
- 일부러 하지 않은 것: `enumOf` 의 오타 fallback(→`manual`)은 그대로 뒀다(던지게 하면 기동 실패 위험만 커진다). `providers/domains.ts`·`dns.ts` 는 읽기만 했다. pg/sms/identity 의 같은 종류 검사는 반환 계약이 달라(`{ok:false}` 대 throw) **통합하지 않았다**. `.env.example`·`engineering/operations.md` 도 과제서 지시대로 건드리지 않았다. `npm install` 이 바꾼 `package-lock.json` 은 이번 과제와 무관해 되돌렸다.
- 다음 역할이 조심할 것: `loadEnv` 가 던지면 워커가 아예 못 뜬다 — 조건을 넓히지 말 것. 새 테스트는 DB 없이 돈다(`npx vitest run packages/core/src/__tests__/env-mock-providers.test.ts`). 기준선은 수정 전 5 files/96 tests → 수정 후 6 files/103 tests, `npm run check` 통과(lint 0 errors / 646 warnings 는 전부 기존 파일의 것).
- [러너 08:26] brief accepted — 채택 — 과제서의 근거가 현재 코드와 정확히 일치했고(env.ts:163, dns.ts:10, wrangler.toml:46/57), 지정한 수용 기준·파일 범위·�
- [러너 08:26] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 판정 approve(risk low, 차단 없음). 가드가 실제로 구멍을 막는지 확인했다 — `dns.ts:10`·`modules/domains.ts:24` 의 mock 분기가 모두 `env.providers.domains === 'mock'` 하나만 보므로 `allowMocks` 없이는 도달 불가. 운영 기동 실패 경로도 없다(`wrangler.toml:46` pages, `:57` false, `[vars]` 가 매 배포 평문 변수 교체, `LAW_DOMAIN_PROVIDER` 는 시크릿 목록에 없음).
- 원장에 `- 실패 재현:` 줄이 없어 직접 재현했다: `main` 의 `env.ts` 로 되돌려 새 테스트를 돌리니 2개가 `expected function to throw an error, but it didn't` 로 실패(나머지 5개는 기존 동작 가드), 되돌린 뒤 작업 트리는 clean. 직접 실행: `vitest run` 6 files/103 tests 통과, `typecheck` 통과, `lint` 0 errors/646 기존 warnings.
- 못 본 것: `e2e`·`acceptance`·`ui-flows`·`test:rls`(DB 필요). CI 가 `.env.example`(LAW_DEV=true)을 복사하므로 영향 없다는 것은 구현 노트와 같은 **추론**이고, 나도 실행 증거는 없다.
- 승인이어도 남는 우려(다음 회차): `providers/domains.ts:84,128` 은 env 가 아니라 **DB 행의 `provider = 'mock'`** 을 본다 — mock 환경에서 만들어진 `law.domains` 행은 env 를 pages 로 돌린 뒤에도 `attachDomain` 을 건너뛰고 `ownership=verified` 만으로 `tls='active'` 가 된다. 이번 변경이 만든 결함은 아니고 소유 확인이 실제 DoH 를 타게 되어 경로는 좁아졌지만, 구멍이 완전히 닫힌 것은 아니다.
- 릴리즈 노트에 적을 것: 운영 동작 변화는 없고(기본값 `manual` 유지) `LAW_DOMAIN_PROVIDER=mock` + `LAW_ALLOW_MOCK_PROVIDERS` 미설정 조합만 기동을 거부한다는 점.
- [러너 08:29] review approved — 리뷰 승인 (risk=low)
- [러너 08:29] pr created — https://github.com/hkjang/law.irumx.app/pull/2
- [러너 08:34] ci passed — 검사 1개 모두 success
- [러너 08:34] merge done — da035e0
- [러너 08:37] release published — 0.1.4
