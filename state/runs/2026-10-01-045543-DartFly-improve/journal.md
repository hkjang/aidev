# 회차 노트 2026-10-01-045543-DartFly-improve — DartFly
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 04:55] base pinned — main@8591080
- [러너 04:55] autonomy release — 

## 구현 노트
- CI 두 실패(36757310307·36763511898)는 **저장소 코드 문제가 아닙니다**: 세 잡 모두 `steps: []` 이고 annotation 이 `The job was not started because recent account payments have failed…`. private 저장소 + 결제 실패 → 잡이 시작 전에 죽습니다. 지난 회차 두 실행까지 네 번 연속 동일. ci.yml 은 손대지 않았습니다. `gh` 는 `GH_CONFIG_DIR=/home/hkjang/.config/gh` 로, 이유는 로그가 아니라 `gh api repos/hkjang/DartFly/check-runs/<jobId>/annotations` 에 있습니다.
- 대신 CI 가 못 돌린 세 잡을 로컬에서 다 돌려 `livedb` 가 **실제로 빨간** 것을 잡아 고쳤습니다(d61a5f0): `generateDDL` 이 node stdout/stderr 를 한 버퍼로 받아 `MODULE_TYPELESS_PACKAGE_JSON` 경고가 DDL 첫 문장이 되어 DB 에 실행됐습니다(`syntax error at or near "node"`). stderr 분리 + 성공 시 t.Logf 진단. 지난 회차가 "로컬 전용 오탐" 이라 적은 것은 오탐이 아니었습니다.
- **확신 없는 곳·검증 못 한 것**: ① CI 가 초록이 되는지는 여전히 확인 불가(결제). ② CI 러너의 node 가 이 경고를 내는지 **확인하지 않았습니다** — 2026-09-28 에 livedb 가 초록이었으니 그 node 는 조용했을 가능성이 큽니다. 즉 이 수정은 "CI 를 빨간 데서 초록으로" 가 아니라 "환경 경고 하나에 잡이 깨지는 구조를 없앰" 입니다(로컬에서는 빨강→초록이 실제로 일어납니다). ③ 경고의 근원(루트 package.json 없음)은 일부러 건드리지 않았습니다 — 저장소 전체의 .js 해석이 바뀌어 재확인 범위가 훨씬 넓습니다. ideas.json 에 남겼습니다.
- PR #19 의 세 커밋(978a809·0b156d6·7fa00ad)을 체리픽해 얹었습니다(6fe63bd·b47dead·aad96ce). 기능은 되돌리지 않았고 그 내용은 재검토하지 않았습니다.
- 다음 역할이 조심할 것: 새 테스트 `TestGeneratedDDLIgnoresNodeStderr` 는 **`-tags livedb` 가 필요하지만 DB 는 필요 없고 node 는 필요합니다**(없으면 Skip). `TestLiveGeneratedDDLActuallyRuns` 는 `test/livedb/setup.sh --fast` + `/tmp/dartfly-livedb.env` 가 있어야 돕니다. livedb 컨테이너는 이 회차가 띄운 채로 남겨 뒀습니다. 스모크 브라우저 단계는 이 환경에서 `PLAYWRIGHT_BROWSERS_PATH=/home/hkjang/.cache/ms-playwright` 를 줘야 돕니다.
- [러너 05:04] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 원장에 `- 실패 재현:` 줄이 없어 직접 재현했습니다: main 임시 워크트리에 새 테스트만 얹으니 JS 저장본 테스트 8개가 `not ok`, generateDDL 의 `Stderr = &errOut` 만 `&out` 으로 되돌리니 Go 테스트가 실제로 FAIL(첫 문장 = MODULE_TYPELESS_PACKAGE_JSON). 브랜치에서는 둘 다 통과 — 두 테스트 모두 진짜 바뀐 경로를 지납니다.
- 돌린 것: gofmt / go vet(livedb 태그 포함) / go test ./... / node admin-history.test.mjs(18개) / bash -n run.sh. **못 본 것**: 브라우저 실제 스모크와 CI(결제로 잡이 시작 안 됨) — 화면 변경이 있으니 릴리즈 전에 `DF_SMOKE_REQUIRE_BROWSER=1 PLAYWRIGHT_BROWSERS_PATH=$HOME/.cache/ms-playwright bash test/smoke/run.sh` 를 한 번 돌리는 것이 좋습니다.
- 러너의 "ts 파일을 건드렸다" 경고는 오탐입니다(diff 에 .ts 없음: .go/.js/.mjs/.md/.sh).
- 승인이어도 남는 우려: 저장본 검색은 페이지 안에서만 돌고, 검색어가 걸린 채 넘어간 페이지에 일치가 없으면 `empty()` 가 이동 바까지 덮습니다(검색창을 비우면 offset 0 으로 복구되므로 갇히지는 않음, 실행 감사 탭도 예전부터 같은 모양). 프로필의 '빈 페이지 갇힘' 계열이니 다음 회차가 손대면 여기부터.
- 릴리즈 노트에 쓸 것: 관리자 이력 '저장된 결과' 탭 100건 벽·무반응 검색창 해소, 스모크 브라우저 실패 원인 한 줄 안내, livedb DDL 검증이 node 경고에 깨지지 않도록. 서버·스키마 변경 없음.
- [러너 05:09] review approved — 리뷰 승인 (risk=low)
- [러너 05:09] pr created — https://github.com/hkjang/DartFly/pull/20
- [러너 05:10] ci failed — 성공이 아닌 검사: livedb=failure, smoke=failure, test=failure
