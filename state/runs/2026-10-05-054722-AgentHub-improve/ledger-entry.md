## 2026-10-05
- 선택: Momento·Matomo 제공자 주소에서 CSP로 전달하는 출처 길이를 제한한다 (가치 3 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: 커밋 6a7c0f1. 프로덕션 tracking.go 한 파일에 MaxProviderOriginRunes=300과 정규화된 두 URL의 originOf 결과 검사만 추가해 비활성·미선택·프록시에서도 과대 출처를 거절하며, URL 전체·긴 경로·기존 불완전 입력 조건은 유지하고 제공자 무제한 주석을 현행화했다. 테스트 두 파일에서 ASCII·한국어 300/301룬 경계와 모드별 검사, 긴 경로·쿼리·포트 및 입력 불변을 고정했고, 8000개의 a 호스트가 만드는 24339바이트 헤더와 두 검증 함수의 거절, 최대 출처의 903룬 증가를 확인했다. 수정 전 실패 → 수정 후 통과 → 검사 제거 시 재실패 → 복구를 확인했으며 go test ./internal/tracking ./internal/api (마지막 tracking cached / api 1.697s), go test -race ./cmd/... ./internal/... 및 git diff --check·gofmt 확인 통과; DSN이 없어 DB live는 미실행이며 API 검증 함수 호출은 실제 HTTP/DB 저장 증명이 아니다.
- 실패 재현: `tracking_test.go:194: Momento origin of 301 runes was accepted in selected mode` / `tracking_test.go:513: API validation refused=false, want true (header 24339 bytes): <nil>` (프로덕션 수정 전 실제 출력; 두 제공자 모두 실패).
- 보류 아이디어: 복원 실패가 guide-shots의 problems 요약을 건너뛰는 문제 (2/1/S).
  콘솔 설정 화면에 추적 상한 안내 (2/1/S; 제공자 300룬도 안내 대상).
  guide-shots 추적 캡처의 기존 CSP 위반 전체 삭제 방지 (3/2/M).
  web/scripts Node 회귀 테스트를 CI 기본 검증에 포함 (3/2/S).
- 과제서: 채택 — 현재 Validate의 제공자 길이 검사 누락과 originOf→PolicySources→pagePolicy의 세 지시문 증폭이 일치했고, 지정 과대 입력의 24339바이트를 새 테스트로 재측정했다.
