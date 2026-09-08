**인가 Endpoint가 처리하지 못한 요청이 이 서비스의 어떤 신호에도 남지 않았습니다.** 이 Endpoint가 만드는 자기 쪽 실패 여섯 중 넷은 RP의 `redirect_uri`로 302에 `error=server_error`를 실어 보냅니다 — 사양이 리다이렉트 가능한 오류를 거기 두라고 하므로 그 자체는 옳습니다. 문제는 **302가 인가에 성공했을 때 나가는 상태와 같다**는 것입니다. `resso_http_requests_total`은 어느 쪽이든 정상 Redirect로 세고 접근 로그도 `status=302`라, **Realm의 모든 인가를 가져간 장애가 바쁘게 잘 도는 Endpoint와 구별되지 않았습니다** — 사람이 로그인 화면에 도달하지 못할 뿐이고, RP들은 **여기서 아무도 볼 수 없는 오류**를 받았습니다.

### 수정

- **넷 중 셋은 store 에러를 그 자리에서 버려 로그 한 줄도 없었습니다** — `SessionAuthenticatedRecently`, `CreateAuthorizationCode`, `CreateAuthorizationRequest`. 즉 장애를 알아볼 방법이 이쪽에는 **하나도 남아 있지 않았습니다.**
- **근거는 같은 모양의 Endpoint 둘에 이미 있었습니다.** Token은 `resso_token_errors_total`, Introspection은 `resso_introspection_errors_total`이 **정확히 이 이유로** 존재합니다 — 응답의 겉모습이 정상과 같아서 요청 카운터가 둘을 구분하지 못하는 자리입니다. 인가에만 그 계열이 없었습니다.
- **이제 `resso_authorization_errors_total{stage}`가 그것을 셉니다.** `stage`는 여섯 값으로 고정 카디널리티입니다 — `realm`(경로의 Realm 조회), `client`(`client_id` 조회), `sso_session`(브라우저 세션 조회), `auth_time`(`max_age` 판정을 위한 최근 인증 시각), `authorization_code`(코드 생성), `authorization_request`(로그인 화면으로 넘길 요청 저장).
- **리다이렉트되지 않고 여기서 500으로 답하는 앞의 둘도 같은 계열에 셉니다.** 그 둘은 요청 카운터에도 이미 보이지만, 한 계열로 모아야 **"처리하지 못한 인가"를 다른 계열과 조인하지 않고 한 번에** 볼 수 있습니다. 그 둘은 각자의 로그 문구가 이미 문서에 적혀 있으므로 **로그는 그대로 두고 카운터만** 더했습니다.
- **호출자 때문에 생기는 답은 세지 않습니다.** 없는 Realm의 404도, 등록되지 않은 `client_id`의 400도 이 계열을 만들지 않습니다 — 이 계열은 **이쪽이 답을 내지 못한 것**만 담습니다.
- 새 연동 테스트가 테이블을 차례로 **RENAME으로 숨기며**(`realms`·`clients`·`sso_sessions`·`authorization_codes`·`authorization_requests`) 여섯 단계를 모두 확인하고, 수정 전 코드에서 **여섯 건이 모두 실제로 실패하는 것을 확인**했습니다(어느 `stage`도 계열이 없었습니다). `auth_time`만은 숨길 테이블이 없어서(그 쿼리가 읽는 테이블은 모두 앞의 세션 조회가 먼저 읽습니다) **데이터베이스가 interval로 만들 수 없는 `max_age`**로 그 쿼리 안에서만 실패하게 했습니다. 정상 인가 둘(로그인 화면으로 park, 세션 재사용으로 코드 발급)과 호출자 쪽 답 둘이 **계열을 만들지 않는 것**, 테이블 복구 후 회복도 같은 테스트가 함께 지킵니다.
- README 지표 표에 한 줄, `docs/operations.md` 경보 절에 `stage` 값 목록과 **"302라 성공한 인가와 상태가 같다"는 읽는 법**을 적었습니다.

### Upgrade notes

동작은 달라지지 않습니다: 응답도, 상태 코드도, 저장되는 데이터도 전과 같고 **마이그레이션도 없습니다.** `/metrics`에 계열 하나(`resso_authorization_errors_total`, 라벨 `stage` 여섯 값)가 늘어나고, 앞서 조용히 버려지던 store 실패 셋에 서버 로그 `an authorization request could not be served`가 생깁니다. 스크랩 직후 이 계열이 **0이 아닌 값으로 보인다면 새 결함이 아니라 지금까지 302에 가려져 있던 장애**이므로, `stage`가 가리키는 단계부터 보시면 됩니다. 경보를 새로 거신다면 `docs/operations.md`의 항목을 그대로 쓰시면 되고, 되돌릴 때 이전 이미지도 그대로 동작합니다.
