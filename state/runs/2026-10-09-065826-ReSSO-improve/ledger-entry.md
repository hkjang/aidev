## 2026-10-09
- 선택: LDAP 개발 가이드의 서비스 준비 실패를 숨기는 eval 예제 수정 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `docs/user-federation.md`의 개발 절만 수정해 준비 스크립트의 실패가 빈 문자열을 평가하는 `eval`의 성공으로 덮이는 원인을 `assignment && eval && go test`로 해결하고, 저장소 루트·같은 Bash 셸·서비스 재사용 기본값·데이터를 버려도 될 때만 선택적 정리·준비/정리의 동일 `RESSO_TEST_CERT_DIR`·README CA 복구 링크를 명시했다(커밋 `c1b71bb`, 문서 1개·프로덕션 코드 0개·신규 영구 테스트 0개). 회차 전용 `verify-guide-failure.py`가 실제 실행 중인 세 컨테이너와 존재하지 않는 인증서 경로를 사용해 문서에서 추출한 준비 명령의 수정 전 실패→수정 후 통과→문서 원복 시 실패→복원 후 통과를 확인했으며, 모두 실제 `ca.crt is not a readable regular file` 진단을 확인했고 수정 후 종료값 1·후속 실행 없음, 최종 두 Bash 블록 `bash -n`, 첫 블록 실제 실행(`go test ./internal/...`, 캐시 사용), TLS `go test -race ./internal/federation -run '^TestDirectoryOverTLSRequiresACertificateItCanVerify$' -count=1 -v` PASS(1.083s, SKIP 없음), `make lint`(golangci-lint 0 issues·호출 취약점 0·ESLint 통과), `make test`(Go 13개 패키지 ok, 일부 캐시, httpserver 154.102s/store 103.007s, vet, vitest 29파일/161테스트, 빌드)와 `git diff --check`가 통과했다. 실제 make SKIP 경고는 없었으나 비상세 전체 Go 출력으로 SKIP 0을 별도 단언하지 않으며, npm ci 취약점 5건(중간 2·높음 3)·govulncheck 미호출 모듈 취약점 3건은 미해결로 남겼고, 정리 블록은 구문만 검사해 실제 --stop/CA 삭제/컨테이너 제거 없이 컨테이너 ID·시작 시각과 CA inode/크기/mtime/ctime 불변을 확인하고 빌드가 바꾼 webui/dist/index.html은 복원했다.
- 실패 재현: `FAIL: preparation example exited 0; expected nonzero and no follow-up` / `FOLLOWUP_RAN` — 문서 수정 전 `python3 "$run_dir/verify-guide-failure.py" implementation-red`의 실제 출력, exit 1; 준비 오류 stderr와 예제 exit 0은 `implementation-red.err`·`.status`에 보존했다.
- 보류 아이디어: README Client 인증 장애 지표의 오래된 401 설명 정정 (가치 2 / 위험 1 / 작업량 S).
  - Makefile SKIP 안내·준비 스크립트 머리말의 단독 eval 예제 정정 (가치 2 / 위험 1 / 작업량 S).
  - make test의 SKIP 집계용 두 번째 Go 실행 실패를 숨기지 않기 (가치 2 / 위험 2 / 작업량 M).
  - web devDependencies 취약점 경로 조사·정리 (가치 3 / 위험 2 / 작업량 M).
- 과제서: 채택 — 현재 문서·README·실제 스크립트의 CA 오류 및 --stop 분기가 과제서 근거와 일치해 지정된 한 절만 수정했고, 서비스/CA를 보존하며 요구한 정상·실패 검증을 완료했다.
