# 과제서 (2026-09-20, 정찰)

- 과제: 마스킹 규칙이 `${VAR}` 같은 변수 참조를 자격증명으로 오인하지 않게 수정 (가치 3 / 위험 2 / 작업량 M)

- 왜: `internal/contentsecurity/sanitize.go`의 `Sanitize`가 값 자리에 온 **변수 참조 자체**를 비밀로 보고 지운다 — 이 정찰 세션에서 임시 테스트로 직접 확인한 수정 전 출력:
  - `password: ${DB_PASSWORD}` → `password: [REDACTED]` (finding=credential_assignment)
  - `password: $DB_PASSWORD`, `password: "${DB_PASSWORD}"`, `password: %(DB_PASSWORD)s` → 전부 `[REDACTED]`
  - `Authorization: Bearer ${API_TOKEN}` → `Bearer [REDACTED]`
  - `DATABASE_URL=postgres://app:${DB_PASSWORD}@db:5432/app` → `postgres://[REDACTED]@db:5432/app` (finding=credential_dsn)
  - `machine api login ${USER} password ${PASS}` → 둘 다 `[REDACTED]`
  - (반면 Helm `{{ .Values.db.password }}`와 GitHub Actions `${{ secrets.X }}`는 공백 때문에 이미 매치되지 않음 — finding 없음)
  이 형태는 docker-compose·`.env.example`·CI YAML·Maven `settings.xml`(`<password>${env.PW}</password>`)·Spring XML(`value="${db.password}"`)이 비밀을 **파일에 두지 않으려고** 쓰는 관용구라, 저장소마다 자격증명이 하나도 없는 파일에 `index_security_events`가 오르고(`internal/indexer/indexer.go:766`), 스니펫에서는 "어느 환경변수를 읽는지"라는 유일한 정보가 사라진다. 고치면 허위 보안 이벤트가 줄고 `find-*`·스니펫이 변수 이름을 그대로 보여 준다.

- 수용 기준:
  1) 아래 입력이 `Sanitize`를 **원문 그대로, finding 없이** 통과한다: `password: ${DB_PASSWORD}`, `password: $DB_PASSWORD`, `password: "${DB_PASSWORD}"`, `"password": "${DB_PASSWORD}"`, `api_key: %(API_KEY)s`, `Authorization: Bearer ${API_TOKEN}`, `postgres://app:${DB_PASSWORD}@db:5432/app`, `<password>${env.DB_PASSWORD}</password>`, `<property name="db.password" value="${db.password}"/>`.
  2) 진짜 값은 여전히 가려진다(기존 테이블 전부 통과): `password: hunter2secret`, `Bearer eyJ…`, `postgres://app:s3cr3t@db`, 그리고 **기본값이 붙은 참조 `password: ${DB_PASSWORD:-hunter2secret}` 은 계속 마스킹**(기본값 자리에 실제 비밀이 들어가는 형태이므로 보수적으로 유지), `api_key: ${API_KEY}$uffix` 처럼 참조 뒤에 다른 문자가 이어지면 계속 마스킹.
  3) 테스트가 증명할 것: (a) `TestOrdinaryContentIsLeftAlone`(`internal/contentsecurity/credential_shapes_test.go:147`)의 목록에 1)의 입력을 추가하고, 소스만 되돌리면 실패함을 확인. (b) `TestCredentialShapesAnInstallationActuallyHolds`(같은 파일 :20)에 2)의 "기본값 붙은 참조"·"참조 뒤 접미사" 2건을 마스킹 케이스로 추가. (c) `TestMaskingNeverChangesTheLineCount`(:191)가 계속 통과 — 이 변경은 치환을 **덜** 하므로 줄 수를 건드리지 않아야 함. (d) `TestTheMaskingRevisionTracksTheRules`(:261)가 통과하고, `Revision()` 값이 변경 전과 달라짐(재색인이 걸리도록 — 아래 참조).

- 건드릴 파일:
  - `internal/contentsecurity/sanitize.go`
    - 새 패키지 변수 `placeholderRE` (예: `^(?:\$\{[A-Za-z_][A-Za-z0-9_.]*\}|\$[A-Za-z_][A-Za-z0-9_]*|%\([A-Za-z_][A-Za-z0-9_.]*\)s)$` — 값 **전체**가 참조 하나일 때만; `:-`·`:=`·`-` 기본값 구문은 일부러 제외) 와 헬퍼 `isPlaceholder(value string) bool` (앞뒤 공백·따옴표 `"'` 를 벗긴 뒤 `placeholderRE.MatchString`).
    - `Sanitize`(:153~208)의 콜백에서 값 부분이 placeholder 이면 `value`를 그대로 반환하고 `finding`을 세우지 않도록: `secretAssignmentRE` 콜백(:162 — `at` 뒤 부분), `authorizationHeaderRE` 콜백(:196 — 스킴 뒤 토큰), `credentialURLRE` 콜백(:172 — `:`와 `@` 사이 비밀번호 부분; 사용자 이름은 판단 대상 아님), `xmlSecretElementRE`(:180 — `>`와 `</` 사이), `xmlSecretAttributeRE`(:184 — `value="` 뒤). `netrcRE`(:192)·`curlUserRE`(:188)는 선택 — 시간이 남으면 같은 헬퍼로.
    - `Revision()`(:141) 의 join 목록에 `placeholderRE.String()` 을 추가 — 지문은 패턴에서만 파생되므로 Go 쪽 제외 로직만 넣으면 지문이 안 바뀌어, 이미 `[REDACTED]`로 저장된 청크와 허위 이벤트가 그대로 남는다. 패턴을 지문에 넣으면 기존 ref 가 자동 재색인된다(`docs/operations.md:220` 데이터 보호 절이 설명하는 동작).
    - 헬퍼 위 주석은 이 파일의 다른 규칙처럼 "왜"(어떤 파일이 어떻게 깨졌나)를 적을 것.
  - `internal/contentsecurity/credential_shapes_test.go` — 위 수용 기준 3)의 케이스 추가. 새 테스트 함수를 만든다면 이름은 이 파일 관례대로 문장형(예: `TestAVariableReferenceIsNotACredential`).
  - `docs/operations.md` "## 데이터 보호"(:220) 문단에 한 문장: 값이 변수 참조(`${VAR}`, `$VAR`, `%(VAR)s`)뿐이면 마스킹하지 않고 이벤트도 올리지 않는다는 것과, 기본값이 붙은 형태는 계속 가린다는 것.

- 검증 명령 (이 저장소에서 실제로 도는 순서):
  1) `go test -tags sqlite_fts5 -race ./internal/contentsecurity/` (수정 전 새 케이스 실패 확인 → 수정 후 통과)
  2) `gofmt -l ./cmd ./internal` (출력 없어야 함), `go vet ./...`, `go build -tags sqlite_fts5 ./...`
  3) `go test -tags sqlite_fts5 ./internal/indexer/ ./internal/search/` (Sanitize 호출자 — `indexer.go:1173`, `search/service.go:2298·3011·4698` 모두 같은 함수를 타므로 경로 간 불일치는 구조적으로 없음)
  4) `go test -tags sqlite_fts5 ./...` (수 분; `internal/app` 은 통과해도 ~105초 걸리는 게 정상)

- 위험과 피할 것:
  - **덜 가리는 방향의 변경**이므로 placeholder 판정은 좁게: 값 전체가 참조 하나일 때만. `${VAR:-default}`, `${VAR}suffix`, `$VAR/path` 같은 것은 계속 마스킹(수용 기준 2). 정규식을 넓혀 "달러가 들어가면 통과" 로 만들지 말 것 — 운영자 규칙: 파서를 넓히지 말고 판정을 좁히는 방향.
  - 규칙의 정규식 자체(`secretAssignmentRE` 등)는 건드리지 말 것 — 줄 수 불변식(`TestMaskingNeverChangesTheLineCount`)을 여러 번 고쳐 온 자리. 콜백 안에서 "그대로 반환" 만 추가.
  - `Revision()` 에 새 패턴을 넣는 것을 잊지 말 것(위 설명). 반대로 지문을 상수로 바꾸지 말 것.
  - `internal/indexer`·`internal/search`·`internal/store` 는 손대지 않음. 보호 경로(auth/migrations/workflows) 무관.
  - 커밋 메시지는 영어 `fix(contentsecurity): …` 관례.

- 차선 후보: `Sanitize` 의 finding 이 가장 심각한 규칙이 아니라 **마지막에 실행된 규칙**을 보고 (가치 2 / 위험 1 / S) — `sanitize.go:157~207` 에서 `finding` 이 매 콜백마다 덮어써지므로 `token = ghp_…` 한 줄이 `known_token` 이 아니라 뒤 규칙의 값으로 기록될 수 있음. 고치려면 심각도 순위(private_key > known_token > cloud_access_key > credential_dsn > credential_assignment > high_entropy_secret 같은)를 정해 더 높은 것만 남기고, 그 순위를 테스트로 고정. 1순위와 같은 파일이므로 둘을 한 커밋에 섞지 말 것.
