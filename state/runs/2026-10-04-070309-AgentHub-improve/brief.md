- 과제: 붙여 넣은 추적 코드에서 읽어 낸 출처의 개수·합계 길이에 상한을 두어, 8KiB 스니펫 하나가 정책 헤더를 리버스 프록시가 거부하는 크기로 부풀리지 못하게 한다 (가치 3 / 위험 2 / 작업량 S)

- 왜: 2026-09-30 회차는 `AllowedHosts` **목록**에 `MaxAllowedHostEntries=64`·`MaxAllowedHostsTotalRunes=4096` 을 두어 3n 증폭된 헤더를 "프록시가 흔히 남겨 두는 8KB 의 두 배 버퍼" 안쪽으로 묶었고, `internal/api/tracking_test.go:297 TestThePagePolicyHeaderIsBoundedByTheAllowListLimit` 이 `carried = 13*1024` 로 그 예산을 고정한다. 그 테스트의 주석은 "허용 출처 목록은 응답 헤더가 되는 **유일한** 관리자 작성 값이다" 라고 적고 있는데 **그것이 사실이 아니다** — `PolicySources()` 의 `ProviderCustom` 분기(tracking.go)는 `SnippetOrigins(s.CustomSnippet)` 가 돌려준 출처를 개수·길이 제한 없이 전부 `everywhere()` 로 img-src·connect-src·script-src 에 넣고, `Validate()` 가 스니펫에 건 유일한 상한은 `MaxSnippetBytes = 8*1024` **바이트** 하나뿐이다. 고치면 "저장할 수 있는 설정은 전송할 수 있는 헤더를 만든다" 가 ProviderCustom 에서도 성립하고, 그 테스트 주석의 주장이 참이 된다.

- 수용 기준:
  1) 스니펫에서 읽어 낸 출처가 너무 많거나 합계가 너무 길면 `tracking.Settings.Validate()` 가 **저장 전에** 한국어 오류로 거절한다 — 두 쓰기 경로가 모두 이 한 곳을 지난다(확인: `internal/api/routes.go:1728` 설정 PUT 의 `settings.Validate()`, `internal/api/tracking.go:292` 한 번 클릭 허용의 `settings.Validate()`). 기존 `MaxSnippetBytes` 검사와 그 오류 메시지는 그대로 둔다.
  2) 상한 안쪽의 스니펫은 지금과 똑같이 저장된다. `internal/tracking/tracking_test.go` 가 "받아들여진다" 고 고정해 둔 가장 큰 ProviderCustom 스니펫은 `TestOriginsAreReadOutOfAPastedSnippet`(89-104행)의 **출처 3개 / 가장 긴 것 33룬** 뿐이므로(확인함), 32개·1024룬 수준의 상한이면 기존 테스트는 전부 통과한다 — 2026-09-30 회차처럼 기존 테스트와 충돌해 숫자를 다시 고르는 일은 없을 것으로 보이지만, 숫자를 정한 뒤 `go test ./internal/tracking ./internal/api` 로 확인할 것.
  3) 테스트가 증명해야 하는 것:
     (a) `internal/api/tracking_test.go` 에 `TestThePagePolicyHeaderIsBoundedByTheAllowListLimit` 의 형제 테스트를 두어, **짧은 출처를 가득 채운 8KiB 스니펫**(예: `http://a1.b http://a2.b …`, 구분자는 공백 — `isURLBoundary` 가 공백을 경계로 센다)이 상한을 켜기 **전에는** `Validate()` 를 통과하고 그 설정으로 **실제 `pagePolicy()`** 가 만든 헤더가 `carried` 를 크게 넘는다는 것을 바이트 수로 찍는다. 켠 뒤에는 `Validate()` 가 거절한다. (`pagePolicy` 는 `internal/api` 의 비공개 함수이므로 이 측정 테스트는 반드시 `internal/api` 쪽에 있어야 한다 — 확인함.)
     (b) 출처 **하나**가 아주 긴 경우도 막힌다: `http://` + 8000 자 호스트(보더 문자 없음) 는 출처 1개지만 헤더에 3×8000 룬을 넣는다. **개수 상한만으로는 이것이 막히지 않으므로 합계 길이 상한이 반드시 필요하다.** 둘 중 하나만 넣지 말 것.
     (c) 상한을 한 개/한 룬 넘긴 스니펫이 거절되고, 한 개/한 룬 안쪽인 것은 통과한다(경계 양쪽).
     (d) 상한을 켠 뒤 "가장 큰 허용 스니펫 + 가장 큰 허용 목록" 을 **함께** 담은 설정이 만드는 헤더가 테스트가 선언한 예산 안에 든다 — 두 예산이 더해진다는 사실을 테스트가 말해야 한다(아래 위험 항목 참조).

- 건드릴 파일 (프로덕션 1 개):
  - `internal/tracking/tracking.go`
    - `MaxAllowedHostEntries`/`MaxAllowedHostsTotalRunes` 상수 블록 **옆에** 새 상수 두 개(예: `MaxSnippetOriginEntries`, `MaxSnippetOriginsTotalRunes`)를 더한다. 숫자를 고른 이유(3n 증폭, 8KB 프록시 예산과 `carried` 와의 관계, 실제 로더가 명명하는 출처는 2~5개)를 `MaxAllowedHostsTotalRunes` 주석과 같은 산문 밀도로 적는다.
    - `Validate()`: `len(s.CustomSnippet) > MaxSnippetBytes` 검사(166행 부근) **직후**, `AllowedHosts` 루프와 섞지 않고 `SnippetOrigins(s.CustomSnippet)` 의 개수·합계 룬 검사를 넣는다. `MaxSnippetBytes` 와 같은 자리에 두는 것이 중요하다 — 자세한 이유는 위험 항목.
  - `internal/api/tracking_test.go`: 수용 기준 3(a)(d) 의 헤더 측정 테스트.
  - `internal/tracking/tracking_test.go`: 수용 기준 3(b)(c) 의 경계 테스트.
  - 건드리지 말 것: `SnippetOrigins`, `PolicySources`, `pagePolicy`, `Sources.all`, `splitHosts`, `AddAllowedHost`, `SingleHost`, `Snippet`, `isURLBoundary`, `MaxSnippetBytes` 의 값. 헤더를 만드는 쪽에서 자르면 저장값과 헤더가 갈라진다(2026-09-30 회차가 같은 이유로 `pagePolicy` 를 건드리지 않았다).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test ./internal/tracking ./internal/api` — DSN 없이 통과한다(2026-10-04 이 환경에서 실행: tracking cached ok, api 1.748s ok).
  - `go build ./...` · `go vet ./internal/tracking ./internal/api`
  - `go test -race -p 1 ./cmd/... ./internal/...` — DSN 이 없으면 live 테스트는 `t.Skip` 한다. live 까지 돌리려면 `docker run --rm -d -p 55447:5432 postgres:16-alpine` 로 일회용 DB 를 띄우고 `AGENTHUB_TEST_DSN` 과 base64 32바이트 `AGENTHUB_ENCRYPTION_KEY` 를 준다(2026-09-27·09-30 선례). 이번 과제는 순수 검증 로직이라 live 가 필수는 아니지만, 설정 PUT 400 을 실제 라우터로 보고 싶으면 `internal/api` 의 live 테스트에 서브테스트를 더하는 길이 있다.
  - BASE_VERSION: `runtime-images.json` 의 어떤 이미지도 `internal/tracking`·`internal/api` 를 `sourcePaths` 에 두지 않으므로 상향 불필요 — 매니페스트를 직접 읽어 확인할 것(`scripts/release-catalog-images.sh` 실행은 과거 회차에 Bash 승인에서 거부된 적이 있다).

- 위험과 피할 것:
  - **먼저 실패를 재현할 것.** 상수만 넣고 배선하지 않은 상태에서 수용 기준 3(a) 테스트가 실제 헤더 바이트 수를 찍게 만든 뒤 검사를 켜라. 이 과제서가 적은 "24KB 쯤" 은 **산술 추정이고 측정값이 아니다**(`http://a1.b` 11바이트 + 공백 1 → 8192/12 ≈ 680 개 × (11룬×3 + 공백 3) ≈ 24.5KB). 실제 숫자는 구현자가 재서 커밋 메시지에 적을 것.
  - **두 예산은 더해진다.** 기존 테스트의 `carried = 13*1024` 는 허용 목록만으로 거의 다 찬다(3×(4096+64) ≈ 12.5KB + 기본 정책). 여기에 스니펫 출처 예산을 더하면 최악의 경우가 `carried` 를 넘으므로, 구현자는 (1) 스니펫 예산을 작게 잡고 `carried` 를 그 합계로 올리며 왜 그 숫자인지 주석에 적거나, (2) 새 테스트에 자기 예산 상수를 두거나 중 하나를 **의식적으로** 골라야 한다. 권장: 1024룬/32개 수준으로 작게 잡고 `carried` 를 함께 올려 한 곳에서 전체 예산을 말하게 하는 (1). 어느 쪽이든 기존 테스트의 "유일한 관리자 작성 값" 주석을 사실에 맞게 고칠 것.
  - **새 검사를 `switch s.Provider` 안쪽에 넣지 말 것.** 기존 `MaxSnippetBytes` 검사는 provider·`Enabled` 와 무관하게 위쪽에서 돌고, `tracking_test.go:126` 이 `Settings{Provider: ProviderNone, CustomSnippet: …}` 로 그것을 고정하며 주석이 이유를 적어 두었다("a stored page-sized blob is a page-sized blob"). 같은 이유가 그대로 적용된다 — custom 이 아닌 동안 큰 스니펫이 저장되면 나중에 provider 만 바꾸는 쓰기가 거절된다.
  - **한 번 클릭 허용이 함께 막힌다.** `internal/api/tracking.go:292` 의 allow 핸들러는 저장된 설정 전체에 `Validate()` 를 돌리므로, 상한을 넘는 스니펫이 **이미 저장되어 있는** 배포에서는 출처 하나를 클릭하는 요청이 스니펫 때문에 400 이 된다. 허용 목록 상한에서도 같은 성질이고(목록이 64 면 다음 클릭이 막힌다) 오류 메시지가 스니펫을 가리키므로 진단은 가능하다 — 그러나 오류 메시지가 "무엇을 줄여야 하는지" 를 말하게 하고, 커밋 메시지에서 이 성질을 밝힐 것. 메시지 형식은 기존 세 오류("…는 %d자를 넘을 수 없습니다 (%d자)")를 그대로 따를 것.
  - 이 저장소의 관례는 **정규화가 아니라 거절**이다(관리자가 읽어 낸 값은 그들이 넣은 값이어야 한다). 스니펫을 잘라 저장하지 말 것. 상한은 룬 단위로 세고(주소를 한국어로 쓸 수 있다), 상수 주석에 숫자를 고른 이유를 적을 것.
  - 보호 경로를 건드리지 않는다: `internal/api/auth.go`·`mcpoauth.go`, 세션 게이트웨이, `internal/store` 마이그레이션, `.github/workflows`, 라우팅 등록 — 이 과제는 그중 어느 것도 필요하지 않다.
  - 미확인: 콘솔 설정 화면(`web/src/pages/AdminSettings.tsx`)이 이 새 오류를 어떻게 보여 주는지는 이번 정찰에서 열어 보지 않았다. 프런트는 범위 밖으로 둘 것(그 안내는 별도 보류 아이디어다).

- 차선 후보: 복원 실패가 guide-shots 의 problems 요약 출력을 건너뛰게 하는 것을 고친다 (2/1/S) — `web/scripts/guide-shots.mjs:128` 의 `await withGuideSettings(...)` 가 던지면 129-136 행의 problems 요약과 `process.exitCode = 1` 이 건너뛰어져, 수십 장을 찍는 동안 쌓인 실패 목록이 복원 오류 한 줄에 묻힌다(2026-10-04 재확인, 여전히 열려 있다). 다만 그 요약 블록은 `guide-settings-check.test.mjs` 의 vm 슬라이스 앵커(`  if (problems.length)`) **밖**이라 기존 하니스로 바로 덮이지 않는다 — 앵커를 옮기거나 블록을 함수로 뽑아야 하고, 그 테스트는 CI 의 `test:sso` 가 부르지 않으므로 검증이 로컬에만 남는다. 1순위가 성립하지 않을 때만 고를 것.
