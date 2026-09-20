## 2026-09-20
- 선택: usePaging.resetPageAndGet의 페이지 초기화 후 중복 목록 조회 제거 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: 초기화 횟수를 page와 함께 관찰하여 직접 조회한 1페이지의 watcher 중복 호출을 없애고, 같은 tick에서 다른 페이지로 이동하는 조회 및 원래 값/Promise 반환을 유지했다. 실제 Vue ref/watch/nextTick/effectScope와 native Promise 테스트로 수정 전 5건 실패를 확인했고, npm ci 후 페이지 테스트 16건·전체 424건 및 build:dev가 통과했다. 회사 technology 스킬/Skill 도구는 발견하지 못했으며 별도 로컬 superpowers systematic-debugging·test-driven-development 지침을 대체 사용했다(회사 스킬 반환 형식 준수는 주장하지 않음).
- 보류 아이디어:
  - useAppList 비배열 SIDEMENU_APP_LIST 캐시 방어 (가치 3 / 위험 2 / 작업량 S)
  - 테스트 가이드의 없는 문서 링크 정리 (가치 2 / 위험 1 / 작업량 S)
  - OCR 미사용 시 Header 폴링 조정 (가치 3 / 위험 3 / 작업량 M; 서버 계약 확인 선행)
  - globalLoading 병렬 요청 참조 카운트 (가치 3 / 위험 3 / 작업량 S; 호출 짝 감사 선행)
- 과제서: 채택 — 실제 코드와 수정 전 Vue 테스트에서 직접 호출 및 watcher의 중복 조회를 확인했고 지정된 두 파일만 변경했다.
