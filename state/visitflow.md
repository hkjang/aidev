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

## 2026-09-03
- 선택: 감사 로그 CSV 내보내기가 화면의 행위자·기간 필터를 무시하는 문제 수정 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 관리자 → 감사 로그 화면은 이벤트 접두어·행위자·시작·종료로 필터링하지만 `exportAuditLogsCSV`는 `action`만 읽었고 프런트의 `exportQuery`도 `action`만 실어 보내, 특정 담당자나 기간으로 좁혀 내려받아도 최근 10,000행 전체가 담기는 잘못된 근거 자료가 만들어졌다. `internal/app/admin.go`에 `auditLogFilters`와 `parseAuditLogFilters`를 두어 목록과 내보내기가 같은 파라미터를 같은 규칙으로 파싱하게 하고, 내보내기 쿼리에 행위자·기간 조건과 목록과 동일한 `a.id DESC` 정렬을 적용했으며, `audit.export` 감사 기록에 실제 적용된 범위를 남기고 `AdminPage.tsx`의 다운로드 링크가 `buildQuery()`를 재사용하도록 했다. 검증은 `go vet ./...`, `npm ci && npm run build`, docker postgres:16-alpine을 띄운 `VISITFLOW_TEST_DSN` 전체 테스트 통과이며, 신규 통합 테스트 1개(행위자·from·to 필터 각각)와 단위 테스트 2개를 추가한 뒤 수정을 임시로 되돌려 통합 테스트가 실제로 실패하는 것까지 확인했다. API_AND_MCP 문서에 한 줄 안내를 덧붙였다.
- 보류 아이디어: `bestAcceptLanguage`의 `q=0`·`q>1`·잘못된 `q` 값 처리 정정 / CSV·XLSX 가져오기 파서(`visitorInputsFromRows`) 엣지케이스 단위 테스트 보강 / `/metrics` 토큰 상수시간 비교와 요청 한도 적용 검토 / 설정 내보내기 JSON을 되돌려 넣는 가져오기 경로와 스키마 검증 / 방문 이력 CSV의 50,000행 상한 초과 시 사용자에게 잘렸음을 알리는 표시

## 2026-09-03
- 선택: 방문 이력 CSV 내보내기가 방문 목록의 검색·상태 필터를 무시하는 문제 수정 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 관리자 → 방문·방문자 관리 화면은 방문번호·회사·담당자·방문자 이름/전화로 목록을 좁히지만 `exportVisitsCSV`는 `days`만 읽었고 화면의 다운로드 버튼도 `?days=90`으로 고정돼 있어, 한 회사만 조회한 상태로 내려받아도 90일치 전체 방문이 담겨 잘못된 근거 자료가 되고 조회하지 않은 방문자의 개인정보까지 함께 내보내졌다. `internal/app/exports.go`에 `visitExportFilters`/`parseVisitExportFilters`(days 검증, 알 수 없는 status는 무시, q 트림)를 두고 내보내기 쿼리에 목록과 같은 검색 조건을 참가자 EXISTS 서브쿼리로 적용해 필터에 걸린 방문은 참가자 전원이 온전히 나오게 했으며, `visit.export` 감사 기록에 실제 적용 범위를 남기고 `AdminPage.tsx`의 링크가 현재 검색어를 싣도록 했다. 검증은 `go vet ./...`, docker postgres:16-alpine을 띄운 `VISITFLOW_TEST_DSN` 전체 테스트 통과, `npm ci && npm run build`이며, 신규 통합 테스트 1개(회사·이름·전화·상태·잘못된 상태)와 단위 테스트 2개를 추가한 뒤 필터를 임시로 무력화해 통합 테스트가 실제로 실패하는 것까지 확인했다. API_AND_MCP 문서에 한 줄 안내를 덧붙였다.
- 보류 아이디어: `bestAcceptLanguage`의 `q=0`(수용 불가)·`q>1` 처리 정정 / CSV·XLSX 가져오기 파서(`visitorInputsFromRows`) 엣지케이스 단위 테스트 보강 / 방문 이력 CSV의 50,000행 상한 초과 시 잘렸음을 사용자에게 알리는 표시 / 설정 내보내기 JSON을 되돌려 넣는 가져오기 경로와 스키마 검증 / 통계 CSV가 사업장 타임존 버킷과 DB `CURRENT_DATE` 기준 날짜 축을 섞어 쓰는 문제 점검

## 2026-09-03
- 선택: 통계 추이 그래프·CSV가 사업장 타임존 버킷과 DB `CURRENT_DATE` 날짜 축을 섞어 써 "오늘"이 통째로 누락되는 문제 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `statistics`와 `exportStatisticsCSV`는 방문을 `(v.start_at AT TIME ZONE si.timezone)::date`로 버킷팅하면서 날짜 축은 DB 세션 기준 `CURRENT_DATE`(배포 컨테이너에서는 UTC)로 만들어, 기본값인 Asia/Seoul 사업장에서는 축이 9시간 모자랐다. 그 결과 매일 자정~09:00 사이에는 대시보드 타일이 "오늘"로 세는 날의 열이 축에 아예 없어 그날 예약된 방문이 그래프와 통계 CSV에서 함께 사라졌다. `admin.go`에 공용 `statisticsTodayCTE`(모든 사업장의 현지 오늘 중 가장 늦은 날짜, 사업장이 없으면 `CURRENT_DATE`)를 두고 화면 쿼리와 내보내기 쿼리가 축과 버킷 조회 구간을 함께 이 기준으로 잡도록 바꿨다. 검증은 `go vet ./...`, docker postgres:16-alpine을 띄운 `VISITFLOW_TEST_DSN` 전체 테스트 통과, `npm ci && npm run build`이며, 세션 날짜와 다른 ±12시간 존을 골라 사업장 시간대를 바꾸고 그 사업장의 현지 오늘 정오에 방문을 만든 뒤 축의 마지막 날짜와 CSV 행을 확인하는 통합 테스트 1개를 추가한 뒤 수정을 임시로 되돌려 실제로 실패하는 것까지 확인했다. API_AND_MCP 문서에 한 줄 안내를 덧붙였다.
- 보류 아이디어: `bestAcceptLanguage`의 `q=0`(수용 불가)·`q>1`·잘못된 `q` 값 처리 정정 / CSV·XLSX 가져오기 파서(`visitorInputsFromRows`) 엣지케이스 단위 테스트 보강 / 방문 이력 CSV의 50,000행 상한 초과 시 잘렸음을 사용자에게 알리는 표시 / 설정 내보내기 JSON을 되돌려 넣는 가져오기 경로와 스키마 검증 / 통계 요약·부문별 집계도 `CURRENT_DATE-days` 대신 사업장 시간대 기준 구간을 쓰도록 통일
- 릴리즈: v2.6.2 (2026-09-03)

## 2026-09-04
- 선택: 통계 요약 타일·부문별 집계가 추이 그래프와 다른 기간을 세는 문제 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 지난 세션에서 추이 그래프·통계 CSV의 날짜 축만 사업장 시간대로 옮겼는데, 같은 화면의 요약 타일(방문자·입실·미방문·취소·본인 사전등록·평균 체류/사전 신청)과 부서·사업장·방문 유형·입실 시간대·신청 경로 집계는 여전히 `v.start_at>=CURRENT_DATE-$1::int`(DB 세션 기준 자정, 배포 컨테이너에서는 UTC)로 걸러, Asia/Seoul 사업장에서는 그래프 첫 열보다 하루 전 09:00부터 세었다. 그래서 하나의 기간 선택기 아래 나란히 놓인 타일 숫자가 바로 옆 막대 합보다 크고, 초과분은 막대가 아예 없는 날에서 왔다. `admin.go`에 `statisticsSpanCTE`(사업장 현지 오늘까지 `days`일)와 `statisticsSpanWhere(column)`(인덱스가 살아 있도록 느슨한 timestamp 전치 필터 + `(column AT TIME ZONE si.timezone)::date BETWEEN` 정밀 조건)를 두고, 추이·요약·다섯 개 부문별 집계·통계 CSV가 모두 같은 구간을 읽도록 통일했으며 `bySource`·`byVisitType`·요약 쿼리에는 필요한 `sites` 조인을 추가했다. 검증은 `go vet ./...`, docker postgres:16-alpine을 띄운 `VISITFLOW_TEST_DSN` 전체 테스트 통과, `npm ci && npm run build`이며, 세션 날짜와 다른 ±12시간 존에서 구간 안(현지 오늘 정오)과 구간 밖(현지 오늘-7일 23:00, 옛 경계에는 걸리던 값) 방문을 하나씩 만들어 그래프 합·요약·네 부문별 집계가 모두 1인지 보는 통합 테스트 1개와 단위 테스트 1개를 추가한 뒤, 필터를 옛 `CURRENT_DATE-days`로 되돌려 통합 테스트가 실제로 실패(요약 2 vs 그래프 1)하는 것까지 확인했다. 테스트 시간대 설정 코드는 `moveSitesOffSessionDate` 헬퍼로 묶었고 API_AND_MCP 문서에 한 문장을 덧붙였다.
- 보류 아이디어: `bestAcceptLanguage`의 `q=0`(수용 불가)·`q>1`·잘못된 `q` 값 처리 정정 / CSV·XLSX 가져오기 파서(`visitorInputsFromRows`) 엣지케이스 단위 테스트 보강 / 방문 이력 CSV의 50,000행 상한 초과 시 잘렸음을 사용자에게 알리는 표시 / 설정 내보내기 JSON을 되돌려 넣는 가져오기 경로와 스키마 검증 / 관리자 대시보드 상단 타일("오늘"·"미방문")도 통계 화면과 같은 사업장 시간대 헬퍼를 쓰도록 정리
- 릴리즈: v2.6.3 (2026-09-04)
