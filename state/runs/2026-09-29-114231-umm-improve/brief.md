- 과제: 공백뿐인 발표 제목도 공간 이름으로 대체하기 (가치 3 / 위험 1 / 작업량 M)
- 왜: `Service.compile`은 제목이 정확히 빈 문자열일 때만 공간 이름을 넣고, 공백뿐인 값은 그대로 `Compile`에 보내므로 다듬어진 제목이 비어 발표 표지와 문서 차례 제목이 사라진다. 공통 서비스 입구에서 공백을 정리한 뒤 기본값을 적용하면 미리보기와 문서 차례가 빈 제목과 같은 결과를 낸다.
- 수용 기준: 1) 이름 있는 공간에서 제목 생략·빈 값·ASCII 공백·탭/개행·유니코드 공백뿐인 값을 보낼 때 preview의 Storyline.Title은 공간 이름이고 Source에 그 이름의 표지가 정확히 한 번 나온다. 2) 같은 입력을 outline HTTP 경로로 보내면 같은 공간 이름의 최상위 제목이 나오고, `  임원 보고  `는 두 경로 모두 `임원 보고`가 된다. 생각 본문·순서·개수는 기본 요청과 동일하다. 3) 실제 DB store와 HTTP 핸들러→presentations→Service.compile 경로를 쓰는 회귀 시험이 수정 전 공백 입력에서 실패하고 수정 후 통과해야 한다. 응답을 실제 `presentation.Preview`로 역직렬화하여 `Storyline.SlideSources()`의 첫 콘텐츠 위치가 표지 다음(2)임도 확인한다. 이는 회귀 방지이며 현재 출처 번호 밀림을 주장하는 시험은 아니다.
- 건드릴 파일: `internal/presentation/service.go:Service.compile`(336행 부근) — `title := strings.TrimSpace(req.Title)` 후 기존 빈 값 대체 유지(프로덕션 1개); `internal/httpapi/presentation_integration_test.go:presentationAPI, TestPreviewWorksBeforePtiumIsConnectedIntegration` — 기존 실제 store/auth/chi 하네스를 재사용하고 exportOutline GET 라우트를 추가하여 제목 입력 표 시험 작성. 필요하면 동일 패키지 신규 테스트 파일 한 개로 분리하되 별도 fake Spaces/Service는 만들지 않는다.
- 검증 명령: 저장소 루트에서 `go test ./internal/presentation -count=1`; `POSTGRES_DSN="$UMM_TEST_DSN" go test ./internal/httpapi -run 'Test.*(PresentationTitle|PreviewWorksBeforePtium)' -count=1 -v`(신규 이름을 `TestPresentationTitleFallbackIntegration`으로); `POSTGRES_DSN="$UMM_TEST_DSN" go test -p 1 ./...`; `go vet ./...`; `git diff --check`. UMM_TEST_DSN에는 구현자가 준비한 격리 PostgreSQL 17의 실제 DSN을 넣고 반드시 비어 있지 않은지 확인한다. 현재 환경에는 umm 전용 DB 컨테이너가 없으며 기존 다른 프로젝트 DB를 임의 사용하지 않는다. 정찰 실행: presentation 패키지 PASS(0.032초), 기존 HTTP preview 시험은 DSN 부재로 SKIP(통과 증거 아님). 새 시험의 실행 및 DB 재현은 구현자 몫이다.
- 위험과 피할 것: auth/migrations/workflows·프런트·버전·백업 Markdown 파서를 건드리지 않는다. 제목 정규화를 각 HTTP/MCP 입구에 복제하지 않고 공통 서비스에서 해결한다. 현재 `storyline.go:Compile`은 이미 Title을 TrimSpace하므로 공백 입력에서 표지와 SlideSources가 같이 없는 상태다. 과거 기록의 '출처 한 칸 밀림'은 현재 서비스 경로에서는 성립하지 않으므로 SlideSources/WriteSource를 수정하지 않는다. 순수 Compile에 빈 제목을 넘겼을 때 표지 없이 시작하는 기존 계약도 유지한다. Ptium 실제 생성/외부 AI 호출은 범위 밖이다.
- 차선 후보: 문서 차례 다운로드 이름에 공간 이름 담기 — `internal/httpapi/export_handlers.go:exportOutline`의 고정 `umm-outline.md`를 `handoff_handlers.go:handoffFilename` 기반 이름으로 변경. 실제 Content-Disposition은 UTF-8 filename*와 안전한 fallback을 쓰는 기존 방식을 확인해 재사용하고 헤더 파싱 시험으로 검증할 것(본문 제목 고침과 한 회차에 묶지 않는다).

실행 순서와 체크포인트(아직 모두 미착수, 사람 승인 대기 없음):
1. 위 HTTP 표 시험을 추가하고 지정한 통합 명령으로 공백 사례의 실패를 확인한다. 기존 빈 제목/일반 제목 사례는 통과해야 한다. DB를 준비하지 못하면 skip을 성공으로 적지 말고 과제서 상태를 갱신한다.
2. compile의 제목 초기화만 수정한 뒤 같은 통합 명령과 presentation 패키지를 실행한다. 두 출력 경로가 일치하면 다음 단계로 간다.
3. 전체 Go 검증 및 diff를 확인한다. 실패가 범위 밖 환경 문제면 근거를 기록하고 기능을 확장하지 않는다. 구현 완료 여부와 과제서 채택 판정을 회차 노트에 남긴다.

대안 판단: 공통 서비스에서 정규화(채택)는 모든 호출자가 같은 기본값을 받으며 1개 제품 파일로 끝난다. HTTP 쿼리에서만 정규화하면 JSON/MCP 경로가 남아 탈락한다. 공백 제목을 400으로 거부하면 기존 빈 제목 기본값과 다른 계약이 생겨 탈락한다. 현상 유지는 구현 비용은 없지만 표지·문서 제목 누락을 그대로 둔다. 가장 큰 가정은 공백 제목을 '의도적 무표지'가 아닌 제목 생략으로 취급한다는 것인데, 기존 Compile의 trim과 빈 제목의 공간 이름 fallback이 그 근거다.

견적 근거: bottom-up으로 시험/DB 준비 12–18분 + 한 줄 수정/인접 검증 3–5분 + 전체 검증/기록 10–14분 = 기본 25–37분, 알려진 DB 준비 변동에 contingency 최대 8분을 별도 둔다(총 25–45분, 통계적 신뢰구간이 아닌 중간 확신의 판단 범위). 과거 rewind 회차와 제품 한 곳+실제 배선 시험이라는 크기는 유사하나 시간 측정 기록은 없어 수치 보정에는 쓰지 않았다. 관리 예비비는 미배정이며 범위 확대에 쓰지 않는다. DB 준비가 길어지면 실제 남은 시간을 기준으로 재견적한다.

적용 스킬: 로컬 `pmo:estimating-and-contingency`, `technology:implementation-planning`, `technology:solution-exploration`의 SKILL.md를 읽고 범위·대안·검증/체크포인트·견적/예비비를 반영했다. 전용 Skill 도구는 이 세션에 없어 파일 읽기로 대신했다. 추정치는 정찰자의 코드 범위 판단이며 외부 원가 지침에 근거한 확률 주장은 하지 않는다.
