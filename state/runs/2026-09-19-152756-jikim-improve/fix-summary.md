# 수리 요약 (2026-09-19, 시도 1)
- 문제: `scripts/build-image.sh`가 git 인덱스에 100644(비실행)로 커밋됨. core.fileMode=false 환경에선 작업 트리 755로 보여 통과했지만, `--no-local` 임시 clone에서 `./scripts/build-image.sh` → `Permission denied` exit 126 재현(Makefile:46·verify.sh:61 모두 직접 exec).
- 수리: `git update-index --chmod=+x scripts/build-image.sh` 후 새 커밋 41089bb(mode change 100644→100755, 내용 변경 없음). 다른 파일은 손대지 않음.
- 검증: 41089bb를 다시 임시 clone → `-rwxr-xr-x`, `bash -n` 통과, 직접 실행 시 126 아님; 작업 트리에서 `make docker` exit 0(출력 2줄, `Docker 이미지 빌드 완료: jikim:v0.2.15`).
- 미실행: `make release-check` 전체(smoke·e2e·package)는 이번에 돌리지 않음 — 결함이 exec 비트 하나뿐이라 `make docker` 통과로 동일 경로(verify.sh:61)가 커버됨.
