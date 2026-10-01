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
