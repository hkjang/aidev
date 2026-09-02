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
