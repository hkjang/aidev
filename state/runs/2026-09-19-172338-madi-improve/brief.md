# 과제서 2026-09-19 — madi (수정 과제)

## 우선 과제의 실제 원인 (정찰 판정)
- 러너가 넘긴 실패 사유는 `hold: budget` 이다. 이것은 **러너의 회차 USD 예산 소진**이지 저장소 `.github/workflows/release.yml` 의 단계 실패가 아니다.
  - 근거: `state/runs/2026-09-19-110948-madi-shepherd`(수리) 와 `2026-09-19-131831-madi-shepherd`(재심사) 는 PR #6 처리 회차이고, 13:18 회차는 `stages.json` 조차 없이(심사 노트만 남기고) 끝났다. 두 회차 모두 전체 `go test -race -timeout 45m ./...` 급 검증을 돌리다 예산에 걸린 것으로 보인다(수리 노트: "기본 10분 타임아웃을 넘겨 CI 플래그로 재실행").
  - 저장소 쪽: `release.yml`(2026-09-19 main@fd2c3b5)은 최근 커밋에서 바뀌지 않았고, GitHub 실행 이력은 이 환경에서 `gh` 승인이 없어 **미확인**. 회차 디렉터리에 CI 로그·실패 단계 출력이 없다.
- 따라서 "워크플로 스크립트·테스트를 고칠 결함"은 재현할 수 없다. 이 회차의 수정 과제는 **예산 안에서 끝나는 작은 변경 + 좁은 검증**으로 회차를 완주해 원장에 '수정 과제: hold:budget 재발 방지(검증 범위 축소, 워크플로 무변경)' 로 기록하는 것이다. **release.yml / ci.yml 은 절대 만지지 않는다.**
- 아직 안 끝난 PR #6(MCP OAuth) 재심사 지적 3건(knowledge_package.go:506 — de9c452 로 수리됨/미push, mcpOAuthProvider 뮤텍스 잡은 채 discovery, 테스트 request_id·Host 단언)은 **다른 브랜치**(PR #6) 것이라 이 main 기반 브랜치에서 손대지 말 것.

## 과제
- 과제: SAML `/api/v1/auth/saml/start` 에 `return_to` 보존 적용 (OIDC 와 동일 규칙) (가치 2 / 위험 2 / 작업량 S)
- 왜: OIDC 는 `oidcReturnTo` 로 같은 출처 경로만 받아 `oidc_attempts.return_to` 에 저장하고 콜백에서 그리로 돌려보내지만, SAML 은 `identity_saml.go:266` 에서 무조건 `/app` 으로 보낸다. 깊은 링크(문서 URL)로 도착해 SAML 로 로그인한 사용자가 매번 홈으로 떨어지는 것을 고치고, 두 SSO 경로가 같은 `return_to` 규칙을 같게 읽게 만든다.
- 수용 기준:
  1) `GET /api/v1/auth/saml/start?return_to=/app/documents/x` → ACS 성공 뒤 302 `Location: /app/documents/x`. `return_to` 없음·빈 값·`//evil`·`https://…`·`/api/...` 등 `oidcReturnTo` 가 거부하는 값은 `/app` 으로(기존 동작 그대로).
  2) 서버 재시작(스키마 재적용)에 안전: `ALTER TABLE saml_authn_requests ADD COLUMN IF NOT EXISTS return_to text NOT NULL DEFAULT '/app'` 을 `identity.sql` 에 추가(`integrations_schema.sql:30` 의 oidc_attempts 와 같은 관례). 기존 행은 `/app`.
  3) 테스트: `identity_saml_test.go` 의 `TestPostgresSAMLSignaturesStateAudienceAndRotation` (또는 같은 파일의 새 함수) 에 실제 서버·실제 DB 로 start→ACS 를 통과시켜 (a) 허용 경로가 그대로 돌아오고 (b) `//evil` 이 `/app` 으로 떨어지는 두 경우를 302 Location 으로 단언. 손으로 주입한 대역 금지 — 기존 테스트가 이미 쓰는 가짜 IdP 서명 응답 경로를 재사용.
- 건드릴 파일:
  - `internal/server/identity_saml.go:samlStart` — `returnTo := oidcReturnTo(r.URL.Query().Get("return_to"))` 를 계산해 INSERT 컬럼에 `return_to` 추가.
  - `internal/server/identity_saml.go:samlACS` — 194행 `DELETE … RETURNING request_id,configuration_hash` 에 `return_to` 추가해 스캔하고, 266행 `http.Redirect(w, r, "/app", …)` 를 `returnTo` 로. 빈 값이면 `/app`.
  - `internal/server/identity.sql` — 17행 `saml_authn_requests` 테이블 뒤에 `ALTER TABLE … ADD COLUMN IF NOT EXISTS return_to …`.
  - `internal/server/identity_saml_test.go` — 위 단언 추가.
  - (선택) `docs/admin-guide.md` SAML 절에 한 줄 — 문서 재생성(`node scripts/build-docs.mjs`) 은 Chromium 이 필요하므로 이 환경에서 못 돌면 생략하고 노트에 적을 것.
  - `oidcReturnTo`(integrations_oidc.go:107) 는 **수정하지 말고 그대로 호출**만 한다. 함수 이름이 oidc 지만 규칙은 동일해야 하므로 복제하지 않는다.
- 검증 명령 (예산 때문에 좁게 — 전체 `go test ./...` 금지):
  - `gofmt -l internal/server && go vet ./internal/server && go build ./...`
  - 임시 PostgreSQL 17 컨테이너로: `MADI_TEST_POSTGRES_DSN=postgres://madi:test-password@127.0.0.1:<port>/madi?sslmode=disable go test -race -count=1 -run 'TestPostgresSAML|TestPostgresOIDCSilentLogin|TestPostgresBackupTables' ./internal/server`
  - `go test -count=1 -run 'SAML|ReturnTo' ./internal/server` (비DB 단위)
  - 수정 전 테스트가 실패하고(Location=/app) 수정 후 통과하는 것을 노트에 적을 것.
- 위험과 피할 것:
  - `.github/workflows/*`, `scripts/verify-browser.sh`, `scripts/release-image.sh` 는 손대지 않는다(우선 과제 규칙: 워크플로 느슨하게 금지, 그리고 워크플로 결함이 확인되지 않음).
  - `backup_tables.go:61` 이 `saml_authn_requests` 를 임시 테이블로 등록하고 있으므로 백업 목록 변경 불필요 — 열만 추가하고 테이블 이름·등록은 바꾸지 말 것.
  - `maintenance.go:188` 의 만료 삭제는 그대로.
  - 쿠키(`samlBrowserCookie`)·RelayState 길이 검사·서명 검증 순서는 바꾸지 말 것(auth 보호 경로). `return_to` 는 RelayState 에 싣지 말고 DB 행에만 둔다(OIDC 와 동일하게 서버가 보관).
  - PR #6 브랜치 파일(`integrations_mcp_oauth*.go`, `knowledge_package.go`) 은 이 브랜치에 없다/건드리지 않는다.
  - 검증에 전체 스위트·브라우저 시험을 돌리지 말 것 — 이번 회차가 실패한 이유가 예산이다.
- 차선 후보: OIDC 콜백의 madi 쪽 검증 오류(상태 불일치·만료 400/409)도 `/login?error=` 로 안내 — 단, 기존 `TestPostgresOIDCSilentLogin` 의 400 기대치를 여러 곳 바꿔야 하므로 1순위보다 크다. 그것도 어려우면 `oidcReturnTo` 단위 테스트만 보강(S, 위험 1).
