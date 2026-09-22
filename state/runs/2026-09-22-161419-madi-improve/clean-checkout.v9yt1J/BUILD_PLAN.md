# madi 제품 개발 및 검증 기록

## 제품 원칙

Go + React + PostgreSQL 모듈형 단일 서버. Markdown 원문과 YAML front matter를 유지한다. 기본 언어 한국어. 서비스 실행에 필요한 환경변수는 POSTGRES_DSN, BOOTSTRAP_ADMIN, BOOTSTRAP_ADMIN_PASSWORD, ENCRYPTION_KEY 네 개뿐이다. 그 밖의 운영 설정과 비밀은 관리자 UI 및 DB에서 관리하고 비밀은 AES-GCM으로 암호화한다. PostgreSQL은 운영자가 제공하는 필수 외부 인프라다. 서비스 이미지에는 웹 자산, 폰트, 실행 파일을 포함하고 인터넷/CDN에 의존하지 않는다.

## 구현 순서

1. [완료] 저장소 초기화, API 계약, 인증/RBAC/DB 모델
2. [완료] 한국어 React 앱·전체 P0~P3 기능 화면, 블록·DB·탐색·AI·기업 연결 구현
3. [완료] OIDC, 범위 제한 API 키/회전, MCP, 스트리밍 AI
4. [완료] PostgreSQL 실제 통합 시험: 문서/권한/키/OIDC/AI/DB/백업 복원/ZIP 왕복/유지관리 및 전체 Go race 회귀
5. [완료] 전체 화면 브라우저 검증 및 데스크톱/모바일 캡처; 실제 새 DB의 25개 시험 묶음과 73경로 대표 캡처 검증
6. [완료] docs 홍보 페이지·30개 매뉴얼·253장 갤러리, 공개 Pages의 PC/390px·링크·선택·새로고침 검증
7. [완료] Docker 재빌드·이미지 저장/삭제/재반입·실제 내부망 실행/첨부/내보내기/백업복원, 커밋/푸시/첫 릴리즈/Pages 게시 및 첨부 다운로드 무결성 검증

## 릴리즈 원칙

사용자 확정(2026-09-08): **P0~P3 전체를 구현한 후 첫 릴리즈한다.** 전체 구현·검증을 마친 뒤 2026-09-08에 [v0.1.0 첫 릴리즈](https://github.com/hkjang/madi/releases/tag/v0.1.0)를 게시했다. 전체 범위와 실제 검증 근거는 [FULL_SCOPE.md](FULL_SCOPE.md)에서 추적한다. 미구현 항목을 로드맵으로 넘겨 첫 릴리즈를 완료 처리하지 않는다. 릴리즈 첨부 파일은 서비스 이미지 `madi:v버전`을 docker save 후 gzip한 `madi-v버전.tar.gz`만 제공한다. PostgreSQL 등 제3자 이미지는 번들하지 않는다. 압축 해제된 이미지는 소스 빌드나 패키지 설치 없이 실행된다.

최종 릴리즈 검증은 [GitHub Actions 34203932033](https://github.com/hkjang/madi/actions/runs/34203932033)에서 확인할 수 있다. 게시 파일은 `madi-v0.1.0.tar.gz` 하나이며, 다운로드한 25,699,214 bytes의 SHA-256 `f42154ddfb0c10054e34c14ad630f8f5c96663fd2393ee88458efc657348edef`를 GitHub digest 및 릴리즈 노트와 대조했다. 아래의 중간 검증·재검증 예정 문구는 당시 작업 이력이다.

## 작업 분담

- 코어: Go 서버, 스키마, 계정/세션/권한, 문서/버전/워크스페이스/DB/파일/운영
- UI: web/ React 앱 및 로고/파비콘
- 연동: internal/server/integrations*.go의 Keycloak OIDC, API 키, MCP, AI
- 배포/문서: Dockerfile, 배포 매니페스트, scripts/, .github/, docs/, 가이드

## 검증 발견 사항과 처리

- 실제 PostgreSQL 18에서 독립 스키마 통합 테스트와 Go race detector 사용.
- OIDC RSA 서명 mock provider로 PKCE/nonce/검증된 이메일/다른 브라우저 및 재사용 공격 차단 시험.
- AI 실제 HTTP mock 스트림으로 권한 없는 문서가 요청 본문에 들어가지 않음과 262144 max_tokens 확인.
- 키 회전/폐기/범위 변경/IP 제한/분당 호출량 제한 시험.
- 동시 문서 수정은 한 요청만 성공하고 다른 요청은409 반환.
- Markdown Vault 2001문서 내보내기, 폴더/첨부/원문/별칭/개인 문서 왕복, 실패롤백 확인.
- DB 속성 유형/옵션 변경 시 기존 행과 충돌하면 거부하며 삭제한 열의 값은 원자적으로 정리.
- 백업 ZIP 실제 복원, 네이티브 pg_dump/pg_restore 독립 대상 복원 검증.
- 시간별 휴지통 보존 정책과 세션/OIDC 상태 청소; 최대500문서씩, commit 후 해당 첨부만 정리.
- 1차 브라우저 시험:18 메뉴 새로고침·문서 원문 저장·모바일 탐색 성공. Field/Select 접근성 이름 및 comments 정렬 SQL 오류 발견·수정.
- 자동저장 중 SPA 이동·늦은 즐겨찾기/공유/속성/복제 응답의 교차 쓰기를 문서·사용자별 세대 검사와 중복 요청 잠금으로 차단. 실제 브라우저 비동기 회귀 8.4초 통과.
- Docker Desktop의 HTTP500 상태는 변경하지 않았다. 작업 전용 rootless Docker에서 비root·읽기전용 이미지, 외부 통신 없는 내부 네트워크, 네 환경변수, 영구 첨부와 백업 복원까지 실제 검증했다. 최종 이미지는 동일 절차로 다시 검증한다.

## 최종 통합 기준선 · 2026-09-08

- 최종 서비스 검증 빌드: `main-B2_oEw5_.js`. 문서 async, 기본 편집·이력·개인화 및 전체 25개 화면 시험을 로컬 새 DB와 원격 CI·릴리즈 환경에서 통과했다.
- 문서 목록은 원문 없는 `DocSummary`이며 상세 GET에서만 전체 Markdown을 반환한다. 2,000개 대량 태그/별칭 목록과 읽기 모델 JSON 합계32MiB 보호를 검증한다. 단건에도 같은 상한을 적용해 4MiB 원문의 JSON 이스케이프 확장까지 보존한다.
- 통합 검색은 평면 문서에 동등한 ACL 빠른 경로를 사용한다. 비공개/선택 공유/소유자/조상/제한 공간/순환/회원 철회/비활성 계정을 기존 재귀 함수와 비교하는 PostgreSQL 회귀 통과.
- 성능은 1천/1만 문서·동시10·각 API60회 실측으로 보고한다. 최초 타임아웃 실패와 최적화 결과를 함께 보관하며 대규모 목표를 달성했다고 추정하지 않는다.
- 전체 Go race 회귀와 실제 브라우저·최종 오프라인 Docker 결과를 합친 뒤에만 출시 상태를 변경한다.
- 정적 매뉴얼·전체 성공 화면 갤러리는 `node scripts/build-docs.mjs`, 실제 Project Pages 경로 검증은 `node tests/docs.mjs`로 재현한다.
- 프로덕션 의존성 419개의 라이선스/NOTICE 원문을 포함하고 `node scripts/licenses.mjs --check`로 파일·의존성 정합을 검사한다.

## 초기 확장 과정의 기록 (아래 후속 표기는 당시 상태)

이 절은 구현 순서의 역사적 기록이다. 현재 완료 여부는 위 최종 통합 기준선과 FULL_SCOPE.md의 기능 표를 따른다.

- 초기 35개 화면 회귀 테스트의 실행 오류를 모두 해결. 소스·블록 편집 전환과 개인별 초안 복구, 선택 옵션, 프로필/관리 메뉴와 모바일 검증 완료.
- Yjs CRDT를 실제 두 Chromium 사용자와 Go 서버/PG에서 검증. 동시 편집·커서·presence·재접속·계정 비활성화·원문 epoch 충돌이 동작하고 권한 회수 이후 새 내용을 받지 못함을 확인.
- Formula/Relation/Rollup/AI property와 6개 Database 보기 통합 시험 통과. 실제 SSE 저장, 잘못된 관계/타 공간 접근 차단, 원문 동시 수정 보호, native 선택 UI·모바일 검증 완료.
- PostgreSQL outbox/job queue, 서명 Webhook, lease/재시도/취소, 자동화 효과의 중복 방지 및 현재 사용자/키 권한 검사 통합 시험 통과.
- Space/문서 조상 ACL을 검색·AI·DB·첨부·CRDT·Vault export에 통합. 조직 멤버와 workspace 접근은 별도로 유지. 구조 변경은 트랜잭션 잠금으로 순환·최대20단계 초과를 거부.
- Workspace AI 설정은 별도 암호화·버전 관리. 브랜딩/설정 이력 복원, 공간/조직 생성·권한 선택·URL 새로고침과 모바일 화면 6장 재검증 완료. HTML pattern의 최신 브라우저 정규식 오류를 찾아 수정했고 콘솔 오류 0 확인.
- 서버 Markdown 링크/할 일/목차 추출은 CommonMark/GFM AST로 처리하여 YAML·코드 예제를 실제 링크나 작업으로 오인하지 않음.
- 논리 백업 테이블 목록을 단일 카탈로그로 통합하고 PG schema 대비 누락 테스트 추가. 복원 시 인증 세션·presence 무효화, 큐 작업 취소·자동화 중지로 외부 작업 재실행을 방지.
- 고급 블록·Canvas/Drawing·Local/S3/MinIO·예약 백업까지 실제 PG 및 브라우저 검증 완료. 초기 웹 자산은 편집기·AI·Canvas를 지연 로딩하여 main 1.37MB→510kB(압축159kB)로 분리.
- 지식 품질·문서 정책·담당자·보존·검토 주기·stale/archive와 opt-in 생명주기 자동 점검, 변경 이력·승인 우회 거부를 구현하고 PG/race·PC/모바일 검증 완료.
- 토론의 답글/해결/담당자/반응/인용/블록/사용자·팀 멘션, 팀 관리·현재 ACL 적용 알림을 구현하고 실제 화면 저장/선택/새로고침·모바일 검증 완료. SMTP 및 외부 알림은 후속 구현.
- Plugin SDK·LDAP/SAML/SCIM·HTTP 커넥터 및 개인 수집함 PC/모바일 검증 통과. 수집 요청/첨부의 멱등 영수증을 DB 트랜잭션으로 기록하고, 동시 재전송 중 연결 풀 고갈 회귀를 수정하여 6개 동시 요청도 단일 문서·이벤트로 처리.
- 진행 중: 선택적 다단계 승인, PWA/Desktop/Web Clipper, 외부 SQL 데이터, 할 일 고도화. 상세 미완료 범위는 FULL_SCOPE.md를 기준으로 추적하며 **전체 P0~P3 완료 이전에는 첫 릴리즈하지 않음**.

## API 계약 (구현 시 이 계약을 유지)

모든 JSON은 snake_case. 성공 응답은 객체 또는 배열 자체. 오류는 {error:string, request_id?:string}. 쿠키 세션(동일 출처), API Key는 Authorization: Bearer. 변경 요청은 X-Madi-Request: 1 헤더; 브라우저 Origin도 검증. /api/v1 접두사. ID는 UUID 문자열. 역할: admin, editor, viewer; 워크스페이스: owner, admin, editor, commenter, viewer. 키 scope: document:read, document:write, database:read, database:write, search:read, ai:execute. 키는 사용자 권한과 교집합.

- GET /api/v1/public → {name,version,oidc_enabled,signup_enabled,approval_enabled}
- POST /api/v1/auth/login {email,password} → User; POST auth/logout; GET auth/me → User
- GET/PUT /api/v1/profile → User; PUT {name,preferences:{theme,font_size,sidebar_width,last_path,...},password?,current_password?}
- User: {id,email,name,role,kind,preferences,created_at,disabled}
- GET/POST /workspaces → Workspace[] / Workspace {id,name,slug,role}; POST {name}
- GET/PUT /workspaces/{id}/members → Member[] / Member; PUT {email,role}; DELETE /workspaces/{id}/members/{userId}
- GET/POST /documents?workspace_id=&q=&tag=&trash=1&favorite=1 → DocSummary[] / Document. DocSummary는 원문/블록 메타데이터를 제외하고 160자 excerpt·제한된 태그/별칭·잘림 표시를 제공한다.
- Document: {id,workspace_id,parent_id:null|string,title,markdown,tags:[],aliases:[],icon,status,visibility,owner_id,version,created_at,updated_at,deleted_at:null|string,is_favorite}
- POST /documents {workspace_id,title,markdown?,tags?,parent_id?,icon?,visibility?}; GET/PUT/DELETE /documents/{id}; PUT document fields + version (optimistic lock returns 409)
- POST /documents/{id}/favorite; POST /documents/{id}/restore; GET /documents/{id}/versions; POST /documents/{id}/versions/{version}/restore
- GET /documents/{id}/backlinks → Document[]; GET /documents/{id}/comments → Comment[]; POST same {body}; Comment {id,body,user_name,created_at}
- POST /documents/{id}/approval {action:submit|approve|reject,comment?}; only enabled when admin settings approval_enabled
- GET /graph?workspace_id= → {nodes:[{id,title,tags,icon}],edges:[{source,target}]}
- GET /tasks?workspace_id= → [{document_id,title,text,done,line}]; PUT /tasks {document_id,line,done,version}
- GET/POST /databases?workspace_id= → Database[] / Database; Database {id,workspace_id,name,properties:[],created_at}; property {id,name,type,options:[]}
- GET/PUT/DELETE /databases/{id}; GET/POST /databases/{id}/rows → Row[] / Row; Row {id,database_id,values:{},created_at}; PUT/DELETE /databases/{id}/rows/{rowId}
- POST /attachments?document_id= multipart file → {id,name,url,size}; GET /attachments/{id}
- GET /export?workspace_id= → application/zip Markdown vault; POST /import?workspace_id= multipart file (.md/.zip) → {imported,errors:[]}
- GET /admin/stats → {users,workspaces,documents,databases,attachments_bytes,ai_calls,recent_activity:[],health:{...}}
- GET/PUT /admin/settings → Settings (flat object; secrets blank with *_configured flags); PUT partial fields. Settings: site_name,site_url,signup_enabled,approval_enabled,reviewer_role,oidc_enabled,oidc_issuer,oidc_client_id,oidc_client_secret,oidc_auto_register,ai_enabled,ai_base_url,ai_api_key,ai_model,ai_max_tokens (1..262144),ai_system_prompt,session_hours,trash_retention_days,default_key_days,allowed_key_scopes,storage_path
- GET/POST /admin/users → User[] / User; POST {email,name,password,role,kind?}; PUT /admin/users/{id} {role,disabled,name}; kind user|service
- GET /admin/audit → [{id,action,user_id,user_name,resource,ip,created_at,details}]
- GET /admin/settings/history; POST /admin/settings/history/{id}/restore
- GET /admin/backup → application/zip logical backup including attachments and encrypted settings (encryption key excluded)
- GET/POST /keys → Key[] / {key,token}; POST {name,scopes:[],workspace_id,expires_in_days,ip_allowlist:[],rate_limit}; Key {id,name,prefix,scopes,workspace_id,expires_at,last_used_at,created_at,revoked_at,ip_allowlist,rate_limit}
- PUT/DELETE /keys/{id} → Key / {ok:true}; POST /keys/{id}/rotate → {key,token} atomically revokes old secret
- GET /auth/oidc/start; GET /auth/oidc/callback (outside API prefix aliases allowed)
- POST /ai/chat {prompt,document_id?,workspace_id?} → SSE data: {text} / {sources:[{id,title}]} / {error}; ends data: [DONE]
- POST /mcp → JSON-RPC initialize, notifications/initialized, tools/list, tools/call; protected by bearer token scopes
- GET /healthz, /readyz, /metrics; static SPA /app/* /admin/* /login, bundled assets.

## Go 연동 코드 계약

package server, Server {DB *pgxpool.Pool, EncryptionKey []byte, Version string, mux *http.ServeMux}. Root defines Principal {ID,Email,Name,Role,Kind string; Scopes []string; WorkspaceID string; TokenID string}, current(r)*Principal, (s *Server) auth(next http.Handler) http.Handler (delegates bearer token to s.tokenPrincipal), (s *Server) settings(ctx) (map[string]any,error), (s *Server) encrypt/decrypt(string)(string,error), jsonResponse(w,status,v), apiError(w,status,message), decode(r,v) error, newID() string, (s *Server) audit(r,action,resource string,details any), (s *Server) canWorkspace(ctx,p,id,write bool) bool, (s *Server) canDocument(ctx,p,id,write bool) bool, (s *Server) createSession(w,r,userID) error. Integration agent implements registerIntegrations(), tokenPrincipal(r)(*Principal,error). Routes registered on s.mux. Register authenticated handlers with s.handle("METHOD /api/v1/...",fn), public routes with s.mux.HandleFunc(). s.handle wraps auth; s.admin wraps auth + service admin check. Root DB tables documented in internal/server/schema.sql.
