## 2026-10-09
- 선택: QR 스캐너의 기준정보 조회 실패를 명시하고 QR 검증 뒤에도 안내 유지하기 (가치 3 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: 기준정보 Promise의 catch 누락으로 실패가 화면에 드러나지 않던 경로에 전용 referenceError 문자열을 추가하고, 별도 한국어 Alert와 처리 로비 error/helperText가 함께 읽도록 했다(프로덕션 1파일 + 테스트 1파일, 커밋 78a7206); QR 검증·카메라·scope·체크인 조건은 그대로 두었다. 영구 Playwright 2개가 실제 배포용 번들·새 PostgreSQL·실서버·Chrome으로 실패→실제 QR 검증 후 안내 유지/미처리 pageerror 없음→새로고침 복구/실제 첫 로비 자동 선택과 다른 실제 로비 수동 선택, 조회 대기/HTTP 200 빈 목록의 무오류 및 체크인 버튼 활성 상태를 확인한다. 검증은 npm ci 후 npm run lint·npm test(98 passed)·npm run build, 별도 E2E TypeScript 검사, go test ./... -count=1(DB 통합 DSN 미설정으로 SKIP), git diff --check가 통과했고 bash scripts/local-e2e.sh는 수정 전 1 failed/25 passed → 수정 후 26 passed → 제품 수정만 되돌림 1 failed/25 passed → 최종 복원 및 수동 선택 보강 후 26 passed(1.1m)였으며, 수정 후 첫 두 실행의 PostgreSQL 준비 경합 실패는 e2e-after.log와 e2e-after-retry.log에 별도로 보존했다.
- 실패 재현: `Error: expect(locator).toBeVisible() failed` / `Locator: getByRole('alert').filter({ hasText: '로비 목록을 불러오지 못했습니다. 페이지를 새로고침해 주세요.' })` / `Error: element(s) not found` — e2e-before.log 및 e2e-reverted.log 모두 새 안내 단정만 실패, 나머지 25개 통과; 제품 빌드 정상.
- 보류 아이디어: web/e2e 정규 TypeScript 검사 배선 (가치 2 / 위험 2 / 작업량 S) — 이번 독립 tsc는 통과했지만 npm lint 범위는 그대로다.
- 보류 아이디어: local-e2e.sh PostgreSQL 임시 서버 준비 확인 경합 (가치 2 / 위험 2 / 작업량 S) — 이번 두 번 실제 재현; 스크립트를 바꾸지 않은 다음 실행과 후속 두 실행은 브라우저까지 정상 진행했다.
- 보류 아이디어: 만료된 API 키 수정·회전 버튼의 활성 표시 (가치 2 / 위험 2 / 작업량 S) — 정찰 후보 유지, 이번 범위 밖이다.
- 보류 아이디어: 공백 QR 입력에서 확인 버튼이 활성인 표시 불일치 (가치 1 / 위험 1 / 작업량 S) — 정찰 후보 유지, 이번 기준정보 실패 처리에 섞지 않았다.
- 과제서: 채택 — 현재 코드에 catch가 없고 verify가 공용 error를 지워 전용 오류가 필요하다는 전제가 맞았으며 지정한 두 파일 안에서 구현·실제 UI 검증을 완료했다.
