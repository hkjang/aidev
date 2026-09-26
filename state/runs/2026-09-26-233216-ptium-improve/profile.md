# ptium 프로필 (2026-09-26)
- 목적: 한국어 브리프와 기존 문서를 슬라이드 덱으로 만들어 편집·공유·PPTX/PDF 내보내기까지 하는 자체 호스팅 서비스.
- 스택: Go 1.24·net/http, PostgreSQL(pgx/v5), 자체 PPTX/SVG/PDF 처리; React 19·TypeScript 7·Vite 8·Vitest 3.
- 구조:
  - server/cmd/ptium: 진입점; internal/httpapi: REST 배선; internal/mcp: MCP; internal/auth: 인증.
  - internal/db/migrations.go: 기동 마이그레이션·설정 시드; settings: 설정 검증; store: 영속화.
  - internal/docs: Read 를 통해 CSV/XLSX/DOCX/PDF/텍스트를 덱 소스로 변환. 최근 7회차가 모두 여기를 고쳤다.
    - workbook.go(~530): XLSX ZIP·시트·공유문자열(readWorkbook/workbookParts/worksheet/gridOf/columnOf/onScreen).
    - cellformat.go(381): 날짜·시각·퍼센트 서식. tables.go(~430): 격자→슬라이드(writeSheet/writeBody/trimmed/rangeOf/columnLetter/allNumeric).
    - writer.go: docx/pdf 공용 덱 작성기(continued() 포함); prose.go·pdf.go: 문장 문서. docs.go: 상수(maximumRows=8, maximumColumns=5, maximumTableSlides=4)·citation.
  - internal/deck: ParseSource·Compile(compile.go 1580행, source.go 813행); pptx: 템플릿·렌더링·내보내기; golden: 출력 회귀.
  - internal/generation: 생성과 워커; mail/analytics/handoff: 알림·추적·문서 전달.
  - web/src/pages·components: 편집기·관리 화면; auth: PKCE/silent SSO; api: 클라이언트.
  - api/openapi.yaml: API 계약; docs: 가이드·문법·설계·릴리즈 노트; scripts/e2e: 실서버 검증.
- 빌드·테스트: make test = server 에서 go test -race ./... + go vet ./... + web typecheck/build.
  - docs 집중 검증: `cd server && go test ./internal/docs`(약 3초). make build 는 Go 바이너리와 웹 번들.
  - 웹 단위: `cd web && npm test`. node_modules 없으면 npm ci 필요. 웹 변경이 없으면 최근 회차들은 웹 단계를 건너뛴다.
  - CI: Go race/vet, Node 22 npm ci/typecheck/build/audit --audit-level=high, Docker build. Vitest 단계 없음.
  - 실서버/브라우저 e2e·Docker 빌드는 서비스·환경 준비로 오래 걸릴 수 있음.
- 관례: 영어 문장형 커밋·테스트 이름("A long table continues…"), 한국어 사용자 가이드와 영문 문법 문서.
  - 주석이 유난히 길다 — 왜 그렇게 읽는지를 산문으로 적는다. 새 코드도 그 밀도를 맞출 것.
  - 설정은 defaultSettings→settings→API 검증→OpenAPI→관리 화면을 맞춘다. 마이그레이션은 Go 로 기동 시 실행.
  - 현재 VERSION 1.69.47, HEAD 6417754(Release 1.69.47). 정찰·구현에서 버전·릴리즈 노트를 바꾸지 않는다.
  - 버전은 VERSION·api/openapi.yaml·deploy/kubernetes.yaml·docs/offline-deployment.md 네 곳에 박혀 있고
    server/internal/config/stamped_test.go(050f946)가 go test 단계에서 어긋남을 잡는다.
- 위험 구역: auth·httpapi 인증, db/migrations.go, 단일 사용 handoff claims, SMTP 비밀, 감사 로그 스크럽.
  - scripts/release.sh·scripts/build-offline.sh: 릴리즈 경로. 고치면 릴리즈까지 통과를 확인하거나 못 한 것을 적을 것.
- 자주 깨지는 곳: docs.amountOf/allNumeric(분류기)와 deck 의 parseNumber/parseBareNumber/chartFields(렌더러)는
  서로 다른 계약이다. 통합·구분자 확장 금지(2026-09-09 에 두 번 반려). 고칠 때는 분류기를 좁히는 방향으로만.
  - 음수 파싱만으로 막대 시각 의미를 고쳤다고 주장하지 말 것 — pptx/blocks.go 는 math.Abs 로 높이를 잡는다.
  - korean 패키지는 소스 문자열 검사로 조사를 잡는다(%s는 처럼 값 뒤에 조사를 붙이면 실패).
  - 웹 errors.ts 번역 누락은 codecover 에서 실패 가능.
- XLSX 현재 상태(최근 회차 누적):
  - 숨김 행·열은 gridOf 뒤 onScreen 에서 제거하고 placement 로 출처를 전달, 경고 한 줄. 생략 행의 실제 r 도 반영(3c3493f).
  - inlineStr rich text 는 InlineRuns 를 Join 해 보존(8739ec8). columnOf 는 XFD=16384 에서 끊고 columnLetter 는 26진 자리올림(80ebe4c).
  - readWorkbook 은 workbookParts 로 필요한 파트만 하나씩 해제(22034a7). 누적 해제량 예산은 아직 없음.
  - 시트 모양(::columns/::table)은 body+carried 로 한 번만 판정하고 writeBody 가 본문을 쓴다(e68ce08).
- 알려진 미해결(이번 정찰에서 writeSheet 직접 호출로 확인): rangeOf 가 범위를 늘 A1 에서 시작해, 이어지는
  슬라이드가 앞장의 행까지 삼킨 인용을 쓴다(20행 시트 → A1:B9 / A1:B17 / A1:B21). 이번 회차 과제.
- 기록 불일치: 과거 노트의 MCP OAuth 캠페인 구현은 이 HEAD 에 없다. MCP OAuth 를 구현된 기능으로 가정하지 않는다. silent SSO 는 존재.
- 검증 함정: PTIUM_TEST_DSN 없으면 DB 테스트 Skip. DB 연결 검증은 최근 회차에서 계속 미실시.
  - scripts/e2e/api.py:call 은 headers={} 를 개발 신원으로 바꾼다. 무인증 검사는 기존 NOBODY 관례 확인.
  - 8099 충돌·컨테이너 loopback 주의; handoff e2e 는 PTIUM_E2E_PEER_HOST 로 상대 주소 지정.
  - docs/roadmap-v2.md 는 v0.44 계획이라 현재 구현과 다르다. 코드 우선.
  - CLAUDE.md·AGENTS.md 는 저장소에 없다. TODO/FIXME 검색 결과 없음.
  - 이 샌드박스는 `python3 -c` 실행과 bash 히어독(따옴표 포함)을 막는다 — 임시 파일은 Write 도구로 만들 것.
  - 요청된 pmo:estimating-and-contingency 는 이번에는 목록에 있었으나 과제 규모가 S 한 건이라 적용하지 않았고,
    technology:implementation-planning·solution-exploration 2종은 적용함.
