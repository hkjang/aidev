# 회차 노트 2026-10-07-075840-ReSSO-improve — ReSSO
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:58] base pinned — main@52e1d49
- [러너 07:58] autonomy release — 

## 정찰 노트
- 골랐다: 코드 재사용 폐기 실패에 `ErrFamilyNotRevoked` 를 붙이기. 아홉 회차를 이어 온 Client 인증 캠페인이 v0.9.100 으로 끝나 새 줄기가 필요했고, 이 과제는 **형제 경로에 답이 이미 구현돼 있다**(store/oidc.go:295·299 + httpserver/oidc.go:687-705) — 설계 결정이 남아 있지 않아 위험이 낮다. 보류 2년차 지표 배선(2/1/S)은 권고대로 rejected 로 내렸고, refresh 500(3/3/M)은 store 변경이 필요해 제쳤다.
- 확신 없는 곳: 컬럼 이동(`revoked_at` → `revoked_at_moved`)이 재사용 분기의 UPDATE 만 깨뜨린다는 것은 SQL 을 읽어 추론한 것이고 **실행으로 확인하지 않았다**(첫 교환의 refresh token INSERT 가 `revoked_at` 을 열거하지 않는 것은 store/oidc.go:336-339 로 확인). docs/operations.md 에 `AUTHORIZATION_CODE_REUSED` 항목이 있는지도 미확인 — 구현자가 먼저 grep 할 것.
- 조심할 것 ①: store 쪽에서 오류를 감쌀 때 `code` 를 **함께** 돌려줘야 한다. 지금 171·174 는 zero value 를 돌려주는데 호출부가 `code.SessionID.String()` 을 쓰므로 nil `*uuid.UUID` 로 panic 한다.
- 조심할 것 ②: 이 분기는 사람이 받지 않은 `5bed9dc`(같은 함수에 500 `server_error`)와 **같은 자리**다. 400 `invalid_grant` 응답을 한 글자도 바꾸지 말 것 — 바꾸면 기각된 접근의 재제출이 된다.
- [러너 08:04] scout done — 코드 재사용 감지의 폐기 실패가 `AUTHORIZATION_CODE_REUSED` 감사 항목을 통째로 잃는 것을 막기 — `RotateRefreshTo

## 구현 노트
- 바꾼 것: `store/oidc.go` 재사용 분기가 폐기 `Exec`·`Commit` 실패를 `ErrCodeReuse` + `ErrFamilyNotRevoked` 로 감싸고 **zero value 대신 `code`** 를 함께 돌려준다. `httpserver/oidc.go:554` 분기는 detail 에 `tokens_revoked:false` 를 더하고 사건을 **한 줄만** 남긴다. 응답은 400 `invalid_grant` 그대로(테스트가 본문 문안까지 단언). 프로덕션 파일 2개. 커밋 87cb26a.
- 과제서와 다르게 한 것 ①: 로그를 Warn+ERROR 두 줄이 아니라 **한 줄**로 했다 — 폐기 성공은 기존 Warn `authorization code replayed`, 실패는 ERROR `authorization code replayed but the tokens it should have revoked were not`. 같은 구절로 시작하므로 `authorization code replayed` grep 이 양쪽을 찾는다. 두 줄로 내는 것이 더 낫다고 보면 되돌리기 쉬운 자리다(`else` 를 지우면 끝).
- 과제서와 다르게 한 것 ②: 과제서의 "zero value 면 nil `*uuid.UUID` 역참조로 panic" 은 **틀렸다** — `UserID`·`SessionID` 는 포인터가 아닌 `uuid.UUID` 값이다. `code` 반환은 그래도 필요하고, 프로브로 실제 증상을 확인했다: `actor_id=uuid.Nil` 이 `audit_events` FK 를 거부해 감사가 **또 다른 이유로** 사라진다.
- 확신 없는 곳: `Commit` 실패 경로(179행)는 `Exec` 경로와 같은 코드지만 **테스트로 따로 재현하지 않았다** — 컬럼 이동으로는 `Exec` 만 깨진다. `make lint` 는 docs 변경 **전**에 돌렸다(Go 파일은 그 뒤로 안 건드렸으므로 결과는 유효하다).
- 일부러 안 한 것: `oidc.go:557` 의 `UserByID` 오류를 `_` 로 버리는 것(같은 분기라 수용 기준이 흐려진다), `metrics.go`(새 계열 없음), `612-618` 주석 문단(코드 재사용 쪽을 지목하지 않는다).
- 다음 역할이 조심할 것: 새 테스트는 **DB 가 있어야 돈다**(`eval "$(scripts/test-services.sh)"` 를 같은 셸에서). `make test` 첫 실행은 `internal/federation` 이 `/tmp/resso-test-certs/ca.crt` 없음으로 깨졌다 — 이 변경과 무관하며 `scripts/test-services.sh --stop` 후 재생성으로 통과했다(ideas.json 에 후보로 적었다). `webui/dist/index.html` 은 복원했다.
- [러너 08:23] brief accepted — 채택 — 근거(166-177 이 raw 오류를 돌려 `errors.Is(ErrCodeReuse)` 를 거짓으로 만드는 것, 형제 경로 295·299 + 687-705 에 답이 완성�
- [러너 08:23] verify passed — 검증 7개 통과 (auto)
