# JASQL v0.31.2

버그픽스 패치 릴리즈입니다. **한국어 토크나이저의 접미사 순서 하나**를 고쳤고,
그 외 기능·API·설정은 v0.31.1과 같습니다.

## "…별로" 와 "…별" 이 같은 토큰이 됩니다

`stripKoreanSuffix`는 접미사 목록을 앞에서부터 훑다가 **처음 일치하는 것에서
바로 반환**합니다. 그런데 `"로"`가 `"별로"`보다 앞에 있었습니다. `"별로"`로 끝나는
문자열은 전부 `"로"`로도 끝나고 `"로"` 쪽 길이 가드가 더 느슨하므로, `"별로"`
분기는 도달하지 못하는 죽은 코드였습니다.

- 고치기 전 — `"회원사별"` → `"회원사"`, `"회원사별로"` → `"회원사별"`
- 고친 뒤 — `"회원사별"` → `"회원사"`, `"회원사별로"` → `"회원사"`

`tokenize`는 검색·조인 계획·지표 조회·분석·확인질문·피드백·검증이 공유하는
1차 토크나이저입니다. 그래서 같은 뜻을 두 가지로 쓴 질문이 서로 다른 토큰으로
갈렸고, 오류 없이 **조용히 재현율만 깎고 있었습니다**.

바꾼 것은 목록에서 `"별로"`를 `"로"` 앞으로 옮긴 것뿐입니다. 길이 가드와 반환
구조, 나머지 접미사 순서는 그대로입니다. `"별로"` 가드가 3글자 초과를 요구하므로
짧은 입력은 영향이 없습니다 — `"월별로"`는 여전히 `"로"`로 떨어져 `"월별"`입니다.

## 테스트

`internal/catalog/tokenize_test.go`를 추가했습니다. `tokenize`와
`stripKoreanSuffix`를 직접 짚는 첫 테스트로, …별로/…별 동치성, 짧은 토큰 가드,
그리고 구분자 분해·소문자화·중복 제거의 현재 동작 고정 케이스를 담았습니다.

## 골든 평가 — 변경 없음

| 지표 | 값 |
| --- | --- |
| cases | 80 |
| table_selection_acc | 0.94 |
| column_recall_avg | 0.9 |
| join_path_acc | 0.94 |
| metric_lookup_acc | 1 |
| expected_sql_valid | 1 |

`go build ./...`, `go vet ./...`, `go test ./...` 모두 통과합니다.

## 업그레이드

이미지 교체뿐입니다. 메타 DB 스키마 변경도, 설정 변경도 없습니다. 볼륨과 메타
DB는 그대로 둡니다. v0.31.1로의 롤백도 같은 방식으로 안전합니다.

```sh
docker load -i jasql-mcp-0.31.2-docker.tar.gz
docker rm -f jasql-mcp
docker run -d --name jasql-mcp --restart unless-stopped \
  -p 8787:8787 -v jasql-data:/app/data/kcb \
  jasql-mcp:v0.31.2
```

## 자산 검증

반입 전에 크기와 해시를 대조하세요. 잘려서 올라간 자산은 폐쇄망의
`docker load` 단계에 가서야 드러납니다.

| 파일 | 크기(bytes) | SHA256 |
| --- | --- | --- |
| `jasql-mcp-0.31.2-docker.tar.gz` | 12804111 | `3307a75e72636cd88329fe6ec217573747747472dad2f60b1a13ca9c518ef1cc` |

나머지 자산의 해시는 `SHA256SUMS-v0.31.2.txt`에 있습니다.

> **이번 릴리즈에는 Oracle Instant Client 내장 이미지
> (`jasql-mcp-v0.31.2.tar.gz`)가 없습니다.** 빌드 머신에 `instantclient/`가
> 준비되지 않아 `scripts/release-image.sh`를 돌릴 수 없었습니다. 폐쇄망에서
> Oracle에 실제로 접속해야 한다면 `Dockerfile.oracle` 머리말을 따라
> Instant Client를 준비한 뒤 해당 스크립트로 직접 빌드하세요. 첨부된
> `jasql-mcp-0.31.2-docker.tar.gz`는 순수 Go 이미지로, 카탈로그·검색·SQL
> 생성·검증은 모두 동작하지만 Oracle 실행은 되지 않습니다.
