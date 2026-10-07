# PR 처리기 노트 2026-10-08-030237-law.irumx.app-shepherd — law.irumx.app PR #1
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-08-013826-law.irumx.app-improve)
# 회차 노트 2026-10-08-013826-law.irumx.app-improve — law.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 01:38] base pinned — main@2bffff3
- [러너 01:38] autonomy release — 

## 구현 노트
- 고친 것: `isSafeLink`(packages/core/src/util/text.ts)가 끝 점(루트 라벨)을 떼고 라벨·localhost 는 검사하면서 `BLOCKED_LINK_HOSTS` 조회만 원본 호스트로 해서 `http://metadata.google.internal./` 가 통과했다. 끝 점을 한 번만 떼어 두 검사가 같은 값을 쓰게 했다. 프로덕션 파일 1개 + 테스트 1개.
- 확신 없는 곳: **`isSafeLink` 는 이 저장소 안에 호출자가 없다**(grep 결과 테스트뿐). 즉 프로덕션 배선을 지나는 증거가 없고, 순수 함수의 계약 테스트로만 증명했다. 운영자 규칙의 "end-to-end 확인"을 이 회차는 만족시키지 못한다 — 비평가는 이 점을 먼저 보시기 바란다. 배선 자체는 동작 변경이라 ideas.json 의 별도 과제로 뺐다.
- URL 정규화 동작은 가정하지 않고 node 22 로 직접 확인했다: 이름에는 끝 점이 남고(`metadata.google.internal.`), IPv4 리터럴에서는 지워진다(`127.0.0.1.` → `127.0.0.1`). 그래서 IPV4_HOST 분기는 건드리지 않았다.
- 일부러 안 한 것: `tenancy/hosts.ts` 의 `normalizeHost` 도 같은 끝 점 처리를 하지만 통합하지 않았다(파서를 합치지 말고 좁히라는 지침). `BLOCKED_LINK_HOSTS` 목록을 늘리지도 않았다 — 결함과 무관한 확장이다.
- 다음 역할이 조심할 것: 테스트는 DB 없이 돈다(`npm run check` = eslint + 3개 워크스페이스 tsc + vitest 96건, 전부 통과). 단 `npm install` 이 package-lock.json 에 무관한 `"peer": true` 메타데이터를 26곳 넣으므로 커밋에서 되돌렸다 — 커밋 전 `git status` 로 다시 확인할 것.
- [러너 01:48] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인한 것: main 의 text.ts 를 되돌려 새 테스트를 돌려 2건이 이번 증상으로 실제 실패하는 것을 봤다(원장에 `- 실패 재현:` 줄이 없어 직접 재현). 수정 코드에서 `npm run check` 전부 통과(96건), `git status` 깨끗 — package-lock.json 오염 없음. URL 정규화(끝 점 유지/IPv4 에서 제거, 대문자·`%2E`)도 node 로 재확인.
- 못 본 것: 프로덕션 경로를 통과하는 end-to-end 증거. `isSafeLink` 는 저장소에 호출자가 없어 순수 함수 계약으로만 증명됐다 — 구현자가 정직하게 적어 둔 그대로다. 결함이 아니므로 승인.
- 남는 우려 1 (릴리즈 노트): "내부 호스트 차단 강화" 로 쓰면 과장이다. 보호가 실현되는 호출자가 아직 없으므로 "차단 유틸의 계약 보강, 배선은 별도 과제" 로 적을 것.
- 남는 우려 2 (다음 회차): 끝 점이 둘이면(`metadata.google.internal..`, `localhost..`) 목록을 비껴간다. 단 빈 라벨이라 이름이 풀리지 않음을 확인했다(ENOTFOUND) — 공격 경로 없는 선행 결함이고 이번 변경이 넓히지 않았다. 앞 점도 같은 종류. 이 파일을 다시 열 때 함께.
- 보안·법무 차단 없음: 변경은 denylist 를 엄격히 좁히는 방향뿐, 새 입력·권한·개인정보·의존성 없음.
- [러너 01:51] review approved — 리뷰 승인 (risk=low)
- [러너 01:51] pr created — https://github.com/hkjang/law.irumx.app/pull/1
- [러너 01:53] ci api-error — API 오류: gh api failed

## 심사 노트
- 확인한 것: origin/main 의 text.ts 를 꺼내 같은 케이스를 실행해 끝 점 2건이 base 에서 통과→HEAD 에서 차단으로 바뀌는 것을 직접 봤다(새 ALLOWED 1건은 base 에서도 통과 = 회귀 가드). `npm ci` 후 test 96건·typecheck 3 워크스페이스·lint 오류 0 통과, `git status` 비어 있음 — npm ci 는 package-lock.json 을 건드리지 않아 `"peer": true` 오염이 재발하지 않았다.
- 확인한 것: 변경이 단방향으로 안전하다 — `name` 은 host 에서 끝 점 1개만 뗀 값이고 BLOCKED_LINK_HOSTS 항목에 끝 점이 없으므로 집합 조회는 miss→hit 로만 바뀌고 hit→miss 가 불가능하다. 새로 허용되는 호스트가 없다. `tenancy/hosts.ts:21` 의 같은 처리를 통합하지 않은 것도 지침에 맞다.
- 못 본 것: 프로덕션 배선을 지나는 end-to-end 증거. `isSafeLink` 는 호출자가 없다(grep 결과 테스트뿐). 다만 통과시킬 배선 자체가 없고, 검증은 대역·소스 문자열 검사가 아니라 실제 함수를 그대로 실행한 것이며 배선은 ideas.json 의 별도 과제다 — 결함으로 보지 않았다.
- 못 본 것(선행, 공격 경로 없음): `metadata.google.internal..` 과 `.metadata.google.internal` 은 여전히 통과하는 것을 실행으로 확인했다. 빈 라벨은 DNS 와이어 포맷에서 길이 0 = 종단이라 이름이 풀리지 않고, base 에서도 같은 값이 통과했다.
- 권고 근거: approve / merge / risk=low. 보호 파일(auth·oidc·session·migrations) 무관, 새 경로·권한·비밀값·개인정보·의존성 변경 없음, revert 는 순수 함수 2줄. 보안·법무 차단 소견 없음. 릴리즈 노트는 "내부 호스트 차단 강화"가 아니라 "isSafeLink 끝 점 처리 계약 보강, 배선은 별도 과제"로 적을 것.
