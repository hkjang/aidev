- 과제: UIDL 성공 + LIST 실패로 사전 oversize 선별이 꺼질 때 진단을 남긴다 (가치 3 / 위험 1 / 작업량 S)

- 왜: `internal/application/sync.go:260` 의 `if listed, lerr := sess.List(ctx); lerr == nil { ... }` 는 `lerr` 를 **통째로 버린다**. UIDL 은 되고 LIST 는 안 되는 서버(실제로 있다 — 이 저장소의 `scriptedMaildrop(..., refuseList=true)` 픽스처가 그 서버를 흉내낸다)에서는 모든 `remote[i].Size` 가 0 으로 남아 `sync.go:312` 의 **사전** 크기 선별(`maxBytes > 0 && rm.Size > maxBytes`)이 통째로 무력화되는데, 작업은 `succeeded` 로 끝나고 어디에도 그 사실이 남지 않는다. 운영자 입장에서는 "크기 제한을 걸었는데 큰 메일을 계속 내려받는다(ingestOne 이 받아 본 뒤에야 거른다)" 가 원인 불명으로 보인다.
- 왜(2): 같은 함수군에 **이미 정답 관용구가 있다** — `sentFolderPass` 의 `sync.go:410/417/422` 는 "동기화를 실패시키지 않는 실패" 를 `slog.Warn("sync: sent folder could not be listed; received mail was synced", "account", acc.ID, "err", providerDiagnostic(err))` 로 남긴다. 이 과제는 그 관용구를 빠진 한 자리에 적용하는 것이다. 정책·동작은 바꾸지 않는다.

- 수용 기준:
  1) UIDL 은 성공하고 LIST 는 `-ERR` 로 거부하는 POP3 서버에 대해 동기화를 돌리면, 사전 크기 선별이 꺼졌다는 사실이 `slog.Warn` 한 줄로 남는다. 메일은 **그대로 다 수집되고**(기존 동작), 잡은 여전히 `succeeded` 로 끝난다 — 실패로 승격하지 말 것.
  2) 로그 한 줄에 메일 서버 원문이 들어가지 않는다. 오류 문자열은 반드시 `providerDiagnostic(lerr)` 를 통과시킨다(프로필의 "진단 문자열은 원문을 담지 않는다" 규약). **읽고 확인함** — `internal/application/provider_diagnostics.go:41-92` 는 고정된 한국어 문장 집합만 돌려주고 `err.Error()` 를 절대 섞지 않는다. POP3 의 `-ERR LIST not available` 은 AuthError·timeout·PublicError 가 아니므로 `default:` 로 떨어져 `providerFailed` 상수 한 문장이 된다 — 즉 원문 누출은 구조적으로 없고, 대신 **분류 정보도 사라진다**. 그래서 선택: `"class"` 키에 `domain.ClassifyInbound(lerr)`(또는 `errors.As` 로 꺼낸 `InboundError.Class`/`.Code`)를 **추가로** 붙이면 운영자에게 쓸모가 생긴다 — 이 둘은 레이블·RFC 코드만 담도록 설계된 필드다. 착수 시 `internal/domain/inbound_error.go` 에서 `ClassifyInbound` 의 시그니처를 확인할 것(이름만 프로필에서 확인했고 서명은 **미확인**).
  3) LIST 가 **성공하는** 경로에서는 그 경고가 나오지 않는다 — 테스트가 양쪽(거부/성공)을 모두 단언해, 수정이 조건 없이 항상 경고를 뱉는 회귀를 잡는다.
  4) `domain.SyncDiagnostic`(`internal/domain/job.go:158`) 에 **필드를 추가하지 않는다**. 추가하면 `cmd/postra-contracts` 생성물·`web/`·`internal/transport/spa/assets` 까지 번져 S 과제가 아니게 된다. 이번 회차는 `slog` 한 줄로 끝낸다.

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `internal/application/sync.go:258-269` (`runSync` 의 UIDL 성공 분기) — `lerr == nil` 분기에 `else`(또는 `if lerr != nil`)를 붙여 `slog.Warn` 한 줄을 남긴다. 문구는 영어 + 기존 관용구 어투로, 예: `slog.Warn("sync: LIST failed; message sizes unknown and the pre-fetch size screen is off for this sync", "account", acc.ID, "err", providerDiagnostic(lerr))`. `slog` 는 이 파일에 이미 import 돼 있다(`sync.go:71` 등). **버린 `lerr` 를 반환하거나 잡을 실패시키지 말 것.**
  - `internal/application/sync_list_sizes_test.go` (새 파일, 패키지 `application`) — 같은 패키지의 기존 헬퍼를 **그대로 재사용**한다: `scriptedMaildrop(t, mails, refuseList)` 와 `maildropAccount(t, app, port)` (둘 다 `internal/application/sync_unlimited_size_test.go:25,98`). 로그 포착은 이 저장소에 이미 있는 관용구를 쓴다 — `internal/application/oidc_errors_test.go:89-90` 의 `previous := slog.Default(); slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil))); defer slog.SetDefault(previous)`. `internal/application` 에 `t.Parallel()` 을 쓰는 테스트는 **0건**(확인함)이므로 전역 default logger 교체가 안전하다.
  - 하위 테스트 2개: `refuseList=true` → 경고 1줄 존재 + 메일이 전부 수집됨(`Search` 또는 stats) + `status=succeeded`; `refuseList=false` → 그 경고 **없음** + 동일하게 수집 성공.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go build ./... && go vet ./...`
  - `go test -race -count=1 -timeout=600s ./internal/application/` — 이전 회차 실측 약 83초(-count=3 은 약 246초). 짧게 보려면 `-run 'TestSync'` 로 좁힐 수 있다.
  - `go test -race -count=1 -timeout=900s ./...` (전체, 수 분)
  - `make lint-format` (gofmt, 빈 출력이어야 함), `make lint-security` (gosec Files 141 Issues 0)
  - `go run ./cmd/postra-contracts -check` (exit 0 — 계약이 안 바뀌었음을 증명)
  - `git status --porcelain internal/transport/spa/assets` 가 **비어 있을 것**(프런트 미변경이므로 자산 재빌드 불필요)
  - `git diff --check`

- 위험과 피할 것:
  - **동작을 바꾸지 말 것.** LIST 실패를 잡 실패로 올리거나(`syncDiagnosticStage`+`finish(JobFailed, ...)`) 사후 oversize 정책을 손대면 위험이 뛴다. `ingestOne`(sync.go:544 부근)·`rawReadLimit`·`maxBytes > 0` 비교·`sync.max_message_bytes` 의 `<=0 = 무제한` 규약(v0.25.4 에서 고친 자리)은 **건드리지 말 것**.
  - `internal/adapters/pop3/retrBody` 는 PR #22 / 04b15be 영역이고 사람 반려 여부가 아직 **미확인**(`gh` 미인증)이다 — 이번 회차에 열지 말 것.
  - 보호 경로 회피: OIDC·session·MCP OAuth·DB migrations·SecretStore/KEK·`.github/workflows`·`internal/transport/spa/assets` 는 이 과제에서 전혀 필요하지 않다.
  - 테스트 함정: `slog.SetDefault` 는 전역이므로 **반드시 `defer` 로 원복**할 것. 로그 단언은 전체 문장 일치가 아니라 안정적인 부분 문자열(예: `"pre-fetch size screen"`)로 하고, `err` 값에 서버 원문(`LIST not available`)이 **들어 있지 않음**도 함께 단언하면 수용 기준 2)가 테스트로 묶인다.
  - 변이 검증(권장): `slog.Warn` 호출을 지우면 `refuseList=true` 하위 테스트만 실패하고 `refuseList=false` 는 통과함을 확인한 뒤 원복.
  - 추측으로 적은 것(구현자가 확인할 것): ① `domain.ClassifyInbound` 의 시그니처는 **미확인**(이름만 프로필 기록) — 쓰기 전에 `internal/domain/inbound_error.go` 를 열 것. 안 맞으면 `providerDiagnostic` 만으로 끝내도 수용 기준은 충족된다. ② `scriptedMaildrop`/`maildropAccount` 는 grep 으로 시그니처·줄번호만 확인했고 본문은 읽지 않았다(반환이 포트 `int` 인 것까지만 확인). ③ 로그 포착 테스트가 `StartSync` 의 **비동기 워커** 안에서 나오는 경고를 보려면 기존 테스트들이 잡 완료를 기다리는 방식(`sync_unlimited_size_test.go` 의 대기 로직)을 그대로 따라야 한다 — 그 대기 코드는 읽지 않았다.

- 차선 후보: `Makefile: frontend-test` 가 의존성 설치를 보장하게 한다 (가치 3 / 위험 1 / 작업량 S) — `Makefile:11` 의 `frontend` 타깃만 `npm ci` 를 하고 `Makefile:15` 의 `frontend-test` 는 typecheck+test 만 해서, 깨끗한 워크트리에서 `make frontend-test` 가 성립하지 않는다. `node_modules` 가드 또는 `npm ci` 선행을 넣고, `rm -rf web/node_modules && make frontend-test` 가 exit 0 인 것으로 검증한다(npm ci 때문에 느림). 프로덕션 코드 0개, 파일 1개.
