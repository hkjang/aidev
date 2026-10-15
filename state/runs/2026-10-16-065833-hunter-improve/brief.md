# 과제서 — 2026-10-16 hunter

- 과제: 발견 건 하나를 저장할 때도 조치 담당자에 일괄 변경·ITSM 동기화가 이미 강제하는 같은 규칙(UTF-8 200바이트 이하·제어문자 없음·앞뒤 공백 제거)을 적용 (가치 4 / 위험 2 / 작업량 S)

- 왜: `assignee` 를 쓰는 세 경로 중 두 곳만 규칙이 있다 — 일괄 변경은 `internal/app/finding_bulk.go:61-66` 에서 `len(name) > 200 || strings.ContainsFunc(name, unicode.IsControl)` 로 거절하고 `strings.TrimSpace` 한 값을 저장하며, ITSM 수입은 `internal/app/automation_workflows_tickets.go:50-64` 에서 `workflowText(s, 200)`(= 같은 두 조건, `automation_workflows.go:151`) 로 거절하고 trim 하는데, **평범한 발견 건 등록·수정 경로에는 검사가 아예 없다**: `internal/app/domain.go:464` 의 `case "findings"` 는 `validateFindingOpsResource(m)`(`finding_ops.go:66` — `due_date` 만 본다)·제목·심각도·상태만 보고 `assignee` 를 한 번도 읽지 않는다. 그 결과 (1) 폼·API 로 길이·제어문자 제한 없는 담당자 문자열이 저장되어 CSV 내보내기(`domain.go:846` `exportReport`)·GraphQL(`graphql.go:417`)·알림 변수 `finding.assignee`(`notifications.go:69`)로 그대로 나가고, (2) 공백만 입력한 담당자가 조치함 SQL(`finding_ops.go:168`·178 의 `btrim(coalesce(data->>'assignee',''))=''`)에서는 **미할당**으로 세어지는데 상세·CSV·수신자 매칭(`notification_automation_recipients.go:77-78` 의 `str(entity.Data,"assignee") == u.ID`)에서는 할당된 값으로 취급되어 같은 발견 건이 두 집계에서 다르게 읽힌다. 한 규칙을 한 곳에 두면 세 경로의 저장값이 같아지고 조치함의 미할당 수와 상세 표시가 어긋나지 않는다.

- 수용 기준:
  1) `POST /api/findings` 와 `PUT /api/findings/{id}` 가 201바이트 담당자, 제어문자가 섞인 담당자, 문자열이 아닌 담당자를 일괄 변경과 **같은 문구**(`조치 담당자는 줄바꿈 없이 UTF-8 200바이트 이하로 입력하세요`)로 400 거절한다. 200바이트 정확히는 수락한다.
  2) 앞뒤 공백이 있는 담당자는 거절이 아니라 `strings.TrimSpace` 된 값으로 저장된다(일괄 변경과 동일). 공백만 입력하면 빈 문자열이 되어 조치함의 `unassigned` 집계와 상세 표시가 같아진다.
  3) **이미 저장된 규칙 밖 값은 수정을 막지 않는다** — 제출한 `assignee` 가 `old["assignee"]` 와 같으면 검사를 건너뛴다(10-05 회차가 `domain.go` 에 targets 타입 검사를 넣지 않은 것과 같은 이유: 기존 행의 모든 수정이 막히면 안 된다).
  4) 규칙이 한 곳에만 있다 — `finding_bulk.go` 의 `case "assignee"` 가 새 헬퍼를 호출하고 인라인 조건·문구가 복제되어 남지 않는다.
  5) 테스트는 DSN 없이 도는 순수 Go 시험으로 1~4를 증명한다(기존 `validateContributionPoints`(`domain.go:409`) 처럼 자유 함수로 빼서 직접 호출). 위 다섯 경계값(200/201/제어문자/비문자열/공백 trim)과 "기존 값 그대로면 통과" 를 각각 단언한다.

- 건드릴 파일 (프로덕션 3 + 테스트 1):
  - `internal/app/finding_ops.go` — `validateFindingOpsResource` 바로 뒤에 자유 함수 `validateFindingAssignee(m, old map[string]any) error` 추가. `m["assignee"]` 가 없거나 nil 이면 통과, `old != nil && m["assignee"] == old["assignee"]`(문자열 비교) 면 통과, 문자열이 아니면 거절, `len(name) > 200 || strings.ContainsFunc(name, unicode.IsControl)` 면 `finding_bulk.go` 의 기존 문구 그대로 거절, 통과 시 `m["assignee"] = strings.TrimSpace(name)` 로 되쓴다(형제 검증기들이 `m[k] = n` 으로 되쓰는 것과 같은 모양). 왜 이 규칙인지와 세 경로를 주석으로 남긴다.
  - `internal/app/domain.go` — `case "findings"`(464행) 의 `validateFindingOpsResource(m)` 호출 직후에 `validateFindingAssignee(m, old)` 를 호출해 에러를 그대로 반환. 그 분기의 다른 줄은 손대지 않는다.
  - `internal/app/finding_bulk.go` — `case "assignee"`(61-66행) 의 인라인 조건·문구를 `validateFindingAssignee(in.Patch, nil)` 호출로 교체(`in.Patch[field] = strings.TrimSpace(name)` 는 헬퍼의 되쓰기가 대신한다). 패치 키 이름이 `assignee` 로 같으므로 맵을 그대로 넘길 수 있다. **한도·롤백·updated_at 검사 등 이 함수의 다른 부분은 건드리지 않는다.**
  - `internal/app/finding_assignee_test.go` (신규) — 위 수용 기준 5. `validateFindingAssignee` 직접 호출 + `validateFindingBulk` 로 일괄 경로도 같은 문구를 내는지 확인(이미 `finding_bulk_due_test.go` 가 DSN 없이 `validateFindingBulk` 를 호출하는 선례가 있다).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  ```sh
  gofmt -l internal/app/finding_ops.go internal/app/domain.go internal/app/finding_bulk.go internal/app/finding_assignee_test.go
  go vet ./internal/app
  go test -run '^TestFinding' -count=1 -v ./internal/app   # DSN 없는 testApp 기반 시험은 SKIP 으로 표시되며 통과로 세지 말 것
  go test -count=1 ./internal/app
  go build ./cmd/hunter
  ```
  웹 변경이 없으면 `npm --prefix web test`(기준선 113 PASS/0 FAIL/0 SKIP — 이번 정찰 미실행, 10-08 기록값)·`internal/webassets/dist` 재복사는 불필요하다. DB 통합 시험이 필요하면 `HUNTER_TEST_DSN='postgres://<TEST_USER>:<TEST_PASSWORD>@<TEST_HOST>:5432/<TEST_DB>?sslmode=disable' go test -race -count=1 ./internal/app` 를 쓰고, DSN 이 없어 SKIP 된 것을 DB 검증 통과로 보고하지 말 것.

- 위험과 피할 것:
  - **기존 행을 잠그지 말 것.** 수용 기준 3 의 "값이 그대로면 건너뛴다" 를 빼면 ITSM·과거 API 로 들어간 긴 담당자를 가진 발견 건의 모든 수정이 400 이 된다. 10-05 회차가 같은 이유로 `domain.go` 에 targets 타입 검사를 넣지 않았다.
  - **trim 되쓰기가 활동 기록을 만든다.** `internal/app/finding_ops_activity.go:154` 가 `assignee` 변경을 기록하므로 공백이 붙어 있던 발견 건은 첫 저장 때 변경 1건으로 남는다. 의도된 결과이니 완료 보고에 적고, 이를 피하려 되쓰기를 빼지 말 것(그러면 수용 기준 2 가 깨진다).
  - **`web/src/settings.tsx` 를 건드리지 말 것.** 2026-10-08 회차의 SLA·risk `integer: true` 변경은 main@9069bcf 에 아직 없고(이 트리에서 `grep -n integer web/src/settings.tsx` 결과 없음) 리뷰 대기 중이다. 같은 파일을 만지면 충돌한다. 설정 화면 숫자 필드 과제는 이번 회차 범위가 아니다.
  - 일괄 변경의 **100건 한도·현재 권한·각 `updated_at`·한 건 충돌 전체 롤백**은 그대로 두고, `case "due_date"`·`case "status"`·`default` 분기와 문구도 바꾸지 말 것.
  - ITSM 쪽 `workflowText(s, 200)` 를 새 헬퍼로 바꾸려 하지 말 것. 외부 티켓 파서는 별도 계약(필드 매핑·revision 고정)을 지키고 있고, 운영자 지침은 여러 경로의 같은 종류 파서를 통합하지 말고 좁히는 방향으로만 고치라고 한다. 이번에는 **검사가 없던 한 경로에 규칙을 추가**하는 것이지 외부 파서를 바꾸는 것이 아니다.
  - 감사 기록(`details`)에 담당자 문자열 원문을 넣지 말 것 — 식별자만 넘긴다는 운영자 지침이 있다. 오류 문구는 입력값을 되읊지 않는 기존 고정 문장을 쓴다.
  - auth/OIDC·마이그레이션(`init*` SQL)·`.github/workflows`·`third_party/pentagi` 는 건드리지 않는다. Go 프로덕션을 바꿨으니 `gofmt` 는 변경한 Hunter 파일에만 적용하고 `gofmt -w .` 는 쓰지 말 것.
  - 폼(`web/src/resources.tsx:257` 의 `{ key: "assignee", label: "조치 담당자" }`)에 `maxLength` 를 선언하는 것은 **이번 범위 밖**이다. 서버가 규칙을 갖게 된 다음 회차에 하면 되고, 지금 UI 만 먼저 좁히면 서버가 받는 값을 폼이 거절하는 10-06 회차의 반례가 된다.

- 미확인 (구현자가 먼저 확인할 것):
  - (확인 완료 — 추측 아님) `internal/app/domain.go:301-341` `saveDomain` 을 읽어 확인했다: 수정 시 `v = old` 로 시작해 `oldData := cloneMap(v.Data)` 를 떠 두고 요청 본문 `m` 의 키만 `v.Data` 에 덮어쓴 뒤 `a.validateResource(ctx, u, &v, oldData, update)` 를 호출한다. 즉 검증기가 보는 맵은 **병합된 전체 데이터**이고 `old` 는 병합 전 사본이다. 따라서 `assignee` 를 보내지 않은 PUT 은 `m["assignee"] == old["assignee"]` 가 되어 수용 기준 3 으로 자동 통과하고, 생성 시 `oldData` 는 빈 맵이라 키가 없어 새 값으로 검사된다. `id`·`owner_id`·`updated_at` 등은 334행에서 이미 삭제되므로 `assignee` 와 무관하다.
  - 이번 정찰은 Go 테스트를 실행하지 않았다. 기준선은 10-07 기록(`go test -count=1 ./...` 전부 ok)을 인용한 것이다. 구현 전에 기준선을 직접 한 번 받고 시작할 것.
  - `assignee` 를 읽는 경로 전수는 확인하지 않았다. 위에 적은 다섯 곳(`finding_ops.go` 조치함 SQL, `exportReport`, `graphql.go:417`, `notifications.go:69`, `notification_automation_recipients.go:78`)은 grep 으로 실제 확인했다.

- 차선 후보: **발견 건 등록·수정 폼의 담당자 입력에 서버가 실제로 받는 길이를 선언** — 1순위의 서버 규칙이 성립하지 않는다고 판단되면, 대신 `internal/app/finding_bulk.go` 의 200바이트 규칙을 공유 벡터(`internal/app/testdata/` 의 기존 JSON 방식)로 고정하고 `web/src/finding-bulk-state.ts:30` 의 TS 검사와 Go 검사가 **같은 바이트 경계**를 내는지 교차 검증하는 과제로 바꾼다(TS 는 `TextEncoder` 로 trim 후 바이트를, Go 는 `len()` 으로 수신 바이트를 세므로 보내는 값이 trim 된 값이라는 전제가 두 쪽을 묶는다 — 그 전제가 코드에 기록되어 있지 않다). 프로덕션 0~1파일, 테스트 2파일.
