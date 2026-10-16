## 먼저 읽어 주십시오

**업그레이드는 `bash deploy/upgrade.sh bbmcp-v0.2.12.tar.gz` 로 하십시오** (`bbmcp-deploy-v0.2.12.tar.gz` 안에 있습니다).
이미지만 적재하고 `docker compose up -d` 를 하면 `deploy/.env` 의 예전 `BBMCP_VERSION` 으로 **예전 버전이 계속 돕니다**.

**MCP 로그인에 invalid_client_metadata 가 나면** — 관리 콘솔 → 인증 → MCP 클라이언트 진단 또는 관리자 가이드 3.5 를 확인하십시오.
Keycloak 13 이하에 직접 등록하는 클라이언트는 고정 클라이언트 ID 를 사용할 수 있습니다:
`claude mcp add --transport http --client-id bbmcp-mcp bbmcp https://<bbmcp>/mcp`.

## 변경 사항

승인 요청에 남는 도구 인자에서 **중첩된 자격 증명이 가려지지 않고 평문으로 저장되던 문제**를 수정했습니다.
승인이 필요한 실행 도구를 쓰는 운영 환경에 해당하는 보안 패치입니다.

- 인자 가리기가 최상위 키와 최상위 문자열만 검사해, `{"body":{"password":"…"}}` 처럼 한 단계 안쪽에 있는
  자격 증명이 `approval_requests.arguments_redacted` 에 평문으로 저장되고 관리 콘솔의 승인 화면
  (`GET /api/admin/approvals`) 에 그대로 보이던 문제를 수정했습니다. 이제 중첩된 맵과 배열까지 내려가며
  민감한 키를 `[redacted]` 로 바꿉니다.
- 깊이 8 단계 또는 요소 200 개 상한을 넘는 값은 내용을 확인하지 않고 `[redacted]` 로 저장합니다.
  상한에 걸린 하위 값은 승인 화면에서 내용이 보이지 않습니다(가려지는 쪽으로 동작합니다).
- 인자 표시를 2000바이트에서 자를 때 UTF-8 문자 중간이 끊겨 깨진 글자가 남던 문제를 수정했습니다.
  이제 문자 경계에서 자릅니다.

**이미 저장된 승인 레코드는 이번 업그레이드로 다시 가려지지 않습니다.** 과거 레코드에 평문 자격 증명이
남아 있을 수 있는 환경이라면, 해당 자격 증명을 교체하시는 것이 안전합니다.

승인 인자의 해시는 원본 인자로 계산하므로, 이번 변경으로 기존 승인 레코드의 해시가 달라지지 않습니다.

## 업그레이드 (이미 운영 중)

이미지만 적재하고 `docker compose up -d` 를 하면 deploy/.env 의 예전 BBMCP_VERSION 때문에 **예전 버전이 계속 돕니다**.
bbmcp-deploy-v0.2.12.tar.gz 의 deploy/upgrade.sh 를 쓰거나, BBMCP_VERSION=0.2.12 로 바꾼 뒤 기동하십시오.

```bash
bash deploy/upgrade.sh bbmcp-v0.2.12.tar.gz    # 적재, BBMCP_VERSION 변경, 재기동, 새 버전 기동 확인
```

확인 (에이전트 PC 에서): 응답 헤더 `X-Bbmcp-Version: 0.2.12`

```bash
curl -sD - https://<bbmcp 주소>/.well-known/oauth-protected-resource/mcp -o /dev/null | grep -i x-bbmcp-version
```

## 새로 설치

```bash
# 1) 이미지 적재
docker load -i bbmcp-v0.2.12.tar.gz

# 2) 환경변수 준비 (네 개만 필요합니다)
cp deploy/.env.example deploy/.env && $EDITOR deploy/.env

# 3) 기동
docker compose -f deploy/docker-compose.yml --env-file deploy/.env up -d
```

| 항목 | 값 |
|---|---|
| 이미지 | `bbmcp:v0.2.12` |
| 파일 | `bbmcp-v0.2.12.tar.gz` |
| SHA-256 | `42283110380f2e040134167c56a2c5e57c8ea015246ac925121ecaf931d7dd5f` |
| 배포 파일 | `bbmcp-deploy-v0.2.12.tar.gz` (docker-compose.yml, .env.example, upgrade.sh) |
| SHA-256 | `99210113737d3e5ee05d5c6ab24c0d2ddbe1e00d438165d5e4d4cc9f9de1e56e` |

SHA-256 은 파일이 손상되지 않았는지(무결성)를 확인할 뿐, 누가 만들었는지를 증명하지는 않습니다.

문서: https://hkjang.github.io/bbmcp/
