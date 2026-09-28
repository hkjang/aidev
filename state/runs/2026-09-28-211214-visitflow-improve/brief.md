# 과제서 — 2026-09-28-211214-visitflow-improve (base main@bd249c5 / v2.8.8)

- 과제: 방문 신청 화면에서 제출 버튼이 **말없이 잠기는 이유**를 화면에 안내 (가치 3 / 위험 1 / 작업량 S)
- 왜: `web/src/pages/VisitFormPage.tsx:139` 의 제출 버튼 `disabled` 식에는 `!checklistSatisfied || !declarationsSatisfied || visitors.some((x) => !x.name || !x.phone || !x.consent) || (walkIn && !hostUserId)` 가 들어 있는데, 이 네 가지는 `submit()` 가드(`VisitFormPage.tsx:98-104`, 현재는 회사명·일정·방문자 이름/전화·인원 상한·반복 상한만 본다)에도 없고 화면 문구에도 없다. 특히 동행 방문자의 "개인정보 수집·이용 동의" 체크박스(`VisitFormPage.tsx:138` 끝)를 풀면 버튼이 이유 없이 잠기고, 서버 `internal/app/visits.go:455` 는 그 값을 이름·전화와 **같은 줄**에서 `invalid_visitor` 로 거절하는데도 `web/src/visitors.ts` 의 `visitorFieldErrors` 는 consent 를 아예 다루지 않는다. 고치면 v2.8.4~v2.8.8 에서 닫아 온 "화면이 서버 경계를 먼저 말해 준다" 계열의 남은 구멍이 닫히고, 사용자가 무엇을 고쳐야 하는지 알게 된다.

## 수용 기준
1) 아래 네 상황에서 제출 버튼이 잠긴 **이유가 한국어로 화면에 보인다**(제출 버튼 근처 한 줄 또는 Alert):
   - 방문자 중 한 명이라도 개인정보 동의 체크를 풀었을 때 → 방문자 번호를 붙여 안내(예: `방문자 2: 개인정보 수집·이용 동의를 확인해 주세요`)
   - 선택한 방문 유형이 `requiresNda`/`requiresSafetyBriefing` 인데 해당 체크가 비었을 때
   - 선택한 방문 유형이 `requiresVehicle`/`requiresEquipment` 인데 그 방문자의 차량번호/반입 장비가 비었을 때(방문자 번호 포함)
   - 현장 등록(`/lobby/walk-in`)에서 방문 담당자를 고르지 않았을 때
2) 같은 값 하나가 **세 곳을 모두** 지배한다: 화면 안내 · 제출 버튼 `disabled` · `submit()` 가드. `disabled` 항만 지운 변이에서도 `submit()` 가드가 요청을 막고 같은 한국어 문구를 띄운다(브라우저로 확인).
3) 순수 함수 단위 테스트(`web/src/visitors.test.ts`, **반드시 `.ts`** — vitest `include` 가 `src/**/*.test.ts` 라 `.tsx` 는 조용히 0개가 된다)가 네 원인의 문구와 "막을 것이 없으면 `""`", 그리고 **여러 원인이 겹칠 때 하나만 보여 준다**(첫 원인 우선)를 고정한다. 새 테스트는 스텁 단계에서 실패하는 것을 먼저 확인할 것.
4) 아직 손대지 않은 빈 필수 칸(이름·전화·방문 목적)만으로는 새 안내가 뜨지 않는다 — 첫 화면을 빨갛게 칠하지 않는 `visitors.ts:1-6` 주석의 기존 방침을 유지한다.
5) 서버 응답 계약과 기존 안내(회사명·일정·인원/반복)는 그대로다.

## 건드릴 파일 (프로덕션 2개)
- `web/src/visitors.ts` — 순수 함수 하나 추가(예: `submitBlockReason(input): string`). 입력은 화면 state 에서 바로 만들 수 있는 평범한 객체로: `{ visitors: Array<{ consent, vehicle, equipment }>, requiresNda, requiresSafetyBriefing, requiresVehicle, requiresEquipment, checklistNda, checklistSafetyBriefing, walkIn, hostUserId }`. 기존 `visitorFieldErrors`·`visitorsError`·`visitorCountError`·`recurrenceError` 의 스타일(한국어 문구·방문자 번호 접두)을 그대로 따르고, 기존 export 의 동작은 바꾸지 말 것(회귀 47개가 그 경계를 고정하고 있다).
- `web/src/pages/VisitFormPage.tsx` —
  - `84-95` 줄 근처의 파생값 블록에 `const blockReason = submitBlockReason({...})` 를 두고, 이미 있는 `checklistSatisfied`/`declarationsSatisfied` 계산은 새 함수로 대체하거나(권장: 단일 출처) 남기더라도 **버튼 disabled 가 읽는 값은 하나여야** 한다.
  - `submit()`(98줄) 가드에 기존 가드와 같은 형태로 `if (blockReason) { setError(blockReason); return; }` 추가(위치는 회사명 가드 앞뒤 어디든, 기존 순서 유지).
  - `139` 줄 제출 버튼: `!checklistSatisfied || !declarationsSatisfied || ... !x.consent ... || (walkIn && !hostUserId)` 를 `blockReason !== ""` 로 정리(같은 조건임을 테스트로 고정할 것). `visitors.some((x) => !x.name || !x.phone)` 와 `busy || templateLoading || !siteId || !purpose` 는 그대로 둔다.
  - 안내 렌더: 취소/제출 버튼이 있는 `Stack`(139줄) 바로 위에 `blockReason` 이 비어 있지 않을 때만 한 줄 표시. 빨간 필드 오류가 아니라 조용한 안내(예: `Alert severity="info"` 또는 `Typography variant="body2" color="error"`)로 두는 편이 4)와 어울린다. 원인 필드(동의 체크박스/차량번호/반입 장비)에 `error`/`helperText` 를 같은 값에서 덧붙이는 것은 선택 사항 — 하되 같은 함수 결과만 읽을 것.
- `web/src/visitors.test.ts` — 테스트(프로덕션 파일 수에 포함하지 않음).

## 검증 명령 (이 저장소에서 실제로 도는 것)
- `cd web && npm ci && npm run lint && npm test && npm run build` — 워크트리에 `web/node_modules` 가 없으므로 `npm ci` 부터 필요(이번 정찰에서 미실행). `lint`=`tsc -b`, `test`=`vitest run`(현재 47개).
- 서버를 건드리지 않더라도 습관대로: `go build ./...`, `go vet ./...`, `gofmt -l .`, `git diff --check`.
- 1급 증거(프로필의 관례): 실제 `web/dist` → `cmd/visitflow/webdist` 복사 → `go build` → postgres + 서버 기동 → `playwright-core` chromium(`executablePath=/usr/bin/google-chrome`)으로 동의 해제·유형 체크리스트·차량/장비·현장 담당자 네 경우를 확인. MUI 함정은 프로필 참고(필수 표시는 thin space+`*`, select 은 `aria-labelledby`, 제출 버튼은 `.MuiCard-root` 로 스코프, `newContext()` 필요). 끝나고 `cmd/visitflow/webdist/index.html` 스텁 복원 필수.

## 위험과 피할 것
- 서버(`internal/app/visits.go`, 특히 `createVisitRecord`·`normalizePhone`)·`auth.go`·마이그레이션·`settings.go` 는 건드리지 말 것. 화면만 고친다.
- **새 정책을 만들지 말 것**: `requiresVehicle`/`requiresEquipment`/체크리스트는 서버가 검사하지 않고(`visits.go` 에 `RequiresVehicle`·`Checklist` 검증이 없음 — grep 으로 확인) **화면만의 기존 게이트**다. 이번 과제는 그 게이트를 *설명*하는 것이지 서버로 옮기거나 느슨하게 하는 게 아니다. consent 만 서버 경계(`visits.go:455`)다.
- 미머지 브랜치 3개(`origin/auto/2026-09-16-1212` 메일, `2026-09-18-0533` MCP OAuth, `2026-09-21-0654` 가져오기)와 파일이 겹치지 않는다 — 그쪽은 손대지 말 것.
- 되돌리기에 `git checkout -- <파일>` 을 쓰지 말 것(미커밋 테스트를 두 번 잃었다). 임시 복사본을 쓸 것.
- `pkill -f 'vf-server'` 는 자기 셸까지 죽인다(exit 144). `ENCRYPTION_KEY` 는 64자 hex.
- 규모: 프로덕션 파일 2개. 커지면(필드별 error 배선까지 다 하려다 파일이 늘면) 안내 한 줄 + 가드까지만 이번 회차에 담고 나머지는 보류로 넘길 것. 예상 소요 30~60분(S, 80% 신뢰). 가이드 문서 한 문장 갱신은 선택(PDF 재생성 불필요).

## 미확인 (추측으로 적은 것 — 구현자가 확인할 것)
- 네 원인을 한 줄에 하나만 보여 줄지, 원인별로 필드에 붙일지는 UX 판단으로 남긴다. 브라우저에서 실제로 읽히는 쪽을 고를 것.
- `checklist` state 는 `Record<string, boolean>` 이고 유형 변경 시 `setChecklist({})` 로 초기화된다(128줄) — 새 함수에 `checklist.nda === true` 형태의 엄격 비교를 그대로 넘겨야 기존 `checklistSatisfied` 와 같은 값이 된다. 동등성은 테스트로 고정할 것.
- `web/e2e/visit-flow.spec.ts` 가 제출 버튼 disabled 를 전제로 하는 단계가 있는지 미확인(이번 회차에 열지 않았다). 안내 한 줄 추가는 무해하겠지만 E2E 를 돌릴 여력이 있으면 확인할 것.

## 차선 후보
- ScannerPage·LobbyPage 의 reference-data 로드 실패 처리(가치 2 / 위험 1 / S) — `web/src/pages/ScannerPage.tsx:21` 의 `api<ReferenceData>("/api/v1/reference-data").then((x) => {...})` 에 `.catch` 가 **없다**(직접 확인). 실패하면 unhandled rejection 이 되고 `lobbyId` 가 비어 화면이 조용히 죽는다. `web/src/pages/LobbyPage.tsx` 는 삼키기만 한다. v2.8.8 에서 `VisitFormPage` 에 세운 `loadReference()` + 별도 `refError`/`refLoading` + 닫히지 않는 재시도 Alert 관례를 그대로 옮기면 된다(프로덕션 파일 2개, 증거는 브라우저 확인).
