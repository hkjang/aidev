## 2026-09-23
- 선택: guide-shots 복원 루프가 PUT 한 건의 예외로 중단되지 않고 네 설정을 모두 되돌린다 (가치 3 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: `withGuideSettings` 의 finally 복원 루프에서 각 `call('PUT', …)` 을 개별 try/catch 로 감싸 실패를 `note('복원 …', false, …)` 로 남기고 다음 설정으로 계속 진행하게 했고, 본문(seed/capture/captureTracking)이 던진 오류는 래핑 없이 원본 그대로 재전파하며 본문이 성공했을 때만 실패 경로를 담은 복원 오류를 던진다. 백업 사전 검증 블록·복원 순서·`{value: …}` 래핑은 건드리지 않았다. 검증: 기존 vm 하니스에 `failWrite`/`writeFailure` 옵션을 더해 실제 guide-shots.mjs 소스 블록과 실제 모듈로 16건을 추가했고, 수정 **전** 코드에서 8건이 옳은 이유로 실패하는 것을 먼저 확인했다(network 는 첫 PUT 뒤 나머지 3건이 기록되지 않음, http 503 은 4건 다 나가지만 아무 오류도 던지지 않음). 수정 후 `cd web && node --test scripts/guide-settings-check.test.mjs` 64건 전부 통과, 인접 3개 파일 포함 121건 통과, `node --check` 두 스크립트, `go test ./internal/api ./cmd/runtime-proxy` 통과 (커밋 7ea3ee5).
- 보류 아이디어: guide-shots API 호출에 요청별 제한 시간(AbortSignal) — call 의 page.evaluate fetch 에 기한 없음 (3/2/M) / guide-shots 추적 캡처가 CSP 위반 기록 전체를 삭제 (3/2/M) / 게이트웨이 reporter.send non-2xx 를 로그로 — BASE_VERSION 상향 필요 (2/2/S) / dlp.tool 보고의 server·tool·direction 문자열 길이 상한 (2/2/S)
- 과제서: 채택 — finally 루프의 개별 catch 부재, 하니스가 non-GET 실패를 흉내낼 수 없던 점, 기존 테스트 78-92행의 원본 오류 기대까지 과제서의 근거가 현재 코드와 정확히 일치했다.
