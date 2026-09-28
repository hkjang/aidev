# 회차 노트 2026-09-28-194203-releasedock-improve — releasedock
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:42] base pinned — main@79a00da
- [러너 19:42] autonomy release — 

## 구현 노트
- 무엇을 왜: SimpleDeployPage 의 업로드 중 라이브 로그가 `log`·`end` 만 청취하고 커서 없이 열려, 스트림이 끊기면(429 stream_limit / 세션 만료 / 서버 30분 상한) 안내도 재연결도 없이 로그만 영구히 멈췄다. 실행 상세가 이미 export 하는 `streamDisconnected`·`streamEndedRun` 을 재사용해 CLOSED 안내 + 수동 재연결 + `?after` 커서를 넣었다. 프로덕션 1 파일(SimpleDeployPage.tsx), 테스트 1 파일(신규).
- 확신 없는 곳·검증 못 한 것: (1) 실제 브라우저에서 확인하지 않았다 — 전부 jsdom 렌더 테스트이고 EventSource 만 대역이다. 실서버 429 를 브라우저가 CLOSED 로 만드는지는 기존 v0.5.21 판단을 그대로 승계했고 이번에 다시 재현하지 않았다. (2) 여러 파일을 연속 업로드하는 배치에서 파일 경계를 넘을 때의 커서·안내 초기화는 `setActiveRunId` 직전 한 줄로 했지만, 다중 파일 배치 자체를 지나는 테스트는 없다(테스트는 모두 1 파일). (3) 로그 행 표시 id 를 단조 ref 로 바꾼 것은 전용 테스트가 없다 — 충돌을 결정적으로 재현하기 어려워 회귀 가드가 아니라 예방 목적이다.
- 일부러 하지 않은 것: 자동 재시도 루프(스트림을 거절한 그 한도를 되레 때린다 — 수동 "다시 연결" 하나만 줬다). 재연결 시 저장 로그 재수집(실행 상세는 하지만 배포 화면은 불필요하다 — 서버 SSE 가 `id > after` 를 DB 에서 읽으므로 커서 재연결만으로 끊긴 구간이 채워진다). rAF 배치 setLogs(별건, ideas.json 에 pending).
- 다음 역할이 조심할 것: 새 테스트는 DB 불필요(웹 전용, `cd web && npx vitest run`)이지만 `npm ci` 가 선행돼야 한다(worktree 에 node_modules 가 없었다). 백엔드 Go 파일은 0 개 변경이므로 backend 결과는 회귀 가드용이다 — 다만 DSN 없이 돌리면 통합 테스트가 조용히 SKIP 되니 도커 PostgreSQL 16 을 띄울 것. `drops the lost-connection notice once the queue has finished with the run` 는 실제 폴링 간격(1500ms)을 기다리므로 waitFor timeout 5000ms 에 의존한다 — 느린 CI 에서 흔들릴 수 있는 유일한 테스트다. SimpleDeployPage 가 이제 SimpleRunDetailPage 에서 두 헬퍼를 import 한다(역방향 import 는 없어 순환은 아니다).
- [러너 19:58] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인함: diff 2파일 전부, 서버 `streamSimpleRunLogs`(JSON data 에 `id` 가 실제로 있어 커서가 성립), `streamDisconnected`/`streamEndedRun` 원본, 라우팅(순환 import 없음). 워크트리에서 `npx tsc -b --noEmit` 0, `npx vitest run` 112/112 통과를 직접 재현. 신규 테스트는 실제 라우트·실제 큐를 타고 `after=0`/`after=7`/`after=4`·안내·재연결을 단언하므로 main 코드에서는 통과할 수 없다(main 은 URL 에 쿼리도 `error` 리스너도 없다). 원장의 `실패 재현` 3건과 4가지 되돌림 실험도 증상과 일치한다.
- 못 봄: 실제 브라우저 동작(429→CLOSED), backend/runner Go 테스트 재실행(변경 0파일이라 생략), 다중 파일 배치 실경로.
- 승인이어도 남는 우려: ① 파일 경계에서 `logCursorRef` 리셋과 새 effect 사이에 이전 런 스트림이 프레임을 흘리면 다음 런이 `after=<이전 id>` 로 열려 앞부분이 누락될 수 있다(좁은 창, main 대비 이론상 퇴행 — 다음 회차에 2파일 테스트 권장). ② `nextLineId()` 가 `setLogs` 업데이터 안에서 ref 를 변이한다(현재 무해, 순수성 냄새). ③ 배포 화면 effect 에 종료 상태 가드가 없어 서버 `end` 프레임 형태에만 의존한다 — 릴리즈 노트에 남길 것.
- 거절 아님이므로 수리 대상 파일 없음.
- [러너 20:02] review approved — 리뷰 승인 (risk=low)
- [러너 20:02] pr created — https://github.com/hkjang/releasedock/pull/27
