# 회차 노트 2026-10-07-161813-igame-improve — igame
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:18] base pinned — main@4e7911e
- [러너 16:18] autonomy release — 

## 정찰 노트
- secretbox 를 골랐다: 프로덕션 0파일·DB 불필요·S 라 파일 상한과 보호 경로를 전혀 건드리지 않고, AAD 를 양쪽에서 지워도 현재 테스트가 통과하는 공백이 지금 코드에 그대로 있다(secretbox.go·secretbox_test.go 전체를 읽어 확인, 기존 테스트 PASS 실측).
- 제친 후보: 페이지네이션 tiebreak(M, PG 정렬 의존이라 실DB Red 를 먼저 만들어야 해 한 세션에 안 끝날 위험) → 차선으로 남겼다. serviceLocation 은 프로덕션 6파일 + playAllowed 와 계약 갈라짐으로 여섯 회차째 보류.
- 추측으로 적은 것: 차선 후보의 행 번호(admin.go:552/727 등)와 playWindowsAllow·loadAPIKeyPolicyContext·README 항목은 이전 회차 근거이며 이번에 재확인하지 않았다 — ideas.json 에 "미확인" 으로 표시했다.
- 구현자가 조심할 것: 거부 테스트는 에러 문구를 문자열 비교하지 말고, 같은 테스트에 성공 왕복을 함께 둬서 "항상 에러" 변이가 통과하지 않게 할 것. 1비트 변조는 nonce 가 아니라 끝의 GCM 태그 바이트를 뒤집을 것. base64 실패 입력에는 알파벳 밖 문자를 반드시 포함시킬 것(영숫자만이면 유효하게 디코드될 수 있다).
- secretbox.go 는 읽기만 할 것 — 설치키 회전 경로가 없어 `v1:` 포맷과 AAD 는 저장된 OIDC client_secret·AI API 키의 영구 계약이다(Open 호출처는 auth.go:203·admin.go:456 둘뿐, server_proof 는 쓰기 전용임을 grep 으로 확인).
- [러너 16:22] scout done — secretbox 의 거부 경로와 AAD 결합을 회귀로 고정 (가치 2 / 위험 1 / 작업량 S)

## 구현 노트
- 무엇/왜: `internal/secretbox/secretbox_reject_test.go` 한 파일만 추가(프로덕션 0파일). 기존 테스트가 왕복만 봐서 AAD 를 양쪽에서 지우거나 `Open` 의 거부 검사를 떼도 녹색이었다 — 설치키 암호화 계약을 회귀로 고정했다.
- 확신 없는 곳: M3(길이 검사 제거) 변이는 에러가 아니라 `panic: slice bounds out of range [:12] with capacity 11` 로 Red 가 된다. 테스트는 여전히 FAIL 이고 인과도 증명되지만, "에러를 돌려준다" 가 아니라 "패닉을 막는다" 가 그 검사의 실제 역할이라는 점은 과제서 예상과 다르다. 그리고 `padded base64` 케이스는 평문 길이에서 나오는 47바이트 → 패딩 1개에 의존하므로, 테스트 안에 `strings.HasSuffix(padded,"=")` 가드를 넣어 전제가 깨지면 바로 Fatal 이 되게 했다.
- 검증 못 한 것: PG 회귀는 DSN 이 없어 `go test ./cmd/... ./internal/... ./migrations/...` 에서 조용히 skip 된다 — 이 과제는 DB 가 필요 없으므로 `make test-db` 를 돌리지 않았다. web/SDK 쪽(`make lint`/`make test`/`make web-build`)도 Go 테스트 한 파일만 바뀌어 돌리지 않았다.
- 일부러 하지 않은 것: `secretbox.go` 를 전혀 수정하지 않았다(에러 문구 통일·`v2:`·AAD 에 key id 는 저장된 ciphertext 의 계약이고 설치키 회전 경로가 없다 — `ideas.json` 에 rejected 로 적어 뒀다). 거부 테스트는 에러 문구를 비교하지 않는다. 차선 후보(목록 tiebreak)는 1순위가 성립해 손대지 않았다.
- 다음 역할이 조심할 것: 이 테스트는 패키지 내부(`package secretbox`)라서 `b.aead.NonceSize()` 에 닿는다 — 외부 패키지로 옮기면 짧은 payload 케이스를 만들 수 없다. DB·네트워크 불필요, `go test ./internal/secretbox/ -race -count=3` 1.017s.
- [러너 16:26] brief accepted — 채택 — 근거가 지금 코드와 정확히 일치했고(`secretbox.go:32/37/41/45` 의 AAD·접두사·길이 검사, 테스트 1개뿐인 상태) 지정�
- [러너 16:27] verify passed — 검증 4개 통과 (policy)
- [러너 16:27] pr created — https://github.com/hkjang/igame/pull/36
- [러너 16:27] guard held — internal/secretbox/secretbox_reject_test.go 
