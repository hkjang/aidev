- 과제: CSP 리포트의 `page` 512바이트 절단이 다중 바이트 문자를 쪼개 관리자 목록에 U+FFFD 를 남기는 문제 수정 (가치 2 / 위험 1 / 작업량 S)

- 왜: `Recorder.Record`(internal/tracking/violations.go:56-59)가 브라우저가 보낸 `document-uri` 를 `page = page[:512]` 로 **바이트** 단위로 자르기 때문에, 512번째 바이트가 한 rune 중간에 떨어지면 저장된 `Page` 가 잘못된 UTF-8 이 되고, 관리자가 `GET /api/v1/tracking/violations` 로 받는 JSON 에서 `encoding/json` 이 그 깨진 바이트를 `�` 로 바꿔 보낸다(internal/httpapi/tracking.go:166 `listTrackingViolations` → `writeData` → `json.Marshal`). 스니펫을 고치려고 차단된 페이지 주소를 보는 운영자가 한국어·비ASCII 경로를 가진 긴 URL 에서 마지막 글자가 깨진 주소를 받게 되므로, rune 경계에서 자르도록 고치면 512바이트 상한을 유지하면서 항상 유효한 UTF-8 주소가 보인다.

- 수용 기준:
  1) `document-uri` 가 512바이트를 넘고 512번째 바이트가 rune 중간인 CSP 리포트를 실제 `POST /api/v1/tracking/report` 로 보낸 뒤, 인증된 `GET /api/v1/tracking/violations` 응답의 `page` 가 `�` 를 포함하지 않고 `utf8.ValidString` 을 통과한다.
  2) 그 `page` 는 원본의 앞쪽을 **온전한 rune 까지만** 담고 있고 길이가 512바이트 이하다(아래 재현값에서는 `"https://example.com/x" + strings.Repeat("가", 163)` = 510바이트와 정확히 같아야 한다).
  3) 기존 계약이 그대로다 — 512바이트 이하 `document-uri`(ASCII·비ASCII 모두)는 **한 바이트도** 바뀌지 않고 그대로 보이고, 512바이트 경계가 마침 rune 경계와 맞는 입력은 512바이트가 그대로 남는다. CSP 리포트 응답은 여전히 204 + 빈 본문이고, 기록되는 violation 개수는 1건이다.
  4) 수정 줄만 되돌리면 1)·2) 테스트가 같은 메시지로 다시 실패하는 것을 확인한다(기존 계약 테스트는 처음부터 통과해야 한다).

- 건드릴 파일 (프로덕션 1개 + 테스트 1~2개):
  - `internal/tracking/violations.go:Record` — `if len(page) > 512 { page = page[:512] }` 를 "512바이트로 자른 뒤 꼬리의 **불완전한** UTF-8 시퀀스만 떼어낸다"로 바꾼다. 권장 형태:
    ```go
    if len(page) > 512 {
        page = page[:512]
        for len(page) > 0 {
            if r, size := utf8.DecodeLastRuneInString(page); r == utf8.RuneError && size <= 1 {
                page = page[:len(page)-1]
                continue
            }
            break
        }
    }
    ```
    `size <= 1` 조건이 핵심이다 — URL 안에 **진짜** U+FFFD 문자가 들어 있으면 `DecodeLastRuneInString` 이 `size == 3` 으로 돌려주므로 그것은 떼어내지 않는다. `strings.ToValidUTF8` 은 쓰지 말 것(떼어내는 대신 치환 문자를 **넣는다**).
  - 같은 파일 import 블록 — `unicode/utf8` 을 **추가해야 한다**. 현재 import 는 `net/url`, `sort`, `strings`, `sync`, `time` 뿐이고 `unicode/utf8` 은 없다(직접 확인).
  - `internal/httpapi/tracking_test.go` — 왕복 회귀 테스트 추가. `TestCSPReportBodySizeBoundary`(:311)가 가장 가까운 본보기다: `trackingServer(t, <추적 켠 config>, nil)` 로 실제 `routes` 를 세우고 `POST /api/v1/tracking/report` 한 뒤 `tokenRequest(http.MethodGet, "/api/v1/tracking/violations")` 로 목록을 받아 단정한다. 추적이 꺼져 있으면 `ReportingActive()` 가 false 라 리포트가 **기록되지 않는다**(tracking.go:146) — `TestPolicyReportsAreRecordedOnceAndListedForAdministrators`(:200)가 쓰는 켜진 config 를 그대로 따라갈 것.
  - (선택) `internal/tracking/tracking_test.go` — `Record`/`List` 를 직접 부르는 경계 표(512 미만/정확히 512/초과·rune 경계 걸림/U+FFFD 포함 URL)를 두면 읽기 쉽다. **단, 이것만으로 끝내지 말 것** — 과거 교훈이 "중간 값만 검사하지 말고 최종 HTTP 응답까지 확인하라"다.

- 재현값 (그대로 쓰면 된다):
  - `"https://example.com/x"` 는 21바이트, `"가"` 는 3바이트. `512 - 21 = 491 = 3*163 + 2` 이므로 `document-uri = "https://example.com/x" + strings.Repeat("가", 200)`(총 621바이트)을 보내면 512번째 바이트가 164번째 `"가"` 의 **2바이트째**에 떨어져 쪼개진다.
  - 수정 전 기대 실패: `page` 가 `"...가��"` 로 끝난다(Go 의 JSON 인코더는 잘못된 바이트마다 U+FFFD 를 하나 낸다). 수정 후 기대값: `"https://example.com/x" + strings.Repeat("가", 163)` (510바이트).
  - 리포트 본문은 `maxCSPReportBytes`(8192) 안에 있어야 한다 — 621바이트 URI 는 여유가 충분하다. 본문 형태는 `TestPolicyReportsAreRecordedOnceAndListedForAdministrators` 의 `{"csp-report":{...,"document-uri":"…"}}` 를 따를 것.
  - 참고: 입력이 `json.Unmarshal` 을 거치므로 저장 **전** 문자열은 항상 유효한 UTF-8 이다(잘못된 입력 바이트는 Unmarshal 이 이미 U+FFFD 로 바꾼다). 즉 깨짐의 원인은 오직 이 절단 한 곳이다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test ./internal/httpapi/ -run 'CSPReport|PolicyReports|Tracking' -count=1 -v`
  - `go test ./internal/tracking/ -count=1 -v`
  - `go test ./... -count=1`
  - `go vet ./...` 와 `gofmt -l internal/tracking/violations.go internal/httpapi/tracking_test.go`(출력 없어야 함)
  - `bash scripts/verify.sh` (시간 소요. 프런트 59/59 와 `검증 완료: jikim v0.2.35` 가 기준)
  - `make release-check` 는 돌리지 말 것(오래 걸리고 스크린샷을 덮어쓴다).

- 위험과 피할 것:
  - **512 라는 상한 자체를 바꾸지 말 것.** 상한을 늘리거나 "rune 512개"로 바꾸면 무인증 엔드포인트의 메모리 예산 계약이 바뀐다. 바이트 상한은 512 그대로 두고 **꼬리만** 떼어낸다.
  - `MaxViolations`(100)·`evictOldest`·`originOf`·`matchesWildcard`·`List` 의 정렬·`Allowed` 판정은 건드리지 말 것. 같은 파일 안이라 손이 가기 쉽다.
  - `internal/tracking/tracking.go`(스니펫·정책 생성)는 **열지 말 것**. 추적이 꺼진 동안의 CSP 헤더는 `shippedPolicy` 와 바이트 단위로 같아야 하는 계약이 있다.
  - CSP 리포트 응답은 어떤 경우에도 204 + 빈 본문이어야 한다(`receiveCSPReport` 의 `defer w.WriteHeader(http.StatusNoContent)`). 거기에 손대지 말 것.
  - 기존 `TestCSPReportBodySizeBoundary`·`TestPolicyReportsAreRecordedOnceAndListedForAdministrators`·`TestMomentoProxy*` 는 수정하지 않고 통과해야 한다.
  - 보호 경로(auth_handlers.go·oidc*.go·mcp_oauth.go·store/users.go·migrations/·.github/workflows/)는 이 과제와 무관하다 — 열지 말 것.
  - 버전·CHANGELOG 변경은 이 과제 범위가 아니다(릴리즈 단계가 따로 한다).
  - 미확인: 프런트(`web/src`)가 `page` 를 어떻게 렌더링하는지는 열어 보지 않았다. 프런트는 건드리지 않는 것을 전제로 한다.

- 차선 후보: **Transit ciphertext 의 버전 토큰이 비정규 표기를 받아들이는 문제 수정** (가치 3 / 위험 1 / 작업량 M)
  - 근거(직접 확인): `Store.TransitDecrypt`(internal/store/transit.go:47)가 `strconv.Atoi(strings.TrimPrefix(parts[1], "v"))` 로 버전을 읽는다. Go 의 `Atoi` 는 부호와 선행 0 을 허용하므로 서버가 절대 발급하지 않는 `vault:v+1:<payload>`·`vault:v01:<payload>` 가 `vault:v1:<payload>` 와 똑같이 200 + 평문으로 복호화된다(AAD 를 파싱된 **정수**로 다시 조립하므로 AEAD 인증도 통과한다). OpenBao 는 이런 토큰을 거절하므로 호환성 표면이 느슨하고, 같은 평문에 여러 ciphertext 표기가 성립한다. 고칠 자리는 숫자만 받는 엄격한 파싱으로 바꾸는 한 곳이다.
  - `internal/httpapi/openbao.go:842` 의 쌍둥이 함수 `transitCiphertextVersion` 은 호출처가 678·709 두 곳뿐이고 **둘 다 서버가 막 만든 ciphertext** 에만 쓴다(직접 확인) — 클라이언트 입력이 닿지 않으므로 범위에서 빼도 된다.
  - **이것을 고르면 비용이 더 든다**: `Store` 에 함수 seam 이 없고(`internal/store/store.go:31` 은 `pool`·`master` 뿐) 버전 파싱 **뒤**에 바로 DB 를 타므로, 최종 HTTP 응답(200→400) 차이를 보려면 `JIKIM_TEST_POSTGRES_DSN` 을 세운 통합 테스트가 필요하다. 전례는 있다 — `internal/store/secrets_openbao_test.go` 와 `internal/httpapi/response_test.go` 가 같은 DSN 게이트를 쓰고, 2026-10-08 회차가 임시 PostgreSQL 17 로 통합 테스트를 돌렸다. PostgreSQL 을 세울 수 없으면 1순위로 돌아갈 것(순수 파싱 단위 테스트만 남기면 "중간 값만 검사" 교훈에 걸린다).
