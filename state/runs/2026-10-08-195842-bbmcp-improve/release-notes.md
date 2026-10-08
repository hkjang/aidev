## 먼저 읽어 주십시오

**업그레이드는 `bash deploy/upgrade.sh bbmcp-v0.2.10.tar.gz` 로 하십시오** (`bbmcp-deploy-v0.2.10.tar.gz` 안에 있습니다).
이미지만 적재하고 `docker compose up -d` 를 하면 `deploy/.env` 의 예전 `BBMCP_VERSION` 으로 **예전 버전이 계속 돕니다**.

**MCP 로그인에 invalid_client_metadata 가 나면** — 관리 콘솔 → 인증 → MCP 클라이언트 진단 또는 관리자 가이드 3.5 를 확인하십시오.
Keycloak 13 이하에 직접 등록하는 클라이언트는 고정 클라이언트 ID 를 사용할 수 있습니다:
`claude mcp add --transport http --client-id bbmcp-mcp bbmcp https://<bbmcp>/mcp`.

## 변경 사항

PR 실행 도구에서 대상 브랜치를 확인할 수 없을 때 브랜치 제한 검사를 건너뛰던 문제를 수정했습니다.
PR 병합 등 승인이 필요한 실행 도구를 사용하는 운영 환경에 해당하는 패치입니다.

- Bitbucket 대상 PR 조회 실패(예: HTTP 503), 빈 ref, 접두사 제거 후 빈 ref 는 승인 검사 전에 `PERMISSION_UNKNOWN` 으로 거부합니다. 오류 설명은 고정 문구를 사용합니다.
- 이 경우 새 승인 요청을 만들지 않으며, 이미 승인된 요청은 소비하지 않고 실행을 차단합니다.
- 조회가 복구되면 브랜치 제한을 다시 검사합니다. 제한에 걸리면 `BRANCH_RESTRICTED` 로 거부하며, 정상 브랜치에서는 여전히 유효한 기존 승인으로 실행할 수 있습니다.
- 새 설정, 의존성 또는 데이터베이스 마이그레이션은 없습니다. 기존 승인 유효성·인자 일치·버전·동시 소비 검사는 유지됩니다.

`PERMISSION_UNKNOWN` 이 계속되면 Bitbucket 연결과 대상 PR 의 브랜치 응답을 확인한 뒤 재시도하십시오.
이 패치는 MCP 로그인 설정을 바꾸거나 새 인증 기능을 추가하지 않습니다.

## 검증

- Go 빌드·vet 및 전용 PostgreSQL 전체 테스트: 9개 패키지, 하위 테스트 포함 131 PASS, SKIP/FAIL 없음.
- 웹 의존성 설치, 타입 검사, 프로덕션 빌드와 릴리즈 버전 일치 검사 통과.
- 생성한 이미지 tar.gz 재적재 후 격리된 테스트 DB 로 기동하여 헬스·설정 API·웹 콘솔·버전/커밋·버전 헤더를 확인했습니다. 두 묶음의 SHA-256 과 배포 파일 내용·권한도 검증했습니다.
- 실제 운영 Bitbucket 연동과 운영 서버 배포는 이번 로컬 릴리즈 단계에서 실행하지 않았습니다.

오프라인망에서 운영 가능한 서비스 도커 이미지입니다.

## 업그레이드 (이미 운영 중)

이미지만 적재하고 `docker compose up -d` 를 하면 deploy/.env 의 예전 BBMCP_VERSION 때문에 **예전 버전이 계속 돕니다**.
bbmcp-deploy-v0.2.10.tar.gz 의 deploy/upgrade.sh 를 쓰거나, BBMCP_VERSION=0.2.10 로 바꾼 뒤 기동하십시오.

```bash
bash deploy/upgrade.sh bbmcp-v0.2.10.tar.gz    # 적재, BBMCP_VERSION 변경, 재기동, 새 버전 기동 확인
```

확인 (에이전트 PC 에서): 응답 헤더 `X-Bbmcp-Version: 0.2.10`

```bash
curl -sD - https://<bbmcp 주소>/.well-known/oauth-protected-resource/mcp -o /dev/null | grep -i x-bbmcp-version
```

## 새로 설치

```bash
# 1) 이미지 적재
docker load -i bbmcp-v0.2.10.tar.gz

# 2) 환경변수 준비 (네 개만 필요합니다)
cp deploy/.env.example deploy/.env && $EDITOR deploy/.env

# 3) 기동
docker compose -f deploy/docker-compose.yml --env-file deploy/.env up -d
```

| 항목 | 값 |
|---|---|
| 이미지 | `bbmcp:v0.2.10` |
| 파일 | `bbmcp-v0.2.10.tar.gz` |
| SHA-256 | `d115e45ef4a681bd5b9ccd3b0406ad2d826cb6ed87d066e8b21cd03b422a0330` |
| 배포 파일 | `bbmcp-deploy-v0.2.10.tar.gz` (docker-compose.yml, .env.example, upgrade.sh) |
| SHA-256 | `b1098374423ac169b57bead6f1d03bbc125cd7b58157ea3dc7e057f6b8defb90` |

SHA-256 은 파일이 손상되지 않았는지(무결성)를 확인할 뿐, 누가 만들었는지를 증명하지는 않습니다.

문서: https://hkjang.github.io/bbmcp/
