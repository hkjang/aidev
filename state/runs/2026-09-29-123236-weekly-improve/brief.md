- 과제: Confluence 수동 동기화가 상태 조회 실패를 접수 성공으로 답하지 않는다 (가치 3 / 위험 2 / 작업량 S)
- 왜: `internal/app/confluence_handlers.go:forceConfluenceSync`는 상태 SELECT의 Scan 오류를 버려 상태를 읽지 못해도 `202 queued=true`를 답하고 요청 감사 기록을 남긴다. 조회 실패를 `500 QUERY_FAILED`로 거절하면 기존 프런트 오류 알림이 동작하여 관리자가 접수되지 않은 요청을 성공으로 오해하지 않는다.
- 수용 기준:
  1) Confluence가 활성화된 상태에서 `confluence_sync_state` 조회가 실패하거나 CONFLUENCE 행이 없으면 `POST /api/v1/admin/confluence/sync`는 `500 QUERY_FAILED`와 안전한 한국어 메시지로 답한다. 오류 분기는 `wakeConfluenceWorker`와 `audit` 전에 반환하고 SQL 원문을 응답에 넣지 않는다.
  2) 비활성은 기존 `409 CONFLUENCE_DISABLED`, RUNNING은 `202 {queued:false,status:"RUNNING"}`, 정상 IDLE은 `202 {queued:true,status:"QUEUED"}`를 유지한다. 실패 요청에 `confluence.sync.request` 감사 행이 늘지 않고 정상 접수에서는 늘어야 한다.
  3) 실제 `newTestServer`·`testServer.request`·scratch PostgreSQL로 테이블 이름 변경 장애와 행 없음 경로를 시험한다. 오류 처리 전 코드에서 202 응답 때문에 실패하고 수정 후 통과해야 한다. fake App/DB/worker, 문자열 존재 검사로 대신하지 않는다.
  4) OpenAPI의 `/admin/confluence/sync` POST 응답에 500을 추가한다. 프런트와 worker의 실행 정책은 변경하지 않는다.
- 건드릴 파일:
  - `internal/app/confluence_handlers.go:forceConfluenceSync` — 현재 `_ = a.db.QueryRow(...).Scan(&status)`를 오류 검사와 즉시 반환으로 교체. 같은 파일의 `adminConfluenceStatus` 오류 코드·문구 관례 사용. 제품 코드 1개.
  - `internal/app/confluence_test.go` — `TestTheForceSyncRefusesAnUnreadableState` 회귀 시험 추가 제안, `// guards: forceConfluenceSync`. 기존 `TestTheConfluenceCardRefusesToAnswerWithAFailedQuery`의 ALTER TABLE/복구 방식을 활용. 정상 상태·RUNNING·행 없음·장애/복구를 확인한다.
  - `docs/openapi.yaml:/admin/confluence/sync` — 500 Error 응답과 상태 조회 실패 설명만 추가.
- 검증 명령: 아래 DB 환경 준비 후 `go test ./internal/app -run 'TestTheForceSyncRefusesAnUnreadableState|TestFeaturesLeftOffAnswerWithTheirOwnCodeNotAFailure|TestTheConfluenceCardRefusesToAnswerWithAFailedQuery|TestTheOperatorLearnsTheDiagnosisTableIsCutOff' -count=1 -v`; `go test ./... -count=1`; `go vet ./...`; `go build ./...`; `python3 scripts/openapi-check.py`; `python3 scripts/guard-check.py --changed 59a684c`. 최종 `git diff --check`.
- 위험과 피할 것: 인증/CSRF, migrations, workflows, `currentConfluenceCandidates`의 의도적 best-effort 조회, 주기 worker의 상태·잠금·재시도 정책을 건드리지 않는다. `runConfluenceSync`에는 이미 PostgreSQL advisory lock이 있어 이번 결함을 중복 실행 보안 결함으로 부풀리지 않는다. 신규 재시도·영속 큐·동시성 원자화는 범위 밖. 폴링 실패/성공 알림 문구 개선도 별도 후보다. worker가 실제로 실행되므로 채널 길이 검사·임의 sleep을 증거로 쓰거나 교체 채널을 주입하지 않는다.
- 차선 후보: README·MCP·관리자 MCP 도움말의 읽기 전용 설명과 CHECKS의 authz 실행 시간·병행 금지 문서 정정 묶음 — 1순위가 이미 해결됐거나 상태 조회 실패도 접수해야 한다는 명시적 제품 계약을 발견한 경우에만 전환.

실행 순서와 체크포인트 (구현자용, 전부 미착수)
1. 기존 실제 DB 시험을 먼저 실행한다. 아래 컨테이너가 없으면 연결을 확보하며 SKIP을 통과로 세지 않는다. 자동 체크포인트: 기존 기준 시험 통과. 사람 승인 단계 없음.
2. 실제 하네스로 새 회귀 시험을 쓰고 구 코드에서 202를 관찰한다. `internal/app/confluencepaging_test.go:TestTheSyncStopsWhenAPageComesBackEmpty`의 PUT `/api/v1/admin/settings` 활성화 배선을 참고한다. 외부 Confluence는 실제 프로토콜을 받는 httptest 서버로 한정한다. `New`가 실제 worker도 시작한다는 점에 주의한다. RUNNING fixture를 worker가 덮어쓰지 않도록 기존 advisory lock 등 실제 동기화 수단을 검토하고, 핵심 실패 재현을 위해 제품 하네스를 개조하지 않는다. ALTER TABLE은 해당 시험의 scratch DB에서만 하고 defer로 복원한다. 감사 행은 `audittrail_test.go:TestAnAuditRecordSurvivesTheClientWalkingAway`처럼 실제 `audit_logs`를 조회한다. 자동 체크포인트: 예상 202/500 차이가 실패 이유임을 확인.
3. 핸들러 오류 분기와 OpenAPI를 수정하고 같은 시험을 통과시킨다. 정상 계약도 함께 확인한다. 자동 체크포인트: 새 시험 및 기존 비활성 시험 통과.
4. 전체 Go 시험·vet·build·OpenAPI·변경 guard를 순차 검증한다. 범위를 넓혀야 하는 사실이 나오면 과제서를 수정하고 사유를 기록한다. mutation/authz 도구는 소스를 직접 바꾸므로 다른 검사와 병행하지 않는다. 이번 과제에서는 긴 mutation 실행을 필수로 추가하지 않는다.

정찰 확인과 한계
- base `59a684c`, 트리 깨끗함. `App.routes`가 ADMIN+CSRF로 POST를 감싼다. `ConfluenceTab.sync`는 이미 post 실패를 error 알림으로 처리한다(렌더 시험은 미실행).
- `migrations/005_confluence.sql`은 CONFLUENCE/IDLE 행을 생성한다. 행 없음도 성공으로 읽을 이유가 없어 같은 500 계약을 제안했다. 이는 이번 과제의 설계 결정이다.
- 정찰에서 기존 진단 잘림·질의 실패 시험 2개를 실제 pgvector DB로 실행: 통과, internal/app 1.800s. 먼저 DSN 없이 실행했을 때는 두 시험 모두 SKIP이었다.
- `python3 scripts/openapi-check.py` 통과(119 경로), `python3 scripts/paging-check.py` 통과(목록 10곳). 이 검사는 HTTP 응답 행동 증명이 아니다.
- 수동 POST의 실제 실패 재현과 신규 시험은 정찰에서 실행하지 않았다. 현재 오류 무시는 코드를 직접 읽은 근거이며 신규 회귀 시험의 red 확인은 구현자의 첫 작업이다. 전체 Go 시험·guard·vet·build는 이번 정찰에서 미실행.

DB 환경 준비 (컨테이너 설정에서 읽으며 비밀번호를 출력하거나 파일에 저장하지 않음)
```bash
export WEEKLY_TEST_POSTGRES_DSN="$(python3 - <<'PY'
import json,subprocess,urllib.parse
c=json.loads(subprocess.check_output(['docker','inspect','weekly-test-pg']))[0]
e=dict(s.split('=',1) for s in c['Config']['Env'] if '=' in s)
q=lambda s:urllib.parse.quote(s,safe='')
print('postgres://'+q(e.get('POSTGRES_USER','postgres'))+':'+q(e.get('POSTGRES_PASSWORD',''))+'@127.0.0.1:15434/'+q(e.get('POSTGRES_DB','postgres'))+'?sslmode=disable')
PY
)"
```
정찰은 위와 동일한 컨테이너 정보로 DSN을 구성하여 Python subprocess의 환경에만 전달해 두 시험을 실행했다. 컨테이너 이름·이미지·포트는 현재 직접 확인했으며 전역 셸 설정 파일은 변경하지 않는다.

대안 비교와 추정 근거
- 채택: 상태 읽기 실패 즉시 거절. 기존 500 계약/프런트 catch를 사용하고 새 구성 요소가 없다.
- 보류: 영속 작업 큐 또는 원자적 상태 전이. 접수·실행 전체 보장은 더 넓은 문제이며 마이그레이션과 worker 변경이 필요해 45분 범위를 넘는다.
- 보류: UI 안내만 수정 또는 현상 유지. 서버의 거짓 queued=true와 감사 기록이 남아 원인을 해결하지 못한다.
- Bottom-up 기본 25~35분: fixture/실패 재현 10~15분, 핸들러·명세 5분, 정상 계약·전체 검증 10~15분. 알려진 불확실성(worker fixture 동기화)에 contingency 5~10분, 총 30~45분(판단상 약 70% 신뢰, 통계 추정 아님). 미정 범위용 management reserve는 배정하지 않으며 범위를 늘리지 않는다.
- 유사 과제 비교: 9/28 상태 조회 오류 처리와 같은 DB/HTTP 시험·500 패턴이며 이번에는 제품 1파일로 더 작다. 이전 회차의 실제 총 구현 시간은 없어 분 단위 독립 추정으로 주장하지 않는다. 가장 큰 가정은 실제 worker와 충돌하지 않는 작은 fixture로 재현할 수 있다는 것이다. 15분 안에 재현이 안 되면 추정과 계획을 재작성하고 미확인 부분을 숨기지 않는다.
- 적용 스킬: pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration. Skill 전용 도구가 없어 `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/`의 해당 SKILL.md 원본을 직접 읽었다. 추정은 저장소 코드/회차 기록에 근거하며 외부 비용 산정 수치 인용은 없다.
