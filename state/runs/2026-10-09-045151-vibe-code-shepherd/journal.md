# PR 처리기 노트 2026-10-09-045151-vibe-code-shepherd — vibe-code PR #20
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-09-022829-vibe-code-improve)
# 회차 노트 2026-10-09-022829-vibe-code-improve — vibe-code
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 02:28] base pinned — main@6eaede8
- [러너 02:28] autonomy release — 

## 정찰 노트
- VSIX 이름의 디렉터리가 실제 파일을 가리는 결함을 Node20.19.2 + production latestVsixVersion + 실제 fs로 재현해 선택했다. 기존 제출 중복·보호 경로를 피하고 프로덕션 1파일로 끝난다.
- 파일명 계약 불일치와 빈 probes 후보는 코드상 성립하지 않아 기각; 다운로드 실패를 무시하던 CI는 수정됨. 기존 12개 보류 항목을 모두 보존·재평가하고 새 후보 2개와 기존 차선을 기록했다.
- 실제 사용자 빈도·원격 PR/CI·Windows 설치 UI는 미확인. 파일 선택 결과를 입증했으며 알림/설치 전체를 E2E 검증했다고 쓰지 않는다.
- 구현자는 파일 symlink 지원을 보존하고 디렉터리/깨진 링크만 제외한다. semver·설치 주기·MCP 인증·workflow·기존 Git 경로 수정으로 넓히지 말 것; brief는 초안 저장 후 재현 근거로 갱신했다.
- [러너 02:36] scout done — 공유 폴더 업데이트 탐색에서 VSIX 이름의 디렉터리를 설치 후보에서 제외 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- ccfeaf2: VSIX 이름 매칭 후 statSync().isFile()로 후보를 좁힘. 파일 링크는 보존하고 디렉터리/디렉터리 링크/깨진 링크는 제외; 프로덕션 1파일 + 신규 테스트 1파일.
- 실제 fs 10개 테스트 추가. 수정 전/프로덕션만 재역전 시 같은 4개 AssertionError, 복원 후 집중 12개 통과(red/causation-red/restored-green 로그).
- Node20.19.2 npm run check: 20 files/162 tests 모두 통과(skip 0), 타입 검사·빌드·두 번들 문법 검사 통과; 기존 테스트/공유 스텁 불변.
- 미검증: Windows 패키징·VS Code 알림/실제 설치·원격 CI. 기존 런타임 자산 누락 경고, npm ci 취약점 4건은 남음; 파일 선택/설치 입력 경로까지의 증거다.
- 제외: ZIP 검증·SemVer 확장·설치 주기/명령·TOCTOU 해결. 과제서대로 파일 후보 필터에 한정했다.
- 주의: TMPDIR을 회차 validation으로 둘 때 GIT_CEILING_DIRECTORIES=$TMPDIR도 지정할 것. 첫 전체 검사는 기존 체크포인트 테스트가 상위 aidev를 탐색해 중단; 잔여 임시 index 정리, ref 없음 확인, 미참조 객체 가능성/사전 index 해시 미수집은 validation/full-check-environment-note.md에 기록.
- 원장·ideas.json 갱신 완료(기존 15개 보존, 선택 항목 done). 최종 전체 근거는 update-check-full-check-isolated.log; 실제 소요 약 5분, 빌드 산출물 제외 및 작업 트리 깨끗함 확인.
- [러너 02:41] brief accepted — 채택 — main@6eaede8의 이름 전용 후보 선택과 실제 fs 재현이 과제서 근거와 일치하여 지정한 두 파일 및 수용 기준으로 완�
- [러너 02:41] verify passed — 검증 6개 통과 (auto)
- [러너 02:41] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 02:41] pr created — https://github.com/hkjang/vibe-code/pull/20
