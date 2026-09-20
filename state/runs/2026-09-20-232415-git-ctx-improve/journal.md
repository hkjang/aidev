# 회차 노트 2026-09-20-232415-git-ctx-improve — git-ctx
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 23:24] base pinned — main@cf3b598
- [러너 23:24] autonomy release — 

## 정찰 노트
- curl 마스킹의 명령어·앞 옵션 소실을 두 입력으로 재현하여 선택; 인증 보호 경로와 반려 이력의 파서 확장보다 위험이 낮고 45분 내 범위가 명확하다.
- cf3b598에는 이전 성공 기록의 일부 변경이 없다. ${VAR}/Authorization/OAuth 재구현 금지; 향후 curl placeholder 콜백 병합 시 캡처 번호 주의.
- contentsecurity·mcp·indexer·search 테스트 통과, 전체 suite/race 및 실제 색인→검색 E2E는 미확인. 구현자는 SQLite ReadFile 경로와 전체 문자열 비교로 증명할 것.
- 요청한 3개 스킬/Skill 도구를 찾지 못해 절차는 미확인. 신규 후보는 YAML 리스트 형제 필드 과마스킹(재현)과 finishCall 예산 이후 진단 추가(코드 확인); 프로필의 기준 불일치를 갱신했다.
- [러너 23:29] scout done — curl 자격증명 마스킹 시 명령어와 앞쪽 옵션을 보존 (가치 2 / 위험 1 / 작업량 S)

## 구현 노트
- 4f3f2c8: curlUserRE가 curl부터 인증 플래그 뒤 공백까지 캡처하도록 수정; 명령어·옵션·URL·공백 보존, 인증값만 치환.
- Revision edeca363cffe → ac7dde4ccaec; 기존 Policy.Revision 배선에 자동 반영. 생산 search/indexer 변경 없음.
- Sanitize 전체 출력/멱등성/줄 수 및 실제 SQLite→ReadFile(Origin=index) 회귀 테스트: 수정 전 실패, 수정 후 통과; 기존 줄 수 테스트 유지.
- 검증 통과: 지정 4개 패키지, contentsecurity/search race, 전체 go test -tags sqlite_fts5 ./..., go vet ./..., go build -tags sqlite_fts5 ./..., gofmt(빈 출력), diff --check.
- 미검증: PostgreSQL·pgvector·Vault 환경 변수 미설정; 원격 서비스 및 전체 indexer 수집→ReadFile E2E, Docker/브라우저/릴리즈 게이트 미실행.
- 요청 technology 스킬 3개/Skill 도구가 없어 절차·반환 형식 미확인; 사용자 지정 실패 재현·최소 수정·완료 검증 절차 적용.
- 따옴표/등호/복수 -u/placeholder/Authorization 및 탐지 범위 확장은 제외; 향후 placeholder 병합 시 curl 캡처 번호와 값 판정 위치에 주의.
- [러너 23:33] brief accepted — 채택 — 현재 코드의 접두부 소실을 실제 회귀 테스트로 확인했고 지정 범위의 최소 수정으로 수용 기준을 충족했다.
- [러너 23:35] verify passed — 검증 3개 통과 (auto)
- [러너 23:35] pr created — https://github.com/hkjang/git-ctx/pull/33
- [러너 23:35] guard held — internal/contentsecurity/credential_shapes_test.go 
