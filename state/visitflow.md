# VisitFlow 자율 개선 기록

## 2026-09-02
- 선택: 위조된 X-Forwarded-For로 접속 IP를 바꿀 수 있던 문제 수정 (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `Routes()`가 chi의 `middleware.RealIP`를 무조건 사용해 모든 요청이 `X-Forwarded-For`로 `RemoteAddr`를 덮어쓸 수 있었고, 그 결과 로그인 잠금·공개 API 요청 한도를 헤더만 바꿔 우회할 수 있었으며 감사 로그·세션·동의 기록에도 위조 IP가 저장됐다. 새 `security.trusted_proxies` 설정(IP/CIDR/`private` 키워드, 기본값 빈 문자열)에 등록된 주소에서 온 요청만 헤더를 신뢰하고 전달 체인을 peer 쪽부터 거꾸로 훑어 신뢰 프록시가 아닌 첫 hop을 채택하도록 `internal/app/clientip.go`를 추가했으며, 해결된 주소를 request context에 실어 기존 `r.RemoteAddr` 사용처(감사/세션/동의 약 70곳)를 `clientIP(r)`로 통일했다. 검증은 `go vet ./...`, docker postgres:16-alpine을 띄운 `VISITFLOW_TEST_DSN` 전체 통합 테스트 3회 연속 통과(신규 단위 테스트 6개 + 위조 헤더로 요청 한도 우회를 막는 통합 테스트 1개 포함), `npm ci && npm run build`로 확인했다. 마이그레이션 0011, 관리자 설정 화면 필드, README·ADMIN_GUIDE 안내를 함께 추가했다.
- 보류 아이디어: CSV/XLSX 가져오기 파서(`visitorInputsFromRows`) 엣지케이스 단위 테스트 보강 / `bestAcceptLanguage`의 `q=0`·`q>1` 처리 정정 / `/metrics` 토큰 상수시간 비교와 요청 한도 적용 검토 / 설정 내보내기 파일을 되돌려 넣는 가져오기 경로와 스키마 검증 / 감사 로그 CSV 내보내기의 스트리밍·메모리 사용량 점검

## 2026-09-02
- 선택: CSV 내보내기의 스프레드시트 수식 주입(Formula Injection) 차단 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 감사 로그·방문 이력·통계 CSV가 셀 값을 그대로 기록해, 셀프 사전등록 방문자가 회사명이나 이름에 `=cmd|'/c calc'!A0` 같은 값을 넣으면 관리자가 내려받아 Excel로 열 때 수식으로 실행될 수 있었다. `internal/app/exports.go`에 `csvCell`(값이 `=`,`+`,`-`,`@`,tab,CR로 시작하면 작은따옴표를 앞에 붙임)과 `writeCSVRow`를 추가하고 세 내보내기의 모든 행을 이를 거쳐 쓰도록 바꿨다. 검증은 `go vet ./...`와 docker postgres:16-alpine으로 띄운 `VISITFLOW_TEST_DSN` 전체 테스트 통과, 신규 단위 테스트 2개와 방문 이력 CSV에 위험 문자열이 중화되어 나오는 통합 테스트 1개를 추가한 뒤 수정을 임시로 무력화해 세 테스트가 실제로 실패하는 것까지 확인했다. ADMIN_GUIDE에 한 문장 안내를 덧붙였다.
- 보류 아이디어: 감사 로그 CSV 내보내기가 화면의 행위자·기간 필터를 무시하고 `action`만 반영하는 문제(프런트 `exportQuery`와 서버 파라미터 불일치) / `bestAcceptLanguage`의 `q=0`·잘못된 `q` 값 처리 정정 / CSV·XLSX 가져오기 파서(`visitorInputsFromRows`) 엣지케이스 단위 테스트 보강 / 설정 내보내기 JSON을 되돌려 넣는 가져오기 경로와 스키마 검증 / 방문 이력 CSV의 50,000행 상한 초과 시 사용자에게 잘렸음을 알리는 표시
- 릴리즈: v2.6.1 (2026-09-02)
