# 회차 노트 2026-09-23-170434-Kkiit-improve — Kkiit
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 17:04] base pinned — main@ee4c612
- [러너 17:04] autonomy release — 

## 정찰 노트
- 필수 스킬 3개(pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration)는 이번에 Skill 도구로 모두 로드됐다. 지난 두 회차를 멈춘 원인은 사라졌고, 과제는 그때 착수하지 못한 배열 조건 검증을 그대로 이어받았다(반려된 적 없음 — 미착수였다).
- 이걸 고른 이유: 코드에서 결함을 직접 확인했다 — approvals.go:49 는 배열 두 키를 안 보고 talents.go:644/656 은 타입 단언 실패 시 조건을 건너뛰어 정책이 전 상품에 걸린다. 차선(README 환경변수)·문서 대조·로그인 문구는 각각 효과가 작거나(문서만) 보호 경로(auth.go)여서 제쳤다. MCP OAuth 캡처 아이디어는 이 체크아웃에 mcpoauth.go 가 없어 rejected 로 내렸다.
- 추측으로 적은 것: 운영 DB 에 이미 잘못된 조건 행이 있는지는 미확인(저장소 마이그레이션에는 승인 정책 시드가 없음을 확인). docker 가용성도 미확인 — `docker` 호출이 이 세션에서 승인되지 않았다.
- 구현자가 조심할 것: 저장 경로가 POST·PUT 둘이니 한쪽만 고치지 말 것, 매칭 쪽을 보정(coerce)하지 말 것, 알 수 없는 키는 이번에 건드리지 말 것, 빈 배열·키 없음 동작은 그대로 둘 것. 프런트는 이미 같은 규칙이라 dist 재빌드 불필요.
- [러너 17:07] scout done — 승인 정책의 배열 조건(service_types·seller_levels)에 잘못된 타입이 들어오면 저장 단계에서 400 으로 거부하기

## 구현 노트
- 무엇을 왜: `validateApprovalPolicy` 가 `service_types`·`seller_levels` 를 보지 않아 잘못된 타입이 201 로 저장되고, 매칭(`approvalConditionsMatch`)이 `[]any` 단언 실패 시 조건을 건너뛰어 정책이 모든 상품에 걸렸다. 저장 검증 한 곳(POST·PUT 양쪽)에서만 거절하고 반환형을 `(reason, ok)` 로 바꿔 400 메시지에 키 이름을 넣었다. README 조건 설명에 한 구절 추가.
- 확신 없는 곳·검증 못 한 것: (1) 운영 DB 에 이미 잘못된 조건 행이 있으면 그 행은 PUT 으로 다시 저장할 수 없다 — 저장소 마이그레이션에는 승인 정책 시드가 없어 리포 안에서 깨질 행은 없지만 운영 DB 는 미확인이다(UI 편집은 폼 필드에서 조건을 재구성하므로 정상화된다). (2) `npm test`·lint·build 는 돌리지 않았다 — 프런트를 건드리지 않았고 `internal/ui/dist` 도 손대지 않았다. (3) 400 메시지 문구는 브라우저 화면에서 실제로 보지 않았다(API 응답 본문으로만 확인).
- 일부러 하지 않은 것: 매칭 쪽 타입 단언을 넓히거나 문자열을 배열로 보정하지 않았다(오타를 추측해 고치면 관리자가 눈치채지 못한다). 알 수 없는 조건 키는 이번에도 통과시킨다(범위 밖, 별도 보류 항목). 키 없음·빈 배열은 매칭이 `len(values) > 0` 에서만 거르므로 종전대로 허용 — 단위 테스트로 고정했다.
- 다음 역할이 조심할 것: 새 통합 테스트(`TestIntegrationApprovalPolicyArrayConditionsAreCheckedOnSave`)는 `KKIIT_TEST_DSN` 이 있어야 돌고, 없으면 SKIP 된다. 이 테스트는 이전 테스트가 켜 둔 `talent_publish` 정책을 잠시 껐다가 cleanup 에서 되돌리므로 `t.Parallel()` 을 넣으면 안 된다.
- 기존 플레이키 둘을 발견했다(내 변경과 무관, baseline 에서 재현): `TestIntegrationAuditLogAnswersWhoChangedThis` 의 `strings.ToUpper(uniqueName("AUD"))[:12]` 는 나노초 하위가 1e8 미만이면 11자가 되어 패닉으로 테스트 바이너리 전체를 죽인다(약 10%). 또 이미 채워진 DB 에 통합 테스트를 다시 돌리면 baseline 에서도 메일 테스트가 실패한다 — 통합은 매번 새 DB 로 돌려야 한다.
- [러너 17:22] brief accepted — 채택 — docker 가 가용(29.7.2)해 수용 기준 1·3 을 실제 HTTP→실제 DB 로 증명했고, 과제서의 근거(approvals.go:49 와 talents.go:644/
- [러너 17:22] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- 확인함: 단위 테스트는 수정 전 로직에서 실제로 실패한다(구 코드가 `service_types:"design"` 에 true 반환). gofmt·go vet·Go 전체 테스트 통과. 통합 테스트는 DSN 없어 SKIP — 코드로만 읽었고 cleanup 순서·sellTalent 의 HUMAN 타입은 문제없었다.
- 거절 사유 1(수리가 먼저 볼 파일): `scripts/approval-policies-smoke.mjs:56-57` 이 만드는 잘못된 배열 두 케이스가 이제 400 이라, `api()` 의 `assert.ok(response.ok)`(42줄)에서 78줄 생성 루프가 중단되고 정상 9 케이스 UI 검증까지 못 돈다. 직전 커밋이 넣은 승인 조건 UI 의 유일한 E2E 하네스다.
- 거절 사유 2: `web/src/pages/AdminPage.tsx:351` 의 활성 전환은 GET 으로 받은 conditions 를 그대로 PUT 한다. 이전에 저장된 잘못된 조건 행은 이제 끄지도 못한다(처리 이력이 있으면 삭제도 409). 구현 노트의 '편집 폼이 정규화한다' 는 전환 경로에는 해당하지 않는다.
- 남는 우려(릴리즈 노트용): 조건에 JSON null 을 보내던 API 직접 호출자는 400 을 받는다 — README 에 그 경우는 적히지 않았다. 알 수 없는 키는 의도적으로 계속 통과하니 수리 때 같이 막지 말 것.
- 보안·법무 차단 없음: 관리자 권한 뒤의 입력을 좁히기만 하고 400 메시지에 들어가는 문자열은 코드에 박힌 키 이름뿐이라 입력 반사·개인정보·비밀값 경로가 없다.
- [러너 17:25] review rejected — 리뷰 거절: scripts/approval-policies-smoke.mjs:56-57 이 브랜치가 깨뜨린 추적 스크립트를 함께 고치지 않았다. 이 스모크는 직전 커밋 1783d3e 가 승인 정책 조건 U

## 수리 노트
- 지적 두 건 다 맞았다. 틀린 지적은 없었다. 스모크는 손대기 전에 실제로 돌려 `POST .../policies: 400`(77줄)으로 죽는 것을 확인했고, 전환 400 은 새 통합 테스트로 재현했다.
- 고친 방법 ①: `updateApprovalPolicy` 가 검증 실패 시 저장된 `conditions` 와 본문이 완전히 같으면 조건을 그대로 둔 채 나머지만 재검증한다(`storedPolicyConditions`+`DeepEqual`). 끄기·삭제가 동시에 막히던 상태만 풀고, *다른* 잘못된 값으로 바꾸는 것은 계속 400. AdminPage.tsx 는 건드리지 않아 dist 재빌드가 필요 없다.
- 고친 방법 ②: 스모크의 잘못된 배열 두 케이스를 `400 + invalid_policy + 키 이름` 기대로 바꾸고, 같은 행을 psql 로 직접 심어 카드·모바일 레이아웃 검증을 11 케이스 모두 되살렸다. 심은 행으로 화면 전환 on/off 와 "조건을 바꾸면 여전히 거절" 을 새로 단언한다. 전체 스모크 PASS(docker+Chrome 실제 실행), 통합 포함 go test·npm test·eslint·gofmt·vet 통과, 새 통합 테스트는 수정을 임시로 되돌리면 그 자리에서 실패함을 확인했다.
- 확신 없는 곳: 전환으로 잘못된 조건 정책을 **켤** 수도 있고 그때 매칭은 그 조건을 건너뛴다(= 전 상품에 적용) — 이 브랜치 이전과 같은 동작이라 그대로 뒀다. 끄는 길을 막지 않는 쪽을 택한 결과다.
- 비평가의 남은 우려(JSON null 을 보내던 API 직접 호출자, 알 수 없는 키 통과)는 범위 밖으로 두었다. `internal/ui/dist` 의 미커밋 빌드 산출물은 내가 만들지 않았고 건드리지도 커밋하지도 않았다.
- [러너 17:34] repair done — # 수리 요약 (2026-09-23)  - 비평 두 건 모두 재현되어 맞았다. 스모크는 `POST .../policies: 400` 으로 77번 줄에서 죽었고(재현 로그 확인), 잘못된 조건이 저장�

## 비평 노트
- 실제로 돌려 확인함: 새 DB 로 httpapi 전체 테스트 통과(86s), 스모크를 docker+Chrome 으로 실행해 PASS(11케이스) — 비평의 거절 사유 1·2 는 해소됐다. approvals.go:216 의 DeepEqual 분기를 임시로 죽이면 새 통합 테스트가 400 으로 실패하므로 테스트가 수정을 고정한다. gofmt·vet 무결, 권한(router.go:161-164)·쿼리 바인딩·400 메시지에 결함 없음.
- 못 본 것: 운영 DB 에 실제로 잘못된 조건 행이 있는지, 브라우저에서 400 문구의 한국어 표현이 어떻게 보이는지(스모크는 message 문자열만 단언).
- 남는 우려 ①(릴리즈 노트): `service_types`/`seller_levels` 에 문자열·null 을 보내던 API 직접 호출자는 이제 400 이다. ② 잘못된 조건 행은 전환으로 **켤** 수도 있고, 그때 매칭이 조건을 건너뛰어 전 상품에 걸린다(브랜치 이전과 동일 동작).
- approvals.go:81-83 주석은 실제보다 넓게 말한다 — 배열이 아닐 때만 정책이 넓어지고 `["AI",3]` 은 넓어지지 않는다. 코드 결함은 아니라 승인했다.
- 다음 회차: 이 워크트리에 미커밋 `internal/ui/dist` 산출물이 남아 있으니 릴리즈 전 정리할 것. 그리고 기존 플레이키 integration_test.go:4391 `[:12]` 패닉(약 10%)은 별도 과제 후보다.
- [러너 17:40] review approved — 리뷰 승인 (risk=low)
- [러너 17:40] pr created — https://github.com/hkjang/Kkiit/pull/10
- [러너 17:40] ci passed — 검사 없음 — 정책으로 허용
- [러너 17:40] merge done — 741f9a6
- [러너 17:43] release published — v0.4.5
- [러너 17:57] assets missing — 이전 v0.4.4 엔 1개, v0.4.5 엔 0개 — 워크플로: Offline image release: completed/success
