- 과제: LDAP 개발 가이드의 서비스 준비 실패를 숨기는 eval 예제 수정 (가치 3 / 위험 1 / 작업량 S)
- 왜: `docs/user-federation.md:10-12`의 단독 `eval "$(scripts/test-services.sh)"`은 준비 스크립트가 실패해도 성공으로 끝나 다음 테스트를 실행하고, 마지막 `--stop`은 재사용하는 테스트 데이터까지 삭제한다. 이미 README에 있는 실패 전파 방식과 선택적 정리 안내를 이 가이드에도 적용하면 실패한 준비를 정상 검증으로 오인하거나 기존 테스트 데이터를 뜻하지 않게 지우는 일을 줄인다.
- 수용 기준:
  1) 「개발 중 디렉터리 연동 테스트」의 실행 예제가 저장소 루트·같은 Bash 셸을 전제로 `test_env="$(scripts/test-services.sh)" && eval "$test_env" && go test ./internal/...`를 여러 줄 `&&` 연결로 제공한다. 준비 또는 eval 실패 시 테스트를 실행하지 않아야 한다. `eval` 앞에 `set -e`만 추가하는 방식은 실패 원인을 해결하지 않으므로 쓰지 않는다.
  2) 기본 검증 블록에서 `scripts/test-services.sh --stop`을 제거하고, 검증 후 서비스 재사용이 기본임을 안내한다. 정리는 별도 선택 절차로 두되 PostgreSQL·LDAP·LDAPS 테스트 컨테이너와 인증서를 삭제하며 기존 테스트 데이터를 버려도 되는 경우에만 실행한다는 조건을 명시한다. 사용자 지정 인증서 디렉터리는 준비·정리에 같은 `RESSO_TEST_CERT_DIR`을 사용한다. 자세한 CA 복구는 README 개발 및 검증 절로 연결하고 자동 삭제/재발급을 권하지 않는다.
  3) 실제 스크립트 실패를 이용해 준비 실패의 종료값이 nonzero이고 후속 명령이 실행되지 않음을 증명한다. 실제 CA를 지우지 말고 실행 중인 기존 LDAPS 컨테이너에 존재하지 않는 인증서 경로를 지정한다. 정상 준비에서는 기존 TLS 테스트가 PASS하고 SKIP이 없어야 한다. 문서의 최종 코드 블록도 Bash 구문을 확인한다.
- 건드릴 파일: `docs/user-federation.md:5` 「개발 중 디렉터리 연동 테스트」 — 명령 예제와 실패·정리 설명만 수정. 문서 1개, 프로덕션 코드 0개, 신규 영구 테스트 0개.
- 검증 명령: 아래 명령과 실행 증거 참조. `git diff --check`로 마무리한다.
- 위험과 피할 것: auth/migrations/workflows, Go 코드, Makefile, scripts/test-services.sh, 의존성, README의 오래된 401 설명을 이번에 같이 고치지 않는다. CA 삭제·컨테이너 제거·실제 `--stop` 실행 금지. mock docker나 소스 문자열 일치 검사를 동작 증거로 쓰지 않는다. ADMIN_GUIDE/USER_GUIDE/PDF/캡처 재생성 불필요. 과거 반려된 코드 교환 500 응답 변경과 무관한 문서 과제로 유지한다.
- 차선 후보: README Client 인증 장애 지표 설명의 오래된 401 응답 문구 수정 — 첫 후보가 구현 시점에 이미 해결된 경우에만 선택. `README.md:192`의 401 문구를 `docs/operations.md:90` 및 `internal/httpserver/oidc.go`의 token 500/revoke 503/introspection 200 active=false와 일치시킨다. 이번 과제에 합치지 않는다.

범위 밖: 스크립트의 CA 내용/만료 검증, 서비스 시작 로직, SKIP 집계 방식, 전체 문서 정비, 의존성 업데이트. 정찰은 코드를 수정하지 않았다.

확인한 근거:
- README.md:209-213은 이미 assignment → eval → make lint → make test를 `&&`로 연결한다. :218 이후는 파괴적 정리 조건과 CA 마운트 경로 복구를 안내한다.
- scripts/test-services.sh:start_tls_directory는 기존 컨테이너가 있을 때 `-f/-r` CA 검사 실패를 exit 1로 보고하며 환경 export는 파일 맨 마지막에 있다. `--stop` 분기는 세 컨테이너와 인증서 디렉터리를 제거한다. 지난 회차 147fce8에서 이 검사가 완료됐으므로 재구현하지 않는다.
- 실제 기존 컨테이너 세 개가 running인 상태에서 없는 `RESSO_TEST_CERT_DIR`로 재현: 현재 예제 형태는 `old_eval_status=0`과 `FOLLOWUP_RAN`, 수정 형태는 status=1과 빈 stdout. 회차 경로의 old-example.out/.err 및 guarded-example.out/.err에 증거가 있다. CA와 컨테이너는 삭제하지 않았다.
- 정상 준비 뒤 `internal/federation/directory_test.go:TestDirectoryOverTLSRequiresACertificateItCanVerify`가 PASS(패키지 1.086s, SKIP 없음). 이 테스트는 미신뢰 CA 거부·실제 CA 연결·alice 실제 LDAP 인증·잘못된 CA 거부를 확인한다. 로그: federation-verification.log.

구현 순서 및 체크포인트(전부 구현자 자체 검증, 사람 승인 대기 없음):
1. [미착수] 지정된 문서 절의 실행 예제를 고친다. 아래 정상 명령을 수행하고 최종 문서 코드 블록을 회차 경로로 추출해 `bash -n <추출한 파일>`로 확인한다. 준비 실패를 보존하는지 확인한 후에만 다음 단계로 간다.
2. [미착수] 같은 절에서 선택적 정리 설명과 README 링크를 작성한다. `scripts/test-services.sh`의 실제 --stop 분기 및 README와 대조한다. 정리 명령은 실행하지 않는다.
3. [미착수] 아래 실패 프로브 및 TLS 확인을 실행한다. `git diff --check`와 `git diff --stat`로 문서 한 파일 변경인지 확인한다. 실제 결과가 계획과 다르면 과제서를 먼저 정정하고 범위를 늘리지 않는다.

정찰 추가 검증: 아래 `go test ./internal/...` 명령을 실제 준비 환경에서 실행해 exit 0 확인(httpserver 95.201s, store 61.504s, 일부 패키지 cached). 로그는 internal-verification.log. 이 비상세 출력만으로 전체 연동 테스트의 SKIP 0을 주장하지 않는다; TLS 테스트는 별도 -v 실행으로 PASS를 확인했다.

정상 검증(저장소 루트, 같은 셸):
```bash
test_env="$(scripts/test-services.sh)" &&
  eval "$test_env" &&
  go test ./internal/...
```
TLS 테스트의 실행 여부를 별도로 확인:
```bash
test_env="$(scripts/test-services.sh)" &&
  eval "$test_env" &&
  go test -race ./internal/federation -run '^TestDirectoryOverTLSRequiresACertificateItCanVerify$' -count=1 -v
```
실패 검증(세 기존 컨테이너가 running인지 먼저 `docker inspect`로 확인; 없는 경우 생성·삭제로 상태를 억지로 만들지 말고 환경 제약을 기록):
```bash
run_dir=/mnt/c/Users/USER/projects/aidev/state/runs/2026-10-09-065826-ReSSO-improve
bad_certs="$run_dir/nonexistent-certs"
test ! -e "$bad_certs" || exit 2
if (
  test_env="$(RESSO_TEST_CERT_DIR="$bad_certs" scripts/test-services.sh)" &&
    eval "$test_env" &&
    printf 'FOLLOWUP_RAN\n'
) > "$run_dir/guarded-example.out" 2> "$run_dir/guarded-example.err"; then
  echo 'FAIL: preparation unexpectedly succeeded' >&2
  exit 1
fi
test ! -s "$run_dir/guarded-example.out"
cat "$run_dir/guarded-example.err"
```
nonzero만 보고 통과시키지 말고 stderr가 실제 `ca.crt is not a readable regular file`인 것도 확인한다. 위 프로브의 마지막 printf는 후속 테스트 명령이 호출되는지 안전하게 관측하는 표식이며 스크립트나 Docker 대역이 아니다.

대안 비교 및 선택 근거:
- 선택: 문서 한 절 수정. 새로운 실행 도구 없이 기존 README 방식으로 재현된 실패를 막고 데이터 삭제 조건까지 분명히 한다.
- 차선 README 지표 정정: 위험은 같지만 이번 실패 전파 문제는 실제 명령 실행과 데이터 보존에 직접 영향을 주므로 먼저 고친다.
- 별도 공통 검증 wrapper 도입: 여러 예제를 함께 관리할 장점은 있으나 새 실행 경로와 테스트가 늘어 이번 요구에 비해 범위가 크다.
- 현상 유지: README만 읽는 독자에게는 문제가 없지만 LDAP 가이드에서 시작하는 독자의 재현된 실패가 남으므로 선택하지 않는다.

추정 근거(estimating-and-contingency): 자체 bottom-up 판단으로 문서 수정 5–8분, 안내/링크 대조 3–5분, 정상·실패 실행 5–10분, diff 검토 2–4분 = 기본 15–27분. 알려진 환경 편차를 위한 contingency 0–8분을 별도 두어 총 15–35분, 중간 확신의 주관적 범위이며 통계적 신뢰구간/약속이 아니다. 이전 회차 CA 수정은 실제 스크립트·픽스처 수정까지 포함했으므로 소요시간을 그대로 유추하지 않는다. 추가 범위용 management reserve는 배정하지 않는다. 핵심 가정은 현재 실행 중인 세 컨테이너와 정상 CA를 재사용할 수 있다는 것; 깨지면 삭제/재생성으로 확대하지 말고 환경 제약을 기록한다.

스킬 적용: 전용 Skill 도구가 노출되지 않아 로컬 headcount의 pmo/estimating-and-contingency, technology/implementation-planning, technology/solution-exploration SKILL.md를 직접 읽었다. 각각 추정 범위·별도 여유, 상태/증명/체크포인트, 대안 비교를 위에 반영했다. 외부 기준의 수치나 규범을 인용한 추정은 아니다.
