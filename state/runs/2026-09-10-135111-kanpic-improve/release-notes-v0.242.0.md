# kanpic v0.242.0

# v0.242.0

## 윈도우에서 저장한 CSV 를 그대로 올릴 수 있습니다

파워셸에서 `Export-Csv` 로 내보낸 표를, 또는 엑셀에서 "유니코드 텍스트" 로
저장한 표를 가져오기에 끌어다 놓습니다. 그러면 이것이 나왔습니다.

```
CSV must be UTF-8 encoded
```

두 도구는 UTF-16LE 로 씁니다. 그리고 파일 맨 앞에 **그것을 알리는 두 바이트**
(BOM)를 둡니다. 무엇으로 적혔는지 짐작할 것이 하나도 없는 파일입니다 — 파일이
첫 두 바이트에 스스로 적어 두었으니까요. 그런데 가져오기는 그 두 바이트를 보고
"UTF-8 이 아니다" 라고만 판단해 업로드를 통째로 거절했습니다.

거절한 것은 **내용이 아니라 표시** 였습니다. 안에 든 것이 한글이든 영문 열 개
뿐이든 똑같이 거절되었습니다. 화면에는 인코딩을 고르는 자리가 없으니, 사람에게
남은 길은 다른 도구를 열어 저장을 다시 하는 것뿐이었습니다.

이제 파일이 밝힌 대로 읽습니다. 표시를 두고 오는 UTF-8·UTF-16LE·UTF-16BE 는
그대로 올라갑니다.

## 같은 파일은 어느 문으로 들어와도 같은 표입니다

읽는 규칙을 새 `internal/delimited.ToUTF8` 한 곳에 두고, 파일이 들어오는 두 문이
함께 씁니다.

- **올리기** — 파일을 끌어다 놓는 가져오기
- **부르기** — `=IMPORTDATA("https://…/report.csv")`

열을 가르는 규칙이 이미 그 자리에 있습니다(v0.239.0). 무엇으로 적혔는지 읽는
일과 무엇으로 갈렸는지 읽는 일은 같은 판단의 앞뒤이므로 함께 두었습니다.
`parseCSV` 가 따로 떼고 있던 `﻿` 도 그 자리로 합쳤습니다 — 한쪽만 고쳐져
어긋나는 일이 없도록 규칙을 하나로 묶었습니다.

윈도우 도구가 서버에 올려 둔 UTF-16LE CSV 를 `IMPORTDATA` 로 불러도 이제 표가
됩니다. 예전에는 한 열짜리 글자 아닌 덩어리였습니다.

## 짐작은 하지 않습니다

표시가 없는 파일은 손대지 않습니다. 표시가 없으면 UTF-16 인지, 그 바이트를
우연히 담은 멀쩡한 파일인지 가릴 길이 없습니다. **짐작은 멀쩡한 파일을
망가뜨리는 쪽으로만 틀립니다.** 그래서 EUC-KR 처럼 아무것도 밝히지 않는 파일은
예전처럼 거절합니다 — 저장하실 때 UTF-8 을 골라 주세요.

UTF-32LE 도 마찬가지입니다. 이 인코딩은 UTF-16LE 와 **똑같은 두 바이트로**
열립니다(뒤에 0 이 둘 더 붙습니다). 앞만 보고 UTF-16 으로 읽으면 반쯤 읽어
글자 아닌 것을 만들어 냅니다. 우리가 읽지 못하는 파일이니, 표를 흉내 내는 대신
예전처럼 거절합니다.

## 미리보기가 방금 고른 파일보다 작은 수를 말하지 않습니다

UTF-16 을 UTF-8 로 옮기면 한글이 아닌 글자는 대개 절반으로 줄어듭니다. 옮긴
뒤의 크기를 미리보기에 적으면, 탐색기에서 20KB 로 보이던 파일이 대화상자에서는
10KB 라고 나옵니다. 다른 파일을 고른 것처럼 보입니다.

크기는 옮겨 읽기 전에 잽니다. 올리신 파일의 크기 그대로입니다(XLSX 는 이미
그렇게 하고 있었습니다).

## 확인

- `internal/delimited/encoding_test.go`
  - `TestToUTF8ReadsWhatTheFileSaysItIs` — 표시가 있는 UTF-8·UTF-16LE·UTF-16BE 와
    표시가 없는 파일 등 일곱 가지를 읽어 보고, 표시가 떨어졌는지 함께 봅니다
  - `TestToUTF8DoesNotMistakeUTF32ForUTF16` — 같은 두 바이트로 여는 UTF-32LE 를
    반쯤 읽지 않고 그대로 두는지 못 박습니다
  - `TestToUTF8GuessesNothing` — 표시가 없는 파일은 한 바이트도 손대지 않습니다
- `internal/importexport/service_test.go`
  - `TestImportReadsCSVThatSaysItIsUTF16` — 파워셸이 쓰는 그대로, 모든 칸에
    따옴표를 두르고 줄 끝을 CRLF 로 쓴 UTF-16LE CSV 를 올려 봅니다. 미리보기의
    크기가 올린 파일의 크기인지, 영문뿐인 파일도 같은 이유로 거절되던 것이
    풀렸는지 함께 봅니다
- `internal/external/fetcher_test.go`
  - `TestImportDataReadsACSVThatSaysItIsUTF16` — 같은 파일이 부르기로 들어와도
    같은 표가 되는지 봅니다

`gofmt`·`go vet`·`go build`·`go test ./...` 전부와, 인코딩을 만지는
`internal/{external,importexport,delimited}` 는 `-race` 로도 함께 돌렸습니다.

## 올리실 때

서버가 파일을 읽는 규칙만 바뀌었습니다. 화면도, 이미 저장한 문서도 그대로이고
따로 하실 일은 없습니다. 예전에 거절당해 저장을 다시 하셨던 파일이 있다면 이제
원본 그대로 올리셔도 됩니다.

사용자 가이드의 가져오기 항목이 있지도 않은 인코딩 선택 자리(UTF-8·EUC-KR
고르기)를 안내하고 있어, 지금 읽는 대로 고쳤습니다.

## 오프라인 설치

이 릴리즈는 웹 자산과 서버 바이너리를 포함한 Docker 이미지 아카이브로 제공됩니다.

```bash
sha256sum -c kanpic-v0.242.0.tar.gz.sha256
gzip -dc kanpic-v0.242.0.tar.gz | docker load
docker run --rm -p 8080:8080 \
  -e POSTGRES_DSN='postgres://kanpic:password@postgres.internal:5432/kanpic?sslmode=require' \
  kanpic:v0.242.0
```

- 필수 런타임 환경 변수: `POSTGRES_DSN`
- Redis 및 외부 인터넷 연결: 불필요
- 관리자 초기 로그인을 보호하려면 `BOOTSTRAP_ADMIN_ID`와 `BOOTSTRAP_ADMIN_PASSWORD`를 함께 지정합니다.
- 무결성 검증 파일: `kanpic-v0.242.0.tar.gz.sha256`
