- 과제: README 통합 테스트 준비를 CI와 같은 test-services.sh로 통일 (가치 2 / 위험 1 / 작업량 S)
- 왜: README.md의 「개발 및 검증」은 resso-test-pg를 testpw/55432로 만드는 옛 절차를 안내하지만 scripts/test-services.sh는 같은 이름의 컨테이너를 재사용하며 resso 비밀번호를 가정하고, README만 따르면 LDAP·LDAPS 테스트는 건너뛴다. 기존 준비 스크립트를 정본으로 안내하면 컨테이너 이름 충돌과 인증 실패를 피하고 CI verify와 같은 세 서비스를 사용해 검증할 수 있다.
- 수용 기준: 1) README의 수동 docker run·고정 DSN 블록을 제거하고 저장소 루트에서 Docker 실행 가능·Go/Node/npm 준비 → scripts/test-services.sh로 환경 설정 → make lint / make test의 순서를 명확히 적는다(셸 환경은 같은 셸에서 유지). 2) PostgreSQL·LDAP·LDAPS가 필요하고 환경변수 누락 시 SKIP도 성공 종료된다는 점, make test의 SKIP 요약 확인, scripts/test-services.sh --stop은 테스트 컨테이너와 인증서 정리라는 점을 적고 docs/user-federation.md의 개발 테스트 절로 연결한다; 이미 옛 수동 컨테이너를 만든 경우 테스트 데이터 삭제를 명시한 선택적 재생성 안내를 둔다. 3) 아래 실제 HTTP/PostgreSQL·LDAP·LDAPS 테스트가 PASS이고 SKIP 0임을 확인한다; 문서 문자열 검사나 모의 서비스 테스트를 새로 만들 필요는 없다. 새 컨테이너 생성이 가능한 깨끗한 환경은 이번 정찰에서 미확인이라고 검증 기록에 구분한다.
- 건드릴 파일: README.md:「개발 및 검증」 — 수동 준비 절차를 제거하고 스크립트 기반 단일 안내로 교체. 참조만 할 파일: scripts/test-services.sh:start_postgres/published_port/start_tls_directory(실제 포트 조회·비밀번호·인증서 재사용 확인), Makefile:test/lint(실제 명령과 SKIP 요약), .github/workflows/ci.yaml:verify(스크립트를 사용하는 CI), docs/user-federation.md:「개발 중 디렉터리 연동 테스트」(기존 정본에 링크). 함수 없는 문서 수정이다.
- 검증 명령: `git diff --check`; `bash -n scripts/test-services.sh`; 저장소 루트에서 아래 명령. 이번 머신의 기존 LDAPS 마운트는 docker inspect로 /tmp/resso-test-certs-0916임을 확인했으므로 이 경로는 검증 때만 사용하고 README 기본 명령에 박지 않는다.

```bash
# 새 환경용 README 예시는 eval "$(scripts/test-services.sh)"와 같은 기존 관례를 따를 수 있다.
# 검증에서는 스크립트 실패를 놓치지 않도록 출력을 받은 명령의 성공을 먼저 확인한다.
test_env="$(RESSO_TEST_CERT_DIR=/tmp/resso-test-certs-0916 scripts/test-services.sh)" &&
  eval "$test_env" &&
  go test -race ./internal/httpserver ./internal/federation     -run '^(TestIntegrationUserInfoLimitsPostBody|TestDirectoryConnectionReachesTheConfiguredBase|TestDirectoryOverTLSRequiresACertificateItCanVerify)$' -count=1 -v
```

- 위험과 피할 것: README만 수정한다. auth.go·oidc.go·migrations·.github/workflows·Makefile·준비 스크립트의 동작을 함께 바꾸지 않는다. 기존 컨테이너를 자동 삭제하거나 운영 PostgreSQL 설정을 바꾸지 않는다. 기존 LDAPS 컨테이너의 bind mount와 RESSO_TEST_CERT_DIR가 다르면 인증서 파일이 없거나 불일치할 수 있으므로 inspect로 확인하며 TLS 검증을 끄지 않는다. CI verify는 세 서비스를 쓰지만 release.yaml은 PostgreSQL만 준비하므로 모든 CI가 동일하다고 단정하지 않는다. PDF/화면 캡처와 webui/dist 재생성은 필요 없다. 과거 사람 반려의 구체적 변경 내용은 미확인이다.
- 차선 후보: 잘못된 logout POST 폼을 세션 종료 전에 거절 (가치 2 / 위험 2 / 작업량 M) — README가 다른 변경으로 이미 해결된 경우만 선택. internal/httpserver/oidc.go:oidcLogout은 1MiB 제한 뒤 `_ = r.ParseForm()`으로 오류를 버리고 endSession/clearBrowserCookies까지 진행한다. ParseForm 실패 시 400 invalid_request로 즉시 반환하고 integration_test.go의 TestIntegrationLogoutRecordsASessionItCouldNotEnd 및 TestIntegrationLogoutTakesTheHintARelyingPartyActuallyHolds를 참고해 실제 HTTP·쿠키·DB 세션으로 malformed `%zz`/1MiB 초과에서 세션·쿠키 유지, 정상 POST 로그아웃 유지 증명. 이 후보의 오류 요청 런타임 재현은 이번 미실행이며 착수 첫 단계에서 재현할 것; hint의 sub/aud 정책 변경은 금지.

추정·진행 계획: 문서 수정 10분 + 실제 서비스 검증 10분 + 검토 5분, 환경 문제 대비 15분으로 총 40분 이내. 새 환경 생성 검증 때문에 공유 테스트 컨테이너를 지우지 말 것. 기존 문서와 동작 비교 → README 변경 → 실제 서비스 검증 → diff 확인 순서로 진행한다.
스킬 가용성: 요청된 pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration은 현재 도구 목록에 Skill/skills.list가 없고 로컬 스킬 검색에서도 찾지 못했다. 절차/반환 형식은 미확인이고 적용했다고 주장하지 않는다; 사용자 지정 과제서 형식과 자체 산정·대안 비교로 정찰을 완료했다.

정찰 검증 결과: 위 스크립트 준비 및 두 패키지 선택 테스트 실제 실행 PASS(httpserver 2.156s, federation 1.097s), SKIP 0. bash -n scripts/test-services.sh 성공, 작업 트리 변경 없음. 전체 make test/make lint는 이번 미실행.
