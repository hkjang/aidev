# 회차 노트 2026-10-05-161658-jupiq-improve — jupiq
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:17] base pinned — main@92ff88c
- [러너 16:17] autonomy release — 
- [러너 16:21] scout done — 추적 설정의 `allowed_hosts` 가 검증 없이 CSP 헤더에 그대로 이어붙어 `object-src 'none'` 을 무력화하는 것을 막�

## 구현 노트
- `allowed_hosts` 토큰이 `;` 를 품은 채 CSP 디렉티브에 이어붙어 `object-src 'none'` 을 무력화하던 것을 막았다. 같은 값을 읽는 경로가 둘이라 `Validate()`(저장)와 `PolicySources()`(렌더)를 공용 헬퍼 `invalidHostReason` 으로 함께 고쳤다. 프로덕션 1파일 + 테스트 2파일, 커밋 69a6ea1.
- **확신 없는 곳**: 거부 기준의 넓이. `;`·`,`·제어문자·`'` 로 시작하는 토큰만 막는다 — CSP 소스로 무의미하지만 구조는 안 깨는 값(예: `javascript:`, 한글 호스트, 공백 없는 쓰레기 문자열)은 그대로 통과한다. 의도적이다(구조 파괴만 막고 CSP 문법 전체를 검증하지 않는다). 과제서가 금지한 `*`·`:`·`/`·`.`·`-` 는 막지 않는 것을 테스트가 단언한다.
- **검증 못 한 것**: 실제 브라우저가 이 헤더를 어떻게 파싱하는지는 확인하지 않았다 — "중복 디렉티브는 첫 번째만 유효" 는 CSP 명세 근거이고 내 증거는 헤더 문자열의 디렉티브 이름 목록(10 → 13)까지다. DB 를 거친 end-to-end(설정 저장 → 재조회 → 헤더)도 안 돌렸다: `core_handlers.go:825` 가 호출하는 `validateSettingsUpdate` 를 실제로 타는 단위 테스트로 대신했다. 통합 테스트(`make test-integration`)와 web 게이트는 이 과제가 그 경로를 건드리지 않아 돌리지 않았다.
- **일부러 안 한 것**: `SplitHosts` 는 exported 라 그대로 뒀다(유일한 다른 호출자 `AddAllowedHost` 는 프로덕션 호출자가 없다). `pagePolicy`·`contentSecurityPolicy` 문자열, 기존 CSP 단언, web 트리, 문서는 무변경 — `openapi.yaml`·`docs/` 에 `allowed_hosts` 규칙 설명이 없어 맞출 것이 없었다(grep 확인).
- **다음 역할이 조심할 것**: 새 테스트는 DB·네트워크 없이 돈다(`go test -count=1 ./internal/analytics ./internal/api`). `internal/api/analytics_handlers_test.go` 의 렌더 테스트는 기준선 디렉티브 **10개**를 하드코딩하므로, 정책 문자열에 디렉티브를 하나라도 더하는 변경은 이 테스트를 함께 고쳐야 한다(그게 이 테스트의 목적이다). `'unsafe-inline'` 개수 단언은 `style-src` 의 정당한 1개를 센다.
- [러너 16:26] brief accepted — 채택 — 과제서의 진단(`Validate()` 미검사, `SplitHosts` 가 `;` 를 안 나눔, `pagePolicy` 가 세 디렉티브에 이어붙임, 10 → 13 디렉�
- [러너 16:26] verify failed — 실패한 검증: npm run lint   # CI에서 가져옴 (exit 1)
