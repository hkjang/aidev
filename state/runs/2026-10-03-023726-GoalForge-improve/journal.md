# 회차 노트 2026-10-03-023726-GoalForge-improve — GoalForge
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 02:37] base pinned — main@a6aa75f
- [러너 02:37] autonomy release — 

## 정찰 노트
- 멱등성 결함을 골랐다: `AutoApproveMerges` 의 루프 조건이 승인 발급으로 변하지 않고 `RequestScopedApproval` 은 중복을 보지 않아 같은 커밋에 승인이 15분마다 쌓인다. 보류 1순위였던 `DailyLimit` 공백(같은 파일)보다 좁고, 기록 경로 설계 없이 닫히며, 2026-10-01 이 복구한 "조용한 스윕" 을 `Merged` 축에서 다시 깨는 중이라 가치가 더 크다. `ParseCriterion` 테스트는 버그가 아니라 차선으로 내렸다.
- 확신하는 것: `automerge.go:27-122` 에 존재 검사가 없고 `approval.go:91` 이 매 호출 새 ID 를 넣는다(둘 다 읽음), `Acted()` 가 `len(Merged)>0` 을 센다(tick.go:40), `automerge_test.go` 에 멱등성 테스트가 없다, 기준선 `go test ./internal/observer/ ./internal/store/sqlite/` 둘 다 `ok`(직접 실행).
- 추측으로 적은 것(구현자가 확인할 것): 중복 발급을 **실행으로** 재현하지는 않았다(코드를 바꾸지 않는 제약 때문에 테스트를 심지 못했다) — 그래서 과제서가 실패 테스트를 먼저 쓰고 빨강을 눈으로 보라고 요구한다. 새 메서드를 둘 승인 테스트 파일 이름도 미확인(`ls internal/store/sqlite/ | grep approval` 로 먼저 볼 것).
- 조심할 것: `automerge.go` 는 위험 구역이다. `admits()`/`AutoApprove` 의 봉투 규칙을 복제하면 회귀다 — 이번 가드는 규칙이 아니라 중복 발급 방지라는 구분을 커밋 본문에 쓸 것. 건너뛰기 조건은 `ConsumeScopedApproval` 의 정확일치 조건과 글자 그대로 같아야 한다(어긋나면 발급은 건너뛰고 소비는 못 찾는 교착).
- 범위를 번지게 하지 말 것: 옛 커밋 승인 철회와 `DailyLimit` 은 각각 별 항목으로 아이디어 파일에 남겼다. `tick.go` 끝의 `var _ = time.Now` 는 건드리지 않는다.
- [러너 02:43] scout failed — 과제서 없음 — 구현자가 직접 고른다

## 구현 노트
- `AutoApproveMerges` 의 루프 조건이 승인 발급으로 변하지 않아 같은 (작업·커밋·대상) 범위 병합 승인이 스윕마다 쌓였다. 발급 전에 `Store.ScopedApprovalExists` 로 "이 릴리즈가 이미 결정됐는가" 를 묻게 했다. 프로덕션 파일 2개(`internal/observer/automerge.go`, `internal/store/sqlite/approval.go`).
- 확신 없는 곳: ① `ScopedApprovalExists` 가 상태를 **걸러내지 않는** 판단. REJECTED 를 넘기면 사람 거절을 자동화가 뒤집고 CONSUMED 를 넘기면 이미 나간 커밋을 다시 승인해서 이렇게 했지만, 부작용은 `Approve` 가 DB 오류로 실패해 PENDING 행이 남으면 자동화가 그 작업을 다시 발급하지 않는다는 것이다(운영자가 `goalforge approval list` 로 보고 승인하면 풀린다. 그 오류 자체는 tick.Err 로 로그에 간다). ② `tickProject` → `Acted()` 가 실제로 조용해지는 것은 **테스트로 안 쌌다.** `tick.Merged = merges.Approved` 한 줄이라 코드로만 확인했고, 전체 tick 수준 픽스처(실제 git + DONE 작업 + AutoMerge 켠 봉투)는 비용 때문에 안 만들었다.
- 일부러 하지 않은 것: `DailyLimit` 공백(기록 경로 설계가 선행, 단독 회차), 옛 커밋 승인 철회(이번 가드는 같은 커밋만 본다 — 전혀 안 닫혔다), `var _ = time.Now`, `AutoDecisions` 에 "대기 중 승인" 필드 추가(`Detail` 한 줄로 CLI 거짓말만 막았다). 셋 다 ideas.json 에 적었다.
- 다음 역할이 조심할 것: 새 테스트 4개는 실제 `*store.Store`(SQLite)를 쓴다 — `internal/store/sqlite` 는 단독 실행 시 ~110s, `internal/observer` ~57s 걸렸다. `scopedApprovalRelease` 상수는 발급 쪽과 소비 쪽이 공유한다. 한쪽만 좁히거나 넓히면 중복 재발 또는 병합 교착이 되므로 상수를 쪼개지 말 것(그 대칭을 `TestScopedApprovalExistsMatchesExactlyTheReleaseConsumeWouldSpend` 가 고정한다).
- [러너 02:55] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 확인한 것: 발급 쪽(`ScopedApprovalExists`)과 소비 쪽의 `scopedApprovalRelease` 대칭(발급이 더 넓고, 넓은 쪽 차이는 PENDING/CONSUMED/REJECTED 로 전부 의도된 것 — 교착 방향 어긋남 없음), 플레이스홀더 순서가 리팩터 전후 동일, `tick.Merged = merges.Approved` 로 스윕이 실제 조용해지는 배선, `Detail` 이 `printDecisions`(standards.go:554)로 CLI 에 닿는 것. 직접 돌림: build/vet/gofmt 무출력, observer·store/sqlite·cmd/goalforge·app 네 패키지 `ok`, 새 테스트 4개 개별 통과.
- 못 본 것: 빨강 재현(코드 수정 금지 제약). 가드가 유일한 신규 프로덕션 분기임을 diff 로 확인해 원장의 실패 출력 세 줄이 세 테스트의 단정 줄과 정확히 맞는 것으로 갈음했다. 전체 `go test ./...` 는 안 돌렸다.
- 승인이어도 남는 우려 ①: `mergeWork`(main.go:623)는 **승인을 먼저 소비하고** 그 뒤 머지한다. 머지가 충돌로 실패하면 승인은 CONSUMED 로 남고 커밋은 안 나갔는데 이제 스윕이 다시 발급하지 않는다 — 이전엔 15분 뒤 저절로 풀렸다. `GuardEffect` 는 effect 를 `RetryEffect` 로 되살리지만 승인은 되살리지 않는다. 실패 방향이 안전(사람 승인 필요)하고 안내 문구가 정확해 차단하지 않았다. **다음 회차 후보: 소비를 머지 성공 뒤로 옮기거나 실패 시 APPROVED 로 되돌리기.**
- 남는 우려 ②: `tick.go:181` 이 `Detail` 을 버려, 구현자가 스스로 적은 "Approve 실패로 PENDING 이 남는 경우" 가 무인 worker 로그에서 완전히 조용하다(approval 인박스에는 보임). 이 집안의 반복 결함 그 자체이므로 `ProjectTick` 에 Detail 축을 붙이는 것이 다음 조각.
- 검토 부서: 보안·법무 둘 다 차단 없음. 변경은 권한을 좁힌다(중복 승인 = 리뷰 없는 재병합 권한, 자동화가 사람의 REJECTED 를 덮어쓰던 경로 — 둘 다 닫혔다). 새 쿼리는 상수 SQL + 바인딩 + project_id 테넌트 필터. 개인정보·새 의존성·외부 약속 없음.
- [러너 03:00] review approved — 리뷰 승인 (risk=low)
- [러너 03:00] pr created — https://github.com/hkjang/goalforge/pull/67
- [러너 03:03] ci passed — 검사 3개 모두 success
- [러너 03:03] merge done — 710d800
- [러너 03:13] release published — v0.42.0
- [러너 03:23] assets verified — v0.42.0 자산 7개 (이전 v0.41.0: 7)
