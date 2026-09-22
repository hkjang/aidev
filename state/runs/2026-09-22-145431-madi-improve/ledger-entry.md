## 2026-09-22
- 선택: 고정 runtime APK 잠금 파일을 전체 빌드 전에 검사하는 사전 점검 스크립트 (가치 3 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: 커밋 2fd17bf. Dockerfile의 runtime-base 이미지를 읽는 scripts/check-apk-lock.sh와 실제 Docker/APK 실행 테스트, deploy/README.md 사용법을 추가했다. 기본 lock의 인증서/tzdata 충돌을 그대로 재현했고, 설치된 busybox는 네트워크 실패 경고에도 APK가 exit 0을 반환하는 추가 원인을 확인하여 stderr 진단을 보존하면서 비영 처리했다. bash -n, bash tests/check-apk-lock.sh(양성·불가능 버전·입력 거부·실제 프록시 연결 실패·다른 cwd/공백 경로·해시 및 git diff 불변), go test -count=1 ./tests/deployment-contract, git diff --check 통과; 기본 lock 실행은 예상 exit 2이고 전체 이미지 빌드는 수행하지 않았다.
- 보류 아이디어: CLAUDE.md에 좁은 검증·웹 빌드 선행 관례 명문화 (3/1/S)
- 보류 아이디어: 메일 알림 Playwright 시험 — 기능 브랜치 병합 선행 (3/2/M)
- 보류 아이디어: silent SSO 안내 브라우저 시나리오 — 안내 기능 병합 선행 (3/2/M)
- 보류 아이디어: 자연어 날짜 연도 경계·상한 회귀 보강 (2/1/S)
- 과제서: 채택 — Docker와 실제 고정 이미지 사용이 가능하고 lock 충돌 근거가 일치하여 지정된 세 파일만 변경했다.
