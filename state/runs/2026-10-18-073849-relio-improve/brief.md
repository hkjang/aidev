- 과제: `date()`·`dateTime()` 가 파싱 불가 날짜에 `RangeError` 를 던지는 것을 `'—'` 폴백으로 접기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `web/src/api.ts:30-31` 의 `date`/`dateTime` 은 `value` 가 truthy 이면 `new Date(value)` 를 그대로 `Intl.DateTimeFormat.format` 에 넘긴다. 파싱 실패 시 `Invalid Date` 가 되어 `format` 이 `RangeError: Invalid time value` 를 던지고, `web/src` 에 ErrorBoundary 가 없어 렌더 중 throw 는 해당 화면 전체를 날린다. 2026-10-16 회차가 `relative()` 에 같은 가드를 넣었으나 `date`/`dateTime` 직접 호출 자리는 그대로 남았다 — 이번에 실제로 센 결과 `date(` 가 6파일 31줄(AdminPages 9 · AppPages 8 · VoicePages 7 · MePages 4 · Intelligence 2 · IntelligencePages 1), `dateTime(` 가 `AdminPages.tsx:621-622` 2줄이다. `grep -rl "ErrorBoundary|componentDidCatch" web/src` 는 **무출력**이므로 폴백이 없다는 전제도 확인했다.
- 수용 기준:
  1) `date('')`·`date(undefined)` 는 지금처럼 `'—'`, 정상 ISO 값은 지금과 **바이트 단위로 같은** 문자열을 돌려준다(`dateTime` 도 동일).
  2) `date('not-a-date')`·`dateTime('2026-13-45')` 같은 파싱 불가 값이 `throw` 하지 않고 `'—'` 를 돌려준다.
  3) 테스트가 "수정 전에는 `RangeError: Invalid time value` 로 실패하고 수정 후 `'—'` 가 된다" 를 증명한다 — 즉 먼저 red 를 눈으로 보고, 고친 뒤 가드를 역-편집으로 지워 같은 테스트만 다시 red 가 되는 것까지 확인한다.
  4) `relative()`·`money()`·`currencyMoney()`·`number()` 와 `api()` 는 **0줄** 변경.
- 건드릴 파일 (프로덕션 1 + 테스트 1):
  - `web/src/api.ts:30-31` — `date`/`dateTime`: `new Date(value)` 결과를 변수로 받아 `Number.isNaN(parsed.getTime())` 이면 `'—'` 를 돌려준다. 바로 아래 `relative()` 가 이미 쓰는 `Number.isNaN(at.getTime())` 관용을 그대로 따를 것. 두 함수가 같은 모양이 되므로 사설 헬퍼(예: `const parsed = (value: string) => { const at = new Date(value); return Number.isNaN(at.getTime()) ? null : at }`)로 묶어도 좋으나 `export` 는 늘리지 말 것.
  - `web/test/dateTime.test.ts` — 기존 파일에 테스트 **추가만**. 이번에 열어 확인한 사실: 파일은 `test(` 10개를 담고 있고, `:13` 이 **이미** `import { date, dateTime, relative } from '../src/api.ts'` 로 `date`·`dateTime` 를 가져오므로 **import 줄은 바꿀 필요가 없다**. `:10` 이 `process.env.TZ = 'Asia/Seoul'` 로 import 보다 앞에 있고, `:36` 의 `dateTime falls back to the same em dash as date for a missing value` 가 이미 기준 1 의 빈 값 쪽을 덮는다 — 새 테스트는 **파싱 불가 값** 쪽만 추가하면 된다. 맨 위의 `process.env.TZ` 선-고정 줄과 `../src/api.ts` import, 기존 `relative` 테스트들은 한 글자도 바꾸지 말 것(`git diff -U0` 의 `-` 행이 비어 있어야 한다). 정상 값 테스트는 하드코딩 리터럴 대신 `Intl.DateTimeFormat` 으로 기대값을 만들거나, 기존 파일의 관용을 따를 것.
- 검증 명령:
  - `npm --prefix web ci` (워크트리에 `web/node_modules` 가 없다 — 프런트 검증 전 필수, 네트워크를 탄다)
  - 수정 전: `npm --prefix web test` → 새 테스트가 `Invalid time value` 로 red 인 것을 확인
  - 수정 후: `npm --prefix web test` (전부 pass) · `npm --prefix web run typecheck` · `npm --prefix web run build`
  - 빌드 뒤 `cmp internal/webui/dist/README web/public/README` → IDENTICAL · `./scripts/check-static-assets.sh`
  - 전체: `make test` (Go 23패키지 → `go vet` → `npm ci` → `npm audit --audit-level=high` → typecheck → 프런트 → build). Go 는 0줄 변경이지만 `make test` 가 최종 게이트다.
  - `git diff --check` 무출력 · 최종 `git status --short` 는 의도한 2파일만(빌드 산출물 없음)
- 위험과 피할 것:
  - **정상 출력이 바뀌면 안 된다.** `'—'` 는 기존 빈 값 폴백과 같은 em dash 문자를 그대로 쓸 것(새 문자·새 문구 금지).
  - `relative()` 는 `days >= 30` 에서 `date(value)` 를 호출한다. `relative` 는 이미 NaN 을 걸러내므로 `date` 의 새 가드가 `relative` 경로로 도달하지 않지만, `relative` 테스트가 깨지지 않는지 확인할 것.
  - `migrations/`·`.github/workflows/`·`internal/auth`·`internal/server/public.go` 는 건드리지 말 것. Go 파일 0줄.
  - `web/package.json`·`web/package-lock.json` 은 0줄. 로컬 npm 10.9.8 로 lockfile 을 재생성하면 rollup 선택적 바이너리의 `libc` 메타데이터 42줄이 사라진다(2026-10-06 실측).
  - 섭동 테스트를 되돌릴 때 `git checkout -- <파일>` 을 쓰지 말 것 — 같은 파일의 커밋되지 않은 본 변경까지 날아간다(2026-10-06 실제 피해). **역-편집으로** 되돌릴 것.
  - Vite 빌드의 `analytics.js` module 속성 경고와 500 kB 초과 번들 경고는 **기존 상시 경고**다. 새 실패로 오해하지 말 것.
  - ErrorBoundary 추가·TSX 일괄 포매팅·`api()` 리팩터는 이번 범위 밖(페이지 TSX 는 한 줄이 매우 긴 밀집 스타일이라 포매터를 돌리면 diff 가 터진다).
- 작업량 산정과 여유(기초·범위·여유를 분리해 적는다 — 점추정 금지):
  - 분해: ① `npm --prefix web ci`(네트워크, 과거 회차 실측 1~3분) ② 새 테스트 2~3건 작성 후 red 확인 ③ `api.ts` 2줄(또는 사설 헬퍼 1개 + 2줄) 수정 ④ 프런트 green + typecheck + build ⑤ `make test` 전체(Go 23패키지 + npm ci + audit 재실행 — 이 단계가 가장 길다) ⑥ 커밋.
  - 방법 두 가지가 일치한다: **유사 추정** — 2026-10-16 회차가 **같은 파일·같은 2파일 구조·같은 가드 관용**으로 끝났고 S 로 맞았다. **파라메트릭** — 프로덕션 2줄 + 테스트 2~3건.
  - 범위: **S, 한 세션(45분) 안에 끝난다**고 본다(열 번 중 여덟). 상한을 밀어 올리는 요인은 코드가 아니라 `npm ci`·`npm audit`·`make test` 의 네트워크·대기 시간이다.
  - 여유(contingency, 이 과제 안의 known unknown): `npm audit` 가 새 advisory 로 빨개질 수 있다 — 2026-10-06 에 실제로 CI 를 깨뜨린 유형이다. 그 경우 **게이트를 느슨하게 하지 말고**(`--audit-level` 하향·`|| true` 금지) 이번 과제와 분리해 회차 노트에 적고 포매터 수정만 올릴 것.
  - 예비(management reserve, 이 과제 밖): `date`/`dateTime` 가드만으로는 렌더 중 throw 가 완전히 없어지지 않는다(ErrorBoundary 부재). 그것은 **이번 범위가 아니다** — ideas.json 의 별도 항목으로 남겼다. 범위를 늘리지 말 것.
- 미확인(구현자가 가장 먼저 확정할 것):
  - `Intl.DateTimeFormat.format(Invalid Date)` 가 `RangeError: Invalid time value` 를 던진다는 것은 **2026-10-16 회차가 프로브로 실제 확인**한 사실이다(그 회차의 red ②번 `error: 'Invalid time value'`). 이번 정찰 세션에서는 `node` 실행이 승인되지 않아 **재실행하지 못했다** — 기준 3 의 red 를 먼저 띄워 다시 확정할 것.
  - 서버가 실제로 파싱 불가 타임스탬프를 보내는 **구체적 경로는 확인하지 못했다**. 이것은 재현된 프로덕션 장애 수정이 아니라 **방어 가드**다(그래서 가치 3). 수용 기준을 "throw 하지 않는다" 로 좁게 잡은 이유이기도 하다.
  - 워크트리에 `web/node_modules` 가 없는 것으로 보인다(`web/node_modules/.package-lock.json` 글롭 무출력). 프런트 명령 전에 `npm --prefix web ci` 를 돌릴 것.
- 차선 후보: `internal/voice/service.go:271` 의 `OpenDays` 경과 시간 셈법(이번에 실제로 열어 확인했다 — `if days := int(end.Sub(x.OccurredAt).Hours() / 24); days > 0 { x.OpenDays = days }`, 바로 위에 `ResolutionOverdue` 판정이 있다) — 달력 일수가 아니라 경과 시간이다. 고르기 전에 `OpenDays` 가 SLA 판정에 쓰이는지부터 확인하고, 쓰인다면 사람이 읽는 라벨과 SLA 계약을 **통합하지 말고** 좁히는 방향으로만 고칠 것.
