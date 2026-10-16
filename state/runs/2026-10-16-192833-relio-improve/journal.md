# 회차 노트 2026-10-16-192833-relio-improve — relio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:28] base pinned — main@13852a0
- [러너 19:28] autonomy low-risk — 롤백 PR 

## 정찰 노트
- `relative()` 를 골랐다: 사용자에게 **틀린 사실**(어제 23:59 → "오늘")을 보여 주는 실제 결함이고, 프로덕션 1파일·테스트 1파일로 끝나며, 고정 리터럴 없이 자정 역산으로 **결정적 red** 가 난다. `date()`/`dateTime()` 의 RangeError 는 Go 서버의 RFC3339 로 도달 경로를 증명하지 못해 차선으로 내렸고, Makefile·release.yml 계열은 2026-10-08 no-change·2026-09-06 롤백 전력이라 제쳤다.
- 추측으로 적은 것: (a) `Intl.DateTimeFormat.format` 이 Invalid Date 에 RangeError 를 던진다는 것은 ECMA-402 근거이며 **이번에 실행해 확인하지 못했다**(node 실행 미승인). (b) 프런트 테스트 베이스라인(38/38)도 이번에 돌리지 못했고 2026-10-09 기록을 인용한 것이다. 구현자가 먼저 `npm --prefix web ci && npm --prefix web test` 로 베이스라인을 세울 것.
- 조심할 것: `Math.floor` 가 아니라 `Math.round` — DST 날의 자정 간격은 23/25시간이다. `internal/voice/service.go:272` 의 OpenDays 는 같은 셈법이지만 **서버는 건드리지 말 것**(VoicePages.tsx:178 에서 나란히 보이더라도 별개 계약).
- [러너 19:34] scout done — `relative()` 의 "오늘/어제/N일 전" 을 경과 시간(ms/86400000)이 아니라 **달력 날짜** 기준으로 계산하도록 고치�

## 구현 노트
- `web/src/api.ts` 의 `relative()` 만 고쳤다(+8/−1): 경과 ms 대신 로컬 자정 사이 **달력 일수**(`Math.round`)로 세고, `Number.isNaN(at.getTime())` 가드로 파싱 불가 값을 `'기록 없음'` 으로 접는다. 테스트는 기존 `web/test/dateTime.test.ts` 에 8건 추가(+40/−1, 삭제 1줄은 import 뿐).
- **red 를 눈으로 봤다**: 수정 전 exit 1 / 2 fail — `not ok 15 … expected '어제' actual '오늘'`, `not ok 21 … error: 'Invalid time value'`. 수정 후 45/45. 역-편집 섭동 2종으로 각 수정이 자기 테스트만 red 로 만드는 것도 확인했다(`git checkout` 은 쓰지 않았다).
- 정찰이 미확인으로 남긴 전제 둘을 실행해 **확정**했다: `Intl.DateTimeFormat.format` 은 Invalid Date 에 실제로 `RangeError: Invalid time value` 를 던지고, 프런트 베이스라인은 38/38 이었다.
- **확신 없는 곳**: DST 는 `TZ=America/New_York` 산술 프로브로만 확인했다 — spring-forward 자정 간격 23시간에서 `floor`=0, `round`=1. `relative()` 자체를 DST 전환 당일에 실행해 본 것은 아니다(`new Date()` 를 고정할 수단이 테스트에 없다). 과제서는 "floor 가 25시간짜리 날에 하루를 잃는다" 고 했으나 실측은 그 반대다(25시간 날은 `floor` 도 1). 결론(`Math.round`)은 같고 주석을 실측에 맞춰 적었다.
- **검증 못 한 것**: 브라우저 화면에서 14개 호출 자리를 눈으로 본 것은 아니다(순수 함수 단위 검증까지). Docker 오프라인·업그레이드 검증, `go test -race` 는 돌리지 않았다 — Go 0줄 변경이다.
- **일부러 하지 않은 것**: `date`/`dateTime` 의 같은 `RangeError`(호출 자리 45곳, 다음 회차 최우선으로 ideas.json 에 올렸다 — 같은 2파일이라 red 판독이 겹친다), `internal/voice/service.go:272` 의 `OpenDays`(SLA 경과 시간은 다른 계약), `web/package.json`·`package-lock.json`·워크플로·마이그레이션은 **0줄**.
- **다음 역할이 조심할 것**: 새 테스트는 DB·네트워크가 필요 없지만 `web/node_modules` 가 없으면 `npm --prefix web ci` 가 먼저다. 이 워크트리의 node 는 v22.23.1 / npm 10.9.8(README 는 Node 24+ 요구)인데 전체 게이트는 통과했다. `make test` exit 0 확인: Go 23패키지 ok, audit 0 vulnerabilities, 프런트 45/45, build 성공. 빌드 뒤 `cmp internal/webui/dist/README web/public/README` 동일, 커밋(0ccf619)에는 소스·테스트 2파일만 들어 있다. Vite 의 analytics.js module 경고와 500 kB 번들 경고는 기존 상시 경고다.
- [러너 19:39] brief accepted — 채택 — 과제서의 실측이 코드와 전부 일치했고(`api.ts:32-38` 의 `Math.floor` 경과식, `relative(` 호출 자리 **14개**, `dateTime.test.t
- [러너 19:39] verify passed — 검증 10개 통과 (auto)

## 비평 노트
- 판정 approve / risk low / blocking 없음. red 를 독립 재현했다 — main 의 api.ts 를 /tmp 로 복사해 HEAD 테스트를 돌려 구현 노트와 동일한 2건 실패(`expected '어제' actual '오늘'`, `error 'Invalid time value'`)를 확인하고, HEAD 에서 45/45 · typecheck · vite build · `go build ./...` · webui·api 패키지 테스트 · dist/README cmp 까지 통과시켰다.
- 구현자가 유일하게 "확신 없는 곳" 으로 남긴 DST 를 가장 먼저 쳤다: 8개 DST 존 × 400일 × 오프셋 6종 × 시각 5종 프로브에서 `Math.round` 실패 0, `Math.floor` 존당 485 실패(첫 실패 New_York 2026-03-09). 자정이 없거나 두 번 오는 Havana·Lord_Howe 도 round 가 흡수한다. 주석 서술이 실측과 맞고 과제서의 "25시간" 쪽이 틀렸다.
- 못 본 것: 브라우저 화면의 14개 호출 자리(코드로 전부 열어 과거 시각뿐임은 확인), Docker 오프라인·업그레이드, `go test -race`(Go 0줄).
- 남는 우려(릴리즈 노트용): `date()`/`dateTime()` 의 RangeError 는 그대로여서 Intelligence.tsx:64 처럼 같은 행에서 `date()` 를 함께 쓰는 자리는 여전히 화면이 날아갈 수 있다 — "일부 경로만 수정" 으로 적을 것. 30일 경계 의미가 경과시간→달력일수로 바뀌어 경계 항목 표시가 최대 하루 당겨진다.
- 다음 회차: `date`/`dateTime` 의 RangeError 가 이미 ideas.json 최우선에 있다 — 같은 2파일이니 red 판독이 겹치는 점만 주의.
- [러너 19:43] review approved — 리뷰 승인 (risk=low)
- [러너 19:43] pr created — https://github.com/hkjang/relio/pull/47
- [러너 19:47] ci passed — 검사 2개 모두 success
- [러너 19:47] merge done — 0ccf619
- [러너 19:47] release skipped — 자율화 단계 low-risk — 릴리즈는 사람이
