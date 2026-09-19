# PR 처리기 노트 2026-09-20-020840-SecCheck-shepherd — SecCheck PR #8
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## 수리 노트
- 지적(CI fail) 은 맞음: step 7 `govulncheck` exit 3, 로컬 재현 동일(GO-2026-6452 excelize, seed.go:66 GetRows). 정찰 프로필의 "vulndb 에 fixed 없음 → 어떤 브랜치도 초록 불가" 도 2026-09-20 02:10 기준 여전히 사실(vulndb range `introduced: 0` 만, 최신 태그 v2.11.0, master 의사버전도 걸림).
- 이 PR 의 변경(mail_test·Settings.tsx·docs) 은 원인과 무관하고, main 의 마지막 초록 run(09-16 17:01Z)은 보고서 공개(18:00Z) 1시간 전이라 main 도 지금 돌리면 같이 빨감.
- 고치지 않음(커밋 0): 셀 읽는 excelize API 전부가 영향 심볼이라 호출부 우회 불가, 워크플로 완화 금지, 라이브러리 교체는 범위 밖. 근거는 fix-summary.md.
- 대신 skipped 된 step 8 상당을 로컬로 돌림: gofmt·go vet·go test(web Mail|Docs)·tsc·vitest 11/11·vite build 모두 통과.
- 확신 없음: 로컬 `npm audit` 이 registry 400(quick 엔드포인트 폐기) — lockfile 은 main 과 동일해 환경 문제로 보지만 CI 에서 확인 필요. 게이트 해제 조건은 vulndb `fixed` 이벤트 또는 excelize 패치 태그 — 그 뒤 go.mod bump 1줄이면 끝(#9·#10 과 충돌 주의).
