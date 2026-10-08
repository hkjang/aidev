## 2026-10-08
- 선택: 관리자 저장 실패 배너에 서버가 반환한 입력 검증 상세 원인 표시 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: admin submitActivityChange가 버리던 유효한 details[].path/message를 기존 요약 아래 줄바꿈된 React 텍스트로 표시하고, 잘못된 상세는 무시하도록 했다. 실제 행 제목301자·시간101자와 상황판 제목101자 PUT 400·동일 브라우저 context 후속 GET의 activities/dashboardTitle/lastUpdated 불변·입력 유지·유효값 재시도 및 상세 제거를 검증했으며, 이는 기존 검증 실패의 설명 개선이지 데이터 손실 수정이나 400 신설이 아니다. 프로덕션1파일·기존 E2E1파일·가이드4.4절1파일만 변경했고 전체 단위108건·E2E48건·타입/lint/build/diff 검사 통과 후 hkjang 작성자·트레일러 없는 c20c037로 커밋했다.
- 실패 재현: 수정 전 집중 회귀 `2 failed / 10 passed`(e2e-red.log). 상세 노출 단정에서만 다음 출력이 나왔고, 시간100자·상황판제목100자 제한도 같은 이유로 실패했다.
  `Expected substring: "title은 300자 이하여야 합니다."`
  `Received string:    "activity.json 데이터 형식이 올바르지 않습니다."`
- 보류 아이디어:
  - level 50 부모 아래 추가 정책 정리 (2/1/S) — 기존 검증 상한과 추가 UX 정책은 이번 제외.
  - 상태 전파를 lib/treeUtils propagateStatus로 옮겨 API 분기와 통일 (3/2/M) — 현재 전파 계약 유지.
  - parentId 자기참조·순환 issues 중복 제거 (2/2/S) — 기존 오류2건 계약 유지.
  - README의 화면별 갱신 주기 정정 (2/1/S) — 저장 오류와 직접 관련 없는 문서 변경 제외.
- 과제서: 채택 — 실제 서버 응답의 상세는 존재하지만 UI에 요약만 나오는 과제서의 원인을 브라우저에서 재현했고 지정 3파일로 해결했다.
- 검증: `PLAYWRIGHT_CHROMIUM_PATH=/usr/bin/google-chrome npm run test:e2e -- e2e/admin-save-failure.spec.ts` 수정 전2 failed/10 passed → 수정 후12 passed(13.2s). 수정만 잠시 되돌려 `--grep '실제 검증 실패'`로 두 테스트의 상세 노출 실패를 재확인한 뒤 수정본을 복원했다(e2e-revert-red.log). 이후 지정 순서대로 `npm run test:unit` 108 pass/0 fail/0 skipped, `npx tsc --noEmit` exit0, `npm run lint` exit0, `PLAYWRIGHT_CHROMIUM_PATH=/usr/bin/google-chrome npm run test:e2e` 48 passed(1.4m, 재시도·skip 없음), `npm run build` exit0, `git diff --check` exit0. 전체 출력은 unit-final.log/tsc-final.log/lint-final.log/e2e-final.log/build-final.log/diff-check-final.log에 보존했다.
- 계약 보존: 기존 상세 없는500·401 회귀 유지, 401 재로그인 뒤에도 이전 상세가 사라짐을 보강. 신규 비정상 details4건과 비JSON502 fallback1건은 page.route로만 주입했고 실제 검증400 두 시나리오는 가로채지 않았다. 기존404·409·GET 실패 중 연속 저장(SWR 성공 응답 캐시 반영) 회귀도 전체 E2E에서 통과했다.
- 체크포인트: 1(설치·가이드·실제400 red), 2(화면 수정·집중 green·타입), 3(가이드4.4·최종 전체 검증) 모두 완료. seed PUT200과 후속 GET, 복원 PUT200도 확인했고 변경 파일 수는 지정대로 총3개/프로덕션1개다.
- 스킬: Skill/skills.list/skills.read 호출 도구가 없어 로컬 정본을 읽고 적용했다. `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/{completion-verification,systematic-debugging,test-driven-development}/SKILL.md`의 실제 실행·원인 증명·red→green→수정 되돌림 재검증을 수행했다. 코드 작성 전 설치본 `node_modules/next/dist/docs/01-app/01-getting-started/05-server-and-client-components.md`와 `01-app/02-guides/server-and-client-boundary.md`를 읽었다.
- 범위·검증 한계: 최대 깊이/활동 개수 초과를 브라우저에서 별도 재현하지 않았으며, HTML 안전성은 JSX 텍스트 렌더 경로로 확인했다(전용 HTML 삽입 테스트 없음). API/저장/인증/메일/의존성/검증 상한/배포/PDF는 변경하지 않았다. npm ci 성공 시 audit 요약10건(9 high/1 critical)이 보고됐지만 영향 분석은 미실시. 단위의 MODULE_TYPELESS_PACKAGE_JSON·E2E 색상 경고 및 릴레이 중단 테스트의 예상 SMTP 실패 로그가 있었고 빌드는 통과했다. 이번 E2E 보고서는 회차 결과 디렉터리로 옮겨 보존했고 생성물은 커밋에 없으며 커밋 후 작업 트리는 깨끗하다.
