# 회차 노트 2026-10-16-192831-releasedock-improve — releasedock
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:28] base pinned — main@da054c9
- [러너 19:28] autonomy release — 

## 정찰 노트
- 2026-10-09 회차가 고른 같은 과제(실행 ID 격리)를 다시 골랐다. 그 회차는 브랜치 3개가 모두 main 과 동일한 채 끝나(`git log main..auto/2026-10-09-*` 가 공백) 구현이 아예 없었던 것이고 사람 반려가 아니다. 코드로 결함을 다시 확인했다: `SimpleRunDetailPage.tsx:185`(useParams 로 id 만 교체)·`:204-223`(loadStoredLogs 에 요청 유효성 가드 없음)·`:239`(스트림이 lastIdRef 를 그대로 after 로 씀)·`:435`(형제 링크가 같은 라우트의 다른 id). `useAsync.ts:14` 는 dep 변경에 data 를 지우지 않는다.
- 다른 후보를 제친 이유: 문서 불일치(Harbor)·복사 문구는 가치 1~2 이고, 스트림 한도 통합 테스트와 릴리즈 로그 DB 픽스처는 `TEST_POSTGRES_DSN` 이 필요해 S 로 끝나지 않는다. 헬퍼 개명과 형제 링크 인코딩은 이번에 `rejected` 로 내렸다(동작 변화 0 / 재현 불가).
- 과제서에서 추측으로 적은 것: `key={id}` 재마운트가 만드는 PageLoading 순간 노출이 어떤 기존 테스트에도 걸리지 않는다고 봤다 — 근거는 `SimpleRunDetailPage.stream.test.tsx` 12건 중 실행 간 이동을 하는 테스트가 없다는 것(`fireEvent.click` 은 전부 `새로 고침`·`다시 연결`)이고, `SimpleRunDetailPage.test.ts`(순수 헬퍼 단위)와 다른 페이지 테스트는 열지 않았다.
- 이번 정찰은 테스트를 실행하지 않았다(예산). 기준선 149건/13파일은 v0.5.31 기록에서 가져온 수치이며 이번에 실측하지 않았으므로 구현자가 먼저 실측할 것.
- 조심할 것: 스트림 effect(`:237-277`)의 의존성·`end`/`error` 처리와 export 된 순수 헬퍼 시그니처는 손대지 말 것. `useAsync` 를 전역으로 고치는 우회로는 v0.5.24 의 전체 모드 테스트에 걸린다. 프로필(2026-10-09)은 지금 코드와 어긋나는 점이 없어 새로 쓰지 않았다.
- [러너 19:32] scout done — 단순 실행 상세를 실행 ID 로 키를 준 내부 컴포넌트로 감싸 로그·커서·지연 응답을 실행마다 격리하기 (�

## 구현 노트
- 과제서 그대로 구현했다. `SimpleRunDetailPage` 본문을 `function RunDetail({ id })` 로 옮기고 export 된 래퍼는 `<RunDetail key={id} id={id} />` 만 반환한다 — 형제 패키지 링크로 실행이 바뀔 때 React 가 재마운트해 로그·안내·`lastIdRef` 커서가 함께 새로 시작하고, 이전 실행의 지연 응답은 사라진 컴포넌트로 resolve 된다. 프로덕션 1 파일 +13줄(로직 2줄 + 주석), 테스트 1 파일 +187줄. 상태 reset effect 는 쓰지 않았다.
- 확신 없는 곳: (1) 신규 테스트 4건 중 `leaves no line…` 은 로그 줄 수 표시 `· 0줄` 을 판정에 쓴다 — 텍스트 매칭이라 그 Typography 문구가 바뀌면 조용히 부정확해질 수 있다(다른 3건은 커서 URL·Alert 문구로 고정돼 더 단단하다). (2) `followSiblingLink` 를 `act` 로 감싸 React 경고를 없앴는데, 그 감싸기 자체가 경쟁 상태를 가려 테스트를 관대하게 만들지는 않았는지는 반증 실험(`key` 제거 시 4건 전부 실패)으로만 확인했다. (3) 실제 브라우저에서 눌러 보지는 않았다 — jsdom + 실제 App 라우트까지만이다.
- 일부러 하지 않은 것: `useAsync` 수정(전역 영향, v0.5.24 전체 모드 테스트가 data 유지 동작에 걸려 있다), 스트림 effect `:237-277` 의 의존성·`end`/`error` 처리, export 된 순수 헬퍼 시그니처, 형제 링크의 `encodeURIComponent`(런 id 계약상 재현 불가), 백엔드·`SimpleDeployPage`·`ReleaseDetailPage` — 모두 0줄.
- 다음 역할이 조심할 것: 이 변경은 **의도한 동작 변화 하나**를 만든다 — 형제 링크로 이동한 직후 `PageLoading`("실행 기록을 불러오는 중입니다")이 잠깐 보인다(이전에는 `run.loading && !run.data` 가드로 A 의 상세가 계속 보였다). 기준선 149건 전부 그대로 통과하므로 이것에 걸리는 기존 테스트는 없고, 올바른 격리의 대가이니 깜빡임을 이유로 되돌리지 말 것. 검증은 `cd web` 안에서만(루트 `npm test` 금지): `npm ci` → `npm test -- --run`(153건/13파일, 약 22초) → `npx tsc -b --noEmit` → `npm run build`. 백엔드 0 파일 변경이라 Go 테스트는 돌리지 않았다 — DB 도 필요 없다. `web/dist` 는 커밋 전에 지웠고 `VERSION` 은 건드리지 않았다.
- [러너 19:40] brief accepted — 채택 — 과제서가 지목한 근거 전부가 지금 코드와 그대로 일치했고(`:185` useParams, `:204-223` 가드 없는 `loadStoredLogs`, `:239` `
- [러너 19:40] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 판정 approve / risk low / blocking 없음. 반증 실험으로 테스트 유효성을 직접 확인했다: `SimpleRunDetailPage.tsx:195` 의 `key={id}` 만 빼면 신규 4건이 전부 실패(11건 통과)하고, 실패가 결함 증상과 일치했다 — run-b 스트림이 `after=5`(A 의 커서)로 열림, 남은 연결 끊김 경고와 `다시 연결` 버튼, A 의 로그 오류, 화면에 남은 `line 5/6`. 실험 후 `git status` 로 트리 복원을 확인했다.
- 구현자의 불안 3개 중 (1)은 과도했다 — 그 테스트가 실제로 실패한 지점은 `· 0줄` 이 아니라 `queryByText('line 5'/'line 6')`(테스트 212-213줄)이라 문구 변경에 덜 취약하다. (2) `act` 가 그대로인 상태에서 4건이 실패하므로 관대해지지 않았다. (3) 실브라우저 미검증은 남지만 React 재마운트 의미론에 jsdom 특이성이 없어 수용.
- 실측: `web/` 안에서 `tsc -b --noEmit` 통과, `vitest run` 153건/13파일 통과(23초, 구현 노트와 일치), `npm run build` 성공 후 `dist` 삭제. 백엔드 0파일이라 Go/DB 테스트는 **돌리지 않았다**(미실행을 성공으로 쓰지 않음) — 그래서 백엔드 쪽은 이번에 못 본 영역이다.
- 보안·법무: 라우트는 여전히 `App.tsx:41` 의 `RequirePermission "simple.read"` 안이고 새 경계는 가드 **내부**라 인가가 넓어지지 않는다. 신규 엔드포인트·비밀값·의존성·개인정보 없음. export 헬퍼 시그니처 무변경이라 `ReleaseDetailPage.tsx:45`·`SimpleDeployPage.tsx:25` 소비자 영향 0.
- 남는 우려(릴리즈 노트용): 형제 링크 이동 직후 `PageLoading` 순간 노출은 의도된 대가이니 되돌리지 말 것. 다음 회차: 같은 종류의 id 격리 결함이 `SimpleDeployPage`·`ReleaseDetailPage` 에도 있는지는 이번에 보지 않았다.
- [러너 19:44] review approved — 리뷰 승인 (risk=low)
- [러너 19:44] pr created — https://github.com/hkjang/releasedock/pull/36
- [러너 19:47] ci passed — 검사 1개 모두 success
- [러너 19:48] merge done — 6b8b4fb
- [러너 19:53] release published — v0.5.32
- [러너 19:56] assets verified — v0.5.32 자산 2개 (이전 v0.5.31: 2)
