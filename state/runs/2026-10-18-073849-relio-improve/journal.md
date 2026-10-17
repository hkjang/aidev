# 회차 노트 2026-10-18-073849-relio-improve — relio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:38] base pinned — main@7290a94
- [러너 07:38] autonomy low-risk — 롤백 PR 

## 정찰 노트
- 선택: `date()`·`dateTime()` 의 파싱 불가 입력 가드. 이전 정찰이 "다음 회차 최우선" 으로 남긴 항목이고, 프로덕션 1파일·테스트 1파일로 끝나며 `relative()` 가 이미 같은 `Number.isNaN(at.getTime())` 관용을 들고 있어 모양이 정해져 있다. 제친 후보: `OpenDays`(SLA 계약 확인이 선행, 운영자 지침이 "통합 금지"), `release.yml` 프런트 테스트(보호 경로 + 이번 러너가 low-risk), `Makefile`/`previous-release-tag-test.sh`(2026-10-08 에 같은 과제로 no-change), 감사 기간 필터·`ClientIP`(M 이상 + 설정축·DB 필요).
- 추측으로 적은 것: `Intl.DateTimeFormat.format(Invalid Date)` 의 `RangeError` 는 2026-10-16 회차 프로브 기록을 근거로 적었고 **이번 세션에서 재실행하지 못했다**(`node`·`npm` 실행 미승인). 서버가 실제로 깨진 타임스탬프를 보내는 경로도 확인하지 못했다 — 이것은 재현된 장애 수정이 아니라 방어 가드이므로 수용 기준을 "throw 하지 않는다" 로 좁게 잡았다.
- 직접 열어 확인한 것: `web/src/api.ts:1-46`(포매터 전체·`relative` 의 `startOfDay`/`Math.round`), `web/test/dateTime.test.ts` 전체(`:10` TZ 선-고정, `:13` 이 `date`·`dateTime` 를 이미 import, `test(` 10개), `internal/voice/service.go:265-274`, `date(` 31줄·`dateTime(` 2줄 집계, `web/src` 에 ErrorBoundary 무출력.
- 구현자가 조심할 것: 정상 값 출력이 바이트 단위로 같아야 한다(`'—'` 는 기존 em dash 재사용). 섭동 되돌리기는 `git checkout` 대신 역-편집(2026-10-06 실제 피해). `web/package-lock.json` 은 0줄 — 로컬 npm 으로 재생성하면 rollup `libc` 42줄이 날아간다. Vite 의 `analytics.js`·500 kB 경고는 상시.
- 프로필은 2026-10-16 판이 지금 코드와 어긋나지 않아 새로 쓰지 않았다(`api.ts` 의 `relative` 수정 반영됨, 마이그레이션 017, 테스트 5파일 모두 일치).
- [러너 07:42] scout done — `date()`·`dateTime()` 가 파싱 불가 날짜에 `RangeError` 를 던지는 것을 `'—'` 폴백으로 접기 (가치 3 / 위험 1 / 작�

## 구현 노트
- 무엇을 왜: `web/src/api.ts` 의 `date`/`dateTime` 이 파싱 불가 값을 `Intl.DateTimeFormat.format` 에 Invalid Date 로 넘겨 `RangeError: Invalid time value` 를 던졌다(프로브로 재확정). `relative` 와 같은 `Number.isNaN(at.getTime())` 관용을 사설 헬퍼 `parsed()` 로 묶어 기존 em dash 로 접었다. 커밋 823599d — 프로덕션 1파일(+6/−2)·테스트 1파일(+24/−0, 추가만).
- 확신 없는 곳·검증 못 한 것: (1) **서버가 실제로 깨진 타임스탬프를 보내는 경로는 끝까지 확인하지 못했다** — 이것은 재현된 장애 수정이 아니라 방어 가드이고, 그래서 기준을 "throw 하지 않는다" 로만 잡았다. (2) 브라우저에서 실제 화면이 백지가 되는 것을 **눈으로 본 적은 없다**; ErrorBoundary 부재(`grep` 무출력)와 `format` 의 throw 라는 두 사실로부터의 추론이다. (3) 정상 출력 불변은 `sample` 한 값과 `Asia/Seoul` 한 TZ 에서만 확인했다(다른 로케일·TZ 조합은 안 돌렸다 — 다만 포맷 옵션과 입력 경로가 그대로이므로 바뀔 수 있는 자리가 없다).
- 일부러 하지 않은 것: **ErrorBoundary 추가**(범위 밖, ideas.json 에 M 으로 남겼다 — 이 가드만으로 렌더 중 throw 가 없어지지는 않는다) · `relative`/`money`/`currencyMoney`/`number`/`api` 0줄 · 호출부 TSX 31+2줄은 손대지 않았다(포매터가 값을 책임지므로 불필요하고, 밀집 스타일이라 diff 가 터진다) · `export` 증설 없음 · Go·migrations·워크플로·`package.json`·`package-lock.json` 0줄.
- 다음 역할이 조심할 것: 워크트리에 `web/node_modules` 가 없으므로 프런트 명령 전 `npm --prefix web ci` 필요(이번엔 1초, 네트워크를 탄다). `make test` 는 exit 0 이었고 `npm audit` 는 `found 0 vulnerabilities` 였다. Vite 의 `analytics.js` module 속성 경고와 500 kB 번들 경고는 **상시 경고**다 — 새 실패가 아니다. 섭동을 되돌릴 때 `git checkout -- web/src/api.ts` 를 쓰지 말 것(역-편집으로 했다). secrets DB 통합 3건은 `RELIO_TEST_POSTGRES_DSN` 없으면 skip 인데 non-verbose 로그는 그것을 보여 주지 않는다 — 이번에 따로 확인하지 않았다.
- [러너 07:46] brief accepted — 채택 — 과제서의 실측이 코드와 전부 일치했다: `api.ts:30-31` 의 두 한 줄 정의와 바로 아래 `relative` 의 `Number.isNaN(at.getTime(
- [러너 07:47] verify passed — 검증 10개 통과 (auto)

## 비평 노트
- 확인: 섭동(역-편집)으로 `date()`·`dateTime()` 를 하나씩 되돌려 새 테스트가 각각 `error: 'Invalid time value'` 로 red 가 되는 것을 봤고, 복원 후 `git diff HEAD` 는 빈 출력이다. 원장의 `- 실패 재현:` 줄과 증상이 일치한다. web 48/48·0 skip, `tsc -b` 무출력, vite build 성공(상시 500 kB 경고만), `go build ./...`·internal/api 통과, 빌드 후 `internal/webui/dist/README` ↔ `web/public/README` 는 `cmp` 동일.
- 못 본 것: Node 24(여기는 v22.23.1 로만 돌렸다), 실제 브라우저 화면, 서버가 깨진 타임스탬프를 보내는 경로 — 구현자와 같은 한계다. 이것은 재현된 장애 수정이 아니라 방어 가드라는 평가에 동의한다.
- 승인이어도 남는 우려 ①: `'—'` 는 '값 없음' 과 '값 깨짐' 을 구분 불가하게 만든다. `dateTime()` 호출부는 감사 로그 표·상세(`web/src/pages/AdminPages.tsx:621-622`) 둘뿐이고 CSV·설정 번들은 서버 href 라 감사 증거는 영향이 없다 — 수정 전에는 같은 입력이 감사 페이지 전체를 죽였으니 교환은 개선이다. 릴리즈 노트에 "깨진 타임스탬프도 빈 값과 같은 em dash" 로 적어 두면 운영자가 혼동하지 않는다.
- 승인이어도 남는 우려 ②: ErrorBoundary 는 여전히 없다(grep 무출력) — 다음 회차가 M 아이디어로 집어야 한다. 또 새 테스트 3개 중 happy-path 1개는 수정 전에도 통과한다(회귀 가드로는 타당, 변경을 고정하는 것은 나머지 2개). `AppPages.tsx:214·346` 의 `toLocale*` 는 Invalid Date 에서 던지지 않고 `'Invalid Date'` 를 돌려주므로(프로브 확인) 이 수정은 부분 수정이 아니다.
- 검토 부서 둘 다 차단 없음 — security: 엔드포인트·식별자·비밀값·의존성 0줄, 신뢰할 수 없는 입력이 닿는 곳은 `new Date()`·Intl 뿐으로 공격 경로가 없다. legal: 개인정보 신규 수집·전송 없음, lock 파일 0줄로 라이선스 쟁점 없음.
- [러너 07:51] review approved — 리뷰 승인 (risk=low)
- [러너 07:51] pr created — https://github.com/hkjang/relio/pull/48
- [러너 07:54] ci passed — 검사 2개 모두 success
- [러너 07:54] merge done — 823599d
- [러너 07:54] release skipped — 자율화 단계 low-risk — 릴리즈는 사람이
