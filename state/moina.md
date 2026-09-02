# MOINA 자율 개선 기록

## 2026-09-02
- 선택: 반복된 `X-Forwarded-For` header 줄을 하나의 chain으로 이어 붙이기 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `backend/internal/httpapi/network.go`의 `forwardedAddresses`가 `Header.Get`으로 첫 번째 `X-Forwarded-For` 줄만 읽어, Proxy가 자기 hop을 별도 header 줄로 덧붙이는 구성에서는 Client가 미리 보낸 줄이 chain 전체를 대신했고 `clientIP`가 위조 값으로 계산되어 감사 기록과 공유 요청 한도 bucket이 오염되었습니다. 같은 함수의 `Forwarded`·`X-Forwarded-Proto` 분기가 이미 쓰던 `Header.Values`로 바꿔 받은 순서대로 이어 붙이도록 고치고, 위조 줄 + Proxy 줄 조합과 신뢰 hop 투명성을 검증하는 테스트 2개를 추가했으며(수정 전 코드에서 실패하는 것을 확인) `docs/configuration.md`에 이 동작을 한 문장 명시했습니다. 검증은 `make fmt`, `make check`, `go test -race ./...`, `go vet ./...`, staticcheck 전부 통과(frontend는 변경 없어 실행 생략).
- 보류 아이디어: Email 알림에 조용한 시간 미적용 — 문서상 Toast·Desktop 전용이 의도라 변경 보류(가치 2 / 위험 3 / M) · `formatRelativeTime`이 연도가 다른 Moin을 월·일만 표시해 모호함(가치 2 / 위험 2 / S) · `frontend/src/utils/format.ts` 단위 테스트 부재(가치 2 / 위험 1 / S) · Makefile `test`가 CI와 달리 `-race` 미사용(가치 2 / 위험 1 / S)
- 릴리즈: v0.1.16 (2026-09-02)
