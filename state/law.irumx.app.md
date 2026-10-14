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
## 2026-10-14
- 선택: redactSecrets 계약을 표 테스트로 고정 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `pipeline.ts:496` 의 `redactSecrets` 는 서버 오류 메시지가 stderr 로그(`logFailure`)·`law_priv.record_error_alert` DB 저장·운영자 메일(`alertOperator`, :145-162)로 나가기 전의 유일한 차단기인데 테스트가 0건이었다. 신규 `packages/core/src/__tests__/redact-secrets.test.ts` 가 `redactSecrets` 를 실제로 import 해 호출하며, 치환 규칙 5개(쿼리 파라미터·Bearer/Basic·접두사 키 7종·postgres DSN·주민번호)를 규칙별 `redacted`/`unchanged` 표로 고정했다. 프로덕션 파일 0개 — `pipeline.ts` 는 읽기만 했고 정규식을 넓히지 않았다. `npm run check` 통과(lint 0 errors / 646 warnings — 기준선과 동일, 신규 파일은 경고 0 / typecheck 3개 워크스페이스 / vitest 7 files 168 tests, 기준선 6 files 103 tests + 신규 65건).
- 실패 재현: 못 함 — 성격상 결함 수정이 아니라 현재 동작을 기록하는 특성화(characterization) 테스트라 65건이 첫 실행에 모두 통과했다(정규식을 읽고 예상값을 먼저 적었고, 함수를 탐침해 맞춘 것이 아니며 예상과 실제가 전부 일치했다). 대신 테스트가 프로덕션 경로를 실제로 지나는지를 돌연변이로 증명했다 — 규칙 5줄을 하나씩 지우면 각각 20 / 6 / 10 / 5 / 7건이 실패하고(`Tests 20 failed | 45 passed (65)` 등) 복원 후 `git diff` 가 비었다. 예: 503행(주민번호) 제거 시 `AssertionError: expected '900101-1234567' to be '[주민번호 숨김]' // Object.is equality`.
- 보류 아이디어: ① env.dev 일 때 `console.error(err.stack)` 가 redact 를 거치지 않는 경로(pipeline.ts:489 — 이번에 줄 번호 확인) 1줄 수정(3/1/S, 이제 표 테스트가 있어 안전) ② util/validate.ts 경계 테스트(3/1/M) ③ loadEnv 전반 단위 테스트 집합(3/1/M) ④ isSafeLink 를 실제 링크 렌더 경로에 배선할지 조사(3/3/M) ⑤ modules/forms.ts 단계·조건 평가 순수 함수 테스트(3/2/L)
- 과제서: 채택 — 과제서의 근거가 현재 코드와 정확히 일치했고(pipeline.ts:496 규칙 5개, 호출자 2곳 :487/:145, 테스트 0건), 수용 기준 4개와 "프로덕션 0파일·정규식 확대 금지" 를 그대로 지켰다.

## 2026-10-14
- 선택: 오류 stderr 로그 세 경로를 redactSecrets 로 통과시키기 — after-commit 실패 로그가 운영에서 원문을 찍는다 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `pipeline.ts` 에서 오류를 stderr 로 쓰는 네 곳 중 `redactSecrets` 를 거치는 것은 `logFailure` 의 JSON 한 줄(:487)뿐이었다. 나머지 세 곳에 호출을 붙였다 — `:54` `Background.run` 의 catch(`env.dev` 게이트가 없어 **운영**에서 돈다. after-commit 작업 실패가 전부 이 경로로 모인다), `:111` dev 한 줄, `:489` 원본 스택. 특히 `:54` 는 `redactSecrets(String(err)).slice(0, 300)` 순서로 넣었다 — 먼저 자르면 300자에 걸친 DSN 이 뒤쪽 `@` 를 잃어 패턴이 어긋나고 비밀번호가 살아남는데, 이것을 테스트로 실증했다. 신규 테스트 3건은 `new Background()`(파이프라인이 `:208`/`:411` 에서 쓰는 바로 그 export 된 클래스, 대역 주입 없음)를 만들어 `vi.spyOn(console, 'error')` 로 실제 출력을 받는 실행 테스트다. `redactSecrets` 정규식은 건드리지 않았다(호출 지점만 늘림). 프로덕션 파일 1개 + 테스트 1개. `npm run check` 통과 — lint 0 errors / 646 warnings(기준선과 동일, 신규 파일 경고 0) / typecheck 3개 워크스페이스 / vitest 7 files 106 tests(기준선 6 files 103 tests + 신규 3건).
- 실패 재현: `AssertionError: expected '{"level":"error","message":"after-com…' not to contain 's3cret'` — 수정 전 3건 중 2건 실패(`Tests 2 failed | 1 passed (3)`). 경계 시험의 실제 출력이 `"detail":"Error: xxx…xxxpostgres://law_app:s3cret"` 로 끝나 "먼저 자르면 `@` 가 잘려 치환을 빠져나간다" 는 과제서의 경고가 추측이 아님을 확인했다. 형식 보존 시험 1건은 수정 전에도 통과(JSON 필드 이름·순서·라벨이 안 바뀜을 고정).
- 보류 아이디어: ① util/validate.ts 경계 테스트 — 모든 라우트 입력이 지나가는데 테스트 0건(3/1/M, 프로덕션 0파일) ② Background.run 실패에 requestId 를 실어 어느 요청의 후속 작업이 깨졌는지 추적 가능하게(3/2/M) ③ loadEnv 전반 단위 테스트 집합 — LAW_DATABASE_URL 없음·32바이트 아닌 키·publicUrl 파싱(3/1/M) ④ isSafeLink 를 실제 링크 렌더 경로에 배선할지 조사(3/3/M) ⑤ modules/forms.ts 단계·조건 평가 순수 함수 테스트(3/2/L)
- 과제서: 채택 — 과제서의 근거가 현재 코드와 정확히 일치했고(`:54` 에 `env.dev` 게이트 없음, `:111`/`:489`, `:487` 만 redact 통과), 수용 기준 5개와 금지 사항(정규식 확대 금지·`alertOperator` 손대지 않기·파일명 `background-redact.test.ts`)을 그대로 지켰다.

## 2026-10-14
- 선택: AT-26 서버 로그 검사가 빈 파일·없는 경로에도 통과하던 구멍 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: 우선 과제(PR #4 의 CI 실패)는 **코드 결함이 아님을 재확인**했다 — 직전 두 회차가 `gh api` 로 받아 둔 체크런 주석("The job was not started because recent account payments have failed or your spending limit needs to be increased.", 잡 소요 3초)이 원인이고 사람이 Billing 에서 풀어야 한다. 이번 회차는 그 두 회차가 **미검증으로 남긴 CI 후반 단계를 로컬 docker(postgres:17-alpine, law-irumx-dev:55460)에서 처음으로 끝까지 재현**해 코드 무결을 증명했다: main@c62b3a0 와 PR #4 내용(0804a58 + 미머지 19bf148) 모두 lint·types·단위(171)·빌드·migrate·RLS 프로브·e2e 27건·acceptance 26/26 통과. 재현 중 **acceptance 자체의 결함**을 찾아 그것을 고쳤다 — `scripts/acceptance.mjs` AT-26 의 로그 검사 5개가 빈 문자열을 훑으면 전부 통과하므로 로그 파일이 비어 있거나 경로가 틀리면 SEC-07 의 마지막 관문이 증명 없이 초록이 되고, 없는 경로는 `readFileSync` 가 던져 AT-26 의 뒤 네 검사(감사 기록·타인 메일·대량 알림·분석 스크립트)가 아예 실행되지 않았다. 내용 검사 앞에 양성 대조 한 줄(이 실행의 가려진 메일 기록 `"channel":"mail" … ***@example.test` 이 로그에 있어야 통과)을 넣고 `existsSync` 로 중단을 막았다. **워크플로·검증을 느슨하게 한 것이 아니라 조이는 방향**이며 프로덕션 코드 0파일(scripts 1파일, 7줄).
- 실패 재현: 고치기 전 — 빈 로그로 CI 와 같은 순서로 돌리면 `✓ 서버 로그에 주민번호·이메일·인증코드·증빙 원문·답변 없음` 과 `26/26 통과`(구멍 그대로 재현), 없는 경로면 `✗ AT-26 실행 - ENOENT: no such file or directory, open '…/data/server.log'` 로 뒤 네 검사가 실행되지 않음. 고친 뒤 — 두 경우 모두 `✗ 서버 로그가 이 실행의 것임(가려진 메일 기록 존재) - /tmp/empty-server.log 0B` + `25/26 통과`(뒤 네 검사는 실행됨), CI 설정(`LAW_SERVER_LOG=server.log`)에서는 `26/26 통과`.
- 보류 아이디어: ① util/validate.ts 경계 테스트 — 모든 라우트 입력이 지나가는데 테스트 0건(3/1/M, 프로덕션 0파일) ② Background.run 실패에 requestId 를 실어 어느 요청의 후속 작업이 깨졌는지 추적(3/2/M) ③ loadEnv 전반 단위 테스트 집합 — LAW_DATABASE_URL 없음·32바이트 아닌 키·publicUrl 파싱(3/1/M) ④ e2e·acceptance 의 다른 양성 대조 공백 점검 — AT-26 처럼 "없음" 을 단언하는 검사가 대상 자체를 못 읽어도 통과하는 자리 찾기(3/2/M) ⑤ modules/forms.ts 단계·조건 평가 순수 함수 테스트(3/2/L)
- 과제서: 차선 — 과제서의 지시("워크플로와 실패 단계의 스크립트·테스트를 읽고 원인을 고쳐라")대로 전 단계를 로컬 재현했지만 CI 실패 원인은 저장소 밖(Actions 결제 차단)이어서 고칠 코드가 없었고, 대신 재현 과정에서 드러난 검증 단계 자체의 빈 통과를 조이는 방향으로 고쳤다.

