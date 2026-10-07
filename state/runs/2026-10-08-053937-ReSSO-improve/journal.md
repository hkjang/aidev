# 회차 노트 2026-10-08-053937-ReSSO-improve — ReSSO
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 05:39] base pinned — main@3046708
- [러너 05:39] autonomy release — 

## 정찰 노트
- 고른 이유: 이전 정찰이 "다음 회차 1순위로 적합" 으로 남긴 자리이고, 형제 경로 `oidc.go:706-711` 주석이 **바로 이 결함을 문장으로 적어 고쳤다** — 코드 재사용 쪽만 `_` 로 남았다. 프로덕션 1파일·응답 무변경이라 위험이 가장 낮다. refresh 그랜트 500(3/3/M)은 `store/oidc.go:306-312` 선행 변경이 필요해 한 세션에 안 들어가고 기각된 5bed9dc 와 성격이 겹쳐 제쳤다. npm audit 정리는 lockfile 회차라 제쳤다.
- 확인한 것: `oidc.go:560` 의 `affected, _ :=`, `userLookupFailed`(:91, `ErrNotFound` 를 false 로 거른다), 형제 경로 :690-698·:706-722, `UserByID`(store/users.go:37, `userColumns` SELECT), `audit`(auth.go:313), `actor_id uuid REFERENCES users(id)`(migrations/001:206), 기존 테스트 2201행, docs/operations.md:87·104 가 **고쳐야 할** 두 줄인 것, `EndSessionsOfDisabledUsers` 를 단언하는 테스트가 **없다**는 것.
- 추측으로 적은 것(미확인): `ALTER TABLE users RENAME TO users_hidden` 중에도 `audit_events` insert 의 FK 가 통과하리라는 것(Postgres FK 가 OID 를 따라간다는 판단). 수용 기준 1(a)가 그 확인이고, 깨지면 컬럼 이동으로 좁히는 대안을 과제서에 적었다. 테스트는 한 번도 돌리지 않았다(예산).
- 구현자가 조심할 것: 응답을 500 으로 바꾸지 말 것(5bed9dc 재제출 금지, 테스트가 본문 문안을 단언). 코드 채굴과 첫 교환을 `users` 치우기 **전에** 끝낼 것. 새 테스트는 `New(..., metrics)` 로 등록기를 넘길 것 — 2201 테스트를 그대로 복사하면 `nil` 이라 지표를 못 읽는다.
- [러너 05:44] scout done — 코드 재사용 감사의 actor 가 `users` 테이블 장애로 조용히 비는 것을 막기 — `handleAuthorizationCodeGrant` 의 `User

## 구현 노트
- `httpserver/oidc.go:560` 의 `affected, _ := UserByID` 를 `actorErr` 로 받아 `userLookupFailed` 에 넘기고 `detail["actor_resolved"]=false` 를 더했다 — `AUTHORIZATION_CODE_REUSED` 의 actor 가 `users` 장애 중 조용히 비던 것이 이제 detail·ERROR 로그·`token_errors{grant_type="authorization_code"}` 로 말한다. 헬퍼 반환값은 **응답 분기로 쓰지 않는다**: 400 `invalid_grant` 본문까지 무변경(기각된 5bed9dc 와 선 긋기, 테스트가 문안 단언). 프로덕션 1파일.
- **확신 없는 곳**: 수용 기준 4(`ErrNotFound` 는 신호를 내지 않는다)를 **단언하지 못했다** — `authorization_codes.user_id` 가 `ON DELETE CASCADE`(migrations/001:152)라 삭제된 계정은 자기 코드를 함께 가져가고, 그러면 재사용 감지 자체가 없어 이 분기에 `ErrNotFound` 가 닿지 않는다. 손으로 상태를 만들어 증명하는 것은 운영자 규칙 위반이라 하지 않았다. 필터는 무수정 `userLookupFailed`(:92) 안에 있고 다른 두 호출부가 이미 쓴다. 비평가는 여기를 먼저 보라.
- 과제서가 미확인으로 남긴 것은 **실행으로 확인했다**: `ALTER TABLE users RENAME TO users_hidden` 중에도 `audit_events.actor_id` FK 가 통과해 감사 항목이 남는다(항목 2건) — 컬럼 이동 대안은 쓰지 않았다.
- 일부러 하지 않은 것: `userLookupFailed` 무수정(세 호출부 공유, 앞 둘은 응답까지 바꾼다), `auth.go`·`store/oidc.go`·`migrations`·README 지표 표 미변경, detail 에 오류 원문 미기입(로그에만).
- 다음 역할이 조심할 것: 새 테스트는 **실제 PostgreSQL 필요**(`eval "$(scripts/test-services.sh)"` 를 같은 셸에서). `New(data, logger, nil, metrics)` 로 등록기를 넘겨야 지표를 읽는다 — 옆 2201 테스트는 `nil` 이다. 지표 **부재** 단언은 계열 이름이 아니라 샘플 줄(`resso_token_errors_total{grant_type=`)로 쓸 것: 등록기가 표본 없는 계열에도 `# HELP`/`# TYPE` 을 출력해 이름만 보면 틀린 이유로 실패한다(첫 red 실행에서 실제로 그랬다).
- [러너 06:02] brief accepted — 채택 — 지정한 분기 위치·`detail` 키 이름·헬퍼 호출 모양·주석 논지 셋·문서 두 줄·장애 주입 순서를 그대로 썼고, 과�
- [러너 06:03] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것(읽기 + **실행**): `oidc.go:556-585` 두 줄 로직, `userLookupFailed`(:91-99, 응답을 쓰지 않고 신호만 낸다 — 400 무변경이 코드로도 성립), `RedeemAuthorizationCode` 의 `consumed` 분기가 세 반환 모두 `code` 를 채워 주는 것(store/oidc.go:166-190, 87a 이후), 테스트가 스키마 격리라 `ALTER TABLE users RENAME` 이 다른 테스트에 닿지 않는 것(integration_test.go:265), `t.Cleanup(restore)` 가 `exchange` 의 `t.Fatal` 보다 먼저 등록된 것, 문서 두 줄이 실제 로그 문구·detail 키와 맞는 것. 실행: `make lint` 0 issues·govulncheck 0, `go test -race ./internal/httpserver -count=1` **137.4s 전체 통과**(실제 PostgreSQL), 신규+형제 재사용 테스트 통과, gofmt/vet clean.
- **auto verify 는 연동 테스트를 돈 적이 없다**(`go test ./...` 1초 = SKIP). 그 공백은 위 실행으로 메웠다. 반대로 못 본 것: 프로덕션 변경을 되돌린 red 재현은 직접 하지 않았다(읽기 전용 지시) — 원장의 red 출력과 `_ = affected` 프로브 기록, 그리고 `actor_resolved` 키가 변경 전에는 존재할 수 없다는 점으로 갈음했다. `internal/store` 전체 suite 도 안 돌렸다(이 diff 가 그 패키지를 건드리지 않는다).
- 구현자의 자기 의심(수용 기준 4)은 결함이 아니다 — `migrations/001_initial.sql:152` 의 `ON DELETE CASCADE` 때문에 삭제된 계정은 자기 코드를 가져가고 재사용 감지가 애초에 일어나지 않아 `ErrNotFound` 가 이 분기에 도달하지 못한다. 단언 불가가 정답.
- 승인이어도 남는 우려 둘: ① `resso_token_errors_total` 이 **400 으로 답하는 경로**에서 처음 오른다(지금까지 500 과 짝) — 문서는 맞지만 **릴리즈 노트가 반드시 말해야 하는 변화**다. ② `userLookupFailed` 가 `context.Canceled` 를 걸러내지 않아 끊긴 RP 가 이 계열을 올린다 — 기존 공유 헬퍼의 성질이고 Client 인증이 필요해 증폭은 아니지만 8f2c73c 관용구에서 벗어나 있다. 다음 회차 후보.
- 릴리즈 전 정리: `webui/dist/index.html` 워킹트리 오염(verify 의 `npm run build` 산출물)을 되돌릴 것. CHANGELOG.md 항목은 아직 없다.
- [러너 06:10] review approved — 리뷰 승인 (risk=low)
- [러너 06:10] pr created — https://github.com/hkjang/ReSSO/pull/40
- [러너 06:17] ci passed — 검사 2개 모두 success
- [러너 06:17] merge done — 1a1d2be
