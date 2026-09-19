# 수리 요약 — PR #9 (6424eb1) CI 실패

- 실패 자리: run 35436507400 · job test-build-scan · step 7 `Go vulnerability scan` 만 failure(테스트·vet 은 success, 이후 단계 skipped). 로컬 재현 `govulncheck ./...` exit 3: `GO-2026-6452 … Found in excelize/v2@v2.11.1-0.20260918021423-0434413565bf … Fixed in: N/A … seed.go:66 GetRows`.
- 원인은 저장소가 아니라 vuln.go.dev 의 엔트리 결함: GO-2026-6452 의 SEMVER range 에 `fixed` 이벤트가 없어(GHSA-fx5j-qcqg-grpf 는 first_patched 2.11.0, 픽스 커밋 93f0b3c 는 v2.11.0 에 포함) 어떤 버전을 핀해도 걸린다. golang/vulndb#6510·#6532 아직 open(2026-09-19 갱신), DB 는 09-16T18:00Z 그대로.
- 증명: vuln.go.dev 엔트리를 그대로 복사한 로컬 DB(`govulncheck -db file://…`)는 같은 exit 3, 여기에 `{"fixed":"2.11.0"}` 한 줄만 더한 DB 는 `No vulnerabilities found` exit 0 — 이 브랜치의 코드·핀은 DB 정정만 되면 그대로 초록.
- 저장소 안에서 초록으로 만드는 길은 워크플로 완화(금지) 또는 GetRows/GetCellValue/Rows.Columns 등 excelize 읽기 API 전부(전부 플래그됨)를 걷어내는 가짜 수정뿐이라 **고치지 않았고 커밋도 없다**. 지적(변경에 결함이 있다)은 틀렸다.
- 다음 행동: `curl -s https://vuln.go.dev/ID/GO-2026-6452.json | grep -c fixed` 가 1 이 되면 CI 재실행만으로 통과. 그 전엔 사람 판단으로 병합(패닉 회귀는 실제로 고쳐졌고 새 테스트가 v2.11.0 에서 떨어짐)하거나 홀드.
