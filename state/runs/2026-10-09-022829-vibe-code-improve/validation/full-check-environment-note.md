# 전체 검증 실행 환경 보정
- 첫 npm run check는 TMPDIR을 회차 validation 아래로 지정했으나 Git 탐색 상한을 지정하지 않았다. 기존 checkpoints.test.ts의 비저장소 사례가 상위 aidev 저장소를 발견해 git add -A를 실행 중인 것을 /proc의 자식 PID 및 cwd로 확인했다.
- 해당 검증 프로세스 트리를 중단했다. 중단 전 완료된 전체 검증 결과로 취급하지 않는다(update-check-full-check.log).
- snapshotTree는 별도 GIT_INDEX_FILE을 쓰며 실제 잔여물은 validation/vibe-checkpoint-zJiBYl/index 및 index.lock이었다. 중단한 실행이 남긴 이 폴더와 빈 vibe-checkpoint-test-E86Kql만 제거했다. aidev의 refs/vibe-checkpoints 목록은 비어 있음을 확인했다. 상위 저장소의 사전 index 해시는 수집하지 않았으므로 바이트 불변을 실측했다고 주장하지 않는다. git add가 남겼을 수 있는 미참조 객체는 삭제하지 않았다.
- 프로덕션/테스트 수정 없이 GIT_CEILING_DIRECTORIES=$TMPDIR로 상위 저장소 탐색을 차단하고 같은 npm run check를 재실행했다(update-check-full-check-isolated.log).
