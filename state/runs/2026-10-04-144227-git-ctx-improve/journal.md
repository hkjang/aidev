# 회차 노트 2026-10-04-144227-git-ctx-improve — git-ctx
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:42] base pinned — main@d198fb3
- [러너 14:42] autonomy release — 

## 정찰 노트
- 보류 1순위([3/2/M] 알림 발송 대기)를 골랐다. 근거를 확인해 가치를 4로 올렸다: `notification.New`(service.go:76)의 `tick: 10*time.Second`는 2026-10-03에 성공한 워커 폴링과 **같은 모양**이고, 두 수혜 테스트가 재시도 백오프에 의존하지 않음을 소스로 확인했다(포기 쪽 `live_integration_test.go:593`이 `maxAttempts:1`, 배달 쪽 `:100`은 첫 시도 성공). 차선인 `fencesContent` 허용목록은 가치 3이지만 동작 변화가 없어 증명할 테스트가 약해 제쳤고, 프로브 테스트 후보는 원인 미특정이라 제쳤다.
- 과제서에서 추측으로 적은 것: ① 리스 만료 창 5분(`service.go:101`)이 25ms 틱에서도 안전하다는 추론은 소스 기반이고 실측하지 않았다 ② 두 테스트 10.31s의 "사실상 전부"가 알림 대기라는 것은 2026-10-03 회차의 실측값과 10초 틱이 맞물린 추정이다 — 그래서 수용 기준 5에 되돌림 프로브를 넣었다 ③ 이 세션에서 `go test`를 돌리지 않았다(시간·예산).
- 구현자가 조심할 것: 25ms 틱을 7개 픽스처 전부에 주지 말 것(`RunOnce`가 매 틱 설정 행을 읽어 공유 in-memory SQLite에 잡음을 심는다 — 알림을 기다리는 `:65`·`:563` 두 곳만). setter는 `a.wg.Add(4)` 앞에 둘 것. `internal/app` 시간은 같은 커밋에서도 흔들리므로 전후 숫자만으로 성과를 주장하지 말 것.
- 프로필은 기준 커밋이 낡아(f1de053/v0.77.20 → d198fb3/v0.77.23) 새로 썼다. "테스트 전용 타이밍 주입 관례" 항목을 추가했다.
- [러너 14:47] scout done — 알림 발송 루프의 10초 고정 틱을 주입 가능하게 해서 `internal/app` 테스트의 남은 벽시계를 줄이기 (가치 4 /

## 구현 노트
- 바꾼 것: `notification.New` 의 하드코딩 `tick: 10*time.Second` 를 `DefaultTickInterval` 상수 + `SetTickInterval`(비양수 무시)로 열고, `config.Config.NotificationTickInterval`(`FromEnv` 무수정) → `startBackground` 한 줄로 주입했다. 2026-10-03 의 `WorkerPollInterval` 과 같은 관례이고 프로덕션 기본값은 값 그대로 10초다. 테스트 쪽은 알림을 기다리는 두 픽스처에만 25ms 를 넘긴다.
- 확신 없는 곳: `TestPlatformDegradationIntegration` 이 3.68s 로 남은 이유를 끝까지 파지 않았다 — 알림 틱은 분명히 빠졌는데(되돌림 프로브로 10.37s 복귀 확인) 남은 3초대가 실패 수신기 쪽 대기인지 색인인지 특정하지 않았다. 또 리스 만료 창 5분이 25ms 틱에서 안전하다는 것은 전 테스트 통과(`-race` 포함)로만 뒷받침했고, `sending` 행이 조기 회수되는지 직접 관측하는 테스트를 쓰지는 않았다.
- 검증 못 한 것: 외부 DB(postgres/pgvector)·Vault·Docker·실브라우저·`govulncheck` 는 돌리지 않았다. `internal/notification/postgres_integration_test.go` 는 DSN 이 없어 SKIP 된다.
- 일부러 하지 않은 것: 재시도 백오프(`service.go:164`)·`MaxAttempts` 검증·리스 만료 창·`Run`/`RunOnce` 본문·`waitFor` 의 90초 상한과 200ms 폴링은 건드리지 않았다. 알림을 기다리지 않는 나머지 5개 픽스처와 `app_test.go` 에는 25ms 를 넘기지 않았다 — `RunOnce` 가 매 틱 설정 행을 읽어 공유 in-memory SQLite 부하 테스트에 잡음이 되기 때문이고, 이 이유는 `testNotificationTick` 주석에 남겼다. 버전·릴리즈 노트·`.github/` 는 무수정(`git diff --stat .github/` 빈 출력).
- 다음 역할이 조심할 것: `internal/app` 시간 주장은 반드시 같은 세션 숫자로만 비교할 것(이 세션 기준선은 50.2s/50.5s, 수정 후 34.47s/34.47s — 과거 회차의 73s·100~117s 와 섞지 말 것). 새 테스트 `TestTheConfiguredNotificationTickReachesTheNotificationLoop` 는 `t.Parallel()` 을 쓰지 않으며 2초 데드라인이 프로덕션 기본 10초의 1/5 이라 배선이 빠지면 바로 red 가 된다 — 느린 머신에서 이 2초를 늘려야 한다면 10초보다 충분히 작게 유지해야 테스트가 의미를 잃지 않는다.
- [러너 14:56] brief accepted — 채택 — 과제서가 지목한 줄(`service.go:76`·`:79-91`, `app.go:359`·`:368`·`:383`, `config.go:26-36`, `live_integration_test.go:65`·`:563`→실제
- [러너 14:56] improve no-change — 커밋 없음
