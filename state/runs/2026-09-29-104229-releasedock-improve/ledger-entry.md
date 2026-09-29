## 2026-09-29
- 선택: 전체 모드 실시간 로그가 서버의 30분 제한 뒤 마지막 커서에서 재연결 (가치 3 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 서버 로그 ID 커서를 화면 sequence와 분리해 보관하고 명시적 reason=max_duration에서만 마지막 커서로 연결 하나를 다시 연다. 정상 end {}, 파싱 불가/알 수 없는 이유와 일반 error는 자동 재연결하지 않으며 지우기·탭 이탈·언마운트·릴리즈 이동을 실제 App 렌더로 검증했다(프로덕션 1파일 + 테스트 1파일). 수정 전 9건 중 5건 실패 후 최종 집중 10건/전체 웹 122건, tsc -b --noEmit, npm run build, git diff --check 통과; 재연결 제거·모든 end 재연결·커서 갱신 제거 반증 실험 모두 예상 테스트 실패 후 원복했다.
- 실패 재현: `→ expected [ EventTarget{ …(6), …(1) } ] to have a length of 2 but got 1` / `Expected: "/api/v1/releases/release-2/logs/stream?after=0" Received: "/api/v1/releases/release-2/logs/stream"` — 수정 전 실제 App 테스트 5 failed | 4 passed.
- 보류 아이디어: 로그 스트림 한도 거절·해제 HTTP 통합 테스트 (가치 2 / 위험 2 / M) — 동시 해제 동기화 필요.
- 보류 아이디어: SSE rAF 배치 갱신 (가치 2 / 위험 2 / M) — 성능 측정 후 선정.
- 보류 아이디어: make vet 및 웹 타입 검사를 make test에 포함 (가치 2 / 위험 1 / S) — 기존 브랜치 중복 주의.
- 보류 아이디어: 릴리즈 ID 전환 시 표시 로그 초기화 (가치 2 / 위험 2 / S) — 이번에는 커서 격리만 수정, 표시 초기화는 별도 과제.
- 과제서: 채택 — 현재 훅의 모든 end 영구 종료 및 서버의 정상 {} / max_duration 계약을 확인했고 지정 두 파일로 수용 기준을 검증했다.
