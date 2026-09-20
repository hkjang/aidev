## 2026-09-20
- 선택: 서버 CSV 보고서의 공백·제어 문자 뒤 수식 접두사 보호를 목록 CSV와 일치시키기 (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: csvSafe의 선행 문자 판정을 기존 웹의 ECMAScript 공백+C0 계약에 맞추고 원문 앞에만 작은따옴표를 추가했다(74f07c7). 공유 JSON 83사례와 PostgreSQL 17 격리 스키마의 실제 testApp→Routes HTTP 생성·CSV 다운로드로 수정 전 실패를 재현한 뒤 Go1.26.7 race 테스트 2개(하위83사례, skip 없음), Node26.9.0 웹93개, go vet·Go 빌드·원본312파일 해시·diff 검사를 통과했으며 JSON 원문·BOM·열·헤더·팀 격리·미인증 거부도 확인했다. 실제 Excel/LibreOffice 실행, 전체 Go 스위트, CI·배포는 미검증이며 지정 technology 스킬은 없어 로컬 superpowers의 TDD·디버깅·완료 전 검증 절차를 대신 적용했다.
- 보류 아이디어:
  - 긴 검색어 목록 주소 복사의 500자 초과 부분 보존 정책 (3/1/S)
  - OIDC Go/TS return_to 의도된 차이의 공유 벡터 검증 (3/2/M)
  - 서버 CSV 개인 키 권한 교집합의 HTTP 회귀 (3/1/S)
  - CSV CRLF 본문 원문 바이트와 디코딩 계약 검증 (2/1/S)
- 과제서: 채택 — 현재 csvSafe와 웹 판정의 차이가 과제서와 일치했고 실제 서버 HTTP 실패로 확정했다.
