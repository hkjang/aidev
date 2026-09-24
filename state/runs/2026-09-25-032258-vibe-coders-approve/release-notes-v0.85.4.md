## AI Proxy Gateway v0.85.4

- Source commit: [`23f709a845bc1cca601420bcf3708bddb3862af9`](https://github.com/hkjang/vibe-coders/commit/23f709a845bc1cca601420bcf3708bddb3862af9)
- Source tag: [`v0.85.4`](https://github.com/hkjang/vibe-coders/tree/v0.85.4)

### 주요 변경 사항
- **엄격한 CSP 를 풀지 않고 방문 추적 스니펫을 붙일 수 있게 했다 (v0.85.4)**: 콘솔의 모든 화면이 `script-src 'self'` 로 나가기 때문에, 관리자가 분석 스니펫을 붙여 넣어도 브라우저가 조용히 거부했고 화면 어디에도 이유가 남지 않았다. `TRACKING-STANDARD.md` 의 절차를 이 저장소의 모양으로 구현했다 — `internal/tracking` 이 공급자 설정(momento 를 먼저 보고 ga4·gtm·matomo·custom 순), 요청마다 새로 만든 nonce 를 모든 `<script>` 에 붙이는 스니펫 렌더링, 대소문자 접기를 ASCII 로만 해 `İ` 나 켈빈 기호 `K`(U+212A)가 태그 인덱스를 흔들지 못하게 하는 처리, 붙여 넣은 스니펫에서 출처를 뽑아내는 파싱, 그리고 정책을 그 출처들만큼만 넓히는 확장('unsafe-inline' 은 어떤 경우에도 쓰지 않는다)을 담당한다. `/app` 콘솔과, `tracking.include_admin` 을 켰을 때의 레거시 `/admin` 콘솔이 head 또는 body 에 스니펫을 주입하고 그 페이지를 `no-store` 로 보내므로 304 가 옛 본문과 새 정책을 짝지을 수 없다. 에셋과 API 경로는 좁은 정책 그대로이고, 추적을 끄면 정책이 원래대로 돌아온다. Momento 는 기본적으로 `/momento/*` 를 통해 동일 출처로 프록시하여(자격 증명은 제거, 256KB·10초 상한) 외부 출처가 정책에 아예 닿지 않는다. 브라우저가 거부한 출처는 게이트웨이가 메모리에 모아 두고(서로 다른 100개까지, 보고 본문은 16KB 상한) 시스템 설정의 새 "방문 추적" 탭이 그 목록과 한 번 눌러 허용하기·비우기·정책이 어떻게 열리는지에 대한 설명을 보여준다. 새 `tracking` 범주의 런타임 설정 13개는 모두 기본이 꺼짐이고 직접 붙여 넣는 스니펫은 8KB 로 제한하며, `POST /tracking/csp-report`, `GET|DELETE /admin/tracking/violations`, 감사 기록을 남기는 설정 쓰기로 `tracking.allowed_hosts` 에 출처를 덧붙이는 `POST .../allow` 를 추가했다(OpenAPI 카탈로그와 스냅샷은 실제 게이트웨이에서 재생성했고 관리자 가이드 3.6 과 PDF 에 설정·CSP 설명을 실었다). 함께, `pnpm run typecheck --silent` 가 `--silent` 를 스크립트 줄에 그대로 넘겨 tsc(TS5072)와 vite(CACError)가 알 수 없는 옵션이라며 거부하던 문제를 고쳤다 — 두 스크립트를 `web/scripts/run-tool.mjs` 로 통과시켜 패키지 매니페스트에서 실행 파일을 찾고 맨 뒤에 붙은 조용히-플래그만 떼어낸다. 검사 자체는 그대로다.
- **Keycloak 세션이 살아 있으면 로그인 화면 없이 콘솔로 들어간다 (v0.85.3)**: Keycloak(또는 같은 realm 을 쓰는 SSO)에 이미 로그인한 사람이 `/app` 콘솔을 열어도 매번 로그인 화면을 거쳐야 했다. 관리자가 자동 로그인(`SSO_KEYCLOAK_AUTO_LOGIN` 또는 SSO 설정 탭의 스위치, 기본 꺼짐)을 켜면 콘솔이 탭 세션당 한 번 최상위 이동으로 `prompt=none` 시도를 하고, 세션이 있으면 평소 SSO 절차로 끝나 깊은 링크로 들어온 자리로 돌아간다(숨은 iframe 이 아니라서 서드파티 쿠키가 막힌 브라우저에서도 동작한다). 서버는 이 설정이 켜져 있을 때만 `prompt=none` 을 Keycloak 에 전달하고 꺼져 있으면 평범한 로그인으로 진행하므로 URL 만으로 흐름을 바꿀 수 없다. 핵심은 되풀이하지 않는 것이다 — 흐름 상태에 조용한 시도였음을 기록해 두고, Keycloak 이 `login_required` 로 답하면 콜백은 이를 "로그인되지 않음"이라는 평범한 대답으로 보아 실패 감사를 남기지 않고 깊은 링크를 보존한 채 `/app/login?sso=none` 으로 보낸다. 콘솔은 그 표식이 있을 때, 스스로 로그아웃한 뒤, 이 탭에서 이미 시도한 뒤에는 다시 시도하지 않으며, `sessionStorage` 를 읽을 수 없으면 "이미 시도했다"로 쳐서 사생활 보호 모드에서도 막히는 쪽으로 동작한다. 스위치를 두 SSO 설정 화면 모두에 넣어 레거시 `/admin` 저장이 값을 되돌리지 않게 했고, `auto_login` 을 `/auth/sso/status` 와 UI 부트스트랩에 공개했으며(OpenAPI 스냅샷은 실제 게이트웨이에서 재생성), 관리자 가이드에 설정과 동작을 적었다.
- **콘솔을 키보드로 움직이고, 첫 화면이 할 일을 알려주고, 워크플로를 조립해 만든다 (v0.85.2)**: 2026년 AI 게이트웨이 제품·콘솔 UX 자료를 조사해 구현 상태와 대조한 뒤, 실제로 비어 있던 세 곳을 채웠다. 명령 팔레트가 화면 이동만 하던 것을 명령·최근 화면·ID 바로 찾기까지 다루도록 했다 — 지금 새로고침, 테마·정보 밀도·자동 갱신 주기 전환, 사이드바 접기를 헤더 메뉴를 찾지 않고 실행하고, 검색창을 비우면 방금 보던 화면이 먼저 나오며, 요청·추적·세션 ID를 붙여넣으면 해당 요청으로 직행한다(값이 URL 쿼리에 들어가므로 필터 폼과 같은 검사로 자격 증명처럼 보이는 문자열은 거부하고 퍼센트 인코딩한다). `?` 로 단축키 시트를 열 수 있게 해 Ctrl+K 의 존재를 알 방법을 만들었고, 입력 칸에 포커스가 있을 때는 물음표가 그냥 물음표로 남는다. 아직 한 건도 처리하지 않은 게이트웨이의 통합 현황은 모든 패널이 0이라 무엇을 설정해야 하는지 알려주지 못했다 — 공급자 연결·인증 켜기·모델 가격 설정·첫 요청 보내기 네 단계를 게이트웨이가 보고하는 값에서 판정해 보여주고, 요청이 들어오면 스스로 사라진다. 특히 가격 설정이 빠지면 비용·예산·절감 화면이 0으로 남아 고장으로 보이던 문제를 표면화한다. 워크플로 정의는 JSON 배열을 직접 타이핑해야 했는데, 서버가 이미 타입이 잡힌 단계 모델을 갖고 있었으므로 단계 편집기로 바꿨다 — 종류를 골라 추가하고 순서를 바꾸며, 그 종류가 쓰는 항목만 표시한다(채팅은 모델과 상한, MCP 는 허용 도구, Text2SQL 은 허용 테이블). 폼이 같은 JSON 문자열을 유지하므로 검증과 요청 본문은 그대로이고 JSON 뷰가 정확한 대안으로 남으며, 이 빌드가 모르는 필드를 가진 단계는 값을 보존하고 그 사실을 화면에 적는다. 한도를 비우면 0 대신 항목을 제거한다 — 서버는 상한이 없으면 제한 없음, 0 이면 진짜 0 으로 읽는다.
- **사용자·관리자 가이드를 실제 콘솔 화면 기준으로 재작성 (v0.85.1)**: 문서 전용 릴리즈로 게이트웨이 코드 변경은 없다. 사용자 가이드는 제품이 하는 일 → 처음 5분 → 화면별 사용법 → 자주 하는 작업(Roo Code·Cline·Cursor·OpenAI SDK 연결) → 막혔을 때(파이프라인이 실제로 반환하는 오류 코드) → 용어 순으로 다시 썼다. 관리자 가이드는 구성 요소, 릴리즈 자산으로 처음부터 설치하는 절차, `config.go` 에서 읽은 환경 변수 표, `roleScopes` 권한 표, 운영, 증상→확인→조치 런북, 보안 기본값을 1부로 두고 기존 `/admin` 탭 레퍼런스를 2부로 유지한다. 두 문서 모두 `/app` 콘솔 24개 화면을 1440x900 으로 실제 캡처한 그림(`docs/images/guide/`)을 싣고 공용 md2pdf 도구로 만든 PDF(`docs/USER_GUIDE.pdf`, `docs/ADMIN_GUIDE.pdf`)를 함께 제공한다. 캡처는 `scripts/guide-screenshots.sh` 가 임시 SQLite 와 무작위 자격 증명으로 일회용 게이트웨이를 띄우고 모의 업스트림으로 가짜 트래픽을 흘린 뒤 `web/scripts/guide-screenshots.mjs` 로 찍으며, 드라이버는 일회용 배포라고 명시하지 않는 한 루프백이 아닌 대상을 거부하고 화면을 읽기만 하므로 실제 배포를 건드리지 않는다.
- **감사가 놓치던 화면 15개를 `/app`으로 이식하고, 조작 URL이 다른 리소스를 지우던 결함을 수정 (v0.85.0)**: Legacy 콘솔과 `/app` 콘솔의 격차를 CI가 기계적으로 검사하도록 하고(`cmd/api-surface-audit`), 그 검사가 스스로 놓치던 두 허점을 막았다. 생성된 OpenAPI 타입 디렉터리를 비교 대상에 포함하고 있어 Legacy 콘솔을 `/app` 콘솔이 아니라 카탈로그와 비교했고, Legacy 호출의 첫 문자열만 읽어 `'/admin/requests/' + id + '/trace'` 가 `/admin/requests/` 로 잘리면서 이어붙인 경로 아래 모든 하위 동작이 이식된 것으로 계산됐다. 둘을 막자 `/app`이 한 번도 호출하지 않던 엔드포인트 15건이 드러났고 전부 이식했다 — 운영 홈(운영 신호 카드·장애 후보·백그라운드 워커·배포 프리플라이트·인덱스 드리프트와 추가 삭제 후보·마이그레이션 SQL 전문, 모두 읽기 전용), 요청 상세의 종단 간 스팬 워터폴과 연관 기록 집계, 요청 탐색기의 필터 자동완성과 두 요청 비교, 통합 현황의 사용 추이 차트와 요일·시간대 히트맵, 데이터 인사이트의 비용 절감·모델 전환 추천, 시스템 설정의 지식 캐시. 함께 고친 서버 결함: 서브 액션 URL이 부모 리소스에 작용해 `DELETE /admin/prompt-lab/test-cases/{id}/run` 이 테스트 케이스를, `.../dw/metrics/{key}/validate` 가 지표를, `/admin/apps/{id}/publish` 가 앱을 삭제하던 문제(URL은 실행·검증·발행을 가리키는데 서버는 삭제를 수행했고 오류조차 없었다), `GET /admin/dw/dashboard/refresh` 가 조회만으로 DW 캐시를 비우고 감사 기록을 남기던 문제, MCP 업스트림 프로브가 GET 이라 읽기 전용 권한으로도 등록된 모든 업스트림에 외부 연결을 발생시킬 수 있던 문제(POST 로 변경, `mcp:admin` 필요). 문서와 구현이 어긋나 호출할 수 없던 계약 3건도 바로잡았다 — 골든 쿼리 삭제(경로가 아닌 `?id=`), 지식 항목 수정(PATCH), 자동 라우팅 학습 상태 조회(GET). 프런트엔드 의존성 js-yaml 을 4.3.2 로 올려 GHSA-2883-xcg3-v3hh 를 해소했다.
- **조회만 해도 기록되던 관리 API를 읽기 전용으로 수정 (v0.84.1)**: `GET /admin/anomalies` 가 응답을 만드는 김에 이상 이벤트를 저장하고 알림까지 보내던 문제를 수정했다. 대시보드를 새로 고칠 때마다 같은 구간이 다시 채점되어, 급증을 지켜보던 운영자가 중복 이벤트와 알림을 스스로 만들어 내고 있었다. 이제 기록과 알림은 게이트웨이 내부 워커가 5분마다 수행하며(기준선 7일, 최근 1시간, z=3, 최소 15분 중복 억제), 엔드포인트는 읽기만 한다. `GET /admin/personalization/profiles/{user_id}?snapshot=1` 이 스냅샷을 기록하던 문제도 수정했다. 크롤러·프리페치·브라우저 재시도만으로 스냅샷 이력이 조용히 늘어날 수 있었으므로, 기록은 같은 경로의 POST 에서만 일어나고 GET 은 어떤 형태로 호출해도 상태를 바꾸지 않는다. 레거시 콘솔은 POST 로 스냅샷을 남기고, React 콘솔은 이상 징후를 안전하게 조회하려고 붙이던 `record=0` 우회를 제거했다. 두 결함을 각각 되살려 회귀 테스트가 실제로 실패하는지 확인했다.
- **Legacy 콘솔의 모든 화면을 신규 `/app` 콘솔로 이식 (v0.84.0)**: 읽기 전용 미리보기 7개뿐이던 React 콘솔이 등록된 36개 기능 전부를 소유한다. 13개 도메인을 레거시 자바스크립트가 아니라 Go 핸들러를 근거로 이식해, 조회뿐 아니라 `/admin`이 제공하던 생성·수정·삭제·실행·내보내기를 동일하게 제공한다. 사용자·팀·API 키·IP·할당량·역할, 내 홈과 팀 포털, Chat 테스트(SSE 스트리밍·멀티 모델 비교·평가·승격·Golden)와 프롬프트 실험실, 공급자·모델·모델 계약·지원 종료 관리, 라우팅 규칙·미리보기·결정 이력·장애 전환·학습 엔진·회로 차단기, 세션 비행기록·XView 실시간 산점도·LLM 관측·진단 프로브, 프롬프트 검색과 자산 검토·승인·롤백·부채, 안전 정책·Kill Switch·알림 규칙·모델 일몰·정책 어드바이저·자동 조치, 팀 성숙도·운영 보고서·AI 업무성과·SBOM·개인화, MCP 업스트림·도구·정책·요청 워터폴과 에이전트 라우트·워크플로·앱·Skill, Text2SQL 전 영역, DW 대시보드·ClickHouse·데이터 상품, 비용 대시보드·배부·예산, 보안 대시보드·프라이버시 원장·Red Team·샌드박스, 시스템 설정·런타임 설정·변경 세트·SSO·감사 이력과 `/app` 전환 설정을 포함한다. 화면이 자기 라우트와 URL 쿼리 키를 직접 선언하는 도메인 등록부를 도입해 라우터·쿼리 허용 목록·구현 판정이 한 곳에서 갈라지며, 선언하지 않은 쿼리 키는 제거되어 비밀값과 프롬프트가 주소창에 남지 않는다. 정적 쿼리 허용 목록이 화면 선언을 가려 라우팅 화면의 필터 6개가 조용히 제거되던 기반 버그를 수정. 핸들러와 어긋나던 OpenAPI 카탈로그 29건을 정정했다 — 문서대로 호출하면 405가 나던 경로(할당량·알림 규칙·라우팅 규칙·MCP 업스트림·템플릿·변경 세트·내 키·멀티 실행), 서버에 구현되어 있으나 문서에 없어 호출할 수 없던 조작(템플릿 검토·승인·롤백·이력, 변경 세트 승인 워크플로, 역할 삭제, 팀 리포트 승인, 키 회전, Skill 접근 신청·평가·적합성 기록, 요청 원인 설명·메모·분석·재실행, DW 지표 삭제·검증, Text2SQL 연결 삭제, MCP 도구 위험등급·연결 진단, 프롬프트 실험 수정·삭제·테스트 실행), 그리고 서버 경로와 철자가 달라 404가 나던 문서 오류 1건. 역할로만 결정되어 클라이언트가 알 수 없던 프롬프트 원문 열람 권한을 부트스트랩이 능력으로 알려 주도록 해 버튼을 사유와 함께 비활성화하고, 변경 세트의 적용·롤백·삭제가 운영자 사유를 감사 이력에 남기도록 수정(삭제는 감사 기록이 전혀 없었음). Mattermost 알림 설정 조회가 채널 게시 권한을 가진 웹훅 토큰을 원문으로 반환하던 문제를 마스킹으로 수정
- **SSO 로그인 시 로컬 지정 역할·팀 강등 방지와 계정 복구 명령 (v0.83.2)**: v0.82.1~v0.83.1은 Keycloak 로그인마다 저장된 역할·팀을 클레임 계산 결과로 덮어써, 토큰의 역할이 역할 매핑에 걸리지 않으면 콘솔에서 승격한 `super_admin`도 기본 역할로 강등되고 groups 클레임이 없으면 팀이 비워져 로그인 직후 운영·보안·설정 메뉴가 거의 사라지던 문제를 해결. 외부 신원(auth_identities)에 IdP가 직전 로그인에서 직접 부여한 역할·팀(`idp_role`/`idp_team`)을 기록하고, 명시적 매핑은 상향·하향 모두 적용하되 매핑이 없을 때는 IdP가 부여했던 것만 철회하고 관리자가 지정한 역할·팀은 유지하도록 변경(Keycloak에서 역할을 제거한 경우의 회수는 유지). 이미 강등돼 남은 관리자가 없는 경우를 위해 데이터 볼륨에 대해 실행하는 `set-user-role --email --role` 보조 명령을 추가(내장·커스텀 역할 검증, 기존 세션 폐기, `role_repaired` 감사 기록)하고 운영 가이드 8.7 런북을 추가
- **nonroot 데이터 디렉터리 권한 진단·복구로 8080 미기동 해결 (v0.83.1)**: 운영 이미지는 distroless nonroot(uid 65532)로 실행되는데, root가 만든 바인드 마운트·이전 배포의 볼륨·fsGroup 없는 Kubernetes PVC처럼 다른 사용자가 기록한 `/data`를 이어받으면 SQLite의 `attempt to write a readonly database`만 남기고 종료해 컨테이너가 재시작을 반복하고 8080이 열리지 않던 문제를 해결. 기동 시 데이터 디렉터리·DB 파일·WAL/SHM 사이드카의 실제 쓰기 가능 여부를 먼저 검사해 실패 경로마다 소유자·권한·원인과 복구 명령을 로그에 남기고, Fallback 로그 디렉터리가 쓰기 불가하면 WARN으로 알림. 이미지에 셸이 없어도 같은 이미지로 진단·복구할 수 있도록 게이트웨이 바이너리에 `check-data-dir`(nonroot, 변경 없음, 종료 코드로 판정)·`repair-data-dir`(root 1회, `/data` 이하 65532:65532 재소유와 소유자 읽기·쓰기 비트 복원, 멱등, 심볼릭 링크 미추적, `--uid/--gid` 지정 가능)·`version` 보조 명령을 추가. 컨테이너 smoke가 root 소유 볼륨에서 nonroot 기동 거부·오류 메시지·`check-data-dir` 실패·`repair-data-dir` 복구·재기동까지 검증하도록 확장했고, 오프라인 README·릴리즈 노트·README·운영 가이드(8.6 런북)·릴리즈 가이드·compose에 볼륨 재사용 시 1회 복구 절차와 8080 미응답 진단 절차를 추가
- **요청 단위 추적 탐색기·추적 조회 권한 및 성능 강화 (v0.83.0)**: `/app/observability/traces` 읽기 전용 미리보기를 추가해 같은 추적 ID로 연결된 요청을 시작 시각·지연 구간·HTTP 상태·모델·안전 공급자 표시명·토큰·비용 기준의 시간축과 표로 비교하고, URL 필터 복원·필터 결합 암호화/서명 양방향 커서·자동 갱신·마지막 정상 데이터·재시도·요청 탐색기 및 기존 화면 연결을 제공. 메뉴·버튼·상태·안내 문구는 한글을 우선하며 키보드와 스크린리더용 의미 구조, 상세 포커스 이동·복원을 유지. 신규 화면은 기존 `/admin/requests`의 `X-Vibe-UI: app` 안전 투영만 재사용해 프롬프트·응답 본문·원시 오류·사용자 에이전트·도구 인자·SQL을 브라우저로 내려보내지 않고, 원문 오류나 Text2SQL 거절 사유가 포함될 수 있는 레거시 상세 추적 API는 호출하지 않음. 안전한 요청 ID는 원문 딥링크로 유지해 `GATEWAY_SECRET` 교체에도 안정적으로 복원하고, 비공개 처리된 중복 표시 ID는 비밀 탐지 규칙과 충돌하지 않는 분절 HMAC 참조와 별도 `selected_ref` URL 공간으로 정확히 구분하며 새로고침 회귀를 검증. 정확한 `trace_id` 조회용 복합 부분 인덱스를 SQLite와 PostgreSQL에 추가하고 20만 행 자연 계획에서 범위 탐색을 검증했으며, 과대한 레거시 식별자를 인덱스에서 제외해 업그레이드 안전성을 유지. 저장된 표기 그대로의 유효한 레거시 IPv6 필터도 정규화로 놓치지 않게 했고, `X-Vibe-App-Requests-Version`을 추가해 헤더 미지정 또는 `1`은 v0.82 안전 투영의 정확한 형태를, `2`는 신규 HMAC 참조·필터 가능 여부를 반환하며 잘못되거나 중복된 버전은 거부하고 두 변형을 `Vary`와 OpenAPI에 명시. 요청 탐색기와 추적 탐색기의 최소 백엔드 계약을 v0.83.0으로 고정하며 서버가 내려주는 더 낮은 런타임 버전으로 하한이 완화되지 않게 해 롤링 배포 중에는 안전하게 기존 화면으로 전환. 기존 요청 추적과 LLM 추적 상세/목록에는 GET 전용 메서드·길이·UTF-8·경로 구분자 검증, 조회 실패 시 안정 오류, exact 팀 소유권 검사, 하위 권한 오류·대체 사유·거절 사유 감사 마스킹, 특수문자 요청 ID 링크 인코딩을 적용해 다른 팀 데이터와 내부 오류가 새지 않도록 강화. 릴리스 파이프라인은 체크섬으로 고정한 Grype 0.117.0으로 실제 Distroless 최종 이미지의 High·Critical 취약점을 패키징 전에 차단하고 `/admin` Legacy 본문도 컨테이너 smoke에서 확인. Text2SQL 검증기의 LIMIT 규칙이 중첩을 고려하지 않고 문장 전체를 훑어 서브쿼리·CTE 본문의 LIMIT이 바깥 쿼리를 대신 대답하던 문제를 수정. 상한 검사는 텍스트 순서상 첫 매치만 보던 방식을 버리고 모든 LIMIT을 `MaxLimit`과 비교해 `(SELECT ... LIMIT 5) ... LIMIT 999999` 형태가 통과하지 못하게 하고, 기본 LIMIT 주입은 괄호 밖 최상위 LIMIT 유무로만 판단해 서브쿼리에만 LIMIT이 있는 무제한 쿼리에도 기본 상한이 붙도록 했으며, 값 파싱은 개행이 섞인 `LIMIT\n999999`나 오버플로 리터럴을 0으로 읽어 상한을 무력화하지 않도록 정규식 캡처와 정수 변환으로 바꾸고 해석할 수 없는 값은 거부

### 배포 파일
| 파일 | 설명 |
|------|------|
| ai-coding-proxy-gateway-v0.85.4.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| ai-coding-proxy-gateway-v0.85.4.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.85.4.md | 오프라인 배포 가이드 |
| SBOM-v0.85.4.spdx.json | Go+npm 통합 SPDX SBOM |
| THIRD_PARTY_LICENSES-v0.85.4.md | Go·Frontend 제3자 라이선스 목록 |
| init-deployment-env-v0.85.4.sh | 운영 env 원자 생성·검증 helper |
| backup-volume-v0.85.4.sh | named volume·env 백업·복구 helper |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c ai-coding-proxy-gateway-v0.85.4.tar.gz | docker load

# 최초 1회 비밀값 파일과 nonroot 호환 named volume 준비
ENV_FILE=/opt/proxy-gateway/gateway.env
install -d -m 0700 "$(dirname "$ENV_FILE")" || exit 1
command -v openssl >/dev/null 2>&1 || { echo "openssl이 필요합니다." >&2; exit 1; }
if [ ! -e "$ENV_FILE" ]; then
  umask 077
  ADMIN_TOKEN_VALUE="$(openssl rand -hex 32)" || exit 1
  GATEWAY_SECRET_VALUE="$(openssl rand -hex 32)" || exit 1
  [ "${#ADMIN_TOKEN_VALUE}" -eq 64 ] && [ "${#GATEWAY_SECRET_VALUE}" -eq 64 ] || exit 1
  printf '%s\n' "$ADMIN_TOKEN_VALUE" | grep -Eq '^[0-9A-Fa-f]{64}$' && 
    printf '%s\n' "$GATEWAY_SECRET_VALUE" | grep -Eq '^[0-9A-Fa-f]{64}$' || exit 1
  UPSTREAM_API_KEY_VALUE="${UPSTREAM_API_KEY:-}"
  if [ -z "$UPSTREAM_API_KEY_VALUE" ]; then
    read -r -s -p 'Upstream API key: ' UPSTREAM_API_KEY_VALUE || exit 1
    echo
  fi
  case "$UPSTREAM_API_KEY_VALUE" in
    ''|*[!A-Za-z0-9._~:/+=-]*) echo 'UPSTREAM_API_KEY 형식이 안전하지 않습니다.' >&2; exit 1 ;;
  esac
  ENV_TMP="$(mktemp "${ENV_FILE}.tmp.XXXXXX")" || exit 1
  trap 'rm -f -- "$ENV_TMP"' EXIT HUP INT TERM
  {
    echo 'UPSTREAM_BASE_URL=https://api.openai.com'
    echo "UPSTREAM_API_KEY=${UPSTREAM_API_KEY_VALUE}"
    echo 'GATEWAY_VERSION=v0.85.4'
    echo "ADMIN_TOKEN=${ADMIN_TOKEN_VALUE}"
    echo "GATEWAY_SECRET=${GATEWAY_SECRET_VALUE}"
    echo 'UI_APP_ENABLED=false'
  } > "$ENV_TMP" || exit 1
  chmod 0600 "$ENV_TMP" && mv -f -- "$ENV_TMP" "$ENV_FILE" || exit 1
  trap - EXIT HUP INT TERM
fi
chmod 0600 "$ENV_FILE" || exit 1
[ "$(grep -Ec '^ADMIN_TOKEN=' "$ENV_FILE")" -eq 1 ] && grep -Eq '^ADMIN_TOKEN=[0-9A-Fa-f]{64}$' "$ENV_FILE" && 
  [ "$(grep -Ec '^GATEWAY_SECRET=' "$ENV_FILE")" -eq 1 ] && grep -Eq '^GATEWAY_SECRET=[0-9A-Fa-f]{64}$' "$ENV_FILE" && 
  [ "$(grep -Ec '^UPSTREAM_API_KEY=' "$ENV_FILE")" -eq 1 ] && grep -Eq '^UPSTREAM_API_KEY=[A-Za-z0-9._~:/+=-]+$' "$ENV_FILE" && 
  ! grep -q '^UPSTREAM_API_KEY=replace-before-start$' "$ENV_FILE" && 
  [ "$(grep -Ec '^GATEWAY_VERSION=' "$ENV_FILE")" -eq 1 ] && 
  grep -qxF 'GATEWAY_VERSION=v0.85.4' "$ENV_FILE" || 
  { echo "gateway.env의 필수 비밀값이 유효하지 않습니다." >&2; exit 1; }
docker volume create proxy-gateway-data >/dev/null || exit 1

# 기존 볼륨·바인드 마운트 재사용 시 nonroot(65532) 소유권 복구 (새 볼륨은 변경 없음)
docker run --rm --user 0:0 --mount source=proxy-gateway-data,target=/data ai-coding-proxy-gateway:v0.85.4 repair-data-dir || exit 1

docker run -d --name proxy-gateway --restart=always \
  -p 8080:8080 \
  --mount source=proxy-gateway-data,target=/data \
  --env-file "$ENV_FILE" \
  ai-coding-proxy-gateway:v0.85.4
```

- Legacy Stable Console: http://localhost:8080/admin
- Next Console Preview: http://localhost:8080/app/ (/app is OFF unless UI_APP_ENABLED=true)
- React assets are embedded in the Go binary; the runtime image does not contain Node.js.
- Reuse and back up the same 0600 env file; rotating GATEWAY_SECRET without migration makes stored Provider Secrets unreadable.
- Transfer, checksum, and chmod 0700 the bundled init-deployment-env-v0.85.4.sh and backup-volume-v0.85.4.sh helpers.
- The init helper preserves secrets and atomically updates only GATEWAY_VERSION during upgrades.
- If the container restarts and 8080 never opens with 'readonly database' in docker logs, /data was written by another user: run the repair-data-dir command above once, then verify with: docker run --rm --mount source=proxy-gateway-data,target=/data ai-coding-proxy-gateway:v0.85.4 check-data-dir
