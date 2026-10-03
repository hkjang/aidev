- 과제: 방문 상세의 「알림 재발송」이 실패를 말없이 삼키고 성공도 거짓으로 단정하는 것 닫기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `web/src/pages/VisitsPage.tsx:89` 의 「알림 재발송」 버튼만 이 파일에서 유일하게 `try/catch` 없는 인라인 `onClick={async () => { await postJSON(...); setNotice(...) }}` 이다 — 서버가 409 를 주면 `setNotice` 에 닿기 전에 throw 되어 **화면에 아무 변화도 없고**(unhandled rejection) 운영자는 눌렀는지조차 알 수 없다. 반대로 성공하면 응답 `{queued:N}` 을 버리고 늘 "알림을 다시 등록했습니다" 로 단정하는데, 같은 파일의 `cancelSeries` 는 `result.cancelled` 를 읽어 건수를 보고하므로 저장소 자체 관례에서 벗어난 유일한 자리다.
- 왜(근거, 실제로 읽은 것):
  - `internal/app/visits.go:1213 resendVisitNotification` 은 세 가지 실패를 **한국어 메시지와 함께** 돌려준다: `403 forbidden`("재발송 권한이 없습니다"), `409 visit_not_notifiable`("진행 중인 방문만 방문 안내를 재발송할 수 있습니다", visits.go:1231-1234), `409 no_valid_qr`("재발송할 수 있는 유효한 방문자 QR이 없습니다", visits.go:1266-1269). 성공은 `writeJSON(w, 200, map[string]int{"queued": queued})`(visits.go:1291).
  - 버튼 렌더 조건은 `["APPROVED","SCHEDULED","ARRIVED","CHECKED_IN"].includes(detail.visit.status)` 로 서버 허용 집합과 같지만, **다이얼로그가 열려 있는 동안 상태가 바뀌면** 어긋난다. 다른 운영자가 그 방문을 취소하거나 자동 퇴실이 돌면 같은 버튼이 409 `visit_not_notifiable` 을 받는다.
  - `no_valid_qr` 은 `qrAvailableForNotification`(visits.go:1135-1137)이 `!now.After(*validUntil)` 를 요구하므로, 방문 창이 지났는데 상태가 아직 SCHEDULED 인 방문에서 도달한다. 이 경로는 재현 비용이 커서 e2e 재현은 아래 `visit_not_notifiable` 쪽을 쓸 것.
  - 이 엔드포인트에 Go 테스트가 **0건**이다(`grep -rn "resend\|no_valid_qr\|visit_not_notifiable" internal/app/*_test.go` 빈 출력).
- 수용 기준:
  1) 「알림 재발송」이 실패하면 다이얼로그 위에 **서버가 준 한국어 메시지**가 기존 `error` Alert 로 보인다(예: `진행 중인 방문만 방문 안내를 재발송할 수 있습니다`). 지금은 아무것도 보이지 않는다.
  2) 실패했을 때 성공 notice(`알림을 다시 등록했습니다`)가 **뜨지 않는다**. 또 콘솔에 unhandled rejection 이 남지 않는다.
  3) 성공하면 notice 가 응답의 실제 건수를 읽어 보고한다(`cancelSeries` 와 같은 형태, 예: `알림 N건을 다시 등록했습니다`). `queued === 0` 이면 "다시 등록했습니다" 라고 단정하지 않는다.
  4) 다른 동작(상세 보기·취소·QR 재발급·사전등록 링크·일정 수정)의 메시지와 동작은 그대로다 — 회귀 없음.
  5) 테스트가 증명할 것: 실제 서버가 409 를 주는 조작에서 **수정 전에는 화면이 침묵**하고 수정 후에는 그 서버 메시지가 보인다(수정 전/후 짝 실행). 그리고 Go 통합 테스트 1개가 `visit_not_notifiable` 과 성공 시 `queued>=1` 을 고정한다.
- 건드릴 파일 (프로덕션 1개):
  - `web/src/pages/VisitsPage.tsx` — 인라인 `onClick={async …}`(89행 `DialogActions` 안, `startIcon={<SendRounded />}` 버튼)을 **이름 붙은 함수**로 꺼낸다. 같은 파일의 `cancelSeries`(55행)·`reissue`(58행)를 그대로 본떠서:
    `const resendNotification = async (id: string) => { try { const result = await postJSON<{ queued: number }>(\`/api/v1/visits/${id}/notifications/resend\`, {}); setNotice(...result.queued...); } catch (e) { setError(e instanceof Error ? e.message : "알림을 다시 등록하지 못했습니다"); } };`
    버튼은 `onClick={() => void resendNotification(detail.visit.id)}` 로 바꾼다. `api.ts` 는 건드리지 않는다(아래 위험 참고).
  - `internal/app/integration_test.go` — 테스트 1개 추가(예: `TestResendVisitNotification`): 승인된 방문에 `POST /api/v1/visits/{id}/notifications/resend` → 200 과 `queued>=1`, 그 방문을 취소한 뒤 같은 호출 → `409 visit_not_notifiable`. 기존 `visitToday` 헬퍼를 쓸 것(자정 경계 구멍 회피, v2.8.12 교훈).
  - (선택) `docs/USER_GUIDE.md` 한 문장. PDF 는 재생성하지 않는다.
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd web && npm ci && npm run lint && npm test && npm run build` — `lint`=`tsc -b --pretty false`, `test`=`vitest run`. 직전 기준 **88개 통과**. `.tsx` 는 `web/vite.config.ts:10` 의 `include: ["src/**/*.test.ts"]` 때문에 **조용히 0개로 수집된다** — 이 변경에 vitest 를 새로 붙이려 하지 말 것.
  - `VISITFLOW_TEST_DSN='postgres://…' go test ./... -count=1` (DSN 없으면 DB 통합이 SKIP — PASS 가 실행을 뜻하지 않는다. internal/app 약 55~60초, CREATE DATABASE 권한 필요), `go vet ./...`, `go build ./...`, `gofmt -l .`(빈 출력), `git diff --check`.
  - `bash scripts/local-e2e.sh` — 저장소에 들어온 한 명령(v2.8.13). 실제 `npm run build` 번들을 `cmd/visitflow/webdist` 에 임베드하고 전용 도커 네트워크로 PostgreSQL+서버를 띄워 실제 Chromium 으로 돈다. 기준 **10 passed / 종료 코드 0**. 포트는 박지 말 것 — 스크립트가 OS 에 맡긴다(고정이 필요하면 `VISITFLOW_LOCAL_E2E_PORT`).
- 실패 재현(권장 경로, 가짜 객체 없이 실제 서버 응답으로):
  1) `scripts/local-e2e.sh` 안에서 임시 집중 스펙을 하나 두고, 방문을 하나 만들어 승인(또는 자동 승인 설정)해 상태를 SCHEDULED 로 만든다.
  2) 상세 다이얼로그를 열어 「알림 재발송」 버튼이 보이는 것을 확인한다.
  3) 다이얼로그를 **열어 둔 채로** Playwright `request`(같은 세션 쿠키)로 `POST /api/v1/visits/{id}/cancel` 을 호출한다.
  4) 「알림 재발송」을 클릭한다 → 서버가 실제로 `409 visit_not_notifiable` 을 준다. **수정 전**: Alert 도 notice 도 없고 화면이 그대로다(이것을 `OBSERVED` 로 적을 것). **수정 후**: `진행 중인 방문만 방문 안내를 재발송할 수 있습니다` 가 보인다.
  - 성공 notice 의 건수는 `page.route` 로 응답을 위조하지 말고 실제 200 응답의 `queued` 로 확인할 것. 네트워크 주입이 꼭 필요하면 **오류 경로만** 주입하고 성공은 `route.continue()` 로 실서버 것을 쓴다.
- 위험과 피할 것:
  - **`web/src/api.ts`(61행)를 건드릴 필요가 없고 건드리지 말 것** — 18개 페이지가 전부 쓰는 공용 fetch 헬퍼라 시그니처를 바꾸면 블라스트 반경이 저장소 전체가 된다. 이번에 읽어 **확인**했다: `api()` 는 `!response.ok` 일 때 `new APIError(status, data.error?.code ?? "request_failed", data.error?.message ?? \`요청 실패 (${status})\`)` 를 throw 하고 `APIError extends Error`(api.ts:14-22, 36-48) 이므로 `e instanceof Error ? e.message` 가 **서버의 한국어 메시지 그대로**를 준다. 수용 기준 1 은 추가 배선 없이 성립한다.
  - 서버 `resendVisitNotification` 의 상태 집합·권한 검사·`qrAvailableForNotification` 을 **고치지 말 것**. 이번 과제는 화면이 서버의 답을 그대로 전하게 만드는 일이다(같은 값을 읽는 두 경로를 어긋나게 늘리지 않는다).
  - 버튼 렌더 조건(`["APPROVED","SCHEDULED","ARRIVED","CHECKED_IN"]`)을 넓히거나 좁히지 말 것 — 서버 집합(visits.go:1231)과 지금 일치한다.
  - 같은 파일의 요청 경합(`load`/`loadMore` 의 늦은 응답 역전)은 **이번 범위가 아니다**. 2026-10-02 의 요청 티켓(`requestSeq`) 접근은 머지되지 않았고(`grep -rn requestSeq web/src/` 0건 — 이번 회차 재확인) 같은 접근 재제출은 금지다. 손대지 말 것.
  - 보호 경로를 건드리지 않는다: auth/session/OIDC, `internal/database/` 마이그레이션, `.github/workflows/`, `scripts/release-image.sh`, `cmd/visitflow/main.go` 의 `:8080`.
  - 끝내고 `cmd/visitflow/webdist/` 에 추적된 스텁 `index.html` 하나만 남는지 확인할 것. 되돌리기는 **임시 복사본**으로 — `git checkout --` 로 미커밋 테스트를 잃은 기록이 있다. `test-results/` 와 임시 스펙은 지운다.
- 차선 후보: **e2e 스펙이 방금 만든 방문 한 건을 결정적으로 집는 헬퍼** (가치 3 / 위험 1 / S, 프로덕션 코드 0개) — `web/e2e/visit-flow.spec.ts` 에 제출 완료 화면의 방문번호(`VF-…`)를 읽어 그 번호로 목록 검색창을 채워 정확히 1행을 돌려주는 헬퍼를 둔다. 최근 두 회차가 브라우저 검증 시간을 전부 여기서 먹었다(`getByRole('row').filter({hasText})` 가 5~7행으로 strict mode violation, 목록 검색창은 **방문자 이름으로 걸러지지 않고** 방문번호/회사/담당자만 본다 — VisitsPage.tsx:82 의 placeholder `"방문번호 / 회사 / 담당자 검색"` 로 확인). 1순위의 재현(열린 다이얼로그 + 외부 취소로 409 를 받는 것)이 e2e 에서 성립하지 않으면 이쪽으로 갈 것 — 그리고 그때 만든 헬퍼로 1순위를 다시 시도하면 된다.

- 미확인(추측으로 적은 것, 구현자가 먼저 확인할 것):
  1) `queued === 0` 이 실제로 일어나는지는 **미확인**이다. `queueNotificationEventCountTx` 를 읽지 않았고, visits.go:1270-1274 의 UPDATE 가 `rule_id IS NULL AND template_key='visitor_pass'` 기본 경로를 언급하므로 보통은 1 이상일 것으로 보인다. 수용 기준 3 은 "건수를 읽어 보고한다" 가 본질이고 `queued===0` 분기는 방어적인 것이다 — 도달 불가로 보이면 분기 대신 건수만 보고해도 된다.
  2) e2e 에서 방문을 SCHEDULED 로 만드는 가장 짧은 경로(자동 승인 설정인지, `/api/v1/visits/{id}/approve` 호출인지)는 **미확인**이다. `web/e2e/visit-flow.spec.ts` 의 기존 승인 흐름을 먼저 읽을 것.
  3) 「알림 재발송」 버튼의 현재 침묵을 **수정 전 번들로 브라우저에서 직접 본 것은 아니다** — 코드(try/catch 부재)와 서버의 409 경로를 읽어 확정했다. 수정 전/후 짝 실행으로 실제 확인할 것(직전 회차가 바로 이 짝 실행에 실패해 한계로 남겼다).
