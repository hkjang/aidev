# 회차 노트 2026-10-03-202753-relio-improve — relio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 20:27] base pinned — main@c6e62b1
- [러너 20:27] autonomy low-risk — 롤백 PR 
- [러너 20:32] scout done — OpenAPI 가 공표하는 정수 질의 파라미터의 **범위·기본값**을 핸들러의 `httpx.IntQuery`/`ClampQuery` 리터럴과 대�

## 구현 노트
- 무엇/왜: `internal/server/openapi_contract_test.go` 에 `TestDocumentedIntegerBoundsMatchTheHandler` 를 더했다. 문서의 모든 `type: integer` 질의 파라미터의 `minimum`/`maximum`/`default` 를, 그 operation 을 답하는 핸들러의 `httpx.IntQuery`/`ClampQuery` 리터럴과 AST 로 대조한다. 두 표가 손으로 따로 적혀 묶이지 않아 한쪽만 고치면 전부 green 인 채로 스키마가 거짓말을 할 수 있었다. 프로덕션 0파일, 커밋 33b9d17.
- **확신 없는 곳**: ① 이것은 **소스 불변식**이다 — 실제 HTTP 요청으로 경계값을 보내 서버가 무엇을 답하는지는 확인하지 않았다. ② 문서→핸들러 **한 방향**만 본다. `IntQuery` 로 읽는 키를 문서가 `type: string` 으로 공표하면 안 걸린다(오늘은 9개 키 전부 `number()` 산이라 해당 자리 없음). ③ 한 핸들러가 여러 route 를 답하고 키가 그중 한쪽에만 문서화된 경우를 만나면 오탐이 날 수 있다 — 오늘 그런 자리가 없어 **검증하지 못했다**. ④ 스캐너는 `httpx.IntQuery`/`ClampQuery` **이름**과 **인자 5개**로만 매칭한다. 다른 이름의 유사 헬퍼가 생기면 조용히 빠지고, 개수 가드(`seen < 38`)는 올라간 수에는 반응하지 않는다.
- 일부러 하지 않은 것: `internal/api/parameters.go` 는 안 고쳤다(38자리 전부 문서와 일치 — 고칠 것이 없다). `queryKeysByHandler` 의 반환형도 안 건드렸다(기존 세 테스트가 무변경으로 남도록 별도 수집기 `intQuerySitesByHandler` 를 뒀다). 역방향 검사와 `choice()` enum 확장은 ideas.json 으로 넘겼다.
- 다음 역할이 조심할 것: DB·네트워크 없이 돈다(`go test ./internal/server/` 만으로 충분). 실패 메시지는 이 파일의 관례대로 **영어**다(과제서 예시는 한국어였다). 삭제된 4줄은 `packageFiles` → `parsePackage` 추출뿐이고 `packageFiles` 는 같은 시그니처로 남아 있다 — 기존 테스트 이름·메시지는 한 글자도 안 바뀌었다. `make test`/npm 계열은 **미실행**(이번 변경과 무관, `npm ci` 가 네트워크를 탄다).
- [러너 20:39] brief accepted — 채택 — 과제서의 실측(호출 38개·35줄, 키별 분포 `limit` 24·`days` 3·`months` 2·`year` 2·`version` 2·`minAgeDays` 2·`expiringDays` 1·`mi
- [러너 20:39] verify passed — 검증 10개 통과 (auto)

## 비평 노트
- 확인한 것: diff 전체(테스트 1파일, 프로덕션 0), 새 테스트를 **독립 섭동**으로 검증 — `parameters.go` 의 `minAgeDays` 상한 3650→3000 → `GET /voices`·`GET /voices/export` 두 건 red, 핸들러 `voice.go:286` path:line 정확. 되돌린 뒤 `git status` 깨끗, `go vet ./...` 0, `go test ./...` 22 ok/FAIL 0, `gofmt -l` 무출력. 원장의 `- 실패 재현:` 줄은 섭동 6종 출력을 담고 있고 성격이 다른 자리를 골라 재현했다.
- 못 본 것: 실제 HTTP 요청으로 경계값 왕복(소스 불변식이라 범위 밖), `ClampQuery` 키의 설명 문구가 클램프 의미를 실제로 적는지, `make test`·npm 계열(변경과 무관).
- **승인이어도 남는 우려(다음 회차 1순위)**: 개수 가드 `knownIntQueries=38`(`openapi_contract_test.go:626`)은 핸들러 쪽만 지킨다. 문서 쪽 `documentedIntegerBounds`(:593-612)에 비어 있음 가드가 없어 `parameters.go:69 number()` 의 `kind` 를 `"integer"` 에서 한 토큰 바꾸면 38자리 비교가 전부 사라지는데도 패키지가 green 이다(섭동 `kind:"number"` → ok 0.286s, 실패 0건 — 직접 확인). 이름 대조 테스트는 이름만 보므로 못 잡는다. 같은 파일 `:456` 의 `len(documented)==0 → Fatalf` 와 같은 모양의 가드 한 줄이 가장 값싼 보강.
- 부차: `:551` 중복 키 검사가 `earlier != site` 로 `position` 까지 비교해, 같은 핸들러가 같은 키를 같은 범위로 두 줄에서 읽으면 동일 범위 두 개를 찍는 오탐이 난다(오늘 해당 자리 없음). 구현자 자평 ③(다중 route 오탐)은 반복이 문서→핸들러 한 방향이라 성립하지 않는다; ②④는 위 우려와 같은 뿌리.
- 릴리즈 노트: 사용자 영향 없음(테스트 전용). 차단 사유 없음 — 새 의존성·마이그레이션·인증/인가·개인정보 변경 없고 revert 는 단일 커밋.
- [러너 20:43] review approved — 리뷰 승인 (risk=low)
- [러너 20:43] pr created — https://github.com/hkjang/relio/pull/41
- [러너 20:47] ci passed — 검사 2개 모두 success
- [러너 20:47] merge done — 33b9d17
- [러너 20:47] release skipped — 자율화 단계 low-risk — 릴리즈는 사람이
