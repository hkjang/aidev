# igame 자율 개선 기록

## 2026-09-02
- 선택: OIDC 로그인 후 복귀 경로의 open redirect 차단 (가치 5 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `safeReturnTo`(internal/api/api.go)가 절대 URL만 걸러내서 scheme 없는 protocol-relative `//host/path`가 통과했고, `/api/v1/auth/oidc/login?return_to=//evil/...` 링크를 따라간 사용자가 실제 SSO 로그인을 마친 직후 외부 사이트로 튕겨 나갔다(auth.go:335의 `http.Redirect`). host/scheme/opaque/userinfo가 있으면 무조건 `/`로 되돌리도록 고치고 `///host`까지 막았으며, 통과·차단 양쪽을 확인하는 테스트를 api_test.go에 추가했다. 검증: gofmt 검사, `go vet`, `go test`, `go test -race` 전체 통과, `npm --prefix web test` 209개 통과.
- 보류 아이디어: (1) `migrations` 패키지에 파일명 규약·번호 연속성·체크섬 안정성을 지키는 테스트 추가 — 지금은 사전식 정렬에 의존하는데 `10_x.sql` 같은 이름이 들어오면 순서가 깨진다. (2) `internal/database`는 커버리지 0% — `Migrate`의 체크섬 불일치 경로를 검증할 방법 마련. (3) 감사 로그 CSV 검색어의 `%`/`_`가 ILIKE 와일드카드로 새어 들어가는 문제 이스케이프. (4) `clockMinutes`가 `+9:+5` 같은 부호 붙은 값을 시간으로 받아들이는 입력 검증 강화. (5) `cmd/igame` 커버리지 13.7% — 기동/종료 경로 테스트 보강.

## 2026-09-03
- 선택: 검색어의 ILIKE 와일드카드(`%`, `_`) 누출 차단 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: 관리자 사용자/감사 로그 목록, 감사 CSV 내보내기, 게임 카탈로그 네 곳이 검색어를 `'%'||$1||'%'`로 그대로 이어 붙여, `50%`·`user_id`·user agent의 Windows 경로처럼 `%`/`_`가 든 검색어가 와일드카드로 새어 들어가 필터가 적용된 것처럼 보이면서 전체 행을 돌려줬다(`%` 하나면 전부 매치). `searchPattern`(internal/api/api.go)이 백슬래시와 두 와일드카드를 이스케이프해 패턴을 만들고 빈 검색어는 빈 문자열로 남겨 기존 `$1=''` 필터 건너뛰기를 유지하도록 고쳤으며, 카탈로그는 정확 일치인 태그 비교용으로 원문 검색어를 별도 파라미터로 유지했다. 검증: gofmt 검사, `go vet`, `go test`, `go test -race` 전체 통과, `npm --prefix web run lint`과 `npm --prefix web test` 209개 통과.
- 보류 아이디어: (1) `migrations` 패키지에 파일명 규약·번호 연속성 테스트 추가 — 사전식 정렬이라 `10_x.sql`이 들어오면 순서가 깨진다. (2) `internal/database` 커버리지 0% — `Migrate`의 체크섬 불일치 경로 검증. (3) `clockMinutes`가 `09:+5` 같은 부호 붙은 값을 받아들이는 입력 검증 강화. (4) `cmd/igame` 커버리지 13.7% — 기동/종료 경로 테스트 보강. (5) `listUsers`만 검색어를 TrimSpace 하지 않아 다른 목록과 동작이 다른 점 정리.
- 릴리즈: v0.7.2 (2026-09-03)

## 2026-09-03 (2)
- 선택: 스키마 마이그레이션을 이름이 아니라 번호 순서로 적용 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `Migrate`(internal/database/database.go)가 마이그레이션 파일을 `sort.Strings`로 정렬해 적용했는데, 사전식 순서는 번호 자릿수가 모두 같을 때만 번호 순서와 일치한다. 기존 `001_`~`009_` 옆에 `10_x.sql`이 처음 들어오는 순간 그 파일이 두 번째로 정렬돼, 아직 존재하지 않는 테이블을 ALTER 하려 들면서 운영 DB 배포 도중에 깨진다(열 번째 파일이 생겨야 비로소 드러나는 결함). 파싱한 번호로 정렬하도록 고치고, 번호가 없는 이름과 중복된 번호는 파일시스템이 고른 위치에 그냥 적용하는 대신 거부하게 했다(`//go:embed`로 컴파일에 박히므로 저장소의 실수이고 모든 기동이 같은 방식으로 보고해야 한다). 정렬 로직을 `fs.FS`를 받는 `migrationNames`로 분리해 DB 없이 테스트 가능하게 만들고, 커버리지 0%였던 `internal/database`에 열 번째 파일 순서·중복 번호·번호 없는 파일·동봉된 마이그레이션의 1..N 연속성 테스트를 추가했다. 검증: gofmt 검사, `go vet`, `go test`, `go test -race` 전체 통과, `npm --prefix web run lint`과 `npm --prefix web test` 209개 통과.
- 보류 아이디어: (1) `clockMinutes`가 `+9:+5`·`0009:30` 같은 값을 시간으로 받아들이는 입력 검증 강화 — 저장된 비정규 값은 `<input type="time">`에서 빈칸으로 보인다. (2) `playAllowed`에서 start==end인 창이 "하루 종일 허용"으로 해석되는 의미 정리. (3) `cmd/igame` 커버리지 13.7% — 기동/종료 경로 테스트 보강. (4) `listUsers`만 검색어를 TrimSpace 하지 않아 다른 목록과 동작이 다른 점 정리. (5) `Migrate`의 체크섬 불일치 경로는 여전히 DB가 있어야만 검증 가능 — 트랜잭션 경계까지 포함한 통합 테스트 마련.
- 릴리즈: v0.7.3 (2026-09-03)

## 2026-09-03 (3)
- 선택: 플레이 시간 창을 설정 화면이 되돌려 보여줄 수 있는 형태로만 저장 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `clockMinutes`(internal/api/admin.go)가 시/분을 `strconv.Atoi`에 그대로 넘겨서 `+9:+5`·`0009:0005`·`9:00`이 모두 09:05/09:00으로 통과해 저장됐다. 그런데 이 값을 설정하는 화면은 `<input type="time">`이라 HH:MM이 아닌 값은 아예 빈칸으로 그려서, 다음 관리자는 스위치가 켜진 채 허용 시작이 비어 있는 화면을 보고 창이 적용 중인지 알 수 없었다(서버는 09:00으로 사람들을 돌려보내는 중). 이제 `clockMinutes`가 부호·공백·패딩·두 번째 콜론 없이 한두 자리 숫자만 직접 읽고, 저장 검증(`playWindowTime`)은 화면이 보여줄 수 있는 0 패딩 형태만 받는다. 런타임 파싱은 `9:00`을 계속 받아들여 이전에 API로 손수 넣은 정책이 깨지지 않게 했고, 24:00은 창의 끝에서만 그대로 유효하다. 새 파일 `internal/api/admin_test.go`에 파싱 거부 목록·정규형 검증·`validateSetting("play_policy", …)` 테스트를 추가했다. 검증: gofmt 검사, `go vet`, `go test`, `go test -race` 전체 통과, `npm --prefix web run lint`과 `npm --prefix web test` 209개 통과.
- 보류 아이디어: (1) `playAllowed`에서 start==end인 창이 "하루 종일 허용"으로 해석되는 의미 정리 — 화면에서는 길이 0인 창으로 보인다. (2) `cmd/igame` 커버리지 13.7% — 기동/종료 경로 테스트 보강. (3) `listUsers`만 검색어를 TrimSpace 하지 않아 다른 목록과 동작이 다른 점 정리. (4) `Migrate`의 체크섬 불일치 경로는 여전히 DB가 있어야만 검증 가능 — 트랜잭션 경계까지 포함한 통합 테스트 마련. (5) 설정 화면의 플레이 창이 항상 `windows[0]` 하나만 편집·저장해서 API가 지원하는 다중 창을 UI에서 잃어버리는 문제 확인.
- 릴리즈: v0.7.4 (2026-09-03)

## 2026-09-04
- 선택: 설정 화면이 보여주지 않는 플레이 시간대를 지우지 않도록 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `AdminSettingsPage`의 `updateWindow`가 매 편집마다 `windows`를 `[{...window, ...next}]`로 통째로 갈아치워서, 관리자가 요일 하나만 눌러도 API로 넣어 둔 두 번째 이후 시간대가 화면에 아무 말 없이 삭제됐다(서버는 다중 창을 전부 검사한다). 같은 화면이 창이 하나도 없는 정책에도 11:30~13:30을 채워 그려서, 서버는 시간 제한 없이 전원 허용 중인데 화면은 점심시간 규칙이 켜져 있는 것처럼 보였고 그 값은 저장도 되지 않았다(fallback이 값이 아니라 렌더에만 있었다). 첫 창이 나머지를 데리고 가도록 고치고, 빈 정책은 빈 칸으로 그리며 그 상태를 경고로 설명하고, 창이 둘 이상이면 나머지는 유지된다고 화면에 알리고, 한쪽만 채운 시각은 요청 전에 한국어로 막는다. 순수 헬퍼 4개를 `__testing`으로 내보내 `web/src/pages/admin/adminSettings.test.ts`(테스트 10개)를 추가했다. 검증: `go build`, `go vet`, `go test` 전체 통과, `npm --prefix web run lint` 통과, `npm --prefix web test` 219개 통과.
- 보류 아이디어: (1) `playAllowed`에서 start==end인 창이 "하루 종일 허용"으로 해석되는 의미 정리 — 화면에서는 길이 0인 창으로 보인다. (2) 시간대 목록 전체를 편집하는 UI(추가/삭제) — 지금은 첫 창만 편집 가능하고 나머지는 보존만 된다. (3) `cmd/igame` 커버리지 13.7% — 기동/종료 경로 테스트 보강. (4) `listUsers`만 검색어를 TrimSpace 하지 않아 다른 목록과 동작이 다른 점 정리. (5) `Migrate`의 체크섬 불일치 경로는 여전히 DB가 있어야만 검증 가능 — 트랜잭션 경계까지 포함한 통합 테스트 마련.
- 릴리즈: 없음 (PDF 재생성에 docker/Playwright 이미지가 필요해 이 세션에서는 버전 승격을 하지 않음)

## 2026-09-04 (2)
- 선택: 길이 0인 플레이 시간대가 하루 종일 허용으로 뒤집히는 문제 + 시간대 목록 전체 편집 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `playAllowed`(internal/api/catalog.go)가 `start == end`인 창을 "그 요일 하루 종일 허용"으로 읽어서, 설정 화면에 09:00~09:00으로 — 즉 길이 0으로 — 보이는 가장 좁은 정책이 실제로는 그 요일에 아무도 막지 않는 상태였다(스위치는 켜져 있고 요일도 체크돼 있는데 서버는 24시간 통과). 이제 창은 시작부터 종료까지(양끝 포함)만 덮고, 같은 두 시각은 그 1분만 뜻하며, 저장 검증(`validateSetting`)은 아예 거부한다 — 시간 제한이 없어야 하는 정책에는 창이 없는 것이지 길이 0인 창이 있는 게 아니다. 판정 로직을 DB 없이 테스트 가능한 `playWindowsAllow(windows, now)`로 분리해 요일·경계·자정 넘김·읽을 수 없는 값까지 표로 검증했다. 같은 화면이 첫 창만 편집·표시해서 나머지는 보존만 되고 고칠 수 없었는데(그 상태로는 새 검증에 걸려 이후 저장이 전부 막혔을 것), 이제 목록 전체를 편집하고 창마다 삭제 버튼과 "허용 시간대 추가"를 제공하며 여러 창은 OR로 합쳐진다고 한국어로 설명한다. 검증: `gofmt -l`, `go vet`, `go build ./...`, `go test` 전체와 `go test -race ./internal/api/...` 통과, `npm --prefix web run lint`, `npx tsc -b` 통과, `npm --prefix web test` 221개 통과.
- 보류 아이디어: (1) `cmd/igame` 커버리지 13.7% — 기동/종료 경로 테스트 보강. (2) `listUsers`만 검색어를 TrimSpace 하지 않아 다른 목록과 동작이 다른 점 정리. (3) `Migrate`의 체크섬 불일치 경로는 여전히 DB가 있어야만 검증 가능 — 트랜잭션 경계까지 포함한 통합 테스트 마련. (4) `<input type="time">`은 24:00을 입력할 수 없어 "종일 허용" 창은 화면에서 00:00~23:59까지만 만들 수 있는 점 정리. (5) 진행 중인 세션은 `duration_ms`가 0이라 일일 제한이 한 세션 안에서는 걸리지 않는 문제 확인.
- 릴리즈: 없음 (PDF 재생성에 docker/Playwright 이미지가 필요해 이번에도 버전 승격은 보류)
- 릴리즈: v0.7.5 (2026-09-04)

## 2026-09-05
- 선택: 대문자 스킴이 세션 쿠키의 Secure 플래그를 떨어뜨리는 문제 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: URL 스킴은 대소문자를 가리지 않아 `validateSetting`이 `public_url="HTTPS://games.example.com"`을 통과시킨다(`url.Parse`가 검사 전에 스킴을 소문자로 만든다). 그런데 저장된 값을 읽는 쪽은 원문 문자열을 그대로 비교했고, 그 비교가 전부 `"https://"` 접두사 검사였다 — 세션 쿠키의 Secure 플래그(auth.go:173), HSTS 헤더(api.go:280), 상태 화면의 TLS 표시(status.go:81). 대문자 하나 때문에 TLS로 서비스되는 배포의 세션 쿠키가 Secure 없이 나가서 같은 호스트로 가는 평문 요청에 실려 나갔다. `canonicalBaseURL`을 추가해 `requestBaseURL`이 스킴을 소문자로 정규화한 값을 돌려주게 하고 상태 화면도 같은 헬퍼를 쓰게 했다 — 이미 그렇게 저장해 둔 배포도 아무도 눈치채고 다시 저장할 필요 없이 고쳐진다. 호스트 표기는 건드리지 않았다(읽는 쪽이 이미 `EqualFold`로 비교한다). 검증: `gofmt -l`, `go vet`, `go build ./...`, `go test`와 `go test -race` 전체 통과. 웹 테스트/린트는 이 워크트리에 `web/node_modules`가 없어(eslint/vitest 미설치, 오프라인) 실행하지 못했고, 변경은 Go 파일 3개뿐이라 프런트엔드에 닿지 않는다.
- 보류 아이디어: (1) 일일 플레이 제한이 진행 중인 세션의 경과 시간을 세지 않아(`duration_ms`가 NULL) 제한을 한 세션만큼 넘겨 시작할 수 있는 문제 — SQL 한 줄 수정이지만 검증에 PostgreSQL이 필요하다. (2) `internal/web`의 정적 핸들러가 디렉터리도 파일로 열어서 `/assets/`에 번들 파일 목록을 그대로 내준다 — `Handler()`를 `fs.FS`를 받는 형태로 분리하면 테스트 가능. (3) `cmd/igame` 커버리지 13.7% — 기동/종료 경로 테스트 보강. (4) `listUsers`만 검색어를 TrimSpace 하지 않아 다른 목록과 동작이 다른 점 정리. (5) `Migrate`의 체크섬 불일치 경로는 여전히 DB가 있어야만 검증 가능 — 트랜잭션 경계까지 포함한 통합 테스트 마련.
- 릴리즈: v0.7.6 (2026-09-05)
## 2026-09-06
- 선택: 정적 핸들러가 `/assets/`에 번들 파일 목록을 그대로 내주는 문제 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `Handler()`(internal/web/web.go)가 "열리면 파일"로 취급해 `root.Open(clean)`이 성공하면 곧장 `http.FileServer`에 넘겼는데, 디렉터리도 열린다. 그래서 `/assets/`는 빌드가 만든 모든 번들 파일 이름이 링크로 박힌 디렉터리 인덱스를 200으로 돌려줬고(재현 확인), `/assets`는 슬래시를 붙이는 301을 냈다 — 기존 테스트가 막으려던 SPA 리다이렉트가 디렉터리 경로에서만 새고 있었다. 이제 `Stat()`으로 정규 파일만 서빙하고 디렉터리는 기존 규칙(확장자 없으면 SPA index, 있으면 404)으로 떨어뜨린다. 테스트가 가능하도록 `Handler()`를 `handlerFor(fs.FS)`로 분리했다 — 저장소에는 `dist/index.html` 자리표시자뿐이라 실제 번들을 `fstest.MapFS`로 세운다. 검증: `gofmt -l`, `go vet ./...`, `go build ./...`, `go test ./...`와 `go test -race ./...` 전체 통과. 웹 테스트/린트는 이 워크트리에 `web/node_modules`가 없어(오프라인) 실행하지 못했고, 변경은 Go 파일 2개뿐이라 프런트엔드 소스에 닿지 않는다.
- 보류 아이디어: (1) 일일 플레이 제한이 진행 중인 세션의 경과 시간을 세지 않아(`duration_ms`가 NULL) 제한을 한 세션만큼 넘겨 시작할 수 있는 문제 — SQL 한 줄이지만 검증에 PostgreSQL이 필요하고 이 환경에는 없다. (2) `cmd/igame` 커버리지 13.7% — 기동/종료 경로 테스트 보강. (3) `listUsers`만 검색어를 TrimSpace 하지 않아 다른 목록과 동작이 다른 점 정리. (4) `Migrate`의 체크섬 불일치 경로는 여전히 DB가 있어야만 검증 가능 — 트랜잭션 경계까지 포함한 통합 테스트 마련. (5) `<input type="time">`은 24:00을 입력할 수 없어 "종일 허용" 창은 화면에서 00:00~23:59까지만 만들 수 있는 점 정리.

- 릴리즈: v0.7.7 (2026-09-06, run 2026-09-06-145047-igame-improve)
## 2026-09-07
- 선택: 진행 중인 세션을 일일 플레이 제한에 반영 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `duration_ms`는 세션이 끝날 때만 기록되므로 `playAllowed`(internal/api/catalog.go)의 `sum(duration_ms)`가 아직 열려 있는 세션을 0분으로 셌다. 제한은 세션을 시작할 때 확인하는데 그 순간 열려 있는 세션이 바로 합계에서 빠진 것이라, 종료된 세션만 제한 아래면 이미 몇 시간을 플레이했든 다시 시작이 허용돼 매일 한 세션만큼 한도를 넘겼다(제한 60분에 55분씩 플레이하면 110분까지 통과). 열린 세션은 시작 이후 경과 시간을 더하도록 고쳤다 — 세션을 finish 할 때와 새 세션이 이전 세션을 abandoned 로 닫을 때 이미 쓰는 것과 같은 계산이다. 판정 주체가 SQL이라 실제로 돌려서 검증했다: `IGAME_TEST_DSN`이 있을 때만 도는 PostgreSQL 테스트(internal/api/catalog_pg_test.go)가 마이그레이션을 버릴 DB에 적용하고 진행 중 세션·서비스 하루 시작 이전 세션·duration 없이 닫힌 세션 세 경우를 확인하며, `make test-db DSN=...` 타깃과 README 안내를 추가했다. 검증: `gofmt -l`, `go vet ./cmd/... ./internal/... ./migrations/...`, `go build ./...`, `go test`·`go test -race` 전체 통과, docker `postgres:17-alpine` 컨테이너로 새 테스트 3개 통과 및 옛 쿼리로 되돌리면 실패함을 확인. 웹 테스트/린트는 이 워크트리에 `web/node_modules`가 없어(오프라인) 실행하지 못했고, 변경은 Go 1개·테스트 1개·Makefile·README 뿐이라 프런트엔드에 닿지 않는다.
- 보류 아이디어: (1) `cmd/igame` 커버리지 13.7% — 기동/종료 경로 테스트 보강. (2) `listUsers`만 검색어를 TrimSpace 하지 않아 다른 목록과 동작이 다른 점 정리. (3) `Migrate`의 체크섬 불일치·트랜잭션 경계 통합 테스트 — 이제 `IGAME_TEST_DSN` 하네스가 생겨 실제로 쓸 수 있다. (4) `<input type="time">`은 24:00을 입력할 수 없어 "종일 허용" 창을 UI에서 만들 수 없는 점 정리. (5) `issueSession`의 쿠키 MaxAge가 `s.Now()`가 아니라 실제 시계(`time.Until`)로 계산돼 주입된 시계와 어긋나는 점 정리.

## 2026-09-08
- (원장 항목에 비밀/내부 정보 의심 문자열이 있어 비공개 기록으로 옮김 — run 2026-09-08-202131-igame-improve)

## 2026-09-09
- 선택: rankings의 group=department|team이 조직 공개 정책을 우회하는 문제 (가치 5 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `rankings`(internal/api/catalog.go)가 부서·팀 분기를 먼저 처리해 `writeJSON`으로 빠져나간 뒤에야 privacy 설정을 읽어서, `show_department`가 꺼져 있어도 `rankings:read`만 있으면 부서·팀 이름과 인원수·합계 점수를 전부 열거할 수 있었다(같은 함수의 개인 분기는 department를 가리고, Defense 랭킹은 같은 요청을 403 `organization_ranking_hidden`으로 막고 있어 의도는 이미 확립돼 있었다). 이제 group 검증과 정책 판정이 첫 쿼리보다 먼저 일어나고, 읽을 수 없는 정책은 "공개"가 아니라 503 `privacy_setting_unavailable`이며, 개인 항목에서도 department와 함께 team을 내린다 — team 역시 조직명이고 랭킹 화면이 department가 없을 때 team을 대신 보여주고 있었다. 게이트가 DB 접근보다 앞이라 새 테스트(internal/api/rankings_test.go)는 설정 캐시만 심고 DB 없이 핸들러를 돌려, 거부된 요청은 상태 코드를, 허용된 요청은 nil 풀 도달을 증거로 확인한다. 검증: `gofmt -l`, `go vet ./cmd/... ./internal/... ./migrations/...`, `go build ./...`, `go test`·`go test -race` 전체 통과. 웹 린트/테스트는 이 워크트리에 `web/node_modules`가 없어(오프라인) 실행하지 못했고, 변경은 Go 1개·테스트 1개·docs/api.md 뿐이라 프런트엔드 소스에 닿지 않는다(403 코드는 errorMessages.ts에 이미 한국어 문구가 있다).
- 보류 아이디어: 일시적 설정 읽기 실패가 OIDC client secret / AI API 키를 지움 (admin.go:358·:458의 무시된 읽기 오류) / playAllowed의 무시된 오류 3곳이 전부 fail-open (catalog.go:401·:420·:427) / 관리자 비밀번호 재설정이 대상 사용자의 auth_sessions를 지우지 않음 / 동점일 때 row_number()와 바깥 ORDER BY가 독립적으로 정렬돼 rank 번호와 행 순서가 어긋날 수 있음 / 잘린 감사 로그 CSV가 200과 완전한 헤더로 내려가고 audit.export가 잘린 개수를 전체인 양 기록함

- 릴리즈: v0.7.8 (2026-09-09, run 2026-09-09-011059-igame-improve)
## 2026-09-09
- 선택: 일시적 설정 읽기 실패가 저장된 OIDC client secret / AI API 키를 지우는 문제 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `putOIDCSetting`·`putAISetting`(internal/api/admin.go)은 요청이 secret을 비워 보내면 저장된 값을 이어받는데, 설정 화면은 secret을 절대 돌려받지 못하므로 **모든 평범한 저장이 이 경로**다. 그런데 기존 값을 읽는 `_ = s.setting(...)`이 오류를 버려서, 캐시가 일시 장애를 담지 않는 이상(cache.go의 의도적 설계) DB가 잠깐 흔들린 순간의 읽기 실패가 "secret이 없다"와 같은 zero 값이 됐고, 그대로 빈 문자열을 덮어써 동작 중인 SSO 로그인이나 AI 연동을 끊으면서 200과 "설정을 갱신했다"는 감사 기록을 남겼다(감사 항목의 이전 값도 zero로 기록됐다). 이제 두 핸들러 모두 `pgx.ErrNoRows`가 아닌 읽기 오류에 503 `oidc_setting_unavailable`·`ai_setting_unavailable`로 거부하고, 아직 key가 없는 최초 저장은 그대로 진행한다 — 지난 회차 rankings 수정과 같은 방식이다. 게이트가 쓰기보다 앞이라 새 테스트(internal/api/admin_secret_test.go)는 설정 캐시만 심고 DB 없이 핸들러를 돌려, 거부된 저장은 상태 코드로, 허용된 저장은 nil 풀 도달로 확인한다. 검증: `gofmt -l`, `go vet ./cmd/... ./internal/... ./migrations/...`, `go build ./...`, `go test`·`go test -race` 전체 통과, 수정을 되돌리면 새 테스트 2개가 실패함도 확인. 웹 린트/테스트는 이 워크트리에 `web/node_modules`가 없어(오프라인) 실행하지 못했고, 프런트엔드 변경은 errorMessages.ts에 한국어 문구 2줄을 더한 것뿐이다.
- 보류 아이디어: playAllowed의 무시된 오류 3곳이 전부 허용 방향으로 fail-open (catalog.go의 정책 읽기·slug 조회·합계 조회) / 관리자 비밀번호 재설정이 대상 사용자의 auth_sessions를 지우지 않아 탈취된 세션이 최대 12시간 더 삶 / 동점일 때 row_number()와 바깥 ORDER BY가 독립적으로 정렬돼 rank 번호와 행 순서가 어긋날 수 있음 / 잘린 감사 로그 CSV가 200과 완전한 헤더로 내려가고 audit.export가 잘린 개수를 전체인 양 기록함 / 게임에 묶이지 않은 전역 업적은 content.go의 `JOIN ... ON gs.game_id=a.game_id`가 NULL과 매치되지 않아 클라이언트에서 해금될 수 없음

## 2026-09-09
- 선택: playAllowed의 무시된 오류 3곳이 전부 허용 방향으로 fail-open (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `playAllowed`(internal/api/catalog.go)는 자신이 의존하는 읽기 세 개의 오류를 전부 버렸고, 셋 다 "통과" 쪽으로 실패했다 — 정책 읽기 실패는 '정책이 꺼져 있음'과 구분되지 않아 허용 시간대와 일일 한도가 한꺼번에 사라졌고, slug 조회 실패는 `DailyLimits[""]`를 물어 한도가 없어졌으며, 합계 조회 실패는 used=0이라 어떤 한도에도 닿지 못했다. 로그도 남지 않아 정책이 조용히 적용을 멈춘 것을 아무도 알 수 없었다. 이제 판정할 수 없는 정책은 503 `play_policy_unavailable`로 세션을 거부하고 원인은 로그로 간다 — 지난 두 회차(rankings 개인정보 게이트, OIDC/AI secret)와 같은 방식이다. key 자체가 없는 경우는 정책을 둔 적 없는 배포이므로 계속 "제한 없음"이고, 시간대 설정도 같은 규칙으로 읽는다(허용 창과 하루 경계 둘 다 그 시간대에서 정해진다). slug 조회는 아예 없앴다 — 세션 핸들러가 이미 읽고 검증한 slug를 인자로 넘긴다. 클라이언트도 이 코드에서는 연습 모드로 넘어가지 않는다(서버가 막으려던 시간에 게임이 열리는 결과였다). 새 테스트(internal/api/play_policy_test.go, 7개)는 설정 캐시만 심어 DB 없이 판정을 돌리고, 합계 실패는 아무도 듣지 않는 포트를 가리키는 풀로 실제 쿼리 오류를 만들어 확인한다. 검증: `gofmt -l`, `go vet ./cmd/... ./internal/... ./migrations/...`, `go build ./...`, `go test ./...`·`go test -race ./...` 전체 통과. 웹 린트/테스트는 이 워크트리에 `web/node_modules`가 없어(오프라인) 실행하지 못했고, 프런트엔드 변경은 errorMessages.ts 1줄과 useGameRuntime.ts의 조건 분기 하나뿐이다.
- 보류 아이디어: 관리자 비밀번호 재설정이 대상 사용자의 auth_sessions를 지우지 않아 탈취된 세션이 최대 12시간 더 삶 / 동점일 때 row_number()와 바깥 ORDER BY가 독립적으로 정렬돼 rank 번호와 행 순서가 어긋날 수 있음 / 잘린 감사 로그 CSV가 200과 완전한 헤더로 내려가고 audit.export가 잘린 개수를 전체인 양 기록함 / 게임에 묶이지 않은 전역 업적은 content.go의 JOIN이 NULL과 매치되지 않아 클라이언트에서 해금될 수 없음 / csrfProtection이 igk_가 아닌 Bearer 값에도 검사를 건너뜀(심층 방어)

## 2026-09-10
- 선택: 관리자 비밀번호 재설정이 대상 사용자의 세션을 지우지 않는 문제 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `updateUser`(internal/api/admin.go)는 `password_hash`만 쓰고 `auth_sessions`는 건드리지 않았다. 비밀번호 재설정은 관리자가 탈취된 계정을 되찾는 수단인데, 옛 비밀번호로 열린 세션이 남은 수명(최대 12시간) 동안 그대로 살아 있어서 훔친 쿠키는 계속 통했고 계정은 사실 회수되지 않았다 — 본인 변경 경로인 `changePassword`는 처음부터 나머지 세션을 지우고 있었으므로 의도는 이미 확립돼 있었다. 이제 삭제가 같은 statement의 DELETE CTE로 UPDATE와 함께 일어나 "새 비밀번호 + 옛 세션" 상태가 생길 수 없고, 살아남는 것은 관리자가 이 화면에서 자기 비밀번호를 재설정할 때의 요청 자신의 세션 하나뿐이다(중간에 로그아웃시켜서 지켜지는 것은 없다). 역할·상태는 `authenticate`가 매 요청 다시 읽으므로 쓸어낼 필요가 없다. 감사 항목에는 폐기한 세션 수를 `sessions_revoked`로 남긴다. 판정 주체가 PostgreSQL이라 실제로 돌려서 검증했다: `IGAME_TEST_DSN`이 있을 때만 도는 테스트(internal/api/admin_pg_test.go, 4개)가 빈 DB에 마이그레이션을 적용하고 재설정·본인 재설정·프로필만 수정·없는 사용자 네 경우를 확인하며, 훔친 쿠키가 정말로 인증되지 않는지 `authenticate`로 직접 확인한다. `make test-db DSN=...` 타깃과 README·docs/security.md 안내를 함께 넣었다. 검증: `gofmt -l`, `go vet ./cmd/... ./internal/... ./migrations/...`, `go build ./...`, `go test`·`go test -race` 전체 통과, docker `postgres:17-alpine`으로 새 테스트 4개 통과 및 폐기를 무력화하면 재설정 관련 2개만 실패하고 나머지 2개는 통과함을 확인, `npm --prefix web run lint`와 `npm --prefix web test` 221개 통과(프런트엔드 소스는 건드리지 않았고 감사 상세 렌더러가 이미 임의 키를 처리한다).
- 보류 아이디어: 일일 플레이 제한이 진행 중인 세션을 세지 않는 문제가 main에 그대로 남아 있음 — 2026-09-07 회차의 수정은 원격에 올라간 적 없는 브랜치에만 있다 / 동점일 때 row_number()와 바깥 ORDER BY가 독립적으로 정렬돼 rank 번호와 행 순서가 어긋날 수 있음 / 잘린 감사 로그 CSV가 200과 완전한 헤더로 내려가고 audit.export가 잘린 개수를 전체인 양 기록함 / Migrate 체크섬 불일치·트랜잭션 경계 통합 테스트 — 이번에 들어온 IGAME_TEST_DSN 하네스로 이제 가능 / 게임에 묶이지 않은 전역 업적은 content.go의 JOIN이 NULL과 매치되지 않아 클라이언트에서 해금될 수 없음

- 릴리즈: v0.7.9 (2026-09-10, run 2026-09-10-121120-igame-improve)
## 2026-09-10
- 선택: 수정 과제 — 릴리즈 워크플로의 "Audit locked Node dependencies" 실패 (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 릴리즈 워크플로는 두 Node 트리를 `--audit-level=low`로 검사하므로 심각도와 무관하게 권고 하나만 있어도 태그가 나가지 못한다. v0.7.9가 두 번 걸린 이유는 두 트리가 같은 권고 두 개를 들고 있었기 때문이다 — GHSA-82fw-gwwq-j7x9은 vitest 2.1.0~4.1.10 전체(잠긴 3.2.7 포함)를, GHSA-2883-xcg3-v3hh은 js-yaml 4.3.2 미만을 덮는다. CI는 audit 게이트를 `--omit=dev`로 돌리고 dev 권고는 step summary에 적기만 해서 PR은 끝까지 초록이었고, 처음 빨개진 곳이 릴리즈였다. 워크플로를 느슨하게 하지 않고 의존성 쪽을 고쳤다: vitest를 5.0.0으로, js-yaml을 eslint가 이미 요구하는 범위 안의 4.3.2로 올렸다. 다른 수정 라인인 4.1.11이 아니라 5.0.0을 고른 이유는 npm 10.9.8(두 워크플로와 이미지가 쓰는 Node 22에 딸려 오는 npm)이 vitest 4의 peer set을 풀다가 `Cannot read properties of null (reading 'edgesOut')`로 죽어 이 저장소에서도 빈 디렉터리에서도 설치 자체가 불가능하기 때문이다. SDK에는 포털과 같은 `vite ^7.1.3`을 명시했다 — SDK에는 vitest의 vite peer를 제약하는 것이 없어 업그레이드가 vite 8과 rolldown을 한쪽 테스트 툴체인에만 끌어왔다. 검증: 실패한 릴리즈 단계를 그대로 재현해 두 audit 모두 `found 0 vulnerabilities`(수정 전에는 같은 명령이 3건으로 exit 1), 새 lockfile로 `make deps`(npm ci), `make lint`, `make test`(SDK 9 / web 221 / Go 전체), `make web-build`와 오프라인 번들 검사, `UPDATE_KERNEL_VECTORS=1 npx vitest run ...`으로 RealmGuard vector가 바이트 동일하게 재생성됨, 그리고 두 트리의 `npm ci`를 node:22-alpine에서 도는 `make docker-build`까지 통과.
- 보류 아이디어: 릴리즈만 dev 권고에 걸리고 CI는 --omit=dev라 PR이 끝까지 초록인 구조적 공백(릴리즈 게이트를 느슨하게 하지 않고 알림을 앞당길 방법) / 일일 플레이 제한이 진행 중인 세션의 경과 시간을 세지 않는 문제가 main에 그대로 남아 있음 / 동점일 때 row_number()와 바깥 ORDER BY가 독립적으로 정렬돼 rank 번호와 행 순서가 어긋날 수 있음 / 잘린 감사 로그 CSV가 200과 완전한 헤더로 내려가고 audit.export가 잘린 개수를 전체인 양 기록함 / 게임에 묶이지 않은 전역 업적은 content.go의 JOIN이 NULL과 매치되지 않아 클라이언트에서 해금될 수 없음

- 릴리즈: v0.7.10 (2026-09-10, run 2026-09-10-130107-igame-improve)
## 2026-09-10
- 선택: 일일 플레이 제한이 진행 중인 세션의 경과 시간을 세지 않는 문제 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `playAllowed`(internal/api/catalog.go)의 `sum(duration_ms)`는 세션이 끝날 때만 기록되는 값만 더해서, 플레이어가 지금 열어 둔 세션을 0분으로 셌다. 제한은 세션을 **시작할 때** 확인하는데 그 순간 열려 있는 세션이 바로 합계에서 빠진 것이라, 종료된 세션만 한도 아래면 이미 몇 시간을 플레이했든 다시 시작이 허용돼 매일 한 세션만큼 한도를 넘겼다(60분 제한에 55분씩 앉으면 110분). 이제 `status='active'` 행은 시작 이후 경과 시간을 더한다 — 세션을 finish 할 때와 새 세션이 이전 세션을 abandoned 로 닫을 때 이미 `duration_ms`에 쓰는 것과 같은 계산이다. 끝난 세션은 기록된 duration을 그대로 쓰고, duration 없이 닫힌 세션은 시작 이후 전부를 청구당하지 않고 0으로 남는다. 판정 주체가 PostgreSQL이라 실제로 돌려서 검증했다: `IGAME_TEST_DSN` 하네스에 붙는 새 테스트(internal/api/catalog_pg_test.go, 4개)가 진행 중 세션·한도 아래 여유·서비스 하루 시작 이전 세션·duration 없이 닫힌 세션을 확인하며, 서울 하루가 시작된 지 몇 분 안 된 때에는 열린 세션을 놓을 자리가 없으므로 skip 한다. 검증: `gofmt -l`, `go vet ./cmd/... ./internal/... ./migrations/...`, `go build ./...`, `go test`·`go test -race` 전체 통과, docker `postgres:17-alpine`으로 새 테스트 4개 통과 — 옛 쿼리로 되돌리면 진행 중 세션 테스트만 실패하고, `status` 구분 없이 경과 시간을 더하는 성급한 변형으로 바꾸면 닫힌 세션 테스트가 실패함을 둘 다 확인했다. `npm ci` 후 SDK 9개·web 221개 테스트와 양쪽 lint 통과(프런트엔드 소스는 건드리지 않았고 docs/api.md 한 문단만 갱신했다).
- 보류 아이디어: 릴리즈만 dev 권고에 걸리고 CI는 --omit=dev라 PR이 끝까지 초록인 구조적 공백(릴리즈 게이트를 느슨하게 하지 않고 알림을 앞당길 방법) / 동점일 때 row_number()와 바깥 ORDER BY가 독립적으로 정렬돼 rank 번호와 행 순서가 어긋날 수 있음(u.id 같은 결정적 tiebreaker 추가) / 잘린 감사 로그 CSV가 200과 완전한 헤더로 내려가고 audit.export가 잘린 개수를 전체인 양 기록함 / Migrate 체크섬 불일치·트랜잭션 경계 통합 테스트 — IGAME_TEST_DSN 하네스로 가능 / 게임에 묶이지 않은 전역 업적은 content.go의 JOIN이 NULL과 매치되지 않아 클라이언트에서 해금될 수 없음

- 릴리즈: v0.7.11 (2026-09-10, run 2026-09-10-195111-igame-improve)
## 2026-09-11
- 선택: 가이드 캠페인 guides-2026-09 — 사용자·관리자 가이드를 실제 화면 캡처가 들어간 완성본으로 (가치 5 / 위험 1 / 작업량 L)
- 결과: 성공
- 요약: 저장소에는 캡처가 한 장도 없는 `docs/guide.md`(69줄)만 있었고 관리자 가이드는 아예 없었다. GUIDE-STANDARD.md 구성 그대로 `docs/USER_GUIDE.md`(19쪽 PDF)와 `docs/ADMIN_GUIDE.md`(22쪽 PDF)를 만들고, 그림 21장을 전부 이번에 실제로 띄운 `v0.7.11`에서 찍었다 — 릴리즈와 같은 방법으로 `make docker-build`한 이미지와 PostgreSQL을 전용 docker 네트워크에 올린 뒤, 릴리즈 브라우저 게이트가 이미 쓰는 고정 Playwright 이미지(`mcr.microsoft.com/playwright:v1.55.0-noble`)에서 1440×900으로 캡처했다. 캡처 도구(`scripts/capture-guide-screenshots.sh` + `scripts/guide-capture/`)는 목록이 빈 화면이 나오지 않도록 지어낸 사용자·점수·공지·이벤트·시즌·개인 API 키·RealmGuard Draft를 먼저 채우므로 파괴적이다 — 그래서 `IGAME_GUIDE_CAPTURE_DISPOSABLE=yes` 없이는 거부하고, 대상 URL은 다른 스크립트와 공유하지 않는 전용 변수에서만 읽고, 시스템 설정은 한 줄도 쓰지 않는다(가리킨 배포의 플레이 정책·개인정보 정책·승인 흐름이 그대로 남는다). 환경 변수 표는 `internal/config/config.go`에서, 역할표는 `Router()`의 라우트 등록과 `AdminLayout`의 `adminOnly`에서, 오류 문구 표는 `errorMessages.ts`에서 읽어 만들었다. PDF는 저장소 변환기가 아니라 공용 `tools/guide/md2pdf.mjs`로 만들었고, 정본이 둘로 갈리지 않도록 옛 `docs/guide.md`와 `docs/igame_User_Guide.pdf`를 같은 커밋에서 지우고 README·한/영 쇼케이스·sitemap 둘·`build-docs-pdf.sh`의 링크를 새 문서로 옮겼다. 검증: `gofmt -l`, `go vet`, `go build ./...`, `make lint`, `make test`(Go 전체 / SDK 9 / web 221) 통과, `scripts/check-release-contract.sh` 통과, 두 PDF를 poppler로 렌더해 표지·표·코드 블록·그림이 깨지지 않았음을 확인, 가드 없이 캡처 스크립트를 돌려 거부되는 것도 확인했다.
- 보류 아이디어: 릴리즈만 dev 권고에 걸리고 CI는 --omit=dev라 PR이 끝까지 초록인 구조적 공백 / 동점일 때 row_number()와 바깥 ORDER BY가 독립적으로 정렬돼 rank 번호와 행 순서가 어긋날 수 있음(u.id tiebreaker 추가) / 잘린 감사 로그 CSV가 200과 완전한 헤더로 내려가고 audit.export가 잘린 개수를 전체인 양 기록함 / Migrate 체크섬 불일치·트랜잭션 경계 통합 테스트 — IGAME_TEST_DSN 하네스로 가능 / 자정을 넘겨 열려 있는 세션은 어느 날의 일일 한도에도 잡히지 않음

- 릴리즈: v0.7.12 (2026-09-11, run 2026-09-11-040525-igame-approve)
