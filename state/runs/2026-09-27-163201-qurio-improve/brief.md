- 과제: sqlsafe PostgreSQL 게이트가 `col=(SELECT …)` 같은 스칼라 서브쿼리를 함수 호출로 오판해 차단하는 오탐 수정 (가치 4 / 위험 3 / 작업량 M)

- 왜: `postgresReadOnlyViolations`(internal/domain/sqlsafe/sqlsafe.go:450-470)이 "`(` 토큰의 바로 앞 토큰이 word 이면 그 word 는 호출되는 함수 이름" 이라고 단정하는데, `scanOraclePolicy`(같은 파일 1002-1098)는 `=`·`>`·`<`·`||` 같은 연산자를 **토큰으로 아예 내보내지 않는다**(1079-1094 의 default 분기가 식별자 룬이 아닌 문자를 조용히 건너뛴다). 그래서 `WHERE version=(SELECT max(version) FROM meta_word_dict)` 의 `version` 이 `version()` 호출로 읽히고, `VERSION` 은 `postgresPureFunctions` 허용목록에 없으므로(237~380행에 LEFT·CONCAT_WS·JSONB_BUILD_OBJECT·JSONB_STRIP_NULLS·MAX·COALESCE 는 있으나 VERSION 은 없음 — grep 으로 확인) "비허용 함수 호출 version()" 으로 차단된다. 이 때문에 `internal/legacyapi/embedding_materialization.go:674` 의 `sqlsafe.ApplyRowLimit(query, "postgres", …)` 가 `defaultEmbeddingMaterializationQuery`(같은 파일 1251행, `WHERE version=(SELECT max(version) FROM meta_word_dict)` 를 6개 UNION 지 all 에서 반복)를 거부해 기본 임베딩 컬렉션 생성 경로가 실패한다. 고치면 이 사용자 기능이 살아나고, 스칼라 서브쿼리를 쓰는 모든 정상 조회 쿼리의 오탐이 사라진다.

- 수용 기준:
  1) `sqlsafe.AnalyzeDialect("SELECT id FROM meta_word_dict WHERE version=(SELECT max(version) FROM meta_word_dict)", "postgres")` 가 `ReadOnly=true`·`Risk=low` 이고 이유 목록에 "비허용 함수 호출" 이 없다. `=`·`>`·`<`·`<>`·`||`·`+` 등 연산자 뒤 괄호 형태를 표로 돌리는 테스트가 있다.
  2) `sqlsafe.ApplyRowLimit(defaultEmbeddingMaterializationQuery, "postgres", maximumMaterializedDocuments+1)` 가 오류 없이 반환된다 — `internal/legacyapi/embedding_materialization_test.go`(기존 `embeddingQueryPermitted` 검사가 158행에 있는 그 파일)에 이 단정을 한 건 추가해 실제 프로덕션 상수로 end-to-end 확인할 것. **미확인**: 이 한 가지 수정만으로 이 상수가 완전히 통과하는지는 이번 정찰에서 실행 검증하지 못했다(격리 실행이 막혔다). 통과하지 못하면 남은 차단 이유를 그대로 기록하고, 허용목록에 함수를 **추가하지 말고**(보안 정책 변경이다) 그 원인을 별건으로 남길 것.
  3) 게이트가 느슨해지지 않았음을 증명하는 테스트: `sqlsafe_test.go` 의 기존 차단 케이스(21건) 전부 유지 + 연산자 앞에 붙였을 때도 여전히 차단되는 케이스, 즉 `SELECT x=dblink_exec('host=internal','DROP TABLE victim')`, `SELECT (1) || custom_reporting_fn(id) FROM public.events`(440aa9f 가 추가한 케이스), `SELECT dblink_exec ('a','b')`(공백), `SELECT dblink_exec/*c*/('a','b')`(주석) 가 모두 차단된다.
  4) 수정 전 red → 수정 후 green → 새 판정 헬퍼의 반환값을 `true` 로 고정해 되돌리면 다시 red 가 되는 것을 실제로 보일 것(이 저장소의 확립된 절차).

- 건드릴 파일 (프로덕션 1개):
  - `internal/domain/sqlsafe/sqlsafe.go:scanOraclePolicy` — **토큰 종류를 새로 만들지 말 것**(토큰 스트림에 항목이 늘면 `tokens[index-1]`/`tokens[index-2]`/`postgresAliasColumnList`/`postgresCTEColumnList`/`postgresCastTypeViolation` 의 인접 가정이 전부 깨진다). 대신 지금 `start` 를 채우지 않는 두 자리에만 소스 오프셋을 기록한다: 1094행 비인용 word(`start: int32(start)`), 1074행 `oraclePolicyLeftParen`(`start: int32(index)`). 현재 `start` 는 `postgresPolicyString` 토큰만 채우고 있고(1026·1035·1046행), 그 값을 읽는 곳은 `postgresTypeLiteralViolation` 호출부(443행)뿐이므로 다른 종류에 값을 채워도 기존 판정은 변하지 않는다. **확인함**: 이 파일에서 `.start` 를 읽는 곳은 443행 단 한 곳이며(`postgresTypeLiteralViolation` 호출부, `postgresPolicyString` 토큰만 대상) `postgresAliasColumnList`·`fromItemContext`·`postgresCastTypeViolation`·`postgresCTEColumnList` 는 `start` 를 전혀 읽지 않는다.
  - `internal/domain/sqlsafe/sqlsafe.go` — 새 헬퍼 `postgresCallAdjacent(sql string, nameToken, parenToken oraclePolicyToken) bool`: `nameEnd := int(nameToken.start) + len(nameToken.value)` 에서 시작해 앞으로 걸으며 공백·`--` 행 주석·`/* */` 블록 주석(`flatBlockCommentEnd` 재사용, 1157행)만 건너뛰고, 커서가 정확히 `int(parenToken.start)` 에 닿으면 true. 오프셋이 범위를 벗어나거나 주석이 닫히지 않거나 그 밖의 문자를 만나면 **true(=호출로 간주 → 차단)로 fail-closed**. 실제 함수 호출은 이름과 `(` 사이에 trivia 만 올 수 있고, 그 사이에 다른 문자가 있으면 그 word 는 호출 대상이 아니라 연산자의 좌변이다.
  - `internal/domain/sqlsafe/sqlsafe.go:postgresReadOnlyViolations` — 456행 `postgresAliasColumnList` 분기 **뒤**, 459행 허용목록 검사 **앞**에 `if !nameToken.quoted && !postgresCallAdjacent(sql, nameToken, token) { continue }` 한 줄. 인용 식별자는 `start` 를 채우지 않으므로(1055행) 반드시 `!nameToken.quoted` 로 제외해 기존대로 차단되게 둘 것. 419행이 넘기는 `sql` 이 `scanOraclePolicy` 가 스캔한 바로 그 문자열임을 확인할 것(`AnalyzeDialect` 가 정규화한 사본을 넘기면 오프셋이 어긋난다 — 이 전제가 깨지면 과제 전체를 중단하고 차선으로 갈 것).
  - `internal/domain/sqlsafe/sqlsafe_test.go` — 위 수용 기준 1·3 의 표 테스트.
  - `internal/legacyapi/embedding_materialization_test.go` — 수용 기준 2 의 `ApplyRowLimit` 단정 1건(빌드 태그 없는 일반 테스트 파일이다).

- 검증 명령:
  - `go test ./internal/domain/sqlsafe/... -count=1`
  - `go test ./internal/legacyapi/... -count=1`
  - `go test ./... -count=1` (28 패키지 ok 가 기준선)
  - `go vet ./...` 및 `go vet -tags=integration ./...`
  - `make check-go-format` (a432f3f 가 추가한 gofmt 읽기 전용 검사)
  - `go build ./...`
  - 이 게이트를 뷰 정의·파티션 식·RLS 정책 검사에 쓰는 통합 테스트: 폐기 PostgreSQL 17 을 띄우고 `QURIO_TEST_POSTGRES_DSN` 을 주어 `go test -race -p=1 -tags=integration ./internal/domain/dbexec -count=1`. (포트 55432/55433/55442/55452 는 과거 회차에서 점유된 적이 있으니 빈 포트를 골라 쓸 것. dbexec 는 `public` 고정명과 pg_catalog 캐스트를 CREATE/DROP 하므로 전용 폐기 DB·직렬 실행 필수.)

- 위험과 피할 것:
  - **게이트 loosening 이다.** 오탐만 줄이고 차단은 하나도 풀지 않았음을 수용 기준 3 으로 반드시 증명할 것. 허용목록(`postgresPureFunctions`/`postgresParenthesisKeywords`)에 이름을 추가하는 것은 보안 정책 변경이므로 금지(코드 주석 228-231행이 명시).
  - Oracle 경로(`oracleReadOnlyViolations`)는 이번에 손대지 말 것 — 440aa9f 회차에서 확인된 바로 Oracle 쪽은 이 형태를 이미 다르게 처리한다. 방언 한쪽만 고치되 다른 쪽 테스트가 깨지지 않는지 확인.
  - `oracleOuterJoinEnd`(1070행)가 Oracle `(+)` 를 토큰 없이 건너뛰므로, `(` 에 `start` 를 채울 때 그 분기에서는 토큰을 만들지 않는 현재 동작을 유지할 것.
  - `WITH ORDINALITY g(v,i)` 오탐은 이 수정으로 고쳐지지 않는다(`ORDINALITY` 와 `g` 사이가 아니라 `g` 와 `(` 가 인접하므로). 같은 회차에 끌어들이지 말고 별건으로 남길 것 — 과거 회차가 diff 를 키워 반려된 유형이다.
  - 보호 경로(auth/session/migrations/.github/workflows)는 전혀 건드리지 않는다. `internal/webui/dist` 재빌드도 불필요하다(SPA 무관).
  - 오탐 스윕을 할 것: 440aa9f 회차가 쓴 "현실 쿼리 17건" 방식으로 **내 변경으로 새로 막히는 쿼리 0건**을 보이고, 새로 통과하게 된 쿼리는 하나하나 왜 안전한지 설명할 것.

- 차선 후보: `defer pool.Close()` → `t.Cleanup(pool.Close)` 순서 역전을 `internal/legacyapi`(9곳) 한 패키지에만 적용 — 이 base(2128ef7)에 54곳이 남아 있고 `a8f0775`(runtimeapi·intelligenceapi·platformapi 19곳)는 **아직 미머지**이므로 그 세 패키지와는 겹치지 않게 legacyapi 로 한정할 것. 검증 절차(폐기 DB bootstrap → 패키지 실행 → psql 잔존 카운트 → 한 파일 되돌려 인과 확인)는 522f0e5·a8f0775 회차에서 확립돼 있다. 단 "수정 후 잔존 0" 은 수용 기준으로 삼지 말 것(정리 미등록 테스트가 별개로 존재한다 — a8f0775 회차가 실측).
