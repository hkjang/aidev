# 회차 노트 2026-10-16-065833-hunter-improve — hunter
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 06:58] base pinned — main@9069bcf
- [러너 06:58] autonomy release — 

## 정찰 노트
- 고른 이유: `assignee` 를 쓰는 세 경로 중 평범한 발견 건 저장만 규칙이 없다 — `domain.go:464` findings 분기가 `assignee` 를 한 번도 읽지 않는 것을 코드로 확인했고, 일괄 변경(`finding_bulk.go:63`)과 ITSM(`workflowText(s,200)`)은 이미 같은 두 조건을 강제한다. 프로덕션 3파일·순수 Go 시험으로 끝난다.
- 제친 후보: 목록 CSV RangeError 는 도달 경로가 없어 rejected(모든 `_at` 컬럼이 `Date.parse` → 범위 밖은 NaN → `Number.isFinite` 에서 걸린다). `resourceSubmitBody` number 분기도 서버 `number()` fallback + 되쓰기가 흡수해 rejected. 설정 화면 숫자 과제 전부는 2026-10-08 변경이 main 에 없어(`grep integer web/src/settings.tsx` 0건) 같은 파일 충돌로 제외.
- 추측으로 적은 것: 없다 — `saveDomain` 의 old 병합까지 직접 읽어 확인했다. 다만 이번 정찰은 Go·웹 테스트를 실행하지 않았다(기준선은 10-07·10-08 기록 인용). 구현자는 기준선을 먼저 직접 받을 것.
- 조심할 것: 수용 기준 3(제출값이 `old` 와 같으면 건너뛰기)을 빼면 ITSM·과거 API 로 들어간 긴 담당자의 모든 수정이 400 이 된다(10-05 회차가 targets 타입 검사를 포기한 이유와 동일). trim 되쓰기는 `finding_ops_activity.go:154` 에 변경 1건을 남긴다 — 의도이니 보고에 적을 것.
- 폼(`resources.tsx:257`)에 `maxLength` 를 같이 넣지 말 것. 서버 규칙이 없는 상태에서 UI 만 좁히는 것은 10-06 회차가 기각한 방향이고, 다음 회차 과제로 ideas.json 에 남겼다.
- [러너 07:04] scout done — 발견 건 하나를 저장할 때도 조치 담당자에 일괄 변경·ITSM 동기화가 이미 강제하는 같은 규칙(UTF-8 200바이

## 구현 노트
- 무엇·왜: `assignee` 를 저장하는 세 경로 중 평범한 발견 건 등록·수정만 검사가 없어, 폼·API 로 들어온 담당자가 CSV·GraphQL·알림 변수로 그대로 나가고 공백만 담당자가 조치함 집계(미할당)와 상세·수신자 매칭(할당)에서 다르게 읽혔다. 규칙을 `finding_ops.go` 의 `validateFindingAssignee(m, old)` 한 곳에 두고 findings 분기와 일괄 변경이 함께 호출하게 했다(프로덕션 3파일 + 테스트 1파일).
- 확신 없는 곳·검증 못 한 것: **HTTP·DB 를 지나는 실제 POST/PUT 은 돌리지 못했다**(`HUNTER_TEST_DSN` 없음) — 배선은 DB 를 건드리지 않는 `(&App{}).validateResource` 호출로 증명했고(`service_id` 가 비면 `a.resource` 를 타지 않음) 수용 기준 1 의 400 응답 자체는 추론이다. `go test -race`·원격 CI·Docker 도 미실행. 비평가가 먼저 볼 곳: 일괄 경로에 새로 넣은 **nil 전용 거절**(아래)과 `old` 비교가 `value == previous` 의 `any` 동등 비교라는 점(둘 다 문자열일 때만 참이 되므로 과거에 숫자/객체가 저장된 행은 "변경" 으로 보고 거절된다 — 의도지만 실데이터로 확인하지 못했다).
- 과제서에서 벗어난 한 가지: 헬퍼의 "nil 은 통과" 규칙을 일괄 경로에 그대로 적용하면 `{"patch":{"assignee":null}}` 이 **기존 400 을 잃고 JSON null 을 저장**한다(먼저 red 로 고정했다). 그래서 일괄에만 nil 거절 한 줄을 다른 문구로 남겼다. 규칙 본문은 여전히 한 곳이다.
- 일부러 하지 않은 것: `web/src/resources.tsx` 의 `maxLength`(서버 바이트 ≠ HTML UTF-16 코드 단위라 한글에서 어긋난다 — ideas.json 에 설계 과제로 남김), ITSM 의 `workflowText(s,200)` 통합(외부 파서 별도 계약), `web/src/settings.tsx`(미머지 PR 충돌).
- 다음 역할이 조심할 것: 새 시험 4개는 **DSN 없이 전부 돈다**(69 PASS / 13 SKIP — SKIP 은 기존 testApp 통합시험). trim 되쓰기 때문에 공백이 붙어 있던 담당자는 **첫 저장 때 `finding_ops_activity.go:154` 에 변경 1건을 남긴다**(수용 기준 2 를 지키려면 불가피; 되쓰기를 빼면 안 된다). 웹 변경이 없어 `internal/webassets/dist` 재복사는 하지 않았다.
- [러너 07:12] brief accepted — 채택 — 지목한 파일·행(`domain.go:464` 가 `assignee` 미독, `finding_bulk.go:61-66` 인라인 조건, `saveDomain` 의 old 병합)과 수정 방식�
- [러너 07:12] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인한 것: diff 4파일 전부, `saveDomain` 의 `cloneMap`+병합 경로, `validateResource` findings 분기, recover(server.go:239), openapi 의 assignee 선언 6곳. 새 시험은 진짜 red 다 — domain.go 의 훅 3줄을 임시로 지우면 `TestFindingAssigneeIsWiredIntoResourceValidation` 이 pre-change 메시지로 실패한다(복원 확인, 트리 clean). gofmt/vet/build/`go test ./internal/app` 전부 통과.
- 거절 사유: `finding_ops.go:106` 의 `value == previous` 는 `any` 동등 비교라 양쪽이 `map[string]any`·`[]any` 면 **panic** 한다. 저장된 assignee 가 JSON 객체/배열인 행(main 에서 POST 로 생성 가능 — domain.go:279 는 assignee 를 타입 검사하지 않는다)은 assignee 를 건드리지 않는 PUT 마다 500 이 된다. 변경 전에는 200 이었다. 실제로 helper 와 `validateResource` 양쪽에서 panic 을 재현했다.
- 수리가 먼저 볼 파일: `internal/app/finding_ops.go:100-118`(비교를 문자열끼리로), 그 다음 `internal/app/openapi.json` 의 `FindingUpdate.assignee`(FindingBulkPatch 와 같은 maxLength/x-max-bytes/설명 추가). 회귀 시험은 "객체/배열 legacy 값 + assignee 미제출 PUT" 벡터가 필요하다.
- 못 본 것: 실 HTTP·DB(HUNTER_TEST_DSN 없음), `go test -race ./...`, 웹 시험, CI, Docker. 수용 기준 1 의 400 본문은 여전히 추론이다.
- 차단은 걸지 않았다: 인가 우회·비밀 노출·개인정보 신규 수집이 없고, panic 은 recover 로 막혀 단일 행 가용성 문제에 머문다(인증된 findings:write 필요).
- [러너 07:17] review rejected — 리뷰 거절: internal/app/finding_ops.go:106 `value == previous` compares two `any` values. Go panics ("comparing uncomparable type") when both sides hold the same uncomparable dynamic type, which i

## 수리 노트
- 비평 3건 전부 맞았다. panic 은 임시 시험으로 helper 와 `(&App{}).validateResource(update=true)` 양쪽에서 재현했고(`comparing uncomparable type map[string]interface {}` / `[]interface {}`), 지적대로 주석의 "skip" 약속과 실동작(500)이 갈라져 있었다. 틀린 지적은 없었다.
- 다만 제안된 "문자열끼리만 비교" 는 **쓰지 않았다**: 그러면 객체/배열 legacy 행은 skip 이 아니라 400 이 되어 수용 기준 3(기존 값 보유 행의 다른 필드 수정 유지)과 주석을 반대로 깬다. 대신 `sameJSONValue`(json.Marshal 바이트 비교 — `domain.go:456` 의 targets 비교와 같은 기존 관용구)로 바꿔 문자열 동작은 동일하게 두고 모든 타입이 skip 되게 했다. 주석도 이 동작으로 다시 썼다. 바뀐 값만 규칙을 받으므로 새 객체/배열 제출은 여전히 400.
- openapi 는 `FindingUpdate` 뿐 아니라 **`Finding` 도** 고쳤다 — POST /api/findings 의 요청 본문이 `Finding` 이다(응답 겸용이지만 같은 스키마가 이미 `evidence`·`contribution_points` 를 쓰기 규칙으로 기술한다). 응답 전용 `FindingQueueItem` 은 그대로 뒀다.
- 회귀 시험 `TestFindingAssigneeToleratesUncomparableStoredValues` 11 케이스 추가. 고치기 전 비교로 되돌리면 같은 panic 으로 FAIL 하는 것을 확인하고 복원했다(트리 clean).
- 여전히 확신 없는 곳: 실 HTTP·DB 를 지나는 PUT 의 500→200 회복은 코드 수준 증명이다(`HUNTER_TEST_DSN` 없음 → `./internal/app` 175개 중 128 SKIP 전부 DSN 게이트, 0 FAIL). `go test -race ./...`·CI·Docker 미실행. `sameJSONValue` 는 `domain.go` 의 인라인 비교와 중복이지만 범위를 넘지 않으려 통합하지 않았다.
- [러너 07:23] repair done — # 수리 요약 (시도 1 → 커밋 df4aef6)  - 비평 1·2 맞음: `finding_ops.go:106` 의 `value == previous` 는 `any` 동등 비교라 저장된 assignee 가 JSON 객체/배열인 행에서 `com
