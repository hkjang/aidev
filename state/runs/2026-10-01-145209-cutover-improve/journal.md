# 회차 노트 2026-10-01-145209-cutover-improve — cutover
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:52] base pinned — main@8e876b7
- [러너 14:52] autonomy release — 

## 정찰 노트
- PUT 의 fall-through 200 을 골랐다: 09-26 이 만든 오류 배너가 유일하게 비어 있는 구멍이고 프로덕션 1파일로 끝난다. 테스트·설정만 손댄 회차가 2연속(09-28, 09-29)이라 제품 계약을 고쳤다. 경계값 테스트는 차선으로 남겼다.
- 추측으로 적은 것: `matched` 플래그 구현 방식은 제안이며 실제 변수명·형태는 구현자 판단. `admin-save-failure.spec.ts:34,40` 본문 두 줄은 열어 보지 않았다.
- 실행 못 한 것: 샌드박스가 파이프·node --test 승인을 막아 이번 세션은 테스트를 한 건도 돌리지 않았다. 건수는 과거 회차 기록 기준.
- 구현자가 조심할 것: `{dashboardTitle}` 단독 PUT 과 `{activities: []}` 는 반드시 200 으로 남아야 한다(제목 편집·빈 트리 e2e 2건이 의존). targetId 부재를 404 로 만드는 확장은 범위 밖.
- 새로 눈에 띈 것(이번 과제 아님): 추적되는 `_bak`/`_local` 파일 8개, `addActivity` 가 부모 부재 시 level 1 자식을 만드는 것 — ideas.json 에 넣었다.
- [러너 14:56] scout done — 활동 PUT 이 인식하지 못한 요청에 200 "저장 성공" 을 돌려주는 것을 400 으로 바로잡기 (가치 3 / 위험 2 / 작�

## 구현 노트
- 무엇/왜: `app/api/activities/route.ts` PUT 에 `matched` 플래그를 넣어, 네 변경 분기 중 아무것도 맞지 않고 `dashboardTitle` 도 없는 요청을 `{error, code:'UNSUPPORTED_ACTION'}` 400 으로 거부한다(종전: 현재 데이터를 조용히 재저장하고 200+새 lastUpdated). 프로덕션 1파일 +16줄, 신규 e2e 1파일. 커밋 083c92b.
- 확신 없는 것 ①: **`npm run test:unit` 이 이 환경에서 스크립트째로 실패한다** — `Could not find '/…/lib/**/*.test.ts'`(Node v22.23.1). 테스트 파일 9개를 명시 경로(`node --test lib/*.test.ts lib/mail/*.test.ts lib/tracking/*.test.ts`)로 돌려 `# pass 95 / # fail 0` 을 확인했다. 09-28·09-29 기록은 같은 npm 스크립트가 통했다고 하므로 원인을 확정하지 못했다. 내 변경은 route.ts 뿐이라 인과는 없다. 수리 역할은 단위 검증 시 명시 경로를 쓸 것.
- 확신 없는 것 ②: 첫 전체 e2e 에서 `tracking.spec.ts:76`(기본값 꺼짐 CSP) 1건이 실패했다. 그 실행은 단위 테스트 명령과 동시에 돌던 중이었고, 이후 단독 1회·전체 2회에서 모두 통과(최종 **23 passed**)했다. 경로가 proxy.ts CSP 라 이번 변경과 겹치지 않아 부하 중 간헐 실패로 보지만 근본 원인은 증명하지 못했다. 비평가가 전체 e2e 를 단독으로 돌려 재확인해 주면 좋겠다.
- 일부러 하지 않은 것: `targetId` 부재를 404 로 바꾸기(동시 조작 경합을 실패로 만들고 treeUtils 계약까지 바꿈), 분기 조건·검증 순서·`validateActivityImport`·49행 `isAdminRequest` 수정, `package.json` 의 글롭 수정, 추적되는 `route.ts_bak`/`_local` 사본 동반 수정(Next 라우트로 잡히지 않음) — 모두 ideas.json 에 남겼다.
- 다음 역할이 조심할 것: `{dashboardTitle}` 단독 PUT 과 `{activities: []}` 는 200 이어야 하고, 이 두 보호 장치는 수정 전에도 통과한 상태로 신규 spec 에 고정돼 있다(기존 `admin-save-failure.spec.ts:169` 도 제목 단독 PUT 에 200 을 기대). e2e 는 `PLAYWRIGHT_CHROMIUM_PATH=/usr/bin/google-chrome` 필요. 검증: tsc exit 0, lint exit 0, build exit 0, e2e 23 passed, 단위 95 pass.
- [러너 15:02] brief accepted — 채택 — 근거가 현재 코드와 정확히 일치했다(60-76행 fall-through, `admin/page.tsx` 의 `res.ok` 검사만, `{dashboardTitle}` 단독 PUT 이 �
- [러너 15:02] verify passed — 검증 1개 통과 (policy)

## 비평 노트
- 확인한 것: route.ts 디프가 순수 추가(+16, 삭제 0)임을 근거로 수정 전 `{}`·`{foo:'bar'}`·`{action:'add',parentId:5}` 가 94-108행 재저장을 타고 200 을 돌려주었음을 확정 — 새 spec 은 그 경로에 400+lastUpdated 불변을 단언하므로 수정 전 반드시 실패한다. 원장에 `- 실패 재현:` 줄이 없어 디프 대조로 대신했다.
- 직접 실행: e2e 단독 1회 **23 passed**(구현자가 의심한 tracking.spec.ts:76 포함 — 불확실 ② 해소), tsc exit 0, lint 통과. `npm run test:unit` 은 exit 1 + `Could not find 'lib/**/*.test.ts'` 로 0건 실행(불확실 ① 사실 확인, package.json 은 dd1f224 이후 미변경이라 선재 결함), 명시 경로로는 95 pass.
- 못 본 것: 수정 전 코드에 새 spec 을 실제로 올려 실패 출력을 받아 보지는 않았다(워크트리 수정 금지). admin-put-contract 를 다른 spec 들과 다른 순서로 돌려 보지는 않았다.
- 승인이어도 남는 우려: `{dashboardTitle:'x', action:'typo'}` 는 여전히 200 으로 title 만 저장하고 알 수 없는 action 을 무시한다 — 릴리즈 노트에 "모든 미인식 PUT 이 400" 이라고 쓰지 말 것. `npm run test:unit` 이 0건 실행이라는 사실은 다음 회차 후보로 가장 값이 크다. ADMIN_GUIDE.md:329 는 400 을 5.2절로 안내하는데 새 UNSUPPORTED_ACTION 은 거기 없다.
- 보안·법무 차단 없음: PUT 은 49행 isAdminRequest 로 먼저 인가되고 새 400 본문에 비밀·경로·스택이 없으며, 받아들이는 입력을 좁히기만 한다. 신규 개인정보·의존성·외부 약속 없음.
- [러너 15:06] review approved — 리뷰 승인 (risk=low)
- [러너 15:06] pr created — https://github.com/hkjang/cutover/pull/10
- [러너 15:07] ci passed — 검사 없음 — 정책으로 허용
- [러너 15:07] merge done — 083c92b
- [러너 15:16] release published — v1.12.0
- [러너 15:16] gh-release created — GitHub Release v1.12.0
- [러너 15:16] manifest ok — cutover-v1.12.0.tar.gz 
- [러너 15:16] assets uploaded — 1개
- [러너 15:16] assets verified — v1.12.0 자산 1개 (이전 v1.11.0: 1)
