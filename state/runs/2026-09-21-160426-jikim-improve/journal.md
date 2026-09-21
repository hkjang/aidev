# 회차 노트 2026-09-21-160426-jikim-improve — jikim
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:04] base pinned — main@9ea43d0
- [러너 16:04] autonomy release — 

## 정찰 노트
- 실제 HTTP/native fetch로 callback 예외의 원문 재전달·성공 오인을 재현해 선택; settings 충돌·DB 삭제·TOCTOU·캡처 계약 변경보다 좁고 확실하다.
- AiPage에서 자연 발생하는 callback 예외와 Vite 차선의 런타임 실패는 미확인; 프런트 전체 검증도 미실행. Go test ./... 통과.
- d6dd63f는 현재 HEAD에 없지만 과거 성공한 HTTP 오류 공통화를 다시 구현하지 말 것; api.ts 인접 변경·진짜 스트림 테스트를 주의.
- 지정 pmo/technology 스킬·Skill 도구 미발견으로 고유 형식은 미확인. profile 갱신, 기존 후보 유지·재평가 및 신규 2건을 ideas.json에 기록.
- [러너 16:08] scout done — streamChat의 콜백 예외를 JSON 파싱 오류로 오인해 원문을 다시 전달하는 문제 수정 (가치 2 / 위험 1 / 작업량 

## 구현 노트
- 4e21a71: streamChat의 catch를 JSON.parse로 제한하여 소비자 오류의 원문 재전달·성공 오인을 제거; api.ts와 새 테스트만 변경.
- 실제 loopback HTTP/native fetch 회귀 9개: 수정 전 6개 실패→수정 후 전부 통과; 첫 콜백만 Error/SyntaxError를 던지는 동일 응답 2개 이벤트 포함.
- Node 24.21.0에서 전체 41개·lint·build 및 최종 verify.sh(Go test/vet·문서·Compose) exit 0; red.log/green.log/frontend.log/verify.log 참고.
- 확신 없는 곳·검증 못 한 것: 실제 AiPage setMessages의 자연 발생 throw·브라우저 E2E·PostgreSQL opt-in 테스트는 미검증; npm ESLint 지원 종료 및 Vite 500kB 청크 경고 유지.
- SSE 전체 규격·EOF 잔여 버퍼·DONE 조기 종료·연결 취소·이전 HTTP 오류 공통화는 별도 과제여서 제외; 사용자 가이드도 내부 콜백 상세로 변경하지 않음.
- 다음 역할: jsdom 환경에서 URL만 바꿔 native fetch를 호출하며 afterEach에서 fetch/서버 연결을 정리한다. 테스트 파일의 Node 타입 reference를 유지하고 api.ts 인접 머지에 주의.
- 요청한 technology 스킬/Skill 도구 미발견; 로컬 superpowers debugging/TDD/verification 대체 적용. 빌드 산출물은 기존 ignore 대상이며 커밋하지 않음.
- [러너 16:12] brief accepted — 채택 — HEAD 9ea43d0의 catch 범위와 실제 HTTP 실패 재현이 과제서 근거에 일치했고 최소 경계 수정으로 수용 기준을 충족했�
- [러너 16:12] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- reject: web/src/lib/api.ts:87의 null 이벤트 TypeError로 후속 정상 청크 유실; 수리는 이 파일과 api.stream-callback.test.ts부터 확인.
- main/HEAD 실제 HTTP 비교: null→later가 main에서는 정상 전달, HEAD에서는 즉시 reject. 전체 프런트 41개 테스트 통과(Node 22).
- diff·커밋·테스트·AiPage·서버 SSE 중계·기여 계약 확인; 범위 이탈/마이그레이션/보안·법무 차단 근거 없음.
- 지정 3개 스킬/Skill 도구 미발견. 실제 공급자 null 빈도·브라우저 E2E·DB·Node 24 재검증 미확인; 연결 취소 등 기존 제한은 후속 과제.
- [러너 16:14] review rejected — 리뷰 거절: web/src/lib/api.ts:87 [P2] JSON null SSE 이벤트가 전체 응답을 중단하는 회귀. 실제 loopback HTTP/native fetch에서 data: null 뒤 data: {"content":"later"}를 보내면
