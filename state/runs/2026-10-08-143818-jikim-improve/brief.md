- 과제: CSP 리포트의 8KiB 초과 본문을 잘라서 유효한 보고서로 기록하는 문제 수정 (가치 3 / 위험 1 / 작업량 S)
- 왜: `receiveCSPReport`는 `io.LimitReader(r.Body, maxCSPReportBytes)`로 앞 8192바이트만 읽어서, 그 부분이 JSON과 공백이면 뒤에 초과 데이터나 잘못된 문자가 있어도 정상 보고서로 기록한다. 상한 초과 보고서를 기록 없이 204로 처리하면 관리자 진단 목록에 잘린 요청이 정상 보고서로 올라오는 일을 막으며 기존 응답 계약을 유지한다.
- 수용 기준: 1) 추적 활성 상태에서 정상 보고서와 공백으로 정확히 8192바이트까지 채운 유효 JSON은 204·빈 응답을 반환하고 관리자 조회에 1건/count=1로 표시된다. 2) 같은 JSON을 8193바이트 이상으로 패딩한 요청 및 유효 JSON+공백으로 앞 8192바이트를 채우고 뒤에 잘못된 문자 `x`를 붙인 요청은 204·빈 응답을 반환하되 관리자 목록에 기록되지 않는다. `ContentLength=-1`인 경우도 같아야 하며 Content-Length 검사만으로 해결하지 않는다. 3) 실제 `Server.routes`로 POST→실제 Recorder→인증된 관리자 GET을 왕복하는 테스트에서 초과 케이스가 수정 전 실패하고 수정 후 통과해야 한다. 기존 추적 꺼짐·설정 장애·비정상 JSON·권한 경계 테스트도 유지한다.
- 건드릴 파일: `internal/httpapi/tracking.go:receiveCSPReport`(140행 부근) — `io.LimitReader`를 상한+1로 읽고 `len(body)>maxCSPReportBytes`를 JSON 파싱 전에 거절한다. `internal/httpapi/tracking_test.go` — 기존 `trackingServer`, `momentoConfig`, `TestPolicyReportsAreOnlyRecordedWhileTrackingIsOn` 패턴으로 `TestCSPReportBodySizeBoundary` 테이블 테스트를 추가한다. 프로덕션 1개 + 테스트 1개로 제한한다.
- 검증 명령: `go test ./internal/httpapi/ -run 'CSPReportBodySizeBoundary|PolicyReports|Tracking' -count=1 -v`; `go test ./... -count=1`; `go vet ./...`; `gofmt -l internal/httpapi/tracking.go internal/httpapi/tracking_test.go`. 최종 저장소 표준 검증은 `bash scripts/verify.sh`(npm 설치·프런트·Docker Compose 도구 필요, 정찰에서는 실행하지 않음).
- 위험과 피할 것: `defer w.WriteHeader(http.StatusNoContent)`와 설정 gate를 유지하고 400/413이나 본문/감사/로그를 새로 내보내지 않는다. 상한+1까지만 읽고 전체 본문을 무제한 읽지 않는다. 이는 입력 완전성 수정이며 rate limit·DoS 해결이라고 주장하지 않는다(현재 코드도 읽는 양과 Recorder 100건을 제한한다). 기존 `ReportingActive`, CSP 정책, nonce 파서, Recorder, Momento 프록시, auth/migrations/workflows는 변경하지 않는다. 스캔/보고 원문을 감사 details로 옮기지 않는다. 버전·CHANGELOG·의존성 변경 제외.
- 차선 후보: Momento 프록시 자격증명 제거 왕복 테스트 공백 보강 (가치 3 / 위험 1 / 작업량 S) — 1순위 경계 실패가 재현되지 않을 때만 `tracking_test.go:TestMomentoProxyForwardsWithoutCredentials`에서 실제 collector가 Cookie뿐 아니라 Authorization·X-Vault-Token도 받지 않음을 검증한다. 현재 구현은 세 헤더를 제거하지만 기존 테스트는 Cookie만 읽고 확인한다. 실제 httptest collector와 routes를 유지하며 소스 문자열 검사·가짜 Transport로 대체하지 않는다. `go test ./internal/httpapi/ -run MomentoProxy -count=1`로 검증한다.

범위와 근거: HEAD `aac28af`, v0.2.31. `server.go:150`은 인증 없는 POST 리포트, 151행은 관리자 GET이다. `tracking_test.go:32`의 trackingServer는 실제 routes·securityHeaders와 임시 SPA, 실제 Recorder를 사용하며 설정 읽기만 기존 loader로 바꾼다. `TestPolicyReportsAreRecordedOnceAndListedForAdministrators`(196행 부근)와 `TestPolicyReportsAreOnlyRecordedWhileTrackingIsOn`(261행 부근)의 `sessionResolver`+`tokenRequest`를 재사용한다. 후자의 listViolations는 함수 내부 지역 함수라 다른 테스트에서 직접 호출할 수 없다; 짧은 지역 헬퍼를 새 테스트 안에 두면 된다. `internal/tracking/violations.go:Record/List`까지 확인했으며 기록 여부는 JSON `data` 길이·count로 검사한다. 새 seam이나 DB는 필요 없다.

재현 입력: 기존 테스트의 정상 csp-report JSON 문자열을 `report`로 두고 `report + strings.Repeat(" ", maxCSPReportBytes-len(report))`를 경계 본문으로 만든다. 여기에 공백 1개(전체 유효 JSON이지만 크기 초과) 또는 `x`(전체 JSON 자체가 무효)를 덧붙인다. 각 케이스마다 별도 trackingServer로 Recorder 상태를 격리한다. ContentLength=-1 케이스는 HTTP 요청 객체의 해당 필드를 바꾸되 실제 Body는 그대로 둔다. 소스상 실패 원인은 확인했으나 정찰은 저장소 코드·테스트를 추가하지 않았으므로 이 신규 실패의 실행 재현은 미확인이다. 구현 첫 단계에서 반드시 확인한다.

접근 비교: (선택) 최대 N+1바이트 읽기+길이 검사: 기존 204 구조 안에서 최소 변경, 알려진/미지 길이 모두 처리. (대안) MaxBytesReader: 가능하지만 크기 오류 분류·204 유지 확인이 추가되므로 이번에는 필요 없다. (보류) 무수정: 메모리 한도는 이미 있지만 초과 본문을 기록하는 입력 계약 결함이 남는다. 가장 큰 가정은 8192바이트가 읽기 상한뿐 아니라 수용 가능한 전체 보고서 상한이어야 한다는 것(상수 주석과 입력 파싱 구조에 근거한 판단)이다.

실행 순서·체크포인트(구현자는 완료 상태를 갱신):
1. [done] 테스트 파일만 추가하고 첫 검증 명령으로 정상/정확한 경계는 통과, 초과 두 종류만 실패하는지 확인한다. 예상과 다르면 근거를 과제서에 수정하고 차선 판정; 사람 승인 체크포인트 없음.
2. [done] receiveCSPReport의 제한 읽기·길이 검사만 수정하고 같은 명령으로 전체 통과를 확인한다. 응답 204와 관리자 최종 조회를 함께 확인; 사람 승인 체크포인트 없음.
3. [done] 전체 Go test/vet·gofmt 및 최종 verify를 수행한다. 실패를 기존 환경 문제와 변경 회귀로 구분해 기록하고 범위를 넓히지 않는다; 사람 승인 체크포인트 없음.

산정: bottom-up으로 재현/테스트 10~15분 + 수정 3~5분 + 검증/기록 7~15분 = 기본 20~35분, 기존 하네스 적응이라는 알려진 불확실성 예비 5분을 별도로 더해 25~40분 예상이다. 이는 중간 확신의 작업 추정이며 통계적 확률·완료 약속은 아니다. 관리 예비는 0분(이번 작업에 미배정), DB·프런트 신규 설치 지연은 범위 밖으로 기록한다. 이전 한 파일 입력 검증 수정 회차와 규모는 비슷하지만 소요시간 실측 기록이 없어 별도의 수치 유추 추정은 하지 않는다. 첫 실패 재현 뒤 재산정한다. 45분을 넘기면 추가 기능을 넣지 않는다.

정찰 검증: `go test ./... -count=1` exit 0, httpapi 0.610초. 기존 테스트의 통과이며 신규 경계 결함 부재의 증거가 아니다. PostgreSQL 실제 연동·프런트·CI 실행은 이번 정찰에서 미검증이다. README·docs/guides/compatibility.md·api-guide.md·CONTRIBUTING·CI/release 워크플로·verify 스크립트·최근 git log 30건을 확인했다. 저장소 파일 목록에서 CLAUDE.md/AGENTS.md 및 별도 로드맵 파일은 발견하지 못했고, README 기능 표와 FeaturePage의 로드맵 표시를 참고했다.

적용 스킬: Skill 도구는 현재 노출 도구 목록에 없어 로컬 원본을 직접 읽었다. `pmo:estimating-and-contingency`, `technology:implementation-planning`, `technology:solution-exploration`의 분해 산정·예비 분리·대안 비교·단계별 증거/체크포인트를 반영했다. 원본 위치는 `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/{pmo,technology}/skills/<스킬명>/SKILL.md`이다. PMO sources 목록도 확인했으나 외부 비용평가 기준이나 수치 주장은 사용하지 않았다.
