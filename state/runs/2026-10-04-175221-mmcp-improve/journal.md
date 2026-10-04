# 회차 노트 2026-10-04-175221-mmcp-improve — mmcp
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 17:52] base pinned — main@b782c5f
- [러너 17:52] autonomy release — 

## 정찰 노트
- 골랐다: `mmcp reset-password` 가 >72바이트 비밀번호에 `password_hash=''` 를 쓰고 성공을 보고하는 것. `bcrypt.GenerateFromPassword` 가 `(nil, ErrPasswordTooLong)` 을 돌려주는 것을 모듈 소스(`x/crypto@v0.57.0/bcrypt/bcrypt.go:96`)에서 직접 확인했고, `main.go:146` 이 그 에러를 `_` 로 버리는 반면 같은 env 를 읽는 `users.go:169 bootstrapAdmin` 은 올바르게 반환한다 — 두 경로가 갈려 있어 운영자 규칙에 정확히 걸리는 자리다. 테스트는 `internal/config`(`[no test files]`)에 들어가고 Postgres 가 필요 없다.
- 제친 이유: web `npm test` 과제는 결함이 base 에 그대로 남아 있지만 세 회차 연속 선택돼 모두 review-pending 으로 머지되지 않았다 — 네 번째 제출은 안 한다(ideas.json 에서 rejected). `internal/mattermost` 테스트는 2026-10-02 PR 과 같은 파일명이라 충돌 위험. logbuf 는 두 회차 연속 손댔다.
- 추측으로 적은 것: `decodeKey` 가 받는 키 형식을 열어 보지 않았다(브리프에 "미확인" 표기, 테스트 값은 `crypto_test.go` 에서 재사용하라고 지시). `ChannelMembers` 호출처는 `tools_mm.go` grep 에서 0건이었다 — 죽은 코드인지 미확인(차선도 아니므로 영향 없음).
- 구현자가 조심할 것: `Load()` 에 72바이트 가드를 넣으면, **관리자 계정이 이미 있어서** 오늘은 기동되는 설치가 다음 부팅에 멈춘다(`bootstrapAdmin` 이 해싱을 건너뛰기 때문). 브리프에 근거와 함께 적어 뒀다 — 에러 메시지에 조치 방법을 반드시 넣을 것. 그리고 `make test` 는 web 을 불러 이 워크트리에서 떨어지므로 검증에 쓰지 말 것.
- [러너 17:57] scout done — `BOOTSTRAP_ADMIN_PASSWORD` 가 72바이트를 넘으면 `mmcp reset-password` 가 `password_hash=''` 를 쓰고 "password reset for ..." 를

## 구현 노트
- 무엇/왜: `config.Load` 에 `len(password) > 72` 가드(기존 `problems`/`strings.Join` 관례·바이트 셈 유지)를 넣고, `resetAdmin()` 이 DB 를 열기 **전에** 해싱해 에러면 `UPDATE` 를 실행하지 않고 exit 1 하게 했다. 버려진 `hash, _ :=` 가 `password_hash=''` 를 쓰고 exit 0 으로 성공을 보고하던 깨어진 글래스 경로를 막는다. 신규 `internal/config/config_test.go` 8개 + `docs/ADMIN_GUIDE.md` 2줄과 그 생성물 `docs/admin-guide.html` 재생성.
- 확신 없는 곳·검증 못 한 것: (1) **업그레이드 회귀는 실제로 존재한다** — 관리자 계정이 이미 있어 오늘 73바이트+ 비밀번호로 기동되는 설치는 이 커밋 후 **기동 실패**한다. 과제서가 맞는 거래라고 판단했고 에러에 한계·실제 길이·조치(72바이트 이하로 줄여 재시작)를 담았지만, 그런 설치가 실제로 존재하는지는 확인할 방법이 없었다. 비평가가 먼저 볼 곳. (2) 에러 메시지를 **영어**로 썼다 — 저장소 관례는 사용자 노출 문자열이 한국어지만 `config.go` 의 기존 4개 메시지가 모두 영어여서 파일 내 일관성을 택했다(과제서는 "같은 기준(72바이트)을 말해야" 로만 요구). 한국어가 맞다고 보면 바꿀 자리는 `config.go:42` 한 줄뿐이다. (3) `resetAdmin()` 의 새 가드는 config 가드 뒤라 BOOTSTRAP_ADMIN_PASSWORD 경로로는 도달 불가 — 죽은 코드가 아님을 config 가드를 `if false` 로 꺼서 확인했고 되돌렸다.
- 일부러 하지 않은 것: `users.go:validatePassword`/`hashPassword` 미변경(이미 옳다). 보호 경로(`auth.go`·`oidc.go`·`keys.go`·`oauth.go`), 마이그레이션, 릴리즈 경로(`VERSION`·`package-offline.sh`·`release.yml`), `web/` 미변경. `config.Load` 에 복잡도 규칙(`validatePassword` 의 3종류 검사)은 넣지 않았다 — 범위 밖이고 기존 설치를 더 깨뜨린다. `.env.example` 은 규칙상 손대지 않았다(72바이트 언급 없음 — 다음 역할이 원하면 거기까지 확장 가능).
- 다음 역할이 조심할 것: `internal/config` 테스트는 **Postgres 불필요**(`go test -race -count=1 ./internal/config/`). `t.Setenv` 때문에 이 파일의 테스트는 `t.Parallel` 을 쓸 수 없다 — 추가할 때 주의. 전체 통합 테스트는 `TEST_POSTGRES_DSN` 이 있어야 돌고(없으면 `internal/server` 가 1초에 `ok` 로 끝난다), 나는 전용 스크래치 컨테이너로 돌려 22 PASS·SKIP 0 을 확인한 뒤 컨테이너를 지웠다. **검증에 `make test` 를 쓰지 말 것**(web 을 부르고 base 의 `web/package.json:10` 때문에 깨진다) — `go test` + `make lint` + `go build ./...` 를 쓸 것. `docs/admin-guide.html` 은 생성물이므로 `ADMIN_GUIDE.md` 를 또 고치면 `node scripts/build-docs.mjs` 를 다시 돌려야 한다(`web/node_modules` 필요; 이 워크트리에는 `npm ci` 로 설치해 뒀고 gitignore 되어 커밋에 없다).
- [러너 18:06] brief accepted — 채택 — 과제서의 근거가 현재 코드와 글자 단위로 일치했고(`main.go:146` 의 `hash, _ :=`, `users.go:169` 의 올바른 에러 반환, `in
- [러너 18:06] verify failed — 실패한 검증: cd web && npm test --silent (exit 1)
