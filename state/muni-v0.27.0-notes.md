한글 오피스가 쓴 `.hwpx` 에서 음영을 칠한 표는 예외 없이 흰 표로 들어왔습니다. 같은 표가 `.hwp` 나 `.docx` 로 오면 색이 남는데도 그랬습니다 — 리더가 색을 칸 안에서 찾고 있었는데, 칸 안에는 애초에 색이 없기 때문입니다.

## 고침

### `.hwpx` — 칸 음영은 칸에 없습니다

칸은 머리(`Contents/header.xml`)의 **`borderFill` 을 번호로 가리키기만** 합니다. 런의 서식이 `charPr` 번호를 거치는 것과 같은 간접입니다. 리더는 `<hp:tc>` 안에서 `fillBrush` 를 찾고 있었는데, 공개된 실제 파일 넷에서 칸 325개 중 칸 안에 `fillBrush` 를 둔 것은 **0개** 였습니다. 이제 머리를 읽을 때 `borderFill` 의 채우기도 함께 담아 두고, 칸이 부르는 번호로 찾습니다.

### `.hwpx` 로 씁니다

쓰는 쪽은 **색마다 `borderFill` 을 하나씩** 머리에 만들고(표의 선은 그대로 두고 안쪽만 그 색으로) 칸이 그 번호를 부릅니다. 같은 색을 쓴 칸들은 하나를 같이 부르므로, 머리글 행이 열 열두 개여도 `borderFill` 은 하나만 늘어납니다. 음영이 없는 칸은 표 자신의 채우기를 그대로 씁니다.

### 흰색은 음영이 아닙니다

흰색과 채우기 없음은 음영이 아니라 **음영의 부재** 이고, 그러데이션은 muni 가 들 수 없는 채우기라 그 첫 색을 평면 음영으로 읽으면 문서에 없던 색이 생깁니다. 그 판단은 `hangul.CellShade` 한 곳에 있어 읽기와 쓰기가 서로 어긋나지 않습니다.

## 확인

한글이 쓰는 모양의 `borderFill` 넷을 갖춘 새 픽스처입니다 — 리더의 상수가 아니라 형식에서 씁니다. 둘 모두 이번 변경 전에 실패하는 것을 먼저 확인했습니다.

- 음영·흰색·채우기 없음·그러데이션 네 칸이 각각 어떻게 들어오는지
- 쓰는 쪽이 같은 색 칸끼리 `borderFill` 을 나눠 쓰고, 왕복해도 같은지

`borderFill` 의 요소 차례와 브러시 모양은 공개된 실제 `.hwpx` 넷에서 맞췄습니다. `gofmt -l`, `go vet ./...`, `go test ./...` 전체가 통과합니다.

## 업그레이드

마이그레이션과 설정 변경은 필요하지 않습니다.

```bash
gzip -dc muni-v0.27.0.tar.gz | docker load
docker compose -f compose.example.yaml --env-file .env up -d
```

**칸에 색이 든 표를 `.hwpx` 로 가져오셨다면 다시 가져와 보세요.** 그 음영은 이번에 처음 들어옵니다. 이미 가져와 둔 문서는 그대로 남습니다.

## 오프라인 설치

릴리스 asset 에는 `muni:v0.27.0` Docker 이미지가 포함되어 있습니다.

```bash
gzip -dc muni-v0.27.0.tar.gz | docker load
docker image inspect muni:v0.27.0

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

- 파일: `muni-v0.27.0.tar.gz`
- 크기: (릴리스 후 기록)
- SHA-256: (릴리스 후 기록)
- 내부 이미지 태그: `muni:v0.27.0`

```bash
sha256sum muni-v0.27.0.tar.gz
```

GitHub Actions가 이미지 빌드, archive 생성, 내부 이미지 태그 검증을 완료한 뒤 이 asset을 게시했습니다.

## 문서

- [설치 및 운영 안내](https://github.com/hkjang/muni#readme)
- [운영 안내](https://github.com/hkjang/muni/blob/v0.27.0/docs/OPERATIONS.md)
- [아키텍처](https://github.com/hkjang/muni/blob/v0.27.0/docs/ARCHITECTURE.md)
- [MCP 사용법](https://github.com/hkjang/muni/blob/v0.27.0/docs/MCP.md)
- [전체 변경 내역](https://github.com/hkjang/muni/compare/v0.26.0...v0.27.0)
