- 과제: 관리자 저장 실패 배너에 서버가 반환한 입력 검증 상세 원인 표시 (가치 3 / 위험 1 / 작업량 S)
- 왜: 작업 제목 301자·시간 101자·상황판 제목 101자 저장 시 서버는 정확한 `details[].path/message`를 반환하지만 `submitActivityChange`가 이를 버려 화면에는 “activity.json 데이터 형식이 올바르지 않습니다.”만 보인다. 기존 검증 상세를 배너에 표시하면 관리자가 무엇을 고쳐야 하는지 알 수 있고 최대 깊이·개수 초과 같은 같은 경로의 실패도 설명할 수 있다.
- 수용 기준:
  1) 실제 행 편집에서 제목 301자와 시간 101자를 함께 입력해 저장하면 PUT 400 `INVALID_DATA`이며 `admin-save-error` 안에 기존 요약과 두 상세 원인(300자·100자 제한)이 모두 보인다. 입력은 열린 채 그대로 남고, 같은 브라우저 context의 후속 GET에서 activities·dashboardTitle·lastUpdated가 저장 전과 같다.
  2) 상황판 제목 101자도 실제 PUT 400 뒤 배너에 100자 제한 원인이 보이고 편집 입력이 유지된다. 유효한 제목으로 다시 저장하면 PUT 200, 후속 GET에 반영되고 배너와 이전 상세가 모두 사라지며 편집이 닫힌다. 행 편집도 올바른 값으로 재시도하면 같은 복구 동작을 확인한다.
  3) 상세 없는 서버 오류(기존 500), 401 재로그인, 409/404 메시지, 성공 응답의 SWR 캐시 반영 계약을 유지한다. `details`가 없거나 배열이 아니거나 항목 형식이 틀린 경우 무시하고 일반 오류를 표시한다. 비JSON 실패 응답은 기존 HTTP 상태 fallback을 유지한다. 상세는 React 텍스트로 표시하며 HTML로 삽입하지 않는다.
  4) 브라우저 회귀는 실제 서버 검증으로 1~2를 증명한다. 수정 전에는 400·저장 불변·입력 유지가 이미 성공하고 **상세 원인 노출 단정만 실패**해야 한다. 데이터 손실 수정·400 신설이라고 주장하지 않는다. 비JSON/잘못된 details fallback만 page.route 실패 주입으로 검증한다.
- 건드릴 파일:
  - `app/admin/page.tsx:submitActivityChange`(76행), `saveError` 상태와 `admin-save-error` 렌더(186행) — 실패 응답의 유효한 details를 기존 요약과 함께 표시. 가장 작은 구현은 기존 string 상태에 message를 읽기 쉽게 합치는 방식이다(줄바꿈을 쓰면 렌더도 줄바꿈 표시). 여러 원인을 모두 보여 주고 성공/401/다음 실패에서 이전 상세가 남지 않게 한다. 필드별 한국어 라벨 매핑이나 새 공통 모듈은 필요 없다.
  - `e2e/admin-save-failure.spec.ts:loginThroughUi/readData/waitForPut` 및 기존 행/상황판 제목 실패 테스트 — 기존 격리 seed·복원 패턴을 활용해 실제 400 두 시나리오와 복구/불변 검증 추가. readData 반환 타입에 lastUpdated를 넣으면 실제 응답의 시점을 비교할 수 있다. 상세 없는 500·401 기존 테스트는 유지한다.
  - `docs/ADMIN_GUIDE.md:4.4 제목·시간 편집` — 검증 실패 때 제한 원인이 보이고 입력이 유지됨을 설명하면서 현재 “편집만 닫힙니다”를 바로잡는다. 다른 절과 PDF 재생성은 범위 밖.
  - 프로덕션 1파일, 테스트 1파일, 문서 1파일. `lib/activityData.ts`, API 라우트, importer는 참고만 한다.
- 검증 명령:
  - 정찰에서 실제 실행: `npm run test:unit` → Node v22.23.1, 108 pass / 0 fail / 0 skipped. 전체 출력은 이 회차의 `unit-baseline.log`.
  - 구현 환경 준비: 현재 node_modules 없음. 필요하면 기존 회차 성공 명령 `npm ci --legacy-peer-deps`로 설치 후 **코드 작성 전에** 설치된 `node_modules/next/dist/docs/`의 관련 Client Components 가이드를 찾고 읽는다. 정찰에서는 설치·가이드 열람·Next 실행을 하지 않았다.
  - 집중 회귀: `PLAYWRIGHT_CHROMIUM_PATH=/usr/bin/google-chrome npm run test:e2e -- e2e/admin-save-failure.spec.ts` (package.json과 playwright.config.ts, /usr/bin/google-chrome 존재 확인; 이번 미실행).
  - 구현 후 순서: `npm run test:unit`, `npx tsc --noEmit`, `npm run lint`, `PLAYWRIGHT_CHROMIUM_PATH=/usr/bin/google-chrome npm run test:e2e`, `npm run build`, `git diff --check`. E2E 전 lint를 실행하며 이미 생성물이 있으면 이번 실행이 만든 산출물만 정리한다. 전체 E2E 현재 총 건수와 통과 여부는 미확인이고 과거 기록 수치를 합산하지 않는다.
- 위험과 피할 것: 인증·세션·proxy·메일·파일 저장·배포/workflows·의존성 버전은 변경하지 않는다. 클라이언트로 `lib/activityData.ts`를 값 import하면 node:fs/node:crypto가 딸려오므로 금지한다. 검증 상한·에러 코드·배열 PUT/expectedLastUpdated·부모 상태 전파는 유지한다. maxLength로 301자 입력 자체를 막는 별도 UX 변경을 섞지 않는다. 401 분기 및 `mutate(result, { revalidate: false })`를 보존한다. SWR 폴링 화면만 보고 저장 여부를 단정하지 말고 PUT과 후속 GET을 본다. E2E의 request fixture와 page는 쿠키 context가 다르므로 화면 조작 후에는 page.request를 쓴다. 반려된 ESLint globalIgnores 수정과 이미 통과하는 단위 글롭 수정은 금지한다.
- 차선 후보: 관리자 가이드 4.4 상황판 제목 저장 실패 시 편집 닫힘 설명 정정 — 첫 과제의 동작이 이미 별도 브랜치에서 해결됐거나 환경 설치가 불가능할 때 문서 1파일만 수정. 직접 확인한 `saveDashboardTitle`은 성공할 때만 닫힌다. `sed -n '108,114p' app/admin/page.tsx`, `sed -n '192,202p' docs/ADMIN_GUIDE.md`, `git diff --check`로 확인하며 앱 코드는 바꾸지 않는다.

근거와 선택:
- main@b5f424a 기준. `lib/activityData.ts:validateActivityImport`를 직접 메모리 호출해 제목301·시간101·상황판제목101의 개별 issues를 확인했다. 유효한 50단계 트리에 `addActivity` 호출 후 검증하면 `activities[50].level`의 1~50 제한 오류도 확인했다. 실제 HTTP·브라우저에서 상세가 숨는 모습은 이번 미실행이며, 라우트 catch의 details 반환과 UI가 error만 읽는 소스 경로에 근거한다.
- `components/ActivityJsonImporter.tsx:handleUpload` 및 feedback 렌더는 이미 동일한 details를 목록으로 표시한다. 이를 동작 참고로만 사용하고 컴포넌트 공통화는 하지 않는다.
- 검토한 대안: (A) 선택안은 서버 계약 추가 없이 공통 저장 경로의 원인을 드러낸다. (B) 초안의 level50 전용 오류 코드는 1개 엣지만 다루고 라우트 계약을 늘려 제외했다. (C) 모든 입력란의 선검증/공유 스키마화는 향후 확장에는 도움이 되지만 클라이언트·서버 제약 동기화 비용이 커 이번 제외했다. (D) 문서만 고치는 안은 가장 작지만 실제 편집 중의 혼란은 남으므로 차선이다.
- 가장 큰 전제: 현 API의 details 메시지가 관리자가 제한을 이해하는 데 충분하다는 판단. `title은 300자 이하여야 합니다.` 등 실제 반환 문구를 확인했으나 사용자 조사는 미확인이다. 프로덕션 API를 바꿀 필요가 없다는 전제가 깨지면 범위를 늘리지 말고 과제서를 수정한다.

구현 순서와 체크포인트 (모두 구현자 자동 검토, 사람 승인 대기 없음):
1. [미착수] 위 Client Components 가이드 확인과 E2E 준비 후, 기존 spec에 실제 400 회귀를 추가한다. 집중 회귀 명령으로 상세 노출만 red인지 확인한다. seed PUT 200 및 후속 GET 검증이 실패하면 먼저 환경/fixture를 바로잡는다.
2. [미착수] admin/page.tsx 한 파일에서 상세 메시지 표시와 형식 방어를 추가한다. 집중 회귀를 green으로 만들고 `npx tsc --noEmit`를 통과한 뒤 다음 단계로 간다.
3. [미착수] ADMIN_GUIDE 4.4 문구를 맞추고 위 최종 검증 명령을 순차 실행한다. 실제 결과를 기록하고 완료 표시한다. 실패 원인이 범위를 넓히면 자동으로 기능을 더하지 말고 계획을 정정한다.

작업량 산정 (구현자 45분 세션):
- Bottom-up: 환경·가이드 4~7분, 실제 400 회귀 7~10분, UI 수정·타입 검증 5~7분, 문서·전체 검증 6~9분 = 기본 22~33분. 알려진 변동(패키지 다운로드·브라우저 seed 안정화) 대비 contingency 5~8분 별도, 합계 27~41분 예상. 통계적 신뢰구간이 아닌 중간 확신의 작업 추정이며 네트워크/환경 정상 가정이다.
- 유사 사례로 10-04 읽기 실패 배너와 09-26 저장 실패 표시의 파일 규모·기존 E2E 재사용을 비교했다. 실소요 시간이 기록되어 있지 않아 유사 추정의 분 단위 교차검증은 불가하다. 관리 예비비(management reserve)는 배정하지 않으며 미발견 기능 확대는 포함하지 않는다. 설치 후나 첫 회귀 실행이 예상보다 길면 재추정하고 차선 적용 여부를 기록한다.
- 추정 방법의 범위·분해·가정·불확실성 기록 원칙 참고: [GAO Cost Estimating and Assessment Guide](https://www.gao.gov/products/gao-20-195g). 이 문헌이 위 분 수치를 제공한 것은 아니다.

적용 스킬: Skill/skills.list/skills.read 도구가 제공되지 않아 아래 정본을 직접 읽었다. solution-exploration의 대안 비교·전제, implementation-planning의 단계별 증명·체크포인트, estimating-and-contingency의 분해·가정·범위·예비 구분을 위에 반영했다.
- [pmo:estimating-and-contingency](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md)
- [technology:implementation-planning](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/implementation-planning/SKILL.md)
- [technology:solution-exploration](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/solution-exploration/SKILL.md)
