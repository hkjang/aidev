# Momento 엔터프라이즈 관리자 가이드 (Admin & Security Guide)

- **문서 버전**: v0.34.35
- **대상**: 시스템 관리자, Security/DevOps 엔지니어, 데이터 보안 담당자, CISO  
- **문서 개요**: Momento 온프레미스 시스템 배포, Keycloak OIDC SSO 연동, RBAC 권한 관리, 개인정보 필터, CIDR 서브넷 매핑 및 Audit Trail 감사 운영

---

## 1. 시스템 아키텍처 및 부트스트랩 (Bootstrap)

Momento 컨테이너 프로세스는 **3개의 필수 환경변수**와 **1개의 권장 환경변수**만을 통해 최소 인프라로 구동됩니다.

```bash
# .env 환경 설정
MOMENTO_POSTGRES_DSN=postgres://momento:Secr3tPass@10.10.20.5:5432/momento?sslmode=disable
MOMENTO_BOOTSTRAP_ADMIN=admin@corporate.internal
MOMENTO_BOOTSTRAP_ADMIN_PASSWORD=SuperSecretAdminPassword123!
# 권장: 발급한 키를 암호화 저장해 재기동 후에도 다시 조회할 수 있게 합니다.
MOMENTO_ENCRYPTION_KEY=$(openssl rand -base64 32)
```

> **설정 원칙 (Design Principles)**:  
> 그 밖의 모든 공개 URL, Keycloak OIDC Client 정보, Claim Mapping, PII 차단 필터, CIDR 망 대역은 DB에 저장되는 동적 관리자 설정입니다. 부트스트랩 비밀번호는 최초 관리자 계정 생성 시에만 사용되며 기존 계정을 덮어쓰지 않습니다.

### 1.1 비밀값 암호화 (MOMENTO_ENCRYPTION_KEY)

`MOMENTO_ENCRYPTION_KEY`를 설정하면 개인 API key, Site Tracking Key, Server API Key, OIDC Client Secret, Delivery Channel Header를 AES-256-GCM으로 암호화해 저장합니다. 값은 32 byte base64/hex 또는 16자 이상 passphrase를 허용하며, 플랫폼이 공용으로 주입하는 `ENCRYPTION_KEY`도 alias로 인식합니다.

- 같은 키를 유지하면 **서비스를 재기동해도 키가 사라지지 않고 다시 입력할 필요가 없습니다.** 관리 → 사이트의 `키 보기`와 프로필 → API 키의 `저장된 키 보기`로 재조회하며, 조회 사실은 Audit Log에 남습니다.
- 변수가 없으면 기존 동작대로 해시만 저장하므로 키는 발급 시 1회만 표시됩니다.
- 키 교체는 `MOMENTO_ENCRYPTION_KEY`에 새 키, `MOMENTO_ENCRYPTION_KEY_PREVIOUS`에 이전 키를 두고 기동한 뒤 관리 → 설정 → 비밀값 암호화에서 재암호화를 실행하고 이전 키 변수를 제거합니다.
- 암호화 이전에 발급된 키는 한 번 회전해야 재조회 대상이 됩니다.

---

## 2. Keycloak OIDC SSO 연동 및 RBAC 매핑

Momento는 PKCE(S256)가 적용된 표준 OIDC(OpenID Connect) SSO 통합을 지원합니다.

### 2.1 Keycloak Client 구성
1. Keycloak Admin Console에서 `momento-web` Client ID 생성.
2. Valid Redirect URIs 설정: `https://momento.internal/api/v1/auth/oidc/callback`
3. Access Token Claim Mappers에 `groups` 및 `roles` 파싱 규칙 추가.

### 2.2 RBAC (Role-Based Access Control) 권한 매트릭스

| 역할 (Role) | 개요 대시보드 | 쿼리 빌더 | 퍼널/경로 분석 | 사이트·보존 정책 | PII 룰 변경 | 사용자·네트워크 | 감사 로그 |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| **Super Admin / Organization Admin** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Workspace Admin** | ✅ | ✅ | ✅ | 소속 Workspace만 | ❌ | ❌ | ✅ |
| **Analyst** | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| **Viewer** | ✅ | 조회 | 조회 | ❌ | ❌ | ❌ | ❌ |

PII 룰, 네트워크 망, 사용자 계정은 **배포 전체에 적용되므로 조직 관리자 이상**만 변경합니다(v0.33.1). 그 이전에는 Workspace 관리자도 변경할 수 있었고, 자기 역할을 최고 관리자로 올릴 수도 있었습니다.

Workspace 관리자의 사이트 관리(설정 변경, 키 회전, 삭제)는 **자신이 속한 Workspace의 사이트로 한정**됩니다(v0.33.0). 그 이전에는 다른 Workspace의 사이트도 대상이 됐습니다.

역할과 무관하게 **자기 역할은 바꿀 수 없고, 자기보다 높은 역할은 부여할 수 없습니다.** 최고 관리자도 스스로를 강등할 수 없습니다 — 배포에 관리자가 하나도 남지 않는 상태를 막기 위해서입니다.

---

## 2.1.9 릴리즈 자산이 누락되지 않는지

오프라인 설치는 릴리즈 자산(`momento-vX.Y.Z.tar.gz`와 체크섬)에 의존합니다. 태그를 밀면 릴리즈 워크플로가 실행되지만, 그 트리거는 한 번뿐이라 플랫폼 장애 시 유실될 수 있습니다.

`Release reconciliation` 워크플로가 매시간 최근 7일 태그와 릴리즈를 대조해, 릴리즈가 없거나 자산이 두 개가 아니면 릴리즈 워크플로를 다시 실행합니다. 수동으로 실행할 수도 있습니다.

## 2.2.0 보존 정책이 실제로 적용되는 범위

| 항목 | 적용 |
| :--- | :--- |
| Raw Event (월) | 적용 — 기간이 지난 Raw Event 삭제 |
| Session (월) | 적용 — Session 요약과 Visitor↔Session 색인(`visitor_sessions`) 둘 다 |
| 신원 (Visitor ID ↔ User ID) | 적용 — v0.32.4부터 남은 이벤트·세션이 없는 방문자의 매핑과 per-visitor 집계 삭제 |
| 집계 (월) | **적용** — 일별 집계 3종에서 기간이 지난 날짜 삭제. 비우면 무기한 보관 |
| Debugger / Dead Letter (일) | 적용 |
| Realtime (시간) | **적용되지 않음** — 별도의 Realtime 저장소가 없어 삭제 대상이 없습니다. API 호환을 위해 값은 계속 저장됩니다 |

집계 보존은 v0.29.0부터 적용됩니다. 이전에는 값을 저장만 하고 읽지 않아, 기간을 설정해도 일별 집계가 무기한 보관됐습니다.

일별 집계 중 방문자·세션 테이블은 하루에 방문자 한 명당 한 행이고 행마다 Visitor ID와 User ID가 있습니다. 집계 기간을 비워 두면 Raw Event가 삭제된 뒤에도 사람 단위 기록이 남습니다.

Visitor↔Session 색인은 v0.34.35부터 Session 보존기간을 따릅니다. 그 이전에는 **어떤 보존 작업도 이 테이블에서 행을 지운 적이 없어**, Raw Event와 Session이 사라진 뒤에도 Visitor ID·Session ID·User ID와 최초/최근 시각이 무기한 남았습니다. 개인 단위 삭제(개인정보 요청)는 파생 테이블을 재구축하므로 영향이 없었고, 기간 기반 보존만 이 테이블을 지나쳤습니다.

`관리 ➔ 보존 정책` 화면 상단이 **직전 보존 작업**을 보고합니다 — 실행 시각, 소요 시간, 테이블별 삭제 행 수, 그리고 실패했다면 그 원인. 이 기록은 v0.32.6부터 남습니다. 이전에는 정책과 수정 시각만 보였고 실패는 stderr 로그 한 줄로 끝났으므로, 로그 수집이 없는 폐쇄망에서는 **한 달째 실패하는 작업과 삭제 대상이 없는 작업을 구분할 방법이 없었습니다.** 삭제 대상이 없었던 회차도 "완료 · 0행"으로 보고되므로, 화면이 조용한 것과 작업이 멈춘 것이 구분됩니다. 기록은 최근 200회만 유지됩니다.

보존 작업은 매시간 무인으로 돌며, 테이블마다 2만 행씩 나눠 삭제하고 각 배치를 즉시 커밋합니다. 정책을 크게 줄인 직후처럼 삭제 대상이 많을 때도 진행한 만큼은 남으므로, 재기동이나 statement timeout으로 중단되어도 다음 시간에 이어서 수렴합니다. 한 번에 삭제하던 v0.32.4까지는 완료 전에 중단되면 아무 진행도 남지 않았습니다.

## 2.2.1 삭제와 보존의 검증 범위

개인정보 삭제는 규정 준수 약속이므로 통합 테스트로 확인합니다. `user_id` 모드 삭제 후 `raw_events`, `sessions`, `visitors`, `visitor_sessions`, `visitor_identities`, `identified_users`, `daily_site_visitors`, `daily_site_sessions` 여덟 개 테이블에 잔존 행이 없고, 같은 사이트의 다른 사람 데이터는 남아 있는지 확인합니다. `visitor`, `period`, `property` 모드도 각각 경계 밖 데이터 보존과 property만 제거되는지 확인합니다.

Retention은 사이트별 정책을 적용해 정책 밖 데이터가 삭제되고 정책 안 데이터가 유지되는지, Aggregate 재집계는 큐가 비고 실패 작업이 없는지 확인합니다.

## 2.3 분석 쿼리 보호

`최대 정확 조회 기간`은 쿼리 빌더뿐 아니라 **모든 분석 리포트 화면에 적용됩니다.** v0.28.0 이전에는 쿼리 빌더 한 곳에서만 확인해, 한도를 낮춰도 무거운 리포트 화면은 제한 없이 조회되고 있었습니다. 콘솔의 기간 선택지도 이 한도를 반영하므로 거절될 기간은 애초에 제시되지 않습니다.

대화형 분석 조회(방문자 인사이트, 이상 감지, 기여도, 방문자 검색·추적, Funnel)는 **25초 제한** 아래에서 실행됩니다. 초과하면 연결을 붙잡아 두지 않고 `504 QUERY_TIMEOUT`으로 즉시 끝나며, 기간 축소·Segment 적용·Scheduled Report 사용을 안내합니다. 요청이 취소되면 데이터베이스 쿼리도 함께 취소됩니다.

방문자 인사이트 보고서는 서로 독립적인 8개 조회를 **동시 실행 4개 상한**으로 병렬 수행합니다. 연결 풀(20)을 한 요청이 소진하지 않도록 상한을 두었고, 하나가 실패하면 나머지를 취소해 부분 결과를 완성된 보고서로 표시하지 않습니다.

이상 감지 기준선은 일별 Rollup(`daily_site_metrics`, `daily_site_visitors`, `daily_site_sessions`)에서 계산합니다. 평가 대상 날짜의 Rollup이 아직 없으면 그때만 Raw Event를 읽습니다. 따라서 Aggregate가 밀려 있으면 이상 감지가 느려질 수 있으며, 관리 → Aggregate Manager에서 재집계 상태를 확인하십시오.

`013_analytical_indexes.sql`은 `sessions` 인덱스와 방문자 검색용 `pg_trgm` 인덱스를 만듭니다. 세션 수가 많은 기존 설치에서는 이 마이그레이션이 최초 기동 시 수 초에서 수 분 걸릴 수 있습니다. `pg_trgm` 확장을 만들 권한이 없으면 인덱스 생성을 건너뛰고 순차 검색으로 동작합니다.

---

## 3. 개인정보 (PII) 필터 & URL 마스킹

수집기(Durable Collector)는 Inbox에 저장하기 전 개인정보 정책을 적용합니다. 기본 정책은 개인정보로 지정된 Property key를 중첩 객체와 Item 배열까지 제거하며, URL Query String과 Fragment를 제거합니다. Query String 수집을 명시적으로 활성화한 경우에만 관리자 목록의 Parameter를 마스킹합니다.

- **기본 차단 Property**: `email`, `phone`, `resident_number`
- **기본 URL 정책**: Query String 및 Fragment 제거
- **Query 수집 활성화 시 기본 마스킹 Parameter**: `token`, `password`, `email`
- **IP 익명화**: IPv4 `/24`, IPv6 `/64`
- **선택 정책**: User ID·User Agent 수집, Query String 제거, DNT, Visitor Profile

> **관리자 제어**: 관리자 콘솔 `관리 ➔ 개인정보` 메뉴에서 차단 key와 URL Parameter를 변경할 수 있습니다. SDK는 자동 DOM text 수집을 기본 비활성화하고 흔한 이메일·전화번호·주민번호 형태의 Error Message를 치환하지만, Custom Property 값까지 판별하지는 않으므로 연동 단계에서도 PII를 보내지 않아야 합니다.

---

## 4. C클래스 / CIDR 서브넷 망대역 맵핑

사내 C-Class 및 CIDR IP 서브넷 대역을 특정 물리적 오피스 또는 사업장 이름으로 매핑합니다.

```json
[
  { "cidr": "10.10.0.0/16", "name": "본사 판교 R&D 센터" },
  { "cidr": "10.20.0.0/16", "name": "서초 디지털 오피스" },
  { "cidr": "192.168.100.0/24", "name": "사내 SSL-VPN 접속망" }
]
```

---

## 5. API 키 관리 (Lifecycle) & 감사 로그 (Audit Trail)

### 5.1 API 키가 할 수 있는 일

개인 API 키(`mom_key_`)는 **분석 데이터를 읽기 위한 것**입니다. 스크립트나 BI 도구에서 사용합니다.

- **관리 작업은 어떤 것도 할 수 없습니다.** 키 소유자가 최고 관리자여도 마찬가지입니다 — 인증 종류로 거부하므로 역할과 무관합니다
- **변경 작업 전체가 막혀 있습니다.** 57개 변경 경로 중 51개가 키를 즉시 거부하고, 나머지 6개는 질의문이 길어 POST를 쓰는 **조회**입니다: 쿼리 빌더(`/query`), 퍼널(`/funnel`), 자연어 질의, 여정 분석 2종, 이벤트 계약 검증. 이 6개는 아무것도 쓰지 않습니다
- 이 경계는 통합 테스트가 라우터를 순회하며 매번 확인합니다. 변경 경로가 새로 추가되면서 이 보호가 빠지면 CI에서 경로 이름과 함께 실패합니다
- 키는 **소유자가 볼 수 있는 모든 사이트**를 읽습니다. 특정 사이트로 좁히는 기능은 없습니다. 응답의 `scopes` 필드는 현재 모든 키가 동일한 값(`analytics:read`)을 가지며, 위 제한은 이 필드가 아니라 인증 계층이 강제합니다
- 서버 사이드 수집에는 개인 키가 아니라 **사이트별 Server API Key**(`mom_server_`)를 사용합니다

### 5.1.1 API 키 발급 및 회전 (Rotation)
- API 키는 발급 시 단 1회만 원문이 표시되며, DB에는 SHA-256 해시값으로만 보관됩니다.
- 개인 키 회전 시 기존 키는 즉시 폐기되고 새 키가 1회 표시됩니다. 무중단 교체가 필요하면 새 키를 별도로 발급한 후 클라이언트를 전환하고 기존 키를 폐기하십시오.

### 5.2 감사 로그 (Audit Trail)
- 사이트 생성, 개인정보 설정 변경, Keycloak 설정 수정, API 키 발급/폐기 등의 작업은 애플리케이션 API를 통해 추가 전용 감사 로그로 기록됩니다. DB 운영자는 별도의 DB 권한 통제·백업·보존 정책으로 로그 무결성을 보호해야 합니다.

---

## 6. 사이트별 보존정책과 Dimension Registry

- `관리 ➔ 보존 정책`에서 Raw Event, Session 요약, Aggregation, Realtime, Debug/Dead Letter 보존기간을 사이트별로 지정합니다.
- Raw Event와 Session 정리는 매시간 실행되며, Session은 Raw Event보다 오래 보존할 수 있도록 별도 요약 테이블에 저장됩니다.
- `관리 ➔ 사용자 정의 차원`에서 User, Session, Event, Item Scope와 데이터 타입을 등록합니다. 활성 User/Session/Event 차원은 `custom.<name>`으로 Query와 Segment에서 사용할 수 있습니다.

## 7. 사이트 Timezone과 참여 세션 기준

- `관리 ➔ 사이트 ➔ 설정`에서 IANA Timezone(예: `Asia/Seoul`)과 참여 기준 시간(기본 10초)을 지정합니다.
- Event Timestamp는 UTC로 저장되지만 날짜 범위, 일별 Trend, Query, Funnel, Export와 MCP는 Site Timezone의 자정 경계를 사용합니다.
- 참여 세션은 `지속시간 ≥ 기준`, `Conversion 1회 이상`, `Page View 2회 이상`, `Active Engagement ≥ 기준` 중 하나를 충족하면 됩니다.
- 기준 시간을 바꾸면 기존 Session 요약의 `engaged` 값도 같은 트랜잭션에서 즉시 다시 계산됩니다.

## 8. 개인정보 삭제 일관성

Visitor, User ID, 기간 또는 Site 삭제는 PostgreSQL Inbox와 Dead Letter 원본 payload를 먼저 정리하고 Raw Event를 삭제한 뒤 남은 Raw Event에서 Session, Visitor, Identity Graph와 일별 집계를 재생성합니다. User ID 삭제 시 Identity Graph에 연결된 로그인 전 익명 Visitor와 다른 기기의 Raw Event도 함께 삭제합니다. Event Property 삭제도 처리 대기/Debug payload와 Raw Event에 함께 적용됩니다. 따라서 삭제된 데이터가 Worker 재시도로 복원되거나 파생 보고서에 잔존하지 않습니다.

## 9. Identity Graph와 파생 집계 운영

- `visitor_identities`는 `(site_id, visitor_id) → user_id`의 결정적 연결을 보관합니다.
- `identified_users`는 연결된 모든 Visitor에서 가장 이른 최초 활동과 최신 User Property를 유지합니다.
- `visitors`, `visitor_sessions`, `daily_site_metrics`, `daily_site_visitors`, `daily_site_sessions`는 Worker가 Raw Event와 같은 transaction에서 증분 갱신합니다.
- Overview의 Site-local 일별 Trend는 일별 집계를 사용하고, 임의 시각 범위는 Raw Event로 정확히 fallback합니다.
- Site Timezone을 바꾸면 Raw Timestamp는 그대로 두고 일별 집계만 새 Calendar 경계로 즉시 재생성합니다.
- 장애 복구와 개인정보 삭제에서는 Raw Event를 Single Source of Truth로 파생 데이터를 재생성합니다.

## 10. Environment와 Event Contract

- `Analytics Governance`에서 Site별 DEV/STG/PRD와 사용자 정의 Environment를 관리합니다.
- 각 Environment는 Event Contract 정책 `allow`, `warn`, `reject`와 일별 Cardinality Limit를 가집니다.
- Event Contract는 Version마다 JSON Schema, Validation Mode, Changelog, 작성자와 활성시각을 보관합니다.
- Draft는 수집에 사용할 수 없습니다. Active Version을 바꾸면 이전 Active는 Deprecated가 되지만 Retry 호환을 위해 계속 검증할 수 있습니다.
- `max_events_per_request`는 실제로 적용됩니다. 한도와 같은 크기의 배치는 수락되고 한도를 넘으면 거부되며, 거부 메시지에 현재 한도가 담기는지 통합 테스트로 확인합니다.
- 트래픽 분류(`known_bot`·`monitoring`·`suspicious`·`normal`)는 **User-Agent만으로** 결정됩니다. 사내망 여부는 이 값을 덮어쓰지 않고 `traffic.internal`이 따로 답합니다 — 사내망 크롤러는 `known_bot`이면서 내부입니다. 어느 쪽도 **리포트에서 자동으로 제외되지 않으며**, 제외는 Segment(`traffic.class`, `traffic.internal`)로 수행합니다. 분류 자체와 Segment 필터가 동작하는지 통합 테스트로 확인합니다.
- 서버 사이드 수집 규칙도 통합 테스트로 확인합니다. Origin 헤더가 없는 요청은 서버 간 호출로 보아 **Server API Key만** 허용하고 Tracking Key는 거부합니다. Tracking Key는 페이지 HTML에 노출되므로, 그것으로 서버 사이드 이벤트를 주입할 수 있어서는 안 됩니다. Origin이 있는 요청은 두 Key 모두 허용하되 허용 도메인 목록을 통과해야 합니다.
- 로그인은 IP 기준으로 제한됩니다. 잘못된 비밀번호를 반복하면 `RATE_LIMITED`(429)를 반환하는지 통합 테스트로 확인합니다.
- 접근 제어도 통합 테스트로 확인합니다. Analyst는 자기 Workspace의 사이트만 조회할 수 있고 다른 조직의 사이트에는 404를 받으며(사이트 존재를 확인해 주지 않기 위해 403이 아닙니다), 사이트 목록에도 나타나지 않습니다. 관리자 전용 엔드포인트는 403을 반환합니다. `user_workspace_roles`에 권한을 부여하면 같은 요청이 성공하고 회수하면 다시 거부되는 것까지 확인합니다.
- `visitor_profiles`를 끄면 super_admin에게도 방문자 목록·식별 사용자 목록·개인 타임라인이 차단됩니다. 권한 등급이 아니라 개인정보 정책이기 때문입니다. 사람을 지목하지 않는 리포트는 계속 동작합니다.
- Environment 격리는 통합 테스트로 확인합니다. `stg` 환경에 전용 Event 이름과 페이지를 시드한 뒤, `prd` 리포트에 그것이 나타나지 않고 `stg` 리포트에는 나타나는지, 그리고 무거운 리포트들이 두 환경에 대해 서로 다른 문서를 반환하는지 검사합니다.
- tracker가 스스로 보내는 Event(`page_view`, `click`, `outbound_click`, `file_download`, `scroll`, `form_start`, `form_submit`, `user_engagement`, `error`, `resource_error`, `web_vital`, `rage_click`, `dead_click`, `rapid_back`, `form_retry`, `repeated_search`, `error_after_click`, `slow_interaction`, `search`, `search_click`, `search_refine`)는 제품 구성 요소이므로 `reject` 모드에서도 미등록으로 거부되지 않습니다. 직접 등록하면 그 Site의 Schema와 Validation Mode가 그대로 적용됩니다. 이 예외가 없으면 `reject`를 켜는 순간 내장 Event가 섞인 모든 배치가 거부되고, 새 자동 신호가 추가될 때마다 기존 Site가 깨집니다.
- PRD를 비활성화할 수 없으며 SDK와 Server API가 Environment를 생략하면 PRD가 적용됩니다.

## 11. Semantic Metric과 Data Quality

- Semantic Metric은 관리자 정의 SQL을 받지 않고 허용된 JSON AST만 저장합니다.
- 수정 시 Version이 증가하고 REST/Query/MCP가 동일한 정의를 사용합니다.
- Session Scope Filter는 SDK/HTTP Event 발생 시점의 `session_properties`를 사용하며 Raw Event 전체 재빌드에서도 최신 Event 값 우선으로 복원됩니다.
- Data Quality는 Received, Accepted, Duplicate, Late, Reject, Contract Warning, Missing User/Feature, Unknown Network, PII Blocked, Dead Letter, Cardinality를 표시합니다.
- Cardinality 원문은 저장하지 않으며 SHA-256 digest로 Daily Distinct만 계산합니다.

## 12. Report / Action 보안

Scheduled Report를 사용하려면 먼저 관리자 설정 `automation`에서 기능을 활성화하고 `allowed_webhook_hosts`를 지정해야 합니다. 빈 Allowlist에서는 어떤 Endpoint도 호출하지 않습니다.

- 지원 Channel: Webhook, Confluence, Mail HTTP Gateway, Internal Message HTTP Gateway, AI Agent
- HTTP(S)만 허용하며 URL 내 Credential과 Redirect는 허용하지 않습니다.
- Channel Header 값은 저장 후 API에서 다시 노출하지 않습니다.
- Segment Delivery는 기본적으로 Aggregate만 전송하고 `max_entity_ids=0`입니다.
- 실행 결과는 Delivery Run과 Audit Log에서 확인합니다.

## 13. Analytics Engineering

- Formula Metric Builder는 Numerator/Denominator, 집계, Event, 최소 사용 횟수를 허용된 AST로 저장합니다. Metric마다 Owner, Entity Scope와 Tag를 지정하십시오.
- Goal Framework는 Metric, 목표값, `gte/lte`, 일/주/월/분기, Environment, 조직·부서 범위를 관리합니다.
- Query Policy는 Exact 최대 기간, Complexity 상한, Guarded 실행 기준과 Fast/Preview 표본 비율을 Site별로 제한합니다.
- Event Contract CI endpoint는 배포 전 미등록 Event, Version, Required Property, Deprecated 계약을 검사합니다.
- Event Catalog와 Lineage는 Event → Metric → Goal 사용 관계, Owner, First/Last Seen과 Volume을 표시합니다.

## 14. Aggregate와 Late Event 운영

Event가 수신 시각보다 한 시간 이상 과거이면 Momento는 Site Timezone의 해당 날짜에 `late_event` 재집계 Job을 한 건만 생성합니다. Maintenance Worker는 Raw Event를 기준으로 Site/Visitor/Session 일별 집계를 다시 계산합니다. 관리자는 Analytics Engineering에서 367일 이하 Date Range 또는 Full Rebuild를 요청할 수 있습니다.

## 15. 값 기반 PII와 Privacy Request

- `privacy.pii_detection_mode`는 `detect`, `warn`, `mask`, `reject` 중 하나입니다. 기본값은 `mask`입니다.
- Email, 한국 전화번호, 주민번호 형태, Luhn 검증 카드번호, Bearer/JWT Credential을 Inbox commit 전에 검사합니다.
- Data Quality Issue에는 Detector 종류만 기록하고 일치한 원문은 저장하지 않습니다.
- Privacy Request는 요청과 승인을 분리합니다. 승인 실행과 영향 건수는 Audit Log에 남습니다.
- 승인된 Export는 임의 행 제한 없이 NDJSON을 streaming하며 사용자·세션 속성도 포함합니다.

## 16. Workspace와 Experiment 운영

- Workspace Roll-Up과 Cross-Site Journey는 SSO User ID만 Site 간 결합합니다. 익명 ID는 Site 범위를 벗어나 결합하지 않습니다.
- Feature Flag는 2~20개 Variant를 등록할 수 있습니다.
- Experiment에는 `experiment_id`, `variant`, Primary Semantic Metric을 지정합니다. 첫 Variant가 Control이며 Lift와 두 비율 정규 근사 Confidence를 제공합니다.
- Change Calendar에 Release, Deployment, Incident, Campaign, Training, Feature Flag와 조직 변경을 기록하면 분석 시점의 원인 후보를 보존할 수 있습니다.

## 17. 관리 센터 운영 UX

- `관리 센터` 홈은 현재 Site·Environment의 7일 데이터 품질, 수신·대기·오류, Privacy Request와 Aggregate Job을 운영 브리핑으로 제공합니다.
- 운영 준비도는 활성 Site, Origin 제한, URL Query 제거, PII 값 정책, 관리자 이중화, OIDC 상태를 점검하며 각 미충족 항목에서 설정 화면으로 바로 이동할 수 있습니다.
- 조치 필요 목록은 수집 오류·Dead Letter·재집계 실패를 Critical로, 개인정보 승인 대기·품질 저하·보안 권고를 Warning/Info로 분류합니다.
- 개인정보 화면에서 PII 값 탐지 정책을 `detect`, `warn`, `mask`, `reject`로 변경할 수 있으며 허용되지 않은 값은 서버가 거부합니다.
- `관리 센터`는 서비스 설정, 보안·데이터, 접근 제어, Tracking 설계, 운영 도구를 업무 단위로 구분합니다.
- 각 설정은 `/admin?section=...` 형태의 URL을 가지므로 담당자에게 정확한 관리 화면을 공유하거나 Bookmark할 수 있습니다.
- `Ctrl+K` 또는 `Cmd+K` 명령 팔레트에서 사이트, 개인정보, 사용자·권한, Event Schema, Tracking Debugger 등으로 바로 이동할 수 있습니다. 명령은 현재 RBAC 권한에 맞게 노출됩니다.
- Analytics Engineering은 Metric·Goal, Query Cost, Aggregate, Change Calendar, Catalog·Lineage 탭으로 분리되고 Product Lab은 Feature Flag와 Experiment 탭으로 분리됩니다. `?panel=aggregate`, `?panel=experiments`처럼 탭 URL도 공유할 수 있습니다.
- Full Aggregate Rebuild와 개인정보 삭제 승인은 실행 전 환경과 영향 범위를 다시 확인하는 Dialog를 표시합니다. Privacy Request의 요청·승인 분리와 Audit 기록 원칙은 그대로 유지됩니다.
- 공통 관리 표는 검색, 페이지네이션, CSV 내보내기를 제공하며 빈 결과, Loading Skeleton, 오류와 재시도 상태를 일관되게 표시합니다.
