# Invenqor Server·Agent v0.2.33 릴리즈 노트

릴리즈 일자: 2026-09-12
호환 Agent: v0.2.33 (Linux·Windows)

이번 릴리즈는 기능 하나를 더합니다. 자리는 **관리 콘솔의 방문 추적**이고, 문제는
지금까지 그것이 **조용히 불가능했다**는 점입니다. 콘솔은
`Content-Security-Policy: script-src 'self'` 로 잠겨 있어, 관리자가 Momento 나
GA4 의 스니펫을 붙여도 브라우저가 실행하지 않았고 그 사실은 개발자 도구에만
나타났습니다. 이제 **설정 → 방문 추적**에서 provider 를 고르면 Server 가 요청마다
nonce 를 만들어 스니펫과 정책에 함께 넣고, 스니펫이 부르는 출처를 정책에 더하며,
그래도 막히는 것은 브라우저의 신고를 받아 화면에 보여 줍니다. `'unsafe-inline'`
은 어떤 경우에도 넣지 않고, **기본값은 꺼짐**입니다.

## 1. 추적 스니펫을 붙일 곳이 없었고, 붙여도 실행되지 않았습니다

콘솔은 하나의 SPA 셸이고 그 HTML 은 Server 바이너리에 박혀 있습니다. 스니펫을
넣으려면 콘솔을 다시 빌드해야 했고, 넣어도 `script-src 'self'` 아래에서는
인라인 `<script>` 도 외부 `<script src>` 도 실행되지 않았습니다. 화면은 멀쩡한데
수집은 하나도 들어오지 않는 상태가 되고, 원인은 브라우저 콘솔의 CSP 경고를 열어
본 사람만 알 수 있었습니다.

이제 **설정 → 방문 추적**(`#/settings/tracking`)이 있습니다.

- **Provider 를 고릅니다.** Momento 가 첫 자리이고 GA4 · GTM · Matomo · 붙여 넣는
  스니펫(`custom`)을 받습니다. 스니펫은 8KB 까지이며, 삽입 위치는 `head` 끝
  (기본)과 `body` 끝 중에서 고릅니다.
- **정책은 공용 DB 에 저장됩니다.** 다른 운영 정책과 같이 `server_metadata` 의
  `tracking_policy` 행에 담겨 모든 Pod 에 즉시 적용되며, 재기동이나 재배포가
  필요 없습니다. 읽기는 쓰지 않으므로 페이지 요청이 DB 에 행을 만들지 않습니다.
- **감사 기록에 남습니다.** 저장은 `tracking.policy.update`, 차단된 출처의
  한 번 허용은 `tracking.allowed_host.add` 로 기록되며 변경 사유를 받습니다.
  동시 변경은 409 로 거절합니다.

## 2. 정책을 느슨하게 하지 않고 스니펫을 실행합니다

추적이 켜져 있으면 Server 는 **페이지 요청마다** 다음을 합니다.

1. 무작위 nonce 를 만들어 스니펫의 모든 `<script>` 태그에 붙이고, 같은 값을
   `script-src 'self' 'nonce-…'` 로 정책에 넣습니다. 값은 요청마다 다르므로 한
   응답을 가로채 다른 페이지에 재사용할 수 없습니다.
2. 스니펫이 부르는 출처를 `script-src` · `connect-src` · `img-src` 에 더합니다.
   provider 가 정해진 것은 알려진 주소를, `custom` 은 붙여 넣은 문자열에서
   `http(s)://…` 출처를 읽습니다. "추가 허용 출처"는 `https://host` 꼴의 http(s)
   출처만 받으므로 `'unsafe-inline'` 같은 키워드는 400 으로 거절됩니다.
3. `report-uri /api/v1/tracking/csp-report` 를 넣어 브라우저가 차단한 요청을
   Server 에 신고하게 합니다.

**추적이 꺼져 있으면 정책 문자열은 이전 버전과 바이트 단위로 같습니다.** 새로
설치한 곳과 켜지 않은 곳에서는 아무것도 달라지지 않습니다. `/api/*` · `/v1/*` ·
`/health/*` · `/mcp` · `/momento/*` 처럼 화면이 아닌 응답에는 스니펫이 붙지 않고
정책도 `default-src 'none'; frame-ancestors 'none'` 으로 더 좁습니다.

## 3. Momento 는 같은 오리진 프록시로 갑니다

Momento 는 사내 자체 호스팅 수집기이므로 데이터가 밖으로 나가지 않는 유일한
선택지입니다. "같은 오리진 프록시 사용"(기본 켜짐)을 두면 브라우저는 Invenqor
Server 의 `/momento/*` 만 부르고 Server 가 그것을 수집기 주소로 넘깁니다. 정책에
외부 출처가 아예 등장하지 않고, Ingress 나 사내 프록시가 브라우저에서 수집기로
가는 길을 열어 주지 않아도 됩니다.

- 방문자의 Invenqor 세션 쿠키와 `Authorization` 헤더는 수집기로 보내지 않고,
  수집기가 돌려주는 `Set-Cookie` 는 버립니다.
- Momento 추적이 켜져 있을 때만 동작하며 그 밖에는 `/momento/*` 가 404 입니다.
- 본문은 256KB 까지, 수집기가 10초 안에 답하지 않으면 502 로 끝냅니다. 정상
  응답은 진단 기록에 남기지 않고 실패한 요청만 남깁니다.

## 4. 막힌 것은 화면에서 보고 한 번에 허용합니다

브라우저의 신고는 인증 없이 `POST /api/v1/tracking/csp-report` 로 들어오고,
Server 는 **출처와 지시어**를 Pod 당 100개까지 메모리에 기억합니다. 같은 차단이
페이지마다 반복되므로 횟수가 아니라 서로 다른 출처가 중요하고, 감사 기록이
아니라 스니펫을 고치는 사람을 위한 작업 보조라 재기동하면 사라집니다. 8KB 를
넘는 본문은 잘라 읽고, 파싱이 안 되면 버리며, 응답은 항상 204 입니다.

**방문 추적** 화면 아래의 "정책이 차단한 출처"가 이 목록입니다. 항목의 **허용
목록에 추가**를 누르면 그 출처가 "추가 허용 출처"에 들어가고, 이미 허용된 출처는
"허용됨"으로 표시됩니다.

새 API 는 `openapi.yaml` 에 추가되었습니다 — `GET·PATCH
/api/v1/admin/settings/tracking`(settings.read / settings.write + CSRF),
`GET·DELETE …/tracking/violations`, `POST …/tracking/allowed-hosts`, 인증 없는
`POST /api/v1/tracking/csp-report`, 그리고 `GET·POST /momento/*`. 관리자
가이드 20장이 설정 항목, nonce 와 CSP 의 동작, 프록시, 차단 목록과 API 를
설명합니다.

## 검증

- **브라우저에서.** 가짜 수집기를 띄우고 headless Chrome 으로 콘솔을 열어,
  nonce 가 붙은 스니펫이 실행되어 `document.title` 을 바꾸고, 수집기에
  `GET /t.js` 가 도착하며, 정책에 없는 출처로의 fetch 는 브라우저가 막고
  `csp-report` 로 신고해 위반 목록에 `connect-src http://127.0.0.1:17173` 으로
  나타나는 것을 보았습니다.
- **프록시.** 실제 Momento 수집기는 빌드 환경에 없어 `httptest` 업스트림으로
  확인했습니다 — 수집기가 방문자의 세션 쿠키를 받지 않고, 수집기의 `Set-Cookie`
  가 브라우저에 닿지 않으며, 추적을 끄면 같은 경로가 404 로 돌아가고 다른 Pod 의
  정책도 원래 문자열로 돌아가는 것을 고정했습니다.
- **정책 문자열.** 추적이 꺼진 상태의 `Content-Security-Policy` 가 이전 버전의
  문자열과 같고, 켜면 `'nonce-…'` 와 `report-uri` 가 들어가며 `'unsafe-inline'`
  은 어떤 입력으로도 들어가지 않는 것을 테스트로 고정했습니다.
- **저장 모드 양쪽에서.** `go test ./...` 가 SQLite fallback 과 실제
  PostgreSQL(`scripts/test-postgres.sh`) 양쪽에서 전 패키지 통과하고,
  `go vet` · `go build` · `gofmt` 가 통과합니다. 콘솔의 vitest 134건이 통과하고,
  `openapi.yaml` 의 redocly lint 는 기존 경고 6건 외에 새 경고가 없습니다.
- 새 화면의 캡처는 가이드에 싣지 않았습니다. 캡처 스크립트·PNG·가이드 세 곳을
  대조하는 테스트가 있어 다음 릴리즈의 과제로 남깁니다.

## 호환성

- 데이터베이스 마이그레이션이 없습니다. 정책은 기존 `server_metadata` 테이블의
  행 하나로 저장되며, 켜기 전에는 그 행도 만들어지지 않습니다.
- **기본값은 꺼짐이고, 꺼져 있으면 동작 변경이 없습니다.** 응답 헤더, 콘솔 HTML,
  API 응답 형식이 이전과 같습니다.
- 켜면 콘솔의 모든 화면 — 로그인 화면을 포함해 — 에 스니펫이 붙습니다. 콘솔은
  화면 구분이 해시 경로라 Server 가 화면을 구분할 수 없어 "관리 화면 제외"
  설정은 두지 않았습니다. 추적 도구가 URL 을 보낼 때 해시 경로가 함께 갈 수
  있으니 개인 식별 값을 보내지 않도록 도구 쪽에서 설정하십시오.
- 추적이 켜져 있으면 `/momento/*` 와 `POST /api/v1/tracking/csp-report` 는
  인증 없이 응답합니다. 전자는 Momento 프록시가 켜져 있을 때만, 후자는 신고를
  기억하고 204 만 돌려줍니다.
- 기존 REST API, Query DSL, MCP 도구는 이번 릴리즈에서 바뀌지 않았습니다.
- Agent(Rust) 동작은 이번 릴리즈에서 바뀌지 않았습니다.
