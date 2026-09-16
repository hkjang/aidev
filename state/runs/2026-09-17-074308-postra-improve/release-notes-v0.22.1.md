# Postra v0.22.1 — OIDC Discovery·제공자 오류 장애 기록

OIDC SSO 로그인이 인증 서버 쪽 사정으로 실패했을 때 그 사유가 로그인을 시도한 사람의 브라우저에만 남던 진단 공백을 메운 패치 릴리즈입니다. Discovery 실패와 인증 서버가 콜백에 돌려준 오류를 관리자 장애 화면에 기록하며, 설정·환경변수·DB 변경은 없습니다.

## OIDC Discovery 실패 분류와 장애 기록

- 로그인 시작(`/auth/oidc/start`)과 콜백 두 경로 모두 같은 Discovery 단계를 거치며, 실패하면 오류 유형과 HTTP 상태만으로 원인을 분류합니다: Issuer URL 의 호스트 없음, 연결 거부, 시간 초과, TLS 인증서 검증 실패, HTTP 상태 코드(`HTTP 404` 등), Discovery 문서의 issuer 불일치, OpenID 설정 JSON 이 아닌 응답.
- 분류한 사유는 서버 로그 경고와 장애 화면의 `oidc` 구성 요소 행으로 남기며, detail 에 설정된 Issuer URL 을 표시합니다. 같은 원인이 반복되면 새 행을 만들지 않고 기존 행의 횟수를 올립니다.
- 로그인 화면에는 이전과 같이 `/app/login?sso=error` 로 이동한 뒤 분류한 사유와 확인할 항목(끝 슬래시·스킴·호스트, 인증서 체인과 신뢰 저장소, DNS, 네트워크 경로·방화벽, 프록시)을 한 번만 보여 줍니다.
- 인증 서버가 돌려준 응답 본문, Discovery 문서의 issuer 값 등 제공자가 통제하는 문자열은 로그·장애 기록·로그인 화면 어디에도 복사하지 않습니다. 잘못 연결된 Issuer 가 로그인 페이지나 프록시 오류 페이지를 돌려주더라도 세 자리 상태 코드만 남깁니다.

## 콜백 제공자 오류 기록

- 인증 서버가 콜백에 `?error=` 로 돌려준 오류를 장애 화면에 기록합니다. `invalid_scope`, `unauthorized_client` 등 Postra 가 알고 있는 코드는 행 이름에 코드를 붙여 따로 모으고, 모르는 코드는 일반 행 하나로 접습니다.
- 사용자 개인의 거절·재로그인 필요를 뜻하는 `access_denied`, `login_required` 계열, `account_selection_required` 는 경고(warning) 심각도로 기록해 클라이언트 설정 오류와 구분합니다.
- 자동 로그인의 조용한 시도(`prompt=none`)가 `login_required` 로 거절되는 것은 정상 트래픽이므로 이전과 같이 기록하지 않습니다.
- 인증 서버가 보낸 `error_description` 등 자유 텍스트는 기록하거나 표시하지 않습니다.

## 검증

- 전체 Go race 테스트, Go vet, `gofmt` 검사, canonical API 계약 검사
- 실제 로컬 HTTP(S) 서버로 issuer 불일치·비JSON 응답·404·자체 서명 TLS·연결 거부 분류 검증, 구성한 오류로 DNS·시간 초과·`ECONNREFUSED` 래핑·기타·미분류 분기 검증과 응답 본문 누출 마커 검사
- Discovery 실패 반복 시 장애 행 1개·횟수 2 확인, 알려진 콜백 오류 코드의 행 분리·반복 접기·경고 심각도·모르는 코드 비노출 검증
- 콜백 라우트에서 조용한 시도 거절 0건 기록, `invalid_scope` 1건 기록, 인증 서버 자유 텍스트 미포함 단언
- 실제 실행 파일(`postra serve`, 연결 거부되는 Issuer)로 `/auth/oidc/start` → `302 /app/login?sso=error`, 로그 경고, `system_incidents` 의 `oidc` 행 1개(재시도 시 횟수 2) 확인
- 브라우저 자산 재빌드 후 포함된 `/app` 번들이 최신인지 검사

테스트는 로컬 모의 OIDC·메일·AI 서비스를 사용하며 운영 인증 서버 연결이나 운영 데이터 변경을 수행하지 않습니다.

## 업그레이드 및 호환성 주의사항

v0.22.0에서 추가 DB 마이그레이션, 새 설정 키, 새 필수 환경변수는 없습니다. DB·SecretStore·ObjectStore·KEK를 백업하고 유지한 상태에서 실행 파일 또는 컨테이너 이미지를 교체한 뒤 재시작하세요. 프런트엔드 소스 변경은 없으며 Go 단일 실행 파일에 React 자산·글꼴을 포함하므로 폐쇄망 런타임에 Node 서버나 외부 CDN이 필요하지 않습니다.

- 장애 화면에 `oidc` 구성 요소의 새 행(Discovery 실패, 인증 서버의 로그인 거절)이 나타날 수 있습니다. 기존 토큰 교환·검증 실패 행과 같은 화면에서 확인합니다. 관리자 안내는 [ADMIN_GUIDE 3.3 절](https://github.com/hkjang/postra/blob/v0.22.1/docs/ADMIN_GUIDE.md)을 참고하세요.
- 로그인 실패 화면의 Discovery 문구가 일반 안내에서 분류된 사유로 바뀝니다. 문구를 파싱하는 자동화가 있었다면 확인하세요.
- Keycloak OAuth MCP 연결, REST OIDC Bearer, API 키 동작은 v0.22.0과 같습니다. [v0.22.0 릴리즈 내용](v0.22.0.md)과 [Keycloak OAuth MCP 설정](https://github.com/hkjang/postra/blob/v0.22.1/docs/MCP_OAUTH.md)을 참고하세요.

## 배포 파일

- `postra-0.22.1-linux-amd64`: 정적 단일 실행 파일
- `postra-0.22.1.tar.gz`: 폐쇄망 `docker load` 이미지, `postra:0.22.1` (linux/amd64)
- `postra-0.22.1-sbom.cdx.json`: Go CycloneDX SBOM
- `postra-0.22.1-frontend-sbom.cdx.json`: 프런트엔드 CycloneDX SBOM
- `SHA256SUMS.txt`: 배포 파일 SHA-256
