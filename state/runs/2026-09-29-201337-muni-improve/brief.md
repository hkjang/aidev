- 과제: 가져오기·첨부·넘겨받기가 저장하는 제목·첨부 이름의 240룬 절단에 AI 안내 문구가 들어가는 것 고치기 (가치 3 / 위험 1 / 작업량 S)

- 왜: `truncateRunes`(`internal/httpapi/ai.go:238`)는 AI 프롬프트용이라 자를 때 `"\n[…문서 컨텍스트가 길어 일부 생략됨…]"` 를 덧붙이는데, DB 에 저장되는 **문서 제목**과 **첨부 이름**의 240룬 절단이 전부 그것을 쓰고 있습니다. `documents.title`·`attachments.name` 은 `text` 컬럼이라 길이 제한이 없어(`internal/database/migrations/001_initial.sql:119`) 문구가 그대로 저장되고, 제목은 문서 목록·검색 색인(`documents_search_idx` 가 title 을 포함)·워크스페이스 ZIP 의 `목록.md` 에, 첨부 이름은 첨부 목록과 내려받기 `Content-Disposition` 의 `filename*`(`import_attachments.go:652`, `extValueEscape(name)`)에 그대로 나갑니다. 2026-09-24 회차가 `safeFilename` 에서, 2026-09-28 회차가 `imageAssetName` 에서 걷어낸 것과 **같은 결함의 남은 자리**이고, 그때 만든 문구 없는 절단기 `cutFilenameRunes`(`internal/httpapi/export.go:335`, 같은 패키지)가 이미 있습니다.

- 수용 기준:
  1) 241룬 이상 파일 이름으로 첨부를 올리면(`POST /api/v1/documents/{id}/attachments`) 저장된 `attachments.name` 이 정확히 앞 240룬이고, 줄바꿈도 `[…문서 컨텍스트가 길어 일부 생략됨…]` 도 없다. 이어서 `GET /api/v1/attachments/{id}` 응답의 `Content-Disposition` 을 **프로덕션 파서 `mime.ParseMediaType`** 으로 되읽은 이름이 그 저장값과 같다.
  2) 241룬 이상 제목으로 문서를 가져오면 저장된 `documents.title` 이 정확히 앞 240룬이고 같은 문구·줄바꿈이 없다.
  3) 240룬 이하의 제목·첨부 이름은 한 글자도 바뀌지 않는다(표-주도 불변 테스트: 240룬 정확히·1룬·빈 값·확장자 포함 이름).
  4) AI 프롬프트 쪽(`ai.go:94`, `ai_patch.go:193`, `ai_tools.go:220,325`)의 `truncateRunes` 사용과 문구 자체는 그대로다 — 거기서는 그 문장이 목적이다.
  5) 바꾼 절단 호출만 `truncateRunes` 로 되돌리면 새 테스트만 다시 실패하고 기존 테스트(직전 기준 239건)는 계속 통과한다.

- 건드릴 파일 (프로덕션 2개):
  - `internal/httpapi/import_attachments.go` — 아래 다섯 자리의 `truncateRunes(…, 240)` 을 `cutFilenameRunes(…, 240)` 로. 길이 240 은 그대로 둔다.
    - `:121` `importDocument` — 저장할 문서 제목
    - `:183` `storeImportedDocument` — `attachments.name` INSERT
    - `:312` `importIntoDocument` 트랜잭션 — `attachments.name` INSERT
    - `:329` `importIntoDocument` 의 `writeData` 응답 `"title"` — **감사 로그가 아니라 편집기가 받아 제목 칸에 넣는 값**이다. 여기를 빼면 같은 입력이 "기존 문서로 가져오기" 에서는 문구가 붙고 "새 문서로 가져오기" 에서는 안 붙어 두 경로가 갈린다. 반드시 함께 바꿀 것.
    - `:570` `uploadAttachment` — `truncateRunes(filepath.Base(header.Filename), 240)`. **재현이 가장 쉬운 자리**(멀티파트 `filename` 이 그대로 들어옴).
  - `internal/httpapi/handoff.go:230` `truncateRunes(title, 240)` — 넘겨받은 문서 제목. 같은 교체.
  - **범위 밖(건드리지 말 것)**: `import_attachments.go:174,330,331` 의 머리말·꼬리말 200룬 절단(DB 데이터이긴 하나 별도 재현 근거가 필요해 이번 회차에서 의도적으로 제외), `handoff.go:257`(사람이 읽는 안내 문장), `mail.go:91,98`(별도 과제), AI 호출부 전부.
  - 테스트: 단위는 `internal/httpapi/import_text_test.go` 또는 새 파일, live 는 `internal/httpapi/content_disposition_live_test.go`(첨부 이름·헤더 왕복 선례가 이미 그 파일에 있음) / `import_into_document_live_test.go`.
  - `cutFilenameRunes`·`truncateRunes`·`safeFilename`·`extValueEscape` 의 **본문과 이름은 한 글자도 바꾸지 말 것**. (`cutFilenameRunes` 는 이름에 "filename" 이 들어가지만 하는 일은 "문구 없이 자르고 자른 자리 끝 공백만 털기" 라 제목에도 맞다. 개명은 호출처가 늘어 이번 범위 밖 — 대신 호출부에 왜 이 절단기인지 저장소 관례대로 산문 주석을 남길 것.)

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - live 에 DB 필요: `docker run -d --rm -p 55432:5432 -e POSTGRES_PASSWORD=muni -e POSTGRES_DB=muni postgres:16-alpine`
    → `export MUNI_TEST_DSN='postgres://postgres:muni@127.0.0.1:55432/muni?sslmode=disable'`
  - `go test -count=1 -v ./internal/httpapi` — 직전 회차 기준 PASS 239 / SKIP 0. **SKIP 이 나오면 DSN 이 안 걸린 것이니 "통과" 로 적지 말 것.**
  - `go test ./...` / `go vet ./...` / `gofmt -l .` / `scripts/check-webui-placeholder.sh`
  - 프런트 미변경이면 npm 검사 생략 가능(직전 네 회차 선례). `make build` 는 tracked `webui/dist/index.html` 을 덮으니 돌리지 말 것.
  - 환경: Chromium 이 없는 워크트리면 `TestDevtoolsPDFHasPageNumbers` 가 실패한다 — 이 과제와 무관하니 먼저 환경을 확인할 것.

- 위험과 피할 것:
  - **미확인(정찰이 동적 재현을 못 함)**: 241룬 입력이 각 자리에 실제로 도달하는지. 코드상 도달 경로는 (a) `:570` 멀티파트 `header.Filename` — 다른 길이 검사 없음, 가장 쉬움, (b) `:121` `r.FormValue("title")` — `importDocument` 안에 다른 길이 검사가 없음, (c) `embeddedTitle`(DOCX/HWP/PDF metadata 제목), (d) handoff 는 피어가 보낸 `parsed.title`·`document.Filename`. 구현자는 **(a) 를 실제 라우트로 먼저 재현**하고(수용 기준 1), (b) 로 제목 쪽도 한 번 확인한 뒤, 실제로 도달하지 않는 자리는 단위로만 덮으세요.
  - `attachments.name` 을 쓰는 세 자리는 값의 출처가 다릅니다(`attachment.Name` 은 reader 가 만든 이름, `header.Filename` 은 업로드 원본). **한쪽만 고치면 같은 값이 경로에 따라 다르게 저장됩니다 — 세 자리를 함께** 바꾸세요.
  - `dropLeadingTitle`(`import_markdown.go`)은 확정된 title 과 **정확히 같은** 첫 H1 만 제거합니다. 제목 절단 결과가 바뀌면 그 비교값도 바뀌므로, 241룬 제목 + 같은 H1 을 가진 마크다운에서 본문 H1 제거 동작이 어떻게 되는지 한 번 확인하고 회차 노트에 적으세요(원래도 문구 때문에 안 맞았을 가능성이 높습니다 — 그렇다면 이 수정이 그것도 고칩니다).
  - 보호 경로 금지: `auth.go`, `internal/database/migrations`, `.github/workflows`. 마이그레이션 불필요(컬럼이 `text`).
  - 관례: 커밋 메시지 한국어 `fix:`, 주석은 "왜" 를 산문으로 길게, 테스트 이름은 문장형(예: `TestALongAttachmentFilenameIsStoredWithoutAContextNotice`).

- 차선 후보: 첨부 내려받기 `Content-Disposition` 의 ASCII fallback 이 상수 `filename="attachment"` 라 확장자가 없는 것(`import_attachments.go:652` — 확인함: `filename="attachment"; filename*=UTF-8''…`). `filename*` 을 못 읽는 클라이언트가 확장자 없는 파일을 받아 열지 못합니다. 저장된 이름의 확장자를 fallback 에 붙이는 프로덕션 1파일 + live 1테스트로 끝납니다.
