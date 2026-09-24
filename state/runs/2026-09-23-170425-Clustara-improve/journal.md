# 회차 노트 2026-09-23-170425-Clustara-improve — Clustara
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 17:04] base pinned — main@907c8c1
- [러너 17:04] autonomy release — 

## 정찰 노트
- 클러스터 격리 계열(RCA·podsec·NodePressure)은 5b98765 까지 모두 해결돼 같은 유형의 남은 결함을 못 찾았다. 대신 notify scan 에서 **전달 전에 dedup 을 쓰는 순서 결함**을 코드로 확인해 골랐다(Mattermost 꺼짐·webhook 미설정·카테고리 mute 셋 다 조용히 반환).
- 제친 후보: quiet_hours 검증(잘못된 값이 알림을 막는 방향이 아니라 영향 작음), 인벤토리 Limit 2000(범위 설계가 M 급), seccomp/PSS 레벨(정책 게이트 파급 산정이 먼저), .github CI(보호 경로).
- 추측으로 적은 것: 실제 운영에서 "cron 을 먼저 켜고 Mattermost 를 나중에 구성한다"는 순서는 docs/ADMIN_GUIDE.md:369 의 안내에서 유추한 것이지 사용자 보고가 아니다. 응답 필드 추가 여부는 구현자 재량으로 남겼다.
- 구현자 주의: 플래그 변경 뒤 `server.invalidateMattermostCache()`(15초 캐시) 없이는 회귀가 원인 오진으로 실패한다. 전송은 비동기라 "안 보냄" 증명은 `select ... default` 로.
- 전달 가능 판정은 반드시 한 함수로 모아 `notifyMattermostTo` 도 같이 쓰게 할 것 — 두 경로가 같은 조건을 따로 읽으면 다음 회차에 어긋난다(운영자 지시).
- [러너 17:08] scout failed — 과제서 없음 — 구현자가 직접 고른다

## 구현 노트
- notify scan 이 전달 판정보다 먼저 dedup 윈도우를 기록해, Mattermost 꺼짐·webhook 미설정·카테고리 mute 중의 스캔이 finding 을 6시간 억제하고도 sent 로 보고하던 순서 결함을 고쳤다. 전달 판정은 `mattermostSnapshot.canNotify` 하나로 모아 `notifyMattermostTo` 와 스캔이 같은 조건을 읽는다(운영자 지시대로 한쪽만 넓히지 않음). 커밋 178bedb.
- 확신 없는 곳·검증 못 한 것: ① 응답·감사에 새 필드 `undeliverable` 을 추가했다 — admin_ui.go·SDK 에서 이 응답을 읽는 곳을 grep 으로 못 찾아 호환 위험은 낮다고 봤지만 브라우저로는 확인 못 했다. ② docs/ADMIN_GUIDE.md 알림 절에 한 문단을 더했다(docs_reference_test·release_gate_test 통과 확인). ③ 실 Kubernetes·실 PostgreSQL·실 Mattermost·브라우저는 전부 미검증(SQLite+httptest 만). ④ quiet_hours 분기는 예전처럼 dedup 이전에 반환하므로 손대지 않았다 — 조용한 시간 중 스캔도 윈도우를 안 쓴다는 기존 동작과 이제 일관된다.
- 일부러 안 한 것: podsec dedup 키의 Kind 누락, quiet_hours 범위 검증, 인벤토리 Limit 2000 — 각각 별개 결함이라 한 커밋에 섞지 않았다(보류로 기록). 버전·changelog 는 세션 규칙대로 건드리지 않았다.
- 다음 역할 주의: 새 테스트는 `openTestStore`(t.TempDir SQLite)만 쓰므로 외부 DB 불필요. 플래그를 바꾼 뒤 `invalidateMattermostCache()` 없이는 15초 캐시 때문에 오진한다(fixture 의 `setFlags` 가 대신 처리). 전송 증명은 httptest webhook 채널+`time.After`, 미전송 증명은 `select ... default`.
- [러너 17:15] verify passed — 검증 3개 통과 (policy)

## 비평 노트
- approve / low / blocking 없음: 4개 변경 파일과 호출 경로·dedup SQL·UI/SDK 소비자·문서·테스트를 검토했고 머지 차단 결함은 발견하지 못했다.
- go build ./... → go vet ./... → go test ./... 통과(일부 캐시), 알림 관련 테스트 -count=1 별도 통과; 새 테스트는 수정 전 sent 단언이 실패하는 구조이나 수정 전 실행은 하지 않았다.
- 남는 우려: webhook 미설정 전용 회귀 부재, 비동기 미전송 select/default의 한계, 설정 동시 변경 경합·HTTP 실패 시 dedup 소비는 남아 있다. 즉시 통지 설명은 기존 dedup·quiet hours 및 best-effort 전송의 제약을 따른다.
- 실 PostgreSQL·Kubernetes·Mattermost·브라우저 미검증. 인증·DDL·의존성 변경 없음; 소스 수정 없이 review.json과 이 노트만 기록했다.
- [러너 17:17] review approved — 리뷰 승인 (risk=low)
- [러너 17:18] pr created — https://github.com/hkjang/clustara/pull/28
- [러너 17:18] ci passed — 검사 없음 — 정책으로 허용
- [러너 17:18] merge done — 178bedb
- [러너 17:23] release published — v0.9.288
- [러너 17:23] gh-release created — GitHub Release v0.9.288
- [러너 17:23] manifest ok — clustara-v0.9.288.tar.gz clustara-v0.9.288.tar.gz.sha256 README-offline-v0.9.288.md 
- [러너 17:24] assets uploaded — 3개
- [러너 17:24] assets verified — v0.9.288 자산 3개 (이전 v0.9.287: 3)
