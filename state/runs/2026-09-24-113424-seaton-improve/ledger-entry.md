## 2026-09-24
- 선택: 좌석 상세에서 직원 소속과 지정 구역 구분 (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공 (커밋 e6f4f70)
- 요약: 상세의 '조직' 행이 좌석에 지정된 구역을 먼저 보여 남의 팀 구역에 앉은 사람이 도면 색·툴팁(seatOrgId·seatSpeech 는 착석자 소속 우선)과 다른 조직으로 읽혔고, 소속·근무지를 왼쪽 검색 결과에서 찾아 검색 없이 좌석을 고르면 비어 보였다. '조직'을 employeeOrganizationName 으로 바꾸고 지정 구역은 버리지 않고 '지정 구역' 행으로 분리해 불일치를 함께 알리며(빈 좌석에도 표시), listSeats 에 employeeWorkplace 를 실어 selectedEmployee 를 없앴다. 검증은 실제 Docker 이미지 + PostgreSQL 16 으로 역검증부터 했다 — vitest 는 옛 규칙을 먼저 구현해 `expected '영업팀' to be '개발팀'` 로 붉게 만든 뒤 고쳤고, E2E 는 표시만 남기고 값 로직만 되돌린 이미지에서 상세가 실제로 '영업팀'을 보이는 것과 employeeWorkplace 부재를 확인한 뒤, 고친 이미지에서 새 spec 2건을 포함해 좌석 관련 8개 파일 31건과 login·admin·api-keys 16건이 모두 통과했다. go test/vet/gofmt 무출력, tsc, Vitest 106건, vite build 통과. USER_GUIDE 에 두 조직을 설명하는 절을 넣고 HTML 만 다시 구웠다(PDF 는 미머지 브랜치와의 이진 충돌 때문에 이전 회차와 같은 이유로 생략).
- 보류 아이디어: 로그인 화면 SSO 단추가 깊은 링크 returnTo 를 들고 가게 (3/1/S — LoginPage 미확인) / PATCH /users/{id} 가 없는 id 에도 204 인지 curl 로 재확인 (2/1/S) / 상세 패널 외 화면(왼쪽 검색·범례)에도 접근 가능한 region 이름 붙이기 (2/1/S — 이번에 좌석 상세에만 붙였다) / CI 에 guide-shots 스모크 추가 (2/2/S) / build-docs.py 파서 단위 테스트 (2/1/S)
- 과제서: 채택 — 프로필이 남긴 결함(상세 '조직'이 지정 구역 우선, selectedEmployee 가 검색 결과 의존)이 지금 코드와 정확히 일치했고, 같은 뿌리인 '근무지'까지 서버 필드 하나로 함께 닫았다.
