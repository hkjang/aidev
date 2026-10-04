# 회차 노트 2026-10-05-005731-releasedock-improve — releasedock
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:57] base pinned — main@ea89496
- [러너 00:57] autonomy release — 

## 정찰 노트
- 골랐다: 전체 모드 로그의 CLOSED 안내 + 수동 재연결. 여섯 회차 밀렸던 아이디어지만 이번에 코드를 열어 보니 결함 강도가 기록보다 셌다 — `onerror`(:145)에 `stopped` 가드조차 없고, 단순 모드가 v0.5.24 에 고친 그 결함이 전체 모드에만 남아 있다. 재연결 배선(`streamAttempt`)과 커서 보존이 이미 있어 프로덕션 1 파일로 끝난다.
- 제친 것: 스트림 한도 통합 테스트(DSN·동시성 위험, M), rAF 배치(측정 방법 미정), make vet(머지 안 된 브랜치와 중복), 전체 모드 로그 내려받기(가치는 더 높지만 백엔드까지 L — ideas.json 에 쪼개는 방법을 적어 뒀다).
- 확인한 것: `:145` onerror, `:87/:140/:150` streamAttempt, `:89-95` 리셋 effect, `:316-345` LogPanel, `SimpleRunDetailPage.tsx:111` streamDisconnected export, `SimpleDeployPage.tsx:25,654,306` 선례, 릴리즈 테스트 대역 `:11-31` 에 readyState 없음, 단순 대역의 `emitError(readyState)` 패턴.
- 추측으로 적은 것: 끊김 Alert 의 정확한 문구(단순 모드 문구를 참고하라고만 적었다), 그리고 `end` 뒤 늦은 error 가 실제 브라우저에서 오는지 — 오지 않더라도 `stopped` 가드는 테스트로 고정할 값어치가 있다. 웹 테스트는 이번에 돌리지 않았다(기준선 140건은 지난 회차 기록).
- 구현자 주의: 단순 모드 파일은 읽기만 하고 수정하지 말 것(공용 util 로 옮기면 파일 4개). 자동 재연결 루프 금지 — 끊김의 주 원인이 사용자당 3스트림 한도다.
- [러너 01:02] scout done — 전체 모드 릴리즈 실시간 로그가 브라우저가 포기한 연결을 알리고 수동 재연결을 주기 (가치 3 / 위험 1 / 

## 구현 노트
- 무엇을: `useReleaseLogs` 의 `onerror` 가 `readyState` 와 `stopped` 를 보게 하고(단순 모드의 `streamDisconnected` 를 import 해 재사용), `streamLost` + `reconnect` 를 훅 반환값에 더해 LogPanel 이 끊김 경고와 `다시 연결` 버튼을 띄우게 했다. 왜: 429 `stream_limit` 로 브라우저가 포기하면 "로그 연결 대기" 만 영구히 남고 되살릴 방법이 없었다. 프로덕션 1 파일 + 테스트 1 파일, 커밋 7f9790f.
- 확신 없는 곳: ① `onopen` 의 `setStreamLost(false)` 는 도달 가능한 경로가 없다 — `reconnect` 가 이미 지우기 때문이다. 단순 모드 두 화면의 선례를 따라 넣었고 "열린 스트림 위에 끊김 경고가 뜨지 않는다" 를 구조로 보장하지만, 이 한 줄만 단독으로 고정하는 테스트는 없다. ② 끊김 플래그 초기화를 리셋 effect 에 둔 것은 테스트로 고정되지 않았다 — 스트림 effect 로 옮겨도 24건 전부 통과한다(반증 실험 4). 두 배치가 도달 가능한 경로에서 동일해서다. 과제서의 역할 분리 지시를 따랐다. ③ 정상 `end` 뒤에 실제 브라우저가 늦은 error 를 보내는지는 확인하지 못했다 — `stopped` 가드는 그 가정 없이도 옳다.
- 일부러 하지 않은 것: 자동 재연결 루프(끊김의 주 원인이 사용자당 3스트림 한도다 — 타이머가 그 한도를 되레 때린다), `streamDisconnected` 를 공용 util 파일로 옮기기(파일이 4개로 늘고 단순 모드 두 화면을 건드린다), 단순 모드 두 화면의 `stopped` 가드 부재(이번 회차에 발견 — ideas.json 에 적었고, 재현 없이는 손대지 말 것).
- 다음 역할 주의: ① `ReleaseDetailPage.tsx` 가 이제 `../simple/SimpleRunDetailPage` 를 import 한다(역방향 import 는 없어 순환은 아니고, `SimpleDeployPage.tsx:25` 의 선례와 같은 방향이다). ② 테스트 대역 `TestEventSource` 에 `readyState`·`emitError(readyState)` 를 추가하고 `close()` 가 `readyState = 2` 를 세우게 했다 — 기존 `emit(source, 'error')` 를 쓰는 테스트는 readyState 1 로 남아 영향받지 않는다. ③ `keeps warning after a manual reconnect` 는 5000줄을 두 번 렌더하므로 `SLOW_RENDER_TIMEOUT`(30초)이 필요하다. ④ 백엔드 0 파일 변경이라 Go 테스트는 돌리지 않았다 — 통합 검증으로 보고하지 않는다. ⑤ `web/dist` 는 커밋 전에 지웠고 `VERSION` 은 손대지 않았다.
- [러너 01:14] brief accepted — 채택 — 과제서가 지목한 모든 근거(`:145` 의 가드 없는 onerror, `:87/:140/:150` 의 streamAttempt 배선, `:89-95` 리셋 effect, `:316-345` L
- [러너 01:15] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인했다(직접 실행): main 의 프로덕션 파일로 되돌려 새 테스트 3건 빨강 재현 — 원장의 `실패 재현` 과 테스트명·사유까지 일치. 통과하던 가드 2건은 돌연변이(`if (!stopped && streamDisconnected(...))` → 무조건 `setStreamLost(true)`)로 **그 둘만** 실패함을 확인해 실효성을 고정. 복원 후 145건/13파일 통과, `tsc -b --noEmit` 통과, 트리 깨끗.
- 확인했다(읽기): HTML 스펙의 fail/reestablish 구분이 `streamDisconnected` 판정과 맞고, releases.go:1487 의 429 `stream_limit` 과 :1479 의 204 가 실제로 CLOSED 로 떨어져 과제가 주장한 경로가 실재한다. simple→releases 역방향 import 없음(순환 아님), App.tsx 에 lazy 없음, 모듈 부수효과 없음. 보안·법무 차단 사유 없음(새 경로·식별자·비밀값·개인정보·의존성 0).
- 못 봤다: 실제 브라우저에서 정상 `end {}` 뒤 늦은 error 가 오는지(구현자와 동일한 미확인 — `stopped` 가드는 그 가정 없이도 옳다), Go 테스트(백엔드 0 파일 변경), 4개 탭 동시 열기 실기동 재현.
- 승인이어도 남는 우려 ①: 단순 모드는 `streamLost && live` 로 게이트하지만(SimpleRunDetailPage.tsx:475) 전체 모드는 `streamLost` 만 본다(ReleaseDetailPage.tsx:375). `hasExecutionLogs` 가 SUCCESS/FAILED/ROLLED_BACK 를 포함하므로(:429) 끝난 릴리즈가 429 를 받으면 '릴리즈 실행 자체는 계속 진행됩니다' 가 틀린 문장이 된다. 변경 전엔 같은 자리에 침묵이 영구히 남았으니 순개선 — 차단 않고 별 과제로 남긴다.
- 우려 ②: `onopen` 의 `setStreamLost(false)`(:112)는 **실제로 도달 불가**(readyState 2 는 종단, 새 source 는 `reconnect` 가 이미 지운 뒤 생긴다) — 죽은 코드지만 선례와 같고 해롭지 않다. 리셋 effect 의 플래그 초기화는 '있다' 만 고정되고 '어디 있나' 는 계약이 아니다(원장 실험 ④) — 이 훅을 다음에 만지는 쪽이 알아야 한다.
- [러너 01:21] review approved — 리뷰 승인 (risk=low)
- [러너 01:22] pr created — https://github.com/hkjang/releasedock/pull/33
- [러너 01:25] ci passed — 검사 1개 모두 success
- [러너 01:25] merge done — 7f9790f
- [러너 01:32] release published — v0.5.29
