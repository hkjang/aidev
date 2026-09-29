- 과제: 첨부 내려받기의 `Content-Disposition` ASCII fallback 이름에 확장자 보존하기 (가치 2 / 위험 1 / 작업량 S)

- 왜: 내려받기 헤더를 내는 다섯 자리 가운데 첨부 내려받기(`internal/httpapi/import_attachments.go:678`)만 ASCII fallback 이 확장자 없는 상수 `filename="attachment"` 입니다 — 나머지 넷은 모두 확장자를 붙입니다(`export.go:61` `document.%s`, `presentations.go:243` `presentation.%s`, `workspace_export.go:131` `workspace-%s.zip`, `handoff.go:150` `document%s` = `filepath.Ext(filename)`). `filename*` 을 읽지 못하는 클라이언트(구형 브라우저·단순 HTTP 클라이언트·일부 폐쇄망 도구)는 fallback 을 쓰므로 `.xlsx` 첨부가 확장자 없는 `attachment` 로 저장되어 연결 프로그램이 열리지 않습니다. 고치면 다섯 경로의 fallback 계약이 같아집니다.

- 수용 기준:
  1) 이름이 `회의록.xlsx` 인 첨부를 `GET /api/v1/attachments/{id}` 로 받으면 응답 `Content-Disposition` 의 ASCII fallback 이 `filename="attachment.xlsx"` 다(현재는 `filename="attachment"`).
  2) `filename*` 파라미터와 그 값(`extValueEscape(name)`)은 한 글자도 바뀌지 않는다 — 기존 `TestAnAttachmentDownloadNamesTheFileSoAParserReadsItBack`(`content_disposition_live_test.go:70`)이 이름 `회의 자료(최종); 100%.txt` 를 그대로 되읽는 것이 계속 통과해야 한다.
  3) **확장자가 fallback 을 깨뜨리지 않는다.** 첨부 이름은 사용자가 올린 멀티파트 `header.Filename` 을 `cutFilenameRunes(…, 240)` 로 길이만 자른 값이고(`import_attachments.go:596`), `cutFilenameRunes`(`export.go:335`)는 절단과 끝 공백 제거만 하므로 이름에 `"` `;` 제어문자가 그대로 남아 있을 수 있다. 이름이 `보고서.p"g` 나 `자료.한글확장자` 처럼 quoted-string 에 넣을 수 없는 확장자를 가지면 fallback 은 안전한 `filename="attachment"` 로 남아야 하고, 응답 헤더를 프로덕션 파서 `mime.ParseMediaType`(테스트 헬퍼 `filenameFrom`, `content_disposition_live_test.go:30`)으로 읽었을 때 **여전히 파싱에 성공하고 `filename*` 의 원래 이름을 돌려주어야** 한다.
  4) 테스트가 증명할 것: 위 세 가지를, 손으로 만든 구조체가 아니라 실제 업로드·내려받기 라우트를 지나는 live 테스트의 표-주도 행으로. 최소 행 — 평범한 `회의록.xlsx`(확장자 붙음) / 확장자 없는 `회의록`(fallback 이 `attachment` 그대로) / 따옴표가 든 확장자(fallback 이 `attachment` 그대로, 헤더는 파싱됨) / 비ASCII 확장자(같음).

- 건드릴 파일 (프로덕션 1 + 테스트 1):
  - `internal/httpapi/import_attachments.go:651 downloadAttachment` — `:678` 의 `fmt.Sprintf` 한 줄. `filename="attachment"` 를 `filename="attachment<안전한 확장자>"` 로. 확장자는 `filepath.Ext(name)` 로 뽑되 **그대로 쓰지 말고** 안전할 때만 채택한다: 점으로 시작하고, 나머지가 전부 ASCII 영숫자이며, 길이가 상식적인 범위(예: 점 포함 12자 이하)일 때만. 그 밖에는 빈 문자열(= 오늘과 같은 `attachment`). 판정은 같은 파일 안의 작은 비공개 헬퍼(예: `asciiFallbackExt(name string) string`)로 빼고, `filepath` 는 이 파일이 이미 import 한다(`:78`, `:116`).
  - `internal/httpapi/content_disposition_live_test.go` — `TestAnAttachmentDownloadNamesTheFileSoAParserReadsItBack`(`:70`) 바로 아래에 새 live 테스트를 추가. 기존 헬퍼 `uploadAttachment`, `markdownDocumentOwnedByAdmin`, `filenameFrom`, `newServerUnderTest` 를 그대로 쓴다. 기존 테스트 본문은 고치지 말 것(2번 기준의 회귀 감시자다).
  - 헬퍼를 순수 함수로 뺐다면 `content_disposition_test.go` 에 불변 표를 몇 줄 더해도 좋다(선택). live 가 이미 프로덕션 배선을 지나므로 필수는 아니다.

- 검증 명령 (이 정찰 세션에서는 샌드박스가 `go` 실행을 승인 대기로 막아 **실행하지 못했습니다 — 미확인**. 구현자가 먼저 재현할 것):
  - `MUNI_TEST_DSN` 없이는 live 테스트가 SKIP 된다. 전용 DB 를 띄울 것:
    `docker run -d --rm -e POSTGRES_PASSWORD=muni -e POSTGRES_DB=muni -p 55432:5432 postgres:16-alpine`
    `export MUNI_TEST_DSN='postgres://postgres:muni@127.0.0.1:55432/muni?sslmode=disable'`
  - `go test -count=1 -run 'Disposition|Attachment|Download' -v ./internal/httpapi` (새 테스트를 먼저 실패시켜 둘 것)
  - `go test -count=1 ./internal/httpapi` — 직전 회차 기준 PASS 242 / SKIP 0 이 기준선. **SKIP 0 인지 확인하고, 단위만 통과한 것을 "통과" 로 적지 말 것.**
  - `go test ./...`, `go vet ./...`, `gofmt -l .`, `scripts/check-webui-placeholder.sh`
  - 인과 확정: 헤더 한 줄만 되돌려 새 테스트만 실패하고 나머지가 계속 통과하는지 볼 것.
  - 프런트 미변경이므로 npm 검사는 불필요. `make build` 는 tracked `webui/dist/index.html` 을 덮으므로 돌리지 말 것.

- 위험과 피할 것:
  - **가장 큰 함정은 `filepath.Ext(name)` 를 그대로 quoted-string 에 넣는 것.** 첨부 이름은 검열되지 않은 사용자 입력이라 `"` 하나로 헤더 전체가 `mime: invalid media parameter` 가 된다 — 오늘(확장자만 없음)보다 **나쁜** 상태다. 2026-09-26 회차가 워크스페이스 ZIP 에서 정확히 이 모양으로 헤더를 깨뜨린 것을 고쳤다. `handoff.go:150` 이 `filepath.Ext` 를 맨몸으로 쓰는 것은 그쪽 이름이 이 경로와 출처가 달라서이니, 그 줄을 근거로 삼아 복사하지 말 것.
  - `extValueEscape`, `safeFilename`, `cutFilenameRunes` 본문은 건드리지 말 것 — 다섯 내려받기 경로가 공유한다.
  - 나머지 네 헤더 자리와 `disposition`(inline/attachment) 분기, CSP sandbox 헤더, 감사 기록은 손대지 말 것. 이번 회차는 fallback 문자열 하나다.
  - 다섯 자리를 공통 헬퍼로 모으는 리팩터는 이번에 하지 말 것(보류 아이디어로 남아 있음) — 파일 수가 늘고 회귀 표면이 넓어진다.
  - 보호 경로(`auth.go`, `internal/database/migrations`, `.github/workflows`)는 닿지 않는다.
  - Chromium 이 없는 워크트리라면 `TestDevtoolsPDFHasPageNumbers` 가 실패한다 — 이번 변경과 무관한 환경 문제다.

- 차선 후보: 워크스페이스 ZIP 정렬 동률에 문서 ID 순서 추가 — `workspace_export.go:71` 의 `ORDER BY d.folder_id NULLS FIRST, d.title` 에 `, d.id` 를 더해 같은 제목 문서들의 `(2)` 번호와 2000건 절단 경계를 결정적으로 만들기 (2/1/S). 1순위가 성립하지 않을 때(예: 이미 고쳐져 있을 때) 이쪽으로.
