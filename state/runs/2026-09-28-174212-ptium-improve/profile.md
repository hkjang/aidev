# ptium 프로필 (2026-09-28)
- 목적: 한국어 브리프와 기존 문서를 슬라이드 덱으로 만들어 편집·공유·PPTX/PDF 내보내기까지 하는 자체 호스팅 서비스.
- 스택: Go 1.24·net/http, PostgreSQL(pgx/v5), 자체 PPTX/SVG/PDF 처리; React 19·TypeScript 7·Vite 8·Vitest 3.
- 구조:
  - server/cmd/ptium: 진입점; internal/httpapi: REST 배선; internal/mcp: MCP; internal/auth: 인증.
  - internal/db/migrations.go: 기동 마이그레이션·설정 시드; settings: 설정 검증; store: 영속화.
  - internal/docs: Read 가 CSV/TSV/XLSX/DOCX/PDF/MD/TXT 를 덱 소스로 바꾼다. 최근 9회차가 모두 여기를 고쳤다.
    - docs.go(85행에 Read 분기): titleOf·citation·escapeField/Note/Line·상수(maximumSlides=30, maximumRows=8,
      maximumColumns=5, maximumTableSlides=4, maximumPoints=5, sheetColumns=16384, sheetCells=1_000_000).
    - workbook.go: XLSX ZIP·시트·공유문자열(readWorkbook/workbookParts/worksheet/gridOf/columnOf/onScreen).
    - cellformat.go: 날짜·시각·퍼센트 서식. tables.go: 격자→슬라이드
      (readSeparated/placement/writeSheet/writeBody/rangeOf/columnLetter/trimmed/allNumeric/amountOf/bareFigure).
    - writer.go(270): deckWriter — docx·pdf·markdown 공용(cover/slide/point/note/table/flush/continued/splitTables).
    - prose.go(365): docx(readWordDocument/blocksIn/tableRows/headingLevel/picturesIn) +
      markdown(readMarkdown 205행, fenceOf, closesFence, isRule). pdf.go: PDF. listmarker.go(55):
      listMarkers/withoutListMarker/isListLine. textbytes.go: UTF-8/CP949. wordtext.go: 런 텍스트.
  - internal/deck: ParseSource·Compile(compile.go 1580행, source.go 813행); pptx: 템플릿·렌더링·내보내기; golden: 출력 회귀.
  - internal/generation: 생성과 워커; mail/analytics/handoff: 알림·추적·문서 전달.
  - web/src/pages·components: 편집기·관리 화면; auth: PKCE/silent SSO; api: 클라이언트.
  - api/openapi.yaml: API 계약; docs: 가이드·문법·설계·릴리즈 노트; scripts/e2e: 실서버 검증.
- 빌드·테스트: make test = server 에서 go test -race ./... + go vet ./... + web typecheck/build.
  - docs 집중 검증: `cd server && go test ./internal/docs`(3~6초). make build 는 Go 바이너리와 웹 번들.
  - 웹 단위: `cd web && npm test`. node_modules 없으면 npm ci 필요. 웹 변경이 없으면 최근 회차들은 웹 단계를 건너뛴다.
  - CI: Go race/vet, Node 22 npm ci/typecheck/build/audit --audit-level=high, Docker build. Vitest 단계 없음.
  - 실서버/브라우저 e2e·Docker 빌드는 서비스·환경 준비로 오래 걸릴 수 있음.
- 관례: 영어 문장형 커밋·테스트 이름("A long table continues…"), 한국어 사용자 가이드와 영문 문법 문서.
  - 주석이 유난히 길다 — 왜 그렇게 읽는지를 산문으로 적는다. 새 코드도 그 밀도를 맞출 것(prose.go:240~305 가 본보기).
  - 사용자에게 보이는 경고·오류 문구는 한국어이고, 못 가져온 것은 조용히 버리지 않고 개수를 말한다
    (prose.go:94 그림 경고, prose.go:300 코드 블록 경고, tables.go 열·행 경고, writer.go 슬라이드 경고).
  - 설정은 defaultSettings→settings→API 검증→OpenAPI→관리 화면을 맞춘다. 마이그레이션은 Go 로 기동 시 실행.
  - 현재 VERSION 1.69.50, base main@de8b1e3("Sign out of Keycloak too…"). 최신 릴리즈 태그는 v1.69.49(64d7a03).
    정찰·구현에서 버전·릴리즈 노트·배포 매니페스트를 바꾸지 않는다.
  - 버전은 VERSION·api/openapi.yaml·deploy/kubernetes.yaml·docs/offline-deployment.md 네 곳에 박혀 있고
    server/internal/config/stamped_test.go(050f946)가 go test 단계에서 어긋남을 잡는다.
- 위험 구역: auth·httpapi 인증, db/migrations.go, 단일 사용 handoff claims, SMTP 비밀, 감사 로그 스크럽.
  - scripts/release.sh·scripts/build-offline.sh: 릴리즈 경로. 고치면 릴리즈까지 통과를 확인하거나 못 한 것을 적을 것.
  - writer.go 의 deckWriter 는 docx·pdf·markdown 세 리더가 함께 쓴다. 한 리더 때문에 여기를 고치면 셋이 같이 움직인다.
    point() 는 maximumPoints 에서 (계속) 슬라이드를 만들므로 이미 쓴 점을 되돌릴 수 없다 — 리더는 앞보기로 판단할 것.
- 자주 깨지는 곳: docs.amountOf/allNumeric(분류기)와 deck 의 parseNumber/parseBareNumber/chartFields(렌더러)는
  서로 다른 계약이다. 통합·구분자 확장 금지(2026-09-09 에 두 번 반려). 고칠 때는 분류기를 좁히는 방향으로만.
  - 음수 파싱만으로 막대 시각 의미를 고쳤다고 주장하지 말 것 — pptx/blocks.go 는 math.Abs 로 높이를 잡는다.
  - korean 패키지는 소스 문자열 검사로 조사를 잡는다(%s는 처럼 값 뒤에 조사를 붙이면 실패).
  - 웹 errors.ts 번역 누락은 codecover 에서 실패 가능.
- XLSX 현재 상태: 숨김 행·열은 onScreen 에서 제거하고 placement 로 출처 전달 + 경고 한 줄, 생략 행의 실제 r 반영(3c3493f);
  inlineStr rich text 보존(8739ec8); columnOf 는 XFD=16384 에서 끊고 columnLetter 는 26진 자리올림(80ebe4c);
  readWorkbook 은 workbookParts 로 필요한 파트만 하나씩 해제(22034a7, 누적 예산은 없음);
  시트 모양(::columns/::table)은 body+carried 로 한 번만 판정(e68ce08); 이어지는 장의 !source 는 그 장의 행 범위(fb538af).
- 마크다운 현재 상태: readMarkdown 은 줄 단위 스캐너다 — `#` 제목, `|` 표, listmarker 불릿, 그 밖은 문장.
  - 펜스 코드 블록(``` / ~~~)은 handle 클로저 + 펜스 상태로 건너뛰고 개수·줄 수를 경고한다(58ec05a, v1.69.49).
    닫히지 않은 펜스는 held 를 handle 로 재생해 아무것도 삼키지 않는다.
  - **setext 제목(=== / ---)은 아직 다루지 않는다** — 2026-09-28 정찰이 실행으로 확인: `분기 요약\n=========` 는
    제목이 사라져 덱 이름이 파일 이름이 되고 `- =========` 불릿이 남는다(`---` 은 escapeLine 때문에 `- \---------`).
    들여쓰기 4칸 코드 블록과 인라인 코드 span(`…`)도 다루지 않는다.
- 검증 함정: PTIUM_TEST_DSN 없으면 DB 테스트 Skip. DB 연결 검증은 최근 회차에서 계속 미실시.
  - scripts/e2e/api.py:call 은 headers={} 를 개발 신원으로 바꾼다. 무인증 검사는 기존 NOBODY 관례 확인.
  - 8099 충돌·컨테이너 loopback 주의; handoff e2e 는 PTIUM_E2E_PEER_HOST 로 상대 주소 지정.
  - docs/roadmap-v2.md 는 v0.44 계획이라 현재 구현과 다르다. 코드 우선. CLAUDE.md·AGENTS.md 는 없고 TODO/FIXME 도 없다.
  - 이 샌드박스는 `python3 -c`·bash 히어독·복합 명령(`A && B`, `A; B` 중 승인 안 된 것)을 막는다. 임시 파일은 Write 로,
    검색은 Grep 도구로, git 은 `git -C <path> …` 를 단독 명령으로 쓸 것. `cd server && go test …` 는 통과한다.
