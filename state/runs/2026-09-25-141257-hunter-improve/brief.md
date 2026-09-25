# 과제서 (2026-09-25 정찰)

- 과제: PR #10 / v1.15.0 태그에서 실패한 Verify·Release 워크플로의 실제 실패 단계를 확인하고, `web/src/tracking-state.ts` 의 **런타임 의존 URL 파서 경로**를 제거해 Node 26 CI 에서도 같은 값이 나오게 고치기 (가치 5 / 위험 2 / 작업량 M)
- 왜: main@4ae4034 이 v1.15.0 을 태그했는데 CI 와 릴리즈 워크플로가 실패했고, 릴리즈 경로 반복 파손은 운영자가 명시적으로 금지한 유형이다. PR #10 이 넣은 TS 원점 정규화는 `new URL()` 의 IDN 처리에 의존하는데 그 동작이 Node 빌드(ICU 포함 여부)·버전마다 다르고 **정찰 로컬은 Node 22, CI 는 Node 26** 이라 로컬 통과가 CI 통과를 뜻하지 않는다.

## 먼저 할 일 (착수 전 5분, 건너뛰지 말 것)

이 정찰 세션에서는 `gh` 실행 승인이 거부되어 **실제 Actions 로그를 보지 못했다**. 아래는 미확인이며 구현자가 제일 먼저 확정해야 한다.

```sh
gh run list --limit 15
gh run list --workflow=ci.yml --limit 5
gh run list --workflow=release.yml --limit 5
gh run view <실패한 run id> --log-failed | tail -120
```

실패 단계가 아래 "확인 결과 제외된 것" 중 하나로 드러나면 그쪽을 고치고, 이 과제서의 1순위는 무시해도 된다.

## 확인 결과 제외된 것 (이 세션에서 실제로 확인함)

- `go test -run TestTrackingOriginVectors ./internal/app` → **ok 0.024s**(DB 불필요, 캐시 아님). 공유 벡터 105개와 Go `trackingOrigin`(internal/app/tracking.go:78) 는 일치한다. e503f24 가 Go 테스트 파일을 안 건드리고 JSON 에 벡터 3개(`undecodable-punycode`, `empty-punycode`, `ascii-only-punycode`)만 추가한 것이 Go 쪽을 깨뜨렸을 것이라는 가설은 **틀렸다**.
- `VERSION`=1.15.0, `web/package.json`=1.15.0, `docs/package.json`=1.15.0 일치 → `release.yml` 의 `Validate tag` 와 `scripts/release.sh:12` 의 VERSION 일치 검사는 통과한다.
- `docs/release-v1.15.0.md` 존재 → 릴리즈 본문 링크 대상 있음.
- `scripts/release-notes.py` 에 `version >= (1,15,0)` 블록 존재, 아카이브 이름 검사(`hunter-v1.15.0.tar.gz`)가 `scripts/release.sh:17` 의 생성 이름과 일치.
- `internal/webassets/dist` 는 `.gitkeep` 만 커밋되어 있고 CI 가 직접 빌드·복사 → "임베드 자산 미갱신" 은 원인이 아니다.
- `Dockerfile` 은 web·upstream·build 3단계 + debian-slim, `.dockerignore` 가 `docs`·`node_modules`·`dist` 를 제외 → 빌드 컨텍스트 비대는 아니다.

## 1순위 가설과 근거

`web/src/tracking-state.ts` 는 직접 쓴 RFC 3492 디코더를 넣어 놓고도 **정규화의 마지막 단계를 여전히 브라우저/Node 의 `new URL()` 에 맡긴다**:

- `web/src/tracking-state.ts:221` — `const inner = new URL("http://" + host + "/").hostname;`
- `web/src/tracking-state.ts:258` — `ascii = new URL("http://" + unicode + "/").hostname;`

그리고 같은 파일 151~153행 주석이 이 의존을 스스로 적어 두었다:
> `URL 파서들은 이미 ASCII 인 xn-- 라벨에서 서로 다르게 동작한다 — Node 26 은 "xn--a.internal" 을 그대로 돌려주고 ICU 빌드는 거절한다`

즉 `web/tests/tracking-state.test.mjs` 의 xn-- 벡터 결과가 **실행 런타임에 따라 달라진다**. 구현자의 로컬(Node 22)에서 통과한 것이 CI(`.github/workflows/ci.yml:28` `node-version: '26'`)에서 통과한다는 보장이 없다. 이것이 운영자가 되풀이해 말한 "같은 값을 읽는 파서가 둘 이상이면 두 경로가 같은 입력을 같은 값으로 읽는지 end-to-end 로 확인하라" 에 정확히 걸리는 자리다.

## 수용 기준

1. 실제 실패한 워크플로 run 의 실패 단계와 로그 마지막 줄을 회차 노트에 인용한다(추측 금지).
2. `web/src/tracking-state.ts` 의 호스트 정규화가 `new URL().hostname` 의 **IDN 변환 결과에 의존하지 않는다**: xn-- 라벨 처리와 유니코드→ASCII 변환이 파일 안의 결정적 코드로만 끝나거나, URL 파서 결과를 쓰더라도 그 결과가 Go 규칙(`internal/app/tracking.go:78` 의 라벨 63바이트·호스트 253바이트·`[a-z0-9-]` 전용·선두/말미 `-` 금지·숫자/0x 최종 라벨 금지)으로 **다시 검사**되어 런타임 차이가 최종 판정에 새지 않는다.
3. `internal/app/testdata/tracking-origins.json` 의 원점 벡터가 Go 테스트와 web 테스트에서 **둘 다** 통과한다. 벡터를 지우거나 기대값을 느슨하게 바꿔 통과시키지 않는다(워크플로·테스트 완화 금지).
4. CI 와 같은 Node 메이저(26)에서 web 테스트가 통과하는 것을 확인한다. 로컬 Node 가 22 라면 `docker run --rm -v "$PWD":/src -w /src/web node:26-bookworm-slim sh -c 'npm ci && npm test'` 로 CI 런타임을 재현한다(Dockerfile 이 쓰는 것과 같은 이미지 계열).
5. 수정 후 태그 재푸시 없이 `.github/workflows/ci.yml` 의 전 단계를 로컬에서 재현해 통과를 확인한다.

## 건드릴 파일

- `web/src/tracking-state.ts` — 149~260행 부근 `decodePunycodeLabel`(이름 미확인, 149행 주석 아래) / 211~260행의 호스트 정규화. `new URL()` 의존 제거 또는 결과 재검사 추가.
- `web/tests/tracking-state.test.mjs` — 런타임 차이를 드러내는 회귀(문제의 xn-- 입력이 Node 26 에서도 같은 값)를 **먼저 실패시켜** 재현.
- `internal/app/testdata/tracking-origins.json` — 원칙적으로 **변경 금지**. 새 사례가 꼭 필요하면 Go 테스트를 함께 돌려 양쪽 통과를 확인할 것.
- `internal/app/tracking.go` — **건드리지 말 것.** 서버가 기준이고 이 세션에서 벡터 통과를 확인했다.

## 검증 명령 (이 저장소에서 실제로 도는 것)

```sh
go test -race -count=1 -run 'TestTrackingOriginVectors|TestTrackingValidation|TestTrackingOriginHeaderBudget' ./internal/app   # DB 불필요
npm --prefix web ci && npm --prefix web test && npm --prefix web run build
mkdir -p internal/webassets/dist && cp -a web/dist/. internal/webassets/dist/
node scripts/verify-pentagi.mjs
go vet ./... && go build ./cmd/hunter
node scripts/check-docs.mjs
bash -n scripts/release.sh && python3 -m py_compile scripts/release-notes.py
docker build --build-arg VERSION="$(cat VERSION)" --tag hunter:ci .
```

`go test -race ./...` 전체는 `HUNTER_TEST_DSN` 과 수백 초가 필요하다(프로필 기준 최대 650초). 실패 단계가 Go 전체 스위트로 확인되면 그때만 돌리고 `-timeout 45m` 을 붙인다. `HUNTER_TEST_DSN` 이 없으면 testApp 기반 테스트는 **skip** 되며 skip 을 통과로 보고하지 말 것.

## 위험과 피할 것

- **워크플로 파일을 느슨하게 만들어 통과시키는 것은 금지.** `ci.yml`·`release.yml` 의 단계·타임아웃·버전 핀을 바꿔서 초록을 만들지 말 것.
- `internal/app/tracking.go` 의 서버 파서를 TS 에 맞추지 말 것. 지난 회차에서 "서버가 기준" 으로 결정했고 벡터가 그것을 고정한다.
- `third_party/pentagi` 312파일에 닿지 말 것. `gofmt -w .` 금지, 변경한 Hunter Go 파일만.
- grep 으로 문자열이 있다는 것을 증거로 제출하지 말 것 — 실제 Node 26 실행 출력으로 증명할 것.
- 릴리즈 워크플로가 별개 이유(에셋 업로드·`gh release create --verify-tag`)로 실패했다면 그것은 2026-09-19 회차에서도 나온 반복 실패다. 로그로 확정한 뒤에만 손댈 것.

## 차선 후보

`oidcReturnTo`(Go, `internal/app/auth_oidc*.go`)와 `safeReturnPath`(TS, `web/src/auth-flow.ts`)의 `return_to` 규칙을 공유 JSON 벡터로 교차 검증 (3 / 2 / M). 단 auth 보호 경로이므로 의도된 정규화 차이를 먼저 문서로 정의하고 착수할 것. 릴리즈 실패가 이미 워크플로 로그로 해결됐을 때만 고른다.
