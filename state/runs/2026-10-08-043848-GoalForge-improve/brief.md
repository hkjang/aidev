- 과제: 실패한 웹훅 전송이 반복 억제 창을 소비하지 않게 한다 (가치 4 / 위험 2 / 작업량 M)
- 왜: `internal/notify/notify.go:Post`는 HTTP 요청 전에 `suppressor.allow`로 기록을 남겨 첫 전송이 HTTP 500·요청 생성 오류·네트워크 오류로 실패해도 같은 알림을 기본 30분간 숨긴다. 실패한 호출의 기록만 해제하면 일시 장애 뒤 다음 호출이 다시 전송을 시도하고, 성공한 알림의 반복 억제는 유지된다.
- 수용 기준:
  1) `httptest.NewServer`가 첫 요청에 500, 다음 요청에 200을 반환할 때 동일 Event의 첫 Post는 오류, 두 번째 Post는 성공이며 서버 요청 수는 2다. 세 번째 동일 Post는 nil을 반환하되 요청 수는 2로 유지된다. 양수 반복 창을 명시하고 창 경과를 기다리지 않는다.
  2) 요청 생성 실패와 HTTP 전송 실패에서도 해당 예약이 해제된다. 잘못된 URL→유효한 로컬 서버 및 취소된 context→새 context 같은 결정적 사례를 사용하고 실제 외부 서비스는 호출하지 않는다. URL을 키에 추가하여 첫 사례만 우회하는 수정은 불충분하다.
  3) 동시에 같은 이벤트를 보낼 때 기존 반복 억제를 유지한다. 오래된 요청의 늦은 실패가 같은 키의 새 예약을 지우면 안 된다. 가짜 시계로 예약 A→창 만료→예약 B→A 실패 해제→B 여전히 억제됨을 직접 검증한다. HTTP I/O 동안 mutex를 잡지 않는다.
  4) 성공 후 창 만료 재전송·변경된 Reason 즉시 전송·0 창에서 억제 안 함·민감값 삭제·미설정 URL no-op이 유지된다. 기존 테스트의 전역 억제기 상태도 격리하여 `go test ./internal/notify -count=2`가 통과해야 한다.
- 건드릴 파일:
  - `internal/notify/notify.go:Post` (현재 47행부터): 예약을 받고 전송 성공이 확정되지 않은 반환 경로에서 그 예약만 해제한다. 기존 5초 timeout, 상태 코드 판단, 오류 반환, redaction, 공개 Post 시그니처는 유지한다.
  - `internal/notify/suppress.go:suppressor, allow` (현재 19/30행부터): 내부 예약에 고유 식별자 또는 참조를 주고 mutex 아래에서 소유자가 일치할 때만 해제하는 작은 API를 마련한다. 현재 성공 기록의 시각 기준(전송 시작), 만료 정리 및 0 창 동작은 유지한다. 단순 무조건 delete나 시각만으로 소유자를 구별하는 방식은 피한다.
  - `internal/notify/notify_test.go:TestPostSendsRedactedSlackCompatiblePayload, TestPostIsNoOpWithoutURLAndSurfacesServerErrors` 및 신규 회귀: 매 테스트 독립 억제기를 설치하고 t.Cleanup으로 복원한다. 전역 변경 테스트는 parallel 금지. 실패→성공→억제와 오류 경로를 실제 Post로 확인한다.
  - `internal/notify/suppress_test.go`: 예약 API에 맞춰 기존 테스트를 최소 조정하고 오래된 실패가 새 예약을 지우지 않는 회귀를 추가한다. 시간 대기 대신 기존 now 주입을 사용한다.
  - 프로덕션 2개 + 테스트 2개, 이 범위 밖 확장 금지.
- 검증 명령: 저장소 루트에서 `go test ./internal/notify -count=1`; `go test ./internal/notify -count=2`; `go test -race ./internal/notify -count=1`; `go test ./... -count=1`; 구현 후 `go vet ./...`, `go build ./...`, `gofmt -l ./internal/notify`.
- 위험과 피할 것: auth·session·migrations·workflows·SQLite 호출부를 건드리지 않는다. 자체 재시도 루프, backoff, 지속 큐, 새 설정, 외부 의존성, 전송 완료 보장은 범위 밖이다. 완료 전이가 다시 발생하지 않으면 이번 수정만으로 알림이 재발송되지는 않는다. 모든 실패의 원격 미수신을 보장할 수도 없다(응답 유실 가능). 오류가 해제된 후 다음 Post가 시도할 수 있게 하는 것까지만 약속한다. nil 반환은 지금처럼 성공 또는 억제/no-op일 수 있다. 기존 redaction 전후 순서와 키 의미는 유지한다. 테스트를 창 0으로 모두 돌려 억제 결함을 숨기지 않는다.
- 차선 후보: MCP `activity_report`의 `since` 설명에서 지원하지 않는 `7d` 예시를 `168h`로 교정하고 실제 tools/list→tools/call 계약 테스트를 붙인다 (가치 2 / 위험 1 / S). 직접 확인한 `internal/mcp/tools.go:toolDescriptors`의 설명(73행)은 7d를 안내하지만 `activityReport`(654행)는 time.ParseDuration만 쓴다. `internal/mcp/server_test.go:TestStdioProtocolLifecycle`, `fixture`를 재사용하고 기간 문법 확장·파서 통합은 하지 않는다. 이 예시 오류는 소스 확인, 실행 재현은 미확인.

실행 계획 (구현 전부 pending; 정찰은 구현하지 않음):
1. 테스트 격리와 실패 전송 회귀를 먼저 작성한다. 증명: `go test ./internal/notify -count=2` 및 첫 수용 기준 테스트. 기존 코드에서 회귀가 실패함을 기록한 후 다음 단계로 진행한다. 체크포인트는 자동 결과 확인이며 사람 승인을 기다리지 않는다. 테스트 격리만으로 새 회귀가 통과한다면 원인 가정을 다시 확인하고 이 과제서를 수정한다.
2. 위 두 프로덕션 파일에서 예약의 소유권을 확인하는 실패 해제를 구현하고 오래된 예약 해제 테스트를 추가한다. 증명: `go test ./internal/notify -count=2`와 `go test -race ./internal/notify -count=1`. 둘 다 통과한 후 진행한다. 별도 스레드·범용 예약 프레임워크는 만들지 않는다.
3. 전체 suite/build/vet/format을 확인하고 변경 경로가 네 파일 이내인지 확인한다. 기존 환경 실패는 수정 결과와 구분하고 skip 추가로 숨기지 않는다. 정찰은 여기까지의 구현 완료를 주장하지 않는다.

근거와 현재 검증:
- main@2cbfeed, 작업 트리 무변경에서 `go test ./internal/notify -count=1` 통과(0.007s), `go test -race ./internal/notify -count=1` 통과(1.022s).
- `go test ./internal/notify -count=2`는 실제 exit 1. 두 번째 반복에서 `notify_test.go:32 payload={Project: Name: State: Reason: Text:}`와 `notify_test.go:53 expected error for 500 response`가 발생했다. 성공 및 실패 기록이 패키지 전역에 남는다는 실행 근거다. 500→200 전용 회귀와 동시 요청 재현은 아직 작성/실행하지 않았으므로 미확인이다.
- `notify.Post` 호출 검색에서 cmd/goalforge/main.go 및 sqlite의 verification/planner/recovery 호출이 오류를 `_ =`로 무시하는 것을 확인했다. 실제 워커 장애 및 사용자 알림 누락 사례는 관측하지 않았다.
- 전체 suite 결과는 이 파일 끝의 최종 검증 메모 참조. Windows/macOS 직접 실행, 외부 웹훅 실송신은 미확인.

대안 판단:
- 선택: 기존 억제 예약을 유지하되 실패 때 자기 예약만 해제. 두 파일 안에서 수신 실패 이후의 재호출을 복구하고 중복 동시 발송도 제한한다.
- 성공 후에만 기록: 간단하지만 전송 중 같은 이벤트를 여러 번 허용한다. 기존 동시 억제 의미를 보존해야 하므로 제외한다.
- 큐·재시도 서비스: 단발 전이까지 복구할 수 있으나 새 상태/재시도 정책/중복 수신 문제로 45분 범위를 벗어난다.
- 현상 유지 및 문서화: best-effort 성격은 설명하나 다음 호출까지 30분 막는 확인된 경로를 그대로 두므로 우선순위가 낮다.
- 핵심 가정: 실패 후 같은 이벤트를 다시 호출하는 경로에서 이익이 생긴다. 이 과제는 '최종 전달 보장' 과제가 아니다.

작업량 근거 (pmo:estimating-and-contingency 적용):
- bottom-up 기본 추정: 테스트 격리/회귀 8–10분 + 예약/해제 10–13분 + 동시성/기존 계약 검사 7–9분 + 전체 검증/검토 5–7분 = 30–39분. 실측 일정이 아닌 정찰자의 중간 확신 추정이며 통계적 80% 등은 주장하지 않는다.
- 알려진 불확실성 예비 3–6분: 오래된 요청 해제 테스트 및 race 실행 환경. 합계 33–45분. 단계별 기본값에 예비를 중복 가산하지 않았다. 미지 범위에 대한 management reserve는 이번 회차에 배정하지 않는다. 새 전달 보장 요구는 별도 과제로 남긴다.
- 유사 추정: 10/06의 작은 상태 계산 수정+실제 경로 회귀와 범위는 비슷하지만 이번은 동시성 때문에 M으로 상향했다. 과거 소요 시간 자료가 없으므로 이를 독립된 수치 검증으로 주장하지 않는다. 예약 API 초안과 첫 회귀 이후 다시 추정한다.
- 적용 절차 출처: 로컬 `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/{pmo/skills/estimating-and-contingency,technology/skills/implementation-planning,technology/skills/solution-exploration}/SKILL.md`를 직접 읽었다(Skill 도구 미제공). pmo references/sources.md도 읽었으며 외부 비용 산정 규범에 근거한 정량 신뢰도는 주장하지 않는다.

최종 검증 메모 (정찰): `go test ./... -count=1` exit 0, 32개 테스트 패키지 통과, procctl/testscript는 [no test files]. sqlite 46.984s, observer 24.243s. skip 개수는 -v/-json이 아니므로 미확인. remote.origin.pushurl=DISABLED는 직접 확인했다. 기존 notify -count=2 실패는 별도로 남아 있다. build/vet는 CI에 명시된 구현 후 검증 명령이며 이번 정찰에서는 실행하지 않았다. git status --short는 무출력이다.
