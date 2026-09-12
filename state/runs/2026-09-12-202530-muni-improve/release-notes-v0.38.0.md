관리자가 화면에서 **방문 추적 스크립트**를 붙입니다. 붙인 코드를 브라우저가 조용히 막지 않도록 콘텐츠 보안 정책(CSP)을 요청마다 함께 만들고, 그래도 막힌 출처는 목록으로 보여 주어 「허용」 한 번으로 풀립니다. 기본은 꺼짐입니다.

## 추가

### 방문 추적 (서비스 설정 → 「방문 추적」)

어떤 화면이 실제로 쓰이는지 세려면 추적 스크립트를 페이지에 붙여야 합니다. 그런데 muni 의 모든 페이지는 `script-src 'self'` 로 잠겨 있어 코드를 그냥 끼우면 브라우저가 **조용히** 막고, 관리자는 수집기가 비어 있는 이유를 알 길이 없었습니다. 그래서 스니펫과 정책을 한 자리(`internal/tracking`)에서 함께 만듭니다.

- **요청마다 nonce** — 페이지를 낼 때마다 새 nonce 를 만들어 스니펫의 모든 `<script>` 와 정책의 `script-src` 에 같이 넣습니다. 헤더와 마크업을 한 자리에서 만들므로 어긋날 수 없습니다. `'unsafe-inline'` 은 어디에도 넣지 않습니다 — 한 번 풀면 앱의 모든 인라인 스크립트가 함께 허용되고, 추적을 끈 뒤에도 느슨한 채 남기 때문입니다.
- **출처는 코드에서 읽습니다** — 붙여 넣은 코드의 `http(s)` 주소를 긁어 `script-src`·`connect-src`·`img-src` 에 더합니다. 관리자가 정책 문법을 알 필요가 없습니다.
- **차단된 것을 기록합니다** — 켜진 동안에만 정책에 `report-uri /api/v1/tracking/csp-report` 를 넣어 브라우저 자신의 신고를 받습니다. 출처와 지시어를 메모리에 남기고(서로 다른 출처 100개까지, 데이터베이스에는 쓰지 않습니다) 탭의 「정책이 차단한 출처」에 보여 줍니다. 「허용」을 누르면 그 출처 하나가 `tracking.allowed_hosts` 에만 저장됩니다 — 저장하지 않은 폼을 함께 실어 나르지 않도록.

제공자는 `Momento` · `Matomo` · `Google Analytics 4` · `Google Tag Manager` · `직접 붙여넣기`(8KB 까지) 입니다. **Momento 가 첫 자리**이고, 기본으로 muni 가 `/momento/*` 를 수집기로 넘기므로(세션 쿠키는 떼고 방문자 주소는 `X-Forwarded-For` 로) 브라우저는 muni 하고만 이야기하고 정책에 바깥 주소가 아예 등장하지 않습니다. 프록시는 Momento 를 고르고 켜 둔 동안에만 답하며 다른 어디로도 중계하지 않습니다. 설정은 저장소에 두고 화면에서 바꾸므로 수집기 주소가 바뀌어도 다시 배포하지 않습니다. 「서비스 관리 화면에서도 추적」은 기본이 아니오이고, 넣을 자리는 `<head>` 끝 또는 `<body>` 끝입니다.

**기본값은 꺼짐**이라 새 설치의 페이지 정책은 예전 문자열과 바이트 하나 다르지 않고, 테스트가 그것을 지킵니다. 화면이 아닌 경로(`/api`·`/mcp`·`/healthz`·`/readyz`·`/metrics`·`/momento`)에는 스니펫이 붙지 않고 정책은 오히려 더 좁은 `default-src 'none'` 입니다.

실제로 띄워 확인했습니다 — 가짜 Momento 수집기를 두고 headless Chromium 으로 로그인 화면을 열자 프록시를 거쳐 방문이 들어왔고(쿠키 없음), 일부러 정책 밖 주소를 부르는 코드를 붙이자 브라우저의 신고가 목록에 `script-src-elem` 으로 나타났으며, 「허용」 뒤 새로 고치니 정책에 그 출처가 들어가고 스크립트가 실렸습니다.

[관리자 가이드](https://github.com/hkjang/muni/blob/v0.38.0/docs/ADMIN_GUIDE.md#방문-추적)에 설정 표와 Momento 를 먼저 고르라는 이유, CSP 설명(nonce·출처 읽기·신고·끄면 원래대로·SPA 화면 전환의 한계)과 새로 찍은 화면을 더하고 PDF 를 다시 구웠습니다.

## 업그레이드

마이그레이션과 설정 변경은 필요하지 않습니다. 방문 추적은 꺼진 채 올라오며, 켜기 전까지 서버와 화면의 동작은 v0.37.0 과 같습니다.

```bash
gzip -dc muni-v0.38.0.tar.gz | docker load
docker compose -f compose.example.yaml --env-file .env up -d
```

## 오프라인 설치

릴리스 asset 에는 `muni:v0.38.0` Docker 이미지가 포함되어 있습니다.

```bash
gzip -dc muni-v0.38.0.tar.gz | docker load
docker image inspect muni:v0.38.0

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
- Momento 프록시: `/momento/*` — 방문 추적을 Momento 로 켜 둔 동안에만 답합니다
- 지원 DB: PostgreSQL 15 이상
- 이미지에는 PDF Export용 Chromium과 Noto CJK 글꼴이 포함되어 있습니다.

## 릴리스 파일 검증

- 파일: `muni-v0.38.0.tar.gz`
- 크기: (릴리스 후 기록)
- SHA-256: (릴리스 후 기록)
- 내부 이미지 태그: `muni:v0.38.0`

```bash
sha256sum muni-v0.38.0.tar.gz
```

GitHub Actions가 이미지 빌드, archive 생성, 내부 이미지 태그 검증을 완료한 뒤 이 asset을 게시했습니다.

## 문서

- [설치 및 운영 안내](https://github.com/hkjang/muni#readme)
- [사용자 가이드](https://github.com/hkjang/muni/blob/v0.38.0/docs/USER_GUIDE.md)
- [관리자 가이드](https://github.com/hkjang/muni/blob/v0.38.0/docs/ADMIN_GUIDE.md)
- [운영 안내](https://github.com/hkjang/muni/blob/v0.38.0/docs/OPERATIONS.md)
- [아키텍처](https://github.com/hkjang/muni/blob/v0.38.0/docs/ARCHITECTURE.md)
- [MCP 사용법](https://github.com/hkjang/muni/blob/v0.38.0/docs/MCP.md)
- [전체 변경 내역](https://github.com/hkjang/muni/compare/v0.37.0...v0.38.0)
