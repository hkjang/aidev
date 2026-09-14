# kanpic v0.246.0

# v0.246.0

## 다른 사내 서비스로 문서를 바로 보냅니다

생각이 다른 서비스의 캔버스에서 문서가 되고 슬라이드가 되는 동안, 단계마다
사람이 파일을 **내려받아 다시 올렸습니다.** 형식은 이미 맞았습니다 — 없는
것은 넘기는 손이었습니다. 사내 서비스 간 문서 넘기기 표준(HANDOFF-STANDARD)
에서 kanpic 이 맡은 칸, CSV·XLSX 를 보내고 CSV·XLSX 를 받는 쪽을 만들었습니다.

편집기의 **파일 › 다른 서비스로 보내기** 를 고르면 받는 서비스의 `/handoff`
가 새 창에서 열리고, 그쪽이 kanpic 에서 직접 받아 갑니다. 반대로 다른
서비스가 kanpic 으로 보내면 새 워크북이 만들어지고 그 워크북으로 넘어갑니다.
로그인이 필요한 설치는 `/login?return_to=` 로 보냈다가 같은 주소로 돌아옵니다.

## 서비스끼리 자격 증명을 나누지 않습니다

넘기는 것은 **한 번만 쓸 수 있는 5분짜리 표(claim)** 입니다.
`POST /api/v1/handoff/claims` 가 발급하는 표는 그 사용자가 읽을 수 있는
(내보내기가 허용된) 그 문서 하나에만 묶입니다 — 남의 문서나 복사 제한
뷰어에게는 만들어지지 않습니다. 문서는 표를 만들 때 미리 뽑아 함께 두므로
표에 적은 크기가 실제로 내줄 것과 같습니다. `GET /api/v1/handoff/claims/{claim}`
은 로그인 없이 표만으로 내주며 **꺼내면서 지웁니다** — 이미 쓴 표든 지난
표든 없는 표든 같은 404 입니다. 표 원문은 데이터베이스에 해시로만 두고 요청
로그에는 `{claim}` 으로 가립니다.

## 받는 곳은 관리자가 적은 곳뿐입니다

`source` 는 밖에서 들어오는 값입니다. 그대로 받아 오면 kanpic 이 사내 아무
주소나 대신 긁어 오는 도구가 됩니다. 그래서 관리자가 `handoff.peers` 에
`서비스이름=오리진` 꼴로 적은 오리진과 **정확히 같은 곳에서만** 받고, 없으면
아무것도 요청하지 않고 거절합니다. 리다이렉트를 따라가지 않고, 25MB 를 넘으면
중간에 끊고, 30초를 넘기면 포기하고, `Content-Type` 이 CSV·XLSX 가 아니면
버립니다. 거절은 사람이 읽는 화면으로 까닭을 보여 줍니다.

허용 목록의 기본값은 **비어 있습니다.** 그때는 보내기 메뉴가 보이지 않고 아무
데서도 받지 않으므로, 새로 설치한 곳도 그냥 올리신 곳도 아무것도 달라지지
않습니다. 보낼 곳은 그 서비스가 kanpic 이 보내는 형식을 받을 때만 메뉴에
보입니다. 받은 워크북은 어디서 왔는지가 남아
`GET /api/v1/workbooks/{id}/handoff` 로 물어볼 수 있습니다. 설정 방법은
관리자 가이드의 **다른 서비스와 문서 주고받기** 절에 있고 PDF 도 다시 구웠습니다.

## 소문자로 바꾸다 자리가 밀리던 것을 잡았습니다

분석 스니펫에 nonce 를 끼워 넣고 `http`·`<script>` 를 찾는 자리가
`strings.ToLower` 로 만든 문자열의 바이트 위치를 원문에 그대로 썼습니다.
켈빈 기호(U+212A)처럼 소문자가 되면 길이가 달라지는 글자가 앞에 하나만 있어도
그 뒤의 모든 자리가 어긋나 `<sc nonce="n">ript>` 같은 것이 나왔고, 스니펫은
돌지 않았습니다. 찾는 것은 모두 ASCII 이므로 ASCII 만 접습니다.

## 이미지에 어느 커밋인지 적혀 있습니다

떠 있는 이미지만 보고는 어느 버전·어느 커밋으로 만든 것인지 알 수
없었습니다. 이제 `docker image inspect` 로
`org.opencontainers.image.{version,revision,created,source}` 를 볼 수 있습니다.

## 확인

- `internal/handoff/handoff_test.go` — 허용 목록 읽기, 보낼 곳 고르기, 오리진
  정확 일치, 단일 사용과 만료, 목록 밖 `source` 는 요청 0건,
  리다이렉트·크기·형식 거절, 시간 초과, 파일 이름
- `internal/httpapi/handoff_test.go` — 발급 → 내주기 → 두 번째 404 와 로그에
  표 없음, 만료, 남의 문서 403, `targets` 가 허용 목록을 따름, 두 서버 사이
  실제 보내기 → 받기 → 출처 확인
- `internal/integration/handoff_test.go` — `TestPostgresHandoffStore`, 실제
  PostgreSQL 에 마이그레이션 041 적용
- `internal/analytics/analytics_fold_test.go` — 길이가 변하는 글자 뒤에서도
  nonce 가 제자리에 들어감
- `web/src/lib/handoff.test.ts` — 창을 먼저 열고 표를 받은 뒤 받는 쪽으로 보냄

`gofmt`·`go vet`·`go build`·`go test ./...`(전체 통과),
`npm run lint`·`npm test`(70 파일 496개 통과)·`npm run build`,
`scripts/check-release-docs.sh`, `scripts/check-commit-identities.sh` 를
돌렸습니다.

## 올리실 때

마이그레이션 **041**(`handoff_claims`·`handoff_receipts`)이 서버가 뜰 때
자동으로 적용됩니다. 필수 런타임 환경 변수는 계속 `POSTGRES_DSN` 하나입니다.
문서를 주고받으려면 관리자가 **설정** 의 `handoff.peers` 에 상대 서비스를
적어야 하고, 리버스 프록시 뒤라면 `server.public_url` 이 프록시 바깥에서 닿는
주소여야 상대가 표를 받으러 올 수 있습니다. 적지 않으면 v0.245.0 과 같게
움직입니다. v0.245.0 에서 올라오시는 분은 이미지만 바꿔 다시 띄우시면 됩니다.

## 오프라인 설치

이 릴리즈는 웹 자산과 서버 바이너리를 포함한 Docker 이미지 아카이브로 제공됩니다.

```bash
sha256sum -c kanpic-v0.246.0.tar.gz.sha256
gzip -dc kanpic-v0.246.0.tar.gz | docker load
docker run --rm -p 8080:8080 \
  -e POSTGRES_DSN='postgres://kanpic:password@postgres.internal:5432/kanpic?sslmode=require' \
  kanpic:v0.246.0
```

- 필수 런타임 환경 변수: `POSTGRES_DSN`
- Redis 및 외부 인터넷 연결: 불필요
- 관리자 초기 로그인을 보호하려면 `BOOTSTRAP_ADMIN_ID`와 `BOOTSTRAP_ADMIN_PASSWORD`를 함께 지정합니다.
- 무결성 검증 파일: `kanpic-v0.246.0.tar.gz.sha256`
