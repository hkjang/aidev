# 과제서 (2026-09-24, base main@bb652c9 / v0.2.18)

- **과제: Vite 개발 서버에 MCP OAuth 보호 리소스 메타데이터(`/.well-known/oauth-protected-resource`) 프록시 추가 (가치 2 / 위험 1 / 작업량 S)**

## 왜
Go 서버는 `internal/httpapi/server.go:150-151`에서 `GET /.well-known/oauth-protected-resource` 와 `GET /.well-known/oauth-protected-resource/mcp` 를 `s.protectedResourceMetadata` 로 내보내지만, `web/vite.config.ts` 의 `server.proxy` 에는 이 경로가 없어(현재 항목은 `/api/openapi.json`, `/api/v1`, `/v1`, `/mcp`, `/healthz`, `/readyz` 6개뿐) 개발 서버(5173)에서는 SPA index.html 로 떨어집니다. 그 결과 개발 중에는 `/mcp` 가 내려 주는 `WWW-Authenticate: Bearer … resource_metadata="…"`(`internal/httpapi/mcp_oauth.go:106`) 와 관리 화면이 보여 주는 메타데이터 주소(`web/src/pages/admin/SettingsPage.tsx:505`, 기본값 `window.location.origin`)가 모두 5173 을 가리키는데 그 주소를 따라가면 JSON 대신 HTML 이 와서 RFC 9728 디스커버리가 끊깁니다 — 2026-09-22 회차가 같은 부류의 `/api/openapi.json` 누락만 고치고 이 항목은 보류 아이디어로 남겨 둔 자리입니다.

## 수용 기준
1. 개발 Vite 서버로 `GET /.well-known/oauth-protected-resource` 와 `GET /.well-known/oauth-protected-resource/mcp` 를 부르면 (Accept 가 `text/html` 이든 `application/json` 이든) 요청이 그대로 백엔드 8080 으로 전달되고 응답 본문·Content-Type 이 백엔드 것과 같다. 쿼리스트링과 퍼센트 인코딩은 원형 유지.
2. `/.well-known/` 아래 다른 경로(예: 브라우저·개발도구가 자동으로 부르는 `/.well-known/appspecific/com.chrome.devtools.json`)는 백엔드로 넘어가지 않고 기존처럼 Vite 가 처리한다. 즉 프록시 키는 `/.well-known` 전체가 아니라 `/.well-known/oauth-protected-resource` 접두사 하나여야 한다(Vite 문자열 키는 접두사 매칭이라 이 하나로 `…/mcp` 까지 덮인다).
3. 기존 6개 프록시 항목과 SPA fallback 동작(`/api-explorer`, `/api/unrelated` 는 HTML)은 그대로다.
4. 테스트가 증명할 것: 수정 **전** 에 새 케이스가 실패(HTML 이 돌아오고 upstream 수신 기록이 비어 있음)하고, 수정 **후** 에 통과한다는 것. 실제 Vite 개발 서버와 실제 HTTP 왕복으로 증명할 것(문자열 검사·설정 객체 직접 조작 금지).

## 건드릴 파일
- `web/vite.config.ts` — `server.proxy` 에 `'/.well-known/oauth-protected-resource': 'http://127.0.0.1:8080'` 한 줄 추가. 다른 항목의 순서·형식(문자열 축약형) 유지.
- `web/src/lib/vite-proxy.test.ts` — 기존 하네스(실제 `loadConfigFromFile` 로 운영 `vite.config.ts` 를 읽어 upstream 포트만 바꾸고 `createViteServer` 로 띄우는 node 환경 테스트)를 그대로 재사용해 케이스 추가:
  - `it.each(['/.well-known/oauth-protected-resource', '/.well-known/oauth-protected-resource/mcp'])` × Accept 두 값 — 전달 확인(기존 `preserves existing proxy for %s` 케이스와 같은 형태로 `received` 배열 검증).
  - 기존 `keeps %s on the SPA server` 목록에 `/.well-known/appspecific/com.chrome.devtools.json` 추가(수용 기준 2). 이 경로에서 Vite 가 200 HTML 이 아니라 404 를 줄 수도 있으니 단정하기 전에 실제 응답을 먼저 확인하고, 핵심 단정은 **`received` 에 아무것도 안 쌓였다** 로 둘 것.
- `docs/` — 개발 실행 안내에 `/api/openapi.json` 프록시를 적어 둔 자리가 있으면(2026-09-22 회차가 추가함) 같은 문장에 이 경로를 덧붙이는 정도만. 없으면 문서는 건드리지 말 것.

## 검증 명령
```
npm --prefix web ci --no-audit --no-fund
npm --prefix web test -- src/lib/vite-proxy.test.ts     # 먼저 수정 전 실패 확인
npm --prefix web test -- --maxWorkers=1
npm --prefix web run lint
go test ./internal/httpapi/
./scripts/verify.sh                                     # 마지막에 전체
```
(현재 작업 트리에 `web/node_modules` 가 없다. 로컬 Node 는 v22.23.1, CI 는 24 — 정찰에서는 npm 을 돌리지 않았다.)

## 위험과 피할 것
- **Go 쪽·`internal/httpapi/mcp_oauth.go`·`server.go` 는 손대지 말 것.** 라우트는 이미 있고 이번 결함은 순수히 개발 서버 배선이다. 리소스 식별자 결정 순서(`mcp.oauth.resource` → `oidc.redirect_url` Origin → `Host`)를 "개선"하려 들지 말 것 — 계약이자 보호 구역이다.
- 프록시 키를 `/.well-known` 로 넓히지 말 것(수용 기준 2). 운영자 지시: 한쪽 파서·매처를 넓히지 말고 좁게 고칠 것.
- `changeOrigin`·`rewrite` 등 옵션을 붙이지 말 것. 기존 항목은 전부 문자열 축약형이고, Host 를 바꾸면 `mcp_oauth.go` 의 `Host` 폴백이 내는 주소가 달라져 계약이 흔들린다.
- 인증·세션·migrations·`.github/workflows/` 는 이번 범위 밖.
- e2e/캡처(`scripts/e2e-docker.sh`, Playwright)는 `docs/screenshots/*.png` 를 덮어쓰므로 돌리지 말 것.
- 이번 하네스는 upstream 이 고정 JSON 을 돌려주는 전송 픽스처다. "OAuth 가 꺼져 있으면 404" 같은 **의미**는 Go 테스트(`internal/httpapi/mcp_oauth_test.go`)의 몫이니 vitest 에서 흉내 내지 말 것.

## 미확인 (정찰이 확인하지 못한 것)
- 실제 `npm run dev` 를 띄워 해당 경로가 HTML 을 돌려주는 것을 **눈으로 확인하지 않았다**(`web/node_modules` 없음). 근거는 `vite.config.ts` 에 항목이 없다는 정적 사실과, 동일 구조의 `/api/openapi.json` 이 2026-09-22 회차에서 같은 하네스로 재현됐다는 기록이다. 구현자는 수정 전 실패를 반드시 먼저 관찰할 것.
- 실제 MCP 클라이언트·Keycloak 으로 개발 서버에 붙는 흐름은 이 환경에 제공자가 없어 미검증.

## 차선 후보
**Transit 기능 안내의 batch 지원·버전 표기 정합성 맞추기 (가치 2 / 위험 1 / 작업량 S).** `web/src/pages/FeaturePage.tsx:23` 은 `'v0.2.0은 AES-256-GCM 단건 encrypt/decrypt만 제한 제공합니다'` 라고 쓰고 `next` 에 `'Batch operation'` 을 올려 두었지만, `docs/guides/compatibility.md:26,119-130` 과 `internal/httpapi/openbao_transit_batch_test.go` 는 `batch_input`/`batch_results` 가 이미 지원됨을 못박는다(현재 버전도 v0.2.18). 같은 파일 `:24` 의 `'Batch 암호화'` 도 같은 문제. FeaturePage 문구만 현재 지원 범위에 맞추고(단건→단건·batch, `next` 에서 batch 제거, 버전 문구는 하드코딩 대신 제거하거나 현재 표기로), `SettingsPage.test.tsx` 계열은 건드리지 말 것. 서버 동작은 바꾸지 않는다.
