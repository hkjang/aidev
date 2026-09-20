목록·지표 API가 정수가 아니거나 음수인 질의 파라미터를 조용히 기본값으로
바꾸던 것을 400 `invalid_query`로 거부하도록 고쳐 릴리스합니다. OpenAPI
계약(`minimum: 1`)과 실제 동작을 맞추는 수정이고 스키마·설정·정상 호출의
동작은 그대로이므로 patch를 올립니다.

- `page`·`page_size`·`limit`·`hub_id`가 정수가 아니거나 음수이면
  400 `invalid_query`("<이름>은(는) 0 이상의 정수여야 합니다")로 응답 —
  `GET /users?hub_id=abc`가 "모든 Hub", `?page=abc`가 1쪽,
  `?page_size=-5`가 20건으로 조용히 응답하던 문제
- 적용 경로: `/users`·`/local-users`·`/servers`·`/audit`·`/metrics`·
  `/usage/consumption`·자원 목록. 빈 값→기본값, 0과 상한 초과는 종전처럼
  store가 보정하고 `hub_id=0`="모든 Hub"는 유지
- OpenAPI에 `InvalidQuery` 응답 컴포넌트를 두고 해당 경로에 `400`을
  문서화. flow mapping 안에 인용 없이 쓰인 `?`·`,`를 인용해 yaml.v3·PyYAML이
  문서를 읽지 못하던 문제 수정
- 프런트는 항상 숫자를 보내므로 화면 변경 없음. 마이그레이션·설정 변경 없음

