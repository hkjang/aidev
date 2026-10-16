# 회차 노트 2026-10-17-013604-Momento-improve — Momento
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 01:36] base pinned — main@2734aff
- [러너 01:36] autonomy release — 

## 정찰 노트
- `make verify` 를 골랐다: 최근 verify-failed 두 건이 모두 `govulncheck` 단계였고 `Makefile` 에 `-race|govulncheck|npm audit|npm ci` 가 **0건**(grep)이라 원인이 코드가 아니라 로컬 게이트 구멍임을 확인했다. 화면 문구 후보들(가치 2)보다 가치가 높고, 프로덕션 파일 1개(Makefile)+테스트 1개로 끝난다.
- 차선은 `cidrRule.ts` 호스트 비트 안내다. PR #32 머지와 `normalizeCIDR`(admin.go:820) 를 눈으로 확인해 전제가 확정됐고, 기존 테스트가 문장이 아니라 주소 정규식만 보는 것도 확인했다 — 1순위가 막히면 바로 쓸 수 있다.
- 추측으로 적은 것: 2026-10-08 의 같은 취지 「make verify」 과제가 **왜 no-change 로 끝났는지는 확인하지 못했다**. 막히면 범위를 Makefile 한 타깃으로 줄이고 무엇이 막았는지 적으라고 과제서에 지시했다.
- 구현자가 조심할 것: ci.yml 은 **읽기만**(빨간불 증명용 임시 수정은 반드시 `git checkout` 으로 되돌릴 것), `make test` 는 지우지 말 것, 가드를 문자열 전체 동일성으로 만들면 `docker build` 인자 차이로 곧 거짓 실패가 난다.
- 실행한 것은 `go test ./internal/version/`(ok, 0.004s) 하나뿐이다. `make verify` 자체의 소요 시간·통과 여부는 미확인 — 구현자가 끝까지 한 번 돌려 exit 0 을 찍어야 한다.
- [러너 01:41] scout done — `make verify` 를 만들어 CI 의 게이트를 로컬에서 빠짐없이 돌리고, ci.yml 에 게이트가 늘어났는데 Makefile 이 �

## 구현 노트
- 무엇을 왜: `Makefile` 에 `verify:` 타깃(ci.yml `test` job 의 다섯 명령, 같은 글자·같은 순서)과 `internal/version/buildgates_test.go` 드리프트 가드. `make test` 는 무변경 — 빠른 루프용으로 남겼다. 커밋 964f507, 프로덕션 1 + 테스트 1 파일.
- 실제로 돌린 것: **`make verify` 끝까지 `VERIFY_EXIT=0`** (govulncheck `0 vulnerabilities`, sdk 29/29 + audit 0, web 259/259 + audit 0, build ok) — 로그는 /tmp/verify.log. `make test` 따로 `MAKE_TEST_EXIT=0`. 가드의 red 세 경로(ci.yml 단계 +1 → 개수 6≠7 / govulncheck 줄 삭제 → 누락 / web 게이트를 앞으로 → 순서) 전부 실제 재현 후 복원.
- **확신 없는 곳**: (1) 가드는 **텍스트만 읽는다** — `make verify` 가 통과한다는 것도, 각 `cd` 가 그 디렉터리에서 실제 실행된다는 것도 증명하지 않는다(그래서 이번에 손으로 돌렸다). 주석에 그 한계를 적어 뒀다. (2) `run:` 단계 세기는 정규식 `^[ \t]*(?:-[ \t]+)?run:[ \t]` 이다 — 현재 ci.yml 에는 블록 스칼라 안에 `run:` 이 없어 맞지만, 누가 멀티라인 `run: |` 안에 `run:` 글자를 쓰면 과다 계수된다(그 경우 빨간불이 나므로 안전한 방향으로 틀린다). (3) `verifyRecipe` 는 `verify:` 로 시작하는 첫 줄을 찾는다 — 타깃 이름이 바뀌면 t.Fatal 로 멈춘다.
- **검증 못 한 것**: `docker build` 단계(면제)는 돌리지 않았다 — Makefile 에 문자열이 있는지만 본다. DB 경로도 미검증: DSN 이 없어 `go test -v` 가 **120건 SKIP** 한다(측정값). `make verify` 녹색이 CI 녹색이 아니라는 뜻이고 Makefile 주석에 적었다.
- 일부러 안 한 것: `.github/workflows/*.yml` 커밋 변경 0줄(증명용 한 줄은 `git checkout` + `cmp` 로 바이트 복원), `make test` 내용 변경 0, `go.mod` 의존성 추가 0(YAML 파서 대신 정규식), 차선 후보(cidrRule.ts)는 1순위가 막히지 않아 미착수.
- 다음 역할이 조심할 것: 이 가드는 **ci.yml 을 고치면 같이 고쳐야 한다** — 단계를 하나 더하면 표도 더해야 빨간불이 풀린다(그게 설계 의도다). `make verify` 는 **네트워크가 필요**하고(govulncheck·npm audit) 몇 분 걸린다; 오프라인에서는 의도적으로 실패한다. 2026-10-08 no-change 를 막은 것이 무엇인지는 **여전히 미확인** — 이번엔 막는 것이 없었다.
- [러너 01:49] brief accepted — 채택 — 인용한 근거가 **전부** 현재 코드와 일치했다: `Makefile:1` 의 `.PHONY`·`20-24` 의 `test:` 네 줄에 `-race`·`govulncheck`·`npm 
- [러너 01:49] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: main 임시 worktree 에 테스트 파일만 올려 **수정 전 FAIL**(`buildgates_test.go:122` no `verify` target)을 직접 재현 — 원장의 실패 출력과 일치. 고친 뒤 PASS, `go test ./internal/version/ -count=1` ok, `go vet` 무출력, ci.yml `run:` 6개·명령 문자열·순서 전부 대조. 임시 worktree 제거, 트리 clean.
- 못 본 것: `make verify` 전체(race·npm ci·audit·govulncheck)는 비용 때문에 재실행하지 않았다 — 구현자의 `VERIFY_EXIT=0` 을 신뢰했다. docker build 와 DB 경로도 미실행.
- 승인이어도 남는 우려 ①: `verify` 는 CI 게이트 6개 중 5개만 돌린다 — `docker build`(ci.yml:57-58)가 빠져 있는데 `Makefile:26-27` 은 "같은 명령을 같은 순서로" 라고 적고 "약속하지 않는 것 둘" 에 그 사실이 없다. 면제 단언은 verify 가 호출하지 않는 `docker:` 타깃의 문자열로도 충족된다. 수리는 Makefile 주석 한 문장 추가가 전부.
- 우려 ②: 레시피의 게이트 줄을 `\t# …` 로 주석 처리하면 가드는 녹색이고 make 는 그 줄을 셸 no-op 으로 돌린다(실측). 고의 조작만 통하므로 차단은 아니다. 또 `run:` 개수는 ci.yml **파일 전체**를 세므로 두 번째 job 이 생기면 표에 면제로 올려야 풀린다.
- 거절이 아니므로 수리 대상은 없다. 다음 회차가 이 영역을 만지면 가장 먼저 볼 파일: `Makefile:26-49`, `internal/version/buildgates_test.go:76-91`(docker 면제).
- [러너 01:53] review approved — 리뷰 승인 (risk=low)
- [러너 01:54] pr created — https://github.com/hkjang/Momento/pull/34
- [러너 01:58] ci passed — 검사 1개 모두 success
- [러너 01:58] merge done — 964f507
