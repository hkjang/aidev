# 회차 노트 2026-10-17-125842-orbit-improve — orbit
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:58] base pinned — main@5c97a1a
- [러너 12:58] autonomy release — 

## 정찰 노트
- 고른 이유: `looksLikeUUID` 가드가 data.go·workflow.go·ai.go 9곳에 있는데 `personal.go` 에만 0곳이라, `revokeAPIKey` 는 모양 어긋난 경로 값을 uuid 컬럼(`api_keys.id`)에 그대로 넘겨 호출자 실수를 500 + slog 에러로 만든다 — 이 저장소가 세 회차 연속 채택한 바로 그 패턴이고 프로덕션 1파일로 끝난다. 같은 결함이 있는 `updateKeyPermission`·`updateUser`·`reviewApproval` 은 관리자·승인 보호 경로라 제쳤고, ai_test.go 계열 두 후보는 미머지 `8ac45f8` 과 같은 파일이라 충돌 위험으로 제쳤다.
- 추측으로 적은 것: 모양 어긋난 `keyID` → postgres 22P02 → 500 은 코드 경로와 과거 회차 실측 기록에서 추론한 것이고 이번에 실 DB(DSN 없음)로 재현하지 않았다. 차선의 `updatePreferences` 0행 UPDATE 가 실제로 도달하는지도 미확인(행을 심는 곳이 세 군데).
- 구현자가 조심할 것: 404 메시지는 기존 `RowsAffected()==0` 리터럴("활성 API 키를 찾을 수 없습니다.")과 바이트로 같게 두고 `assertAPIErrorMessage` 로 고정할 것. `looksLikeUUID` 본문·시그니처와 공유 SQL 은 건드리지 말고, 공백·중괄호 입력은 경로만 `url.PathEscape` 하고 라우트 파라미터에는 날값을 넣을 것.
- 프로필은 8일 전 것이고 pinned HEAD(5c97a1a)·파일 구성·미머지 두 커밋(a5de472·8ac45f8) 기록이 이번 확인과 어긋나지 않아 새로 쓰지 않았다.
- [러너 13:02] scout done — 개인 API 키 폐기(`revokeAPIKey`)가 uuid 모양 아닌 `keyID` 를 uuid 컬럼에 그대로 넘겨 호출자 실수를 500 internal_err

## 구현 노트
- `personal.go:revokeAPIKey` 의 `chi.URLParam(r,"keyID")` 와 `s.store.DB.Exec` 사이에 `looksLikeUUID` 가드 5줄(주석 1줄 포함)을 넣었다 — `api_keys.id` 가 uuid 컬럼이라 모양 어긋난 경로 값이 22P02 → 500 `internal_error` + `slog.Error` 가 되던 것을 형제 핸들러와 같은 404 `not_found` 로 끝낸다. 404 리터럴은 기존 `RowsAffected()==0` 분기와 바이트로 같다. 커밋 57aaa97, 프로덕션 1파일 + 신규 시험 1파일.
- **확신 없는 곳·검증 못 한 것**: 실 postgres 에서 22P02 → 500 이 나는 것을 이번에 재현하지 않았다(DSN 없음) — 코드 경로(uuid 컬럼 + `internalError`)와 과거 회차 실측 기록에 따른 추론이다. 가드가 **없던** 상태에서 nil store 패닉으로 "DB 질의까지 도달한다" 는 것까지는 실행으로 확인했고, 그 다음 단계(postgres 가 실제로 캐스팅 오류를 낸다)만 미실측이다. `revokeAPIKey` 의 정상 경로(204 + audit 행 기록)도 DB 가 없어 실행하지 않았다 — SQL·인자·`u.ID` 조건·audit 호출을 한 글자도 바꾸지 않았다는 것이 근거다.
- **일부러 하지 않은 것**: 같은 결함이 있는 `updateKeyPermission`(personal.go:384)·`updateUser`(settings.go:303)·`reviewApproval`(workflow.go:332) 은 관리자·승인 보호 경로라 손대지 않았다(ideas.json 에 남김). `looksLikeUUID` 의 판정·시그니처는 9개 호출자가 공유하므로 그대로 뒀고, 기존 404 문구를 새로 짓지도 않았다. 차선 후보(`updatePreferences` 0행 UPDATE)는 1순위가 성립했으므로 건드리지 않았다.
- **다음 역할이 조심할 것**: 신규 `internal/server/personal_test.go` 는 **DB 가 필요하지 않다** — `callWithoutStore`(workflow_test.go:18) + `&Server{store: nil}` 로 프로덕션 핸들러를 직접 부르고, CI 에서 실제로 돈다. `TestRevokeAPIKeyLetsWellFormedKeyIDReachDB` 는 "패닉이 나야 통과" 하는 역방향 시험이다(= 가드를 지나 DB 까지 갔다는 증거) — 패닉을 없애려고 store 를 채우면 그 시험의 뜻이 사라진다. 경로에 공백·중괄호가 든 입력은 `url.PathEscape` 로만 escape 하고 라우트 파라미터에는 날값을 넣었다(핸들러가 읽는 것은 후자다) — 둘을 같게 맞추면 `httptest.NewRequest` 가 패닉한다.
- [러너 13:06] brief accepted — 채택 — 지정한 자리(`personal.go:341` 의 무검사 `chi.URLParam`, `api_keys.id uuid`, `looksLikeUUID` 호출자 9곳 대 personal.go 0곳, `deletePers
- [러너 13:06] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 가드 5줄만 떼고 `go test`를 돌려 시험이 정말 변경을 잡는 것을 확인했다 — malformed 4개 서브시험이 구현 노트가 말한 증상 그대로 실패하고, 되돌리니 통과. 작업 트리 clean 복구 확인. `gofmt -l internal/server`·`go vet ./...` 무출력, `go test -race -count=1 ./...` 전부 ok. 라우트 파라미터 이름(`server.go:70` `{keyID}`)·404 리터럴 동일성·`user_id=$2` 유지·형제 `deletePersonLink`와의 일관성도 파일을 열어 대조했다.
- 못 본 것: 실 postgres 22P02 → 500(DSN 없음)과 정상 폐기 경로(204 + audit 행). 다만 전제가 틀려도 결과는 0행 → 같은 404이므로 승인 근거가 미검증 전제에 기대지 않는다. 프런트(`web/`)는 이 변경과 접점이 없어 돌리지 않았다.
- 승인이어도 남는 우려 ①: `TestRevokeAPIKeyLetsWellFormedKeyIDReachDB`는 수정 전후 모두 통과하는 "패닉=통과" 그물이고, 질의 앞의 다른 이유로 패닉해도 초록이 된다. 이번엔 패닉 출처가 nil store의 `s.store.DB`인 것을 확인했다(`userFromContext`는 값 타입 `User` 반환).
- 승인이어도 남는 우려 ②: 구현 노트는 `updateKeyPermission`·`updateUser`·`reviewApproval` 셋을 ideas.json에 남겼다고 적었지만 실제로는 앞의 둘만 있다 — 다음 정찰이 ideas.json만 보고 고르면 `reviewApproval`을 놓친다.
- 릴리즈 노트용: 사용자 눈에 보이는 변화는 `DELETE /api/v1/personal/api-keys/{keyID}`가 모양 어긋난 id에 500 대신 404를 주는 것뿐. 보안·법무 차단 사유 없음.
- [러너 13:09] review approved — 리뷰 승인 (risk=low)
- [러너 13:09] pr created — https://github.com/hkjang/orbit/pull/25
- [러너 13:12] ci passed — 검사 1개 모두 success
- [러너 13:12] merge done — 57aaa97
- [러너 13:17] release published — v0.7.13
- [러너 13:18] assets verified — v0.7.13 자산 1개 (이전 v0.7.12: 1)
