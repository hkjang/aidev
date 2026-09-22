# 회차 노트 2026-09-22-200440-moyro-improve — moyro
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 20:04] base pinned — main@960d3a1
- [러너 20:04] autonomy release — 

## 정찰 노트
- getPreferenceByName 을 골랐다: 직전 두 회차가 채택된 것과 똑같은 패턴(오류 원인별 상태 코드 분리 + 실제 DB 회귀)이고, 서비스가 이미 pgx.ErrNoRows 를 구분해 주므로 핸들러 한 곳만 고치면 된다. 재정렬 건은 이미 origin/auto/2026-09-21-0304 에 구현돼 있어 제외했고, 메일 후속은 main 에 mail 패키지가 없어 제외했다.
- 옛 브랜치 auto/2026-09-07-0240 은 origin 에 푸시된 적이 없어 PR 도 반려도 없다는 것을 확인했다. 다만 그 브랜치가 들고 있는 값 상한·ci.yml·upsert/delete 500 분리는 이번 범위에서 뺐다(읽기 경로 한 곳만).
- 추측으로 적은 것: preferences 테이블의 정확한 컬럼 구성(서비스 쿼리 기준 4개만 확인, 마이그레이션 원문 미확인)과 500 응답의 오류 id 선택지. 구현자는 테스트 fixture 를 넣기 전에 마이그레이션을 확인할 것.
- 미검증: 실제 DB(DSN 미설정으로 httpapi DB 테스트 전부 skip), 전체 go test, 웹.
- 조심할 것: 404 본문의 오류 id·메시지를 바꾸지 말 것(클라이언트 계약). early.go 는 소스 크기 상한에 여유가 적으니 테스트는 새 파일로.
- [러너 20:08] scout done — `getPreferenceByName` 이 DB 장애까지 404 로 내는 것을 404/500 으로 분리 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- 무엇을 왜: `getPreferenceByName` 이 `prefs.GetByName` 의 모든 오류를 404 `api.preference.get.not_found` 로 접어 DB 장애가 "설정 없음"으로 보였다. `errors.Is(err, pgx.ErrNoRows)` 만 404(본문 id·메시지 그대로), 그 외는 500 `api.preference.get.app_error` 로 갈랐다(같은 파일 list 핸들러들의 `.app_error` 관례). 서비스·라우팅·게이트는 미변경. 커밋 ded7e4e.
- 확신 없는 곳·검증 못 한 것: 500 에 `err.Error()` 를 본문 message 로 싣는다 — 형제 list 핸들러와 같은 방식이지만 pgx 오류 문구(테이블명 등)가 클라이언트로 나간다. 500 전용 id 를 새로 만든 것은 과제서가 허용한 선택지이며 404 id 는 건드리지 않았으나, 500 id 를 기존 id 그대로 두길 원하면 한 줄 되돌리면 된다. 장애 재현은 `DROP TABLE preferences CASCADE` 한 가지뿐 — 연결 끊김·컨텍스트 취소는 같은 분기를 타지만 실제로 돌려보지는 않았다. 웹은 건드리지 않아 typecheck/build 미실행.
- 일부러 하지 않은 것: 같은 파일 `upsertPreferences`/`deletePreferences` 의 400→500 분리(서비스 센티널이 필요하고 미머지 브랜치와 겹침), 사이드바 재정렬(이미 origin/auto/2026-09-21-0304 에 구현됨), 404 본문 정리(클라이언트 계약).
- 다음 역할이 조심할 것: 새 테스트 `internal/httpapi/preferences_errors_postgres_test.go` 는 `MOYRO_TEST_POSTGRES_DSN` 이 있어야 돌고 없으면 통째로 skip 된다(`go test ./internal/httpapi/` 의 ok 만으로 통과라고 하지 말 것). 검증에 쓴 컨테이너는 포트 55460(55433 은 다른 세션 점유). `gofmt -l internal/httpapi` 가 보고하는 `native_activity.go` 는 main 에서도 그런 기존 상태다.
- [러너 20:15] brief accepted — 채택 — 과제서의 근거(핸들러가 모든 오류를 404 로 접고 서비스는 이미 pgx.ErrNoRows 를 구분)가 현재 코드와 정확히 일치�
- [러너 20:15] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- 판정 approve(risk low, blocking 없음). 실제 PG16 컨테이너(55471)를 띄워 새 테스트를 돌렸다: 현재 HEAD 에서 PASS, 핸들러만 main 판으로 되돌리면 `storage fault: status = 404, want 500` 으로 FAIL — 테스트가 변경을 진짜로 구분한다. 404 본문 id·message 도 단언으로 고정돼 클라이언트 계약이 지켜진다.
- 구현자가 의심한 자리를 먼저 봤다. (1) 500 에 실리는 err.Error() 는 인증된 본인/관리자에게 가는 pgx 스키마 문구뿐이고 바로 위 두 형제 list 핸들러가 이미 같은 방식이라 차단 아님(notes). (2) 500 전용 id 신설은 404 계약을 건드리지 않으므로 무해. (3) webapp 의 prefsApi.getOne 은 404/500 을 구분하지 않고 모두 throw 하므로 클라이언트 회귀 없음.
- 같이 돌린 것: DSN 있는 상태의 `go test -count=1 ./internal/httpapi`(23s 전부 PASS), `go vet`, `scripts/check-source-sizes.sh`. gofmt 의 native_activity.go 는 main 에서도 같은 기존 상태. CI 는 ci.yml:63 에서 DSN 을 주므로 이 테스트가 실제로 돈다.
- 못 본 것: 전체 `go test -race -p 1 ./...`, 웹 typecheck/build(웹 무변경), e2e. 마이그레이션·인증·워크플로 변경 없음이라 revert 는 커밋 하나 되돌리면 끝난다.
- 릴리즈 노트에 남길 것: 같은 파일 upsertPreferences/deletePreferences 의 400/403 위장은 여전히 남아 있다(의도적 제외, 로컬 미머지 auto/2026-09-07-0240 과 겹침) — 다음 회차 후보.
- [러너 20:18] review approved — 리뷰 승인 (risk=low)
- [러너 20:18] pr created — https://github.com/hkjang/moyro/pull/22
- [러너 20:31] ci passed — 검사 3개 모두 success
- [러너 20:31] merge done — ded7e4e
