- 과제: IMAP 열거 실패 뒤 빈 목록·부분 인덱스를 정상 캐시로 반환하지 않게 한다 (가치 4 / 위험 2 / 작업량 M)
- 왜: `ensureIndex`가 FETCH 성공 전에 `indexed=true`를 설정하고 배치마다 `s.index`에 누적하므로, 첫 호출이 실패해도 다음 `List`는 빈 목록이나 부분 목록을 성공으로 반환한다. `runSync`의 실제 `UIDL → List` 폴백이 이 경로를 지나므로, 전체 열거에 성공한 결과만 캐시해 메일 누락을 정상 동기화로 오인하는 일을 막는다.
- 수용 기준:
  1) 첫 FETCH 또는 후속 배치가 실패하면 열거 완료 캐시가 생기지 않는다. 같은 세션의 다음 UIDL/List는 첫 배치부터 다시 시도하고, 다시 실패하면 오류를 반환하며 부분 결과를 성공으로 노출하지 않는다.
  2) 2001개 메일 중 1:2000 성공 → 2001:2001 실패 → 재시도 성공 시 정확히 2001개의 중복 없는 번호·UIDL·크기를 반환한다. 이후 UIDL/List는 추가 FETCH 없이 성공 캐시를 사용하며 빈 메일함도 정상 동작한다.
  3) 실제 TCP IMAP 서버 + 프로덕션 `imap.Dialer{}` + `CreateAccount → StartSync → GetJob` 경로에서 첫 메타데이터 FETCH를 계속 거절하면 최종 작업은 `failed`, 진단 단계는 `enumerate`가 된다. 기존 코드가 `succeeded`와 빈 목록으로 끝나는지 먼저 재현한다. 서버 오류에 넣은 표식 문자열은 저장 작업 오류·진단에 없어야 한다.
- 건드릴 파일:
  - `internal/adapters/imap/client.go:ensureIndex`(약 515행): 로컬 임시 `[]domain.RemoteMessage`에 전 배치 결과를 모은 뒤에만 `s.index`와 `s.indexed`를 커밋한다. 실패는 기존 오류를 그대로 반환한다. `exists==0`의 정상 완료도 캐시한다. 단순히 indexed 설정만 끝으로 옮기면 재시도 때 앞 배치가 중복 누적되므로 부족하다.
  - `internal/adapters/imap/client_test.go`: 기존 `batchServer`, `dial`/`dialMax`, `TestIMAPFullEnumerateBatchStaysUnderResponseBound`를 참고해 실패→재시도·성공 캐시 테스트를 추가한다. 기존 batchServer는 요청 범위를 무시하고 모든 행을 매번 보내므로 2001개 테스트에 그대로 사용하지 말고 요청 범위대로 응답하는 작은 전용 서버를 둔다.
  - `internal/application/imap_sync_test.go:TestSyncViaIMAP` 인근: 기존 fakeInbound 테스트는 유지하고, 실제 TCP 서버를 쓰는 별도 회귀 테스트를 추가한다. `newTestApp`을 만든 뒤 `app.IMAP = imap.Dialer{}`를 연결하고 InboundProtocol="imap", POP3Host="127.0.0.1", POP3Port=리스너 포트, POP3Security="none" 계정을 만든다. Secret 등록·CreateAccount 인자는 기존 테스트를 따르고 `syncAndWait`로 최종 상태를 읽는다. 손으로 만든 InboundSession 대역으로 새 회귀를 증명하지 않는다.
  - 프로덕션 변경은 1파일, 테스트 포함 총 3파일을 목표로 한다. application/sync.go와 app.go는 읽기 근거이며 수정 대상이 아니다.
- 검증 명령:
  - `go test -race -count=1 -timeout=180s ./internal/adapters/imap/ ./internal/adapters/pop3/`
  - `go test -race -count=1 -timeout=180s ./internal/application -run '^(TestSyncViaIMAP|TestSyncFailureNamesTheStepAndKeepsNoServerText|TestSyncWithFailedFetchesIsNotReportedAsSuccess)$'`
  - 새 테스트까지 포함한 최종 검사: `go test -race -count=1 -timeout=900s ./internal/application ./internal/adapters/imap ./internal/adapters/pop3`
  - `make lint-format` / `go run ./cmd/postra-contracts -check` / `git diff --check`
  - 정찰 실측: 첫 명령 IMAP 4.450초·POP3 5.126초 PASS, 둘째 명령 application 2.092초 PASS, 마지막 포맷·계약·diff 검사 exit 0. 최종 패키지 전체 명령은 이번 정찰에서 미실행이며 application은 이전 회차 약 82초였다. 새 실패 재현·수정 후 결과는 구현자 검증 사항이다.
- 위험과 피할 것: POP3 UIDL 미지원 폴백 정책 자체를 바꾸지 않는다. IMAP의 enumerateBatch=2000·프레이밍 상한·abandon/broken·IDLE 연결 수·UIDVALIDITY·폴더 전환 로직·원문 진단 정책을 함께 고치지 않는다. 캐시와 broken 상태를 임의 초기화해 끊긴 세션을 살리지 않는다. auth/session/migrations/.github/workflows/계약/web/spa/assets는 범위 밖이다. PR #22의 POP3 retrBody 접근은 반려 여부 미확인이므로 건드리지 않는다. 테스트 서버 스레드의 명령 기록은 채널/뮤텍스로 읽고 cleanup 전에 결과를 단언한다.
- 차선 후보: POP3·IMAP greeting 실패 진단의 Timeout을 실제 connectTO로 맞춘다 (가치 3 / 위험 1 / S). pop3 Dial은 readGreeting(connectTO) 뒤 WrapInbound에 commandTO를 전달하고, imap Dial은 connectTO 데드라인 뒤 commandTO를 쓰는 s.failed를 호출한다. 양쪽 실제 Dial에 서로 다른 두 예산을 주고 침묵 서버로 검증해야 한다. 1순위가 이미 해결되었거나 실제 경로에서 성립하지 않음을 증명한 경우에만 전환하며 둘을 묶지 않는다.

확인 근거와 미확인 범위:
- base main@c40b91d, 작업 트리 깨끗함. `client.go:ensureIndex`, `UIDL`, `List`, `execRaw`, `commandStage`, `SelectMailbox`와 `application/sync.go:runSync`를 실제로 읽었다. runSync는 UIDL 오류 종류에 관계없이 같은 세션 List로 폴백한다.
- `execRaw`는 태그 NO를 오류로 반환하지만 정상 프레이밍의 세션은 유지하므로 같은 연결에서 다시 FETCH할 수 있다. commandStage는 RFC822.SIZE 요청을 이미 enumerate로 분류한다.
- 첫 FETCH 실패 시 성공한 빈 결과로 바뀌는 경로와 중간 실패 뒤 부분 캐시 재사용은 코드로 확인했다. 운영 서버 사례와 새 테스트의 실패 출력은 미확인이다. 영구 데이터 유실로 단정하지 않는다(다음 동기화는 새 세션으로 재시도할 수 있다).
- 앱 픽스처 `newTestApp`은 로컬 SQLite·실제 암호화 저장소를 만들고 private host/평문 메일을 허용한다. 기존 `TestSyncViaIMAP`은 fakeInbound이므로 새 결함을 잡는 증거로 대체할 수 없다.

실행 순서와 체크포인트(구현자가 각 단계 상태를 갱신; 사람 확인 없이 자동 검증):
1. [대기] 어댑터 재현·최소 수정: 영구 거절(빈 결과 금지), 후속 배치 1회 거절(재시도 중복 금지) 테스트를 만들어 현재 코드의 예상 실패를 기록한 뒤, 같은 단계에서 ensureIndex의 로컬 누적과 성공 커밋만 적용한다. `go test -race -count=1 -timeout=180s ./internal/adapters/imap/`가 통과해야 다음 단계로 간다. 성공 후 추가 FETCH가 없는지 실제 서버 수신 명령으로 단언한다.
2. [대기] 앱 결과를 고정: SELECT 1 EXISTS, LOGIN OK, 메타데이터 FETCH NO, LIST 빈 OK, LOGOUT OK를 답하는 실제 서버와 앱 회귀 테스트를 만든다. 수정 전 ensureIndex를 잠시 복원해 succeeded 오판으로 테스트가 실패함을 확인하고 즉시 수정본으로 원복한다. `go test -race -count=1 -timeout=180s ./internal/application -run 'TestSync.*IMAP'`로 기존 및 새 테스트가 통과한 상태에서 단계 종료(새 테스트 이름은 TestSyncIMAPEnumerationFailureIsNotSuccess로 정한다).
3. [대기] 통합 확인: 최종 패키지 전체·포맷·계약·diff 검사를 실행하고 변경 3파일을 확인한다. 기존 테스트 실패나 범위 확장이 생기면 계획을 수정해 좁히고 원인을 기록한다. 재시도 루프·새 정책·로그는 추가하지 않는다.

선택지 비교:
- 선택: 성공시에만 캐시 커밋 — 프로덕션 1파일에서 같은 세션 재시도와 정상 캐시를 모두 보존한다.
- 오류를 영구 캐시 — 부분 성공은 막지만 일시적인 태그 NO 이후 정상 재시도를 불필요하게 차단하므로 제외한다.
- application에서 IMAP 폴백만 금지 — 표면적인 작업 상태는 고치지만 어댑터 List/UIDL 계약의 부분 캐시는 남으므로 제외한다.
- 현상 유지 — 다음 동기화에 회복될 수 있으나 현재 작업의 거짓 성공이 남아 제외한다.
- 가장 큰 가정: 정상 태그 NO 뒤 같은 세션을 재사용할 수 있다는 현재 execRaw 계약. 테스트는 타임아웃/스트림 파손 대신 태그 NO로 이 가정을 분리한다.

추정 근거(pmo 스킬 적용; 약속이 아닌 범위):
- 바텀업 기본 작업 25~32분: 어댑터 재현·수정 10~12분, 앱 경로 테스트 10~13분, 검증·기록 5~7분.
- 알려진 불확실성 예비 5~8분: 범위별 응답 서버·비동기 cleanup 디버깅. 합계 30~40분, 45분 내 완료 가능성은 중간 수준의 정성 판단이며 통계적 신뢰구간은 아니다.
- 별도 관리 예비 0분: 이번 회차에서 추가 범위를 승인하지 않는다. 기존 TLS 회차의 루프백 서버·실제 Dialer 검증과 유사하지만 실제 작업 시간이 기록되지 않아 유사사례 기반 숫자 추정은 하지 않았다.
- 저장소 읽기·기준 테스트는 완료. 외부 PG/브라우저 설치·문서 재생성·새 의존성은 추정에서 제외한다.

적용 스킬 출처: `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md`, `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/implementation-planning/SKILL.md`, `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/solution-exploration/SKILL.md`. 전용 Skill 호출 도구가 없어 로컬 파일을 직접 읽었다. pmo references/sources.md도 확인했으며 이 과제의 시간 범위는 외부 비용모형이 아닌 저장소 범위에 대한 정성적 작업 추정이다.
