# 과제서 — 2026-09-23-150423-visitflow-improve (visitflow)

- **과제: 방문 신청 화면에서 방문자 이름·휴대전화를 서버와 같은 규칙으로 제출 전에 검사 (가치 3 / 위험 1 / 작업량 S)**

## 왜

서버 `internal/app/visits.go:454-457`의 `createVisitRecord`는 방문자마다 `strings.TrimSpace(visitor.Name) == "" || len(normalizePhone(visitor.Phone)) < 7 || !visitor.Consent` 이면 400 `invalid_visitor` "방문자 이름, 휴대전화, 개인정보 동의는 필수입니다"로 거절하는데, 화면(`web/src/pages/VisitFormPage.tsx:117`)의 제출 차단 조건은 `visitors.some((x) => !x.name || !x.phone || !x.consent)` — **비어 있지 않기만 하면 통과**한다. 그래서 휴대전화에 `010`(3자리)이나 `없음`을 적거나 이름을 공백만 넣으면 버튼이 활성화되고, 서버 왕복 뒤에야 거절되며, 돌아오는 메시지에는 **몇 번째 방문자가 문제인지가 없다**(최대 100명까지 등록 가능한 화면이라 CSV로 불러온 뒤에는 사실상 찾을 수 없다).

고치면 잘못된 칸이 그 자리에서 빨갛게 표시되고, 제출 자체가 막히며, 서버 왕복과 "어느 줄인지 모르는 400"이 사라진다. 직전 두 회차에서 채택된 회사명 필수(c9f53e3)·일정 사전 검사(cf0bd7e)와 **같은 모양의 마지막 남은 구멍**이다.

## 수용 기준

1. 휴대전화 칸에 숫자가 7자리 미만인 값(`010`, `없음`, `010-1234` → 6자리)을 넣으면, 그 방문자 카드의 휴대전화 `TextField`가 `error` 상태가 되고 한국어 `helperText`가 뜨며 **제출 버튼이 비활성**된다. 이름 칸에 공백만(`"   "`) 넣어도 같다.
2. 그 상태에서 제출을 시도해도 **네트워크 요청이 나가지 않고**, `submit()` 가드가 `방문자 3: 휴대전화를 숫자 7자리 이상 입력하세요`처럼 **몇 번째 방문자인지 번호를 붙여** 상단 `Alert`에 적는다(서버 메시지는 줄 번호가 없다는 것이 이 과제의 핵심).
3. **아직 아무것도 입력하지 않은 빈 칸은 종전 그대로**다 — 빨간 표시도 `helperText`도 뜨지 않고(첫 화면이 온통 빨갛게 되면 안 된다), 버튼은 지금과 똑같이 기존 `!x.name || !x.phone` 조건으로 비활성된 채다.
4. 정상 값(`010-0000-0000`, `01000000000`, `+82 10 0000 0000`, 이름 `홍길동`)은 아무 표시 없이 통과하고 제출된다. 서버가 받는 `body.visitors`는 **지금과 한 글자도 달라지지 않는다**(값을 정규화해서 보내지 말 것 — 전화 해시 입력이다).
5. 테스트는 (a) 순수 함수가 서버 규칙과 같은 경계를 지키는 것(6자리 거절 / 정확히 7자리 통과 / 하이픈·공백·`+82`는 숫자만 세기 / 공백만 이름 거절 / 빈 값은 ""(무표시) 반환), (b) 표시·버튼·submit 가드 **세 곳이 같은 함수 하나를 읽는 것**을 증명해야 한다. 고치기 전에 먼저 실패하는 것을 눈으로 확인하고 적을 것.

## 건드릴 파일

- **`web/src/visitor.ts` (신규)** — 순수 함수. `web/src/schedule.ts`가 이미 같은 역할을 하고 있으니 그 파일의 모양(주석으로 서버 대응 지점 명시, 문자열 반환, `""`=문제 없음)을 그대로 따를 것.
  - `visitorPhoneDigits(phone: string): string` — `internal/app/visits.go:25` `normalizePhone`의 거울. `'0'..'9'`만 남긴다. 유니코드 숫자·로케일 처리를 넣지 말 것(Go 쪽은 ASCII만 센다).
  - `visitorNameError(name: string): string` — `name === ""` 이면 `""`(미입력은 종전 처리에 맡김). `name.trim() === ""` 이면 `"방문자 이름을 입력하세요"`.
  - `visitorPhoneError(phone: string): string` — `phone === ""` 이면 `""`. `visitorPhoneDigits(phone).length < 7` 이면 `"휴대전화를 숫자 7자리 이상 입력하세요"`.
  - `visitorRowError(v: { name: string; phone: string }): string` — 이름 메시지 우선, 없으면 전화 메시지, 없으면 `""`. (`consent`는 이미 체크박스라 종전 조건으로 충분 — 넣지 말 것.)
- **`web/src/pages/VisitFormPage.tsx`** — 배선만. 로직을 이 파일 안에 복제하지 말 것.
  - 114행 방문자 카드의 `map` 안에서 `const nameMessage = visitorNameError(visitor.name)`, `const phoneMessage = visitorPhoneError(visitor.phone)`를 구해 이름/휴대전화 `TextField`에 `error={nameMessage !== ""}` / `helperText={nameMessage || undefined}`(전화도 같은 식)를 붙인다. 전화 칸의 기존 `placeholder="010-0000-0000"`는 남길 것.
  - 117행 제출 버튼 `disabled`에 `|| visitors.some((x) => visitorRowError(x) !== "")`를 더한다. **기존 `visitors.some((x) => !x.name || !x.phone || !x.consent)` 항은 지우지 말 것**(미입력 차단은 그 항이 담당한다 — 수용 기준 3).
  - `submit()` (79행) 안, 기존 `companiesSatisfied` / `scheduleMessage` 가드 **바로 뒤**에 줄 번호를 붙인 가드를 넣는다:
    ```ts
    const badVisitor = visitors.findIndex((v) => visitorRowError(v) !== "");
    if (badVisitor >= 0) { setError(`방문자 ${badVisitor + 1}: ${visitorRowError(visitors[badVisitor])}`); return; }
    ```
- **`web/src/visitor.test.ts` (신규)** — **반드시 `.ts`**. `web/vite.config.ts`의 `test.include`가 `src/**/*.test.ts` 라서 `.tsx` 는 수집되지 않는다(현재 통과 23개: `silentSso.test.ts` 8 + `schedule.test.ts` 15). `web/src/schedule.test.ts`의 서술 방식을 따를 것.
- (선택) `docs/USER_GUIDE.md` 의 **방문 신청** 절에 한 문장. **가져오기 절은 건드리지 말 것**(미머지 `origin/auto/2026-09-21-0654`가 그 절을 이미 고쳐 두었다). PDF 재생성은 불필요.

## 검증 명령

```bash
cd web && npm ci            # 이 워크트리에 node_modules 가 없다 — 먼저 실행 필요
npm run lint                # tsc -b
npm test                    # vitest run — 고치기 전 23개, 새 테스트 추가 뒤 그 이상
npm run build
cd .. && go build ./... && go vet ./...   # Go 는 안 건드리지만 싼 확인
git diff --check
```

- **TDD 순서로**: `visitor.test.ts` 를 먼저 쓰고 `npm test` 가 **올바른 이유로** 실패하는 것을 보고 나서 `visitor.ts` 를 채울 것.
- 화면 확인이 필요하면 직전 회차가 쓴 방식이 이 환경에서 실제로 돌았다: `npm run build` 산출물(`web/dist`)을 정적 서빙하고 `/api/v1/auth/config`·`/api/v1/auth/me`·`/api/v1/reference-data`·`/api/v1/visits` 만 스텁으로 답하는 임시 서버를 `/tmp` 에 띄워 Chromium 으로 본다(Go·PostgreSQL 불필요). 스텁을 저장소 안에 커밋하지는 말 것.
- **Go 통합 테스트는 이번 과제에 불필요**하다(서버 코드를 안 건드린다). 돌리더라도 `VISITFLOW_TEST_DSN` 없이 나온 PASS 는 DB 통합 증거가 아니다.

## 위험과 피할 것

- **`internal/app/visits.go` 를 건드리지 말 것.** 이 규칙은 REST·현장(lobby)·MCP 세 경로가 공유한다(`visits.go:390`, `visits.go:1289`, `mcp.go:166` 모두 `createVisitRecord` 호출 — 확인함). 상태코드·`invalid_visitor` 코드·문구를 화면 편의로 바꾸면 서버 계약이 깨진다. **화면은 서버를 거울처럼 따라가기만 한다.**
- **`normalizePhone` 을 건드리지 말 것** — 전화 해시(`watchlist_entries.phone_hash`)의 입력이다.
- **표시·버튼·`submit()` 가드가 같은 값 하나를 읽게 할 것.** 한쪽만 넓히면(예: 표시만 달고 `disabled` 는 그대로) 정책이 어긋난다. 이 저장소가 반복해서 걸린 자리다.
- **빈 칸을 빨갛게 칠하지 말 것**(수용 기준 3). 이 과제가 반려될 가장 큰 이유다 — 첫 화면에서 아무 입력도 안 한 방문자 카드가 오류 상태로 보이면 회귀다.
- **`web/src/pages/TemplatesPage.tsx:387` 의 자주 방문자 휴대전화 칸은 이번 범위가 아니다.** 그쪽은 `internal/app/templates.go:158` 의 `invalid_frequent_visitor` 라는 **다른 계약**이고, 규칙이 같은지 이번 정찰에서 확인하지 않았다(미확인). 같이 고치지 말고 보류 아이디어로 남겼다.
- **보호 경로 회피**: `auth.go`·`keys.go`·`internal/database/migrations/`·`settings.go`·`server.go` 근처에 갈 일이 전혀 없는 과제다. 갔다면 범위가 샌 것이다.
- **미머지 브랜치 충돌 회피**: `origin/auto/2026-09-16-1212`(메일)·`origin/auto/2026-09-18-0533`(MCP OAuth·SettingsPage)·`origin/auto/2026-09-21-0654`(import.go·가이드 가져오기 절)와 파일이 겹치지 않는다. 그대로 유지할 것.
- **변이 확인 뒤 되돌릴 때 `git checkout -- <파일>` 을 쓰지 말 것.** 이 저장소에서 미커밋 테스트를 이렇게 두 번 잃었다. `sed` 나 임시 복사본을 쓸 것.

## 작업량 근거 (S, 2.5~4시간 / 10회 중 8회)

분해: 순수 함수 파일 0.5h · 단위 테스트(경계 8~12케이스) 1h · 화면 배선 3곳 0.5h · 검증(`npm ci` 포함, 첫 설치가 느릴 수 있음) 0.5h · 변이 확인과 가이드 한 문장 0.5h. 제외한 것: Go 통합 테스트, Playwright E2E, PDF 재생성, TemplatesPage. 가장 크게 흔들릴 가정은 **`npm ci` 가 이 환경에서 한 번에 되는가**(node_modules 부재는 확인했으나 설치는 이번 정찰에서 실행하지 않음 — 미확인)이고, 막히면 여기서 +1h.

## 차선 후보

**방문자 추가 버튼을 100명에서 차단 (가치 2 / 위험 1 / S)** — `VisitFormPage.tsx:110` 의 "방문자 추가"는 무제한 `append` 인데 `visits.go:438` 은 101명부터 `required_fields` 로 거절한다(확인함). 100명에서 버튼을 `disabled` 로 두고 `최대 100명` 안내를 현재 인원과 함께 보이면 된다. 가져오기 경로는 이미 `import.go:244` 에서 101번째 행에 명시적 오류를 내므로 **손대지 말 것**. 1순위와 같은 파일·같은 모양이라 1순위가 성립하지 않을 때만 고를 것(둘을 한 회차에 섞지 말 것).
