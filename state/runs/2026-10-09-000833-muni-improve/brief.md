- 과제: 워크스페이스 ZIP의 루트 목록.md 폴더가 고정 안내 파일과 충돌하지 않게 하기 (가치 3 / 위험 1 / 작업량 S)
- 왜: exportWorkspace는 안내 파일 목록.md를 문서 이름 맵에 예약하지만 folderPaths의 claimed는 빈 맵이라 루트 목록.md 폴더가 같은 경로를 차지한다. 목록.md/하위.md와 목록.md 파일은 압축 해제 때 공존할 수 없으므로 폴더 이름 할당 시 안내 파일을 예약하여 안내와 문서를 함께 보존한다.
- 수용 기준:
  1) 실제 API로 만든 루트 폴더 목록.md의 문서는 목록.md (2)/하위.md로 내보내며 루트 안내 파일 목록.md는 정확히 하나 남는다(이 사례에 다른 접미사 폴더가 없을 때). 목록.MD도 기존 entryKey 규칙으로 충돌을 피하고 원래 대소문자를 유지한다. md/html/txt 모두 고정 안내 파일이 있으므로 같은 규칙을 적용한다.
  2) 충돌 폴더의 자식도 변경된 부모 경로 아래에 남는다. 충돌하지 않는 README 폴더와 README/목록.md 같은 중첩 폴더는 이름이 그대로다. 기존 문서 제목 목록 → 목록 (2).md 및 휴지통 분리 동작을 보존한다. 폴더가 여럿이면 모두 다른 경로이며 어느 사용자 폴더가 (2)를 먼저 받는지는 고정하지 않는다.
  3) 실제 GET /api/v1/workspaces/{id}/export.zip 회귀 테스트가 기존 코드에서 실패하고 수정 뒤 통과한다. ZIP 이름 집합의 중복만 보지 말고 목록.md 파일 경로가 어떤 문서 경로의 디렉터리 접두사도 아닌지 확인한다. 안내 본문에 최종 변경된 문서 경로가 기재되고 문서 본문도 존재해야 한다.
- 건드릴 파일:
  - internal/httpapi/workspace_export.go:folderPaths — claimed 초기값에 entryKey(workspaceManifestName)를 예약한다. 기존 resolve의 접미사 생성과 부모 재귀를 재사용하고 예약 이유를 짧게 설명한다. exportWorkspace의 used 예약은 유지한다. 프로덕션 변경은 이 1파일로 제한한다.
  - internal/httpapi/workspace_export_live_test.go — TestAFolderNamedLikeTheIndexKeepsItsDocuments라는 새 live 테스트(이름 제안). newServerUnderTest, folderNamed, documentInFolder 및 기존 TestADocumentTitledLikeTheIndexKeepsItsOwnEntry를 참고한다. exportEntryNames는 md+trash=true만 고정하므로 다른 형식 및 목록 본문 확인은 새 테스트 안에서 직접 GET/zip.NewReader를 쓴다. 공용 헬퍼 개편은 하지 않는다.
- 검증 명령:
  - 정찰에서 실제 실행: go test -count=1 -v ./internal/httpapi -run 'Test(AFolder|TwoDocumentsWithOneTitle|TheSameTitleInDifferentFolders|ADocumentWithNoTitle|TwoTitlesDifferingOnlyInCase|AnEntryAtTheTop|TwoFoldersWithOneName)' → PASS 7 / SKIP 2 / FAIL 0, exit 0. scripts/check-webui-placeholder.sh → exit 0.
  - 구현의 DB 전제: 전용 PostgreSQL에 MUNI_TEST_DSN을 설정한다. provision_live_test.go:liveServer는 테스트 계정 데이터를 삭제하므로 기존 운영·타 프로젝트 DB를 재사용하지 않는다. 저장소에 기재된 예: docker run -d --name muni-test -e POSTGRES_PASSWORD=muni -e POSTGRES_DB=muni -p 5433:5432 postgres:16-alpine; MUNI_TEST_DSN=postgres://postgres:muni@127.0.0.1:5433/muni?sslmode=disable (환경변수로 export하여 아래 모든 Go 명령에 전달). 컨테이너 이름·포트 충돌이면 별도 값을 사용한다. 정찰은 새 DB를 만들지 않았다.
  - 새 테스트: go test -count=1 -v ./internal/httpapi -run '^TestAFolderNamedLikeTheIndexKeepsItsDocuments$'
  - 관련 회귀: go test -count=1 -v ./internal/httpapi -run 'Test(AFolderNamedLikeTheIndex|ADocumentTitledLikeTheIndex|TwoFoldersWithOneName|TwoTitlesDifferingOnlyInCase)'
  - 최종: go test -count=1 ./...; go vet ./...; gofmt -l internal/httpapi/workspace_export.go internal/httpapi/workspace_export_live_test.go; scripts/check-webui-placeholder.sh; git diff --check. 정찰에서 전체 test/vet는 미실행. live SKIP을 검증 성공으로 기록하지 않는다.
- 위험과 피할 것: auth/migrations/workflows, SQL 정렬, safeFilename, safeFolderSegment, 문서명/폴더명 API 검증, 안내 파일 이름은 변경하지 않는다. 임의 문서와 폴더 경로 충돌(Report.md 대 Report.md/하위.md), 가상 휴지통과 사용자 휴지통 폴더 충돌은 별건으로 남긴다. DB 원본 폴더 이름은 변경하지 않고 ZIP에서만 접미사를 붙인다. folderPaths는 휴지통 접두사를 붙이기 전에 실행되므로 해당 루트 폴더의 휴지통 문서도 휴지통/목록.md (2)/…로 이동하는 것은 동일 폴더 대응을 유지하기 위한 허용 동작이다. 과거 교훈대로 map 순회로 SQL 순서를 버리지 말고, 새 테스트는 자기 문서·폴더 또는 새 워크스페이스 cleanup을 등록한다. documentInFolder와 folderNamed에는 이미 자기 ID cleanup이 있다. make build는 tracked placeholder를 덮어쓰므로 실행하지 않는다.
- 차선 후보: 운영 안내의 외부 PostgreSQL 백업·복구 명령 정정 (가치 3 / 위험 1 / 작업량 S) — 1순위가 실제 라우트에서 재현되지 않거나 이미 수정됐다면 docs/OPERATIONS.md의 백업·복구 절차만 README/compose.example.yaml의 외부 DB 계약에 맞춘다. 호스트의 PostgreSQL 클라이언트·명시적 접속 대상·전용 복원 DB를 전제로 쓰고 존재하지 않는 compose postgres 및 muni_postgres_data 삭제 명령을 없앤다. compose에 DB 서비스를 새로 넣는 것은 범위 밖이며 복구 명령 실행은 전용 임시 DB에서만 검증한다. 실제 복구 명령은 정찰 미검증.

근거와 미확인:
- 읽은 코드: workspace_export.go의 workspaceManifestName, exportWorkspace, folderPaths, entryKey, uniqueEntryName; workspaces.go:createFolder(공백/120룬/부모 검사, 목록.md 금지 없음); export.go:safeFilename(목록.md 보존). folderPaths의 예약은 현재 빈 맵이다.
- 합성 ZIP에 목록.md/하위.md를 먼저 쓰고 목록.md를 마지막에 쓰면 Python zipfile.extractall이 Linux에서도 IsADirectoryError를 내는 것을 실행 확인했다. 이는 실제 코드에서 예상되는 경로의 압축 해제 증명이며 HTTP 라우트 재현은 아니다. Windows/macOS 실제 압축 해제는 미확인이다.
- main@9d781ef는 이전 프로필과 같다. 10/08 성공 기록의 HWPX 수정 e503055는 현재 체크아웃에 반영되지 않았고 네 truncateRunes 호출이 남아 있지만 이미 수행된 과제라 재선택하지 않는다.

접근 선택:
- 선택: 고정 안내 파일 경로만 기존 claimed에 예약한다. 기존 접미사 규칙으로 처리하고 변경 범위가 가장 작다.
- 보류: 문서·폴더 전체의 경로 할당기를 통합하면 임의 파일/폴더 충돌까지 해결되지만 trash와 조상 경로를 포함한 테스트 범위가 커진다.
- 미선택: 폴더 생성 API에서 이름 금지는 기존 폴더를 해결하지 못하며 허용 입력 계약을 바꾼다. 현상 유지는 안내 파일 또는 폴더 추출 실패를 남긴다.
- 핵심 가정: 고정 루트 안내 파일 이름을 유지하고 충돌한 사용자 폴더의 ZIP 경로에만 기존 접미사를 붙이는 것이 현재 내보내기 정책과 일치한다. 실데이터 빈도는 미확인이다.

구현 순서와 체크포인트(현재 모두 미착수, 사람 승인 대기 없음):
1. 전용 DB에서 새 live 테스트를 작성해 위 새 테스트 명령으로 기존 코드의 예상 실패를 확인한다. 실패 지점은 컴파일/환경 오류가 아니라 목록.md/… 충돌이어야 한다. 자동 체크포인트: 재현 불가면 근거를 갱신하고 차선 판단; 이 실패 상태는 로컬 검증용이고 커밋하지 않는다.
2. folderPaths 예약 1곳을 변경하고 새 테스트 및 관련 회귀 명령을 실행한다. 자동 체크포인트: 수용 기준 전체가 통과하고 기존 폴더 불변 단정이 유지되어야 다음 단계로 간다.
3. 최종 검증 명령을 실행하고 diff에서 프로덕션 1파일·테스트 1파일 범위를 확인한다. 자동 체크포인트: 예상 밖 실패는 이번 변경과 기존 문제를 구분해 기록하고 범위를 임의 확대하지 않는다.

추정 근거:
- 작업 분해에 따른 순수 작업 25~35분: 전용 DB 준비 및 실패 재현 10~15분, 예약 수정·감시자 확인 5분, 전체 검증·기록 10~15분. 알려진 변동(DB 준비/테스트 조정)에 별도 예비 5~10분을 두어 총 30~45분으로 예상한다. 중간 확신의 판단 범위이며 통계적 신뢰구간이나 보장 시간이 아니다.
- 비교 근거는 제공된 10/04~06의 동일 파일·동일 live 헬퍼를 쓰는 소규모 ZIP 개선이다. 당시 소요 시간 자료가 없으므로 시간 단가나 성공 확률을 만들어 내지 않고 범위 유사성만 교차 확인했다. 의존성 다운로드·DB 환경 수리·새 경로 할당기 개발은 제외한다. 관리 예비는 이 과제에 배정하지 않으며 새로운 범위는 보류 아이디어로 넘긴다.
- 추정 절차의 근거: [GAO Cost Estimating and Assessment Guide](https://www.gao.gov/products/gao-20-195g)의 범위·작업 분해·가정·위험·검증 기록 원칙을 적용했다. 분 단위 수치는 정찰의 판단이며 GAO의 수치가 아니다.
