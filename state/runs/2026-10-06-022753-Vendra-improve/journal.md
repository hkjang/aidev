# 회차 노트 2026-10-06-022753-Vendra-improve — Vendra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 02:28] base pinned — main@96bfabf
- [러너 02:28] autonomy release — 

## 정찰 노트
- 골랐다: 업무 목록이 `order=amount_desc` 를 금액 권한 없이 조용히 최신순으로 떨어뜨리면서 응답에 그 말을 넣지 않는 것. 이 저장소가 세 회차 연속 고쳐 온 「가림을 사실로 답한다」와 같은 가족인데 권한을 넓히는 쪽이 아니라 **응답이 진실을 말하게** 하는 쪽이라 위험이 더 낮고, `truncated` 라는 선례가 응답 모양을 이미 정해 준다. 프로덕션 1파일. 제친 후보: 지출 0/null 통일(계약 결정 선행, M), 대시보드 업무 카운트 권한(좁히는 변경은 기존 역할의 화면을 0 으로 만든다 — 기각), MCP 쪽 전부(미병합 fcce3d7 과 충돌).
- 확인한 것: `objectOrderBy` 의 조용한 낙하(objects.go:119-131), `listObjects` 응답 키 4개(:113), 웹 드롭다운 「금액 높은순」(Objects.tsx:329)과 URL 값을 그대로 보여 주는 select(:176), `order`/`sort` 를 읽는 자리가 저장소에 1곳뿐, 픽스처가 모든 계약을 `amount=99000000`·기본 `now()` 로 심는 것, 응답 키 집합을 단정하는 테스트와 웹 스키마 검증이 없는 것, `TestObjectOrderByProtectsRedactedAmounts` 가 지금 PASS 하는 것(실제 실행).
- 추측(미확인): 응답 필드 이름을 `order` 로 한 것과 값 `updated_desc`(웹 기본값 Objects.tsx:176 에서 가져왔다 — 더 나은 이름이 있으면 구현자가 바꿀 것). 통합 테스트를 DB 로 실행해 보지는 않았다(이번 회차는 DB 불필요 테스트만 돌렸다). 기각한 후보 하나는 근거를 확인했다: `listObjects` 가 스캔 실패 행을 조용히 버리는 것은 비포인터 컬럼이 전부 NOT NULL 이라 실패를 만들 수 없어 ideas.json 에 rejected 로 적었다.
- 구현자가 조심할 것: `objectOrderBy` 의 시그니처·반환 SQL 을 바꾸면 기존 테스트를 고쳐야 하게 되고 그것이 이 회차의 대조군을 없앤다 — `objectOrderApplied` 를 더해 `objectOrderBy` 가 그것을 읽게 하는 방향으로. 테스트에서 금액 순서와 `updated_at` 순서를 어긋나게 심지 않으면 두 정렬이 같은 순서를 내 아무것도 증명하지 못한다. `grantOnly` 는 역할 표 루프 **안에서** 부를 것(2026-10-05 함정).
- [러너 02:34] scout done — 업무 목록이 **적용하지 못한 정렬**을 적용한 척 답하는 것 고치기 — `order=amount_desc` 가 `<type>.amount.read` �

## 구현 노트
- `listObjects` 의 응답에 **적용된** 정렬(`"order"`)을 담았다. `order=amount_desc` 는 `<type>.amount.read` 없이 조용히 최신순으로 떨어지는데(의도된 보호) 응답이 그 말을 안 해서 웹 드롭다운·URL 은 계속 `amount_desc` 를 보여 주고 사용자는 최신순 목록의 맨 위를 「가장 큰 계약」으로 읽었다. `truncated` 와 같은 모양이다.
- 새 `objectOrderApplied` 가 「무엇이 적용됐나」의 유일한 출처이고 `objectOrderBy` 는 그것을 `switch` 한다. 그래서 `case "amount_desc"` 안의 `if amountVisible` 은 죽은 분기가 되어 걷어냈다 — **반환 SQL 문자열과 시그니처는 무변경**이고 `TestObjectOrderByProtectsRedactedAmounts` 가 고치지 않은 채 통과하는 것이 그 증인이다.
- 확신 없는 곳: 필드 이름 `order` 와 값 `updated_desc` 는 웹 드롭다운의 `<option value>` 네 개(Objects.tsx:327-330)와 정확히 맞춰 고른 것이지 외부 계약이 아니다. 다른 목록 핸들러(suppliers 등)는 이 필드를 갖지 않으므로 응답 모양이 핸들러별로 다르다 — 의도한 범위다.
- 검증 못 한 것: **웹 스위트를 돌리지 못했다** — 이 워크트리에 `web/node_modules` 가 없어 `npm test` 가 `vitest is not installed` 로 끝난다. `web/` 은 한 줄도 바뀌지 않았고 `Objects.tsx:198` 의 응답 타입은 구조적 타입 캐스트라 JSON 키 추가에 영향받지 않는다(zod 류 스키마 검증 없음, 키 집합을 단정하는 Go 테스트도 없음) — 그래도 돌려서 확인한 것은 아니다.
- 일부러 안 한 것: 웹이 이 필드를 화면에 보여 주는 것(다음 회차 후보, ideas.json 2번), `order` 를 400 으로 거절하는 것(웹이 URL 값을 그대로 되보내 목록이 통째로 빈다), 다른 목록 핸들러로 넓히는 것(`order` 를 읽는 자리는 저장소에 objects.go 한 곳뿐).
- 다음 역할이 조심할 것: 새 `internal/httpapi/object_order_integration_test.go` 는 **DB 가 있어야 돈다**(세 DSN). 전용 컨테이너는 `vendra-1006-improve-pg`(127.0.0.1:55491, trust)로 아직 떠 있다. 테스트가 `amount` 와 `updated_at` 을 **어긋나게** 심는 것이 핵심이다 — 픽스처는 모든 계약을 `amount=99000000`·기본 `now()` 로 넣어 두 정렬이 우연히 같아지므로, `seedOrderedContract` 의 명시 값을 건드리면 테스트가 아무것도 증명하지 않게 된다.
- [러너 03:05] brief accepted — 채택 — 근거가 코드와 좌표까지 그대로 맞았고(`:119-131` 의 조용한 낙하, `:113` 의 응답 키 4개, `order` 를 읽는 자리 1곳) 수
- [러너 03:06] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 판정 approve(risk low, blocking 없음). 실패 재현을 손으로 만들었다 — `git checkout main -- internal/httpapi/objects.go` 로 프로덕션 1파일만 되돌리면(테스트가 `objectOrderApplied` 를 참조하지 않아 빌드된다) 새 테스트 둘이 `answered no order field` 로 FAIL 하고 그 본문에 `count:4`·`truncated:false` 는 있고 `order` 는 없다 — 이번 증상 그대로. 복원 후 PASS.
- DB 회귀를 실제로 돌렸다: 세 DSN 을 `vendra-1004-improve-pg`(127.0.0.1:55471)에 붙여 `go test ./internal/... ./cmd/...` 전부 ok(httpapi 28.1s), gofmt·vet 깨끗. 구현 노트의 `vendra-1006-improve-pg` 는 이미 사라졌다 — 다음 역할은 1004 를 쓸 것.
- 안전 확인: `objectOrderApplied` 는 네 리터럴만 반환하고 SQL 보간 경로에 닿지 않으며, `objectOrderBy` 의 반환 SQL·시그니처 무변경으로 `TestObjectOrderByProtectsRedactedAmounts` 가 수정 없이 통과한다(실행). 권한 확대·개인정보·의존성 추가 없음 → security/legal 차단 사유 없음.
- 못 본 것: 웹 스위트 미실행. 다만 `web/` 0줄 변경이고 `Objects.tsx:198` 이 구조적 캐스트(zod 없음, 응답 키 집합 단정 테스트 없음)라 키 추가가 깨뜨릴 경로가 코드상 없다. 드롭다운 option 값 네 개가 새 필드 값과 정확히 일치함도 확인.
- 남는 우려(릴리즈 노트): 웹은 아직 `order` 를 읽지 않아 **화면의 드롭다운은 여전히 「금액 높은순」** 이다 — 고친 것은 API 응답이지 화면이 아니니 노트가 「화면이 바로잡힘」으로 읽히지 않게 할 것. 사소: objects.go:93-94 가 `hasPermission` 을 두 번 평가(무해).
- [러너 03:09] review approved — 리뷰 승인 (risk=low)
- [러너 03:10] pr created — https://github.com/hkjang/Vendra/pull/141
- [러너 03:12] ci passed — 검사 2개 모두 success
- [러너 03:12] merge done — fe3f618
- [러너 03:15] release published — v0.7.70
- [러너 03:16] assets verified — v0.7.70 자산 1개 (이전 v0.7.69: 1)
