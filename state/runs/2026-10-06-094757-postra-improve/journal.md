# 회차 노트 2026-10-06-094757-postra-improve — postra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 09:48] base pinned — main@e6d2de0
- [러너 09:48] autonomy release — 

## 정찰 노트
- 고른 이유: STARTTLS 주입 가드의 거부만 평문 error 라 `sync_diagnostics.go:584` 의 fallback 때문에 `tcp_connect` 로 보고된다 — 이 저장소가 이미 세 번 머지한 유형(elapsed 0, 동사 오기, 버려진 lerr)과 같은 결함이고 필요한 Stage·Class 레이블과 테스트 픽스처가 양쪽 패키지에 **전부 이미 있어** 프로덕션 2줄 수준이다. retrBody 계열(가치 4)은 PR #22 반려 여부가 여전히 미확인이라 규칙대로 제외했고, 차선이던 implicit TLS 회귀 테스트는 프로덕션 결함을 고치지 않아 뒤로 밀었다.
- 확신 없는 곳: `./internal/adapters/imap/` 테스트 소요 시간을 실측하지 않았다(과제서에 미확인으로 적음). 진단이 실제로 "TCP 연결" 로 렌더되는 것을 end-to-end(`StartSync`→진단 문자열)로 돌려 보지는 않았다 — `syncDiagnosticStage` 의 fallback 경로를 코드로만 확인했다. 가드 거부에 가장 맞는 `Class` 가 `"other"` 인지는 판단이고(새 레이블 금지가 제약), 등록 여부만 확인했다.
- 구현자가 조심할 것: 가드의 판정 로직·`conn.Close()`·업그레이드 중단을 바꾸지 말 것(보안 경로, 거부 동작은 불변). `domain.SyncDiagnostic` 에 필드를 추가하지 말 것 — 계약·web·spa/assets 로 번진다. POP3·IMAP 양쪽을 고치되 IMAP 이 커지면 POP3 만 담고 쪼갤 것.
- [러너 09:53] scout done — STARTTLS 주입 거부가 `tcp_connect` 단계로 잘못 보고되는 것을 `starttls` 단계로 고정 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- POP3 `client.go:115-118`·IMAP `client.go:112-115` 의 STARTTLS 주입 가드가 돌려주던 평문 `fmt.Errorf` 를 `&domain.InboundError{Stage: StageStartTLS, Command: STLS/STARTTLS, Class: ClassifyInbound(refused), Elapsed, Timeout: commandTO}` 로 감쌌다 — 그래야 `syncDiagnosticStage` 의 `errors.As` 가 잡고 dial fallback(`tcp_connect`)으로 떨어지지 않는다. 가드의 판정·`conn.Close()`·업그레이드 중단은 불변.
- 확신 없는 곳: `StartSync` → 실제 진단 문자열까지의 end-to-end 렌더는 **돌리지 않았다**(주입 서버를 띄우는 application 레벨 픽스처가 없다). 어댑터 단언 + `syncDiagnosticStage`/`syncStageLabels`/`syncClassLabels` 코드 확인으로 연결을 논증했고, `./internal/application/` 전체(82s)는 회귀 없이 통과한다. `Class == "other"` 가 이 거부에 가장 맞는 분류인지는 여전히 판단이다(새 레이블 금지가 제약) — 등록돼 있어 `allowed()` 가 버리지 않는 것만 테스트로 못 박았다.
- 일부러 하지 않은 것: `domain.SyncDiagnostic` 필드 추가 없음(계약·web·assets 로 번진다), 새 Stage/Class 레이블 없음, 새 테스트 픽스처·새 파일 없음(기존 테스트에 단언만 추가), `retrBody`/`rawReadLimit`/설정 파싱/워크플로/`spa/assets`/`web/` 미접촉. PR #22 영역도 미접촉(반려 여부 미확인).
- 다음 역할이 조심할 것: 프런트 미변경이라 자산 재빌드 불필요(`spa/assets` 드리프트 0 확인). 외부 PostgreSQL(`POSTRA_TEST_PG`)·브라우저 e2e 는 돌리지 않았다. 실측 소요: pop3 5.0s / imap 4.3s(지금까지 미실측이던 값) / application 82.4s / 전체 race 실패 0.
- [러너 10:01] brief accepted — 채택 — 근거가 전부 코드와 맞았다(가드 두 곳만 평문 error, 바로 다음 줄은 `StageTLS` 로 감쌈, `syncDiagnosticStage` 의 fallback, 
- [러너 10:02] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인한 것: 수정 전 두 가드가 `%w` 없는 새 `fmt.Errorf` 를 돌려주므로 추가된 `errors.As(err, &inbound)` 단언은 수정 전에 통과할 수 없다 — 원장에 `- 실패 재현:` 줄이 없어 diff 로 직접 확인했고, 테스트는 기존 스크립트 서버 + 실제 Dialer 픽스처를 그대로 지난다.
- 구현자가 못 돌린 end-to-end 렌더를 코드로 끝까지 추적했다: `syncDiagnosticStage`(:166) → `safeSyncDiagnostic`(:362) 에서 `starttls`(:126)·`other`(:150) 등록 확인, `allowedCommand`(:406) 가 "STLS"/"STARTTLS" 통과. 또 `docs/SYNC_DIAGNOSTICS.md:28` 이 이미 이 단계를 '핸드셰이크 전 평문 데이터' 로 문서화했다 — 문서가 맞고 코드가 틀렸던 자리다.
- 부수 효과: `retryableInbound`(:532) 는 전후 모두 `other`→재시도 true 라 변화 없음. `syncHint`(:318) 의 `connecting` 이 더는 맞지 않아 엉뚱했던 'IDLE 끄기' 힌트가 사라진다(개선). 보안·법무 차단 사유 없음 — 가드 판정식·`conn.Close()`·에러에 바이트 수만 담는 규약 모두 불변.
- 승인이어도 남는 우려(릴리즈 노트용): 가장 보안 관련성 높은 거부가 class `other`("알 수 없는 오류") 를 일반 오류와 공유해 단계로만 구분된다 — 전용 라벨은 syncClassLabels·docs·web 까지 번지니 별도 회차. Elapsed 는 명령 왕복부터 재어 아래 StageTLS 분기와 기준이 다르다(두 어댑터는 서로 일치, 결함 아님).
- 돌린 검증: gofmt 0건 / go build ./... / go vet / `go test -race` pop3 5.1s·imap 4.4s·application 79.9s 전부 ok. 못 본 것: 전체 race 스위트, `POSTRA_TEST_PG`, 브라우저 e2e, gosec.
- [러너 10:07] review approved — 리뷰 승인 (risk=low)
- [러너 10:07] pr created — https://github.com/hkjang/postra/pull/36
