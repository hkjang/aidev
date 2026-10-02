# 회차 노트 2026-10-03-050752-ai-admin-improve — ai-admin
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 05:07] base pinned — main@60beda9
- [러너 05:07] autonomy release — 

## 정찰 노트
- 고른 이유: 이 저장소가 `request_id`(varchar(80))에 대해서는 `boundedRequestID`(server.go:413-440)로 "호출자가 자기 감사 기록을 떨어뜨릴 수 있다"는 결함을 명시적으로 막아 뒀는데, 바로 옆 `resource_id`(varchar(500))에는 같은 보호가 없고 `login`(auth_handlers.go:70)이 **인증 전 공격자 입력(username)** 을 길이 검증 없이 그 컬럼에 넣는다. `s.audit`(server.go:566-585)이 INSERT 오류를 Warn 으로 삼키므로 결과는 "401 은 그대로, 감사 행은 없음" 이다 — 가치 3·위험 2·프로덕션 1파일로 이번 회차 후보 중 가장 높았다.
- 제친 후보: 세션 `user_agent`/`ip` 상한(이미 `truncate(userAgent,1000)`·`net.ParseIP` 로 막혀 있어 기각), 여덟 회차 연속 차선인 chat 절단 reason 테스트(프로덕션 0개로 싸지만 가치 2), `updateKeyScope` 이름 검증(결함은 여전히 남아 있으나 2026-10-01 회차가 그 접근으로 verify-failed 라 재시도 금지에 가깝게 뒀다).
- 추측으로 적은 것(미확인): 레이트 리미터의 정확한 동작은 `login_limit.go:13-32` 상수(backoff 5회·10분·IP 전역 잠금 없음)까지만 읽었고 `allowed()` 본문은 끝까지 보지 않았다. `docs/api.md` 의 감사 이벤트 문단 위치도 이번에 열지 않았다(그래서 문서는 선택 항목으로 뒀다). 통합 테스트·`make lint`·`go build` 는 이번 회차에 실행하지 않았다.
- 구현자가 조심할 것: `audit` 과 `auditInTx` 를 **같은 규칙으로 함께** 고칠 것(한쪽은 오류를 삼키고 한쪽은 트랜잭션을 되돌려 증상이 다르다). username 을 400 으로 거부하는 쪽으로 틀지 말 것 — 401 계약이 바뀌고 길이로 계정 존재를 흘린다. rune 기준 절단을 바이트 기준으로 바꾸면 한글 사례가 깨지는 역검증을 반드시 할 것.
- [러너 05:12] scout done — 실패한 로그인의 감사 기록이 사라지지 않게 `audit_event.resource_id`(varchar(500)) 상한을 감사 기록 경로에서 지

## 구현 노트
- `audit`·`auditInTx` 가 공유하는 `storableResourceID`(`server.go:566` 앞, `truncateRunes` 재사용)로 `resource_id` 를 저장 전에 rune 500자로 잘랐다. 실패 로그인이 인증 전 username 원문을 varchar(500) 컬럼에 넣고 INSERT 오류는 Warn 으로 삼켜져, 501자 username 으로 401 을 받으면서 감사 행을 0건으로 만들 수 있었다(실측).
- 확신 없는 곳·검증 못 한 것: ① 새 테스트는 `audit` 경로만 지난다 — `auditInTx` 는 같은 헬퍼를 쓰지만 호출부 resourceID 가 전부 UUID·상수여서 500자를 넘길 경로를 라우터로 만들 수 없었고, 그래서 테스트로 덮지 않았다(동일 규칙 적용은 코드로만 보장). ② UTF-8 이 아닌 byte 는 다루지 않았다 — JSON 디코딩이 이미 U+FFFD 로 치환하므로 이 경로에서는 도달 불가라고 판단했지만 모든 `audit` 호출부를 그 관점으로 감사하지는 않았다. ③ 500자에서 잘린 사실이 감사 독자에게 표시되지 않는다(의도적, ideas.json 에 후속으로 남김).
- 일부러 하지 않은 것: username 자체를 400 으로 거부하지 않았다(401 `invalid_credentials` 계약이 바뀌고 길이로 계정 존재가 샌다). `internal/auth`·세션 경로, `boundedRequestID`, 마이그레이션, `details`/`action`/`result` 는 손대지 않았다. 같은 `docs/api.md` 감사 절에 있는 CSV 50,000건 문구(열두 회차 보류)는 다른 주장이라 묶지 않았다.
- 다음 역할이 조심할 것: `TestLoginAuditSurvivesOverlongUsername` 은 `TEST_POSTGRES_DSN` 이 없으면 SKIP 된다(전용 폐기 DB 필요 — `DROP SCHEMA ai_admin/aiportal CASCADE` 를 한다). 이번 회차는 포트 55511 의 `ai-admin-pg-1003` 컨테이너를 썼다(세션 종료 후 남아 있을 수 있다). 사례마다 username 이 다른 것은 username 단위 레이트 리미터의 429 를 피하려는 의도이니 합치지 말 것. 웹 변경 없음 → `internal/ui/dist` 재빌드 불필요. VERSION·CHANGELOG 는 건드리지 않았다(1.2.32 그대로).
- [러너 05:20] brief accepted — 채택 — 결함·코드 위치(`auth_handlers.go:70`·`server.go:566`/`:588`·`001_ai_admin.sql` varchar(500))·재사용 헬퍼(`truncateRunes`)·재사용 �
- [러너 05:20] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 판정 approve(risk low, blocking 없음). 전용 postgres(55527, 삭제 완료)로 red→green→byte-revert-red 를 **직접 재현**했다: 7개 서브테스트 PASS(SKIP 아님), `storableResourceID` 래핑만 되돌리면 `:102 resource_id = 0, want 1` 3건과 `:121` 누적 불일치 3건, `truncateRunes` 를 바이트 절단으로 바꾸면 한글 600/501/500자가 다시 FAIL. ledger 의 실패 재현 출력이 전부 사실이었다.
- `go build`·`go vet`·`gofmt -l`·DSN 지정 전체 `go test -count=1 ./...`(internal/server 21.8s, 통합 포함) 통과. 검증용 임시 수정은 전부 되돌려 트리 clean. `-race` 전체와 웹은 이번에 돌리지 않았다(러너 verify 가 npm test/build 를 이미 통과).
- 남는 우려(차단 아님): `truncateRunes` 는 상한 이하면 입력을 그대로 돌려주므로 **유효하지 않은 UTF-8 byte 는 아직 INSERT 에 도달**해 같은 '감사 행 증발' 을 낸다. 로그인은 encoding/json 의 U+FFFD 치환으로 도달 불가이고 다른 resourceID 는 UUID·allowlist 뿐이라 오늘 공격 경로는 없다 — 구현자 불확실 ②의 정확한 범위이고, `boundedRequestID`/`safeReferenceRunes` 선례가 있으니 다음 회차 1순위 후보로 권한다.
- 불확실 ①(auditInTx 미테스트)은 server.go:623 에서 같은 헬퍼가 동일하게 적용된 것을 눈으로 확인했고 호출부가 전부 UUID·상수라 라우터로 500자를 만들 수 없다 — 수용. `resource_type` 은 여전히 무제한이나 전 호출부가 상수/allowlist 라 note 수준.
- 릴리즈 노트에 넣을 것: 실패 로그인의 username(인증 전 입력)이 이제 500자로 잘려 **확실히 보존**된다는 점. 감사 이벤트 자동 보존 삭제가 없다는 사실은 `docs/operations.md:184` 에 이미 공개돼 있어 privacy 소견은 차단이 아니다.
- [러너 05:25] review approved — 리뷰 승인 (risk=low)
- [러너 05:25] pr created — https://github.com/hkjang/ai-admin/pull/37
- [러너 05:33] ci passed — 검사 2개 모두 success
- [러너 05:33] merge done — 7ce2332
- [러너 05:44] release published — v1.2.33
- [러너 05:46] assets verified — v1.2.33 자산 2개 (이전 v1.2.32: 2)
