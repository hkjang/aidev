# 회차 노트 2026-10-01-014247-GoalForge-improve — GoalForge
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 01:42] base pinned — main@0c9e799
- [러너 01:42] autonomy release — 
- [러너 01:48] scout failed — 과제서 없음 — 구현자가 직접 고른다 (예산 초과로 중단)

## 구현 노트
- 무인 스윕이 "등록됐지만 목표가 아직 없는 프로젝트" 를 고장으로 읽어 15분마다 stderr 에 같은 줄을 찍고, Acted() 가 영원히 true 가 되어 침묵 설계가 무력화되던 것을 고쳤다. 같은 함수의 기존 판례(예산 소진→Note)를 따랐고, 지금까지 프로덕션에서 아무도 읽지 않던 Note 를 sweepReport 로 꺼내 실제로 출력되게 했다.
- 확신 없는 곳: (1) Note 출력 위치 — Acted() 가 true 일 때만 찍히므로 "조용한 프로젝트만 있는 스윕" 은 여전히 완전 침묵이다. 이게 옳은 절충이라 판단했지만, 운영자가 "왜 아무것도 안 도나" 를 로그만으로 알 길은 여전히 없다(수동 `standards autonomy` 는 됨). (2) ErrNotFound 외의 CurrentGoal 실패 경로는 실제로 유발해 보지 않았다 — 코드상 Err 로 남는 것만 읽어서 확인했다.
- 일부러 안 한 것: AutoDecisions.Refused/Detail 은 여전히 tickProject 에서 버려진다. 봉투 거절 사유를 15분마다 찍으면 노이즈가 되고, 중복 제거 설계가 필요해 이번 범위 밖으로 뒀다(ideas.json 에 남기지 않음 — 같은 결정을 반복하지 않도록 여기 적는다). AutoApproveMerges 의 DailyLimit 누락은 실물 결함으로 보이나 위험 구역이라 단독 회차 후보로만 올렸다.
- 다음 역할이 조심할 것: internal/observer 테스트는 실제 SQLite 와 실제 git 저장소를 쓴다(git 없으면 t.Skipf). tickProjectIn 을 tickRepoIn(목표 없음) + SetGoal 로 쪼갰으므로 기존 호출부 동작은 그대로다. cmd/goalforge/sweep_test.go 는 순수 함수만 보므로 DB·git 불필요.
- 검증: gofmt -l ./cmd ./internal 무출력, go vet ./..., go build ./..., go test ./... -count=1 exit 0.
- [러너 01:57] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 확인: tick.go:101 분기와 sweepReport 를 diff 로 읽고 두 테스트가 정말 그 분기를 지나는지 대조했다(제거 시 무엇이 깨지는지까지). CurrentGoal→ErrNotFound 계약을 store.go:640/652 에서 확인, ProjectTick/TickResult 소비자는 sweepReport 하나뿐임을 grep 으로 확인. gofmt·vet·go test ./... 전부 직접 돌려 통과. 판정: approve, risk low, 차단 없음.
- 못 본 것: ErrNotFound 가 아닌 CurrentGoal 실패 경로(구현자도 미유발)는 나도 코드 독해로만 확인했다. 워커를 실제로 15분 돌려 본 것은 아니다.
- 남는 우려 1 (다음 회차): 새로 채운 Note 는 같은 스윕의 다른 프로젝트가 Acted() 를 켜 줄 때만 출력된다 — 단일 프로젝트 프로그램에서는 영원히 계산만 되고 안 찍힌다. 의도된 절충이고 테스트가 고정하고 있으나, 커밋이 비판한 바로 그 패턴이므로 후속(하루 1회 요약 틱 등)이 필요하다.
- 남는 우려 2 (릴리즈 노트): "목표가 아직 없습니다" 는 목표를 **달성(COMPLETED)** 한 프로젝트에도 나온다(CurrentGoal 은 ACTIVE 만 본다). 처방은 맞지만 문구가 부정확하다. docs/STANDARDS.md:276-280 예시에 새 줄이 없으니 한 줄 언급 권장.
- 다음 회차 후보로 원장 ③(AutoApproveMerges 가 DailyLimit 을 보지 않고 auto_approvals 에 기록도 없음)은 읽어 봐도 실물 결함으로 보인다 — 봉투 위험 구역이라 단독 회차가 맞다.
- [러너 02:01] review approved — 리뷰 승인 (risk=low)
- [러너 02:01] pr created — https://github.com/hkjang/goalforge/pull/49
- [러너 02:05] ci passed — 검사 3개 모두 success
- [러너 02:05] merge done — 0925b19
- [러너 02:08] release published — v0.31.0
- [러너 02:08] gh-release created — GitHub Release v0.31.0
