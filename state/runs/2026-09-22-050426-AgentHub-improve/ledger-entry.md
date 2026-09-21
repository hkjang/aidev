## 2026-09-22
- 선택: guide-shots 전역 설정 네 가지를 모두 백업한 뒤에만 seed·추적 캡처를 시작한다 (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: 네 전역 API 값의 HTTP·JSON·구조를 검증하고 깊게 복제한 뒤에만 seed·capture·captureTracking에 진입하도록 실제 스크립트에 전용 경계를 연결했다(af47666). 기존 코드에서 새 테스트 51건 중 41건 실패를 먼저 확인했고, 최종 신규 52건 포함 Node 109건·두 스크립트 구문 검사·go test ./internal/api ./cmd/runtime-proxy·git diff --check가 통과했다. 실제 스크립트의 transport/호출 블록과 모듈을 함께 실행해 마지막 백업 실패의 변경 0회, 정상·예외 복원, 깊은 복사 및 skip을 검증했으며 DSN 미설정으로 DB live와 실물 브라우저 촬영은 미실행; 요청된 technology 세 스킬은 callable 도구/로컬 파일을 찾지 못해 해당 반환 형식은 적용하지 못했다.
- 보류 아이디어:
  - guide-shots 복원 한 건 실패 후에도 나머지 설정 복원 (가치 3 / 위험 1 / 작업량 M)
  - guide-shots API 호출 제한 시간 (가치 3 / 위험 2 / 작업량 M)
  - guide-shots 기존 CSP 위반 기록 전체 삭제 방지 (가치 3 / 위험 2 / 작업량 M)
  - 게이트웨이 reporter.send non-2xx 보고 실패 로그 (가치 2 / 위험 2 / 작업량 S)
- 과제서: 채택 — 현재 코드의 실패 무시·빈 정책 대체·설정 누락 복원 생략과 실제 API의 GET/PUT 계약이 과제서와 일치했다.
