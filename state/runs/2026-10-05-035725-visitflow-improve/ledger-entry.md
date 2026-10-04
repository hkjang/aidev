## 2026-10-05
- 선택: 승인 메모 창에서 취소해도 방문이 승인되는 문제 수정 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 목록 decide의 prompt 반환값에서 null을 빈 문자열로 바꾸던 처리를 제거하고 null이면 busy/error/POST/load 전에 즉시 반환하도록 했다(프로덕션 1파일 + 영구 Playwright 테스트 1파일, 커밋 0b5d077). 실제 로그인·새 PostgreSQL·실서버·배포용 UI 번들·Chrome으로 취소 POST 0건/PENDING_APPROVAL 유지 및 재진입 후 재승인, 빈 메모 POST 1건/{reason:""}/204/SCHEDULED/행 제거, 공백 포함 입력 메모의 요청·상세 approvalReason 보존, 반려 취소·빈값·공백 POST 0건과 유효 사유의 204/REJECTED/상세 일치를 검증했다. `bash scripts/local-e2e.sh` 수정 전 1 failed/13 passed → 수정 후 14 passed(38.9s) → 수정만 되돌림 1 failed/13 passed → 최종 복원 14 passed(38.0s), `cd web && npm run lint && npm test && npm run build` 88 passed 및 빌드 성공, 별도 e2e TypeScript 검사·`go test ./... -count=1`·`git diff --check` 통과; Go DB 통합은 VISITFLOW_TEST_DSN 미설정으로 SKIP이며 webdist 스텁 복원·커밋 후 clean 확인.
- 실패 재현: `Expected length: 0 / Received length: 1` (cancelling the memo must not POST an approval) / `Expected: "PENDING_APPROVAL" / Received: "SCHEDULED"`. 같은 방문은 화면 재진입 후 행 수도 Expected 1 / Received 0으로 실패했다. 원문 e2e-before-retry.log, 되돌림 e2e-reverted.log. 첫 실행 e2e-before.log는 PostgreSQL 초기화 임시 서버 종료와 준비 확인이 겹쳐 브라우저 이전에 실패한 인프라 문제이며 제품 실패로 세지 않았다.
- 보류 아이디어: web/e2e 정규 타입 검사 배선 (가치 2 / 위험 2 / S) — 이번 독립 tsc는 통과했지만 기존 npm lint는 e2e를 포함하지 않는다.
- 보류 아이디어: 비상 대피 명단 최초 조회 실패와 실제 0명 구별 (가치 3 / 위험 1 / S) — 코드 근거 유지, 실브라우저 실패 재현은 미수행.
- 보류 아이디어: 비상 대피 명단의 오프라인·오래된 데이터 경고를 인쇄물에도 표시 (가치 3 / 위험 1 / S) — print 영역에서 경고 제외, 인쇄 미디어 실측은 미수행.
- 보류 아이디어: local-e2e PostgreSQL 임시 서버 준비 확인 경합 (가치 2 / 위험 2 / S) — 이번 첫 실행 실패로 관찰, 스크립트 수정은 별도 과제로 남겼다.
- 과제서: 채택 — 현재 코드·실제 API 계약이 일치했고 수용 기준 1~5를 영구 실제 e2e로 확인했으며 지정한 2파일 범위를 지켰다.
