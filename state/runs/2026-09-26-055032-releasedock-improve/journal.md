# 회차 노트 2026-09-26-055032-releasedock-improve — releasedock
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 05:50] base pinned — main@d539386
- [러너 05:50] autonomy release — 
- [러너 05:59] scout timeout — 단계 제한 시간 초과
- [러너 05:59] scout done — 단순 모드 실행 상세의 라이브 로그 연결이 끊긴 것을 화면에 알리고 다시 시도할 수 있게 하기 (가치 3 / �

## 구현 노트
- SimpleRunDetailPage 의 스트림 effect 에 `error`/`open` 리스너를 더해, 브라우저가 포기한 스트림(readyState CLOSED)일 때만 로그 영역에 안내 Alert 와 "다시 연결" 을 띄운다. 조작은 저장 로그를 다시 수집한 뒤 스트림 하나를 `after=<lastId>` 로 연다. 커밋 f49181a.
- 확신 없는 곳: (1) 실제 브라우저에서 429 를 재현해 readyState 를 관찰하지 못했다 — CLOSED/CONNECTING 구분은 EventSource 스펙(비-200 → fail the connection, 네트워크 단절 → reestablish)에 근거한 판단이고 테스트 대역이 그 숫자를 흉내 낸다. (2) `reconnectStream` 의 `setStreamAttempt` 증가를 빼도 테스트 4건이 모두 통과한다(loadingLogs true→false 전이만으로 effect 가 다시 돈다) — 즉 그 한 줄은 테스트로 고정돼 있지 않다. React 가 두 전이를 한 배치로 합칠 경우를 막는 보험으로 일부러 남겼고, 남긴 상태에서도 스트림은 1개만 열린다(테스트가 `instances` 2개·open 1개로 고정).
- 일부러 하지 않은 것: 자동 재시도·폴링 없음(스트림을 거절한 그 한도를 되레 때린다). SimpleDeployPage·ReleaseDetailPage 의 같은 빈틈은 과제서 범위 밖이라 ideas.json 에 pending 으로 남겼다. 백엔드·문서·VERSION 무변경.
- 다음 역할이 조심할 것: `EventSource.CLOSED` 같은 정적 상수를 쓰지 말 것(테스트에서 전역이 대역으로 바뀌어 undefined). 리스너는 반드시 `addEventListener` — 대역이 EventTarget 이라 `onerror =` 는 호출되지 않는다. backend `go test ./...` 는 통과했으나 TEST_POSTGRES_DSN 이 없어 통합 테스트는 Skip 됐다(이번 변경은 웹 전용이라 그대로 뒀다).
- [러너 06:03] brief accepted — 채택 — 과제서의 근거(effect 에 error/open 리스너 없음, 서버 429 stream_limit, 전체 모드에만 connected 표시)가 지금 코드와 그대
- [러너 06:04] verify passed — 검증 7개 통과 (auto)
- [러너 06:04] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 06:04] pr created — https://github.com/hkjang/releasedock/pull/24
