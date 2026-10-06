# 회차 노트 2026-10-07-020743-yeopjari-improve — yeopjari
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 02:07] base pinned — main@cd18b14
- [러너 02:07] autonomy release — 

## 정찰 노트
- 어제 들어온 b576143(퍼징으로 찾은 500 일괄 수정)이 validate.ts 에 queryInt·isRealDay·isUuid 를 더했는데 validate.test.ts 의 describe 목록에 이 셋이 없다 — 호출처 17곳을 grep 으로 확인했고, 테스트 1파일·프로덕션 0파일로 끝나는 가장 깨끗한 공백이라 골랐다.
- 쿠키 decodeURIComponent·image·indexnow 후보는 제쳤다: 이 main(cd18b14)에 그 세 PR 이 머지되지 않아(테스트 파일 부재로 확인) 같은 파일을 다시 만들면 충돌한다.
- 추측으로 적은 것: 과제서 2-c 의 queryInt fallback 결함은 registry.ts:120~127 이 min 5·max 50 으로 제한하므로 실발생을 확인하지 못했다 — 구현자는 이것을 버그로 주장하지 말 것.
- client-errors.ts(신규 공개 POST)와 normalisePath·redactSecrets 의 외부 입력 노출은 아이디어로만 남겼다. DB 없이 라우트를 지나는 증명이 안 되고, 상한(IP 20/일·전역 500/일)이 이미 있다.
- 조심할 것: node_modules 가 없으니 npm ci 먼저. 라우트 17곳은 열지 말 것(파일 수 폭증·limit 의미 회귀).
- [러너 02:12] scout done — 어제 추가된 입력 가드 3개(queryInt·isRealDay·isUuid)를 회귀 테스트로 묶는다 (가치 4 / 위험 1 / 작업량 S)

## 구현 노트
- `validate.test.ts` 한 파일에 `queryInt`·`isRealDay`·`isUuid` describe 3개(12건)를 추가했다. b576143 의 가드 3개에 테스트가 0건이어서, 가드가 느슨해지면 `limit=1.5`·`2026-02-31`·UUID 흉내가 다시 SQL 로 내려간다. 프로덕션 코드는 건드리지 않았다(변경 파일 1개).
- 확신 없는 곳: **프로덕션 결함의 red→green 은 없다.** 기존 가드가 이미 올바르다. 대신 가드를 b576143 이전의 느슨한 형태로 되돌리는 돌연변이로 5건 red(앵커 제거 변형은 2건 red)를 확인해 테스트가 가드 코드를 실제로 지난다는 것만 증명했다 — 이 red 를 현재 코드의 결함으로 읽지 말 것. 복구 후 `git diff packages/core/src/util/validate.ts` 가 비어 있음을 확인했다.
- 과제서 2-c(`fallback` 미검사)는 **넣지 않았다.** `ui.list_page_size`/`ui.list_page_max` 가 `registry.ts:120~127` 에서 `type:'int', min:5, max:50` 이고 나머지 호출처는 리터럴이라 비정수 유입 경로를 못 찾았다. 실발생 없는 방어보다 반환 계약 고정을 택했다. (설정 검증이 실제로 int 를 강제하는지 런타임으로 따라가 보지는 않았다 — 선언만 읽었다.)
- 일부러 안 한 것: 라우트 21+6+7곳은 열지 않았다(파일 수 폭증·`limit` 의미 회귀). `pipeline.ts`/`db.ts`/보호 경로도 손대지 않았다.
- 다음 역할이 조심할 것: 이 테스트는 DB·네트워크 없이 돈다(순수 함수 import). 다만 작업 트리에 `node_modules` 가 없으면 `npm ci` 를 먼저 해야 한다. `npm run check` = lint + 3 워크스페이스 typecheck + 28파일 244건, 전부 exit 0 (커밋 df2be97).
- 머지 상태 주의: `cookies-params.test.ts`·`image.test.ts`·`indexnow.test.ts` 는 이 main(cd18b14)에 여전히 없다. 앞선 3회차 PR 이 미머지이며 이번 변경은 그 파일들과 겹치지 않는다.
- [러너 02:18] brief accepted — 채택 — 근거가 지금 코드와 정확히 맞았다(validate.test.ts 의 describe 6개에 새 가드가 없고, 지정된 네 가지 인자 조합과 isRe
- [러너 02:19] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인한 것: diff 1파일(validate.test.ts, 테스트 전용·프로덕션 0줄)의 단정 전부를 util/validate.ts:47-69 구현과 대조했고, 인용된 호출처 12곳의 file:line 을 실제로 열어 일치를 확인했다(catalog.ts 는 admin/catalog.ts:210-211 로 생략 표기). `npm run check` lint+3워크스페이스 typecheck+28파일 244건 exit 0.
- 구현자가 의심한 red→green 부재는 직접 돌연변이로 시험했다: 유한성 검사 제거 2건·isRealDay 정규식만 1건·isUuid 앵커 제거 2건 실패 → 테스트가 가드를 실제로 지난다. 이후 작업 트리 클린 확인.
- 승인이어도 남는 우려(릴리즈 노트에 넣을 것): 커밋 메시지의 "테스트가 0건이었다" 는 queryInt 에만 맞다 — b576143 이 weekly-report.test.ts:37-49 에 isRealDay/isUuid 단정 8건을 이미 넣었다. 틀린 전제이지만 코드는 그 상위집합이라 결함은 아니다. 'describe 3개(12건)' 도 실제 13건.
- 다음 회차 작은 과제: weekly-report.test.ts:37-49 의 주제 무관 중복 블록을 validate.test.ts 로 옮기기. 그리고 "공백" 을 고를 때 describe 이름이 아니라 함수명 전역 grep 으로 확인할 것(이번 정찰이 같은 함정에 걸렸다).
- 못 본 것: 라우트 17곳의 limit 의미·RLS·운영 실환경. 보안/법무 차단 사유 없음(신규 경로·개인정보·비밀값·의존성 변경 없음, revert 로 완전 복구).
- [러너 02:22] review approved — 리뷰 승인 (risk=low)
- [러너 02:22] pr created — https://github.com/hkjang/yeopjari/pull/4
- [러너 02:27] ci no-ci — 이 커밋에 검사가 없음 (정책 allow_merge_without_ci 가 없으면 차단)
