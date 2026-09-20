## 2026-09-21
- 선택: make test에서 PostgreSQL 통합 테스트 생략을 명확히 알리기 (가치 2 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: Makefile 첫 recipe에서 미설정·빈 값·공백 DSN에 WARN을 한 번 알리고 기존 테스트 명령을 유지했으며 README에 npm ci와 테스트 DB 준비·검증 범위를 문서화했다. 회차 폴더 verify_make.py가 실제 make/go/npm을 실행해 변경 전 및 경고 제거 시 실패, 변경 후 미설정·빈 값·공백·탭/개행에서 경고 1회와 종료 0, 유효 PostgreSQL 16에서 경고 0회와 종료 0, 잘못된 DSN에서 경고 0회와 종료 2를 확인했다. 두 지정 DB 테스트는 -count=1 -v로 SKIP 없이 PASS했고 웹 94건, backend/runner go vet, npm run build도 성공했다.
- 보류 아이디어: 전체 모드 업로드 스트리밍 (가치 3 / 위험 4 / L) — staging 설계 필요.
- 보류 아이디어: SSE 500행 페이지 즉시 조회 (가치 2 / 위험 2 / M) — 실제 DB/HTTP 검증 필요.
- 보류 아이디어: vendor 청크 분할 (가치 2 / 위험 3 / M) — 현재 617.30 kB.
- 보류 아이디어: make vet 웹 타입 검사 (가치 2 / 위험 1 / S) — 기존 브랜치 머지 이후.
- 과제서: 채택 — 안내 부재와 네 fixture의 TrimSpace/Skip이 현재 코드와 일치하여 지정된 두 파일만 변경했다.
