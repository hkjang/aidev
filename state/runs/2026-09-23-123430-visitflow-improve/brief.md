# 정찰 과제서 (2026-09-23)

- 과제: 방문 신청 화면에서 방문 시작·종료 시각을 제출 전에 검사 (가치 3 / 위험 1 / 작업량 S)

- 왜: `web/src/pages/VisitFormPage.tsx:77`의 `submit()`은 `new Date(startAt).toISOString()`을 무방비로 호출하는데, 제출 버튼(같은 파일 114행)의 `disabled` 조건에 startAt·endAt이 전혀 없어 `type="datetime-local"` 칸을 비운 채로 누를 수 있다. 값이 비면 `new Date("")`가 Invalid Date라 `toISOString()`이 RangeError를 던지고(ECMA-262 계약), 그 예외를 `catch`가 그대로 받아 한국어 화면에 영어 엔진 메시지("Invalid time value")를 띄운다. 종료<시작이나 31일 초과는 서버 왕복 뒤에야 `invalid_schedule`·`schedule_too_long`으로 거절되어, 이미 화면에서 미리 막고 있는 회사명·체크리스트·장비 정책(`companiesSatisfied`/`checklistSatisfied`/`declarationsSatisfied`)과 일관되지 않는다. 고치면 같은 방식의 사전 안내로 빈 값 크래시가 사라지고 잘못된 일정이 네트워크 전에 걸러진다.

- 수용 기준:
  1) 시작·종료 중 하나라도 비었거나 날짜로 해석되지 않으면 제출 버튼이 비활성이고, 해당 입력 칸에 한국어 안내가 보이며, 영어 "Invalid time value"는 어떤 경로로도 나오지 않는다.
  2) 종료 ≤ 시작이면 서버로 요청을 보내지 않고 서버와 같은 문구("방문 종료시간은 시작시간 이후여야 합니다")로 막고, 시작~종료가 31일을 넘으면 같은 방식으로 "한 방문 일정은 31일을 초과할 수 없습니다"로 막는다.
  3) 경계가 서버와 정확히 일치함을 테스트가 증명한다 — `endAt == startAt`은 거절(서버 `!in.EndAt.After(in.StartAt)`), 정확히 31일 0분은 **통과**(서버는 `> 31*24*time.Hour`만 거절), 31일 + 1분은 거절. 즉 화면이 서버가 받아 줄 일정을 막지도, 서버가 거절할 일정을 통과시키지도 않는다.
  4) 기존 정상 경로(기본값 그대로 일반 신청 / `walkIn` 현장 등록)는 종전대로 제출되고, 추가한 검사가 기존 `disabled` 조건을 바꾸지 않는다.

- 건드릴 파일:
  - `web/src/schedule.ts` (신규) — 순수 함수 `scheduleError(startAt: string, endAt: string): string` 하나. 빈 값/파싱 실패/종료≤시작/31일 초과를 위 문구로 돌려주고 정상이면 `""`. 파싱은 `submit()`과 동일하게 `new Date(value)`로 하고 `Number.isNaN(d.getTime())`으로 판정할 것(다른 파서를 새로 만들면 화면과 제출이 서로 다른 값을 읽게 된다).
  - `web/src/schedule.test.ts` (신규) — vitest는 `web/vite.config.ts`의 `test.include: ["src/**/*.test.ts"]`로 `.ts`만 수집한다. `.tsx` 컴포넌트 테스트는 **수집되지 않으니** 만들지 말 것. 기존 `web/src/silentSso.test.ts`가 이 파일 배치의 선례다.
  - `web/src/pages/VisitFormPage.tsx` — (a) `companyRequired`/`companiesSatisfied` 바로 아래에 `const scheduleMessage = scheduleError(startAt, endAt)` 한 줄, (b) `submit()` 첫머리의 `if (!companiesSatisfied) {...}` 가드 옆에 같은 형태의 `if (scheduleMessage) { setError(scheduleMessage); return; }`, (c) 114행 제출 버튼 `disabled={...}`에 `|| scheduleMessage !== ""` 추가, (d) 101행 "방문 시작"·"방문 종료" `TextField`에 `error`/`helperText`로 같은 문구 표시.
  - 문서: `docs/USER_GUIDE.md`의 방문 신청 절에 한 문장(선택). PDF 재생성은 하지 말 것(이번 변경은 캡처 대상이 아님).

- 검증 명령:
  - `cd web && npm ci && npm run lint && npm test && npm run build` (lint는 `tsc -b`. 현 HEAD의 vitest 통과 수는 8개였으니 새 케이스만큼 늘어야 한다.)
  - 변이 확인: 새 `disabled` 항과 submit 가드를 지운 상태에서 새 테스트가 실제로 실패하는지 볼 것. 되돌릴 때 `git checkout -- <파일>`을 쓰지 말 것(과거 회차에서 미커밋 편집을 두 번 잃었다) — `sed`나 임시 복사본으로 되돌린다.
  - Go 쪽은 이번 변경 대상이 아니지만 저장소 관례상 `go build ./...`·`go vet ./...`는 통과 상태를 유지할 것.
  - 브라우저 확인(권장, 인프라가 되면): `cd web && npm run test:e2e`. **주의 — Playwright 설정은 서버를 자동 기동하지 않는다.** 실제 `web/dist`를 `cmd/visitflow/webdist`에 복사해 Go 서버·PostgreSQL을 띄운 뒤에만 의미가 있다. e2e가 준비되지 않으면 그 사실을 보고서에 적고 단위 테스트로 증명할 것.

- 위험과 피할 것:
  - 서버의 `createVisitRecord`(`internal/app/visits.go:440-445`)는 **건드리지 말 것.** 서버는 이미 올바르게 거절한다. 이번 과제는 화면 사전 검사만이며, 서버 문구·상태코드를 바꾸면 MCP·API 경로의 계약까지 흔든다.
  - `localInput`(VisitFormPage.tsx:28)의 기본값 생성 로직과 타임존 처리(datetime-local은 로컬 시각)를 바꾸지 말 것. 새 헬퍼는 문자열을 읽기만 한다.
  - 미머지 브랜치와의 충돌 회피: `origin/auto/2026-09-16-1212`(메일, AdminPage·settings), `origin/auto/2026-09-18-0533`(MCP OAuth, SettingsPage), `origin/auto/2026-09-21-0654`(가져오기 중복 헤더, import.go·import_test.go·가이드)는 HEAD 미머지다. 이 세 브랜치가 손대는 파일은 피하고, `VisitFormPage.tsx`는 이들과 겹치지 않는다(확인함).
  - 효과 없는 변경 금지: 기존 `required` 속성만 늘리거나 `min` 속성만 추가하는 식으로는 버튼이 여전히 눌리므로 수용 기준 1)을 만족하지 못한다.
  - 보호 경로(auth.go·keys.go·migrations·workflows)는 이번 과제와 무관하니 열지 말 것.

- 차선 후보: 방문자 추가 버튼을 100명에서 차단 — `VisitFormPage.tsx`의 "방문자 추가"는 무제한 append인데 `createVisitRecord`는 101명부터 `required_fields`로 거절한다(확인함). CSV로 100명을 채운 뒤 추가 버튼을 막고 안내를 띄우는 같은 성격·같은 파일의 S 과제라, 1순위가 성립하지 않으면 그대로 대체할 수 있다.

## 미확인 (구현자가 직접 확인할 것)
- 실제 Chromium에서 `datetime-local` 칸을 비웠을 때 React state가 `""`가 되어 위 크래시가 재현되는지는 **이 정찰에서 브라우저로 재현하지 않았다.** 근거는 소스(무가드 `toISOString()`, `disabled`에 날짜 없음)와 `Date.prototype.toISOString`의 RangeError 계약이다. 구현 전에 브라우저나 단위 테스트로 먼저 재현할 것.
- `web/node_modules`가 이 워크트리에 없어 `npm test`/`lint`는 이번에 실행하지 못했다(미실행). `npm ci`부터 돌릴 것.
