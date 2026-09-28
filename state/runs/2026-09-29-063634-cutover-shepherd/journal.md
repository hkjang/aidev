# PR 처리기 노트 2026-09-29-063634-cutover-shepherd — cutover PR #9
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-29-060144-cutover-improve)
# 회차 노트 2026-09-29-060144-cutover-improve — cutover
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 06:01] base pinned — main@8e876b7
- [러너 06:01] autonomy release — 

## 정찰 노트
- 반복된 보고서 삭제 우회를 설정 1파일로 없애는 과제를 선택; 상태 전파·로그인·Docker 변경보다 위험과 검증 비용이 작다. 초안을 먼저 쓴 뒤 실제 ESLint 실험 결과로 확정했다.
- headcount 세 스킬 정본을 로컬에서 발견·직접 읽음(전용 Skill 호출 도구 없음). 기존 12개 아이디어 유지·재평가, 해결된 테스트 1개와 신규 후보 2개를 기록했다.
- 단위 95건/30 suites 통과. 정식 lint는 의존성 부재로 127; 별도 설치 ESLint+현재 설정 복사본으로 생성물도 검사됨을 확인했으므로 정식 lint 통과와 혼동 금지.
- 구현자는 보고서 삭제·검사 대상 축소·의존성 갱신을 피할 것. 가정은 기본 출력 경로 유지; 현재 설정에서 확인했고 20–35분 추정은 실측 통계가 아니다.
- [러너 06:07] scout done — ESLint에서 Playwright 생성 보고서·실행 결과 제외 (가치 2 / 위험 1 / 작업량 S)

## 구현 노트
- 0de2f1e: eslint.config.mjs globalIgnores에 Playwright 생성 경로 2개만 추가(1파일·2줄); 실패 진단 산출물을 지우지 않고 lint 가능.
- 실제 ESLint API: 수정 전 exit 1(false !== true), 수정 후 exit 0; 원복 exit 1 및 재적용 exit 0으로 원인 확인. 소스 3경로 검사 유지.
- npm run lint exit 0; npm run test:unit 95건/30 suites 통과(실패·skip 0); git diff --check 통과, 커밋 후 작업 트리 깨끗함.
- 검증 못 한 것: 실제 실패 trace 생성·E2E·build는 과제서에 따라 생략. 시작 시 보고서/결과 디렉터리 없었으며 삭제하지 않음.
- npm ci --legacy-peer-deps 성공; high severity 2건 보고. 의존성·lockfile 수정은 하지 않았으므로 별도 평가 필요.
- technology 세 스킬 정본 직접 읽음(전용 Skill 도구 없음), Next ESLint 가이드 확인. 영구 테스트·다른 설정 변경은 과제서대로 생략.
- 다음 역할: 기본 Playwright 출력 경로 전제 유지. ideas.json 기존 후보 및 정찰의 신규 2개를 보존했고 선택 항목만 done 처리.
- [러너 06:10] brief accepted — 채택 — 두 생성 경로가 실제 ESLint에서 제외되지 않는다는 근거가 현재 코드·실제 엔진 실행과 일치했다.
- [러너 06:10] verify passed — 검증 1개 통과 (policy)

## 비평 노트
- reject: lib/mail/smtp.ts:128 STARTTLS 핸드셰이크 타임아웃 공백을 실제 sendMail+로컬 TCP로 재현(100ms 설정, 1004ms에도 대기); 수리는 이 파일과 smtp.test.ts부터 확인.
- 로컬 main cd9d1f3은 원장 기준 8e876b7과 달라 main...HEAD 49파일을 대상으로 검토; ESLint 2줄 변경 자체는 실제 엔진에서 정상.
- 세 부서 스킬 정본 직접 읽음; 인증·인가·데이터 흐름·오류 경로·관련 테스트 확인, lint/단위 95건/diff check 통과; 부서 차단 없음.
- E2E·build·외부 SMTP/수집기 운영 정책은 미검증; 추적 보존·계약과 의존성 high 2건은 별도 평가 필요.
- [러너 06:13] review rejected — 리뷰 거절: lib/mail/smtp.ts:128 [P2 정확성·머지 차단] STARTTLS 전환 시 plain.setTimeout(0)으로 제한 시간을 없앤 뒤 secureConnect가 발생해야 타임아웃을 다시 설정�

## 수리 노트
- 맞았던 지적: STARTTLS 핸드셰이크 제한 시간이 없어 발송이 무기한 대기함. 틀린 지적 없음.
- 전용 타이머와 TLS 소켓 소유권·종료 처리를 추가하고 실제 TCP 회귀 테스트로 오류 및 연결 정리를 검증함.
- 수정 제거 시 동일한 2건 실패, 복원 후 단위 97건·린트·타입 검사·빌드 통과.
- 확신 없는 곳: 실제 외부 TLS 릴레이와 E2E는 실행하지 않았으며, 이번 검증은 로컬 무응답 경로에 집중함.
- [러너 06:15] repair done — 비평 지적 확인: STARTTLS 승인 후 TLS 무응답 시 timeoutMs=100이어도 1초 넘게 sendMail이 미완료였다. upgrade 전용 타이머와 오류·종료 처리를 추가하고, TLS 소�

## 비평 노트
- reject: lib/mail/events.ts:77에서 마지막 미완료 최상위 작업 삭제가 훈련 완료 메일을 유발함; 실제 서비스·sendMail·로컬 TCP 릴레이로 1통 수신 재현. 수리는 events.ts와 events.test.ts부터. 부서 차단 없음.
- STARTTLS 새 테스트는 HEAD^ 임시 복사본에서 STILL_PENDING after 1000ms로 2건 실패, 현재 단위 97건·lint·타입 검사·diff check 통과; ESLint 생성물 제외/소스 유지도 실제 API 확인.
- 세 부서 스킬 로컬 정본과 인증·인가·데이터 흐름·삭제·문서·테스트 확인. main=cd9d1f3과 pinned base=8e876b7 차이로 이전 기능을 포함한 49파일 diff가 범위임.
- E2E·build·외부 TLS 릴레이·수집기 보존/계약은 미검증. 수신자 기록 500건 상한 확인; 운영 추적 정책 확인 필요. 코드 수정 없음.
- [러너 06:18] review rejected — 리뷰 거절: lib/mail/events.ts:77 [P2 정확성·머지 차단] 완료된 최상위 작업 A와 대기 중인 B가 있을 때 B만 삭제해도 allDone 전후 비교가 drill.completed를 생성한
- [러너 06:18] pr created — https://github.com/hkjang/cutover/pull/9
