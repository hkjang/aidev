## 2026-10-08
- 선택: isSafeLink — 루트 라벨(끝 점)이 붙은 내부 호스트가 차단 목록을 비껴가는 결함 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `isSafeLink`(packages/core/src/util/text.ts)는 끝 점을 떼고 라벨 수·localhost 를 검사하면서도 `BLOCKED_LINK_HOSTS` 조회만은 떼지 않은 호스트로 했다. WHATWG URL 은 끝 점을 IPv4 리터럴에서만 지우고 이름에는 남기므로(`new URL('http://metadata.google.internal./').hostname === 'metadata.google.internal.'`, node 22 로 직접 확인) GCP 메타데이터 호스트가 공개 호스트로 통과해, 함수 주석이 약속한 내부 호스트 차단(COMM-03)이 깨졌다. 끝 점을 한 번만 떼어 라벨 검사와 차단 목록 조회가 같은 값을 쓰게 고쳤다(프로덕션 파일 1개). 검증: 차단 표에 끝 점 형태 2건, 허용 표에 공개 호스트의 루트 라벨 표기 1건을 먼저 추가해 실패를 확인하고, 고친 뒤 `npm run check`(eslint + 3개 워크스페이스 tsc --noEmit + vitest 96건) 전체 통과.
- 실패 재현: `FAIL packages/core/src/__tests__/text.test.ts > isSafeLink > rejects metadata hostname with a trailing dot` / `AssertionError: expected true to be false` (2 failed | 61 passed)
- 보류 아이디어: ① `redactSecrets`(pipeline.ts, 로그·운영 알림 메일·law_priv.error_alerts 에 모두 쓰임) 테스트 0건 — 주민번호·Bearer·DSN 패턴의 계약을 표로 고정 ② `util/validate.ts`(모든 라우트 입력 경로) 테스트 0건 — optional/array/int 경계와 한국어 조사 메시지 고정 ③ `isSafeLink` 는 현재 저장소 내 호출자가 없음 — 사용자 콘텐츠 링크 렌더 경로에 실제로 배선할지 판단 ④ `loadEnv`: `providers.domains='mock'` 이 `allowMocks` 로 막히지 않아 운영에서도 선택 가능(pg·sms·identity 는 모두 막힘) ⑤ `toResponse` 204 분기가 `extra` 를 `set` 으로 복사해 set-cookie 를 1개로 줄이고 runRoute 가 다시 append — 중복 Set-Cookie(현재 무해, bare 라우트는 쿠키를 쓰지 않음)
- 과제서: 기각 — 정찰 과제서가 비어 있어 1~4단계를 직접 수행했다.
