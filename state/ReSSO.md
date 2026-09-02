# ReSSO 자율 개선 기록

## 2026-09-02
- 선택: JWKS Cache-Control이 writeJSON에 덮어써지던 문제 수정 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공 (커밋 35fd2d9)
- 요약: `jwks` 핸들러가 설정한 `Cache-Control: public, max-age=300`을 `writeJSON`이 `no-store`로 무조건 덮어써서, 코드는 캐시 가능하다고 말하는데 응답은 아니라고 말하는 상태였다(직접 재현해 확인). `writeJSON`의 `no-store`를 규칙이 아닌 기본값으로 바꾸고, JWKS의 max-age는 인스턴스 자신의 키 집합 캐시 수명인 `store.SigningKeyTTL`(30초)에서 가져오도록 했다 — 기존 300초는 그 창의 10배라 회전 직후 최대 5분간 새 `kid` 검증이 실패할 수 있어 그대로 되살리지 않았다. 캐시 가능해진 응답의 CORS 헤더가 Origin에 의존하므로 `Vary: Origin`을 허용된 Origin일 때만이 아니라 항상 붙이도록 했다. 검증: 새 테스트 2개(단위 + 연동)가 수정 전 코드에서 실제로 실패함을 확인했고, `go test -race ./...` 전체 통과(연동 테스트 SKIP 0건), `go vet`, `golangci-lint`(0 issues), `govulncheck`(0), `npm run lint`, `npm run test`(22파일/104테스트, 디스크 파일 수와 일치), `npm run build` 모두 통과. 빌드가 만든 `webui/dist/index.html` 해시 변경은 무관하므로 되돌렸다.
- 보류 아이디어:
  - Discovery에 `authorization_response_iss_parameter_supported: true` 추가 — 서버는 이미 인가 응답에 RFC 9207 `iss`를 넣는데 메타데이터로 알리지 않아 RP가 강제 검증을 켜지 못한다 (가치 3 / 위험 1 / S)
  - UserInfo POST에서 form-encoded `access_token` 파라미터 수용 (RFC 6750 §2.2). 현재는 Authorization 헤더만 읽는다 (가치 2 / 위험 1 / S)
  - `authorization` 엔드포인트가 기존 SSO Session 재사용 시 계정 상태를 다시 보지 않아, 토큰 교환에서야 거절될 코드를 발급하는 경로가 있는지 점검 (가치 2 / 위험 2 / M)
  - 잠긴(locked) 계정의 기존 SSO Session이 계속 새 인가 코드를 받는 동작을 의도된 것으로 문서화할지 검토 (가치 2 / 위험 2 / S)
  - `login`에서 `prompt=login` 재인증 시 기존 Session을 정리하지 않아 한 사용자에게 Session이 둘 남는 문제 (가치 2 / 위험 2 / M)
- 릴리즈: v0.9.66 (2026-09-02)
