- 과제: 가이드 문서 복수 선택 시 첫 실패 사유와 전체 실패 수를 표시한다 (가치 2 / 위험 1 / 작업량 S)
- 왜: `useGuideDocumentDraft.add`는 파일마다 실패를 `error = failure`로 덮어써 여러 첨부가 거부되어도 마지막 한 건만 안내한다. 기존 오류 영역에 첫 실패 사유와 이번 선택의 전체 실패 수를 표시하면 소유자와 관리자가 빠진 첨부를 알아차리고 다시 선택할 수 있다.
- 수용 기준:
  1) 실패 1개면 기존 메시지를 그대로 표시한다. 실패가 2개 이상이면 `첫 실패 사유 (총 N개 파일 첨부 실패)`를 단일 `role="alert"`에 표시한다. 예: 빈 `첫째.pdf`, 빈 `둘째.pdf` 선택 → `첫째.pdf: 빈 파일은 첨부할 수 없습니다. (총 2개 파일 첨부 실패)`.
  2) 실패 수는 현재 선택 배치에서 거부된 파일 수다. 정상 파일은 선택 순서대로 pending에 추가하며, 이전 pending/stored 파일은 유지한다. 중복명·파일 수 초과도 각 거부 파일을 한 번씩 센다. 모두 정상인 다음 선택은 이전 pickError를 지우고 단일 오류는 복수 요약을 붙이지 않는다.
  3) 실제 hook+field DOM 테스트에서 복수 오류의 첫 사유·총수, 정상/실패 혼합 선택, 다음 정상 선택의 오류 해제, 기존 단일 오류 동작을 증명한다. 실제 AppFormPage에 배선된 `/submit`을 Chromium desktop/mobile로 열어 같은 복수 오류와 정상 파일의 대기 표시를 검증한다. 파일 선택만으로 POST/DELETE가 발생하지 않는 기존 계약을 유지한다.
- 건드릴 파일:
  - `web/src/features/apps/guide-documents.tsx:useGuideDocumentDraft.add` (현재 117~148행) — 루프에서 첫 오류만 보관하고 거부 개수를 세어 마지막 setPickError에 요약 문자열을 전달한다. GuideDocumentDraft.pickError 타입과 GuideDocumentsField의 단일 alert를 유지한다. 프로덕션 변경은 이 1파일로 제한한다.
  - `web/src/features/apps/guide-documents.test.tsx:DraftHarness, describe("guide documents on an app form")` — 실제 useGuideDocumentDraft·GuideDocumentsField·QueryClient를 사용하는 기존 하네스와 fetch 경계 대체를 재사용한다. 현재 파일의 6개 테스트는 유지하고 사용자 파일 선택을 통한 회귀를 추가한다.
  - `web/e2e/core.spec.ts` — `installMockApi(page, { authenticated: true })` 후 `/submit`에서 실행하는 `가이드 문서 복수 선택 실패를 요약한다` 테스트 1개 추가. 기존 관리자 첨부/삭제 E2E는 그대로 둔다. mock-api.ts 변경 불필요.
- 검증 명령:
  - 준비: `npm --prefix web ci --no-audit --no-fund`; Chromium이 없으면 `(cd web && npx playwright install chromium)`.
  - 대상: `npm --prefix web test -- src/features/apps/guide-documents.test.tsx`
  - 전체/정적: `npm --prefix web test`; `npm --prefix web run lint`; `(cd web && npx prettier --check src/features/apps/guide-documents.tsx src/features/apps/guide-documents.test.tsx e2e/core.spec.ts)`
  - 번들: `npm --prefix web run build`; `./scripts/check-offline-assets.sh web/dist`
  - 실제 화면: `CI=true npm --prefix web run test:e2e -- --grep '가이드 문서 복수 선택 실패를 요약한다|소유자는 앱 수정 화면에서 가이드 문서를 첨부하고 삭제를 예약한다' --workers=2 --retries=0 --reporter=list` (desktop/mobile 모두, build 선행).
  - 계약/마무리: `./scripts/check-env-contract.sh`; `./scripts/check-docs.sh`; `git diff --check`
- 위험과 피할 것: 확장자·20MB·10개 제한, 대소문자 무시 중복 검사, 성공 파일 순서, 삭제 예약/취소, 저장·업로드 재시도 계약은 바꾸지 않는다. 서버·auth·session·migrations·.github/workflows·문서/PDF·공용 UI·의존성·버전·embed 산출물 변경은 범위 밖이다. 테스트용 가짜 draft를 주입하거나 소스 문자열만 검사하지 않는다. 실패 파일명은 기존처럼 React 텍스트로 표시하고 감사 로그에 기록하지 않는다. 이미 반려된 순수 테스트/입력 하드닝 과제로 확장하지 않는다.
- 차선 후보: `AppGuideDocuments`의 문서 목록 조회 실패를 빈 목록과 구별해 한국어 안내 및 다시 시도로 표시 (가치 3 / 위험 1 / 작업량 S). 같은 `guide-documents.tsx:AppGuideDocuments`는 `documents.data ?? []` 후 `if (!items.length) return null`로 초기 500도 숨긴다. 1순위가 이미 해결됐거나 재현되지 않을 때만 전환한다. `documents.error` 분기에 로컬 한국어 안내와 refetch 버튼을 넣고, HTTP 500→재시도 200→내려받기 링크·정상 빈 목록의 카드 미노출을 기존 테스트와 실제 상세 화면 E2E로 증명한다. 다운로드 자체의 에러 처리는 이 차선의 범위가 아니다.

확인 근거와 한계:
- 기준 HEAD bb1ca54(main, PR #41 병합), package.json/README 버전 2.11.16. git log -30과 이전 회차를 대조했다. 완료 상태 표시는 edc94a7로 해결되어 재선택하지 않았다.
- 실제 호출은 `app-form-page.tsx:85,353`, `admin-pages.tsx:734,1024`에 있다. `guide-documents.tsx:229~244` 입력 onChange는 add 호출 후 value를 비우고, 264행 부근 alert가 pickError를 읽는다. docs/USER_GUIDE.md:64~67와 ADMIN_GUIDE.md:332의 제한 및 저장 전 대기 계약을 유지한다.
- `guide-documents.test.tsx`의 기존 DraftHarness는 실제 hook+field를 렌더한다. E2E의 기존 “소유자는 …” 테스트는 이름과 달리 `/admin/apps/...`를 연다. 신규 E2E는 `/submit`으로 소유자 등록 배선도 검증한다.
- 이번 정찰의 대상 Vitest 실행은 `vitest: not found`, exit 127이었다(node_modules 없음). 실행 재현·전체 테스트·lint/build·E2E 결과는 미확인이다. 이전 회차 126 passed는 전달 기록이지 이번 측정값이 아니다. 환경변수·문서 계약 스크립트는 이번에 각각 성공했다. 정찰은 설치나 코드 수정을 하지 않았다.
- 파일 크기 대역이 필요 없는 빈 PDF 2개와 정상 PDF 1개로 회귀를 만든다. 세 파일 모두 accept에 맞으므로 user-event의 확장자 필터 때문에 실패 파일이 테스트에서 빠질 위험도 피한다. count/duplicate 보존은 필요 시 같은 이름의 정상 PDF 반복 선택으로 검증한다.

구현 순서와 체크포인트 (모두 미착수, 사람 승인 단계 없음):
1. 준비·기준선: 위 설치 명령 후 대상 Vitest 실행, 기존 6건 결과를 기록한다. 환경 준비 실패라면 제품 결함과 구분하고 과제서를 수정한다. 자동 체크포인트: 대상 테스트 실행 가능 여부.
2. 제품 수정+회귀: 복수 오류 테스트를 먼저 실행해 현재 DOM이 마지막 오류만 보여 실패함을 확인한 뒤 add만 수정한다. 대상 Vitest로 단일/혼합/후속 선택까지 통과시킨다. 임시 red 상태는 커밋하지 않고, 코드가 작동하는 상태에서 단계 종료. 자동 체크포인트: 기대 실패→통과 로그.
3. 실제 배선: core.spec.ts에 위 신규 E2E를 추가하고 build 후 지정 grep을 desktop/mobile 모두 실행한다. 정상 PDF만 대기 목록에 남고 두 빈 PDF는 거부되는지 확인한다. 자동 체크포인트: 실제 번들에서 요약 및 정상 첨부 표시, 기존 관리자 저장 흐름 보존.
4. 마무리: 전체 React·lint·변경 파일 Prettier·offline/env/docs·diff 검사를 실행해 결과를 기록한다. 최종 제품 변경을 임시 제거했을 때 대상 회귀가 다시 실패하고 복원하면 통과하는지 확인해 가짜 성공을 배제한다. 자동 체크포인트: 프로덕션 1파일+테스트 2파일 이내, 산출물 미포함. 확대 작업은 ideas에만 남긴다.

대안 비교와 추정 근거:
- 선택: 첫 사유+총 실패 수. 기존 문자열/단일 alert 계약 안에서 해결되며 작고 검증 경로가 이미 있다. 모든 파일의 개별 사유를 보여 주지는 않는 것이 한계다.
- 대안: 파일별 오류 배열/목록은 모든 사유를 보여 주지만 상태·UI 계약과 접근성 설계 범위가 늘어난다. 대량 첨부 지원을 실제로 요구할 때 별도 과제로 적합하다.
- 대안: 현상 유지 또는 복수 선택 제한은 코드 비용은 적지만 누락 인지를 해결하지 못하거나 정상 다중 첨부 기능을 줄이므로 채택하지 않는다.
- 가장 큰 가정: 기존 단일 alert에 첫 사유와 총수만 표시해도 이번 사용자 문제를 해결할 수 있다는 정찰 판단이다. 실행 재현은 구현자가 위 첫 검증 단계에서 확인한다.
- 상향식 예상: 환경·기준선 4~6분, 제품+DOM 회귀 8~10분, 실제 배선 E2E 7~9분, 전체 검증·검토 6~10분 = 기본 25~35분. 알려진 변동(패키지/Chromium 준비·E2E 선택자 조정)에만 별도 contingency 5~10분을 두어 총 30~45분, 주관적 신뢰 중간(약 70%, 통계 보장 아님). 설치 장애가 이 범위를 넘으면 재추정하고 기능을 확대하지 않는다. 관리 예비비는 이번 범위에 배정하지 않으며 미지의 추가 기능은 다음 회차다.
- 유사 추정 교차검토: 10/07은 제품 1파일+테스트 2파일의 표시 수정과 기존 HTTP/E2E 하네스 재사용으로 성공했다. 이번도 같은 변경 규모지만 당시 실제 소요 분 기록이 없으므로 수치 보정 자료로 쓰지 않았다. 일정 가정·작업 분해·불확실성·실측 갱신을 기록하는 근거: [GAO Cost Estimating and Assessment Guide](https://www.gao.gov/products/gao-20-195g). 위 분 단위 수치는 GAO의 수치가 아니라 현재 소스와 작업 분해에 근거한 정찰 추정이다.
- 적용 스킬: Skill 도구가 없어 `/mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md`, `/mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md`, `/mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md` 원문 및 PMO sources.md를 읽었다. 이 과제서에 대안·가정·범위·단계별 증거/체크포인트·추정/예비비를 반영했다.
