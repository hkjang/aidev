# kanpic v0.251.0

# v0.251.0

## IMPORTDATA 가 업로드 가져오기와 같은 자로 번호를 글자로 둡니다

업로드 가져오기는 CSV 의 `00123`(우편번호·사번)과 스무 자리 계좌번호를 **글자**로
둡니다. 수로 담으면 `123` 과 `1.2345678901234567e+19` 가 되어 파일에 적힌 것과
달라지고 되돌릴 길이 없기 때문입니다(v0.192.0). 그런데 같은 CSV 를 URL 로
불러오는 `IMPORTDATA` 는 수식 엔진의 자(`formula.DecimalNumber`)만 써서 그 두
가드가 없었습니다. 같은 파일이 **어느 문으로 들어오느냐**에 따라 다른 표가
되었고, IMPORTDATA 로 받은 명단의 사번은 앞자리 0 이 사라진 채 VLOOKUP 에서
빠졌습니다.

이제 파일에서 온 값을 칸에 담는 자는 `delimited.Number` 하나입니다 — 수식
엔진의 자에 앞자리 0 과 열여섯 자리 넘는 번호 두 가드를 얹은 것으로, 업로드
`parseScalar` 와 `IMPORTDATA` 의 `parseCSV` 가 같은 함수를 부릅니다. 업로드
쪽의 규칙은 그대로이고(함수를 `importexport` 에서 `delimited` 로 옮겼을 뿐)
`IMPORTDATA` 만 그 규칙을 따르게 됩니다: `"00123"`·`"007"`·스무 자리 번호·
`"1234567890123456.5"` 는 글자, `"0"`·`"0.5"`·`"-0"`·`"1e30"` 은 지금까지처럼
수입니다.

셈의 자는 건드리지 않았습니다. `=SUM` 에 글자 `"00123"` 이 들어오면 여전히
123 으로 셉니다 — 그것은 파일의 규칙이 아니라 셈의 규칙이고 엑셀도 그렇습니다.

## XLSX 의 형식 없는 칸은 예전 그대로입니다

위 정리 중간에 XLSX 의 형식 없는 칸에도 열여섯 자리 한도를 대었다가
되돌렸습니다. 엑셀의 raw 값은 사람이 적은 글자가 아니라 이미 실수이고, 이진
오차가 있는 값을 되돌릴 수 있게 17자리로 적습니다(`2.2` →
`2.2000000000000002`, `37.02` → `37.019999999999996` — 계산 결과를 값으로 붙여
넣은 칸이면 흔합니다). 거기에 그 한도를 대면 수 칸이 글자가 되어 가져온 열의
SUM 이 조용히 0 이 됩니다. 그래서 그 칸은 지금까지처럼 **앞자리 0 만** 거르고,
엑셀이 수라고 밝힌 칸(`CellTypeNumber`)도 예전대로 수입니다. 이 릴리즈로 XLSX
가져오기의 결과가 바뀌는 칸은 없습니다.

## 확인

- `internal/delimited/number_test.go` — 표 30가지로 어떤 글자가 수가 되고 어떤
  것이 글자로 남는지 못 박습니다.
- `internal/external/fetcher_test.go` — 실제 `httptest` 서버에서 받은 CSV 를
  `IMPORTDATA` 로 읽은 것과 같은 바이트를 `importexport.Parse` 로 읽은 것이
  칸마다 같은 타입·같은 값인지 비교합니다. 고치기 전 코드에서 실제로
  `"00123"`→`123` 으로 깨지는 것을 확인했습니다.
- `internal/importexport/scalar_test.go` — 형식 없는 XLSX 칸이 17자리 실수를
  수로, `"00123"` 을 글자로 두는 것을 못 박습니다.

`gofmt`·`go vet`·`go build ./...`·`go test ./...`(전체 통과),
`scripts/check-release-docs.sh`, `scripts/check-commit-identities.sh` 를
돌렸습니다. 웹과 문서는 손대지 않았습니다.

## 올리실 때

서버 안쪽의 CSV 파싱 규칙 하나만 바뀌었습니다. 마이그레이션과 새 설정은 없고
필수 런타임 환경 변수는 계속 `POSTGRES_DSN` 하나입니다. `IMPORTDATA` 의 답
캐시는 서버 메모리에만 있으므로 새 이미지로 다시 띄우는 순간 비고, 이미 열려
있던 워크북의 `IMPORTDATA` 도 다음 계산부터 새 규칙으로 읽습니다. v0.250.0
에서 올라오시는 분은 이미지만 바꿔 다시 띄우시면 됩니다.

## 오프라인 설치

이 릴리즈는 웹 자산과 서버 바이너리를 포함한 Docker 이미지 아카이브로 제공됩니다.

```bash
sha256sum -c kanpic-v0.251.0.tar.gz.sha256
gzip -dc kanpic-v0.251.0.tar.gz | docker load
docker run --rm -p 8080:8080 \
  -e POSTGRES_DSN='postgres://kanpic:password@postgres.internal:5432/kanpic?sslmode=require' \
  kanpic:v0.251.0
```

- 필수 런타임 환경 변수: `POSTGRES_DSN`
- Redis 및 외부 인터넷 연결: 불필요
- 관리자 초기 로그인을 보호하려면 `BOOTSTRAP_ADMIN_ID`와 `BOOTSTRAP_ADMIN_PASSWORD`를 함께 지정합니다.
- 무결성 검증 파일: `kanpic-v0.251.0.tar.gz.sha256`
