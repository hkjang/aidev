# 회차 노트 2026-10-17-180839-moina-improve — moina
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 18:08] base pinned — main@c9bfca2
- [러너 18:08] autonomy release — 

## 정찰 노트
- 배정된 "워크플로 실패" 는 사실이 아니다 — 이 run 디렉터리에 `ci-failure-*.txt` 가 없고 워크트리는 clean 에 `origin` 에 새 브랜치도 없다. 직전 회차는 CI 가 깬 게 아니라 **PR 전에 끊겼다**. 그래서 워크플로(보호 경로)를 건드리지 않고 **같은 과제(admin.go 25곳)를 재시도**하도록 적었다.
- 제친 후보: 프런트 `findBy` 경합 전수 점검(가치 4)은 `make image` 증명이 필요해 중단 재발 위험이 가장 크고, 미머지 브랜치 재제출은 운영자가 금지했으며, `$GITHUB_STEP_SUMMARY`·과거 커밋 전수조사는 이번에 `rejected` 로 내렸다.
- 과제서에서 추측으로 적은 것은 하나뿐: `role_permissions` BEFORE INSERT sentinel 트리거가 `adminPutRoles:499` 를 실제로 거절하는지 **미실행**(코드만 읽었다). 25곳 줄번호·함수 매핑·`scanErr` 4곳·`tx.Commit` 3곳 분리·테스트 배선 줄번호는 전부 실행으로 확인했다.
- 구현자가 조심할 것: 중단이 이번 실패 모드이므로 **25곳 치환 직후 커밋**하고 검증을 반복하지 말 것. `social.go`·`AuthContext.test.tsx` 는 미머지 브랜치와 충돌하니 금지.
- [러너 18:13] scout done — (수정 과제 — 직전 회차 `error: interrupted before PR` 재시도) `admin.go` 의 `storage_error` 500 출구 25곳을 `writeStorageE

## 구현 노트
- `admin.go` 의 `storage_error` 500 출구 25곳을 `writeStorageError` 로 이관했다 — `adminPutRoles` 의 다섯 출구가 같은 message 를 내보내 운영자가 어느 쿼리가 거절됐는지 알 수 없었다. `handler` 는 `^func ` 경계에서 기계적으로 유도하고 재유도로 30개 호출 mismatch 0 을 확인했다. 응답은 한글 리터럴 multiset 비교로 불변(사라진 0·늘어난 0).
- 확신 없는 곳: 없음에 가깝다. 과제서가 미확인으로 남긴 sentinel 트리거의 `:499` 거절은 실행으로 확인했다(`pg_code":"P0001"` 단언 통과). 단 과제서와 달리 트리거를 `NEW.permission` 대신 **`NEW.role_name`** 에 걸었다 — `permission` 은 `permissionCatalog` 검증을 통과해야 해 실제 권한 값만 쓸 수 있고, 한 DB 를 네 패키지가 공유하는 CI 에서 실제 권한 값을 전역 거절하면 남의 테스트를 깨뜨린다. 비평가가 먼저 볼 곳은 이 트리거 범위다.
- 검증 못 한 것: `make lint` 의 프런트 절반(`eslint: not found` — 이 워크트리에 `frontend/node_modules` 가 없다). 프런트 파일을 하나도 건드리지 않았고 과제서가 프런트 실행을 금지했으므로 `npm ci` 를 돌리지 않았다. Go 절반(`go vet` + `staticcheck@2025.1.1`)은 별도로 exit 0 을 확인했다.
- 일부러 하지 않은 것: `make image`·e2e·프런트 테스트(응답 불변 + 프런트 무변경), 다른 파일의 이관(settings 9·workflow 9·auth 7 등), `storage_error.go` 헬퍼 수정, 새 로그 필드. `adminPutRoles` 다섯 출구가 `handler` 만으로 구분되지 않는 한계는 `followUser` 선례대로 그대로 남겼다.
- 다음 역할이 조심할 것: 새 테스트는 **DB 가 있어야 돈다**(`MOINA_TEST_POSTGRES_DSN` 없으면 `t.Skip`). `role_permissions` 에 BEFORE INSERT 트리거를 만들고 `t.Cleanup` 에서 DROP 하므로 같은 DB 를 쓰는 다른 테스트와 동시 실행하면 안 된다(이 패키지는 병렬 실행하지 않는다). 성공 케이스가 시드된 `super_admin` 역할 행의 description 을 같은 값으로 upsert 하므로 `updated_at` 만 바뀐다.
- [러너 18:20] brief accepted — 채택 — 과제서가 실행으로 확인했다고 적은 것을 전부 재확인했고 모두 참이었습니다(`grep -c` 25곳, 25개 출구의 줄번호�
- [러너 18:20] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인: throwaway postgres 로 직접 red/green 을 재현했습니다 — HEAD 3/3 PASS, `admin.go` 만 `c9bfca2` 로 되돌리면 로그 단언 4건만 FAIL 하고 응답(500·`storage_error`·message·롤백) 단언은 red 에서도 통과 → 테스트가 실제로 바뀐 경로를 지납니다. 전체 `go test -race -count=1 -v ./...` exit 0 / `--- SKIP` 0줄 / `TestPostgreSQL*` PASS 45건, `go build`·`go vet`·`gofmt -l`·OpenAPI 120 route 모두 통과, 되돌린 파일 복구해 워크트리 clean.
- 확인: 한글 message 멀티셋 30개 동일(사라진 0·늘어난 0), `writeError(...,"storage_error")` 잔존 0건, `^func ` 재유도로 30개 호출 handler mismatch 0, 30개 전부 `err != nil`/`scanErr != nil` 가드 안 → `cause_type:"<nil>"` 생길 자리 없음. 보안·법무 차단 사유 없음(인가 로직 무변경, 새 로그 필드는 상수·Go 타입명·SQLSTATE 셋뿐, pg message/Detail/Hint 여전히 미출력).
- 승인이어도 남는 우려 ①: `adminPutRoles` 다섯 출구가 모두 `handler:"adminPutRoles"` 라 handler 만으로는 여전히 어느 쿼리인지 못 가릅니다(pg_code 로 좁히게 됨). 선례와 동일한 의도된 한계 — **릴리즈 노트에 그대로 적을 것**.
- 승인이어도 남는 우려 ②: 새 테스트가 인증/인가 경로가 읽는 `role_permissions` 에 테스트 트리거를 거는 첫 사례입니다(CREATE/DROP 이 ACCESS EXCLUSIVE). 실측 0.41s·cleanup 이 별도 context 로 DROP·LIFO 순서 정상이라 막지 않지만, CI source 잡에서 `role_permissions` 관련 간헐 실패가 보이면 여기를 먼저 보세요. 성공 subtest 가 시드 `super_admin` 의 `updated_at` 만 바꾸고 복원하지 않습니다(값 변화 없음).
- 못 본 것: `make lint` 프런트 절반(node_modules 없음)·staticcheck(PATH 없음)·`make image`·e2e. 프런트 변경 0 + 응답 바이트 불변이라 위험 없다고 판단했습니다. 대기 브랜치의 `social.go`·`AuthContext.test.tsx` 와 충돌 0건, revert 는 코드 2파일로 끝납니다.
- [러너 18:25] review approved — 리뷰 승인 (risk=low)
- [러너 18:25] pr created — https://github.com/hkjang/moina/pull/45
- [러너 18:34] ci passed — 검사 2개 모두 success
- [러너 18:34] merge done — 08e782c
- [러너 18:50] release published — v0.1.44
- [러너 18:56] assets verified — v0.1.44 자산 1개 (이전 v0.1.43: 1)
