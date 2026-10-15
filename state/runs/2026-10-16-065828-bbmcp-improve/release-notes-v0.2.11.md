## 먼저 읽어 주십시오

**업그레이드는 `bash deploy/upgrade.sh bbmcp-v0.2.11.tar.gz` 로 하십시오** (`bbmcp-deploy-v0.2.11.tar.gz` 안에 있습니다).
이미지만 적재하고 `docker compose up -d` 를 하면 `deploy/.env` 의 예전 `BBMCP_VERSION` 으로 **예전 버전이 계속 돕니다**.

**MCP 로그인에 invalid_client_metadata 가 나면** — 관리 콘솔 → 인증 → MCP 클라이언트 진단 또는 관리자 가이드 3.5 를 확인하십시오.
Keycloak 13 이하에 직접 등록하는 클라이언트는 고정 클라이언트 ID 를 사용할 수 있습니다:
`claude mcp add --transport http --client-id bbmcp-mcp bbmcp https://<bbmcp>/mcp`.

## 변경 사항

PR 버전을 읽지 못하는 동안 **버전 고정이 없는 승인 요청**이 만들어지던 문제를 수정했습니다.
PR 병합·댓글 등 승인이 필요한 실행 도구를 쓰는 운영 환경에 해당하는 보안 패치입니다.

- Bitbucket PR 조회 실패(자격 증명 오류, 어댑터 오류, HTTP 오류)를 삼키고 `pr_version` 없이 승인 요청을
  저장하던 경로를 닫았습니다. 버전이 고정되지 않은 승인 요청은 승인 이후 PR 에 새 커밋이 올라와도
  승인 검사의 버전 비교를 건너뛰어 **영구히 통과**했습니다.
- 이제 승인 요청을 **만드는 시점**에 PR 버전을 읽지 못하면 `PERMISSION_UNKNOWN` 으로 거부하고
  승인 요청을 생성하지 않습니다. 운영자는 같은 요청을 Bitbucket 이 응답할 때 다시 보내면 됩니다.
- 승인을 **쓰는 시점**의 동작은 그대로입니다. 조회 실패 시 기존의 fail-closed 경로(`ErrStale`)로
  거부합니다.

이미 저장되어 있는 `pr_version` 이 비어 있는 대기 중 승인 요청은 이번 변경으로 정리되지 않습니다.
해당 요청이 남아 있다면 승인 전에 거절하고 다시 생성하도록 하십시오.

## 업그레이드 (이미 운영 중)

이미지만 적재하고 `docker compose up -d` 를 하면 deploy/.env 의 예전 BBMCP_VERSION 때문에 **예전 버전이 계속 돕니다**.
bbmcp-deploy-v0.2.11.tar.gz 의 deploy/upgrade.sh 를 쓰거나, BBMCP_VERSION=0.2.11 로 바꾼 뒤 기동하십시오.

```bash
bash deploy/upgrade.sh bbmcp-v0.2.11.tar.gz    # 적재, BBMCP_VERSION 변경, 재기동, 새 버전 기동 확인
```

확인 (에이전트 PC 에서): 응답 헤더 `X-Bbmcp-Version: 0.2.11`

```bash
curl -sD - https://<bbmcp 주소>/.well-known/oauth-protected-resource/mcp -o /dev/null | grep -i x-bbmcp-version
```

## 새로 설치

```bash
# 1) 이미지 적재
docker load -i bbmcp-v0.2.11.tar.gz

# 2) 환경변수 준비 (네 개만 필요합니다)
cp deploy/.env.example deploy/.env && $EDITOR deploy/.env

# 3) 기동
docker compose -f deploy/docker-compose.yml --env-file deploy/.env up -d
```

| 항목 | 값 |
|---|---|
| 이미지 | `bbmcp:v0.2.11` |
| 파일 | `bbmcp-v0.2.11.tar.gz` |
| SHA-256 | `31f0711e3751ab3610e3058a27eef9bd2ef3bfd79e38702c6e444447a9e08581` |
| 배포 파일 | `bbmcp-deploy-v0.2.11.tar.gz` (docker-compose.yml, .env.example, upgrade.sh) |
| SHA-256 | `0b86697a41950ae27cb9358cbb99363400c6a16647d04c387d49c92938942f65` |

SHA-256 은 파일이 손상되지 않았는지(무결성)를 확인할 뿐, 누가 만들었는지를 증명하지는 않습니다.

문서: https://hkjang.github.io/bbmcp/
