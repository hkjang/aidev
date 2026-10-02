# 회차 노트 2026-10-02-095806-visitflow-improve — visitflow
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 09:58] base pinned — main@0aa37db
- [러너 09:58] autonomy release — 
- [러너 09:58] scout skipped — 요청된 작업이 있어 정찰을 건너뛴다 (과제는 이미 정해져 있다)

## 구현 노트
- `TestEveryListReturnsItsRows`·`TestStatisticsBreakdowns` 가 `visitBody`(startAt=now+30m)로 방문을 심고 사이트 현지 날짜로 고르는 `/api/v1/lobby/today`·통계 span 에 나타나길 단정했다. 현지 23:30 이후엔 그 방문이 다음 날이어서 정당하게 빠진다 — 제품이 아니라 픽스처 결함이다. `startsTodayAtSite`/`visitToday` 가 사이트의 다음 현지 자정을 DB 에서 읽어 자정을 넘기는 대신 lead 를 줄인다. 프로덕션 코드 0개 파일.
- 확신 없는 곳: CI 로그 본문을 못 봤다(actions logs 는 인증 필요, 403). 원인은 **run 93(bc02eb5)과 run 94(0ebdfd6)의 트리가 동일(`git diff` 빈 출력)한데 14:28 UTC 통과 / 14:32·14:37 UTC 실패**라는 사실 + 재현으로 좁혔다. 사람 명세가 인용한 postgres 중복키 ERROR 두 줄은 `integration_test.go:697`·`:993` 의 의도된 중복 거절 테스트 소음이고 원인이 아니다. 이 두 테스트 외에 다른 테스트도 CI 에서 같이 실패했을 가능성은 배제 못 한다(23:10·22:10·20:10·00:10·12:10 다섯 지점 전체 실행은 전부 통과).
- 재현 하네스는 **사이트 timezone 을 POSIX 오프셋으로 바꿔 현지 시각을 옮기는** 방식이다(벽시계를 못 바꿔서). 수정이 timezone 을 건드리지 않으므로 하네스가 무력화되지 않는다 — 그게 Option B(timezone 고정) 대신 이 수정을 고른 이유다. 하네스는 커밋하지 않았고 `grep repro` 가 빈 출력임을 확인했다.
- 일부러 안 한 것: `visitBody` 자체를 사이트 달력 인식형으로 바꾸지 않았다 — 호출 지점 약 40곳이고 대부분 현지 날짜와 무관하다(ideas.json 에 보류로 남겼다). 서버 코드·웹·릴리즈 경로는 손대지 않았다.
- 다음 역할이 조심할 것: 두 테스트는 **PostgreSQL 이 있어야 돈다**(`VISITFLOW_TEST_DSN` 없으면 SKIP — "PASS" 가 실행을 뜻하지 않는다). `startsTodayAtSite` 는 `now()`·현지 자정을 **DB 에서** 읽으므로 호스트 시계와의 편차에 의존하지 않는다. endAt 은 현지 다음 날로 넘어갈 수 있는데 두 뷰 모두 start_at 으로만 고르므로 의도된 것이다.
- [러너 10:18] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 판정 approve. 임시 PostgreSQL 16 + `git archive HEAD` 사본으로 현지 23:59 하네스를 **직접 다시 만들어** 양방향 확인했다: 수정 적용 5/5 PASS, 두 호출 지점만 `visitBody` 로 되돌리면 3/3 FAIL 이고 문구가 원장의 `- 실패 재현:` 과 일치. 실제 DB 전체 `go test ./...` 통과(internal/app 58.9s). 리뷰 워크트리는 끝까지 clean.
- 제품 쪽도 읽었다: `visits.go:1662`·`admin.go:49,59` 가 사이트 현지 날짜로 고르고 `byHour` 만 `checked_in_at` 기준이며, `createVisitRecord`(visits.go:441)에 최소 사전통보 검증이 없고 QR 창이 `start-60m`(visits.go:763)이라 lead 축소가 체크인을 깨지 않는다 — 구현 설명과 코드가 일치.
- 못 본 것: CI 러너 자체(로그 403 은 나도 동일), 실제 Asia/Seoul 23:3x 벽시계, 프런트·e2e 브라우저. e2e 는 `/lobby/roster`(상태 기준)만 보므로 같은 노출이 없음을 소스로 확인했다.
- 승인이어도 남는 우려: `visitToday` 가 `extra` 의 `startAt`/`endAt` 를 조용히 덮어쓴다(integration_test.go:1659-1661). `visitBody` 는 여전히 달력 비인식(호출 지점 약 40곳)이라 새 today-기반 단정이 같은 구멍에 재진입 가능 — ideas.json 보류 항목을 다음 회차가 승격 검토할 것.
- 릴리즈/다음 회차 주의: `VISITFLOW_TEST_DSN` 없는 `go test ./...`(verify.json 1s)는 이 두 테스트를 실행하지 않는다. 'PASS' 를 실행으로 읽지 말 것.
- [러너 10:30] review approved — 리뷰 승인 (risk=low)
- [러너 10:31] pr created — https://github.com/hkjang/visitflow/pull/31
- [러너 10:35] ci passed — 검사 2개 모두 success
- [러너 10:35] merge done — 571ab67
- [러너 10:44] release published — v2.8.12
- [러너 10:45] assets verified — v2.8.12 자산 1개 (이전 v2.8.10: 1)
