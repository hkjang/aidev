## 2026-09-21
- 선택: Makefile에 CI 고정 버전의 gosec·읽기 전용 포맷 검사 타깃 추가 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: Makefile에 lint/lint-format/lint-security와 .PHONY를 추가하고, gosec v2.28.0·medium·scripts 제외·./... 범위를 CI와 맞췄으며 README에 실행법·다운로드 가능성·전체 CI 대체 불가를 설명했다(커밋 0bbff34). 실제 세 타깃 정상 통과, 격리 복제본의 포맷 위반·MEDIUM G306 및 구문 오류에서 실패 전파, 각 검사 전후 전체 파일 SHA-256 불변·임시 입력 삭제·원본 status 불변을 검증했고 go build/vet, go test -race ./..., 계약 -check, make -n test/기본 타깃, git diff --check도 통과했다. 요청된 technology 스킬 3종과 Skill 도구는 발견하지 못해 해당 절차·반환 형식을 적용했다고 주장하지 않으며, 프런트엔드·외부 PostgreSQL·브라우저 전체 CI는 미실행이다.
- 보류 아이디어:
  - CI gofmt 검사 추가 (가치 2 / 위험 1 / 작업량 S) — workflows는 이번 범위 제외.
  - gosec 잡 -stdout 추가 (가치 2 / 위험 1 / 작업량 S) — SARIF 외 로그 가시성 개선.
  - POP3 실제 TCP 어댑터 회귀 테스트 (가치 2 / 위험 1 / 작업량 M) — 차선 유지.
  - README govulncheck 고정 버전 정렬 (가치 2 / 위험 1 / 작업량 S) — @latest와 CI v1.6.0 차이 후속.
- 과제서: 채택 — 현재 Makefile·README·CI가 정찰 근거와 일치했고 런타임·워크플로·의존성 변경 없이 수용 기준을 충족했다.
