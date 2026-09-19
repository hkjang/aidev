# fix-summary — PR #8 (693b733) CI 실패

- 실패 단계는 `test-build-scan` 의 **"Go vulnerability scan"(govulncheck)** 하나뿐이며 "Unit and database integration tests" 는 성공. 로컬 재현: `govulncheck ./...` → `GO-2026-6452 excelize v2.11.0, Fixed in: N/A, internal/store/seed.go:66 ExtractWorkbookDefaults → File.GetRows`. 이 PR 이 건드린 파일(mail 설정·가이드)과 무관.
- 원인: Go 취약점 DB 항목 GO-2026-6452(2026-09-16 게시)가 수정 버전 없이 등록됨. 실제로는 GHSA-fx5j-qcqg-grpf 가 `first_patched_version: 2.11.0` 이고, 수정 커밋 93f0b3c(2026-06-21)는 v2.11.0 태그의 조상이며(`compare v2.11.0...93f0b3c` = behind 2, ahead 0), 모듈 캐시의 v2.11.0 `cell.go:getValueFrom` 에 `xlsxSI < 0` 가드가 이미 있음 → **오탐**.
- 같은 이유로 2026-09-18 이후 저장소의 모든 브랜치(auto/2026-09-19-2333 등 3개 확인) CI 가 같은 단계에서 실패하고, main 의 마지막 성공은 DB 게시 직전인 2026-09-16 eb2a3b0.
- 고칠 수 없음: excelize 는 v2.11.0 이 최신 태그이고 DB 에 fixed 이벤트가 없어 어떤 버전(마스터 pseudo-version 포함)도 govulncheck 를 통과하지 못하며, GetRows/GetCellValue/Rows.Columns 등 셀 읽기 API 전부가 취약 심볼로 등록돼 호출을 피할 수도 없음. 남는 길은 워크플로 수정(금지) 또는 라이브러리 교체(이 변경과 무관한 새 작업)뿐이라 **커밋 없이 종료**.
- 그 외 검증은 로컬에서 모두 통과: `gofmt -l`(출력 없음), `go vet ./...`, `go test ./...`(전 패키지 ok), `web: tsc --noEmit · vitest 11 passed · vite build`. 조치 제안: vuln DB 항목 정정(https://github.com/golang/vulndb 이슈)을 기다리거나 사람이 CI 정책을 결정.
