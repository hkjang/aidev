# 회차 노트 2026-09-20-073406-SecCheck-improve — SecCheck
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:34] base pinned — main@eb2a3b0
- [러너 07:34] autonomy release — 

## 정찰 노트
- 선택 이유: 우선 과제가 없는 일반 회차. vulndb 게이트(GO-2026-6452 fixed 누락)는 저장소 안에서 못 풀고 같은 수정 과제가 5회 반복되었으므로, 게이트와 무관하게 로컬 vitest 로 완전히 검증되는 과제(화면 API 경로 ↔ server.go 라우트 대조 테스트)를 골랐다. 차선(docs_test 설정 키 대조)·precheck 확장·admin 문서 정리는 열린 PR #9~#12 와 같은 파일을 건드려 제쳤다.
- 확신 없는 곳: vulndb fixed 유무는 이 세션에서 curl 이 거부되어 미확인(01:40 시점 없음) — 구현자가 0단계에서 재판정. vitest 판 번호(4.x 추정) 미확인. 화면 호출 44곳 중 실제 불일치가 있는지는 안 돌려 봐서 모름 — 나오면 그것이 발견.
- 조심할 것: 스캐너를 payloads.test.ts 와 두 벌로 복제하지 말고 공용 함수 한 곳; `Audit.tsx:48` 의 `verify${full ? '?full=1' : ''}` 형태는 단순 split('?') 로 안 잘림; 부정 검증 전 WIP 커밋 필수; go.mod·워크플로·precheck·operations.md·docs_test.go 는 건드리지 말 것.
- [러너 07:37] scout done — 화면이 부르는 모든 API 경로(get/del/upload/post/put/patch)를 `server.go` 의 `s.handle` 등록과 대조하는 vitest 추가 (가

## 구현 노트
- 무엇·왜: `web/src/lib/routes.test.ts`(신규) 가 `server.go` 의 `s.handle` 134건과 `web/src` 의 리터럴 API 호출 121곳(get/post/put/patch/del/upload/api)을 대조한다. 스캐너 원시 함수는 `payloads.test.ts` 에서 `web/test/sourceScan.ts` 로 옮겨 두 테스트가 한 파서를 쓴다(payloads 의 세 단언 불변). 커밋 07c19c9. 0단계: vulndb GO-2026-6452 는 여전히 fixed 없음 — CI step 7 은 이 코드와 무관하게 빨강.
- 확신 없는 곳: (1) 공용 모듈을 과제서의 `web/src/lib/` 가 아니라 `web/test/` 에 둔 것 — `tsconfig.app.json` 이 `src` 전체를 `tsc -b` 하는데 `@types/node` 가 없어 `node:fs` 가 빌드를 깨기 때문(테스트 파일만 exclude). 다른 자리를 원하면 tsconfig 를 건드려야 한다. (2) `${…}` 세그먼트가 서버의 리터럴 세그먼트와도 맞도록 한 와일드카드 규칙 — `ReviewDetail.tsx:47/101` 의 변수 액션 이름 때문인데, 그 두 호출의 실제 액션 이름 오타는 이 테스트로 못 잡는다(예외 2건을 명시하는 대신 일반 규칙을 택함; 진짜 오타·삭제된 라우트는 부정 검증 (a)(b) 로 잡힘을 확인).
- 일부러 하지 않은 것: 화면 수정(실제 불일치 0건), `download(path)` 포함(경로가 ui.tsx:98 에 변수로 들어옴), `payloads.test.ts` 의 다른 변경, go.mod·워크플로·precheck·docs.
- 다음 역할이 조심할 것: 이 테스트는 소스를 읽는 정적 계약 대조라 DB·서버 없이 `cd web && npm test` 로 돈다. `sourceFiles` 는 `web/src` 만 훑고 `web/test/` 는 안 훑는다. 실패 메시지의 줄 번호는 호출 시작 줄이다(ReviewDetail 처럼 한 줄에 여러 호출이면 같은 줄이 여러 번 나올 수 있음). precheck 는 DSN 없이 돌려 통합 테스트는 건너뛰었지만 Go 코드는 바꾸지 않았다.
- [러너 07:43] brief accepted — 채택 — 전제(payloads.test.ts 가 본문 키만 대조, s.handle 134건 전부 server.go, api.ts 68~73행 헬퍼, Audit.tsx:48 의 템플릿 꼬리)는 모
- [러너 07:43] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인: diff 3파일(sourceScan.ts 신규·routes.test.ts 신규·payloads.test.ts 함수 이동) 전량 정독, `tsc -b`·`vitest run` 13건 통과, 부정 검증 재현(화면 경로 오타 1건·서버 라우트 이름 변경 1건 → 각각 파일:줄로 실패, 원복 확인). s.handle 134건 전부 정규식에 잡힘. CI 는 전체 체크아웃에서 `npm test` 를 돌려 server.go 를 읽을 수 있음(Docker 는 build 만).
- 못 본 것: 통합 Go 테스트(DSN 없음, Go 코드 미변경이라 무관), 원격 CI 상태(govulncheck 는 여전히 빨강일 것).
- 남는 우려(승인): `web/test/sourceScan.ts` 는 tsc 타입검사 범위 밖; `${…}` 양방향 와일드카드라 ReviewDetail 의 변수 액션 이름 오타는 못 잡음; 변수 경로 호출(TemplateDetail:23, download) 미검사 — 커밋 제목 'every' 는 리터럴 한정.
- 판정: approve, risk low, blocking 없음.
- [러너 07:46] review approved — 리뷰 승인 (risk=low)
- [러너 07:46] pr created — https://github.com/hkjang/SecCheck/pull/13
- [러너 07:51] ci failed — 성공이 아닌 검사: test-build-scan=failure
