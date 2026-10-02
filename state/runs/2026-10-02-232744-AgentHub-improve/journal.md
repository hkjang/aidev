# 회차 노트 2026-10-02-232744-AgentHub-improve — AgentHub
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 23:27] base pinned — main@741af80
- [러너 23:27] autonomy release — 

## 정찰 노트
- 왜 이것: `allowTrackingOrigin`(api/tracking.go:254)·`AddAllowedHost`·`splitHosts`(tracking.go:432,446) 세 함수를 실제로 읽고 확인한 **배선 차이**다 — 핸들러는 "출처 하나" 를 받는다고 믿고 뒤쪽 파서는 쉼표·공백을 구분자로 쓴다. 이미 live 테스트 파일(trackingallowlist_live_test.go)이 이 라우트를 실제 라우터·관리자 세션·CSRF 로 통과하므로 대역 없이 증명된다.
- 제친 후보: PolicySources 중복 제거와 matchesWildcard 는 열어 보고 기각했다(전자는 2026-09-30 의 합계 상한이 이미 영향을 묶었고 '정규화가 아니라 거절' 관례와 어긋남, 후자는 CSP 와일드카드 의미와 사실상 같고 영향이 배지 하나). 콘솔 상한 안내는 숫자를 프런트에 손으로 적는 위험이 있어 차선으로 남겼다.
- 추측으로 적은 것: `store.AuditTrail` 이 `details` 를 그대로 돌려주는지와 `decodeJSON` 의 본문 상한은 이번에 열어 보지 않았다(과제서에 "미확인" 으로 적음). 수용 기준 3 의 두 선택지(변화 없음 표기 vs 감사 행 생략) 중 하나를 고르는 것은 구현자에게 맡겼다.
- 구현자가 조심할 것: 단일 항목 판정을 `Validate()` 나 `splitHosts` 에 넣으면 설정 폼(여러 항목 textarea)이 깨진다 — `allowTrackingOrigin` 한 곳에만 걸 것. 응답 `{allowedHosts: string}` 형태는 AdminSettings.tsx:201 이 읽으므로 유지.
- 검증: `go test ./internal/tracking ./internal/api` 를 이 워크트리에서 실제로 돌려 통과 확인(api 1.709s, DSN 없음). 코드는 바꾸지 않았다.
- [러너 23:32] scout done — 한 번 클릭 허용(`POST /api/v1/admin/tracking/violations/allow`)이 **한 개의 출처만** 받고, 감사 기록이 실제로 저장�

## 구현 노트
- 바꾼 것: `tracking.SingleHost` 를 더해 `allowTrackingOrigin` **한 곳에서만** 단일 항목을 판정하고(쉼표·공백·탭·줄바꿈이 섞이면 `invalid_origin` 400), 감사 `details.origin` 을 `AddAllowedHost` 가 실제로 넣는 저장 형태로, 목록이 늘었는지는 새 `details.added` 로 말하게 했다. 프로덕션 파일 2개.
- 확신 없는 곳: `details.added` 라는 **새 감사 키**는 이 저장소에 선례가 없다 — 감사 화면/내보내기가 알려진 키만 그리는지 확인하지 않았다(`details` 는 jsonb 라 스키마 변경은 없고, `AuditTrail` 이 그대로 돌려주는 것은 실측했다). 수용 기준 3 의 두 선택지 중 "감사 행을 남기지 않는" 쪽을 고르지 않은 이유는 핸들러 주석에 적었다.
- 확신 없는 곳: 변화가 없을 때도 `PutSetting` 을 그대로 호출한다(기존 동작 유지) — 같은 값을 다시 쓰므로 `updated_at`·`updated_by` 가 갱신된다. 일부러 건드리지 않았다.
- 일부러 하지 않은 것: 프런트(`AdminSettings.tsx`)는 손대지 않았다 — 응답 `{allowedHosts: string}` 그대로다. 다만 새 거절 문구("한 번에 한 곳만 더할 수 있습니다")가 화면에 그대로 뜨는지는 실물 브라우저로 확인 못 했다. `Validate()`·`splitHosts`·`AddAllowedHost`·`PolicySources`·설정 폼 경로(routes.go 1728행)는 한 줄도 바꾸지 않았다.
- 새로 안 사실: 위반 보고에서 온 출처 중 `SingleHost` 가 새로 거절할 수 있는 유일한 형태는 authority 에 쉼표가 든 것(`https://a,b.corp.example` — `url.Parse` 를 통과한다). 어떤 브라우저도 해석하지 못한 호스트이고 그대로 넣으면 정책 소스가 둘이 되므로 거절이 맞다고 판단했고, 그 관계를 `tracking_test.go` 에 `Recorder` 로 고정했다. 공백·탭·줄바꿈이 든 blocked-uri 는 `originOf`/Recorder 단계에서 이미 버려진다(실측).
- 다음 역할이 조심할 것: 새 live 서브테스트(`internal/api/trackingallowlist_live_test.go`, "one click adds one origin…")는 **DB 가 있어야 돈다** — `AGENTHUB_TEST_DSN` 과 base64 32바이트 `AGENTHUB_ENCRYPTION_KEY` 가 없으면 파일 전체가 `t.Skip`. 이 회차에는 `docker run --rm -d -p 55447:5432 postgres:16-alpine` 로 띄워 `go test -race -count=1 -p 1 ./cmd/... ./internal/...` 를 exit 0 으로 돌렸다. `errorCode(t, recorder)` 헬퍼를 이 파일에 패키지 수준으로 더했으니 같은 이름을 `internal/api` 에 또 만들지 말 것. BASE_VERSION 상향 불필요(`runtime-images.json` 직접 확인).
- [러너 23:43] brief accepted — 채택 — `allowTrackingOrigin`·`AddAllowedHost`·`splitHosts` 의 배선 차이와 감사 행의 불일치가 현재 코드와 정확히 일치했고, 지정
- [러너 23:43] verify passed — 검증 5개 통과 (auto)

## 비평 노트
- 확인한 것: diff 전체(프로덕션 2파일), `SingleHost` 호출자가 `allowTrackingOrigin` 한 곳뿐임, `Normalized()` 가 AllowedHosts 를 건드리지 않아 `added := updated != settings.AllowedHosts` 가 정확함(중복 시 `AddAllowedHost` 가 기존 문자열을 그대로 돌려주고, 추가 시 항상 길어진다), 원장의 실패 재현 3건 행 번호 314·345·364 가 현 테스트 파일의 해당 단언과 일치함, 콘솔은 `item.origin` 하나만 보내고 `details` 를 `unknown` 으로 받음, `docs/ADMIN_GUIDE.md:404` 가 이미 "출처 하나 허용" 임. `go test ./internal/tracking ./internal/api` 통과·gofmt·vet 깨끗.
- 못 본 것: DB live 테스트(이 세션에서 postgres 를 띄우지 않았다 — 구현자의 exit 0 보고를 그대로 받았다), 실물 브라우저에서 새 거절 문구가 ErrorBanner 로 뜨는지, 감사 내보내기 화면의 눈으로 본 모습.
- 승인이어도 남는 우려(릴리즈 노트·다음 회차): ① 핸들러가 `SingleHost` 를 거친다는 고정은 DSN 게이트 안에만 있어 CI 에서 돌지 않는다 — 다음에 호출을 지우면 CI 는 녹색이다. ② 변화 없는 클릭도 `PutSetting` 을 호출해 설정 행의 `updated_at`·`updated_by` 가 갱신되므로 "목록이 마지막으로 바뀐 시점" 은 `details.added` 로만 알 수 있다. ③ 대소문자만 다른 중복 클릭에서 `details.origin` 은 저장된 항목이 아니라 보낸 대소문자를 적는다(301행 주석과 이 한 경우 어긋남).
- 차단 소견 없음: 인가·라우팅·마이그레이션·의존성 변경 없고, 관리자 입력이 CSP 헤더로 가는 경로를 좁히는 쪽이며 새로 수집·전송하는 개인정보가 없다.
- [러너 23:47] review approved — 리뷰 승인 (risk=low)
- [러너 23:47] pr created — https://github.com/hkjang/AgentHub/pull/40
- [러너 23:50] ci passed — 검사 1개 모두 success
- [러너 23:50] merge done — d257da1
- [러너 23:56] release published — v0.257.0
- [러너 00:01] assets verified — v0.257.0 자산 8개 (이전 v0.256.0: 8)
