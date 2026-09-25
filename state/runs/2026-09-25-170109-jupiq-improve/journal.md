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

## 비평 노트
- 판정 reject(차단 부서 없음, risk low). 확인한 것: postgres:16-alpine 을 직접 띄워 새 통합 테스트가 통과하고, mail.go:169 의 새 조건 한 줄만 임시로 지우면 mail_integration_test.go:195 에서 FAIL 함을 확인했다 — 테스트는 유효하고 변경 방향도 옳다. 확인 뒤 파일 원복, 워킹트리 깨끗(git status 무출력), 컨테이너 제거.
- 거절 사유 하나: `btrim(email)<>''` 의 btrim 기본 제거 문자가 **공백 하나뿐**이라 `email=E'\t'`(또는 `E'\n'`)은 통과한다. 같은 컨테이너에서 `btrim(E'\t')<>''` = true 를 확인했고, 그 키는 expiry_notified_at 이 찍힌 뒤 resolve 의 TrimSpace+validAddress 에 걸려 Notify 가 0개로 조기 반환 — mail_deliveries 도 로그도 없이 삼켜진다. 즉 **이 커밋이 닫았다고 말하는 그 계열이 남았다**. 도달 경로: PATCH /auth/me → UpdateProfile(users.go:221) 이 email 을 다듬지도 검증하지도 않는다.
- 부수적으로 mail.go:159-163 주석의 "UserEmails 와 같은 조건" 단언이 사실이 아니며(등가 주장이 '보내기 전 표시' 설계의 유일한 근거다), 테스트는 공백뿐인 주소를 `'   '` 로만 세워 덮는 범위를 실제보다 넓게 말한다.
- 수리가 먼저 볼 곳: internal/store/mail.go:169 을 `email ~ '[^[:space:]]'` 로(empty/spaces/tab/newline/CRLF false, `'a@b.c'`·`E'\t a@b.c \n'` true 를 확인했다), 위 주석 정정, mail_integration_test.go:182 의 blankID 에 탭까지 포함. 스키마 변경 불필요, revert 로 온전히 복구된다.
- 못 본 것·남는 우려: 프런트·실제 SMTP·실제 Keycloak 은 손대지 않았고 store 외 패키지 통합은 돌리지 않았다. 릴리즈 노트용: `email='nonsense'` 계열은 이 범위 밖으로 남으며 근본 해법은 UpdateProfile·UpsertOIDCUser 의 입력 정규화·검증이다(그것이 들어가면 SQL 흉내가 필요 없어진다). CI 는 DSN 을 주므로(ci.yml:59) 새 단언은 CI 에서 실제로 실행된다 — 구현 노트의 우려보다는 낫다.
- [러너 17:18] review rejected — 리뷰 거절: internal/store/mail.go:169 새 조건 `btrim(email)<>''` 은 PostgreSQL btrim 의 기본 제거 문자가 공백 하나뿐이어서 탭·줄바꿈만 든 주소를 '보낼 주소가 있

## 수리 노트
- 맞았던 지적: btrim은 공백만 지워 탭·줄바꿈만 든 주소가 통과한다(컨테이너에서 재현), 159-163행 "UserEmails와 같은 조건" 주석이 사실이 아니다, 테스트가 공백 한 형태만 고정한다 — 셋 다 그대로 맞았다.
- 부분적으로 틀렸던 지적: 권고안 `email ~ '[^[:space:]]'`는 부족하다. PG16 UTF8에서 `[[:space:]]`가 NBSP(U+00A0)·Ogham(U+1680)·narrow NBSP(U+202F)를 공백으로 세지 않아(직접 측정) 같은 계열이 남는다. 그래서 unicode.IsSpace 집합을 그대로 열거한 `nonBlankEmailPattern` 상수로 두고 `email ~ $2`로 넘겼다(로케일 의존 제거).
- 테스트는 spaces·tab·crlf·nbsp·mixed 5형태 소유자를 세우고 각 주소가 TrimSpace로 비는지 먼저 단언한 뒤 전부 제외됨을 확인하며, 반대쪽 경계(공백에 둘러싸인 멀쩡한 주소는 안내된다)도 새로 단언한다. 주석의 등가 주장은 삭제하고 남은 차이(validAddress가 더 좁다)를 적었다.
- 확신 없는 곳: `email='nonsense'` 계열은 여전히 표시만 되고 안내되지 않는다(범위 밖, 근본은 UpdateProfile·UpsertOIDCUser 입력 검증). Postgres 대괄호식이 문자 그대로를 담으므로 collation이 바뀌어도 판정은 같지만, PG14(릴리스 통합 경로)에서는 직접 돌리지 않았다 — 문자 열거식이라 버전 의존은 없다고 본다.
- 검증: make lint·make test-integration·go test ./... 통과, SKIP 0건, 되돌림 시 mail_integration_test.go:212 FAIL 확인. 새 커밋 5a67beb, 컨테이너 제거.
- [러너 17:25] repair done — # 수리 요약 (5a67beb)  - 문제(비평 지적 그대로 확인): `btrim(email)<>''`는 공백만 지우므로 `email=E'\t'`·`E'\r\n'`·NBSP만 든 주소가 "보낼 주소 있음"으로 통과해
