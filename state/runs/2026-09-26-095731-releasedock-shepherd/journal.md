# PR 처리기 노트 2026-09-26-095731-releasedock-shepherd — releasedock PR #24
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-26-055032-releasedock-improve)
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

## 수리 노트
- 지적 3건 다 맞았다. streamLost 를 내리는 유일한 곳이 'open' 리스너였고, 종료된 run 은 스트림을 열지 않으므로 그 프레임이 영원히 안 온다 — 새 테스트 3건이 수정 전 전부 "안내가 남아 있다" 로 실패하는 것을 먼저 확인했다(틀린 지적 없음).
- 고침: loadStoredLogs 성공 경로에 setStreamLost(false), 안내 조건을 `streamLost && live` 로. 커밋 1561c2f(새 커밋, amend/force 없음).
- 두 수정을 각각 되돌려 서로 다른 테스트가 실패함을 확인했다 — (a) 는 '끊김 뒤 RUNNING 유지 + 새로 고침', (b) 는 '끊김 뒤 SUCCESS + 로그 재수집 실패'. 어느 쪽도 다른 쪽에 가려지지 않는다.
- 확신 없는 곳: 세 번째 테스트(재수집 실패)는 (b) 를 고정하려고 고른 경로라 다소 인위적이다 — 다만 종료된 run 에서 '다시 연결' 이 아무 스트림도 못 여는 것은 실제 동작이다. 앞 회차의 '확신 없는 곳'(실제 브라우저 429 의 readyState 미관측, reconnectStream 의 setStreamAttempt 가 테스트로 고정되지 않음)은 이번 범위 밖이라 그대로 남아 있다.
- 검증: tsc 0, npm test 105/105, backend·runner go test 통과(DSN 없어 통합은 Skip). 백엔드·문서·VERSION 무변경.

## 심사 노트
- 확인한 것: npm ci 후 tsc 0 / npm test 105·105 통과. 프로덕션 파일만 origin/main 로 되돌리면 새 테스트 7건 중 6건 실패(남는 1건은 음성 단언이라 정상). 거절이 지목한 경로 — emitError(2) → SUCCESS → '새로 고침' → 안내·'다시 연결' 소멸 — 이 테스트로 고정됐다.
- 두 수정을 각각 되돌려 독립성 확인: setStreamLost(false) 제거 → 'RUNNING 유지 + 새로 고침' 1건만 실패, `&& live` 제거 → '종료된 run + 재수집 실패' 1건만 실패. 서로 가리지 않는다.
- 못 본 것: 실제 브라우저에서 429 의 readyState 를 관측하지 못했다(스펙 근거 판단). reconnectStream 의 setStreamAttempt 증가는 여전히 테스트로 고정되지 않았으나, 열린 스트림 1개 단언이 중복을 막아 해롭지 않다. 다른 run 으로 이동 + 저장 로그 수집 실패 시 안내가 잠시 남는 잔가지는 남겨 뒀다.
- 보안·법무: 새 공개 경로·인증 분기·권한 확대·의존성·비밀값·개인정보 처리 없음. 재연결은 simple.read 로 게이트된 기존 SSE 경로 그대로. 차단 소견 없음.
- 권고 merge / risk low — 변경은 web 2개 파일뿐이고 마이그레이션·외부 상태가 없어 revert 로 완전히 되돌아온다.
