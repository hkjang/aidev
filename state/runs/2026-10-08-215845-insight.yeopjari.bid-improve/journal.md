# 회차 노트 2026-10-08-215845-insight.yeopjari.bid-improve — insight.yeopjari.bid
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:58] base pinned — main@c6a3399
- [러너 21:58] autonomy release — 

## 정찰 노트
- judgeSubmission URL의 501/700→500 절단을 원본 함수 실행으로 재현해 선택했다. 사용자 제출 경로가 확인돼 미사용 캐시·주석·문서 후보보다 가치가 높고 프로덕션 1개로 제한된다.
- 초안을 먼저 저장한 뒤 실제 POST/GET·UI 오류 배선·studyB e2e 지점을 확인해 보완했다. 기존 6개 아이디어 유지·재평가 + 신규 2개, 선택도 pending.
- 의존성 부재로 Vitest는 exit 127; HTTP/DB·Docker/포트 상태는 미확인이다. 35~45분은 환경 준비가 가능한 조건의 추정이며 실제 통과로 쓰지 않았다.
- 구현자는 parseLinks/공통 파서/auth/migration을 건드리지 말고 실제 제출→조회, 초과 재제출 시 기존 값·시각 보존을 검증한다. gateway 이름·JSONB 구조 비교·fake OAuth 고정 포트 주의. 요청된 3개 스킬은 도구 부재로 로컬 SKILL.md를 직접 읽어 적용했다.
- [러너 22:05] scout done — 제출 답변 URL이 500자를 넘으면 자르지 말고 입력 오류로 거절하기 (가치 4 / 위험 2 / 작업량 M)

## 구현 노트
- judgeSubmission URL만 전체 정규화 후 500자 초과 오류를 수집한다. 검증 전 절단으로 증빙 주소가 바뀌는 결함을 수정했고 지정 3개 파일(프로덕션 1개), 커밋 739ea33.
- 단위: 기준 32통과, 추가 후 7실패/60통과 → 수정 후 67통과; 수정만 되돌려 같은 7실패를 확인하고 복구했다. 500/501/700·정규화·UTF-16·필수/선택·오류 수집·text/long 제한을 검증한다.
- 최종 npm run check(lint/3 workspace 타입 검사/98개 단위), npm run build, git diff --check 통과. 실제 격리 Postgres 16+정식 서버 E2E 180통과/0실패.
- 500자 제출→참여자/기업 GET 완전 보존, 501자와 다른 답변 변경 재제출→400/field_errors 및 기존 전체 submission/submitted_at 보존 확인.
- 확신 없는 곳·검증 못 한 것: 브라우저 필드 표시와 운영 환경 실행은 미검증(이번 범위 밖); 로컬 요구 검증의 미해결 실패는 없다.
- 일부러 하지 않은 것: parseLinks/공통 파서·라우트·UI·auth·migration 변경, 릴리즈·배포. 첫 E2E의 변수 중복 SyntaxError는 이름 변경 후 전체 재실행해 해결했다.
- 다음 역할: E2E는 격리 DB/정식 서버 및 빈 :18999 필요, gateway=irumx_gateway; JSONB 비교는 isDeepStrictEqual 유지. 이번 서버·DB·임시 credential fixture 정리 완료, 기존 서비스 보존.
- [러너 22:11] brief accepted — 채택 — 현재 코드에서도 검증 전 URL 절단이 재현되어 지정한 프로덕션 1개와 테스트 2개 파일 범위 그대로 구현했다.
- [러너 22:11] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- approve / low, blocking 없음. 요청한 3개 스킬은 도구 부재로 로컬 SKILL.md를 읽어 적용; 3개 변경 파일·커밋·과제서·실패 원장을 대조했다.
- 직접 검증: npm run check 98개 단위·lint·타입 검사, E2E 문법·diff 검사 통과. 소스 무수정 main/HEAD 실행으로 필수·선택 500자 보존, 501/700자 절단→거절을 확인했다.
- 인가·입력→저장/조회·오류 UI 배선과 갱신 전 검증을 확인했다. 새 개인정보 처리·권한·외부 전송·의존성·마이그레이션 변경이나 구체적 공격 경로는 발견하지 못했다.
- 미실행: 브라우저 표시·운영·HTTP/DB E2E/테넌트 격리 재실행(구현자 180통과 기록과 단언만 검토). 릴리즈에는 500자 초과 제출 URL이 절단 대신 거절됨을 안내한다.
- [러너 22:14] review approved — 리뷰 승인 (risk=low)
- [러너 22:14] pr created — https://github.com/hkjang/insight.yeopjari.bid/pull/3
- [러너 22:19] ci passed — 검사 1개 모두 success
- [러너 22:19] merge done — 739ea33
- [러너 22:19] release missing — 릴리즈 결과 없음/손상: missing
