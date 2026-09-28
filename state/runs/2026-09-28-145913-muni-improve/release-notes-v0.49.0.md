HTML·마크다운을 가져올 때 **그림 이름에 AI 안내 문구와 줄바꿈이 들어가던 것**을 고쳤습니다. 화면은 바뀌지 않았고, 마이그레이션도 설정 변경도 없습니다.

## 고친 것

### 그림 설명이 길면 이름에 `[…문서 컨텍스트가 길어 일부 생략됨…]` 이 붙던 것

가져온 문서 안의 그림(`data:` URI)은 첨부로 저장되고, 그 이름은 그림 설명(HTML 은 `<img alt>`, 마크다운은 `![…]`)에서 만들어집니다. `imageAssetName`(`internal/httpapi/import_html.go:45`)은 설명에서 `/`, `\`, 그리고 `r < 0x20` 제어문자를 걷어낸 **직후** 200룬으로 자릅니다. 그 자르기가 이름 전용 절단기가 아니라 **AI 프롬프트용 `truncateRunes`** 였습니다.

`truncateRunes` 는 값을 줄이면 줄였다고 말합니다 — 줄바꿈 하나와 생략을 알리는 문장을 덧붙입니다. AI 에게 넘기는 문서 컨텍스트에서는 그것이 의도된 동작이지만, **파일 이름에서는 방금 지운 줄바꿈이 절단 단계에서 다시 들어오는 것**입니다. 그래서 설명이 201룬을 넘는 그림의 이름이 이렇게 됐습니다.

```
<200룬><줄바꿈>[…문서 컨텍스트가 길어 일부 생략됨…].png
```

문구 뒤에 확장자가 붙는 순서까지 그대로입니다. 그 이름은 228룬이라 **첨부 이름의 240룬 절단(`import_attachments.go`)에도 걸리지 않고 `attachments.name` 에 그대로 저장되며**, 첨부를 내려받을 때 `Content-Disposition` 의 `filename*` 으로 나갑니다. 결과는 두 자리에서 보입니다 — 편집기의 첨부 목록에 안내 문구가 이름으로 적히고, 내려받은 파일 이름에도 그것이 들어갑니다.

이제 **v0.45.0 이 `safeFilename` 에서 같은 결함을 없앨 때 만든 문구 없는 절단기 `cutFilenameRunes`** (같은 패키지, `export.go`)를 씁니다. 자르기만 하고 아무 말도 하지 않으며, 자른 자리에 남는 뒤쪽 공백도 지웁니다. `inlineContext` 를 HTML 과 마크다운이 함께 쓰므로 두 가져오기 경로가 같이 고쳐집니다.

**바뀌지 않은 것:** 고친 것은 절단기 두 줄뿐입니다.

- **200룬 한도, 확장자 판정, 제어문자 치환기는 한 글자도 다르지 않습니다.** 설명이 200룬 이하면 이름은 v0.48.0 과 완전히 같고, 빈 설명은 여전히 `image.png` 이며, 이미 `.png`·`.jpg`·`.gif`·`.webp` 로 끝나는 설명에 확장자를 두 번 붙이지 않습니다.
- **`truncateRunes` 는 그대로입니다.** 호출자가 스무 곳이고 AI 컨텍스트에서는 그 문구가 필요한 동작입니다. 첨부 이름·제목의 240룬 절단도 이번 범위 밖입니다.
- **내려받기 헤더를 만드는 규칙(v0.46.0·v0.47.0)과 `safeFilename`(v0.45.0)은 손대지 않았습니다.**

검증은 단위와 live 양쪽입니다. 단위는 **HTML 과 마크다운 두 하위 경로에 201룬 설명을 넣어** 이름에 줄바꿈이 없는지 보고, 위의 불변 12건(200룬 이하·빈 설명·이미 확장자로 끝나는 설명·`/`·`\`·제어문자 제거·jpg/gif/webp 선택)을 함께 지킵니다. live 는 **실제 라우트(`POST /api/v1/import`)로 HTML 을 올려 DB 의 `attachments.name` 을 직접 읽고**, 첨부 내려받기 헤더를 프로덕션과 같은 파서(`mime.ParseMediaType`)로 되읽습니다. 고치기 전에 긴 설명을 보는 단위 테스트와 live 테스트가 먼저 실패하는 것을 확인했고, 절단기 두 줄만 되돌리자 그 둘만 다시 실패(나머지 236건은 계속 통과)해 인과를 확정했습니다.

## 업그레이드

마이그레이션과 설정 변경은 필요하지 않습니다. 화면은 손대지 않았고, 서버는 가져온 그림의 첨부 이름을 정하는 자리만 v0.48.0 과 다릅니다.

```bash
gzip -dc muni-v0.49.0.tar.gz | docker load
docker compose -f compose.example.yaml --env-file .env up -d
```

**이미 저장된 첨부 이름은 바뀌지 않습니다.** 설명이 긴 그림을 가져온 적이 있다면 그 첨부 이름에는 안내 문구가 남아 있으므로, 편집기에서 이름을 고치거나 그 문서를 다시 가져오면 됩니다. 새로 가져오는 문서부터는 이름에 문구가 들어가지 않습니다.

## 오프라인 설치

릴리스 asset 에는 `muni:v0.49.0` Docker 이미지가 포함되어 있습니다.

```bash
gzip -dc muni-v0.49.0.tar.gz | docker load
docker image inspect muni:v0.49.0

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
- 메일 알림: 사내 SMTP 릴레이로 1분마다 배경 발송, 기본 꺼짐 · 발송 기록 `/api/v1/admin/mail/deliveries` (관리자 인증 필요)
- OpenAPI: `/api/openapi.yaml` — 실제 라우트와 일치하며, 테스트가 그것을 지킵니다
- Prometheus: `/metrics` — 관리자 인증 필요
- MCP: `/mcp`
- Momento 프록시: `/momento/*` — 방문 추적을 Momento 로 켜 둔 동안에만 답합니다
- 지원 DB: PostgreSQL 15 이상
- 이미지에는 PDF Export용 Chromium과 Noto CJK 글꼴이 포함되어 있습니다.

## 릴리스 파일 검증

- 파일: `muni-v0.49.0.tar.gz`
- 크기: (릴리스 후 기록)
- SHA-256: (릴리스 후 기록)
- 내부 이미지 태그: `muni:v0.49.0`

```bash
sha256sum muni-v0.49.0.tar.gz
```

GitHub Actions가 이미지 빌드, archive 생성, 내부 이미지 태그 검증을 완료한 뒤 이 asset을 게시했습니다.

## 문서

- [설치 및 운영 안내](https://github.com/hkjang/muni#readme)
- [사용자 가이드](https://github.com/hkjang/muni/blob/v0.49.0/docs/USER_GUIDE.md)
- [관리자 가이드](https://github.com/hkjang/muni/blob/v0.49.0/docs/ADMIN_GUIDE.md)
- [운영 안내](https://github.com/hkjang/muni/blob/v0.49.0/docs/OPERATIONS.md)
- [아키텍처](https://github.com/hkjang/muni/blob/v0.49.0/docs/ARCHITECTURE.md)
- [MCP 사용법](https://github.com/hkjang/muni/blob/v0.49.0/docs/MCP.md)
- [전체 변경 내역](https://github.com/hkjang/muni/compare/v0.48.0...v0.49.0)
