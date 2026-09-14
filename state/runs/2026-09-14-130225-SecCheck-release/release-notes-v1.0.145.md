
### 추가
- **메일 알림이 사내 SMTP 릴레이 표준을 따릅니다**: 설정 행이 `notification`에서
  `mail`로 바뀌고 키 이름이 표준(`enabled`·`smtp_host`·`smtp_port`·`security`·
  `skip_tls_verify`·`username`·`password`·`from_address`·`from_name`·`base_url`·
  `timeout_seconds`·`digest_hour`·`notify_*`)과 같아졌습니다. 기본값은 사내
  릴레이(포트 25, 인증 없음, `security=auto` — 릴레이가 STARTTLS를 알리면 쓰고
  아니면 평문)이며 **새 설치는 꺼져 있어 아무것도 보내지 않습니다**.
- **기다리는 사람이 있는 알림만** 메일로 갑니다. 네 묶음 스위치 — 내 차례
  (`notify_turn`), 결과(`notify_decision`), 기한(`notify_deadline`), 장애
  (`notify_failure`) — 로 고르고, 댓글·API 키 폐기는 종에만 남습니다. 자기 행동으로
  생긴 알림은 본인에게 메일하지 않습니다(장애 묶음은 예외).
- **시도마다 발송 기록**이 남습니다(`mail_deliveries`, 마이그레이션 **036**):
  시각·이벤트·수신자·제목·`SENT`/`FAILED`/`SKIPPED`·시도 횟수·오류. 본문은
  남기지 않습니다. 메일 탭 아래와 `GET /api/v1/admin/mail/deliveries`에서 보며,
  정기 점검이 `retention_days`로 정리합니다. 관리자 가이드 3-5 절.
- **Keycloak에 이미 로그인한 사람은 로그인 화면을 건너뜁니다**: OIDC 설정의
  `auto_login`(기본 꺼짐, 마이그레이션 **035**)을 켜면 세션 없는 브라우저가
  `prompt=none`으로 한 번만 물어보고, 세션이 있으면 열려던 주소로 바로 들어갑니다.
  없으면 `/login?sso=none`으로 돌아오며, 로그아웃 직후·만료 직후·거절을 받은
  뒤에는 다시 묻지 않으므로 무한 리다이렉트가 생기지 않습니다.
- **방문 추적 스니펫**: 서비스 설정에 `방문 추적` 탭이 생겼습니다(마이그레이션
  **034**, 기본 꺼짐). Momento(사내 수집기, 같은 출처 프록시 `/momento/*`)·GA4·
  GTM·Matomo·직접 붙여 넣기 중 고르면 서버가 앱 셸에 넣습니다. 콘텐츠 보안 정책은
  `'unsafe-inline'`을 쓰지 않고 요청마다 nonce를 발급하며, 스니펫이 부르는 출처만
  허용 목록에 더합니다. 켜져 있는 동안 브라우저가 차단한 출처가 탭에 기록되어
  한 번에 허용할 수 있습니다(`GET/DELETE /admin/analytics/violations`,
  `POST /admin/analytics/allow`). 관리자 가이드 3-4 절.
- **`seccheck verify-schema`**: 데이터베이스가 이 빌드와 맞는지 명령줄에서
  묻습니다 — 빌드가 기대하는 것, 데이터베이스가 가진 것, 빠진 것(`--json` 가능).
  v1.0.144가 고친 것과 같은 문제를 로그인 화면이 막혀 있어도 확인할 수 있습니다.

### 수정
- **`결재 요청 회수`·`보완 재개` 대화상자가 화면에서 동작하지 않던 문제**: 사유를
  `comment`로 보냈는데 서버는 `reason`을 읽어 매번 `INVALID_JSON`이었습니다.
  API로는 됐고 화면만 안 됐습니다.
- **`verify-schema`가 늘 실패하던 문제**: 스크래치 스키마 이름에 UUID의 `-`가
  들어가 `CREATE SCHEMA`가 구문 오류였습니다.
- 이미지 취약점 게이트: 베이스 이미지의 `libpcre2-8-0`(CVE-2026-86145·
  CVE-2026-89161)을 `apt-get upgrade`로, `golang.org/x/crypto/ssh`
  (CVE-2026-56855·CVE-2026-78662)를 v0.56.0으로 닫았습니다.

### 문서
- 사용자·관리자 가이드를 실제 화면 캡처로 다시 썼고(승인자 화면·빠른 검색·참여자·
  재심의 복사 포함), 가이드가 인용하는 버튼·메시지·로그 줄·설정 키·환경 변수·
  상태 엔드포인트·작업 유형·알림 제목이 코드와 같은지 테스트가 붙잡습니다.
- 관리자 가이드 2-4 절이 `compose.yaml` 전문을 싣습니다 — 오프라인 설치본은
  저장소를 볼 수 없으므로.
- `scripts/precheck.sh`가 그림 누락·너비, PDF 신선도, vitest까지 확인하며 모든
  가이드 PDF는 공용 md2pdf로 굽습니다.

### 주의
- **`PUT /api/v1/admin/settings/notification`과
  `POST /api/v1/admin/settings/notification/test`가 없어졌습니다.** 각각
  `/admin/settings/mail`과 `/admin/settings/mail/test`이며 키 이름도 위와 같이
  바뀝니다(`email_enabled`→`enabled`, `smtp_tls_mode`→`security`,
  `smtp_username`→`username`, `from`→`from_address`). 릴레이를 설정해 둔 설치는
  마이그레이션 036이 값과 `general.base_url`을 새 행으로 옮기므로 그대로
  보내지고, 옛 이름으로 봉인된 비밀번호는 다음 저장 때 다시 봉인됩니다.
  `general.base_url`은 메일 탭의 `base_url`이 됩니다.
- 메일 탭의 네 스위치가 **모두 켜진 채** 이관됩니다. 특정 유형만 보내고 싶으면
  업그레이드 뒤 스위치를 끄십시오. 발송 대상은 이전보다 좁습니다(댓글·API 키
  폐기 알림은 더 이상 메일로 가지 않음).

