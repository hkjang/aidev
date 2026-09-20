# 과제서 — 2026-09-20-162411-relio-improve

- 과제: 감사 로그 화면이 로그인 조사를 한 화면에서 끝내게 하기 — 시각에 시·분·초, 채널 필터에 LOGIN·SSO, 상세 모달에 시각·User-Agent (가치 3 / 위험 1 / 작업량 S)
- 왜: 직전 회차(c191148)가 LOGIN(`bootstrap`)·SSO(`silent`) 이벤트의 metadata 를 화면에 냈지만, 감사 화면의 채널 `<select>` 는 `['WEB','API','MCP','ADMIN']` 뿐이라 정작 그 두 채널로 좁혀 볼 수 없고(API 는 `channel=LOGIN` 을 이미 받음 — `admin_operations.go:86,92`), 목록의 "시각" 열은 `date()`(`web/src/api.ts:27`, 연·월·일만) 라 같은 날 로그인 여러 건의 순서·시각을 알 수 없으며, 상세 모달은 응답에 있는 `occurredAt`·`userAgent` 를 그리지 않습니다(`AdminPages.tsx:575`). 세 가지 모두 프런트 표시 문제이고 Go 응답은 이미 값을 다 싣고 있습니다.
- 수용 기준:
  1) `web/src/api.ts` 에 `dateTime(value?: string)` 헬퍼 추가 — `Intl.DateTimeFormat('ko-KR', {year:'numeric',month:'short',day:'numeric',hour:'2-digit',minute:'2-digit',second:'2-digit',hour12:false})`, 빈 값이면 `date()` 와 같이 `'—'`. 기존 `date()` 는 그대로(다른 페이지가 씀).
  2) 감사 목록(`AdminPages.tsx:574`) 의 시각 셀이 `dateTime(x.occurredAt)` 로 초 단위까지 보임. 상세 모달(`:575`) `audit-meta` 그리드에 `<div><span>시각</span><b>{dateTime(selected.occurredAt)}</b></div>` 와 `<div><span>User-Agent</span><code>{selected.userAgent||'—'}</code></div>` 두 칸 추가(CSS `.audit-meta` 는 2열 grid + `overflow-wrap:anywhere` 라 긴 UA 도 넘치지 않음 — `styles.css:67`, 추가 CSS 불필요).
  3) 채널 `<select>`(`AdminPages.tsx:573`) 의 목록이 `['WEB','API','MCP','ADMIN','LOGIN','SSO']` — 서버 producer 가 실제로 쓰는 채널 전부(`Channel: "LOGIN"` public.go:76/82/94, `"SSO"` oidc/service.go:575/720, `"API"/"WEB"` server.go:653, `"ADMIN"` 20곳, `"MCP"` mcp/server.go:715). 검색 placeholder 는 그대로.
  4) `docs/ADMIN_GUIDE.md:319` 의 "Web·REST·MCP·Admin·Login·Key 채널" 을 실제 채널값에 맞게 고침 — "WEB·API·MCP·ADMIN·LOGIN·SSO 채널(개인 키 작업은 WEB 채널의 `KEY_*` 동작)" 정도. PDF 는 재생성하지 않음(보류 항목과 함께).
  5) 테스트: `web/test/dateTime.test.ts`(node --test, `../src/api.ts` 에서 import) 로 `dateTime('2026-09-20T07:05:09Z')` 가 시·분·초를 담고(정규식 `/\d{2}:\d{2}:\d{2}/`) 같은 입력의 `date()` 는 담지 않으며, `dateTime(undefined)==='—'` 임을 증명. 시간대는 테스트에서 `process.env.TZ='Asia/Seoul'` 를 import 전에 고정하거나 시·분·초 존재만 검사(값 자체를 고정하지 말 것).
- 건드릴 파일:
  - `web/src/api.ts:27` — `dateTime` export 추가(한 줄, 기존 스타일대로).
  - `web/src/pages/AdminPages.tsx:2` import 에 `dateTime` 추가, `:573` select 목록, `:574` 시각 셀, `:575` 모달 meta 두 칸. 이 파일은 한 줄 압축 스타일 — 포매터 돌리지 말고 해당 줄만 편집.
  - `web/test/dateTime.test.ts` — 신규.
  - `docs/ADMIN_GUIDE.md:319` — 한 문장.
- 검증 명령:
  - `cd web && npm ci && npm run typecheck && npm run build && npm test`(기존 8 + 신규 pass)
  - web build 뒤 `git status --porcelain` 이 anchor(`internal/webui/dist/README`) 포함 깨끗한지
  - `go build ./... && go test -race ./internal/server/ && go vet ./...`(Go 무변경 확인용; `openapi.go` 도 무변경 — `channel` 파라미터는 이미 문서화돼 계약 테스트 영향 없음, 미확인이면 `go test ./internal/api/` 로 확인)
  - `./scripts/check-static-assets.sh && ./scripts/check-env-contract.sh`
  - 선택(예산 있으면): 이전 회차처럼 정적 바이너리 + throwaway PostgreSQL 17 로 띄워 로그인 뒤 `GET /api/v1/admin/audit?channel=LOGIN` 이 LOGIN 행만 돌려주는지 curl 로 확인 — UI 는 이 값을 그대로 보냄.
- 위험과 피할 것:
  - Go·SQL·`openapi.go`·migrations·`internal/auth`·`public.go` 는 건드리지 않음(값은 이미 응답에 있음). `admin_operations.go` 의 SELECT/`auditItem` 도 무변경.
  - `date()` 시그니처·출력을 바꾸지 말 것 — 계약·고객 등 다른 페이지가 씀. 새 헬퍼만 추가.
  - 채널 목록에 `PHONE`·`PIGEON` 같은 값을 넣지 말 것 — grep 에 잡히지만 audit 채널이 아니라 VOC 접수 채널(미확인: internal/voice 쪽으로 추정). 감사 채널은 위 6개.
  - `web/test` 에서 `../src/api.ts` 를 import 하면 모듈 최상위가 Node 에서 평가됨 — `api.ts` 최상위에는 `window`/`document` 참조가 없어 보이나(1~30행 확인, 나머지 미확인) import 가 실패하면 `dateTime` 을 `web/src/format.ts` 같은 새 파일로 빼고 `api.ts` 가 re-export 하는 방식으로 우회.
  - 운영자 규칙: 소스 문자열 검사(select 옵션 문자열을 grep 하는 식)를 테스트로 삼지 말 것. 채널 목록은 검증 명령의 typecheck·build 와 (가능하면) 실제 API 호출로 증명.
- 차선 후보: 감사 목록 검색 `q` 가 `metadata::text ILIKE` 도 훑게 하기(`admin_operations.go:95` 한 줄 + `auditItem` 테스트 옆에 SQL 문자열이 아닌 실제 DB 로 확인 필요 — DB 통합 테스트가 없어 throwaway PostgreSQL 로 증명해야 함) (가치 2 / 위험 1 / S).
