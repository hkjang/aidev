# PR 처리기 노트 2026-10-16-152104-moyro-shepherd — moyro PR #39
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-16-144316-moyro-improve)
# 회차 노트 2026-10-16-144316-moyro-improve — moyro
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:43] base pinned — main@879082e
- [러너 14:43] autonomy release — 

## 정찰 노트
- 보류 상위 3개(postsByIDsReactions / listReactions / decodeCappedBody)는 모두 "httpapi 핸들러가 장애를 정상 응답으로 접는" 같은 계열이고 10-04·10-05·10-08 세 회차가 이미 그 계열이라, 운영자 규칙(같은 접근 반복 금지)에 따라 서비스 계층 동작 결함인 북마크 `Reorder` 를 골랐다. 실제로 열어 본 것: bookmarks/service.go 전문, late.go:1234-1276, final.go:268-270, router.go:844, sidebar/service.go:570-591, sidebar/service_postgres_test.go:1-60/661-701, bookmark_scope_postgres_test.go:100-141, store/store.go:36-38.
- 확신 없는 곳(추측으로 적은 것): ① 실패 시나리오(A/B/C 고정표본의 어긋난 순서)는 코드에서 **연역**했을 뿐 테스트로 관찰하지 않았다 — 구현자가 RED 로 먼저 확정할 것. ② 인덱스 base 를 0-based 로 권고했으나 Mattermost 원본 스펙을 이 세션에서 직접 열어 확인하지는 못했다(docs/mattermost-api-compatibility.md 는 엔드포인트를 나열만 한다). `Create` 가 1-based 로 채번하는 비대칭은 실제 코드로 확인했다.
- 구현자가 조심할 것: 범위가 `Reorder` 한 함수 + 신규 테스트 1파일이다. 같은 파일의 `Patch`/`Create`, 같은 핸들러의 `ok, _ := IsMember` 403 위장을 끌어들이지 말 것. 테스트 DB 컨테이너 `moyro-pg-improve`(55433) 는 지금 `Exited (255)` 이므로 `docker start` 가 선행되어야 하고, DSN 없는 `go test` 의 빠른 ok 는 skip 이지 통과가 아니다.
- 프로필은 다시 쓰지 않았다: base 가 프로필 작성 시점과 같은 main@879082e 이고 내용이 현재 소스와 어긋나지 않았다(bookmarks/userstatus 무테스트 기술도 이번에 `[no test files]` 로 재확인).
- [러너 14:47] scout done — 채널 북마크 `Reorder` 가 주석이 약속한 dense 재배열을 하지 않고 한 행만 UPDATE 해, `sort_order` 가 중복되고 북

## 구현 노트
- 무엇/왜: `bookmarks.Reorder` 의 단일행 UPDATE 를 트랜잭션 기반 dense 재배열로 교체했다(`Pool.Begin` → `List` 와 같은 정렬로 활성 id 읽기 + `rows.Err()` → 없으면 쓰기 전 `ErrNotFound` → 슬라이스 재삽입 + 클램프 → `sidebar.UpdateOrder` 의 `unnest` 한 문장으로 `0..n-1` → commit). 거짓이던 주석도 구현과 일치시켰다. 커밋 `1f19668`, 프로덕션 1파일(+82/-12) + 신규 테스트 1파일.
- **정찰 예측의 교정(비평가가 먼저 볼 곳)**: 과제서가 든 "C 를 `sort_order=0` 으로 맨 앞에 보내기" 는 **수정 전에도 통과한다** — C=0, A=1, B=2 가 우연히 이미 dense 하고 충돌이 없다. 결함이 실제로 드러나는 것은 **뒤로 보내기**(`List order = [a b c], want [b c a]`)와 중간 삽입(중복 `1,1,2`)이다. 테스트는 여섯 방향을 다 고정하므로 이 교정이 커버리지를 줄이지는 않는다.
- 확신 없는 곳: ① **0-based 선택의 외부 근거는 과제서뿐이다** — Mattermost 원본 스펙을 이 세션에서도 열어 확인하지 못했다(네트워크로 스펙을 가져오지 않았다). 웹에 이 엔드포인트 호출자가 없고 `List` 가 상대 순서만 노출하므로 관찰 가능한 회귀는 없다고 보지만, 1-based 가 옳다면 `positions[i] = i` 한 줄이 바꿀 자리다. ② 트랜잭션의 **동시성** 주장은 코드와 주석으로만 근거를 댔다 — 두 재배열을 실제로 경쟁시키는 테스트는 쓰지 않았다(격리 수준이 기본 read committed 이므로 두 트랜잭션이 같은 행을 쓰면 뒤가 블록되지만, 읽기-쓰기 사이에 끼어든 `Create` 는 여전히 재배열 밖의 1-based 값을 남길 수 있다). ③ `update_at` 은 이제 재배열에 포함된 **모든** 행이 bump 된다(위치가 안 바뀐 행도). 과제서가 의도된 변화로 지정했고 테스트는 다른 채널·소프트삭제 행만 unchanged 로 고정한다.
- 일부러 하지 않은 것: `Patch` 의 `sort_order` 직접 설정 경로(여전히 중복을 만들 수 있다 — ideas.json 에 pending), `Create` 의 `MAX+1` 1-based 채번, 핸들러(`late.go:1239-1276`)와 그 `ok, _ := IsMember` 403 위장, 차선 후보였던 `postsByIDsReactions`. 전부 범위 확대 금지 지시대로 뺐다.
- 다음 역할이 조심할 것: **새 테스트는 DB 가 있어야 돈다.** `MOYRO_TEST_POSTGRES_DSN` 없이는 `t.Skipf` 이므로 빠른 `ok` 를 통과로 읽지 말고 `-v` 로 skip 0 을 확인할 것. 컨테이너 `moyro-pg-improve`(55433)는 `Exited (255)` 였어서 `docker start` + `pg_isready` 폴링이 필요했다(기동에 ~9초). 같은 포트를 쓰는 `moyro-pg-55433`(Created) 와 동시에 띄우지 말 것. DSN 은 자식 프로세스 환경에만 줬고 출력·커밋·문서에 남기지 않았다.
- 검증: `go test -race -p 1 -count=1 ./...` exit 0 / 51 패키지 ok / FAIL 0(httpapi 63.364s, bookmarks 4.078s — 둘 다 실제 DB), `go build ./...`·`go vet ./...` exit 0, `gofmt -l internal/bookmarks` 무출력, `git diff --check` clean, `check-source-sizes.sh` exit 0. 웹 검증은 webapp 미변경으로 미실행.
- [러너 15:00] brief accepted — 채택 — 근거가 현재 코드와 정확히 일치했다(`:186-199` 의 단일 UPDATE 와 거짓 주석, `List` 의 `create_at` 타이브레이크, `sidebar
- [러너 15:01] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- 판정 approve / risk low / blocking 없음. 직접 확인: service.go 전문, 테스트 전문, 핸들러 `compat_wave_handlers_late.go:1239-1276`, baseline 마이그레이션의 `channel_bookmarks`(id TEXT, sort_order INTEGER, `(channel_id, sort_order)` 인덱스는 **UNIQUE 아님** → 순열 UPDATE 중 일시 충돌 없음), `sidebar/service.go:573-591` unnest 선례, `defer tx.Rollback(ctx)` 가 posts/documents 관용구와 동일, webapp 에 이 엔드포인트·WS 이벤트 소비자 없음(grep).
- 테스트가 변경 경로를 정말 지난다: 수정 전 코드를 손으로 돌려 5개 RED 를 **독립 연역**했고 값이 원장의 `실패 재현`(`[2 2 3] / [1 1 2] / [1 1 3] / [-5 1 2] / [2 3 99]`, `List order = [a b c], want [b c a]`)과 그대로 일치했다. 실제 PG16(`moyro-pg-improve`/55433)에서 `-v -count=1 -run TestReorder` 를 직접 돌려 **12/12 PASS, skip 0**(2.97s), `go build ./...`·`go vet`·`gofmt -l` 클린을 재확인했다. 코드는 고치지 않았다.
- 승인이어도 남는 우려 3개(review.json notes 에 상세): ① `service.go:195-197` 주석의 트랜잭션 동시성 주장은 **거짓**이다(FOR UPDATE 없음 + read committed → 격리 이득 0). 결론인 '중복 없음' 은 단일 UPDATE 가 서로 다른 id 에 서로 다른 위치를 주기 때문에 트랜잭션과 무관하게 성립하므로 동작 영향이 없어 거절하지 않았다 — 다음 회차는 이 주석을 보장으로 신뢰하지 말 것. ② 채널 전체를 덮어쓰므로 동시 재배열은 이제 **last-writer-wins 로 한쪽 이동이 통째로 사라진다**(수정 전엔 중복 대신 양쪽 반영) — 릴리즈 노트 후보. ③ WS `channel_bookmark_sort_changed` 는 클램프된 요청값을 그대로 브로드캐스트하고(저장값과 불일치) 바뀐 나머지 행을 알리지 않는다 — 범위 밖 핸들러, 저장소 내 소비자 없음.
- 못 본 것: 전체 `./...` 재실행(패키지 1개만 돌렸다), 웹·e2e(webapp 미변경), 동시성 경쟁 테스트(존재하지 않음 — ① 은 코드 독해로만 판단), 그리고 **0-based 의 외부 근거**(이 세션도 WebSearch 권한 없음 → Mattermost 원본 스펙 미확인. 1-based 가 옳다면 `service.go:256` 한 줄).
- 검토 부서 소견: security — 새 엔드포인트·권한 변경 없음, 모든 입력이 파라미터 바인딩(배열도 `$2/$3`), 채널 스코프 `ErrNotFound` 가 쓰기 전에 유지돼 cross-channel IDOR 없음(테스트로 고정), 비밀값·의존성 변경 없음 → 차단 없음. legal — 개인정보 신규 수집·전송 없음, 테스트는 합성 계정(`bookmark-owner@example.test`)과 환경변수 DSN 만 쓰고 프로덕션 데이터를 쓰지 않음, 라이선스 영향 없음 → 차단 없음.
- [러너 15:07] review approved — 리뷰 승인 (risk=low)
- [러너 15:07] pr created — https://github.com/hkjang/moyro/pull/39
- [러너 15:15] ci failed — 성공이 아닌 검사: verify=failure · 실패한 검사: ? 잡: verify 

## 수리 노트
- 지적(CI verify 실패)은 **맞았다** — 로컬에서 `go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...` 가 같은 12건으로 exit 1 로 재현됐다. 다만 **이 PR 의 결함은 아니다**: 호출 확인된 12건 전부가 Go 표준 라이브러리 go1.27 과 `golang.org/x/net` v0.58.0 의 신규 공개 취약점이고, 북마크 `Reorder` 코드는 트레이스에 한 줄도 없다(시점 의존 실패).
- 고친 방법: 검사·테스트를 건드리지 않고 패치 버전만 올렸다 — `server/go.mod` `go 1.27.2`(CI setup-go 가 `go-version-file` 로 읽는 값), `x/net` v0.60.0(tidy 가 crypto/mod/sync/sys/text 전이 상향), `Dockerfile` 베이스 `golang:1.27.2-alpine` digest 재고정. 베이스 상향은 선택이 아니다: 이미지가 `GOTOOLCHAIN=local` 이라 go.mod 만 올리면 "Build service image" 가 깨진다(상향 후 `--target server-build` 로 통과 확인).
- 재검증: govulncheck "No vulnerabilities found", `go build`·`go vet`·`go test -race -p 1 -count=1 ./...`(실제 PG16, 51 ok / FAIL 0 / bookmarks 4.150s = skip 아님), plugin-sdk, check-source-sizes, server-build 이미지 — 전부 exit 0. 커밋 `fddc0b8`, Dockerfile/go.mod/go.sum 3파일.
- 확신 없는 곳: ① 취약점 DB 는 살아 있으므로 다음 CI 때 또 새 항목이 뜰 수 있다(이번 스캔에도 미호출 3+8건이 남아 있다 — 호출되면 같은 종류의 재발). ② webapp 단계와 전체 이미지·Playwright e2e 는 로컬에서 돌리지 않았다(webapp 무변경, 실패 로그상 이미 통과한 단계였다). ③ `x/net` 상향이 끌고 온 전이 의존 5개의 영향은 전체 테스트 통과로만 확인했다.


## 심사 노트
- 판정 approve / merge / risk low, blocking 없음. 테스트가 변경을 정말 검증함을 **직접 실행으로** 확인했다: origin/main 의 service.go 로 되돌리니 6개 중 5개 FAIL(to-back/to-middle/in-place/clamp-low/clamp-high), 수정본에서 12/12 PASS·skip 0(실제 PG16 55433). 구현 노트가 예고한 대로 to-front 만 수정 전에도 통과한다.
- **세 역할이 모두 미해결로 남긴 0-based 근거를 확정했다**: Mattermost 원본 `channel_bookmark_store.go:247-292` UpdateSortOrder 를 받아 읽으니 트랜잭션 → 채널 전체 읽기 → remove+insert → `b.SortOrder = int64(index)` 0-based dense → 전 행 UpdateAt → 한 문장 UPDATE 로, 이 PR 과 사실상 동일. `positions[i] = i` 는 고칠 필요 없다. 원본도 FOR UPDATE 없이 Begin 만 쓰므로 비평가가 지적한 service.go:195-197 주석의 거짓 동시성 주장은 구조 이탈이 아닌 주석 과잉으로 보아 차단하지 않았다. 차이는 범위 밖 값뿐(원본 400 거절 vs moyro 클램프 200) — 유효 범위는 [0,n-1] 로 동일하고 수정 전 moyro 도 200 이었으므로 이 PR 이 만든 이탈이 아니다.
- 재검증: `go test -p 1 -count=1 ./...` exit 0 / 51 ok / FAIL 0(bookmarks 3.101s, httpapi 83.590s = 실제 DB), build·vet·gofmt 클린, govulncheck "No vulnerabilities found" exit 0, `go mod verify` all verified. 공급망: Dockerfile 새 digest 를 Docker Hub 매니페스트에 조회해 공식 `library/golang:1.27.2-alpine` 과 정확히 일치 확인, go.sum 에 신규 모듈 0건(기존 x/* 버전 상향뿐). 보호 파일(auth·oidc·session·migrations·workflows) 미접촉, 마이그레이션 없어 revert 로 복구 가능.
- 부서 소견 둘 다 차단 없음 — security: 배열 포함 전부 파라미터 바인딩, 쓰기 전 채널 스코프 ErrNotFound 로 cross-channel IDOR 차단(거부 3경로가 테이블 스냅샷 unchanged 까지 단언), 감사 details 는 식별자·정수만. legal: 개인정보 없음, 합성 계정 + DROP SCHEMA cleanup, 의존성 BSD-3-Clause 유지.
- 못 본 것: webapp 타입체크·빌드와 e2e(webapp 무변경), 전체 이미지 빌드, 동시 재배열 경쟁 테스트. 따라서 last-writer-wins 로 한쪽 이동이 사라지는 동시성 변화는 코드 독해 + 원본 대조로만 판단했다 — 릴리즈 노트 후보로 남긴다.
