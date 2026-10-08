## 2026-10-08
- 선택: 재사용하는 LDAPS 테스트 컨테이너의 CA 파일이 없으면 준비 스크립트가 성공하지 않게 하기 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공 (커밋 147fce8)
- 요약: `start_tls_directory`의 기존 컨테이너 분기에 `ca.crt` 일반 파일·읽기 가능 검사를 추가해 누락 시 exit 1·빈 stdout과 경로/원인/Mounts/RESSO_TEST_CERT_DIR 복구 안내를 남기고, README의 준비 실패 설명과 기존 파괴적 --stop 경고를 함께 유지했다. 실제 Docker의 누락 CA 프로브를 수정 전 실패→수정 후 통과→검사 제거 시 실패→복원 후 통과로 확인했으며 정상 준비는 네 export/exit 0, `go test -race ./internal/federation -run '^TestDirectoryOverTLSRequiresACertificateItCanVerify$' -count=1 -v`는 PASS(1.088s, SKIP 없음)로 TLS 검증과 실제 LDAP 인증을 유지했다. `bash -n`, `git diff --check`, 최종 `make lint`(golangci-lint 0 issues, ESLint 통과, govulncheck 호출 취약점 0), `make test`(Go 13개 패키지 ok, vet, vitest 29파일/161개, 빌드, 실제 SKIP 경고 0) 통과; 빌드가 바꾼 webui/dist/index.html은 복원했고 컨테이너 ID/시작 시각과 인증서 inode/크기/mtime/ctime 불변도 확인했다.
- 실패 재현: `FAIL: missing CA preparation exited 0; expected nonzero (stdout bytes: 328)` — 수정 전에 회차 경로의 `verify-missing-ca.sh implementation-before` 실행, 실제 스크립트 exit 0/네 export/빈 stderr를 보존했다. 검사 코드 제거 후 같은 프로브도 같은 문구로 실패했다.
- 보류 아이디어: README Client 인증 장애 지표의 오래된 401 설명 정정 (2/1/S).
  - LDAP 개발 가이드의 준비 실패를 숨기는 단독 eval 예제 정정 (2/1/S).
  - web devDependencies 취약점 정리 (3/2/M): 이번 npm ci가 중간 2·높음 3, 합계 5건 보고; 개별 경로는 미분석.
  - RotateRefreshToken 세션 조회 장애 분리 (3/3/M): 프로덕션 경로 도달 여부 조사 필요, 기각된 코드 교환 변경 반복 금지.
- 과제서: 채택 — 실제 코드·컨테이너·누락 CA 현상이 일치해 지정한 과제를 구현했으며, 전체 검증으로 발견한 기존 포트 테스트의 CA 없는 픽스처 전제만 `brief.md`에 정정하고 보완했다(서버 프로덕션 0개·스크립트 1개·문서 1개·기존 테스트 1개, 신규 영구 하네스 없음).

검증 한계·환경 조치: 실제 인증서 삭제/권한 박탈/일반 파일이 아닌 CA 상태는 별도 실행하지 않았고, CA 내용·만료·다른 CA와의 일치 검사는 범위 밖이다. 초기 make lint는 Go 1.25.14로 빌드된 govulncheck v1.6.0의 Go 1.26 분석 불가로 실패해 실행 파일을 회차 경로에 백업한 뒤 같은 버전을 Go 1.26.7로 재빌드했다. 초기 make test는 기존 포트 검사 두 개가 새 CA 검사에서 멈춰 실패했으나, 해당 픽스처만 보완한 뒤 두 테스트와 전체 검증을 다시 통과했다. 최종 전체 Go 실행 중 cmd/resso 외 패키지는 이번 회차 첫 전체 실행의 통과 캐시를 사용했다. govulncheck는 현재 코드에서 호출하지 않는 의존 모듈 취약점 3건을 별도로 보고했다. 이 의존성 경고들은 미해결이며 이번 변경으로 해결했다고 주장하지 않는다. 프로브·초기/최종 검증 로그는 이 회차 디렉터리에 보존했다.
