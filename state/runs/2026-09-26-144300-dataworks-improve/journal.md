# 회차 노트 2026-09-26-144300-dataworks-improve — dataworks
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:43] base pinned — main@264ce5f
- [러너 14:43] autonomy release — 

## 정찰 노트
- 최근 4회차 연속 채택된 유형(같은 값을 두 경로가 다르게 읽는 불일치, Go 단일 파일, 실제 store+Routes HTTP 회귀로 증명)을 그대로 따랐다. `valid_to`·`expires_at` 트림과 horizon 오버플로는 이미 다 고쳐졌고, 남은 같은 계열 구멍이 "런타임은 `valid_from` 이 해석 불가라 영구 403 인데 액션 센터는 `valid_to` 만 봐서 화면에 아무것도 안 뜬다" 였다.
- 기각한 후보: `contractScopeCanServe` 의 해석 불가 `valid_from` → `true` 는 publish gate masking 증거를 **막는** 보수적 방향이고 `dataworks_masking_policy_test.go:252` 로 고정돼 있어 고치면 민감 상품 게이트가 느슨해진다. 중복 엔타이틀먼트 경고는 `usableEntitlement` 주석이 복수 행을 설계로 못박아 기각. 새 action type 계열(미래 `valid_from`·고아 행)은 웹 4파일+문서까지 번져 파일 한도를 넘는다.
- 추측으로 적은 것: 액션 맵에 `valid_from` 필드를 더해도 `go run ./cmd/api-surface-audit` 가 gap 0 을 유지하는지는 실행해 보지 않았다(라우트를 더하지 않으므로 문제없을 것으로 봤다). `admin_dataworks_action_center_test.go` 는 존재와 159행 주석만 확인했고 내부 헬퍼 이름까지는 읽지 않았다.
- 조심할 것: 루프 재구성 시 계약당 append 1회 유지(수용 기준 2가 중복을 잡는다), 판정에만 trim 하고 응답·저장은 원문 유지, `gofmt -l` 은 같은 패키지 `dataworks_runtime.go` 가 CRLF 라 HEAD 에서도 파일명을 뱉으므로 `admin_dataworks.go` 만 볼 것.
- [러너 14:48] scout done — 액션 센터가 `valid_from` 이 깨져 모든 조회를 막는 계약도 `contract_expiring` 으로 보고하게 하기 (가치 3 / 위험

## 구현 노트
- 액션 센터 계약 루프(admin_dataworks.go:252~)를 scope 당 `report`/`severity` 선계산 + append 1회 구조로 바꾸고, 트림 후 비어 있지 않은 `valid_from` 이 RFC3339Nano 로 안 읽히면 보고 + `high` 를 켰다. 액션 맵에 `valid_from` 원문 추가. 런타임 게이트(`contractScopeActive`·`contractScopeCanServe`)는 손대지 않았다.
- `valid_to` 분기는 동치로 옮겼다: 예전 `err==nil && !Before(deadline) → continue` + 사후 severity 계산을 `err!=nil || Before(now) → high` / `Before(deadline) → medium` / 그 외 무보고 로 다시 썼다. `deadline = now+horizon` 이고 horizon>0 이 `parseExpiryHorizon` 에서 보장되므로 `Before(now) ⇒ Before(deadline)` — 이 전제가 이 동치의 유일한 근거다.
- 검증 못 한 것: 실제 브라우저에서 web 액션 카드가 새 `valid_from` 키를 무시하는지는 눈으로 안 봤다(`ActionItem` 은 optional 필드 인터페이스이고 type/summary 키를 안 늘렸으므로 렌더 영향 없다고 판단, npm 미실행). 시간 의존 픽스처라 UTC 아닌 환경·자정 경계에서 돌려 보지 않았다.
- 일부러 안 한 것: 미래 `valid_from`(아직 안 열린 창)은 오류가 아니라 예정 계약이라 제외 — 별도 아이디어로 남겼다. `contract_expiring` 이 이제 세 가지 상태(임박·만료·창 해석 불가)를 한 type 으로 묶는 문제도 ideas.json 에 새 항목으로 적고 이번엔 뺐다(웹 동반 변경).
- 다음 역할이 조심할 것: 새 테스트는 SQLite 임시 DB + `httptest` 서버를 사례마다 새로 띄운다(8사례 × 3창, 약 0.8초). 되돌림 검증은 프로덕션 파일만 `git checkout` 해서 했고 `unparseable_from_no_end`/`_future_end`/`_both` 3사례만 실패, 기존 ActionCenter 6건은 통과했다. `gofmt -l` 은 `admin_dataworks.go`·테스트 파일만 확인(둘 다 클린), CRLF 인 `dataworks_runtime.go` 는 건드리지 않았다.
- [러너 14:54] brief accepted — 채택 — 근거(`contractScopeActive` 의 `valid_from` 처리와 액션 센터 루프의 `valid_to`-only `continue`)가 현재 코드와 정확히 일치했�
- [러너 14:55] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: 프로덕션 파일만 origin/main 으로 되돌려 새 테스트가 3사례 실패함을 실제로 재현(테스트가 변경을 검증한다). `valid_to` 분기 재작성은 동치 — 구현자가 유일한 근거로 든 `horizon>0`을 `parseExpiryHorizon`(admin_dataworks.go:370-401)에서 직접 확인했고, `deadline<now` 는 도달 불가능하다. `contractScopeActive`(dataworks_runtime.go:317-331)가 해석 불가 `valid_from` 을 실제로 거부함도 확인. build·vet·proxy 테스트·api-surface-audit(gap 0)·gofmt 모두 통과, 변경 파일은 LF 유지.
- 구현 노트의 '검증 못 한 것' 둘 다 해소: `ActionItem` 은 옵셔널 인터페이스이고 `ActionRow`(home-page.tsx:197)는 새 키를 렌더하지 않아 무해, 픽스처는 상대 오프셋뿐이라 자정 경계 플레이크 없음.
- 못 본 것: web 빌드/lint/vitest 미실행(TS 변경 없음), Playwright 미실행, PostgreSQL 백엔드 미검증(SQLite 만).
- 승인이어도 남는 우려(릴리즈 노트): `contract_expiring` 한 type 이 이제 임박·만료·창 해석 불가 세 상태를 묶고 `expiring_contracts` 카운터에 만료 예정이 아닌 계약이 섞인다. `next_action` 문구는 여전히 "renew, narrow, or retire" 라 실제 작업(깨진 날짜 행 수리)과 어긋난다 — 구현자가 ideas.json 에 남긴 type 분리를 다음 회차 후보로.
- 커버리지 구멍(결함 아님): '`valid_from` 해석 불가 + `valid_to` 가 조회 창 안' 조합(medium→high 승격)이 표에 없다. 웹 `ActionItem` 에 `valid_from` 을 안 넣어 화면에선 어느 날짜가 깨졌는지 여전히 안 보인다.
- [러너 14:58] review approved — 리뷰 승인 (risk=low)
- [러너 14:59] pr created — https://github.com/hkjang/dataworks/pull/29
- [러너 15:01] ci passed — 검사 2개 모두 success
- [러너 15:01] merge done — 20f0a86
- [러너 15:10] release published — v0.9.62
- [러너 15:10] gh-release created — GitHub Release v0.9.62
- [러너 15:10] manifest ok — dataworks-v0.9.62.tar.gz 
- [러너 15:10] assets uploaded — 1개
- [러너 15:10] assets verified — v0.9.62 자산 1개 (이전 v0.9.61: 1)
