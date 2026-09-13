# Kanvas 엔터프라이즈 관리자 가이드 (Admin & Operational Guide)

- **문서 버전**: v0.4.0
- **작성일자**: 2026년 8월 9일  
- **대상**: 시스템 관리자, Security/DevOps 엔지니어, 데이터 마이그레이션 담당자  
- **문서 개요**: Kanvas 4대 환경변수 부트스트랩, Confluence Discovery & Migration Machine 제어, Keycloak OIDC SSO 및 감사 로그 운영  

---

## 1. 시스템 부트스트랩 (Bootstrap Environment Variables)

Kanvas 프로세스는 오직 **4개의 필수 환경변수**만으로 전체 시스템 부트스트랩을 완수합니다.

```bash
# deploy 환경변수 설정 예시
KANVAS_POSTGRES_DSN=postgres://kanvas:Secr3tPass@10.10.40.5:5432/kanvas?sslmode=disable
KANVAS_CONFLUENCE_DSN=root:ReadOnlyMySQLPass@tcp(10.10.40.8:3306)/confluence
KANVAS_BOOTSTRAP_ADMIN=admin
KANVAS_BOOTSTRAP_ADMIN_PASSWORD=SuperSecretAdminPassword123!
```

> **설정 원칙**:  
> `KANVAS_CONFLUENCE_DSN`은 마이그레이션 탐색 전까지 빈 문자열(`""`)로 설정 가능합니다. 부트스트랩 관리자는 최초 계정이 없을 때만 생성되며, OIDC 설정, 첨부파일 경로, Migration 상태는 모두 관리자 대시보드에서 동적으로 제어됩니다.

---

## 2. Confluence Migration Machine 및 Cutover Gate 운영

Confluence 데이터를 무중단 이관하기 위한 4단계 상태 머신 운영 절차:

1. **DISCOVERY**: Read-Only Confluence MySQL 스키마 탐색 및 연결 검증.
2. **SNAPSHOT**: 테이블을 `Core`, `AO`, `Unknown`으로 분류한 Profile을 기준으로 canonical schema에 반복 가능한 Initial Copy를 수행합니다.
3. **RECONCILIATION**: Snapshot과 독립된 Job으로 건수·해시·권한·링크·Macro 근거를 다시 계산합니다.
4. **CUTOVER GATE**: 13개 필수 근거가 모두 `PASS` 또는 사유가 기록된 `APPROVED`일 때만 다음 전환을 허용합니다. CDC가 제공되기 전에는 `CDC_SYNC` 진입을 서버에서 거부합니다.

---

## 3. Keycloak OIDC SSO 및 계정 관리

- **OIDC Discovery**: Keycloak Discovery 엔드포인트를 등록하고 Authorization Code + PKCE (S256) 인증을 켭니다.
- **Valid Redirect URI**: `https://kanvas.internal/api/v1/auth/oidc/callback`
- **Group mapping**: Keycloak 그룹을 Kanvas 계층 그룹으로 맵핑하여 자동 ACL 권한 할당.

---

## 4. API / MCP 키 관리 & 감사 로그 (Audit Log)

- **원자적 개인 키 회전**: 개인화 → API 및 MCP 키에서 새 키를 발급하거나 회전합니다. 회전 시 기존 키는 같은 트랜잭션에서 즉시 폐기되고 새 평문은 한 번만 표시됩니다.
- **감사 로그 (Audit Trail)**: 문서 삭제, ACL 변경, Migration Cutover 승인 등 모든 관리자 및 사용자 활동이 DB 감사 테이블에 무결성 보장 상태로 영구 보관됩니다.

---

## 5. 서비스 관리 메뉴

서비스 관리 화면은 일반 개인화 화면과 분리되어 있으며 `ADMIN` 역할만 접근할 수 있습니다.

| 메뉴 그룹 | 메뉴 | 운영 기능 |
|---|---|---|
| 운영 | 개요 | 사용자·Space·세션·키·감사·Migration 준비도 통합 관제 |
| 운영 | 서비스 상태 | PostgreSQL pool, uptime, goroutine, memory, build 정보 5초 갱신 |
| 운영 | 감사 로그 | 작업·행위자·리소스 검색, 작업 필터, 상세 보기, CSV 내보내기 |
| 워크스페이스 | 사용자 | 역할 변경, 활성·비활성 전환, 사용자 검색과 상태 필터 |
| 워크스페이스 | 그룹 | 그룹 생성, 멤버 추가·제거, ACL 그룹 현황 확인 |
| 워크스페이스 | Space | 문서·첨부 수 확인, Space 보관과 복원 |
| 데이터 및 전환 | 데이터 원본 | DSN fingerprint, PostgreSQL 진단, Snapshot 처리 정책 |
| 데이터 및 전환 | Migration Center | Discovery, Snapshot, 오류 확인·재개, Macro·Cutover Gate 근거 |
| 데이터 및 전환 | 예외 콘텐츠 | 미지원 Macro·XHTML·고아 객체 검색, 일괄 위험 승인·해결·재오픈 및 감사 근거 |
| 플랫폼 | 인증 및 SSO | Keycloak Issuer, Client ID/Secret, 그룹 claim과 관리자 그룹 |
| 플랫폼 | API 및 MCP | 개인 키·세션 현황, REST/MCP endpoint와 보안 기본값 |
| 플랫폼 | 운영 설정 | 서비스 표시명, 기준 URL, 세션 유지 시간, 부팅 환경 확인 |

사용자 비활성화 시 해당 사용자의 세션은 삭제되고 개인 API 키는 모두 폐기됩니다. 현재 로그인한 관리자 자신의 비활성화와 마지막 활성 관리자 강등·비활성화는 서버에서 거부합니다.
