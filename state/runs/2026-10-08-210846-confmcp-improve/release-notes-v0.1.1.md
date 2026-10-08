오프라인망에서 운영할 수 있는 confmcp 서비스 도커 이미지입니다. 웹 콘솔·폰트·마이그레이션이 모두 이미지에 들어 있어 실행 중 외부 다운로드가 없습니다.

## 설치

```bash
# 1) 이미지 적재
docker load -i confmcp-v0.1.1.tar.gz

# 2) 실행 (환경변수는 네 개뿐입니다)
docker run -d --name confmcp -p 8080:8080 \
  -e DATABASE_URL='postgres://confmcp:<비밀번호>@<DB 호스트>:5432/confmcp?sslmode=disable' \
  -e BOOTSTRAP_ADMIN=admin -e BOOTSTRAP_ADMIN_PASSWORD='<관리자 비밀번호>' \
  -e ENCRYPTION_KEY="$(openssl rand -base64 32)" \
  confmcp:v0.1.1
```

그다음 관리 콘솔(http://<호스트>:8080)에서 Keycloak, Confluence 연결, 권한 플러그인, 접근 정책을 설정합니다.
docker compose 구성과 업그레이드 절차는 [관리자 가이드](https://hkjang.github.io/confmcp/guide-admin.html)를 참고하십시오.

| 항목 | 값 |
|---|---|
| 이미지 | `confmcp:v0.1.1` |
| 파일 | `confmcp-v0.1.1.tar.gz` (15M) |
| SHA-256 | `bc649e7b19f74fb14364df44be9b7fd328ac9d8eef33b1338a401bc1e7815663` |

확인: `curl -s http://<호스트>:8080/api/version` 또는 로그인 화면·프로필 메뉴의 버전 표시

문서: https://hkjang.github.io/confmcp/

## 수정 사항

- `confluence_update_page`의 `replace_text` 모드에서 `find`의 앞뒤 공백을 그대로 검색합니다. 예를 들어 `<p>cat| cat |cat</p>`에서 ` cat `을 ` dog `으로 바꾸면 `<p>cat| dog |cat</p>`가 됩니다.
- `find`와 `body`의 공백 변경도 승인 해시에 반영하여, 승인 후 공백만 바꾼 요청으로 다른 내용을 쓰지 못하도록 수정했습니다.
- 대상은 승인 기반 텍스트 치환을 사용하는 기존 운영자·사용자입니다. DB 스키마 변경은 없으며 Java 권한 플러그인은 기존 v0.1.0을 사용합니다.

## 업그레이드·롤백

기존 설치는 DB를 백업하고 기존 환경변수·암호화 키를 유지한 채 `bash deploy/upgrade.sh /path/to/confmcp-v0.1.1.tar.gz`로 업그레이드합니다. 위 신규 설치 예시의 키 생성 명령은 기존 설치에 사용하지 마십시오.

- 공백이 포함된 기존 `replace_text` 변경안은 새 해시와 달라 재승인이 필요할 수 있습니다.
- 배포·롤백 시 같은 재시도 키로 이전 성공 결과를 조회하는 대신 충돌 또는 `APPROVAL_STALE` 응답을 받을 수 있습니다. 현재 문서와 실행 이력을 먼저 확인한 뒤 변경안을 다시 작성하고 승인하십시오.
- 롤백이 필요하면 기존 v0.1.0 이미지 파일로 같은 업그레이드 절차를 실행하고 기존 DB·암호화 키를 유지합니다. v0.1.0에는 이번 공백 처리 문제가 남아 있으므로 문제가 되는 치환 요청은 중지하십시오. 롤백해도 이미 쓴 문서 내용은 되돌아가지 않습니다.
- 운영 담당자는 먼저 제한된 사용자로 공백 포함 치환 결과와 승인 후 공백 변경 거부를 확인하십시오. 잘못된 위치의 치환 또는 변경된 인자의 승인 우회가 1건이라도 나타나면 쓰기를 중지하고 확대 배포를 보류하십시오.

## 검증

- 릴리즈 단계에서 별도 PostgreSQL 테스트 DB를 사용하는 전체 `go test ./... -count=1 -p 1`, `go build ./...`, `go vet ./...`, 웹 `npm ci`·`npm run check`·`npm run build` 통과.
- 기존 `scripts/build.sh --save` 방식으로 이미지·tar.gz·SHA-256 생성. 압축 파일의 체크섬 검증과 `docker load` 후 동일 이미지에서 `/healthz`, `/readyz`, `/api/config`, `/`, 내장 JS/CSS, `/api/version`, OAuth 응답 버전 헤더 확인. 버전 `0.1.1`, 빌드 커밋 `9a153b4` 확인.
- 구현 단계에서는 실제 cmd/server·PostgreSQL·모의 Confluence를 통한 승인→쓰기→재조회 6개 회귀 사례의 수정 전 실패·수정 후 통과·원복 후 재실패, 전체 e2e 통과(34.022초)를 확인했습니다. 릴리즈 단계에서 e2e는 재실행하지 않았습니다.
- 실제 Confluence 7.2·Java 권한 플러그인·Keycloak 연동과 운영 배포는 이 환경에서 검증하지 않았습니다. 실제 연동 검증이 필요한 운영 환경에서는 먼저 해당 검증을 완료하십시오.
