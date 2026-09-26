# 회차 노트 2026-09-27-061158-Clustara-improve — Clustara
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 06:12] base pinned — main@cf1e4a5
- [러너 06:12] autonomy release — 

## 구현 노트
- `POST /admin/k8s/notify/scan` 이 kind 제한 없이 2000행(updated_at 역순)만 읽어, 분석이 보지 않는 kind 가 창을 채우면 privileged 워크로드·과도한 Role 이 밀려나 알림이 조용히 사라졌다. `analyzer.RCARelevantKinds()` 를 추가해 `SecurityRelevantKinds()` 와의 합집합만 조회하고, budget+1 로 잘림을 감지해 `truncated`·`resources`·`truncation_notice` 를 응답·감사에 보고한다(ceb42b9, 프로덕션 파일 2개).
- **확신 없는 곳**: ① `RCARelevantKinds()` 는 rca.go/workload.go 의 kind switch 를 손으로 옮긴 것이라 앞으로 새 kind 를 읽는 분석이 생기면 여기도 고쳐야 하고, 빠뜨려도 컴파일은 통과한다 — 테스트가 목록을 고정하지만 "분석이 읽는 것" 과 자동 동기화되지는 않는다. ② `SecurityRelevantKinds()` 의 NetworkPolicy·Secret 은 이 스캔이 알리지 않는 finding 용이지만 합집합에 남겼다(가져오는 게 읽는 것의 상위집합이어야 한다는 판단). 원하면 좁힐 수 있다. ③ 이번 변경으로 그동안 창 밖이라 조용하던 finding 이 첫 스캔에서 한꺼번에 나갈 수 있다 — 의도된 동작이고 커밋 메시지에 적었지만 실 운영 규모에서는 미검증. ④ 실 PostgreSQL·실 Kubernetes·브라우저 모두 미검증(SQLite+httptest 만).
- **일부러 하지 않은 것**: `ListK8sEvents(clusterID, 500)`·`ListK8sRevisions(Limit 1000)` 의 무보고 잘림은 그대로 뒀다(이벤트는 involved_kind 축이라 별도 설계 필요 — ideas.json 에 신규 항목으로 적음). quiet_hours 조기 반환 응답에는 truncated 를 넣지 않았다(분석 전에 반환하므로 보고할 스캔 결과가 없음). podsec dedup 키의 Kind 누락도 이번 범위 밖.
- **다음 역할 주의**: 새 테스트는 `internal/proxy/k8s_notify_scope_test.go`. `notifyScanBudget` 은 테스트가 줄이는 var 이므로 const 로 바꾸지 말 것(기존 `securityScanBudget` 과 같은 관례). 종단 3개는 SQLite 임시 DB + httptest Mattermost webhook 을 쓰고 `server.invalidateMattermostCache()` 에 의존한다. 검증 명령: `go test ./internal/proxy -run NotifyScan -count=1` → `go build ./...` → `go vet ./...` → `go test ./...`(19 패키지 ok). 문서는 `docs/K8S_OPERATIONS_HUB.md` 에만 절을 추가했고 버전 마커는 건드리지 않았다.
- [러너 06:21] verify passed — 검증 3개 통과 (policy)

## 비평 노트
- 확인: `RCARelevantKinds()` 7개를 rca.go:65·workload.go:20/44/53 kind switch 와 1:1 대조 — 누락 없음. `SecurityRelevantKinds()` 도 security.go:108/115/127 만 읽어 덮인다. truncated 경계(Limit=budget+1, `boundedLimit(...,200,10000)` 가 2001 통과, ORDER BY updated_at DESC → items[:budget] 은 최신) 전부 코드로 확인.
- 테스트 유효성: budget=3 + 더 최신 ConfigMap 5행이라 `Kinds:` 없으면 sent:0, 있으면 sent:1 — 두 상태를 구분한다. 원장 `- 실패 재현:` 출력이 수정 전 응답(resources/truncated 키 없는 `sent:0`)과 일치하고 `Kinds:` 한 줄 되빼기 인과 확인도 남아 있다. 전체 게이트 직접 재실행 통과(19 패키지 ok, proxy 62.8s).
- 보안/법무 차단 없음: authorizeAdmin 그대로, Kinds 는 고정 목록+`lower(?)` 바인드(주입면 없음), 감사에 추가된 것은 카운트뿐, 새 개인정보·의존성 없음.
- 승인이어도 남는 우려(릴리즈 노트): 창 밖이라 조용했던 finding 이 첫 스캔에서 한꺼번에 나가고 이 키들은 dedup 원장에 없어 6h 윈도우가 완충이 안 된다 — finding 당 Mattermost 1건(레이트 리밋 없음)이라 채널 폭주 가능. 실 규모 미검증.
- 다음 회차: notifyScanBudget 2000 < posture 8000 인데 kind 집합은 넓어져 무인 경로가 더 잘 잘리고, truncated 를 알려주는 것이 감사 로그뿐(cron 은 truncation_notice 를 읽지도 따르지도 못함). 예산 상향 또는 truncated 알림화를 검토.
- [러너 06:28] review approved — 리뷰 승인 (risk=low)
- [러너 06:28] pr created — https://github.com/hkjang/clustara/pull/30
- [러너 06:29] ci passed — 검사 없음 — 정책으로 허용
- [러너 06:29] merge done — ceb42b9
- [러너 06:37] release published — v0.9.290
- [러너 06:37] gh-release created — GitHub Release v0.9.290
- [러너 06:37] manifest ok — clustara-v0.9.290.tar.gz clustara-v0.9.290.tar.gz.sha256 README-offline-v0.9.290.md 
- [러너 06:37] assets uploaded — 3개
- [러너 06:37] assets verified — v0.9.290 자산 3개 (이전 v0.9.289: 3)
