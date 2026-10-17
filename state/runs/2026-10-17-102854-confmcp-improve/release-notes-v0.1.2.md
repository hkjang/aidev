오프라인망에서 운영할 수 있는 confmcp 서비스 도커 이미지입니다. 웹 콘솔·폰트·마이그레이션이 모두 이미지에 들어 있어 실행 중 외부 다운로드가 없습니다.

## 설치

```bash
# 1) 이미지 적재
docker load -i confmcp-v0.1.2.tar.gz

# 2) 실행 (환경변수는 네 개뿐입니다)
docker run -d --name confmcp -p 8080:8080 \
  -e DATABASE_URL='postgres://confmcp:<비밀번호>@<DB 호스트>:5432/confmcp?sslmode=disable' \
  -e BOOTSTRAP_ADMIN=admin -e BOOTSTRAP_ADMIN_PASSWORD='<관리자 비밀번호>' \
  -e ENCRYPTION_KEY="$(openssl rand -base64 32)" \
  confmcp:v0.1.2
```

그다음 관리 콘솔(http://<호스트>:8080)에서 Keycloak, Confluence 연결, 권한 플러그인, 접근 정책을 설정합니다.
docker compose 구성과 업그레이드 절차는 [관리자 가이드](https://hkjang.github.io/confmcp/guide-admin.html)를 참고하십시오.

| 항목 | 값 |
|---|---|
| 이미지 | `confmcp:v0.1.2` |
| 파일 | `confmcp-v0.1.2.tar.gz` (15M) |
| SHA-256 | `3eb9aa480c9f9ba173d8a42756f01042a0077ce34c6fcd8bbbda8b3eade022b7` |

확인: `curl -s http://<호스트>:8080/api/version` 또는 로그인 화면·프로필 메뉴의 버전 표시

문서: https://hkjang.github.io/confmcp/
