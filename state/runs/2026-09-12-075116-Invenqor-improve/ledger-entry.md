## 2026-09-12
- 선택: 캠페인 tracking-2026-09 — 관리자가 화면에서 방문 추적 스크립트를 붙이는 체계(nonce CSP · Momento 같은 오리진 프록시 · 차단 출처 기록) (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 콘솔은 `script-src 'self'` 로 잠겨 있어 스니펫을 붙여도 브라우저가 조용히
  버렸다. `server/internal/tracking` 패키지(설정·검증·스니펫 렌더링·출처 추출·위반
  기록)와 `httpapi/tracking.go`(DB `server_metadata.tracking_policy` 저장, GET/PATCH
  `/api/v1/admin/settings/tracking`, 위반 목록·비우기·한 번에 허용, 무인증
  `POST /api/v1/tracking/csp-report`, `/momento/*` 리버스 프록시)를 만들고
  `securityHeaders` 가 페이지 요청마다 nonce 를 만들어 정책과 스니펫에 같이 넣게
  했다. `webui.Decorated` 가 임베드된 index.html 에 스니펫을 삽입한다(`/index.html`
  직접 요청도 같은 셸). provider 는 momento·ga4·gtm·matomo·custom 이고 Momento 는
  기본으로 같은 오리진 프록시(세션 쿠키·Authorization 제거, Set-Cookie 폐기, 256KB,
  10초)를 써 외부 출처가 정책에 안 들어간다. `'unsafe-inline'` 은 어떤 입력으로도
  나오지 않고(allowed_hosts 는 http(s) 출처만 통과), 꺼져 있으면 정책 문자열이
  이전과 바이트 단위로 같으며 DB 에 아무것도 쓰지 않는다. 비화면 경로
  (`/api`·`/v1`·`/health`·`/mcp`·`/momento`)는 `default-src 'none'` 으로 좁혔다.
  `include_admin` 은 두지 않았다 — Invenqor 콘솔은 SPA 셸 하나에 해시 경로라 Server
  가 관리 화면을 구분할 수 없고 콘솔 전체가 관리 화면이다(가이드에 이유를 적음).
  콘솔에 설정 → 방문 추적 탭(provider 카드, 항목별 입력, 정책에 더해진 출처, 차단
  목록과 "허용 목록에 추가"·"기록 비우기")을 넣고 `webui/dist` 를 재빌드했다(이
  저장소는 dist 를 추적하고 CI 가 대조한다). openapi.yaml 에 8개 오퍼레이션과
  2개 스키마를 더했고, ADMIN_GUIDE.md 에 20장(설정·nonce/CSP 설명·프록시·차단
  출처·API)을 넣고 PDF 를 다시 구웠다(51쪽, `file://` 없음). 검증:
  `tracking` 단위 테스트 8개(기본 꺼짐, 프록시 시 출처 0, 모든 `<script>` 에 nonce,
  8KB·키워드 거절, 위반 100개 고리·와일드카드 허용 표시)와 httpapi 통합 테스트 4개
  (기본 정책 불변·DB 무기록, 다른 Pod 에서 같은 스니펫과 요청마다 다른 nonce,
  placement=body, 프록시 쿠키 제거, 끄면 원래 정책, 붙여 넣은 스니펫 출처가 세
  지시어에, 잘못된 입력 400, 신고→목록→허용→비우기)를 추가해 `go test ./...` 를
  SQLite 와 실 PostgreSQL(`scripts/test-postgres.sh`) 양쪽에서 전 패키지 통과,
  `go vet`·`go build`·`gofmt`, vitest 134개 통과, `@redocly/cli@2.47.0 lint` 경고
  6개(모두 기존). 표준 검증 항목 "실제로 띄워 수집이 들어오는 것"은 임시 Server 와
  가짜 수집기를 띄우고 headless Chrome 으로 열어 확인했다 — nonce 붙은 스니펫이
  실행돼 `document.title` 을 바꿨고, 수집기에 `GET /t.js` 가 도착했으며, 정책에
  없는 출처로의 fetch 는 브라우저가 막고 `csp-report` 로 신고돼 위반 목록에
  `connect-src http://127.0.0.1:17173` 로 나타났다. 실제 Momento 수집기는 이
  환경에 없어 프록시는 httptest 업스트림으로만 확인했다. 새 화면 캡처는 넣지
  않았다(캡처 스크립트·PNG·가이드 세 곳을 대조하는 테스트가 있어 다음 회차 과제).
  버전 범프·릴리즈 노트는 하지 않았다.
- 보류 아이디어: 설정 → 방문 추적 화면 캡처를 캡처 스크립트에 추가해 가이드 20장에 싣기 (가치 2 / 위험 1 / S) · Query DSL 에 `attributes.<키>` 존재/부재 연산자가 없어 `>= ""` 우회가 필요함 (가치 3 / 위험 2 / M) · MCP `asset_relations` 가 상대 자산의 이름·종류를 주지 않아 edge 마다 `asset_get` 을 한 번 더 부르게 만듦 (가치 3 / 위험 2 / M) · 콘솔 Query DSL 화면이 새 `total`·`offset` 을 쓰지 않아 여전히 첫 페이지만 보여줌 (가치 3 / 위험 2 / M) · 추적 정책을 페이지 요청마다 DB 에서 읽음 — 트래픽이 많은 설치에서는 짧은 TTL 캐시가 필요할 수 있음 (가치 2 / 위험 2 / S)
