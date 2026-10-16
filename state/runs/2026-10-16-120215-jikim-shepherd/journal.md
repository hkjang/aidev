# PR 처리기 노트 2026-10-16-120215-jikim-shepherd — jikim PR #64
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-16-111605-jikim-improve)
# 회차 노트 2026-10-16-111605-jikim-improve — jikim
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:16] base pinned — main@306196e
- [러너 11:16] autonomy release — 

## 정찰 노트
- 보류 1순위였던 transit ciphertext 버전 토큰 laxness 를 골랐다. 막고 있던 것이 "PostgreSQL 통합 테스트 필요" 였는데, `package store` 내부 테스트에서 닿지 않는 DSN(127.0.0.1:1, `MinConns=0`)의 lazy pool 로 `&Store{pool:p}` 를 만들면 "DB 에 닿기 전 거절" 과 "DB 까지 내려감" 이 `errors.Is(err, ErrInvalid)` 로 깔끔히 갈리는 것을 **실행으로 확인**해(0.139s) 작업량이 M→S 로 내려갔다. 수정 전 `v01`·`v+1`·`v001` 이 연결 거부까지 내려가는 출력을 과제서에 그대로 옮겼다.
- 제친 후보: `baoKVMutateVersions` 0·음수(store 동작과 실제 OpenBao 판정 둘 다 여전히 미확인), `decodeBaoJSON` DisallowUnknownFields(계약 변경이라 compatibility.md 개정 선행), `requestedOpenBaoVersion`(결과 버전이 같아 관찰 가능한 피해 없음), 단건/batch 동등성 표(프로덕션 변경 없음 → 차선으로 남김).
- 추측으로 적지 않은 것: 실제 OpenBao 가 `vault:v01:` 을 거절한다는 것은 **미확인**이다(근거는 jikim 자신의 compatibility.md:120 `vault:vN:` 정규형과 136 행의 400 계약). 과잉 거절 방지 단정(`v1`·`v12` 는 그대로 통과)을 반드시 함께 넣어 달라.
- 구현자가 조심할 것: `&store.Store{}`(nil pool)은 panic 하므로 쓰지 말 것. httpapi 의 `transitCiphertextVersion`(openbao.go:871)은 서버 산출물만 파싱하니 통합하지 말고 store 만 좁힐 것. `TransitEncrypt` 의 출력 형식과 AAD 문자열(`transit:%s:v%d`)은 건드리면 기존 암호문이 복호화 불가가 된다.
- [러너 11:22] scout done — Transit decrypt 가 비정규 버전 토큰(`vault:v01:`·`vault:v+1:`·`vault:v001:`)을 거절하지 않고 저장소·복호화까지 내

## 구현 노트
- `store.TransitDecrypt` 의 `vN` 토큰 파싱을 순수 헬퍼 `transitCiphertextKeyVersion`(internal/store/transit.go)으로 좁혀 `vault:v01:`·`vault:v001:`·`vault:v+1:`·`vault:v+01:` 을 DB 에 닿기 전 `ErrInvalid` 로 거절한다. `strconv.Atoi` 가 부호·선행 0 을 통과시키고 AAD 를 파싱된 정수로 재조립해 한 암호문에 여러 표기가 생기던 문제다.
- 확신 없는 곳: 실제 OpenBao 가 `vault:v01:` 을 거절하는지는 **미확인**이다(근거는 jikim 자신의 compatibility.md:120 `vault:vN:` 정규형과 136 행 400 계약). 외부 클라이언트가 지금껏 비정규 표기를 저장해 두고 보냈다면 이번 변경으로 400 을 받는다 — 다만 jikim 이 발급하는 ciphertext 는 언제나 정규형이라 실제 영향은 없다고 본다.
- 검증 못 한 것: PostgreSQL 왕복(정상 암호문을 쓰고 다시 읽기)은 `JIKIM_TEST_POSTGRES_DSN` 이 없어 돌리지 않았다. 새 테스트는 DB 를 **쓰지 않으며**(닿지 않는 lazy pool 이 목적), 정규 토큰 케이스는 `!errors.Is(err, ErrInvalid)` 만 단정해 과잉 거절이 없음을 고정한다. 이미지·브라우저 E2E 미실행.
- 일부러 하지 않은 것: `httpapi/openbao.go:871 transitCiphertextVersion` 통합(서버 산출물만 파싱하는 다른 계약), `TransitEncrypt` 출력 형식·AAD 문자열 변경(기존 암호문이 복호화 불가가 된다), `requestedOpenBaoVersion` 의 같은 모양 laxness(결과 버전이 같아 관찰 가능한 피해 없음 — ideas.json 에 후속으로 남김).
- 다음 역할이 조심할 것: 새 테스트는 DB 가 **없어야** 의도대로 돈다(127.0.0.1:1 에 무엇이든 떠 있으면 연결 거부 가정이 깨진다). `&store.Store{}`(nil pool)은 여전히 panic 하니 쓰지 말 것. 버전·CHANGELOG 는 손대지 않았다(v0.2.38 그대로).
- [러너 11:26] brief accepted — 채택 — 행 번호(transit.go:42-54·47·66, compatibility.md:120·136)·권장 헬퍼 형태·재현 골격(lazy pool, 0.14초)·`strconv.Atoi` 의 실제 �
- [러너 11:26] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 테스트는 진짜다 — HEAD 를 scratch 로 복사해 호출부만 옛 `strconv.Atoi` 로 되돌리니 `v01`·`v001`·`v+1`·`v+01` 4개만 실패하고(원장 메시지와 일치) 나머지 11개는 통과했다. 헬퍼 경계(선행 0·부호·비숫자·overflow)와 `TransitEncrypt` 정규형 발급(과잉 거절 없음), AAD 무변경도 확인했다. vet·gofmt·store 테스트 깨끗.
- 거절 사유는 문서 한 곳이다: `docs/guides/compatibility.md:136` 이 "저장소를 호출하기 전에 400" 이라고 쓰지만 거절은 저장소 **안**(`internal/store/transit.go:67`)에서 일어난다 — httpapi decrypt 단건은 `nil` 검사만 하고 바로 `decryptTransit` 를 부르며 `openbao.go:871` 파서는 encrypt 산출물 전용이다. 같은 문서 123·125 행의 동일 문구는 실제로 httpapi 에서 끊기는 성질을 가리키므로 136 행이 성립하지 않는 약속을 한다. 덧붙여 136 행에는 batch 판정(항목별 `error`, 성공 1건이면 `200`)이 빠졌다.
- 수리가 가장 먼저 볼 파일: `docs/guides/compatibility.md:136` 뿐이다. 코드·테스트는 손댈 것이 없다("저장소가 DB 에 닿기 전에" 로 고치거나 해당 절 삭제 + batch 괄호 추가).
- 차단 없음(security·legal 모두 해당 없음) — 변경은 입력 수용 범위를 좁히기만 하고 새 경로·권한 확대·비밀값 노출·개인정보·의존성이 없다. revert 는 3파일로 끝난다.
- 못 본 것: PostgreSQL 왕복·E2E·verify.sh 전체. 다음 회차가 알 것 — 새 테스트는 127.0.0.1:1 이 비어 있다는 가정에 의존하고, 실제 OpenBao 가 `vault:v01:` 을 거절하는지는 여전히 미확인이며, 손으로 만든 비정규 암호문을 보관한 외부 클라이언트는 이제 400 을 받는다(릴리즈 노트 후보).
- [러너 11:29] review rejected — 리뷰 거절: docs/guides/compatibility.md:136 새 문장이 "비정규 표기는 저장소를 호출하기 전에 400"이라고 단정하지만 실제 거절은 저장소 **안**(internal/store/tran
- [러너 11:30] pr created — https://github.com/hkjang/jikim/pull/64

## 수리 노트
- 지적 두 개 모두 맞았다: (1) 비정규 토큰 거절은 저장소 **안**(`internal/store/transit.go:67`)에서 일어나고 httpapi 단건 decrypt(`openbao.go:768-772`)는 `nil` 만 보고 store 를 부른다 — "저장소를 호출하기 전에" 는 성립하지 않는다. (2) batch 는 항목별 `error` 라서 성공 항목이 있으면 `200`(`baoTransitBatchStatus`, openbao.go:628-636)이다. 틀린 지적은 없었다.
- 고친 방법: `docs/guides/compatibility.md` 의 그 한 문장만 "저장소가 DB에 닿기 전에 거절" + "httpapi 가 아니라 저장소 안에서의 판정이므로 단건 `400`, batch 는 항목별 `error`(성공 1건이면 `200`)" 로 교체. 코드·테스트는 비평가 판단대로 무변경.
- 검증: verify-docs.mjs 통과, `go vet ./...` 깨끗, `go test ./... -count=1` 전체 통과, `gofmt -l` 없음. batch `200` 주장은 기존 테스트 `TestOpenBaoTransitDecryptBatchKeepsOrderAndReportsItemErrors` 가 ErrInvalid 항목+성공 항목으로 이미 고정한다.
- 여전히 확신 없는 곳: 실제 OpenBao 가 `vault:v01:` 을 거절하는지는 미확인(이전 회차와 동일). PostgreSQL 왕복·E2E·verify.sh 전체·프런트 미실행 — 문서 한 줄 변경이라 영향은 없다고 본다. 새 store 테스트가 127.0.0.1:1 공백 가정에 의존하는 점도 그대로다.
- 커밋: `fee6e17` (브랜치 위 새 커밋). push 하지 않았다.

## 심사 노트
- 확인한 것: HEAD 트리를 /tmp 로 복사해 호출부만 옛 `strconv.Atoi` 로 되돌리니 새 테스트의 `v01`·`v001`·`v+1`·`v+01` 4개가 connection-refused 로 실패 — 테스트가 변경을 실제로 고정하며, 프로덕션 `(*Store).TransitDecrypt` 를 seam 없이 부른다. 정규 `v1`·`v12` 는 `errors.Is(err, ErrInvalid)==false` 로 과잉 거절도 막는다.
- 이전 거절 사유 2개 해소를 소스로 확인: 판정이 `s.transitKey` 쿼리(transit.go:75)보다 앞서므로 "DB에 닿기 전" 이 사실이고, 단건 `400` 은 `baoTransitFailure`(openbao.go:643-652), batch 항목별 `error`+성공 1건 `200` 은 `baoTransitBatchStatus`(openbao.go:628-636)로 성립한다. httpapi 의 동종 파서는 호출부 두 곳(openbao.go:695·726)이 모두 자사 encrypt 산출물뿐이라 통합하지 않은 설계가 맞다.
- 실행: `go test ./... -count=1` 전체 통과, `go vet`·`gofmt -l` 무출력, `verify-docs.mjs` 통과. 버전·CHANGELOG 무변경.
- 못 본 것: 실제 OpenBao 의 `vault:v01:` 판정(여전히 미확인), PostgreSQL 왕복, E2E, `verify.sh` 전체, 프런트. 문서 136 행의 "선행 0이 없는 10진수" 는 `v0` 을 유효로 읽히게 하지만 `v0` 거절은 변경 전부터의 동작이고 문장의 실질 약속은 성립하므로 notes 로만 남겼다.
- 권고 merge(risk low): 코드는 입력 수용 범위를 좁히기만 하고 보호 파일·마이그레이션·권한·개인정보에 닿지 않으며 revert 가 3파일로 끝난다. 릴리즈 노트 후보 — 손으로 만든 비정규 ciphertext 보관 클라이언트는 이제 400 을 받는다.
