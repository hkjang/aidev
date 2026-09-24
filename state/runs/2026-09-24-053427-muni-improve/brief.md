# 과제서 (2026-09-24, base main@b40de35 / VERSION 0.43.0)

- 과제: 내려받기 파일 이름과 ZIP 항목 이름에 AI 컨텍스트 안내 문구가 붙는 것 고치기 (가치 3 / 위험 1 / 작업량 S)

- 왜: `internal/httpapi/export.go:307` 의 `safeFilename` 이 절단에 `truncateRunes`(`internal/httpapi/ai.go:238`)를 재사용하는데, 그 함수는 잘린 뒤 `"\n[…문서 컨텍스트가 길어 일부 생략됨…]"` 를 덧붙입니다 — AI 프롬프트 컨텍스트용 문구입니다. 문서 제목은 API 에서 240자까지 허용되므로(`internal/httpapi/documents.go:127`,`:322` 의 "제목은 240자 이하여야 합니다") 101자 이상 제목은 평범한 UI 조작으로 만들 수 있고, 그 문서의 내려받기 파일 이름과 워크스페이스 ZIP 항목 이름에 줄바꿈과 이 문구가 그대로 들어갑니다. 고치면 긴 제목 문서의 파일 이름이 제목 그대로(100자까지)가 되고, ZIP 항목 이름에서 줄바꿈이 사라집니다.

- 수용 기준:
  1) 제목이 101자 이상인 문서를 `GET /api/v1/documents/{id}/export?format=md` 로 내려받으면 `Content-Disposition` 의 `filename*=UTF-8''…` 이 "제목 앞 100자 + `.md`" 이고, `[…문서 컨텍스트가 길어 일부 생략됨…]` 도 줄바꿈도 들어 있지 않다.
  2) 같은 제목의 문서가 들어 있는 워크스페이스를 `GET /api/v1/workspaces/{id}/export.zip?format=md` 로 내려받으면 ZIP 항목 이름에 그 문구와 줄바꿈이 없고, `목록.md` 가 적는 이름과 실제 항목 이름이 같다.
  3) 100자 이하 제목·빈 제목의 동작은 한 바이트도 변하지 않는다(빈 제목은 계속 `muni-document`, `uniqueEntryName` 의 `제목 없는 문서` 경로와 `safeFolderSegment` 의 `.`·`..` 판정도 그대로).
  4) 테스트가 증명할 것: 수정 전에는 새 테스트가 "파일 이름에 안내 문구/줄바꿈이 있다" 로 **실패**하고, 수정 후 통과하며, 수정만 되돌리면 다시 실패한다(인과 확인). 기존 `workspace_export_test.go` 6개와 `import_export_test.go` 의 `safeFilename` 을 쓰는 케이스는 손대지 않고 그대로 통과한다.

- 건드릴 파일:
  - `internal/httpapi/export.go:307 safeFilename` — 마지막 줄 `return truncateRunes(value, 100)` 을 **파일 이름 전용 절단**으로 바꿉니다. 같은 파일에 작은 헬퍼(예: `cutFilenameRunes(value string, max int) string`)를 두어 100 룬에서 자르기만 하고 아무 문구도 붙이지 않으며, 자른 뒤 끝에 남는 공백은 `TrimRight`/`TrimSpace` 로 털어 냅니다. 빈 값 처리(`"" → "muni-document"`)와 치환기(`/`,`\`,`\r`,`\n`,`\x00`)와 **절단 길이 100 자체는 바꾸지 마세요.** 이 저장소 관례대로 "왜 AI 쪽 헬퍼를 쓰지 않는가" 를 산문 주석으로 답니다.
  - `internal/httpapi/export_coverage_test.go` 또는 새 `export_filename_test.go` — 단위 테스트: 120자 제목 → 결과가 정확히 앞 100 룬이고 `strings.Contains(got, "생략됨")==false`, `strings.ContainsAny(got, "\r\n")==false`; 100자·짧은 제목·빈 제목·`/` 포함 제목은 기존과 동일(표-주도).
  - `internal/httpapi/workspace_export_live_test.go` 또는 새 live 테스트 — 실제 라우트로 ZIP 을 받아 항목 이름을 읽습니다. 기존 헬퍼 `serverUnderTest`, `folderNamed`, `documentInFolder`, `postJSON` 를 그대로 재사용하세요(같은 패키지, 같은 파일에 있음). 문서 내려받기 쪽 live 확인도 한 자리에서 하면 좋습니다 — `mime.ParseMediaType` 으로 헤더를 되읽는 방식은 `handoff_live_test.go:112` 에 선례가 있습니다.

- 검증 명령:
  - `go vet ./...`
  - `gofmt -l internal/httpapi`  (출력이 비어 있어야 함)
  - `go test ./internal/httpapi/ -run 'Filename|Export|Workspace' -count=1`
  - 전체(권장): postgres:16-alpine 컨테이너를 띄워 `MUNI_TEST_DSN=…` 을 주고 `go test ./... -count=1` — DSN 없이 돌리면 live 테스트가 SKIP 되므로 "통과" 로 적지 마세요.
  - `scripts/check-webui-placeholder.sh`
  - **사전 확인한 환경 사실**: 이 작업 트리에서 `go test ./internal/httpapi/` 를 그대로 돌리면 `TestDevtoolsPDFHasPageNumbers` 가 `Chromium이 디버그 주소를 알려주지 않았습니다` 로 실패합니다(이 환경에 쓸 수 있는 Chromium 이 없음). 이 과제와 무관한 기존 실패이니 고치려 들지 말고, 나머지가 통과하는지로 판단하세요.

- 위험과 피할 것:
  - `safeFilename` 은 문서 내려받기(`export.go:60`), 발표자료 내려받기(`presentations.go:241`), 워크스페이스 ZIP 파일 이름(`workspace_export.go:109`)·항목 이름(`:139`)·폴더 세그먼트(`safeFolderSegment`, `:240`) 다섯 자리가 공유합니다. 이번에는 **절단 뒤에 붙는 꼬리만** 없애고 나머지 계약은 건드리지 마세요. 다섯 경로가 같은 이름을 같게 읽는지 단위+live 로 함께 확인하세요(운영자 지침: 같은 값을 읽는 경로가 여럿이면 한쪽만 고치지 말 것).
  - `internal/httpapi/ai.go:238 truncateRunes` 는 **고치지 마세요.** AI 컨텍스트(`ai.go:94`, `ai_tools.go`, `ai_patch.go`)와 DB 길이 상한(제목 240, 첨부 이름 240, 머리말/꼬리말 200) 이 같이 씁니다. 거기서 문구를 떼면 AI 프롬프트 쪽 계약이 바뀝니다.
  - `internal/httpapi/import_attachments.go:121`·`handoff.go:227` 의 `truncateRunes(title, 240)` 도 같은 문구를 **문서 제목**에 붙이지만(241자 이상 제목을 가진 파일을 가져올 때), 이번 범위 밖입니다 — 데이터가 바뀌는 자리라 따로 다뤄야 합니다. ideas.json 에 별도 항목으로 남겼습니다.
  - `urlPathEscape`(`export.go:315`)는 줄바꿈을 이스케이프하지 않습니다. 증상의 일부이지만 이번에 넓히지 마세요 — 원인을 없애면 줄바꿈이 애초에 생기지 않습니다(운영자 지침: 파서/이스케이프 계약을 넓히지 말고 원인 쪽을 좁힐 것).
  - 보호 경로(auth.go, settings 봉인, internal/database/migrations, .github/workflows) 는 건드리지 않습니다. 이 과제는 그럴 이유가 없습니다.
  - **미확인**: 정찰은 저장소 코드를 고치지 않는 규칙 때문에 동적 재현을 하지 않았습니다. 위 결론은 `safeFilename` → `truncateRunes` 호출과 제목 240자 허용을 코드에서 읽은 것입니다. 또 Go 의 `net/http` 는 헤더 값의 줄바꿈을 공백으로 바꿔 내보내는 것으로 알고 있어 **헤더 주입은 아닐 가능성이 큽니다** — 실제 바이트는 확인하지 않았으니, 구현자가 live 테스트로 헤더 원문을 먼저 찍어 보고 증상을 확정하세요. ZIP 항목 이름에는 그런 정화가 없으므로 줄바꿈이 그대로 들어갑니다(이쪽이 더 확실한 증거 자리).

- 차선 후보: 워크스페이스 ZIP 한도 초과 안내를 실제로 넘쳤을 때만 넣기 (가치 2 / 위험 1 / 작업량 S) — `workspace_export.go:155` 의 `if len(items) == maxWorkspaceExport` 는 정확히 2000건인 워크스페이스에도 "넘어 그만큼만 담았습니다" 를 적습니다. `:65` 의 `LIMIT $3` 을 `maxWorkspaceExport+1` 로 읽고, 2001번째가 있을 때만 안내하며 목록은 2000건으로 자르면 됩니다. 코드 위치는 이번 회차에 다시 확인했습니다.
