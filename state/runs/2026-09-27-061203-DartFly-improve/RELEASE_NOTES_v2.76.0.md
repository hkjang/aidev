내려받은 CSV 와 공개 API 의 CSV 응답에서 값 셀만 받던 수식 방지를 헤더 행에도 적용했습니다. Excel 은 첫 줄도 다른 셀과 똑같이 읽습니다.

**값 칸은 지키면서 컬럼 이름은 그대로 내보냈습니다.**

DartFly 는 내려받는 CSV 가 Excel 에서 열린다는 전제로 UTF-8 BOM 까지 붙여 보내고, 그래서 `=`·`+`·`-`·`@` 로 시작하는 **값**에는 이미 작은따옴표를 붙여 수식으로 읽히지 않게 하고 있었습니다. 그런데 첫 줄, 컬럼 이름은 예외였습니다.

컬럼 이름은 조회 대상 DB 가 정하는 것이 아니라 쓰는 사람의 SELECT 절이 정합니다. 별칭을 붙이지 않으면 DBMS 가 식을 그대로 이름으로 쓰기 때문에, 악의가 없어도 수식으로 시작하는 이름이 흔히 나옵니다 — `SELECT @@version` 은 `@@version`, `SELECT -amount FROM sales` 는 `-amount` 입니다. Excel 에서 `@@version` 열은 `#NAME?` 로 바뀌어 어떤 열인지 알 수 없게 되고, `=` 로 시작하는 이름이라면 값 셀에서 막아 둔 것과 같은 수식 주입이 됩니다.

같은 CSV 를 내려받기(`/api/v1/query/download`)와 API Hub 의 CSV 응답이 각자 따로 만들고 있었습니다. 한쪽만 고치면 다시 어긋나므로 `writeResultCSV` 하나로 합치고, 헤더에 기존 `csvSafeCell` 을 그대로 적용했습니다. 값 셀 규칙은 건드리지 않았습니다.

| 상황 | 이후 동작 |
| --- | --- |
| `SELECT @@version` 의 헤더 | `'@@version` — Excel 에서 `@@version` 으로 보임(`#NAME?` 아님) |
| `SELECT -amount ...` 의 헤더 | `'-amount` |
| 평범한 컬럼 이름 (`id`, `name`) | 그대로 |
| 값 셀의 숫자 `-1` | 그대로 `-1` (숫자에는 붙이지 않음) |
| 앞에 공백을 끼운 이름 (`  =1+1`) | 앞 공백을 건너뛰고 판단해 막음 |
| `DARTFLY_CSV_FORMULA_GUARD=off` | 값 셀과 헤더가 함께 꺼짐 |
| XML·JSON·HTML·마크다운 출력 | 변화 없음 |

CSV 를 스크립트로 읽어 컬럼 이름으로 값을 찾는 쪽이 있다면, `=+-@` 로 시작하는 이름만 앞에 작은따옴표가 붙습니다. 값 셀에서 이미 받아들인 것과 같은 맞바꿈이고, 필요하면 `DARTFLY_CSV_FORMULA_GUARD=off` 로 함께 끌 수 있습니다.

**검증.** 새 `internal/server/csvheader_test.go` 4개(구현 전 실패 확인 후 통과)를 포함해 `go test -race ./...` 38개 패키지, `go vet ./...`, `gofmt -l .`, `go build ./cmd/dartfly` 가 모두 통과했습니다. 여기에 실제 바이너리와 실제 MariaDB 컨테이너를 띄워 메타 DB 를 대상 DB 로 등록하고 `SELECT @@version, -1, +2` 를 두 경로(내려받기 · API Hub 엔드포인트 공개 후 tdb 키 호출)로 모두 받아, 헤더 행이 고치기 전 `@@version,-1,2` 에서 `'@@version,'-1,2` 로 바뀌고 값 행은 `11.4.13-MariaDB-ubu2404,-1,2` 그대로인 것을 확인했습니다. 릴리스 이미지는 `deploy/build-release.sh` 가 `test/smoke/artifact.sh` 로 기동 점검(healthz/readyz, 임베드 정적 자산, 로그인, 내장 마스터 키 질의, TZ)까지 마친 것입니다.

오프라인 배포: `gunzip -c dartfly-v2.76.0.tar.gz | docker load`
