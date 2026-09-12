# MOINA 관리자 가이드

버전 `v0.1.28` · linux/amd64 단일 컨테이너 · 외부 PostgreSQL · 폐쇄망 운영

이 문서는 MOINA를 **설치하고 지키는 사람**을 위한 안내입니다. 화면을 쓰는 방법은
[사용자 가이드](USER_GUIDE.md)에 있으니 사용자에게는 그 문서를 안내하세요.

이 문서의 화면 캡처는 `v0.1.28` 서비스를 실제로 띄워 캡처 전용 데이터베이스에서 찍은
것이며, 화면에 보이는 계정·주소는 모두 예시입니다.

---

## 1. 구성 요소

| 구성 | 내용 |
| --- | --- |
| 서비스 컨테이너 | `moina:v0.1.28` 하나. distroless·non-root·read-only로 실행하며 Go 서버와 React 웹 앱을 함께 담고 있습니다. |
| 데이터 저장소 | **외부 PostgreSQL**. 이미지에 포함되지 않으므로 기관 표준 PostgreSQL을 먼저 준비합니다. 미디어는 Large Object로 저장합니다. |
| 노출 포트 | 컨테이너 `8080`(HTTP). 기본 compose는 `127.0.0.1:8080`에만 bind합니다. |
| 앞단 | 기관 표준 TLS reverse proxy. WebSocket과 SSE를 통과시켜야 합니다. |
| 선택 연동 | Keycloak 등 OIDC Provider, OpenAI 호환 AI endpoint, SMTP 메일 서버. 모두 관리 화면에서 켜고 끕니다. |
| 관측 | `GET /metrics`(Prometheus text), 구조화 JSON 로그, 응답의 `X-Request-ID` |

브라우저 런타임은 외부 CDN·웹 폰트·분석 스크립트를 쓰지 않고, 완성된 이미지는 registry나
package repository에 접근하지 않습니다. 선택 연동은 폐쇄망 안에서 도달할 수 있는 내부
endpoint여야 합니다.

자원은 최소 2 CPU·2 GiB RAM과 시간 동기화된 linux/amd64 호스트를 권장합니다. 컨테이너
자체는 `/tmp`만 쓰므로 별도 볼륨이 필요 없고, 보존해야 하는 상태는 전부 PostgreSQL에
있습니다.

### 배포 전 점검

| 항목 | 확인할 것 |
| --- | --- |
| Container runtime | Docker 24+ 또는 호환 runtime, Compose v2 |
| PostgreSQL | 전용 database·user, UTF-8, TLS, migration 권한, `pg_trgm` 생성 권한, backup에 Large Object 포함 |
| TLS | reverse proxy에서 HTTPS 종료, WebSocket·SSE 허용, `/metrics`는 운영망에서만 접근 |
| 선택 endpoint | 내부 Keycloak·AI·SMTP의 DNS·CA·방화벽과 정확한 host 확인 |

migration 계정이 extension을 만들 수 없다면 DBA가 대상 database에서 먼저 실행합니다.

```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;
```

---

## 2. 설치

GitHub Release에는 서비스 이미지 `moina-v0.1.28.tar.gz` 하나만 올라가고 SHA256 값은 릴리스
본문에 적힙니다. 파일과 해시를 서로 다른 경로로 반입하세요.

### 2.1 이미지 반입과 확인

```bash
sha256sum moina-v0.1.28.tar.gz
gzip -t moina-v0.1.28.tar.gz
gzip -dc moina-v0.1.28.tar.gz | docker image load
docker image inspect moina:v0.1.28 --format '{{.Os}}/{{.Architecture}} {{.Config.User}}'
```

마지막 명령의 결과는 `linux/amd64 nonroot:nonroot`여야 합니다.

### 2.2 환경 파일 준비

```bash
cp .env.example .env
chmod 600 .env
```

`.env`에 [3장](#3-설정)의 네 값을 채웁니다. `MOINA_ENCRYPTION_KEY`는 다음처럼 만듭니다.

```bash
openssl rand -base64 32
```

### 2.3 기동

```bash
docker compose --env-file .env \
  -f deploy/docker-compose.offline.yml \
  up -d --pull never
```

`deploy/docker-compose.offline.yml`의 `image:` 줄에는 태그가 고정되어 있습니다. 반입한
태그(`moina:v0.1.28`)와 다르면 그 줄을 반입한 버전으로 맞춘 뒤 기동하세요. `pull_policy: never`
이므로 이미지가 로컬에 없으면 그 자리에서 실패합니다.

사설 CA가 필요하면 기관의 **전체** PEM CA bundle을 mount합니다. 추가 환경변수는 없습니다.

```bash
install -m 444 /secure/ca-certificates.crt deploy/certs/ca-certificates.crt
docker compose --env-file .env \
  -f deploy/docker-compose.offline.yml \
  -f deploy/docker-compose.private-ca.yml \
  up -d --pull never
```

> 사설 루트만 넣으면 기존 공개 TLS 검증이 실패할 수 있습니다. 운영에 필요한 공개 루트와
> 중간 인증서를 함께 포함하세요.

### 2.4 기동 확인

| 확인 | 메서드와 경로 |
| --- | --- |
| 프로세스 살아 있음 | `GET /healthz` |
| 요청 받을 준비 완료 | `GET /readyz` |
| 실행 중인 버전 | `GET /api/v1/version` |
| 지표 | `GET /metrics` |

```bash
curl --fail http://127.0.0.1:8080/healthz
curl --fail http://127.0.0.1:8080/readyz
curl --fail http://127.0.0.1:8080/api/v1/version
curl --fail http://127.0.0.1:8080/metrics
```

시작 migration은 대규모 검색 index 생성까지 고려해 **최대 30분**까지 실행되며, 끝나기 전에는
`/readyz`가 성공하지 않습니다. 배포 관리자의 startup 허용 시간도 여기에 맞추세요.

### 2.5 최초 관리자 계정

1. reverse proxy의 HTTPS 주소로 접속해 로그인 화면 아래의 버전이 방금 올린 버전인지
   확인합니다.
2. `MOINA_BOOTSTRAP_ADMIN`과 그 비밀번호로 로그인합니다. 이 계정은 `super_admin`과 `admin`
   역할을 함께 가집니다.
3. **설정 → 로그인 보안**에서 bootstrap 비밀번호를 즉시 바꿉니다.
4. 운영 관리자와 검토 역할을 따로 만들고 최소 권한만 부여합니다([4장](#4-계정과-권한)).
5. **감사 로그**에 최초 로그인·비밀번호 변경·사용자 생성이 남았는지 확인합니다.

![로그인 화면 — 카드 아래에 실행 중인 서비스 버전이 표시된다](assets/screenshots/desktop-login.webp)

> bootstrap 값은 **최초 계정을 만들기 위한** 값입니다. 계정이 이미 만들어진 뒤에 환경변수
> 비밀번호를 바꿔도 기존 계정은 재설정되지 않습니다.

---

## 3. 설정

### 3.1 환경 변수 — 전부 네 개

서버가 읽는 환경 변수는 정확히 네 개이며, `scripts/check-runtime-contract.sh`가 이 계약을
CI에서 검사합니다. 나머지는 전부 PostgreSQL에 저장되고 관리 화면에서 바꿉니다.

| 이름 | 기본값 | 필수 | 설명 |
| --- | --- | --- | --- |
| `MOINA_POSTGRES_DSN` | 없음 | 필수 | 외부 PostgreSQL DSN. 운영에서는 TLS를 검증하는 설정(`sslmode=verify-full`)을 씁니다. 예: `postgres://moina:비밀값@postgres.internal:5432/moina?sslmode=verify-full` |
| `MOINA_BOOTSTRAP_ADMIN` | 없음 | 필수 | 최초 로컬 최고 관리자 아이디. 줄바꿈 없이 1~120자. 예: `moina-admin` |
| `MOINA_BOOTSTRAP_ADMIN_PASSWORD` | 없음 | 필수 | 최초 관리자 비밀번호. UTF-8 기준 12자 이상, 72바이트 이하. 비밀 관리 시스템으로 전달합니다. |
| `MOINA_ENCRYPTION_KEY` | 없음 | 필수 | 저장 비밀값을 보호하는 32바이트 root key(base64). `openssl rand -base64 32`로 만들고 DB backup과 **다른 곳**에 보관합니다. |

> `MOINA_ENCRYPTION_KEY`를 잃으면 암호화된 OIDC·AI·SMTP 비밀과 기존 session·API key 검증
> 정보를 복구할 수 없습니다. `v0.1.28`은 온라인 root key 교체를 제공하지 않으므로 값을 바꾸지
> 마세요.

### 3.2 일반 설정

**서비스 관리자 → 일반 설정**에서 서비스 공통 정책을 관리합니다. 저장하면 변경 내용과
수행자가 감사 로그에 남습니다(비밀값 원문은 남지 않습니다). 설정 캐시는 `pg_notify`로 다른
인스턴스에 곧바로 전파되며, 알림을 놓친 경우에도 30초 안에 최신 값으로 바뀝니다.

![일반 설정 — 서비스 기본, API·MCP, 이미지·영상 업로드, Reverse Proxy 신뢰 정책, 검토·승인 프로세스](assets/screenshots/desktop-admin-settings.webp)

| 구역 | 항목 | 캡처 시점의 값 |
| --- | --- | --- |
| 서비스 기본 | 서비스 이름 | `moina` |
| 서비스 기본 | 사이트 기본 주소 | 사용자가 접근하는 path 없는 origin. OIDC callback URL 생성에 씁니다. 비우면 현재 요청 주소를 사용합니다. |
| 서비스 기본 | 세션 유지 시간(분) | `720` |
| 서비스 기본 | 기본 시간대 | `Asia/Seoul` |
| 서비스 기본 | 사용자 가입 허용 | 꺼짐(기본). 끄면 관리자 생성 또는 OIDC 자동 등록만 허용합니다. |
| API 및 MCP | 개인 API 키 인증 | 켜짐 |
| API 및 MCP | Streamable HTTP MCP | 켜짐 |
| API 및 MCP | 키별 분당 요청 한도 | `120` |
| 이미지·영상 업로드 | 파일당 최대 용량(MiB) | `10` (1~50) |
| 이미지·영상 업로드 | 모인당 최대 미디어 | `4` (1~12) |
| 이미지·영상 업로드 | 미사용 업로드 정리 시간 | `24`시간 (1~720) |
| Reverse Proxy 신뢰 정책 | 신뢰 Proxy IP 또는 CIDR | 비어 있음 |
| 검토·승인 프로세스 | 팀장 검토·승인 사용 | 꺼짐 |

몇 가지 고정 규칙은 화면에서 바꿀 수 없습니다.

- 허용 형식은 JPEG·PNG·GIF·WebP 이미지와 MP4·WebM 동영상입니다. HEIC·HEIF는 거절하며,
  거절 응답에 아이폰에서 JPEG으로 바꾸는 방법을 함께 안내합니다.
- 사용자별 **미첨부 미디어 100개·512 MiB** quota는 고정 한도입니다. 넘으면
  `media_quota_exceeded`가 표시되고, 기존 업로드를 글·프로필에 연결하거나 정리 시간을
  기다려야 합니다.
- 정리 worker는 매시간 500개씩 최대 20 batch, 인스턴스당 시간당 최대 10,000개를 지웁니다.
- 작성 화면은 `GET /api/v1/media/config`에서 현재 최대 byte·개수·허용 MIME을 읽습니다. 관리자
  전용 정리 시간과 고정 quota는 이 응답에 들어 있지 않습니다.
- 요청 본문 읽기 기한은 미디어 업로드 15분, 그 밖의 요청 30초, header 10초입니다. reverse
  proxy의 timeout과 body 크기 제한도 최대 업로드 크기에 맞추세요.
- 로그인·가입과 개인 키 요청 한도는 PostgreSQL에서 계산하므로 여러 인스턴스가 하나의
  quota를 공유합니다.

**Reverse Proxy 신뢰 정책**에는 MOINA에 직접 연결하는 정확한 IP 또는 CIDR만 등록합니다.
호스트 이름과 와일드카드는 허용하지 않고, 비워 두면 전달 header를 모두 무시합니다.
등록된 Peer의 header만 신뢰하며 주소 chain은 오른쪽부터 검증합니다. 감사 로그에는 소켓 IP,
계산된 실제 Client IP와 Proxy Chain이 각각 남습니다.

### 3.3 보존 기간

일반 설정의 보존 기간(`service.retention`)으로 정리 주기를 정합니다. 기본값은 다음과 같으며
`0`은 그 테이블을 정리하지 않는다는 뜻입니다.

| 대상 | 기본 보존 |
| --- | --- |
| 감사 기록 | 무기한(`0`) |
| 알림 | 90일 |
| 전달을 마친 Outbox event | 14일 |
| AI 사용 기록 | 180일 |

기동 직후와 매시간, 한 인스턴스가 만료 session과 보존 기간이 지난 행을 테이블당 최대
5,000행씩 20회까지 정리합니다. Dead Letter Outbox event는 보존 기간과 무관하게 남습니다.

### 3.4 Keycloak OIDC

![Keycloak OIDC — Issuer·Client 정보, 허용 Host, 실제 Redirect URI 확인과 연결 테스트](assets/screenshots/desktop-admin-oidc.webp)

1. Keycloak Client에서 Standard flow와 PKCE(`S256`)를 허용합니다. Confidential client는 Client
   authentication을 켜고, Public client는 끕니다.
2. **Issuer URL**과 **Client ID**를 입력합니다. Confidential client만 **Client Secret**을 넣고
   Public client는 비웁니다. 예전에 Secret을 저장한 Public client는 저장된 Secret 삭제를 켭니다.
3. 화면 아래 **실제 로그인 요청 Redirect URI**에 표시된 주소를 Keycloak의 Valid redirect URIs에
   공백 없이 그대로 등록합니다. **고급 Redirect URI 직접 지정**은 특별한 경우가 아니면 비웁니다.
4. **OIDC 허용 Host**에 issuer와 discovery가 쓰는 정확한 DNS 이름 또는 `host:port`를 줄바꿈·쉼표로
   구분해 등록합니다. port가 없는 항목은 scheme 기본 port에만 일치합니다.
5. DNS가 RFC1918·ULA 사설 주소를 반환하면 **사설망 OIDC Host**에 그 정확한 DNS 이름을 함께
   넣고 **Issuer 사설망 연결 허용**을 켭니다. IP literal은 넣을 수 없고 loopback·link-local·
   metadata·CGNAT은 항상 차단됩니다.
6. Scopes는 `openid profile email`부터 시작하고 기본 역할, 역할 Claim 경로, 역할 매핑 JSON과
   **첫 로그인 사용자 자동 등록**을 정합니다.
7. **저장 후 연결 테스트**로 discovery·authorization redirect·token endpoint 인증·DNS/IP·TLS를 모두
   확인한 다음 **Keycloak 로그인 사용**을 켭니다.

Client secret은 저장 후 다시 보여 주지 않고 설정 여부만 표시합니다. 폐쇄망 HTTP는 **폐쇄망
HTTP 허용**을 명시적으로 켠 등록 host에만 허용됩니다.

오류 코드는 다음과 같이 읽습니다.

| 코드 | 뜻 |
| --- | --- |
| `oidc_client_auth_failed` | Keycloak의 Client authentication 설정과 저장된 Secret/Public client 설정이 어긋남 |
| `oidc_code_rejected` | 만료·재사용된 코드, PKCE·Redirect URI 불일치 또는 서버 시간 문제 |

> 긴급 접근을 위해 로컬 최고 관리자 계정을 최소 한 개 유지하고, 강한 비밀번호와 접근 통제를
> 적용하세요.

### 3.5 AI 스트리밍

![AI 설정 — OpenAI 호환 공급자 연결과 AI 기능 사용 스위치](assets/screenshots/desktop-admin-ai.webp)

내부 OpenAI 호환 endpoint의 base URL, API key, model, API style, timeout, token 정책과 허용
호스트를 저장합니다. API key는 `MOINA_ENCRYPTION_KEY`로 암호화되고 조회 응답에 반환되지
않습니다.

- output token 허용 범위는 1~262,144이지만 실제 model 상한 이하로 설정합니다.
- 허용 호스트에는 정확한 DNS/IP 또는 `host:port`만 등록합니다. RFC1918·ULA로 해석되는
  내부 endpoint는 전체 허용과 사설망 허용 목록에 함께 넣습니다. 차단되는 주소 범주는
  OIDC와 같습니다.
- 요청마다 DNS를 다시 해석해 검증한 IP로 연결하고 모든 redirect를 다시 검사합니다.
- **저장 후 연결 테스트**는 저장된 설정으로 `/models` 응답을 확인합니다.
- streaming(SSE)을 reverse proxy가 buffering하지 않도록 하세요.

AI 기능을 켜면 `ai:use` 권한이 있는 사용자에게만 메뉴가 보입니다. host allowlist와 별개로
outbound 방화벽에서도 같은 목적지만 허용하세요.

### 3.6 SMTP 메일

![SMTP 메일 설정 — 메일 전달 서버, 이메일 알림 사용, 저장 후 테스트 메일](assets/screenshots/desktop-admin-smtp.webp)

1. 서버 DNS 이름과 port를 나눠 입력합니다.
2. STARTTLS를 기본으로 쓰고 공급자가 요구할 때만 TLS(SMTPS)를 고릅니다. 암호화 없음은
   폐쇄망 무인증 relay에서만 선택합니다.
3. 사용자 이름·password와 보내는 이메일·이름을 입력합니다. Password는 암호화되어 조회
   응답과 감사 로그에 원문이 나오지 않습니다.
4. DNS가 사설 주소를 반환하는 내부 서버만 사설망 SMTP 허용을 켭니다. 입력한 정확한
   `host:port` 하나만 열립니다.
5. **저장 후 테스트 메일**로 현재 관리자 프로필 이메일에 실제 메일이 도착하는지 확인합니다.
6. **이메일 알림 사용**을 켜면 사용자가 개인 알림 설정에서 이메일 채널을 고를 수 있습니다.

메일은 별도 `notification.email` Outbox event로 보내므로 SMTP 장애가 Moin 게시를 실패시키지
않습니다. 실패는 지수 backoff 뒤 Dead Letter에 남고, 원인을 해결한 다음 감사 로그의 **실패
이벤트 복구**에서 재처리합니다. 사설 CA를 쓰는 SMTP TLS는 기관 CA bundle에 root와
intermediate를 모두 포함해야 합니다.

### 3.7 선택형 검토·승인 정책

![검토·승인 — 승인 프로세스가 꺼져 있을 때의 안내](assets/screenshots/desktop-admin-approvals.webp)

승인 정책이 꺼져 있으면 제출·검토·승인·반려 단계와 승인 메뉴가 화면에서 완전히 빠집니다.
켤 때는 일반 설정의 **검토·승인 프로세스**에서 지정 Action만 고릅니다.

- Action 패턴은 전체 `*`, 정확한 dot Action, 마지막 segment만 wildcard인 `post.*` 형태만
  허용합니다. `post*`, `post.`, `post:*`, `*.publish`, `post..publish`는 거절됩니다.
- 모든 approver 역할에 최종 유효 `approvals:review`, `approvals:*` 또는 `*` 권한이 있어야 하며,
  하나라도 어긋나면 저장이 거부됩니다.
- 요청자의 자기 승인·반려는 금지되고, 승인 대기 중인 Moin은 수정할 수 없습니다.
- 승인 대상과 Pending 요청 접근 권한은 요청마다 다시 검사하므로 권한을 회수한 사용자는
  즉시 검토할 수 없습니다.
- 승인·반려 event는 수행자·시각·대상·결과·검토 comment와 함께 감사 로그에 남습니다.

---

## 4. 계정과 권한

### 4.1 사용자 관리

![사용자 관리 — 로컬·OIDC 사용자 목록과 역할 변경, 비활성화](assets/screenshots/desktop-admin-users.webp)

목록에서 인증 방식(로컬·OIDC), 역할, 상태와 등록일을 확인하고 역할을 바꾸거나 계정을
비활성화합니다. 최고 관리자 계정은 최고 관리자만 바꿀 수 있고(`super_admin_protected`),
마지막 최고 관리자는 변경할 수 없습니다(`last_super_admin`).

### 4.2 역할과 권한

![역할·권한 — 역할별 기능 권한 선택과 권한 저장](assets/screenshots/desktop-admin-roles.webp)

기본 역할과 권한은 다음과 같습니다. 역할 이름과 권한 묶음은 이 화면에서 바꿀 수 있고,
서버는 화면 숨김이 아니라 **모든 API 요청에서** 권한을 다시 검사합니다.

| 역할 | 설명 | 기본 권한 |
| --- | --- | --- |
| `super_admin` | 전체 서비스 관리 | `*` (보호되어 변경·삭제 불가) |
| `admin` | 서비스 운영 관리 | `admin:access`, `users:manage`, `posts:manage`, `moderation:manage`, `approvals:review`, `roles:manage`, `settings:manage`, `audit:read`, `outbox:manage`, `keys:manage`, `posts:read`, `posts:write`, `social:write`, `ai:use`, `mcp:use` |
| `team_lead` | 검토 및 승인 | `approvals:review`, `posts:read`, `ai:use`, `mcp:use` |
| `moderator` | 신고 및 콘텐츠 운영 | `moderation:manage`, `posts:manage`, `posts:read`, `ai:use` |
| `member` | 일반 사용자 | `posts:read`, `posts:write`, `social:write`, `ai:use`, `mcp:use` |

권한이 뜻하는 것은 다음과 같습니다.

| 권한 | 할 수 있는 일 |
| --- | --- |
| `admin:access` | 서비스 관리자 영역 진입 |
| `users:manage` | 사용자 생성·역할 변경·비활성화·비밀번호 초기화 |
| `posts:manage` | 콘텐츠 관리에서 Moin 조회·수정·삭제 |
| `moderation:manage` | 신고 검토와 제재 처리 |
| `approvals:review` | 검토·승인 요청 승인·반려 |
| `roles:manage` | 역할과 권한 편집 |
| `settings:manage` | 일반·OIDC·AI·SMTP·승인 설정 변경 |
| `audit:read` | 감사 로그와 실패 이벤트 조회 |
| `outbox:manage` | 실패 이벤트 재처리 |
| `keys:manage` | 다른 사용자 API·MCP 키 조회·회수 |
| `posts:read` / `posts:write` | Moin 읽기 / 쓰기 |
| `social:write` | Link·Signal·Pocket·신고 등 사회적 활동 |
| `ai:use` / `mcp:use` | AI 메뉴 사용 / MCP endpoint 사용 |

**조사 역할과 복구 역할을 분리하세요.** 실패 이벤트 *조회*는 `audit:read`, 상태를 바꾸는
*재처리*는 `outbox:manage`로 나뉘어 있습니다.

권한이 없는 화면을 열면 사용자에게는 다음 화면이 보입니다. 사용자가 이 화면을 신고해 오면
역할과 유효 권한을 확인해 주세요.

![접근 권한이 없습니다 — 권한 없는 화면을 열었을 때 사용자에게 보이는 화면](assets/screenshots/desktop-access-denied.webp)

### 4.3 개인 API·MCP 키

키는 소유자별로 hash만 저장하고 원문은 생성·회전 직후 한 번만 표시됩니다. 만료와 최근 사용
시각이 함께 기록되므로 장기 미사용 키와 만료 없는 키를 주기적으로 검토하세요. MCP도 같은
권한 체계와 요청 한도를 씁니다. 역할을 바꾸면 활성 session과 키의 유효 권한에 즉시
반영됩니다.

---

## 5. 운영

### 5.1 관리 대시보드

![관리 대시보드 — 사용자·모인·신고·승인 현황과 API·PostgreSQL·실시간 알림·AI 상태](assets/screenshots/desktop-admin-dashboard.webp)

전체 사용자, 오늘의 모인, 처리 대기 신고, 승인 대기 수와 함께 API·PostgreSQL·실시간 알림·AI
공급자의 상태를 한 화면에서 봅니다. 관리 활동은 모두 감사 로그에 남습니다.

### 5.2 콘텐츠와 신고

![콘텐츠 관리 — 내용·작성자로 검색하고 상태를 확인한 뒤 삭제한다](assets/screenshots/desktop-admin-content.webp)

**콘텐츠 관리**에서 내용이나 작성자로 Moin을 찾아 상태(`published` 등)와 신고 건수를 확인하고
필요할 때 삭제합니다.

![신고·제재 — 접수된 신고를 검토·해결·기각한다](assets/screenshots/desktop-admin-reports.webp)

**신고·제재**에서는 접수된 신고를 검토해 해결하거나 기각하고, 필요하면 사용자 활성 상태를
바꾸거나 게시물을 지웁니다. `v0.1.28`은 신고 유형·우선순위·SLA를 사용자 정의하는 설정을
제공하지 않으므로, 기관별 기준과 escalation은 별도 운영 정책으로 관리하고 검토 메모에
판단 근거를 남기세요. 반복 패턴은 관찰하되 자동 판단만으로 영구 제재하지 않습니다.

### 5.3 감사 로그와 실패 이벤트 복구

![감사 로그 — 실패 이벤트 복구 영역과 시각·수행자·작업·대상·결과·IP 기록](assets/screenshots/desktop-admin-audit.webp)

감사 로그는 로그인, 관리 설정, 권한과 콘텐츠 조치를 변경 불가능한 기록으로 남기고 수행자,
작업(`auth.login`, `post.create`, `profile.preferences.update` 등), 대상, 결과와 IP를 함께
보여 줍니다.

Dead Letter로 떨어진 비동기 event는 같은 화면 위쪽 **실패 이벤트 복구**에서 처리합니다.

1. event 종류, 대상, 시도 횟수와 마지막 오류를 확인합니다.
2. PostgreSQL·권한·대상 데이터 같은 원인을 **먼저** 해결합니다.
3. 재처리를 눌러 대기열로 되돌리고 Outbox 지연과 감사 기록을 확인합니다.

원인을 해결하지 않은 반복 재처리는 피하세요.

### 5.4 관측

`GET /metrics`는 Prometheus text 형식으로 Flow·검색 지연, Flow 요청당 SQL 수, Outbox 지연과
실패, DB pool과 WebSocket 연결 수를 제공합니다. 이 endpoint는 reverse proxy에서 운영
수집망에만 허용하세요. 응답의 `X-Request-ID`는 JSON 로그의 `request_id`와 같은 값이므로
장애 추적에 함께 씁니다. 컨테이너 로그는 `docker logs`(compose 기본 json-file, 10 MiB×5)로
봅니다.

알림 경로에는 안전 장치가 있습니다. LISTEN consumer는 채널이 포화되면 backpressure를
적용하고, 느린 browser socket은 연결을 끊어 최대 30초 backoff로 재연결하게 하며, client는
연결 직후와 60초마다 REST로 미확인 수를 다시 맞춥니다. 일시적인 WebSocket 재연결만으로
알림이 유실됐다고 판단하지 마세요.

### 5.5 백업과 복구

- Large Object를 포함한 PostgreSQL 전체 backup을 만듭니다. 논리 backup은 blob 포함 옵션
  (`pg_dump -b` 등)을 명시합니다.
- 같은 시점의 `MOINA_ENCRYPTION_KEY`와 현재 이미지 `tar.gz`를 **서로 다른 보안 영역**에
  보관합니다.
- 복구 훈련에서 미디어·DB backup·원래 key가 함께 유효한지 확인합니다. `v0.1.28`은 온라인
  root key 교체를 제공하지 않습니다.

### 5.6 업그레이드와 되돌리기

1. release notes와 migration을 검토하고 DB backup을 만듭니다.
2. 서비스 중단 전에 **로컬 bootstrap 최고 관리자 로그인**을 확인합니다.
3. 새 `tar.gz`의 SHA256을 확인하고 이미지를 load합니다.
4. staging에서 최대 30분 migration, 로그인, Flow·검색·알림과 rollback을 검증합니다.
5. compose의 image tag를 바꾸고 재시작한 뒤 `/readyz`, 버전과 `/metrics`를 확인합니다.

적용된 migration의 SHA-256 checksum이 달라지면 서버가 시작을 거부합니다. DB에 현재 binary가
모르는 migration version이 있어도 downgrade guard가 기동을 거부합니다. 과거 SQL이나
`schema_migrations`를 고치지 말고 DB schema와 맞는 정식 이미지를 쓰세요. **되돌리기는 이전
앱이 새 schema를 읽을 수 있다고 확인한 경우에만** 수행합니다.

버전별 주의 사항은 [오프라인 운영 가이드](operations.md)에 정리되어 있습니다. 특히 `v0.1.0`에서
올라오는 배포는 사설 OIDC·AI host가 자동으로 사설망 허용 목록에 들어가지 않으므로, 업그레이드
뒤 로컬 관리자로 각 정확한 hostname을 전체·사설망 허용 목록에 저장하고 연결 테스트를 다시
실행해야 합니다.

---

## 6. 장애 대응

| 증상 | 확인할 곳 | 조치 |
| --- | --- | --- |
| `/readyz` 실패 | 컨테이너 로그, `schema_migrations` | 최대 30분 migration이 진행 중인지, 알 수 없는 migration·downgrade인지, PostgreSQL 계정·권한·checksum 문제인지 구분합니다. |
| Flow 새로고침이 429 | 응답 code `feed_snapshot_busy`, `Retry-After: 1` | 같은 계정의 반복 refresh를 멈추게 안내합니다. |
| 로그인·가입·API 키 요청이 503 | 응답 code `rate_limit_unavailable` | PostgreSQL 연결과 `rate_limit_buckets` migration을 확인합니다. |
| 감사 로그의 Client IP가 이상함 | 감사 로그의 소켓 IP·Client IP·Proxy Chain | 직접 연결하는 Peer의 IP/CIDR이 신뢰 Proxy 목록에 있는지 확인합니다. |
| OIDC callback 실패 | `oidc_client_auth_failed`, `oidc_code_rejected` | Issuer·port·discovery·Redirect URI·DNS/CA와 서버 시간을 확인합니다. `v0.1.0` 업그레이드라면 사설 host를 명시 저장합니다. |
| AI 응답(SSE)이 끊김 | AI 허용 host, reverse proxy 설정 | 전체·사설망 허용 host와 port, DNS/CA, proxy의 buffering·timeout을 확인합니다. |
| WebSocket 연결 실패 | proxy 설정 | `Upgrade` header 전달, idle timeout, origin 정책을 확인합니다. 느린 client는 재연결과 60초 reconcile로 회복합니다. |
| Outbox 지연 | `moina_outbox_lag_seconds`, 실패 이벤트 | DB lock과 원인을 해결한 뒤 재처리합니다. |
| 대용량 업로드가 중단됨 | proxy request timeout·body 제한 | 서버 기한은 업로드 15분입니다. proxy 값을 최대 업로드 크기에 맞춥니다. |
| 미디어 업로드 거부 | 응답 code, `GET /api/v1/media/config` | MIME·파일 byte·개수와 미첨부 100개·512 MiB quota를 확인합니다. |
| 미디어 용량이 계속 증가 | 정리 로그, Large Object backup | 미사용 업로드 정리 시간과 시간당 최대 10,000개 정리 한도를 확인합니다. |
| 복호화 오류 | 주입된 `MOINA_ENCRYPTION_KEY` | 올바른 key인지 확인하고 임의 교체를 중단합니다. |

장애 기록에는 버전, image digest, 발생 시각, `X-Request-ID`, 화면 오류 code, 관련 감사 event
ID와 영향 범위를 남기고 DSN·비밀번호·token·암호화 key는 남기지 않습니다.

---

## 7. 보안

기본값 중 배포 전에 반드시 손봐야 하는 것들입니다.

- **bootstrap 비밀번호** — 최초 로그인 직후 화면에서 바꿉니다. 환경변수만 바꾸는 것으로는
  이미 만들어진 계정이 바뀌지 않습니다.
- **`.env` 권한** — `chmod 600`. 비밀 관리 시스템에서 주입하는 쪽이 더 낫습니다.
- **`MOINA_ENCRYPTION_KEY`** — DB backup과 다른 보안 영역에 보관합니다.
- **PostgreSQL DSN** — 운영에서는 TLS를 검증하는 `sslmode`를 씁니다.

외부에 열지 말아야 할 것들입니다.

- **컨테이너 포트 8080** — 기본 compose대로 `127.0.0.1`에만 bind하고 앞단 TLS proxy를 통해서만
  노출합니다.
- **`GET /metrics`** — 운영 수집망에서만 접근하게 proxy에서 제한합니다.
- **PostgreSQL** — 서비스 호스트에서만 접근하게 방화벽으로 막습니다.

인증과 연동에서 지킬 것들입니다.

- 로컬 최고 관리자 계정을 최소 한 개 유지해 SSO 장애 시 접근 경로를 남깁니다.
- OIDC·AI·SMTP의 허용 host는 정확한 authority만 등록합니다. wildcard·scheme·경로는 넣지
  않으며, loopback·link-local·cloud metadata·CGNAT·unspecified·multicast는 항상 차단됩니다.
- 폐쇄망 HTTP는 명시적으로 켠 등록 host에만 허용합니다.
- 최소 권한 원칙으로 역할을 나누고, 특히 `audit:read`(조사)와 `outbox:manage`(복구),
  `settings:manage`(설정)를 서로 다른 역할에 둡니다.
- 컨테이너는 read-only·non-root·`cap_drop: ALL`·`no-new-privileges`로 실행합니다. 기본
  compose가 이미 그렇게 되어 있으니 임의로 완화하지 마세요.

---

## 더 보기

- [사용자 가이드](USER_GUIDE.md) — 화면 사용법(사용자에게 안내할 문서)
- [설정](configuration.md) · [오프라인 운영](operations.md) · [보안과 키 관리](security.md)
- [REST API와 MCP](api-mcp.md) · [OpenAPI 3.1](../api/openapi.yaml)
- [아키텍처](architecture.md)
