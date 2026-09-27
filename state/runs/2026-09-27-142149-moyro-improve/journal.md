# 회차 노트 2026-09-27-142149-moyro-improve — moyro
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:21] base pinned — main@c098d04
- [러너 14:21] autonomy release — 

## 정찰 노트
- 골랐다: 업로드 스텁 3개의 MaxBytesReader 초과 삼킴(413 + 감사 0건). 이 저장소가 6회 연속 채택한 "한 리소스의 여러 경로가 같은 장애에 다른 답을 한다" 유형이고, 세 핸들러가 compat_wave_handlers_final.go 한 파일에 모여 있어 프로덕션 파일 2개로 끝난다. 팀 이미지 하나만 고치는 옛 아이디어(가치 2)는 스텁이라 '효과 없는 정리' 로 몰릴 소지가 있어, 감사 로그 오염 + uploadChunk 의 거짓 file_offset 까지 묶어 실제 결함으로 세웠다.
- 제친 것: patchPost 되읽기(차선) — 주입 방법이 3회차 연속 미해결이라 45분 안에 끝날 보장이 없다. 북마크 IsMember·preferences 400 분리는 설계 결정 선행(M). 메일 관련 2건은 mail 패키지가 main 에 없어 선행 조건 미충족.
- 추측으로 적은 것(구현자가 확인할 것): `&handlers{}` 에 어떤 필드를 채우면 callerCanAdminTeam/callerIsSystemAdmin 이 통과하는지 미확인. 헬퍼를 request_body.go 에 둘지 핸들러 파일에 둘지는 구현자 판단. 새 오류 id 이름은 제안일 뿐이다.
- 이번 세션에 DSN 이 없어 DB 회귀는 하나도 돌려 보지 못했다 — 413/감사-0건은 코드 독해 근거이며 RED 확인은 구현자 몫이다. 컴파일(`go test -run TestUploadStubNothing ./internal/httpapi/`)만 확인했다.
- 조심할 것: `api.user.image.too_large` 를 재사용하지 말 것(handlers.go:891 이 400 에 쓴다). 테스트 라우트 패턴은 router.go 에서 글자 그대로 복사할 것 — `{botID}` 로 편하게 등록하면 기준 5 의 버그가 가려진다. 상한값(10MB/256KB/50MB)과 uploadTeamImage 의 게스트 비대칭은 건드리지 말 것.
- [러너 14:26] scout done — 업로드 스텁 3개가 본문 상한 초과를 삼키고 200 + 거짓 감사 기록을 남기는 것 차단 (가치 3 / 위험 1 / 작업�

## 구현 노트
- 무엇/왜: `96c9c5e` — 세 업로드 스텁(`uploadTeamImage`/`uploadBotIcon`/`uploadChunk`)이 `MaxBytesReader` 초과 오류를 버려 200 + 거짓 감사 행 + 거짓 `file_offset` 을 내던 것을, `request_body.go` 의 새 `drainCappedBody(w,r,limit,errID)` 로 413 + 감사 호출 전 early-return 으로 바꿨다. 곁가지로 `uploadBotIcon`/`deleteBotIcon` 의 `chi.URLParam(r,"botID")` → `"botUserID"`(라우트 선언과 일치).
- 확신 없는 곳: (1) 헬퍼의 **비-MaxBytes 읽기 오류 → 400** 분기는 `decodeCollectionBody` 와의 대칭으로 넣었을 뿐, 실제로 그 분기를 타는 케이스는 테스트하지 못했다(클라이언트 중도 절단을 httptest 로 결정론적으로 만들지 않았다). 그 경로에서는 응답 자체가 대개 무의미하다. (2) 새 오류 id 3개를 소비하는 클라이언트는 없다 — webapp/docs/OpenAPI 에 이 세 엔드포인트 참조가 0건인 것을 grep 으로 확인했으나, 외부 공식 클라이언트가 413 을 어떻게 표시하는지는 실기 확인 못 했다. (3) 실제 브라우저 e2e·플러그인 archive 시나리오 미실행.
- 일부러 안 한 것: `uploadTeamImage` 의 `denyGuestMutation` 비대칭(게스트 권한 모델 결정 선행), 상한값(10MB/256KB/50MB) 조정, `internal/httpapi` 의 나머지 `MaxBytesReader` 호출자 5곳(계약이 달라 일괄 확대하면 파일 수가 늘고 경로가 섞인다). 모두 ideas.json 에 남겼다.
- 다음 역할이 조심할 것: 신규 `upload_stub_limits_postgres_test.go` 는 **DSN 필수**다 — `MOYRO_TEST_POSTGRES_DSN` 없이는 통째로 skip 되므로 `ok` 만 보고 통과라고 하지 말 것. 이번 회차는 로컬 컨테이너 `moyro-pg-improve`(포트 **55433**, postgres:16-alpine, moyro/moyro/moyro)로 돌렸다. 테스트는 마지막에 LogAsync 의 3초 DB 타임아웃을 넘겨 감사 행 수를 감시하므로 단독 실행에 ~4초, 초과 케이스 포함 전체가 ~4-11초다. 감사 target 이 빈 문자열이면 `audit.Log` 의 `nullIfEmpty` 때문에 DB 에는 **NULL** 로 들어간다(진단 쿼리를 쓸 때 `target=''` 로 찾으면 안 보인다). 그리고 roles 는 콤마가 아니라 **공백 구분**(`auth.splitRoles`)이다 — 콤마로 심으면 403 이 나서 RED 를 오해하게 된다.
- [러너 14:36] brief accepted — 채택 — 과제서의 근거(세 핸들러의 `io.Copy` 오류 무시 + 감사 호출 + 200, `{botUserID}` vs `"botID"` 불일치, 재사용 금지 id)가 현
- [러너 14:37] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- 판정 approve(risk low, blocking 없음). 확인: 프로덕션 diff 2파일 전체, request_body.go 의 신규 `drainCappedBody`, router.go:904/957-959/977 의 파라미터 이름, 신규 테스트 260줄 전체, scripts/check-source-sizes.sh.
- 검증 게이트는 DSN 없이 5초에 끝나 신규 테스트를 skip 했다 — 직접 DSN(55433)으로 재현했다: main + 신규 테스트 = 5개 FAIL(413 대신 200 + `file_offset:52428800`, team.image 감사 2행, bot icon/delete target NULL), HEAD = 전부 PASS. `./internal/httpapi/` 전체 실DB 37.6s ok, vet clean. 원장의 `- 실패 재현:` 줄과 내 출력이 일치한다.
- 승인이어도 남는 것: (1) 신규 id 3개가 `too_large` 라는 이름으로 비-크기 읽기 오류의 400 에도 나간다(소비자 0건이라 결함으로 세우지 않음), (2) `n == limit` 경계 케이스 미커버, (3) `deleteBotIcon` 의 botID→botUserID 는 곁가지지만 옳고 테스트가 잡는다 — 릴리즈 노트에 "봇 아이콘 감사 로그가 대상 봇을 기록" 을 따로 적을 것.
- 못 본 것: 실브라우저 e2e, 플러그인 archive 시나리오, 공식 Mattermost 클라이언트가 413 을 화면에 어떻게 표시하는지.
- 다음 회차용: 같은 유형(MaxBytesReader 초과 삼킴)이 `internal/httpapi` 에 5곳 더 남아 있고 `uploadTeamImage` 의 게스트 비대칭도 그대로다 — 둘 다 ideas.json 에 있다.
- [러너 14:41] review approved — 리뷰 승인 (risk=low)
- [러너 14:41] pr created — https://github.com/hkjang/moyro/pull/27
- [러너 14:54] ci passed — 검사 3개 모두 success
- [러너 14:54] merge done — 96c9c5e
