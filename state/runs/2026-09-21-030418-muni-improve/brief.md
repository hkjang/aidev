- 과제: 워크스페이스 ZIP의 목록.md와 동명 문서가 충돌하지 않게 하기 (가치 4 / 위험 1 / 작업량 S)
- 왜: `exportWorkspace`는 문서 이름만 `uniqueEntryName`으로 중복을 피하고 마지막 안내 파일은 `archive.Create("목록.md")`로 직접 쓰므로, Markdown 내보내기에서 루트의 「목록」 문서와 안내 파일에 같은 경로를 부여한다. 안내 파일 이름을 먼저 예약하면 압축 해제 도구가 어느 항목을 선택하든 문서 본문과 목록을 각각 보존할 수 있다.
- 수용 기준: 1) 기본 형식 및 `?format=md`로 내려받은 실제 ZIP에서 루트 「목록」 문서, 같은 제목의 두 번째 문서, 「목록 (2)」 문서와 안내 파일이 모두 고유한 경로를 가지며 각 본문이 보존되고 안내 파일은 정확히 하나의 `목록.md`이다. 2) 안내 파일이 나열하는 문서 경로가 실제 ZIP 항목과 일치하며, 하위 폴더의 `목록.md`와 HTML/TXT 문서명에는 불필요한 접미사를 붙이지 않는다. 3) PostgreSQL + 실제 등록된 HTTP 라우트를 통과한 회귀 테스트가 변경 전 중복 경로로 실패하고 변경 후 모든 항목의 이름·본문·안내 참조를 검증해야 한다. 헬퍼에 예약 map을 손으로 넣은 단위 테스트만으로는 완료하지 않는다.
- 건드릴 파일: `internal/httpapi/workspace_export.go:Server.exportWorkspace` — 안내 파일 경로를 지역 상수 하나로 정하고 문서 순회 전 `used`에 예약한 뒤 마지막 `archive.Create`도 같은 값을 사용한다; `internal/httpapi/workspace_export_test.go` — 기존 `uniqueEntryName` 테스트를 유지하고 필요한 경우만 보강한다; `internal/httpapi/workspace_export_live_test.go`(새 파일) — 아래 실제 라우트 ZIP 회귀 테스트. 기존 `server.go`, 인증, SQL 스키마는 변경하지 않는다.
- 검증 명령: 저장소 루트에서 `go test ./...`, `go vet ./...`, `gofmt -l internal/httpapi/workspace_export.go internal/httpapi/workspace_export_live_test.go`, `scripts/check-webui-placeholder.sh`. 회귀 테스트 이름을 `TestWorkspaceExportReservesManifestName`으로 두고 전용 테스트 PostgreSQL DSN을 설정한 환경에서 `go test ./internal/httpapi -run '^TestWorkspaceExportReservesManifestName$' -count=1 -v`, 이어 `go test ./...`를 실행한다. `MUNI_TEST_DSN` 미설정이면 live 테스트가 SKIP하므로 성공 근거로 삼지 않는다. 정찰에서는 `go test ./...`와 placeholder 검사를 실제 통과했으나 DSN 미설정으로 DB 검증은 미확인이다.
- 위험과 피할 것: 범위 밖은 ZIP 스트리밍·2,000건 제한·폴더 경로 계산·ACL·safeFilename 정규화·권한 정책·auth/migrations/workflows·가져오기 제목 규칙이다. 안내 파일을 없애거나 문서를 생략해서 해결하지 않는다. DB 테스트의 기존 공유 상태를 고려해 만든 문서/폴더만 cleanup하고 t.Parallel을 쓰지 않는다. 기존 문서 제목이 같으면 SQL 순서가 비결정적이므로 특정 본문에 반드시 `(2)`가 붙는다고 단정하지 말고 모든 본문 및 실제 이름의 일대일 대응을 확인한다. 2026-09-07 반려의 구체 접근은 미확인이며 재사용하지 않았다.
- 차선 후보: DOCX 가져오기에서 제목과 같은 첫 H1 중복 제거 (가치 3 / 위험 1 / 작업량 S) — 첫 과제가 구현 착수 코드에서 이미 해결돼 있을 때만. 현 기준에는 `titleInBody`가 아니라 `upload.markdown`이므로 먼저 현재 코드를 확인한다. `import_attachments.go:parseUpload/importDocument`와 `handoff.go:storeHandoff`의 필드 참조를 함께 일반화하고 DOCX에 기존 `dropLeadingTitle`만 적용; DOCX parser 자체와 끼워 넣기는 유지. `docx.Build` 실제 출력으로 POST /api/v1/import를 검증한다. 지난 HTML 변경 36dccc6은 현재 HEAD에 없으므로 이를 다시 구현하거나 임의 cherry-pick하지 않는다.

구현 순서와 점검 지점 (1~3단계 모두 done, 사람 승인 대기 없음):
1. `internal/httpapi/workspace_export_live_test.go`에 `newServerUnderTest`(provision_live_test.go), `adminWorkspace`(import_word_live_test.go), `ownedDocument`(links_live_test.go)를 재사용해 테스트를 작성한다. 고유 본문을 가진 루트 동명 문서들, 하위 폴더 문서를 만들고 인증된 `srv.admin.Get(srv.URL + "/api/v1/workspaces/" + workspaceID.String() + "/export.zip?format=md")` 응답을 `archive/zip.NewReader`로 읽는다. 파일명을 map으로 덮어쓰기 전에 ZIP의 모든 항목을 순회해 중복을 잡는다. 위 focused 명령으로 기존 결함 때문에 실패하는지 확인한 뒤에만 2단계로 간다(의도한 red 상태이므로 최종 제출하지 않는다).
2. `exportWorkspace`에서 안내 파일 이름 예약만 추가한다. focused 테스트가 통과하는지 확인한다. `used`가 문서보다 먼저 안내 이름을 점유하면 기존 접미사 탐색이 「목록 (2)」와의 연쇄 충돌도 피하므로 새 이름 생성기는 만들지 않는다.
3. 같은 테스트에서 기본 형식, md/html/txt, 하위 폴더 이름 보존, 안내 파일의 실제 경로 참조를 검증한다. 위 전체 검증 명령을 실행하고 brief의 단계 상태와 journal에 결과/DSN 사용 여부를 남긴다. 코드가 이 근거와 다르면 과제서를 수정한 뒤 진행한다.

확인한 근거와 한계:
- 기준 HEAD `1547faa`, VERSION `0.41.0`; `git log -30`, README, Makefile, go.mod, frontend/package.json, CI, ARCHITECTURE/OPERATIONS/USER_GUIDE/ADMIN_GUIDE 관련 절을 확인했다. 저장소 내부 CLAUDE.md/AGENTS.md/로드맵 파일 및 internal/frontend/src의 TODO/FIXME는 검색에서 발견하지 못했다.
- `server.go`에 `GET /api/v1/workspaces/{id}/export.zip` 등록을 확인했다. `workspace_export.go`의 `used := map[string]bool{}`, `uniqueEntryName(...)`, 마지막 고정 이름 생성이 근거다. 현재 테스트 네 개는 헬퍼만 검증하며 안내 파일과의 실제 충돌 테스트는 없다. 정찰은 코드를 추가하지 않아 실제 HTTP ZIP 재현은 미확인; 구현 1단계가 이를 증명한다.
- 현재 가이드는 ZIP 다운로드를 설명한다. 동작 내부의 이름 충돌 수정이므로 가이드/PDF/프런트 빌드는 필수 범위가 아니다.

대안 비교와 선택 근거:
- 선택: 고정 안내 이름을 먼저 예약. 기존 접미사 규칙 재사용, 문서 내용/폴더 계약 유지, 수정 표면 최소.
- 대안: 안내 파일도 마지막에 uniqueEntryName으로 이름 짓기. 중복은 피하지만 안내 위치가 문서에 따라 바뀌므로 고정 `목록.md`를 유지하는 선택보다 불리하다.
- 대안: 문서/메타데이터 디렉터리 분리. 장기 확장에는 유리하나 모든 ZIP 경로가 바뀌고 이번 범위를 넘는다.
- 보류: 문서에 동명 제목을 쓰지 말라고 안내만 하기. 코드 변경은 없지만 기존 문서를 안전하게 내보내는 결과를 충족하지 못한다.
- 가장 큰 가정: 안내 파일 경로를 예약해 문서 쪽에 기존 접미사를 붙이는 정책이 현 고정 안내 이름 계약과 맞는다. 별도 외부 소비자가 문서의 고정 파일명을 전제하는지는 미확인이나 기존에도 동명 문서는 접미사가 붙는다.

작업량 근거와 예비 시간:
- Bottom-up 기본 25~35분: 실서버 회귀 픽스처 12~17분, 예약 처리 3~5분, 경계 검증·전체 확인·기록 10~13분. 알려진 불확실성인 테스트 DB 준비/공유 상태 정리에 contingency 5~10분을 별도로 잡아 총 30~45분, 확신 중간(통계적 P80을 뜻하지 않음).
- 과거 Markdown/HTML 회차도 기존 규칙과 live 테스트를 확장한 작은 변경이라 S 분류의 유사 사례지만 실제 소요시간 자료는 없어 정량 교차 추정은 미확인이다. 전용 PostgreSQL을 사용할 수 있다는 가정이며 DB 준비가 길어지면 검증 완료를 주장하지 말고 추정/진행 상태를 갱신한다.
- 관리 예비비는 0분/승인된 추가 범위 없음. 새 아키텍처·문서 캠페인은 이 예비 시간으로 끼워 넣지 않는다. 분해·가정·불확실성 문서화 방식은 적용 스킬과 [GAO 추정 지침](https://www.gao.gov/products/gao-20-195g)을 참고했으며 분 단위 수치는 정찰의 판단이다.
- 적용 스킬: `pmo:estimating-and-contingency`, `technology:implementation-planning`, `technology:solution-exploration`. 전용 Skill 도구가 노출되지 않아 `/mnt/c/Users/USER/projects/headcount/plugins/{pmo,technology}/skills/`의 해당 SKILL.md를 직접 읽었다.

## 구현 단계 결과
- 1 done: PostgreSQL 16 전용 컨테이너, MUNI_TEST_DSN 설정. focused live 테스트가 수정 전 duplicate ZIP entry: "목록.md"로 실패.
- 2 done: 지역 상수 manifestName 예약 및 마지막 archive.Create에서 재사용. focused 통과. 수정만 되돌렸을 때 default/md 재실패, html/txt 통과; 수정 복원 후 모두 통과.
- 3 done: default/md/html/txt 네 하위 테스트에서 ZIP 중복·고유 본문·실제 목록 참조·폴더 이름·HTML/TXT 무접미사 검사. DSN을 설정한 go test ./... 통과(httpapi 7.222s), go vet ./...·gofmt -l 대상 두 파일·scripts/check-webui-placeholder.sh·git diff --check 모두 종료 0. 기존 단위 테스트 유지.
