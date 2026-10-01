# mmcp 관리자 가이드

**mmcp v1.0.0 관리자 가이드** — Mattermost MCP Gateway

이 문서는 폐쇄망(오프라인) 환경에 mmcp v1.0.0을 설치하고 운영하는 관리자를 위한 안내서입니다. 일반 사용자의 MCP 클라이언트 연결 방법은 콘솔의 **내 공간 > MCP 연결 가이드** 화면을 참고하세요.

---

## 목차

1. [개요와 아키텍처](#1-개요와-아키텍처)
2. [설치 (오프라인)](#2-설치-오프라인)
3. [첫 설정 체크리스트](#3-첫-설정-체크리스트)
4. [Keycloak 연동](#4-keycloak-연동)
5. [Mattermost 연동](#5-mattermost-연동)
6. [사용자·역할](#6-사용자역할)
7. [계정 매핑과 충돌 처리](#7-계정-매핑과-충돌-처리)
8. [MCP 도구 정책](#8-mcp-도구-정책)
9. [데이터·호출 정책](#9-데이터호출-정책)
10. [클라이언트·세션](#10-클라이언트세션)
11. [키 관리와 회전](#11-키-관리와-회전)
12. [접근 제어](#12-접근-제어)
13. [감사 로그·승인 이력](#13-감사-로그승인-이력)
14. [서버 로그](#14-서버-로그)
15. [백업·복원·업그레이드](#15-백업복원업그레이드)
16. [모니터링 엔드포인트](#16-모니터링-엔드포인트)
17. [문제 해결](#17-문제-해결)
18. [보안 체크리스트](#18-보안-체크리스트)

---

## 1. 개요와 아키텍처

mmcp는 Claude Code, Codex, Cursor, 사내 AI 포털 같은 MCP 클라이언트가 **각 사용자 본인의 Mattermost 권한으로** 메시지를 조회·검색하고 작성할 수 있게 해 주는 게이트웨이입니다.

- **인증**: Keycloak OIDC(사일런트 SSO), mmcp 자체 OAuth 2.1 인가 서버, Keycloak 액세스 토큰 직접 검증, 개인 API 키를 지원합니다.
- **권한**: 조회는 항상 사용자 본인의 Mattermost 자격증명으로 수행하므로 Mattermost 채널 멤버십이 그대로 적용됩니다. 그 위에 mmcp 역할, 도구 정책, 채널 유형 정책이 한 겹 더 적용됩니다.
- **구성**: 단일 컨테이너(React 콘솔 + Go 서버)와 PostgreSQL만 있으면 됩니다. 환경변수는 4개뿐이며, 나머지 설정은 모두 관리 콘솔에서 관리하고 PostgreSQL에 저장합니다.

```mermaid
flowchart LR
    subgraph Clients["MCP 클라이언트"]
        CC["Claude Code / Codex / Cursor"]
        Portal["사내 AI 포털"]
    end
    Browser["브라우저<br/>(관리 콘솔 · 내 공간)"]

    subgraph mmcp["mmcp v1.0.0 컨테이너 :8080"]
        MCP["/mcp<br/>Streamable HTTP"]
        OAuth["/oauth/*<br/>OAuth 2.1 인가 서버"]
        Console["관리 콘솔 / API"]
        Policy["역할 · 도구 정책<br/>채널 정책 · 호출 한도 · 승인"]
        Keys["봉투 암호화<br/>마스터 키 → 데이터 키"]
    end

    KC["Keycloak<br/>(OIDC)"]
    MM["Mattermost<br/>REST API v4"]
    PG[("PostgreSQL<br/>설정 · 매핑 · 감사")]

    CC -- "OAuth 토큰 / API 키" --> MCP
    Portal -- "Keycloak 액세스 토큰" --> MCP
    CC -. "DCR · PKCE" .-> OAuth
    Browser --> Console
    Browser -. "로그인 (prompt=none)" .-> KC
    OAuth -. "사용자 인증" .-> KC
    MCP --> Policy
    Policy -- "사용자 본인 토큰<br/>(조회·작성)" --> MM
    Policy -- "서비스 토큰<br/>(디렉터리 조회·PAT 발급)" --> MM
    Policy -- "봇 토큰<br/>(봇 발신)" --> MM
    Keys --- PG
    Console --- PG
```

### 콘솔 구성

콘솔은 **관리 콘솔**(admin 역할 전용)과 **내 공간**(모든 사용자)으로 나뉩니다.

| 그룹 | 관리 콘솔 메뉴 |
|---|---|
| 개요 | 대시보드 |
| 아이덴티티 | 사용자, 계정 매핑, Keycloak 연동, Mattermost 연동 |
| MCP | 도구 정책, 데이터·호출 정책, 클라이언트·세션 |
| 보안 | 키 관리, API 키 현황, 웹 세션, 접근 제어 |
| 감사 | 감사 로그, 승인 이력 |
| 시스템 | 서버 로그, 일반 설정, 백업·복원, 시스템 정보 |

내 공간에는 내 홈, MCP 연결 가이드, 내 API 키, Mattermost 연결, 개인 암호화 키, 도구 실행기, 승인 요청, 내 활동, 세션·연결된 앱, 프로필·환경설정이 있습니다.

![대시보드](screenshots/dashboard.webp)

### 키보드 단축키

| 키 | 동작 |
|---|---|
| `Ctrl+K` / `⌘+K` 또는 `/` | 명령 팔레트 열기 (페이지·작업 검색, 초성 검색 지원: 예 `ㄷㅅㅂㄷ`) |
| `?` | 단축키 도움말 |
| `Shift+T` | 라이트/다크 테마 전환 |
| `G` → `D` / `U` / `M` / `T` / `K` / `A` / `L` / `S` | 대시보드 / 사용자 / 계정 매핑 / 도구 정책 / 키 관리 / 감사 로그 / 서버 로그 / 일반 설정 |
| `G` → `H` / `C` / `I` / `R` / `P` | 내 홈 / MCP 연결 가이드 / 내 API 키 / 도구 실행기 / 프로필·환경설정 |

![명령 팔레트](screenshots/command-palette.webp)

콘솔은 다크 모드와 모바일 화면을 지원합니다.

| 다크 모드 | 모바일 | 모바일 내비게이션 |
|---|---|---|
| ![다크 대시보드](screenshots/dark-dashboard.webp) | ![모바일 대시보드](screenshots/mobile-dashboard.webp) | ![모바일 내비게이션](screenshots/mobile-nav.webp) |

---

## 2. 설치 (오프라인)

### 2.1 사전 준비

| 항목 | 요구 사항 |
|---|---|
| 컨테이너 런타임 | Docker + Docker Compose |
| 데이터베이스 | PostgreSQL (별도 준비. `compose.offline.yml`에는 PostgreSQL 서비스가 포함되어 있지 않습니다) |
| 네트워크 | mmcp → PostgreSQL, Keycloak, Mattermost 접근 가능 / 사용자·MCP 클라이언트 → mmcp `8080` 접근 가능 |
| 릴리스 파일 | `mmcp-v1.0.0.tar.gz`, `릴리스 노트의 SHA-256 값` ([GitHub Releases](https://github.com/hkjang/mmcp/releases)에서 받아 반입) |

이미지 특성(mmcp v1.0.0):

- 이미지 태그 `mmcp:v1.0.0`, `linux/amd64`, distroless `static-debian12:nonroot` 기반, 비루트 사용자로 실행
- 포트 `8080`, `HEALTHCHECK`는 `/mmcp healthcheck` (컨테이너 내부에서 `http://127.0.0.1:8080/api/health` 호출)
- 읽기 전용 루트 파일시스템 지원 (`compose.offline.yml`은 `read_only: true`, `/tmp` tmpfs, `no-new-privileges`, `cap_drop: ALL`로 실행)
- 시작 시 DB 마이그레이션을 자동 적용

### 2.2 이미지 반입과 검증

```bash
# 무결성 확인 (GitHub 릴리스 노트에 적힌 SHA-256 값과 같아야 합니다)
sha256sum mmcp-v1.0.0.tar.gz   # 릴리스 노트의 SHA-256 값과 비교

# 이미지 로드 → "Loaded image: mmcp:v1.0.0"
docker load -i mmcp-v1.0.0.tar.gz

# 네트워크 없이 실행되는지 확인
docker run --rm --network none mmcp:v1.0.0 version
```

### 2.3 환경변수 (.env)

mmcp가 읽는 환경변수는 정확히 4개입니다. 그 외 모든 설정은 관리 콘솔에서 합니다.

| 변수 | 필수 | 설명 |
|---|---|---|
| `POSTGRES_DSN` | ✔ | PostgreSQL 접속 문자열. `DATABASE_URL`도 별칭으로 인식합니다. |
| `BOOTSTRAP_ADMIN` | ✔ | 최초 관리자 계정 이름. 첫 기동 때 한 번 생성됩니다. |
| `BOOTSTRAP_ADMIN_PASSWORD` | ✔ | 최초 관리자 비밀번호 (8자 이상). 이후 재기동 때 비밀번호를 덮어쓰지 않습니다. |
| `ENCRYPTION_KEY` | ✔ | 마스터 키. 32바이트 키(base64 또는 hex) 또는 32자 이상 패스프레이즈. 쉼표로 여러 개를 나열하면 첫 번째가 주 키, 나머지는 복호화 전용입니다(키 회전용). |

키 생성:

```bash
docker run --rm mmcp:v1.0.0 gen-key
```

`.env` 예시 (`.env.example` 기반):

```dotenv
POSTGRES_DSN=postgres://mmcp:change-me@postgres:5432/mmcp?sslmode=disable
BOOTSTRAP_ADMIN=admin
BOOTSTRAP_ADMIN_PASSWORD=Change-Me-Now-2026!
ENCRYPTION_KEY=<gen-key 출력값>
```

> **중요**: `ENCRYPTION_KEY`를 잃어버리면 DB에 암호화되어 저장된 비밀값(Keycloak Client Secret, Mattermost 서비스/봇 토큰, 사용자 Mattermost 토큰)을 복구할 수 없습니다. 키는 DB 백업과 **분리해서** 안전하게 보관하세요.

### 2.4 기동

```bash
docker compose -f compose.offline.yml up -d
docker compose -f compose.offline.yml ps      # health: healthy 확인
docker logs mmcp | head                        # "mmcp ready" 로그 확인
```

### 2.5 CLI 하위 명령

이미지의 엔트리포인트는 `/mmcp`이며, 다음 하위 명령을 지원합니다.

| 명령 | 용도 |
|---|---|
| `serve` (기본) | 서버 실행. `-addr` 플래그 기본값은 `:8080` (컨테이너 HEALTHCHECK가 8080을 가정하므로 변경하지 않는 것을 권장) |
| `healthcheck` | `/api/health`가 200이면 0으로 종료 (컨테이너 HEALTHCHECK용) |
| `version` | 버전·커밋·빌드 시각 출력 (예: `mmcp v1.0.0 (...)`) |
| `gen-key` | 32바이트 무작위 키를 base64로 출력 |
| `reset-admin-password` | `BOOTSTRAP_ADMIN` 계정의 비밀번호를 `BOOTSTRAP_ADMIN_PASSWORD`로 재설정하고 잠금 해제·활성화 (비상 복구용) |

---

## 3. 첫 설정 체크리스트

브라우저에서 `http://<호스트>:8080`에 접속해 부트스트랩 관리자로 로그인합니다. 부트스트랩 관리자에게는 `admin`, `mcp-user`, `mcp-writer` 역할이 부여됩니다.

![로그인](screenshots/login.webp)

| 순서 | 위치 | 작업 |
|---|---|---|
| 1 | 일반 설정 | **Public URL** 저장 (사용자·MCP 클라이언트가 접속하는 외부 주소. 리버스 프록시 뒤라면 프록시 주소). 비어 있으면 요청 Host로 추정하며 대시보드에 경고가 표시됩니다. |
| 2 | 접근 제어 | 리버스 프록시 사용 시 **X-Forwarded-For / X-Forwarded-Host 신뢰** 켜기 |
| 3 | Keycloak 연동 | Issuer URL, Client ID, Client Secret 입력 → **연결 테스트** (Discovery·JWKS 확인) → 저장 |
| 4 | Mattermost 연동 | API Base URL, 서비스 계정 토큰, (선택) 봇 토큰, 발신 계정 방식 설정 → **연결 테스트** → 저장 |
| 5 | 계정 매핑 | **미매핑 사용자 자동 매핑** 실행, 실패 항목은 수동 매핑 |
| 6 | 도구 정책 | 조직 정책에 맞게 도구 활성화·승인 모드·역할 조정 |
| 7 | 데이터·호출 정책 | DM/그룹/비공개 채널 허용 범위, 호출 한도, OAuth 모드 확인 |
| 8 | 접근 제어 | 관리 콘솔·MCP IP 허용목록, 로그인 잠금 정책 |
| 9 | 백업·복원 | 초기 설정을 내보내기로 보관 |
| 10 | 내 공간 > 도구 실행기 | `mattermost_me` 등을 직접 실행해 동작 확인 |

![일반 설정](screenshots/general-settings.webp)

**일반 설정** 화면 항목

| 항목 | 기본값 | 범위/비고 |
|---|---|---|
| Public URL | (없음) | http(s) 절대 URL. MCP 엔드포인트는 `{Public URL}/mcp` |
| 로그인 화면 공지 | (없음) | 점검 안내, 문의처 등 |
| 감사 로그 보존 기간 | 180일 | 1~3650일 |
| 서버 로그 레벨 | info | debug / info / warn / error |
| Prometheus /metrics 노출 | 켜짐 | 관리자 IP 허용목록이 함께 적용 |

---

## 4. Keycloak 연동

![Keycloak 연동](screenshots/keycloak.webp)

### 4.1 Keycloak 클라이언트 만들기

Realm에 OIDC 클라이언트를 만들고 다음과 같이 설정합니다. (`scripts/dev/keycloak-realm.json`에 개발용 예시가 있습니다.)

| Keycloak 설정 | 값 |
|---|---|
| Client ID | 예: `mmcp` |
| Client authentication | **ON** (confidential) |
| Standard flow | **ON**, PKCE 메서드 **S256** |
| Direct access grants | OFF 권장 |
| Valid redirect URI | `{Public URL}/auth/oidc/callback` |
| Valid post logout redirect URI | `{Public URL}/login?logged_out=1` |
| Access token 수명 | 5~15분 권장 |
| Realm 역할 | `mcp-user`, `mcp-writer`, `mcp-file`, `mcp-channel-admin`, `mcp-admin` |
| Audience 매퍼 (권장) | `oidc-audience-mapper`, Included Client Audience = `mmcp`, access token에 포함 |

Audience 매퍼는 Keycloak 액세스 토큰을 `/mcp`에 직접 Bearer로 보내는 경우(사내 AI 포털 등)에 필요합니다. mmcp는 토큰의 `aud` 또는 `azp`가 허용 목록(기본: Client ID)에 있어야 받아들이므로, 다른 클라이언트가 발급받은 토큰을 쓰려면 매퍼로 `aud`에 mmcp Client ID를 넣거나 **허용 audience / azp**에 해당 값을 추가하세요.

### 4.2 mmcp 설정 항목

| 구역 | 항목 | 기본값 | 설명 |
|---|---|---|---|
| 연결 | Keycloak SSO 사용 | 꺼짐 | |
| | Issuer URL | | 예: `https://sso.company.co.kr/realms/company` (켜면 필수) |
| | Client ID / Client Secret | | Secret은 시스템 데이터 키로 암호화 저장, 화면에는 `********` 표시. 바꿀 때만 입력 |
| | Scopes | `openid profile email` | `openid`가 없으면 자동 추가 |
| | TLS 인증서 검증 생략 | 꺼짐 | 테스트 전용 |
| | 사설 CA 인증서 (PEM) | | 사내 CA로 서명된 Keycloak 사용 시 |
| 사용자·역할 | 사용자명 클레임 | `preferred_username` | Mattermost username과 매칭할 클레임 |
| | 클레임이 없으면 이메일 앞부분 사용 | 꺼짐 | |
| | 첫 로그인 시 mmcp 사용자 자동 생성 | 켜짐 | JIT 프로비저닝 |
| | 로그인마다 Keycloak 역할 동기화 | 켜짐 | |
| | 관리자로 승격할 Keycloak 역할 | `mcp-admin` | 이 역할 보유자는 mmcp `admin`이 됨 |
| | 모든 SSO 사용자에게 부여할 기본 역할 | `mcp-user` | |
| 로그인 경험 | 사일런트 SSO | 켜짐 | Keycloak 세션이 있으면 화면 없이 로그인 (`prompt=none`) |
| | 로그인 화면 대신 Keycloak으로 바로 이동 | 꺼짐 | 사일런트 SSO 실패 시 Keycloak 로그인 화면으로 |
| | 로그인 버튼 문구 | `Keycloak SSO로 로그인` | |
| | 로그아웃 시 Keycloak 세션도 종료 | 켜짐 | end_session_endpoint 사용 |
| MCP 토큰 직접 인증 | Keycloak 액세스 토큰을 /mcp Bearer로 허용 | 켜짐 | JWKS로 서명·만료·발급자 검증 |
| | 허용 audience / azp | (비움 = Client ID) | |

### 4.3 로그인 동작

- **사일런트 SSO**: 브라우저 최상위 리다이렉트(`prompt=none`)로 동작하므로 서드파티 쿠키 차단의 영향을 받지 않습니다. Keycloak 세션이 없으면 조용히 로그인 화면으로 돌아옵니다.
- **JIT 사용자 생성**: 첫 로그인 시 Keycloak `sub`를 고정 키로 mmcp 사용자를 만듭니다. 같은 username의 로컬 계정이 이미 있고 Keycloak 연결이 없다면 그 계정에 `sub`를 연결합니다. 자동 생성이 꺼져 있으면 미등록 사용자는 로그인이 거부됩니다.
- **역할 매핑** (로그인마다 동기화):
  - realm 역할, 이 클라이언트의 client 역할, `roles`/`groups` 클레임(그룹은 앞의 `/` 제거) 중 이름이 `mcp-`로 시작하는 것은 그대로 mmcp 역할이 됩니다.
  - **관리자로 승격할 Keycloak 역할**에 해당하면 `admin` 역할이 추가됩니다.
  - **기본 역할**은 모든 SSO 사용자에게 부여됩니다.
  - Keycloak에서 온 역할(IdP 역할)과 콘솔에서 직접 준 역할(수동 역할)은 합쳐서 적용됩니다.
- 로그인 직후 백그라운드에서 Mattermost 계정 매핑을 미리 수행해 첫 MCP 호출을 빠르게 합니다.

---

## 5. Mattermost 연동

![Mattermost 연동](screenshots/mattermost.webp)

### 5.1 권한 모델

| 작업 | 사용하는 자격증명 |
|---|---|
| 조회·검색·DM 읽기 | **사용자 본인 토큰** — Mattermost 채널 멤버십 그대로 적용 |
| 사용자 찾기·매핑 | 서비스 토큰 (사용자 디렉터리 조회만) |
| 개인 토큰 자동 발급 | 서비스 토큰 (system_admin) → 사용자 명의 PAT |
| 메시지·DM 발송 | 선택한 발신 계정 + 승인 정책 |

- 서비스 토큰은 **메시지 조회에 절대 사용되지 않습니다.** 시스템 관리자 계정의 개인 액세스 토큰(PAT)을 권장합니다.
- mmcp는 **Mattermost 비밀번호를 받거나 저장하지 않습니다.** 비밀번호 기반 세션 로그인은 지원하지 않습니다.
- 개인 토큰 자동 발급을 쓰려면 Mattermost 시스템 콘솔에서 **개인 액세스 토큰 사용(EnableUserAccessTokens)** 을 켜야 합니다.

### 5.2 설정 항목

| 구역 | 항목 | 기본값 | 설명 |
|---|---|---|---|
| 서버 | API Base URL | | `/api/v4`는 붙이지 않아도 됨 (붙여도 자동 제거) |
| | Site URL (퍼머링크용) | | 비우면 API URL 사용 |
| | 타임아웃(초) | 15 | 1~120 |
| | 매핑 검증 캐시(분) | 10 | 1~1440. 이 시간 동안은 매핑 재검증 생략 |
| | TLS 검증 생략 / 사설 CA 인증서 | 꺼짐 / 없음 | |
| 서비스 계정 | 서비스 계정 액세스 토큰 | | 암호화 저장, `********` 표시 |
| | 사용자 개인 토큰 자동 발급 | 꺼짐 | 켜려면 서비스 토큰 필수 |
| | username이 없으면 이메일로 매칭 | 꺼짐 | |
| | 자동 발급 토큰 설명 | `mmcp gateway (auto-issued)` | Mattermost 토큰 목록에 `설명 · 날짜`로 표시 |
| 발신 계정 | 누구 명의로 보낼까요? | 자동(`auto`) | 아래 표 참고 |
| | 봇 계정 액세스 토큰 | | `bot` 선택 시 필수 |
| | 봇 DM을 어느 대화방에 넣을까요? | 봇 ↔ 수신자(`bot_pair`) | |
| | 요청자 표시 문구 | `_{display_name}({username})님의 요청으로 AI Assistant가 전송한 메시지입니다._` | `{display_name}`, `{username}`, `{mmcp_username}` 치환. 비우면 문구 없이 props에만 기록 |

**연결 테스트**는 `system/ping`, 서비스 토큰 계정(그리고 system_admin 여부), 봇 토큰 계정을 확인합니다. 서비스 계정이 system_admin이 아니면 사용자 조회만 가능하고 토큰 자동 발급은 불가합니다.

### 5.3 발신 계정 선택표

| 방식 | Mattermost에 표시되는 작성자 | 필요한 것 | 사용자 자격증명이 없을 때 | 권장 상황 |
|---|---|---|---|---|
| 사용자 본인 (`user`) | 실제 사용자 | 사용자 PAT(직접 등록 또는 자동 발급) | 발송 실패 (`CREDENTIAL_REQUIRED` 등) | 본인 명의 발송이 필수인 조직 |
| 봇 계정 (`bot`) | 봇("AI Assistant") + 요청자 표시 문구 | 봇 토큰. 채널 게시에는 봇의 `post:channels` 또는 `post:all` 권한 또는 채널 멤버십 | 영향 없음 | 사용자 PAT를 쓰지 않는 조직 |
| 자동 (`auto`, 기본·권장) | 자격증명이 있으면 본인, 없으면 봇 | 사용자 PAT 및/또는 봇 토큰 | `CREDENTIAL_REQUIRED`·`IDENTITY_NOT_MAPPED`·`CREDENTIAL_INVALID`이면 봇으로 대체 | 대부분의 경우 |

모든 게시물에는 provenance props(`mcp`, `source=mmcp`, `requested_by`, `requested_by_mmcp`, `mmcp_request_id`, `mmcp_client`)가 기록됩니다.

**봇 DM 대화방** (`bot`/`auto`에서 봇으로 DM을 보낼 때)

| 값 | 동작 | 요구 권한 |
|---|---|---|
| `bot_pair` (권장) | 봇 ↔ 수신자 DM에 게시 | 일반 봇 권한 |
| `requester_pair` | 요청자 ↔ 수신자 DM에 봇이 게시 | 서비스 토큰 **system_admin** + 봇 **post:all** (부족하면 `PERMISSION_DENIED`) |

---

## 6. 사용자·역할

![사용자](screenshots/users.webp)

### 6.1 역할

| 역할 | 의미 |
|---|---|
| `admin` | mmcp 서비스 관리자. 관리 콘솔 접근, **모든 도구 역할 검사 통과** |
| `mcp-user` | 조회·검색 도구 (read, search, dm_read, dm_search 범주 기본 역할) |
| `mcp-writer` | 메시지 작성·답글·수정·반응, DM 발송 (write, dm_write 범주 기본 역할) |
| `mcp-file` | 파일 조회·첨부 게시 (file 범주 기본 역할) |
| `mcp-channel-admin` | 채널 생성, 멤버 추가 |
| `mcp-admin` | **모든 도구 역할 검사 통과** (위험 도구 포함) |

- `mcp-*` 역할을 하나라도 가지면 기본 조회 권한(`mcp-user`)이 함께 인정됩니다. 예를 들어 `mcp-writer`만 가진 사용자도 조회 도구를 쓸 수 있지만, `mcp-file`·`mcp-channel-admin` 도구는 해당 역할이 따로 필요합니다. `admin`, `mcp-admin`은 모든 역할 검사를 통과합니다.
- 도구 정책에서는 위 역할 외에 `mcp-`로 시작하는 사용자 정의 역할도 지정할 수 있습니다.

### 6.2 사용자 관리

사용자 화면에서 할 수 있는 작업: 로컬 사용자 생성, 수동 역할 변경, 활성/비활성 전환, 비밀번호 재설정, 잠금 해제, 세션 일괄 종료, 삭제.

- 각 사용자의 역할은 **수동 역할 ∪ IdP 역할**입니다. IdP 역할은 Keycloak 로그인 시 동기화되므로, Keycloak 사용자의 권한은 Keycloak에서 관리하는 것이 원칙입니다.
- 본인 계정과 부트스트랩 관리자는 비활성화·삭제하거나 `admin` 역할을 제거할 수 없습니다.
- 부트스트랩 관리자는 기동할 때마다 활성 상태와 `admin` 역할이 보장됩니다(비밀번호는 유지).

---

## 7. 계정 매핑과 충돌 처리

> **로컬 계정과 Keycloak 자동 연결 규칙**: Keycloak 사용자명과 같은 로컬 계정이 있을 때, 그 계정이 **비밀번호도 관리자 역할도 없는 SSO 전용 계정**이면 첫 로그인 때 자동 연결됩니다. 비밀번호·`admin` 역할이 있는 계정이나 부트스트랩 관리자는 계정 탈취를 막기 위해 자동 연결하지 않습니다.
>
> **DM 정책**: DM·그룹 메시지는 DM 전용 도구(`mattermost_read_dm`, `mattermost_send_dm` 등)로만 다룰 수 있습니다. 일반 채널 도구에 DM 채널 ID를 넣어도 `POLICY_BLOCKED`로 거부되므로, DM 도구의 활성화·승인·역할 정책이 항상 적용됩니다.

![계정 매핑](screenshots/mappings.webp)

### 7.1 매핑 규칙

1. **최초 탐색**: mmcp username(Keycloak 사용자명 클레임, 기본 `preferred_username`)과 같은 Mattermost username을 **서비스 토큰**으로 찾습니다. Mattermost 설정의 **이메일로 매칭**이 켜져 있으면 username이 없을 때 이메일로 찾습니다. 단, 이메일 매칭은 Keycloak이 `email_verified=true`로 확인한 Keycloak 사용자에게만 적용되며, 사용자가 직접 바꿀 수 있는 로컬 계정 이메일은 절대 사용하지 않습니다. 비활성 계정·봇 계정은 매핑하지 않습니다.
2. **고정**: 찾은 뒤에는 Keycloak `sub`(mmcp 사용자) ↔ Mattermost `user_id`가 고정됩니다. 이후 username이 같아도 다른 `user_id`로 바뀌어 연결되지 않습니다.
3. **주기적 재검증**: 매핑 검증 캐시 시간이 지나면 매핑된 `user_id`가 여전히 존재·활성인지, 그리고 username이 여전히 같은 `user_id`를 가리키는지 확인합니다.
4. **사용자 직접 등록**: 사용자가 내 공간 > Mattermost 연결에서 PAT를 등록할 수 있습니다. 토큰 소유자가 로그인 사용자(username 또는 이메일 일치) 또는 관리자가 수동 매핑한 계정과 다르면 거부(`IDENTITY_CONFLICT`)됩니다.

### 7.2 매핑 상태

| 상태 | 의미 | MCP 호출 결과 |
|---|---|---|
| `ok` | 정상 매핑 | 정상 |
| `pending` | 매핑 대기 (서비스 토큰 없음 등) | `IDENTITY_NOT_MAPPED` / `CREDENTIAL_REQUIRED` |
| `not_found` | Mattermost에 해당 사용자가 없거나 비활성/봇 | `IDENTITY_NOT_MAPPED` |
| `conflict` | 식별 충돌 감지 → **차단** | `IDENTITY_CONFLICT` |
| `disabled` | 관리자가 연동 비활성화 | `IDENTITY_DISABLED` |

### 7.3 충돌 감지

다음 경우 자동으로 `conflict` 상태가 되어 해당 사용자의 모든 Mattermost 호출이 차단되고, 감사 로그(`mapping` / `conflict`)와 서버 경고 로그가 남습니다.

- 매핑된 Mattermost 계정이 삭제되어 더 이상 존재하지 않음
- 같은 username이 다른 `user_id`를 가리킴 (계정 삭제 후 재생성, 다른 계정이 그 username으로 변경)
- (서비스 토큰이 없을 때) 저장된 토큰의 소유자 ID가 매핑된 ID와 다름
- 이미 다른 mmcp 사용자와 매핑된 Mattermost 계정에 연결하려 함

자동 경로는 절대 다른 계정으로 재연결하지 않습니다.

### 7.4 충돌 해결 절차

1. 계정 매핑 화면에서 충돌 사용자의 사유(`last_error`)를 확인합니다.
2. Mattermost에서 올바른 계정인지 확인합니다.
3. 다음 중 하나를 수행합니다.
   - **재검증**: 관리자가 명시적으로 다시 찾도록 허용합니다. 다른 계정으로 재연결되는 경우 **이전 계정의 저장 자격증명은 삭제**(mmcp가 자동 발급한 토큰이면 Mattermost에서 폐기)됩니다.
   - **수동 매핑**: username이 서로 다른 경우 Mattermost username을 직접 지정합니다. 이전과 다른 계정이면 기존 자격증명을 삭제합니다. 서비스 토큰이 없으면 `pending`으로 저장되고 사용자가 그 계정의 PAT를 등록하면 완료됩니다.

그 밖의 작업: **미매핑 사용자 자동 매핑**(활성 사용자 중 미매핑·`pending`·`not_found` 대상, 최대 1000명), 연동 비활성화/활성화, 사용자 토큰 발급·회전, 매핑 삭제(저장 토큰도 삭제).

---

## 8. MCP 도구 정책

![도구 정책](screenshots/tools.webp)

### 8.1 정책 요소

각 도구마다 다음을 설정합니다.

- **활성화**: 꺼진 도구는 목록에 나타나지 않고 호출 시 `TOOL_DISABLED`
- **모드**: `auto`(즉시 실행) 또는 `confirm`(2단계 승인). 읽기 도구에는 `confirm`을 쓸 수 없습니다.
- **역할**: 나열된 역할 중 **하나라도** 가진 사용자만 사용 가능 (없으면 `PERMISSION_DENIED`)

추가로 다음 조건이 적용됩니다.

- **정책 게이트**: 데이터·호출 정책에 따라 도구 자체가 막힐 수 있습니다(아래 표의 "정책 조건").
- **읽기 전용 자격증명**: 읽기 전용 API 키나 `mcp:read` 범위의 OAuth 토큰으로는 쓰기 도구를 쓸 수 없습니다.
- 도구 정책은 첫 기동 때 기본값으로 생성되며, 이후에는 콘솔에서 바꾼 값이 유지됩니다. 화면에서 도구별 최근 24시간 호출·오류 수와 평균 지연 시간도 확인할 수 있습니다.

### 8.2 2단계 승인 (confirm 모드)

1. 첫 호출(`confirm_token` 없음) → `APPROVAL_REQUIRED` 오류와 함께 미리보기(`preview`), `confirm_token`, 만료 시각을 반환합니다.
2. AI 클라이언트가 미리보기를 사용자에게 보여 주고, 사용자가 동의하면 **같은 인자**에 `confirm_token`을 더해 다시 호출합니다.
3. 사용자는 **내 공간 > 승인 요청**에서 대기 중인 요청을 승인·거절할 수도 있습니다.

| 결과 코드 | 의미 |
|---|---|
| `APPROVAL_REJECTED` | 사용자가 거절함 |
| `APPROVAL_EXPIRED` | 승인 유효시간(기본 10분) 경과 |
| `APPROVAL_MISMATCH` | 승인 후 인자가 바뀜 |
| `APPROVAL_USED` | 이미 실행된 승인 |
| `APPROVAL_INVALID` | 존재하지 않거나 다른 사용자/도구의 토큰 |

### 8.3 도구 목록과 기본값 (28종)

| 도구 | 설명 | 범주 | 기본 활성 | 기본 모드 | 기본 역할 | 호출 한도 버킷 | 정책 조건 |
|---|---|---|---|---|---|---|---|
| `mattermost_me` | 내 Mattermost 정보 | read | ✔ | auto | mcp-user | read | |
| `mattermost_teams` | 내 팀 목록 | read | ✔ | auto | mcp-user | read | |
| `mattermost_channels` | 내 채널 목록 | read | ✔ | auto | mcp-user | read | |
| `mattermost_channel` | 채널 상세 | read | ✔ | auto | mcp-user | read | |
| `mattermost_search_posts` | 메시지 검색 | search | ✔ | auto | mcp-user | search | |
| `mattermost_get_post` | 메시지 조회 | read | ✔ | auto | mcp-user | read | |
| `mattermost_get_thread` | 스레드 조회 | read | ✔ | auto | mcp-user | read | |
| `mattermost_recent_posts` | 최근 메시지 | read | ✔ | auto | mcp-user | read | |
| `mattermost_users` | 사용자 검색 | read | ✔ | auto | mcp-user | read | |
| `mattermost_unread` | 읽지 않은 메시지 | read | ✔ | auto | mcp-user | read | |
| `mattermost_pinned_posts` | 고정 메시지 | read | ✔ | auto | mcp-user | read | |
| `mattermost_get_file` | 첨부 파일 조회 | file | ✔ | auto | mcp-file | file | 첨부 파일 내용 조회 허용 (기본 꺼짐) |
| `mattermost_post_message` | 채널에 메시지 작성 | write | ✔ | **confirm** | mcp-writer | write | |
| `mattermost_reply` | 스레드 답글 | write | ✔ | **confirm** | mcp-writer | write | |
| `mattermost_update_post` | 메시지 수정 | write | ✔ | **confirm** | mcp-writer | write | |
| `mattermost_add_reaction` | 이모지 반응 | write | ✔ | auto | mcp-writer | write | |
| `mattermost_upload_file` | 파일 첨부 게시 | file | ✔ | **confirm** | mcp-file | file | |
| `mattermost_create_channel` | 채널 생성 | channel_admin | ✔ | **confirm** | mcp-channel-admin | write | |
| `mattermost_add_member` | 채널 멤버 추가 | channel_admin | ✔ | **confirm** | mcp-channel-admin | write | |
| `mattermost_delete_post` | 메시지 삭제 | destructive | ✘ | **confirm** | mcp-admin | write | |
| `mattermost_archive_channel` | 채널 보관 | destructive | ✘ | **confirm** | mcp-admin | write | |
| `mattermost_send_dm` | DM 보내기 | dm_write | ✔ | **confirm** | mcp-writer | write | DM 또는 그룹 허용 (발송은 DM 허용 필요) |
| `mattermost_send_group_dm` | 그룹 DM 보내기 (2~7명) | dm_write | ✔ | **confirm** | mcp-writer | write | DM 또는 그룹 허용 (발송은 그룹 허용 필요) |
| `mattermost_reply_dm` | DM 답장 | dm_write | ✔ | **confirm** | mcp-writer | write | DM 또는 그룹 허용 |
| `mattermost_upload_dm_file` | 파일 포함 DM | dm_write | ✔ | **confirm** | mcp-writer, mcp-file | write | DM 또는 그룹 허용 |
| `mattermost_read_dm` | DM 대화 조회 | dm_read | ✔ | auto | mcp-user | read | DM 또는 그룹 허용 |
| `mattermost_recent_dms` | 최근 DM 목록 | dm_read | ✔ | auto | mcp-user | read | DM 또는 그룹 허용 |
| `mattermost_search_dm` | DM 검색 | dm_search | ✔ | auto | mcp-user | search | DM 검색 허용 (기본 꺼짐) |

- 쓰기 도구는 `mattermost_add_reaction`을 제외하고 모두 기본 **confirm** 모드입니다.
- 위험 도구(`mattermost_delete_post`, `mattermost_archive_channel`)는 기본 **비활성**입니다.
- `mattermost_upload_dm_file`은 `mcp-writer` 또는 `mcp-file` 중 하나만 있어도 사용할 수 있습니다.

**호환 별칭** (데이터·호출 정책에서 켰을 때만 노출, 정책은 원본 도구를 따름)

| 별칭 | 원본 도구 |
|---|---|
| `dm` | `mattermost_send_dm` |
| `group_message` | `mattermost_send_group_dm` |
| `create_post` | `mattermost_post_message` |
| `search_users` | `mattermost_users` |

MCP 리소스(`mattermost://me`, `teams`, `channels`, `unread`, `channels/{channel_id}`, `threads/{post_id}`, `users/{username}`, `dms/{username}`)와 프롬프트(`summarize_channel`, `find_decisions`, `daily_digest`, `draft_dm`)도 제공되며, 리소스 읽기는 해당 도구의 정책·한도·감사를 그대로 거칩니다.

---

## 9. 데이터·호출 정책

![데이터·호출 정책](screenshots/policies.webp)

Mattermost 권한 위에 한 겹 더 거는 정책입니다. 사용자가 Mattermost에서 볼 수 있더라도 여기서 끄면 MCP로는 접근할 수 없습니다(`POLICY_BLOCKED`).

### 9.1 데이터 접근 범위

| 항목 | 기본값 | 비고 |
|---|---|---|
| 공개 채널 | 켜짐 | |
| 비공개 채널 | 켜짐 | |
| DM (읽기·발송) | 켜짐 | |
| 그룹 메시지 | 켜짐 | DM과 그룹을 모두 끄면 DM 계열 도구가 숨겨짐 |
| DM 검색 | **꺼짐** | 개인정보 보호를 위해 기본 꺼짐 |
| 첨부 파일 내용 조회 | **꺼짐** | `mattermost_get_file` 제어 |

### 9.2 결과·감사

| 항목 | 기본값 | 범위 |
|---|---|---|
| 메시지 최대 길이 | 4000자 | 200~100000 (초과분은 잘라서 반환) |
| 결과 최대 건수 | 100건 | 5~200 |
| 파일 최대 크기 | 1024 KB | 1~51200 |
| 승인 유효시간 | 10분 | 1~1440 |
| 콘솔 승인 필수 (엄격 모드) | 꺼짐 | 켜면 AI 대화 속 동의만으로는 실행되지 않고 사용자가 콘솔 **승인 요청** 화면에서 직접 승인해야 함(`APPROVAL_PENDING`). 위험 도구는 항상 이 방식 |
| 조회 결과에 신뢰불가 표시 | 켜짐 | 읽기 도구 결과를 `untrusted_content`로 감싸 프롬프트 인젝션 방어 |
| 감사 로그에 본문 저장 | **꺼짐** | 꺼져 있으면 발송 메시지는 글자 수만 기록 |
| 서버 안내문 (instructions) | (없음) | MCP `initialize` 응답의 instructions 뒤에 덧붙임 (최대 4000자) |

### 9.3 사용자별 분당 호출 한도

사용자 단위, 버킷별 1분 창으로 계산합니다. 초과 시 `RATE_LIMITED`와 재시도 대기 시간을 반환합니다.

| 버킷 | 기본값 | 해당 범주 |
|---|---|---|
| 조회 (read) | 60회/분 | read, dm_read |
| 검색 (search) | 20회/분 | search, dm_search |
| 작성 (write) | 10회/분 | write, dm_write, channel_admin, destructive |
| 파일 (file) | 5회/분 | file |

### 9.4 MCP OAuth

| 항목 | 기본값 | 설명 |
|---|---|---|
| 인가 서버 | mmcp 게이트웨이(`gateway`) | `keycloak` 선택 시 보호 리소스 메타데이터가 Keycloak Issuer를 가리킴 (Keycloak 연동이 켜져 있어야 함, 클라이언트별 Keycloak 등록 필요) |
| 동적 클라이언트 등록(DCR) | 켜짐 | `/oauth/register` (IP당 시간당 20회 제한) |
| 동의 기억 | 켜짐 | 같은 클라이언트·범위는 동의 화면 생략 |
| 액세스 토큰 수명 | 60분 | 5~1440 |
| 리프레시 토큰 수명 | 30일 | 1~365 |
| Mattermost 공식 MCP 호환 도구 이름 | 꺼짐 | `dm`, `group_message`, `create_post`, `search_users` |

게이트웨이 모드의 동작:

- 인가 코드 + **PKCE S256 필수**, 공개 클라이언트(`token_endpoint_auth_method=none`)만 발급
- redirect URI는 https, 루프백 http(포트 무관), 네이티브 앱용 사설 스킴만 허용
- **회전형 리프레시 토큰**: 사용할 때마다 새 토큰을 발급하고 이전 토큰은 폐기. 폐기된 리프레시 토큰이 재사용되면 탈취로 간주해 그 토큰 계열 전체를 폐기하고 감사 로그(`oauth` / `refresh_reuse`, `TOKEN_REUSE`)를 남깁니다.
- 범위: `mcp`(전체) / `mcp:read`(읽기 전용)

### 9.5 API 키·키 회전 정책

| 항목 | 기본값 | 범위 |
|---|---|---|
| 최대 유효기간 | 365일 | 1~3650 |
| 사용자당 최대 키 | 10개 | 1~100 |
| 만료일 필수 | 꺼짐 | |
| 회전 권장 주기 | 90일 | API 키·개인 데이터 키 공통 |

개인 API 키(`mmcp_pk_` 접두사)는 사용자가 내 공간 > 내 API 키에서 발급합니다. 읽기 전용 옵션이 있고, 회전 시 이전 키를 유예 시간(최대 7일) 동안 계속 쓸 수 있게 할 수 있습니다.

---

## 10. 클라이언트·세션

![클라이언트·세션](screenshots/clients.webp)

### 10.1 MCP 엔드포인트와 인증

- 엔드포인트: `{Public URL}/mcp` (Streamable HTTP, `POST`/`DELETE`. 서버 발신 SSE 스트림은 제공하지 않음)
- 지원 프로토콜 버전: `2025-11-25`, `2025-06-18`, `2025-03-26`, `2024-11-05`
- 인증 실패 시 `401`과 `WWW-Authenticate: Bearer ... resource_metadata="{Public URL}/.well-known/oauth-protected-resource/mcp"`를 반환합니다.

| Bearer 형식 | 인증 방식 | 비고 |
|---|---|---|
| `mmcp_at_...` | 게이트웨이 OAuth 액세스 토큰 | Claude Code, Codex, Cursor 등 (DCR + PKCE) |
| `mmcp_pk_...` | 개인 API 키 | 읽기 전용 옵션, 유예 회전 |
| JWT (`xxx.yyy.zzz`) | Keycloak 액세스 토큰 직접 | Keycloak 연동의 "Bearer로 허용"이 켜져 있어야 함 |

### 10.2 화면

- **OAuth 클라이언트** 탭: 동적 등록(DCR) 클라이언트 목록, 사용자 수, 활성 토큰 수, 마지막 사용. **허용** 스위치로 차단하면 해당 클라이언트의 토큰이 즉시 무효가 되고, 삭제하면 클라이언트와 발급 토큰이 함께 삭제됩니다.
- **MCP 세션** 탭: 클라이언트 이름·버전, 인증 방식, 프로토콜, IP, 호출 수. 세션을 종료하면 클라이언트는 다시 `initialize`해야 합니다. 24시간 활동이 없으면 자동 종료, 30일 지난 세션은 삭제됩니다.

### 10.3 웹 세션과 API 키

| 웹 세션 | API 키 현황 |
|---|---|
| ![웹 세션](screenshots/web-sessions.webp) | ![API 키 현황](screenshots/api-keys-admin.webp) |

- **웹 세션**: 콘솔 로그인 세션(인증 방식, IP, User-Agent)을 확인하고 강제 종료합니다. 세션 유효시간은 접근 제어에서 설정합니다(기본 12시간).
- **API 키 현황**: 전체 사용자의 API 키 상태(활성/만료/폐기, 회전 주기 초과 여부, 마지막 사용 IP)를 보고 폐기할 수 있습니다.

---

## 11. 키 관리와 회전

![키 관리](screenshots/key-management.webp)

### 11.1 봉투 암호화 구조

```
ENCRYPTION_KEY (마스터 키, 환경변수 — DB에 저장되지 않음)
 ├─ 시스템 데이터 키 (AES-256-GCM)  → Keycloak Client Secret, Mattermost 서비스/봇 토큰
 └─ 사용자별 데이터 키 (AES-256-GCM) → 사용자의 Mattermost 토큰
```

- 데이터 키는 마스터 키로 감싸서(wrap) DB에 저장되며, 감쌀 때 키 ID·소유자를 AAD로 묶어 다른 행으로 옮겨 쓸 수 없습니다.
- 각 암호문도 용도별 AAD(예: 사용자 ID)에 묶입니다.
- API 키·OAuth 토큰은 원문이 아닌 해시로만 저장됩니다.

화면 상단 지표: 주 마스터 키, 재래핑 필요 데이터 키, 회전 주기 초과 사용자 키, 자동 발급 Mattermost 토큰 수.

### 11.2 회전 작업

| 작업 | 위치 | 동작 |
|---|---|---|
| 사용자 키 회전 | 사용자 데이터 키 표 > **키 회전** | 새 데이터 키 버전 생성 → 해당 사용자의 토큰 재암호화 → 이전 버전 retired. 사용자도 내 공간 > 개인 암호화 키에서 직접 회전 가능 |
| 주기 초과 키 일괄 회전 | **주기 초과 키 일괄 회전** | 회전 권장 주기(기본 90일)를 넘긴 사용자 키 전부 회전 |
| 자동 발급 토큰 일괄 회전 | **MM 토큰 일괄 회전** | 자동 발급된 Mattermost 토큰을 새로 발급·저장한 뒤 이전 토큰을 Mattermost에서 폐기 (사용자 작업 중단 없음). 사용자가 직접 등록한 토큰은 대상이 아니며 사용자 본인이 교체해야 함 |
| 시스템 데이터 키 회전 | 시스템 데이터 키 > **회전** | 새 시스템 키 생성 후 비밀 설정 재암호화 |
| 재래핑 | 마스터 키 > **재래핑** | 주 마스터 키가 아닌 키로 감싸진 데이터 키를 모두 주 키로 다시 감쌈 |

### 11.3 런북: 마스터 키 회전 (무중단 절차)

```bash
# 1) 새 키 생성
docker run --rm mmcp:v1.0.0 gen-key

# 2) .env에서 새 키를 앞에, 이전 키를 뒤에 두고 재시작
#    ENCRYPTION_KEY="<새 키>,<이전 키>"
docker compose -f compose.offline.yml up -d

# 3) 관리 콘솔 > 키 관리 > 마스터 키 > "재래핑" 실행

# 4) "재래핑 필요 데이터 키"가 0인지 확인

# 5) .env에서 이전 키를 제거하고 재시작
#    ENCRYPTION_KEY="<새 키>"
docker compose -f compose.offline.yml up -d
```

주의:

- 4단계 확인 전에 이전 키를 제거하면 해당 데이터 키로 암호화된 값을 복호화할 수 없습니다.
- 마스터 키 표에서 상태가 **없음**인 키 ID가 보이면, 그 키로 감싼 데이터 키가 남아 있는데 `ENCRYPTION_KEY`에 그 키가 없다는 뜻입니다.
- 재시작 직전에 시작된 Keycloak 로그인 진행 중 요청은 실패할 수 있으니(로그인 상태 쿠키 서명이 주 키에서 파생) 사용량이 적은 시간에 진행하세요.

---

## 12. 접근 제어

![접근 제어](screenshots/access-control.webp)

| 구역 | 항목 | 기본값 | 설명 |
|---|---|---|---|
| IP 허용목록 | 관리 콘솔·메트릭 허용 IP/CIDR | (비움 = 제한 없음) | 관리 API와 `/metrics`에 적용. 위반 시 `IP_DENIED` |
| | MCP 엔드포인트 허용 IP/CIDR | (비움 = 제한 없음) | `/mcp`에 적용 |
| | X-Forwarded-For / X-Forwarded-Host 신뢰 | 꺼짐 | 리버스 프록시가 있을 때만 켜기. 꺼져 있으면 프록시 IP가 클라이언트 IP로 기록됨 |
| CORS | MCP·OAuth 엔드포인트 허용 Origin | (없음) | 브라우저 기반 MCP 클라이언트의 Origin. Public URL과 같은 Origin은 자동 허용. 허용되지 않은 Origin의 `/mcp` 요청은 `ORIGIN_DENIED` (DNS 리바인딩 방어) |
| 로그인 정책 | 로컬 계정 로그인 허용 | 켜짐 | 꺼도 부트스트랩 관리자는 비상용으로 로그인 가능. Keycloak SSO가 꺼진 상태에서는 끌 수 없음 |
| | 연속 실패 허용 횟수 | 5회 | 1~100. 초과 시 계정 잠금 |
| | 잠금 시간 | 15분 | 1~1440 |
| | 웹 세션 유효시간 | 12시간 | 1~720 |

- IP 항목은 단일 IP(자동으로 `/32`, `/128`)나 CIDR로 입력합니다.
- **관리자 허용목록을 잘못 지정하면 관리 콘솔에 접근할 수 없게 됩니다.** 저장 전 현재 접속 IP가 포함되는지 확인하세요.
- 상태 변경 API는 `X-Requested-With: mmcp` 헤더를 요구해 CSRF를 막습니다.

---

## 13. 감사 로그·승인 이력

### 13.1 감사 로그

![감사 로그](screenshots/audit.webp)

| 범주 | 기록 대상 |
|---|---|
| `tool` | MCP 도구 호출(결과, 오류 코드, 지연, 클라이언트, 인증 방식, 발신 계정, 승인 ID), 사용자 승인/거절 |
| `auth` | 웹 로그인 성공/실패, Keycloak 계정 연결, JIT 생성, MCP 인증 실패 |
| `oauth` | 클라이언트 등록, 인가 승인/거부, 토큰 발급, 리프레시 토큰 재사용 감지 |
| `key` | Mattermost 토큰 발급·회전·등록, 데이터 키 회전, API 키 회전 |
| `mapping` | 계정 연결, 탐색 실패, 충돌 감지, 수동 매핑 |
| `admin` | 설정 변경, 사용자·매핑·키 관리 등 관리 작업 |

- 결과 값: `success` / `failure` / `denied` / `pending`
- 범주, 결과, 행위자, 기간, 자유 검색(작업·대상·오류 코드·클라이언트·Mattermost 사용자·요청 ID)으로 필터링
- **CSV 내보내기**: 현재 필터 기준 최대 50,000행, Excel용 BOM 포함. `=`, `+`, `-`, `@` 등으로 시작하는 값에는 `'`를 붙여 스프레드시트 수식 주입을 막습니다.
- **보존 기간**: 일반 설정의 감사 로그 보존 기간(기본 180일)이 지난 이벤트는 10분 주기 정리 작업에서 삭제됩니다.
- 모든 HTTP 응답의 요청 ID(`X-Request-Id`)로 감사 이벤트와 서버 로그를 서로 찾아볼 수 있습니다.

### 13.2 승인 이력

![승인 이력](screenshots/approvals-admin.webp)

쓰기 도구의 승인 요청을 사용자·도구·미리보기 요약·상태(pending / approved / rejected / executed / expired)·클라이언트 별로 확인합니다. 만료된 대기 요청은 정리 작업에서 `expired`로 바뀝니다.

---

## 14. 서버 로그

![서버 로그](screenshots/server-logs.webp)

- 서버는 JSON 구조화 로그를 표준 출력(`docker logs mmcp`)으로 내보내고, 최근 **5,000건**을 메모리에 보관해 콘솔에 보여 줍니다.
- **실시간 tail**: SSE 스트림으로 새 로그를 즉시 표시합니다. 리버스 프록시에서는 버퍼링을 끄세요(서버가 `X-Accel-Buffering: no`를 보냄).
- **레벨 필터**, **텍스트 검색**(메시지·속성)
- **다운로드**: 메모리 버퍼 전체를 NDJSON으로 저장
- **로그 레벨 변경**: debug / info / warn / error. 변경 즉시 적용되며 일반 설정에 저장되어 재기동 후에도 유지됩니다.
- 장기 보관이 필요하면 Docker 로깅 드라이버나 로그 수집기로 표준 출력을 수집하세요.

---

## 15. 백업·복원·업그레이드

![백업·복원](screenshots/backup.webp)

### 15.1 설정 백업 (콘솔)

- **내보내기**: 4개 설정 구역(general, keycloak, mattermost, mcp)과 도구 정책을 `mmcp-config/v1` 형식 JSON(`mmcp-config-<시각>.json`)으로 저장합니다. **비밀값은 포함되지 않습니다.**
- **가져오기**: 검증을 거쳐 구역별로 저장합니다. 백업 파일의 비어 있는 비밀값은 "변경 없음"으로 처리되어 기존 비밀값이 유지됩니다. 새 설치에 가져온 경우 Client Secret, 서비스/봇 토큰은 다시 입력하세요.
- 설정 백업에는 사용자, 매핑, 토큰, API 키, 감사 로그가 들어 있지 않습니다. 전체 백업은 DB 덤프로 하세요.

### 15.2 데이터베이스 백업과 복원

```bash
# 백업 (예시)
pg_dump "$POSTGRES_DSN" -Fc -f mmcp-$(date +%Y%m%d).dump

# 복원 (예시: 빈 DB에)
pg_restore -d "$POSTGRES_DSN" --clean --if-exists mmcp-YYYYMMDD.dump
```

- DB에는 마스터 키로 감싼 데이터 키만 있으므로, 복원한 DB를 쓰려면 **백업 당시와 같은 `ENCRYPTION_KEY`** (또는 그 키를 포함한 키 목록)가 필요합니다.
- DB 덤프와 `ENCRYPTION_KEY`는 서로 다른 위치에 보관하세요.

### 15.3 업그레이드

1. 설정 내보내기와 DB 덤프를 받습니다.
2. 새 릴리스 아카이브를 반입해 `sha256sum` 값을 릴리스 노트와 비교하고 `docker load` 합니다.
3. `compose.offline.yml`의 `image:` 태그(현재 `mmcp:v1.0.0`)를 새 태그로 바꿉니다.
4. `docker compose -f compose.offline.yml up -d` — 시작 시 DB 마이그레이션이 자동 적용되며 로그에 `migrations applied`가 남습니다.
5. 시스템 정보 화면에서 버전(현재 v1.0.0에서 바뀌었는지)과 마이그레이션을 확인합니다.

![시스템 정보](screenshots/about.webp)

### 15.4 비상 복구: 관리자 비밀번호 분실

부트스트랩 관리자의 비밀번호는 재기동해도 바뀌지 않습니다. 분실·잠금 시:

```bash
# .env의 BOOTSTRAP_ADMIN_PASSWORD를 새 값으로 바꾼 뒤
docker run --rm --env-file .env mmcp:v1.0.0 reset-admin-password
# → "password reset for admin"
```

비밀번호 재설정과 함께 잠금 해제·활성화가 이루어집니다. 컨테이너가 DB에 접근할 수 있는 네트워크에서 실행해야 합니다(필요하면 `--network` 지정).

---

## 16. 모니터링 엔드포인트

| 경로 | 인증 | 용도 |
|---|---|---|
| `/api/health` | 없음 | 생존 확인. `{"status":"ok","version":"v1.0.0",...}` |
| `/api/ready` | 없음 | 준비 상태. DB와 설정된 상위 시스템(Keycloak JWKS, Mattermost ping)이 응답해야 200, 아니면 503과 구성요소별 상태 |
| `/api/version` | 없음 | 빌드 정보 |
| `/metrics` | 관리자 IP 허용목록 | Prometheus 형식. 일반 설정에서 끄면 404 |
| `/api/openapi.json` | 없음 | OpenAPI 명세 |
| `/.well-known/oauth-protected-resource/mcp` | 없음 | MCP 보호 리소스 메타데이터 |
| `/.well-known/oauth-authorization-server` | 없음 | OAuth 인가 서버 메타데이터 (`/.well-known/openid-configuration`도 동일 응답) |

주요 메트릭:

| 메트릭 | 설명 |
|---|---|
| `mmcp_build_info` | 버전·커밋·Go 버전 |
| `mmcp_uptime_seconds` | 가동 시간 |
| `mmcp_http_requests_total{code}` | 상태 코드 계열별 HTTP 응답 수 |
| `mmcp_tool_calls_total` | 도구·결과별 호출 수 |
| `mmcp_tool_duration_seconds` | 도구 실행 시간 (`_sum`, `_count`) |
| `mmcp_logins_total` | 로그인 방식·결과별 횟수 |
| `mmcp_web_sessions` | 웹 세션 수 |
| `mmcp_db_connections` | DB 연결 수 |

권장 경보: `/api/ready` 503 지속, `mmcp_tool_calls_total`의 실패·거부 비율 급증, 감사 로그의 `IDENTITY_CONFLICT`·`TOKEN_REUSE` 발생.

---

## 17. 문제 해결

### 17.1 설치·접속

| 증상 | 원인 | 조치 |
|---|---|---|
| 컨테이너가 바로 종료, `configuration error` | 필수 환경변수 누락·형식 오류 | 로그 메시지 확인. `BOOTSTRAP_ADMIN_PASSWORD` 8자 이상, `ENCRYPTION_KEY`는 32바이트 키 또는 32자 이상 |
| `database` / `migration failed` 로그 | DB 접속 불가·권한 부족 | `POSTGRES_DSN`, 네트워크, DB 사용자 권한 확인 |
| 컨테이너 unhealthy | 서버 미기동 또는 8080 이외 포트로 실행 | `docker logs mmcp` 확인, `-addr` 변경 여부 확인 |
| 관리 콘솔 `IP_DENIED` | 관리자 IP 허용목록에 현재 IP 없음 | 허용된 IP에서 접속해 수정. 프록시 뒤라면 프록시 헤더 신뢰 설정 확인 |
| 로그인 시 `ACCOUNT_LOCKED` | 연속 실패로 잠김 | 사용자 화면에서 잠금 해제, 관리자는 `reset-admin-password` |
| MCP 메타데이터/리다이렉트 주소가 내부 주소로 나옴 | Public URL 미설정 | 일반 설정에서 Public URL 저장 |

### 17.2 Keycloak

| 증상 | 원인 | 조치 |
|---|---|---|
| 로그인 화면에 SSO 오류 (`sso=config`) | Discovery 실패, CA 미신뢰 | Keycloak 연동 > 연결 테스트, 사설 CA PEM 등록 |
| `sso=exchange` | Client Secret 불일치, redirect URI 미등록 | Secret 재입력, `{Public URL}/auth/oidc/callback` 등록 |
| `sso=state` | 로그인 상태 쿠키 없음·만료·서명 불일치 | 쿠키 허용 확인, 다시 로그인. 마스터 키 교체 직후라면 재시도 |
| `sso=provision` | 사용자명 클레임 없음, 자동 생성 꺼짐, 비활성 계정, 다른 Keycloak 계정에 이미 연결된 username | 클레임/매퍼 확인, 자동 생성 설정, 사용자 상태 확인 |
| Keycloak 토큰으로 `/mcp` 호출 시 `TOKEN_INVALID` | audience/azp 불일치, 직접 인증 꺼짐, 만료 | Audience 매퍼 추가 또는 허용 audience 등록, "Bearer로 허용" 켜기 |
| 역할이 반영되지 않음 | 역할 이름이 `mcp-`로 시작하지 않음, 역할 동기화 꺼짐 | Keycloak 역할 이름 확인, 사용자 재로그인 |

### 17.3 LLM 클라이언트가 받는 오류 코드

| 오류 코드 | 원인 | 조치 |
|---|---|---|
| `AUTH_REQUIRED` | Bearer 토큰 없음 | 클라이언트에서 OAuth 연결 또는 API 키 설정 |
| `TOKEN_INVALID` | 토큰 만료·폐기·형식 오류, 사용자 비활성, 클라이언트 차단 | 재연결, API 키 재발급, 클라이언트·세션 화면에서 차단 여부 확인 |
| `IDENTITY_NOT_MAPPED` | Mattermost 계정 없음·비활성·봇, 매핑 대기 | 계정 매핑 화면에서 자동/수동 매핑, username 일치 여부 확인 |
| `IDENTITY_CONFLICT` | 계정 삭제·재생성 또는 username 이동 감지, 타인 토큰 등록 시도 | [7.4 충돌 해결 절차](#74-충돌-해결-절차) |
| `IDENTITY_DISABLED` | 관리자가 연동 비활성화 | 계정 매핑에서 다시 활성화 |
| `CREDENTIAL_REQUIRED` | 사용자 토큰 없음, 서비스 토큰 없음, Mattermost PAT 기능 꺼짐 | 자동 발급 켜기(서비스 토큰 system_admin + EnableUserAccessTokens) 또는 사용자가 PAT 등록 |
| `CREDENTIAL_INVALID` | Mattermost가 토큰 거부(401), 또는 복호화 실패 | 토큰 회전/재등록. 복호화 실패라면 `ENCRYPTION_KEY`가 바뀌지 않았는지 확인 |
| `PERMISSION_DENIED` | mmcp 역할 부족, 읽기 전용 키, Mattermost 403, 봇 게시 권한 부족 | 역할 부여, 쓰기 가능한 자격증명 사용, 봇을 채널에 추가하거나 권한 부여 |
| `TOOL_DISABLED` | 도구 비활성 또는 정책 게이트(DM 검색·파일 다운로드 꺼짐 등), 별칭 꺼짐 | 도구 정책·데이터·호출 정책 확인 |
| `TOOL_NOT_FOUND` | 존재하지 않는 도구 이름 | 클라이언트의 도구 목록 갱신 |
| `POLICY_BLOCKED` | 채널 유형(공개/비공개/DM/그룹) 차단 | 데이터·호출 정책의 접근 범위 확인 |
| `CHANNEL_NOT_FOUND` / `AMBIGUOUS_CHANNEL` | 채널을 찾지 못함 / 여러 후보 | `team/channel` 형식이나 채널 ID 사용 |
| `TEAM_NOT_FOUND` | 참여 중인 팀에 없음 | 팀 이름 확인 |
| `RECIPIENT_NOT_FOUND` / `AMBIGUOUS_RECIPIENT` / `RECIPIENT_INACTIVE` | 수신자 없음 / 동명이인 후보 반환 / 비활성 계정 | username으로 다시 지정 (후보 중 사용자가 선택) |
| `APPROVAL_REQUIRED` | confirm 모드 첫 호출 (정상 흐름) | 미리보기 확인 후 `confirm_token`으로 재호출 |
| `APPROVAL_REJECTED` / `APPROVAL_EXPIRED` / `APPROVAL_MISMATCH` / `APPROVAL_USED` / `APPROVAL_INVALID` | [8.2](#82-2단계-승인-confirm-모드) 참고 | `confirm_token` 없이 다시 호출해 새 미리보기 받기 |
| `RATE_LIMITED` | mmcp 분당 한도 또는 Mattermost API 한도 초과 | 대기 후 재시도, 필요 시 한도 조정 |
| `BOT_NOT_CONFIGURED` | 봇 발신 설정인데 봇 토큰 없음 | Mattermost 연동에서 봇 토큰 입력 |
| `MATTERMOST_NOT_CONFIGURED` | Mattermost URL 미설정 | Mattermost 연동 설정 |
| `MATTERMOST_UNAVAILABLE` | Mattermost 연결 불가 | 네트워크·TLS·CA 확인, 연결 테스트 |
| `MATTERMOST_ERROR` | 기타 Mattermost API 오류 | 서버 로그에서 요청 ID로 상세 확인 |
| `NOT_FOUND` | 메시지·파일 등 대상 없음 | 대상 ID 확인 |
| `INVALID_ARGUMENT` | 인자 누락·형식 오류 | 도구 스키마 확인 |
| `INTERNAL` | 내부 오류 (응답에 요청 ID 포함) | 서버 로그에서 해당 요청 ID 검색 |

HTTP 수준 오류: `IP_DENIED`(IP 허용목록), `ORIGIN_DENIED`(CORS Origin), `SESSION_NOT_FOUND`(종료된 MCP 세션 — 클라이언트 재연결 필요).

### 17.4 Mattermost 연동

| 증상 | 원인 | 조치 |
|---|---|---|
| 연결 테스트에서 "system_admin 아님" | 서비스 토큰 계정 권한 부족 | 시스템 관리자 계정 PAT로 교체 |
| 자동 발급 실패 "개인 액세스 토큰이 비활성화…" | Mattermost에서 PAT 기능 꺼짐 | Mattermost 시스템 콘솔에서 개인 액세스 토큰 허용 |
| 봇 발신 시 `PERMISSION_DENIED` | 봇이 채널에 없거나 권한 부족 | 봇을 채널에 추가하거나 `post:channels`/`post:all` 부여 |
| `requester_pair` DM 실패 | 서비스 토큰 system_admin 아님 또는 봇 `post:all` 없음 | 권한 부여 또는 `bot_pair` 사용 |

---

## 18. 보안 체크리스트

**배포**

- [ ] `ENCRYPTION_KEY`를 `gen-key`로 생성하고 DB 백업과 분리 보관
- [ ] `BOOTSTRAP_ADMIN_PASSWORD`를 강력한 값으로 설정하고 첫 로그인 후 비밀번호 변경
- [ ] `compose.offline.yml`의 읽기 전용 루트 FS, `no-new-privileges`, `cap_drop: ALL` 유지
- [ ] TLS 종단 리버스 프록시 뒤에 배치하고 Public URL을 `https://`로 설정
- [ ] 프록시 사용 시에만 프록시 헤더 신뢰 켜기
- [ ] PostgreSQL 접속에 TLS(`sslmode`)와 최소 권한 계정 사용

**인증·접근**

- [ ] Keycloak 클라이언트: confidential, Standard flow + PKCE S256, Direct access grants OFF, 짧은 액세스 토큰 수명
- [ ] Keycloak SSO 안정화 후 로컬 계정 로그인 끄기 (부트스트랩 관리자만 비상용 유지)
- [ ] 관리 콘솔·메트릭 IP 허용목록 설정 (현재 IP 포함 확인)
- [ ] 필요 시 MCP IP 허용목록 설정, CORS Origin은 필요한 것만 등록 (`*` 지양)
- [ ] `admin` / `mcp-admin` 역할 보유자 최소화, 정기 점검
- [ ] 테스트 전용 TLS 검증 생략 옵션이 꺼져 있는지 확인

**데이터·도구**

- [ ] DM 검색, 첨부 파일 내용 조회는 필요할 때만 켜기
- [ ] 위험 도구(`mattermost_delete_post`, `mattermost_archive_channel`)는 비활성 유지
- [ ] 쓰기 도구의 confirm 모드 유지, 승인 유효시간 최소화
- [ ] 민감 환경에서는 **콘솔 승인 필수(엄격 모드)** 켜기 — 프롬프트 인젝션으로 AI가 스스로 승인하는 것을 차단
- [ ] 리버스 프록시 뒤라면 **프록시 헤더 신뢰**를 켜고, 꺼져 있을 때는 X-Forwarded-* 헤더가 무시됨을 확인 (X-Forwarded-For는 가장 오른쪽 값만 사용)
- [ ] **일반 설정 > 표시 시간대**를 조직 시간대(기본 Asia/Seoul)로 확인
- [ ] 조회 결과 신뢰불가 표시 유지
- [ ] 감사 로그 본문 저장은 규정상 필요할 때만 켜기
- [ ] 서비스 토큰은 디렉터리 조회·토큰 발급 전용 계정으로 분리

**운영**

- [ ] API 키 만료일 필수·최대 유효기간 설정 검토
- [ ] 회전 주기 초과 사용자 키·API 키 정기 회전, 자동 발급 토큰 정기 일괄 회전
- [ ] 마스터 키 정기 회전 ([11.3 런북](#113-런북-마스터-키-회전-무중단-절차))
- [ ] 감사 로그의 `IDENTITY_CONFLICT`, `TOKEN_REUSE`, 반복 `denied` 모니터링
- [ ] 미사용 OAuth 클라이언트 차단·삭제, 오래된 웹 세션 정리
- [ ] 설정 내보내기 + DB 덤프 정기 백업과 복원 리허설
