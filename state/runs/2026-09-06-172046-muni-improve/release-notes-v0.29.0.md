실제 정부 문서의 머리글 그림 — 1632픽셀짜리를 163픽셀로 줄여 그린 것입니다. 그림의 바이트는 원본 크기 그대로이고 문서는 그것을 줄여서 그리는데, 한글 두 리더가 그린 크기를 버려서 그 머리글 그림이 쪽을 가득 채운 채로 들어왔습니다. `.docx` 리더는 예전부터 `<wp:extent>` 를 지켰고 `.hwpx` 쓰기도 `width` 를 읽고 있었는데, 한글 쪽 입구만 비어 있었습니다.

## 추가

### 크기는 그림 자신에게 없습니다

`.hwp` 는 그림 위의 **SHAPE_COMPONENT** 에 크기를 두 개 적고 그 **둘째가 그려진 크기** 입니다. 그 낱말이 어디서 시작하는지는 컴포넌트가 컨트롤 바로 아래인지(컨트롤 아이디를 두 번 적습니다) 묶음 안인지(한 번)에 따라 다르니, 한 자리만 보고 세면 다른 문서에서 어긋납니다. `.hwpx` 는 **`<hp:sz>`** 가 그려진 크기이고 `<hp:orgSz>` 가 원본입니다.

### 무엇을 크기로 볼지는 한 곳에 있습니다

`hangul.PictureSize` 한 곳에 두어 두 리더가 함께 씁니다. **가로세로 둘 다 실제 길이일 때만** 기록합니다 — 반쪽짜리 크기는 없느니만 못합니다.

### `.hwpx` 쓰기는 높이도 적힌 대로 씁니다

바이트에서 다시 계산하면, 일부러 눌러 놓은 그림의 모양이 왕복에서 풀립니다.

## 고침

### `.hwp` — 그림이 이웃의 그림으로 들어오던 것

그림이 적어 둔 번호는 **스트림의 번호가 아니라 DocInfo 의 BIN_DATA 레코드를 세는 번호** 이고, 어느 스트림인지는 그 레코드가 말합니다. 그림 셋을 쓴 차례와 적어 둔 차례가 다른 실제 보고서(basicsReport.hwp)에서는 세 그림이 모두 이웃의 그림으로 들어왔습니다.

그린 크기를 읽게 되면서 드러났습니다 — 그림마다 그려진 크기가 자기 그림이 아니라 다른 그림의 생김새와 꼭 맞았고, 셋이 한 바퀴 도는 모양이었습니다. 레코드가 그 번호에 대해 아무 말도 하지 않는 파일은 예전처럼 번호를 스트림의 것으로 읽습니다. 두 차례가 같은 파일은 늘 그래 왔습니다.

## 확인

변경 전에 실패하는 것을 먼저 확인한 새 테스트 넷입니다.

- `.hwp` 가 SHAPE_COMPONENT 의 둘째 크기를, `.hwpx` 가 `<hp:sz>` 를 읽는지
- 눌러 놓은 그림이 `.hwpx` 왕복에서 가로세로 그대로인지
- 그림이 BIN_DATA 레코드가 가리키는 스트림과 짝지어지는지

공개된 실제 파일 스물둘 — `.hwp` 열여섯과 `.hwpx` 여섯 — 의 Corpus·RealRoundTrip 이 통과합니다. 고친 뒤에는 basicsReport.hwp 의 세 그림 모두 그려진 크기와 원본 크기의 비율이 같습니다.

`gofmt -l`, `go vet ./...`, `go test ./...` 전체가 통과합니다.

## 업그레이드

마이그레이션과 설정 변경은 필요하지 않습니다.

```bash
gzip -dc muni-v0.29.0.tar.gz | docker load
docker compose -f compose.example.yaml --env-file .env up -d
```

**그림이 든 한글 문서를 가져오셨다면 다시 가져와 보세요.** 줄여 그린 그림은 이번에 처음 그 크기대로 들어오고, 그림이 여럿인 옛 문서는 짝이 바로잡힙니다. 이미 가져와 둔 문서는 그대로 남습니다.

## 오프라인 설치

릴리스 asset 에는 `muni:v0.29.0` Docker 이미지가 포함되어 있습니다.

```bash
gzip -dc muni-v0.29.0.tar.gz | docker load
docker image inspect muni:v0.29.0

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

- 파일: `muni-v0.29.0.tar.gz`
- 크기: (릴리스 후 기록)
- SHA-256: (릴리스 후 기록)
- 내부 이미지 태그: `muni:v0.29.0`

```bash
sha256sum muni-v0.29.0.tar.gz
```

GitHub Actions가 이미지 빌드, archive 생성, 내부 이미지 태그 검증을 완료한 뒤 이 asset을 게시했습니다.

## 문서

- [설치 및 운영 안내](https://github.com/hkjang/muni#readme)
- [운영 안내](https://github.com/hkjang/muni/blob/v0.29.0/docs/OPERATIONS.md)
- [아키텍처](https://github.com/hkjang/muni/blob/v0.29.0/docs/ARCHITECTURE.md)
- [MCP 사용법](https://github.com/hkjang/muni/blob/v0.29.0/docs/MCP.md)
- [전체 변경 내역](https://github.com/hkjang/muni/compare/v0.28.0...v0.29.0)
