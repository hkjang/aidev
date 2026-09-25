# 과제서 — 2026-09-25-000113-muni-improve (muni)

- **과제**: 내려받기 `Content-Disposition` 의 `filename*` 를 RFC 8187 ext-value 로 제대로 퍼센트 인코딩하기 (가치 3 / 위험 2 / 작업량 S)

- **왜**: `internal/httpapi/export.go:343` 의 `urlPathEscape` 는 `%`, 공백, `#`, `?`, `"` 다섯 글자만 바꾸고 한글 UTF-8 바이트를 헤더에 그대로 싣는데, RFC 8187 의 ext-value 는 attr-char 밖을 모두 퍼센트 인코딩하라고 합니다. 그 결과 한글 제목 문서의 내려받기 헤더는 표준 파서가 거부합니다 — 저장소 안에 이미 그 증거가 있습니다: `internal/httpapi/export_filename_live_test.go:88-96` 이 "이 헤더는 아직 `mime.ParseMediaType` 을 통과하지 못한다, 제목 길이와 무관하게 모든 한글 제목이 그렇다" 를 **알려진 공백으로 기록**해 두었고(2026-09-24 회차가 live 로 확인), 같은 패키지의 넘겨받기 경로 `internal/httpapi/handoff.go:147` 은 이미 `url.PathEscape` 로 전부 인코딩하며 "이름은 (RFC 8187) 전부 퍼센트 인코딩한다" 는 주석까지 달고 있어 **같은 값을 쓰는 네 경로가 헤더를 서로 다르게 만들고 있습니다**. 고치면 한글 제목 문서·발표자료·첨부를 내려받는 클라이언트가 파일 이름을 표준대로 읽고, 네 경로가 한 헬퍼로 같은 헤더를 냅니다.

- **수용 기준**
  1. 한글 제목(예: `9월 회의록`) 문서를 `GET /api/v1/documents/{id}/export/md` 로 내려받으면 `Content-Disposition` 이 `mime.ParseMediaType` 을 err 없이 통과하고, `params["filename"]` 이 정확히 `9월 회의록.md` 로 되읽힙니다. (고치기 전에 이 단언이 `err != nil` 로 실패하는 것을 먼저 볼 것.)
  2. 같은 것이 발표자료 내려받기(`presentations.go:243`)와 첨부 내려받기(`import_attachments.go:652`)에서도 성립하고, 넘겨받기 클레임 내려받기(`handoff.go:147`)의 기존 동작은 그대로입니다 — `handoff_live_test.go:112` 의 `params["filename"] != "2026년 3분기 개편안.md"` 단언이 계속 통과해야 합니다.
  3. 새 이스케이퍼의 표-주도 단위 테스트가, attr-char(`ALPHA DIGIT ! # $ & + - . ^ _ \` | ~`)는 한 바이트도 바꾸지 않고 그 밖(한글, 공백, `;`, `=`, `'`, `,`, `(`, `)`, `*`, `:`, `@`, `%`, `"`, `?`)은 모두 대문자 `%XX` 로 바꾸는 것을 증명합니다. 특히 `;` `=` `'` 는 `url.PathEscape` 가 남기는 글자라, 그냥 `url.PathEscape` 로 바꾸는 것으로는 `제목;a=b` 같은 제목에서 헤더가 다시 깨집니다(미확인 — 착수 시 그 입력으로 실패 테스트를 먼저 세워 확인할 것).
  4. `filename="document.md"` / `filename="presentation.md"` / `filename="attachment"` 라는 ASCII 폴백 파라미터와 `safeFilename`·`cutFilenameRunes` 의 결과는 한 글자도 달라지지 않습니다.

- **건드릴 파일**
  - `internal/httpapi/export.go` — `urlPathEscape`(343) 자리에 RFC 8187 ext-value 이스케이퍼를 두세요. 바이트 단위로 돌며 attr-char 이 아니면 `%%%02X` 를 쓰는 형태면 충분하고, UTF-8 멀티바이트는 바이트마다 인코딩되는 것이 맞습니다. 이름은 `extValueEscape` 처럼 무엇을 만드는지 드러나게 하고, 왜 `url.PathEscape` 로는 부족한지(sub-delims 를 남김)를 이 저장소의 주석 밀도대로 산문으로 적으세요. `urlPathEscape` 에 호출자가 더 없으면(현재 export.go:61, presentations.go:243, import_attachments.go:652 세 곳뿐 — `grep -rn urlPathEscape` 로 확인함) 헬퍼를 대체하고 남기지 마세요.
  - `internal/httpapi/export.go:61` / `internal/httpapi/presentations.go:243` / `internal/httpapi/import_attachments.go:652` — `urlPathEscape(...)` 를 새 헬퍼로 교체.
  - `internal/httpapi/handoff.go:147` — `url.PathEscape(filename)` 을 같은 헬퍼로 바꿔 네 경로가 한 규칙을 쓰게 하세요. `url.PathEscape` 결과는 attr-char 로만 이루어진 이름에 대해 새 헬퍼와 같으므로 기존 live 단언은 유지되어야 합니다(한글은 양쪽 다 `%XX`). `net/url` import 가 다른 데서 안 쓰이면 정리.
  - `internal/httpapi/export_filename_live_test.go:70-96` — **기존 기대값이 반드시 깨집니다.** `want` 가 지금은 "공백만 `%20`, 한글은 그대로" 를 기대하고, 89~96행이 `mime.ParseMediaType` 실패를 '알려진 공백' 으로 적어 둡니다. 기대값을 완전 인코딩된 형태로 고치고, 그 주석과 `t.Logf` 자리를 "통과해야 한다" 는 단언으로 바꾸세요(주석이 스스로 "이것이 고쳐지면 이 메모는 없어져도 된다" 고 적어 둔 자리입니다).
  - 새 테스트: 위 3)의 단위 표 + 한글 제목 live 1개(문서 내려받기 헤더를 `mime.ParseMediaType` 으로 되읽기). 첨부·발표자료는 live 준비 비용이 크면 단위/헬퍼 수준으로 두되, 최소한 문서 내려받기 하나는 실제 라우트로 갈 것.
  - 사용자 문서 수정은 불필요해 보임(내부 헤더 동작).

- **검증 명령** (이 저장소에서 실제로 도는 것)
  - `go test ./internal/httpapi/ -run 'Filename|Handoff|Export' -count=1 -v`
  - postgres:16-alpine 컨테이너를 띄워 `MUNI_TEST_DSN` 을 주고 `go test ./... -count=1` — **DSN 없이 통과한 것을 "통과" 로 적지 말 것** (live 테스트가 통째로 SKIP 됩니다). httpapi 의 SKIP 이 0 인지 확인하세요.
  - `go vet ./...`, `gofmt -l internal/httpapi`(빈 출력), `scripts/check-webui-placeholder.sh`
  - 고친 뒤 **헬퍼 본문만 되돌려** 새 테스트만 다시 실패하고 기존 테스트는 계속 통과하는지 확인(이 저장소가 최근 네 회차에 써 온 인과 확인 방식).

- **위험과 피할 것**
  - `safeFilename`·`cutFilenameRunes`(export.go:308-341)를 건드리지 마세요. 절단 길이 100, 빈 제목 `muni-document`, 치환기는 2026-09-24 회차가 막 고친 자리입니다.
  - `internal/handoff/handoff.go:333` 의 **수신** 파서 `filenameOf` 는 손대지 마세요. 이 과제는 송신 헤더만 바꿉니다. 두 경로가 같은 입력을 같은 값으로 읽는지는 `handoff_live_test.go` 의 왕복으로 확인하세요.
  - `workspace_export.go:111` 은 `filename*` 파라미터 자체가 없고 `filename="…"` 안에 `safeFilename` 결과를 그대로 넣습니다(큰따옴표 미처리). 같은 증상군이지만 **이번 범위 밖** — 보류 아이디어로 남겨 두세요. 한 번에 다 고치면 회귀 범위가 넓어집니다.
  - 보호 경로(`auth.go`, `internal/database/migrations`, `.github/workflows`, `internal/settings` 의 AAD)는 이 과제에서 전혀 필요 없습니다.
  - 이 정찰 세션 환경에서는 `go` 명령 실행이 승인 대기로 막혀 **동적 재현을 하지 못했습니다.** 위 1)의 "고치기 전 실패" 를 구현자가 반드시 먼저 눈으로 확인하고 시작하세요. 근거는 코드 독해와, 저장소에 커밋되어 있는 2026-09-24 회차의 기록(`export_filename_live_test.go` 의 알려진-공백 주석)입니다.

- **차선 후보**: 워크스페이스 ZIP 한도 초과 안내를 실제로 넘쳤을 때만 넣기 (2/1/S) — `workspace_export.go:155` 의 `len(items)==maxWorkspaceExport` 가 정확히 2000건인 워크스페이스에도 '넘어 그만큼만 담았습니다' 를 적습니다. `:65` 의 `LIMIT` 을 `maxWorkspaceExport+1` 로 읽고 2001번째가 있을 때만 안내하며 목록은 2000건으로 자르면 됩니다. 1순위가 성립하지 않을 때(예: 기존 헤더가 이미 파싱된다면) 이것으로 갈 것.
