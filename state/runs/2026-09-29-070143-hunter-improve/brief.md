- 과제: validateFindingBulk의 개수·ID·revision·patch·상태 계약을 DB 없는 회귀 테스트로 고정 (가치 3 / 위험 1 / 작업량 S)
- 왜: 현재 순수 검사는 담당자 공유 벡터 21개만 다루며, 요청 개수·ID 길이·revision 반환값의 경계를 DB 없이 검증하지 못한다. 실제 서버 요청 타입과 검증 함수를 직접 실행하는 테스트를 추가하면 데이터베이스가 없는 개발 환경에서도 잘못된 일괄 입력의 거절과 정확한 revision 보존을 확인할 수 있다.
- 수용 기준:
  1) 새 TestFindingBulkValidationContract(권장 이름)가 findingBulkRequest/findingBulkItem과 실제 validateFindingBulk를 사용한다. nil/빈 items 및 101개는 거절, 유일한 ID와 유효한 일시의 1개/100개는 허용한다. 생성한 100개 입력은 ID마다 반환된 revision을 대조한다.
  2) 빈 ID·201바이트 ID·중복 ID를 거절하고 200바이트 ID를 허용한다. ASCII 200/201과 한글 66개+ASCII 2개(200바이트)/한글 67개(201바이트)를 포함한다. updated_at 누락·임의 문자열·시간대 없는 일시는 거절하고, UTC·+09:00·9자리 소수 초는 허용하며 반환 time.Time의 순간과 나노초 정밀도를 명시적 기대값(time.Date 등)과 비교한다. 테스트 기대값을 같은 time.Parse 호출로만 만들지 않는다.
  3) nil/빈 patch, 미지원 필드(title·owner_id·service_id·evidence·verification 각각), status의 nil·JSON 숫자·배열·빈 문자열·resolved·accepted·false_positive·임의 문자열을 거절한다. candidate/confirmed/in_progress/retest/inconclusive 각각은 허용한다. 모든 거절은 errors.As로 findingBulkError 값 타입과 Status==400, nil revisions를 확인한다. 각 사례는 다른 필드를 정상으로 두어 의도한 조건 하나 때문에 거절되는지 분리한다. 기존 담당자 공유 벡터는 그대로 통과해야 한다.
- 건드릴 파일: internal/app/finding_bulk_validate_test.go — 기존 TestFindingBulkAssigneeSharedVectors 보존, 독립적인 테이블 테스트와 필요시 테스트용 입력 생성 헬퍼만 추가. 프로덕션 코드 0파일, 총 변경 목표 1파일.
- 검증 명령:
  - 기준선: go test -run '^TestFindingBulkAssigneeSharedVectors$' -count=1 -v ./internal/app
  - 추가 후: go test -run '^TestFindingBulk(ValidationContract|AssigneeSharedVectors)$' -count=1 -v ./internal/app
  - go vet ./internal/app
  - gofmt -l internal/app/finding_bulk_validate_test.go
  - git diff --check
  - DB 통합 검증은 선택적으로 기존 안전한 HUNTER_TEST_DSN이 준비된 환경에서 go test -run '^TestFindingBulk' -count=1 -v ./internal/app. skip은 DB 검증 통과가 아니다. 이번 과제의 필수 순수 테스트에서 skip은 허용하지 않는다.
- 위험과 피할 것: auth·SQL·마이그레이션·워크플로·원본 PentAGI·프런트·버전·문서 생성물은 범위 밖. 함수 nil 포인터 허용, ID 공백 정규화, RFC3339 파서 강화 같은 새 계약을 만들지 않는다. status 검사는 요청 허용 목록이며 현재 발견 건 상태의 전이 승인/원자적 롤백을 증명하지 않는다. map 순회 순서에 의존한 첫 오류 문구 단언은 금지한다. 각 사례마다 새 map/slice를 만들어 담당자 정규화 등 변경이 사례 사이에 누출되지 않게 한다. 모조 App/DB/validator는 만들지 않는다. 테스트만 추가하는 작업이므로 인위적으로 프로덕션 버그를 주입해 red를 만들 필요가 없다.
- 차선 후보: validateFindingBulk의 due_date 위임 계약을 같은 순수 테스트 파일에서 고정 — 1순위가 착수 시 이미 구현된 경우만 선택. nil/빈 문자열/시간대 있는 일시 허용, 숫자·배열·날짜만 있는 문자열 거절과 findingBulkError 400을 실제 함수로 확인한다. validateFindingOpsResource 기존 테스트는 유효/무효 문자열 두 사례만 있으므로 bulk 래핑과 해제 입력이 보강 지점이다.

근거와 조사 결과
- 기준 main@f0f7667, VERSION 1.18.0. 시작·마지막 git status --short는 깨끗했다. CLAUDE.md 및 별도 ROADMAP/TODO 파일은 검색에서 발견하지 못했다. README, docs/validation.md, 사용자 가이드 일괄 변경 구간, 최근 git log -30, CI/Release/Pages 설정을 확인했다. 선택 영역 TODO/FIXME 검색 결과는 없었다.
- 실제 읽은 배선: finding_bulk.go의 registerFindingBulk → bulkFindings → applyFindingBulk → validateFindingBulk(트랜잭션 시작 전). OpenAPI POST /api/findings/bulk의 400 입력 오류·1~100개 계약도 확인했다. DB 기반 TestFindingBulkRejectsInvalidScopesRevisionsAndPartialChanges는 일부 거절을 이미 다루므로 '서버 입력 테스트가 전혀 없다'고 설명하지 말 것. 추가 가치는 DSN 없는 경계/성공 출력 검증이다.
- 중요 정정: 제공된 09-28 성공 기록은 최종 main과 다르다. b591b40/e7d2bb2에서 웹은 trim 후 실제 wire 값의 바이트를 세도록 되돌렸고, Go 공유 테스트도 v.Wire를 넣는다. 원문 200바이트+공백을 UI에서 거절하도록 다시 바꾸지 않는다. 본 과제는 담당자 계약을 변경하지 않는다.
- 실제 실행: Node 22.23.1에서 npm --prefix web test 104 PASS/0 FAIL/0 SKIP. Go 1.26.7에서 위 기준선 테스트 1개와 하위 21개 PASS, 패키지 0.026초. 새 테스트·vet·전체 Go·DB·빌드·원격 CI는 이번 정찰에서 실행하지 않았다. Node 26 CI와 로컬 Node 22 차이는 남는다.

실행 순서 및 체크포인트 (현재 모두 구현 대기)
1. 현재 파일과 기준선 명령 확인. 이미 수용 기준이 모두 테스트됐다면 차선으로 수정하고 이유를 journal에 기록한다. 사람 승인 체크포인트 없음.
2. 같은 테스트 파일에 개수·ID·revision 묶음을 추가하고 위 새 테스트 명령으로 검증한다. 통과하면 patch·status 묶음을 추가해 같은 명령으로 검증한다. 실패가 실제 동작 변경을 요구하면 범위를 넓히지 말고 발견 내용을 기록한다. 사람 승인 체크포인트 없음.
3. vet·gofmt 확인·diff 검사 후 실제 통과/실패/skip 수를 기록한다. 담당자 fixture와 프로덕션 diff가 없는지 확인한다. 사람 승인 체크포인트 없음.

선택 비교 및 추정
- 최소안(채택): 기존 파일의 실제 validator 직접 검사. 새 하네스 없이 한 파일로 완료하며 API의 형식·한도·revision 회귀를 잡는다.
- DB HTTP 검사 확대: 권한·롤백까지 관찰 가능하지만 PostgreSQL 준비와 기존 회귀 중복이 커서 이번 45분 과제로 선택하지 않는다.
- Go/TS 전체 bulk 공유 계약 벡터화: 성장 시 유용하나 웹은 ID/revision 검증자가 아니며 계약이 다른 파서를 통합할 위험이 있다. 이번에는 담당자 기존 공유 벡터만 유지한다.
- 현상 유지: CI의 DB 검사로 일부 보호되지만 100개 성공/200바이트 경계/정밀도 반환 보장은 보강되지 않는다.
- 추정 방식은 bottom-up: 기준 확인 3~5분 + 경계 테스트 작성 12~18분 + 검증/기록 5~7분 = 기본 20~30분. 알려진 불확실성(컴파일 캐시·테이블 오류)에 별도 contingency 5~10분, 합계 25~40분. 통계적으로 보정된 신뢰구간이 아닌 중간 확신의 작업 추정이며 기존 의존성/Go 도구가 준비됐다는 가정이다. 별도 management reserve는 배정하지 않으며 새 범위는 이번 세션에서 받지 않는다. 과거 유사 회차의 실제 작업 시간은 없어 정량 유추는 하지 않았다.
- 가장 중요한 가정: 현재 순수 테스트 공백이 구현 착수 시에도 남아 있다. 실제 DB 동작 결함을 발견했다는 과제가 아니다.

적용한 회사 스킬
전용 Skill 도구는 제공되지 않아 로컬 정본을 읽었다. 후보 비교, 명령/체크포인트, 근거·범위·예비시간을 반영했다.
- [pmo:estimating-and-contingency](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md) 및 references/sources.md 열람. 외부 비용 산정 권위나 통계 신뢰도에 관한 주장은 사용하지 않았다.
- [technology:implementation-planning](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/implementation-planning/SKILL.md)
- [technology:solution-exploration](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/solution-exploration/SKILL.md)
