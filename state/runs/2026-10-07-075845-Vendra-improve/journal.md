# 회차 노트 2026-10-07-075845-Vendra-improve — Vendra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:58] base pinned — main@9f50392
- [러너 07:58] autonomy release — 

## 정찰 노트
- 지난 회차(fe3f618)가 응답에 `"order":applied` 를 담았는데 웹이 그 키를 읽지 않는다 — 좌표를 눈으로 확인했다(`objects.go:118` 대 `Objects.tsx:198`·`:176`·`:329`). 같은 결함의 나머지 절반이고 프로덕션 1파일이라 1순위로 골랐다.
- 제친 후보: 「load 에 catch 가 없어 실패가 영원한 스피너가 되는 것」은 가치가 비슷하나 **같은 함수**를 건드려 한 회차에 섞으면 둘 다 흐려지므로 차선으로 내렸다. ELSE 0/END(null) 은 네 표면의 계약 결정이 선행이라 또 보류, CI node-version 은 보호 경로에 검증 불가.
- 추측으로 적은 것: 안내 문구의 **한국어 표현**(지정하지 않고 형태만 제시했다)과 `appliedOrder` 를 select 의 `value` 로 쓸 때의 「되돌아옴」이 사람에게 어떻게 읽히는지 — 코드로는 맞지만 UX 판단은 구현자 몫으로 남겼다.
- 돌려 보지 못한 것: 웹 테스트 전체. 이 워크트리에 `web/node_modules` 가 없어 `npm ci` 가 선행이고(몇 분) 정찰 예산에서 돌리지 않았다. 기준선은 22 files / 98 tests 라고 기록에만 의존한다.
- 구현자가 조심할 것: Go·권한·SQL 무변경, 400 거절 금지(objects.go:124-130 의 결정), `beforeEach` 로 mock reset 금지, mock `api` 는 세 URL 로 분기해 **셋 다 resolve**.
- [러너 08:04] scout done — 업무 목록 화면이 **서버가 실제로 적용한 정렬**을 읽고 사용자에게 말하게 하기 — 지금 드롭다운은 떨�

## 구현 노트
- 바꾼 것: `web/src/pages/Objects.tsx` 한 곳 — 목록 응답의 `order`(fe3f618 이 넣은 「실제 적용된 정렬」)를 읽어, 요청과 다르면 `truncated` 와 같은 모양의 안내 한 줄을 내고 정렬 `<select>` 의 `value` 를 적용된 값으로 바꿨다. 인라인 `<option>` 네 개는 `objectOrderOptions` 표로 올려 안내 문구의 정렬 이름을 그 표에서만 읽는다(화면이 권한을 다시 판단하지 않는다). Go·권한·SQL 무변경.
- 확신 없는 곳: ① **안내 문구의 한국어**와 「금액을 볼 수 없는 목록은 금액순으로 정렬되지 않습니다」라는 암시의 수위 — 권한 이름은 노출하지 않았지만 이것이 사용자에게 충분히 이해되는지는 UX 판단이고 테스트가 보증하지 않는다. ② **되돌아옴의 UX**: 권한 없는 사용자가 「금액 높은순」을 고르면 URL 은 `amount_desc` 가 되고 드롭다운은 「최근 수정순」으로 돌아오며 안내가 뜬다 — 과제서가 의도한 동작이고 그대로 구현했지만 실제 브라우저에서 사람에게 어떻게 읽히는지는 보지 않았다(헤드리스 확인 안 함).
- 검증 못 한 것: 실제 브라우저·실제 서버 조합(권한 없는 역할로 `?order=amount_desc` 를 눌러 보는 end-to-end)은 하지 않았다 — jsdom 테스트와 mock 응답까지다. DB 통합/Go 테스트는 코드 무변경이라 `gofmt`·`go vet` 만 돌렸다(`go test` 는 돌리지 않음).
- 일부러 안 한 것: `load` 의 `.catch` 누락(실패가 영원한 스피너가 되는 것) — **같은 함수**라 한 회차에 섞으면 둘 다 흐려진다. 다음 회차 1순위로 남겼고 이번 테스트 하네스를 그대로 쓸 수 있다. `saveView` 가 요청한 `order` 를 저장하는 것도 남겼다(어느 값을 저장할지 결정 선행).
- 과제서에 없던 보강 하나: 응답이 **아는 네 값이 아닌** 정렬을 답하면 매칭 `<option>` 이 없어 컨트롤이 첫 옵션으로 조용히 넘어간다(jsdom 실측: `value` 가 `updated_desc` 로 읽혔다) — 또 다른 거짓이라 아는 값일 때만 응답을 믿게 하고 다섯 번째 테스트로 고정했다.
- 다음 역할이 조심할 것: `objects-order.test.tsx` 는 DB 불필요(jsdom + mock `api`)지만 `web/node_modules` 가 필요하다 — 새 워크트리에서는 `npm ci --ignore-scripts` 가 먼저다. mock `api` 는 세 URL(목록/suppliers/saved-views)로 분기해 **셋 다 resolve** 하며, `restoreMocks` 때문에 `beforeEach` 에서 reset 하지 않는다(건드리면 다른 파일에 rejection 이 붙는다). `object-status.test.tsx` 는 `ObjectTable` 만 렌더해 이 변경에 닿지 않는다(대조군).
- [러너 08:11] brief accepted — 채택 — 근거가 코드와 좌표까지 그대로 맞았고(응답 타입 `{items,truncated?}`, `value={order}` 가 URL 파라미터, 네 값이 `<option va
- [러너 08:11] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: 원장에 `- 실패 재현:` 줄이 없어 직접 재현했다 — 프로덕션 파일만 `git checkout main -- web/src/pages/Objects.tsx` 로 되돌리면 새 테스트 5개 중 2개가 이번 결함의 증상 그대로 실패한다(안내 문구 부재, 그리고 `expected 'amount_desc' to be 'updated_desc'`). 나머지 3개는 main 에서도 통과하는 회귀 가드다.
- 돌린 것: web 전체 23 files / 103 tests 통과(기준선 22/98 과 +1/+5 로 일치), `tsc -b --noEmit`·`eslint src --max-warnings 0` 무출력. Go·SQL·권한 무변경이라 Go 테스트는 돌리지 않았다. 서버 계약(`objects.go:118` 의 `order`, `objectOrderApplied` 가 네 값만 답하는 것)과 `Objects.tsx:188` 의 `|| "updated_desc"` 기본값을 눈으로 맞춰 허위 안내 경로가 없음을 확인했다.
- 승인이어도 남는 우려(다음 회차·릴리즈 노트용): 되돌아온 뒤 드롭다운이 이미 `최근 수정순`이라 사용자가 그것을 다시 골라도 onChange 가 뜨지 않는다 — 다른 정렬을 한 번 거치지 않으면 URL 의 `amount_desc` 와 안내가 남는다. `saveView`(:272)가 요청값을 저장하므로 그렇게 저장한 보기는 적용할 때마다 경고를 데려온다(구현자가 의도적으로 남긴 자리).
- 못 본 것: 실제 서버·브라우저 end-to-end(구현자도 안 봄). `role="status"` 조건부 마운트의 스크린리더 낭독 — 같은 파일 :324 의 `list-truncated` 관례와 동일해 문제 삼지 않았다.
- 보안·법무: 차단 소견 없음. 새 엔드포인트·식별자·비밀값·외부 입력·의존성·암호 비교 없고 권한을 넓히지 않으며(프런트 표시 전용, 권한 이름 비노출) 개인정보 신규 처리나 외부 약속 문구도 없다.
- [러너 08:15] review approved — 리뷰 승인 (risk=low)
- [러너 08:15] pr created — https://github.com/hkjang/Vendra/pull/142
- [러너 08:17] ci passed — 검사 2개 모두 success
- [러너 08:17] merge done — 62af6fd
- [러너 08:20] release published — v0.7.71
- [러너 08:21] assets verified — v0.7.71 자산 1개 (이전 v0.7.70: 1)
