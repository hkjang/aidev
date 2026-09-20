# 과제서 — 2026-09-20 muni

- 과제: HTML 가져오기도 제목과 같은 첫 H1 을 본문에서 뺀다 (가치 3 / 위험 1 / 작업량 S)

- 왜: muni 의 HTML 내보내기(`internal/httpapi/export.go:359-361` `fullHTMLWithDrawing`)는 `<title>제목</title>` 과 `<body><h1 class="doc-title">제목</h1>…` 을 쓰고 파일 이름도 제목으로 짓는데, HTML 가져오기(`import_html.go:148` 이 `head`·`title` 을 통째로 버리고 `:238-245` 가 `h1` 을 level-1 heading 으로 만듦)는 그 H1 을 본문에 그대로 두어 내보낸 .html 을 다시 가져오면 제목이 두 번(위 제목줄 + 본문 첫 줄) 보이고 검색 텍스트에도 두 번 들어갑니다 — 지난 회차(e8c53e9)가 `.md` 에 고친 것과 정확히 같은 결함이 `.html` 에 남아 있습니다. 이미 있는 `dropLeadingTitle` 을 `.html`/`.htm` 에도 적용하면 왕복이 깨끗해지고 새 판단 로직은 하나도 늘지 않습니다.

- 수용 기준:
  1) `renderHTML(content)` 를 `fullHTMLWithDrawing("회의록", false, …, false)` 로 감싼 바이트를 `회의록.html` 로 `POST /api/v1/workspaces/{id}/documents/import` 하면(폼 title 비움) 문서 제목이 `회의록` 이고 첫 블록이 heading 이 아니라 그 다음 블록(paragraph)이며 `content_text` 에 `회의록` 이 없다.
  2) 같은 파일을 폼 title `9월 회의` 로 올리면 첫 블록이 level-1 heading `회의록` 그대로 남는다(정확히 같을 때만 뺀다는 규칙 유지). 첫 H1 이 제목과 다르거나 H2 이거나 문단 뒤에 오는 HTML 은 content 가 바이트 그대로다.
  3) `.txt`·`.docx`·`.hwpx`·`.hwp`·`.pdf` 는 이 경로가 닿지 않는다(플래그가 `.md/.markdown/.html/.htm` 에서만 참). `importIntoDocument`(문서 안에 끼워 넣기)는 `.html` 도 heading 을 유지한다 — 지난 회차의 `.md` 케이스(`import_markdown_live_test.go:113`)와 같은 형태로 한 줄 확인.
  4) 넘겨받기(`handoff.go:230`)는 여전히 markdown 만 받으므로 동작이 바뀌지 않아야 하며 기존 `TestHandoff*` 가 그대로 통과한다.
  5) 테스트는 손으로 지은 `<h1>` 이 아니라 **실제 내보내기 함수(`fullHTMLWithDrawing`)의 출력**으로 만든 파일을 써서, 내보내기 모양이 바뀌면 함께 깨지게 한다(기억: 픽스처는 리더에 맞춰 짓지 말고 앱이 쓰는 파일 모양을 쓸 것).

- 건드릴 파일:
  - `internal/httpapi/import_attachments.go:196-200, 214-241` `upload.markdown` / `parseUpload` — 필드 이름을 `titleInBody`(또는 비슷한 이름, 주석은 "muni 자신의 Markdown·HTML 내보내기가 제목을 첫 heading 으로 적는다") 로 바꾸고 `case ".html", ".htm":` 에서도 참으로 둔다. `:125` `if parsed.markdown` → 새 이름.
  - `internal/httpapi/handoff.go:230` `if parsed.markdown` → 새 이름(동작 동일).
  - `internal/httpapi/import_markdown.go:27-35` `dropLeadingTitle` 주석 한 줄 — Markdown 만이 아니라 HTML 내보내기도 해당한다고 적기. 함수 본문은 손대지 않는다.
  - `internal/httpapi/import_markdown_live_test.go` — `TestAnExportedMarkdownFileImportsWithoutItsTitleTwice`(`:83`) 옆에 HTML 판 live 테스트 하나(수용 기준 1·2·3). 헬퍼 `importFile`·`firstBlockType`·`importIntoDocument` 가 이미 있다.
  - `internal/httpapi/import_export_test.go:46` 근처 — `fullHTMLWithDrawing` 출력 → `htmlDocument` → `dropLeadingTitle` 단위 왕복 하나(첫 블록만 빠지고 나머지 블록 JSON 이 원본 렌더 결과와 같음). 기존 `TestMarkdownRoundTripDropsTheTitleHeading` 을 본뜬다.
  - `docs/USER_GUIDE.md:183` — "Markdown 파일의 첫 줄이 …" 문장에 HTML 도 같다고 한 구절 덧붙임.

- 검증 명령:
  - `gofmt -l .` (출력 없어야 함), `go vet ./...`
  - `go test ./internal/httpapi -run 'Title|HTMLExportImport|Handoff' -count=1` 뒤 `go test ./...`
  - live 테스트는 `MUNI_TEST_DSN` 없으면 조용히 skip 되므로 반드시 postgres 를 띄워 돌릴 것: `docker run -d --name muni-pg -e POSTGRES_PASSWORD=muni -e POSTGRES_DB=muni -p 5432:5432 postgres:16-alpine` 뒤 `MUNI_TEST_DSN='postgres://postgres:muni@127.0.0.1:5432/muni?sslmode=disable' go test ./internal/httpapi -count=1 -v -run 'ExportedMarkdownFile|ExportedHTMLFile|Handoff'` 에서 SKIP 0 을 확인(지난 회차는 8080 충돌로 컨테이너 네트워크 네임스페이스에 붙여 띄웠음 — 테스트는 포트를 열지 않으므로 DSN 만 맞으면 됨).
  - 고치기 전에 새 테스트가 "첫 블록 = heading" 으로 실패하는 것을 먼저 보고 나서 고칠 것.
  - `scripts/check-webui-placeholder.sh` (프런트는 손대지 않지만 커밋 전 습관).

- 위험과 피할 것:
  - `<title>` 을 embeddedTitle 로 읽어 올리는 것은 **이번에 하지 말 것** — 저장한 웹 페이지의 `<title>` 은 "글 - 사이트명" 꼴이라 지금의 파일 이름 제목보다 나쁠 수 있고, 별도 판단이 필요한 범위 확장이다(차선 아이디어로 남김). 이번 과제는 "이미 확정된 제목과 정확히 같은 첫 H1 을 뺀다" 는 기존 규칙을 확장자 하나 더에 적용하는 것뿐.
  - `class="doc-title"` 을 단서로 쓰는 별도 판단을 새로 만들지 말 것 — 같은 값을 읽는 경로(md/html)가 규칙 하나(`dropLeadingTitle`)를 공유해야 하고, 운영자 지침("같은 값을 읽는 경로가 여럿이면 모두 같게")에 맞는다.
  - `importIntoDocument`(끼워 넣기)·`renderHTML`·`fullHTMLWithDrawing`·PDF 가져오기(`pdfImport` 가 자체 embeddedTitle 을 냄, 본문 첫 줄의 제목은 별개 문제)는 손대지 않는다.
  - `internal/httpapi/notify.go`·mail·`admin_overview.go` 근처는 이번 회차에 건드리지 말 것 — 메일 캠페인 커밋 `7afd103`(브랜치 `auto/2026-09-16-0842`, MAIL-STANDARD) 이 **main 에 합쳐져 있지 않다**(main 의 notify.go 는 `all.SMTP` 옛 형태, 019 마이그레이션·`mail_deliveries` 없음). 병합 여부/반려 여부는 미확인이라 그 영역의 변경은 충돌이나 "반려된 접근 반복" 이 될 수 있다.
  - 보호 경로(auth/migrations/workflows) 는 닿지 않는다.
  - `applyHTMLAlignment` 이 heading 에 `textAlign` attr 을 붙일 수 있는데 `dropLeadingTitle` 은 type·level·PlainText 만 보므로 상관없음(코드로 확인, `import_html.go:241`).

- 차선 후보: Markdown·HTML 가져오기 live 테스트를 `.txt` 로 넓혀 "다른 확장자는 바이트 하나 안 바뀜" 을 못 박기 (2/1/S) — 1순위가 어떤 이유로 성립하지 않으면(예: HTML 가져오기가 첫 블록 앞에 빈 블록을 만들어 규칙이 안 닿는 경우 — 미확인) 그 원인을 테스트로 고정하고 `.txt` 경로 불변을 함께 증명한다.
