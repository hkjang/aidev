# 회차 노트 2026-10-06-155844-relio-improve — relio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 15:58] base pinned — main@dd69a26
- [러너 15:58] autonomy low-risk — 롤백 PR 

## 정찰 노트
- 고른 이유: 최근 4회차가 모두 "손 목록 두 개를 테스트로 묶는 그물" 이었는데, eventType 자리는 저장소가 **이미 더 좋은 답을 갖고 있다** — workspace.go:61-66 의 CauseEvidenceLevels/KnowledgeStatuses 공유 var + setOf() 관용이고 mcp/voice_tools.go:128 이 그것을 참조한다. 그래서 그물 대신 그 관용을 적용해 손 목록을 1개로 줄이고, 남는 DB CHECK 와만 부분집합 테스트로 묶는다. 보류 목록의 release.yml(보호 경로)·esbuild(네트워크)·net.OpError·ClientIP(폐기용 DB·설정축 필요)는 전부 짧은 자율 회차 범위 밖이라 제쳤다.
- 실측한 것: 베이스라인 `go test ./internal/voice/ ./internal/mcp/ ./internal/api/` 3개 ok · server.go:24 가 voice 를 이미 import · internal/api→internal/voice 순환 없음 · auth.Has 가 비공개 perm 맵을 본다 · Comment 의 검사 순서가 s.Get 앞 · 016:56 CHECK 10값 상위집합.
- 추측으로 적은 것: 기준 3 테스트를 `internal/api/enum_contract_test.go` 에 두는 것이 최선이라는 판단(파서 재사용 때문). 구현자가 `internal/voice` 쪽에 작은 전용 파서를 두는 편이 낫다고 보면 바꿔도 되지만 **파서를 복제하지는 말 것**.
- 조심할 것: 기준 5 테스트는 `&auth.Principal{IsBootstrap: true}` 를 쓰고 **유효한** eventType 으로 Comment 를 부르지 말 것(nil DB 패닉). `"invalid eventType"` 문장과 service.go:836 SLA 분기, 두 엔드포인트의 서로 다른 기본값은 손대지 말 것.
- 프로필은 1일 전 것이고 이번에 읽은 자리와 어긋나지 않아 새로 쓰지 않았다. 코드·커밋 변경 없음.
- [러너 16:05] scout done — VOC 기록 `eventType` 값 목록을 저장소의 기존 관용(`voice.CauseEvidenceLevels` 식 공유 var)으로 한 곳에 모으고, 그 

## 구현 노트
- `{CUSTOMER_CONTACT, COMMENT, ESCALATED}` 가 MCP 스키마(`mcp/server.go:683`)와 집행(`voice/service.go:821`) 두 곳에 따로 하드코딩돼 있던 것을, 같은 패키지에 이미 있던 `setOf()` 관용으로 `voice.CommentEventTypes`/`commentEventTypes` 하나로 모았다. 공표·집행 일치는 이제 타입 시스템이 지키고, 남는 손 목록인 `customer_voice_events.event_type` CHECK 와의 관계만 `internal/api` 의 새 부분집합 테스트가 지킨다. 커밋 e00f244, 프로덕션 3파일 +10/−2.
- **확신 없는 곳** — ① 기준 5 테스트의 공표값 수락 검사는 `recover()` 로 nil 풀 패닉을 삼켜 "eventType 검사를 통과했다" 를 증명한다. 패닉이 그 검사 **뒤**에서만 날 수 있다는 코드 순서(`service.go:818-826`)에 의존하므로, 누가 검사 순서를 바꾸면 이 테스트의 의미가 조용히 약해진다(틀린 green 이 되지는 않지만 증명력이 떨어진다). 비평가가 여기를 먼저 봐 주면 좋겠다. ② 과제서의 섭동 ③ 예측은 **틀렸다** — `causeEvidence` 오타도 `"BOGUS"` 를 거부하므로 거부 assertion 만으로는 red 가 안 난다. 그래서 테스트를 강화했고 강화 뒤 red 를 눈으로 확인했다.
- **검증 못 한 것** — 유효 eventType 이 실제로 **수락되어 이벤트가 기록된다**는 것은 증명하지 않았다(폐기용 PostgreSQL 없이는 불가; 과제서 함정 2). 증명한 것은 "유효 값이 eventType 검사를 통과한다" 뿐이다. `make test`·프런트·Docker·릴리즈 경로는 돌리지 않았다(Go 전용 변경, `npm ci` 가 네트워크를 탄다).
- **일부러 하지 않은 것** — MCP(`CUSTOMER_CONTACT`)와 REST(`COMMENT`)의 서로 다른 기본값 통일(제품 의도 판단이 필요한 동작 변경, `first_responded_at` SLA 의미가 다르다), `docs/api-mcp.md:96` 의 산문 네 번째 사본(오늘 내용이 맞다), `CommentEventTypes` 의 JSON 노출(OpenAPI 계약 변경), `knowledge_status` 쪽 동일 작업(미확인, 다음 회차 후보).
- **다음 역할이 조심할 것** — 두 새 테스트는 DB 없이 돈다(embed 된 `migrations.Files` 와 nil 풀 거부 경로만). `"invalid eventType"` 문장은 `serviceError` 의 영어 substring 분류가 읽으므로 한 글자도 바꾸지 말 것. 섭동 되돌리기에 `git checkout -- <자기 작업이 든 파일>` 을 쓰면 본 변경까지 날아간다(이번에 한 번 겪어 복원했다) — `migrations/` 에만 쓰고 나머지는 역-sed 로.
- [러너 16:11] brief accepted — 채택 — 과제서의 실측이 코드와 전부 일치했다: `server.go:683` 의 인라인 리터럴과 `server.go:24` 의 기존 `voice` import, `service.g
- [러너 16:11] verify passed — 검증 10개 통과 (auto)

## 비평 노트
- 판정 approve(risk low, blocking 없음). 두 새 테스트를 섭동으로 직접 red 확인: 016:56 CHECK 에서 ESCALATED 제거 → enum_contract_test.go:435 FAIL(파서가 008 옛 CHECK 대신 016 최신본을 집는 것도 확인), service.go:821 을 causeEvidence 로 교체 → service_test.go:283 이 세 값 모두 FAIL. 섭동은 역-sed/백업으로 전부 되돌려 git status clean.
- 구현자의 '확신 없는 곳 ①' 은 염려보다 약하다 — 루프 앞 BOGUS 단언은 recover() 밖이라 검사 순서가 바뀌면 nil 풀 패닉이 테스트를 큰 소리로 깨뜨린다(Get 은 s.DB 를 바로 역참조). 조용한 green 으로 썩지 않는다.
- 원장에 `- 실패 재현:` 줄이 없다. 버그 수정이 아니라 불변식 리팩터라 재현할 증상이 없어 과제서대로 diff·섭동으로 대신 확인했다. 보안·법무: 인가·allowlist 의미 완전 보존($3 바인딩 유지, 개인정보/비밀값/엔드포인트 변경 없음) — 차단 사유 없음.
- 실행: go build·vet·go test ./... 전체 PASS, -race(voice·mcp·api) PASS, gofmt -l internal/ 공백. 못 본 것: 프런트(npm)·Docker·릴리즈 경로, 실제 PostgreSQL 에서 유효 eventType 이 기록되는지(구현자와 동일한 한계).
- 다음 회차가 알 것: ① api 테스트는 부분집합 한 방향만 본다(CHECK→공표 방향 오염은 못 잡음). ② CommentEventTypes 는 export 공유 슬라이스가 MCP 스키마에 그대로 들어간다 — 지금 in-place sort/append 하는 코드는 없음을 확인했으나(arguments.go 의 sort 는 전부 지역) 나중에 생기면 전역 검증 집합이 오염된다. ③ OpenAPI 는 REST eventType enum 을 공표하지 않는 기존 공백이 남아 있다. ④ knowledge_status 동일 작업이 다음 후보로 유효하다. 릴리즈 노트는 사용자 가시 동작 변경 없음(순수 내부 통합+테스트).
- [러너 16:15] review approved — 리뷰 승인 (risk=low)
- [러너 16:15] pr created — https://github.com/hkjang/relio/pull/44
- [러너 16:16] ci failed — 성공이 아닌 검사: test=failure · 실패한 검사: ? 잡: test 
