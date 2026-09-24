**로그아웃이 `post_logout_redirect_uri`를 쓰지 못하고 버렸을 때, 그 이유를 이제 기록합니다.** 이전 버전은 요청받은 주소를 쓰지 못해도 평범한 로그아웃과 똑같이 답했고 아무것도 남기지 않았습니다. 주소가 쓰이려면 세 가지가 모두 맞아야 하는데(요청에 담겨 있을 것 · Client가 확인될 것 · 그 Client의 등록 목록에 있을 것), 어느 하나가 어긋나도 결과는 같은 302(이 서비스의 로그인 화면)나 204였고, `LOGOUT` 감사 항목은 주소를 요청하지 않은 로그아웃과 구별되지 않았습니다. 특히 RP 설정 오류에서 가장 흔한 두 경우 — 모르거나 꺼진 `client_id`, 등록되지 않은 주소 — 는 로그에도 트레일에도 닿지 않아, 이 Endpoint가 만들어내는 문의("로그아웃하면 애플리케이션으로 돌아오지 않는다")의 답이 정작 그 상황을 만든 서비스 안에 없었습니다.

### 수정

- **버린 이유를 이유 코드로 남깁니다.** `oidcLogout`이 세 조건을 하나의 and로 묶는 대신 풀어 판정하고 `client_not_named`(`id_token_hint`도 `client_id`도 없음) · `client_unknown`(없거나 꺼진 Client, 이 Realm의 키로 검증되지 않는 `id_token_hint`) · `client_unavailable`(`clients` 조회 실패 — 이쪽 장애) · `uri_not_registered`(Client는 찾았으나 주소가 등록 목록에 없음) 중 하나를 고릅니다.
- **`LOGOUT` 감사 상세와 서버 로그 양쪽에 씁니다.** 감사 상세에 `post_logout_redirect_uri: dropped`와 `reason`이, 해결된 경우에만 `client_id`가 붙습니다. 같은 사실이 `logout dropped the post-logout redirect it was asked for`(level `WARN`) 한 줄로도 남습니다 — 쿠키 세션이 없는 요청은 감사 항목 자체가 생기지 않으므로 그때는 이 로그가 유일한 기록입니다.
- **요청된 주소 원문은 감사에도 로그에도 넣지 않습니다.** 호출자가 임의로 정하는 값이라 트레일에 저장했다가 다시 내보내지 않습니다. 기록되는 식별자는 이 서비스가 저장하고 있는 `client_id`뿐입니다.
- **거절은 늘어나지 않습니다.** 상태 코드, 리다이렉트, 쿠키 삭제 순서, `post_logout_redirect_uris`의 정확 일치 규칙이 모두 그대로이고, 요청한 대로 리다이렉트한 로그아웃은 감사 상세에 아무것도 더하지 않습니다.
- `docs/operations.md`에 «로그아웃 뒤 RP로 돌아가지 못할 때» 절을 추가해 네 가지 `reason`의 뜻과 담당자가 할 일을 표로 적고, 기존 `the client named at logout could not be looked up` 로그 행이 `client_unavailable`에 대응함을 밝혔습니다.

### 확인

- 새 연동 테스트 `TestIntegrationLogoutSaysWhyItDroppedTheRedirect` — 실제 PostgreSQL과 프로덕션 `New(...).Handler()`, `httptest` 서버로 정상 302(+`state`)와 204 기준선을 먼저 잡은 뒤, 주소가 버려지는 여섯 경우의 감사 상세를 검사하고, 세션 쿠키가 없는 경로에서 `WARN` 로그만 남는 것과 `clients` 테이블 RENAME으로 만든 `client_unavailable`까지 확인합니다. 요청 주소 원문이 감사·로그 어디에도 없음을 단언합니다.
- 수정 전 `oidc.go`에서 이 테스트의 단언 10개가 실제로 실패하고(기준선 둘은 통과) 수정 후 모두 통과하는 것을 확인했습니다.
- 릴리즈 준비에서도 `make lint`, `make test`(Go race 전 패키지·연동 SKIP 0·`go vet`·프런트 테스트·빌드), `make build VERSION=v0.9.90`, 버전 일치 검사, `git diff --check`가 통과했습니다.

### Upgrade notes

관측 정보만 늘어나며 **로그아웃의 동작은 달라지지 않습니다** — 상태 코드, 리다이렉트 목적지, 쿠키 삭제, 등록 주소 정확 일치 규칙이 모두 이전과 같습니다. 마이그레이션도 설정 변경도 없고 이전 `v0.9.89` 이미지로 롤백할 수 있습니다(되돌리면 버린 이유를 다시 알 수 없게 될 뿐입니다).

감사 로그를 기계로 읽는 도구를 쓴다면, `LOGOUT` 항목의 상세에 `post_logout_redirect_uri` · `reason` · `client_id` 키가 **새로 나타날 수 있다**는 점만 확인하세요. 주소를 요청한 적 없거나 요청대로 리다이렉트한 로그아웃의 상세는 전과 같습니다. `WARN` 로그가 한 종류 늘어나므로 로그 양이 조금 증가할 수 있는데, RP 설정이 올바르면 나오지 않는 줄입니다.

배포 후에는 이 `WARN`의 발생량을 먼저 보세요. `uri_not_registered`나 `client_unknown`이 꾸준히 보이면 그 RP의 설정을 고치라는 뜻이고, `client_unavailable`이 보이면 RP가 아니라 데이터베이스 쪽을 확인해야 합니다.
