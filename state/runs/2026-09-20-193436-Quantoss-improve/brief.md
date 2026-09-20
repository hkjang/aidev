- 과제: Config.Validate에서 gap_reclaim 설정의 유효 범위 검증 (가치 3 / 위험 1 / 작업량 S)
- 왜: `FromEnv()`는 네 갭 설정을 읽고 `Validate()`를 호출하지만 현재 검증에는 이 필드가 없어, 예를 들어 `GapMin > GapMax`인 불가능한 신호 범위도 시작 단계에서 통과한다. 잘못된 설정을 환경변수 이름과 함께 한 번에 보고하면 신호가 나오지 않는 원인을 운영 후에 찾는 일을 줄인다.
- 수용 기준:
  1) `GapMin`, `GapMax`, `GapPullMin`, `GapVolMult`는 유한수이고 `0 < GapMin < GapMax`, `GapPullMin >= 0`, `GapVolMult > 0`을 만족해야 한다. 각 필드의 NaN/±Inf도 거부하며 범위 오류에는 해당 `QUANTOSS_GAP_*` 이름을 포함한다. 기존 `bad` 누적 보고 방식을 유지한다.
  2) `Default().Validate()`와 정상 환경변수 설정은 통과하고, `GapPullMin=0`도 허용한다. 기존 다른 전략 파라미터처럼 전략 이름과 무관하게 검증한다(FromEnv 이후 ForMarket의 USStrategy/PaperStrategy 선택이 있으므로 현재 Strategy만 조건으로 삼지 않는다). 기본값 0.015/0.08/0.01/1.0 및 정상 전략 동작은 바꾸지 않는다.
  3) 테이블 테스트는 최소값 0/음수, 최대값 0/음수, 최소=최대/최소>최대, 되돌림 음수, 거래량 0/음수, 네 필드의 NaN/±Inf를 실제 `Config.Validate`로 검증한다. `t.Setenv`와 임시 작업 디렉터리에서 실제 `FromEnv()`를 호출하여 네 환경변수 각각의 오류 전파 및 정상값 반영을 검증한다. 여러 독립 오류를 넣었을 때 모든 키가 한 오류에 나오는지도 확인한다. 소스 문자열 검사나 검증 대역으로 대신하지 않는다.
- 건드릴 파일:
  - `internal/config/config.go:Validate`(650행부터) — 전략 설정 검증 근처에 네 필드의 유한수/범위/상호 관계 검사 추가. `bad`로 누적하고 기존 한국어 메시지 형식 사용. 일반 `pos`/`nonNeg`/`envF`를 전역적으로 강화하지 말고 이 네 필드만 검사한다.
  - `internal/config/config_test.go:TestValidateRejectsDangerousValues`, `TestValidateReportsAllProblems`, `TestFromEnvValidates`, `TestFromEnvParsesEverything` — 기존 패턴 참고; 독립된 갭 전용 테이블/환경변수 테스트 추가 권장. 프로세스 환경·작업 디렉터리를 복구하고 `t.Parallel`은 쓰지 않는다.
  - 읽기 참고: `internal/config/config.go:Default`(216행 기본값), `FromEnv`(517~524행 로딩, 644행 Validate 호출), `internal/config/market_test.go:TestForMarketStrategyOverride`, `internal/strategy/research.go:GapReclaim.OnBars`(52행 갭, 72행 되돌림, 76행 거래량 조건). 전략 파일은 수정하지 않는다.
- 검증 명령: 저장소 루트에서 `go test -count=1 ./internal/config` → `go vet ./...` → `go build ./...` → `go test -count=1 ./...` → `gofmt -l internal/config/config.go internal/config/config_test.go`(출력 없음). 정찰에서 전체 테스트·vet·build·gofmt 검사가 통과했고 config 0.015s, broker 7.775s였다(전체 벽시계는 별도 계측하지 않음).
- 위험과 피할 것: auth/토큰, broker 주문, migrations, workflows, 스윙 실행기, VersionHash 및 기존 데이터는 건드리지 않는다. GapMin==GapMax 거부는 이번 계약이며 현재 OnBars의 양끝 포함 비교 자체는 바꾸지 않는다. 설정을 자동 보정하거나 임의 상한(예: GapMax<1)을 추가하지 않는다. NaN은 단순 <= 비교로 잡히지 않으므로 명시적으로 검사한다. 실API·라이브 CLI·`-tags zzdbg` 실행 금지. 새 제한 때문에 잘못된 갭 값이 있던 비갭 전략 실행도 오류가 날 수 있으므로 오류 키를 분명히 한다; 배포 환경의 실제 값은 미확인이다.
- 차선 후보: `scripts/check.sh` 로컬 검증 진입점 추가 (가치 3 / 위험 1 / S) — 1순위가 이미 다른 변경으로 해결된 경우만 선택. `scripts/check_dashboard.sh`는 별도 브라우저 검사이며 일반 Go 검사 스크립트는 없다. 저장소 루트를 기준으로 gofmt 미준수 시 실패, vet/build/test -race를 순차 실행하고 첫 실패를 전파한다. 자동 포맷·전역 설정 변경·workflow 추가는 하지 않는다.

실행 순서·추정: 테스트/경계 계약 10분 → Validate 구현 10분 → 실제 FromEnv 통합과 전체 검증 10분, 예비 15분(총 45분). 테스트를 먼저 추가해 현재 코드에서 실패하고 수정 후 통과하는지 확인한다. 범위 초과 시 다른 아이디어를 끼워 넣지 않는다.
대안 판단: 전략 OnBars에서 값을 보정하면 설정 오타가 숨겨지고, envF 전역 변경은 다른 모든 수치 설정에 영향을 준다. 가장 좁은 공통 진입점인 Validate만 보강한다.
스킬 제약: 요청된 `pmo:estimating-and-contingency`, `technology:implementation-planning`, `technology:solution-exploration`의 Skill 도구/skills.list·read가 제공되지 않았고 로컬 스킬 검색에서도 찾지 못했다. 고유 절차·반환 형식은 미확인으로, 위 추정·예비시간·대안 비교·구현 계획은 사용자 절차에 따라 직접 작성했다.
