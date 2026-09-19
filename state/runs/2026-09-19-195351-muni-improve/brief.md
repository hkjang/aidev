# 과제서 — 2026-09-19 muni

- 과제: 마크다운 가져오기·넘겨받기에서 첫 H1 이 문서 제목과 같으면 본문에서 덜어내기 (가치 3 / 위험 1 / 작업량 S)

- 왜: muni 의 마크다운 내보내기(`internal/httpapi/render.go:500 renderMarkdown`)는 제목을 `# 제목` 으로 본문 첫 줄에 적고, 파일 이름도 제목으로 짓는데(`export.go:88 file.filename = safeFilename(title)`), 마크다운 가져오기(`import_attachments.go:parseUpload`)는 `.md` 에서 embeddedTitle 을 만들지 않아 제목이 파일 이름 스템에서 오고 H1 은 본문에 그대로 남습니다. 그래서 muni→muni 넘기기(HANDOFF, `handoff.go:220` 이 `document.Filename` 스템을 제목으로 씀)나 내보낸 .md 를 다시 가져오면 문서 제목과 본문 첫 줄에 같은 제목이 두 번 보입니다(v0.39.0 회차에 실제 띄워 확인된 현상). 고치면 넘겨받은 문서와 다시 가져온 문서가 원본과 같은 모양이 됩니다.

- 수용 기준:
  1) `.md` 본문의 **첫 블록**이 level-1 heading 이고 그 PlainText 가(TrimSpace 뒤) 최종 결정된 제목과 정확히 같으면, 저장되는 content 에 그 heading 이 없다. 다른 블록(그 뒤의 H1 포함)은 그대로.
  2) 첫 블록이 H1 이어도 제목과 다르면(또는 폼 `title` 이 다른 값이면) 본문은 손대지 않는다. 첫 블록이 H1 이 아니면(예: 문단 뒤 H1) 손대지 않는다. `.txt`·`.html`·`.docx`·`.hwp`·`.hwpx`·`.pdf` 경로는 바이트 하나 달라지지 않는다.
  3) 적용 자리는 둘: 파일 업로드 가져오기(`import_attachments.go` importDocument, 제목 결정 뒤 ~line 121) 와 넘겨받기(`handoff.go:storeHandoff`, ~line 226). `importIntoDocument`(열린 문서에 끼워 넣기) 는 **적용하지 않는다** — 그 자리는 제목을 편집기에 따로 돌려주고 본문은 끼워 넣는 것이라 H1 을 지우면 글이 사라진다.
  4) 테스트가 증명할 것: (a) `renderMarkdown("제목", doc)` 의 결과를 `markdownDocument` → 새 헬퍼로 넘겼을 때 H1 이 사라지고 나머지 블록이 같다(왕복), (b) 제목이 다르면 그대로, (c) 첫 블록이 H1 이 아니면 그대로, (d) 이스케이프된 제목(`renderMarkdown` 은 `escapeMarkdown` 으로 `*`·`_`·`[` 를 `\*` 로 적음 — `import_inline.go:66` 이 되돌림)도 같다고 판단한다 — 예: 제목 `A_B*C`. live 테스트(`handoff_live_test.go:TestReceivingTakesOnlyFromAListedSource` 의 peer 가 markdown 을 내주는 case) 에서 `# <제목>` 으로 시작하는 본문을 보내 받은 문서의 content_json 첫 블록이 heading 이 아님을 확인하면 좋음(MUNI_TEST_DSN 필요).

- 건드릴 파일:
  - `internal/httpapi/import_markdown.go` — 새 헬퍼 하나. 제안: `func dropLeadingTitle(content json.RawMessage, title string) (json.RawMessage, error)` — `richdoc.Parse` → `document.Content[0]` 가 `heading` 이고 attrs level==1 이고 `PlainText()` 를 TrimSpace 한 값이 `strings.TrimSpace(title)` 와 같으면 `Content = Content[1:]` 후 `document.JSON()`. 아니면 content 를 그대로 돌려줌. richdoc 노드의 heading 타입 이름·level attr 이름은 `internal/richdoc/richdoc.go`(`Text`, `PlainText` line 161) 와 `toc.go:27 Headings` 에서 실제 이름을 확인해 쓸 것(미확인 — 여기서 추측하지 않음). 본문이 H1 하나뿐이었다면 비어 버리므로 `richdoc.Doc()` 가 빈 content 를 허용하는지 확인하고, 아니면 빈 문단 하나를 남길 것.
  - `internal/httpapi/import_attachments.go` — `parseUpload` 가 마크다운이었는지 알 수 있게 `upload` 에 `markdown bool`(또는 extension) 필드를 두고, importDocument 에서 `title` 확정 뒤·`extractDocumentText` 전에 `if parsed.markdown { content, err = dropLeadingTitle(content, title) }`. 순서: `withBlockIDs` 전이든 후든 결과는 같으나 `prepareImportedAssets`·`withBlockIDs` 가 content 를 만드는 흐름과 어긋나지 않게 **title 확정 직후, text 추출 직전**에 둘 것(text 는 content 에서 뽑으므로 검색 텍스트에도 제목이 두 번 남지 않는다).
  - `internal/httpapi/handoff.go:storeHandoff` — 같은 한 줄. 형식은 `handoff.Extension(document.Format)` 가 `.md` 일 때(markdown 만 받으니 사실상 항상). `text: extractDocumentText(content)` 가 새 content 를 보도록 위치 주의.
  - `internal/httpapi/import_export_test.go` — 위 (a)~(d) 단위 테스트. 기존 `TestMarkdownImport`(line 13) 은 `markdownDocument` 만 부르므로 그대로 통과해야 함(헬퍼는 별도 함수).
  - (선택) `internal/httpapi/handoff_live_test.go` — 넘겨받기 통합 확인.

- 검증 명령:
  - `gofmt -l .` (출력 없음), `go vet ./...`
  - `go test ./internal/httpapi/ -run 'Markdown|LeadingTitle|Import'` (DSN 없이 단위만, 약 2초)
  - `go test ./...` — live 테스트까지 돌리려면 `MUNI_TEST_DSN=postgres://postgres:muni@127.0.0.1:5432/muni?sslmode=disable` (CI 와 같은 postgres:16-alpine 컨테이너; 이전 회차는 docker 로 띄웠음). DSN 없으면 live 테스트는 스스로 skip 하므로 그것만으로 "통과" 라 쓰지 말 것.
  - 프런트는 건드리지 않으므로 `npm` 단계 불필요. 커밋 전 `scripts/check-webui-placeholder.sh`.

- 위험과 피할 것:
  - **제목이 같을 때만** 지울 것. 첫 H1 을 무조건 제목으로 올리는 설계(pandoc 식)는 사용자 마크다운의 H1 을 본문에서 없애 버려 기존 행동을 바꾸므로 이번엔 하지 않는다. 대소문자·공백 정규화도 TrimSpace 이상은 하지 말 것(오탐이 글을 지운다).
  - `safeFilename` 은 `/`→`-` 치환·100자 절단을 하므로 그런 제목은 H1 과 파일 이름 스템이 달라 지워지지 않는다 — 의도된 보수적 동작, 테스트로 "지우지 않음" 을 못 박아도 좋음.
  - `importIntoDocument`(끼워 넣기) 와 `.html` 가져오기는 손대지 않는다. `renderMarkdown`(내보내기) 도 바꾸지 않는다 — 내보낸 파일이 GitHub 등에서 제목을 보이려면 H1 이 있어야 한다.
  - 보호 경로(auth·migrations·workflows) 는 전혀 필요 없다. DB 스키마 변경 없음.
  - 관리자 가이드·README 는 바꿀 것 없음. 사용자 가이드에 마크다운 가져오기 절이 있으면 한 줄("첫 H1 이 제목과 같으면 본문에서 뺀다") 정도만 — 미확인, `docs/USER_GUIDE.md` 를 grep 해 볼 것.
  - 커밋 메시지는 한국어 `fix: …` 형식(최근 로그 참고), Co-Authored-By 트레일러 금지.

- 차선 후보: 관리 화면 「운영 현황」에 최근 24시간 메일 실패 수 보이기 (2/1/S) — `internal/httpapi/admin_overview.go` 의 연결 상태 블록에 `mail_deliveries` 의 `status='failed' AND created_at > now()-interval '24 hours'` 합계 한 칸(열 이름은 019 마이그레이션에서 확인 — 미확인) + 프런트 관리 개요 카드 한 줄 + live 테스트 하나. 1순위가 richdoc 노드 구조 때문에 뜻밖에 커지면 이것으로.
