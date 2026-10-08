## 2026-10-09
- 선택: README 개발 명령의 누적 cd 때문에 make docker가 web에서 실행되는 문제를 고친다 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: README의 SDK/web npm 명령 두 줄을 각각 서브셸로 감싸 마지막 make가 저장소 루트에서 실행되게 했다(프로덕션 코드 0파일·문서 1파일, 커밋 79a14e5); Go/npm 인자·순서와 Docker 타깃·이미지 이름은 유지했다. 실제 README에서 추출한 블록의 bash -n 및 readme-path-check.py 경로 실행은 수정 전 exit 2 → 수정 후 exit 0이며, 원문 복원 시 exit 2와 수정본 복구 후 exit 0도 재확인했고 git diff --check가 통과했다. 검증은 실제 cd·서브셸·pwd 및 make -n docker의 경로·Makefile 해석에 한정되며 npm/Go/Docker 전체 빌드·전체 CI·DB·보안 감사는 이번 미실행이다.
- 실패 재현: `make: *** No rule to make target 'docker'.  Stop.` / 경로 `sdk → web → web`, exit 2 (수정 전 readme-path-red.log; 원문 복원 대조 readme-path-revert-red.log도 exit 2).
- 보류 아이디어: Custom Dimension 입력 칸 규칙 helperText (가치 2 / 위험 1 / 작업량 S) — 지정 차선이나 1순위 재현으로 미선택.
  설정 저장 실패 한국어 안내 (가치 2 / 위험 2 / 작업량 M) — 여러 그룹 계약 확인 필요.
  보존 정책 빈 입력 Number("")→0 수정 (가치 2 / 위험 2 / 작업량 M) — 상태·payload 분리와 브라우저 재현 필요.
  JSON Schema 객체 아님 hint 안내 (가치 2 / 위험 1 / 작업량 S) — 서버 허용 범위를 좁히지 않는 별도 과제.
- 과제서: 채택 — 현행 README와 실제 실패가 과제서에 일치하여 지정한 두 줄만 수정하고 같은 실행 검증으로 수용 기준을 확인했다.
