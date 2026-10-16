- 과제: `relative()` 의 "오늘/어제/N일 전" 을 경과 시간(ms/86400000)이 아니라 **달력 날짜** 기준으로 계산하도록 고치기 (가치 4 / 위험 1 / 작업량 S)

- 왜: `web/src/api.ts:32-38` 의 `relative()` 는 `Math.floor((Date.now() - new Date(value).getTime()) / 86400000)` 로 날짜 차이를 구한다. 이 값은 **경과 시간**이지 달력 일수가 아니므로, 어제 23:59 에 일어난 일은 하루 전인데도 경과가 24시간 미만이라 **언제 보더라도 "오늘"** 로 표시된다(같은 이유로 그저께 일을 "어제" 로 표시하는 구간도 생긴다). 한국어 "오늘/어제" 는 달력 날짜를 뜻하므로 이것은 사용자에게 **틀린 사실**이고, 이 함수는 대시보드 최근 활동·고객 `updatedAt`·VOC 접수 시각·Intelligence 탐지 시각·로그인 세션 "최근"·Personal Key 마지막 사용·알림 시각 등 **14개 호출 자리**(아래 목록)에서 쓰인다. 특히 Personal Key 의 "마지막 사용" 과 로그인 세션 "최근" 은 운영자가 키를 폐기할지 세션을 끊을지 판단하는 근거라, 하루 전 사용을 "오늘" 로 보여 주면 판단이 틀어진다.

- 수용 기준:
  1) 로컬 시간대 기준으로 **어제 23:59:59** 에 해당하는 타임스탬프를 넣으면 `relative()` 가 `'어제'` 를 돌려준다(현재는 하루 중 어느 시각에 실행해도 `'오늘'`). 오늘 00:00:00.x 는 계속 `'오늘'`.
  2) 경계가 달력 날짜로 움직여도 기존 계약은 그대로다: 값 없음 → `'기록 없음'`, 같은 날 → `'오늘'`, 미래 시각(시계 어긋남) → `'오늘'`, 2~29일 전 → `` `${days}일 전` ``, 30일 이상 → `date(value)` 위임. `date`/`dateTime`/`money` 등 다른 포매터의 동작과 서명은 **바꾸지 않는다**.
  3) `relative()` 에 파싱되지 않는 문자열(예: `'not-a-date'`)을 넣어도 예외를 던지지 않고 `'기록 없음'` 을 돌려준다 — 현재는 `days` 가 `NaN` 이라 모든 분기를 빠져나가 `date(value)` 로 떨어지고, `Intl.DateTimeFormat.prototype.format` 은 Invalid Date 에 `RangeError` 를 던진다(ECMA-402 PartitionDateTimePattern: 유한수가 아니면 throw). 이 앱에는 ErrorBoundary 가 없으므로(`grep -rn "ErrorBoundary|componentDidCatch" web/src` → 매치 0) 렌더 중 throw 는 화면 전체를 날린다.
  4) 테스트가 증명해야 하는 것: **수정 전에는 반드시 빨갛고 수정 후에는 녹색**인 테스트로 기준 1 을 잡을 것. 고정 문자열 리터럴을 쓰지 말고 **실행 시각의 로컬 자정에서 역산**해 만들 것 —
     `const midnight = new Date(); midnight.setHours(0,0,0,0); const yesterdayLate = new Date(midnight.getTime() - 1000).toISOString()` 는 하루 중 어느 시각에 돌려도 "어제" 이고, 현재 구현은 항상 `'오늘'` 을 돌려주므로 red 가 **결정적으로** 재현된다. 기준 2·3 의 각 분기도 같은 방식(자정 기준 역산)으로 덮을 것.

- 건드릴 파일 (프로덕션 1 + 테스트 1, 총 2개):
  - `web/src/api.ts:32-38` — `relative` 하나만. 달력 날짜 차이로 바꾼다. 권장 형태:
    ```ts
    const startOfDay = (d: Date) => { const x = new Date(d); x.setHours(0, 0, 0, 0); return x }
    export const relative = (value?: string) => {
      if (!value) return '기록 없음'
      const at = new Date(value)
      if (Number.isNaN(at.getTime())) return '기록 없음'
      const days = Math.round((startOfDay(new Date()).getTime() - startOfDay(at).getTime()) / 86400000)
      if (days <= 0) return '오늘'
      if (days === 1) return '어제'
      if (days < 30) return `${days}일 전`
      return date(value)
    }
    ```
    `Math.floor` 가 아니라 **`Math.round`** 를 쓸 것 — 자정과 자정 사이는 DST 전환일에 23시간/25시간이 되므로(서버는 Asia/Seoul 이어도 브라우저 시간대는 사용자 것이다) `floor` 는 25시간짜리 날에 하루를 잃는다. 이 이유를 한 줄 주석으로 남길 것(파일의 기존 주석 밀도에 맞춰 짧게).
  - `web/test/dateTime.test.ts` — 기존 파일에 `relative` 테스트를 **추가**한다(새 파일을 만들지 않는 이유: 이 파일이 이미 `process.env.TZ = 'Asia/Seoul'` 을 import 앞에서 고정하고 `../src/api.ts` 에서 포매터를 가져온다. 기존 3개 테스트와 머리말 주석은 **한 글자도 바꾸지 말 것**). import 줄에 `relative` 를 더하고 기준 1~3 테스트를 덧붙인다.

- 검증 명령 (이 저장소에서 실제로 도는 것. **이번 정찰 세션에서는 node 실행이 허용되지 않아 돌려 보지 못했다 — 구현자가 반드시 직접 돌릴 것**):
  - 이 워크트리에는 `web/node_modules` 가 **없다**(`web/node_modules/.package-lock.json` 미존재 확인). 먼저 `npm --prefix web ci` — 네트워크를 탄다.
  - `npm --prefix web test` — `web/package.json:test` 는 `"$npm_node_execpath" --test 'test/**/*.test.ts'` 다. 2026-10-09 회차에서 38/38 통과 기록이 있다. 수정 전 red → 수정 후 green 을 **둘 다 로그로 남길 것**.
  - `npm --prefix web run typecheck` (= `tsc -b --pretty false`), `npm --prefix web run build`.
  - 여유가 있으면 `make test` 전체(`Makefile:13-16`: `go test ./...` → `go vet ./...` → `cd web && npm ci && npm audit --audit-level=high && npm run typecheck && npm test && npm run build`). Go 쪽은 이번 변경과 무관하므로 필수는 아니다.

- 위험과 피할 것:
  - **`date`/`dateTime` 은 건드리지 말 것.** 이 둘도 잘못된 날짜에 `RangeError` 를 던지지만(`api.ts:30-31`), 손대면 `web/test/dateTime.test.ts` 의 기존 3개 계약과 45곳 넘는 호출 자리가 함께 흔들린다. 그쪽은 차선 후보로 남겨 둔다.
  - **`internal/voice/service.go:272` 의 `OpenDays`** (`int(end.Sub(x.OccurredAt).Hours() / 24)`) 는 서버가 같은 "경과 시간" 셈법을 쓰는 자리다. `web/src/pages/VoicePages.tsx:178` 은 `relative(v.occurredAt) 접수 · {openDays}일 경과` 로 둘을 나란히 보여 주므로, 이번 수정 뒤 두 숫자가 어긋나 보일 수 있다. **그래도 서버는 건드리지 말 것** — 서로 다른 계약(한쪽은 SLA 경과 시간, 한쪽은 사람이 읽는 날짜 라벨)이고, 운영자 지침("같은 종류의 파서가 여러 경로에 있으면 통합하지 말고 좁히는 방향으로만")이 정확히 이 모양이다. 서버 변경은 별도 회차.
  - **고정 날짜 리터럴로 red 를 만들지 말 것.** `'2026-09-20T07:05:09Z'` 같은 과거 고정 값은 시간이 지나면 30일 분기로 넘어가 테스트가 썩는다. 반드시 실행 시각의 자정에서 역산할 것.
  - 보호 경로는 전부 피한다: `internal/auth`·`internal/oidc`·`migrations/`·`.github/workflows/` 는 **0줄**이어야 한다.
  - `web/package.json`·`web/package-lock.json` 은 **건드리지 말 것** — 2026-10-06 회차에서 로컬 npm 10.9.8 이 lockfile 을 재생성하면 rollup 선택적 바이너리의 `libc` 메타데이터 42줄을 지우는 것이 확인됐다.
  - 빌드 뒤 `internal/webui/dist/README` 와 `web/public/README` 의 embed 앵커가 유지되는지(`cmp` 로 동일) 확인하고, 빌드 산출물을 커밋에 넣지 말 것. 최종 `git status --short` 는 의도한 2파일만이어야 한다.
  - 섭동으로 red 를 되돌릴 때 `git checkout -- <파일>` 을 쓰면 **커밋되지 않은 본 변경까지 날아간다**(2026-10-06 회차에서 실제로 겪음). 자기 작업이 든 파일은 역-편집으로 되돌릴 것.

- 차선 후보: **`date()`·`dateTime()` 의 잘못된 날짜 입력에 `RangeError` 대신 `'—'` 폴백** (가치 3 / 위험 1 / S) — 위 1순위가 이미 머지됐거나 성립하지 않을 때 고를 것. `api.ts:30-31` 두 줄에 `Number.isNaN(new Date(value).getTime())` 가드를 넣고 `web/test/dateTime.test.ts` 에 테스트를 더한다. 같은 2파일이지만 1순위와 **같은 회차에 섞지 말 것**(두 변경의 red 판독이 서로를 가린다). 그다음 후보는 `Makefile:test` 에 `./scripts/previous-release-tag-test.sh` 연결(가치 3 / 위험 1 / S)인데, 2026-10-08 회차가 같은 과제로 no-change 로 끝난 전력이 있으니 왜 그때 안 됐는지부터 확인할 것.
