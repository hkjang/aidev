# 회차 노트 2026-09-21-055409-relio-improve — relio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 05:54] base pinned — main@38c88ce
- [러너 05:54] autonomy low-risk — 롤백 PR 

## 정찰 노트
- 공용 JSON 크기 초과 오분류를 실제 createCustomer로 재현해 선택: 기존 문구 정리보다 효과가 크고 프록시 신뢰·페이징·DB 변경보다 좁다.
- go test ./... 통과; 과대 첫 값 413/과대 후행 공백 400 확인. 전체 인증 라우트 및 배포 검증은 미실시.
- 두 Decode의 오류만 맞추고 한도·구문 계약·auth/migrations/workflows는 유지; 테스트는 실제 MaxBytesReader와 서비스 핸들러 사용.
- 세 요청 스킬을 headcount 파일에서 읽어 계획/대안/30~40분 추정 반영. 감사 Exec 실패 로깅 아이디어는 이미 구현되어 done 처리.
- [러너 05:59] scout done — JSON 본문 크기 초과를 두 디코딩 단계 모두에서 413으로 응답하기 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- b497df5: 두 번째 Decode의 MaxBytesError를 413/request_too_large로 우선 분류하는 5줄 추가; 한도·기존 400 문구 유지.
- Skill 도구 미노출로 headcount technology의 completion-verification/systematic-debugging/test-driven-development SKILL.md를 직접 읽고 적용.
- overlay 기준 413/400 재현 → 새 테스트 red → 수정 green → 분기 제거 재실패 → 복원 green으로 원인 확인.
- 실제 디코더 18개 케이스(길이 유무·2MiB 경계·기존 오류·requestId), 실제 createCustomer 2개 초과 케이스 통과.
- go test ./internal/platform/httpx ./internal/server; go test -race ./...; go vet ./...; go build ./...; 지정 gofmt -l; git diff --check 모두 통과.
- 확신 없는 곳·검증 못 한 것: 인증 미들웨어를 포함한 배포 HTTP 경로와 외부 소비자의 특정 400 의존 여부는 미검증.
- DB·프런트·인증·마이그레이션·릴리즈는 범위 밖으로 수정하지 않음. 다음 역할: 서버 테스트는 CRM·DB 없는 실제 핸들러의 입력 거절만 증명하며 DB 작업 성공을 증명하지 않음.
- [러너 06:02] brief accepted — 채택 — 기준 코드와 overlay 재현 결과가 과제서와 일치하여 지정된 공용 오류 분류와 회귀 테스트만 구현했다.
- [러너 06:02] verify passed — 검증 9개 통과 (auto)
