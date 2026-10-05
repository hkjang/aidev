- 과제: 워크스페이스 ZIP 문서 쿼리의 정렬 동률을 문서 ID 로 깨서 같은 이름 문서가 어느 파일에 담길지 고정하기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `internal/httpapi/workspace_export.go:71` 의 `ORDER BY d.folder_id NULLS FIRST, d.title` 은 같은 폴더에 제목이 같은 문서가 둘 이상이면 동률이고 PostgreSQL 은 동률 행의 순서를 보장하지 않는다. 그리고 이 동률은 예외가 아니라 평범한 상태다 — `documents.go:124` 가 제목을 비워 만든 문서 전부에 같은 제목 `제목 없는 문서` 를 준다(프런트 `NewDocumentDialog.tsx:106` 도 같은 값을 보낸다). 그 순서가 `uniqueEntryName` 의 `(2)` 접미사가 어느 문서에 붙는지와 2000건 절단에서 어느 문서가 빠지는지를 정하므로, **바뀌지 않은 워크스페이스를 두 번 내려받으면 `제목 없는 문서 (2).md` 안의 내용과 `목록.md` 가 적은 작성자가 서로 바뀔 수 있다**. `, d.id` 한 토큰으로 동률이 사라진다.
- 수용 기준:
  1) 쿼리가 `ORDER BY d.folder_id NULLS FIRST, d.title, d.id` 가 되고, 같은 제목 문서 여러 건이 있는 워크스페이스를 거듭 내려받아도 **항목 이름 → 본문(그리고 `목록.md` 의 이름 → 작성자 줄) 대응이 매번 같다**.
  2) 테스트는 이름 *집합* 이 아니라 **대응**을 본다. 2026-10-05 회차가 폴더 쪽에서 쓴 프로브는 항목 이름 집합을 비교했는데, 이 결함은 집합을 바꾸지 않는다(`제목 없는 문서.md` 와 `제목 없는 문서 (2).md` 는 어느 순서에서도 둘 다 나온다). 집합만 비교하면 테스트가 통과해 버린다 — 반드시 본문이나 작성자까지 읽을 것.
  3) 기존 live 246건 + 단위 테스트가 그대로 통과하고, 제목이 서로 다른 워크스페이스의 ZIP 항목 순서·이름은 한 글자도 바뀌지 않는다(`workspace_export_test.go` 의 기존 단정들이 감시자).
- 건드릴 파일 (프로덕션 1개):
  - `internal/httpapi/workspace_export.go:71` `exportWorkspace` 의 `SELECT … ORDER BY` — `, d.id` 추가. 이 저장소 관례대로 "왜" 를 산문 주석으로 붙일 것(기본 제목이 전부 같아 동률이 상례라는 점, 접미사와 2000건 경계가 그 순서에 달려 있다는 점).
  - `internal/httpapi/workspace_export_live_test.go` — 새 테스트 하나. 기존 테스트 헬퍼(`liveServer`, 워크스페이스·폴더·문서 생성, ZIP 읽기)는 `TestTwoFoldersWithOneNameStayTwoDirectories`(`:317`)와 `TestTwoTitlesDifferingOnlyInCaseStayTwoEntries`(`:212`)에 이미 있으니 그대로 재사용.
- 수정 전 실패를 만드는 방법 (**미확인 — 실행해 보지 않았다**):
  - 1순위: 같은 폴더에 제목을 비운 문서 셋을 만들고 본문을 서로 다르게 둔 뒤, `GET /api/v1/workspaces/{id}/export.zip?format=md` 로 한 번 내려받아 이름→본문 대응을 기록한다. 그다음 그중 한 문서를 **제목을 바꾸지 않고** 본문만 PATCH 해서(행이 다시 써지며 힙 순서가 바뀐다) 다시 내려받고 대응을 비교한다. 동률을 깨지 않은 상태에서는 정렬 입력 순서가 바뀌어 접미사가 다른 문서로 옮겨갈 수 있다.
  - 2순위: 같은 요청을 열두 번 반복해 대응을 비교(폴더 쪽에서 통했던 형식).
  - **둘 다 재현되지 않을 수 있다.** 문서 쪽은 폴더 쪽과 달리 Go 맵을 거치지 않고 PostgreSQL 이 직접 정렬하므로, 같은 계획·같은 힙 순서면 같은 결과를 돌려준다. 재현이 안 되면 거짓 통과를 만들지 말고 (a) 새 테스트를 "고친 뒤 대응이 항상 같다" 는 회귀 감시자로만 남기고 (b) `EXPLAIN (ANALYZE) …` 출력으로 정렬 키가 `d.folder_id, d.title` 둘뿐이라 동률이 남는다는 것을 회차 노트에 근거로 적을 것. 그래도 통과하지 않으면 차선 후보로 갈 것.
- 검증 명령 (전용 postgres:16-alpine 에 `MUNI_TEST_DSN` 을 주고 — 없으면 live 가 SKIP 되고 그것은 통과가 아니다):
  - `MUNI_TEST_DSN=... go test -count=1 -v ./internal/httpapi` (직전 기준 PASS 246 / SKIP 0)
  - `MUNI_TEST_DSN=... go test -count=1 ./...`, `go vet ./...`, `gofmt -l .`, `scripts/check-webui-placeholder.sh`
  - 프런트 미변경이므로 npm 검사·`make build` 는 돌리지 않을 것(`make build` 는 tracked `webui/dist/index.html` 을 덮는다).
- 위험과 피할 것:
  - `folderPaths`·`uniqueEntryName`·`entryKey`·`safeFilename`·`safeFolderSegment`·`workspaceManifestName` 예약은 **한 글자도 건드리지 말 것**. 최근 세 회차(86476c4·a55bf5f·17a7cd6)가 모두 이 함수들을 고쳤고 지금 상태가 맞다. 이번 과제는 SQL 한 줄이다.
  - `LIMIT maxWorkspaceExport+1` 과 `truncated` 판정 로직(`:105~111`)도 그대로 둘 것 — 86476c4 가 고친 자리다.
  - `folderPaths` 의 `ORDER BY name, id` 는 이미 있고 순회 순서는 `order` 슬라이스로 고정돼 있다. 같은 수정을 또 하지 말 것.
  - 2026-10-05 교훈: SQL `ORDER BY` 만으로는 Go 쪽이 순서를 버리면 효과가 없다. 여기는 `items` 슬라이스에 읽은 순서로 쌓고(`:88~99`) 그 슬라이스를 순회하므로(`:154`) 맵을 거치지 않는다 — **확인함**. 추가로 손볼 곳은 없다.
  - 보호 경로(`auth.go`, `internal/database/migrations`, `.github/workflows`)는 건드리지 않는다. 마이그레이션으로 제목 유일 제약을 거는 것은 금지(기존 데이터를 깬다).
- 차선 후보: 운영 안내의 외부 PostgreSQL 백업·복구 명령 정정 — `compose.example.yaml` 은 muni 서비스만 포함하는데 `docs/OPERATIONS` 의 백업/복구는 postgres 서비스와 postgres_data 볼륨을 가정한다(가치 3 / 위험 1 / 작업량 S). 테스트로 증명할 것이 없어 '먼저 실패시킨다' 관례와 맞지 않으므로, 고를 경우 배포 계약(외부 DB 전제)을 `deploy/`·`Dockerfile`·README 에서 먼저 확인하고 정본 하나만 남길 것.
