- 과제: Makefile test에 기존 이전 공개 릴리즈 선택 회귀 테스트 연결 (가치 3 / 위험 1 / 작업량 S)
- 왜: CI는 scripts/previous-release-tag-test.sh를 실행하지만 로컬 make test는 이 검증을 빠뜨린다. 이미 있는 8개 선택 회귀 검증을 로컬 진입점에 연결하면 과거 미공개 태그 선택 문제가 CI 전에 드러난다.
- 수용 기준: 1) make test가 기존 scripts/previous-release-tag-test.sh를 실행한다. 2) 스크립트가 실패하면 make test도 실패한다. 3) 기존 Go·프런트·audit 검증을 모두 유지하고 README에 검증 범위를 정확히 적는다.
- 건드릴 파일: Makefile:test — 기존 스크립트를 직접 실행하는 한 줄 추가; README.md:개발 — 로컬 테스트 범위에 공개 릴리즈 선택 검증을 명시.
- 검증 명령: ./scripts/previous-release-tag-test.sh ; make -n test ; make test ; git diff --check. 현재 정찰에서는 실행 결과 미확인, 후속 확인 뒤 갱신한다.
- 위험과 피할 것: scripts/previous-release-tag.sh와 release workflow는 변경 금지. gh 대역의 8건은 선택 로직 검증이며 실이미지 업그레이드 증거가 아니다. auth/migrations/workflows 및 의존성·lockfile은 범위 밖.
- 차선 후보: web/package.json의 esbuild 직접 devDependency 선언 — 현재 login.test.ts가 전이 의존성을 사용한다는 전제는 후속 재확인.

초안: main@329656c의 Makefile·CI·스크립트 원문 확인 완료. 최종 실행 증거·추정·구현 순서는 후속 갱신한다.
