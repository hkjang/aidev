한글 문서에 없는 것 셋 — **인용문, 코드 블록, 구분선** — 을 한글이 쓰는 방법으로 적고, 그 방법대로 다시 읽습니다. 두 번 오간 문서에서 사라지던 자리들입니다.

## 추가

### 인용문과 코드 블록 (`.hwpx`)

한글에는 인용문도 코드 블록도 없습니다. 쓰는 쪽은 인용을 들여쓴 문단으로, 코드를 고정폭 문단 여러 개로 흘려보내고 있었고 — 읽는 쪽에는 그것을 되살릴 단서가 없었습니다. **들여쓰고 고정폭으로 그린 문단은 읽는 쪽에 그저 문단입니다.**

그래서 `.docx` 쓰기가 하는 것과 같이 머리에 스타일 둘을 심고(`인용`/`Quote`, `코드`/`Code`) 문단이 그것을 번호로 가리킵니다. 스타일 이름을 무엇으로 읽을지는 한 곳에 두어 워드에서 옮겨온 이름(`Source Code` 따위)도 함께 받습니다.

코드 블록은 한 문단 안에 줄바꿈으로 넣습니다. 줄마다 문단이면 끝을 알릴 것이 없어 **나란한 코드 블록 둘이 하나로 돌아옵니다.** 인용문은 목록처럼 시작도 끝도 표시가 없으므로 이어지는 줄을 하나로 모읍니다. 인용이 여백에서 한 칸 물러난 것은 인용을 그리는 방법이지 글쓴이가 준 들여쓰기가 아니므로, 읽을 때 덜어냅니다 — 그러지 않으면 왕복마다 한 칸씩 밀립니다.

### 구분선 (`.hwp` 읽기, `.hwpx` 양방향)

구분선은 muni 가 `.docx`·HTML·마크다운에서 예전부터 주고받던 블록인데, 한글 두 형식은 읽는 쪽이 아예 없었고 쓰는 쪽은 `─` 서른 개를 **글자로** 적고 있었습니다. 글자는 글자로 돌아오므로, 두 번 오간 문서에는 선이 있던 자리에 그 문단이 그대로 남았습니다.

한글에도 구분선은 없습니다. 한글이 하이픈 한 줄을 그리는 방법 — 아무것도 담지 않은 문단 아래에 선 하나를 긋는 것 — 이 곧 muni 가 쓰는 방법입니다. 선은 문단에 있지 않고 문단은 머리의 `borderFill` 을 번호로 가리키기만 하며(칸 음영과 같은 간접입니다. `.hwp` 는 `PARA_SHAPE` 의 32번째 바이트, `.hwpx` 는 `paraPr` 의 `<hh:border>`), **네 면에 두른 선은 글을 감싼 상자이지 구분선이 아니므로** 위나 아래에만 그은 것을 읽습니다. 선이 그어진 문단이 글자를 지녔거나 표를 매달고 있으면 그 선은 그것을 그리는 방법이므로 구분선으로 세지 않습니다.

무엇이 구분선인지는 한 곳의 판단이라 두 리더와 쓰는 쪽이 어긋나지 않습니다.

## 그 밖에

`gofmt` 과 `go vet` 을 이제 손이 아니라 CI 가 돌립니다. 여덟 회차 내리 손으로 돌린 검사입니다. **습관에 들어 있는 검사는 무언가를 잡아낼 바로 그 한 번에 빠집니다.** 이제 모든 푸시와 PR 에서 `gofmt -l` 과 `go vet ./...` 이 함께 돕니다.

## 업그레이드

마이그레이션과 설정 변경은 필요하지 않습니다.

```bash
gzip -dc muni-v0.36.0.tar.gz | docker load
docker compose -f compose.example.yaml --env-file .env up -d
```

**인용문이나 코드 블록, 구분선이 든 문서를 한글로 내보내셨다면 다시 내보내 보세요.** 예전에 내보낸 `.hwpx` 를 다시 가져오면 인용은 들여쓴 문단으로, 구분선은 `─` 가 늘어선 문단으로 들어옵니다.

## 오프라인 설치

릴리스 asset 에는 `muni:v0.36.0` Docker 이미지가 포함되어 있습니다.

```bash
gzip -dc muni-v0.36.0.tar.gz | docker load
docker image inspect muni:v0.36.0

cp .env.example .env
# .env의 네 값을 운영 환경에 맞게 변경합니다.
docker compose -f compose.example.yaml --env-file .env up -d
```

애플리케이션이 반드시 필요로 하는 런타임 환경변수는 다음 네 개입니다.

| 환경변수                   | 설명                                 |
| -------------------------- | ------------------------------------ |
| `POSTGRES_DSN`             | PostgreSQL 접속 문자열               |
| `BOOTSTRAP_ADMIN`          | 최초 관리자 아이디 또는 이메일       |
| `BOOTSTRAP_ADMIN_PASSWORD` | 최초 관리자 비밀번호(12자 이상)      |
| `ENCRYPTION_KEY`           | base64로 인코딩한 32-byte master key |

PDF 변환 동작만 조정하는 선택 환경변수가 두 개 있습니다.

| 환경변수               | 기본값    | 설명                                       |
| ---------------------- | --------- | ------------------------------------------ |
| `MUNI_CHROMIUM_PATH`   | 자동 탐색 | PDF Export에 사용할 headless 브라우저 경로 |
| `MUNI_PDF_CONCURRENCY` | `2`       | 동시에 실행할 Chromium 프로세스 수(1~32)   |

## 운영 정보

- 서비스 포트: `8080`
- Liveness: `/healthz`
- Readiness: `/readyz`
- REST API: `/api/v1`
- 공개 링크: `/s/{token}` — 인증 없이 열리는 유일한 화면입니다
- OpenAPI: `/api/openapi.yaml` — 실제 라우트와 일치하며, 테스트가 그것을 지킵니다
- Prometheus: `/metrics` — 관리자 인증 필요
- MCP: `/mcp`
- 지원 DB: PostgreSQL 15 이상
- 이미지에는 PDF Export용 Chromium과 Noto CJK 글꼴이 포함되어 있습니다.

## 릴리스 파일 검증

- 파일: `muni-v0.36.0.tar.gz`
- 크기: (릴리스 후 기록)
- SHA-256: (릴리스 후 기록)
- 내부 이미지 태그: `muni:v0.36.0`

```bash
sha256sum muni-v0.36.0.tar.gz
```

GitHub Actions가 이미지 빌드, archive 생성, 내부 이미지 태그 검증을 완료한 뒤 이 asset을 게시했습니다.

## 문서

- [설치 및 운영 안내](https://github.com/hkjang/muni#readme)
- [운영 안내](https://github.com/hkjang/muni/blob/v0.36.0/docs/OPERATIONS.md)
- [아키텍처](https://github.com/hkjang/muni/blob/v0.36.0/docs/ARCHITECTURE.md)
- [MCP 사용법](https://github.com/hkjang/muni/blob/v0.36.0/docs/MCP.md)
- [전체 변경 내역](https://github.com/hkjang/muni/compare/v0.35.0...v0.36.0)
