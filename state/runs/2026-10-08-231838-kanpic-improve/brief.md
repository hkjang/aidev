- 과제: 대사가 `0x10`·`0b10`·`0o10` 문자열 식별자를 십진 숫자와 잘못 일치시키지 않게 한다 (가치 3 / 위험 2 / 작업량 S)
- 왜: `compareKey`의 `Number(text)`가 문자열 `0x10`·`0b10`·`0o10`을 각각 숫자 16·2·8과 같은 키로 만들어, 다른 목록인데도 화면에 “두 목록이 키 기준으로 완전히 맞습니다”라고 표시한다. 비십진 문자열을 식별자로 유지하면 잘못된 일치와 가짜 중복을 막고, 사용자 가이드의 “번호는 번호로 둡니다” 계약을 지킨다.
- 수용 기준:
  1) `compareKey('0x10')==='0x10'`, `compareKey('0b10')==='0b10'`, `compareKey('0o10')==='0o10'`이고 각각 숫자 16·2·8의 키와 다르다. 대문자 접두어와 앞뒤 공백도 같은 텍스트 키로 정규화한다(`' 0X10 '` → `'0x10'`). 같은 식별자끼리는 기존 대소문자 무시 규칙대로 일치한다.
  2) 프로덕션 `compareLists`에서 왼쪽 `['0x10']`, 오른쪽 `[16]`은 `both=0`, `onlyLeft`/`onlyRight` 각각 1개이고 label은 원래 값인 `0x10`/`16`이다. 나머지 두 진법도 같다. 왼쪽 `['0x10',16]`, 오른쪽 `[16]`은 `both=1`, `onlyLeft`의 label은 `0x10`, `duplicated=[]`이다. 실제 반복 `['0x10','0X10']`은 여전히 중복으로 표시한다.
  3) 기존 십진수/지수/금액 계약을 유지한다: `'300'`↔300, `'+5'`↔5, `'.5'`↔0.5, `'1.'`↔1, `'1e3'`↔1000, `'1,234'`↔1234, `'₩5,000'`↔5000, `'50%'`↔0.5. `007`과 `7`, 서로 다른 20자리 계좌번호는 여전히 분리한다. 기존 18개 대사 테스트를 바꾸어 기대치를 느슨하게 만들지 않는다.
  4) 실제 `CompareDialog`를 렌더링하고 머리글 포함 `['번호','0x10']`와 `['번호',16]`을 전달하면 “완전히 맞습니다” 문구가 없어지고 왼쪽/오른쪽에만 있는 항목이 각각 보인다. helper를 mock하거나 소스 문자열을 검색해서 증명하지 않는다. 저장 API까지의 왕복 검증은 이번 범위 밖이며 미확인으로 남긴다.
- 건드릴 파일:
  - `web/src/lib/compareLists.ts:compareKey` — 이미 있는 `spreadsheetNumber`를 import하고, **현재 위치에서 문자열 `text`에 대해서만** `const numeric=spreadsheetNumber(text)`와 `numeric!==undefined`로 숫자 인식 분기를 좁히는 방식을 권장한다. `looksLikeIdentifier`를 먼저 통과하고, 실패하면 기존 `textToNumber`→`toLowerCase`로 가는 순서는 유지한다. 숫자 0도 정상 키가 되므로 truthy 검사 금지. 다른 타입의 조기 반환은 유지한다.
  - `web/src/lib/compareLists.test.ts` — 위 키/최종 결과/정상 입력 표를 추가한다. 기존 `side` 헬퍼는 실제 Cell Map을 만들므로 재사용 가능하다.
  - `web/src/components/CompareDialog.test.tsx` (새 테스트 파일) — 실제 컴포넌트에서 수용 기준 4를 검증한다. React Testing Library·jsdom이 이미 설치 대상이며 vitest.config.ts가 이 경로를 수집한다. 가짜 compareLists를 주입하지 않는다.
  - 프로덕션 변경 1파일 + 테스트 2파일. `CompareDialog.tsx`, `EditorPage.tsx`, `spreadsheetNumber.ts`는 읽기 참고이며 수정 대상이 아니다.
- 검증 명령:
  - 의존성이 없으면 `cd web && npm ci` (현재 체크아웃에는 node_modules 없음).
  - 표적: `cd web && npm test -- compareLists CompareDialog` — 새 실패 테스트를 먼저 실행하고, 수정 후 같은 명령을 통과시킨다.
  - 회귀: `cd web && npm test && npm run lint && npm run build`. 공용 fixture가 저장소 루트 `testdata`에 있으므로 원래 저장소에서 실행한다. 전체 검증은 구현자 몫이며 정찰에서는 미실행.
  - 정찰에서 실제 실행한 명령: 회차 폴더의 `web-probe`에서 `npm ci --ignore-scripts --no-audit --no-fund`, `npm test -- compareLists scout-compare` → 2파일·21테스트 통과(기존 18 + 오동작 관찰 probe 3). 원본 프로덕션 소스를 그대로 복사해 실행했으며 원본 저장소는 수정하지 않았다.
  - 증거: 같은 회차 폴더 `compare-probe.log`, `web-probe/src/lib/scout-compare.test.tsx`. probe는 **수정 전 오동작을 기대해 PASS하는 관찰용**이므로 회귀 테스트로 그대로 복사하지 말고 위 정상 기대값으로 작성한다.
- 위험과 피할 것:
  - 확인한 배선: `EditorPage.tsx:openCompare`가 실제 셀을 읽음 → `CompareDialog.tsx`의 useMemo가 `compareLists` 호출 → 결과를 화면과 `onReport`에 전달 → `EditorPage.tsx:writeCompareReport`가 onlyLeft/onlyRight/duplicated를 새 시트 행으로 씀. 함수 최종 결과와 실제 컴포넌트 표시까지 실행 확인했으며 서버 저장 왕복은 미확인.
  - `spreadsheetNumber.ts`의 DECIMAL_TEXT는 이미 비십진 입력을 거절한다. 이를 문자열 분기에 재사용하는 작은 변경만 한다. 업로드·붙여넣기·계산 파서 통합, 새 공통 파서 추출, 숫자 정밀도 정책 변경은 금지한다.
  - `looksLikeIdentifier('007.5')`의 기존 결과, boolean과 문자열 TRUE의 키 차이, 인용된 시트 이름의 작은따옴표 처리는 별도 후보이며 이번에 고치지 않는다. 원본 Cell 값·라벨을 숫자로 바꾸거나 저장된 워크북을 마이그레이션하지 않는다.
  - 보호 경로 auth/session/migrations/workflows 및 Go 코드를 피한다. 현재 기준 main@6e71a7a에는 이전 회차 DST 커밋 f2c1ad1이 아직 없지만 이미 진행한 과제이므로 다시 구현하지 않는다.
  - 문서 `docs/USER_GUIDE.md:254`는 이미 번호 보호를 설명한다. 이번에는 문서/PDF 변경 없이 그 계약을 만족시킨다.
  - 작업 추정 35분 + 여유 10분: 실패 테스트 10분, 한 분기 수정 5분, 표적/UI 확인 10분, 전체 웹 검사 10분, 설치/수리 여유 10분. 전역 설정 변경이나 의존성 업그레이드는 필요 없다.
- 차선 후보: cron step 자리의 `+`부호 거절 (가치 2 / 위험 2 / S) — 1순위가 이미 다른 변경으로 해결된 경우에만 `internal/automation/schedule.go:parseCronField`의 `stepRaw`를 기존 `isDecimalDigits`로 제한하고 schedule_test.go에 ParseSchedule→Next 표를 추가한다. 이번 정찰에서 Atoi 직행은 확인했으나 +step 실행 재현은 미실행이므로 먼저 재현할 것. 저장된 +step 호환성과 진행 중 DST 작업의 파일 충돌을 확인하고, Next/matchesDay를 함께 수정하지 않는다.

정찰 환경 제한: 요청된 pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration은 현재 도구 목록/리소스 및 로컬 스킬 경로에서 찾지 못했다. 해당 스킬의 절차·반환 형식 적용은 미확인이며, 위 내용은 사용자가 제공한 정찰 절차에 따른 결과이다.
