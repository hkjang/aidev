- 과제: README 환경변수 계약을 필수 4개 + 선택 SHUTDOWN_DRAIN_SECONDS로 정리 (가치 2 / 위험 1 / 작업량 S)
- 왜: README.md 실행 계약은 프로세스가 읽는 환경변수가 “네 개뿐”이라고 하지만 internal/config/config.go:Load는 SHUTDOWN_DRAIN_SECONDS도 읽고, 같은 README 종료 절은 이를 사용하라고 안내한다. 필수 부트스트랩 설정과 선택 종료 대기 설정을 일관되게 설명하면 운영자가 설정 누락과 선택 튜닝을 구별할 수 있다.
- 수용 기준: 1) 실행 계약은 기존 필수 4개 이름·설명·필수성을 유지하고 별도 선택 설정으로 SHUTDOWN_DRAIN_SECONDS(초, 기본 5, 포함 범위 0~120)를 명시한다. 2) 0은 사전 drain 대기만 생략하고, 미지정·빈 값·정수가 아닌 값·범위 밖 값은 5초로 돌아간다고 설명한다. 0이 진행 중 요청 및 워커를 강제 종료한다는 의미로 쓰지 않는다. 3) .env.example에는 설명과 주석 처리된 선택 예시만 추가하고 기존 필수 설정값은 유지한다. Config 주석도 필수 4개 및 선택 종료 설정을 구분한다. 4) 기존 config 테스트 3개와 전체 Go 테스트가 통과해 필수 설정·약한 비밀번호 거절·암호화 키 길이의 기존 계약을 보존한다. 기존 테스트는 DrainSeconds 경계를 검사하지 않으므로 그 동작을 테스트로 증명했다고 주장하지 말고 Load/intFromEnv 및 main 종료 코드와 문구를 대조한다.
- 건드릴 파일: README.md:실행 계약·무중단 배포와 종료 — 필수/선택 구분 및 오류 입력의 기본값 설명; .env.example:파일 끝 — 주석 처리한 SHUTDOWN_DRAIN_SECONDS=5 예시와 0~120 안내; internal/config/config.go:Config 바로 위 주석 — “only the four” 모순 수정. 프로덕션 Go 파일 1개이며 실행문 변경은 없다.
- 검증 명령: `go test ./internal/config -count=1 -v`; `go test ./cmd/... ./internal/...`; `git diff --check`; `git diff -- README.md .env.example internal/config/config.go`. 모두 저장소 루트에서 실행한다. 앞 세 명령은 정찰에서 종료 코드 0 확인. config 테스트 3개 PASS, 전체 Go 테스트 PASS(일부 캐시 사용); KKIIT_TEST_DSN 미설정으로 DB 통합 테스트는 실행되지 않았다(기존 주문 통합 테스트를 -v로 실행해 SKIP 확인).
- 위험과 피할 것: 이번 범위는 문서와 주석이다. Load/intFromEnv/main의 동작, auth/session/migrations/workflows, compose 설정, web 및 internal/ui/dist, 릴리스 버전은 바꾸지 않는다. 문서만을 검증하기 위한 소스 문자열 테스트나 새 테스트 프레임워크를 만들지 않는다. SHUTDOWN_DRAIN_SECONDS를 필수로 만들거나 system_settings로 옮기지 않는다. 실제 .env나 전역 설정을 덮어쓰지 않는다.
- 차선 후보: Config.Load의 선택 drain 설정 경계 회귀 테스트 — README 모순이 이미 다른 변경으로 해결됐다면 internal/config/config_test.go 한 파일에 실제 Load를 호출하는 표 기반 테스트를 추가한다. 필수 환경변수 4개를 t.Setenv로 고정하고 미지정/빈 값/0/120/음수/121/비정수/주변 공백을 검증한다. intFromEnv만 직접 호출해 배선을 우회하지 않는다. 현재 테스트에 해당 사례가 없는 것은 확인했다.

범위와 선택 근거: 옵션 조회 rows.Err/Scan 실패 처리도 코드상 결함이지만 자연 HTTP→실제 DB로 재현하는 방법은 미확인이다. 오류 주입 인터페이스를 새로 만들거나 DB 스키마를 훼손해 증거를 만드는 접근은 선택하지 않는다. 승인 조건 보존은 브라우저 검증, 메일 경합은 시간·데이터 격리 설계, REQUIREMENT_PENDING은 제품 결정이 필요하다. 문서 그대로 두기도 가능하지만 같은 README 안의 직접적인 모순을 계속 유지하므로 작은 수정의 이득이 있다. 환경변수 목록을 자동 생성하는 큰 대안은 새 도구·유지 비용이 이 세 파일 변경보다 크다.

실행 순서와 체크포인트(구현 전, 모두 미착수; 사용자 지시에 따라 사람 승인 대기 없음):
1. README.md와 .env.example 문구를 바꾼다. `git diff --check` 후 `git diff -- README.md .env.example`로 필수 설정 4개 및 기존 종료 절과 일치하는지 자체 검토한다.
2. internal/config/config.go의 Config 설명 주석만 맞춘다. `go test ./internal/config -count=1 -v`로 기존 계약을 확인한 뒤 진행한다. runtime diff가 생기면 이 단계에서 범위를 다시 좁힌다.
3. `go test ./cmd/... ./internal/...`와 `git diff --check` 후 세 파일 diff를 검토한다. DB 통합 미실행을 기록하고 완료한다. 문서 작업에는 DB 기동·웹 빌드를 요구하지 않는다.

견적 근거: bottom-up으로 문구 수정 5~8분 + 주석/예시 정합성 검토 3~5분 + 기존 검증/최종 diff 검토 4~7분 = 기본 12~20분. 알려진 불확실성(문구 정합성 재수정)에 예비 3~5분을 별도로 두어 총 15~25분을 예상한다. 정찰의 판단 범위이며 통계적 신뢰구간이나 약속은 아니다. 과거 같은 README 후보는 구현 완료 기록이 없어 실측 유사 과제 시간으로 교차검증할 수 없다. 별도 관리 예비는 0분 배정이며 새 범위가 생기면 늘리지 않고 보류한다. Go 캐시가 가용하고 세 파일을 넘지 않는다는 가정이 핵심이다.

적용 스킬: 전용 Skill 도구가 없어 로컬 SKILL.md를 직접 읽었다. pmo:estimating-and-contingency의 작업 분해·범위 견적·예비 분리, technology:implementation-planning의 파일/증거/체크포인트, technology:solution-exploration의 대안 비교를 반영했다. 경로는 /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/{pmo,technology}/skills/ 아래의 각 스킬 디렉터리다. PMO references/sources.md도 읽었으며 외부 원문은 열지 않았고 외부 표준에 근거한 신뢰수준을 주장하지 않는다.
