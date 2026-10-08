- 과제: ENCRYPTION_KEY의 32바이트·인코딩 거부 계약을 실제 config.Load 경로에서 회귀로 고정 (가치 2 / 위험 1 / 작업량 S)
- 왜: `internal/config/config.go`의 `ParseEncryptionKey`·`validateKey`는 정확히 32바이트를 요구하지만, 현재 `TestParseEncryptionKey`는 정상 입력 세 개의 길이와 `short` 거부만 확인한다. 환경변수를 실제로 읽는 `Load`까지 바이트 경계·디코딩·오류의 비밀 비노출 계약을 고정하면 기동 설정 검증이 약해지는 회귀를 발견할 수 있다.
- 수용 기준:
  1) plain·hex·표준 padded base64 각각에서 합성 32바이트 키는 길이뿐 아니라 기대 바이트와 정확히 일치한다. 각 형식의 decoded 0·16·24·31·33바이트는 거부하고, `hex:`/`base64:` 단독, 홀수 hex, 비hex 문자, `!` 포함 base64, 32바이트 키의 padding을 제거한 base64도 거부한다. UTF-8 plain 입력으로 32바이트 성공(예: `가` 10개+`ab`) 및 32문자지만 32바이트가 아닌 값의 거부도 고정한다.
  2) 새 `TestLoadEncryptionKeyContract`가 `t.Setenv`로 네 필수 환경변수를 모두 통제하고 실제 `Load()`를 호출한다. 위의 성공/거부 표를 이 관문에도 적용하고, 유효한 세 형식에 바깥 공백·탭·개행을 붙인 값은 정상 복원하며 빈 값·공백만 있는 값은 거부한다. 다른 세 변수는 항상 유효하게 두어 키 검증 이전의 실패를 키 검증 성공으로 오판하지 않는다.
  3) 실패 때 에러가 존재하며 반환 Config에 EncryptionKey가 없고, 비어 있지 않은 합성 비밀 payload가 오류에 포함되지 않는다. 성공 대조군이 함께 있어 항상 실패하는 Load/파서는 통과할 수 없고, decoded 바이트 비교는 길이만 맞는 오디코딩도 발견한다. 오류 문구 전체나 숫자 표현은 고정하지 않는다.
- 건드릴 파일: `internal/config/config_test.go:TestParseEncryptionKey`, 새 `TestLoadEncryptionKeyContract` — 표 기반 테스트와 필요한 테스트 전용 입력 생성만 추가. 기존 bootstrap 암호 경계 테스트 유지. 프로덕션 파일 0개, 총 변경 파일 1개.
- 검증 명령: 저장소 루트에서 `go test ./internal/config -count=1 -v`; `go test ./internal/config -race -count=3`; `gofmt -l internal/config/config_test.go`; `git diff --check`. 추가 Go 전체 확인은 기존 Makefile과 같은 `go test ./cmd/... ./internal/... ./migrations/...`이며 PostgreSQL 회귀는 DSN 없으면 skip된다는 점을 결과에 표시한다. Node 의존성·Docker·DB는 이 과제 검증에 필요하지 않다.
- 위험과 피할 것: 제품 코드·auth/session·migrations·workflows·의존성·VERSION·문서를 함께 수정하지 않는다. `secretbox.New`는 AES의 16·24·32바이트를 받아도 설치키 관문은 32바이트만 허용한다. 이 차이를 통합하거나 변경하지 않는다. `ParseEncryptionKey` 자체는 trim하지 않고 `Load`가 trim하므로 parser에 trim을 요구하는 잘못된 테스트를 만들지 않는다. 표준 base64 디코더는 내부 CR/LF를 허용하므로 모든 내부 whitespace를 거부한다고 가정하지 않는다. `t.Setenv` 때문에 `t.Parallel`을 쓰지 않는다. 실제 비밀을 사용하지 않고, 빈 payload에 `strings.Contains(error, "")` 검사를 하지 않는다. 이번은 확인된 버그 수정이 아닌 테스트 공백 보강이다. secretbox 거부 테스트와 PG fixture 복원은 이전 구현 기록이 있으므로 현재 main에 안 보이더라도 이번에 재구현하지 않는다.
- 차선 후보: 공개 카탈로그 `listGames`의 동명 게임 페이지 정렬에 `g.id` 추가 — 1순위 테스트가 이미 다른 변경으로 채워져 있을 때만 전환. `internal/api/catalog.go:listGames`의 `ORDER BY g.name` 한 질의, 신규 `catalog_order_pg_test.go`, `docs/api.md`의 보증 범위만 대상으로 한다. `admin_order_pg_test.go`의 `pageIDs`·`assertEachRowHandedOutOnce`·`putGame` 패턴을 재사용해 동명 active 게임 6개를 실제 PG에 준비하고 `/api/v1/games?q=<고유표식>&limit=2&offset=...`를 실제 router/세션 쿠키로 순회한다. 첫 페이지에서 **이미 본 행**을 PUT으로 수정하되 name과 필터 포함 여부는 유지한다(기존 putGame 예제를 그대로 복사해 이름을 바꾸면 이 시험은 무효). 각 id가 한 번씩 나오는지 검증하며 실제 Red부터 확인한다. `limit=1` 정렬 알고리즘 의존 재현은 피한다. `make test-db DSN='postgres://igame:igame@127.0.0.1:15432/igame?sslmode=disable&search_path=public,igame_test_extensions'`는 README의 일회용 PG17/전용 pgcrypto 스키마 준비 후 사용한다. 이번 정찰에서 이 차선 Red/DB 실행은 미확인이다.

근거와 경계:
- 기준 `main@73ee379`, VERSION 0.7.30. config.go의 `Load`(28행), `ParseEncryptionKey`(67행), `validateKey`(85행), config_test.go 전체와 cmd/igame/main.go:37–64를 직접 읽었다. main은 Load 실패 시 종료하고, 성공해야 DB 연결/마이그레이션/secretbox.New로 진행한다. 테스트는 실제 설정 로딩 경로까지 증명하며 프로세스 종료·로그 출력·DB 기동 end-to-end를 증명한다고 주장하지 않는다.
- 기존 `go test ./internal/config -count=1 -v`: PASS, 0.004s, 상위 테스트 3개. 새 테스트는 아직 없으므로 구현 전 Red를 주장하지 않는다. -race 결과는 정찰 노트에 기록한다.
- 구조화 후보 8개를 재평가하고 신규 2개를 더했다(ideas.json). 페이지 정렬은 사용자 가치는 있으나 실DB와 안정적인 Red에 시간이 들고, README 정합성은 위험이 낮지만 이번 키 관문 공백보다 우선하지 않았다. serviceLocation 전면 오류 전파는 범위가 넓고, legacy 정책 fail-open 주장은 현재 코드와 맞지 않는다.

실행 순서와 체크포인트(현재 모두 구현 대기, 사람 승인 불필요):
1. 테스트 전용 키 표를 만들고 `TestParseEncryptionKey`를 보강한다. 확인: `go test ./internal/config -run '^TestParseEncryptionKey$' -count=1 -v`. 성공/실패 및 bytes.Equal 단정을 보고 다음 단계로 진행한다.
2. 같은 계약을 환경변수→`Load`로 통과시키고 trim·빈 값·비밀 비노출 단정을 추가한다. 확인: `go test ./internal/config -run '^TestLoadEncryptionKeyContract$' -count=1 -v`. 다른 필수 설정 오류가 키 오류를 가리지 않는지 확인한다.
3. 위 검증 명령의 패키지 전체·race·format/diff 검사를 실행한다. 결과와 실제 변경 파일 1개를 기록하고 인계한다. 증거가 과제와 다르면 과제서를 먼저 수정하며 제품 코드 수정으로 확대하지 않는다.

추정과 대안:
- 방법: 실제 읽은 단일 파일 범위의 bottom-up 추정. 표/파서 단정 8–12분, Load 경로·실패 단정 10–15분, 검증·정리 5–8분으로 순수 작업 23–35분. 알려진 불확실성(입력 생성·환경변수 테스트 충돌) contingency 5분을 별도로 둬 28–40분을 예상한다. 경험적 추정이며 통계적으로 보정된 확률은 없다. Go 도구와 캐시는 이번 패키지 실행으로 확인했고 DB·외부 서비스는 제외한다. 관리 예비비는 0분이며 45분 이상 범위 확장은 하지 않는다.
- 대안 비교: (A) parser만 보강하면 빠르지만 Load 배선/trim을 놓침; (B) parser+실제 Load는 1파일로 관문까지 검증하므로 선택; (C) 실행 바이너리와 DB까지 기동하면 종료·로그를 검증할 수 있으나 이 순수 설정 계약에는 환경 비용이 큼; (D) 현상 유지는 비용 0이나 16/24바이트 허용으로 약화되는 회귀를 막지 못함. 가장 큰 가정은 32바이트·현재 세 입력 형식이 계속 제품 계약이라는 것(README와 config.go가 현재 일치).
- 적용 스킬: 전용 Skill 도구가 노출되지 않아 로컬 원본 `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/{pmo,technology}/skills/`에서 estimating-and-contingency, implementation-planning, solution-exploration의 SKILL.md를 직접 읽었다. pmo의 references/sources.md도 읽었으며 외부 기관의 수치/확률을 이 소규모 로컬 추정의 근거로 인용하지 않았다.
