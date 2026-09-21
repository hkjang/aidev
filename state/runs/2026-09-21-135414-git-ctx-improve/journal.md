# 회차 노트 2026-09-21-135414-git-ctx-improve — git-ctx
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 13:54] base pinned — main@cf3b598
- [러너 13:54] autonomy release — 

## 정찰 노트
- YAML 리스트 형제 필드 과마스킹을 선택: 실제 name·port 소실/빈 본문 허위 finding을 재현했고 인증 변경·반려 파서 확장·미재현 flake보다 근거와 범위가 명확하다.
- 원본 복사 probe 및 contentsecurity/indexer/search/mcp 테스트 exit 0; 전체·race·외부 서비스 미검증. YAML 라이브러리 대조는 미실행이며 키 열 계산은 구현 제안이다.
- Revision 현재 edeca363cffe; 경계 계산만 바꾸고 지문을 놓치지 말 것. 이전 성공이나 main 미반영인 curl/변수/OAuth 작업을 다시 묶지 말 것.
- 기존 후보 12개 유지·재평가, 이전 노트 후보 1개와 신규 2개 포함 총 15개 pending. 요청한 세 스킬·Skill 도구 부재로 반환 형식 미확인.
- [러너 13:58] scout done — YAML 리스트 블록 스칼라가 형제 필드까지 과마스킹하지 않게 수정 (가치 3 / 위험 2 / 작업량 M)

## 구현 노트
- 70074df: 블록 헤더의 키 앞 접두부를 캡처해 대시가 아닌 키 열로 경계를 계산; 형제 필드 과마스킹/빈 본문 허위 finding 수정. Revision 74c72e911ce2.
- 전체 출력 24개 조합·두 번 Sanitize·ReadFile 회귀를 추가했고 수정 전 실패(red-tests.log), 수정 후 통과를 확인했다.
- 지정 패키지, 비캐시 contentsecurity/search, 해당 race, 전체 go test(sqlite_fts5), go vet, go build(sqlite_fts5), gofmt와 diff 검사 모두 통과(full-tests.log).
- 확신 없는 곳·검증 못 한 것: 실제 YAML 파서 대조, 외부 PostgreSQL·pgvector·Vault(환경변수 없음), Docker·브라우저·릴리즈 게이트는 미검증.
- 요청한 technology:completion-verification/systematic-debugging/test-driven-development 및 Skill 도구를 찾지 못해 해당 절차·반환 형식은 미확인; 과제서 절차를 따랐다.
- 새 YAML 문법·비밀 키 이름·변수/curl/OAuth 변경은 범위 밖이라 제외. 정찰의 기존 15개 아이디어를 유지하고 선택 항목만 done으로 갱신했다.
- 다음 역할 주의: 전역 정규식 교체 지문 테스트에는 t.Parallel 금지; CRLF 보존은 Sanitize에서 검증(ReadFile은 LF 정규화). 푸시·릴리즈하지 않았다.
- [러너 14:03] brief accepted — 채택 — 현재 코드에서 과마스킹과 빈 본문의 허위 finding을 회귀 테스트로 확인했고 지정된 최소 수정으로 수용 기준을 �
- [러너 14:05] verify passed — 검증 3개 통과 (auto)
- [러너 14:05] pr created — https://github.com/hkjang/git-ctx/pull/35
- [러너 14:05] guard held — internal/contentsecurity/credential_shapes_test.go 
