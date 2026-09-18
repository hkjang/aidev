관리 화면 없이 환경변수(`DARTFLY_OIDC_*`)로만 SSO 를 구성한 배포에서도 자동 로그인(`prompt=none`)을 켤 수 있게 했고, 그 환경변수들을 처음으로 문서화했습니다.

**환경변수로만 SSO 를 둔 배포는 자동 로그인을 켤 방법이 없었습니다.**

v2.68.0 에서 들어온 자동 로그인(`auto_login`)은 관리 화면 `/admin/sso` 의 스위치였습니다. compose·k8s 처럼 관리 화면을 쓰지 않고 `DARTFLY_OIDC_ISSUER` 등 환경변수로만 SSO 를 두는 배포에는 그에 해당하는 값이 없어, IdP 세션이 있어도 늘 로그인 화면을 거쳐야 했습니다.

| 환경변수 | 기본값 | 동작 |
| --- | --- | --- |
| `DARTFLY_OIDC_AUTO_LOGIN=true` | | IdP 세션이 있으면 로그인 화면 없이 들어옴(`/api/system/info` 의 `sso_auto_login: true`, 인가 URL 에 `prompt=none`) |
| 미설정 / `false` | `false` | 평소 로그인 화면(인가 URL 에 `prompt` 없음) |
| `maybe` 같은 잘못된 값 | | 다른 스위치와 같이 기동을 거부하고 이유를 출력 |

우선순위는 예전과 같습니다. 관리 화면에 저장된 SSO 설정이 하나라도 있으면 환경변수는 보지 않습니다(끈 채로 저장해도 마찬가지). 화면의 "설정 삭제"로 지워야 다시 환경변수로 돌아갑니다.

환경변수 → SSO 설정 변환이 `runtime.go` 에 흩어져 있어 값을 하나 더할 때 빠뜨리기 쉬웠습니다(`auto_login` 이 그렇게 빠졌습니다). `Config.ssoEnvFallback()` 한 곳으로 모으고, 9개 값이 모두 그대로 옮겨지는지 테스트로 묶어 두었습니다.

**`DARTFLY_OIDC_*` 가 문서에 한 줄도 없었습니다.**

환경변수 가이드(`docs/environment-variables.md`)에 "1.5 SSO(OIDC) 설정" 절을 새로 두었습니다. 9개 변수의 타입·기본값·설명, 관리 화면 설정과의 우선순위, `DARTFLY_OIDC_CLIENT_SECRET_FILE` 등 `_FILE` 접미사를 함께 적었고, MCP 절에 잘못 들어 있던 `DARTFLY_SSO_ALLOW_INSECURE` 를 이 절로 옮겼습니다. 관리자 가이드의 자동 로그인 항목에서 이 절로 링크합니다.

**검증.** `gofmt`/`go vet`/`go test -race ./...` 38개 패키지 통과(새 테스트 5개 — 환경변수 9개 값이 `sso.Config` 로 그대로 옮겨짐(하나를 빼면 실패하는 것 확인)·기본 꺼짐·`maybe` 거절·env 폴백 `Init` 이 실제 Discover 를 지나 `Provider.Config().AutoLogin` 까지 전달·저장 설정이 env 를 이김). JS 테스트 12파일 통과. `test/smoke/run.sh` 실제 바이너리+MariaDB+Chromium 브라우저 31페이지 통과. 같은 메타 DB 로 환경변수만 준 두 번째 바이너리와 가짜 디스커버리 서버에서 `AUTO_LOGIN=true` → `sso_auto_login: true`·인가 URL 에 `prompt=none`, 미설정 → `false`·`prompt` 없음, `maybe` → 기동 거부 메시지까지 확인했습니다. 배포 이미지는 `test/smoke/artifact.sh` 로 실제 기동해 healthz/readyz·임베드 정적 자산·로그인·내장 마스터 키 질의·TZ 적용을 확인했습니다.

오프라인 배포: `gunzip -c dartfly-v2.70.0.tar.gz | docker load`
