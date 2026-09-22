# 과제서 — 2026-09-22 (base main@a7ef69e, VERSION 0.42.0)

- 과제: 워크스페이스 ZIP 에서 폴더 이름 `.`·`..` 가 압축 파일 경로를 밖으로 끌고 나가지 못하게 하기 (가치 4 / 위험 1 / 작업량 S)

- 왜: `createFolder`(`internal/httpapi/workspaces.go:108`)는 공백만 아니고 120자 이하면 어떤 이름이든 받아서 `..` 라는 폴더를 만들 수 있는데, 워크스페이스 내보내기의 `folderPaths`(`internal/httpapi/workspace_export.go:195` 의 `resolve`)는 그 이름을 `safeFilename` 에만 통과시키고 — `safeFilename`(`internal/httpapi/export.go:307`)은 `/`·`\`·개행·NUL 만 지우고 `.`·`..` 는 그대로 돌려줍니다 — `path.Join(prefix, ...)` 으로 이어 붙입니다. 그래서 워크스페이스 관리자가 받는 ZIP 안에 `../문서.md` 같은 항목이 들어가고(순진한 압축 풀기 도구는 대상 폴더 밖에 씁니다), 같은 이유로 휴지통 문서가 `path.Join("휴지통", "..")` = `.` 로 접혀 살아 있는 문서 옆에 섞입니다. 폴더 이름 한 칸을 경로 요소로 안전하게 만들면 두 문제가 같이 사라집니다.

- 수용 기준:
  1) 이름이 `..` 인 폴더(및 그 아래 하위 폴더)의 문서가 ZIP 안에서 `..` 로 시작하거나 `..` 요소를 포함하는 경로를 갖지 않는다 — 모든 항목 이름이 아카이브 루트 안쪽에 머문다.
  2) 이름이 `.` 인 폴더가 부모 폴더로 접히지 않는다 — 서로 다른 두 폴더가 같은 디렉터리 경로를 갖지 않는다.
  3) 휴지통 문서는 폴더 이름이 무엇이든 `휴지통/` 아래에 남는다.
  4) 평범한 폴더 이름(한글·공백·`/` 가 섞인 이름 등)의 경로는 지금과 **한 바이트도** 달라지지 않는다. 기존 `workspace_export_test.go` 의 네 테스트가 그대로 통과한다.
  5) 테스트가 증명할 것: 새 헬퍼에 대해 `.`·`..`·`./..`·`..` 가 `/` 치환으로 생겨난 경우(예: 이름 `a/..` → `safeFilename` 이 `a-..` 로 만들므로 안전)까지 표-주도로 확인하고, 그 결과를 `path.Join` 으로 이어 붙여도 `..` 요소가 남지 않음을 확인한다.

- 건드릴 파일:
  - `internal/httpapi/workspace_export.go` — 새 헬퍼(예: `safeFolderSegment(name string) string`)를 `uniqueEntryName` 옆에 두고, `folderPaths` 의 `resolve` 안 `path.Join(prefix, safeFilename(entry.name))` 을 `path.Join(prefix, safeFolderSegment(entry.name))` 으로 바꾼다. 헬퍼는 `safeFilename` 을 먼저 통과시킨 뒤 결과가 `.` 또는 `..` 일 때만 앞에 `_` 를 붙이는(`_.` / `_..`) 최소 규칙으로 — 다른 이름은 손대지 않는다.
  - `internal/httpapi/workspace_export_test.go` — 표-주도 단위 테스트 하나(입력 → 기대 세그먼트)와, 세그먼트를 `path.Join("휴지통", seg)` / `path.Join("", seg)` 로 이어 붙였을 때 `..` 요소가 없음을 확인하는 테스트 하나. 기존 네 테스트는 손대지 않는다.
  - (선택) 같은 파일 또는 live 테스트 파일에 인증 HTTP 로 `..` 폴더를 만들고 `GET /api/v1/workspaces/{id}/export?format=md` 의 ZIP 항목 이름을 읽는 테스트. `MUNI_TEST_DSN` 이 있을 때만 도는 기존 `liveServer` 틀(`internal/httpapi/handoff_live_test.go` 참고)을 쓴다. DSN 이 없으면 단위 테스트만으로 수용 기준 1·2·4를 증명한다.

- 검증 명령:
  - `go test ./internal/httpapi/` — DSN 없이도 도는 단위 테스트. **고치기 전에 새 테스트가 실패하는 것을 먼저 확인할 것.**
  - `go test ./...`, `go vet ./...`, `gofmt -l internal/httpapi`
  - `scripts/check-webui-placeholder.sh`
  - live 까지 보려면 postgres:16-alpine 컨테이너를 띄워 `MUNI_TEST_DSN` 을 주고 `go test -count=1 ./internal/httpapi/`(SKIP 0 확인).

- 위험과 피할 것:
  - `safeFilename` **자체를 바꾸지 말 것**. 이 함수는 문서 내려받기의 `Content-Disposition` 파일 이름과 여러 내보내기 경로가 함께 쓰므로, 여기서 `.`/`..` 규칙을 넣으면 관계없는 테스트와 사용자에게 보이는 파일 이름이 같이 바뀝니다. 새 헬퍼는 워크스페이스 ZIP 의 **폴더 세그먼트**에만 씁니다.
  - 문서 제목 쪽(`uniqueEntryName` 에 넘기는 `safeFilename(item.title)`)은 건드리지 마세요 — 제목이 `..` 여도 항목 이름은 `...md` 라 경로를 벗어나지 않습니다(확인함). 범위를 넓히면 지난 회차의 `목록.md` 수정과 겹칩니다.
  - `createFolder`/`updateFolder` 에서 이름을 **거부하지 마세요**. API 계약 변경이고, 이미 DB 에 있는 행은 고쳐지지 않아 내보내기는 여전히 깨집니다.
  - 보호 경로(`auth.go`, `internal/database/migrations`, `.github/workflows`, 설정 봉인)는 손대지 않습니다.
  - 미확인: `updateFolder` 의 이름 검증을 눈으로 확인하지 못했습니다(`createFolder` 만 읽음). 어느 쪽이든 이번 수정은 내보내기 쪽이므로 영향 없습니다.
  - 미확인: `..` 폴더를 실제로 만들어 ZIP 을 받아 보는 동적 재현은 이번 정찰에서 하지 않았습니다(코드 경로만 읽었습니다). 구현자는 새 테스트가 **고치기 전에 실패**하는 것으로 인과를 먼저 고정하세요.

- 차선 후보: 워크스페이스 ZIP 한도 초과 안내를 실제로 넘쳤을 때만 넣기 — `workspace_export.go:155` 의 `len(items) == maxWorkspaceExport` 가 정확히 2000건인 워크스페이스에도 "넘어서 그만큼만 담았습니다" 를 적습니다. `LIMIT $3` 을 `maxWorkspaceExport+1` 로 읽고 2001번째가 있을 때만 안내하며 2000건으로 자르면 됩니다(가치 2 / 위험 1 / 작업량 S).

## 견적 근거 (basis of estimate)
- 분해: 헬퍼 + 호출 한 줄(≈10 LOC) / 표-주도 단위 테스트 2개(≈45 LOC) / 선택적 live 테스트(≈60 LOC, DSN 필요).
- 방법: 유사 추정 — 같은 파일·같은 크기의 지난 회차(목록.md 충돌, 마크다운 표 파이프)가 한 세션 안에 끝났습니다.
- 범위: 25~45분(10번 중 8번). live 테스트까지 포함하면 컨테이너 기동 때문에 위쪽. 포함하지 않는 것: 폴더 이름 입력 검증, `safeFilename` 정리, 프런트, 문서.
- 가장 크게 기대고 있는 가정: `folderPaths` 의 `resolve` 가 ZIP 폴더 경로를 만드는 **유일한** 자리라는 것 — `s.folderPaths` 의 Go 호출자는 `workspace_export.go:100` 하나뿐임을 확인했습니다(`frontend/src/features/editor/folderTree.ts` 의 같은 이름 함수는 화면 표시용이라 무관).
