# 회차 노트 2026-10-02-013734-jikim-improve — jikim
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 01:37] base pinned — main@eeca591
- [러너 01:37] autonomy release — 

## 정찰 노트
- `_ =` 광맥은 소진됐다(internal·cmd 전수 grep: 남은 자리는 전부 정당). 그래서 같은 **"원인 X 를 원인 Y 로 보고"** 가족에서 새 자리를 찾아 `aiChat` 의 단일 error code 를 골랐다 — CSP 속도 제한·nonce 파싱·TOCTOU 는 전부 새 계약이나 외부 재현(브라우저·PostgreSQL)이 필요해 한 세션에 안 끝난다.
- 확신하는 것: 라우트(server.go:141), 거절이 `s.store` 앞에서 반환되는 순서(ai.go:89→93→97), `secret_material_rejected` 가 저장소에 ai.go:94 한 곳뿐(go/ts/tsx/md 전체 grep), `go test ./internal/httpapi/ -count=1` 이 base 에서 0.589s green.
- 추측으로 적은 것: 새 code 이름 `invalid_ai_request` 는 이웃 코드 스타일에서 고른 제안이다(못박힌 계약 아님). 수용 기준 1 을 만족하면 다른 이름도 된다.
- 구현자가 조심할 것: `ValidateAIInput` 은 exported 이고 `security_test.go:16-40` 이 6번 직접 호출한다 — 시그니처를 바꾸지 말고 sentinel + `errors.Is` 로 갈 것. `decodeJSON` 이 `DisallowUnknownFields` 라 테스트 본문에 모르는 키를 넣으면 `invalid_json` 으로 빠진다. `secretMaterialPatterns`·한도 값·role 집합은 건드리지 말 것(매처 확장은 과거 반려 사유).
- [러너 01:42] scout done — AI 채팅 입력 검증 실패를 전부 `secret_material_rejected` 로 보고하는 문제 분리 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- 무엇을 왜: `aiChat` 이 `ValidateAIInput` 의 모든 오류를 `secret_material_rejected` 로 내보내 Secret 과 무관한 네 가지 거절(빈 입력·잘못된 role·빈 message·`max_tokens` 범위)까지 "Secret 평문이 섞였다"로 보고했다. sentinel `errAISecretMaterial` + `errors.Is` 분기로 나머지를 `invalid_ai_request` 로 돌렸다. 프로덕션 파일 1개(`internal/httpapi/ai.go`, 11줄 추가) + 테스트 파일 1개 신규.
- 확신 없는 곳·검증 못 한 것: **`bash scripts/verify.sh` 전체를 통과시키지 못했다** — 프런트 vitest 단계에서 새로 설치된 jsdom/undici 가 Node 24 를 요구해 `webidl.util.markAsUncloneable is not a function` 으로 jsdom 환경 테스트 파일 8개가 시작조차 못 한다(로컬 Node v22.23.1). `web/` 는 base 와 바이트 단위로 동일하므로 이 변경과 무관한 환경 사유다(ideas.json 에 새 후보로 기록). 대신 lint·build 는 직접 돌려 통과했다. 262144바이트 초과 분기는 왕복 테스트에 넣지 않았다(큰 본문이 `maxJSONBody` 2MiB 와 얽혀 엉뚱한 걸 증명할 위험 — 같은 `invalid_ai_request` 경로를 다른 네 케이스가 지난다).
- 일부러 하지 않은 것: `secretMaterialPatterns`·`maximumAITokens`·262144 한도·role 허용 집합·HTTP 상태코드·`ValidateAIInput` 시그니처는 손대지 않았다. `aiStream`·`aiIntegrationTest` 는 `ValidateAIInput` 을 부르지 않거나 다른 경로여서 범위에서 뺐다 — 그쪽도 같은 분기가 필요하면 다음 회차 후보다.
- 다음 역할이 조심할 것: 새 테스트는 DB 없이 돈다(nil store — 검증 거절이 `s.store.AIConfig` 앞에서 반환된다). `invalid_ai_request` 라는 이름은 이웃 code 스타일에서 고른 것이고 못박힌 외부 계약이 아니다. Secret 분기의 한국어 메시지 원문은 테스트가 바이트 단위로 고정했다 — 바꾸지 말 것.
- [러너 01:46] brief accepted — 채택 — 배선 사실(라우트 server.go:141, 거절이 `s.store` 앞에서 반환되는 ai.go:89→93→97 순서, `secret_material_rejected` 가 저장소
- [러너 01:46] verify failed — 실패한 검증: cd web && npm test --silent (exit 1)
