**RP의 silent SSO 무한 루프가 이 서비스의 어떤 신호에도 보이지 않았습니다.** 이미 로그인한 사람이 다른 사내 서비스를 열 때 로그인 화면 없이 들어가는 것(silent SSO)은 RP가 인가 요청에 `prompt=none`을 붙여 시작하고, ReSSO는 제공자로서 둘 중 하나로만 답합니다 — 이 브라우저에 SSO 세션이 있으면 302로 인가 코드, 없으면 302로 `error=login_required`. 뒤의 것은 실패가 아니라 "세션이 없다"는 평범한 답이라 RP는 자기 로그인 화면을 보여 주고 멈춰야 하는데, 그것을 실패로 읽고 다시 `prompt=none`을 보내는 RP는 브라우저를 두 호스트 사이에서 Redirect 속도로 오가게 하고 사용자는 화면이 깜빡이는 것만 봅니다. 문제는 **두 답이 모두 302**라는 것입니다. `resso_http_requests_total`은 어느 쪽이든 정상 Redirect로 세고 접근 로그도 `status=302`라, **RP 하나가 고장 난 루프가 바쁘게 잘 도는 Endpoint와 구별되지 않았습니다** — 캠페인의 RP 스무 개가 `prompt=none`을 보내는 상대가 ReSSO이고, 그 루프는 제공자에서만 한눈에 보이는데도 그랬습니다.

### 추가

- **`resso_silent_authentications_total{result}`가 `prompt=none` 요청을 답변별로 셉니다.** `result`는 `code`(세션이 있어 코드 발급)와 `login_required` 둘로 고정 카디널리티이고, 코드 발급은 `prompt=none`일 때만 셉니다. 근거는 같은 모양의 계열 셋(`resso_token_errors_total`·`resso_introspection_errors_total`·`resso_authorization_errors_total`)에 이미 있습니다 — 응답의 겉모습이 정상과 같아서 요청 카운터가 둘을 구분하지 못하는 자리입니다.
- **거절 한 건에 로그 한 줄.** 계열에 RP별 라벨을 두지 않는 대신 서버 로그 `a silent authentication found no session to reuse`가 `client_id`·`remote_ip`·`trace_id`를 담습니다. 루프는 **한 `remote_ip`에서 같은 `client_id`가 초당 여러 번** 나타나는 모양이라 어느 RP인지는 여기서 찾습니다.
- **세션을 읽지 못한 것은 여전히 `server_error`이고 이 계열이 아닙니다.** 그 요청은 전과 같이 `resso_authorization_errors_total{stage="sso_session"}`에 셉니다 — 거절이 아니고, RP는 `login_required`를 "로그아웃했다"로 읽고 자기 세션도 끝내므로 이쪽 장애를 그렇게 알리면 전 RP 로그아웃이 됩니다. 표준의 "실패가 아니라 평범한 답"이 성립하려면 진짜 실패는 다른 답이어야 합니다.
- **루프 방지 장치와 `auto_login`은 RP 쪽입니다.** ReSSO에는 그에 해당하는 설정이 없고 새로 만들지도 않았습니다 — 제공자가 받은 `prompt=none`을 무시하면 OIDC Core 3.1.2.1을 어기는 것이고, 리다이렉트가 생기는 자리를 묶는 설정은 요청을 만드는 쪽에 있어야 합니다. `docs/compatibility.md`에 «RP가 silent SSO를 구현할 때» 절(제공자가 하는 것 셋, 한 탭 세션에 한 번만 시도·로그아웃 뒤 억제·거절을 주소에 남기기 같은 RP 쪽 규칙, 숨은 iframe이 아니라 최상위 이동을 써야 하는 이유 — ReSSO는 `frame-ancestors 'none'`입니다)을 적었습니다.
- 관리자 가이드 5-1에 «로그인 화면 없이 들어가기 (silent SSO)» 소절 — **`auto_login`은 ReSSO가 아니라 각 서비스에 있고 기본 꺼짐**이라는 것, 콘솔 자체는 같은 쿠키라 설정 없이 이미 그렇게 동작한다는 것, 루프를 보는 자리와 고칠 곳 — 과 장애 대응 표에 "화면이 깜빡이며 로그인 화면이 뜨지 않는다" 행을 더했습니다. `docs/operations.md` 경보 목록에 급증 읽는 법, README 지표 표에 한 줄을 적었습니다.

### 확인

- 새 연동 테스트가 대화형 인가 둘은 세지 않음 → 세션 없는 `prompt=none`이 `state`를 실은 `login_required`로 1 → 세션 있는 `prompt=none`이 코드로 1 → `sso_sessions` 테이블을 RENAME으로 숨긴 `prompt=none`이 `server_error`이며 이 계열은 그대로이고 `authorization_errors{stage="sso_session"}`만 1 → 거절 반복 셋이 4까지 오르는 것을 고정하고, **수정 전 핸들러에서 여섯 단언이 실제로 실패하는 것을 확인**했습니다.

### Upgrade notes

동작은 달라지지 않습니다: 응답도, 상태 코드도, 저장되는 데이터도 전과 같고 **마이그레이션도, 설정 변경도 없습니다.** `/metrics`에 계열 하나(`resso_silent_authentications_total`, 라벨 `result` 두 값)가 늘어나고, `prompt=none` 거절마다 서버 로그 한 줄이 생깁니다. 스크랩 직후 `result="login_required"`가 **초당 여러 번으로 보인다면 새 결함이 아니라 지금까지 302에 가려져 있던 RP의 루프**이므로, 로그의 `client_id`로 그 서비스를 찾아 개발자에게 알리시면 됩니다 — 고칠 곳은 ReSSO가 아니라 그 서비스이고, ReSSO에서 Client를 끄면 그 서비스의 로그인 전체가 멎으니 마지막 수단으로만 씁니다. 되돌릴 때 이전 이미지도 그대로 동작합니다.
