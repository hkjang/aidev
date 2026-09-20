## 2026-09-21
- 선택: dbexec PostgreSQL 통합 테스트의 명시적 테스트 DB 선택과 연결 실패 판정 바로잡기 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: QURIO_TEST_POSTGRES_DSN만 trim해 사용하고 미설정일 때만 Skip하도록 바꾸었으며 초기 오류 7곳을 Fatal로 전환했다; README에 폐기 DB와 bootstrap 절차를 설명했다. 회귀 테스트가 수정 전 잘못된 DSN 선택·미설정 처리·자식 프로세스 Skip/PASS를 재현했고 수정 후 통과했으며, PostgreSQL 17 격리 DB bootstrap 후 기존 7개 통합 테스트와 새 계약 테스트가 -race로 통과했다. DB 없는 dbexec 전체 테스트, go vet -tags=integration ./internal/domain/dbexec, go test ./..., go build ./..., gofmt 검사 통과; 컨테이너 제거 완료, 커밋 9a9577b.
- 보류 아이디어: gofmt 읽기 전용 검사를 make lint에 추가 (가치 2 / 위험 1 / S)
- 보류 아이디어: dbexec 고정 이름 fixture 충돌 격리 (가치 3 / 위험 2 / M)
- 보류 아이디어: Go 지시 버전과 컨테이너 Go 버전 구분 안내 (가치 1 / 위험 1 / S)
- 보류 아이디어: mail 머지 후 만료된 승인 알림 훅 연결 (가치 3 / 위험 1 / S)
- 과제서: 채택 — 현재 코드에서 전용 DSN 무시와 초기 오류 Skip 7곳이 그대로 확인되어 지정 범위와 수용 기준을 구현했다.
