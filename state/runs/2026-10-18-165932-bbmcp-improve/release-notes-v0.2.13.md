## 먼저 읽어 주십시오

**업그레이드는 `bash deploy/upgrade.sh bbmcp-v0.2.13.tar.gz` 로 하십시오** (`bbmcp-deploy-v0.2.13.tar.gz` 안에 있습니다).
이미지만 적재하고 `docker compose up -d` 를 하면 `deploy/.env` 의 예전 `BBMCP_VERSION` 으로 **예전 버전이 계속 돕니다**.

**MCP 로그인에 invalid_client_metadata 가 나면** — 관리 콘솔 → 인증 → MCP 클라이언트 진단 또는 관리자 가이드 3.5 를 확인하십시오.
Keycloak 13 이하에 직접 등록하는 클라이언트는 고정 클라이언트 ID 를 사용할 수 있습니다:
`claude mcp add --transport http --client-id bbmcp-mcp bbmcp https://<bbmcp>/mcp`.

## 변경 사항

권한 결정 캐시가 **프로세스 수명 내내 무제한으로 커지던 문제**를 수정했습니다.
재기동 없이 오래 띄워 두는 운영 환경에 해당하는 안정성 패치입니다.

- 권한 결정 TTL 캐시에서 항목이 사라지는 경로는 "그 키를 다시 조회했을 때 만료되어 있으면 삭제" 와 설정 변경 시의 전체 비우기뿐이었습니다. 그래서 **다시 조회되지 않는 키는 만료된 뒤에도 맵에 그대로 남았습니다.** 존재하지 않는 저장소도 "권한 없음" 결정으로 캐시되므로, 서로 다른 저장소 이름으로 들어오는 호출마다 항목이 하나씩 쌓였습니다.
- 이제 항목 수 상한(10000)에 닿으면 삽입 전에 만료된 항목을 먼저 쓸어내고, 전부 유효하면 만료가 가장 이른 항목 하나를 비웁니다. 백그라운드 정리 고루틴은 두지 않았습니다.
- 회수는 어떤 항목의 만료 시각도 늘리지 않고, 만료된 결정을 되살리지도 않습니다. 회수된 키는 다음 호출에서 Bitbucket 에 다시 조회될 뿐이므로 **캐시 때문에 허용되는 범위가 넓어지는 변화는 없습니다.**
- 그동안 비어 있던 권한 캐시 경로에 단위 테스트 6건을 새로 붙였습니다 — 상한 유지, TTL 0 이하일 때 미캐시, 만료 항목 미적중·삭제, 회수 시 만료 시각 불변·만료 결정 미부활, 설정 변경 후 전체 미적중, 동시 삽입 시 상한 유지.

설정·API·권한 판정 결과는 바뀌지 않았습니다. 마이그레이션은 기동 시 자동 적용됩니다.

## 업그레이드 (이미 운영 중)

이미지만 적재하고 `docker compose up -d` 를 하면 deploy/.env 의 예전 BBMCP_VERSION 때문에 **예전 버전이 계속 돕니다**.
bbmcp-deploy-v0.2.13.tar.gz 의 deploy/upgrade.sh 를 쓰거나, BBMCP_VERSION=0.2.13 로 바꾼 뒤 기동하십시오.

```bash
bash deploy/upgrade.sh bbmcp-v0.2.13.tar.gz    # 적재, BBMCP_VERSION 변경, 재기동, 새 버전 기동 확인
```

확인 (에이전트 PC 에서): 응답 헤더 `X-Bbmcp-Version: 0.2.13`

```bash
curl -sD - https://<bbmcp 주소>/.well-known/oauth-protected-resource/mcp -o /dev/null | grep -i x-bbmcp-version
```

## 새로 설치

```bash
# 1) 이미지 적재
docker load -i bbmcp-v0.2.13.tar.gz

# 2) 환경변수 준비 (네 개만 필요합니다)
cp deploy/.env.example deploy/.env && $EDITOR deploy/.env

# 3) 기동
docker compose -f deploy/docker-compose.yml --env-file deploy/.env up -d
```

| 항목 | 값 |
|---|---|
| 이미지 | `bbmcp:v0.2.13` |
| 파일 | `bbmcp-v0.2.13.tar.gz` |
| SHA-256 | `7b2ff846d6e8af2c1c8df524e348b2f358cb88ac8c2cb26fa9cdc607064943c4` |
| 배포 파일 | `bbmcp-deploy-v0.2.13.tar.gz` (docker-compose.yml, .env.example, upgrade.sh) |
| SHA-256 | `e6df6acd957a47ad7888e1d69afce9d63e4a8179ef349c2f8ef6982bfd8720b6` |

SHA-256 은 파일이 손상되지 않았는지(무결성)를 확인할 뿐, 누가 만들었는지를 증명하지는 않습니다.

문서: https://hkjang.github.io/bbmcp/
