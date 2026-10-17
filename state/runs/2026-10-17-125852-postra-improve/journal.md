# 회차 노트 2026-10-17-125852-postra-improve — postra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:59] base pinned — main@c40b91d
- [러너 12:59] autonomy release — 

## 구현 노트
- 623dee1: IMAP Dial이 아무 첫 행이나 greeting으로 받던 것을 RFC 3501 §7.1.1의 untagged OK/PREAUTH/BYE로 좁혔다(`isGreeting`). 수정 전에는 프레임을 맞춰 답하는 비-IMAP 상대에서 동기화가 **빈 메일함 성공**으로 보고됐다(앱 단 실측).
- 확신 없는 곳: (1) 최종 운영 문구는 `greeting 단계: 알 수 없는 오류`(class=other)로, 포트·프로토콜 오설정을 지목하지 못한다 — 힌트 추가는 일부러 뺐고 ideas.json에 pending으로 남겼다. (2) `"*  OK"`(공백 2개)·탭 구분 같은 비표준 변형은 이제 거부된다. 그런 서버를 실측한 적은 없고 좁히는 방향이라 수용했다. (3) class=other가 재시도 대상이어서 오설정에 연결 시도 3회가 붙는다(재시도 정책은 손대지 않음).
- 일부러 하지 않은 것: greeting 진단의 Timeout 메타데이터(commandTO 기록)는 PR #39로 닫힌 접근이라 미접촉. PREAUTH·BYE 분기의 대소문자 민감 prefix 검사도 그대로 — 넓히면 LOGIN 생략/거부 판정이 바뀐다.
- 다음 역할이 조심할 것: `internal/application/imap_greeting_test.go` 는 실제 루프백 TCP + `imap.Dialer{}` 로 돌고(DB·네트워크 외부 의존 없음, SQLite 테스트 앱) `fastRetries` 를 쓰므로 `t.Parallel` 과 함께 쓰지 말 것. 어댑터 테이블 테스트는 `CommandTimeoutSec: 2` 로 묶여 있어 가드가 사라지면 느리게 실패하지 않고 "명령을 보냈다"로 실패한다.
- [러너 13:12] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인: `isGreeting` 경계(bare `* OK`, `* OKAY`, 소문자, CRLF 트림)를 손으로 따졌고 vet + imap 패키지 전체 + application 3개 테스트를 재실행해 PASS. 원장의 실패 재현 출력이 이번 증상과 일치하고, 어댑터 테스트가 `rec.seen()` 공백을 단언하므로 가드 없이는 통과 불가 — 테스트가 실제로 바뀐 경로를 지난다.
- 확인: 서버 원문 미유출(오류는 길이만, BYE Code 는 `inboundCodes` 허용목록), 가드가 STARTTLS Buffered() 판정보다 앞서 실행, 인증·마이그레이션·비밀값 미접촉. security/legal 차단 사유 없음.
- 못 봄: 전체 `go test ./...`, PG·브라우저 e2e·프런트, lint-security/contracts(원장 기록만 신뢰). spa/assets 무변경.
- 승인 후 우려(릴리즈 노트): 비표준 greeting(`*  OK`·탭·`*OK`)을 보내던 서버는 이제 greeting 단계 실패다 — 운영자에게 보이는 동작 변경이므로 노트에 명시 권장.
- 다음 회차: class=other 가 재시도 대상이어서 포트 오설정에 연결 시도 3회가 붙고, `syncHint` 의 AccountSessions>1 분기가 먼저 걸려 '동시 접속 제한/IDLE' 힌트가 오설정에도 나올 수 있다(기존 정책). 원장 보류 아이디어의 greeting 전용 힌트가 이 자리.
- [러너 13:16] review approved — 리뷰 승인 (risk=low)
- [러너 13:16] pr created — https://github.com/hkjang/postra/pull/40
- [러너 13:21] ci failed — 성공이 아닌 검사: govulncheck=failure · 실패한 검사: ? 잡: govulncheck 
