# kanpic v0.237.0

# v0.237.0

## 외부 호출 크기 상한을 올리면 그 자리에서 통합니다

`WEBSERVICE`·`IMPORTDATA` 가 가져온 답은 `external.cache_seconds` 동안 담아
둡니다. 같은 주소를 여러 칸이 부를 때 남의 서버를 두드리지 않기 위해서입니다.
그런데 담아 두는 자리에 **답만이 아니라 거절도** 들어갔고, 그 자리를 부른
함수와 주소만으로 이름 지었습니다. 그래서 이런 일이 있었습니다.

1. 어떤 칸이 `=IMPORTDATA("https://…/big.csv")` 로 1.5 MB 짜리 표를 부릅니다.
2. `external.max_kb` 가 1024 라서 칸에 **"응답이 허용된 크기를 넘습니다"** 가
   뜹니다.
3. 관리자가 콘솔에서 `external.max_kb` 를 4096 으로 올립니다.
4. 그런데 칸은 그대로 **"응답이 허용된 크기를 넘습니다 (1024 KB)"** 입니다.
   기본값이면 5분, 길게 잡아 두었다면 하루까지 그렇습니다.

관리자에게는 설정이 먹지 않은 것으로 보입니다. 실제로는 옛 상한에서 받은
거절을 캐시가 내주고 있을 뿐인데, 그것을 알 길이 없습니다.

허용 호스트 목록은 이미 캐시 바깥에 있었습니다 — `external.allowed_hosts` 를
고치면 캐시가 살아 있어도 곧바로 통합니다. 크기 상한과 시간 제한도 **부르는
방법을 정하는 정책** 이라는 점에서 다르지 않습니다. 옛 상한에서 받은 답은 새
상한에 대한 답이 아닙니다.

이제 캐시의 자리 이름에 그때의 `external.max_kb` 와
`external.timeout_seconds` 가 함께 들어갑니다.

- 상한을 **올리면** 담아 둔 거절을 내주지 않고 다시 부릅니다.
- 상한을 **내리면** 담아 둔 큰 답을 내주지 않고 다시 부릅니다. 새 상한을
  넘으면 거절합니다.
- 정책이 그대로면 예전과 똑같이 담아 둔 것을 내줍니다. 남의 서버를 두 번
  두드리지 않습니다.

## 확인

- `internal/external/fetcher_test.go`
  - `TestRaisingTheSizeCeilingIsBelievedAtOnce` — 증상 그대로, 상한을 올리고
    내릴 때마다 캐시가 옛 정책의 답을 내주지 않는지, 정책이 그대로면 다시
    부르지 않는지
  - `TestResolveIsSafeToShare` — `Fetcher` 는 함께 써도 안전하다고 적혀
    있으므로, 여덟 갈래가 같은 주소들을 한꺼번에 부르는 길을 `-race` 로
    못 박았습니다

## 오프라인 설치

이 릴리즈는 웹 자산과 서버 바이너리를 포함한 Docker 이미지 아카이브로 제공됩니다.

```bash
sha256sum -c kanpic-v0.237.0.tar.gz.sha256
gzip -dc kanpic-v0.237.0.tar.gz | docker load
docker run --rm -p 8080:8080 \
  -e POSTGRES_DSN='postgres://kanpic:password@postgres.internal:5432/kanpic?sslmode=require' \
  kanpic:v0.237.0
```

- 필수 런타임 환경 변수: `POSTGRES_DSN`
- Redis 및 외부 인터넷 연결: 불필요
- 관리자 초기 로그인을 보호하려면 `BOOTSTRAP_ADMIN_ID`와 `BOOTSTRAP_ADMIN_PASSWORD`를 함께 지정합니다.
- 무결성 검증 파일: `kanpic-v0.237.0.tar.gz.sha256`
