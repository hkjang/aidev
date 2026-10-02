# 회차 노트 2026-10-03-074745-igame-improve — igame
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:47] base pinned — main@b2860a3
- [러너 07:47] autonomy release — 
- [러너 07:52] scout done — OIDC 로그인/콜백이 "설정을 읽을 수 없음"을 "SSO 미설정"으로 바꿔 보고하고 로그도 남기지 않는 것을 고치

## 구현 노트
- `oidcLogin`·`oidcCallback` 이 `loadOIDCSetting` 의 err 를 `!cfg.Enabled` 와 같은 분기에 넣어 전사 로그인 장애(일시적 DB 실패 / 설치키 변경에 따른 `Secrets.Open` 실패)를 "SSO 미설정" 으로 답하고 로그도 남기지 않았다. 헬퍼 `oidcUnreadable` 로 `pgx.ErrNoRows` 가 아닌 에러만 `serverError` 를 거쳐 503 `oidc_unavailable` 로 보낸다. 미설정 응답(404/400 `oidc_disabled`)과 콜백의 state 선소비 순서는 손대지 않았다.
- 확신 없는 곳: (1) `oidc_unavailable` 이 새 코드인데 `docs/api.md:96` 에 이미 **다른** 코드 `oidc_setting_unavailable`(관리자 PUT 경로)이 있다 — 비슷한 이름 둘이 생겼다. 과제서가 `oidc_unavailable` 을 지정했고 계약이 서로 다른 경로(읽기 vs 쓰기)라 그대로 뒀지만, 통일을 원하면 이름 판단이 필요하다. (2) 응답 message 를 주변 관례대로 영어로 뒀다(`oidc_discovery_failed` 와 동일); 과제서는 한국어 예시를 들었다. 한국어 문구는 `errorMessages.ts` 에만 있다. (3) SDK(`sdk/gamehub-js`)에 이 코드를 다루는 곳이 있는지는 `rg oidc_disabled` 결과가 비어 있다는 것만 확인했다 — 코드 문자열 목록을 따로 들고 있지는 않아 보였다.
- 일부러 하지 않은 것: `loadAISetting`/`ai.go` 의 같은 모양(응답 계약이 별개, 파일 수 증가), `serviceLocation`(호출처 6곳), `migrations/`·`.github/workflows/`, 세션 쿠키·토큰 교환·ID 토큰 검증·`oidc_flows` 쿼리.
- 다음 역할 주의: `internal/api/oidc_unavailable_pg_test.go` 는 **실DB 가 있어야 돈다**(`IGAME_TEST_DSN` 없으면 skip). 콜백의 503 은 DB 없이 증명할 수 없다 — `DELETE FROM oidc_flows … RETURNING` 이 먼저 실패해 400 `invalid_state` 가 되기 때문이고, 과제서가 순서 변경을 금지했으므로 이것이 의도된 구조다. 반면 `oidc_unavailable_test.go`(login) 는 DB 없이 돈다. 두 파일은 헬퍼(`oidcSettingServer`, `sealedWithAnotherKey`, `errorEnvelope`)를 공유하므로 한쪽만 지우면 빌드가 깨진다.
- 이 회차에서 `make deps` 를 돌려 `node_modules` 를 설치했다(`make web-build` 타입 검사용). 커밋에는 들어가지 않았음을 `git status` 로 확인했다.
- [러너 08:00] brief accepted — 채택 — 근거가 지금 코드와 정확히 일치했고(auth.go:212/302 의 접힌 분기, `writeError` 의 로그 부재, `playAllowed` 선례) 지정한 
- [러너 08:01] verify passed — 검증 4개 통과 (policy)

## 비평 노트
- 판정 reject. 수리가 먼저 볼 곳: `docs/api.md:98` 과 `internal/api/auth.go:321-324`. 문서가 "DB 일시 장애면 두 경로 모두 503 oidc_unavailable + 로그" 라고 단언하지만, callback 은 설정보다 먼저 flow DELETE 를 하므로 DB 장애에서는 `400 invalid_state` + 무로그다(구현자 자신이 pg_test 주석과 ledger 에 적어 둔 사실). 문장을 좁히거나, 순서를 바꾸지 않고 ErrNoRows 만 400 으로 남기고 나머지를 oidcUnreadable 로 보내면 해소된다.
- 확인한 것: auth.go 전체 OIDC 흐름, serverError/writeError/logRequestError/settingRaw/secretbox.Open, 테스트 2파일 전부, web 의 login 호출 방식. `go vet ./internal/api/` 무출력, `go test ./internal/api -run OIDC -v` 통과(pg 4개는 DSN 없어 skip). 수정 전 404 가 되는 것은 diff 로 확인했고 코드를 되돌려 돌려보지는 않았다.
- 못 본 것: 실DB 회귀(IGAME_TEST_DSN 없음 — callback 503 은 이 세션에서 미증명, ledger 의 make test-db 기록만 있음), web 빌드·vitest, 전체 go test.
- 보안·법무: 차단 없음. 응답 본문은 고정 문구, err 는 로그만; secretbox 오류에 평문·ciphertext 없고 테스트가 누출을 직접 단언. 새 개인정보 수집·전송·의존성 없음.
- 승인했더라도 남을 우려(릴리즈 노트/다음 회차): login/callback 은 브라우저 내비게이션이라 errorMessages.ts 의 한국어 문구가 이 경로에서 쓰이지 않음(원시 JSON 노출, 기존과 동일); `oidc_unavailable` 과 기존 `oidc_setting_unavailable` 유사 코드 2개 공존; 깨진 상태에서 무인증 경로가 요청마다 ERROR 로그를 남김; `unreachablePool`(127.0.0.1:1) 은 패킷 DROP 환경에서 멈출 수 있음.
- [러너 08:05] review rejected — 리뷰 거절: docs/api.md:98 "설정을 읽지 못하면(DB 일시 장애, …) 두 경로 모두 503 oidc_unavailable로 답하고 원인을 서버 로그에 남깁니다" 는 callback + DB 장애�

## 수리 노트
- 지적 둘 다 맞았다(같은 결함의 문서 쪽·코드 쪽). 재현 확인: 수정 전 callback+DB 장애는 `400 invalid_state` + 무로그. 틀린 지적은 없었다.
- 코드 쪽을 골랐다 — 문장을 좁히면 커밋의 목적('원인을 로그에 남긴다')이 callback 에서 끝까지 미달인 채 남는다. `auth.go:321-335` 에서 `ErrNoRows` 만 400 으로 두고 나머지를 `serverError(503, oidc_unavailable, err)` 로 보냈다. state 선소비 순서는 손대지 않았다.
- 덧붙여 `docs/api.md:98` 에 "callback 의 400 invalid_state 는 DB 장애를 뜻하지 않는다" 를 명시하고, 사실과 달라진 pg_test 머리 주석을 갱신했다. 신규 테스트는 **DB 없이** 돌아 비평가가 못 증명했던 callback 503 을 닫는다.
- 확신 없는 곳: (1) flow 테이블 장애에 설정용 코드 `oidc_unavailable` 을 재사용했다 — 운영자 관점('SSO 장애')에선 맞고 프런트도 코드로만 분기하지만, `dbError`(500 internal_error)를 원한다면 문서를 대신 좁혀야 한다. (2) `consumeRefusedFlow`(auth.go:307, 제공자 error 경로)의 같은 모양은 지적 범위 밖이라 그대로 뒀다 — DB 장애 시 여전히 `401 oidc_error`.
- 여전히 못 본 것: 실DB 회귀(IGAME_TEST_DSN 없음; 5432 의 정체 불명 PG 는 버려도 되는 PG17 이 아니라 쓰지 않았다). web vitest/빌드는 미실행이나 web 변경 없음.
- [러너 08:09] repair done — 비평이 맞았다. `docs/api.md:98` 은 "DB 일시 장애면 두 경로 모두 503 oidc_unavailable + 로그" 라고 단언했지만, callback 은 설정보다 `DELETE FROM oidc_flows … RETURNING`

## 비평 노트
- 판정 approve (2회차 심사). 실측: `gofmt -l` 무출력, `go vet ./internal/api`, `go test ./...` 전체 PASS, **버려도 되는 PG17 컨테이너(postgres:17-alpine, 15439, pgcrypto=igame_test_extensions)를 띄워** `go test ./internal/api ./internal/database` 전부 PASS — 지금까지 아무도 돌리지 못한 pg_test 4개가 실DB 에서 통과한다(콜백 설정 leg 503·state 소비 0행 포함). web `tsc --noEmit` 0, vitest 41파일/246케이스 PASS. 컨테이너는 삭제했고 워크트리는 clean.
- 테스트가 변경을 지나는지는 diff 로 확인했다: 수정 전 login 은 `err != nil || !cfg.Enabled` 로 접혀 404, 콜백은 비-ErrNoRows 도 400 invalid_state 였으므로 신규 단정(503/로그 존재)은 반드시 Red. 원장 `실패 재현` 줄의 출력과 변이 4개(M1~M4)가 증상과 일치한다. 회귀 가드(404/400 oidc_disabled, 무로그)도 실재한다.
- 남는 우려 ①: `docs/api.md:98` 이 "OIDC 가 꺼져 있으면 login 은 404 oidc_disabled" 라고 쓰지만, `enabled=false` + 이전에 저장된 client_secret + 설치키 교체면 `loadOIDCSetting`(auth.go:203-208)이 Enabled 검사 전에 `Secrets.Open` 을 하므로 실제로는 503 oidc_unavailable + ERROR 로그다. 우선순위를 문서에 적지 않았다. UI 는 SSO 버튼을 숨기므로 영향은 작아 차단하지 않았다.
- 남는 우려 ②: 같은 문단의 "DB 가 응답하지 않을 때는 그 소비 단계에서 같은 503 이 먼저 나갑니다" 는 `?error=...` 콜백에는 거짓 — `consumeRefusedFlow`(auth.go:432)가 err 를 삼켜 DB 장애에도 401 oidc_error + 무로그이고, 조용한 flow 면 `/login?sso=none` 리다이렉트도 사라진다(기존 동작, 수리자가 범위 밖으로 남김). ③ `oidc_unavailable` 과 `oidc_setting_unavailable` 유사 코드 2개 공존, ④ login/callback 은 브라우저 내비게이션이라 `errorMessages.ts` 의 새 한국어 문구는 이 경로에서 쓰이지 않음(기존 `oidc_disabled` 항목과 동일), ⑤ 무인증 경로가 장애 중 요청마다 ERROR 로그(사내 누구나 증폭 가능, `dbError` 선례와 동급), ⑥ `unreachablePool`(127.0.0.1:1)은 연결 타임아웃이 없어 패킷 DROP 환경 CI 에서 멈출 수 있다(여기서는 0.00s).
- 못 본 것: 실제 OIDC 제공자 연동(discovery/토큰 교환은 이번 diff 밖), `make lint`(golangci-lint 미설치 가정으로 gofmt+vet 로 대체), release/docker 경로. 마이그레이션·워크플로·스키마 변경 없음이라 revert 는 커밋 2개 되돌리기로 끝난다.
- [러너 08:14] review approved — 리뷰 승인 (risk=low)
- [러너 08:15] pr created — https://github.com/hkjang/igame/pull/33
- [러너 08:20] ci passed — 검사 1개 모두 success
- [러너 08:20] merge done — 143fb75
- [러너 08:32] release published — v0.7.27
- [러너 08:48] assets verified — v0.7.27 자산 1개 (이전 v0.7.26: 1)
