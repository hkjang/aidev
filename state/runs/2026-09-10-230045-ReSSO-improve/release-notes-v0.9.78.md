**이 저장소에는 화면을 쓰는 사람을 위한 문서가 한 줄도, 화면 캡처가 한 장도 없었습니다.** `docs/operations.md`·`compatibility.md`·`user-federation.md`가 있었지만 셋 다 **이미 아는 사람이 찾아 읽는 참고서**입니다 — 키 회전 절차, OIDC 사양 준수 범위, LDAP 속성 매핑. 처음 로그인 화면 앞에 앉은 사람도, ReSSO를 처음 설치하는 사람도 읽을 것이 README뿐이었고, README는 개발·연동 참고 문서라 **어느 화면의 어느 버튼인지는 어디에도 없었습니다.**

### 추가

- **`docs/USER_GUIDE.md`(사용자 가이드)와 `docs/ADMIN_GUIDE.md`(관리자 가이드), 그리고 두 PDF를 더했습니다.** 사용자 가이드는 로그인·비밀번호 변경·내 세션 끊기·개인 API 키까지, 관리자 가이드는 오프라인 설치·환경 변수·Realm과 Client 등록·User Federation·승인 절차·감사 로그·백업과 되돌리기까지를 다룹니다.
- **화면 캡처 19장은 그린 것이 아니라 실제로 띄워 찍은 것입니다.** v0.9.77 바이너리를 빈 PostgreSQL 위에 올리고, 새 스크립트 `scripts/guide-screenshots.mjs`가 데모 데이터를 REST API로 심은 뒤 headless Chrome을 CDP로 몰아 1440x900에서 찍습니다(외부 의존성 없이 Node 22의 내장 WebSocket으로 붙습니다).
- **문서에 적힌 사실은 모두 코드에서 읽었습니다.** 환경 변수 표는 `internal/config`가 실제로 읽는 전부이고, 안내한 Endpoint는 메서드까지 라우트 등록 자리에서 확인했으며(`/metrics`는 GET이지만 관리 권한이 필요하고 `/mcp`는 POST 전용입니다), 「막혔을 때」 표의 문구는 `LoginPage.tsx`와 `auth.go`에 있는 그대로입니다. 권한 모델도 마찬가지입니다 — **서비스 관리자는 API로도 화면으로도 부여할 수 없고**(`BOOTSTRAP_ADMIN` 최초 생성과 `admin recover`만 세웁니다), Realm 관리자는 `realm-admin` Role입니다.
- **캡처 스크립트는 남의 배포에 닿지 않습니다.** 대상은 다른 스크립트와 공유하지 않는 전용 변수 `RESSO_GUIDE_URL`로만 받고 **기본값이 없으며, loopback이 아니면 그 자리에서 멈춥니다** — 이 스크립트는 Realm·사용자·Client·API 키·승인 요청을 만들어 넣으므로 잘못 겨누면 감사 트레일에 그대로 남습니다. 전역 설정을 덮어쓰지 않고 새 객체만 만들며, 유일하게 건드리는 기존 레코드는 방금 만든 Realm의 승인 절차 스위치 하나입니다.
- **화면에 실명·실제 주소·실제 비밀값은 없습니다** — `홍길동`/`hong.gildong@example.com`, `데모 회사`, `sso.example.com`, `ldaps://ad.example.com:636`이고, API 키 Prefix는 버릴 인스턴스에서 방금 발급된 값입니다.
- **정본은 하나입니다.** README 맨 위에 두 문서(+PDF) 표를 더해 README가 그 요약이 아니라 개발·연동 참고용임을 밝혔고, 두 가이드는 `operations.md`·`compatibility.md`·`user-federation.md`를 **가리키기만 하고 겹쳐 쓰지 않습니다.**
- 코드는 **한 줄도 바뀌지 않았습니다.** 이번 릴리즈가 담은 것은 문서와 캡처, 그리고 그 캡처를 만드는 스크립트뿐입니다.

### Upgrade notes

**동작이 달라지는 곳은 없습니다.** API도, 화면도, 상태 코드도, 저장되는 데이터도 v0.9.77과 같고 **마이그레이션도 설정 변경도 없습니다.** 새 이미지를 받지 않아도 두 가이드는 저장소에서 그대로 읽을 수 있습니다. 두 가이드는 **v0.9.77 화면을 찍은 것이라 그렇게 적혀 있는데, 이번 릴리즈가 화면을 바꾸지 않았으므로 v0.9.78에서도 그대로 맞습니다.** 지원 창구와 신규 관리자에게는 README 대신 `docs/USER_GUIDE.md`·`docs/ADMIN_GUIDE.md`(또는 같은 이름의 PDF)를 안내하시면 됩니다.

