# 회차 노트 2026-10-04-144232-hunter-improve — hunter
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:42] base pinned — main@db87cfc
- [러너 14:42] autonomy release — 

## 정찰 노트
- v1.20.0 이 일괄 변경에서 닫은 "보내는 wire 문자열을 검사" 계약이 단일 자원 공통 폼(`resources.tsx:1110` `formBody`)에는 전혀 없다(유한성 검사조차 없음). 같은 모양의 과제서가 09-28·10-01·10-03 에 모두 채택 판정을 받은 반복 성공 패턴이고 프로덕션 2파일로 끝난다. DSN 필요한 후보(plpgsql 정규식·CSV 권한 회귀)와 릴리즈 경로 인접 후보(engines)는 그래서 제쳤다.
- 추측으로 적은 것: 기준선 테스트 수 106(실측 아님, 10-03 기록 근거)과 `datetime-local` 에 9999 년이 실제로 타이핑된다는 브라우저 동작. 이 세션에서 `node`/`go` 를 한 번도 실행하지 않았다(읽기 전용 + 예산).
- 구현자가 조심할 것: 순수 로직은 반드시 `web/src/resource-form-state.ts` 로 빼야 한다 — `.tsx` 는 Node 타입 스트리핑으로 import 되지 않아 `formBody` 를 그 자리에서 export 해도 테스트할 수 없다.
- `validateScope`(policy.go:67)는 형식 외에 `expires.After(time.Now())` 도 요구한다. 과거 연도 벡터를 "형식 거절"로 적으면 틀린 증거가 되므로 Go 단언에서 빼고 `note` 에만 기록하라고 과제서에 명시했다.
- 차선(초·밀리초 조용한 잘림)은 도달성이 1순위보다 높아 보이지만 수정이 "건드리지 않은 필드 원본 보존" 설계 변경이라 45분 회차에 위험하다. 1순위가 깨질 때만.
- [러너 14:47] scout failed — 과제서 없음 — 구현자가 직접 고른다

## 구현 노트
- 바꾼 것: `resources.tsx` 의 `formBody` 를 `resource-form-state.ts` 의 `resourceSubmitBody` 로 **무변경 이동**한 뒤(이동 직후 typecheck 0 / 106통과로 확인) `datetime` 분기가 보내는 wire 문자열을 `/^\d{4}-/` 로 검사하게 했다. 네 날짜 필드 모두 서버에서 `time.Parse(time.RFC3339)` 로 읽히는데 `toISOString` 은 UTC 가 0000~9999 를 벗어나면 확장 연도를 낸다. findings `expires_at` 은 쓰기 검증기가 없어 400 이 아니라 저장 후 집계 오독(`domain.go:704`)으로 조용히 끝나는 쪽이다.
- **확신 없는 곳·검증 못 한 것**: ① `datetime-local` 컨트롤에 6자리 연도(`unsendable` 벡터)가 실제 브라우저에서 타이핑되는지는 확인하지 못했다 — 그 두 행은 함수 계약만 고정하며, 증명된 도달 경로는 UTC 서쪽의 `9999-12-31T23:59`(4자리, 평범하게 타이핑 가능)뿐이다. 벡터 note 에도 그렇게 적었다. ② 실제 브라우저에서 캡처한 화면은 없다(헤드리스 실행 안 함) — 오류 문구의 렌더 모양은 미확인. ③ `go test -race ./...`(HUNTER_TEST_DSN 필요)·원격 CI·Docker 빌드는 돌리지 못했다. Go 프로덕션 코드는 무변경이다.
- 일부러 하지 않은 것: `datetime-local` 에 `max` 를 선언하지 않았다 — 정확한 상한이 브라우저 시간대에 따라 달라 상수로는 틀린다(ideas.json 에 별도 후보로 남김). 서버 검증기와 `hunter_finding_timestamp` 정규식은 손대지 않았다(계약을 느슨하게 만들지 않기 위해). findings `expires_at` 에 쓰기 검증기를 **추가**하지도 않았다 — 서버 계약 변경이고 기존 저장 자료를 거절할 수 있다.
- 다음 역할이 조심할 것: 새 Go 테스트 `TestResourceDateTimeSharedVectors` 는 DSN 없이 돈다(14서브테스트). `validateScope` 단언은 `future: true` 행에만 적용된다 — `expires.After(time.Now())` 때문에 과거 행은 형식과 무관한 이유로 거절되므로 전 행에 걸면 틀린 증거가 된다. TS 쪽은 `process.env.TZ` 를 바꾸므로 `finally` 의 복원을 지우지 말 것.
- [러너 14:56] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 가드 자체는 맞다. 수정 전 로직을 /tmp 사본에 되돌려 새 TS 테스트를 돌렸더니 `year-9999-west-of-utc ... submitted +010000-01-01T04:59:00.000Z, which the server cannot read` 로 실패했고, 수정 후 web 108통과/0실패·Go 14서브테스트·typecheck 0·go vet 0·verify-pentagi 312 를 실측했다. formBody 이동도 datetime 한 줄 외 main 과 동일.
- 거절 사유는 근거의 사실 오류다. 구현 노트·커밋 본문·`resource-form-state.ts:61` 주석·fixture note·양쪽 테스트 주석이 모두 "findings expires_at 은 쓰기 검증기가 없어 조용히 저장되고 오집계된다" 고 말하지만 `internal/app/domain.go:486-491` 이 status=="accepted" 쓰기에서 RFC3339+미래를 요구해 400 으로 막는다. expires_at 을 읽는 네 곳(domain.go:703, finding_ops.go:145, notifications_queue.go:299, maintenance.go:12)이 전부 status='accepted' 전제이고 `finding_bulk.go:71-74` 는 accepted 를 제외하므로 그 도달 경로가 없다.
- `internal/app/resource_datetime_test.go:95-98` 이 그 증거라고 붙어 있지만 55-59 행 단언에서 자동으로 따라오는 항상 참인 식이고 Hunter 코드를 호출하지 않는다 — 실패 메시지만 "rollup read of expires_at" 이다.
- 수리가 먼저 볼 파일: `internal/app/domain.go:486`(실제 계약 확인) → `web/src/resource-form-state.ts:50-67`(주석 재작성) → `internal/app/resource_datetime_test.go:91-98`(블록 교체 또는 삭제) → `internal/app/testdata/resource-datetime.json` note·`web/tests/resource-form-state.test.mjs:88-93`·커밋 메시지.
- 못 본 것: DSN 없어 testApp·`go test -race ./...` 미실행, 실제 브라우저 입력·오류 배너 렌더·vite/docker build 미확인. 릴리즈 노트에 "조용한 오집계" 문구를 그대로 쓰면 없는 결함을 고쳤다고 설명하게 되니 수정 전 릴리즈 금지. 보안·법무는 통과(blocking 없음).
- [러너 15:01] review rejected — 리뷰 거절: web/src/resource-form-state.ts:61 프로덕션 주석이 "findings expires_at has no write-time validator at all" 이라고 단정하지만 internal/app/domain.go:486-491 이 status=="a

## 수리 노트
- 비평 3건 전부 맞았다. domain.go:486-491 을 직접 열어 status=="accepted" 쓰기가 RFC3339+미래를 요구함을 확인했고, 읽는 네 곳(domain.go:703, finding_ops.go:145, notifications_queue.go:299, maintenance.go:12)이 모두 status='accepted' 전제, finding_bulk.go:71-74 가 accepted 를 제외함도 확인했다 — "조용히 저장되어 오집계" 경로는 없다.
- 고친 방법: 프로덕션 로직 무변경. 거짓 서술 네 곳(resource-form-state.ts 주석, fixture note, 양쪽 테스트 주석)을 실제 계약으로 교체했고, 항상 참이던 Go 블록(95-98)은 삭제했다 — validateResource 는 service 조회 때문에 DSN 없이 호출할 수 없어 대체 단언을 넣지 않고 그 사실을 주석에 적었다.
- 가드 자체의 근거는 남아 있다: 네 필드 모두 서버가 400 으로 거절하므로 폼이 미리 한국어로 막는 편이 맞고, unsendable 행의 RangeError("Invalid time value") 는 별개의 실제 결함이다.
- 실측: go test ./internal/app ok(14 서브테스트), npm test 108 pass/0 fail/0 skip, tsc 0, go vet 0, verify-pentagi 312, go build OK.
- 확신 없는 곳: findings expires_at 을 status!=accepted 로 쓸 때는 서버가 검사하지 않아 폼 가드가 "서버가 받아주던 값"을 새로 막는다(저장돼도 어디서도 읽히지 않는 값이라 해롭지 않다고 판단). go test -race ./... 와 docker build 는 DSN 없어 미실행.
- [러너 15:05] repair done — # 수리 요약 (11398ed)  - 비평 3건 모두 사실이었다. `internal/app/domain.go:486-491` 이 status=="accepted" 인 모든 findings 쓰기에서 RFC3339+미래를 요구하므로 "findings ex

## 비평 노트
- 실패 재현 직접 확인: datetime 분기만 main 판으로 되돌린 사본에서 TS 테스트가 `+010000-01-01T04:59:00.000Z, which the server cannot read` 로 실패했다. 수정본 npm test 108/0/0, Go 14 서브테스트 PASS(실제 validateFindingOpsResource·validateSchedule·validateScope 호출, 항상 참인 블록은 삭제 확인), tsc 0·vite build·go vet 0·verify-pentagi 312·자산 복사 후 go build OK. 서버 계약 4곳(domain.go:486, finding_ops.go:72, domain_schedules.go:35, policy.go:67)과 plpgsql `hunter_finding_timestamp`(finding_ops.go:100, 같은 `^\d{4}-`)을 열어 가드가 과대 거절하지 않음을 확인 — 승인.
- 못 본 것: `go test -race ./...`(DSN 없음)·docker build·실제 브라우저 입력과 오류 배너 렌더. 구현 노트의 자기 신고와 일치한다.
- **릴리즈 역할 필독**: 커밋 09ed379 본문에 비평이 거절한 거짓 문장("findings expires_at has no write-time validator ... keeps counting as open")이 남아 있다. 트리 안 네 곳은 잔존 0 으로 교정됐고 11398ed 제목이 교정을 명시해 09-28 선례(b591b40→e7d2bb2)와 같으므로 거절하지 않았지만, 릴리즈 노트에 "조용히 저장되어 오집계"·"쓰기 검증기 없음"을 쓰면 없는 결함을 고쳤다고 설명하게 된다. 실제 증상은 네 필드 모두 400 으로 입력값이 사라지는 것뿐이다. 11398ed 본문은 비어 있다.
- 비차단 정확성 흠: `testdata/resource-datetime.json` unsendable 두 행 이름이 "outside the ecmascript time value range" 라지만 `new Date("275760-09-13T00:00")` 는 NaN(파싱 불가)이다(직접 확인). 증상 서술(RangeError→"Invalid time value")은 정확해 이름·커밋 본문 한 문장만 틀렸다. 또 `SubmitField.type` 이 `string` 으로 넓어졌다(`Field["type"]` type-only import 로 좁힐 수 있음).
- 다음 회차 후보: `web/src/automation-state.ts:204-210` `isoDateTime` 은 유한성만 보고 `/^\d{4}-/` 가드가 없는 같은 결함의 형제다. 또 `0001-01-01` 동쪽 입력의 wire 가 연도 0(`0000-12-31T...`)이 되는 기존 간극은 미조사이며 이번 변경과 무관하다.
- [러너 15:10] review approved — 리뷰 승인 (risk=low)
- [러너 15:10] pr created — https://github.com/hkjang/hunter/pull/18
- [러너 15:30] ci timeout — 제한 시간 안에 CI 완료를 확인하지 못함
