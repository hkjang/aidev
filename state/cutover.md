## 2026-09-11
- 선택: 사용자·관리자 가이드를 실제 화면 캡처가 들어간 표준 형식으로 완성 (가치 5 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: 기존 docs/USER_GUIDE.md·ADMIN_GUIDE.md(캡처 0장, 일부 내용이 코드와 불일치)를 GUIDE-STANDARD 구성으로 다시 쓰고, Playwright 기반 캡처 스크립트(scripts/guide-screenshots, 격리 데이터 파일 + 자체 dev 서버 + GUIDE_SHOT_* 전용 환경 변수)로 실제 화면 13장을 1440x900으로 찍어 docs/assets/guide/에 실었으며 공용 md2pdf로 두 PDF를 만들었다. 환경 변수 표·API 메서드·연동 규칙은 코드에서 읽어 작성했고, 운영 빌드(NODE_ENV=production)에서 Secure 쿠키 때문에 평문 HTTP LAN 주소로는 관리자 세션이 유지되지 않는 문제를 실측(LAN IP 401 / 127.0.0.1 200)으로 확인해 문서에 경고로 넣었다. 검증: eslint, tsc --noEmit, 기존 e2e 2건 통과, 캡처 스크립트 5건 통과, PDF 페이지를 렌더해 표지·표·그림 확인.
- 보류 아이디어: (1) 운영 빌드 관리자 세션 쿠키 Secure 속성을 환경 변수나 x-forwarded-proto 로 제어해 HTTP 사내망 배포에서도 관리자 로그인이 되게 한다 — 가치 5 / 위험 2 / S. (2) 삭제·상태 전파가 ID 접두사(`<부모ID>-`)에 의존해 업로드한 JSON 의 자식이 고아로 남아 삭제가 400 으로 조용히 실패한다 — parentId 기반 하위 탐색으로 바꾼다 — 가치 4 / 위험 2 / S. (3) 관리자 콘솔의 PUT 실패(400/401)를 화면에 표시하고 삭제 확인창이 두 번 뜨는 중복을 제거한다 — 가치 3 / 위험 1 / S. (4) 미사용 lib/s3.ts·amplify.yml·*_bak 파일과 사용되지 않는 reset-visitor API 정리 — 가치 2 / 위험 1 / S. (5) lib/activityData.ts validateActivityImport 단위 테스트(Node test runner) 추가 — 가치 3 / 위험 1 / S.

