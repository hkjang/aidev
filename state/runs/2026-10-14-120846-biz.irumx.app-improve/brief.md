- 과제: PR #5 의 CI 실패 원인 확정 후 v.day 계약 시험 재착지 (가치 4 / 위험 2 / 작업량 S)

- 왜: PR #5(`auto/2026-10-14-1028`, head `116f32d4f2a7`)의 변경은 `packages/core/src/__tests__/domain.test.ts` +57행**뿐**(프로덕션 0파일)인데 CI `check` 잡이 실패해 회차가 verify-failed 로 끝났다. 그런데 그 잡은 `started_at 2026-10-14T01:40:53Z` → `completed_at 01:40:56Z` = **3초**만에 `conclusion: failure` 로 끝났고 `output.title/summary/text` 는 전부 null, `annotations_count: 2` 다 — `actions/checkout@v4` 한 걸음도 3초 안에 끝나지 않으므로 저장소 쪽 단계(`npm ci`·`npm run check`·`npm run build`·`scripts/*.mjs`)는 **하나도 실행되지 않았다**. 같은 시각 다른 저장소 law.irumx.app 의 `check` 도 `01:40:07Z → 01:40:10Z` 3초·failure·annotations 2 로 같은 모양이고(yeopjari 도 같은 주기에 ci-failure 기록이 있다), `.github/workflows/check.yml` 은 `13d52c5`(v0.1.1) 이후 한 줄도 바뀌지 않았으며 그 뒤 PR #2·#3·#4 가 같은 워크플로로 통과해 머지됐다. 즉 **지금 아는 증거는 전부 "저장소 밖 원인" 을 가리키고, 시험 코드가 깨졌다는 증거는 없다**(로컬 전체 게이트는 그 브랜치 내용으로 164개 통과 + build 성공 — `.../2026-10-14-102831-biz.irumx.app-improve/verify.txt`). 이 회차가 할 일은 ① 남아 있는 2개 annotation 을 실제로 읽어 원인을 확정하고 ② 이미 비평 승인을 받은 그 57행을 이번 브랜치에 다시 올려 로컬에서 재현 검증까지 끝내는 것이다.

- 수용 기준:
  1) **원인 확정 증거가 회차 노트에 남는다.** 실패 잡의 annotation 2건을 그대로 인용한다. 정찰 세션에서는 `gh` 와 외부 fetch 가 권한에 막혀 읽지 못했다 — 구현 세션에서 가능하면 다음을 돌려 출력을 붙여라:
     - `gh api repos/hkjang/biz.irumx.app/check-runs/115635020490/annotations`
     - `gh run view 38497025178 --log-failed` (또는 `gh run view 38497025178 --json jobs`)
     권한에 또 막히면 "막혔다" 고 적고 (3) 으로 간다. **추측을 사실로 쓰지 말 것.**
  2) **원인에 따라 갈라진다 — 아래 분기 중 하나만 수행한다.**
     - (a) annotation 이 **없어진/은퇴한 액션 버전**을 가리킨다면(예: `actions/upload-artifact`·`actions/checkout`·`actions/setup-node` 의 버전이 더 이상 실행되지 않는다는 메시지) → `.github/workflows/check.yml` 의 해당 `uses:` 줄만 지원되는 최신 메이저로 올린다. 이것은 게이트를 **느슨하게 하는 것이 아니라** 유지보수다. 단계 추가·삭제, `continue-on-error`, `timeout` 완화, 단계 생략, `if:` 추가는 **금지**.
     - (b) annotation 이 **서비스 컨테이너 `postgres:16-alpine` 의 이미지 받기/시작 실패**(pull rate limit, 네트워크)를 가리킨다면 → 워크플로를 고치지 말고 "외부 일시 장애" 로 기록만 하고 (3) 으로 간다. 이미지 태그·레지스트리 교체는 로컬에서 증명할 수 없으므로 이번 회차 범위 밖이다.
     - (c) annotation 이 **계정·권한·과금 차단**(Actions 사용 불가, 지출 한도)을 가리킨다면 → 저장소에서 고칠 것이 없다. 기록만 하고 (3) 으로 간다.
     - (d) annotation 이 **저장소 파일**(워크플로 문법 오류 등)을 가리킨다면 → 그 줄만 고친다.
  3) **PR #5 의 시험 57행이 이번 브랜치에 그대로 올라가고 로컬에서 통과한다.** `git show 116f32d4f2a7 -- packages/core/src/__tests__/domain.test.ts` 의 추가분(`describe('입력 검증: 달력 날짜 v.day', …)`, 통과 7 / 거절 18 / 문구 1)을 그대로 가져온다. 새로 쓰지 말고 승인된 내용을 재사용할 것 — 비평에서 이미 변이 4건으로 유효성이 확인됐다. 기대값을 임의로 바꾸지 말 것.
  4) **시험이 실제로 가드를 본다는 것을 이 세션에서 다시 보인다.** `packages/core/src/util/validate.ts` 의 `v.day`(:233 근처)에서 ① ISO 왕복 조건(`|| d.toISOString().slice(0, 10) !== value`) ② 연도 범위 줄, 두 곳을 각각 임시로 떼어 실패 개수를 기록하고 `git checkout -- packages/core/src/util/validate.ts` 로 되돌린다. 커밋에 **프로덕션 변경 0파일**(분기 (a)/(d) 를 타면 워크플로 1파일) 이 들어가는 것을 `git show --stat` 으로 확인한다.

- 건드릴 파일:
  - `packages/core/src/__tests__/domain.test.ts` — 파일 끝에 `116f32d4f2a7` 의 `v.day` describe 블록(+57행)을 그대로 추가. 기존 `describe('입력 검증: 금액·정수·소수', …)` 뒤에 붙는다.
  - `.github/workflows/check.yml` — **분기 (a) 또는 (d) 일 때만**, 해당 `uses:`/문법 줄만. 다른 분기면 열지 말 것.
  - `packages/core/src/util/validate.ts` — 변이 확인용 **임시** 수정만. 반드시 되돌릴 것.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `npm ci --no-audit --fund=false` (체크아웃에 node_modules 가 없다. ESLint 지원 종료 경고 1건은 정상)
  - `npm test -- packages/core/src/__tests__/domain.test.ts` → 85개 통과 기대 (직전 회차 실측)
  - `npm run check` (eslint + 세 workspace tsc + vitest 전체) → 164개 통과 기대
  - `npm run build`
  - `node scripts/ci-env.mjs` — CI 가 실제로 돌리는 다음 단계(workflow 46행). 여기까지가 DB·브라우저 없이 재현할 수 있는 CI 구간이다. CI 는 이번에 이 중 **어느 것도** 도달하지 못했으므로, 전부 통과하면 "저장소 쪽 게이트는 로컬에서 재현 통과" 라고 쓸 수 있다.
  - (선택) PostgreSQL 16 을 띄울 수 있으면 workflow 47~60행대로 `scripts/migrate.mjs` → `scripts/seed-local.mjs` → `scripts/e2e.mjs` 까지. 안 되면 "미실행" 이라고 적을 것 — 추정으로 통과했다고 쓰지 말 것.

- 위험과 피할 것:
  - **워크플로를 통과시키려 느슨하게 만드는 것 금지**(운영자 지시). `continue-on-error`, 단계 삭제·주석화, `timeout-minutes` 늘리기, `if: always()` 로 실패 감추기, e2e/a11y 단계 빼기 — 전부 금지. `.github/workflows` 는 보호 경로이니 분기 (a)/(d) 가 아니면 **열지 말 것**.
  - `validate.ts` 의 동작을 "개선" 하지 말 것 — 특히 `' 2026-01-15'` 의 앞 공백을 trim 해 통과시키는 것, 연도 범위를 넓히는 것. 운영자 지시는 **좁히는 방향만**. 이 회차는 계약 고정이지 수정이 아니다.
  - 시험 기대값을 실제 동작에 맞추는 조정은 직전 회차에서 이미 실행 확인됐다: `'2026-13-01'`·`'2026-00-10'`·`'2026-01-00'`·`'2026-01-32'` 는 `new Date` 가 NaN, `'2100-02-29'` → `2100-03-01`, `'2026-02-30'` → `2026-03-02` 로 ISO 왕복 조건에서 걸린다. 형식 정규식 층만 떼면 거절 **여부** 는 안 바뀌고 문구 시험 1건만 깨진다(비평이 재현). 그래서 머리 주석의 "한 겹이라도 느슨해지면 DB 로 내려간다" 는 정규식 층에는 과장 — 가져올 때 그 문장만 실제에 맞게 다듬어도 좋다(선택).
  - 다른 검증기(`v.instant`·`v.string`/`v.text`·`v.bool`)나 `v.day` 호출 라우트(`routes/finance.ts:734-735` 의 `'1900-01-01'`·`'9999-12-31'` 기본값)는 이번 회차에 손대지 말 것 — 다음 회차 거리로 남긴다.
  - CI 가 또 같은 3초 실패로 끝날 수 있다. 그때는 **저장소를 더 고치려 들지 말고** annotation 을 그대로 기록해 원장에 남겨라. 저장소 밖 원인을 코드 변경으로 덮으려는 시도가 더 큰 손해다.

- 차선 후보: KST 일 경계 `kstDateKey`·`minuteStart` 의 쿼터 창 계약 시험 (3/1/S) — `packages/core/src/util/time.ts` 는 시험 0건이고 `quota.ts` 가 유일한 소비처(일일 리셋 키 `kstDateKey`, 분당 창 `minuteStart`). UTC 15:00 = KST 자정 경계(`2026-03-14T14:59:59.999Z` → `'2026-03-14'`, `15:00:00Z` → `'2026-03-15'`)와 `minuteStart` 의 초·밀리초 절삭만 못 박으면 끝. 순수 함수라 node environment 로 충분하고 프로덕션 0파일. 같은 파일의 `kstWindowOn`·`overlaps`·`isoDay`·`formatKst`·`backoffMs` 는 `quota.ts` 도 쓰지 않는 표면이라 표를 늘리지 말 것.
