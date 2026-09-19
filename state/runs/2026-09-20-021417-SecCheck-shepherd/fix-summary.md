# 수리 요약 — PR #10 (9b4b7aa) security-ci 실패

- 문제: `test-build-scan` 의 실패 step 은 GitHub API(`/actions/runs/35442392185/jobs`)로 `Go vulnerability scan` 하나뿐. 로컬 `govulncheck ./...`(go1.26, govulncheck 최신) 로 같은 자리 재현 — exit 3, `GO-2026-6452 … Found in: excelize/v2@v2.11.1-0.20260918021423-0434413565bf / Fixed in: N/A / internal/store/seed.go:66:25 ExtractWorkbookDefaults → GetRows`.
- 원인(코드 아님): vulndb 보고서(2026-09-20 조회, modified 09-16 그대로)는 references 에 upstream FIX 커밋 93f0b3c·v2.11.0 릴리즈를 적어 두고도 `ranges.events` 에 `introduced: 0` 만 있고 `fixed` 가 없다 → 어떤 excelize 버전(새 태그도 없음, 마지막 v2.11.0)으로도 초록 불가. golang/vulndb#6510("v2.11.0 contains the patch") 은 09-19 15:50 갱신, 여전히 open.
- 고치지 않음: 남는 수단은 `GetRows` 호출 제거(템플릿 가져오기 기능 파괴) 또는 ci.yml 완화·`replace` 뿐이며 둘 다 절대 규칙 위반. 이 PR 의 코드(excelize 핀·회귀 테스트·precheck 새 단계·operations.md)는 결함이 아니라 정확히 이 상태를 설명하는 변경이므로 손대지 않았고 커밋 없음.
- 필요한 조치(저장소 밖): vulndb 에 `fixed: 2.11.0` 이 기록되면 코드 변경 없이 CI 재실행만으로 통과. 그 전엔 PR #10 을 머지해도 main 의 security-ci·release 게이트는 같은 step 에서 빨갛다(비평 노트가 이미 예고한 의도된 상태).
