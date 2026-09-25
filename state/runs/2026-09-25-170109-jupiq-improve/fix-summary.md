# 수리 요약 (5a67beb)

- 문제(비평 지적 그대로 확인): `btrim(email)<>''`는 공백만 지우므로 `email=E'\t'`·`E'\r\n'`·NBSP만 든 주소가 "보낼 주소 있음"으로 통과해, 그 키에 expiry_notified_at이 찍힌 뒤 resolve의 TrimSpace+validAddress가 버려 흔적 없이 삼켜졌다. postgres:16-alpine에서 `btrim(E'\t')<>''`=true를 직접 확인했다.
- 고침 1: `internal/store/mail.go`에 `nonBlankEmailPattern`(unicode.IsSpace 집합 그대로 열거)을 두고 조건을 `email ~ $2`로 바꿨다. 비평이 제안한 `[[:space:]]`는 쓰지 않았다 — PG16 UTF8에서 NBSP·Ogham 공백을 공백으로 세지 않아(직접 측정) 그 계열이 남는다.
- 고침 2: 틀린 "UserEmails와 같은 조건" 주석을 정정하고, 실제로 남은 차이(validAddress가 더 좁아 `email='nonsense'`는 여전히 표시만 된다 → 근본은 입력 검증)를 명시했다.
- 고침 3: 테스트의 공백 주소를 spaces·tab·crlf·nbsp·mixed 5형태로 넓히고(각각 TrimSpace로 빈 문자열임을 먼저 단언), 반대쪽 경계로 공백에 둘러싸인 멀쩡한 주소는 반드시 안내되는지도 단언했다.
- 검증: gofmt/go vet/make lint 통과, `make test-integration`(store 4.33s·api 0.62s)·`go test -count=1 ./...` 전부 통과, store 통합 -v SKIP 0건. 되돌림 검증으로 `btrim` 복원 시 mail_integration_test.go:212가 실제로 FAIL함을 확인했고(탭·crlf·nbsp·mixed 4명이 보고됨) 테스트 컨테이너는 제거했다.
