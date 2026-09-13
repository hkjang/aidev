## 2026-09-13
- 선택: 캠페인 mail-2026-09 — 사내 SMTP 릴레이로 이벤트 알림을 보내는 체계(설정·배경 발송·발송 기록·시험 발송·관리 화면·가이드), 기본 꺼짐 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: MAIL-STANDARD.md 를 따라 `server/internal/mail`(릴레이 연결·`auto/none/starttls/tls`
  협상·PLAIN/LOGIN/CRAM-MD5 선택 인증·MIME 조립·`Delivery` 기록·배경 발송 1회 재시도)과
  `httpapi/mail.go`(설정 읽기/쓰기·시험 발송·기록 조회·사용자 표에서 id→메일 조회)를 만들었다.
  설정은 표준 표의 이름 그대로 `mail.enabled`, `mail.smtp_host`, `mail.smtp_port`(25), `mail.security`(auto),
  `mail.skip_tls_verify`, `mail.username`·`mail.password`(선택), `mail.from_address`·`mail.from_name`,
  `mail.base_url`, `mail.timeout_seconds`(10), `mail.notify_<이벤트>` 를 공용 `settings` 표에 행으로 저장해
  모든 Pod 가 같이 읽고, `mail.password` 는 Keycloak client secret 과 같은 마스터 키로 봉인해 `secret=TRUE`
  로 저장하며 GET 은 `password_configured` 만 돌려주고 범용 설정 API 는 이 키를 거절한다(로그·감사에도 없음).
  migration 009 `mail_deliveries`(SQLite·PostgreSQL). 고른 이벤트 4개와 이유 — (1) `account.locked`
  로그인 실패 누적 잠김 → 계정 주인 + 모든 super_admin: 잠긴 사람은 해제를 기다리고 관리자는 전화로만
  알게 된다(auth.Service 에 잠금 hook 추가) (2) `account.unlocked` → 계정 주인: 기다리던 바로 그 소식
  (3) `account.created` → 새 사용자: 로그인할 수 있게 됐다는 것을 기다린다, 비밀번호는 담지 않음
  (4) `agent_update.halted` 배포 0% → 행위자를 뺀 super_admin: 멈춘 배포를 고장으로 오해해 화면을
  새로고침하지 않게. 단순 '무언가 바뀜'(자산 수정·설정 변경·키 발급 등)은 제외했고, 만료 임박류(API key
  만료)는 주기 검사기가 없어 이번엔 넣지 않았다. 행위자는 수신자에서 빠지고 한 작업당 한 사람에게 한
  통이며, 주소가 없거나 비활성인 계정은 건너뛴다. 콘솔에 설정 → 메일 알림 탭(설정 폼·이벤트 스위치·
  시험 발송·발송 기록·상태 필터)을 추가하고 `webui/dist` 재빌드, openapi.yaml 에 4개 경로와 스키마,
  ADMIN_GUIDE.md 21장(설정 표·비밀번호 취급·이벤트 선정 이유·시험 발송·발송 기록·API)과 PDF 재생성,
  README 한 줄. 검증: mail 단위 11개(테스트용 인프로세스 SMTP 서버로 실제 한 통 도착·헤더 인코딩·
  dot-stuffing, RCPT 거절 단계 보고, 죽은 릴레이에서 2초 내 실패·비밀번호 미노출, 기본값 25/auto/꺼짐과
  465→tls, 검증 거절, 꺼짐·미완성이면 무발송, 행위자 제외·중복 제거, 스위치는 그 종류만, 성공·실패 모두
  기록·본문 없음), httpapi 통합 2개(기본 꺼짐·읽어도 행 없음, 비밀번호 봉인·마스킹·범용 API 409·유지/
  지우기, 잘못된 값 400; 꺼짐이면 무발송 → 켠 뒤 계정 생성/5회 실패 잠김/해제 각각 기다리는 사람에게만,
  스위치 끄기, 죽은 릴레이(닫힌 포트)로도 해제 요청 200 즉시 + failed 기록 2회 시도, 시험 발송 502 와
  자기 주소 기본값), vitest 2개 추가로 `go test ./...` SQLite·실 PostgreSQL(`scripts/test-postgres.sh`)
  전 패키지 통과, `go vet`·`gofmt`·`go build`, vitest 144개 통과, OpenAPI 경로 대조 테스트 통과. 실제
  사내 릴레이는 이 환경에 없어 테스트용 SMTP 서버로 도착을 확인했다. 새 화면 캡처는 넣지 않았다
  (캡처 대조 테스트가 있어 다음 회차). 버전 범프·릴리즈 노트는 하지 않았다.
- 보류 아이디어: API key 만료 임박 메일 — 주기 검사기가 없어 하루 1회 점검 루프와 '이미 보냄' 표시가 먼저 필요 (가치 3 / 위험 2 / M) · 설정 → 메일 알림·Keycloak(auto_login)·방문 추적 화면 캡처를 캡처 스크립트에 추가해 가이드에 싣기 (가치 2 / 위험 1 / S) · Query DSL 에 `attributes.<키>` 존재/부재 연산자가 없어 `>= ""` 우회가 필요함 (가치 3 / 위험 2 / M) · MCP `asset_relations` 가 상대 자산의 이름·종류를 주지 않아 edge 마다 `asset_get` 을 한 번 더 부르게 만듦 (가치 3 / 위험 2 / M) · 콘솔 Query DSL 화면이 새 `total`·`offset` 을 쓰지 않아 여전히 첫 페이지만 보여줌 (가치 3 / 위험 2 / M)
