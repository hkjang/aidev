별지 서식처럼 항목마다 새 쪽에서 시작하는 문서 — 그 쪽 나누기는 muni 가 `.docx`·HTML·PDF 에서 예전부터 지키던 블록이고 `.hwpx` 쓰기도 이미 적고 있었는데, 한글 두 리더만 그것을 버렸습니다. 그래서 같은 문서라도 한글 파일로 들어오면 끊긴 자리 없이 한 덩어리가 되었고, 그대로 내보내면 쪽 구성이 사라졌습니다.

## 추가

### 나뉜 자리에는 표시가 없습니다

쪽 나누기는 그 자리에 놓인 무엇이 아니라 **다음 문단이 지니는 성질** 입니다. `.hwp` 는 PARA_HEADER 의 문단 나누기 종류(스타일 다음 바이트)의 **셋째 비트**, `.hwpx` 는 `<hp:p pageBreak="1">` 입니다. 그래서 리더는 그 문단을 만나면 문단 *앞* 에 쪽 나누기 블록을 놓습니다.

### 맨 앞의 나누기는 버립니다

문서의 첫 문단이 나누기를 지니면 그대로 읽어 빈 쪽으로 문서를 열게 됩니다. 특히 `.hwp` 는 **모든 파일의 첫 문단** 이 0x03(구역·다단 나누기)을 지니고 있어, 비트를 가리지 않으면 열어 보는 파일마다 쪽 나누기가 하나씩 생깁니다.

### 나누기만 든 빈 문단은 나누기 하나입니다

`.hwpx` 에서 나누기를 지녔지만 아무 글자도 없는 문단은 나누기 하나로만 읽습니다. muni 가 쓰는 모양이 바로 그것이라, 그렇지 않으면 문서가 오갈 때마다 빈 줄이 하나씩 쌓입니다.

## 확인

변경 전에 실패하는 것을 먼저 확인한 새 테스트 넷과 왕복 하나입니다.

- `.hwp` 와 `.hwpx` 가 문단 앞에 쪽 나누기를 놓는지, 첫 문단의 나누기를 버리는지
- 나누기만 든 빈 문단이 빈 줄을 남기지 않고, 왕복해도 같은지

공개된 실제 파일 스물둘로 맞췄습니다. `.hwpx` 여섯 개의 `pageBreak="1"` 열둘과 `.hwp` 열여섯 개의 나누기 넷이 읽은 수와 정확히 같고, 실제 `.hwp` 열여섯 개를 `.hwpx` 로 왕복시켜 글자와 구조가 그대로인지 보았습니다. 그 김에 `.hwp` 픽스처가 글자 모양 개수를 형식이 정한 12번이 아니라 10번 바이트에 적던 것을 고쳤습니다 — 그 자리는 스타일이고, 그 다음이 이번에 읽는 나누기 종류입니다.

`gofmt -l`, `go vet ./...`, `go test ./...` 전체가 통과합니다.

## 업그레이드

마이그레이션과 설정 변경은 필요하지 않습니다.

```bash
gzip -dc muni-v0.28.0.tar.gz | docker load
docker compose -f compose.example.yaml --env-file .env up -d
```

**쪽이 나뉜 한글 문서를 가져오셨다면 다시 가져와 보세요.** 그 나누기는 이번에 처음 들어옵니다. 이미 가져와 둔 문서는 그대로 남습니다.

## 오프라인 설치

릴리스 asset 에는 `muni:v0.28.0` Docker 이미지가 포함되어 있습니다.

```bash
gzip -dc muni-v0.28.0.tar.gz | docker load
docker image inspect muni:v0.28.0

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

- 파일: `muni-v0.28.0.tar.gz`
- 크기: (릴리스 후 기록)
- SHA-256: (릴리스 후 기록)
- 내부 이미지 태그: `muni:v0.28.0`

```bash
sha256sum muni-v0.28.0.tar.gz
```

GitHub Actions가 이미지 빌드, archive 생성, 내부 이미지 태그 검증을 완료한 뒤 이 asset을 게시했습니다.

## 문서

- [설치 및 운영 안내](https://github.com/hkjang/muni#readme)
- [운영 안내](https://github.com/hkjang/muni/blob/v0.28.0/docs/OPERATIONS.md)
- [아키텍처](https://github.com/hkjang/muni/blob/v0.28.0/docs/ARCHITECTURE.md)
- [MCP 사용법](https://github.com/hkjang/muni/blob/v0.28.0/docs/MCP.md)
- [전체 변경 내역](https://github.com/hkjang/muni/compare/v0.27.0...v0.28.0)
