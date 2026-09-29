## 2026-09-29
- 선택: 워크스페이스 ZIP 한도 초과 안내를 실제로 넘쳤을 때만 넣기 (가치 2 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 최대 2000행만 읽은 뒤 2000건이면 초과라고 안내하던 원인을 실제 PostgreSQL과 HTTP ZIP 응답으로 확인했습니다. LIMIT을 2001로 늘려 초과 여부를 판정하고 목록·렌더링·감사 기록 전에 2000건으로 잘랐으며, 새 live 테스트가 1999/2000/2001건의 안내 여부·ZIP 항목 수·목록 일치를 검증하고 수정 복원 시 2000건에서 다시 실패했습니다. 전용 postgres:16-alpine으로 go test -count=1 -json ./...(httpapi 239 PASS, SKIP 0; 전체 외부 코퍼스/수동 출력 테스트 4 SKIP), make test(프런트 42파일/297 PASS), go vet ./..., gofmt, git diff --check, npm run build, CGO_ENABLED=0 go build -trimpath 모두 통과했으며 생성 번들은 제거하고 placeholder를 복원했습니다. 커밋 86476c4; 프로덕션 1파일·테스트 1파일.
- 실패 재현: `workspace_export_live_test.go:71: omission warning = true for 2000 documents, want false` / `--- FAIL: TestAWorkspaceExportWarnsOnlyWhenDocumentsAreOmitted/2000 (0.46s)`
- 보류 아이디어:
  - Content-Disposition 공통 헬퍼 및 첨부 fallback 확장자 — 2/1/S, 현재 이름 전달 계약 유지하며 후속 검토.
  - 가져오기·넘겨받기 제목의 AI 안내 문구 — 2/2/S, 두 경로의 241룬 실제 입력 검증부터.
  - ZIP 정렬 동률에 문서 ID 순서 추가 — 2/1/S, 신규 후보이며 동적 재현 필요.
  - 운영 안내의 외부 PostgreSQL 백업·복구 명령 정정 — 3/1/S, 신규 후보이며 배포 계약 확인 필요.
