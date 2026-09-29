# 회차 노트 2026-09-30-034258-releasedock-improve — releasedock
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:43] base pinned — main@f21b147
- [러너 03:43] autonomy release — 

## 정찰 노트
- 골랐다: 전체 모드에서 릴리즈를 옮기면 이전 릴리즈 로그가 화면에 남는 결함. 사용자가 보는 잘못된 데이터이고 프로덕션 1파일·테스트 1파일에서 끝난다. rAF 배치(성능 측정 없음)와 스트림 한도 테스트(비동기 해제 동기화 위험, DSN 필요)를 제쳤고, 업로드 스트리밍은 잠금 계약 변경이라 기각했다.
- 근거는 코드로 확인했다: ReleaseDetailPage.tsx:62-64 리셋 effect가 cursor만 0으로 돌림, useAsync가 로딩 중 이전 data 유지, LogPanel은 tab===1일 때만 마운트되고 enabled는 항상 리터럴 true. 단순 모드는 loadStoredLogs의 setLogs 전체 교체로 같은 결함이 없음도 확인했다.
- 추측으로 남긴 것: 웹 스위트를 실제로 돌리지 못했다(web/node_modules 없음). 기존 테스트 145행에 assertion 한 줄을 더하면 빨강이 된다는 것은 코드 독해에 근거한 예측이며 구현자가 먼저 빨강을 확인해야 한다.
- 구현자가 조심할 것: 초기화를 스트림 effect(streamAttempt 의존)에 넣으면 v0.5.24의 max_duration 재연결 로그 유지를 되돌린다 — 70행 테스트로 반증 실험할 것. sequence.current는 초기화 금지(React key 충돌).
- [러너 03:46] scout done — 다른 릴리즈로 이동할 때 이전 릴리즈의 실시간 로그가 화면에 남는 결함 수정 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- 무엇을 왜: `useReleaseLogs` 의 리셋 effect(ReleaseDetailPage.tsx:62, 의존성 `[releaseId, enabled]`)에 `setLogs([])` 와 `setConnected(false)` 를 더했다. `useAsync` 가 로딩 중 이전 릴리즈 data 를 유지해 LogPanel 이 언마운트되지 않으므로 릴리즈 A 의 줄이 B 화면에 남고 복사 버튼이 둘을 섞어 내보냈다. 프로덕션 1 파일 + 테스트 1 파일.
- 확신 없는 곳·검증 못 한 것: 백엔드는 `go test ./... -count=1` 이 전 패키지 ok 이지만 `TEST_POSTGRES_DSN` 이 없어 DB 통합 테스트는 SKIP 됐다 — 통합 검증이 아니다(이번 변경과 무관한 영역이다). `npm run build`(vite) 는 돌리지 않았고 `npx tsc -b --noEmit` 만 통과시켰다. 브라우저 실물 확인은 하지 않았고 근거는 실제 App 라우트를 지나는 렌더 테스트뿐이다.
- 일부러 하지 않은 것: 초기화를 스트림 effect(`streamAttempt` 포함)에 넣지 않았다 — 반증 실험에서 `retaining lines…` 테스트가 실패해 v0.5.24 의 max_duration 재연결 유지가 깨짐을 확인했다. `sequence.current` 는 React key 충돌을 막으려고 초기화하지 않았다. 단순 모드(`SimpleRunDetailPage`/`SimpleDeployPage`)는 `loadStoredLogs` 가 `setLogs` 로 전체 교체해 같은 결함이 없어 건드리지 않았다. `onerror` 의 readyState 미구분은 다음 회차로 남겼다.
- 다음 역할이 조심할 것: 검증 명령은 반드시 `web/` 에서 실행한다(루트에 `npm test` 없음). 이 워크트리는 `npm ci` 로 `web/node_modules` 를 새로 채웠다. `web/dist` 는 생성되지 않았고 `VERSION` 은 건드리지 않았다. 웹 122건 전부 통과(신규 테스트 파일 없이 기존 이동 테스트에 assertion 3건 추가).
- [러너 03:49] brief accepted — 채택 — 과제서의 근거(리셋 effect 가 `cursor.current = 0` 한 줄, 스트림 effect 의 `streamAttempt` 의존성, 기존 145행 테스트가 실제
- [러너 03:49] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인함: diff 2파일 전체, useReleaseLogs 의 두 effect 의존성, LogPanel 문자열. 돌연변이 검증 — 프로덕션 두 줄을 빼면 이동 테스트만 빨강(server line 57 잔존)이 되고 다른 9건은 초록, 원장 실패 재현과 일치. 원복 후 워크트리 clean. web 122건 전부 통과, tsc -b --noEmit 종료 0.
- 못 본 것: 실제 브라우저, 백엔드(diff 에 없음), DB 통합(DSN 없음). A→B 이동 중 useAsync 가 이전 data 를 유지하는 시간창은 테스트가 즉시 resolve 라 재현되지 않는다.
- 승인이어도 남는 우려: 리셋 의존성에 enabled 가 있어 탭 이탈→복귀에서도 화면이 비워진다(서버가 after=0 으로 재전송하므로 이전 중복 쌓임보다 낫지만 커밋 메시지 범위보다 넓고 테스트가 없다) — 릴리즈 노트에 한 줄 적을 것.
- 다음 회차: clear() 가 cursor 를 안 돌리는 점, 전체 모드 onerror 의 readyState 미구분이 그대로 남았다.
- 보안·법무 차단 없음: 인가·비밀값·의존성·마이그레이션 무관하고, 릴리즈 간 로그 혼입(복사 포함)을 줄이는 방향이다.
- [러너 03:52] review approved — 리뷰 승인 (risk=low)
- [러너 03:52] pr created — https://github.com/hkjang/releasedock/pull/29
- [러너 03:55] ci passed — 검사 1개 모두 success
- [러너 03:55] merge done — 46b11ea
- [러너 04:00] release published — v0.5.25
- [러너 04:03] assets verified — v0.5.25 자산 2개 (이전 v0.5.24: 2)
