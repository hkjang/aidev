# 정찰 과제서 — 2026-09-23 14:34 (base main@37006f3, 작업 트리 무변경)

## A. 우선 과제(자동 배정)의 처리 — 조사 재실행 금지, 판정 유지

- 과제: [수정 과제] 릴리즈 버전 결정 입력 복구 — 승인 근거 확보 전까지 pending 유지 (가치 5 / 위험 2 / 작업량 S)
- 왜: 릴리즈가 같은 이유로 9회 연속 실패했지만 원인은 저장소 코드가 아니라 **릴리즈 관례를 정할 입력이 없다는 것**이다. 이번 회차에 근거를 재확인했고(아래), 저장소 안에서 게이트를 통과시키는 유일한 수단은 `package.json:3` 의 `"version": "0.0.0"` 삭제뿐인데 이는 AGENTS.md 와 docs/RELEASE.md 가 명시적으로 금지한 게이트 완화다.
- 이번 회차 재확인한 사실(모두 이 워크트리에서 직접 실행):
  - `git tag | wc -l` → `0`
  - `ls -a` → `.github` 디렉터리 없음(= 태그/릴리즈에 반응하는 워크플로 파일 자체가 없음). `.gitlab-ci.yml` 만 존재.
  - `package.json:3` → `"version": "0.0.0"`, `package.json:4` → `"private": true`
  - `git status --short | wc -l` → `0`
  - `git log --oneline -10` → 릴리즈 커밋/태그 커밋 0건. 최근 머지는 모두 `auto/*` 개선 PR.
  - `package.json` scripts(1-20행 확인) → `dev*`/`build:*` 뿐. `version`/`release`/`publish` 타깃 없음.
- **구현자가 할 일: 이 항목에 대해 코드/버전/태그/CHANGELOG 를 만들지 말 것.** 그리고 위 6개 명령을 다시 돌리지도 말 것(9회째 같은 결과이며, 재조사는 회차 시간만 쓴다). 원장에는 "진입 조건 미충족으로 무변경" 을 5줄 이내로만 적는다. 판정을 `skipped`/`released` 로 낮추지 말 것.
- 진입 조건(이 4가지가 **출처와 함께** 인계되면 그때 즉시 릴리즈 가능): ① 승인된 증가 단위와 시작 버전(0.0.0 → 0.0.1 인지 0.1.0 인지) ② 태그 형식(`v0.0.1` vs `0.0.1`)과 주석 태그 여부 ③ 릴리즈 커밋 메시지 양식 ④ 릴리즈 노트 위치·양식·언어(CHANGELOG.md 신설 / docs/ / GitHub Release 사용 여부). README·docs/01 의 "0.0.1 / 2025-01-01 / 초기 릴리스" 기재는 실제 릴리즈가 확인되지 않은 문서 문구이므로 근거로 쓰지 말 것.
- 추정(pmo:estimating-and-contingency 기준): 진입 조건이 채워진 뒤의 릴리즈 실행은 bottom-up 으로 S(30분 이내: 버전 필드 2곳 + 태그 + 노트 1개 + 커밋). 진입 조건 자체는 사람의 결정이라 **소요 추정 불가**(estimate 가 아니라 blocker). 이 구분을 원장에 유지할 것.

## B. 이번 회차의 기본 실행 대상 (A 가 진입 조건 미충족이므로 **이쪽을 구현한다**)

- 과제: OCR 업로드 실패가 조용히 삼켜지는 문제 수정 — `ocrParse` 미-await 와 실패 경로 `stopLoading` 누락 (가치 4 / 위험 3 / 작업량 M)
- 왜: `src/views/Support/Ocr/SupportOcr.vue:206` 의 `inf.tools.ocrParse.call(formData);` 는 **저장소에서 유일하게 `await` 없이 호출되는 `inf.tools.*` 호출**이다(19개 호출부 중 18개는 `await`, `grep -rn "inf.tools.\w*\.call" src/` 로 확인). 그래서 업로드가 실패해도 아래 `catch` 는 절대 실행되지 않고(rejection 은 unhandled 로 흘러감) 사용자는 실패 알림 없이 2초 뒤 변환 목록으로 이동해, "변환한 적 없는 파일" 을 기다리게 된다. 또 `catch` 경로에는 `stopLoading()` 이 없어(성공 경로의 `setTimeout` 안에만 있음) 실패 시 전역 스피너가 `loading.vue` 의 60초 자동 해제까지 남는다.
- 정답 형태는 추측이 아니라 **같은 저장소의 형제 구현**에 있다: `src/views/Support/Stt/SupportStt.vue:131-155 startConversion` 은 이름·구조가 동일한 STT 업로드인데 `const res = await inf.tools.sttParse.call(formData)` → `if(!isSuccess(res)) { openAlert(...); return }` → `catch(e){ if(e=='COM') return; openAlert(...) }` → `finally { stopLoading() }` 로 되어 있다. `docs/03-API-개발가이드.md:415` 도 `await inf.tools.ocrParse.call(formData)` 를 정본 사용례로 적고 있다. `isSuccess` 는 `SupportOcr.vue:8` 에 이미 import 되어 있다(현재 이 함수에서 미사용).
- 수용 기준:
  1. `ocrParse` 가 reject 하면(예: 인터셉터의 `'COM'` 이 아닌 네트워크 오류) `openAlert('업로드 실패하였습니다.\n재시도 해주세요.')` 가 실제로 호출되고, `router.push('/support/ocr/list')` 로 **이동하지 않는다**.
  2. `ocrParse` 가 reject 하거나 `isSuccess(res)` 가 false 이면 `stopLoading()` 이 호출되어 `isLoading` 이 `false` 로 돌아온다(60초 자동 해제에 의존하지 않는다).
  3. `e == 'COM'` 인 경우에는 인터셉터가 이미 공통 처리했으므로 **추가 알림을 띄우지 않는다**(`SupportStt.vue:149` 와 동일 계약). 단 이 경우에도 스피너는 꺼져야 한다.
  4. 성공 경로의 관측 가능한 동작은 그대로다: `resetStatus()` → `startPolling(3000)` → 2초 뒤 `/support/ocr/list?mode=...` 이동. **스피너는 이동 직전까지 켜져 있어야 한다**(STT 처럼 `finally { stopLoading() }` 만 쓰면 await 직후 스피너가 꺼져 2초간 빈 화면에서 버튼을 다시 누를 수 있게 되는 새 회귀가 난다 — 이 점이 STT 를 그대로 복사하면 안 되는 유일한 지점).
  5. 테스트는 실제 컴포넌트를 마운트해 위 1·2·4 를 증명해야 한다. 수정을 되돌리면(= `await` 제거) 1·2 단정이 실패해야 한다.
- 건드릴 파일:
  - `src/views/Support/Ocr/SupportOcr.vue:189-222 startConversion` — 206행에 `const res = await` 추가, `isSuccess(res)` 실패 분기 추가(알림 + `stopLoading()` + `return`), `catch` 에 `stopLoading()` 추가. 성공 경로의 `setTimeout` 안 `stopLoading()` 은 그대로 둔다(수용 기준 4).
  - `tests/unit/supportOcrUpload.spec.js` (신규) — `@vue/test-utils` 의 `mount` 사용. 참고할 기존 스펙: `tests/unit/chatStorageLawViewer.spec.js`(무거운 뷰를 실제로 마운트한 선례), `tests/unit/alertPopup.spec.js`.
- 테스트 배선 힌트(실제로 열어 확인한 것):
  - `vi.mock` 대상: `@/api/interface`(`inf.tools.ocrParse.call`), `@/storage/ocrStatusCheckStore`(`canUseOcr` → `{canUse:true,useCount:0,maxUse:3}`, `startPolling`, `resetStatus`), `@/utils/appList`(`getAppInfo`), `@/storage/myNoteBookStorage`(`modeCheckCount`), `@/storage/userStorage`(`readUser`), `@/utils/useFileAccept`(`accept`='.png,.jpg', `size`, `sizeMb`, `cnt`=3, `loading`), `@/utils/globalAlert`(`openAlert` 스파이), `vue-router`(`useRouter` → `{push: vi.fn()}`, `useRoute` → `{query:{mode:'ocr'}}`).
  - 파일 주입: 숨겨진 `<input type=file>`(`SupportOcr.vue:252`) 대신 드롭 경로가 쉽다 — `.file-upload-wrap2`(249행)에 `trigger('drop', { dataTransfer: { files: [new File(['x'],'a.png',{type:'image/png'})] } })`. `onDrop` → `onFileChange(e.dataTransfer,'drop')` 로 들어간다.
  - 변환 실행: `findComponent(MainBtns).vm.$emit('click1')` (245행이 `@click1="startConversion"`).
  - 스피너 단정은 `useGlobalLoading().isLoading.value` 를 직접 읽는다(`src/utils/globalLoading.js` 는 모듈 싱글턴이라 테스트에서 그대로 관측 가능 — `tests/unit/globalLoading.spec.js` 가 이미 그렇게 한다). 케이스 간 누수를 막으려면 각 테스트 끝에 `stopLoading()` 으로 정리할 것.
  - 2초 `setTimeout` 은 `vi.useFakeTimers()` + `vi.advanceTimersByTime(2000)` 로 진행시키되, `await` 가 섞이므로 `await flushPromises()` 를 함께 쓸 것.
- 검증 명령(이 저장소에서 실제로 도는 것 — `node_modules` 가 비어 있으므로 `npm ci` 가 선행):
  - `npm ci`
  - `npm test` — 기준선 23파일 452테스트(직전 회차 기록). 수정 후 통과 + 새 스펙만큼 증가해야 한다.
  - `npx vitest run tests/unit/supportOcrUpload.spec.js` — 신규 스펙 단독.
  - `npm run build:dev` — 통과 확인 후 `dist/` 삭제(커밋 금지).
  - Red 증명: 수정 전 신규 스펙이 실패하는 것을 먼저 확인하고, 수정 후 `await` 만 되돌려 다시 실패하는지 확인(인과 증명). 운영자 지시대로 소스 문자열 검사·대역 주입이 아니라 **실제 마운트·실제 promise** 로 증명할 것.
- 위험과 피할 것:
  - **동작 변화가 하나 있다**: `await` 를 넣으면 `resetStatus()`/`startPolling(3000)`/네비게이션이 서버 응답 **뒤에** 일어난다. `/tools/parse` 의 응답이 즉시 오는지(작업 큐 등록) 오래 걸리는지(동기 변환)는 **미확인**이다 — 백엔드가 이 저장소에 없다. 근거는 형제 STT 가 같은 구조로 await 하고 있다는 것과 docs/03 의 정본 사용례 두 가지뿐이다. 응답이 오래 걸리면 스피너가 그만큼 길어진다. 이 불확실성을 구현 노트에 명시할 것.
  - `finally { stopLoading() }` 로 STT 를 그대로 복사하지 말 것(수용 기준 4).
  - 이 회차에 `globalLoading` 참조 카운트를 도입하지 말 것. `loading.vue` 의 60초 자동 해제가 카운터를 되돌리지 않는 문제가 그대로라 지금 넣으면 스피너가 영구히 안 꺼지는 더 나쁜 회귀가 난다.
  - `src/api/common/interceptors.js`(인증·공통 오류 처리)와 `src/api/interface.js` 는 건드리지 말 것 — 보호 경로이며 이번 수정에 필요 없다.
  - `ChatStorageDetail.vue:887` 의 중복 `startLoading()` 은 직전 회차에 "누수 아님/동작 변화 0" 으로 확정됐다(rejected). 다시 손대지 말 것 — 운영자 지시 "출력·동작이 바뀌지 않는 수정은 넣지 말 것" 에 걸린다.
- 차선 후보: **Alert/Confirm 이 `showHeader` 를 넘기지 않아 `title` 이 영구히 렌더링되지 않는 문제 확정·수정** — `openAlert(msg, error)` 처럼 title 에 Error 객체를 넘기는 호출부 2곳의 증상이 이 때문에 감춰져 있다. 직전 회차에 SFC 마운트가 가능해졌고 `tests/unit/alertPopup.spec.js` 가 이미 있어 증명 수단이 있다. 단 "title 을 보이게 하는 것" 이 기대 계약인지 먼저 `Alert.vue`/`Base.vue` 의 `showHeader` 사용처를 읽어 확정할 것(보이게 했더니 Error 객체가 화면에 찍히면 그건 회귀다 — 그 경우 호출부를 고치는 쪽이 정답).

## C. 옵션 비교(technology:solution-exploration — B 에 대해)

- (1) 아무것도 안 함: 실패가 계속 조용히 삼켜진다. 탈락.
- (2) `.catch()` 만 붙여 fire-and-forget 유지: 네비게이션 타이밍이 안 바뀌어 가장 안전하지만, 실패 알림이 이미 목록 화면으로 이동한 **뒤에** 뜬다. 인터셉터의 `'COM'` 규약(`if(e=='COM') return`)과도 맞지 않는다. 탈락(차선으로만 보관).
- (3) **`await` + `isSuccess` 분기 + 실패 경로 `stopLoading` (채택)**: 형제 STT 와 docs 정본 두 출처가 동일 형태를 지지한다. 비용은 응답 지연이 스피너 시간에 반영되는 것.
- (4) OCR/STT 업로드 로직을 공용 컴포저블로 추출: 이번 회차 범위 밖(대규모 리팩터 금지). 탈락.
- 가장 크게 기대는 가정: `/tools/parse`(`src/api/interface.js:394-399`) 가 형제 엔드포인트 `/tools/parse/stt`(`src/api/interface.js:421-425`, await 로 소비됨)와 같은 응답 특성을 가진다는 것. 여기가 틀리면(동기 변환이라 응답이 수십 초) 이 계획의 UX 비용이 커진다. 백엔드가 이 저장소에 없어 **미확인**.
