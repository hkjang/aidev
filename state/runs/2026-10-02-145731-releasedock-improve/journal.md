# 회차 노트 2026-10-02-145731-releasedock-improve — releasedock
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:57] base pinned — main@56d1cb3
- [러너 14:57] autonomy release — 

## 정찰 노트
- 보류 아이디어 중 배포 화면 잘림 안내를 골랐다. 재평가 중에 기존 평가(가치 2, "실행 상세로 가면 다시 볼 수 있어 약하다")가 틀렸음을 코드로 확인해 가치를 3 으로 올렸다 — `setLogs([])` 가 `:338` 에서 배치 시작 때 한 번만 돌아 4999줄 상한이 **배치 전체**에 걸리고(패키지 사이는 `:361` 구분선뿐), 그 화면에는 복사 버튼도 실행 상세 링크도 없다. 스트림 한도 통합 테스트(차선)는 `TEST_POSTGRES_DSN` 과 동시 해제 동기화가 필요해 밀었고, `onerror` readyState 는 결함 강도가 약해(네 회차 연속 밀림) 또 밀었다.
- 확신 없는 곳: ① 웹 기준선 테스트 수를 **실측하지 않았다** — `web/node_modules` 가 없어 `npm ci` 를 돌리지 않았다. 과제서의 131건은 v0.5.26 기록이고, 구현자가 먼저 기준선을 찍어야 한다. ② `backend/internal/server/server.go` 의 `acquireLogStream`/`releaseLogStream` 본문은 이번에 열지 않았다(차선 후보 근거는 기록 기반, 미확인). ③ `ReleaseDetailPage` 의 `onerror` 도 다시 열지 않았다(미확인).
- 구현자가 조심할 것: 실행 상세의 `LOG_TRUNCATED_NOTICE`(`SimpleRunDetailPage.tsx:51`)를 **문구째 재사용하면 거짓말이다** — 그쪽은 `처음 N줄만` 즉 뒤쪽을 버리고, 배포 화면은 `slice(-4998)` 로 앞쪽을 버린다. 헬퍼를 `releases/ReleaseDetailPage.tsx` 에서 import 해 모드 경계를 섞지 말 것(타입도 다르다). `nextLineId()` 를 `setLogs` updater 안으로 끌어들이면 v0.5.26 이 고친 React key 중복이 재발한다. 잘림 초기화는 `:338` 에만, 스트림 effect(`[activeRunId, streamAttempt]`, `:265`)에는 넣지 말 것.
- 프로필은 2026-09-29 자로 3 일 전이고 구조·위험 구역 서술이 이번에 본 코드와 일치해 다시 쓰지 않았다. 다만 머리글의 `HEAD 103db16, v0.5.24` 는 이제 `main@56d1cb3, v0.5.26` 이고, `git branch --no-merged main` 에는 `auto/2026-09-06-1820`·`auto/2026-09-17-0853`·`auto/2026-09-18-1213` 세 개가 남아 있다(프로필은 하나만 적었다).
- [러너 15:02] scout done — 단순 모드 배포 화면의 라이브 로그가 오래된 줄을 조용히 버리는 것을 읽는 사람에게 알리기 (가치 3 / 위

## 구현 노트
- 배포 화면 라이브 로그의 `slice(-4998)` 가 앞쪽 줄을 조용히 버리던 것을 알린다. 로그 상태를 `{lines,truncated}` 로 합치고 순수 export 헬퍼 `appendDeployLogLine` 하나로 다섯 군데 추가(스트림 2 + 구분선 + 오류 2)를 모았다 — 구분선·오류 세 경로는 상한이 아예 없어 버퍼가 4999줄을 넘어 자랄 수 있었다.
- 확신 없는 곳: ① 구분선·오류 두 경로가 **헬퍼를 지난다는 것은 타입(`DeployLogState`)과 `tsc -b --noEmit` 로만 보장**되고, 그 세 줄이 상한을 넘기는 장면을 end-to-end 로 돌린 테스트는 없다(그 경로로 4999줄을 채우려면 렌더 테스트를 하나 더 늘려야 해서 과제서의 비용 지침을 따랐다). ② Alert 문구의 `실행 기록`·`로그 내려받기` 는 `AppShell.tsx:79` 와 `SimpleRunDetailPage.tsx:331` 에서 확인한 실제 라벨이지만, 배포 화면에서 그 화면으로 가는 **링크는 넣지 않았다**(범위 밖) — 문구만으로 안내한다. ③ 과제서가 "`nextLineId()` 는 updater 밖에서 호출된다" 고 한 것은 코드와 달랐다. 다섯 군데 모두 updater **안**이었고 이번에 밖으로 꺼냈다. React key 중복이 실제로 났던 흔적은 못 봤고(각 updater 호출이 ref 를 증가시키므로 id 는 겹치지 않는다) 전체 모드 v0.5.26 과 같은 모양으로 맞춘 예방 조치다 — 이 한 줄들에 대한 전용 테스트는 없다.
- 일부러 하지 않은 것: `ReleaseDetailPage` 의 `appendReleaseLogLine` 재사용(모드 경계·`LogLine`/`LogEntry` 타입 차이), 실행 상세 링크 추가, 복사 버튼, `LOG_TRUNCATED_NOTICE` 문구 재사용(그쪽은 뒤쪽을 버리는 반대 방향이라 거짓말이 된다).
- 다음 역할이 조심할 것: 새 렌더 테스트는 4999 프레임을 흘려 약 2.3초 걸리고 기본 5초 타임아웃을 넘기므로 `it(..., 20000)` 로 늘려 두었다(이 저장소에서 명시 타임아웃을 쓴 첫 테스트다). 백엔드는 0 파일 변경이라 Go 테스트를 돌리지 않았다 — 검증은 `web` 에서 `npm ci`(선행 필수) → `npm test -- --run` 135건 / `npx tsc -b --noEmit` / `npm run build` 뿐이고, 기준선은 실측 131건이었다. `web/dist` 는 커밋 전에 지웠다.
- [러너 15:11] brief accepted — 채택 — 과제서의 근거(`:232,234` 의 조용한 `slice(-4998)`, `:338` 의 배치 1회 초기화, `:361/:378/:387` 의 무제한 추가, 실행 상세 �
- [러너 15:11] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 구현자가 의심한 세 자리를 모두 추적했다. ① 구분선·오류 경로는 타입만의 보장이 아니다 — `grep "logs" SimpleDeployPage.tsx` 로 `logs` 참조가 6곳(199/306/603/610/612/644)뿐임을 확인했고 전부 새 상태를 지난다. ② Alert 문구의 약속을 끝까지 따라가 AppShell.tsx:79 → SimpleRunDetailPage.tsx:331 → client.ts:585 → simple.go:1330 `downloadSimpleRunLog`(LIMIT 없음)까지 실물로 확인했다, 거짓 아님. ③ `nextLineId()` 하이스팅은 동작 중립(각 호출이 ref 증가 → id 단조 증가)이라 거절 사유가 아니다.
- 경계는 올바르다: `> limit` 이 '한도 도달'과 '밀려남'을 구분하고 순수 테스트 1·2 가 양쪽을 고정한다. 렌더 테스트는 구분선 1 + 4999 = 정확히 한 줄 초과를 흘려 구분선 소멸과 `line 4999` 생존을 동시에 단언하므로 변경 경로를 실제로 지난다(수정 전엔 export 자체가 없어 통과 불가). `logs.lines.length` 를 쓰는 Alert 는 truncated 가 참인 동안 버퍼가 limit 에 고정되므로 limit 과 영구히 같다.
- 실측 검증: `npm test -- --run` 135건 통과(신규 렌더 2366ms / 타임아웃 20000ms), `npx tsc -b --noEmit` exit 0, `npm run build` 성공, `web/dist` 삭제 후 `git status` 깨끗. 보안·법무 모두 소견 없음(클라이언트 표시 로직뿐, 공격 경로·개인정보 변화 없음).
- 승인이어도 남는 우려: ① `simple.deploy` 만 있고 `simple.read` 가 없는 사용자에겐 '실행 기록' 메뉴가 숨겨져 Alert 가 막힌 길을 가리킨다(이번 결함 아님, 다음 회차 후보). ② `startSimpleRun` 실패/`RunStateUnknown` 의 클라이언트 생성 줄은 밀려나면 다운로드로 복구 불가하나 큐 항목 오류(`:558`)로 화면에 남는다. ③ docs/simple-mode.md 에 배포 화면 4999줄 한도 서술이 없다 — 선례 31d92f5 도 같았으니 관례 일치, 릴리즈 노트에서 다룰 것.
- [러너 15:16] review approved — 리뷰 승인 (risk=low)
- [러너 15:16] pr created — https://github.com/hkjang/releasedock/pull/31
- [러너 15:19] ci passed — 검사 1개 모두 success
- [러너 15:19] merge done — 8e354f9
- [러너 15:27] release published — v0.5.27
- [러너 15:30] assets verified — v0.5.27 자산 2개 (이전 v0.5.26: 2)
