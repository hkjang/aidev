# 회차 노트 2026-09-29-172225-jupiq-improve — jupiq
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 17:22] base pinned — main@93943d7
- [러너 17:22] autonomy release — 

## 정찰 노트
- 실제 HTTP에서 재현된 requestList·streamAI의 error.code 누락을 선택: 프로덕션 1파일이며 예방 헬퍼 테스트·화면 캡처보다 근거가 분명하다.
- request는 403/forbidden, 나머지는 403/undefined였다. UI의 code 활용 효과·Vitest 새 HTTP 서버 환경은 미확인; 사용자 화면 변경은 범위 밖이다.
- 성공 envelope·meta·SSE·AbortError를 합치지 말 것. 초기 brief 작성 후 실행 증거와 단계·예비시간으로 덮어썼다.
- Go api/store 테스트 통과(DB 통합 제외). 세 스킬은 전용 도구 부재로 로컬 원문을 읽어 적용했고, 코드는 수정하지 않았다.
- [러너 17:27] scout done — 목록·AI HTTP 오류에서도 서버 error.code를 보존 (가치 2 / 위험 1 / 작업량 S)

## 구현 노트
- client.ts의 HTTP 오류 생성만 공유하여 목록·AI도 서버 code를 보존; 프로덕션 1파일, 신규 테스트 1파일.
- 실제 node:http 서버·네이티브 fetch·동적 import·jsdom으로 33개 검증; 수정 전 4개 실패, 집중 44개 통과, code 제거 변형에서 신규 6개/기존 1개 재실패 후 원복.
- 원복 후 npm --prefix web test: 21파일/123개 통과; npm --prefix web run lint 및 run build, git diff --check 통과.
- 확신 없는 곳·미검증: 실제 Go 서버·DB·브라우저 E2E·화면의 code 활용 효과·release-check. 테스트는 로컬 HTTP 클라이언트 경계만 입증한다.
- 성공 envelope/meta/평탄화, SSE 파서·정리, 네트워크·AbortError 처리, 잘못된 JSON 정책은 범위 밖이므로 변경하지 않았다.
- 다음 역할: Node 네이티브 fetch와 로컬 listen 필요. 전체 테스트는 ResourceListPage jsdom getComputedStyle 경고를 출력하나 통과; npm ci는 moderate 취약점 2건을 보고했다.
- 요청된 세 technology 스킬은 전용 도구 부재로 로컬 SKILL.md 원문을 읽어 적용. 정찰 과제서로 선택 절차를 갈음하고 기존 아이디어를 유지했다.
- [러너 17:30] brief accepted — 채택 — 현재 소스와 실제 HTTP 테스트가 진단에 일치하여 지정된 두 파일에서 구현하고 정찰의 후보·평가를 유지했다.
- [러너 17:30] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- approve / low / blocking 없음: diff·커밋·두 변경 파일·클라이언트 전체·테스트 설정·원장 실패 재현 확인, 실제 결함 없음.
- 신규 실제 HTTP 33개와 기존 클라이언트/SSE 11개 재실행 모두 통과; diff --check 통과.
- 새 개인정보 수집·저장·외부 전송·권한 확대 없음; 세 부서 스킬은 전용 도구 부재로 로컬 원문 적용.
- Go·DB·브라우저 E2E·release-check와 전체 프런트 검증은 재실행하지 않음; 화면 개선 효과는 미확인으로 릴리즈 설명은 HTTP 오류 code 보존에 한정.
- [러너 17:32] review approved — 리뷰 승인 (risk=low)
- [러너 17:32] pr created — https://github.com/hkjang/jupiq/pull/29
- [러너 17:33] ci failed — 성공이 아닌 검사: React 검사=failure
