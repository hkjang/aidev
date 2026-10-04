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
