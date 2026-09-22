- 과제: bootstrap 암호의 bcrypt 72바이트 상한을 DB 초기화 전에 검증 (가치 3 / 위험 1 / 작업량 S)
- 왜: config.Load는 최소 12문자만 검사하므로 ASCII 73자·한글 25자 암호도 통과하고, main은 DB 연결과 Migrate 뒤 EnsureBootstrapAdmin의 bcrypt 단계에서야 실패한다. 실제 해시 함수의 제한을 설정 단계에서 명확히 알려 주면 잘못된 배포 설정으로 DB 초기화를 시도하지 않고 바로 수정할 수 있다.
- 수용 기준: 1) BOOTSTRAP_ADMIN_PASSWORD가 UTF-8 기준 72바이트를 넘으면 Load가 변수명과 “at most 72 bytes”가 포함된 오류를 반환하고, 암호 원문은 오류에 포함하지 않는다. 기존 최소 12 rune 검사와 누락 변수 검사는 유지하며 암호를 자르거나 TrimSpace·정규화하지 않는다. 2) 12~72바이트의 유효한 ASCII 및 최소 12문자·최대 72바이트인 다국어 암호는 원문 그대로 Config.BootstrapPassword에 담긴다. ASCII 72자와 한글 24자는 허용하고 ASCII 73자·한글 25자·혼합 73바이트는 거부한다. 3) 기존 config.Load를 직접 실행하는 환경변수 기반 테스트로 위 경계와 11문자 다국어의 최소 길이 거부를 증명하고, 실제 cmd/igame 기동 경로에서 초과 암호+잘못된 DSN을 주면 DB 파싱 오류보다 먼저 configuration error가 출력됨을 확인한다. 올바른 72바이트 암호+같은 DSN은 기존처럼 startup failed/parse postgres DSN까지 도달해야 한다.
- 건드릴 파일: internal/config/config.go:Load — 최소 문자 수 검사 다음에 len(c.BootstrapPassword)>72 검사 추가; internal/config/config_test.go:TestLoadRejectsWeakBootstrapPassword 및 새 경계 테스트 — t.Setenv로 실제 Load 실행, 반환 암호 보존·오류의 원문 미포함 확인; README.md:빠른 시작 환경변수 표 — 최소 12문자·최대 72 UTF-8 바이트 명시; docs/offline-install.md:bootstrap 암호 설명 — 기존 16자 이상 권장과 최대 72바이트를 함께 설명. cmd/igame/main.go:main과 internal/database/database.go:EnsureBootstrapAdmin은 읽어서 확인한 배선이며 수정하지 않는다.
- 검증 명령: `go test ./internal/config -count=1`; `go test ./cmd/... ./internal/... ./migrations/...`; `go vet ./cmd/... ./internal/... ./migrations/...`; `bash scripts/check-release-contract.sh`; `git diff --check`. 아래 실제 기동 검사도 실행한다.
- 위험과 피할 것: auth.go·관리자 비밀번호 변경 정책·migrations/*.sql·.github/workflows·해시 알고리즘·의존성·VERSION은 범위 밖이다. 이 과제는 이미 bcrypt가 받을 수 없는 bootstrap 입력의 실패 위치와 설명만 개선한다. 72는 문자 수가 아니라 바이트 수이며 암호 자르기/사전 해싱/공백 제거를 하지 않는다. 기존 DB라도 EnsureBootstrapAdmin은 매번 해시를 계산하므로 기존 암호를 덮어쓰는 변경은 불필요하다. 최소 길이 미달·키 오류 등 기존 계약을 유지한다. .env나 전역 설정은 쓰지 않고 테스트 하위 프로세스 환경만 설정한다. 테스트 대역이나 소스 문자열 검사를 배선 증거로 사용하지 않는다. 지난 audit-release 작업은 main에 없더라도 재구현하지 않는다.
- 차선 후보: README의 RealmGuard 서버 재현 설명 정합성 개선 — README RealmGuard 문단의 “완전한 서버 게임 시뮬레이션은 아닙니다”를 internal/api/realmguard.go의 replayRealmGuardBattle 호출과 docs/realmguard.md의 server_replay_v1 및 보조 telemetry 검증 설명에 맞춘다. 같은 README 끝의 서비스 버전 0.7.17도 VERSION 0.7.18에 맞추되 게임 콘텐츠 0.3.1/0.4.0은 유지한다. 1순위가 이미 다른 변경에 반영됐음이 확인된 경우에만 선택한다.

실제 확인과 실행 순서:
- 기준은 main@abd8579, VERSION 0.7.18이다. config.go:Load, database.go:EnsureBootstrapAdmin, cmd/igame/main.go:main을 직접 읽었다.
- 수정 전 ASCII 73바이트와 한글 75바이트를 실제 `go run ./cmd/igame`에 주니 모두 configuration error가 아니라 startup failed/parse postgres DSN으로 종료했다. 일부러 파싱 불가 DSN을 사용했으므로 DB에 연결하거나 마이그레이션하지 않았다.
- 잠긴 x/crypto v0.55.0 소스의 GenerateFromPassword는 len(password)>72일 때 ErrPasswordTooLong을 반환한다. `go test golang.org/x/crypto/bcrypt -run '^TestPasswordTooLong$' -count=1`도 통과했다. 정상 DB에서 전체 bootstrap 실패는 이번 정찰에서 미실행이다.
- 정찰에서 Go 전체 테스트와 릴리즈 계약 검사 통과. IGAME_TEST_DSN 미설정으로 PG 테스트는 skip; Web/SDK node_modules가 없어서 프런트 테스트 미실행. 외부 감사·원격 CI 결과는 이번에 확인하지 않았다.
- 예상 30분(경계 테스트 10분, 구현·문서 5분, 기동 및 전체 검증 10분, 여유 5분). 45분 내 완료하며 DB fixture·PDF 재생성·인증 정책 확장으로 범위를 늘리지 않는다.

수정 후 기동 검증(저장소 루트에서 실행; 정찰에서 같은 경로의 수정 전 실패를 확인):
```bash
python3 - <<'CHECK'
import os, subprocess
cases = [('ascii72', 'x'*72, False), ('ascii73', 'x'*73, True),
         ('korean24', '가'*24, False), ('korean25', '가'*25, True)]
for name, password, reject in cases:
    env = os.environ.copy()
    env.update(POSTGRES_DSN='postgres://[invalid', BOOTSTRAP_ADMIN='scout-test',
               BOOTSTRAP_ADMIN_PASSWORD=password, ENCRYPTION_KEY='01234567890123456789012345678901')
    result = subprocess.run(['go', 'run', './cmd/igame'], env=env,
                            capture_output=True, text=True, timeout=60)
    output = result.stdout + result.stderr
    assert result.returncode != 0, name
    assert password not in output, name
    if reject:
        assert 'configuration error' in output and 'BOOTSTRAP_ADMIN_PASSWORD' in output and '72 bytes' in output, name
        assert 'parse postgres DSN' not in output, name
    else:
        assert 'startup failed' in output and 'parse postgres DSN' in output, name
    print(name, 'PASS')
CHECK
```

스킬: 요청된 pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration은 현재 Skill/skills 도구 목록에 없고 ~/.codex, ~/.claude, aidev 아래 SKILL.md 검색에서도 찾지 못했다. 절차·반환 형식은 미확인이고 적용했다고 주장하지 않는다. 사용자 지정 형식에 후보 비교·실행 계획·시간 여유·차선 조건을 명시했다.
