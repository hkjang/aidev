# 회차 노트 2026-09-22-220439-muni-improve — muni
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 22:04] base pinned — main@a7ef69e
- [러너 22:04] autonomy release — 

## 정찰 노트
- 보류 목록이 네 회차째 "가져오기 제목 중복" 계열만 돌고 있어 이번엔 새로 읽었고, `folderPaths.resolve` 가 폴더 이름을 `safeFilename` 만 거쳐 `path.Join` 하는 자리를 찾았다 — `..` 폴더 하나로 ZIP 항목이 아카이브 밖을 가리키고 휴지통 격리도 무너지므로, 남은 후보(한도 안내 경계 2/1/S, PDF 첫 줄 2/2/S 근거 부족, e2e CI 4/2/M 보호 경로)보다 가치·위험 대비가 낫다.
- 추측으로 적은 것: `..` 폴더를 실제로 만들어 ZIP 을 받아 보는 동적 재현은 하지 않았다(코드 경로와 `path.Join` 의 Clean 동작에 근거). `updateFolder` 의 이름 검증도 눈으로 보지 않았다 — 다만 수정은 내보내기 쪽이라 영향 없음.
- 구현자가 조심할 것: `safeFilename` 은 절대 바꾸지 말 것(문서 내려받기 파일 이름을 함께 씀). 문서 제목 쪽은 `..` 여도 `...md` 라 안전하니 범위를 넓히지 말 것. 새 테스트가 고치기 전에 실패하는 것을 먼저 확인해 인과를 고정할 것.
- 기준선: DSN 없는 `go test ./internal/httpapi/` 통과(live SKIP). 저장소 코드 수정·커밋 없음.
- [러너 22:08] scout done — 워크스페이스 ZIP 에서 폴더 이름 `.`·`..` 가 압축 파일 경로를 밖으로 끌고 나가지 못하게 하기 (가치 4 / �

## 구현 노트
- `folderPaths` 의 `resolve` 가 폴더 이름을 `safeFilename` 에만 통과시켜 `path.Join` 하던 자리에 폴더 세그먼트 전용 `safeFolderSegment` 을 두었다 — `safeFilename` 결과가 정확히 `.`·`..` 일 때만 앞에 `_`. `safeFilename` 과 문서 제목 쪽은 건드리지 않았다.
- 확신하는 것: 실제 라우트 `GET /api/v1/workspaces/{id}/export.zip` 의 live 테스트가 고치기 전 `entry "../밖으로.md" points outside the archive` 와 `a trashed document left 휴지통: ""` 로 실패하고 고친 뒤 통과, 헬퍼 본문만 되돌리면 다시 실패한다(인과 확정). `go test ./...`·`go vet`·`gofmt`·placeholder 통과, httpapi SKIP 0 / PASS 209.
- 확신 없는 것: `_..` 라는 폴더 이름이 화면에 어떻게 보이는지는 확인하지 않았다(내보낸 ZIP 안에서만 쓰이고 DB·UI 이름은 그대로다). 프런트는 전혀 돌리지 않았다(Go 만 바뀜).
- 일부러 하지 않은 것: `createFolder`/`updateFolder` 의 이름 거부(API 계약 변경이고 이미 DB 에 있는 행은 안 고쳐짐), `safeFilename` 정리, 과제서의 차선 후보(한도 안내 경계).
- 새로 본 것(미수정, ideas 에 기록): `safeFilename` 이 `ai.go` 의 `truncateRunes` 를 재사용해 **100자 넘는 제목의 파일 이름에 줄바꿈 + `[…문서 컨텍스트가 길어 일부 생략됨…]` 이 붙는다** — 코드 읽기로만 확인했고 실제 내려받기 이름은 보지 않았다.
- 다음 역할이 조심할 것: 새 `workspace_export_live_test.go` 는 `MUNI_TEST_DSN` 이 있어야 돈다(없으면 SKIP, CI 에서는 Fatal). 이 테스트는 admin 워크스페이스에 폴더·문서를 만들고 `t.Cleanup` 으로 지운다 — `-count=2` 로 두 번 돌려 남는 행이 0 인 것을 확인했다.
- [러너 22:16] brief accepted — 채택 — 근거(createFolder 가 `..` 를 받음, `safeFilename` 이 `.`·`..` 를 그대로 돌려줌, `folderPaths` 의 Go 호출자가 exportWorkspace 하�
- [러너 22:17] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 도커로 postgres 를 띄워 live 테스트를 직접 돌렸다: 고친 코드는 통과, 스크래치 사본에서 `safeFolderSegment` 본문만 되돌리면 `../경로탈출 확인.md` 등 6건으로 실패 — 인과 확정. `go test ./...` 전부 통과(SKIP 0), `-count=2` 후 잔여 행 0, gofmt·vet 무출력.
- `createFolder`(workspaces.go:109)가 `..` 를 실제로 받는 것을 읽어 공격 경로가 실존했음을 확인했다. `safeFolderSegment` → `folderPaths` → `exportWorkspace` 외 호출자가 없어 DB·UI 폴더 이름에는 닿지 않는다(구현자가 확신 없다고 적은 `_..` 표시 우려는 해당 없음).
- 못 본 것: 프런트엔드(변경 없음), Playwright, 실제 압축 해제 도구의 동작. 차단 사유 없음 — 보안·법무 모두 blocking 없음, risk low.
- 남는 우려(범위 밖, 다음 회차 후보): ① workspace_export.go:111 이 `safeFilename(workspaceName)` 을 Content-Disposition 따옴표 안에 그대로 넣는데 `"` 를 지우지 않는다(export.go:60 은 urlPathEscape 를 씀). ② safeFilename 의 truncateRunes 가 100룬 초과 이름에 줄바꿈+생략 문구를 붙인다.
- 릴리즈가 볼 것: 워크트리의 webui/dist/index.html 이 커밋되지 않은 실제 빌드 산출물이다. 스테이징에 딸려 들어가면 check-webui-placeholder.sh 로 CI 가 깨진다.
- [러너 22:22] review approved — 리뷰 승인 (risk=low)
- [러너 22:22] pr created — https://github.com/hkjang/muni/pull/24
- [러너 22:28] ci passed — 검사 2개 모두 success
- [러너 22:28] merge done — 6f3bd8a
- [러너 22:41] release published — v0.43.0
