# 회차 노트 2026-10-05-222752-yeopjari-improve — yeopjari
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 22:28] base pinned — main@c730b67
- [러너 22:28] autonomy release — 

## 정찰 노트
- HEAD 두 커밋(6208b51·c730b67)이 모두 seo/indexnow.ts 를 건드렸는데 단위 테스트가 0이고, `new URL(env.publicUrl)` 이 try 밖에 있어 "never throws" 주석과 어긋난다(jobs/disclosures.ts:296 은 트랜잭션 커밋 뒤 호출). 프로덕션 1파일 + 테스트 1파일로 끝난다.
- 제친 후보: 연봉 RLS E2E·pipeline CSRF 는 격리 PostgreSQL 필요(10분 정찰·45분 구현에 안 맞음), CI 워크플로는 보호 경로, StaticFiles 테스트는 결함 근거 없는 순수 테스트 추가라 차선으로 남겼다.
- 추측으로 적은 것: 잘못된 publicUrl 이 실제로 설정된 적이 있는지는 미확인(스킴 없는 값이 그럴듯한 운영 실수라는 판단). 아침 공시 작업이 실패를 어떻게 기록/재시도하는지(runner/queue)는 열어 보지 않았다.
- 확인한 것: /tools/salary 쿼리 폼은 엣지 캐시 대상이 아니라 캐시 오염 없음(후보에서 기각), isThin↔SUBSTANTIAL_SQL 의 빈 배열 불일치는 summariseFinancials 가 빈 배열을 쓰지 않아 실발생 미확인(가치 2로 내림).
- 구현자 주의: 테스트에서 실제 api.indexnow.org 로 POST 가 나가면 안 된다(fetch 스텁 필수). indexNowKey 파생식·반환 문자열('sent'/'skipped'/'http n')은 ops.ts 가 그대로 노출하므로 바꾸지 말 것. node_modules 가 없어 npm ci 부터.
- [러너 22:32] scout done — IndexNow 알림이 publicUrl 설정 오류에 트랜잭션 뒤에서 터지지 않게 하고, 키·요청 본문 계약을 테스트로 묶�

## 구현 노트
- `seo/indexnow.ts` 의 `new URL(env.publicUrl)` 을 try 로 감쌌다. 트랜잭션 커밋 뒤에 불리는 함수가 "never throws" 주석과 달리 스킴 없는 설정값 하나로 TypeError 를 올려보내고 있었다. 반환 문자열(`sent`/`skipped`/`http n`/`error …`)은 `ops.ts` 가 운영자에게 그대로 노출하므로 손대지 않고, 설정 오류만 `error publicUrl 설정이 주소가 아닙니다` 를 새로 돌려준다. 프로덕션 파일 1개 + 테스트 1개(신규 19건).
- 확신 없는 곳: (1) 설정 오류를 `skipped` 대신 `error …` 로 돌린 선택 — 과제서는 둘 다 허용했고 운영자에게 보이는 쪽이 낫다고 판단했지만, `ops.ts` 응답의 `ok` 가 `result === 'sent'` 라서 어느 쪽이든 false 다(동작 차이는 표시 문자열뿐). (2) 판정 순서를 `dev/offline/빈 목록` → URL 파싱 → `https:` 로 바꿨다. 네 조건이 모두 `skipped` 를 돌려주므로 정상 설정에서 관측 가능한 차이는 없다고 보지만, 덕분에 dev 모드 + 잘못된 publicUrl 이 `error` 가 아니라 `skipped` 가 된다(의도한 것, 테스트로 고정).
- 일부러 하지 않은 것: `env.ts` 에 publicUrl 검증을 넣지 않았다(과제서 지시). `public-pages.ts`·`rankings.ts`·`tools.ts` 는 여전히 날것으로 `new URL(ctx.env.publicUrl)` 을 쓰므로 같은 설정 실수로 공개 페이지가 500 이 될 수 있다 — ideas.json 에 별 과제로 남겼다. `indexNowKey` 파생식도 그대로다.
- 다음 역할이 조심할 것: 이 테스트는 DB 도 네트워크도 필요 없다(`globalThis.fetch` 를 `vi.spyOn` 으로 스텁하고 `afterEach` 의 `restoreAllMocks` 로 되돌린다). 실제 `api.indexnow.org` 로 나가는 요청은 없고, 본문·헤더는 스텁이 받은 값을 JSON.parse 해서 assert 한다. 키 값은 하드코딩하지 않고 형식·결정성만 고정했다.
- 발견(이 회차 과제 밖): 2026-10-04 원장이 성공으로 기록한 쿠키 파서 수정이 이 base(c730b67)에 없다. `auth/cookies.ts:39` 는 여전히 보호 없는 `decodeURIComponent` 이고 `cookies-params.test.ts` 도 없다 — 그 PR 이 머지되지 않았을 가능성이 높으니 다음 회차에 중복 구현하기 전에 확인이 필요하다.
- [러너 22:38] brief accepted — 채택 — 근거가 지금 코드와 정확히 맞았다(indexnow.ts:31 의 try 밖 `new URL`, disclosures.ts:296 의 커밋 뒤 호출, public-pages.ts:775 �
- [러너 22:38] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 원장에 `- 실패 재현:` 줄이 없어 직접 재현했다: indexnow.ts 만 main 판으로 되돌려 재실행 → 19건 중 7건이 `TypeError: Invalid URL`(input '' / 'yeopjari.bid')로 실패, 스택이 pingIndexNowStatus → pingIndexNow 로 증상과 일치. 트리는 복구(git status 깨끗). 수정판에서 19건 통과, 전체 lint·typecheck·test(28파일 247건) 통과. fetch 는 스텁이라 실제 api.indexnow.org 요청 없음.
- 구현자가 의심한 두 자리를 시험했다. 설정 오류를 `error …` 로 돌린 선택은 disclosures.ts:296 의 `report.pinged` 가 'skipped'/'http n' 에서도 이미 false 였으므로 기존 의미와 어긋나지 않는다. 판정 순서 변경의 유일한 관측 차이(dev + 잘못된 publicUrl → 'skipped')는 61행에 고정돼 있다. env.ts:222 adminDomains 가 이미 try/catch 라 잘못된 publicUrl 로도 부팅돼 이 경로에 도달한다 — 정찰의 추측 전제가 코드로 확인됐다.
- 못 본 것: 아침 공시 작업의 실패 기록·재시도(runner/queue), 통합 e2e·DB/RLS(격리 DB 필요). 남는 우려(둘 다 기존 코드, 이번 diff 소관 아님): public-pages.ts:775·rankings.ts·tools.ts 의 날것 `new URL(ctx.env.publicUrl)` 는 같은 설정 실수로 공개 페이지 500; indexNowKey 가 공개 /<key>.txt 를 tokenSign HMAC 에서 파생하므로 키 교체가 엔진에 알린 키를 조용히 바꾼다(공격 경로 없음, 런북에 적을 일).
- 느슨한 단언 1건: indexnow.test.ts:48 은 'skipped' 와 'error …' 둘 다 통과시켜 결과 문자열을 고정하지 않는다. 의도된 허용 범위이므로 거절 사유 아님.
- 판정 approve / risk low / blocking 없음. 다음 회차 확인: auth/cookies.ts:39 의 무방호 `decodeURIComponent` 와 cookie 테스트 부재를 재확인했다 — 그 수정은 이 base 에 없다.
- [러너 22:41] review approved — 리뷰 승인 (risk=low)
- [러너 22:41] pr created — https://github.com/hkjang/yeopjari/pull/3
- [러너 22:46] ci no-ci — 이 커밋에 검사가 없음 (정책 allow_merge_without_ci 가 없으면 차단)
