# 과제서

- 과제: CSP 위반의 "허용" 원클릭 버튼이 서버가 거절할 원점을 제안하는 문제 — TS `trackingViolationOrigin`/`validateTrackingDraft`를 Go `trackingOrigin`과 같은 계약으로 맞추고 공유 JSON 벡터로 교차 검증 (가치 3 / 위험 1 / 작업량 M)

## 왜

추적 미리보기 화면(`web/src/tracking.tsx:667`)은 **브라우저에서 릴레이된** CSP 위반의 `blocked_uri`를 TS `trackingViolationOrigin`(브라우저 `URL.origin`만 사용)으로 정규화해 "허용" 버튼을 띄우고, 누르면 `addOrigin(origin)`으로 허용 주소 입력란에 넣는다. 그 값을 실제로 판정하는 것은 서버 `trackingOrigin`(`internal/app/tracking.go:78`)인데 이쪽이 훨씬 엄격하다 — 마지막 라벨이 전부 숫자이거나 `0x`로 시작하면 거절(레거시 IPv4 별칭 차단), 호스트에 `%` 포함 거절, 원문 300바이트 초과 거절, 포트 범위 검사. 그래서 관리자가 화면이 권하는 버튼을 눌러 저장하면 PUT이 400으로 떨어진다. 두 파서를 한 벡터로 묶으면 서버가 못 받을 원점은 버튼 자체가 뜨지 않고, 입력란에서도 저장 전에 같은 이유로 걸린다.

## 확인한 사실 (이번 정찰이 실제로 연 것)

- `internal/app/tracking.go:78 trackingOrigin` — **순수 함수**(DB·App 리시버 없음). 따라서 Go 쪽 벡터 테스트는 `HUNTER_TEST_DSN` 없이 돈다. 이게 이 과제의 위험을 낮추는 핵심이다.
  - 거절 조건: `len(s) > 300`, scheme이 http/https 아님, `u.Hostname() == ""`, `u.User != nil`, RawQuery/ForceQuery/Fragment/Opaque 있음, `u.Path`가 `""`·`"/"` 아님, `s`에 `*;,'"\` 공백 탭 CR LF 포함, `u.Host`에 `%` 포함.
  - 호스트: IP면 `net.ParseIP` 정규화(IPv6는 `[...]`), 아니면 `idna.Lookup.ToASCII` → 253바이트·라벨 63바이트·하이픈 시작/끝·`[a-z0-9-]`만 검사 → **마지막 라벨이 `strconv.ParseUint` 성공하거나 `0x` 접두면 거절**.
  - 포트: 기본 포트(80/http, 443/https)는 제거, 나머지는 `:n`으로 붙임.
- `internal/app/tracking_violations.go:45 trackingViolationOrigin`(Go) — `u.Scheme+"://"+u.Host`를 위 `trackingOrigin`에 넘겨 실패하면 `""`를 돌려 **기록 자체를 버린다**(`record()`가 false). 즉 서버에 저장된 위반 목록에는 이런 원점이 없지만, 화면의 `previewViolations`는 서버 저장본이 아니라 iframe이 postMessage로 릴레이한 클라이언트 이벤트라서 이 필터를 거치지 않는다 — 여기가 불일치가 드러나는 지점이다.
- `web/src/tracking-state.ts:138 trackingViolationOrigin`(TS) — 검사는 `["http:","https:"].includes(url.protocol) && url.host`뿐이고 `url.origin`을 그대로 반환. 숫자 끝 라벨·`%`·300바이트·포트 범위 검사가 전혀 없다.
- `web/src/tracking-state.ts:149 validateTrackingDraft` — 각 origin에 대해 300바이트, `new URL` 파싱, protocol/username/password/`pathname !== "/"`/search/hash/`/[?*#;,"'\\\s]/`/앱 원점 동일을 검사. **숫자 끝 라벨·`%` 호스트 검사 없음.** 또 전체 길이를 `allowed_origins.join(" ")` 바이트로 재는 데 반해 Go `validateTracking`은 **정규화·중복 제거 후** `len(o)+1`의 누적으로 재므로(`internal/app/tracking.go:134` 부근) 중복 입력과 경계값에서 판정이 갈린다.
- 이번 정찰에서 브라우저 `URL`의 실제 출력값(예: `http://0177.0.0.1`이 `127.0.0.1`로 접히는지)은 **런타임으로 확인하지 못했다(미확인)**. 구현자는 벡터를 쓰기 전에 `node -e`로 각 후보 입력의 `new URL(s).host`/`.origin`을 먼저 찍어 보고, 브라우저가 이미 정규화해 Go가 받아들이는 입력은 벡터에서 "차이 없음"으로 분류할 것.

## 수용 기준

1. Go `trackingOrigin`이 거절하는 원점에 대해 미리보기 위반 목록의 "허용" 버튼이 **뜨지 않는다**(`origin`이 null이 되어 `item.blocked_uri` 원문만 텍스트로 보임). 최소한 마지막 라벨이 전부 숫자이거나 `0x` 접두인 호스트, 호스트에 `%`가 있는 경우, 300바이트 초과.
2. 같은 규칙이 `validateTrackingDraft`에도 들어가, 손으로 입력한 경우에도 저장 PUT 400 대신 입력 단계에서 기존 한국어 문구로 걸린다. 전체 길이 판정은 Go와 같이 **정규화·중복 제거 후 `len+1` 누적**으로 맞춘다.
3. 새 공유 JSON 벡터 파일 하나(입력 문자열 → 기대 정규화 결과 또는 거절)를 웹 테스트와 Go 테스트가 **둘 다 읽어** 같은 판정을 내는 것을 증명한다. 벡터에는 적어도: 기본 포트 제거, 비기본 포트 유지, 대문자 호스트, IDN(한글 호스트→punycode), IPv6 리터럴, 숫자 끝 라벨, `0x` 끝 라벨, 호스트 `%`, 자격정보(`user@`), 경로·쿼리·프래그먼트, 와일드카드 `*`, 공백/탭/CR/LF, 300바이트 초과, 중복 원점을 포함한다.
4. 기존 테스트 회귀 없음: `web/tests/tracking-state.test.mjs`의 현재 케이스(특히 `trackingViolationOrigin("HTTPS://Stats.Internal:443/collect?id=1#frag")`, `"http://collector.internal:8443/pixel.gif"`)가 그대로 통과한다.

## 건드릴 파일

- `web/src/tracking-state.ts` — `trackingViolationOrigin`(138행)과 `validateTrackingDraft`(149행). 공통 헬퍼 하나(예: `normalizeTrackingOrigin(raw): string | null`)를 두고 두 곳이 같은 함수를 쓰게 할 것. 새 export를 추가하면 테스트에서 직접 호출 가능.
- `internal/app/tracking.go` — `trackingOrigin`은 **바꾸지 말 것**(서버가 기준이다). 총 길이 산식도 서버 쪽을 기준으로 두고 TS를 맞춘다.
- 공유 벡터: 기존 CSV 회차의 선례를 따라 양쪽에서 읽을 수 있는 위치에 JSON 하나(예: `web/tests/fixtures/tracking-origins.json`, Go에서는 상대 경로로 `os.ReadFile`). 파일 위치는 구현자가 정하되 **두 테스트가 같은 파일을 읽어야** 한다(복사본 두 개 금지).
- `web/tests/tracking-state.test.mjs` — 벡터 루프 추가.
- `internal/app/tracking_test.go` — 벡터 루프 추가. `trackingOrigin`이 순수 함수이므로 `testApp`/DB 없이 별도 `func TestTrackingOriginVectors(t *testing.T)`로 작성해 skip 없이 돌게 할 것.
- `web/src/tracking.tsx`는 가능하면 손대지 않는다(667행은 `origin`이 null이면 이미 버튼을 숨기므로 로직 변경 불필요).

## 검증 명령

```sh
npm --prefix web test                                  # 현재 기준선 93 통과/0 실패/0 skip (2026-09-23 회차 기록; 실행해 기준선 먼저 확인)
npx --prefix web tsc -p web/tsconfig.json --noEmit
go test -race -count=1 -run TestTrackingOrigin ./internal/app   # DB 불필요 (순수 함수)
go vet ./...
git diff --check
```

TS만 바꾸고 Go는 테스트만 추가하는 경우 `internal/webassets/dist` 재복사와 `go build ./cmd/hunter`는 프런트 빌드 후에만 의미가 있다. 프런트를 바꿨으므로 완료 보고 전에 `npm --prefix web run build` → `cp -a web/dist/. internal/webassets/dist/` → `go build ./cmd/hunter`까지 하고, 하지 않았으면 하지 않았다고 적을 것.

## 위험과 피할 것

- **서버 `trackingOrigin`을 느슨하게 고쳐 맞추지 말 것.** 숫자 끝 라벨 거절은 "브라우저가 레거시 IPv4로 해석하는 별칭으로 앱 자체 원점 차단을 우회"하는 것을 막는 보안 주석이 달린 규칙이다(tracking.go 해당 주석). 방향은 항상 TS를 Go에 맞추는 쪽이다.
- 위반 기록 저장 경로(`tracking_violations.go record()`)와 `revision`·낙관적 잠금 계약은 건드리지 말 것. `tracking_concurrency_test.go`가 그 계약에 의존한다.
- 앱 전체 CSP를 완화하지 말 것, `preview` 5분 1회 규칙 유지(AGENTS.md §9).
- auth·migrations·`.github/workflows`·`third_party/pentagi`·`scripts/release.sh` 미접촉.
- **grep 결과를 증거로 내지 말 것.** 벡터는 실제로 두 런타임에서 실행해 값이 일치하는 것을 보여야 한다. 고치기 전에 벡터를 먼저 돌려 **실패하는 케이스를 실제로 확인**하고(TDD), 그 실패 목록을 보고에 적을 것.
- 브라우저 `URL`이 이미 Go와 같게 정규화하는 입력이 많을 수 있다. 그 경우 실제 차이가 나는 케이스가 1~2개로 줄 수 있으니, 먼저 `node -e`로 후보를 한 번에 찍어 **진짜 차이가 나는 입력을 확정한 뒤** 구현 범위를 정할 것. 차이가 전혀 없다고 판명되면 이 과제를 접고 아래 차선으로 갈 것.

## 차선 후보

1. **`validateTrackingDraft`의 전체 길이 산식만 Go와 일치시키기 (가치 2 / 위험 1 / 작업량 S)** — 위 과제에서 정규화 차이가 없다고 판명돼도 이 부분(중복 제거 전 `join(" ")` vs 중복 제거 후 `len+1` 누적)은 확실히 남는다. 중복 원점 2개를 입력하면 클라이언트가 서버보다 먼저 막는다.
2. **저장한 목록 보기 주소 복사(`listSharePath`)에 저장소용 500자 잘림을 적용하지 않도록 분리 (가치 3 / 위험 1 / 작업량 S)** — `web/src/saved-list-views.ts`의 `clip`/`savedListQuery`. 정책 판단("주소 복사는 자르지 않는다")이 선행되어야 하며, `web/tests/convenience.test.mjs`의 기존 공유 테스트를 보존해야 한다.
