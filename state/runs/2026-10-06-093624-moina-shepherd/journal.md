# PR 처리기 노트 2026-10-06-093624-moina-shepherd — moina PR #42
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-06-075751-moina-improve)
# 회차 노트 2026-10-06-075751-moina-improve — moina
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:57] base pinned — main@c9bfca2
- [러너 07:57] autonomy release — 

## 정찰 노트
- `social.go` 이관을 골랐다 — 직전 회차의 `posts.go`와 구조가 같고(프로덕션 파일 1개) release-ready로 끝난 유형이며, 무엇보다 **기존** `moim_join_postgres_integration_test.go:193`이 실제 pg 실패를 핸들러 경로로 통과시켜 red→green 수단이 이미 있다. 차선인 settings+preferences+analytics(19곳)는 곳 수가 적어도 세 파일 모두 저장 실패 테스트가 없어 새 트리거 설계가 필요해 더 위험하다. admin.go(25곳)는 다음 회차로 미뤘다 — social이 더 크고 message 중복도 심하다.
- 확인한 것: 29곳의 줄번호와 감싼 함수 매핑(`^func ` 경계), `blockUser:141`·`createMoim:524`의 합친 `Exec || Commit` 조건, `writeProfile`의 두 호출자, 트리거의 `RAISE EXCEPTION`(→P0001), `writeStorageError`/`deepestErrorType`의 nil 처리, 로그 캡처 배선 줄번호.
- 추측(미실행): 테스트를 돌리지 않았다 — `TestPostgreSQL*` 44건·`make check` 120개는 직전 회차 기록을 그대로 옮긴 값이고 이번에 재확인하지 않았다. `createReport`의 Scan 무시도 이번엔 미확인.
- 구현자가 조심할 것: 합친 두 조건을 **그대로** 이관하면 `cause_type:"<nil>"`을 새로 만든다(분할 필수, `createMoim`의 `IsConflict`는 Exec 오류에만 유지). `handler` 이름은 손으로 적지 말고 함수 경계에서 유도할 것. `followUser` 네 출구가 같은 handler라 로그만으로 단계가 갈리지 않는 한계는 받아들이고 새 필드를 발명하지 말 것.
- [러너 08:03] scout done — `social.go`의 `storage_error` 500 출구 29곳을 `writeStorageError`로 이관한다 (가치 3 / 위험 1 / 작업량 M)

## 구현 노트
- `social.go`의 `storage_error` 500 출구 29곳을 `writeStorageError`로 이관해 `handler`·`cause_type`·`pg_code`가 로그에 남게 했다(238eff1, 프로덕션 1개 + 테스트 2개). `handler`는 `^func ` 경계에서 유도해 치환하고 같은 유도로 32개 호출 mismatch 0을 재확인했으며, 응답 불변은 HEAD와의 한글 리터럴 multiset 비교(HEAD에만 있는 리터럴 **0개**)로 증명했다.
- **확신 없는 곳**: `blockUser` 분할 두 출구(Exec 실패·Commit 실패)를 **지나는 테스트가 없다** — block 라우트는 저장소 전체에 integration 테스트가 0건이라 컴파일·전체 회귀·단락 평가 추론만으로 보증된다. 비평가가 먼저 볼 곳. `createMoim` 쪽 분할은 새 테스트로 덮고 되돌려 검증까지 했다(409는 분할 전후 양쪽 PASS). `followUser` 네 출구는 `handler`가 같아 로그로 단계가 안 갈리고, `Begin`·`Commit`은 `pg_code`가 둘 다 빌 수 있어 실무 구분이 불완전하다.
- **일부러 하지 않은 것**: 과제서 지시대로 `stage` 필드를 발명하지 않았고 `writeProfile`에 `handler` 파라미터를 뚫지 않았다(출구 1곳이라 이름만으로 충분, 파일 수 1개 유지). `joinMoim`/`createReport`/`getMedia`의 `_ = Scan(&exists)` 404 은폐 계약은 범위 밖이라 그대로 뒀다. 응답이 불변이라 프런트 `npm test`·e2e·시각 회귀·`api/openapi.yaml`은 손대지 않았다.
- **다음 역할이 조심할 것**: 두 테스트 파일은 **DB가 있어야 돈다**. `MOINA_TEST_POSTGRES_DSN` 없이는 `t.Skip`이라 통과처럼 보인다 — `-v`로 `--- SKIP` 0줄을 확인할 것. 포트 55432는 다른 세션 컨테이너가 점유 중이어서 **55812**를 썼다(검증 끝나고 컨테이너 `moina-1006-pg`는 정리했다). 새 `moim_create_…_test.go`는 `moims`에 `BEFORE INSERT` sentinel 트리거를 만들고 `t.Cleanup`에서 지운다(공유 DB에서도 slug 한 개만 거부).
- [러너 08:15] brief accepted — 채택 — 근거가 현재 코드와 전부 일치했습니다(29곳의 줄번호, `^func ` 경계로 유도한 29개 handler 매핑이 과제서의 수기 매
- [러너 08:15] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 로컬 `main`이 v0.1.28에 멈춰 있어 `git diff main...HEAD`가 87파일로 보인다 — 실제 base는 `origin/main`(c9bfca2)이고 PR은 커밋 1개·파일 3개다. 다음 역할도 `origin/main`으로 비교할 것.
- 독립 재현함: throwaway pg(55913)로 두 테스트 PASS → `social.go`만 origin/main으로 되돌려 **같은 4건 red**(로그 단언만, 201·409 `slug_taken`은 양쪽 PASS) → 복원. 전체 `go test -race -count=1 -v ./...` exit 0·`--- SKIP` **0줄**·`--- FAIL` 0줄·최상위 `TestPostgreSQL*` **45건**. build/vet/gofmt 무지적. 원장의 "실패 재현"은 내가 본 출력과 일치한다.
- 기계 확인: `writeStorageError` 32곳 전부 `^func` 경계와 handler 라벨 **mismatch 0**, 한글 리터럴 multiset이 불변(+2는 분할로 같은 message 재사용, +1 `Topic을 Link할 수 없습니다`는 main에 이미 있던 followTopic), social.go `StatusInternalServerError` **0건** 잔존. 보안·법무 차단 사유 없음 — 새 로그 필드는 error_code/handler/`%T`/SQLSTATE뿐이고 `%v`·`%s` 0건이라 행 값·PII 경로가 없으며 인증·인가·마이그레이션은 손대지 않았다.
- 못 본 것: 프런트 `npm test`·e2e·시각 회귀·`make check`를 돌리지 않았다(응답 불변을 리터럴 비교로 증명했으므로 영향 없다고 판단). blockUser의 두 출구는 실행 경로로 밟지 못해 읽기 등가성으로만 보증했다.
- 승인 후 남는 우려는 review.json `notes` 5건 — 특히 blockUser 미커버와 남은 **66곳**(프로필의 95는 낡음)을 다음 회차 프로필에 반영할 것.
- [러너 08:20] review approved — 리뷰 승인 (risk=low)
- [러너 08:20] pr created — https://github.com/hkjang/moina/pull/42
- [러너 08:23] ci failed — 성공이 아닌 검사: Image and browser smoke=failure · 실패한 검사: ? 잡: Image and browser smoke 
