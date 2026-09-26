- 과제: 단순 모드 통합 테스트 픽스처를 프로덕션 New() 생성자로 통일하기 (가치 2 / 위험 1 / 작업량 S)
- 왜: `simple_batch_test.go:newSimpleBatchFixture`는 `&Server{store,log}`를 반환해 `streams`·`aiActive` 등의 초기화를 빠뜨리고, `simple_stream_test.go:newSimpleStreamFixture`는 이를 피하려 같은 store로 서버를 다시 만든다. 공통 픽스처를 실제 생성자로 바꾸고 우회를 제거하면 후속 HTTP 테스트의 nil map 패닉을 예방하고 기존 테스트가 프로덕션 초기화 경로를 검증한다.
- 수용 기준:
  1) newSimpleBatchFixture가 기존 schema 격리·마이그레이션·seed·Cleanup 순서를 유지하면서 `New(st, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), BuildInfo{}, "")`로 서버를 반환한다.
  2) newSimpleStreamFixture는 공통 픽스처가 반환한 동일 서버에 simpleStreamSession을 만들고 그 서버를 그대로 반환한다. 별도 New 호출, map 수동 주입, acquireLogStream의 nil 방어 추가로 문제를 가리지 않는다.
  3) 실제 PostgreSQL에서 기존 TestSimpleRunLogStreamDrainsFullPagesWithoutWaiting 및 TestSimpleRunLogStreamWaitsWhenAPageIsNotFull이 실제 세션 쿠키→Handler()→스트림 경로로 SKIP 없이 통과한다. 변경 후 공통 픽스처 반환만 옛 리터럴로 잠시 되돌리면 해당 테스트가 패닉 또는 HTTP 실패로 실패하고, 복구 후 통과해야 한다. 단순 map!=nil 검사나 소스 문자열 검사는 증거가 아니다.
  4) backend 전체 테스트가 통과한다. TEST_POSTGRES_DSN이 없는 실행의 PASS를 통합 검증 성공으로 보고하지 않는다.
- 건드릴 파일:
  - backend/internal/server/simple_batch_test.go:newSimpleBatchFixture — 반환만 New() 기반으로 변경. logger와 DB 수명은 유지.
  - backend/internal/server/simple_stream_test.go:newSimpleStreamFixture — 서버 재생성 제거, 동일 서버에 세션 생성; 설명 주석을 최종 배선에 맞춤. 기존 실제 HTTP 테스트를 회귀 증거로 사용하므로 중복 테스트 추가는 불필요.
  - 프로덕션 코드 변경 0개, 테스트 파일 2개.
- 검증 명령:
  - 저장소 루트에서 `(cd backend && go test ./internal/server -run 'TestSimpleRunLogStream|TestUploadHasFailedPackages' -count=1 -v)`
  - `(cd backend && go test ./... -count=1)`
  - `(cd backend && go vet ./...)`
  - `gofmt -l backend/internal/server/simple_batch_test.go backend/internal/server/simple_stream_test.go` (출력 없음)
  - 모든 DB 검증에는 schema 생성 권한이 있는 PostgreSQL 16의 TEST_POSTGRES_DSN을 환경으로 제공한다. CI의 .github/workflows/ci.yml에 PostgreSQL 16 서비스 설정이 있다. 정찰에서는 첫 명령을 실제 실행했고 종료 0이었지만 DSN 미설정으로 세 테스트 모두 SKIP했다. DB가 있는 결과·패닉 재현은 미확인이다.
- 위험과 피할 것: 프로덕션의 결함이라고 주장하지 말 것(New는 이미 안전하다). auth·session 구현·migrations·workflows·VERSION·웹·릴리즈/빌드 스크립트는 범위 밖이다. vault=nil은 기존 스트림 픽스처와 같은 조건이며 암호화 기능 테스트로 범위를 확장하지 않는다. DB 소유권과 Cleanup을 옮기거나 두 번 Close하지 않는다. 손으로 만든 Server 대역·map 주입을 회귀 증거로 쓰지 않는다. HELD enum 및 사람 반려 접근은 재시도하지 않는다.
- 차선 후보: SimpleDeployPage 업로드 라이브 로그의 영구 연결 종료 안내 — `SimpleDeployPage.tsx`의 activeRunId effect는 log/end만 듣는다. CLOSED 오류만 경고하고 해당 실행 상세로 안내하는 최소 변경으로 제한; CONNECTING 경고·자동 재시도·로그 저장 방식 개편은 제외. 화면 테스트 도달 비용 때문에 1순위에서 제외했다.

실행 계획과 체크포인트 (구현자용; 모두 미착수)
1. [ ] PostgreSQL 준비와 위 특정 테스트의 SKIP 없는 기준선 확보. 실패하면 환경 문제부터 기록하고 과제서의 가정을 수정한다. 사람 승인 체크포인트 없음.
2. [ ] 위 두 테스트 파일을 함께 변경하여 공통 New 경로를 사용하게 하고 같은 특정 테스트 실행. 단계 종료 시 시스템이 통과 상태여야 한다. 사람 승인 체크포인트 없음.
3. [ ] 공통 반환만 리터럴로 임시 되돌리는 반증 실험 후 반드시 복구; 특정 테스트 재통과, 전체 backend 테스트·vet·gofmt 확인. 실제 실행 결과와 SKIP 여부를 인계한다. 사람 승인 체크포인트 없음.

대안 비교·산정 근거
- 선택: 공통 New 사용은 새 런타임 부품 없이 2개 테스트 파일로 초기화 차이를 없앤다.
- 유지: 현재 스트림 우회는 동작하지만 다른 공통 픽스처 이용 테스트에 초기화 누락이 남는다.
- 수동 map 초기화: 당장은 가능하나 다음 생성자 필드가 추가될 때 다시 어긋나므로 제외한다. 모든 서버 픽스처 통합은 범위가 커 제외한다.
- 핵심 가정: 기존 공통 픽스처 호출자는 미초기화 필드 자체에 의존하지 않는다. 전체 backend 테스트로 확인할 것.
- bottom-up 예상: 환경/기준선 5–10분, 두 반환 경로 변경 3–5분, 반증·전체 검증 12–20분 = 기본 20–35분. 알려진 DB 준비 변동에 예비 5–10분을 별도 배정(총 25–45분, 주관적 중간 확신; 통계적 신뢰구간 아님). 경영 예비·새 범위는 배정하지 않는다.
- 유사 사례: 9/24 실제 DB·Handler 스트림 검증이 사용됐지만 당시 소요 시간은 제공되지 않았다. 따라서 시간 기반 유사추정은 미확인이고 수치 교차 검증은 할 수 없다. DB 준비가 예비를 넘으면 범위를 늘리지 말고 미검증 사유를 남긴다.
- 적용 스킬: pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration. 전용 Skill 도구는 없어서 /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/{pmo,technology}/skills/ 아래 SKILL.md 원문을 읽었다. pmo references/sources.md도 확인했으며 외부 비용·편익 수치나 권위 자료 인용 없이 이 저장소 범위의 작업 추정만 작성했다.
