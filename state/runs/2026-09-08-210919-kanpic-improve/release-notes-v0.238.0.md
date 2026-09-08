# kanpic v0.238.0

# v0.238.0

## 원격이 다시 멀쩡해지면 칸도 곧바로 따라옵니다

`WEBSERVICE`·`IMPORTDATA` 가 가져온 답은 `external.cache_seconds` 동안 담아
둡니다. 그런데 담아 두는 자리에는 **답만이 아니라 실패도** 들어갔고, 둘을
같은 시간만큼 붙들었습니다. 그래서 이런 일이 있었습니다.

1. 어떤 칸이 `=WEBSERVICE("https://api.example.com/…")` 로 값을 부릅니다.
2. 마침 저쪽이 배포 중이라 **`주소가 503 로 답했습니다`** 가 뜹니다.
3. 1분 뒤 배포가 끝나고 원격은 멀쩡해집니다.
4. 그런데 칸은 그대로 **`주소가 503 로 답했습니다`** 입니다. 기본값이면 5분,
   길게 잡아 두었다면 **하루** 까지 그렇습니다.

한 번의 시간 초과도 같았습니다. 저쪽이 잠깐 느렸다는 사실이 하루치 답으로
굳었습니다.

앞선 릴리즈에서 허용 호스트와 크기 상한·시간 제한은 캐시 바깥으로 나왔습니다
— 관리자가 고치면 그 자리에서 통합니다. 정작 **이쪽에서 고칠 수 없는 저쪽
사정** 만 가장 오래 남아 있던 셈입니다.

이제 실패를 두 가지로 나눠 봅니다.

- **원격이 책임지는 실패** 는 최대 **30초** 만 담습니다. 오류 상태로
  답했거나(`주소가 503 로 답했습니다`), 아예 닿지 못했거나(시간 초과·연결
  실패), 응답을 읽다 끊긴 것입니다. 저쪽이 고쳐지면 곧 통합니다.
- **이 응답의 성질이 정한 거절** 은 예전 그대로 `external.cache_seconds`
  동안 담습니다. 크기 상한을 넘음, 리다이렉트, 사설망 주소입니다. 같은
  응답이면 언제나 같은 답이므로 다시 물어봐야 할 까닭이 없습니다.

`external.cache_seconds` 가 30초보다 짧으면 그 값을 그대로 씁니다. 실패를
성공보다 오래 붙들지 않습니다.

아예 담지 않는 쪽은 고르지 않았습니다. 닿지 못하는 주소를 부르는 칸이
여럿이면 다시 계산할 때마다 `external.timeout_seconds` 를 처음부터 새로
기다리게 됩니다. 30초는 그 값이 통째로 되풀이되는 것을 막으면서, 원격이
돌아온 것을 알아채기에는 충분히 짧습니다.

## 확인

- `internal/external/fetcher_test.go`
  - `TestRemoteFailuresAreKeptOnlyBriefly` — 증상 그대로, 503 이 30초 안에는
    담긴 채로 나오고 그 뒤에는 다시 부르는지, 같은 시험에서 크기 상한 거절은
    `cache_seconds` 내내 담긴 채로 있는지 두 갈래를 갈라 봅니다
- `go test -race ./internal/external/` 로 캐시를 함께 쓰는 길을 다시 못
  박았습니다.

## 오프라인 설치

이 릴리즈는 웹 자산과 서버 바이너리를 포함한 Docker 이미지 아카이브로 제공됩니다.

```bash
sha256sum -c kanpic-v0.238.0.tar.gz.sha256
gzip -dc kanpic-v0.238.0.tar.gz | docker load
docker run --rm -p 8080:8080 \
  -e POSTGRES_DSN='postgres://kanpic:password@postgres.internal:5432/kanpic?sslmode=require' \
  kanpic:v0.238.0
```

- 필수 런타임 환경 변수: `POSTGRES_DSN`
- Redis 및 외부 인터넷 연결: 불필요
- 관리자 초기 로그인을 보호하려면 `BOOTSTRAP_ADMIN_ID`와 `BOOTSTRAP_ADMIN_PASSWORD`를 함께 지정합니다.
- 무결성 검증 파일: `kanpic-v0.238.0.tar.gz.sha256`
