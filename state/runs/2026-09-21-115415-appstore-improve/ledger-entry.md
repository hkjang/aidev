## 2026-09-21
- 선택: LoginPage 기본 로그인 방식 3종과 복구용 관리자 토글 회귀 검증 (가치 2 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 기존 테스트가 bootstrapAvailable=false로 고정되어 놓친 SSO 전용·Bootstrap 전용·병행 설치의 로그인 UI를 실제 AuthProvider/API 경유 Vitest 5건과 production bundle E2E 2건으로 보강했다. 세션과 설정 조회 완료 뒤 단언하도록 하여 로딩 중 폼으로 잘못 통과하지 않게 했고, 로그인 분기를 각각 훼손한 mutation 3종에서 새 테스트가 실패하고 원본 복원 뒤 통과함을 확인했다(제품 코드 변경 없음). npm test 65개, lint·Prettier·build·오프라인/환경/문서 검사·Go race 테스트(캐시)·Go build·desktop/mobile 전체 E2E 71 passed/1 skipped 후 721ad4a로 커밋했다.
- 보류 아이디어: 즐겨찾기 개수·100개 밖 앱 페이지 탐색 (가치 3 / 위험 2 / 작업량 M) — 목록 계약 전체를 함께 검증해야 함.
- 보류 아이디어: USER_GUIDE의 기기 간 즐겨찾기 동기화 안내 정정 (가치 2 / 위험 1 / 작업량 S) — 3.6·4.2와 PDF를 함께 수정.
- 보류 아이디어: E2E public config override의 기본값 덮어쓰기 순서 정리 (가치 2 / 위험 1 / 작업량 S) — 신규, 공용 fixture 영향 확인 필요.
- 보류 아이디어: 로그인 방식이 모두 없는 설치의 브라우저 안내 검증 (가치 2 / 위험 1 / 작업량 S) — 신규, Vitest 외 실제 번들 검증 보강 후보.
- 스킬: headcount/plugins/technology/skills의 completion-verification·systematic-debugging·test-driven-development 원본을 읽음. Skill 호출 도구는 제공되지 않았음. 기존 정상 동작의 테스트 보강이므로 최초 테스트는 통과했으며 제품 버그의 TDD red 재현으로 주장하지 않음; mutation 실패와 원본 복원 통과로 회귀 감지력을 확인.
- 검증 한계: 초기 E2E는 임시 HOME에 Chromium 실행 파일이 없어 시작 실패, 설치 후 전체 재실행 통과. 기존 모바일 전용 테스트의 desktop 제외 1건이며 retry 없이 71개 통과. 실제 DB/Keycloak·Docker 이미지 smoke는 실행하지 않았고 DSN 미설정으로 DB 통합 테스트는 제외됨. E2E는 HTTP fixture와 실제 번들/Chromium을 사용. 빌드 산출물·버전·릴리즈·원격 변경 없음.
