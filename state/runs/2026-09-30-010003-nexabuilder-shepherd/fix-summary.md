# 수리 요약 — PR #37 (커밋 없음)

- **고칠 결함이 없어 커밋하지 않았다.** CI 실패는 코드가 아니라 GitHub 계정 과금 차단이다. 잡 주석: "The job was not started because recent account payments have failed or your spending limit needs to be increased." `gh api .../jobs` 의 `steps` 가 빈 배열 — 체크아웃조차 안 돈 채 39초 만에 실패 처리됐다.
- 브랜치 고유 문제가 아니다. `gh run list` 상 최근 ci 런이 이미 머지된 #35/#36 것까지 전부 같은 사유로 failure 다.
- CI 가 돌렸어야 할 두 명령을 0421f6d 그대로 로컬에서 재현 실행: `sh ./gradlew --no-daemon test` → BUILD SUCCESSFUL 5분51초, XML 집계 615건·failures 0·errors 0·skipped 0(155 클래스). `sh ./gradlew --no-daemon bootJar -x test` → BUILD SUCCESSFUL, `build/libs/nexabuilder-1.26.0.jar` 생성.
- 따라서 PR 내용(all-hidden 목록의 내보내기 fallback 차단)에는 손대지 않았고 작업 트리는 clean 이다. 해소는 저장소 소유자의 Billing & plans 설정 뒤 재실행뿐이며, 워크플로 파일을 고쳐 통과시키는 것은 금지 규칙에 해당해 하지 않았다.
