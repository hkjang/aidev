# 회차 노트 2026-09-30-165613-releasedock-improve — releasedock
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:56] base pinned — main@4b450c7
- [러너 16:56] autonomy release — 

## 정찰 노트
- 보류 아이디어의 [1/1/S] "복사 버튼이 5000행 잘림을 알리지 않음" 을 코드로 확인해 가치 3 으로 격상해 골랐다: api 에 릴리즈 로그 함수가 releaseLogStreamUrl 하나뿐(client.ts:470)이라 전체 모드에서 버려진 앞줄은 UI 어디에서도 되찾을 수 없는데, 복사 버튼은 aria-label="전체 로그 복사" 로 전체를 약속한다. 단순 모드에 이미 같은 개념(logTruncated)이 있어 설계 논쟁이 없다.
- 제친 후보: 스트림 한도 통합 테스트(M, TEST_POSTGRES_DSN 필요 + 동시 해제 동기화), rAF 배치(측정 방법 미정이라 수용 기준을 쓸 수 없음), onerror readyState(가치 1, 두 회차 연속 밀림), ci/release.yml 통합(보호 경로 — 기각), LogPanel enabled 정리(v0.5.24/25 가 고정한 의존성 배열을 가치 1 로 건드리게 됨 — 기각).
- 추측으로 적은 것(구현자가 확인할 것): (1) 표시 상한 4999 는 slice(-4998) 에서 계산한 값이고 실행으로 확인하지 않았다 (2) LOG_DISPLAY_LIMIT + 1 프레임을 하나의 act() 안에서 dispatch 하면 React 가 한 커밋으로 합쳐 렌더 테스트가 현실적인 시간에 끝난다는 전제 — 틀리면 시간을 측정해 보고하고 차선(SimpleDeployPage 동일 패턴)으로 갈 것 (3) web/node_modules 가 없어(vitest 부재 확인) 이번 회차에는 웹 테스트를 한 번도 돌리지 않았다. 기준선 122건은 이전 회차 기록이며 이번에 미확인.
- 구현자가 조심할 것: 잘림 상태 초기화는 리셋 effect([releaseId, enabled])에만 — 스트림 effect 에는 streamAttempt 가 있어 max_duration 재연결마다 날아가고 v0.5.24 의 결함이 되살아난다. sequence.current(React key, 단조 증가)를 잘림 카운터로 재사용하지 말 것. 안내 문구에 "내려받기" 를 쓰지 말 것 — 전체 모드에는 없다.
- [러너 17:01] scout done — 전체 모드 릴리즈 실시간 로그의 화면 상한 초과(오래된 줄 버림)를 화면과 복사 결과에 알리기 (가치 3 / �

## 구현 노트
- 무엇을·왜: 전체 모드 실시간 로그가 4999줄 상한을 넘겨 오래된 줄을 조용히 버리는데 되찾을 경로가 없다. `logs` 를 `{lines, truncated}` 한 객체로 합치고 순수 헬퍼 `appendReleaseLogLine` 이 자르기와 플래그를 함께 계산하게 한 뒤, warning Alert(표시 줄 수 포함)과 복사 문자열 **앞**의 안내를 붙였다. 프로덕션 1 파일(ReleaseDetailPage.tsx) + 테스트 1 파일.
- 범위 밖이었는데 함께 고친 것 1줄: 행 번호를 `setLogs` updater **안에서** 읽어, 한 배치로 들어온 프레임이 모두 flush 시점의 같은 카운터를 읽고 React key 가 전부 같아졌다(`Encountered two children with the same key`). 대량 emit 테스트가 드러냈고 방치하면 테스트 출력이 경고로 도배된다. `const id = (sequence.current += 1)` 로 밖에서 읽는다 — 비평가가 범위 판단을 원하면 이 한 줄이 후보다.
- 확신 없는 곳: (a) Alert 의 줄 수는 `logs.length.toLocaleString()` 이고 테스트도 같은 `toLocaleString()` 으로 기대값을 만든다 — 로케일이 다른 CI 에서는 양쪽이 같이 움직이므로 통과하지만, 화면 표기는 환경에 따라 달라진다(단순 모드 선례와 동일한 방식). (b) 안내 문구 한국어 표현은 내 판단이며 "내려받기" 는 전체 모드에 없어 쓰지 않았다. (c) 같은 updater-안에서-ref-읽기 패턴이 SimpleDeployPage·SimpleRunDetailPage 에도 있는지는 **확인하지 않았다**.
- 일부러 하지 않은 것: `cursor.current`/`advanceCursor`/`sequence.current` 의 단조 증가·서버 커서 계약, `aria-label="전체 로그 복사"`, 잘리지 않은 복사 문자열 형식, `onerror` 의 readyState 판정, SimpleDeployPage 의 같은 `slice(-4998)` — 모두 범위 밖으로 두고 ideas.json 에 남겼다.
- 다음 역할이 조심할 것: 잘림 렌더 테스트 5건은 프레임을 하나의 `act()` 안에서 5000회 dispatch 하고 4999행을 렌더하므로 각 1.6~4.7초가 걸린다. 그래서 `SLOW_RENDER_TIMEOUT = 30_000` 을 명시했다 — 느린 CI 에서 기본 5초로 되돌리면 타임아웃으로 깨진다. 파일 전체 약 16초. 클립보드는 `Object.defineProperty` 로 세우고 `afterEach` 에서 원복하며(`vi.restoreAllMocks()` 는 되돌리지 않는다) `vi.stubGlobal('navigator', …)` 는 쓰지 않았다.
- 검증: `cd web && npm ci` 후 `npm test -- --run` 131 passed (기준선 122 + 신규 9), `npx tsc -b --noEmit` 통과(테스트의 reduce 누산기 타입 오류를 실제로 잡아 고쳤다 — npm test 만으로는 통과했다), `npm run build` 통과, `git diff --check` 깨끗. 백엔드는 0 파일 변경이라 실행하지 않았다(통합 검증 주장 없음). `web/dist` 는 커밋 전에 삭제, `VERSION` 미변경.
- [러너 17:12] brief accepted — 채택 — 과제서의 근거(`slice(-4998)` 의 조용한 버림, 전체 모드 로그 API 가 스트림 하나뿐, 단순 모드 `logTruncated` 선례, 리�
- [러너 17:12] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인함: 변경 2파일 전체, 구현자가 의심한 (a)(b)(c) 세 자리, 그리고 독립 뮤테이션 3건으로 테스트가 헛돌지 않음을 직접 증명 — `sequence.current` 를 updater 안으로 되돌림 → same key 로 빨강, `>`→`>=` → 경계 2건 빨강, 안내를 뒤로 옮김 → 클립보드 1건 빨강. 뮤테이션은 모두 원복했고 트리는 clean.
- 실행함: web 전체 131 passed (13 files, 15.5s), `npx tsc -b --noEmit` exit 0, `git diff --check` clean. 경계 등가성도 손으로 확인 — 구 `slice(-4998)` 과 신 `limit 4999` 는 최대 보유 줄 수가 같다.
- 못 본 것: 백엔드·러너 테스트(0 파일 변경이라 미실행), 실제 브라우저에서의 4999행 렌더 체감과 로케일별 숫자 표기, 구현자가 (c)로 남긴 "같은 updater-안에서-ref-읽기 패턴이 SimpleDeployPage·SimpleRunDetailPage 에도 있는가" — **다음 회차 후보다**(전체 모드에서는 실재한 결함이었다).
- 승인이어도 남는 우려: ① truncated 일 때 Alert 숫자는 사실상 상수 4,999 ② `clear()` 가 안내까지 지워 이후 복사본이 유실을 숨긴다 ③ `aria-label="전체 로그 복사"` 는 여전히 '전체' 를 약속 ④ SLOW_RENDER_TIMEOUT=30_000 을 낮추면 느린 CI 에서 먼저 깨진다.
- 릴리즈 노트용: 전체 모드 실시간 로그가 표시 한도(4,999줄)를 넘기면 화면 경고와 복사본 머리말로 앞부분 유실을 알린다. 함께 고친 선행 결함 — 한 배치로 도착한 로그 줄들이 같은 React key 를 받던 문제.
- [러너 17:17] review approved — 리뷰 승인 (risk=low)
- [러너 17:17] pr created — https://github.com/hkjang/releasedock/pull/30
- [러너 17:20] ci passed — 검사 1개 모두 success
- [러너 17:20] merge done — 31d92f5
- [러너 17:27] release published — v0.5.26
- [러너 17:30] assets verified — v0.5.26 자산 2개 (이전 v0.5.25: 2)
