# ptium 프로필 (2026-10-03)
- 목적: 한국어 브리프와 기존 문서를 슬라이드 덱으로 만들어 편집·공유·PPTX/PDF 내보내기까지 하는 자체 호스팅 서비스.
- 스택: Go 1.24·net/http, PostgreSQL(pgx/v5), 자체 PPTX/SVG/PDF 처리; React 19·TypeScript 7·Vite 8·Vitest 3.
- 구조:
  - server/cmd/ptium: 진입점; internal/httpapi: REST 배선; internal/mcp: MCP; internal/auth: 인증.
  - internal/db/migrations.go: 기동 마이그레이션·설정 시드; settings: 설정 검증; store: 영속화.
  - internal/docs: Read(docs.go:70~85 확장자 분기)가 CSV/TSV/XLSX/DOCX/PDF/MD/TXT 를 덱 소스로 바꾼다.
    최근 12회차가 거의 전부 여기를 고쳤다. `.md`·`.markdown`·`.txt` 는 모두 readMarkdown 으로 간다(docs.go:81).
    - docs.go: titleOf·citation·escapeField(110)/escapeNote(120)/escapeLine(125, 선두 `#\-*>!@` 를 `\` 로 지킴)·
      상수(maximumSlides=30, maximumRows=8, maximumColumns=5, maximumTableSlides=4, maximumPoints=5).
    - workbook.go: XLSX ZIP·시트·공유문자열. cellformat.go: 날짜·시각·퍼센트 서식.
      tables.go: 격자→슬라이드(readSeparated/placement/writeSheet/writeBody/rangeOf/columnLetter/trimmed/
      allNumeric/amountOf/bareFigure).
    - writer.go(271행): deckWriter — docx·pdf·markdown 공용(cover:52/slide:89/point:102/note/table/flush:149/
      writeSlide:191/continued:132/splitTables:232/document:256).
    - prose.go(490행): docx(readWordDocument/blocksIn/tableRows/headingLevel/picturesIn) + markdown
      (readMarkdown:205, handle 클로저:218, fenceOf:338, closesFence, lineAfter, atxHeading:432,
      underlinesHeading:479, isRule:483). pdf.go: PDF. listmarker.go: listMarkers/needsSpaceAfter/
      withoutListMarker:32/isListLine:51.
  - internal/deck: ParseSource·Compile(compile.go, source.go — source.go 의 `#` 제목은 덱 DSL 이고
    docs 의 마크다운 리더와 **다른 계약**이다); pptx: 템플릿·렌더링·내보내기; golden: 출력 회귀.
  - internal/generation: 생성과 워커; mail/analytics/handoff: 알림·추적·문서 전달.
  - web/src/pages·components: 편집기·관리 화면; api/openapi.yaml: API 계약; docs: 가이드·문법·설계·릴리즈 노트.
- 빌드·테스트: make test = server 에서 go test -race ./... + go vet ./... + web typecheck/build.
  - docs 집중 검증: `cd server && go test -count=1 ./internal/docs` — 2026-10-03 실측 ok(단일 테스트 ~0.1s, 전체 ~3s).
  - 웹 단위: `cd web && npm test`(node_modules 없으면 npm ci). 웹 변경이 없으면 웹 단계를 건너뛴다(최근 회차 관례).
  - CI: Go race/vet, Node 22 npm ci/typecheck/build/audit --audit-level=high, Docker build. Vitest 단계 없음.
- 관례: 영어 문장형 커밋·테스트 이름("A long table continues…"), 한국어 사용자 가이드와 영문 문법 문서.
  - 주석이 유난히 길다 — 왜 그렇게 읽는지를 산문으로 적는다(prose.go:251~257, 413~431, 454~478 이 표본).
    새 코드도 그 밀도를 맞출 것. **거절한 선택지의 이유도 주석에 남긴다**(474~478 이 `-` 를 setext 로 안 읽는 이유).
  - 사용자에게 보이는 경고·오류는 한국어이고, 못 가져온 것은 조용히 버리지 않고 **개수를 말한다**
    (prose.go:94 그림, prose.go:327 코드 블록, tables.go 열·행, writer.go:267 슬라이드).
    단, 내용이 없는 구두점(표의 `|---|` 규칙 줄 등)은 세지 않는다 — 경고는 "잃은 내용" 에만 붙인다.
  - 설정은 defaultSettings→settings→API 검증→OpenAPI→관리 화면을 맞춘다. 마이그레이션은 Go 로 기동 시 실행.
  - **현재 VERSION 1.69.54, base main@e10d356("Release 1.69.54")**. 정찰·구현은 버전·릴리즈 노트·배포 매니페스트를 바꾸지 않는다.
  - 버전은 VERSION·api/openapi.yaml·deploy/kubernetes.yaml·docs/offline-deployment.md 네 곳에 박혀 있고
    server/internal/config/stamped_test.go 가 go test 단계에서 어긋남을 잡는다.
  - 사용자 가이드의 확장자 표는 docs/USER_GUIDE.md:276 — 마크다운 규칙을 바꾸면 여기 한 칸이 따라간다.
    이 칸은 이미 세 규칙이 쌓여 300자가 넘는다(별도 아이디어로 쪼갤 것).
- 위험 구역: auth·httpapi 인증, db/migrations.go, 단일 사용 handoff claims, SMTP 비밀, 감사 로그 스크럽.
  - scripts/release.sh·scripts/build-offline.sh: 릴리즈 경로. 고치면 릴리즈까지 통과를 확인하거나 못 한 것을 적을 것.
  - writer.go 의 deckWriter 는 docx·pdf·markdown 세 리더가 함께 쓴다. point()는 maximumPoints 에서 (계속)
    슬라이드를 만들어 되돌릴 수 없다 — 리더는 **쓰기 전에·앞보기로** 판단할 것(handle(line, next) bool 이 그 구조다).
  - internal/deck/source.go 는 덱 DSL 이다. docs 리더의 마크다운 규칙을 거기로 옮기지 말 것(golden 회귀가 그 위에 있다).
- 자주 깨지는 곳: docs.amountOf/allNumeric(분류기)와 deck 의 parseNumber/parseBareNumber/chartFields(렌더러)는
  서로 다른 계약이다. 통합·구분자 확장 금지(2026-09-09 에 두 번 반려). 고칠 때는 분류기를 좁히는 방향으로만.
  - 음수 파싱만으로 막대 시각 의미를 고쳤다고 주장하지 말 것 — pptx/blocks.go 는 math.Abs 로 높이를 잡는다.
  - korean 패키지는 소스 문자열 검사로 조사를 잡는다(`%s는` 처럼 값 뒤에 조사를 붙이면 실패).
  - 웹 errors.ts 번역 누락은 codecover 에서 실패 가능.
- XLSX 현재 상태: 숨김 행·열 제거 + 경고(3c3493f); inlineStr rich text 보존(8739ec8); columnOf 는 XFD 에서 끊고
  columnLetter 는 26진 자리올림(80ebe4c); workbookParts 로 필요한 파트만 하나씩 해제(22034a7, 누적 예산은 없음);
  시트 모양은 body+carried 로 한 번만 판정(e68ce08); 이어지는 장의 !source 는 그 장의 행 범위(fb538af);
  이어지는 장 제목은 continued() 를 거쳐 '(계속)' 이 두 번 붙지 않는다(6f41ea4).
- 마크다운 현재 상태(2026-10-03 실행 확인): readMarkdown 은 줄 단위 스캐너다 — ATX `#` 제목, `|` 표,
  listmarker 불릿, 그 밖은 문장.
  - 펜스 코드 블록(``` / ~~~)은 건너뛰고 블록·줄 수를 경고한다(58ec05a). 닫히지 않은 펜스는 held 를 재생한다.
  - setext h1(`===`)은 제목으로 읽는다(80b1be5). **h2 밑줄 `---` 은 아직 아니다**(prose.go:474 에 이유).
  - `#` 뒤 공백 또는 줄 끝을 요구하고 1~6개만 제목이다(70f65fe) — `#출시`·`#1 우선순위`·`#######` 는 문장.
  - 닫는 `#` 시퀀스는 뗀다(5bad44a, v1.69.54): `## 분기 요약 ##` → `분기 요약`. `# C# 도입`·`# 제목#` 유지.
  - **수평선은 아직 요점으로 남는다**: `---`→`- \---`, `***`→`- \***`, `___`→`- ___`, `- - -`→`- \- -`,
    `* * *`→`- \* *`. `--`·`-5% 감소` 는 문장으로 남는 것이 맞다. YAML front matter 문서는 파일명 슬라이드에
    `\---`/`title: 보고서`/`\---` 세 줄이 쌓인다. **이번 회차 과제**(front matter 건너뛰기는 그 다음).
  - **인용 `> 문장` 은 `- \> 문장`** 으로 남아 `>` 가 사용자에게 보인다. 차선 후보.
  - 들여쓰기 4칸 코드 블록, 인라인 코드 span, 표 셀의 `\|`, 메타 정보 붙은 펜스(```js {1,3})는 아직 안 다룬다.
- 검증 함정: PTIUM_TEST_DSN 없으면 DB 테스트 Skip. DB 연결 검증은 최근 회차에서 계속 미실시.
  - scripts/e2e/api.py:call 은 headers={} 를 개발 신원으로 바꾼다. 무인증 검사는 기존 NOBODY 관례 확인.
  - 8099 충돌·컨테이너 loopback 주의; handoff e2e 는 PTIUM_E2E_PEER_HOST 로 상대 주소 지정.
  - docs/roadmap-v2.md 는 v0.44 계획이라 현재 구현과 다르다. 코드 우선. CLAUDE.md·AGENTS.md 는 없고 TODO/FIXME 도 없다.
  - 이 샌드박스는 `python3 -c`·히어독·복합 명령(`A && B`)을 막고, `cd <repo> && git …` 는 승인을 요구한다.
    임시 파일은 Write 로, 검색은 Grep 도구로. `cd server && go test …` 는 통과한다.
