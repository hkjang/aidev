# 회차 노트 2026-09-25-170109-jupiq-improve — jupiq
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 17:01] base pinned — main@234387a
- [러너 17:01] autonomy release — 
- [러너 17:05] scout done — API 키 만료 안내가 **받을 주소가 없는 소유자의 키까지 "안내함"으로 표시해** 그 키를 영구히 삼키는 것�

## 구현 노트
- ExpiringAPIKeys의 UPDATE WHERE에 소유자 조건(`active AND btrim(email)<>''`)을 더해, 보낼 주소가 없는 소유자의 키를 "안내함"으로 표시한 뒤 흔적 없이 삼키는 것을 막았다(7b6f5b3). 수신자 해석(UserEmails)과 같은 계정만 표시하므로 두 경로가 같은 입력을 같게 읽는다.
- 확신 없는 곳: SQL 조건은 "다듬으면 비어 있지 않음"까지만 본다. mail/service.go:resolve는 validAddress로 더 좁게 거르므로 `email='nonsense'`인 소유자의 키는 **여전히 표시만 되고 안내되지 않는다** — 남은 같은 계열 결함이다(ideas.json에 적었다). SQL로 주소 모양을 흉내 내면 두 파서의 계약이 갈라지므로 일부러 하지 않았다.
- 일부러 하지 않은 것: 표시를 발송 성공 뒤로 미루는 두 단계 재설계(릴레이 장애 시 되풀이 발송이 생긴다), collector.go:notifyExpiringKeys 변경, 마이그레이션 추가(스키마 변경 불필요), UpdateProfile의 입력 정규화(프로필 API 응답이 바뀌어 프런트·OpenAPI 확인이 필요 — 별도 과제로 남겼다).
- 과제서의 함정 예고는 실제로는 발생하지 않았다: mail_integration_test.go 92행이 adminID의 email을 이미 채우므로 161행 기존 단언은 그대로 통과한다(정찰의 근거와 다른 점).
- 다음 역할이 조심할 것: 새 단언은 **DB가 있어야 돈다**(JUPIQ_INTEGRATION_TEST_DSN). DSN 없이 `go test ./...`는 전부 skip되어 이 수정의 증거가 되지 않는다. 정리 defer는 `DELETE FROM users WHERE username LIKE 'marker%'`로 넓혔다(api_keys는 ON DELETE CASCADE).
- 검증: gofmt -l 무출력, go vet ./..., make test-integration(store 2.55s·api 0.64s), go test -count=1 ./... 통과, store 통합 -v SKIP 0건, check-version 1.8.1. 되돌림 검증으로 기준 1)·2)가 실제로 실패함을 확인했다. postgres:16-alpine 컨테이너는 제거했다.
- [러너 17:11] brief accepted — 채택 — 진단(표시가 수신자 해석보다 먼저라 주소 없는 소유자의 키가 흔적 없이 삼켜진다)이 코드와 정확히 일치했고 �
- [러너 17:12] verify passed — 검증 7개 통과 (auto)
