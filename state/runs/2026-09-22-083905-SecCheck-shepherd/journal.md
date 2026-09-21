# PR 처리기 노트 2026-09-22-083905-SecCheck-shepherd — SecCheck PR #15
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-22-075422-SecCheck-improve)
# 회차 노트 2026-09-22-075422-SecCheck-improve — SecCheck
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:54] base pinned — main@eb2a3b0
- [러너 07:54] autonomy release — 

## 정찰 노트
- MCP 리포트 세 필드의 JSON 타입 소실을 선택: 실제 집계가 무필터로 넓어지는 경로이며 2파일 내에서 고칠 수 있다. 반복 게이트/문서/정적 검사 과제는 제외.
- 서비스 timezone 계약 때문에 날짜 프리셋은 여러 화면을 함께 검증해야 해 후순위; 새 후보는 MCP query 혼입과 리포트 응답 순서 역전.
- HTTP 결함 재현은 미확인. 기존 리포트 테스트 실행은 DSN 부재로 SKIP; 전용 DB에서 실패→통과를 증명해야 한다. 원격 CI/vulndb 최신 상태도 미확인.
- 350d36c 날짜 검증은 현재 HEAD에 없으므로 재구현 금지. 역할 검사 유지, 전역 stringValue/인증/마이그레이션/워크플로는 건드리지 말 것.
- [러너 07:58] scout done — MCP 심의 리포트의 잘못된 JSON 타입을 필터 생략으로 처리하지 않기 (가치 4 / 위험 1 / 작업량 M)

## 구현 노트
- 837d1e8: callMCPTool의 기존 역할 검사 뒤 세 필드의 비문자열(null 포함)을 -32602로 거부. 변경은 mcp.go 7줄과 실 HTTP 회귀 테스트 한 파일.
- 전용 PostgreSQL 16에서 두 프로토콜×세 필드×5타입 30건 Red→Green→검사 제거 시 Red 확인. 정상 9조건의 REST 전체 JSON=MCP structuredContent 및 실제 제외 행 수, REQUESTER -32001, 성공 감사 기록 부재 확인.
- 회귀 44 PASS/0 FAIL/0 SKIP; race PASS 3.128s; go test ./... -json 419 PASS/0 FAIL/0 테스트 SKIP(web 175.602s); vet/build exit 0. 테스트 없는 2패키지의 package skip은 DB 테스트 skip과 구분.
- 확신 없는 곳·미검증: 원격 CI/vulndb 최신 상태, 프런트/PDF/전체 race는 미검증. 날짜 유효성·URL query 혼입·다른 도구 스키마/정상 감사 정책은 의도적으로 범위 밖.
- 다음 역할: TEST_POSTGRES_DSN 필수. 이번 전용 컨테이너는 검증 뒤 제거. 감사 테이블은 audit_logs. fixture는 네 심의를 먼저 생성한 뒤 날짜 조정(생성 사이 전년으로 옮기면 번호 재사용에 따른 CREATE_FAILED 관찰).
- 필수 세 technology SKILL.md를 지정 headcount 경로에서 직접 읽고 적용; Skill 호출 도구 없음. 상세 증거는 red/green/reverted-red/regression/race.log 및 all-tests.jsonl, vet/build.log.
- [러너 08:06] brief accepted — 채택 — 역할 검사 뒤 타입 소실 경로와 실제 무필터 성공 응답을 확인했으며, 실제 감사 테이블명은 audit_events가 아닌 aud
- [러너 08:07] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- reject / security: 실제 main...HEAD에 포함된 무인증 Momento 범용 프록시. 수리는 internal/web/analytics.go:208 및 server.go:245부터 확인.
- 로컬 main=c32eec7로 pinned eb2a3b0와 다름: 146파일을 포함하며 프록시 결함은 523cd0f 유래. MCP 커밋 자체에서는 차단 결함을 찾지 못함.
- MCP 타입·권한·감사·실 HTTP 단언, 인증/추적 배선·마이그레이션을 확인. DB 없는 추적 테스트 5 PASS; MCP DB 테스트는 DSN 부재 SKIP.
- 원격 CI/vulndb·프런트/PDF·전체 race 및 실제 수집기 공격은 미검증. 릴리즈 전 비교 기준 불일치와 034~036 DB 변경 범위를 확인해야 함.
- [러너 08:08] review rejected — 리뷰 거절: internal/web/analytics.go:208-218 [P1] Momento 프록시를 추적용 경로·메서드로 제한해야 합니다. server.go:245에서 인증 래퍼 없이 /momento/ 전체를 등록하
- [러너 08:08] review blocked — 검토 부서 차단 소견(security) — 수리·중재 없이 운영자의 위험 수용(risk-accepted) 필요
- [러너 08:08] pr created — https://github.com/hkjang/SecCheck/pull/15

## 심사 노트
- origin/main eb2a3b0...HEAD 837d1e8은 MCP 2파일뿐; 이전 146파일 비교와 구분했고 지정 3스킬을 직접 읽음.
- 전용 PostgreSQL 16의 실제 NewServer/HTTP 회귀 2개를 -race로 실행해 PASS, SKIP 없음; 타입·권한·감사·REST/MCP 일치 확인, 소스 수정 없음.
- Momento 범용 무인증 프록시는 기준/HEAD 모두에 잔존(analytics.go:208-218, server.go:245); 기존 security 차단 해제 권한이 없어 reject/human, risk medium 권고.
- 원격 CI·취약점 DB·전체/프런트/PDF·실제 수집기 공격·수정 전 실행은 미검증; 운영자가 기존 차단 위험 수용 여부를 결정해야 함.
