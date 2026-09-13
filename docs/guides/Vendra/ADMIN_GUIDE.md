# Vendra 엔터프라이즈 관리자 가이드 (Admin & Operational Guide)

- **문서 버전**: v1.0.0-ENTERPRISE  
- **작성일자**: 2026년 8월 13일  
- **대상**: 시스템 관리자, Security/DevOps 엔지니어, 구매/조달 책임자  
- **문서 개요**: Vendra 4대 환경변수 부트스트랩, Keycloak OIDC SSO, ENCRYPTION_KEY 시크릿/계좌 암호화, 감사 로그 및 백업 관리  

---

## 1. 시스템 부트스트랩 및 필수 환경변수 (Bootstrap Specification)

Vendra 컨테이너 프로세스는 오직 **4개의 애플리케이션 환경변수**만으로 최소 인프라 구축 및 부트스트랩을 완수합니다.

```bash
# Vendra 실행 환경변수 명세
POSTGRES_DSN=postgres://vendra:Secr3tPass@10.10.60.5:5432/vendra?sslmode=disable
BOOTSTRAP_ADMIN=admin@company.com
BOOTSTRAP_ADMIN_PASSWORD=SuperSecretAdminPassword123!
ENCRYPTION_KEY=Base64Encoded32ByteKeyHere==
```

> **비상 관리자(Break Glass) 및 암호화 마스터 키**:  
> `BOOTSTRAP_ADMIN` 계정은 삭제가 불가능한 비상 복구 계정이며, `ENCRYPTION_KEY`는 Base64로 인코딩된 32바이트 암호화 키입니다 (`openssl rand -base64 32`).

---

## 2. 데이터 백업 및 시크릿 암호화 (`ENCRYPTION_KEY`)

```bash
docker run -d \
  --name vendra \
  -p 8080:8080 \
  -e POSTGRES_DSN="postgres://vendra:password@postgres:5432/vendra" \
  -e BOOTSTRAP_ADMIN="admin@company.com" \
  -e BOOTSTRAP_ADMIN_PASSWORD="change-this-strong-password" \
  -e ENCRYPTION_KEY="Base64Encoded32ByteKeyHere==" \
  vendra:latest
```

### 2.1 계좌정보 및 시크릿 AES-256-GCM 암호화
`ENCRYPTION_KEY` 환경변수는 DB 내에 보관된 Keycloak Client Secret, 공급업체 계좌정보 및 개인 API/MCP 시크릿을 암호화(AES-256-GCM)하는 마스터 키로 정기 백업이 필수적입니다.

---

## 3. Keycloak OIDC SSO 및 RBAC 그룹 매핑

- **OIDC Discovery**: Keycloak Discovery 엔드포인트를 등록하고 Authorization Code + PKCE (S256) 인증을 켭니다.
- **Valid Redirect URI**: `https://vendra.internal/api/auth/oidc/callback`
  - 인증 경로는 `/api/v1` 아래가 아닙니다. `/api/v1/...`로 등록하면 SSO를 마치고 돌아온 사용자가 세션 인증 미들웨어에 걸려 `401 로그인이 필요합니다`를 받습니다 — 방금 로그인한 사람에게 가장 헷갈리는 응답입니다.
  - 서비스 관리에서 **공개 URL**을 설정하면 서버가 `<공개 URL>/api/auth/oidc/callback`을 authorization 요청과 token 교환 양쪽에 같은 문자열로 사용합니다. 비워두면 요청에서 유추하므로, 프록시와 원본처럼 이름이 둘 이상인 배포에서는 `redirect_uri_mismatch`가 날 수 있습니다.
- **그룹 매핑**: Keycloak `/vendra-admins`, `/vendra-buyers` 그룹을 사내 권한 그룹으로 맵핑하여 자동 RBAC 부여.

---

## 4. 무결성 감사 로그 (Audit Trail) 및 문서 저장소 백업

- **감사 로그 (Audit Trail)**: 계약 체결, 계좌 정보 변경, 승인 수락/반려, 스코어카드 수정 등 모든 중요 액션이 사용자 ID 및 IP 주소와 함께 무결성 감사 레코드로 영구 기록됩니다.
- **문서 저장소 백업**: DB 백업과 함께 `vendra_documents` 볼륨을 동일 시점에 백업합니다.
