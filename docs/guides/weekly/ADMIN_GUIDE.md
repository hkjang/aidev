# Weekly 관리자 가이드

- **문서 버전**: v0.299.0
- **대상**: 시스템 관리자, Security/DevOps 엔지니어, 데이터 보안 담당자
- **문서 개요**: 구성 요소, 릴리즈 자산으로 설치, 환경 변수와 관리자 설정, 계정과 권한, 운영(백업·상태·업그레이드), 장애 대응, 보안. 화면을 쓰는 사람의 안내는 [사용자 가이드](USER_GUIDE.md)에 있습니다.

이 문서의 그림은 실제로 띄운 Weekly v0.295.0 을 headless Chrome 이 1440×900 으로 찍은 것입니다(`scripts/guide-captures.py`). 화면의 이름·조직은 시험용 씨앗 데이터이며 실제 사람이 아닙니다. 운영 세부(백업 순서, 보안 통제 목록, 업그레이드 절차)는 [운영 및 보안 가이드](OPERATIONS.md)가 정본이고 여기서는 가리킵니다.

---

## 1. 구성 요소

| 구성 요소 | 무엇 | 주고받는 것 |
|---|---|---|
| `weekly` 컨테이너 | Go 단일 프로세스. HTTP API·정적 프런트엔드·자동화 Worker(메일·Import·Confluence·자동 복제·알림)가 한 프로세스 안에 있습니다 | `:8080` HTTP. 반드시 **한 프로세스만** 같은 DB에 붙입니다(5.4 절) |
| PostgreSQL 15+ | 모든 데이터. `pg_trgm`이 있으면 유사 검색, `pgvector`가 있으면 의미 검색이 켜집니다(없으면 그 단계만 자동 비활성) | `WEEKLY_POSTGRES_DSN` |
| `/var/lib/weekly` 볼륨 | 상태 볼륨. 첨부 이미지, Import 원본 PPTX(`imports/`), 등록한 PPTX 템플릿, 환경 키가 없을 때의 `instance.key` | 컨테이너에 마운트 |
| Reverse Proxy / Ingress | TLS 종료. `X-Forwarded-For`·`X-Forwarded-Proto`를 Proxy 가 덮어써야 합니다 | → `:8080` |
| Keycloak (선택) | OIDC SSO. 관리자 화면에서 설정 | `GET /api/v1/auth/oidc/callback` 으로 돌아옵니다 |
| SMTP 릴레이 (선택) | 제출 메일·팀원 권고·마감 알림 발송 | 관리자 화면 `주간보고 메일 발송` |
| OpenAI 호환 AI Gateway (선택) | 자유 텍스트 구조화, PPTX 분석, Confluence 요약, 임베딩 | 관리자 화면 `AI Gateway · 과거 자료 Import` |
| Confluence Server 6.9.1 (선택) | 주간 작성·수정 활동에서 업무 후보 수집 | 관리자 화면 `Confluence 6.9.1 자동화` |
| ITSM (선택) | 상황판 일정의 SR 번호 → 제목 조회와 링크 | 관리자 화면 `ITSM 연동` |

자원: Compose 기본값은 `mem_limit: 1g`, `cpus: 2`, `/tmp` tmpfs 512MiB 입니다. PPTX 내보내기는 한 요청이 `보고서당 최대 캡처 수 × 캡처 파일당 최대 MB × 3.5` 만큼 메모리를 쓰므로 첨부를 많이 허용하면 한도를 그만큼 올립니다(관리자 화면 `화면 캡처 첨부` 카드가 같은 계산을 보여 줍니다).

---

## 2. 설치

GitHub Release 에서 `weekly-v0.299.0.tar.gz` 하나만 반입합니다. 자세한 검증 근거(파일 해시와 이미지 다이제스트가 왜 둘 다 필요한지)는 [README 오프라인 설치](../README.md#오프라인-설치)에 있고, 여기서는 순서대로 붙여 넣을 명령만 적습니다.

```bash
# 1. 파일이 온전히 왔는지 — 릴리즈에 적힌 SHA-256 과 비교
sha256sum weekly-v0.299.0.tar.gz

# 2. 이미지 적재. 같은 버전의 weekly:v0.299.0 이 생깁니다
gzip -dc weekly-v0.299.0.tar.gz | docker load

# 3. 환경 파일. deploy/.env.example 을 복사해 값을 채웁니다
cp deploy/.env.example deploy/.env
#    WEEKLY_POSTGRES_DSN            — PostgreSQL 연결 문자열
#    WEEKLY_BOOTSTRAP_ADMIN         — 최초 관리자 아이디 (첫 기동만)
#    WEEKLY_BOOTSTRAP_ADMIN_PASSWORD — 12자 이상 (첫 기동만)
#    WEEKLY_ENCRYPTION_KEY          — openssl rand -base64 32

# 4. 기동
docker compose --env-file deploy/.env -f deploy/compose.yaml up -d

# 5. 준비 확인 — PostgreSQL 연결까지 포함해 200 이면 준비된 것
curl -fsS http://127.0.0.1:8080/readyz
```

첫 기동은 마이그레이션을 돌리고 `WEEKLY_BOOTSTRAP_ADMIN`으로 최초 관리자를 만듭니다. 기동 로그에 `bootstrap administrator ensured` 가 찍힙니다. 브라우저로 열어 그 계정으로 로그인하면 왼쪽 메뉴 맨 아래 `관리자 설정`이 보입니다.

**첫 기동이 끝나면** `.env`·Compose·Kubernetes Secret·CI 변수에서 `WEEKLY_BOOTSTRAP_ADMIN`과 `WEEKLY_BOOTSTRAP_ADMIN_PASSWORD`를 지우십시오. 최초 관리자가 만들어진 뒤에는 다시 읽히지 않고, 지운 뒤에도 정상 기동합니다. 관리자가 전부 사라진 데이터베이스로 기동하면 그때는 두 값을 다시 요구하며 그 사실을 로그에 적고 멈춥니다.

| 포트·볼륨 | 값 |
|---|---|
| 포트 | `8080/tcp` (Compose 는 `8080:8080` 으로 공개. 운영에서는 Proxy 뒤에 두고 직접 노출하지 않습니다) |
| 볼륨 | `weekly-data` → `/var/lib/weekly` |
| 임시 공간 | `/tmp` tmpfs 512MiB (대용량 multipart 처리용) |
| 헬스체크 | `wget -q -O - http://127.0.0.1:8080/healthz`, 30초 간격, `start_period` 120초 |

Kubernetes 는 `deploy/kubernetes.yaml`을 씁니다. `strategy: Recreate`와 `startupProbe`(5초 × 60회)를 그대로 두십시오 — 이유는 5.4 절과 [OPERATIONS.md](OPERATIONS.md#상태-확인)에 있습니다.

---

## 3. 설정

### 3.1 환경 변수 (전수)

프로세스가 읽는 환경 변수는 아래 다섯 개가 전부입니다(`internal/app/config.go`). 나머지 설정은 모두 관리자 화면에 있고 PostgreSQL 에 저장됩니다.

| 이름 | 기본값 | 필수 | 설명 |
|---|---|---|---|
| `WEEKLY_POSTGRES_DSN` | 없음 | **필수** | PostgreSQL 연결 문자열. 예: `postgres://weekly:비밀번호@db.example.internal:5432/weekly?sslmode=require` |
| `WEEKLY_BOOTSTRAP_ADMIN` | 없음 | 첫 기동만 | 관리자가 하나도 없는 DB 에서만 읽습니다. 최초 관리자 아이디 |
| `WEEKLY_BOOTSTRAP_ADMIN_PASSWORD` | 없음 | 첫 기동만 | 최초 관리자 비밀번호. **12자 이상**이 아니면 기동을 거절합니다. 기동 뒤 화면에서 바꿀 수 있습니다 |
| `WEEKLY_ENCRYPTION_KEY` | 없음 (볼륨의 `instance.key` 로 대체) | 강력 권장 | 관리자 화면에서 입력한 OIDC Client Secret·AI API Key·Confluence 비밀번호·ITSM 토큰·SMTP 비밀번호를 보호하는 마스터 키. `openssl rand -base64 32` 로 한 번 만들고 업그레이드마다 같은 값을 유지합니다. 비우면 키가 상태 볼륨에만 저장되는 하위 호환 모드이며 기동 로그가 그 사실을 적습니다 |
| `WEEKLY_ALLOW_SECRET_RESET` | `false` | 선택 | 비밀 설정을 복호화할 수 없는 상태에서 기동을 강행하고 모두 다시 입력하기로 했을 때만 `true`. 기존 암호문은 지우지 않고 화면에 `다시 입력 필요`로 표시합니다 |

Compose 파일은 `WEEKLY_VERSION`(이미지 태그, 기본 `0.299.0`)도 읽습니다. 이것은 프로세스가 아니라 `deploy/compose.yaml`의 변수입니다.

### 3.2 관리자 화면의 서비스 설정

`관리자 설정 → 서비스 설정` 탭입니다. 카드마다 그 설정이 하는 일이 화면에 적혀 있고, `설정 저장`은 화면 전체를 한 번에 저장합니다. 비밀값(Client Secret, API Key, 비밀번호, 토큰)은 암호화되어 저장되고 브라우저로 다시 내려가지 않으며, 입력란을 비워 저장하면 기존 값이 유지됩니다. `지우기`로만 지웁니다.

![관리자 설정 — 서비스 설정 탭. 일반·워크플로 카드와 인증·Keycloak OIDC 카드](assets/guide/admin-settings.png)

| 카드 | 주요 설정 | 비고 |
|---|---|---|
| 일반 · 워크플로 | 서비스 이름·공지·시간대, `팀장 검토·승인 사용`, `주차 시작 요일`, 제출 마감(주차 시작 후 며칠째·몇 시까지) | **주차 시작 요일은 사실상 데이터 이관입니다**(3.3 절) |
| 인증 · Keycloak OIDC | 로컬 로그인 사용, 세션 유효시간, 계정당·IP당 로그인 실패 허용 횟수, 차단 시간, OIDC Issuer·Client·Scopes·Claim·관리자 그룹·자동 등록 | `OIDC 연결 시험` 버튼. IP당 제한은 기본 꺼짐 |
| AI Gateway · 과거 자료 Import | AI 사용, Chat Completions Endpoint, API Key, 모델, 제한시간, 최대 입력 글자수, Import 파일 수·크기·원본 보관일 | `AI Structured Output 연결 시험` |
| Confluence 6.9.1 자동화 | 사용 여부, Base URL, 인증 방식, Service Account, Space 포함·제외, 수집 주기, AI 분류, 점수 규칙 | `Confluence REST 연결 시험`. 상세는 [CONFLUENCE.md](CONFLUENCE.md) |
| 주간보고 메일 발송 | 사용, SMTP 호스트·포트·보안·계정·보내는 주소·이름·제한시간·재시도 | `메일 발송 시험`은 관리자 본인 주소로 갑니다. 최근 발송·대기·실패 건수가 카드 아래에 보입니다 |
| ITSM 연동 | 사용, 조회 주소, 이동 주소, 파라미터 이름, 제목 JSON 경로·정규식, SR 번호 형식, 인증 헤더·토큰, 화면 표기, 제한시간 | `SR 조회 시험` |
| 화면 캡처 첨부 | 보고서당 최대 캡처 수, 파일당 최대 MB | 메모리 계산이 카드에 있습니다 |
| 기간 업무보고 집계 | 병합 유사도, 정체·이슈 지속 판정 주차 수, 최대 주차 수 | |
| 보안 · 분석 | API 키 최대 유효일, 분석 데이터 보관일, 감사 로그 보관일(0=무기한) | |
| 검색 | 유사 검색 최소 점수, 의미 기반 검색 사용, Embeddings Endpoint·모델, 의미 검색 최소 점수 | `임베딩 다시 생성`. pgvector 가 없으면 의미 검색 비활성 |

### 3.3 주차 시작 요일

`주차 시작 요일`은 현재 주차 조회, 팀 분석, Confluence 후보, PPTX 날짜와 Import 의 날짜 보정에 공통 적용됩니다. 요일을 바꾸면 주 격자만 옮겨지고 이미 저장된 보고서의 `week_start`는 그대로 남습니다. 서비스는 기존 보고서가 있으면 영향 범위를 알리고 확인을 요구하며, 강행하면 경고 로그를 남깁니다. 운영 중 변경은 피하고 꼭 필요하면 분기 경계처럼 영향이 적은 시점에 하십시오.

전환되는 한 주 동안의 동작은 버전마다 고쳐 왔습니다. 작성 화면은 이번 7일을 덮는 기존 보고서를 열어 이어 쓰게 하고(v0.282.0), 분석 화면과 MCP 주간 요약(v0.287.0), 회의 모드와 주간 변화 요약(v0.289.0), 참여 분석의 제출률 추이와 미제출자 명단(v0.290.0), MCP 보고서 검색(v0.294.0)은 모두 날짜가 아니라 **기간이 겹치는** 보고서를 세며 한 사람은 한 주에 한 번만 세어집니다. 정시·지각은 옮긴 뒤에도 보고서 자신의 주차 마감으로 잽니다 — 그것이 작성자가 실제로 지켜야 했던 마감이기 때문입니다. 같은 기간의 두 번째 보고서는 만들 수 없습니다.

### 3.4 Keycloak OIDC

1. 관리자로 로그인 → `관리자 설정 → 서비스 설정 → 인증 · Keycloak OIDC`.
2. Keycloak 클라이언트의 Valid Redirect URI 에 `https://<weekly 주소>/api/v1/auth/oidc/callback` 을 등록합니다(라우트는 `GET`).
3. Issuer URL·Client ID·Client Secret 을 넣고, 그룹 Claim 과 `관리자 그룹`을 지정하면 그 그룹의 사용자가 자동으로 ADMIN 이 됩니다. `사용자 자동 등록`을 켜면 처음 로그인하는 사용자가 계정을 얻습니다.
4. `OIDC 연결 시험` → `설정 저장` → `Keycloak OIDC 사용`을 켭니다.

OIDC 가 켜진 배포에서 Weekly 세션이 없는 브라우저가 앱을 처음 열면 Authorization Code + PKCE 흐름을 `prompt=none`으로 한 번 시작합니다. Keycloak SSO 세션이 있으면 로그인 화면 없이 원래 화면으로 돌아오고, `login_required`·`interaction_required`·`consent_required`·`account_selection_required`이면 일반 로그인 화면으로 돌아옵니다. state·nonce·PKCE 와 10분 만료를 검증하며 ID·Access·Refresh Token 은 DB 에 보관하지 않습니다. 이 자동 확인은 앱 최초 세션 조회가 401 인 경우에만 하므로 PostgreSQL 장애를 로그아웃으로 오인해 Keycloak 으로 보내지 않고, 작성 중 세션 만료 시에는 탭을 이동하지 않습니다. Weekly 의 `로그아웃`은 Keycloak Realm 로그아웃을 호출하지 않고 현재 탭(`sessionStorage`)의 자동 확인만 억제합니다.

### 3.5 AI Gateway 와 Import

`AI Gateway · 과거 자료 Import` 카드에서 관리합니다. 배포 환경에는 연결값 대신 공통 암호화 키만 둡니다.

| 설정 | 의미 | 기본값 |
|---|---|---:|
| AI 작성·PPTX 분석 사용 | 사용자 AI 작성·PPTX 분석 허용 | 꺼짐 |
| Chat Completions Endpoint | OpenAI 호환 `/v1/chat/completions` 전체 URL | 없음 |
| AI 모델 | Gateway 가 제공하는 모델 식별자 | 없음 |
| AI API Key | 선택형 Bearer 토큰, 암호화 저장 | 없음 |
| AI 요청 제한시간(초) | 한 번의 AI 호출 제한 | 90 |
| AI 최대 입력 글자수 | 정규화 입력 상한 | 50,000 |
| 한 번에 업로드할 PPTX 수 | 다중 업로드 상한 | 20 |
| PPTX 파일당 최대 MB | PPTX 한 개 상한 | 25 |
| Import 원본 보관일 | 확정·건너뜀·실패 원본 보존기간 | 365 |

Endpoint 와 모델을 먼저 저장하고 `AI Structured Output 연결 시험`으로 JSON Schema 지원을 확인한 다음 AI 사용을 켭니다. 모델 계약은 실적·계획·이슈를 원자 문자열 배열로 반환하며, PPTX 분석은 각 항목에 실제 근거 슬라이드 번호와 업무 구분 신뢰도를 요구합니다. 존재하지 않는 슬라이드나 근거 없는 항목을 반환하는 모델은 분석 실패로 처리되므로 업그레이드 전에 운영 모델을 다시 시험하십시오. AI 에는 PPTX 파일 자체가 아니라 서버에서 추출·정규화한 텍스트만 갑니다. 외부 AI 호출이 허용되지 않는 환경에서는 사내망 OpenAI 호환 Gateway 를 쓰십시오.

원본 보존기간이 지나면 PPTX 바이너리만 상태 볼륨에서 삭제되고 해시·추출 텍스트·구조화 결과·연결 보고서·감사 기록은 PostgreSQL 에 남습니다.

### 3.6 Confluence 6.9.1 자동화

`Confluence 6.9.1 자동화` 카드에서 Base URL, `BASIC` 인증, 연동 전용 계정과 비밀번호를 저장한 뒤 `Confluence REST 연결 시험`을 합니다. 대상·제외 Space, 5분 이상의 수집 주기, Blog 포함 여부, AI/본문 분석, 후보 점수와 업무 키워드를 지정합니다.

`Confluence 자동화` 탭에서는 마지막 성공·시도 시각, 조회/변경 Page 수, 생성 후보, 실패 수와 최근 단계별 오류를 보고 강제 증분 Sync 를 요청합니다. 사용자 매핑은 관리자 명시값 → 이메일의 `@` 앞부분 → Weekly 로그인 아이디 순서로 자동 연결됩니다(예: `hong@example.com` → `hong`). 유일하게 판정할 수 없는 사용자만 표에서 직접 지정합니다.

![관리자 설정 — Confluence 자동화 탭. 수집 상태와 Weekly ↔ Confluence 사용자 매핑 표](assets/guide/admin-confluence.png)

Confluence 본문은 PostgreSQL 이나 로그에 저장되지 않습니다. 운영 규격과 장애 코드(`BODY_VERSION_CHANGED`, `AI_SUMMARY` 등)는 [CONFLUENCE.md](CONFLUENCE.md)에 있습니다.

### 3.7 메일 (SMTP)

`주간보고 메일 발송` 카드에서 `주간보고 메일 발송 사용`, 호스트·포트·보안·계정·보내는 주소를 저장하고 `메일 발송 시험`(`POST /api/v1/admin/settings/mail/test`, 관리자 본인 주소로 짧은 메일)을 통과시킨 뒤 사용자에게 안내합니다. 사내 릴레이가 포트 25 에 인증 없이 열려 있는 경우가 흔해 보안 `없음(평문)`이 기본이지만, **계정을 쓰려면 STARTTLS 나 TLS 가 필요합니다** — 평문 연결에는 비밀번호를 싣지 않습니다.

이 설정 하나가 세 가지 메일을 보냅니다. 발송은 모두 제출 트랜잭션 밖의 큐이므로 릴레이가 죽어 있어도 제출은 실패하지 않습니다.

| 메일 | 누가 켜나 | 언제 | 큐 테이블 |
|---|---|---|---|
| 제출 메일 (본문 + PPTX 첨부) | 사용자 개인 설정 `제출할 때 보내기` | 제출 직후 | `report_mail_deliveries` |
| 팀원 작성 권고 | 팀장 이상 개인 설정, 또는 `지금 권고 메일 보내기` | 선택 요일 09:00 이후 주 1회 / 즉시 | `team_reminder_deliveries` (`origin` AUTO/MANUAL) |
| 상황판 마감 임박 알림 | 사용자 개인 설정 `마감 임박 알림 받기` | 매일 09:00 이후, 보낼 것이 있을 때만 | `schedule_reminder_deliveries` (`UNIQUE(user_id, reminder_on)`) |

- 제출 메일의 첨부는 `GET /api/v1/reports/{id}/export.pptx`와 같은 렌더링 경로라 바이트 단위로 같은 파일입니다. 상한 10MiB 를 넘거나 렌더링에 실패하면 **메일은 그대로 보내고** 본문 첫 줄에 이유를 적으며 로그(`report mail deck`, `report mail deck too large`)에 남깁니다. 300명이 같은 월요일에 제출하면 사람당 수백 KB~수 MB 가 추가로 나가니 릴레이의 메시지 크기 제한과 큐 용량을 확인하십시오.
- 사용자 `시험 발송`(`POST /api/v1/me/mail/test`)은 언제나 요청한 본인에게 가장 최근 보고서를 제출 때와 같은 본문·첨부로 보냅니다. 한 사용자당 30초에 한 번(`429 TOO_MANY_ATTEMPTS`), SMTP 미설정이면 `400 MAIL_CONFIGURATION_INVALID`, 릴레이 거부면 `502 MAIL_SEND_FAILED`. 감사 이벤트 `mail.test`.
- 팀원 권고(`POST /api/v1/me/team-reminders`, `TEAM_LEADER`·`ORG_MANAGER`·`ADMIN`)는 서버가 발송 직전 역할·조직 범위·제출 여부를 다시 계산합니다. 팀장·조직장은 본인 조직과 하위 조직, 관리자는 전체 활성 사용자, 본인 제외. `recipient_user_id + week_start` 유일 제약으로 같은 주에 한 통만 만들고, SMTP 가 준비되지 않았으면 `409 MAIL_RELAY_NOT_READY`. 감사 이벤트 `mail.team_reminders_queued`·`mail.team_reminders_manual`.
- 마감 알림은 담당자 본인의 미완료 일정 중 종료일이 내일부터 5일 이내인 것(`scheduleReminderDays = 5`, 한 통 최대 100건)이며 발송 직전 수신 여부·주소·활성·남은 일감을 다시 확인해 보낼 것이 없으면 행을 지웁니다. 실패는 로그(`schedule reminder retry`, `schedule reminder gave up`).
- 서비스 시간대가 실제 조직 시간대와 다르면 오전 9시도 그만큼 어긋나므로 `서비스 시간대`를 함께 확인하십시오. 관리자 카드의 발송 현황은 제출 메일만 집계하므로 권고·알림 장애는 서버 로그와 큐 테이블에서 봅니다.

### 3.8 ITSM 연동

`ITSM 연동` 카드에서 `ITSM 연동 사용`을 켜고 주소를 채운 뒤 `설정 저장`을 누르면 상황판의 일정 추가 창에 SR 번호 칸이 나타납니다. **주소가 두 개**인 이유는 실제로 다른 시스템인 경우가 많기 때문입니다 — 하나로 가정하면 상황판의 모든 링크가 브라우저에서 JSON 을 여는 주소가 됩니다.

| 설정 | 무엇 |
|---|---|
| `조회 주소 (API)` | 서버가 제목을 물어보는 API 주소 |
| `이동 주소 (포털)` | 사용자가 클릭했을 때 열리는 포털 주소. 비우면 조회 주소를 씁니다 |
| `SR 번호 파라미터 이름` | `{id}` 자리표가 없을 때 붙일 파라미터 이름 (기본 `sr_id`) |
| `제목 JSON 경로` | `result.request.0.subject` 처럼 점 구분, 숫자는 배열 위치 |
| `제목 추출 정규식 (HTML용)` | API 가 없고 HTML 만 있는 시스템용. 괄호 하나가 제목 |
| `SR 번호 형식 정규식` | 기본 `^[A-Za-z0-9._-]{1,64}$` |
| `인증 헤더 이름` / `인증 토큰` | 토큰은 암호화 저장, 브라우저로 내려가지 않음 |
| `화면 표기 이름` | 기본 `SR` |
| `조회 제한시간(초)` | 기본 10 |

일정에는 SR 번호만 저장하고 링크는 읽을 때마다 현재 이동 주소로 만들므로 포털이 이사하면 설정 한 줄로 기존 링크가 모두 따라갑니다. SR 번호는 형식 검사 후 URL 인코딩하고, 주소는 http·https 만, 응답은 1MB 까지만 읽습니다. `SR 조회 시험`(`POST /api/v1/admin/settings/itsm/test`)은 실제 호출 주소·HTTP 상태·응답 앞부분을 돌려주니 JSON 경로를 맞출 때 이것부터 쓰십시오.

### 3.9 PPTX 템플릿

`PPTX 템플릿` 탭에서 사내 표준 PPTX 를 등록합니다. 등록 시 ZIP 구조와 필수 토큰 포함 여부를 검증합니다.

![관리자 설정 — PPTX 템플릿 탭. 적용 대상 조직, 전사 기본 템플릿과 토큰 안내](assets/guide/admin-pptx.png)

- `{{WEEK_SCHEDULE}}` 주간 일정 범위 · `{{THIS_WEEK}}` 업무별 이번 주 한 일(필수) · `{{NEXT_WEEK}}` 업무별 다음 주 할 일(필수) · `{{ISSUES}}` 이슈 및 지원 요청 · `{{AUTHOR}}` 작성자 · `{{TEAM}}` 조직명
- 조직별로 다른 템플릿을 지정할 수 있고, 없으면 전사 기본 템플릿을 씁니다. 관리자 등록 템플릿은 원본 슬라이드 수와 레이아웃을 바꾸지 않습니다.
- 기본 4장 참조 형식은 항목 순서를 유지하면서 실적·계획 중 더 긴 열을 비용으로 계산해 슬라이드 높이를 균형화하고, 같은 구분이 한 장을 넘으면 각 장에 구분 제목을 반복합니다.
- 작성자가 선택한 팀원 자료는 출력 직전에 `선택 팀원 · 이름 (아이디)` 표기가 붙은 임시 항목으로 본인 항목 뒤에 들어가며 DB 에 저장되거나 집계되지 않습니다.

---

## 4. 계정과 권한

### 4.1 역할

| 역할 | 개인 보고서 작성 | 팀 승인/반려 | 조직 전체 조회 | 팀원 자료 선택 | 상황판 쓰기 범위 | PPTX 템플릿 등록 | 시스템 설정 | 감사 로그 |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| **ADMIN** | ✅ | ✅ | ✅ | ✅ (전체 활성 사용자) | 전체 | ✅ | ✅ | ✅ |
| **ORG_MANAGER** (조직장) | ✅ | ✅ | ✅ (소속·하위 조직) | ✅ (소속·하위 조직) | 소속·하위 조직원의 일정 | ❌ | ❌ | ❌ |
| **TEAM_LEADER** (팀장) | ✅ | ✅ | ❌ (소속 팀만) | ✅ (소속·하위 조직) | 소속·하위 조직원의 일정 | ❌ | ❌ | ❌ |
| **USER** (사용자) | ✅ | ❌ | ❌ (본인 것만) | ❌ | 본인 일정 | ❌ | ❌ | ❌ |

`경영 요약`·`업무 인사이트`·`팀 주간보고`·`보고 분석`·`회의 모드`의 조직 단위 조회는 팀장 이상만 가능합니다. 상황판의 **읽기**는 예외적으로 사용자도 본인 조직과 하위 조직 전체를 봅니다 — 부서의 공유 계획이기 때문이며, 부서 일정 제목이 조직 밖으로 나가면 곤란한 배포라면 조직 트리를 그 경계에 맞춰 두어야 합니다.

### 4.2 사용자 관리

`관리자 설정 → 사용자` 탭입니다. 위 `사용자 등록` 카드에서 아이디·표시 이름·이메일·초기 비밀번호(12자 이상 또는 비움)·역할·조직을 넣고 `등록`합니다. 목록에서 `편집`을 누르면 표시 이름·이메일·역할·조직·검토 책임자·새 비밀번호·활성 여부를 바꿉니다. 비활성 계정은 로그인할 수 없고 권고 메일과 팀원 자료 선택에서도 빠집니다.

![관리자 설정 — 사용자 탭. 등록 카드와 305명의 목록](assets/guide/admin-users.png)

### 4.3 조직

`조직` 탭에서 조직을 트리로 등록합니다(조직명·코드·상위 조직). 팀장·조직장의 범위, 상황판의 `부서 전체`, 팀원 권고의 대상이 모두 이 트리를 따릅니다. 그림의 씨앗 데이터는 회사 → 본부 → 실 → 팀 네 단계입니다.

![관리자 설정 — 조직 탭. 조직 등록과 조직 구조](assets/guide/admin-organizations.png)

### 4.4 API / MCP 키

사용자는 개인 설정에서 `wky_…` 키를 발급받고 원문은 발급 응답에서 한 번만 봅니다. 최대 유효일은 `보안 · 분석` 카드의 `API 키 최대 유효일`입니다. 키의 폐기는 사용자가 개인 설정에서 직접 합니다(개별 `폐기`, 또는 `모든 키 회전`으로 전부 한 번에). 관리자 화면에는 전체 키를 한 번에 폐기하는 자리가 없으므로, 사고 때는 해당 계정을 비활성으로 돌리십시오 — 비활성 계정의 키는 인증되지 않습니다. MCP 엔드포인트는 `GET/POST /mcp`(Streamable HTTP)이며 도구 목록은 [MCP.md](MCP.md)에 있습니다. 도구는 호출자 권한 안에서만 답합니다 — 미제출자 **명단**은 팀장 이상, API 운영 분석은 관리자만 부를 수 있고, 그 밖의 계정에게는 목록에 보이지도 않습니다.

---

## 5. 운영

### 5.1 상태 점검

| 무엇 | 어디 |
|---|---|
| 프로세스 생존 | `GET /healthz` |
| PostgreSQL 연결 포함 준비 상태 | `GET /readyz` |
| 최근 24시간 경로별 호출·지연·4xx/5xx | `관리자 설정 → 분석` 탭의 서비스 분석 |
| 메일 발송 현황 | `서비스 설정 → 주간보고 메일 발송` 카드 아래 (최근 n일 발송·대기·실패) |
| Confluence 수집 상태 | `Confluence 자동화` 탭 |
| 감사 이력 | `감사 로그` 탭 — 작업·행위자·기간으로 검색 |

![관리자 설정 — 분석 탭. 업무 키워드 워드클라우드와 조직별 보고 현황](assets/guide/admin-analytics.png)

![관리자 설정 — 감사 로그 탭. 로그인·상황판 생성·삭제 같은 이벤트가 행위자·IP 와 함께 남습니다](assets/guide/admin-audit.png)

기동 시 한 번 도는 정리 작업(업무 식별자 백필, 첨부 파일 무결성 확인)은 서버가 응답을 시작한 뒤 배경에서 수행하며 진행은 `work item backfill progress` 로그로 봅니다.

### 5.2 로그

로그는 표준 출력에 JSON 한 줄씩(`time`·`level`·`msg`·필드)입니다. `docker logs weekly` 또는 `kubectl logs`로 봅니다. 정상 기동은 이 순서로 찍힙니다.

```
{"level":"INFO","msg":"secret encryption initialized","key_source":"environment","stored_secrets":0}
{"level":"INFO","msg":"bootstrap administrator ensured","username":"admin"}
{"level":"INFO","msg":"database capabilities detected","pg_trgm":true,"pgvector":false}
{"level":"INFO","msg":"password hashing pool sized","workers":8,"reserved_mib":512,"container_limit_mib":0}
{"level":"INFO","msg":"Weekly started","address":":8080","version":"0.299.0"}
```

`key_source`가 `environment`가 아니라 볼륨이면 `WEEKLY_ENCRYPTION_KEY`가 없는 하위 호환 모드입니다. `container_limit_mib`가 0 이면 메모리 한도 없이 호스트 메모리를 상속한 것이니 Compose 의 `mem_limit`를 확인하십시오.

### 5.3 백업과 복구

같은 복구 시점의 **PostgreSQL 데이터베이스**와 **`/var/lib/weekly` 볼륨**을 함께 백업하고, `WEEKLY_ENCRYPTION_KEY`는 Secret Manager 에 따로 보관합니다. `scripts/weekly-backup.sh`가 두 대상을 한 번에 처리합니다.

```
weekly-backup.sh backup  -o OUT_DIR  [-d DSN] [-s STATE_DIR]
weekly-backup.sh verify  -i ARCHIVE_DIR
weekly-backup.sh restore -i ARCHIVE_DIR [-d DSN] [-s STATE_DIR] [--force]
```

`backup`은 DB 를 먼저 덤프하고 파일을 나중에 복사합니다(순서에 의미가 있습니다). `verify`는 참조된 첨부 파일이 실제로 보관됐는지 비교해 부족하면 0 이 아닌 종료 코드를 내므로 스케줄 백업이 깨진 복구 지점을 성공으로 기록하지 않습니다. `restore`는 대상 DB 의 모든 테이블을 지우고 볼륨을 비우고 덮어쓰며 `--force` 없이는 DB 이름을 직접 입력해야 진행합니다. Compose 환경에서 임시 컨테이너로 돌리는 명령과 복구 뒤 점검 목록은 [OPERATIONS.md 백업](OPERATIONS.md#백업)에 있습니다.

### 5.4 업그레이드와 되돌리기

1. `weekly-backup.sh backup` 후 `verify`가 통과하는지 확인합니다. 여기서 첨부 파일 경고가 나오면 업그레이드 전에 이미 볼륨이 어긋나 있는 것입니다.
2. 아직 `WEEKLY_ENCRYPTION_KEY`가 없다면 기존 볼륨을 유지한 채 설정합니다. 첫 기동이 볼륨 키의 비밀값을 새 키로 자동 재암호화합니다.
3. 새 릴리즈의 SHA-256 을 확인하고 `docker load` 한 뒤 `deploy/.env`의 `WEEKLY_VERSION`(또는 Kubernetes 이미지 태그)을 올립니다.
4. `docker compose --env-file deploy/.env -f deploy/compose.yaml up -d` — 컨테이너를 그 자리에서 교체합니다.
5. 로그에서 `secret encryption initialized`와 마이그레이션 완료, `Weekly started`의 `version`을 확인하고 `/readyz`, 로그인, 관리자 비밀값의 `현재 비밀값이 안전하게 설정됨`을 봅니다.
6. AI·Confluence·메일·ITSM 을 쓰는 환경은 각 카드의 연결 시험을 다시 돌립니다.

**두 벌을 같은 데이터베이스에 붙이지 마십시오.** 기동할 때 하는 복구가 `PROCESSING` 인 Import 를 `PENDING` 으로 되돌리는데, 겹쳐 있는 동안 그 작업은 중단된 것이 아니라 옛 프로세스가 붙들고 있는 것입니다. Kubernetes 는 `strategy: Recreate`, Compose 는 `up -d`가 그 자리에서 교체하므로 별도 설정이 없습니다.

**되돌리기**: 마이그레이션은 자동 down 을 수행하지 않습니다. 이전 이미지 태그로 돌아가는 것은 스키마 하위 호환이 보장된 버전 사이에서만 하고, 그렇지 않으면 1단계 백업으로 `restore` 합니다.

### 5.5 자동화 Worker 와 데이터 보관

- 주간 자동화(자동 복제·권고·마감 알림)는 기동 시 한 번 실행한 뒤 매분 현재 주차를 확인합니다. 자동 복제는 지난주 7일과 날짜가 겹치는 보고서에 업무 항목이 있을 때 새 `DRAFT`로 복제하고 `user_weekly_preferences.auto_clone_processed_week`을 먼저 잠그므로 여러 번 확인해도 초안을 둘 만들지 않습니다. 감사 이벤트 `weekly.preference`, `report.auto_clone`.
- Import Worker 는 `FOR UPDATE SKIP LOCKED`로 작업을 하나씩 가져가고 재시작 시 `PROCESSING`을 `PENDING/QUEUED`로 복구합니다. 멈춘 것처럼 보이면 AI Gateway 연결, 모델의 JSON Schema 지원, `/var/lib/weekly/imports` 쓰기 권한, Import 상세 오류 순서로 봅니다.
- `Import 원본 보관일`이 지난 `CONFIRMED`·`SKIPPED`·`FAILED` 원본은 30분 유지보수 주기에 최대 500개씩 제거됩니다. `감사 로그 보관일`(기본 365, 0=무기한)과 `분석 데이터 보관일`도 같은 주기입니다.
- 상황판은 `schedule_tasks` 한 표(마이그레이션 `029_schedule_tasks.sql`)에 담당자와 등록자를 따로 두고 완료를 `done_at`·`done_by`로 저장합니다. 조회는 기간이 겹치는 행(`start_date <= to AND end_date >= from`)이며 감사 이벤트는 `schedule.create`·`schedule.update`·`schedule.done`·`schedule.delete`입니다. 편집은 전체 치환(`PUT /api/v1/schedule/{id}`)입니다.
- 선택 팀원 자료는 `user_report_inclusions`(마이그레이션 `027_report_inclusions.sql`)에 관계만 저장하고 보고서를 복제하지 않습니다. `GET/PUT /api/v1/me/report-inclusions`, 상한 500명, 범위 밖 대상은 `403 REPORT_INCLUSION_MEMBER_FORBIDDEN`, 일반 사용자의 비어 있지 않은 저장은 `403 REPORT_INCLUSION_ROLE_REQUIRED`. 강등·조직 이동·비활성화 때 행을 지우지 않고 조회에서 가리며 다시 범위에 들어오면 되살아납니다. 감사 이벤트 `weekly.report_inclusions`.

---

## 6. 장애 대응

| 증상 | 확인할 곳 | 조치 |
|---|---|---|
| 기동하지 않고 `WEEKLY_POSTGRES_DSN 환경변수가 없습니다. deploy/.env.example 을 복사해 채운 뒤 --env-file 로 넘기십시오` | 컨테이너 로그 | `.env`의 DSN 을 채우고 `--env-file`로 넘겼는지 확인 |
| 기동하지 않고 `WEEKLY_BOOTSTRAP_ADMIN_PASSWORD 는 12자 이상이어야 합니다` | 컨테이너 로그 | 첫 기동용 비밀번호를 12자 이상으로 |
| 기동하지 않고 두 부트스트랩 변수를 요구 | 컨테이너 로그 | DB 에 관리자가 하나도 없습니다(새 DB 이거나 전부 비활성·삭제). 두 값을 넣어 한 번 기동한 뒤 지웁니다 |
| 기동하지 않고 `n개의 비밀 설정을 현재 암호화 키로 복호화할 수 없습니다: ai.api_key, …` | 컨테이너 로그 | 그 비밀값을 암호화했던 `WEEKLY_ENCRYPTION_KEY`를 되돌리거나 `instance.key`가 있던 볼륨을 복구합니다. 둘 다 없으면 `WEEKLY_ALLOW_SECRET_RESET=true`로 기동하고 화면에서 `다시 입력 필요`인 값을 모두 다시 넣습니다 |
| `/healthz`는 200 인데 `/readyz`가 실패 | `/readyz` 응답, PostgreSQL | DB 연결·자격·`sslmode` 확인. Proxy 헬스체크는 `/readyz`를 보게 합니다 |
| 첫 부팅이 Kubernetes 에서 계속 재시작 | 파드 이벤트 | `startupProbe`(5초 × 60회)가 빠졌는지 확인. 기존 데이터 위의 마이그레이션·백필은 livenessProbe 기본값보다 오래 걸립니다 |
| 사용자에게 캡처 패널이 `파일 없음`, 로그에 `attachment files are missing` | 기동 로그의 `hint` (`확인: /var/lib/weekly 를 영속 볼륨으로 마운트했는지`) | 볼륨이 빠졌거나 바뀌었습니다. 볼륨을 복구하거나(`weekly-backup.sh restore`) 사용자에게 재업로드 안내 |
| SSO 로그인이 전부 실패 | `서비스 설정 → 인증` 카드의 Client Secret 상태 | `복호화할 수 없음 · 다시 입력 필요`면 키가 바뀐 것입니다(위 복호화 항목). `OIDC 연결 시험`으로 Issuer·Redirect 확인 |
| 사용자가 `Keycloak 자동 로그인을 확인하지 못했습니다`를 봄 | Keycloak 세션·클라이언트 설정 | 정상 경로일 수 있습니다(세션 없음). 반복되면 Valid Redirect URI 와 `prompt=none` 허용 여부 |
| 사용자가 `로그인 시도가 너무 많습니다. n분 후에 다시 시도하세요.` | `감사 로그`의 로그인 차단 이벤트 | 본인 오타면 `로그인 차단 시간(분)`만큼 기다립니다. 한 층이 통째로 잠기면 `IP당 로그인 실패 허용 횟수`가 켜져 있는지 — NAT 뒤에서는 끕니다 |
| 제출 메일이 안 감, 사용자가 `메일 릴레이가 준비되지 않았습니다` | `주간보고 메일 발송` 카드의 현황과 `마지막 실패 사유`, `report_mail_deliveries` | `메일 발송 시험`으로 릴레이 확인. 큐에 남은 것은 릴레이가 살아나면 재시도됩니다 |
| 첨부 없이 온 제출 메일, 본문 첫 줄에 이유 | 로그 `report mail deck too large` / `report mail deck` | 10MiB 상한을 넘었거나 렌더링 실패. 첨부 상한(`화면 캡처 첨부`)을 낮추거나 템플릿을 점검 |
| 팀원 권고·마감 알림이 안 옴 | `team_reminder_deliveries`·`schedule_reminder_deliveries`, 로그 `schedule reminder retry`·`schedule reminder gave up` | SMTP 준비 여부, `서비스 시간대`(09:00 기준), 수신자 주소 유무. 관리자 카드의 현황에는 잡히지 않습니다 |
| Import 가 `PROCESSING`에서 멈춤 | Import 상세 오류, AI 카드 | AI Gateway 연결 시험, 모델의 Structured Output 지원, `/var/lib/weekly/imports` 쓰기 권한. 프로세스가 둘 붙어 있지 않은지 |
| 사용자가 `AI Gateway가 요청 한도를 초과했다고 답했습니다(HTTP 429)` / `AI 분석 시간이 초과되었습니다` | AI Gateway 측 로그 | Gateway 의 rate limit·응답 시간. `AI 요청 제한시간(초)` 조정 |
| Confluence 후보가 안 생김 | `Confluence 자동화` 탭의 단계별 오류 | `BODY_VERSION_CHANGED`는 재시도 가능한 정상 상태. 매핑 표의 미매핑 사용자, Space 포함·제외, 연결 시험 |
| 상황판 SR 조회가 `ITSM_LOOKUP_FAILED` | `SR 조회 시험`의 실제 호출 주소·상태·응답 | `제목 JSON 경로`·정규식·인증 헤더. 서버에서 ITSM 에 닿는지(브라우저가 아니라 서버가 호출) |
| 주차 시작 요일을 바꾼 뒤 사용자가 `… 주차 보고서가 같은 기간을 이미 담고 있습니다. 그 보고서를 여십시오.` | — | 설계된 동작입니다. 그 보고서를 열어 이어 쓰게 안내(3.3 절) |
| 기동 로그 `work item backfill progress`가 오래 이어짐 | 로그 | 정상. 500건씩 커밋하며 재시작해도 이어서 진행합니다 |

---

## 7. 보안

**기본값 중 바꿔야 하는 것**

- `WEEKLY_ENCRYPTION_KEY`를 반드시 설정하고 Secret Manager 에 보관합니다. 비우면 볼륨의 `instance.key`가 유일한 키입니다.
- DSN 의 `sslmode=disable`은 예시용입니다. 운영에서는 `sslmode=require` 또는 사내 CA 검증이 가능한 `verify-full`.
- 첫 기동 뒤 `WEEKLY_BOOTSTRAP_ADMIN`·`WEEKLY_BOOTSTRAP_ADMIN_PASSWORD`를 지우고, 최초 관리자 비밀번호를 화면에서 바꿉니다.
- SMTP 에 계정을 쓰면 STARTTLS/TLS 로. 평문 연결에는 비밀번호를 싣지 않습니다.

**외부에 열면 안 되는 것**

- `:8080`을 인터넷에 직접 노출하지 않습니다. TLS 를 종료하는 Reverse Proxy 뒤에 두고 Proxy 가 `X-Forwarded-For`·`X-Forwarded-Proto`를 덮어쓰게 합니다.
- PostgreSQL 은 Weekly 와 백업 작업만 닿는 망에 둡니다.
- 씨앗·검사 스크립트(`scripts/seed-scale.sql`, `load-check.py`, `guide-captures.py`)는 버려도 되는 배포에만 씁니다. 씨앗은 보고서가 있는 DB 에는 스스로 실행을 거부합니다.

**컨테이너**: 비루트 사용자, 모든 capability 제거, `read_only` 루트, `no-new-privileges`. `deploy/compose.yaml`과 `deploy/kubernetes.yaml`이 이미 그렇게 되어 있으니 완화하지 마십시오.

**인증 연동**: OIDC 는 state·nonce·PKCE 와 10분 만료를 검증하고 토큰을 저장하지 않습니다. 로컬 로그인은 계정당 실패 횟수 제한(기본 15분에 10회, 실패마다 0.25~2초 지연)과 계정 유무를 구분하지 않는 응답을 갖습니다. 세션 쿠키는 HttpOnly/SameSite 이고 변경 요청은 동일 출처(`Origin`) 검사를 받습니다. 개인 API/MCP 키 원문은 발급 응답에서 한 번만 노출됩니다.

**감사**: 로그인·차단, 보고서 상태·내용, 사용자·조직·설정, 키, 템플릿, PPTX 다운로드, AI 분석, Import, Confluence Sync, 상황판, 메일이 `감사 로그` 탭에 남습니다. 보관 기간은 `감사 로그 보관일`. 전체 통제 목록은 [OPERATIONS.md 보안 통제](OPERATIONS.md#보안-통제)입니다.
