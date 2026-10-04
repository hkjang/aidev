# 회차 노트 2026-10-04-132237-cutover-improve — cutover
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 13:22] base pinned — main@a5dcc23
- [러너 13:22] autonomy release — 

## 정찰 노트
- GET 500 을 세 화면이 전부 정상 데이터로 받아 "0% 완료 (0/0)" 빈 전광판을 보여주는 것을 골랐다. 보류 목록의 1순위였던 targetId 미존재 PUT(2/2/S)은 UI 가 존재하는 행에서만 호출해 실제 영향이 API 직접 호출에 한정되고, 전파 통일(3/2/M)은 두 계약 비교가 필요해 작은 회차에 안 맞는다.
- 코드로 확인한 것: 세 fetcher 가 r.ok 미검사, GET 500 본문이 activities:[] 를 담은 정상 JSON(route.ts:33-44), data.error 독자 0건, page.tsx:87·pc:80 의 if(error) 가 HTTP 오류에 죽은 코드, 기존 PUT 실패 e2e 3곳이 GET 을 route.fallback() 으로 통과시킴.
- 추측으로 적은 것: 수용 기준 2)의 "실패 시 직전 트리 유지 + 경고" 는 SWR 이 오류 때 마지막 data 를 유지한다는 전제다(문서 기준, 이 저장소에서 실측하지 않음). 유지되지 않으면 전체 화면 오류 분기만으로 1)·3)·4)를 충족시켜도 된다.
- 구현자가 조심할 것: 200 + 빈 배열은 정상(09-27 의 empty-activity-tree 안내)이므로 HTTP 실패에만 새 표시를 내야 한다. GET 라우트 응답 모양은 바꾸지 말고 변경은 클라이언트에 한정한다. npm run test:unit 은 글롭이 안 풀려 조용히 0건 통과하므로 쓰지 말고 파일을 명시해 돌린다.
- 이 세션 한계: 샌드박스가 node --test 실행을 거부해 테스트를 직접 돌려 보지 못했다. 프로필(09-29)은 14일 이내이고 현재 코드와 어긋나는 점을 찾지 못해 다시 쓰지 않았다(10-03 수정 항목이 '해결됨' 에 빠져 있는 것만 미반영).
- [러너 13:27] scout done — GET /api/activities 가 500 일 때 상황판이 "0% 완료 (0/0)" 빈 화면을 조용히 보여주는 것 수정 (가치 4 / 위험 2 / �

## 구현 노트
- 세 화면의 지역 fetcher 가 r.ok 를 안 봐서 GET 500(activities:[] 담은 정상 JSON)이 "0% 완료 (0/0)" 로 보이던 것을, 공유 `lib/fetchJson.ts` 가 res.ok 를 보고 서버 error 문구를 담아 throw 하게 고쳤다. 표준·전광판은 첫 로드 실패면 전체 화면 오류(`dashboard-load-error`), 성공 후 실패면 직전 트리 유지 + 상단 경고(`dashboard-stale-warning`); 관리자는 `admin-load-error` 배너. 커밋 60e659a.
- 확신 없는 곳: 경고/오류 배너의 문구·testid 만 e2e 로 고정했고 **레이아웃은 눈으로 확인하지 않았다**(전광판 sticky 헤더에 배너가 들어가며 아래 트리가 밀린다 — 화면 높이 영향 미검증). 폴링 실패→복구 전환은 SWR 의 refreshInterval/errorRetry 타이밍에 의존해 10초 폴링인 `/` 테스트가 20.7s 걸린다(여유로 test.setTimeout(90s)); 느린 러너에서 간헐 실패 가능성을 배제하지 못했다.
- 일부러 하지 않은 것: GET 라우트 응답 모양(route.ts:33-44) 불변 — 바꾸면 세 화면과 e2e 가 같이 움직인다. 첫 로드 실패 시 관리자 화면에 `empty-activity-tree`("등록된 작업이 없습니다")가 `admin-load-error` 와 함께 보이는 모순은 수용 기준 밖이라 손대지 않았다(09-27 e2e 가 그 안내를 고정하고 있다). 과제서 차선 후보(targetId 미존재 PUT 400)도 안 했다.
- **다음 역할이 조심할 것 — 프로필·과제서가 틀렸다**: "playwright-report/·test-results/ 가 09-29 이후 globalIgnores 에 있어 rm -rf 우회 불필요" 는 사실이 아니다. eslint.config.mjs 에는 기본 4항목뿐이고 09-29 커밋 0de2f1e 는 main 의 조상이 아니다(머지 안 됨). **e2e 를 돌린 뒤 npm run lint 는 playwright-report/trace/** 번들에서 257 errors 를 낸다** — lint 전에 `rm -rf playwright-report test-results` 를 해야 한다(두 경로는 .gitignore 15-16행, 커밋 위험 없음).
- 이 워크트리는 node_modules 가 없어 `npm ci --legacy-peer-deps` 를 먼저 돌려야 한다. 단위 테스트는 `npm run test:unit` 대신 파일 명시로 돌릴 것(글롭 미확장, 조용한 0건 통과).
- [러너 13:39] brief accepted — 채택 — 근거가 현재 코드와 정확히 일치했고(세 fetcher 의 `r.ok` 미검사, 500 본문의 정상 JSON, `if (error)` 가 HTTP 오류에 죽은
- [러너 13:39] verify passed — 검증 1개 통과 (policy)

## 비평 노트
- 확인한 것: diff(실제 base a5dcc23 — 워크트리의 `main` ref 는 cd9d1f3 로 낡아 `git diff main...HEAD` 는 20여 커밋을 함께 보여 준다. 다음 역할도 주의)와 세 화면 전문, route.ts GET catch 분기, 기존 e2e 의 가로채기 패턴. 직접 실행: `npx playwright test e2e/dashboard-load-failure.spec.ts` → **4 passed (45.9s)**, 느린 케이스 20.7s(구현 노트와 일치), `npx tsc --noEmit` 0, `npm run lint` 0(산출물 rm 후), git status 깨끗. 새 testid 3개가 base 에 0건이라 수정 전 코드에서는 반드시 실패한다.
- 못 본 것: 구현자가 스스로 의심한 **배너 레이아웃**은 나도 눈으로 확인하지 않았다(테스트는 가시성·문구만 고정). 전광판 sticky 헤더에 21px 짜리 배너가 들어가 트리가 밀리는 높이 영향은 여전히 미검증. 간헐 실패 여부는 1회 실행으로는 단정 못 한다.
- 승인이어도 남는 우려(릴리즈 노트·다음 회차): ① 관리자 **첫 로드 실패**는 처리·테스트 모두 없다 — `admin-load-error`("최신이 아닐 수 있습니다")와 `empty-activity-tree`("등록된 작업이 없습니다")가 동시에 보인다(admin/page.tsx:150·274). ② 이번 수정은 error 상태만 가르므로 첫 GET 이 **진행 중인 구간**에는 `/`·`/pc` 가 여전히 "0% 완료 (0/0)" 를 그린다(page.tsx:114, pc:105) — 느린 폐쇄망에서 같은 오해가 수 초 남는다.
- 보안·법무 차단 없음: 인증/인가/세션/엔드포인트 변경 0, 새 출력은 서버 고정 문구를 React escape 로 표시, 비-JSON 본문은 'HTTP <status>' 로 떨어져 내부 정보 미노출, 개인정보 변화 없음.
- 프로필 정정은 구현 노트가 맞다: `eslint.config.mjs` 는 기본 4항목뿐이라 e2e 뒤 lint 전에 `rm -rf playwright-report test-results` 가 필요하다(이번에도 그렇게 해야 lint 0 이었다).
- [러너 13:44] review approved — 리뷰 승인 (risk=low)
- [러너 13:44] pr created — https://github.com/hkjang/cutover/pull/12
- [러너 13:45] ci passed — 검사 없음 — 정책으로 허용
- [러너 13:45] merge done — 60e659a
- [러너 13:52] release published — v1.14.0
- [러너 13:52] gh-release created — GitHub Release v1.14.0
- [러너 13:52] manifest ok — cutover-v1.14.0.tar.gz 
- [러너 13:52] assets uploaded — 1개
- [러너 13:52] assets verified — v1.14.0 자산 1개 (이전 v1.13.0: 1)
