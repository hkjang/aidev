## 2026-10-08
- 선택: 서비스 HTML의 사이트맵 누락을 빌드 검사에서 동적으로 감지 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 기존 필수 17개 주소·404 배제 검사를 유지하고, 실제 dist/services/<slug>.html마다 확장자·끝 슬래시 없는 정확한 loc를 요구하는 7줄을 추가했다(커밋 2513dc4, 작성자 hkjang, 프로덕션 1개·테스트 1개 파일). 임시 dist 복사본에서 실제 CLI를 실행하는 회귀는 최초 5 통과·1 실패였고, 잘못된 loc 변형 2건 추가 후 5 통과·3 실패 → 수정 후 8 통과였으며 새 루프 제거 시 동일 3건 실패·복원 후 8 통과로 원인을 확인했다. npm ci → npm run build, node --check scripts/verify-build.mjs, node --test tests/verify-build-sitemap.test.mjs, node scripts/verify-build.mjs, npm run test:build → npx playwright install chromium → npm test --silent 모두 exit 0(Playwright 77 통과·기존 폭 조건 4 건너뜀, 30.5초); 정상 빌드·정확한 loc·비서비스 HTML/중첩 HTML/이미지 제외·/about 누락·404 포함을 검증하고 빌드 산출물은 커밋하지 않았다.
- 실패 재현: `not ok 2 - 고정 목록에 없는 서비스 HTML의 loc 누락을 거절한다` / `사이트맵 오류를 거절해야 함: https://www.irumx.app/services/scout-future` — 실제 CLI는 `✓ 모두 통과`, 실제 exit 0 / 기대 exit 1(node:test exit 1, assets/sitemap-red.log).
- 보류 아이디어: 나머지 inquiry 순수 함수 계약·경계값 단위 테스트 (가치 3 / 위험 1 / 작업량 S)
  - README·배포 문서의 Worker 경로와 메일 관문 설명 정합성 (가치 2 / 위험 1 / 작업량 S)
  - gzip;q=0 압축 협상 회귀 검증 (가치 3 / 위험 3 / 작업량 S)
  - canonical URL과 실제 HTML 경로 불일치 감지 (가치 3 / 위험 1 / 작업량 S)
- 과제서: 채택 — 현재 고정 목록과 공개 서비스 생성 방식이 정찰 근거와 일치했고, 지정한 두 파일 안에서 실제 CLI로 세 수용 기준을 검증했다.
