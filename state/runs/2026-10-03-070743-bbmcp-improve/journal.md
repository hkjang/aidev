# 회차 노트 2026-10-03-070743-bbmcp-improve — bbmcp
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:07] base pinned — main@aef897e
- [러너 07:07] autonomy release — 

## 정찰 노트
- `approval.Check` 의 fail-open 2건을 골랐다. 테스트 공백 보강 후보(mcp/identity/aiproxy)보다 실제 동작 결함이고, 프로덕션 파일이 1개로 끝나며, 이미 프로덕션 배선으로 도는 `internal/tools/integration_test.go` 하네스 위에서 증명할 수 있다(손으로 만든 대역 금지 지침 충족). go.mod tidy 류는 릴리즈 경로를 흔들 뿐이라 기각했다.
- 추측으로 적은 것: 20개 고루틴 동시 `Check` 가 수정 전에 2건 이상 성공한다는 것은 코드상 레이스(ByID → 별도 UPDATE)에서 추론한 것이고 **직접 재현하지는 않았다**. 로컬에 Postgres 가 떠 있는지도 확인하지 못해 테스트를 한 번도 돌리지 못했다(DB 없으면 `t.Skip` 으로 조용히 넘어간다).
- 구현자 주의: `prFetchFails` 시나리오는 머지 핸들러가 PR 을 다시 조회하므로 수정 전에도 머지는 실패한다. "머지 안 됨"이 아니라 **오류 코드와 승인 레코드 상태(approved vs consumed)**로 증거를 잡아야 한다.
- `Check` 안의 검사 순서(해시 → PR 버전)를 유지해야 PR 무관 쓰기 도구에 오탐이 생기지 않는다. `executor.go:prVersion` 의 오류 삼킴은 일부러 손대지 않았다(영향 범위가 브랜치 제한 경로로 번진다).
- 프로필이 없어 `profile.md` 를 새로 썼다(테스트 조용한 skip, `-p 1` 필수, 테스트 없는 패키지 목록 포함).
- [러너 07:12] scout done — 승인 검증의 fail-open 두 곳 닫기 — PR 버전 확인 실패 시 통과, 단일 사용 승인의 동시 재사용 (가치 5 / 위�

## 구현 노트
- `internal/approval/approval.go:Check` 의 fail-open 2건을 닫았다(커밋 4e26f2d, 프로덕션 파일 1개): PR 버전을 고정한 승인인데 현재 버전이 `nil` 이면 `ErrStale` 로 거부하고 승인은 `approved` 로 남긴다(재시도 가능), 소비는 `WHERE id=$1 AND status='approved'` 조건부 UPDATE + `RowsAffected()==0 → ErrStale`.
- 수정 전 결함을 둘 다 눈으로 재현했다: `expected APPROVAL_STALE, got BITBUCKET_ERROR` + `approval status = "consumed", want "approved"`, 그리고 `5 of 20 concurrent checks consumed the same approval`. 정찰이 추측으로 남긴 "동시에 2건 이상 성공" 은 사실로 확인됐다.
- 확신 없는 곳: 조건부 UPDATE 는 단일 Postgres 행 잠금에 의존한다 — 단일 프로세스 20 고루틴 × 5회 반복은 확인했지만 **여러 게이트웨이 인스턴스(다중 프로세스) 동시 호출은 돌려보지 않았다**(원리상 동일한 행 잠금이라 같아야 한다). `Check` 가 성공 시 반환하는 `req` 는 UPDATE 이전에 읽은 스냅샷이라 `Status`/`ConsumedAt` 이 여전히 `approved`/`nil` 이다 — 수정 전과 같은 동작이고 호출자(`executor.go:295`)는 이 필드를 보지 않아 그대로 뒀다.
- 일부러 안 한 것: `executor.go:prVersion`/`prTargetBranch` 의 오류 삼킴(브랜치 제한 경로의 의미까지 바뀐다 — ideas.json 에 별 항목으로 남겼다), `redactArgs` 스크럽 키 보강(같은 파일이지만 범위를 넓히지 않았다).
- 다음 역할 주의: 새 테스트 2건은 **Postgres 가 있어야 돈다**. `TEST_DATABASE_URL` 없이 돌리면 `t.Skip` 으로 조용히 넘어가 아무것도 증명하지 못한다. 이 회차는 `docker run -d --name bbmcp-test-pg -e POSTGRES_USER=bbmcp -e POSTGRES_PASSWORD=bbmcp -e POSTGRES_DB=bbmcp_test -p 15532:5432 postgres:16-alpine` 으로 띄우고(로컬 5432 는 다른 컨테이너가 점유) `-p 1 -race` 로 전체 통과를 확인했다. 컨테이너는 그대로 남겨 두었다.
- 빌드·릴리즈 경로(Dockerfile, CI, web/, migrations)는 건드리지 않았다 — 변경은 Go 파일 2개뿐이다.
- [러너 07:16] brief accepted — 채택 — 과제서의 근거 2건이 현재 코드와 정확히 일치하고 수용 기준 3개를 그대로 충족했다.
- [러너 07:17] verify passed — 검증 5개 통과 (auto)

## 비평 노트
- 확인함: 구현자가 의심한 두 자리를 직접 시험했다 — `main` 의 approval.go 로 되돌려 신규 테스트 2건이 실제로 실패하는 것을 봤고(`expected APPROVAL_STALE, got BITBUCKET_ERROR`, `7 of 20 concurrent checks consumed the same approval`) 복원 후 postgres(15532) 로 전체 `-p 1 -race` 통과, 신규+기존 승인 테스트 `-count=5 -race` 안정. 반환 `req` 스냅샷은 유일 호출자(`executor.go:242` → record)가 `appr.ID` 만 읽어 무해함을 코드로 확인했다.
- 못 봄: 다중 프로세스 동시 호출(단일 행 잠금이라 프로세스 수와 무관해 결함으로 보지 않음), `executor.go:prVersion` 의 오류 삼킴(범위 밖).
- 승인이어도 남는 우려 — 릴리즈 노트에 쓸 것: `approval.go:227` 의 거부는 머지 전용이 아니다. `executor.go:318` 이 args 에 pullRequest 가 있는 **모든** 승인에 버전을 고정하므로, Bitbucket PR 조회가 일시 실패하면 코멘트·decline 승인도 APPROVAL_STALE 로 막힌다(승인은 approved 로 남아 재시도 가능 — 테스트로 확인). 가용성 트레이드오프를 사용자에게 알릴 것.
- 작은 우려: 조건부 UPDATE 가 `StatusApproved` 상수 대신 리터럴 `'approved'` 를 쓴다(기존 `Decide` 와 같은 스타일). 동시성 테스트는 실패 사유를 단언하지 않아 무관한 오류로도 통과할 수 있다.
- 보안·법무 차단 없음: 양쪽 모두 fail-open → fail-closed 로 좁히는 변경이고, 새 엔드포인트·개인정보·비밀값·수제 암호 비교·마이그레이션이 없어 revert 로 완전히 되돌아온다.
- [러너 07:20] review approved — 리뷰 승인 (risk=low)
- [러너 07:20] pr created — https://github.com/hkjang/bbmcp/pull/1
