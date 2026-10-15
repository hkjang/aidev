# 회차 노트 2026-10-15-232905-jikim-improve — jikim
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 23:29] base pinned — main@53fa9f5
- [러너 23:29] autonomy release — 

## 정찰 노트
- Transit 단건의 미지원 파라미터 조용한 무시를 골랐다. 보류였던 "응답 형태를 새로 정해야 한다"는 장벽이 v0.2.37 로 사라졌다 — 단건은 이미 `missing plaintext to encrypt` 400 을 내므로 같은 모양에 batch 와 같은 문자열만 쓰면 되고, 프로덕션은 openbao.go 1파일이다.
- 제친 것: ciphertext 버전 토큰 비정규 표기(차선, PostgreSQL 통합 테스트가 필요해 45분 안에 정직한 재현이 어렵다), decodeBaoJSON DisallowUnknownFields(호환성 계약 변경 선행), CSP 속도 제한·KV TOCTOU(새 계약·동시성), 문서 단독 후보(관찰 가능한 변화 없음).
- 추측으로 적은 것: 수정 전 단건 `context` 요청이 200 을 준다는 것은 코드 독해(입력 구조체에 필드 없음 + decodeBaoJSON 이 미지의 필드 허용)로만 단정했고 실행 재현은 하지 않았다. compatibility.md 에 넣을 정확한 문구도 구현자 재량으로 남겼다.
- 구현자가 조심할 것: 검사를 반드시 `input.BatchInput != nil` 분기 **뒤**에 둘 것(compatibility.md:125 계약), 메시지를 batch 와 글자까지 같게 할 것, seam 호출 카운터로 "암호화기가 불리지 않음"까지 단정할 것(지난 회차에 주입 seam 때문에 결함이 200 으로 보인 전례).
- [러너 23:34] scout done — OpenBao Transit **단건** encrypt·decrypt 가 미지원 파라미터(`context`·`nonce`·`associated_data`·`key_version`)를 조용히 �

## 구현 노트
- 무엇·왜: Transit 단건 encrypt·decrypt 의 입력 구조체에 `baoTransitBatchItem` 을 익명 임베딩하고, batch 와 **글자까지 같은** `unsupported transit parameter: <이름>` 을 `400` 으로 돌려주는 검사를 양쪽에 넣었다. 수정 전에는 `{"plaintext":"aGk=","context":"Y3R4"}` 가 context 를 버린 채 200+ciphertext 를 받아, 같은 서버의 batch 경로와 정반대 계약이었다. 커밋 `2d909f1`(프로덕션 1파일 +19/-3, 테스트 1파일 신규, compatibility.md 1문단+1줄).
- 확신 없는 곳·검증 못 한 것: (1) **임베딩의 부수효과** — encrypt 단건이 이제 `ciphertext`·`reference` 를, decrypt 단건이 `plaintext` 를 디코드한다. 둘 다 읽지 않으므로 관찰 가능한 변화는 없다고 판단했고(지금도 `decodeBaoJSON` 이 미지의 필드로 받아 무시했다) 기존 transit 테스트 전부가 통과하지만, "디코드되지만 쓰이지 않는 필드" 가 비평가에게 걸릴 수 있는 지점이다. 비평가가 먼저 볼 곳. (2) 실제 OpenBao 와의 differential 비교는 하지 않았다 — 단건에서 `400` 이 OpenBao 의 판정과 같은지는 확인하지 않았고, 근거는 "jikim 의 batch 경로가 이미 거절한다" 는 **내부 일관성**뿐이다. OpenBao 자체는 이 파라미터를 구현하므로 거절하지 않는다. (3) `withBaoAuth`·실제 routes 를 통과하는 통합 경로는 미검증(store 가 필요해 이 파일 관례대로 핸들러를 직접 불렀다). (4) PostgreSQL 통합 테스트는 DSN 미설정으로 skip, 브라우저·이미지 E2E 미실행.
- 일부러 하지 않은 것: `DisallowUnknownFields` 를 켜지 않았다(별도 호환성 계약 변경). 빈 문자열·`key_version: 0` 을 거절로 바꾸지 않았다(`unsupportedParameter()` 가 이미 걸러내며 batch 와 갈라지면 안 된다). 검사를 `BatchInput != nil` 분기 앞으로 올리지 않았다(compatibility.md:125 계약) — 이 경계는 batch 요청에 top-level `context`/`key_version` 을 섞은 테스트로 고정했다. decrypt 의 `recordSensitiveDisclosure` 503 순서는 건드리지 않았다. 버전 bump·CHANGELOG 는 범위 밖.
- 다음 역할이 조심할 것: 새 테스트는 DB 가 필요 없고 `go test ./internal/httpapi/ -run 'Transit' -count=1 -v` 로 1초 안에 전부 돈다. 결함을 고친 것이 임베딩이 아니라 두 `unsupportedParameter()` 검사임을 분리 확인했다 — 임베딩만 남기고 검사 블록 두 개를 떼면 같은 10개가 다시 실패한다(검사만 떼는 것은 빌드가 깨지지 않으니 재현이 쉽다). `bash scripts/verify.sh` exit 0(`검증 완료: jikim v0.2.37`, 프런트 59/59). 빌드의 500kB 청크 경고는 이전부터 있던 것이다.
- [러너 23:38] brief accepted — 채택 — 행 번호(655-658·668-676·734-737·747-753·602-614·703-708)·권장 수정 형태(`baoTransitBatchItem` 임베딩)·분기 순서 경고·하네
- [러너 23:38] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: 검사 블록 두 개만 떼면 새 서브테스트 10개가 전부 실패함을 직접 재현(임베딩만으로는 안 고쳐짐 — 구현자 주장 사실). gofmt/vet 무출력, `go test ./... -count=1` 전부 통과. 검사가 `authorizeTransit` **뒤**·`BatchInput != nil` 분기 뒤·decrypt 의 `recordSensitiveDisclosure` 앞에 있어 인가 누수·평문 선반환 없음. 오류 문구에 파라미터 **이름**만 들어가고 값은 안 들어감.
- 임베딩 부수효과를 main 과 대조 실행해 확인했다(구현자가 의심한 자리): 단건 `{"plaintext":"aGk=","nonce":5}`·`"key_version":"2"`·`"reference":7` 이 main 200 → HEAD `400 invalid request body`, batch 요청 `{"batch_input":[…],"key_version":"x"}` 도 main 200 → HEAD 400. **타입이 틀린** 값만 영향이며(정상 타입은 문서대로 batch 는 200, 단건은 `unsupported transit parameter: <이름>`), main 도 상단 `plaintext:123` 에서 이미 같은 모양으로 400 이었으므로 기존 비일관성의 확장이다. 거절 사유로는 약하다고 보고 approve.
- 못 본 것: 실제 OpenBao 와의 differential, `withBaoAuth`·실제 routes 통과 경로, PostgreSQL 통합(DSN 없음), 브라우저·이미지 E2E.
- 릴리즈가 알아야 할 것: 이것은 0.2.37 까지 `context`/`nonce`/`associated_data`/`key_version` 을 단건에 보내 200 을 받던 클라이언트에 **깨지는 변경**이다(실제 OpenBao 는 거절하지 않는다). CHANGELOG 에 호환성 깨짐으로 적고, 타입 오류는 `unsupported transit parameter:` 가 아니라 `invalid request body` 로 간다는 점도 함께 적을 것. CHANGELOG:18 의 "단건 미지원 파라미터 무시는 범위 밖" 문장은 0.2.37 기술로는 여전히 맞다.
- [러너 23:42] review approved — 리뷰 승인 (risk=low)
- [러너 23:42] pr created — https://github.com/hkjang/jikim/pull/63
- [러너 23:45] ci passed — 검사 2개 모두 success
- [러너 23:45] merge done — 2d909f1
- [러너 23:53] release published — v0.2.38
- [러너 23:56] assets verified — v0.2.38 자산 2개 (이전 v0.2.37: 2)
