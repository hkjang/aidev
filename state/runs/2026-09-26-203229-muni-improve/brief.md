- 과제: 워크스페이스 ZIP 내려받기의 `Content-Disposition` 에도 `filename*` 을 주고 따옴표가 헤더를 깨지 않게 하기 (가치 3 / 위험 1 / 작업량 S)

- 왜: 내려받기 헤더를 내는 다섯 자리 가운데 `internal/httpapi/workspace_export.go:111` 만 지난 회차(299b4dc)의 `extValueEscape` 를 받지 못해, 사용자가 지은 워크스페이스 이름을 `filename="…"` 안에 원문 그대로 넣는다(`attachment; filename="`+filename+`.zip"`). 워크스페이스 이름은 80룬까지 아무 문자나 허용되므로(`workspaces.go:55` 의 검사는 공백 제거와 길이·slug 뿐) 이름에 큰따옴표가 하나 있으면 `filename="내 "특별" 공간-20260926.zip"` 가 되어 파서가 이름을 `내 ` 로 끊거나 헤더 전체를 거부하고, 한글 이름은 quoted-string 안의 raw UTF-8 바이트라 규격상 클라이언트가 latin-1 로 읽어 깨진 이름으로 저장된다. 고치면 다섯 경로가 같은 규칙 하나로 정리되고(문서·발표자료·첨부·넘겨받기는 이미 정리됨) 사람이 지은 이름이 그대로 저장된다.

- 수용 기준:
  1) `GET /api/v1/workspaces/{id}/export.zip?format=md` 의 `Content-Disposition` 이 `filename*=UTF-8''…` 을 가지고, `mime.ParseMediaType` 으로 읽은 `filename` 이 `safeFilename(워크스페이스 이름) + "-" + YYYYMMDD + ".zip"` 과 룬 단위로 같다 — 이름이 `[대외비] 연구 "특별"; 2026` 같이 큰따옴표·쌍반점·한글·공백을 모두 담고 있을 때도.
  2) 다른 경로와 같은 모양으로 ASCII fallback 인 `filename="…"` 이 남아 있고 그 값에는 attr-char 밖 문자·따옴표가 없다(권장: `filename="workspace-20060102.zip"` — 날짜만 ASCII 로 남기고 이름은 `filename*` 쪽이 전달). `safeFilename` 결과를 quoted-string 에 그대로 넣는 지금 방식은 버린다.
  3) 같은 응답의 ZIP 본문이 예전과 같다 — 항목 이름·`목록.md`·휴지통 분리가 변하지 않았음을 테스트가 실제 아카이브를 열어 확인한다(기존 `workspace_export_live_test.go:67 exportEntryNames` 가 이미 그 일을 하므로 그 테스트 넷이 손대지 않고 통과해야 한다).
  4) 고친 뒤 헤더 한 줄만 되돌리면 새 테스트만 다시 실패하고 기존 테스트는 계속 통과한다(인과 확인).

- 건드릴 파일:
  - `internal/httpapi/workspace_export.go:109-111` (`exportWorkspace`) — `filename` 을 만든 뒤 헤더를 `fmt.Sprintf(`attachment; filename="workspace-%s.zip"; filename*=UTF-8''%s.zip`, 날짜, extValueEscape(filename))` 꼴로 바꾼다. 날짜 문자열(`time.Now().Format("20060102")`)을 한 번만 계산해 두 자리에 쓰면 두 이름이 어긋나지 않는다. 주석은 이 저장소 관례대로 "왜" 를 산문으로 — 이미 `export.go:344` 의 `extValueEscape` 주석이 이유를 길게 적고 있으니 여기서는 "이 라우트만 남아 있었고 이름에 따옴표가 들어가면 파서가 끊었다" 만 짧게.
  - `internal/httpapi/content_disposition_live_test.go` — 새 live 테스트 하나. `newServerUnderTest` → `POST /api/v1/workspaces` (`srv.admin`, body `{"name":"…","slug":"quote-zip"}`, slug 은 소문자 3~48자) → `srv.admin.Get(srv.URL+"/api/v1/workspaces/"+id+"/export.zip?format=md")` → 본문을 `io.ReadAll` 로 비우고 `filenameFrom(t, disposition)`(같은 파일 24행의 헬퍼, 프로덕션 파서 `mime.ParseMediaType` 을 쓴다) 로 이름을 되읽는다. 기존 `awkwardTitle` 상수는 80룬 안이니 워크스페이스 이름으로 그대로 쓸 수 있다(=38룬).
  - `internal/httpapi/content_disposition_test.go` — 표에 워크스페이스 라우트의 헤더 모양 한 줄을 더해 단위에서도 왕복을 고정(선택. 위 live 가 이미 프로덕션 배선을 지나므로 없어도 수용 기준은 충족).
  - 프로덕션 파일은 하나. 나머지는 테스트다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `gofmt -l internal/httpapi` → 출력 없음
  - `go vet ./...`
  - postgres:16-alpine 컨테이너를 띄우고 `MUNI_TEST_DSN` 을 주어 `go test -count=1 ./internal/httpapi` (DSN 없으면 live 테스트가 SKIP 되고 "통과" 가 아니다 — SKIP 0 을 확인할 것), 이어서 `go test ./...`
  - `scripts/check-webui-placeholder.sh`
  - 프런트는 건드리지 않으니 `npm` 검사는 불필요.
  - **미확인**: 이번 정찰 샌드박스에서 `go` 실행이 승인 대기로 막혀 동적 재현을 못 했다. 위 두 증상(따옴표로 끊김 / 한글 raw 바이트)은 코드를 읽어 단정한 것이고 Go `mime.ParseMediaType` 이 quoted-string 안의 raw UTF-8 을 거부하는지 허용하는지는 실행으로 확인하지 않았다. 구현자는 **먼저 실패하는 테스트로 재현**하고, 만약 한글만으로는 파서가 통과한다면 따옴표·쌍반점 쪽 실패를 근거로 삼되 수용 기준 1·2 는 그대로 유지할 것.

- 위험과 피할 것:
  - `safeFilename`(export.go:308) 을 바꾸지 말 것 — 문서 내려받기·발표자료·ZIP 항목 이름·폴더 세그먼트 다섯 경로가 공유한다. 따옴표를 지우는 방식으로 고치면 ZIP 항목 이름까지 조용히 바뀌고 `목록.md` 의 기록과 항목 이름이 어긋날 수 있다. 이번 수정은 헤더 한 줄에서 끝난다.
  - `extValueEscape`·`cutFilenameRunes`·`uniqueEntryName`·`safeFolderSegment` 도 그대로. 다섯 자리를 헬퍼 하나로 모으는 리팩터(보류 아이디어)는 이번 범위가 아니다 — 이번엔 빠진 자리 하나만 같은 규칙에 올린다.
  - 헤더는 첫 문서가 렌더되기 전에 나가고(주석 106행) 그 뒤엔 오류를 보고할 곳이 없으니 헤더 계산에 새 실패 경로를 만들지 말 것(에러를 돌려주는 헬퍼 도입 금지).
  - 보호 경로(auth.go, internal/database/migrations, .github/workflows, settings 봉인)는 건드리지 않는다.
  - 증거로 grep 결과를 쓰지 말고 실제 라우트 응답 헤더를 읽을 것. 손으로 만든 `http.ResponseWriter` 대역이 아니라 `newServerUnderTest` 의 실제 서버로.

- 차선 후보: 워크스페이스 ZIP 한도 초과 안내를 실제로 넘쳤을 때만 넣기 (2/1/S) — `workspace_export.go:155` 의 `len(items) == maxWorkspaceExport` 가 정확히 2000건인 워크스페이스에도 "문서가 2000건을 넘어 그만큼만 담았습니다" 를 `목록.md` 에 적는다. `:65` 근처의 `LIMIT` 을 `maxWorkspaceExport+1` 로 읽고 2001번째 행이 있을 때만 안내하며 목록은 2000건으로 자르면 된다. 2000건을 만드는 live 테스트는 무겁기 때문에 상수를 테스트에서 낮출 수 없다면 단위로 판정 함수만 검사하는 쪽으로 쪼갤 것.
