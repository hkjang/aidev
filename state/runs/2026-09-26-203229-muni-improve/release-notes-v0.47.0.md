워크스페이스를 ZIP 으로 내보낼 때 **이름에 큰따옴표가 하나만 있어도 파일 이름이 통째로 버려지던 것**을 고쳤습니다. 화면은 바뀌지 않았고, 마이그레이션도 설정 변경도 없습니다.

## 고친 것

### 워크스페이스 이름의 큰따옴표가 `Content-Disposition` 을 깨뜨리던 것

v0.46.0 은 내려받기 이름을 실어 나르는 `filename*` 을 RFC 8187 의 ext-value 규칙으로 통일했지만, **헤더를 내는 다섯 자리 가운데 워크스페이스 ZIP 내려받기(`export.zip`) 하나는 그 범위 밖**이었습니다. 그 자리에는 `filename*` 이 아예 없었고, 이름은 ASCII 쪽 quoted-string 에 원문 그대로 들어갔습니다.

```
attachment; filename="<워크스페이스 이름>-20260926.zip"
```

quoted-string 안에서 큰따옴표는 값의 끝입니다. 그리고 **워크스페이스 이름은 만들 때 길이(80룬)와 앞뒤 공백만 검사받습니다**(`workspaces.go`) — 따옴표도, 쌍반점도, 한글도 그대로 통과합니다. 그러므로 `[대외비] 2026년 계획(초안); 달성률 100% "검토용"` 같은 평범한 이름 하나로 헤더는 이렇게 됩니다.

- **큰따옴표가 그 자리에서 매개변수를 끝냅니다.** 뒤에 남은 글자들은 매개변수도, 값도 아닌 것이 되고, 받는 쪽 파서(`mime.ParseMediaType`)는 `mime: invalid media parameter` 로 **헤더 전체를 버립니다**. 이름이 잘리는 것이 아니라 아무 이름도 남지 않습니다.
- 쌍반점만 있어도 그 자리에서 매개변수가 끝나 이름이 조용히 잘립니다 — v0.46.0 이 다른 네 경로에서 닫은 것과 같은 고장입니다.
- 한글 이름은 quoted-string 안의 raw UTF-8 이라 규격상 latin-1 로 읽힙니다.

이제 이름은 다른 넷과 같이 **`extValueEscape` 를 지난 `filename*`** 으로 가고, ASCII fallback 에는 **날짜만** 남습니다 — 이름이 무엇이든 ASCII 인 것은 그것뿐입니다.

```
attachment; filename="workspace-20260926.zip"; filename*=UTF-8''<escape 된 이름>-20260926.zip
```

날짜 문자열은 한 번만 계산해 두 자리에 씁니다. 자정을 걸쳐 헤더를 쓰더라도 두 이름의 날짜가 어긋나지 않습니다. 이것으로 **내려받기 헤더를 내는 다섯 자리가 모두 같은 규칙**을 씁니다.

**`filename*` 을 모르는 아주 오래된 클라이언트에서는 ZIP 이 `workspace-20260926.zip` 으로 저장됩니다** — 이름이 대신 들어가던 자리이지만, 그 자리에서는 이름이 헤더를 깨뜨리거나 latin-1 로 읽히는 것 말고는 선택이 없었습니다. 브라우저와 `curl` 을 포함해 `filename*` 을 읽는 클라이언트는 전과 같이 워크스페이스 이름으로 저장합니다.

**바뀌지 않은 것:** `safeFilename` 의 치환과 100룬 절단(v0.45.0), `extValueEscape` 의 attr-char 경계(v0.46.0), **ZIP 안의 폴더·문서 항목 이름과 `목록.md` 의 기록은 한 바이트도 달라지지 않습니다.** `safeFilename` 은 다섯 경로가 함께 쓰는 것이라, 따옴표를 지우는 방식으로 고치면 헤더는 나아도 ZIP 항목 이름과 `목록.md` 가 서로 어긋났을 것입니다. 고친 것은 헤더를 적는 한 줄입니다.

검증은 v0.46.0 과 같은 방식입니다 — **실제 서버에 워크스페이스를 만들고 `export.zip?format=md` 를 받아, 응답 헤더를 프로덕션과 같은 파서로 되읽어** 원래 이름과 같은지 봅니다. 고치기 전에 그 테스트가 위의 `mime: invalid media parameter` 로 먼저 실패하는 것을 확인했고, 헤더 한 줄만 되돌리자 그 하나만 다시 실패(나머지 233건은 계속 통과)해 인과를 확정했습니다.

## 업그레이드

마이그레이션과 설정 변경은 필요하지 않습니다. 화면은 손대지 않았고, 서버는 워크스페이스 ZIP 의 내려받기 이름을 적는 자리만 v0.46.0 과 다릅니다.

```bash
gzip -dc muni-v0.47.0.tar.gz | docker load
docker compose -f compose.example.yaml --env-file .env up -d
```

**이미 내려받은 ZIP 파일의 이름은 그대로입니다.** 이 고침은 헤더를 만드는 순간에만 적용되므로, 이름을 잃고 저장된 파일은 다시 내보내면 워크스페이스 이름으로 저장됩니다. **ZIP 안의 내용과 항목 이름은 v0.46.0 과 완전히 같습니다.**

## 오프라인 설치

릴리스 asset 에는 `muni:v0.47.0` Docker 이미지가 포함되어 있습니다.

```bash
gzip -dc muni-v0.47.0.tar.gz | docker load
docker image inspect muni:v0.47.0

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

- 파일: `muni-v0.47.0.tar.gz`
- 크기: (릴리스 후 기록)
- SHA-256: (릴리스 후 기록)
- 내부 이미지 태그: `muni:v0.47.0`

```bash
sha256sum muni-v0.47.0.tar.gz
```

GitHub Actions가 이미지 빌드, archive 생성, 내부 이미지 태그 검증을 완료한 뒤 이 asset을 게시했습니다.

## 문서

- [설치 및 운영 안내](https://github.com/hkjang/muni#readme)
- [사용자 가이드](https://github.com/hkjang/muni/blob/v0.47.0/docs/USER_GUIDE.md)
- [관리자 가이드](https://github.com/hkjang/muni/blob/v0.47.0/docs/ADMIN_GUIDE.md)
- [운영 안내](https://github.com/hkjang/muni/blob/v0.47.0/docs/OPERATIONS.md)
- [아키텍처](https://github.com/hkjang/muni/blob/v0.47.0/docs/ARCHITECTURE.md)
- [MCP 사용법](https://github.com/hkjang/muni/blob/v0.47.0/docs/MCP.md)
- [전체 변경 내역](https://github.com/hkjang/muni/compare/v0.46.0...v0.47.0)
