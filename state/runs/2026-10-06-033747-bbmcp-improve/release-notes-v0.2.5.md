오프라인망에서 운영 가능한 서비스 도커 이미지입니다.

```bash
# 1) 이미지 적재
docker load -i bbmcp-v0.2.5.tar.gz

# 2) 환경변수 준비 (네 개만 필요합니다)
cp deploy/.env.example deploy/.env && $EDITOR deploy/.env

# 3) 기동
docker compose -f deploy/docker-compose.yml --env-file deploy/.env up -d
```

| 항목 | 값 |
|---|---|
| 이미지 | `bbmcp:v0.2.5` |
| 파일 | `bbmcp-v0.2.5.tar.gz` |
| SHA-256 | `9bfc309d3c149670df2121a9a4fce2df75277196e6ded5a710f3856753f93247` |

문서: https://hkjang.github.io/bbmcp/
