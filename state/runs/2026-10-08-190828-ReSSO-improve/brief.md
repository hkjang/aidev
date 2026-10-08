- 과제: 재사용하는 LDAPS 테스트 컨테이너의 CA 파일이 없으면 준비 스크립트가 성공하지 않게 하기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `scripts/test-services.sh:start_tls_directory`는 기존 컨테이너를 발견하면 CA 파일 확인 없이 반환하여, 실제로 존재하지 않는 `RESSO_TEST_LDAP_CA`까지 export하고 성공 종료한다. 준비 단계에서 실패 사유와 복구 경로를 알려 주면 지난 회차처럼 전체 테스트를 돌린 뒤 인증서 누락을 발견하는 낭비를 막는다.
- 수용 기준:
  1) 기존 `resso-test-ldaps`가 있는 상태에서 `RESSO_TEST_CERT_DIR`를 CA 파일이 없는 경로로 설정하면 준비 스크립트는 nonzero로 종료하고 stdout에 환경변수 export를 남기지 않는다. stderr에는 문제가 된 `ca.crt` 경로와 읽을 수 없다는 진단, 기존 컨테이너의 Mounts 확인 및 올바른 `RESSO_TEST_CERT_DIR` 지정 안내가 있다.
  2) 기존 컨테이너 + 읽을 수 있는 정상 CA는 기존과 같은 네 export와 exit 0을 유지한다. 인증서를 새로 만들거나 덮어쓰고, 컨테이너를 지우거나 재시작하는 자동 복구를 추가하지 않는다.
  3) 실제 Docker 컨테이너를 둔 채 실제 스크립트를 실행하여 누락 CA의 실패와 정상 CA의 성공을 확인한다. 정상 export를 받은 `TestDirectoryOverTLSRequiresACertificateItCanVerify`가 SKIP 없이 PASS하여 TLS 검증과 실제 LDAP 인증까지 유지됨을 증명한다. Docker/OpenSSL 대역, 소스 문자열 검사는 이 증거를 대신할 수 없다.
- 건드릴 파일:
  - `scripts/test-services.sh:start_tls_directory`(현재 268행 부근) — 기존 컨테이너 분기의 조기 return 전에 `ca.crt`가 일반 파일이고 읽을 수 있는지 검사하고, 실패 시 기존 `log`로 stderr 진단 후 exit 1. 신규 컨테이너 생성·`make_certificates`·`published_port` 계약은 그대로 둔다.
  - `README.md:개발 및 검증`(219행 부근) — 기존 인증서 디렉터리 안내를 고쳐 누락 CA는 준비 단계에서 실패한다는 점과 Mounts 확인을 설명. `--stop`이 모든 테스트 컨테이너·인증서를 지운다는 기존 경고를 유지한다.
  - 서버 프로덕션 파일 0개, 실행 스크립트 1개, 문서 1개. 영구 테스트 하네스 신설은 이번 범위 밖이며 아래 실제 실행 검증을 기록한다.
- 검증 명령: 저장소 루트의 Bash에서 아래 순서대로 실행한다. Docker 세 서비스가 이미 실행 중이라는 전제가 확인되어 있다. 다른 환경이라면 정상 `scripts/test-services.sh` 실행으로 먼저 준비한다.

```bash
bash -n scripts/test-services.sh
# 실제 컨테이너의 인증서 경로를 확인한다. 정찰 시 /tmp/resso-test-certs였다.
docker inspect -f '{{range .Mounts}}{{.Source}}:{{.Destination}} {{end}}' resso-test-ldaps
# 없다고 확인한 별도 경로만 지정한다. 실제 인증서는 삭제하거나 이동하지 않는다.
probe_dir=/mnt/c/Users/USER/projects/aidev/state/runs/2026-10-08-190828-ReSSO-improve
missing_ca="$probe_dir/missing-ca-probe"
test ! -e "$missing_ca"
probe_status=0
RESSO_TEST_CERT_DIR="$missing_ca" bash scripts/test-services.sh >"$probe_dir/ca-probe-after.stdout" 2>"$probe_dir/ca-probe-after.stderr" || probe_status=$?
test "$probe_status" -ne 0
test ! -s "$probe_dir/ca-probe-after.stdout"
cat "$probe_dir/ca-probe-after.stderr"
# stderr가 경로·원인·Mounts/RESSO_TEST_CERT_DIR 복구 안내를 포함하는지 확인한다.
test_env="$(scripts/test-services.sh)" && eval "$test_env" &&
  go test -race ./internal/federation -run '^TestDirectoryOverTLSRequiresACertificateItCanVerify$' -count=1 -v
git diff --check
```

- 위험과 피할 것: `auth`·토큰 응답·migrations·workflows·의존성·Makefile은 변경하지 않는다. CA를 새로 만들기만 하면 실행 중 slapd가 쓰는 인증서와 달라질 수 있으므로 자동 재발급/자동 `--stop`은 금지한다. 인증서 내용·만료·다른 CA와의 일치 여부까지 검사하는 확장은 별도 과제다. `eval "$(...)"` 자체는 내부 명령 실패를 감추므로 반드시 README의 `test_env=... && eval ...` 형태로 검증한다. `set -euo pipefail` 아래 검증 실패를 정상으로 삼는 명령은 위처럼 상태를 명시적으로 받는다.
- 차선 후보: README의 `resso_client_auth_errors_total` 설명에 남은 “401로 나가는 것은 … 같다”를 실제 token 500 / revoke 503 / introspection 200 active=false와 일치시키기 — README 192행과 `docs/operations.md:90`, `token`·`revoke`·`introspect`를 대조한 문서 1개 수정. 1순위가 이미 고쳐졌거나 재현되지 않을 때만 선택한다. 과거 차선이었던 UpdateUser 테스트 추가는 이미 존재하므로 선택하지 않는다.

진행 단계와 검토 지점(구현자 갱신용):
1. [대기] 위 누락 CA 프로브로 현재 exit 0/잘못된 export를 다시 확인한다. 증거는 stdout/stderr/종료 코드이며 사람 검토 지점은 없다.
2. [대기] 스크립트의 재사용 분기만 수정하고 `bash -n` 및 같은 프로브로 nonzero/빈 stdout/진단을 확인한다. 이 단계도 빌드 가능한 상태를 유지한다. 전제가 다르면 과제서를 고친 뒤 진행하며 범위를 넓히지 않는다.
3. [대기] README 해당 문단을 맞추고 정상 준비 + 위 TLS 테스트 + `git diff --check`를 실행한다. PASS와 SKIP 없음이 자동 검토 지점이며 별도 사람 승인은 필요 없다.

선택 근거·대안:
- 선택한 최소안은 누락/읽기 불가 CA를 준비 시점에 거절한다. 복구는 올바른 기존 CA 경로 지정이며, 기존 데이터를 보존한다.
- 자동 재발급/LDAPS 재생성안은 더 넓은 복구를 제공하지만 실행 중 서버와 CA 불일치, 기존 컨테이너/인증서 파괴를 다뤄야 해 이번 범위를 넘는다.
- 문서만으로 우회하는 안은 변경량이 가장 작지만 성공처럼 보이는 출력이 남으므로 차선 이하로 둔다.
- 핵심 가정: 재사용 시 지정된 CA 파일이 읽을 수 있어야 테스트 환경을 제공했다고 말할 수 있다. 파일이 읽혀도 다른 CA이거나 만료되었는지는 이번에 판정하지 않는다.

작업량 근거(pmo 스킬 적용): 상향식으로 재현/상태 확인 4–6분, 분기 수정 5–8분, 정상·실패 검증 및 문서 8–12분 = 기본 17–26분. Docker 응답 지연이라는 알려진 변동에 별도 예비 5–8분을 두어 총 22–34분으로 예상한다(경험적 판단, 통계적 신뢰수준 미측정; 확신 중간). 이전 S 회차와 달리 DB 장애 주입·HTTP 테스트 신설이 없어 범위는 더 작지만 과거 실제 소요시간은 제공되지 않아 수치 유추는 하지 않았다. 새 인증서 수명/권한/컨테이너 복구 요구는 관리 예비로 자동 흡수하지 않고 후속 회차로 분리한다. 45분에 맞추기 위한 범위 확대는 없다.

정찰 검증 결과(코드 무수정): HEAD 8ef825f(v0.9.102). 기존 세 컨테이너 running, LDAPS mount `/tmp/resso-test-certs`. 없는 `missing-ca-probe` 경로로 실제 스크립트 exit 0, 네 export 출력, stderr 빈 값 — `ca-probe-before.stdout`/`.stderr`에 보존. `bash -n` 통과. 정상 준비 후 위 TLS 테스트 PASS(1.131s, SKIP 없음). 전체 make lint/make test는 이번 정찰에서 실행하지 않았다. 실제 인증서 삭제 상태와 읽기 권한 부족 상태는 별도 재현하지 않았으며, 재사용 분기가 같은 파일 확인을 건너뛰는 것은 소스에서 확인했다.

적용 스킬: 로컬 headcount의 `pmo/skills/estimating-and-contingency/SKILL.md`, `technology/skills/implementation-planning/SKILL.md`, `technology/skills/solution-exploration/SKILL.md`를 읽었다. 이 세션에는 Skill 호출 도구가 없어 `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/` 아래 파일로 불러왔다. 외부 원가·편익 수치는 사용하지 않았다.


## 구현 중 확인한 전제 정정
- 전체 `make test`에서 기존 `cmd/resso/testservices_test.go`의 포트 검사 두 개가 CA 없는 임시 경로를 전제로 해 새 준비 검사에서 실패했다. 두 테스트의 단언은 유지하고 기존 공용 픽스처에 읽을 수 있는 `ca.crt` 일반 파일을 마련한다. 신규 영구 테스트 하네스는 만들지 않는다.
- 최종 범위는 서버 프로덕션 파일 0개, 실행 스크립트 1개, 문서 1개, 기존 테스트 픽스처 1개다. CA 결함 수용 기준은 대역 테스트가 아닌 실제 Docker 프로브와 실제 TLS/LDAP 테스트로 계속 입증한다.
- 초기 `make lint`는 Go 1.25.14로 빌드된 govulncheck v1.6.0이 Go 1.26 소스를 분석하지 못했다. 기존 실행 파일을 회차 경로에 백업하고 같은 v1.6.0을 Go 1.26.7로 다시 빌드했다. 저장소 의존성·Makefile은 변경하지 않는다.
