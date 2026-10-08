# Invenqor Server·Agent v0.2.34 릴리즈 노트

릴리즈 일자: 2026-09-13
호환 Agent: v0.2.34 (Linux·Windows)

이번 릴리즈는 기능 하나를 더합니다. 자리는 **Keycloak 로그인**이고, 문제는
Keycloak 에 **이미 로그인한** 사용자도 콘솔을 열 때마다 로그인 화면을 보고
**Keycloak으로 계속** 을 눌러야 했다는 점입니다. 이제 **설정 → Keycloak** 의
**Keycloak 세션이 있으면 자동 로그인**(`auto_login`)을 켜면, 콘솔은 로그인 화면을
그리기 전에 OIDC `prompt=none` 으로 한 번 조용히 시도하고, 세션이 있으면 사용자를
열었던 자리로 바로 들여보냅니다. 세션이 없으면 로그인 화면이 한 번 보이고 **다시
시도하지 않습니다.** 기본값은 꺼짐입니다.

## 1. Keycloak 에 로그인한 사용자도 매번 로그인 화면을 거쳤습니다

콘솔은 `/api/v1/auth/me` 가 세션 없음을 답하면 곧바로 로그인 화면을 그렸습니다.
Keycloak 쪽에 살아 있는 세션이 있어도 그것을 물어볼 길이 없었으므로, 사내 포털에서
링크를 타고 들어온 사용자는 화면 하나를 더 거치고 버튼 하나를 더 눌러야 했고,
북마크한 깊은 주소는 로그인 뒤 `/` 로 돌아갔습니다.

이제 `auto_login` 이 켜져 있으면 다음 순서로 동작합니다.

1. 콘솔은 `/api/v1/auth/me` 와 `/api/v1/auth/methods` 의 답을 **둘 다** 기다립니다.
   세션이 없고 `keycloak_auto_login` 이 참이면 로그인 화면을 그리지 않고
   `/api/v1/auth/keycloak/start?prompt=none&return_to=<현재 주소>` 로 **최상위
   이동**합니다. 숨은 iframe 이 아니므로 서드파티 쿠키 차단이나 Keycloak 의 프레임
   정책과 무관합니다.
2. Keycloak 은 `prompt=none` 에 화면을 그리지 않습니다. 세션이 있으면 인가 코드가
   바로 돌아와 평소 로그인 흐름을 타고, 사용자는 `return_to` 로 돌아갑니다.
   `return_to` 는 `/` 로 시작하고 `//` 로 시작하지 않는 같은 오리진 경로만 받으며
   그 밖의 값은 `/` 가 됩니다.
3. 세션이 없으면 `error=login_required` 로 돌아옵니다. 이것은 실패가 아니라 "로그인
   안 되어 있음" 이라는 평범한 대답이므로 Server 는 `KEYCLOAK_PROVIDER_REJECTED` 로
   기록하지 않고 `/?sso=none` 으로 보내 로그인 화면을 보여 줍니다.

로그인 화면은 두 답이 모두 오기 전에는 그려지지 않고, Keycloak 으로 떠나는 동안에도
그려지지 않습니다. 화면이 잠깐 보였다가 사라지는 깜빡임이 이 기능이 없애려는
바로 그것이기 때문입니다.

## 2. 거절 뒤에 다시 시도하면 무한 루프이므로 세 겹으로 막습니다

`prompt=none` 의 어려움은 시도가 아니라 **거절에 반응하지 않는 것**입니다. 거절
뒤에 한 번 더 시도하면 브라우저는 Keycloak 과 콘솔 사이를 끝없이 오갑니다.

| 장치 | 동작 |
|---|---|
| 한 탭 세션에 한 번 | 시도 여부를 `sessionStorage` 에 남깁니다. 거절 뒤 새로고침해도 다시 시도하지 않고, 새 탭은 다시 시도합니다. |
| 로그아웃 뒤 억제 | 콘솔에서 로그아웃하면 억제 표시를 남깁니다. 로그아웃 직후 조용히 다시 로그인되면 로그아웃이 고장 난 것처럼 보이기 때문입니다. 다시 세션이 생기면 지워집니다. |
| 주소의 표시 | 거절은 `/?sso=none` 으로 돌아옵니다. 브라우저 저장소가 지워졌어도 이 주소와 `auth_error` 가 붙은 주소에서는 시도하지 않습니다. |

브라우저 저장소를 **읽지 못하면 "이미 시도했다" 로 간주**합니다. 사생활 보호
모드와 사이트 데이터 차단은 읽기에서 예외를 던지는데, 그것을 "아직 안 했다" 로
읽으면 곧 루프입니다. `/api` · `/v1` · `/health` · `/mcp` · `/momento` 처럼 Server 가
직접 답하는 경로에서는 시도하지 않으며, 이미 세션이 있으면 시도하지 않습니다.

## 3. 리다이렉트가 생기는 자리는 관리자 설정에만 묶입니다

`prompt=none` 은 최상위 리다이렉트를 만들므로, 그것이 일어날 수 있는 자리는 누구나
붙일 수 있는 쿼리 문자열이 아니라 관리자의 결정에 묶여야 합니다.

- `auto_login` 이 꺼져 있으면 `?prompt=none` 을 붙여 불러도 Server 는 **조용히
  평범한 로그인으로 바꿉니다.** `/api/v1/auth/methods` 의 `keycloak_auto_login` 은
  Keycloak 이 실제로 로그인을 완료할 수 있는 상태(client secret 까지 갖춰진 상태)
  에서만 참이 되므로, 완료할 수 없는 provider 로 리다이렉트만 만드는 일이 없습니다.
- 거절을 인식하는 것은 **이 Server 가 `prompt=none` 으로 시작한 흐름**에 한합니다.
  `oidc_flows` 에 `silent` 열이 더해져 흐름마다 그 사실을 기억하고, 콜백은 그
  `state` 를 한 번 쓰고 폐기하므로 재사용할 수 없습니다. 만료되었거나 이미 소비된
  흐름, 이 Server 가 모르는 `state` 는 거절로 인정하지 않습니다.
- 평범한 로그인의 `login_required` 나 `access_denied` 같은 실제 오류는 전과 같이
  `KEYCLOAK_PROVIDER_REJECTED` 로 기록되고 사용자에게 안내됩니다.

새 항목은 `openapi.yaml` 에 반영되었습니다 — Keycloak 설정의 `auto_login`,
`GET /api/v1/auth/methods` 의 `keycloak_auto_login`, `GET /api/v1/auth/keycloak/start`
의 `prompt=none`, 그리고 콜백의 `/?sso=none` 리다이렉트. 관리자 가이드 16.2절이
설정, 동작 순서, 루프 방지 장치와 확인 방법을 설명합니다.

## 검증

- **Server.** `auto_login` 이 꺼진 동안 `prompt=none` 이 URL 에 들어가지 않고
  켜면 들어가는 것, `login_required` 가 `silent` 흐름에서만 거절로 인식되고 평범한
  흐름에서는 실패로 기록되는 것, 거절 뒤 `state` 가 폐기되는 것, 깊은 주소가
  `return_to` 로 돌아오는 것, 거절 응답 네 종류(`login_required` ·
  `interaction_required` · `consent_required` · `account_selection_required`)만
  인정되는 것을 `internal/auth` 와 `internal/httpapi` 의 테스트로 고정했습니다.
  HTTP 계층 테스트는 가짜 provider 를 띄워 설정이 꺼진 상태·켜진 상태·거절·재시도
  차단을 한 흐름으로 확인합니다.
- **콘솔.** 시도 여부를 정하는 함수는 브라우저 없이 시험할 수 있도록 순수하게
  두었고, 설정 꺼짐·Server 경로·`sso=none`·`auth_error`·로그아웃 표시·이미 시도·
  저장소 읽기 예외의 각 경우에 시도하지 않는 것과, `return_to` 가 같은 오리진
  경로만 받는 것을 vitest 로 고정했습니다.
- **저장 모드 양쪽에서.** `go test ./...` 가 SQLite fallback 과 실제
  PostgreSQL(`scripts/test-postgres.sh`) 양쪽에서 전 패키지 통과하고,
  `go vet` · `go build` 가 통과합니다. 콘솔의 vitest 가 통과하고, 체크인된
  `server/internal/webui/dist` 는 React 빌드 결과와 같습니다.
- 새 토글의 캡처는 가이드에 싣지 않았습니다. 캡처 스크립트·PNG·가이드 세 곳을
  대조하는 테스트가 있어 다음 릴리즈의 과제로 남깁니다.

## 호환성

- **데이터베이스 마이그레이션 008** 이 `oidc_flows` 에 `silent` 열을 더합니다.
  기본값이 있으므로 기동 시 자동으로 적용되며, 진행 중이던 로그인 흐름은 평범한
  흐름으로 남습니다. 멀티 Pod 에서는 전과 같이 PostgreSQL advisory lock 아래에서
  한 Pod 만 적용하므로 기동 순서를 맞출 필요가 없습니다.
- **기본값은 꺼짐이고, 꺼져 있으면 동작 변경이 없습니다.** 로그인 화면, Keycloak
  로그인 흐름, `/api/v1/auth/methods` 의 기존 필드가 이전과 같습니다. 새 필드
  `keycloak_auto_login` 하나가 더해질 뿐입니다.
- 켜면 Keycloak 에 로그인한 사용자는 로그인 화면을 보지 않습니다. 로컬 계정으로
  들어가야 하는 사용자는 Keycloak 에서 로그아웃하거나 콘솔에서 로그아웃한 뒤
  같은 탭에서 로그인 화면을 쓰면 됩니다. 로그인 화면의 **Keycloak으로 계속** 은
  설정과 무관하게 평범한 로그인을 시작합니다.
- 기존 REST API, Query DSL, MCP 도구는 이번 릴리즈에서 바뀌지 않았습니다.
- Agent(Rust) 동작은 이번 릴리즈에서 바뀌지 않았습니다.
