muni 가 내보낸 Markdown 파일을 **다시 가져오거나 다른 muni 에서 넘겨받으면** 제목이 두 번 보이던 것을 고쳤습니다. 화면은 바뀌지 않았고, 마이그레이션도 설정 변경도 없습니다.

## 고친 것

### 다시 가져온 문서와 넘겨받은 문서에 제목이 두 번 보이던 것

muni 의 Markdown 내보내기는 제목을 첫 줄 `# 제목` 으로 적고 파일 이름도 제목으로 짓습니다. 그런데 가져오기(`.md` 업로드)와 v0.39.0 의 문서 넘기기(받는 쪽)는 제목을 **파일 이름에서** 정하고 첫 줄의 `# 제목` 은 본문에 그대로 두었습니다. 그래서 muni 가 내보낸 파일이 돌아오면 — 다시 올리거나, 다른 muni 에서 넘겨받거나 — 페이지 위의 제목과 본문 첫 줄의 제목이 나란히 보였고, 검색 텍스트에도 제목이 두 번 들어갔습니다.

이제 두 자리 모두, 제목이 확정된 직후에 **첫 블록이 level-1 제목이고 그 글이 확정된 제목과 정확히 같을 때만** 그 블록을 본문에서 뺍니다. 비교는 앞뒤 공백만 걷어낸 정확한 일치라서, 글쓴이가 붙인 다른 제목·`##` 제목·문단 뒤에 오는 `#` 제목은 바이트 그대로 남습니다. 파일 이름으로 쓸 수 없는 글자 때문에 제목이 바뀐 경우(`safeFilename`)도 일치하지 않으므로 손대지 않습니다. 본문이 `# 제목` 한 줄뿐이던 파일은 빈 문서가 되어 빈 문단 하나로 열립니다. 검색 텍스트는 뺀 뒤의 본문에서 뽑으므로 제목이 두 번 색인되지 않습니다.

**바뀌지 않은 것:** 이미 열어 둔 문서에 파일 내용을 끼워 넣는 경로, Markdown 내보내기, `.txt`·`.html`·`.docx`·`.hwp`·`.hwpx`·PDF 등 다른 형식의 가져오기는 그대로입니다. 업로드 폼의 title 이 파일 첫 줄과 다르면 그 `#` 제목도 그대로 남습니다.

단위 테스트 4개(왕복에서 첫 H1 만 사라지고 나머지 블록은 같음 / 다른 제목·H2·문단 뒤 H1·바뀐 제목은 바이트 그대로 / 이스케이프된 제목 일치 / H1 만 있던 문서는 빈 문단)와 PostgreSQL 을 띄운 live 테스트 2개(`.md` 업로드와 넘겨받기에서 첫 블록이 문단이 되고 `content_text` 에 제목이 없음, 끼워 넣기와 다른 제목은 heading 유지)가 이것을 지킵니다. 사용자 가이드 「받은 파일을 muni 문서로 만들기」에 두 줄을 더했습니다.

## 업그레이드

마이그레이션과 설정 변경은 필요하지 않습니다. 화면은 손대지 않았고, 서버는 Markdown 가져오기와 넘겨받기에서 제목과 같은 첫 제목 블록을 빼는 것만 v0.40.0 과 다릅니다.

```bash
gzip -dc muni-v0.41.0.tar.gz | docker load
docker compose -f compose.example.yaml --env-file .env up -d
```

**이미 두 번 보이는 문서는 그대로입니다.** 이 고침은 가져오거나 넘겨받는 순간에만 적용되므로, 이전에 만들어져 제목이 두 번 들어 있는 문서는 편집기에서 첫 줄을 지워 주세요.

## 오프라인 설치

릴리스 asset 에는 `muni:v0.41.0` Docker 이미지가 포함되어 있습니다.

```bash
gzip -dc muni-v0.41.0.tar.gz | docker load
docker image inspect muni:v0.41.0

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
- 문서 넘기기: `/handoff?source&claim` (받는 쪽, 세션 필요, 허용 목록의 오리진만) · `/api/v1/handoff/claims/{claim}` (내주는 쪽, 인증 없이 5분짜리 단일 사용 표로만 열립니다)
- OpenAPI: `/api/openapi.yaml` — 실제 라우트와 일치하며, 테스트가 그것을 지킵니다
- Prometheus: `/metrics` — 관리자 인증 필요
- MCP: `/mcp`
- Momento 프록시: `/momento/*` — 방문 추적을 Momento 로 켜 둔 동안에만 답합니다
- 지원 DB: PostgreSQL 15 이상
- 이미지에는 PDF Export용 Chromium과 Noto CJK 글꼴이 포함되어 있습니다.

## 릴리스 파일 검증

- 파일: `muni-v0.41.0.tar.gz`
- 크기: (릴리스 후 기록)
- SHA-256: (릴리스 후 기록)
- 내부 이미지 태그: `muni:v0.41.0`

```bash
sha256sum muni-v0.41.0.tar.gz
```

GitHub Actions가 이미지 빌드, archive 생성, 내부 이미지 태그 검증을 완료한 뒤 이 asset을 게시했습니다.

## 문서

- [설치 및 운영 안내](https://github.com/hkjang/muni#readme)
- [사용자 가이드](https://github.com/hkjang/muni/blob/v0.41.0/docs/USER_GUIDE.md)
- [관리자 가이드](https://github.com/hkjang/muni/blob/v0.41.0/docs/ADMIN_GUIDE.md)
- [운영 안내](https://github.com/hkjang/muni/blob/v0.41.0/docs/OPERATIONS.md)
- [아키텍처](https://github.com/hkjang/muni/blob/v0.41.0/docs/ARCHITECTURE.md)
- [MCP 사용법](https://github.com/hkjang/muni/blob/v0.41.0/docs/MCP.md)
- [전체 변경 내역](https://github.com/hkjang/muni/compare/v0.40.0...v0.41.0)
