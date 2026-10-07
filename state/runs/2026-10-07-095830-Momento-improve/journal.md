# 회차 노트 2026-10-07-095830-Momento-improve — Momento
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 09:58] base pinned — main@9d82c28
- [러너 09:58] autonomy release — 

## 정찰 노트
- 배정된 실패(PR #27 / ci.yml:55 `npm audit`)의 직접 원인은 **이미 고쳐져 있다** — 97e3063 이 `web/package-lock.json:3723-3732` 를 source-map-js 1.2.2 로 정합하게 올렸고 sdk 에는 그 항목이 없다. 그래서 '다시 고치기' 대신 **같은 실패의 남은 절반**을 과제로 냈다: `web/package.json:41-43` 이 이미 `overrides`(js-yaml) 관례를 갖고 있는데 source-map-js 는 거기 없고, 잠금 루트 `packages[""]` 에는 `overrides` 키가 **0건**이라 하한이 잠금 재생성을 못 견딘다(3459행이 요구 범위를 아직 `^1.2.1` 로 적는다).
- 차선(보존 정책 helperText)을 제친 이유: 배정이 '수정 과제' 이고, 느슨하게 하지 않는 방향의 재발 방지가 지금 더 값이 크다. `--audit-level`·`--omit=dev` 는 금지 사항이라 처음부터 제외했다.
- **확신 없는 곳**: 샌드박스가 네트워크를 막아 `gh run view` 도 `npm ci`/`npm audit` 도 **한 번도 돌리지 못했다**. "실패 단계가 ci.yml:55" 는 shepherd 의 fix-summary 와 커밋 메시지에 근거한 것이고 CI 로그로 직접 보지 않았다. 과제서 머리에 그렇게 적었다.
- **구현자가 조심할 것**: 시작 전에 `cd web && npm ci && npm audit` 로 지금 main 이 녹색인지 먼저 확인하라(다른 권고로 빨간불이면 그것이 과제다). 그리고 `npm install --package-lock-only` 가 다른 전이 의존성까지 끌어올리면 그건 사실상 의존성 업그레이드다 — `git diff` 의 `"version"` 줄을 세어 보고 번지면 과제서의 되돌림 경로를 쓸 것. 워크플로 파일은 한 줄도 건드리지 말 것.
- 프로필은 2026-10-01 판(v0.34.52 기준)을 그대로 둔다 — 이 세션에서 확인한 범위(테스트 명령, 순수 모듈 관례, `.ts` 확장자 함정, prettier 금지, MOMENTO_TEST_POSTGRES_DSN skip)는 모두 아직 맞았다. 바뀐 것은 테스트 수 223→234 와 버전 v0.34.59, 그리고 **web/package.json 에 `overrides` 블록이 생긴 것**뿐이다.
- [러너 10:06] scout done — `web` 의 보안 하한을 매니페스트와 잠금 파일이 **함께** 담게 맞춘다 — `overrides` 에 `source-map-js` 를 더하고

## 구현 노트
- 과제서 1순위(web overrides)는 **실행으로 반증해 기각**하고 차선(보존 정책 범위 helperText)을 했다. 이 환경엔 네트워크가 있어 과제서가 못 돌린 것을 돌렸다: main 의 `npm audit` 이 이미 녹색이고, 잠금을 치워 매니페스트만으로 재생성해도 overrides 없이 source-map-js 1.2.2 가 나오며(= 전제가 거짓), npm 10.9.8 은 이 잠금 루트에 overrides 를 아예 쓰지 않는다(cmp 바이트 동일 = 수용 기준 2 불가능). 근거는 ledger-entry.md 에 수치로 적었다.
- 바꾼 것: 순수 모듈 `web/src/pages/retentionRule.ts` 가 다섯 칸의 범위·라벨·설명의 단일 출처가 되고, helperText·저장 실패 Alert·라벨이 같은 상수에서 나온다. 안내 다섯 문장은 글자 그대로 유지(기존 adminErrors 테스트 33건이 증명). 더해 `sdk/package.json` 의 `test` 가 맨 `node` 라 PATH 가림에 노출된 것을 v0.34.53 과 같은 방식으로 고쳤다(가드 포함).
- **확신 없는 곳**: (1) `RETENTION_SAVE_FAILED`·`UNKNOWN_SITE` 의 화면 도달성은 이번에도 확인하지 않았다(Postgres 없음) — 문구는 앞 회차 그대로 중립. (2) `retentionLimit` 이 모르는 칸 이름에 **던지는** 선택을 했다. 지금 호출자는 다섯 리터럴뿐이고 TS 가 묶지만, 배선 실수가 런타임 예외가 되는 쪽을 고른 것이니 비평가가 먼저 볼 자리로 적어 둔다. (3) 하네스의 mock 서버는 `/api/` 나머지에 `{}` 를 돌려주므로 내가 안 본 쿼리가 조용히 빈 값을 받았을 수 있다.
- **일부러 하지 않은 것**: 입력 차단과 「저장」 버튼 닫기 — 서버가 정본이고 화면은 거울이며, 앞 회차 과제서가 입력 차단을 금지했다. `advanced_analytics.go`·`validateRetention` 의 범위, 나머지 Alert 다섯 곳, `.github/workflows`, `web/package-lock.json`, sdk 의 `engines` 는 손대지 않았다. 「Realtime (시간)」·「Aggregation (개월)」 의 기존 설명은 통째로 바꾸지 않고 범위만 덧붙였다.
- **다음 역할이 조심할 것**: 새 `sdk/test/testCommand.test.mjs` 2건은 PATH 에 가짜 `node` 를 끼워 자식 프로세스를 띄우므로 **네트워크는 필요 없지만 `sh` 와 tmpdir 쓰기 권한이 필요**하다. 브라우저 확인은 커밋에 없다(하네스는 /tmp/retention-harness 에만 있고 dist 는 지웠다) — 다시 보려면 `npm run build` 후 하네스를 dist 경로와 함께 돌려야 한다. `/admin` 에 하네스를 쓰면 `useUnsavedWarning` 의 beforeunload 가 같은 탭 재이동을 막으니 사례마다 새 탭을 쓸 것. 그리고 `type="number"` 칸은 triple-click 선택이 미덥지 않고, 비울 수 없는 네 칸은 지우면 값이 "0" 으로 되돌아온다.
- [러너 10:28] brief fallback — 차선 — 1순위의 근거를 **실행으로 반증**했다: 네트워크가 있어 과제서가 못 돌린 것을 돌려 보니 main 의 `npm audit` 이 이�
- [러너 10:29] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인함: 서버 validateRetention(advanced_analytics.go:81-96)의 다섯 범위가 retentionRule.ts:43-90 과 글자 그대로 일치, 안내 다섯 문장이 삭제 전과 **동일**, 그리고 이 세션에서 직접 돌린 sdk 29/29 · web 248/248 · lint 0 · `tsc -b` 0 · `git status` 비어 있음. sdk 가드는 공허하지 않다 — 1번 테스트가 대조군이라 package.json 을 되돌리면 2번이 반드시 깨진다.
- 못 봄: Postgres 없음 → Go 통합 테스트 skip, `RETENTION_SAVE_FAILED`·`UNKNOWN_SITE` 화면 도달성 미확인(구현자와 같은 공백). 브라우저 하네스는 커밋에 없어 원장의 36/36 을 재현하지 않았다.
- 승인이어도 남는 것: (1) `retentionRangeText`(retentionRule.ts:112)는 프로덕션 호출자가 없는 **죽은 export** — cidrRule·passwordRule 선례와 어긋나니 다음 회차가 지우거나 쓸 자리를 정하라. (2) `retentionFieldProps(column: string, …)` 가 `RetentionColumn` 이 아니라 넓은 `string` 을 받아 **칸 이름 오타를 tsc 가 못 잡는다** — 오타 시 패널 전체가 렌더 예외로 날아가므로 인자 타입을 좁히는 것이 구조적 수리다(오늘 동작 결함은 없음). (3) sdk 에 `engines` 가 없어 Node 20 npm 에서는 같은 ERR_UNKNOWN_FILE_EXTENSION 이 그대로 난다.
- 릴리즈 노트: 입력 차단·저장 버튼 닫기는 **하지 않았다**. 원장의 「이제 화면이 저장 전에 막는다」 와 모듈의 `blocking`·「막는다」 표현은 과장이고, 실제는 helperText 교체 + 오류 색뿐이다 — 커밋 8a84b59 의 표현을 쓸 것.
- 법무·보안 차단 없음(공격 경로 없음, 새 개인정보 수집 없음). 다만 서버가 허용하는 Aggregation 상한 1200개월(100년)과 null(무기한)이 이제 화면에 숫자로 드러난다 — 사람 식별자를 담은 표의 보존 외곽 한계는 별도 회차에서 법무 기준과 맞출 자리다.
- [러너 10:34] review approved — 리뷰 승인 (risk=low)
- [러너 10:35] pr created — https://github.com/hkjang/Momento/pull/28
- [러너 10:41] ci passed — 검사 1개 모두 success
- [러너 10:41] merge done — deea337
- [러너 10:56] release published — v0.34.60
- [러너 10:58] assets verified — v0.34.60 자산 1개 (이전 v0.34.59: 2)
