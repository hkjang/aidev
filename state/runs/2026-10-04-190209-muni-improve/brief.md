- 과제: 워크스페이스 ZIP 항목 이름이 대소문자만 다를 때 압축 해제에서 한쪽이 덮이는 것 막기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `uniqueEntryName`(`internal/httpapi/workspace_export.go:281`)이 중복 추적 맵 `used` 를 **대소문자를 구분해서** 쓰기 때문에, 제목이 `Report` 와 `report` 인 문서 둘은 ZIP 안에 `Report.md`·`report.md` 두 항목으로 그대로 들어갑니다. 폐쇄망 배포 대상인 Windows(와 macOS 기본 설정)는 파일 이름 대소문자를 구분하지 않으므로 순서대로 쓰는 압축 해제 도구에서 뒤 항목이 앞 항목을 덮고 문서 하나가 조용히 사라집니다 — a8f56b6(`목록.md` 예약)이 막은 바로 그 문서 분실의 남은 자리이고, `uniqueEntryName` 의 주석이 스스로 "that is how an export quietly loses a document" 라고 쓴 사고입니다.
- 수용 기준:
  1) 제목이 `Report` 와 `report` 인 문서 둘이 있는 워크스페이스를 md 로 내보내면 ZIP 항목 이름이 대소문자를 무시해도 서로 다르다(예: `Report.md` 와 `report (2).md`). 두 문서 모두 자기 항목을 가진다(어느 쪽도 사라지지 않는다).
  2) 같은 규칙이 디렉터리에도 적용된다 — 폴더 이름이 `Report` 와 `report` 로 갈린 두 폴더 안에 같은 제목 문서가 있을 때도 항목 이름이 대소문자 무시로 유일하다.
  3) 기존 동작은 한 글자도 바뀌지 않는다: 완전히 같은 제목 둘 → `X.md`/`X (2).md`, 다른 디렉터리의 같은 제목 → 둘 다 접미사 없음, 빈 제목 → `제목 없는 문서`, 안내 파일 `목록.md` 예약. (`workspace_export_test.go:67,83,95,103` 네 단위 테스트가 지금 그대로 통과해야 한다.)
  4) 테스트가 증명할 것: 수정 전 실패 / 수정 후 통과 / 수정 한 줄만 되돌리면 그 테스트만 다시 실패(다른 243건은 계속 통과).
- 건드릴 파일:
  - `internal/httpapi/workspace_export.go:281 uniqueEntryName` — `used` 의 **키만** 대소문자 접기(`strings.ToLower(name)`)로 바꾸고, 반환하는 이름과 ZIP 에 실제로 쓰는 이름은 원래 대소문자를 그대로 유지할 것. 키는 `path.Join` 으로 만든 전체 경로를 접는다(디렉터리까지 함께 접혀 기준 2)가 따라온다).
  - 같은 파일 `:149` 의 안내 파일 예약 `used := map[string]bool{workspaceManifestName: true}` — 키를 같은 접기 함수로 통과시킬 것. `workspaceManifestName` 은 `목록.md` 라 `ToLower` 가 항등이지만, 예약과 조회가 다른 규칙을 쓰면 다음 사람이 그 이름을 ASCII 로 바꾸는 순간 조용히 깨진다. 작은 비공개 헬퍼(예: `entryKey(name string) string`)를 두고 두 자리가 같은 것을 쓰게 하는 쪽을 권한다. **`safeFilename`·`safeFolderSegment`·`workspaceManifestName` 의 값과 본문은 건드리지 말 것.**
  - 테스트: `internal/httpapi/workspace_export_live_test.go` 에 live 하나를 더하는 것이 이 저장소의 관례(최근 다섯 회차 전부 실제 라우트로 증명). 실제 라우트는 `GET /api/v1/workspaces/{id}/export.zip?format=md&trash=true` 이고 같은 파일의 기존 테스트들이 워크스페이스·폴더·문서를 만드는 방법을 그대로 보여 준다. `internal/httpapi/workspace_export_test.go` 에 `uniqueEntryName` 단위 표를 한 줄 더하는 것은 선택(불변 항목을 지키는 감시자로는 유용).
  - 프로덕션 파일 1개 + 테스트 1~2개. 그 이상으로 번지면 과제를 잘못 잡은 것이다.
- 검증 명령:
  - `go test -count=1 -run 'EntryName|Workspace' ./internal/httpapi` (단위 — DSN 없이 돈다)
  - live 포함: 전용 DB 를 띄우고 `MUNI_TEST_DSN=... go test -count=1 -v ./internal/httpapi` — 직전 회차 기준 PASS 243 / SKIP 0. **DSN 없이 통과한 것을 "통과" 로 적지 말 것**(live 가 SKIP 된다).
  - `go test -count=1 ./...`, `go vet ./...`, `gofmt -l .`, `scripts/check-webui-placeholder.sh`
  - 프런트 미변경이므로 npm 검사·`make build` 는 돌리지 않는다(`make build` 는 tracked `webui/dist/index.html` 을 덮는다).
- 위험과 피할 것:
  - **접기를 반환값에 적용하지 말 것.** `used` 의 키만 접는다. 반환 이름을 소문자로 만들면 `회의록.md` 는 그대로지만 `README.md` 같은 제목이 `readme.md` 로 바뀌어 사용자가 보는 파일 이름이 달라지고, `목록.md` 를 기대하는 안내 파일 계약도 흔들린다.
  - `uniqueEntryName` 은 이 패키지 안에서 워크스페이스 내보내기만 쓴다(`workspace_export.go:171` 한 자리 + 단위 테스트). `safeFilename` 과 달리 파급이 좁다 — 그 둘을 혼동해 `safeFilename` 을 건드리면 문서 내려받기·발표자료·첨부·handoff 다섯 경로가 함께 움직인다.
  - 보호 경로(`auth.go`, `internal/database/migrations`, `.github/workflows`, settings 봉인 AAD)는 이 과제와 무관하다 — 열지 말 것.
  - `ORDER BY d.folder_id NULLS FIRST, d.title` 에 동률 해소(`, d.id`)를 더하고 싶은 유혹이 있다. **이번에는 하지 말 것** — 별 과제이고, 섞으면 "어느 변경이 테스트를 바꿨는지" 가 흐려진다.
  - Chromium 이 없는 워크트리에서는 `TestDevtoolsPDFHasPageNumbers` 가 실패한다(환경 문제). 이 과제와 무관하니 먼저 확인할 것.
  - 미확인: 실제 Windows/macOS 에서 덮어쓰기가 일어나는 것을 이 환경에서 재현하지는 못했습니다(리눅스 FS 는 대소문자를 구분). 근거는 파일 시스템 계약이고, 테스트가 증명하는 것은 "ZIP 항목 이름이 대소문자 무시로 유일하다" 까지입니다 — 수용 기준도 그 선에서 쓰세요. 한글 제목이 대다수라 실제 빈도는 낮습니다(그래서 가치 3, 5 가 아님).
- 차선 후보: 워크스페이스 ZIP 정렬 동률에 문서 ID 순서 추가 (`workspace_export.go:71`, `ORDER BY d.folder_id NULLS FIRST, d.title` → `…, d.title, d.id`; `(2)` 번호와 2000건 절단 경계가 결정적이 된다, 2/1/S). 단 **지금 코드에서도 PostgreSQL 이 우연히 같은 순서를 돌려주면 수정 전 테스트가 실패하지 않는다** — 그 경우 이 저장소의 "먼저 실패시킨다" 관례를 만족시킬 수 없으니, 1순위가 성립하지 않을 때에도 재현을 먼저 만들어 보고 안 되면 과제를 바꾸세요.
