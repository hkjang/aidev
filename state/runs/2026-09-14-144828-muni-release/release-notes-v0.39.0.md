문서를 **내려받지 않고 다른 사내 서비스로 바로 넘기고**, 허용한 서비스에서 **바로 받습니다**. Keycloak 에 이미 로그인한 사람은 **로그인 화면 없이** 들어옵니다. 둘 다 기본은 꺼짐이라 켜기 전까지 아무것도 달라지지 않습니다.

## 추가

### 문서 넘기기 (서비스 설정 → 「문서 넘기기」)

생각은 캔버스에서 시작해 문서가 되고 슬라이드가 되어 보고로 들어갑니다. 단계마다 사람이 파일을 내려받아 다시 올리고 있었습니다. 사내 서비스 표준(HANDOFF-STANDARD)의 형식 표에서 muni 가 맡은 칸 — 보냄 `markdown`·`docx`, 받음 `markdown` — 을 양쪽 다 만들었습니다.

서비스끼리 서로의 자격 증명을 들고 있지 않습니다. 보내는 쪽 `POST /api/v1/handoff/claims` 가 그 사람이 읽을 수 있는 문서 하나를 **지금 렌더링해** 5분짜리 단일 사용 표(claim)로 묶고, `GET /api/v1/handoff/claims/{claim}` 이 로그인 없이 표를 내줍니다. **내주는 것이 곧 삭제**라 두 번째 요청은 404 이고, 만료된 표도 없는 표도 같은 404 입니다. 표는 SHA-256 해시로만 저장하며(`handoff_claims`), 요청 로그의 경로에서는 `{claim}` 으로 가리고 감사 로그(`HANDOFF_ISSUE`·`HANDOFF_SERVE`)에도 표를 넣지 않습니다.

받는 쪽 `GET /handoff?source&claim` 의 `source` 는 밖에서 온 값입니다. 그대로 받아 오면 muni 가 사내 아무 주소나 대신 긁어 오는 도구가 되므로:

- 허용 목록에 없는 `source` 는 **아무것도 요청하지 않고** 거절합니다. 호스트가 같아도 경로·쿼리·userinfo 가 붙은 값은 오리진이 아니므로 거절합니다.
- 리다이렉트는 따라가지 않고 그 자리에서 실패합니다.
- `Content-Type` 을 본문을 읽기 전에 보고, `text/markdown` 이 아니면 읽지 않습니다. 크기는 **25MB**, 시간은 **30초**에서 끊습니다.
- 실패는 JSON 이 아니라 사람이 읽을 화면입니다 — 허용되지 않은 서비스(403), 만료·사용됨(404), 너무 큼(413), 읽을 수 없는 형식(415), 상대가 응답하지 않음(502). 세션이 없으면 로그인 화면으로 보냈다가 같은 주소로 돌아옵니다.
- 받은 문서는 그 사람의 **개인 워크스페이스**에 생기고 `/docs/{id}` 로 이동하며, 첫 버전의 사유가 `handoff:<출처>` 이고 감사 로그에 `HANDOFF_RECEIVE` 와 출처가 남습니다.

허용 목록(`handoff.peers`)은 방문 추적과 같은 자리인 서비스 설정에 두고 화면에서 고칩니다 — 주소(오리진), 이름, 그 서비스가 받는 형식. 같은 목록이 양쪽에 다 쓰여 여기 적은 서비스로만 보내고 여기 적은 주소에서만 받습니다. **기본은 비어 있어** 내보내기 메뉴에 「…으로 보내기」가 나타나지 않고 아무 데서도 받지 않으므로 새 설치는 달라지지 않습니다. 항목은 `/api/v1/system/capabilities` 의 `handoffTargets` 가 그 서비스가 받는 형식으로만 채워질 때 나타나고(`docx` 는 「보안·내보내기」의 DOCX 내보내기도 켜져야), 표를 받기 전에 탭을 먼저 열어 팝업 차단을 피합니다. 파일 이름은 RFC 8187 로 온전히 퍼센트 인코딩합니다 — 실제로 띄워 보니 한글 제목이 받는 쪽에서 비어 있었습니다.

실제로 띄워 확인했습니다 — muni 를 자기 자신의 peer 로 등록하고 headless Chromium 으로, 목록이 비면 메뉴에 항목이 없고, 목록 밖 `source` 는 403 화면이며, 「…으로 보내기 (Markdown)」을 누르면 새 탭이 `/docs/<새 id>` 에 열리고 그 문서를 내보내면 본문이 그대로이고, 감사 로그에 세 행, 요청 로그에는 `{claim}` 만 있고 실제 표는 0건이었습니다.

[관리자 가이드](https://github.com/hkjang/muni/blob/v0.39.0/docs/ADMIN_GUIDE.md#문서-넘기기)에 설정 표와 받는 쪽이 지키는 것, 장애 대응을 더했습니다.

### 자동 로그인 (서비스 설정 → Keycloak OIDC)

SSO 가 있어도 앱을 열 때마다 로그인 화면을 한 번 지나야 했습니다. OIDC 의 `prompt=none` 으로 제공자에게 "이미 있는 세션으로만 답하라" 고 묻습니다 — 세션이 있으면 코드가 바로 돌아와 평소 로그인처럼 끝나고 깊은 링크로 돌아가며, 없으면 `error=login_required` 가 돌아오는데 그것은 실패가 아니라 평범한 대답이라 콜백이 오류 없이 `/login?sso=none` 으로 보냅니다. 숨은 iframe 이 아니라 최상위 이동이어서 서드파티 쿠키를 막은 브라우저에서도 동작합니다.

이 일의 전부는 **무한 루프를 막는 것**입니다. 거절을 받고 다시 시도하면 브라우저가 제공자와 앱 사이를 끝없이 오가므로 막는 장치를 세 겹으로 두었습니다 — 한 탭 세션에 한 번(`sessionStorage`), 스스로 로그아웃했으면 억제(세션이 다시 생기면 해제), 콜백이 주소에 남기는 `sso=none`. 저장소를 읽지 못하면 '이미 시도했다' 로 칩니다. 로그인·콜백과 `/api`·`/mcp`·`/healthz` 경로에서는 시도하지 않습니다.

설정은 `oidc.auto_login` 이고 **기본은 꺼짐**입니다. 서버는 그 설정이 꺼져 있으면 `?prompt=none` 이 와도 조용히 평범한 로그인으로 바꾸므로 주소에 붙이는 것으로 흐름을 바꿀 수 없고, 조용한 시도였는지는 `oidc_states` 행에 적어 콜백이 읽습니다. `return_to` 는 `/` 로 시작하고 `//` 로 시작하지 않는 값만 받습니다.

실제로 띄워 확인했습니다 — 세션 없는 가짜 Keycloak 에 대해 탭 세션마다 authorize 호출이 정확히 한 번이고, 새로 고침·다른 화면 이동·로그아웃 뒤에 다시 묻지 않으며, 꺼진 설치에서는 제공자를 부르지 않습니다.

[관리자 가이드](https://github.com/hkjang/muni/blob/v0.39.0/docs/ADMIN_GUIDE.md#자동-로그인silent-sso)에 설정과 동작을 더했습니다.

## 그 밖에

떠 있는 이미지만 보고는 어느 버전·어느 커밋으로 만든 것인지 알 수 없었습니다. 빌드 인자로 이미 바이너리에 새기던 버전·커밋·빌드 시각을 이미지 메타데이터에도 `org.opencontainers.image.{title,description,source,version,revision,created}` 여섯 개 라벨로 붙입니다. `docker image inspect muni:v0.39.0` 에서 보입니다.

## 업그레이드

마이그레이션이 두 개 있습니다(`017` `oidc_states` 에 열 하나, `018` `handoff_claims` 표). 기동할 때 자동으로 적용되고 기존 데이터를 바꾸지 않으며, 설정 변경은 필요하지 않습니다. 문서 넘기기의 허용 목록은 빈 채, 자동 로그인은 꺼진 채 올라오며, 켜기 전까지 서버와 화면의 동작은 v0.38.0 과 같습니다.

```bash
gzip -dc muni-v0.39.0.tar.gz | docker load
docker compose -f compose.example.yaml --env-file .env up -d
```

## 오프라인 설치

릴리스 asset 에는 `muni:v0.39.0` Docker 이미지가 포함되어 있습니다.

```bash
gzip -dc muni-v0.39.0.tar.gz | docker load
docker image inspect muni:v0.39.0

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

- 파일: `muni-v0.39.0.tar.gz`
- 크기: (릴리스 후 기록)
- SHA-256: (릴리스 후 기록)
- 내부 이미지 태그: `muni:v0.39.0`

```bash
sha256sum muni-v0.39.0.tar.gz
```

GitHub Actions가 이미지 빌드, archive 생성, 내부 이미지 태그 검증을 완료한 뒤 이 asset을 게시했습니다.

## 문서

- [설치 및 운영 안내](https://github.com/hkjang/muni#readme)
- [사용자 가이드](https://github.com/hkjang/muni/blob/v0.39.0/docs/USER_GUIDE.md)
- [관리자 가이드](https://github.com/hkjang/muni/blob/v0.39.0/docs/ADMIN_GUIDE.md)
- [운영 안내](https://github.com/hkjang/muni/blob/v0.39.0/docs/OPERATIONS.md)
- [아키텍처](https://github.com/hkjang/muni/blob/v0.39.0/docs/ARCHITECTURE.md)
- [MCP 사용법](https://github.com/hkjang/muni/blob/v0.39.0/docs/MCP.md)
- [전체 변경 내역](https://github.com/hkjang/muni/compare/v0.38.0...v0.39.0)
