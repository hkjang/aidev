# 회차 노트 2026-09-25-104126-git-ctx-improve — git-ctx
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 10:41] base pinned — main@ad093d9
- [러너 10:41] autonomy release — 

## 정찰 노트
- `mcp.cacheKey` 의 제자리 정렬을 골랐다: 별칭 사슬(cache.go:32 → args.go:171 → search/service.go:232)을 코드로 끝까지 확인했고, 같은 함수 34번 줄이 AllowedRepositories 를 복사한 뒤 정렬하는 비대칭이 누락의 증거다. S·위험1 로 45분 안에 끝난다.
- 제친 후보: 예산 초과 2건(budget.go responseNoticeBytes 320, dispatch.go finishCall 의 후처리 노트) — 전자는 초과분이 몇 바이트 규모라 가치가 없고, 후자는 "예산이 노트를 잘라선 안 된다"는 의도가 주석에 명시돼 있어 버그로 제출하면 반려 위험이 크다. manifest 파서 확장 3건은 이 저장소의 반려 이력 유형이라 제외.
- 확신 없는 곳: **순서에 의존하는 소비자를 찾지 못했다** — 사용자에게 보이는 잘못된 출력은 없을 가능성이 높다. 과제서에 그렇게 명시했으니 구현자는 없는 피해를 만들어 쓰지 말고 "함수가 입력을 변경한다"는 사실 자체를 증명 대상으로 삼을 것.
- 구현자가 조심할 것: `s.search` 가 구체 타입이라 스파이 주입이 불가능하다(대역 금지 교훈과 충돌하니 억지로 끼우지 말 것). `WithUnrestricted` 를 고치려면 호출자 3곳(mcp/args.go, app/health.go:616, app/auth.go:923) 전부 확인하든 아예 손대지 말 것. mcp fixture 는 공유 in-memory SQLite 이므로 `t.Parallel()` 금지.
- 실행 확인: `go test -tags sqlite_fts5 -count=1 ./internal/mcp` → `ok 0.840s`. 프로필은 기준 커밋이 낡아 profile.md 를 새로 썼다.
- [러너 10:48] scout done — `mcp.cacheKey` 가 호출자의 ACL 주체 슬라이스를 제자리 정렬하지 않게 복사 (가치 2 / 위험 1 / 작업량 S)

## 구현 노트
- 무엇을 왜: `internal/mcp/cache.go:32` 을 `principals := append([]string(nil), principalACLs(p)...)` 로 바꿔 캐시 키 계산이 호출자의 `p.ACLPrincipals` 를 제자리 정렬하지 않게 했다(같은 함수 34번 줄의 `AllowedRepositories` 관용구와 동일). `sort.Strings` 는 그대로 — 순서 무관 키가 본래 목적.
- 검증: TDD 로 `internal/mcp/cache_test.go` 에 `TestCacheKeyLeavesCallerPrincipalsUntouched`(실제 sqlite fixture·실제 `auth.Principal`, 제한/무제한 테이블) 추가. 수정 전 제한 분기만 FAIL(`got [alice middle zeta], want [zeta alice middle]`), 복사 경로인 무제한 분기는 PASS — 과제서 예측과 일치. 수정 후 mcp(0.859s)·mcp race(3.953s)·gofmt·vet·build·전체 `go test -tags sqlite_fts5 ./...`(32 ok, app 100.580s) 전부 exit 0.
- **확신 없는 곳**: 이 변경으로 고쳐지는 *사용자 관측 가능한* 오동작은 찾지 못했다 — 주체 순서에 의존하는 소비자를 확인하지 못했고(ACL 은 SQL `IN` 절), 테스트가 고정하는 것은 "키 계산 함수가 입력을 변경하지 않는다"는 사실 자체다. 이것을 사용자 피해로 포장하지 말 것.
- 일부러 하지 않은 것: `search.WithUnrestricted` 를 항상 복사하게 바꾸는 선택안 — 호출자 3곳(`mcp/args.go`, `app/health.go:616`, `app/auth.go:923`)을 다 확인할 근거가 없어 "전부 확인하든 아예 손대지 말든" 규칙에 따라 손대지 않았다. 차선 후보(예산 초과 2건)도 손대지 않았다.
- 다음 역할이 조심할 것: 새 테스트는 공유 in-memory SQLite fixture 를 쓰므로 `t.Parallel()` 을 붙이면 안 된다. `contentsecurity.Revision` 은 건드리지 않았으므로 재색인 지문 변화 없음. 릴리즈·버전 파일도 무변경(commit 7ddab8b 하나뿐).
- [러너 10:52] brief accepted — 채택 — 지정한 별칭 사슬(cache.go:32-33 / args.go:170 / service.go:230-233)과 비대칭 근거가 현재 코드와 정확히 맞았고, 예측대로 
- [러너 10:54] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 확인한 것: 별칭 사슬(cache.go:35 → args.go:167 → service.go:230-233)을 코드로 검증, main@ad093d9 임시 worktree 에 새 테스트만 얹어 수정 전 restricted 분기 FAIL·수정 후 PASS 를 실측, mcp 패키지 전체 `ok 0.852s`, gofmt·vet 무출력, in-place sort 전수 조사(다른 별칭 결함 없음).
- 못 본 것: 외부 Postgres/pgvector/Vault/실브라우저/govulncheck 및 app 패키지 장기 suite — 이번 diff 가 닿지 않는 영역이라 재실행하지 않았다.
- 승인이어도 남는 우려: unrestricted 분기는 수정 전에도 PASS 하므로 load-bearing 한 단언은 restricted 하나다. 사용자 관측 가능한 오동작은 나도 찾지 못했다.
- **릴리즈 노트 주의**: ACL 우회·권한 버그·캐시 오염으로 쓰면 과장이다. "cacheKey 가 호출자 슬라이스를 제자리 정렬하지 않게 함"까지만 적을 것.
- 다음 회차: `search.WithUnrestricted` 가 제한 주체일 때 입력을 그대로 반환하는 구조는 그대로 남아 있다(호출자 3곳). 같은 유형의 후속 과제로 쓰려면 3곳 전부를 근거와 함께 다룰 것.
- [러너 10:56] review approved — 리뷰 승인 (risk=low)
- [러너 10:57] pr created — https://github.com/hkjang/git-ctx/pull/38
- [러너 11:04] ci passed — 검사 5개 모두 success
- [러너 11:04] merge done — 7ddab8b
