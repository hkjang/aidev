## 2026-09-22
- 선택: MCP 긴 한 줄 응답 절단 시 UTF-8 글자 경계 보존 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: cutAtBoundary의 fallback이 이미 잘린 window 대신 원본 text와 limit를 runeSafeCut에 넘기도록 한 줄 수정했다(commit 1dd6363). 운영 함수의 모든 바이트 경계·절단 없는 입력과 SQLite→ServeHTTP tools/call(read-file)→handleReadFile→formatFileContent→finishCall→JSON 경로에서 2·3·4바이트/ASCII 혼합, 3000~3011 연속 예산, 캐시 미스·히트의 수정 전 실패와 수정 후 통과를 확인했으며 U+FFFD 부재·원문 접두부·캐시 감사 기록·경계 선호·60% 활용·펜스·Notes를 검사했다. 지정 MCP 비캐시/race·전체 sqlite_fts5 테스트·vet·build·gofmt 검증은 모두 exit 0이고, 요청한 Skill 도구 및 technology 스킬 3개는 도구/로컬 검색에서 발견되지 않아 해당 절차·반환 형식은 미확인이다.
- 보류 아이디어:
  - mcp.cacheKey의 호출자 ACL 슬라이스 복사 (가치 2 / 위험 1 / 작업량 S)
  - finishCall 진단 추가 뒤 최종 응답 크기 검증 (가치 2 / 위험 2 / 작업량 M)
  - clampResponse 공지 예약치 320B 부족 재현 (가치 2 / 위험 2 / 작업량 S)
  - ReadFile 자체 192KiB 본문 제한의 UTF-8 경계 런타임 검증 (가치 3 / 위험 2 / 작업량 S)
- 과제서: 채택 — 현재 코드의 잘못된 호출과 실제 함수·HTTP 캐시 미스/히트의 문자 손상을 재현했으며 지정된 한 줄 수정으로 회귀 테스트를 통과했다.
