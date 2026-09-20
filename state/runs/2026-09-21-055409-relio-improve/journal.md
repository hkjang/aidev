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

## 비평 노트
- 판정 reject(main...HEAD): 저장소 차단 시 SSO 로그인 렌더링 예외. 수리는 web/src/App.tsx:33 및 web/src/pages/Login.tsx:35부터 확인; security/legal 부서 차단 없음.
- 로컬 main=6bfe9bf는 회차 base=38c88ce보다 과거여서 32개 파일이 비교됨. 위 결함은 이전 SSO 변경이며 b497df5의 JSON 3파일 수정 자체는 결함 미발견.
- Go 전체·웹 11개·JSON 비캐시 테스트 통과; 수정 전 httpx overlay에서 후행 공백 회귀 테스트 실패 확인. 실제 pendingReturn의 저장소 예외 전파 재현.
- 브라우저·인증/Keycloak·배포·실제 DB/마이그레이션·외부 400 의존·수집기 보존/접근 조건 미검증; 기준 SHA 정합성을 다음 단계에서 확인해야 함.
- [러너 06:05] review rejected — 리뷰 거절: web/src/App.tsx:33 [P2·정확성·머지 차단] SSO가 활성화된 로그인 화면은 web/src/pages/Login.tsx:35에서 렌더링 도중 pendingReturn()을 호출한다. sessionStora

## 수리 노트
- 맞았던 지적: 저장소 접근/getItem 예외와 URL 파싱 실패가 Login 렌더를 중단함; 실제 컴포넌트 테스트 6개 실패로 확인.
- 기준 정정: 지정 base 38c88ce...수리 전 HEAD는 JSON 3파일뿐이므로 이번 JSON 변경의 회귀는 아님; 요청된 현재 결함은 수리함.
- 수정·검증: 82980bf에서 pendingReturn 예외 시 dashboard 대체, 정상 복귀 유지; 수정 제거 재실패 및 복원 후 웹 23개·typecheck/build·Go 전체·정적 자산 검사 통과.
- 확신 없는 곳: 실제 브라우저·Keycloak 미검증, 빌드 analytics/청크 경고 존재; Skill 도구 미노출로 요청한 headcount 스킬 3개의 SKILL.md를 직접 읽고 적용.
- [러너 06:08] repair done — - 재현: 실제 Login 렌더링에서 sessionStorage 접근/getItem의 SecurityError 및 잘못된 URL로 6개 테스트 실패(autoLogin false/true). - 원인·수정: pendingReturn의 저장소 읽

## 비평 노트
- 판정 reject(main...82980bf): internal/server/momento.go:126의 헤더 전용 타임아웃으로 본문 정지 시 10초 제한 미작동; 수리는 114·126줄부터 확인. security/legal 차단 없음.
- 실제 수집기·프록시로 200 헤더 후 본문 정지 시 11.012초 클라이언트 취소까지 대기함을 재현; 전체 요청 context 제한과 본문 정지 테스트 필요.
- Go 전체·웹 23개·typecheck 통과. 임시 overlay/복사본으로 수정 전 JSON 후행 공백 테스트 및 수리 전 Login 6개 테스트 실패 확인; 저장소 코드 무수정.
- 로컬 main=6bfe9bf로 33파일 비교; 지적은 기존 01aba63의 결함이며 pinned 38c88ce 이후 JSON·SSO 수리에는 새 결함 미발견. 브라우저·Keycloak·DB/배포·외부 400 의존·수집기 보존/접근 조건 미검증.
- [러너 06:11] review rejected — 리뷰 거절: internal/server/momento.go:126 [P2·정확성·머지 차단] ResponseHeaderTimeout만 설정되어 수집기가 헤더를 보낸 뒤 본문 전송을 멈추면 문서의 10초 제한�
- [러너 06:11] pr created — https://github.com/hkjang/relio/pull/28
