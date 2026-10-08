# PR 처리기 노트 2026-10-08-235022-hunter-shepherd — hunter PR #22
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-08-215834-hunter-improve)
# 회차 노트 2026-10-08-215834-hunter-improve — hunter
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:58] base pinned — main@9069bcf
- [러너 21:58] autonomy release — 

## 정찰 노트
- SLA/risk 정수 9필드를 선택: 기존 렌더러 옵션 연결만으로 저장 후 오류를 줄이며 프로덕션 1파일. engines no-change 재시도·auth/DB 의존 후보보다 작고 근거가 명확하다.
- 초안을 먼저 저장한 후 서버·save 경로를 확인해 보완. 웹 113 PASS/0 FAIL/0 SKIP, Go 순수 시험 1 PASS. 저장소 수정·커밋 없음.
- 실제 브라우저 입력은 미확인; EPSS 소수 예외를 지키고 소수 자동 절삭·다른 설정 그룹·폼 리팩터를 섞지 말 것. 서버 계약 테스트는 UI 수정의 red/green 증명이 아니다.
- 요청한 세 스킬은 전용 도구 부재로 로컬 headcount SKILL.md를 읽어 적용. CSV 차선은 RangeError 재현했으나 실자료 도달성 미확인. profile/ideas를 현재 근거로 갱신했다.
- [러너 22:05] scout done — SLA·조치 우선순위 설정의 정수 필드 9개에서 소수 입력을 제한 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- `83d1dad`: settings.tsx 다섯 위치에 integer:true 추가(SLA 6/risk 3); EPSS 소수·기존 min/max/default/문구·save 유지. 프로덕션 1파일 + Go 시험 1파일.
- Go 1.26.7 지정 순수 시험 상위 2 PASS/0 SKIP, 새 계약 51사례. Node 26.11.1 npm ci·웹 113 PASS/0 FAIL/0 SKIP·typecheck·build 및 원본 312파일·diff 검사 통과.
- 실제 앱 Chromium+합성 API에서 수정 전 fractional PUT 실패 → 수정 후 통과 → 옵션 제거 시 재실패 → 복구 후 통과. 1440/390 폭에서 9필드 타이핑·클립보드·blur·전송값·경계·EPSS 0/0.1/0.125/1·타 그룹 초안 보존 확인.
- 캡처 green-risk-1440.png/green-risk-390.png는 합성 자료 표시; browser-numeric-contract.py와 red/green/reverted 로그를 이 디렉터리에 보존. Mantine 자체 동작은 1.5 타이핑→15, 붙여넣기→1이며 별도 반올림/절삭은 추가하지 않음.
- 미검증: 실제 API/DB·Go 바이너리·전체 race·원격 CI/배포. HUNTER_TEST_DSN 없음; 브라우저 PUT은 모의 수신이며 DB 저장 성공이 아님. 추가 Prettier는 기존 MCP/OIDC 3곳 차이로 실패(HEAD도 동일), 무관한 서식 수정 제외.
- 서버 검증기·다른 설정 그룹·버전·보호 경로·임베드는 미변경; 산출물은 무커밋. 후속 역할은 Go green을 UI red/green으로 혼동하지 말고 verification.md·브라우저 로그 참조. 요청 스킬 3종은 전용 도구 부재로 로컬 headcount SKILL.md를 읽어 적용.
- [러너 22:17] brief accepted — 채택 — 9개 정수 옵션 누락·공통 렌더러 배선·서버 정수 거절·설정 원값 PUT 전제가 모두 현재 코드와 맞아 지정된 두 �
- [러너 22:17] verify passed — 검증 9개 통과 (auto)
- [러너 22:17] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 22:17] pr created — https://github.com/hkjang/hunter/pull/22
