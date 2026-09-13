## 2026-09-14
- 선택: Silent SSO 표준 정합 — auto_login 기본 꺼짐과 저장소 읽기 실패 시 억제(fail-closed) (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: v1.9.0 에 이미 최상위 이동 prompt=none 자동 진입(서버 억제 쿠키·sessionStorage 표시·/login?sso=skip&error=… 주소 표시·return_to 검증·로그아웃 억제)이 있어 새로 만들지 않고 SILENT-SSO-STANDARD.md 와 어긋난 두 곳만 고쳤다. (1) `auto_login` 기본값이 켜짐이고 키가 없으면 켜진 것으로 취급하던 것을 서버 기본 설정·`oidcAutoEnabled`·설정 화면·openapi 모두 기본 꺼짐으로 바꿔 관리자가 명시적으로 켠 경우에만 prompt=none 을 보낸다. (2) 프런트 `automaticLoginSuppressed` 가 sessionStorage 예외 시 `false`(아직 안 했다)를 돌려 사생활 보호 모드에서 루프 위험이 있던 것을 `true`(이미 시도했다)로 바꿨다. Go 테스트 `TestOIDCAutomaticLoginIsOptInByDefault`(키 없음·비불리언·`?mode=auto` 요청이 IdP 접속 없이 평범한 로그인으로 감, 켜면 prompt=none 시작)와 web 테스트(차단된 저장소·저장소 없음 → 억제)를 추가/갱신했고 관리자 가이드(md·html)와 README 를 갱신했다. 검증: `go vet`, 임시 PostgreSQL 컨테이너로 `go test -race ./...`(OIDC 하위 집합 통과, 전체 스위트는 아래 참조), `npm test`(86 통과), `npm run build`(tsc 포함), `node scripts/check-docs.mjs` 통과.
- 보류 아이디어: 콜백 거절 표시를 표준 이름 `/login?sso=none` 으로도 인식·발행(현재 `sso=skip`+`error=` 조합, 기능 동등) / 프런트에서 `/login` 경로 자체에서는 자동 시도를 시작하지 않도록 pathname 제외(현재는 주소 표시·저장소·쿠키로 막힘) / 서버가 `mode=auto`+auto_login 꺼짐을 `/login` 복귀 대신 interactive 로 조용히 전환(releasedock 방식) / `oidcReturnTo` 와 프런트 `safeReturnPath` 규칙을 공유 테스트 벡터로 교차 검증

- 릴리즈: v1.10.0 (2026-09-14, run 2026-09-14-064216-hunter-improve)
