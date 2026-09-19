# 회차 노트 2026-09-19-122335-igame-improve — igame
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:23] base pinned — main@3aad115
- [러너 12:23] autonomy release — 
## 정찰 노트
- [정찰 12:30] 우선 과제(릴리즈 워크플로 2회 실패 수정)를 그대로 과제로 잡았다. 이 세션은 gh·npm audit·WebFetch 승인이 전부 거부돼 실패 로그와 audit 결과를 **직접 보지 못했다** — 로컬 `go test` 전체 통과만 확인했으므로 Node audit(dev 포함) 이 유력하다는 것은 추측이다.
- 구현자는 코드를 만지기 전에 `gh run view <id> --log-failed` 로 실패 step 을 확정하고, 원인이 재현되지 않거나 일시 장애면 코드를 바꾸지 말고 차선(rank tiebreaker)으로 갈 것.
- 워크플로 완화 금지(`|| true`, `--omit=dev`, `continue-on-error`). vitest 4.x 로 내리지 말 것(npm 10.9.8 크래시). SDK 의 `vite ^7.1.3` 명시 유지.
- profile.md 를 새로 썼다(이전 프로필 없음).
- [러너 12:26] scout done — 릴리즈 워크플로가 두 번 연속 실패한 원인 진단·수정 — 워크플로는 그대로 두고 의존성/스크립트 쪽을 

## 구현 노트
- 우선 과제(릴리즈 2회 실패)는 전제가 틀렸다: GitHub API(무인증, `/repos/hkjang/igame/actions/workflows/release.yml/runs`)로 확인하니 v0.7.15 릴리즈 run #30 이 success 이고 그 뒤 실패 run 이 없다. 러너의 'error' 는 `hold: budget`. 로컬 audit(SDK·web) 0건, govulncheck 0건 — 의존성·워크플로 미변경. 차선(랭킹 tiebreaker)을 구현해 커밋 7d39665.
- 차선 후보의 진술도 틀렸다: rank 와 행 순서는 어긋나지 않는다(EXPLAIN: Limit→WindowAgg→Sort, 두 번째 정렬 없음). 실제 결함은 동점 순서가 HashAggregate 해시 순서라 무관한 점수가 들어오면 동점 1위가 바뀌는 것 — psql 과 PG 테스트로 재현(옛 쿼리: 11개 경로 중 10개 실패).
- 확신 없는 곳: 테스트가 3,000명을 심고 ANALYZE 해 HashAggregate 플랜을 유도하는데, PG 버전·설정(work_mem 등)이 다르면 플랜이 달라 옛 쿼리에서도 우연히 통과할 수 있다(고친 쿼리는 어느 플랜에서도 통과). 테스트 실행 ~3.5초/케이스×2 — `make test-db` 총 7초.
- 일부러 하지 않은 것: 동점을 "먼저 도달한 사람이 앞" 으로 바꾸는 공정성 규칙(catalog·defense 의 best CTE 를 DISTINCT ON 으로 바꿔야 하는 정책 변경) — 결정성만 넣었고 ideas.json 에 남김. 웹 화면은 `rank` 필드를 그대로 쓰므로 미변경.
- 다음 역할이 조심할 것: 새 테스트는 `IGAME_TEST_DSN` 없으면 skip — `make test-db DSN=...`(docker postgres:17-alpine 로 검증). 테스트 데이터는 username 'rank-pg-<tag>-%' 로 정리되고 users 삭제가 sessions/scores/results 로 cascade 한다. web 테스트 수 241 은 main 기준(과제서의 246~247 은 아직 main 에 없는 mail/mcp-oauth 브랜치 수치).
- [러너 12:39] brief fallback — 차선 — 실패한 릴리즈 run 이 존재하지 않고(v0.7.15 #30 success, 그 뒤 run 없음) 로컬 audit·govulncheck 도 0건이라 수정 과제의 �
- [러너 12:39] verify passed — 검증 4개 통과 (policy)
## 비평 노트
- [비평 12:55] approve(risk low). 일회용 docker postgres:17-alpine 로 새 PG 테스트를 직접 돌렸다: HEAD 22/22 통과, 프로덕션 3파일만 main 으로 되돌리면 10/11·11/11 실패 — 테스트가 수정을 실제로 잡는다. DSN 포함 internal/api 전체 8.1s 통과, 픽스처 잔여 0건.
- 확인: row_number() 쿼리 8개 전부 tiebreak 추가, `/rankings/{gameID}`·MCP 캡처도 같은 핸들러, 웹은 재정렬 없음, 범위 이탈·마이그레이션·워크플로 변경 없음. 못 본 것: 실제 CI 워크플로 실행(gh 권한 없음), 프로덕션 규모 플랜.
- 남는 우려: 테스트는 HashAggregate 플랜에 기대 옛 코드에서 실패하므로 PG 설정이 다르면 '거짓 통과'만 가능(거짓 실패 없음). 이름 tiebreak 은 collation 의존이지만 배포 내에서는 결정적.
- 릴리즈 노트에 적을 것: 동점자 노출 순서가 바뀔 수 있음(개인 user_id, 그룹 이름, RealmGuard 개인 created_at→id). 공정성 규칙(먼저 도달한 사람 우선)은 미도입 — ideas.json 참고.
- [러너 12:43] review approved — 리뷰 승인 (risk=low)
- [러너 12:43] pr created — https://github.com/hkjang/igame/pull/21
- [러너 12:48] ci passed — 검사 1개 모두 success
- [러너 12:49] merge done — 7d39665
- [러너 12:58] release published — v0.7.16
- [러너 13:14] assets verified — v0.7.16 자산 1개 (이전 v0.7.15: 1)
