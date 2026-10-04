# 회차 노트 2026-10-04-190204-moyro-improve — moyro
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:02] base pinned — main@c51868d
- [러너 19:02] autonomy release — 

## 정찰 노트
- `bulkDeleteUsers` 를 골랐다: `_ = decodeCappedBody`(final.go:1114) 가 상한 초과를 삼켜 쓰기 배치가 200 `count:0` + 감사 0행으로 끝나는데, `request_body.go:26-30` 이 "쓰기 배치는 거부한다" 고 스스로 적어 둔 계약과 모순된다. 2026-09-27 의 같은 계열(업로드 스텁 상한 삼킴)이 채택·release-ready 로 끝난 성공 선례가 있다.
- 1순위 후보였던 `inviteGuestsByEmail` 게스트 가드는 기각했다 — 결함은 실재하지만(final.go:1146 에 `denyGuestMutation` 없음), 같은 접근의 2026-09-29 회차가 머지되지 않았음을 `uploadTeamImage`(final.go:134)에 가드가 **여전히 없는 것**으로 확인했다. 운영자 규칙상 재제출 금지. `createIncomingWebhook` 404→500 도 미머지 커밋 0b2d41c 때문에 또 피했다.
- 과제서의 추측(미확인): 이 파일에 `io` import 가 있는지, 본문 없는 `DELETE /api/v4/users` 호출자가 실제로 있는지, 차선 후보의 `customProf.CreateField("","",nil)` 응답 코드.
- 구현자가 조심할 것: `decodeCollectionBody` 로 **그냥 치환하면 안 된다** — 빈 본문은 `io.EOF` 라서 오늘의 200 이 400 으로 바뀐다. 수용 기준 3 이 그 지점을 고정한다. 나머지 `_ = decodeCappedBody` 16곳은 의도된 관용이므로 범위를 넓히지 말 것.
- 곁가지: router.go 중복 등록 회귀 가드는 런타임으로 불가해 기각했다(chi 가 재등록 시 핸들러를 교체하므로 완성된 트리에 중복이 남지 않고 `chi.Walk` 로 셀 수 없다). 프로필은 5일 전 것이 현 코드와 어긋나지 않아 다시 쓰지 않았다.
- [러너 19:07] scout done — `bulkDeleteUsers` 가 상한 초과·깨진 본문을 삼키고 200 `{"status":"OK","count":0}` 으로 "삭제 성공" 을 보고하는 것 

## 구현 노트
- `bulkDeleteUsers`(final.go:1114)의 `_ = decodeCappedBody` 를 신규 `decodeOptionalCollectionBody`(request_body.go)로 바꿨다 — 상한 초과 413 / 깨진 본문 400 `api.user.bulk_delete.invalid_body`, 단 `json.Decode` 의 `io.EOF`(본문 없는 요청)만 통과시켜 오늘의 200 `count:0` 을 보존한다. 거부는 감사 루프보다 앞.
- 과제서의 권장 코드를 그대로 넣지 않고 헬퍼로 뺀 이유: 이 핸들러 파일에 `io` import 가 **없다**(`errors` 는 있다). `io`·`fmt`·`errors` 를 이미 쓰는 `request_body.go` 에 분기를 두면 새 import 가 없고 형제 `decodeCollectionBody` 와 나란히 읽힌다. 과제서가 허용한 형태다. 기존 네 헬퍼의 동작은 한 글자도 안 바꿨다.
- **확신 없는 곳**: (1) 본문 없는 `DELETE /api/v4/users` 호출자가 실제로 있는지는 여전히 **미확인**이다 — 안전한 쪽(200 유지)을 택해 서브테스트로 고정했을 뿐이다. (2) 새 id `api.user.bulk_delete.invalid_body` 에 대해 — `webapp/src`·`webapp/e2e`·`docs` 에 `bulk_delete` 문자열 소비자가 0건임은 확인했지만(문자열 부재는 '소비자 없음' 의 근거로만 썼다), 이 엔드포인트의 선언된 대상은 **저장소 밖 관리자 도구 통합**이라 그쪽이 413/400 을 어떻게 다루는지는 확인할 방법이 없다. 거짓 200 보다는 낫다는 판단이다. (3) 비관리자 403 은 실제 `r.Group` 배선에서 `requireRole` 미들웨어가 먼저 잡으므로 메시지가 핸들러의 "system_admin required" 가 아니라 "missing role: system_admin" 이다 — 그래서 테스트는 id 만 단언한다. 둘 다 수정 전후 동일하므로 계약 변경은 아니다.
- **일부러 하지 않은 것**: 나머지 `_ = decodeCappedBody` 14곳(+ customprofile 2곳). `request_body.go:52-56` 이 그 관용을 의도라고 명시하고, `bulkDeleteUsers` 만 쓰기 배치라 계약과 모순됐다. 차선 후보도 같은 PR 에 넣지 않았다. webapp·마이그레이션·`router.go`·`.github/workflows` 무변경.
- **다음 역할이 조심할 것**: 새 테스트는 DB 가 있어야 돈다 — `docker start moyro-pg-improve`(55433) + `MOYRO_TEST_POSTGRES_DSN` 없으면 **skip** 되니 `ok` 를 통과로 오인하지 말고 `-v` 로 확인할 것. `a_refused_batch_leaves_no_audit_row` 는 `LogAsync` 의 3초 창을 넘기려고 3.2초 폴링하므로 이 테스트만 4.3초가 정상이다. `gofmt -l server/internal` 은 이번 변경과 무관한 기존 9파일을 출력한다(변경 3파일은 clean).
- [러너 19:22] brief accepted — 채택 — 과제서의 근거가 현재 코드와 정확히 일치했고(1114 의 `_ =`, `decodeCappedBody` 가 응답을 쓰지 않는 것, `request_body.go:2
- [러너 19:23] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- 원장에 `- 실패 재현:` 줄이 없어 직접 RED 를 재현했다: `main` 에 테스트 파일만 복사한 임시 worktree 에서 4개 서브테스트가 `status = 200, want 413/400 (body {"count":0,"status":"OK"})` 로 실패하고 대조군 4개(본문없음·정상배치·too_many·비관리자 403)는 통과 — 증상이 이번 수정이 고치는 것과 정확히 일치. 임시 worktree 는 제거했다.
- 실제 DB(moyro-pg-improve:55433)로 확인: 새 테스트 8/8 통과(4.25s, `a_refused_batch_leaves_no_audit_row` 3.23s 정상), `go test ./internal/httpapi` 전체 통과(49.3s — skip 아님), `go build ./...`·`go vet ./...`·`check-source-sizes.sh` 통과, 변경 3파일 gofmt clean.
- 보안·법무 소견: 차단 없음. 새 거부는 `requireRole("system_admin")` + `callerIsSystemAdmin` **뒤**, 감사 루프 **앞**이라 권한을 넓히지 않고 새 식별자·비밀값·의존성·개인정보 수집이 없다. 400 본문의 `err.Error()` 는 요청자 자신의 입력 1글자 수준이고 기존 `decodeCollectionBody` 와 동일 패턴. 되돌리기도 코드 3파일 revert 로 끝난다.
- 남는 우려(릴리즈 노트에 적을 것): **관측 가능한 API 동작 변경**이다 — `POST /api/v4/users/bulk_delete` 및 `DELETE /api/v4/users` 가 상한 초과 시 200→413, 깨진 본문에 200→400 `api.user.bulk_delete.invalid_body` 를 돌려준다. 저장소 안 소비자는 0건임을 직접 확인했으나(`docs/`·`webapp/src`·`webapp/e2e` 에 `bulk_delete` 문자열 없음) 저장소 밖 관리자 도구는 확인 불가.
- 사소한 흠(차단 아님): `request_body.go:54-56` 이 본문없음 관용을 "a published contract" 라 적었는데 실제로 `docs/` 어디에도 공표된 바 없고 구현 노트 스스로 "의존 호출자 미확인" 이라 한다 — 다음 회차가 이 주석을 근거로 쓰지 않도록 "미확인 기존 동작이라 보수적으로 보존" 으로 완화하면 좋다. 또 쓰기 배치 7곳 중 여기만 본문없음을 허용하는 유일한 예외가 되었다(나머지는 모두 엄격한 `decodeCollectionBody`).
- [러너 19:27] review approved — 리뷰 승인 (risk=low)
- [러너 19:27] pr created — https://github.com/hkjang/moyro/pull/33
- [러너 19:35] ci failed — 성공이 아닌 검사: verify=failure · 실패한 검사: ? 잡: verify 
