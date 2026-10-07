# 회차 노트 2026-10-08-053942-Vendra-improve — Vendra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 05:39] base pinned — main@d2b41c8
- [러너 05:39] autonomy release — 

## 정찰 노트
- 골랐다: 보류 1순위(`ObjectList.load` 의 실패 처리). 「가려진 지출 0」(3/3/M)은 여덟 회차째 계약 결정이 선행해 또 제쳤고, CI node-version 고정·build PATH 는 구현 세션에서 실패를 재현할 수 없어(지난 회차가 실측) 제쳤다. 차선(저장된 보기의 order)은 같은 파일이지만 「요청값·적용값 중 무엇을 저장할지」 결정이 선행한다.
- 보류 메모를 그대로 믿지 않고 코드를 열어 **메모가 틀린 것 하나를 고쳤다**: `components.tsx:21-31` 의 `Loading` 은 이미 12초 뒤 탈출구를 낸다 — 「영원히 돈다」는 과장이다. 실제 결함은 12초간 실패를 로딩으로 주장하고, 그 뒤 원인을 추측하고(403 에 「연결이 끊겼을 수 있습니다」), `window.location.reload()` 가 같은 거부로 되돌아오며, `APIError` 의 message/status/code 가 전부 버려지는 것이다. 과제서의 「왜」를 이 사실로 다시 썼다.
- 추측으로 적은 것(구현자가 확인할 것): 웹 기준선 **23 files / 103 tests** 는 2026-10-07 회차의 실측 기록을 인용한 것이고 이 세션에서는 돌리지 않았다(`web/node_modules` 없음 — 없는 것만 확인했다). `npm ci` 가 먼저이고 몇 분 걸린다.
- 조심할 것 둘: ① `restoreMocks` 환경에서 **거부를 만드는 첫 Objects 테스트**다 — `beforeEach` reset 금지(objects-order.test.tsx:40-42 의 경고), 떠다니는 rejection 을 남기지 말 것. ② 실패를 `requestKey` 와 함께 담을 것 — `load` 시작에서 지우는 방식은 필터 변경 직후 한 프레임 동안 옛 실패가 번쩍인다(`loading` 의 정의 :298 은 건드리지 말고 분기만 더한다).
- 프로필을 다시 썼다: Objects.tsx 의 좌표가 지난 회차 +46줄로 전부 밀렸고(`load` :196→:209, `saveView` :250→:268, 정렬 select 는 `objectOrderOptions` 표로 올라갔다) `components.tsx`·`api.ts`·WorkInbox 의 실패 처리 본보기가 프로필에 없었다.
- [러너 05:46] scout done — 업무 목록 조회가 실패했을 때 화면이 **서버가 말한 이유**를 말하고 다시 시도할 수 있게 하기 — 지금은

## 구현 노트
- 무엇을 왜: `ObjectList.load` 에 `.then` 과 똑같은 시퀀스 가드를 쓰는 `.catch` 를 달아, 거부된 목록 요청이 12초간 「불러오는 중」을 주장하고 원인을 추측하는 대신 서버가 보낸 문구를 그대로 내고 `load()` 를 다시 부르는 「다시 시도」를 주게 했다. 실패는 `result` 와 같은 관용구로 `requestKey` 와 함께 담았다. 프로덕션 1파일(+59/-4) + 신규 테스트 1파일, commit 28572eb.
- **확신 없는 곳·검증 못 한 것**: ① `!loading && failed` 배너(행이 떠 있는데 새로고침만 실패한 경우)는 **테스트로 덮지 못했다** — 도달하려면 `onSaved`/`ObjectTable.onSubmit` 을 지나야 하고 그러려면 `post` 까지 모킹해야 한다(이 저장소의 하네스는 `api` 만 바꾸고 `post` 는 원본 `api` 를 닫아 잡는다). 새 테스트 6개는 전부 `loading && failed` 쪽만 지난다. 비평가는 여기부터 보라 — ideas.json 에 후보로 적어 뒀다. ② 브라우저 실물로는 확인하지 않았다(jsdom 만). ③ Go 통합 테스트는 돌리지 않았다 — Go 코드를 한 글자도 건드리지 않아 `gofmt`/`go vet` 만 회귀 확인용으로 돌렸다.
- 과제서와 다르게 한 한 곳: 기본 문구를 `"목록을 조회하지 못했습니다"` 로 했다(과제서는 `"목록을 불러오지 못했습니다"`). 과제서가 지정한 `<Empty title>` 과 글자가 같아 제목·설명에 같은 문장이 두 번 찍히고 실제로 `Found multiple elements` 로 터졌다; WorkInbox 가 바로 이것을 피해 둘을 다르게 쓴다.
- 일부러 하지 않은 것: 401 특별취급·리다이렉트(범위 밖), 공급업체/저장된 보기의 `.catch(() => {})`, `saveView` 의 `filters.order`, 공용 `Loading`·`Empty` 수정, `loading` 의 정의(:298) — 분기만 더했다.
- 다음 역할이 조심할 것: `.then` 에 더한 `setFailure` 는 **함수형 갱신**이다(실패가 없으면 같은 객체를 돌려줘 성공 경로의 렌더 횟수가 그대로다) — 단순 대입으로 바꾸면 성공 경로가 매 로드마다 한 번 더 렌더된다. 인과를 다시 보려면 `.catch` 안의 `setFailure` 블록만 `void cause;` 로 바꿀 것(파일 전체를 되돌리면 「0 test」 빌드 에러가 나 아무것도 증명하지 못한다).
- [러너 05:58] brief accepted — 채택 — 근거가 코드와 좌표까지 그대로 맞았고(`load` :209 에 `.catch` 없음, `loading` :298 의 파생, `Loading` 의 12초 추측, 모든 �
- [러너 05:58] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 승인(risk low, 차단 없음). 실측했다: `npm test` 24 files / 109 tests 통과(기준선 23/103 +6), `tsc -b --noEmit`·`eslint --max-warnings 0` 0. 테스트 선택자를 전부 실물과 맞췄다 — `Loading` 기본 label 이 정확히 `"불러오는 중"` 이라 `toBeNull()` 이 공허하지 않고, `정렬 기준` aria-label·`title_asc`·`계약` 제목 모두 존재한다. ledger-entry.md:5 의 실패 재현(5 failed + 테스트 수만큼의 Unhandled Rejection, 인과는 `setFailure` 블록만 중성화)은 이번 변경이 고치는 증상과 정확히 맞는다.
- 구현자가 의심한 `!loading && failed` 배너는 **프로덕션에서 도달한다**(K1 성공 후 onSaved 재조회 거부, 또는 K2 실패 → K1 복귀 후 그 재조회 거부). 렌더는 `viewError` 와 같은 모양이라 안전하지만 그 배너에만 「다시 시도」가 없다. 결함이라기엔 약해 승인했고 다음 회차 1순위 후보로 남긴다.
- 못 본 것: 브라우저 실물(jsdom 만), Go 통합(Go 미변경이라 불필요), 원격 PR 상태.
- 승인이어도 남는 우려 둘 — ① `loading && failed` 가 `<Loading/>` 을 대체하므로 이 다섯 화면에서 12초 `window.location.reload()` 탈출구가 사라진다: 401 에서 「다시 시도」는 만료 세션으로 영원히 실패하고 401 전역 처리는 부트 한 곳뿐이라 앱 안에서 로그인으로 돌아갈 길이 없다(WorkInbox 가 같은 빈틈을 이미 갖고 있어 차단하지 않았다 — `APIError.code`/`status` 를 쓰는 401 분기를 단독 회차로). ② 재시도 중 스피너도 disabled 도 없어 두 번째 거부 때 버튼이 죽은 것처럼 보인다(WorkInbox 는 `setError("")`+`busy` 로 피한다).
- 릴리즈 노트용 한 줄: 업무 목록 조회가 거부되면 12초 스피너 대신 서버가 말한 이유와 「다시 시도」가 나온다.
- [러너 06:04] review approved — 리뷰 승인 (risk=low)
- [러너 06:04] pr created — https://github.com/hkjang/Vendra/pull/143
- [러너 06:07] ci passed — 검사 2개 모두 success
- [러너 06:07] merge done — 28572eb
- [러너 06:10] release published — v0.7.72
- [러너 06:11] assets verified — v0.7.72 자산 1개 (이전 v0.7.71: 1)
