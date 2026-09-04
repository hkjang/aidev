# kanpic v0.236.0

# v0.236.0

## "NaN" 한 칸 때문에 파일이 통째로 들어오지 못하던 것을 고쳤습니다

pandas 나 R 로 자료를 다듬어 CSV 로 내보내면 빠진 값이 **`NaN`**, 무한대가
**`Inf`**·**`-Inf`** 라는 글자로 적혀 나옵니다. 흔한 일입니다. 그런데 그런
파일을 kanpic 에 올리면 가져오기가 통째로 실패하고, 화면에는

```
json: unsupported value: NaN
```

이라는 말만 떴습니다. 몇 번째 줄 몇 번째 칸이 문제인지도, 무엇을 고쳐야
하는지도 적혀 있지 않았습니다.

까닭은 값을 수로 읽는 자가 두 벌이었기 때문입니다. 수식 엔진과 격자는
**스프레드시트가 수라고 부르는 것** 만 수로 읽는데, 가져오기만 **Go 가 수라고
부르는 것** 을 그대로 받았습니다. Go 는 `NaN`·`Inf`·`Infinity` 를 물론이고
`1_000`(밑줄로 자리를 가른 것)과 `0x1p4`(16진 실수)까지 수로 읽습니다.

이제 파일을 읽어 들이는 자리도 엔진과 같은 자를 씁니다.

- `NaN`·`Inf`·`Infinity` 는 **글자로** 들어옵니다. 파일은 통째로 들어오고,
  그 칸에는 파일에 적힌 그대로가 남습니다.
- `1_000` 은 조용히 `1000` 이 되지 않습니다. 예전에는 밑줄이 사라진 값이
  칸에 남아, 파일에 적힌 것과 다른 것을 보게 되었습니다.
- `0x1p4` 도 `16` 이 되지 않습니다.
- 평범한 수는 그대로 수입니다 — `1000`, `-2.5`, `1e3`, `.5`, `5.` 입니다.

CSV·TSV 와 XLSX 가 모두 같습니다. `IMPORTDATA` 로 가져오는 CSV 도 같은 자를
씁니다.

`NaN` 이 담긴 칸은 글자이므로 `=SUM` 이 조용히 빼고 셈합니다. 그것은 예전부터
그랬고 맞는 동작입니다 — 빠진 값을 0 으로 세면 평균이 달라집니다.

## 확인

- `internal/importexport/scalar_test.go`
  - `TestParseScalarKeepsGoOnlyNumbersAsText` — Go 만 수로 읽는 열 가지가
    글자로 남고 JSON 으로 적히는지
  - `TestOneExportedNaNDoesNotStopTheWholeImport` — 증상 그대로, `NaN` 과
    `inf` 가 섞인 CSV 가 통째로 들어오는지
  - `TestParseXLSXValueKeepsGoOnlyNumbersAsText` — XLSX 의 형식 없는 칸과
    숫자 칸도 같은 자를 쓰는지
- 수로 읽는 자는 이제 `internal/formula` 의 `DecimalNumber` 한 곳입니다.
  `internal/external` 이 따로 들고 있던 같은 규칙을 지웠습니다.
  `testdata/numeric-text.json` 이 엔진과 격자를 함께 붙들어 둡니다.

## 오프라인 설치

이 릴리즈는 웹 자산과 서버 바이너리를 포함한 Docker 이미지 아카이브로 제공됩니다.

```bash
sha256sum -c kanpic-v0.236.0.tar.gz.sha256
gzip -dc kanpic-v0.236.0.tar.gz | docker load
docker run --rm -p 8080:8080 \
  -e POSTGRES_DSN='postgres://kanpic:password@postgres.internal:5432/kanpic?sslmode=require' \
  kanpic:v0.236.0
```

- 필수 런타임 환경 변수: `POSTGRES_DSN`
- Redis 및 외부 인터넷 연결: 불필요
- 관리자 초기 로그인을 보호하려면 `BOOTSTRAP_ADMIN_ID`와 `BOOTSTRAP_ADMIN_PASSWORD`를 함께 지정합니다.
- 무결성 검증 파일: `kanpic-v0.236.0.tar.gz.sha256`
