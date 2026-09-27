# 회차 노트 2026-09-28-043226-ReSSO-improve — ReSSO
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 04:32] base pinned — main@44acfdd
- [러너 04:32] autonomy release —

## 정찰 노트
- 고른 이유: `InspectRefreshToken` 호출자 셋(refresh 그랜트 oidc.go:576 · introspection 1016 · revocation 1106) 중 refresh 그랜트만 "읽지 못함"과 "없음"을 400 `invalid_grant`로 합친다 — 나머지 둘은 이미 `ErrNotFound`를 갈랐고, 같은 논지가 두 줄 아래 597행(`userLookupFailed`, v0.9.94)에 있다. 최근 세 회차가 이 부류로 전부 채택·릴리즈됐고 프로덕션 파일 1개로 끝난다. 차선인 `resso_userinfo_errors_total`(2/1/S)은 가치가 낮고, CSP 좁히기는 외부 표준 원문 미확인이라 7회째 제쳤다.
- 확인한 것(추측 아님): `InspectRefreshToken`(internal/store/oidc.go:348)이 `pgx.ErrNoRows`에만 `ErrNotFound`를 돌리고 나머지는 원본 오류를 그대로 돌린다 / `refresh_tokens`를 참조하는 외래키는 자기 참조 하나뿐(001_initial.sql:168)이라 RENAME이 안전하다 / 지표 단언 배선은 `observability.NewRegistry()` → `New(data, logger, nil, metrics)` → `WritePrometheus`(선례 integration_test.go:2199·2229)이고, 베낄 형제 테스트 1130행은 `nil` 등록기라 그대로 베끼면 지표를 못 읽는다.
- 추측으로 남긴 것: 500 본문 문안(권장안만 제시 — 2026-09-27 정찰이 "토큰이 유효하다고 말할 근거가 약하다"며 이 과제를 보류했으므로, "검증됐다"가 아니라 "회전·거절되지 않았다"로 쓰라고 지정해 해소했다) / `docs/operations.md` 불릿에 지난 두 회차 문장과 중복이 생기는지는 열어 보지 않았다(구현자가 읽고 판단할 것).
- 구현자가 조심할 것: 577행의 불일치 조건식과 `_`로 버려지는 `active` 반환값은 **건드리지 말 것**(만료·회전·취소 판정은 뒤의 `RotateRefreshToken`이 하며, 여기서 바꾸면 만료 토큰의 현재 계약이 바뀐다 — ideas.json에 별건으로 남겼다). 세 경로를 하나의 헬퍼로 통합하지 말 것(답이 500/200/503으로 계약이 다르다). 호출자가 한 곳이니 새 헬퍼 대신 인라인 분기. RENAME은 `t.Cleanup` 복구 필수. `raw` 토큰 값은 로그·지표에 넣지 말 것.
- [러너 04:38] scout done — refresh 그랜트가 `refresh_tokens` 조회 장애를 `invalid_grant`로 답하는 것을 500 `server_error`로 (가치 3 / 위험 2 / 작�

## 구현 노트
- 무엇을 왜: `handleRefreshGrant`(oidc.go:576)가 `InspectRefreshToken`의 모든 오류를 400 `invalid_grant`로 합쳐, `refresh_tokens`를 읽지 못한 것이 "그런 토큰 없음"과 같은 답이 됐다. 577행 **앞**에 인라인 분기를 두어 `store.ErrNotFound`가 아닌 오류만 500 `server_error` + `resso_token_errors_total{grant_type="refresh_token"}` + ERROR 한 줄로 갈랐다. 프로덕션 파일 1개(oidc.go), 그 외 테스트와 docs/operations.md 한 문장.
- 확신 없는 곳·검증 못 한 것: 500 본문 문안은 과제서의 권장안 그대로다(문구 심사는 비평가 몫). 이 실패가 `writeOAuthError`를 쓰므로 `Cache-Control` 등 헤더는 기존 500들과 동일할 것이나 헤더를 따로 단언하지는 않았다. `docs/operations.md` 불릿은 이제 아주 길다 — 읽히는지는 판단하지 않았고 쪼개지도 않았다.
- 일부러 하지 않은 것: 577행의 불일치 조건식과 `ErrNotFound` 경로, `_`로 버려지는 `active` 반환값은 손대지 않았다(만료 토큰의 현재 400 계약이 바뀐다 — ideas.json에 별건). introspection(200 active=false)·revocation(503)도 무변경이고 헬퍼로 통합하지 않았다(계약이 다르고 호출자가 한 곳이라 인라인). 감사 항목도 더하지 않았다(이 시점에 사용자·세션이 특정되지 않는다). ADMIN_GUIDE.md는 이미 맞아 그대로, PDF·캡처 재생성 없음.
- 다음 역할이 조심할 것: 새 테스트는 실제 PostgreSQL이 필요하다 — `eval "$(scripts/test-services.sh)"`를 **같은 셸에서** 먼저 하고, 출력이 비면 eval 하지 말 것(없으면 조용히 SKIP되고 exit 0이다). 테스트는 `ALTER TABLE refresh_tokens RENAME`으로 장애를 만들고 `t.Cleanup`으로 되돌린다 — 그 정리를 지우면 컨테이너를 공유하는 뒤따르는 테스트가 전멸한다. 지표를 읽으려면 서버를 `New(data, logger, nil, metrics)`로 세워야 한다(`nil` 등록기로는 못 읽는다). `make test`가 `webui/dist/index.html`을 바꾸므로 커밋 전 복원했다.
- [러너 04:50] brief accepted — 채택 — 근거(576-580행이 `ErrNotFound` 구분 없이 400 `invalid_grant`, `InspectRefreshToken` 호출자 셋 중 유일, 597행에 같은 논지가 이�
- [러너 04:50] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: 테스트를 main 코드에서 돌려 integration_test.go:1456 실패(400 invalid_grant — 이번 변경이 고치는 바로 그 증상)를 직접 재현했고, 되돌린 뒤 PASS·`Refresh|Token` 전부 통과·gofmt/vet 깨끗. 지표 배선(`New(…, nil, metrics)`), `t.Cleanup` 복구, 테스트별 전용 schema, raw 토큰 비노출, 불일치 분기 무변경을 모두 파일에서 확인.
- 거절 사유는 하나뿐이고 동작이 아니라 oidc.go:578 주석의 사실 오류다: "The last place in this endpoint" 가 거짓이다 — 같은 endpoint 에 oidc.go:665(RotateRefreshToken 의 저장소 장애), store/oidc.go:310(세션 질의 장애가 ErrNotFound 로 둔갑), oidc.go:530(코드 상환 장애)이 아직 400 invalid_grant 다. 이 저장소는 주석이 "왜"의 정본이라 다음 정찰이 이 문장을 읽고 남은 셋을 목록에서 지운다.
- 수리가 가장 먼저 볼 파일: internal/httpserver/oidc.go:578 — 그 한 절만 사실에 맞게 좁히면 된다(코드·테스트·docs 는 손댈 것 없음). 남은 셋은 이번 범위가 아니므로 ideas.json 별건으로.
- 못 본 것: `make lint`(golangci-lint PATH 미확인)와 vitest/프런트, `make test` 전체 SKIP 집계 경고. docs/operations.md 불릿 길이는 판단하지 않았다(가독성은 취향 영역으로 두었다).
- 승인 뒤에도 남는 우려: 이 worktree 에 webui/dist/index.html 이 커밋 없이 수정된 채 남아 있다 — 릴리즈 빌드 전에 복원할 것.
- [러너 04:54] review rejected — 리뷰 거절: internal/httpserver/oidc.go:578 주석이 "The last place in this endpoint where a fault on this side was still answered as a spent grant" 라고 단언하지만 사실이 아니다. �

## 수리 노트
- 지적은 맞았다. 세 자리를 직접 열어 확인했다: oidc.go:530(RedeemAuthorizationCode 의 모든 실패 → 400), oidc.go:665(ErrTokenReuse 아닌 모든 것 → 400, store/oidc.go:281·318·324·340·343 의 순수 저장소 장애가 여기로 온다), store/oidc.go:310(`err != nil || !sessionActive` → ErrNotFound). 틀린 지적은 없었다.
- 고친 방법: oidc.go:578 첫 단락만 교체 — "the earliest … and not the last of them" 으로 좁히고 남은 두 자리를 이름으로 적어 "separate changes and are still to make" 라고 명시했다. 코드·테스트·docs 는 손대지 않았다(커밋 a3d0e56).
- 검증: gofmt/vet/build 깨끗, 대상 테스트 PASS, `-run 'Refresh|Token'` 전부 PASS(실제 PostgreSQL). 주석만 바뀌었으므로 동작 회귀는 없다.
- 비평 노트의 우려대로 커밋 안 된 webui/dist/index.html 을 복원했다. 커밋에는 oidc.go 만 들어갔다.
- 여전히 확신 없는 곳: `make lint`(golangci-lint PATH 미확인)와 vitest/프런트는 이번에도 돌리지 않았다 — 주석 한 단락이라 위험은 낮다고 판단했다. 남은 세 자리는 ideas.json 에 별건으로 남길 일이며 이번 범위에서 손대지 않았다.
- [러너 04:58] repair done — - 문제: oidc.go:578 주석 첫 절이 이 자리를 "The last place in this endpoint where a fault on this side was still answered as a spent grant" 라고 단언했다. 거짓이다 — 같은 endp
