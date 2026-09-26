# 회차 노트 2026-09-26-203224-moyro-improve — moyro
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 20:32] base pinned — main@47d2c66
- [러너 20:32] autonomy release — 

## 정찰 노트
- custom profile PATCH 되읽기 삼킴(final.go:1290/1334)을 골랐다. 재고에서 가치·위험이 비슷했던 patchPost 건(early.go:524)은 같은 결함이지만 **되읽기만 실패시키는 주입 방법이 여전히 없어서** 제쳤고, 이쪽은 이번에 그 방법을 찾았다: `PatchUserValues` 가 `len(values)==0` 이면 DB 를 아예 건드리지 않고 성공하는데 `GetUserValues` 는 무조건 SELECT 하므로, `DROP TABLE custom_profile_values` + 본문 `{}` PATCH 면 쓰기 성공·되읽기 실패가 대역 없이 재현된다. IsMember·preferences 건은 M(설계 선행), gofmt·bookmarks 단위테스트는 가치가 낮아 제쳤다.
- 추측으로 적은 것은 하나다: 위 주입을 **실제로 돌려 보지 않았다**(정찰은 코드 변경 금지 + 컨테이너 비용). 소스 두 함수를 읽고 도출했을 뿐이니 구현자는 RED 실행으로 이것을 가장 먼저 확인할 것 — `{}` PATCH 가 이미 500 이면 전제가 틀린 것이고, 그때는 차선(버려진 `decodeCappedBody` 오류)으로 가면 같은 세션에 끝난다.
- 구현자가 조심할 것: 감사 로그는 되읽기 *전에* 이미 남고 값 있는 PATCH 는 쓰기가 실제로 커밋된 뒤라 그 기록이 옳다 — 500 을 내더라도 감사 순서·유무를 건드리지 말 것. `decodeCappedBody` 도 이번에 같이 고치지 말 것(빈 본문까지 400 이 되면 호환 클라이언트가 깨진다). 테스트 본문은 명시적으로 `{}` 로 보내 decode 오류 경로와 섞지 말 것.
- 오류 id 는 새로 만들지 말고 형제 GET 의 `api.custom_profile.values.get.app_error` 를 그대로 재사용하도록 적었다 — 이 저장소가 반복해 고른 방식(2026-09-22 preferences, 2026-09-25 reminders)이고, 세 경로가 같은 장애에 같은 답을 하게 만드는 것이 이 과제의 요점이다.
- 프로필(2026-09-22)은 이번에 본 범위와 어긋나지 않아 새로 쓰지 않았다. `customprofile` 패키지와 custom profile httpapi 테스트는 현재 0건이라 이 과제가 첫 커버리지가 된다.
- [러너 20:37] scout failed — 과제서 없음 — 구현자가 직접 고른다

## 구현 노트
- 무엇을 왜: compat_wave_handlers_final.go 의 custom profile 값 PATCH 두 경로(1290/1334)가 되읽기 오류를 버려 `200 null` 을 내던 것을, 형제 GET 과 같은 500 `api.custom_profile.values.get.app_error` 로 통일했다(커밋 8fe8a75). 같은 맵을 읽는 세 경로가 같은 장애에 같은 답을 한다. 프로덕션 파일 1개 + 신규 테스트 1개.
- 정찰 전제는 RED 로 실제 확인됐다: `DROP TABLE custom_profile_values` + 본문 `{}` 에서 두 경로 모두 `status = 200, want 500 (body null)` 로 실패했고, 두 호출 지점을 각각 따로 되돌리면 그 경로의 서브테스트만 깨진다.
- **확신 없는 곳**: (1) 장애 주입이 `PatchUserValues` 의 `len(values)==0` → no-op 계약에 의존한다. 그 계약이 바뀌면(빈 맵도 tx 를 열면) 새 테스트는 조용히 "PATCH 가 먼저 실패" 로 바뀌어 여전히 500 을 보고 통과한다 — 되읽기를 더는 검증하지 못한다. `customprofile` 단위 테스트로 그 계약을 고정하는 것을 ideas.json 에 남겼다. (2) **값이 실제로 커밋된 뒤 되읽기만 실패하는 경우는 직접 재현하지 못했다.** 그런 주입 방법(트리거 안 DDL·RLS·권한)을 검토했지만 전부 쓰기까지 같이 깨져서, 빈-맵 경로로 같은 핸들러 줄을 지나는 것으로 갈음했다. 코드 경로는 동일하다.
- 일부러 안 한 것: `_ = decodeCappedBody(...)` 오류 삼킴(정찰 지시대로 — 빈 본문까지 400 이 되면 호환 클라이언트가 깨진다; ideas.json 에 `io.EOF` 만 허용하는 설계와 함께 남겼다). 감사 로그의 위치·순서도 그대로 뒀다 — 쓰기가 커밋된 뒤 되읽기가 돌므로 그 기록은 되읽기 결과와 무관하게 옳다.
- 다음 역할이 조심할 것: 새 테스트는 `MOYRO_TEST_POSTGRES_DSN` 이 있어야 돈다(없으면 `t.Skip` — ok 만 보고 통과라고 하면 안 된다). 로컬은 55433 컨테이너로 돌렸다. 검증: DSN 을 준 `go test -race -p 1 ./...` 49 패키지 통과, `go vet ./...`, `go build ./...`, `gofmt -l`(2파일 clean), `check-source-sizes.sh`(final.go 72407/77000), `verify-pages.mjs` 통과. 웹 변경 없음.
- [러너 20:44] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- 확인한 것: main 핸들러로 되돌려 새 테스트가 `status = 200, want 500 (body null)` 로 두 서브테스트 모두 실패함을 직접 재현하고 HEAD 에서 통과함을 확인(트리 복원, status clean). 장애 주입 전제(`len(values)==0` no-op vs 무조건 SELECT)를 소스로 검증. 실제 DSN 으로 `./internal/httpapi` 전체 통과(29.6s, skip 아님), vet·gofmt·source-sizes 통과.
- 못 본 것: 전체 `go test -race -p 1 ./...`(구현자 보고로 갈음), CI 의 PG15/16 실행, 웹(변경 0). 커밋 후 되읽기만 실패하는 시나리오는 나도 재현하지 않았다 — 핸들러 줄이 동일해 수용.
- 승인이어도 남는 우려 1: 테스트가 `PatchUserValues` 의 빈-맵 no-op 계약에 의존한다. 그 계약이 바뀌면 테스트는 500 을 계속 보며 통과하지만 되읽기를 더는 검증하지 않는다 — customprofile 단위 테스트로 못박는 ideas.json 항목을 다음 회차가 집어갈 것.
- 승인이어도 남는 우려 2(릴리즈 노트): 쓰기 성공 + 되읽기 실패가 이제 500 이라 클라이언트 재시도가 가능하고 업서트는 멱등이지만 audit `ActionCustomValuesPatch` 는 재시도마다 기록이 늘어난다. webapp·e2e·docs 에 `custom_profile_attributes` 참조 0건이라 자체 클라이언트 영향은 없다.
- 보안·법무 차단 없음: 인가 경로·개인정보 수집·비밀값 무변경. 500 `message` 의 `err.Error()` 노출은 형제 GET 과 같은 파일 patch 실패 경로가 이미 하던 것으로 신규 면이 아니다(저장소 전반 과제로 남김).
- [러너 20:49] review approved — 리뷰 승인 (risk=low)
- [러너 20:49] pr created — https://github.com/hkjang/moyro/pull/26
