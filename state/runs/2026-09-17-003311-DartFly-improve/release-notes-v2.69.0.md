깊은 링크로 들어온 사람이 조용한 SSO 에서 거절당하거나 SSO 가 실패한 뒤 로그인하면 원래 자리 대신 `/` 로 떨어지던 문제를 고쳤고, 조용한 시도가 설정 오류로 돌아올 때 로그에 남기도록 했습니다.

**깊은 링크의 돌아갈 자리가 사라졌습니다.**

`/admin/users?q=1` 같은 깊은 링크로 들어오면 401 → `/login?return_to=` → 조용한 시도(`prompt=none`)로 IdP 에 다녀옵니다. 그런데 IdP 가 `login_required` 로 거절하면 콜백이 `/login?sso=none` 으로만 보내 `return_to` 를 버렸습니다. 이어서 비밀번호나 SSO 단추로 로그인하면 원래 자리가 아니라 `/` 로 떨어집니다. SSO 오류(`access_denied`, state 검증 실패 등)로 돌아올 때도 같았습니다.

| 상황 | 이전 | 이후 |
| --- | --- | --- |
| 조용한 시도 거절(`login_required`) | `/login?sso=none` → 로그인 후 `/` | `/login?sso=none&return_to=…` → 로그인 후 원래 자리 |
| SSO 오류(`access_denied` 등) | `/login?sso_error=…` → 로그인 후 `/` | `/login?sso_error=…&return_to=…` → 로그인 후 원래 자리 |
| state 검증 실패 | `/login?sso_error=…` → 로그인 후 `/` | `/login?sso_error=…&return_to=…` → 로그인 후 원래 자리 |

콜백이 로그인 화면으로 보내는 모든 경로에서 흐름 쿠키의 `return_to` 를 함께 들고 갑니다. 같은 출처의 경로(`safeReturnTo` 를 통과한 값)만 붙고, 밖을 가리키는 값이나 `/` 는 붙지 않습니다.

**조용한 시도의 설정 오류가 어디에도 남지 않았습니다.**

`prompt=none` 에 대해 IdP 가 세션 없음 4종(`login_required`·`interaction_required`·`consent_required`·`account_selection_required`, OIDC Core 3.1.2.6)이 아닌 값(`invalid_client`, `invalid_request` 등)으로 돌아오면 IdP 클라이언트 설정 문제입니다. 그런데 아무 데도 남지 않아 자동 로그인이 왜 한 번도 되지 않는지 알 길이 없었습니다.

이제 사용자에게는 똑같이 조용히 넘어가되 서버 로그에 `조용한 SSO 시도가 세션 없음이 아닌 오류로 돌아옴` 경고를 오류값·설명과 함께 남깁니다. 세션 없음은 정상 대답이므로 로그를 남기지 않습니다. 진단 방법은 `docs/ADMIN_GUIDE.md` 에 적었습니다.

**검증.** `gofmt`/`go vet`/`go test -race ./...` 38개 패키지 통과(새 테스트 2개 — 거절·오류·state 실패에서 `return_to` 유지, 밖을 가리키는 값은 붙지 않음, 설정 오류만 Warn 로그·세션 없음은 로그 없음). JS 테스트 12개 통과. `test/smoke/run.sh` 실제 바이너리+MariaDB+Chromium 브라우저 31페이지 통과. 같은 서버에서 curl 로 흐름 쿠키의 `return_to` 가 `/login` 주소까지 따라오고, 쿠키가 없으면 붙지 않는 것을 확인했습니다. 배포 이미지는 `test/smoke/artifact.sh` 로 실제 기동해 healthz/readyz·임베드 정적 자산·로그인·내장 마스터 키 질의·TZ 적용을 확인했습니다.

오프라인 배포: `gunzip -c dartfly-v2.69.0.tar.gz | docker load`
