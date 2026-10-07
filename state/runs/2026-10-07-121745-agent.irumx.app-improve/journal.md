# 회차 노트 2026-10-07-121745-agent.irumx.app-improve — agent.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:17] base pinned — main@0e65235
- [러너 12:17] autonomy release — 

## 정찰 노트
- 선택: DST shift의 n별 첫 실행 불일치를 실제 소스로 재현; 사용자 회차 누락과 연결되고 프로덕션 1개 파일로 끝나 입력 검증·UI 후보보다 우선했다.
- 근거: Node 검사 3 pass/1 fail(결함 재현); nextAfter와 HTTP 상세 upcoming 배선 확인. 초안을 먼저 저장한 뒤 검증 근거로 보완했다.
- 미확인: tsc/@playwright/test 부재로 전체 빌드·API 테스트 미실행. 시간 추정 30~45분은 의존성 설치/로컬 Worker 정상 기동을 전제로 한다.
- 주의: skip/overlap 계약·권한·위임·auth/migrations 유지, 하루 후보로 범위 제한. HTTP 테스트는 pause 후 날짜를 고정해 Cron 경합을 피하고 구 빌드 재사용을 확인한다.
- [러너 12:22] scout done — DST shift 예약의 다음 실행 순서와 조회 개수 일관성 수정 (가치 4 / 위험 2 / 작업량 M)

## 구현 노트
- 변경: shift에서 날짜 안의 후보를 끝까지 검토하도록 nextRuns의 중단 조건 두 개만 수정; skip 빠른 경로·UTC 중복 제거·5년 상한 유지. 커밋 6b7ac2b, 프로덕션 1개+테스트 2개.
- 재현: 과제서 Node 실패, 새 단위/API 3 failed / 4 passed; 수정 재철회 시 같은 단위 2 failed / 4 passed. 스킬 3개는 도구 부재로 로컬 SKILL.md를 읽어 적용.
- 검증: npm ci, npm run check, npm run build 통과; npx playwright test --project=unit tests/unit-core.spec.ts → 27 passed; 최종 npx playwright test --project=unit --project=api tests/unit-core.spec.ts tests/api-schedule.spec.ts → 34 passed. 명령·로그·환경은 verification.md.
- 실제 HTTP: person→addModel→publishAgent→POST 예약→pause→next_run_at만 고정→GET 상세로 16:00Z, 다음 날 15:30Z, 15:45Z 및 이후 2회를 확인; 계산 함수 대역 없음.
- 확신 없는 곳·미검증: 역사적 날짜 단위 전환, 실제 운영 큐 DST 실행, 전체 브라우저/다른 API 파일. 기존 npm audit 경고 8개는 별도 조사 대상.
- 일부러 제외: auth·권한·파서·localToUtc·API 계약·의존성/lockfile·릴리즈 변경은 범위 밖. 기존 보류 항목 유지, 테스트 서버/Node 환경 후보 2개를 ideas.json에 추가.
- 다음 역할 주의: 기존 8870은 타 작업 트리 서버라 종료/재사용하지 말 것. user/network 네임스페이스에서 새 서버와 로컬 D1을 실행하고 node22-shell.sh로 npm의 상위 Node 20 선택을 우회해야 한다. 전역 설정/비밀 파일은 수정하지 않았다.
- [러너 12:29] brief accepted — 채택 — 지정된 Lord Howe 결함과 상세 HTTP 회차 누락이 현재 코드에서도 재현되어 기본 과제를 그대로 구현했다.
- [러너 12:32] verify failed — 실패한 검증: npm test --silent (exit 1)
