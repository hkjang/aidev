- 과제: validateActivityImport의 ID·상황판 제목·활동 개수 상한 경계 테스트 보강 (가치 2 / 위험 1 / 작업량 S)
- 왜: 실제 PUT·JSON 업로드가 공유하는 검증기에 ID 120자·상황판 제목 100자·활동 5000개 제한이 있지만, 각 상한의 허용/거절 쌍을 고정하는 테스트가 없다. 현재 정상 동작을 회귀 테스트로 남기면 제한을 실수로 완화하거나 유효한 최대 크기 입력을 거절하는 변경을 즉시 발견할 수 있다.
- 수용 기준:
  1) lib/activityData.test.ts에서 실제 validateActivityImport를 호출해 ID 120자 허용(반환 id 보존), 121자 거절(issues 경로 activities[0].id)을 각각 증명한다. lib/treeUtils.test.ts의 기존 120자 부모 추가 테스트는 그대로 유지한다.
  2) dashboardTitle 100자 허용(반환 제목 보존), 101자 거절(경로 dashboardTitle), 앞뒤 공백을 붙인 실질 100자 제목 허용(반환값은 공백 제거)을 증명한다. 현재 제목 제한은 trim 후 길이를 검사한다.
  3) 고유 ID를 가진 유효한 루트 5000개를 허용하고 반환 개수·첫/마지막 ID를 확인한다. 5001개는 ActivityImportValidationError이며 issues가 activities 경로 하나만 가리킨다. 제목·ID 초과 사례도 같은 오류 클래스와 정확한 단일 경로를 확인하여 다른 이유로 실패한 테스트가 통과하지 않게 한다.
  4) 위 7개 사례를 별도 it 또는 표 기반 사례로 식별 가능하게 추가한다. 현재 101개를 유지하면 전체 108 pass / 0 fail / 0 skipped가 기대된다. 현재 정상 동작을 고정하는 과제이므로 수정 전 신규 테스트가 실패해야 한다는 조건은 없다.
  5) 변경은 테스트 파일 1개, 프로덕션 파일 0개다. 원본 데이터·API·저장·UI·검증 제한값·의존성·실행 스크립트는 바꾸지 않는다.
- 건드릴 파일: lib/activityData.test.ts: raw(overrides), issuesOf(value), 새 describe 블록 — 기존 헬퍼와 node:assert/strict·node:test 관례를 재사용해 위 7개 경계 사례를 추가한다. 상한 숫자 120·100·5000은 사용자 계약의 기대값으로 테스트에 명시하고, 프로덕션 MAX_ACTIVITY_COUNT를 가져와 입력과 기대값을 함께 이동시키지 않는다.
- 검증 명령: 저장소 루트에서 `npm run test:unit`, `git diff --check`. 이번 실제 실행은 Node v22.23.1에서 npm 명령 exit 0, 101 pass / 0 fail / 0 skipped다. node_modules 없이 동작했다. 이번 테스트 전용 변경에는 npm ci·Next build·브라우저 E2E가 필요하지 않으며 그 결과를 통과했다고 보고하지 않는다.
- 위험과 피할 것: 5000개 입력은 각기 다른 id, parentId:null, level:1로 만들어 순환·중복·계층 오류와 개수 경계를 분리한다. 기존 자기참조 2개 issues 계약은 변경하지 않는다. 한국어 오류 문구 전체 문자열이나 소스 문자열 검사 대신 실제 반환값·오류 클래스·경로를 검증한다. 앱/auth·migrations·workflows·저장 I/O·상태 전파·UUID 생성은 범위 밖이다. 과거 반려된 ESLint globalIgnores 추가, package.json type 변경, 단위 스크립트 수정도 금지한다. AGENTS.md에 따라 코드를 작성하기 전 관련 설치본 Next 가이드를 읽되 이번 환경에는 node_modules/next/dist/docs가 없어 미열람이다. 테스트 전용이며 Next API를 작성하지 않으므로 이 과제 때문에 설치를 추가할 필요는 없다; 앱 코드로 범위를 넓히지 않는다.
- 차선 후보: docs/ADMIN_GUIDE.md 4.4의 상황판 제목 저장 실패 설명 정정 (가치 2 / 위험 1 / 작업량 S) — 첫 과제가 선행 작업으로 이미 해결됐을 때만 선택한다. 현 문서는 100자 초과 시 편집이 닫힌다고 하나 app/admin/page.tsx:110 saveDashboardTitle은 성공할 때만 setIsEditingTitle(false)를 호출한다. 문서 한 문장을 실패 시 입력 유지로 고치고 두 경로를 대조한다.

근거 및 확인 범위 (main@aec0adc, 2026-10-08)
- lib/activityData.ts: validateActivityImport와 MAX_ACTIVITY_COUNT, ActivityImportValidationError를 읽었다. app/api/activities/route.ts PUT과 app/api/activities/import/route.ts POST가 이 함수를 직접 사용하는 것을 확인했다. 모의 검증기나 주입한 의존성을 테스트하지 않는다.
- lib/activityData.test.ts 전체를 읽었다. 관계/정규화/빈 배열/오류 개수 제한 테스트는 있으나 위 경계 쌍은 없다. lib/treeUtils.test.ts에는 120자 부모를 addActivity에 전달하는 기존 회귀가 있어 이를 삭제·복제하지 않는다.
- 실제 함수를 메모리 입력으로 호출한 정찰 실측: id120 통과, id121 → activities[0].id; title100 통과, title101 → dashboardTitle; 공백+title100 통과 및 반환 100자; count5000 통과, count5001 → activities. 거절 3건은 모두 ActivityImportValidationError/단일 issue였다. 이는 아직 파일에 남지 않은 탐색 결과이므로 구현자가 영구 테스트로 작성한다.
- 실제 npm run test:unit이 성공했다. 과거 반복 기록의 '글롭이 확장되지 않아 exit 1'은 이번 환경에서는 재현되지 않는다. 이전 환경과 달라진 원인은 미확인이다.

진행 계획과 점검점 — 모든 구현 단계는 미착수
1. [ ] lib/activityData.test.ts의 raw·issuesOf와 검증기 세 제한을 대조하고 `npm run test:unit`으로 기준선을 확인한다. 자동 점검점: 현재 HEAD에 같은 7사례가 이미 추가됐으면 중복 작성하지 않고 과제서를 갱신한 뒤 차선 후보로 전환한다. 사람 승인 대기는 없다.
2. [ ] 같은 파일에 7사례를 추가한다. `npm run test:unit`으로 기존+신규 전체 통과를 확인한다. 자동 점검점: 기존 계약과 다른 결과가 나면 프로덕션 코드를 고치지 말고 입력의 고유 ID/루트 관계와 기대값부터 점검한다.
3. [ ] `git diff --check`와 `git diff -- lib/activityData.test.ts`로 테스트 1파일 범위를 확인한다. 실행 건수·실패·skip을 구현 기록에 남긴다. 자동 점검점: 불일치가 남으면 완료로 표시하지 않는다. 커밋/릴리즈 절차는 구현자의 별도 지시를 따른다.

접근 비교 (technology:solution-exploration)
- 선택: 기존 Node 단위 파일에서 공개 검증 계약을 직접 확인. 의존성 설치·서버·파일 I/O 없이 데이터 한계 세 쌍을 독립적으로 검증할 수 있다.
- 대안: HTTP 업로드/PUT의 모든 경계 E2E를 추가. 응답·저장 배선까지 확인할 장점이 있지만 이번에는 배선 변경이 없고 브라우저/서버 설치 비용이 커서 보류한다.
- 대안: 외부 속성 기반 테스트 도구 도입. 입력 조합 확대에 유리하지만 세 상한에 비해 의존성과 설계 범위가 크다.
- 현상 유지: 기존 101개만 실행. 추가 비용은 없지만 유효한 최대 입력과 한 칸 초과의 회귀를 계속 놓치므로 선택하지 않는다.
- 핵심 전제: 120/100/5000 제한과 제목 trim 규칙을 그대로 보존하는 작업이다. 제한 정책 변경 요청이 생기면 이 과제의 범위를 늘리지 말고 다시 계획한다.

작업량 근거 (pmo:estimating-and-contingency)
- bottom-up 추정: 기준선/계약 대조 5분, 7사례 작성 12분, 전체 실행 3분, diff/결과 기록 5분 = 기본 25분. 이는 정찰자의 판단값이며 관측된 구현 소요시간은 아니다.
- 알려진 변동 예비비: 대용량 fixture의 중복 ID나 잘못된 오류 경로 수정에 5분을 별도로 둔다. 작업별 값 안에 예비비를 중복 포함하지 않았다. 총 예상 범위 20~35분, 확신 중간, 45분 안의 독립 작업이다.
- management reserve는 배정하지 않는다. 새 API 정책이나 검증기 수정은 미지 범위로 이번 회차에 흡수하지 않는다. 과거 단위 회귀 추가 기록은 방법의 유사성만 참고했고 측정 시간 자료가 없어 통계적 신뢰수준을 주장하지 않는다.
- 적용 스킬: /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md, /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/implementation-planning/SKILL.md, /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/solution-exploration/SKILL.md. Skill/skills.list/read 도구가 없어 정본 파일을 직접 읽었다. 추정 스킬의 references/sources.md도 확인했으며 외부 비용 산식·정량 신뢰수준을 인용하지 않았다.
