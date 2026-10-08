- 과제: 앱 등록·수정 완료 화면에 저장 응답의 실제 상태를 표시한다 (가치 3 / 위험 1 / 작업량 M)
- 왜: `AppFormPage`는 저장 결과와 무관하게 “승인 Workflow 설정에 따라 즉시 게시되거나 검토 대기 상태가 됩니다.”라고 안내하지만 서버는 보안 게이트에서 초안을 정상 반환하고 수정 시 반려·보관 상태도 유지할 수 있다. 완료 화면에 실제 저장 상태를 표시하면 소유자가 아직 게시·제출되지 않은 앱을 완료된 것으로 오해하지 않는다.
- 수용 기준:
  1) `/submit`에서 필수 입력을 채워 등록한 뒤 POST 응답 `status=draft`이면 완료 패널에 `현재 상태: 초안`이 표시되고 기존 즉시 게시/검토 대기 예고 문구는 없다. 보안 심의가 필수라고 상태만 보고 단정하지 않는다.
  2) 같은 완료 패널에서 `published`는 `게시됨`, `pending_review`는 `검토 대기`, 수정 응답의 `rejected`/`archived`는 `반려`/`보관됨`을 표시한다. `appStatusLabel`을 그대로 재사용하고 별도 번역표를 만들지 않는다. 상태 누락은 `—`, 모르는 값은 기존 함수의 원문 폴백을 유지한다.
  3) 수정 전 조회 상태가 published여도 PUT 응답이 pending_review이면 완료 화면은 검토 대기를 표시한다. `앱이 등록되었습니다`/`앱이 수정되었습니다` 제목, `내 앱으로 이동` 버튼을 유지하고 실제 클릭 시 `/my/apps`의 `내가 등록한 앱` 화면으로 이동한다.
  4) POST/PUT 또는 후속 문서 적용 실패는 성공 화면으로 바뀌지 않고 기존 폼 오류를 유지한다. 기존 수정 대상 없음/조회 500 테스트 4건을 보존한다.
  5) 프로덕션 AppFormPage·API 클라이언트·라우터·QueryClient·AuthProvider·FavoritesProvider를 사용하고 HTTP 경계만 대체한 실행 테스트로 등록/수정 후 DOM을 확인한다. 새 회귀 테스트가 제품 수정 전 실패하고 수정 후 통과해야 한다. 실제 Vite 번들+Playwright에서도 초안 등록 완료 화면을 desktop/mobile 모두 확인한다.
- 건드릴 파일:
  - `web/src/pages/app-form-page.tsx:AppFormPage`, `save`, `if (complete)` — 현재 mutationFn이 반환하는 saved를 `onSuccess(saved)` 인자로 받아 완료 결과 상태(예: `completedApp: StoreApp | null`)에 보관하고 그 status로 공용 `appStatusLabel`을 호출한다. 별도 boolean complete를 성공 결과 존재 여부로 대체해 표시와 결과가 함께 반영되게 한다. 현재 onSuccess는 invalidateQueries를 await하므로 boolean만 먼저 켠 뒤 save.data를 읽는 방식은 결과 반영 전 빈 상태를 표시할 수 있어 피한다. 기존 예고 문구는 중립적인 안내(예: “내 앱에서 상태와 다음 단계를 확인할 수 있습니다.”)로 교체한다. 프로덕션 변경은 이 1파일로 제한한다.
  - `web/src/pages/app-form-page.test.tsx:renderEdit`, `myApp` 및 새 제출 테스트 — `/submit` 라우트를 추가하고 fetch의 URL뿐 아니라 method도 구별하여 POST/PUT에 StoreApp 객체를 반환한다. 현재 categories fixture는 []이므로 실제 카테고리를 제공하고 앱 이름·Slug·한 줄 설명·서비스 URL·카테고리·상세 설명을 채워 버튼으로 제출한다. edit fixture도 serviceUrl/category/description을 채워 HTML 필수 검증에 막히지 않게 한다. mutation/hook/API 자체를 mock하지 않는다.
  - `web/e2e/core.spec.ts` — 이름이 `초안 등록 완료 상태를 표시한다`인 시나리오 1건. `installMockApi(page, { authenticated: true })` 후 page.route로 `/api/v1/apps`의 POST만 201+draft StoreApp 응답으로 덮고 다른 method는 route.fallback()한다. 실제 `/submit` 폼 입력→등록→완료 상태까지 검증한다. 기존 보안 심의 테스트는 등록 화면을 지나지 않으므로 그것만으로 이번 변경을 검증했다고 쓰지 않는다.
- 검증 명령: 저장소 루트 기준 아래 순서. 현재 node_modules가 없어 설치 선행이 필요하며 정찰에서는 저장소 쓰기 금지 때문에 설치하지 않았다.
  ```bash
  npm --prefix web ci --no-audit --no-fund
  npm --prefix web test -- src/pages/app-form-page.test.tsx
  npm --prefix web test
  npm --prefix web run lint
  (cd web && npx prettier --check src/pages/app-form-page.tsx src/pages/app-form-page.test.tsx e2e/core.spec.ts)
  npm --prefix web run build
  ./scripts/check-offline-assets.sh web/dist
  ./scripts/check-env-contract.sh
  ./scripts/check-docs.sh
  (cd web && npx playwright install chromium)
  CI=true npm --prefix web run test:e2e -- --grep '초안 등록 완료 상태를 표시한다' --workers=2 --retries=0 --reporter=list
  git diff --check
  ```
  대상 Vitest는 정찰에서 실제 실행했으나 `vitest: not found`로 exit 127이며 통과 수는 미확인이다. env/docs 검사는 이번 정찰에서 각각 exit 0. build/E2E 및 실제 DB/SecCheck 연동은 미실행이다.
- 위험과 피할 것: auth/session, migrations, .github/workflows, internal/webui/dist, 서버/API/승인 정책, 문서 업로드 재시도 구조를 변경하지 않는다. `app-status.tsx`는 이미 원하는 폴백을 제공하므로 수정하지 않는다. 완료 결과는 documents.apply 성공 후에만 설정되는 현재 순서를 보존한다. published는 공개 범위와 별개이므로 “모두에게 공개됨”이라고 쓰지 않는다. 기존 E2E mock의 `/api/v1/apps`는 method 분기 없이 목록을 반환한다(352행): 이를 저장 성공 fixture로 그대로 쓰면 안 된다. public config override 버그도 이번에는 고치지 않는다. 문서·PDF·릴리즈 버전·스크린샷 전면 갱신은 범위 밖이다. 두 과거 human-rejected의 구체 diff는 미확인이라 입력 하드닝/출력 변화 없는 테스트 단독 접근을 피한다.
- 차선 후보: 가이드 문서 복수 선택 시 첫 실패 사유와 실패 파일 수를 표시 — `web/src/features/apps/guide-documents.tsx:useGuideDocumentDraft.add`의 마지막 failure 덮어쓰기를 좁게 수정하고 기존 `guide-documents.test.tsx` 하네스/alert로 확인. 1순위가 이미 수정됐음이 새 기준선에서 확인될 때만 전환한다.

근거와 선택:
- 기준 HEAD `755d25b`(v2.11.16), 작업 트리 깨끗함. `app-form-page.tsx`의 108~122행 mutation 반환/onSuccess와 157~177행 완료 분기를 직접 확인했다.
- `internal/httpapi/app_handlers.go:createApp`은 draft 생성→submitOrHold→201 result.App 반환. `security_check_handlers.go:submitOrHold`(429행)는 ErrSecurityCheckRequired일 때 저장 앱을 다시 읽어 성공으로 반환한다. updateApp은 updated 또는 재제출 result.App을 반환한다. `api.ts:createApp/updateApp`은 둘 다 StoreApp을 받는다.
- `docs/USER_GUIDE.md` 71~73행도 보안 심의 필요 조직에서 초안이 남는 흐름을 설명한다. 결함은 코드 경로로 확인했고 실제 브라우저 재현은 구현 단계에서 수행할 부분이다.
- 선택 대안: (A) 완료 문구를 “내 앱에서 확인”으로만 바꾸기(S)는 거짓 예고를 없애지만 현재 상태를 알려 주지 못한다. (B) 저장 응답+기존 라벨 재사용(M)을 선택한다. (C) 별도 심의 API 조회·보안 화면 자동 이동(M 이상)은 이유 판정과 실패 경로가 늘어나 제외한다. (D) 문서 안내에만 의존하면 잘못된 완료 화면이 남는다.
- 다른 후보보다 우선한 이유: 즐겨찾기는 미병합 변경 중복 위험, 프록시/페이지 크기는 정책·API 범위, Field는 공용 계약 영향이 있다. 이 변경은 실제 사용자 출력을 고치면서 기존 제출 테스트 공백의 일부도 메운다. 가장 중요한 전제는 저장 응답이 상태의 정본이라는 것이며 위 두 서버/API 경로로 확인했다.

실행 순서와 체크포인트(인간 승인 없음, 각 단계 자동 검증):
1. [미착수] 의존성 준비, 기존 대상 4건 확인, HTTP 제출 fixture 확장 및 실패 회귀 관찰 후 같은 단계에서 제품 1파일을 수정한다. 증명: `npm --prefix web test -- src/pages/app-form-page.test.tsx`의 red→green. 단계 종료 전에 green이어야 한다.
2. [미착수] 수정 전 상태와 다른 응답, 기존 오류/돌아가기 동작, 폴백을 검증하고 전체 React·lint·수정 파일 prettier·build를 위 명령으로 통과시킨다. 새로 파악된 원인이 과제와 다르면 범위를 넓히기 전에 이 과제서를 수정한다.
3. [미착수] scoped POST fixture로 브라우저 테스트를 추가하고 build 후 위 대상 E2E와 env/docs/offline/diff 검사를 실행한다. desktop/mobile 결과와 미실행 범위를 구분해 기록한다. 생성물을 커밋하지 않는다.

공수 근거(정찰자의 판단, 확정 일정 아님):
- Bottom-up: 준비·기준선 4~6분 + 제출 회귀/제품 수정 11~14분 + 보존 검증 4~6분 + 브라우저/전체 검사 6~9분 = 기본 25~35분.
- 알려진 변동에 대한 contingency 5~10분(필수 폼 fixture, 의존성/브라우저 준비)으로 총 30~45분 예상. 통계적 80% 신뢰구간은 산출할 실측이 없어 주장하지 않으며 판단 신뢰도는 중간이다. 외부 다운로드/CI 대기는 범위에 들지만 소요 상한은 미확인이다.
- 유사 10/05·10/07 회차는 제품 1파일+HTTP 회귀, 후자는 대상 E2E까지 성공해 규모 비교에는 적합하다. 실제 작업시간 기록은 없어 시간 추정의 독립 검증으로 삼지 않는다. 두 번째 수치 추정과 25% 차이 검증은 자료 부족으로 미확인이다.
- 관리 예비비(management reserve)는 별도 0분 배정: 새로운 업로드 재시도/페이지네이션 요구는 이번 범위에 넣지 않는다. 준비 후 다시 추정해 45분 초과가 명백하면 별도 원인을 기록하며, 검증 성공을 가장하지 않는다.
- 적용 스킬: `/mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md`, `/mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md`, `/mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md` 원문을 읽음(Skill 호출 도구 없음). PMO references/sources.md도 확인했으며 위 시간은 외부 공수 표준이 아닌 저장소 범위에 대한 자체 추정이다.
