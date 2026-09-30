# 회차 노트 2026-10-01-014237-Clustara-improve — Clustara
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 01:42] base pinned — main@ffaeb11
- [러너 01:42] autonomy release — 

## 정찰 노트
- 보류 1순위였던 "events·revisions 상한 무보고" 를 골랐다. 소스를 보니 상한 포화보다 **오류를 `_` 로 버리는 것**이 더 나쁜 증상이어서(DB 오류 = "이상 없음" 과 동일 응답) 둘을 한 과제로 묶었다. SEC-01 classifyPodSecurity(3/3/M)와 PSS volume 통제는 지난 회차가 같은 파일(security.go)을 건드려 영향 분리가 안 되므로 제쳤고, podsec dedup 키(2/1/S)는 4회째 밀린 가치 2라 1순위에서 뺐다.
- 새로 확인한 사실이 과제 성격을 바꿨다: `store/k8s.go:1082 boundedLimit` 이 events 를 500, revisions 를 1000 으로 **하드 클램프**한다. 인벤토리(상한 10000)에서 쓴 "상한+1 요청" 감지 트릭이 여기서는 통하지 않는다 — 과제서에 함정으로 명시했다. 인벤토리 기존 로직은 정상이라 건드릴 필요 없다.
- 추측으로 적은 것: 조회 오류를 테스트에서 재현하는 방법(`s.db` 가 구체 타입 `*store.SQLStore` 라 주입 지점이 없어 보임 — **미확인**). 무리해서 대역을 만들지 말고 포화 경로 테스트에 집중하라고 과제서에 적었다.
- 구현자 주의: `truncated` 키는 인벤토리 의미로 `k8s_notify_scope_test.go:170,185` 가 단언하니 의미를 바꾸지 말고 새 키를 더할 것. 응답과 감사 로그(`k8s_notify.go:277`)가 같은 사실을 말해야 한다. 알림 개수·dedup 동작은 무변경 — 이번 변경은 보고만 추가한다.
- 차선 후보로 남긴 "스토어 상한이 호출자 요청(2000)을 조용히 500/1000 으로 깎는 계약 불일치" 는 가치 4지만 호출자가 40곳 넘어 6파일 규칙을 넘는다 — 쪼개서 다음 회차에.
- [러너 01:46] scout done — notify scan 의 events·revisions 조회가 실패하거나 상한에 걸려도 "이상 없음" 과 똑같이 보고되는 것을 고치기 

## 구현 노트
- `handleK8sNotifyScan` 의 events·revisions 조회가 오류를 `_` 로 버리고 포화도 안 봐서 저하된 스캔이 깨끗한 클러스터와 동일한 응답을 냈다. 오류(`events_error`/`revisions_error`)·실제 개수(`events`/`revisions`)·포화(`*_truncated`)·안내(`window_notice`)를 응답과 감사 로그에 같이 적는다. 프로덕션 파일 1개(`internal/proxy/k8s_notify.go`), 테스트 1개 신규, docs 1개.
- **확신 없는 곳**: (1) 포화 판정 `len(events) >= notifyScanEventBudget` 은 마침 정확히 500행인 창을 포화로 오탐한다 — 스토어 하드 클램프 때문에 구별 불가하고, 오탐 방향(보고 과다)이 미보고보다 안전하다고 판단했다. (2) `window_notice` 한국어 문구는 UI에 노출되는 경로를 확인하지 않았다(응답 JSON/감사 로그만). (3) 조용한 시간에 걸린 스캔은 두 조회를 아예 하지 않게 순서를 옮겼다 — 기존에도 quiet 응답에 인벤토리 `truncated` 를 안 적었으니 계약은 일관되지만, quiet 응답의 키 집합은 그대로임을 확인만 했고 별도 테스트는 없다.
- **검증 못 한 것**: 실 PostgreSQL(`DROP TABLE` 재현 테스트는 SQLite 전용 — 두 번째 `sql.Open("sqlite", dsn)` 커넥션을 쓴다), 실 Kubernetes/ClickHouse, 브라우저. 감사 로그 단언은 `ListAdminAudit` 의 `after_value` JSON 을 읽는다.
- **일부러 하지 않은 것**: 스토어의 하드 상한(events 500 / revisions 1000)과 다른 40여 호출자의 요청값은 손대지 않았다(범위 밖, ideas.json 2번 항목). 알림 개수·dedup·조용한 시간·라우팅·딥링크는 무변경 — 이번은 보고만 추가다. 기존 `truncated` 의 의미도 그대로 두었다(`k8s_notify_scope_test.go:170,185` 가 직접 단언).
- **다음 역할 주의**: 새 테스트는 SQLite 파일 DB 가 필요하고(`t.TempDir()`), `TestNotifyScanReportsAFailedEventLookup` 은 `store.Open`+`Migrate` 를 직접 부르는 자체 배선이다(`openTestStore` 가 DSN 을 안 돌려줘서). 창 크기를 조정하는 테스트는 `notifyScanEventBudget`/`notifyScanRevisionBudget` 을 `t.Cleanup` 으로 반드시 되돌릴 것 — 패키지 var 이라 병렬 테스트와 섞으면 샌다.
- [러너 01:55] brief accepted — 채택 — 과제서의 근거(`k8s_notify.go:195-196` 의 `_` 폐기, 스토어 하드 상한 500/1000 때문에 인벤토리의 '상한+1' 트릭이 복사되
- [러너 01:56] verify passed — 검증 3개 통과 (policy)

## 비평 노트
- 확인: `git diff main...HEAD` 전문, `k8s_notify.go:183-341` 전문, `store/k8s.go` 의 `boundedLimit`·`ListK8sEvents`(clamp 500)·`ListK8sRevisions`(clamp 1000) — 커밋 메시지가 주장한 하드 상한이 실제와 정확히 일치했다. 직접 실행: `go test ./internal/proxy -run 'NotifyScan|K8sNotify' -count=1` → ok 1.645s, `go test ./... -count=1` → 전부 ok, `gofmt -l`(수정 2파일) 무출력, `go vet ./internal/proxy` 무출력.
- 테스트가 변경을 검증하는가: 4개 모두 `events_truncated`/`events`/`window_notice`/`events_error` 처럼 **수정 전에는 응답에 존재하지 않던 키**를 단언하므로 구코드에서 통과할 수 없다. 원장의 `- 실패 재현:` 줄(`k8s_notify_window_test.go:194` 와 `:61`)의 맵이 main 의 응답 형태(`map[evaluated_rca:0 ... truncated:false ...]`)와 정확히 맞아 증상 일치도 확인했다. 병렬 테스트 없음(`t.Parallel()` 0건)이라 패키지 var 변조는 안전하고 `t.Cleanup` 복원도 있다.
- 못 본 것: 실 PostgreSQL 에서의 `events_error` 경로(재현 테스트는 SQLite `DROP TABLE` 전용), 실 K8s·브라우저. `newNotifyScanServer` 의 webhook 채널은 버퍼 16이라 미드레인 누수 없음을 확인.
- 승인이어도 남는 우려: (1) 포화 판정 `len >= budget` 은 budget 이 스토어 상한과 같아야만 성립 — 상한 위로 올리면 조용히 영구 false 가 된다, 보류 아이디어 '스토어 상한 계약 불일치'를 다음 회차가 건드릴 때 같이 볼 것. (2) 두 조회가 quiet-hours 조기 반환 아래로 옮겨져 조용한 시간엔 실행되지 않는다 — 관측 계약은 불변이나 '보고만 추가' 문구가 덮지 못하는 동작 변화, 테스트 없음. (3) `admin_ui.go` 는 이 엔드포인트를 호출하지 않아 `window_notice` 는 화면에 안 뜬다 — 릴리즈 노트는 UI 가 아니라 API·감사 필드 추가로 적을 것.
- 검토 부서 소견 없음: 인가는 기존 `authorizeAdmin` 그대로이고 새 식별자·비밀값·쿼리 주입면이 없으며, 추가된 것은 개수·불리언·DB 오류 문자열뿐이라 개인정보 신규 수집·보존·전송 변화도 없다. `*_error` 의 raw 오류 노출은 인벤토리 실패 경로(`writeOpenAIError(..., err.Error())`)와 동급이라 새 노출 등급이 아니다 — 차단 아님.
- [러너 02:00] review approved — 리뷰 승인 (risk=low)
- [러너 02:00] pr created — https://github.com/hkjang/clustara/pull/33
- [러너 02:01] ci passed — 검사 없음 — 정책으로 허용
- [러너 02:01] merge done — c025508
- [러너 02:07] release published — v0.9.293
- [러너 02:07] gh-release created — GitHub Release v0.9.293
- [러너 02:07] manifest ok — clustara-v0.9.293.tar.gz clustara-v0.9.293.tar.gz.sha256 README-offline-v0.9.293.md 
- [러너 02:07] assets uploaded — 3개
- [러너 02:07] assets verified — v0.9.293 자산 3개 (이전 v0.9.292: 3)
