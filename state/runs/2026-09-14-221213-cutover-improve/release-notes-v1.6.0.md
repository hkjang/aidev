## KCB Cutover Dashboard v1.6.0

### 신규 기능
- **사내 SMTP 릴레이로 보내는 메일 알림**: 관리자 콘솔(`/admin`)의 **메일 알림 (SMTP 릴레이)** 카드에서 상황판 이벤트를 메일로 알릴 수 있습니다. **기본값은 꺼짐**이며, 켜기 전까지는 아무것도 나가지 않고 설정 파일(`data/mail.json`)도 만들지 않습니다. 설정은 재배포 없이 화면에서 바뀝니다.
  - 알리는 이벤트 4가지 — `activity.delayed`(작업 지연 발생), `activity.delay_cleared`(지연 해소), `task.completed`(최상위 Task 완료), `drill.completed`(모든 Task 완료). 이벤트별로 켜고 끌 수 있고, 제목·시간 편집이나 추가·삭제·이동, JSON 업로드처럼 단순히 "무언가 바뀐" 것은 보내지 않습니다.
  - 한 번의 조작으로 여러 항목이 바뀌면 **한 통으로 묶어** 보내며, 가장 급한 것(지연 > 훈련 완료 > 작업 완료 > 지연 해소)이 제목이 되고 나머지는 `외 n건`으로 접힙니다.
  - 발송은 **응답이 나간 뒤 배경에서**(Next `after()`) 이뤄지므로 릴레이가 느리거나 죽어 있어도 상태 클릭은 즉시 끝납니다. 연결 거부 시 2초 뒤 1회 재시도.
  - 설정 키는 사내 메일 표준(MAIL-STANDARD)과 같습니다: `mail.enabled`, `mail.smtp_host`, `mail.smtp_port`(기본 25), `mail.security`(`auto`/`none`/`starttls`/`tls`), `mail.skip_tls_verify`, `mail.username`/`mail.password`(선택, PLAIN·LOGIN), `mail.from_address`/`mail.from_name`, `mail.base_url`(메일 속 바로 열기 링크), `mail.timeout_seconds`, `mail.recipients`(쉼표·줄바꿈 구분, 최대 50명), `mail.notify_*` 스위치.
  - 이 앱에는 개인 계정이 없으므로 받는 사람은 **상황실 배포 목록**(`mail.recipients`) 하나로 관리합니다.
  - **SMTP 비밀번호는 되읽히지 않습니다.** `GET /api/mail`은 `password_set`으로 "설정됨" 여부만 알리고, 화면에서도 **설정됨**만 보입니다. 빈 값으로 저장하면 기존 값 유지, 지울 때는 **지우기**(`password_clear`). 로그·발송 기록·오류 문장에도 비밀번호가 들어가지 않습니다.
  - **시험 발송**: 저장된 설정으로 실제 한 통을 보내고 결과(릴레이가 돌려준 이유 포함)를 그 자리에서 보여 줍니다.
  - **최근 발송 기록**: 시도마다 시각·이벤트·수신자·제목·결과(`성공`/`실패`/`대기`)·시도 횟수·실패 이유를 남깁니다(본문은 기록하지 않음). 최근 500건을 `data/mail-deliveries.json`에 보관하며 카드에는 20건을 표시합니다.
  - 같은 설정을 API로도 조작 가능(관리자 세션 필요): `GET/PUT /api/mail`, `POST /api/mail/test`(꺼짐·설정 부족 409, 릴레이 오류 502), `GET /api/mail/deliveries?limit=&status=`
- SMTP 클라이언트는 `node:net`/`node:tls`만으로 직접 구현(EHLO → STARTTLS → AUTH → MAIL/RCPT/DATA, RFC 2047 제목 인코딩, 마침표 겹침)하여 폐쇄망 이미지에 외부 의존성을 더하지 않았습니다.

### 문서
- `docs/ADMIN_GUIDE.md` 9장 **메일 알림 설정 (SMTP 릴레이)** 추가 — 이벤트 표, 설정 항목 표, 시험 발송과 발송 기록, 자주 겪는 문제. 5.6절 API 표에 `/api/mail*` 추가.
- `README.md` — `cutover-data` 볼륨 보관 파일에 `mail.json`, `mail-deliveries.json` 추가.

### 변경 파일
- `lib/mail/` — `config.ts`, `events.ts`, `message.ts`, `smtp.ts`, `store.ts`, `service.ts`, `testing.ts`(가짜 SMTP 릴레이) 및 단위 테스트
- `app/api/mail/route.ts`, `app/api/mail/test/route.ts`, `app/api/mail/deliveries/route.ts`
- `app/api/activities/route.ts` — 저장 전후 diff로 이벤트 감지, `after()`로 배경 발송
- `components/MailSettings.tsx`, `app/admin/page.tsx`
- `e2e/mail.spec.ts`

### 검증
- eslint · tsc 통과, `node --test` 단위 테스트 65건(신규 35건: 설정·이벤트·MIME·가짜 릴레이 실소켓 인증/거절/죽은 릴레이/STARTTLS 강제 실패·서비스 재시도/기록) 통과
- Playwright e2e 8건(신규 3건: 기본 꺼짐이면 아무것도 안 나가고 `mail.json`도 안 생김·401, 가짜 릴레이(AUTH 요구)로 두 명에게 지연 메일 도달·기록 `sent`·GET에 비밀번호 없음·시험 발송 성공, 죽은 포트로 PUT이 즉시 200·`failed` 2회 시도 기록·시험 발송 502) 통과
- Docker 멀티스테이지 빌드 내 Next.js 프로덕션 빌드(`npm run build`) 통과, 이미지에 `/api/mail`·`/api/mail/test`·`/api/mail/deliveries` 라우트 포함 확인
- 실제 사내 릴레이(postra)로의 도착 확인은 빌드 환경에서 불가하여 가짜 릴레이로 대신함

### 업그레이드
데이터 파일 형식은 이전 버전과 같으므로 `cutover-data` 볼륨을 그대로 물려주면 됩니다. 메일 알림은 기본 꺼짐이므로 업그레이드 뒤 동작이 달라지지 않습니다.

```bash
docker load -i cutover-v1.6.0.tar.gz
docker stop cutover-app && docker rm cutover-app
docker run -d --name cutover-app --restart always -p 3000:3000 \
  -e ADMIN_PASSWORD='<관리자-비밀번호>' -e USER_PASSWORD='<사용자-비밀번호>' -e AUTH_SECRET='<기존과-같은-값>' \
  -v cutover-data:/app/data cutover:v1.6.0
```

### 자산
- `cutover-v1.6.0.tar.gz` — `docker save cutover:v1.6.0 | gzip` (node:22-alpine, standalone)
