# 과제서 — 2026-10-05 (base main@92ff88c, VERSION 1.8.7)

- 과제: 추적 설정의 `allowed_hosts` 가 검증 없이 CSP 헤더에 그대로 이어붙어 `object-src 'none'` 을 무력화하는 것을 막는다 (가치 3 / 위험 1 / 작업량 S)

- 왜: `internal/analytics/analytics.go` 의 `Validate()` 는 `AllowedHosts` 를 **전혀 검사하지 않고**, `PolicySources()` 는 `SplitHosts` 가 쪼갠 토큰을 그대로 돌려주며, `internal/api/analytics_handlers.go:68 pagePolicy` 가 그것을 `strings.Join(..., " ")` 로 `img-src`·`script-src`·`connect-src` 세 디렉티브에 붙인다. `SplitHosts`(analytics.go:258)는 `,`·공백·탭·CR·LF 만 구분자로 보므로 **`;` 가 들어간 토큰이 살아남아** 디렉티브를 조기 종료시키고 그 뒤를 새 디렉티브로 만든다. 실제 핸들러로 확인했다(아래 재현): 추적을 켠 모든 페이지의 CSP 디렉티브가 10개 → 13개가 되고, 앞쪽에 값 없는 `object-src` 가 3개 먼저 끼어들어 뒤의 진짜 `object-src 'none'` 이 (CSP 의 중복 디렉티브 규칙상 첫 번째만 유효) 무시된다. 고치면 잘못 입력한 값이 저장 시점에 400 으로 막히고, 이미 저장된 불량 값이라도 정책 구조를 바꾸지 못한다.

## 실제로 확인한 재현 (프로브 테스트, 저장소 변경 없이 실행 후 삭제)
기존 테스트 헬퍼 `servePage`(`internal/api/analytics_handlers_test.go:26` — 실제 `(*Server).serveSPAWith` + 실제 `httptest.ResponseRecorder`, 손으로 만든 대역 없음)와 `momentoConfig(true)` 를 그대로 써서:

```
INPUT "evil.example;object-src" -> Validate ACCEPTED
  PolicySources scripts=["evil.example;object-src"] connects=[…] images=[…]
  CSP = default-src 'self'; img-src 'self' data: evil.example;object-src; style-src 'self' 'unsafe-inline';
        script-src 'self' 'nonce-…' evil.example;object-src; connect-src 'self' evil.example;object-src;
        object-src 'none'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'; report-uri /api/v1/analytics/csp-report
  DIRECTIVES = ["default-src" "img-src" "object-src" "style-src" "script-src" "object-src"
                "connect-src" "object-src" "object-src" "frame-ancestors" "base-uri" "form-action" "report-uri"]
```
같은 프로브에서 정상 입력은 기준선과 **정확히 같은 10개 디렉티브**를 냈다(`cdn.example.com`, `https://cdn.example.com`):
`["default-src" "img-src" "style-src" "script-src" "connect-src" "object-src" "frame-ancestors" "base-uri" "form-action" "report-uri"]`.
`'unsafe-inline'` 도 `Validate ACCEPTED` 로 세 디렉티브에 들어갔다(nonce 가 있으니 script-src 에서는 브라우저가 무시하지만 `img-src`·`connect-src` 에는 무의미한 토큰이 남는다).

- 수용 기준:
  1) **저장 경로** — `allowed_hosts` 에 `;` 를 포함한 토큰이 있으면 `Config.Validate()` 가 오류를 돌려주고, `internal/api/core_handlers.go:825`(`key == analytics.SettingKey` → `analytics.ReadConfig(object).Validate()`)를 타는 설정 저장이 거부된다. **`Validate()` 의 `if !c.Enabled { return nil }`(analytics.go:132) 보다 위**, `Placement`·`Provider`·`CustomSnippet` 검사와 같은 무조건 블록에 넣을 것 — 꺼진 상태로 불량 값을 저장해 두고 나중에 켜는 경로를 막아야 한다.
  2) **렌더 경로** — 같은 불량 값이 이미 DB 에 저장돼 있어도 `PolicySources()` 가 그 토큰을 내놓지 않아, `servePage` 로 받은 `Content-Security-Policy` 를 `;` 로 쪼갠 **디렉티브 이름 목록이 기준선 10개와 같다**. (한쪽만 고치면 안 된다 — 같은 값을 읽는 경로가 둘이다.)
  3) **무변경 보장** — `cdn.example.com`, `https://cdn.example.com`, `*.example.com` 같은 정상 입력은 수정 전과 똑같이 `scripts`·`connects`·`images` 세 목록에 모두 들어가고 헤더에도 그대로 나타난다. 테스트가 이것을 함께 단언해 "전부 걸러서 통과" 가 아님을 증명해야 한다.
  4) 테스트는 손으로 만든 대역이 아니라 `Config.Validate()`/`Config.PolicySources()` 실제 함수와 기존 `servePage` 헬퍼(실제 핸들러·실제 ResponseRecorder)로 증명한다.

- 건드릴 파일 (프로덕션 1개 + 테스트 1~2개):
  - `internal/analytics/analytics.go`
    - `Validate()` (122행) — 무조건 블록에 `allowed_hosts` 토큰 검사 추가. 메시지는 이 파일의 관례대로 한국어(`analytics.allowed_hosts …`).
    - `PolicySources()` (231행) 의 마지막 `for _, host := range SplitHosts(c.AllowedHosts)` 루프 — 불량 토큰은 `add` 하지 않고 건너뛴다. (또는 `SplitHosts` 안에서 걸러도 되지만 `SplitHosts` 는 **exported 이고 다른 호출자가 있는지 먼저 `grep -rn "SplitHosts"` 로 확인할 것** — 나는 확인하지 않았다(미확인).)
  - `internal/analytics/analytics_test.go` 또는 신규 `internal/analytics/allowed_hosts_test.go` — `Validate` 거부 + `PolicySources` 필터 표 기반 테스트.
  - `internal/api/analytics_handlers_test.go` — 렌더 경로 서브테스트 1개(디렉티브 이름 목록 비교). 기존 CSP 단언(45·70·78·98행)은 **한 글자도 약화시키지 말 것**.

- 거부 기준(최소한 이것만 막으면 수용 기준을 만족한다): 토큰에 `;` 가 있으면 거부. 넓히려면 `'` 로 시작하는 키워드 토큰(`'unsafe-inline'` 등)도 함께 거부하는 것이 자연스럽다 — 다만 `'self'` 는 이미 하드코딩돼 있으므로 허용 목록에 넣을 필요가 없다. **`*`·`:`·`/`·`.`·`-` 는 정상 CSP 소스에 들어가므로 절대 거부하지 말 것**(`*.example.com`, `https://cdn.example.com:8443/path` 가 전부 유효하다).

- 검증 명령 (이 저장소에서 실제로 돈다 — 기준선을 내가 돌려 둘 다 ok 를 봤다):
  - `go test -count=1 ./internal/analytics ./internal/api` ← 기준선 `ok analytics 0.003s` / `ok api 0.051s`
  - `gofmt -l .` (무출력) · `go vet ./...` · `go build ./...` · `go test -count=1 ./...`
  - `go test -count=1 -race ./internal/analytics ./internal/api`
  - 선택: `./scripts/check-version.sh` (1.8.7 를 낸다)
  - DB 불필요 — 이 과제는 통합 테스트(`make test-integration`)를 건드리지 않는다.

- 위험과 피할 것:
  - **두 경로를 모두 고칠 것.** `Validate()` 만 고치면 이미 저장된 값이 그대로 헤더를 깨고, `PolicySources()` 만 고치면 관리자가 잘못 입력했다는 것을 영원히 모른다.
  - `pagePolicy`(`internal/api/analytics_handlers.go:58`)의 디렉티브 문자열과 `contentSecurityPolicy`(`internal/api/middleware.go:52`)는 건드리지 말 것 — `middleware_test.go:56`·`analytics_handlers_test.go:70` 이 그 문자열을 고정하고 있다.
  - 보호 경로(`internal/auth`, `migrations/`, `.github/workflows/`)와 무관한 과제다 — 그대로 두면 된다. 릴리즈·빌드 경로도 건드리지 않는다.
  - `openapi/openapi.yaml`·`docs/` 에 `allowed_hosts` 규칙이 적혀 있는지는 **미확인**이다. 설명이 있으면 한 줄 맞춰 주고, 없으면 새로 문서를 쓰지 말 것(파일 수를 늘리지 않는다).
  - `web/src/pages/SettingsPage.tsx` 에 `allowed_hosts` 입력란이 있다(이 파일에 `analytics` 폼이 있다). 프런트 검증은 **이번 범위 밖**이다 — 서버가 400 을 내면 화면은 기존 오류 표시 경로로 보여 준다. web 게이트(`npm --prefix web …`)를 돌릴 필요가 없게 web 트리는 손대지 말 것.

- 차선 후보: `internal/analytics/analytics.go` 의 `withNonce`(207행)·`SnippetOrigins`(265행)가 `strings.ToLower(snippet)` 의 바이트 인덱스를 원본 문자열에 그대로 쓰는 것 — 터키어 대문자 `İ`(U+0130, 2바이트)처럼 ToLower 가 바이트 길이를 늘리는 글자가 `<script` 앞에 있으면 인덱스가 어긋나 nonce 가 엉뚱한 자리에 삽입된다. 코드를 읽어 확인했으나 **실행으로 재현하지는 않았다(미확인)**. 1순위가 성립하지 않으면 먼저 프로브로 재현한 뒤 고칠 것(가치 2 / 위험 1 / 작업량 S).
