- 과제: 같은 이름(또는 대소문자만 다른 이름)의 폴더 둘이 워크스페이스 ZIP 안에서 한 디렉터리로 합쳐지는 것 막기 (가치 3 / 위험 2 / 작업량 S)

- 왜: `folderPaths`(`internal/httpapi/workspace_export.go:200`)가 폴더를 디렉터리 경로 문자열로 펴면서 중복을 전혀 보지 않습니다 — 같은 부모 아래 이름이 같은 폴더 둘은 같은 경로 문자열(`기획`)을 받고, ZIP 안에서 한 디렉터리로 합쳐져 사용자가 워크스페이스에서 보던 폴더 구조가 뭉개집니다. 폴더 이름 중복은 막혀 있지 않아 실제 API 로 만들 수 있습니다(`createFolder`, `workspaces.go:109~` — `TrimSpace` 와 120룬 길이만 보고 형제 이름 중복 검사가 없는 것을 이번 정찰이 코드로 확인). 직전 회차(a55bf5f)가 **항목 이름**을 대소문자 무시로 유일하게 만들어 문서 분실은 막았지만, 디렉터리 쪽은 그대로여서 `Report/회의록.md` 와 `report/회의록 (2).md` 는 Windows·macOS 에서 여전히 **한 폴더**로 풀립니다 — 그 수정의 남은 절반입니다.

- 수용 기준:
  1) 같은 부모(루트 포함) 아래 이름이 완전히 같은 폴더 둘이 있으면 ZIP 항목의 디렉터리 부분이 서로 달라진다(예: `기획/…` 과 `기획 (2)/…`). 두 폴더의 문서가 같은 디렉터리 접두사를 공유하지 않는다.
  2) 이름이 대소문자만 다른 형제 폴더 둘(`Report`/`report`)도 디렉터리 부분이 대소문자를 무시해도 서로 다르다 — 즉 ZIP 안의 **어떤 두 디렉터리 경로도 `strings.ToLower` 후 같지 않다**.
  3) 바뀌지 않아야 하는 것: 중복이 없는 워크스페이스의 폴더 경로는 한 글자도 바뀌지 않는다(대소문자 보존 — `README` 폴더가 `readme` 가 되지 않는다), 중첩 폴더의 부모/자식 관계와 `휴지통/` 접두사(`:169`)도 그대로다. 테스트가 이 둘(변하는 것 / 변하지 않는 것)을 함께 증명해야 한다.
  4) 테스트는 실제 라우트를 지난다: `POST /api/v1/workspaces/{id}/folders`(`server.go:82` 에서 확인) 로 같은 이름 폴더 둘을 만들고 문서를 각각 넣은 뒤 `GET /api/v1/workspaces/{id}/export.zip?format=md` 의 ZIP 항목 이름을 읽는다. 수정을 되돌리면 그 테스트만 다시 실패해야 한다.
  5) 어느 폴더가 ` (2)` 를 받는지는 **고정하지 말 것** — 아래 "위험" 참조.

- 건드릴 파일 (프로덕션 1개):
  - `internal/httpapi/workspace_export.go:200 folderPaths` — `resolve` 가 `result := path.Join(prefix, safeFolderSegment(entry.name))` 로 경로를 만든 **직후**, 이미 다른 폴더가 claim 한 경로면 ` (2)`, ` (3)` … 접미사를 붙여 유일하게 만들고 claim 을 기록한다. claim 맵의 키는 같은 파일에 이미 있는 `entryKey`(`:289`, `strings.ToLower`)를 지나게 할 것 — 그래야 수용 기준 2 가 성립하고 직전 회차와 규칙이 하나로 유지된다. **반환·ZIP 에 쓰는 경로 문자열은 원래 대소문자를 그대로 둘 것**(키만 접는다 — a55bf5f 가 `uniqueEntryName` 에서 한 것과 같은 모양이고, `uniqueEntryName:304` 의 접미사 루프가 그대로 베낄 본보기다).
  - 같은 함수의 SQL(`:201`)에 `ORDER BY name, id` 를 더해 순회 출발 순서를 고정할 것. 지금은 `for id := range all` 맵 순회라 접미사가 매 요청마다 다른 폴더에 붙을 수 있다.
  - `internal/httpapi/workspace_export_live_test.go` — 새 live 테스트(수용 기준 4). 테스트 발판은 이미 다 있다: `folderNamed(t, srv, workspaceID, name, parent *uuid.UUID) uuid.UUID`(`:91`), `documentInFolder(t, srv, workspaceID, folder *uuid.UUID, title string, trashed bool)`(`:115`), `exportEntryNames(…) []string`(`:138`), `exportManifest(…) string`(`:296`) — 새 헬퍼를 만들 필요 없다(정찰이 서명을 직접 확인).
  - **같은 파일 `:271~291` 은 이 수정으로 반드시 깨진다 — 정찰이 코드를 읽어 확정했다.** 직전 회차의 `TestTwoTitlesDifferingOnlyInCaseStayTwoEntries`(`:212`)가 `Report`/`report` 폴더 쌍을 만들고 `:273` 에서 `strings.Contains(strings.ToLower(name), "report/")` 로 항목을 고르는데, 이 수정 뒤 한쪽은 `report (2)/회의록.md` 가 되어 그 필터에 걸리지 않아 `:277 len(inFolders) != 2` 의 `t.Fatalf` 가 터지고, `:282~287` 의 `HasPrefix(name, "report/")` 기대도 어긋난다. 그 블록을 갱신할 것: 필터를 `strings.HasPrefix(strings.ToLower(name), "report")` + `strings.Contains(name, "/")` 로 넓히고, 단정은 **(a) 두 디렉터리 접두사가 `ToLower` 후 서로 다르다, (b) 한쪽은 `Report` 로 다른 한쪽은 `report` 로 시작해 대소문자가 보존된다, (c) 디렉터리 접두사 중 정확히 하나가 ` (2)` 를 갖는다** 로 바꿀 것(어느 쪽인지는 고정하지 말 것). 같은 테스트의 `:180~210`·`:247~270`(문서 제목 쪽 보장)은 계속 참이어야 하므로 **건드리지 말 것** — 깨지면 수정이 디렉터리 밖으로 번진 신호다. 왜 기대값이 바뀌는지 주석으로 남길 것(이 저장소 관례).
  - `internal/httpapi/workspace_export_test.go` — 선택. 순수 함수 단위 감시자를 넣을 자리가 마땅치 않다(`folderPaths` 는 `*Server`·DB 를 받는 메서드라 단위로 부르려면 대역이 필요하고, 운영자 지침이 대역으로 증명하는 것을 금한다). 접미사 생성을 작은 순수 헬퍼로 떼어내면 단위로 볼 수 있지만, live 테스트가 프로덕션 배선을 지나므로 **없어도 수용 기준은 충족**이다. 이 파일에 있는 `safeFolderSegment` 표는 건드리지 말 것.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - 전용 postgres 를 띄워 `MUNI_TEST_DSN` 을 주고 `go test -count=1 -v ./internal/httpapi` — **이것 없이는 live 테스트가 SKIP 되므로 "통과" 라고 쓰지 말 것.** 직전 회차 기준 PASS 245 / SKIP 0.
    `docker run -d --rm -e POSTGRES_PASSWORD=muni -e POSTGRES_DB=muni -p 55432:5432 postgres:16-alpine` → `MUNI_TEST_DSN=postgres://postgres:muni@127.0.0.1:55432/muni?sslmode=disable`
  - `go test -count=1 ./...`, `go vet ./...`, `gofmt -l .`(출력 없어야 함), `scripts/check-webui-placeholder.sh`
  - 인과 확정: 접미사 블록(또는 `entryKey` 호출)만 되돌려 새 테스트만 다시 실패하고 나머지가 계속 통과하는 것을 확인할 것 — 최근 다섯 회차가 모두 이 형식을 지켰다.
  - 프런트 미변경이면 npm 검사는 불필요. **`make build` 는 tracked `webui/dist/index.html` 을 덮으므로 돌리지 말 것.**

- 위험과 피할 것:
  - **어느 폴더가 ` (2)` 를 받는지 테스트로 고정하지 말 것.** `ORDER BY name, id` 를 더해도 `resolve` 가 재귀라, 자식이 먼저 순회되면 그 부모가 형제보다 먼저 claim 할 수 있다(이름이 같은 형제 둘 중 하나가 자식을 가진 경우). 데이터가 같으면 결과는 같지만 "항상 먼저 만든 폴더가 원래 이름" 은 보장되지 않는다. 직전 회차도 같은 이유로 어느 쪽이 접미사를 받는지 고정하지 않았다 — 그 선례를 따를 것.
  - `safeFilename`(`export.go:308`)·`safeFolderSegment`(`:271`)·`uniqueEntryName`(`:295`)·`entryKey`(`:289`)·`workspaceManifestName` 예약(`:149`)·문서 `ORDER BY`(`:71`)·2000건 한도 판정은 **한 글자도 건드리지 말 것.** 특히 `safeFilename` 은 내려받기 다섯 경로가 공유한다(Windows 금지 문자 과제는 별건이고 위험 3 이다).
  - 빈 세그먼트는 신경 쓰지 않아도 된다 — `safeFilename` 은 빈 값에 `"muni-document"` 를 돌려주므로(`export.go:311`) `safeFolderSegment` 는 절대 `""` 가 아니고, 따라서 `result == prefix`(자식이 부모 디렉터리로 접히는 경우)는 생기지 않는다. 이번 정찰이 코드로 확인했다.
  - `depth > 32` 순환 방어(`:233`)와 "부모가 없거나 삭제된 폴더의 자식은 루트에 놓인다"(미발견 부모 → `resolve` 가 `""` 반환) 동작을 보존할 것. 재귀를 BFS 로 바꾸는 식의 재작성은 이 두 동작을 조용히 바꾼다 — `resolve` 안에 claim 만 끼워 넣는 최소 수정으로 갈 것.
  - 보호 경로(`auth.go`, `internal/database/migrations`, `.github/workflows`, settings 봉인)는 건드리지 않는다. 마이그레이션으로 폴더 이름에 유일 제약을 걸려는 시도는 **금지** — 기존 데이터가 깨지고 과제 범위를 벗어난다. 고치는 곳은 내보내기 경로 하나다.
  - 리눅스 파일 시스템은 대소문자를 구분하므로 **실제 Windows 덮어쓰기·폴더 병합은 이 워크트리에서 재현할 수 없다.** 수용 기준대로 "ZIP 안의 디렉터리 경로가 대소문자 무시로 유일하다" 까지만 증명하고, 그 이상을 증명했다고 쓰지 말 것.
  - Chromium 이 없는 워크트리면 `TestDevtoolsPDFHasPageNumbers` 가 실패한다 — 이 과제와 무관한 환경 문제다.

- 차선 후보: 워크스페이스 ZIP 문서 정렬 동률에 문서 ID 추가 (`workspace_export.go:71` 의 `ORDER BY d.folder_id NULLS FIRST, d.title` 에 `, d.id`) — 1순위가 성립하지 않을 때. 다만 **수정 전에 실패하는 테스트를 만들기 어렵다**(PostgreSQL 이 우연히 같은 순서를 돌려주면 재현되지 않음). 착수하려면 먼저 재현(같은 제목 문서 여러 건을 넣고 `(2)` 가 붙는 문서가 요청마다 달라지는 것, 또는 `EXPLAIN` 으로 정렬 키가 동률임을 보이는 것)을 만들 것. 1순위 과제에서 `folderPaths` 에 `ORDER BY` 를 더하는 것과 **같은 성격의 결함이지만 다른 쿼리**이므로, 1순위를 하면서 `:71` 을 같이 고치지는 말 것(회차 하나에 한 조각).
