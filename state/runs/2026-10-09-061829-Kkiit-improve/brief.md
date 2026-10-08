- 과제: Config.Load의 선택 종료 대기 환경변수 경계 회귀 테스트 (가치 2 / 위험 1 / 작업량 S)
- 왜: internal/config/config.go의 Load는 SHUTDOWN_DRAIN_SECONDS를 기본 5초, 허용 0~120초로 읽지만 config_test.go의 기존 3개 테스트는 DrainSeconds를 검증하지 않는다. 실제 Load를 호출하는 테스트로 환경변수 이름·기본값·양 끝 경계·잘못된 입력의 대체값을 고정하면 배포 시 종료 대기 설정이 조용히 달라지는 회귀를 막을 수 있다.
- 수용 기준:
  1) 환경변수가 실제로 없는 경우, 빈 문자열, 공백만, -1, 121, 비정수 abc, 소수 1.5, int 범위를 넘는 긴 정수 문자열은 Load 오류 없이 DrainSeconds=5를 반환한다.
  2) 0, 1, 5, 120은 그대로 반환하고 앞뒤 공백이 있는 " 12 \t"는 12를 반환한다. 특히 0을 기본값으로 치환하거나 120을 거절하는 회귀를 잡는다.
  3) 새 TestLoadShutdownDrainSeconds(권장 이름)는 각 서브테스트에서 실제 Load를 호출하고 err 및 cfg.DrainSeconds를 함께 확인한다. 필수 네 환경변수는 기존 TestLoadAcceptsExactlyTheBootstrapContract처럼 유효한 테스트 값으로 고정하며 외부 DB 접속은 필요 없다. intFromEnv 직접 호출만으로 대체하거나 소스 문자열을 검사하지 않는다.
  4) 프로세스 외부에서 SHUTDOWN_DRAIN_SECONDS=99를 넣고 실행해도 같은 결과다. 미지정/빈 값은 서로 다른 사례이고 테스트 종료 후 원래 환경을 복원한다. 기존 세 테스트와 전체 Go 테스트가 통과한다.
- 건드릴 파일: internal/config/config_test.go: TestLoadShutdownDrainSeconds 추가 — 표 기반 서브테스트와 필요한 os import. 프로덕션 파일 0개, 변경 파일 총 1개. 참고만 할 파일: internal/config/config.go:Load/intFromEnv, cmd/kkiit/main.go:main의 drain→Shutdown→dispatcher 대기 구간.
- 검증 명령:
  - go test ./internal/config -run '^TestLoadShutdownDrainSeconds$' -count=1 -v
  - SHUTDOWN_DRAIN_SECONDS=99 go test ./internal/config -count=1 -v
  - go test ./cmd/... ./internal/...
  - go vet ./internal/config
  - gofmt -l internal/config/config_test.go
  - git diff --check
  정찰에서 go test ./internal/config -count=1 -v(기존 3개 PASS), go test ./cmd/... ./internal/...(종료 0, 일부 캐시), go vet ./internal/config, git diff --check(무출력)를 실행했다. 새 테스트 이름의 명령과 외부 값 99 실행은 구현 후 검증용이며 아직 실행하지 않았다. 기존 TestIntegrationOrderRejectsOptionTotalPastInt64 단독 -v는 KKIIT_TEST_DSN 미설정으로 SKIP였다. 전체 Go 성공을 실제 DB 통합 성공으로 기록하지 말 것.
- 위험과 피할 것: t.Setenv를 사용하므로 부모·자식 어느 쪽에도 t.Parallel을 추가하지 않는다. 실제 미지정 사례는 해당 서브테스트에서 t.Setenv로 원래 값을 보존한 뒤 os.Unsetenv를 호출하고 오류를 확인하면 cleanup으로 복원된다. 미지정을 빈 문자열 설정으로 대신하지 않는다. 필수값 fixture는 테스트용 상수만 쓰고 호스트 비밀값을 읽거나 로그에 출력하지 않는다. README/.env.example/config.go 주석 정리는 10/08의 28d274f로 구현 완료 기록이 있는 별도 회차다. 현재 pinned main@01e6b8b에는 그 문구가 남아 있지만 같은 작업을 다시 하지 않는다. main의 종료 순서·실제 sleep·워커·auth·migrations·workflows·웹 번들·의존성 변경 금지. 이 테스트는 설정 해석과 Load 배선을 증명하며 SIGTERM의 실제 경과시간이나 서버 종료 완료를 증명하지 않는다.
- 차선 후보: Config.Load 암호화 키의 Base64/hex 입력 계약 회귀 테스트 (가치 2 / 위험 1 / S) — 1순위 테스트가 이미 추가돼 공백이 없어진 경우에만 config_test.go 하나에서 진행. 유효한 32바이트 키를 Base64/64자리 hex로 각각 제공해 실제 Load의 EncryptionKey가 원본 바이트와 일치하는지, 둘의 앞뒤 공백이 허용되는지, 길이가 틀리거나 인코딩이 깨진 키는 오류인지 검증한다. 기존 hex 입력은 약한 비밀번호 테스트에만 쓰여 성공 디코딩을 증명하지 않는다. 암호 알고리즘·키 회전 정책·프로덕션 코드는 바꾸지 않는다. 검증은 go test ./internal/config -count=1 -v 및 위 전체 Go 명령이다.

범위와 대안 판단:
- 실제 Load를 통한 환경 계약 테스트를 선택한다. intFromEnv만 직접 테스트하는 대안은 짧지만 변수 이름이나 Load에서의 연결 누락을 잡지 못한다.
- 프로세스를 실행해 SIGTERM부터 종료까지 재는 대안은 사용자 관찰에 더 가깝지만 DB·워커·실시간 대기까지 필요해 이번 설정 해석 공백보다 범위가 크다. 설정 테스트가 실제 종료 통합 검증을 대체한다고 주장하지 않는다.
- 그대로 두는 대안은 비용 0이지만 이미 설명된 운영 설정의 0·120 및 fallback을 고정할 증거가 계속 없다.
- 승인 조건 보존은 가치 3이지만 브라우저 재현과 정책 판단이 필요하다. 옵션 rows.Err 처리와 delivery_days 상한은 자연 HTTP→DB 실패 증거가 미확인이다. 현재는 계약·검증 비용이 명확한 이 과제를 먼저 한다.

구현 순서와 체크포인트(모두 미착수):
1. [ ] config_test.go에 유효한 필수값과 0/120/미지정 사례부터 추가. 위 새 테스트 단독 명령에서 최소 3개 서브테스트 PASS를 확인하고 다음 단계로 간다. 사람 승인 대기 없음.
2. [ ] 나머지 잘못된 입력·공백·정상값을 표에 추가하고 환경 복원을 확인. 외부 값 99를 넣은 config 전체 명령으로 실제 환경 오염에 독립적인지 확인한다. 사람 승인 대기 없음.
3. [ ] 전체 Go 테스트·vet·포맷·diff를 검증하고 한 파일만 변경됐는지 확인한다. 예상과 코드가 다르면 과제서/회차 노트의 전제를 수정하고 범위를 넓히지 않는다. 기존 로직은 계약대로이므로 새 회귀 테스트가 수정 전 통과하는 것이 정상이다.

추정 근거:
- bottom-up 기본 작업 25분: fixture/미지정 복원 설계 5분 + 13개 입력 사례 작성 12분 + 명령 실행/결과 검토 8분. 알려진 불확실성인 환경 cleanup·명령 실행 변동에 contingency 5~10분을 별도 배정하여 총 30~35분을 예상한다.
- 교차 점검은 13개 사례 × 약 1분 + 준비/검토 12분 = 약 25분이다. 분당 생산성은 측정 이력이 없는 정찰자 판단이며 과거 회차 소요시간으로 가장하지 않는다. 45분 내 종료 가능성은 정성적으로 높지만 통계적 신뢰구간은 산출하지 않았다.
- 관리 예비비는 0분: 새 기능/발견된 무관 결함은 이번 과제에 흡수하지 않고 ideas에 남긴다. 핵심 가정은 현재 Load/intFromEnv 계약 유지와 Go 도구 가용이며, 정찰에서 기존 테스트 실행으로 후자는 확인했다.

적용 스킬: Skill 전용 도구는 이 세션에 없어 로컬 원본을 읽고 적용했다.
- [pmo:estimating-and-contingency](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md): 작업 분해, 근거·범위, contingency/관리 예비비 분리. references/sources.md도 확인했으며 외부 기관의 정량 추정치를 사용하지 않았다.
- [technology:implementation-planning](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/implementation-planning/SKILL.md): 파일·증명 명령·체크포인트 및 미착수 상태.
- [technology:solution-exploration](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/solution-exploration/SKILL.md): 서로 다른 대안과 핵심 가정 비교.
