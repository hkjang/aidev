# 회차 노트 2026-09-28-224225-AgentHub-improve — AgentHub
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 22:42] base pinned — main@2bb1512
- [러너 22:42] autonomy release — 

## 정찰 노트
- allowedHosts 가 CSP 헤더로 검사 없이 나가는 경로를 골랐다: 다른 provider 분기는 전부 originOf() 를 거치는데 305-307 행만 splitHosts 결과를 그대로 돌려준다는 비대칭이 코드에 그대로 보였고, Validate() 가 두 쓰기 경로(routes.go:1736, tracking.go:274)의 유일한 공통 지점이라 프로덕션 파일 하나로 끝난다.
- 보류 1순위였던 감사 details origin 길이(2/2/S)를 제친 이유: 같은 Validate 지점이 그것도 함께 닫으면서 헤더 오염까지 덮는다. 잘린 origin 이 allowedHosts 에 들어가는 신규 아이디어는 Allowed 판정이 잘린 값으로 이뤄져 권한이 넓어지지 않음을 확인하고 rejected 로 내렸다.
- 확신 없는 곳: url.Parse 가 와일드카드 https://*.corp.example 의 Host 를 그대로 받는지 실행해 확인하지 못했다(Bash 승인 거부). 그래서 과제서를 정규화가 아니라 거절로 설계했다 — 구현자는 이 미확인에 기대는 길로 가지 말 것.
- 조심할 것: tracking_test.go 의 195(와일드카드)·135(포트)·99(뒤 슬래시) 행이 기존 통과 값을 고정하고 있다. 새 검사가 이 셋을 건드리면 설계가 틀린 것이다. 증명은 손으로 만든 Settings 가 아니라 server.Handler() 를 통과하는 두 라우트로 할 것.
- [러너 22:46] scout done — 추적 허용 출처(allowedHosts)의 한 항목이 CSP 헤더의 지시문 경계를 넘거나 무한히 길어지지 못하게 Validate �

## 구현 노트
- 무엇을: `internal/tracking/tracking.go` 의 `Validate()` 에만 손댔다(프로덕션 파일 1개) — `MaxAllowedHostRunes=300` 상수와, `splitHosts` 루프에 세미콜론 거절·룬 상한 거절. 정규화가 아니라 거절이라 저장된 값의 뜻이 바뀌지 않는다. 커밋 13de176.
- 검증 방법: DSN 이 없어 `docker run postgres:16-alpine`(이름 agenthub-allowhosts-pg, 포트 55471) 로 일회용 DB 를 띄우고 신규 `internal/api/trackingallowlist_live_test.go` 로 두 쓰기 경로를 실제 `Server.Handler()`·실제 세션/CSRF 로 때렸다. 수정 전 헤더에 `…; script-src-elem; …` 가 실제로 열리는 것을 눈으로 확인했다.
- 과제서와 어긋난 곳(중요): 수용 기준 3 의 "DB 없이 도는 테스트" 는 불가능하다 — 두 라우트 모두 `s.authentication` 을 지나고 `Server.store` 는 구체 타입 `*store.Store` 다. 그래서 live 테스트로 갔고, 그 파일은 `AGENTHUB_TEST_DSN` 이 없으면 Skip 한다. 비평가는 DSN 없이 돌리면 이 테스트가 건너뛰어진다는 점을 먼저 볼 것.
- 확신 없는 곳: 길이 오류 메시지는 값을 `cutRunes(host, 300)` 로 잘라 인용하는데, 바로 위 기존 줄(형식 오류)은 원문을 그대로 인용한다 — 두 줄의 인용 방식이 다르다. 의도한 것(같은 경로에서 응답 본문이 커지지 않게)이지만 일관성 지적이 나올 수 있는 자리다.
- 일부러 안 한 것: `PolicySources()`·`pagePolicy()`·`splitHosts`·`AddAllowedHost` 는 건드리지 않았다. 감사 details 의 `"origin"`(internal/api/tracking.go:283)도 따로 자르지 않았다 — 이제 274행 Validate 가 300 룬을 먼저 막는다.
- 다음 역할이 조심할 것: 새 live 테스트는 `system_settings` 의 `tracking` 행을 쓰고 지우며, 원래 행(없음 포함)을 cleanup 에서 되돌린다. 공용 DB 에 `-p 1` 없이 돌리면 다른 패키지의 설정 테스트와 부딪힐 수 있다. 검증에 쓴 컨테이너는 이 회차에서 지웠다.
- [러너 22:55] brief accepted — 채택 — 근거가 코드와 정확히 일치했다(305-307 행만 originOf 를 거치지 않는 비대칭, `pagePolicy` 의 공백 join, `splitHosts` 가 세
- [러너 22:56] verify passed — 검증 5개 통과 (auto)

## 비평 노트
- 새 단위 테스트를 main 분리 워크트리로 옮겨 실제로 돌렸다 — 거절 5건이 전부 pre-fix 에서 수락되어 실패했고, 수락 6건은 pre-fix 에서도 통과했다. 회귀 테스트가 진짜다. 두 쓰기 경로가 Validate 를 지나는 것도 grep 으로 확인했다(routes.go:1736, tracking.go:274 외에 tracking 행을 쓰는 프로덕션 경로는 없다).
- 구현자가 의심한 자리(인용 방식 불일치)는 결함이 아니었다. 대신 tracking_test.go:190-192 주석이 틀렸다: originOf 는 호스트의 ';' 를 남기므로(실행해 확인) 무인증 csp-report → 한 번 클릭 허용이 바로 이번에 닫은 공격 경로였고, 그런 출처는 이제 400 으로 거절된다 — 동작은 옳고 주석만 과장.
- 못 본 것: 실물 브라우저·클러스터·live 테스트(DSN 없음). live 파일은 컴파일만 확인했다.
- 승인이어도 남는 것: (1) 항목 상한은 헤더 전체 길이를 막지 않는다(짧은 줄 다수). (2) 같은 계열이 Momento/MatomoURL 에 그대로 열려 있다(tracking.go:169·180) — 관리자 전용이라 차단은 아니나 다음 회차 후보. (3) 기존에 나쁜 항목이 저장된 배포는 그 줄을 지울 때까지 두 경로가 모두 400 — 릴리즈 노트에 적을 것.
- BASE_VERSION 상향 불필요(base sourcePaths 확인). docs/ADMIN_GUIDE.md:375 에 새 거절 규칙 미반영 — 선택 사항.
- [러너 23:00] review approved — 리뷰 승인 (risk=low)
- [러너 23:00] pr created — https://github.com/hkjang/AgentHub/pull/38
- [러너 23:02] ci passed — 검사 1개 모두 success
- [러너 23:03] merge done — 13de176
- [러너 23:08] release published — v0.255.0
- [러너 23:15] assets verified — v0.255.0 자산 8개 (이전 v0.254.0: 8)
