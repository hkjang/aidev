# madi v0.3.0

릴리즈는 전체 회귀와 새 서비스 이미지의 재반입·폐쇄망 검증을 통과한 뒤 게시합니다. 실제 검사·게시 상태는 [검증 장부](https://github.com/hkjang/madi/blob/main/OPERATIONS_UPGRADE.md)와 [배포 검증 기록](https://github.com/hkjang/madi/blob/main/docs/deployment-verification.md)에서 확인하세요. 게시 파일의 정확한 크기와 SHA-256은 릴리즈 워크플로가 이 본문 하단에 기록하며, 이 게시 정책 자체가 검사 통과나 게시 완료를 뜻하지는 않습니다.

한국어 중심의 자체 호스팅 Knowledge & AI Workspace입니다. Go 서버가 React 화면을 함께 제공하며 외부 PostgreSQL과 영구 파일 볼륨을 사용합니다. 작성·검색·이관·AI 근거를 연결한 v0.2.0 이후, Keycloak 세션을 가진 사용자가 로그인 화면 없이 바로 들어오는 자동 로그인을 관리자가 선택해 켤 수 있게 한 업데이트입니다. 게시 워크플로의 빌드·검증·오프라인 실행 검사를 통과한 이미지만 첨부합니다.

## 이번 업데이트

- **자동 로그인 (silent SSO):** 관리자 SSO 설정의 **자동 로그인 (silent SSO)**(`oidc_auto_login`, 기본 `false`)을 켜면 Keycloak에 이미 로그인한 사용자는 `/app` 등 화면 경로를 열 때 로그인 화면을 거치지 않고 원래 열려던 경로로 바로 들어갑니다. `oidc_enabled`가 함께 켜져 있어야 동작하며, 꺼진 설치에서는 아무것도 달라지지 않습니다.
- **OIDC `prompt=none` 최상위 이동:** 숨은 iframe 대신 최상위 이동으로 `/api/v1/auth/oidc/start?prompt=none&return_to=<경로>`를 거쳐 기존 세션으로만 답하도록 요청합니다. 서드파티 쿠키가 막힌 브라우저에서도 동작하고 Keycloak의 프레임 허용 설정과 무관하며, Keycloak 클라이언트에 리디렉션 URI를 추가할 필요가 없습니다.
- **재시도 루프 방지:** 조용한 시도는 탭 세션당 한 번만 하며(`sessionStorage` 표시, 저장소를 읽을 수 없으면 이미 시도한 것으로 간주) 사용자가 직접 로그아웃한 직후, 주소에 `?sso=none`·`?sso=error`가 있는 경우, `/login`·OIDC 콜백·`/api`·`/mcp`·`/healthz`·`/readyz`·공개 공유 경로에서는 시도하지 않습니다. Keycloak이 `error=login_required`로 답하면 `/login?sso=none`으로 보내 평소 로그인 화면을 보여 줍니다.
- **서버 측 보호:** `oidc_auto_login`이 꺼져 있으면 `?prompt=none`이 붙은 요청도 평범한 로그인으로 처리하므로 주소만으로 흐름을 바꿀 수 없습니다. `return_to`는 `/`로 시작하고 `//`로 시작하지 않는 같은 출처 경로만 받으며 그 밖의 값은 `/app`으로 대체합니다. 서명·issuer·audience·nonce·state·PKCE·만료 검증과 기존 계정의 명시적 연결 보호는 그대로입니다.

관리자 설정 화면 SSO 탭과 [관리자 가이드](https://hkjang.github.io/madi/manuals/admin-guide.html)에 설정과 동작 조건을 기록했습니다. 상세 사용법과 한도는 [전체 매뉴얼](https://hkjang.github.io/madi/manuals.html), 검증 조건과 진행 기록은 저장소의 `OPERATIONS_UPGRADE.md`를 확인하세요. 자동 브라우저 시험은 실제 사람 사용성 관찰이나 운영체제 IME 시험을 대신하지 않습니다.

v0.3.0 후보의 로컬 검증은 웹 빌드(tsc·Vite), `go build`·`go vet`·비DB `go test ./...`, 임시 PostgreSQL 17 컨테이너에서 OIDC 코드·PKCE·nonce, 조용한 로그인(`TestPostgresOIDCSilentLogin`)·`return_to`, 설정·키·백업·지원 통합 시험 11개, Node 검사 27개(silent SSO 루프 방지 포함)와 매뉴얼 54쪽 재생성·검증 통과를 확인했습니다. 전체 Go `-race`·독립 브라우저 옵션·PG17/18 호환성, 새 이미지 저장·재반입·폐쇄망 실행은 태그 릴리즈 워크플로의 별도 관문이며 로컬 통과가 이를 대신하지 않습니다.

## 업그레이드 주의

기동 시 스키마를 자동 갱신하며, 이번 업데이트는 SSO 설정에 `oidc_auto_login` 항목을 추가합니다. 기존 저장 설정에 항목이 없으면 `false`로 취급하므로 업그레이드만으로 로그인 흐름이 바뀌지 않습니다. **교체 전에 PostgreSQL·모든 첨부 저장소·ENCRYPTION_KEY·기존 이미지를 같은 복구 계획으로 보관**하고, 복제한 환경에서 먼저 시험하세요. 변경된 DB에 예전 바이너리만 다시 올리는 방법을 롤백으로 사용하지 마세요. 복구에는 이전 버전과 그 버전의 백업을 함께 사용합니다.

자동 로그인을 켜기 전에 Keycloak 세션 수명과 조직의 공용 단말 정책을 확인하세요. 사용자가 madi에서 로그아웃해도 Keycloak 세션은 남으며, 같은 탭에서는 다시 로그인하기 전까지 조용한 시도를 억제하지만 새 탭에서는 한 번 다시 시도합니다. 운영 중인 Keycloak 서버를 이 변경으로 수정하거나 배포한 것은 아닙니다. `oidc_require_verified_email`을 비롯한 v0.2.0의 OIDC 동작과 백업·복원·첨부 추출·OCR 요구사항은 그대로입니다.

## 제공 범위

- 워크스페이스, Markdown 문서와 블록 ID·다중 선택·이동, 위키 링크·백링크·그래프, 문서 트리·검색·즐겨찾기·할 일
- 문서 버전·휴지통 보존기간 정리·첨부파일, 데이터베이스 표·보드·캘린더, Vault 폴더·첨부를 보존하는 Markdown ZIP 입출력
- 로컬 로그인·Keycloak OIDC와 선택형 자동 로그인(silent SSO), 역할과 문서 권한, 개인화 및 분리된 관리자 화면
- 워크스페이스·권한·만료·IP·호출 제한을 설정하는 API 키와 즉시 키 회전
- REST API·OpenAPI 목록·HTTP MCP, OpenAI 호환 스트리밍 AI 및 최대 262,144 토큰 설정
- 관리자 사용자·설정·이력·감사와 논리 백업·복원, 설정할 때만 사용하는 승인 절차
- 한국어 반응형 화면, 로고·파비콘, 로그인·프로필 메뉴의 서비스 버전
- 오프라인 서비스 이미지, Compose·Kubernetes 예제, 제품 소개와 사용자·관리자·운영 가이드
- Yjs 실시간 협업·커서·presence, 고급 중첩·동기화·임베드 블록·KaTeX·Mermaid·다단 편집
- 데이터베이스 6개 보기·관계·안전 수식·롤업·AI 속성, 팀 할 일·칸반·달력·토론·멘션·알림
- 문서 소유권·생명주기·보존·품질·지식 건강, 작업 큐·outbox·Webhook·자동화
- 동의 기반 임베딩·하이브리드 RAG·재정렬·현재 원문 인용, 자연어 검색·선택 문서 요약·개인 AI/검색 기록
- AI 그래프·엔터티·주제·중복·지식 보완 후보와 사용자 단건 확인, Workspace Agent의 도구 제한·쓰기 확인
- SAML·TLS LDAP·SCIM·그룹 매핑, 1/N단계·부서·병렬 승인, 읽기 전용 지원 진단
- 기업 문서 커넥터·외부 SQL 소스·Entity·읽기 전용 Text2SQL, 고정 원격 Git 동기화
- 민감정보 탐지·마스킹·분류·워터마크·보안 공개 공유·법적 보존
- Canvas·Drawing·격리 Plugin SDK, Tauri·PWA·오프라인 개인 보관함·Web Clipper
- 격리·권한·확인 기반 실행형 Runbook, 운영·오류·OpenTelemetry·기능 플래그·워크스페이스 브랜딩

## 배포

검증 완료 후 게시할 릴리즈 자산은 `madi:v0.3.0` 이미지를 저장한 `madi-v0.3.0.tar.gz` 하나입니다. 기본 플랫폼은 Linux amd64입니다. PostgreSQL은 별도로 준비하며 릴리즈에 제3자 서비스 이미지는 포함하지 않습니다. PDF/OCR 도구·영어/한국어 모델·로컬 웹 자산·런타임 구성 요소의 대응 소스와 고지를 같은 이미지에 넣도록 빌드하며, 최종 이미지에서 다시 검증합니다. 소스·체크섬·PostgreSQL 파일을 추가 자산으로 올리지 않습니다.

```sh
gzip -dc madi-v0.3.0.tar.gz | docker load
```

서비스 실행에는 `POSTGRES_DSN`, `BOOTSTRAP_ADMIN`, `BOOTSTRAP_ADMIN_PASSWORD`, `ENCRYPTION_KEY` 네 환경변수가 필요합니다. 그 밖의 운영 설정은 관리자 화면에서 관리합니다. ENCRYPTION_KEY는 32바이트 무작위 값의 Base64 표현이며 별도 안전한 장소에 백업해야 합니다.

## 범위와 제한

P0~P3 기능과 v0.2.0의 운영·UX 연결 위에 선택형 자동 로그인을 더한 업데이트입니다. 각 기능은 관리 정책·현재 권한을 적용하며 선택 외부 인프라는 별도 연결합니다. 프로토콜 시험을 실제 조직 자격증명 연결 시험으로 확대 해석하지 않습니다. Linux Tauri와 Windows·macOS 실행 검증, HTTP MCP와 stdio 전송을 구분합니다. Notion·Obsidian의 모든 독자적 기능을 무손실 변환한다고 보장하지 않습니다. [전체 기능 범위와 운영 한계](https://hkjang.github.io/madi/manuals/roadmap.html)를 확인하세요.

관리자 백업 ZIP은 같은 서비스 버전과 ENCRYPTION_KEY를 사용하는 환경에서 복원할 수 있습니다. 복원은 현재 데이터를 교체하고 세션을 폐기하므로 먼저 격리 환경에서 시험하세요. ZIP 256MB·압축 해제 1GB·10,000개 항목으로 제한하며 대규모 복구와 PITR에는 PostgreSQL·첨부파일의 같은 시점 백업을 사용합니다. 대규모 처리량과 지연 시간은 별도의 부하 시험 전에는 보장하지 않습니다.

릴리즈 워크플로는 웹·Go 빌드, Go 검사, 이미지 재반입 및 외부 통신이 차단된 Docker 네트워크에서 준비 상태·로그인 기동 검사를 통과한 뒤 이미지를 게시합니다. 실제 운영망의 TLS·DNS·프록시·IdP·AI 호환성은 운영 환경에서 추가로 확인하세요.

2026-09-14 v0.3.0 후보(`fc79295d` 기준)의 로컬 검사에서 `govulncheck`는 Go 호출 경로 0건·가져오는 패키지 0건을 보고했고, 의존하는 모듈에는 애플리케이션에서 호출되지 않은 취약점 1건이 남아 있습니다. `go vet`·서식 검사, 라이선스 423개 검증도 통과했습니다. 이는 새 배포 이미지의 검사를 면제하지 않습니다. 2026-09-08에 별도로 검사한 데스크톱 Rust SDK에는 GTK 계열의 정보형 경고 17건이 남아 있으며, 해당 SDK는 서비스 Docker 런타임에 포함되지 않습니다. 현재 확인된 입력 경계와 경고의 한계는 [보안 검증 안내](https://hkjang.github.io/madi/manuals/security-verification.html)에 명시합니다.
