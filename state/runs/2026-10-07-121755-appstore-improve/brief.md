- 과제: 보안 심의 화면의 “최종 결과”를 한국어로 표시 (가치 3 / 위험 1 / 작업량 S)
- 왜: `AppSecurityCheckPage`의 최종 결과 행은 `{view.finalResult || "—"}`를 그대로 출력하여 `APPROVED`·`CONDITIONAL`·`REJECTED`라는 내부 코드가 한국어 화면에 노출된다. 서버가 허용하는 값을 확인한 라벨로 바꾸면 소유자가 승인·조건부 승인·반려를 바로 구별할 수 있고 같은 카드의 한국어 상태 안내와도 일치한다.
- 수용 기준:
  1) `/my/apps/:id/security`의 “최종 결과” 행이 `APPROVED → 승인됨`, `CONDITIONAL → 조건부 승인`, `REJECTED → 반려됨`으로 표시된다. 코드는 그 행에 남지 않는다.
  2) 필드가 빠졌거나 빈 문자열이면 기존 `—`를 유지한다. 미래의 알 수 없는 값(예: `DEFERRED`)은 원문을 보존한다. 심의 상태 `REMOTE_STATUS`, 앱 상태 `appStatusLabel`, 배지, `view.verified`에 따른 제출 조건은 그대로다.
  3) 기존 HTTP 하네스에서 위 3개 값·빈 값·누락·미지 값의 실제 DOM을 검사한다. `screen.getByText("최종 결과").closest(".meta-row")` 내부를 검사하여 다른 행의 “승인됨”으로 거짓 통과하지 않게 한다. 번역 3건은 수정 전 실패, 폴백은 수정 전후 통과해야 한다.
  4) 기존 E2E “보안 심의를 마쳐야 앱을 제출할 수 있다”에서 확인 전 최종 결과는 `—`, 확인 후 `승인됨`이며 `APPROVED`가 없음을 단정한다. 기존 제출 버튼 노출 단정도 desktop/mobile 모두 통과한다. 기존 mock은 이미 확인 응답에 finalResult를 보내므로 수정할 필요 없다.
- 건드릴 파일:
  - `web/src/features/security-check/owner-page.tsx:REMOTE_STATUS 인접 위치, AppSecurityCheckPage` — 최종 결과용 로컬 `Record<string, string>` 표를 추가하고 현재 250행의 `<dd>{view.finalResult || "—"}</dd>`만 표 조회 + 원값 + 빈 값 폴백으로 바꾼다. 프로덕션 파일은 이 1개로 제한한다.
  - `web/src/features/security-check/security-check.test.tsx:view, renderOwnerPage, describe("App security check")` — 기존 컴포넌트·라우터·QueryClient·HTTP 경계 하네스를 재사용해 위 사례를 추가한다. 미지 값은 미래 호환성 방어 테스트이며 현재 서버가 이를 허용한다는 뜻이 아니다.
  - `web/e2e/core.spec.ts:보안 심의를 마쳐야 앱을 제출할 수 있다`(556행 부근) — 이미 있는 클릭 시나리오에 최종 결과 행 단정만 추가한다.
- 검증 명령:
  - 준비: `npm --prefix web ci --no-audit --no-fund` (현재 워크트리에는 node_modules 없음).
  - 회귀/기준선: `npm --prefix web test -- src/features/security-check/security-check.test.tsx`
  - 전체 React: `npm --prefix web test`
  - 정적 검사/번들: `npm --prefix web run lint`, `npm --prefix web run build`
  - 브라우저 준비(필요할 때): web 디렉터리에서 `npx playwright install chromium`.
  - 브라우저: `CI=true npm --prefix web run test:e2e -- --grep '보안 심의를 마쳐야 앱을 제출할 수 있다' --workers=2 --retries=0 --reporter=list` (빌드 선행, desktop/mobile 2개 프로젝트).
  - 계약 검사: `./scripts/check-offline-assets.sh web/dist`, `./scripts/check-env-contract.sh`, `./scripts/check-docs.sh`.
  - CI 전체 Go 검사가 필요하면 `go test -race . ./cmd/... ./internal/... ./migrations/... ./openapi/...`; 이번 정찰에서는 미실행. DB 통합 성공으로 해석하지 말 것.
- 위험과 피할 것: `internal/seccheck`의 승인 판정, `security_check_handlers.go`, auth·session·migrations·.github/workflows 및 API 계약을 바꾸지 않는다. 특히 “조건부 승인”을 verified=true로 바꾸는 것은 범위 밖이다. `APP_STATUSES`는 앱 상태용 소문자 enum 표이므로 여기에 SecCheck 결과를 섞지 않는다. 공용 번역 계층, 배지 디자인, 가이드/PDF/캡처 갱신, 버전·임베드 산출물도 범위 밖이다. 즐겨찾기 과제는 `0898f7a`가 HEAD의 조상이 아니며 외부 PR 상태는 미확인이므로 중복 제출하지 않는다.
- 차선 후보: 가이드 문서 여러 개 선택 시 마지막 실패만 남는 안내 개선 (가치 2 / 위험 1 / 작업량 S) — `web/src/features/apps/guide-documents.tsx:useGuideDocumentDraft.add`에서 실패 파일 수와 첫 실패 사유를 한 문구에 남기고 기존 `GuideDocumentsField`의 alert로 보여 주기. `guideDocumentError`, 개수·중복 판정 및 업로드 처리에는 손대지 않는다. 1순위가 이미 해결됐거나 제품 요구와 충돌함이 확인될 때만 전환한다.

범위와 근거
- 기준 HEAD: `8370b8d` (v2.11.15), 시작 시 작업 트리 깨끗함. 코드 변경·커밋 없이 정찰함.
- `internal/seccheck/review.go:parseReview/validResult`(91–96행): 허용값은 빈 문자열, APPROVED, CONDITIONAL, REJECTED뿐이다. 자유 문자열이라는 기존 보류 사유가 해소됐다.
- `internal/httpapi/security_check_handlers.go`의 `FinalResult`는 JSON에서 빈 값이면 생략되며, 응답 생성은 `state.FinalResult`를 그대로 전달한다(198·239행). 따라서 누락/빈 값 둘 다 보존해야 한다.
- 같은 파일의 확인 처리(350–375행)는 바인딩이 맞는 미승인 결과도 기록하므로 조건부/반려 라벨은 실제 저장 가능한 데이터에 해당한다.
- `web/e2e/mock-api.ts`는 초기 finalResult가 빈 문자열(130행), verify POST 결과가 APPROVED(391행)이다. config override를 고칠 필요 없다.
- 정찰에서 `npm --prefix web test -- src/features/security-check/security-check.test.tsx` 실행은 exit 127(`vitest: not found`). 테스트 통과·브라우저 렌더 재현은 미확인이다. 정적 원값 렌더 경로와 서버 계약은 확인했다. 이전 회차 105건은 전달받은 기록이며 이번에 실측하지 않았다.
- `./scripts/check-env-contract.sh`와 `./scripts/check-docs.sh`는 이번 정찰에서 각각 exit 0. CLAUDE.md/AGENTS.md/별도 로드맵·TODO 파일, 검색한 web/src·internal·scripts·docs·README의 TODO/FIXME 표시는 발견하지 못했다.

구현 순서와 체크포인트 (전부 미착수, 사람 승인 대기 없음)
1. 위 준비 명령 후 기존 대상 Vitest 기준선을 실행한다. 이후 회귀 테스트와 최소 제품 수정을 한 묶음으로 작업하되, 제품 수정 전 실행으로 번역 3건의 실패를 기록하고 수정 후 같은 명령의 통과를 확인한다. 체크포인트: 기존 보존 테스트 포함 통과 후 다음 단계로 간다.
2. 기존 E2E에 행 범위를 좁힌 단정을 추가한다. build 후 위 grep 명령으로 두 프로젝트를 실행한다. 체크포인트: 실제 번들에서 최종 결과와 기존 제출 버튼 동작이 함께 통과해야 한다.
3. 전체 React·lint·build·계약 검사를 실행하고 diff가 프로덕션 1개+테스트 2개인지 확인한다. 실패가 환경 원인이면 명령과 원인을 기록하며 성공으로 간주하지 않는다. 기존 파일/사용자 변경을 덮는 무차별 checkout·stash는 피한다. 전제가 틀리면 brief의 근거·범위부터 갱신한다.

대안 비교와 선택 이유 (solution-exploration)
- 선택: 화면 로컬 표 3개 + 원문 폴백. 1개 소비처에 맞는 최소 비용이며 서버 계약을 건드리지 않는다.
- 확장안: 공용 SecCheck 상태/결과 번역 모듈. 소비처가 늘면 유효하지만 지금은 재사용 요구가 없어 새 추상화 비용이 더 크다.
- 현상 유지/기존 REMOTE_STATUS만 재사용: 변경비가 없거나 작지만 CONDITIONAL은 기존 표에 없어 핵심 결과를 해결하지 못한다.
- 기존 즐겨찾기 오류·폼 제출 테스트 후보는 각각 미병합 변경과의 중복 위험·순수 테스트 추가 반려 이력이 있다. 이번 후보는 관찰 가능한 변화와 확정된 계약, 기존 단위/E2E 하네스를 모두 갖춘다.
- 가장 중요한 전제: 현재 백엔드가 정의한 최종 결과 enum이 UI 계약이라는 것. 알 수 없는 값의 원문 폴백으로 향후 확장 시 정보 손실을 피한다.

작업량 근거와 예비시간 (estimating-and-contingency)
- bottom-up 순수 작업 예상: 기준선·하네스 확인 4–6분, 라벨/DOM 테스트 7–10분, E2E 단정과 검증 8–12분, 최종 검사·차이 확인 5–7분 = 24–35분.
- 알려진 변동인 의존성/Chromium 준비에 contingency 5–10분을 별도 배정하여 합계 29–45분. 이는 정찰자의 중간 확신 예상 범위이며 실측 통계나 80% 신뢰구간은 아니다. 네트워크/브라우저 설치 장애 시 초과할 수 있다.
- analogous 교차검토: 2026-10-06의 인접 앱 상태 한국어화(프로덕션 2파일+동일 단위/E2E 하네스)보다 범위가 작다. 과거 소요시간은 제공되지 않아 시간 수치의 독립 검증은 불가하다.
- management reserve는 이번 과제에 배정하지 않는다. API·다른 화면 확장은 별도 아이디어로 남긴다. 설치 완료 시 실제 경과시간으로 다시 추정하며 시간에 맞추려고 검증을 성공 처리하지 않는다.
- 추정의 범위·가정·분해·불확실성을 기록하는 원칙은 [GAO Cost Estimating and Assessment Guide](https://www.gao.gov/products/gao-20-195g)의 공개 요약을 참고했다. 위 분 단위 수치는 이 저장소에 대한 판단이며 GAO가 제공한 값이 아니다.

적용 스킬
- Skill 도구가 노출되지 않아 `/mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md`와 technology의 `implementation-planning/SKILL.md`, `solution-exploration/SKILL.md` 원본을 읽고 적용했다. 질문·추가 승인 없이 이 과제서를 넘긴다.
