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
