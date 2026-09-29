## 2026-09-29
- 선택: 공백뿐인 발표 제목도 공간 이름으로 대체하기 (가치 3 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: Service.compile의 제목을 TrimSpace한 뒤 기존 공간 이름 기본값을 적용해 preview 표지와 outline 최상위 제목을 복원했다. 기존 실제 DB/auth/chi 하네스로 생략·빈 값·ASCII 공백·탭/개행·유니코드 공백·일반/앞뒤 공백 제목 7개를 검증하고 실제 presentation.Preview의 생각 본문·순서·개수 및 SlideSources 첫 위치 2를 확인했다. 격리 PostgreSQL 17.11에서 지정 통합 시험 PASS(0.420s), presentation 시험 PASS(0.039s), POSTGRES_DSN="$UMM_TEST_DSN" go test -p 1 ./... 전체 PASS, go vet ./..., go build ./cmd/umm(출력은 회차 폴더 후 삭제), git diff --check 통과; 수정 한 줄 되돌림도 동일 공백 실패를 재현했다.
- 실패 재현: `presentation_integration_test.go:974: preview title = "", want "회고 주기 재검토"` / `presentation_integration_test.go:996: outline must start with "# 회고 주기 재검토\n\n": "### 회고 주기를 격주로 줄여 보자\n\n- 주기가 짧으면 논의가 얕아진다\n\n"` — 수정 전 공백 3종 실패, 생략/빈 값/일반/앞뒤 공백 제목 통과. 원문은 title-red.log에 보존.
- 보류 아이디어: exportOutline 다운로드 이름에 공간 이름 담기 (가치 2 / 위험 1 / S).
- 보류 아이디어: 업로드 라벨 경로의 마지막 조각 처리 계약 검토 (가치 2 / 위험 2 / S).
- 보류 아이디어: 빈 공간 실시간 유입 뒤 초기 fit 자리 기억 재현 (가치 2 / 위험 2 / M).
- 보류 아이디어: make test-go를 CI와 같은 DB 패키지 직렬 실행으로 맞추기 (가치 3 / 위험 1 / S).
- 과제서: 채택 — 현재 코드의 fallback 전 공백 미정리와 두 HTTP 경로의 제목 소실이 실제 DB 시험에서 그대로 재현되어 지정된 제품 한 줄만 수정했다.
