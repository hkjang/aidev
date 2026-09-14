## Data Works v0.9.54

### 주요 변경 사항
- **Keycloak 세션이 있으면 로그인 화면 없이 진입하는 Silent SSO(OIDC prompt=none)와 `auto_login` 설정 추가 (v0.9.54)**: Keycloak 에 이미 로그인한 사용자도 워크벤치에 들어올 때마다 로그인 화면에서 'Keycloak SSO로 계속' 을 눌러야 했음. 서버에 `KeycloakConfig.AutoLogin`(env `SSO_KEYCLOAK_AUTO_LOGIN`, 기본 false)과 DB 오버라이드 `sso_provider_config.auto_login`(관리 API `GET/PUT /admin/sso/keycloak/config` 의 `auto_login`, React 설정 화면·레거시 콘솔 토글)을 추가하고 `GET /auth/sso/status` 가 `auto_login`(SSO 켜짐 && auto_login)을 공개함. `GET /auth/keycloak/login?prompt=none&return_to=…` 는 auto_login 이 꺼져 있으면 prompt 를 조용히 버리고 평범한 대화형 로그인으로 바꾸며, 흐름 상태(`oidc_flow_states` 에 `silent`·`return_to` 컬럼 추가, in-memory 미러 동일)에 silent 여부와 착지 경로를 실음. 콜백은 silent 흐름의 `error=login_required` 를 실패로 보고하지 않고 원래 깊은 링크에 `?sso=none` 을 붙여 로그인 화면으로 보내고, 성공 시 `return_to` 로 돌아감. `return_to` 는 `/` 로 시작·`//` 아님·상대 경로·host 없음에 더해 토큰이 fragment 로 전달되는 `/dataworks*`·`/admin*` 만 허용(`safeReturnTo`). 브라우저(`web/src/features/auth/silent-sso.ts`)는 sessionStorage 의 '한 탭 세션에 한 번'(`dataworks.sso.silentAttempted`)·로그아웃 억제(`dataworks.sso.signedOut`)·주소의 `?sso=none|error` 표시·kc_error 동반 시 억제·`/auth/*` 및 워크벤치 밖 경로 제외로 루프를 막고, 저장소 읽기 예외는 '이미 시도했다'로 판정(fail-closed). auth-store 는 세션이 없을 때 status 를 보고 조건이 맞으면 'checking' 로더를 유지한 채 최상위 이동하고, 세션이 생기면 억제를 풀고 주소의 sso 표시를 지우며, 로그아웃 시 억제 표시를 남김. 'Keycloak SSO로 계속' 버튼도 return_to 를 실어 대화형 로그인 뒤 같은 깊은 링크로 돌아감. Go 테스트 8건(safeReturnTo·refusal 주소 표, status 공개 조건, auto_login off 에서 prompt 제거·on 에서 전달과 `//` return_to 폐기, silent 거절 → 깊은 링크+`?sso=none`, state 재사용 시 실패, 대화형 오류는 종전 kc_error 유지, 스텁 토큰 엔드포인트로 실제 라우트 경유 silent 성공, 설정 저장 왕복)과 vitest 11건을 추가했고, 빌드된 SPA 를 임베드한 서버·Node 스텁 Keycloak·headless Chromium 으로 IdP 세션 없음/있음·로그아웃 후·auto_login 꺼짐 시나리오 15개 항목을 확인. `docs/ADMIN_GUIDE.md` 4.3 절에 "자동 로그인 (Silent SSO)" 소절(동작·세 겹 루프 방지 표·점검 항목)과 4.6 환경 변수 표에 `SSO_KEYCLOAK_AUTO_LOGIN` 을 추가하고 PDF 를 다시 생성.
- **버전·커밋·빌드 시각이 바이너리와 이미지에 실제로 새겨지게 수정 (v0.9.54)**: Dockerfile 이 `-X main.version=${VERSION}` 으로 버전을 새기고 있었지만 main 패키지에 `version` 이 없어 링커가 조용히 무시했고, 어느 빌드가 떠 있는지 알 길이 없었음. `internal/buildinfo`(Version·Commit·BuildTime, 미주입 시 `dev`·`unknown`)를 추가해 링크 시 주입하고 기동 첫 로그 줄에 `Data Works starting version=… commit=… built_at=…` 으로 남기며, 이미지에 OCI 라벨(`org.opencontainers.image.version`·`revision`·`created` 등 6개)을 붙임. `dataworks:v0.9.54` 이미지를 `dataworks-v0.9.54.tar.gz` 단일 오프라인 GitHub Release asset으로 배포.


### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.54.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c dataworks-v0.9.54.tar.gz | docker load

# 실행
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres:5432/dataworks?sslmode=disable' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='change-me' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.54
```
