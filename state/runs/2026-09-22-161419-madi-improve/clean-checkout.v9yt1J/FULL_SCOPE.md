# 첫 릴리즈 전체 범위

2026-09-08 사용자가 **P0~P3 전체 구현 후 첫 릴리즈**를 명시했다. 전체 기능과 통합 검증을 마친 뒤 같은 날 17:44 KST에 `v0.1.0` 첫 릴리즈를 게시했다. 외부 IdP·AI·기업 시스템은 설정 가능한 실제 프로토콜 어댑터와 계약 시험을 제공하고, 운영 연결 값은 관리자 UI에서 입력한다. 테스트 mock은 검증용이며 서비스에서 성공 응답을 가장하는 대체 구현으로 사용하지 않는다.

| 단계 | 기능 묶음 | 수락 기준 | 상태 |
| --- | --- | --- | --- |
| P0 | 실행·배포 | Go+React+PG, 네 환경변수, 자동 migration, offline 자산, 비root 서비스 이미지·K8s | 검증 완료 |
| P0 | 계정·OIDC·RBAC | 로컬, Keycloak discovery/PKCE/nonce, 워크스페이스/문서 ACL, 서비스 계정 | 검증 완료 |
| P0 | 기본 문서 | Markdown/GFM/front matter/별칭/블록/코드/표/태그/이미지/첨부 | 검증 완료 |
| P0 | 블록 조작 | ID 보존, 드래그·다중 선택·이동·중첩·삭제 | 검증 완료 |
| P0 | 탐색 | 트리·접기·즐겨찾기·최근·개인/공유·휴지통·bread crumbs·드래그 이동·폭/상태 보존 | 검증 완료 |
| P0 | 명령·편집 모드 | 검색·빠른 열기·생성·복제·이동·내보내기·일일노트·AI·링크, 단축키 개인화·읽기/집중/프레젠테이션/인쇄 | 검증 완료 |
| P0 | 그래프·링크 | 전역/로컬·깊이1~5·태그/폴더/사용자 필터·검색·관계 유형·깨진 링크/고립 문서 | 검증 완료 |
| P0 | 입출력 | Markdown/Obsidian/Notion/HTML/CSV/JSON, 폴더·첨부·링크, 미리보기·결과·비동기 job | 검증 완료 |
| P0 | 문서 이력 | 자동 snapshot·diff·복원·Git-friendly export | 검증 완료 |
| P0 | 기본 DB | table·filter·sort·columns·select·multiselect·타입 검사·동시변경 안전성 | 검증 완료 |
| P0 | 관리자·개인화 | 모든 운영 설정 DB/UI, 한국어·폰트·테마·서비스버전·스크롤·새로고침 | 검증 완료 |
| P0 | API·MCP·키 | REST/OpenAPI·HTTP MCP 도구, 범위/IP/유효기간/호출 제한/회전/감사 | 검증 완료 |
| P0 | 백업·운영 | ZIP/pg_dump 복원·보존 정책·Health/Readiness/Metrics·로그·RequestID·Graceful shutdown | 검증 완료 |
| P1 | 실제 협업 | Yjs durable CRDT·동시 편집·커서·presence·재접속·문서원문 정합성 | 검증 완료 |
| P1 | 고급 블록 | block link/embed/synced·2/3단·toggle·고급 표·각주·KaTeX·Mermaid·bookmark/embed | 검증 완료 |
| P1 | 고급 DB | relation·board/calendar/list/gallery/timeline·sandbox formula·rollup·AI property·버튼·progress·자동 작성자/시각 | 검증 완료 |
| P1 | 할 일·업무 | 문서 Todo의 담당자/기한·내/팀/기한초과/오늘/이번주·Kanban·통합Calendar·회의/ADR 템플릿 | 검증 완료 |
| P1 | 댓글·멘션·알림 | 문서/블록/인라인·thread/resolve/assignment/reaction, @사용자/팀/문서/날짜, inbox·email·webhook채널 | 검증 완료 |
| P1 | 지식 생명주기 | owner/reviewer/maintainer·검토주기·만료·stale/archive·소유자 없는 문서 | 검증 완료 |
| P1 | 통합 검색 | 문서/블록/파일/댓글/사용자/태그/작업/AI대화·필터·제목/태그/신선도/인기도 가중치 | 검증 완료 |
| P1 | RAG·AI | chunk·embedding·pgvector 옵션·hybrid/rerank·ACL·정확한 citation·8종 이상 문서지원·자연어검색 | 검증 완료 |
| P1 | 지식 품질·분석 | stale/orphan/duplicate/missing owner/broken/no tags·quality score·reader/writer/search gap 지표 | 검증 완료 |
| P1 | 자동화·Webhook | PG queue/outbox·조건/액션·재시도/timeout/event filter/history·AI 자동화 | 검증 완료 |
| P1 | 스토리지·예약작업 | Local/S3/MinIO·storage test·예약 백업/보존·PG job queue 취소/재시도/상태 | 검증 완료 |
| P2 | 조직·공간 | 조직/멀티워크스페이스/Space/Page 권한·개인공간·workspace별 AI/storage/template/audit/token/branding | 검증 완료 |
| P2 | 수집 | Quick capture/inbox·텍스트/URL/이미지/파일/API/Webhook/email·Web Clipper 5가지 모드 | 검증 완료 |
| P2 | 그래프 AI·Entity | 엔티티/주제/관계/유사문서 추출·추천→사용자 승인·종류별 Entity 페이지 | 검증 완료 |
| P2 | 승인 | 선택적 1/N단계·부서/병렬 승인·반려/재요청·미설정시 완전 제외 | 검증 완료 |
| P2 | 커넥터 | 공통framework·Jira/GitLab/Confluence/Drive/SharePoint/GitHub/REST ingestion·상태/매핑 | 검증 완료 |
| P2 | 외부 데이터 | PG/Oracle/MySQL/MariaDB/MSSQL metadata·source page·관련SQL/문서·읽기전용 접근 | 검증 완료 |
| P2 | 정보보호 | 민감정보 탐지 warn/block/mask/audit·등급·워터마크·만료/비밀번호/IP/다운로드 제한 공개링크·legal hold | 검증 완료 |
| P2 | Canvas·Drawing | 문서/이미지/URL/DB/AI/다이어그램 노드·배치/연결/저장·도형/펜 편집 | 검증 완료 |
| P2 | Plugin SDK | block/command/sidebar/menu/import/export/AI provider 등록·권한·격리·설치/활성 | 검증 완료 |
| P2 | Desktop·PWA | Tauri·shortcut/tray/deeplink/cache·responsive/PWA·capture/camera/simple edit | 검증 완료 |
| P2 | 기업 계정 | LDAP/SAML/SCIM·그룹매핑·감사된읽기전용 impersonation | 검증 완료 |
| P2 | 운영 확장 | feature flags·workspace branding·설정 version/rollback·관리자 health/errors·OpenTelemetry | 검증 완료 |
| P2 | Git 연동 | Markdown export·원격 Git sync/변경 커밋·충돌 보고 | 검증 완료 |
| P3 | 실행형 Runbook | 목적/조건/단계/검증/롤백/owner/last tested·실행권한/승인/격리·Shell/Ansible/AWX | 검증 완료 |
| P3 | Workspace Agent | knowledge/tool 범위 설정·사용자 ACL 상속·스트리밍 실행·작업 이력·도구 호출 통제 | 검증 완료 |
| P3 | 기업 데이터 지식 | 통합데이터소스 탐색·문서/SQL/Entity 연결·권한 있는 Text2SQL 미리보기/승인/읽기전용 조회 | 검증 완료 |
| 완료 | 종합 검증·문서·출시 | 전체 메뉴/버튼 E2E·모바일·권한·오프라인 Docker 검증·전체 캡처·가이드·SEO/AEO Pages·commit/push/release | 검증 완료 |

## 대규모 운영 목표

10,000 사용자/1,000 workspace/10,000,000문서/100,000,000블록/2,000동접은 구조·부하 시험 대상이다. 실제 자원으로 재현할 수 있는 벤치마크를 제공하고 측정 없이 목표치를 달성했다고 쓰지 않는다. 외부 커넥터의 실제 조직별 연결은 운영자 자격증명이 필요하므로 어댑터 계약시험과 연결 진단 UI를 구분한다.

## 최종 통합 및 게시

- Go·PostgreSQL·브라우저·실제 오프라인 이미지 검증과 GitHub 첫 릴리즈·Pages 게시를 완료했다. 릴리즈 태그 `v0.1.0`은 검증된 커밋 `842de2b46b40f5c0180371188038db3f568d3331`을 가리킨다.
- 30개 매뉴얼, 정적 35페이지, 갤러리 화면 253장과 전체 73경로의 대표 캡처를 검증했다.
- [첫 릴리즈](https://github.com/hkjang/madi/releases/tag/v0.1.0) · [전체 CI](https://github.com/hkjang/madi/actions/runs/34201456415) · [릴리즈 검증](https://github.com/hkjang/madi/actions/runs/34203932033) · [홍보 페이지·가이드](https://hkjang.github.io/madi/)
- 실제 게시 파일은 `madi-v0.1.0.tar.gz` 하나이며 25,699,214 bytes, SHA-256 `f42154ddfb0c10054e34c14ad630f8f5c96663fd2393ee88458efc657348edef`이다. 독립 다운로드 후 GitHub digest·릴리즈 노트·gzip 무결성·단일 `madi:v0.1.0` 이미지·Linux amd64·UID10001을 대조했다. 로컬 후보 아카이브의 이전 해시와 구분한다.
- 아래 기록은 최신순 작업 이력이다. 이전 항목의 ‘재검증 예정’ 및 ‘아직 게시하지 않음’은 당시 상태이며 최신 완료 기록으로 대체된다.

## 최근 검증

- 최종 릴리즈 워크플로 `34203932033` 전체 SUCCESS: 실제 PostgreSQL·Go race server 909.500초, 새 DB의 사용자·관리자 25개 화면 묶음, 이미지 저장·삭제·재반입·외부 차단망 4환경변수·비root·읽기 전용·CRUD·첨부·내보내기·백업 복원 PASS. 정식 릴리즈의 첨부 1개와 독립 다운로드 무결성을 확인했다. 공개 Pages의 PC·390px·주요 링크·갤러리 동작도 PASS다.
- 첫 GitHub CI `34199420110`에서 시험 환경 차이 3건을 확인했다. 임시 SQL 계정에 실제 SCRAM 비밀번호를 부여하고, 공통 통합 시험의 첨부 경로를 전용 임시 디렉터리로 격리했으며, 협업 본문 검사에서 커서 이름 위젯을 분리했다. 제품 코드·운영 인증 정책은 변경하지 않았다. 실제 SCRAM 3회, 협업 2브라우저 3회, 첨부 원본·경로 검증 및 문서/설정/백업/입출력 race 회귀 24.540초 PASS 후 원격 CI를 다시 실행한다. 첫 릴리즈는 아직 게시하지 않았다.
- 최종 후보 `main-B2_oEw5_.js`: 공유 브라우저 25개 suite PASS, 최종 변경된 가져오기·작업 이력·백업 보관 파일 표는 최신 빌드에서 390px 스크롤·44px 버튼·한국어 파일 입력을 추가 검증했다.
- Go1.26.8 전체 PG/race 315 PASS·옵트인 12 skip, server 536.440초. 최종 compress1.18.7·32MiB JSON 경계는 저장소/백업/원문 회귀 27.858초 PASS. 실제 4MiB 원문의 JSON 이스케이프 확장도 보존한다. 호출 경로 및 가져오는 패키지의 Go 취약점은 0건이다.
- 최종 `madi:v0.1.0` 후보는 이미지 삭제·재반입 후 외부 차단망에서 네 환경변수·UID10001·읽기 전용·2CPU/4GiB로 CRUD·첨부·내보내기·백업 복원을 모두 통과했다. 후보 아카이브와 CI 게시 파일의 무결성 값은 구분하여 기록한다.
- 최종 정적 검증: 35페이지·내부 링크 328개·390px·구조화 데이터·갤러리 필터 새로고침·외부 자산 0 PASS. 72개 앱 경로와 로그인, 총 73개 매핑의 PNG 크기·해시 PASS.
- 최종 Go1.26.8·기본 pool20·1만 문서·동시10 부하 360요청 오류0, 검색 P95 411.052ms. 단일 호스트의 측정 조건이며 최대 규모 성능을 보장하는 수치가 아니다.
- Desktop SDK Cargo 570개 검사: 취약점 0건, glib 안전성 1건과 유지보수 중단 16건의 정보형 경고가 있다. 해당 SDK는 서비스 이미지에 포함되지 않으며 호출 조사와 검증 한계는 보안 가이드에 공개한다.
- 2026-09-08 15:57 Go1.26.8 전체 PG/race server 536.440초 PASS. 후속 compress1.18.7과 원문 JSON확장32MiB 경계는 추가저장소/원문회귀로검증한다. govulncheck 최종 호출경로0/가져오는패키지0, 미사용 x/crypto/openpgp 모듈경고1은 실제미포함 의존성경로로구분했다.
- 오프라인후보: 단일 `madi:v0.1.0` 이미지 tar.gz→이미지삭제→재반입, 네 환경변수·UID10001·읽기전용·내부Docker망·2CPU/4GiB에서Bootstrap/CRUD/첨부bytes/작업내보내기/논리복원/세션및외부작업정지 PASS. 최종 모바일표 CSS 반영 후 같은검증을재수행한다.
- 실제 App72경로+로그인=73 대표화면 매핑 및 PNG형식·크기·해시 검증 PASS. 실패진단은 공개폴더에서분리하고 원본보존했다. 정적35페이지·내부링크323개·390px·구조화데이터·외부자산0 PASS.
- 2026-09-08 15:43 전체 Go/실제 PostgreSQL race 회귀 server 527.599초 PASS, 공유 최신 화면 25 suites 381.8초 모두 PASS. 이후 보안 의존성 업데이트는 별도 최종 재검증 대상이다.
- 출시 전 취약점 검사에서 Go1.26.0/pgx/xcrypto/XML서명 의존성 경고를 발견하여 Go1.26.8·pgx5.9.2·xcrypto0.56.0·goxmldsig1.6.0으로 갱신했다. 경고가 해소되고 재회귀를 통과하기 전에는 출시하지 않는다. 웹·Desktop 프로덕션 npm audit은 경고0이다.
- 최신 공유 빌드 `main-DYFyyc18.js`: 문서별 async 세대/중복 요청/미저장 원문 보호 8.4초, 기본 편집·이력·개인화 13.4초 실제 브라우저 PASS. 나머지 전체 화면 순차 회귀 중.
- 최종 API/MCP 검색 scope·raw query 감사 제외·문서 요약 목록·대량 태그 방어·Agent/지원 그래프 제한 PG/race PASS. 검색 빠른 ACL 경로는 원 재귀 권한 함수와 차분 비교 PASS(소유/개인/선택 공유·조상/제한 공간·순환/철회/비활성 포함).
- 1만 문서·동시10 실측에서 첫 검색 타임아웃 5%를 수정했다. 최종 페이지 크기만 원문 조각/태그 JSON으로 투영한 측정은 기본 pool20, 검색 P95 868.908ms, 360요청 오류0. 단일 공유 호스트·평면 ACL·네이티브 문서 FTS 조건이며 전체 용량 목표의 보장은 아니다.
- 2026-09-08 15:01 전체 PostgreSQL 포함 `go test -race ./...` PASS, server 469.512초. 이후 최종 API/MCP 목록/scope 보강은 별도 및 마지막 전체 재검증 대상이다.
- 개인 검색 이력: 기본 비활성·별도 저장/AI전송 동의·사용자 격리·일별중복·공용감사 원문제외·보존축소/확대·TTL실삭제·PII mask/block·키차단·quiet SSE 동의회수 PG/race 9.083초 PASS. 실제 개인비공개초안·옵트아웃·삭제·390px·외부자산0 브라우저 PASS. 자연어/선택문서요약도 최종16px 빌드 재검증 PASS.
- 그래프 AI: 고정 공급자/원문 버전·hash·현재ACL·정확인용·다중private자료의공개주제전파차단·단건승인·동시6요청단일효과·개인최고등급문서·PII·취소/복원 PG/race22.596초, 실제브라우저11.521초 및 실제연결취소/개인이력삭제9.332초 PASS.
- 입출력: Markdown 원문/이동용·HTML·JSON Vault·DB CSV 5형식 실제job/download·Notion HTML/CSV혼합·폴더/첨부왕복·암호화임시보관·currentsource409·TTL·복원취소 PG/race 및 PC/390px실제브라우저 PASS. 8장 최종캡처 완료.
- 운영/개인화: 16px공용글자·관리자409초안유지·실제OTLP·3단계false상한·브랜딩이미지정본·운영390pxoverflow0 PASS. 문서이력metadata/diff/동시복원409·폰트/날짜/폭/코드/맞춤법·워크스페이스감사현재ACL PG/race 및 foundation브라우저 PASS.
- 홍보·매뉴얼: 현재 상세HTML26개 포함31페이지, 성공화면209개 갤러리, 구조화데이터·정적내부링크271개·갤러리옵션/검색/리프레시·390px·외부자산0 실제브라우저 PASS. 추가최종화면은계속반영한다.

- 자연어 검색/선택 문서 요약: 명시 전송 동의·닫힌 검색 조건 JSON·SQL/임의 ID 거부·조건 확인 후 검색·수동 공간/소유자 필터 유지·URL 새로고침·선택한 최대10개 문서의 현재 원문 조각만 AI 전달·선택 외 문서 미전송·390px/외부자산0 실제 브라우저 PASS. quiet SSE 취소·stale 원문/검색·개인 AI 기록 및 인기도 중복열람 제한 PG/race 16.181초 PASS. 가독성 후속 조정과 개인 검색 이력/실패 분석은 추가 검증 중.
- 지원 진단: 관리자∩대상 현재 ACL·원래 로그인 Principal 불변·개인문서 양방향 미노출·원래 쿠키 결합·읽기전용·DB 관계/수식/롤업·500ms 화면 ACL heartbeat·원문제거·설정/활성/이탈/로그아웃 철회 PG/race 20.066초 및 PC/390px 브라우저 5.985초 PASS.
- Git: 실제 TLS/SSH smart-Git bare fetch/CAS push·고정원격/경로/버전·원문/첨부 manifest·암호화미리보기·명시동의·PG job·충돌/전송불명확 상태·리프레시·390px 폭·브라우저500/JS오류0 PASS, 최종6장 캡처 완료.
- 운영: 실제 OTel SDK/OTLP protobuf·TLS/redirect거부·본문/개인ID/query/secret 미포함·설정변경중단·OTEL 환경변수 영향차단·3단계 기능 false 상한·원본보존·PNG 재인코딩/현재권한·설정CAS PG/race 17.604초, PC/mobile 기능 흐름 PASS. 파일선택 숨김 input의 가로폭 후속 수정은 다음 빌드에서 재검증한다.

- 기준 브라우저 35장: 문서 원문 저장·새로고침·선택/다중선택·테마/글꼴·전체 초기 메뉴·모바일 오류 0.
- CRDT: 실제 TipTap/Yjs JS fixture와 Go projection, 동시변경·재접속·DB 영속·epoch 충돌·Origin/ACL/활성세션 검사. 실제 두 Chromium 사용자 테스트 통과.
- 고급 DB: 안전 수식·순환/자원 제한·관계/롤업·타 워크스페이스/제한 공간 ACL·AI SSE 동시 수정 방지·6개 뷰와 URL 상태·모바일 통과.
- 작업/자동화: 트랜잭션 롤백·SKIP LOCKED·lease/cancel/panic·HMAC 전송·재시도·중복효과 방지·계정/키 폐기·승인 우회 거부·실제 SSE와 관리자 UI 통과.
- 공간: 제한된 공간·상위 개인문서의 ACL 상속을 문서/DB/검색/AI/CRDT/export에 적용. workspace AI 설정 암호화·버전 충돌/복원, 조직 멤버와 문서 접근 분리 PG 테스트 통과.
- 논리 백업: 새 영속 테이블을 공통 카탈로그에 추가하고 누락 탐지 테스트 적용. 복원 시 세션/presence 무효화·과거 작업 취소·자동화 중지·시퀀스 복구 검증.
- 지식 운영: 소유권 이전 ACL, 정책 변경 이력, 검토 주기, 보존 기한/법적 보존 삭제 차단, 현재 권한 내 규칙 기반 품질·30일 지표 PG/race 테스트 및 PC/모바일 화면 6장 통과. 분석은 최근 2,000개·본문 합계16MiB로 명시 제한.
- 생명주기: 기본 비활성, 관리자가 켠 워크스페이스에 한해 매시간 게시 문서 만료→stale·담당자 알림·버전/outbox. 중복 처리·비활성 소유자·승인 우회 거부 PG/race, 설정/보관 상태 선택·저장·새로고침 UI 통과.
- 토론: 페이지/블록/인용 댓글, 5단계 답글·해결/재열기·담당자·멱등 반응·사용자/팀 멘션·팀 관리·커서 페이지 조회 구현. PG/race 및 데스크톱/모바일 5장 UI 통과. 문서 권한 회수 시 기존 알림에서도 해당 정보 제외. 외부 알림 채널은 별도 후속 구현.
- 할 일 회귀: CommonMark TextBlock이 체크 마커를 원문에 포함하는 문제를 수정. 완료 이벤트가 같은 트랜잭션에 기록되는 실제 PG/race 회귀 통과.
- Plugin: 오프라인 ZIP 설치·워크스페이스 권한 승인·Worker 격리·현재 ACL/scopes·비동기 작업 제약 상속·명시적 블록 실행·명령/사이드바/메뉴/입출력/AI 도구. PG/race·공격 시험 및 실제 서비스 PC/모바일 5장 통과.
- 기업 인증: 실제 TLS LDAP, 서명 SAML fixture, SCIM 서비스 전용 키·워크스페이스 격리·그룹 매핑·비활성화·키 회전 검증 및 관리자/로그인 모바일 UI 통과. 감사된 지원 impersonation은 후속 구현.
- 수집함: 기본 비공개 텍스트/URL/태그/파일·사진, 명시적 분류/공유 변경, 문서 첨부 목록. URL 저장은 외부 접속 없음. 6개 동시 요청의 단일 문서/outbox 보장·첨부파일 재전송 멱등성·키 scope·ACL·문서 버전 검증 PG/race, PC/모바일 4장 통과. 메일/Webhook 수집·Web Clipper는 후속 구현.
- 커넥터: 7개 공급자 HTTP 계약·429/페이지·멱등반영·로컬 수정 충돌·SSRF·현재 실행계정 ACL PG/race 통과. 실제 서비스 REST 미리보기/수집/편집/새로고침/모바일 검증 완료. SQL 소스 UI/타 DB 드라이버 검증 진행 중.
- 할 일·일정: Markdown 고유 참조·동일 줄의 HTML 표 할 일 원문 위치·담당자/기한/상태·동시변경/중복분리·완료 outbox PG/race 통과. 실제 PC/모바일 목록·칸반 손잡이 드래그·통합 날짜·회의/마일스톤·URL 유지 7장 검증. 모바일 7열 요약 달력. DB 일정에는 별도 database:read 키 scope 필요. 회의록·의사결정 기본 템플릿 제공; 작업 메타데이터는 서비스 백업에 보존하고 단순 Markdown 내보내기와 구분.
- AI 스트림: 참조 버전을 출처와 함께 반환하고, 모델 전송 전·텍스트 반환 전 현재 세션/키/스코프/ACL·문서 버전·공급자 설정을 재검사. 처리 도중 문서 비공개 전환/본문 변경/키 권한 회수/로그아웃 후 신규 텍스트 차단·retract PG/race 18.527초 통과. 원문 조각 및 TLS 임베딩 벡터 프로토콜은 구현 중인 RAG 기반이며 전체 하이브리드 검색은 아직 미완료.
- 통합 검색 기반: 문서 변경 트랜잭션의 로컬 색인 큐·원문 byte/행·현재 버전 조각·동시 작업/롤백 PG 검증. 문서/블록/코드/할 일/파일 이름/댓글/태그/DB/행/사용자 10종, 필터/정렬/페이지/와일드카드 입력·권한 회수·키 scope 분리 PG/race 4.335초 통과. AI 대화·벡터 검색·인기도 가중치는 후속 구현.
- 기기: 실제 Chromium PWA의 정적 오프라인 캐시/API 제외, AES-GCM 개인 보관함 잠금·충돌 동기화·모바일 검증. Clipper Chrome 실제 권한·5모드·회전·연결 해제 검증 및 Firefox 패키지 lint 통과(실제 Firefox 실행은 별도). Linux Tauri를 실제 GTK/WebKit 환경에서 실행해 파일 선택·UTF-8 입출력·전역 단축키·시작 딥 링크 확인/취소·원격 webview IPC 차단·오프라인 잠금을 검증. Windows/macOS 실행 검증을 했다고 주장하지 않음.
- 탐색·블록: 실제 우클릭 메뉴·서버 저장 트리/사이드바 폭·사용자별 단축키·URL 읽기/집중/발표·원문 줄 이동·중첩 다중선택 Drag/Drop·복제 ID·Undo/Redo·재접속 PC/모바일 검증 및 6장 캡처 완료.
- 정보보호 정본: create/save/capture 메타데이터가 마스킹된 제목·원문·태그·별칭·블록 정보로 버전/outbox에 반영되고, 차단·충돌 요청은 부수 효과 없이 롤백됨을 PG/race 검증. 기존 민감 제목 제거 시 자동 별칭 재삽입 방지. 순수 읽기와 CRDT 초기 원문 보존은 별도 회귀 중.
- RAG 기반: 글로벌/워크스페이스 암호화 설정·상속·현재 공급자 변경 시 전역 키 비전달·고정 진단 실제 화면 PASS. 문서별 동의·재색인·철회·재정렬·worker 진행/취소·권한/키/공급자 변경 PG/race 28.912초, 실제 pgvector 확장과 배열 코사인 비교 6.385초, 브라우저 동의/취소/철회·모바일 12.837초 PASS. 대화 이력·문서별 다양한 AI 작업·자연어 검색은 여전히 후속 항목.
- 인용: 실제 SSE 첫 조각 표시, 최대 출력 262144 전달, 원문 UTF-8 바이트 범위/hash 확인, 버전 변경 409와 이전 원문 제외, 현재 원문 줄 이동·모바일·외부 자산 0 브라우저 PASS. 열린 모달의 화면 크기 전환은 추가 회귀 중.
- 그래프: 현재 ACL·버전 링크 투영, 즉시 조회 보완, 명시 관계/실제 트리 유형, 동일 이름 임의 연결 방지, 별칭 및 ID 백링크, 비공개 대상 미노출 PG/race 4.264초 PASS. 신규 UI 모바일 최종 검증 진행.
- 전체 Go 회귀: 2026-09-08 12:46 기준 PostgreSQL 포함 `go test -race ./...` 285.971초 PASS. 이후 추가한 기능은 별도 회귀와 최종 전체 재검증 대상이다. 플러그인 스트리밍 테스트는 실제 클라이언트 첫 수신 이후 권한을 회수하도록 동기화하여 3회 반복 PASS.
- 그래프 최종 UI: 모든 URL 필터·빠른 초기화/선택·관계 CRUD·조회자 쓰기 제한·깊이1~5·PC→390px 자동 맞춤 및 초기 zoom 상한 PASS. 템플릿 library 기본/개인/공간 공유·복제·YAML/날짜 치환·PII·현재 ACL/version·이력/휴지통·실제 비공개 문서 생성 PG/race 8.013초와 PC/모바일 5장 PASS.
- 정보보호 최종 UI: 관리자 정책 선택/저장·문서 등급/워터마크·비밀번호 공개링크·회전/철회·익명 390px·본문 권한 해제 PASS 및 7장 캡처. 복원 후 공개링크 폐기/정책 비활성/접근grant 제거 PG/race 3.128초 PASS. CRDT strict정책의 숨은 바이너리 차단, 최초 seed의 CRLF/frontmatter/줄바꿈/버전 무변경 PG/race 24.336초 및 고급블록 브라우저 17.685초 PASS.
- AI 문서 작업: 질문/초안/개선/요약/번역/태그/관련링크/중복/회의록/템플릿/지식공백 11종 선택→실제 SSE→사용자 미리보기→비공개 초안 생성·출처 표시·기존 원문 무변경·stale 출처 저장 거부·390px·외부 자산 0 PASS. 서버는 자동 문서/관계/업무 변경을 하지 않는다.
- 개인 AI 기록: 완료한 응답의 사용자·유효기간 바인딩 암호화 티켓, 명시 동의·멱등 저장·개인 API/검색·관리자/키 우회 거부·현재 출처 ACL/버전/해시·삭제 PG/race 및 실제 저장/새로고침/인용/통합 검색/390px/권한 회수 시 화면 본문 제거 PASS. 다중 턴의 이전 답변/원문 재전송·명시 추가 저장·동일 대화 version 2·동시 버전 충돌·공급자 변경 차단도 PG/race 9.089초 및 실제 브라우저 PASS.
- Workspace Agent: 실제 분할 SSE 도구 호출·허용 KB 밖 미전송·DB 수식/롤업 의존성·동일 트랜잭션 원본 비교·제안 후 사용자 hash 확인·REST 저장/단일 효과 영수증·세션 회수 PG/race 13.998초, 실제 설정/변경 확인/실행/이력/390px 브라우저 7.459초 PASS. 기능 플래그 변경 중단·영수증 복구 재실행 무중복 PG/race 20.924초 PASS.
