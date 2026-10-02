# 회차 노트 2026-10-03-074740-hunter-improve — hunter
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:47] base pinned — main@4dc6979
- [러너 07:47] autonomy release — 

## 정찰 노트
- 고른 이유: 이 저장소에서 두 번 성공한 "폼이 검사한 값 = 서버가 받는 값" 패턴의 남은 구멍을 코드로 찾았다 — `findingBulkPatch` 가 `toISOString()` 결과를 검사 없이 보내고 Go `validateFindingOpsResource` 는 4자리 연도 RFC3339 만 받는다. pending 1순위였던 due_date 순수 테스트는 기각했다(테스트 전용 PR #15 가 닫힌 이력). 09-29·10-01 산출물은 main 에 없으니 그 주제 재제출 금지.
- 확신 없는 곳: 이 샌드박스에서 `node`·`go run`·`gh` 가 모두 권한 거부되어 **런타임 재현을 하지 못했다**. ① `toISOString()` 의 확장 연도 출력 ② Go 의 거절 ③ 브라우저 `datetime-local` 이 5자리 연도를 허용하는지 — 셋 다 미확인이며 과제서의 "선행 확인" 에 명령으로 적어 뒀다.
- 구현자 주의: 서버 계약은 손대지 말고 폼만 맞출 것. `resources.tsx:1110` 에 같은 모양이 있으나 이번 범위 밖(후속 아이디어). 프로덕션 거절 추가가 중심이고 테스트만 늘리는 범위로 축소하면 또 닫힌다. `npm test` pass 개수를 실측하고 SKIP 을 통과로 세지 말 것.
- [러너 07:54] scout done — 발견 건 일괄 변경의 조치 기한 폼이 서버가 거절하는 **확장 연도(±YYYYYY) RFC3339** 를 보내지 않게 제출 직�

## 구현 노트
- 바꾼 것: `web/src/finding-bulk-state.ts` 의 `findingBulkPatch` 가 `date.toISOString()` 결과를 변수로 받아 `/^\d{4}-/`(= 서버 `time.Parse(time.RFC3339)` 가 읽을 수 있는 4자리 연도 형태)인지 **보내는 값 그대로** 검사하고 아니면 한국어 오류를 던진다. 프로덕션 1파일, 서버 계약 무변경. 새 공유 벡터 `internal/app/testdata/finding-bulk-due-date.json` 을 Go(`validateFindingBulk`)와 TS(`findingBulkPatch`) 양쪽이 읽는다.
- 과제서와 다른 점(비평가가 먼저 볼 곳): 선행 확인 1이 **재현되지 않았다**. V8 은 `new Date("10000-01-01T00:00")` 를 Invalid Date 로 만들므로 5자리 연도는 기존 유한성 검사가 이미 막는다. 대신 더 평범한 도달 경로를 실측해 증명했다 — `9999-12-31T23:59`(사양상 유효·타이핑 가능한 datetime-local 값)이 UTC 서쪽 시간대에서 `+010000-01-01T04:59:00.000Z` 로 직렬화된다. 즉 **같은 입력이 브라우저 시간대에 따라 수락/거절이 갈리고**, 그래서 과제서 수용 기준 3의 "`9999-12-31T23:59` 은 수락 쪽" 전제는 UTC 이동 이상인 시간대에서만 참이다. 벡터는 그 쌍을 양쪽 다 담았다.
- 확신 없는 것: ① **브라우저 도달 가능성은 여전히 부분 확인**이다 — 연도 9999 는 평범한 4자리라 Chrome 에서 타이핑 가능하다고 보지만 실제 브라우저로 확인하지 못했다(이 세션에 브라우저 없음). 연도 0000 벡터는 HTML 이 year>0 을 요구하므로 **브라우저로 도달하지 않는다**고 보고 JSON note 에 그렇게 적었다 — 그 두 행은 함수 계약을 고정하는 용도다. ② UTC 서쪽 시간대가 이 제품(한국 폐쇄망)에서 얼마나 현실적인지는 판단하지 않았다. 결함 자체는 시간대와 무관한 계약 위반이다.
- 일부러 하지 않은 것: `finding-bulk.tsx` 에 `max="9999-12-31T23:59"` 를 **넣지 않았다** — 내가 찾은 실패 입력이 바로 그 값이라 이 `max` 로는 버그가 안 막힌다(과제서의 선택 제안은 이 경우 역효과). `resources.tsx:1110` 의 같은 모양 결함, `automation-state.ts` 차선(영향 주장이 재현되지 않아 ideas.json 에서 `rejected`), 서버 파서 두 개(`hunter_finding_timestamp` 정규식) 조사는 범위 밖으로 남겼다.
- 다음 역할이 조심할 것: 새 TS 테스트는 `process.env.TZ` 를 케이스마다 바꾸고 `finally` 에서 복원한다(Node 가 `new Date` 마다 다시 읽는 것에 의존 — v22.23.1 에서 실측 확인). 테스트 순서에 의존하지는 않지만 TZ 를 읽는 테스트를 같은 파일에 추가하려면 주의. Go 쪽 `TestFindingBulkDueDateSharedVectors` 는 **DSN 없이** 돈다(같은 `-run '^TestFinding'` 에 걸리는 testApp 13개는 SKIP 이며 통과로 세지 않았다). Go 프로덕션 무변경이라 `internal/webassets/dist` 재복사와 `go test -race ./...` 는 하지 않았고, 원격 CI(Node 26)·Docker·릴리즈는 미검증이다.
- [러너 08:04] brief accepted — 채택 — 지목한 파일·행·수정 방식("보내는 값 그대로 4자리 연도 검사")·공유 벡터 설계는 현재 코드와 정확히 맞았으�
- [러너 08:04] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인한 것: main 소스 + 새 테스트만으로 **실패를 직접 재현**(`+010000-01-01T04:59:00.000Z ... which the server answers with 400`, 1 fail/3) → 테스트가 바뀐 경로를 정말 지난다. 벡터 `wire` 9개를 Node 로 실측해 `toISOString()` 과 바이트 일치 확인. 수정본 106/106, Go 서브테스트 12개 PASS, `typecheck` 통과, 범위 이탈·보안·법무 차단 사유 없음.
- 못 본 것: DB(`HUNTER_TEST_DSN` 없음), Node 26/Docker/CI, 실제 브라우저의 `datetime-local` 9999 입력.
- 승인이어도 남는 우려 ①`testdata/finding-bulk-due-date.json` 의 Seoul 연도 0001/0000 두 행은 tzdata LMT(+8:27:52)를 정확히 단언해 ICU 갱신 시 CI 가 깨질 수 있다(고정 오프셋 존 고려). ②`finding-bulk.test.mjs:95` 의 `/조치 기한/u` 는 새 가드 메시지와 기존 유효성 메시지를 구분하지 못한다. ③연도 0000 은 `validateFindingOpsResource` 가 수락하지만 `finding_ops.go:98` 의 `hunter_finding_timestamp` 에서 NULL→SLA invalid 로 떨어진다(KST 에서 `0001-01-01T00:00` 로 도달, 기존 동작·범위 밖 → 후속 아이디어).
- 릴리즈: `VERSION` 1.19.0 그대로이고 main 에 이미 v1.19.0 태그가 있다 — 버전 상향은 릴리즈 단계가 해야 한다.
- [러너 08:08] review approved — 리뷰 승인 (risk=low)
- [러너 08:08] pr created — https://github.com/hkjang/hunter/pull/17
- [러너 08:26] ci passed — 검사 1개 모두 success
- [러너 08:26] merge done — 441ad13
- [러너 08:53] release ci-blocked — 릴리즈 커밋 CI: timeout — 제한 시간 안에 CI 완료를 확인하지 못함 (태그 보류)
