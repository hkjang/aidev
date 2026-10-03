# PR 처리기 노트 2026-10-04-052107-yeopjari-shepherd — yeopjari PR #1
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-04-043727-yeopjari-improve)
# 회차 노트 2026-10-04-043727-yeopjari-improve — yeopjari
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 04:37] base pinned — main@6b62fe1
- [러너 04:37] autonomy release — 
- [러너 04:46] scout timeout — 단계 제한 시간 초과
- [러너 04:46] scout done — 클라이언트가 보낸 퍼센트 인코딩 때문에 요청이 500 이 되고 운영자가 오류를 지울 수 없는 문제 (가치 4 /

## 구현 노트
- 무엇·왜: `readCookies` 의 값 디코딩을 쿠키 단위 try/catch 로 감싸(router.ts:97-103 과 같은 방식) `Cookie: yj_s=%` 하나가 500 + 운영자 알림이 되는 경로를 막고, `admin/errors/:signature` 핸들러의 중복 디코딩을 없앴다. 프로덕션 파일 2개(cookies.ts, ops.ts) + 테스트 1개. 커밋 6d4e3a4.
- 확신 없는 곳: **`ops.ts` 변경은 핸들러를 실제로 지나는 red 테스트가 없다.** 핸들러 호출에는 `Ctx`(sql/audit/principal 전체)가 필요하고 운영자가 손으로 만든 주입 객체로 결함을 증명하는 것을 금지했으므로, 과제서 기준 3(b) 대로 프로덕션 `adminOpsRoutes()` 라우터에 실제 등록된 패턴을 match 해서 "웹의 `encodeURIComponent(signature)` 가 `params.signature` 로 원본 그대로 도착한다 / 한 번 더 디코딩하면 값이 달라진다" 를 라우터 쪽에서 고정했다. 즉 두 번째 디코딩이 해롭다는 것은 증명했지만, 그것을 지운 뒤 DELETE 가 실제로 행을 지우는 것은 DB 없이 확인하지 못했다 — 비평가가 먼저 볼 곳.
- 쿠키 쪽은 red → green 을 눈으로 확인했다(고치기 전 3건이 `URIError: URI malformed ... cookies.ts:39:27`).
- 일부러 안 한 것: `db.ts:64,82-83`·`mail/smtp.ts:30-31` 의 같은 계열 보호 없는 디코딩(입력이 운영자 설정값이라 공격면이 다르고, 거기선 "던지지 않기" 보다 "어느 설정이 왜 못 읽히는지 말하기" 가 맞는 수정이다). `auth/session.ts`·`principal.ts`·`pipeline.ts` 의 세션 판정, 웹 `Errors.tsx` 의 인코딩(라우터 계약이 정본이라 그대로 둠), `.github/workflows`·마이그레이션도 손대지 않았다.
- 다음 역할이 조심할 것: 새 테스트는 DB·서버가 필요 없다(`npx vitest run packages/core/src/__tests__`, 9 파일 118건 통과). 워크트리에 node_modules 가 없으면 `npm ci` 가 먼저다. `scripts/e2e.mjs` 는 이 변경에 필요 없어 실행하지 않았다 — 미확인. 정상 인코딩 값(`abc%20def` → `abc def`)의 해석이 그대로인지를 고정하는 테스트가 있으니, 쿠키 파서를 더 만지려면 그 케이스를 먼저 볼 것.
- [러너 04:51] brief accepted — 채택 — 근거가 지금 코드와 정확히 맞았다(cookies.ts:39, ops.ts:518, router.ts:97-103 선례까지). 다만 수용 기준 3(b) 는 라우터 왕
- [러너 04:51] verify passed — 검증 6개 통과 (auto)
- [러너 04:51] pr created — https://github.com/hkjang/yeopjari/pull/1
- [러너 04:51] guard held — packages/core/src/auth/cookies.ts 

## 심사 노트
- 확인한 것: origin/main 의 cookies.ts·ops.ts 를 되돌려 red(3건, `URIError ... cookies.ts:39:27`) → HEAD green(9건) 을 눈으로 봤다. 쿠키 테스트는 실제 `new Request` + 프로덕션 `readCookies` 를 쓴다. `npm run check`(118건) 와 `npm run build` 통과.
- 구현 노트의 "확신 없는 곳"(ops.ts)은 프로덕션 URL 경로를 직접 실행해 풀었다: `new Request` → `new URL().pathname` 이 `%2F/%20/%25` 를 보존하고 router.ts:100 이 한 번만 디코딩해 저장 서명을 그대로 준다. ASCII 서명 3종에서 제거된 호출은 항등 — 기존에 동작하던 경로에 회귀가 없다.
- 못 본 것: `DELETE ... WHERE signature = $1` 의 실제 행 삭제(DB 없음). 다만 바인딩 값과 SQL 이 이전과 바이트 동일해 위험이 남지 않는다. `admin error signature round trip` 4건은 수정 전에도 통과한다(핸들러를 지나지 않음) — 회귀가 테스트로 고정되지 않은 약점이나, Ctx 대역 금지 + DB 없는 하니스에서 대안이 없어 반려 사유로 보지 않았다.
- 보호 파일 점검: 새 공개 경로·권한 확대·비밀 노출·마이그레이션·의존성 변경 0. catch 는 값을 버리기만 하고 로그하지 않으며, 할당 전 throw 라 같은 이름의 정상 쿠키를 덮지 않는다. CSRF 는 헤더/세션 비교이므로 약해지지 않는다. 미인증 공격자의 `Cookie: yj_s=%` 500 + 운영자 메일 경로가 닫힌다.
- 권고: approve / merge, risk medium(쿠키 파싱). 참고로 priv.error_alerts 의 오류 원문 저장·감사 노출은 이 diff 가 들여온 것이 아니라 별 과제다.
