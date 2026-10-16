# 재현 결과 — 과제서의 전제가 관찰과 어긋남 (2026-10-17)

`gh` 는 미인증이라 `gh run list` 를 쓸 수 없었으나, **인증 없는 GitHub REST API 는 읽을 수 있었다.**
아래는 그 실제 출력이다(추측 아님).

## 1. "Offline image release" 가 v0.9.104 에서 두 번 실패했다 → 거짓

`GET /repos/hkjang/ReSSO/actions/workflows/release.yaml/runs?per_page=100` (total 133):

    38958145291 v0.9.104 push completed success att 1 2026-10-16T19:52:47Z edf2c2f1
    37854453193 v0.9.103 push completed success att 1 2026-10-08T22:35:42Z 2dc5657f
    37691498799 v0.9.102 ... success att 1
    37549327281 v0.9.101 ... success att 1
    37346306391 v0.9.100 ... success att 1

    non-success runs: [v0.9.80, v0.9.64 … v0.9.43]   # 전부 2026-08 이전
    attempts>1: []                                   # 재실행 자체가 한 건도 없다

v0.9.104 의 릴리즈 런은 **1회, 첫 시도에 success**. 재실행 기록 0건이므로 "두 번 실패" 는 성립하지 않는다.

## 2. 오프라인 이미지 자산이 Release 에 붙지 않았다 → 거짓

`GET /repos/hkjang/ReSSO/releases/tags/v0.9.104` → id 415331003, published_at 2026-10-16T19:57:34Z
`GET /repos/hkjang/ReSSO/releases/415331003/assets`:

    release-sha256.txt       88       2026-10-16T19:57:32Z
    resso-v0.9.104.tar.gz    5500057  2026-10-16T19:57:32Z

두 자산이 모두 붙어 있다. 즉 과제서의 후보 4(자산 목록 완전일치 단언)도 통과했다는 뜻이다.

## 3. 실제로 실패한 런은 단 하나 — 그리고 이미 고쳐져 머지되었다

`GET /repos/hkjang/ReSSO/actions/runs?per_page=40` 중 non-success 는 1건뿐:

    38950481314 CI | auto/2026-10-17-0321 | pull_request | completed failure att 1 2026-10-16T18:51:01Z 8d3b77ac

`GET .../runs/38950481314/jobs`:

    JOB: lint   failure
        4 Lint backend                        -> success
        5 Check for known vulnerabilities     -> failure
        6 Run actions/setup-node@v7           -> skipped
    JOB: verify failure
        6 Test backend                        -> success
        9 Build image                         -> success
       10 Scan image for known vulnerabilities -> failure

실패 단계는 **govulncheck 두 곳**(백엔드 스캔·이미지 스캔)이고, 원인은 Go 1.26.7 의 stdlib
취약점이다 — 지난 회차 원장이 이미 그렇게 적었다. 같은 PR 의 다음 커밋 `dcfd26f`
(toolchain/Dockerfile → 1.26.9)가 그것을 고쳤고, 그 커밋의 CI 런 38953981509 는
`verify success / lint success` 이며 PR #43 은 443a309 로 머지되었다.

러너가 적재한 "CI failed, PR open #43" 은 그 **첫 런**(8d3b77a)의 상태이고, PR 은 그 뒤 통과·머지되었다.

## 4. 과제서 후보들의 로컬 재현 — 전부 통과(= 실패를 재현할 수 없음)

HEAD = edf2c2f (태그 커밋 그 자체, worktree 가 이미 이 커밋에 있다).

- 후보 1 (`golang:1.26.9-alpine` 부재): `docker manifest inspect golang:1.26.9-alpine` → **EXISTS**.
  `go version` → go1.26.9 (GOPATH/bin 에 go1.26.9 심 존재, go.mod toolchain 해석됨).
- 후보 3 (핀 대조): `go test ./cmd/resso -count=1 -run 'Pin|Version|Toolchain|Compose|Postgres|Linter|Node' -v`
  → 다섯 개 전부 PASS (Linter/ImageVersion/Toolchain/Compose/Postgres).
- 후보 2 (릴리즈 잡 모양의 백엔드 검증): PostgreSQL 만, LDAP 변수 unset, `-race` 없이
  `go test ./...` → **exit 0** (httpserver 85.186s, store 46.438s, 전 패키지 ok),
  `go vet ./...` → **exit 0**.
- 후보 4: 위 2번에서 두 자산이 실제로 붙은 것으로 반증.

**결론: 고칠 실패가 없다.** 체계적 디버깅의 1단계(결정적 재현)에서 멈추는 것이 맞고,
재현 없이 후보 하나를 "고치는" 것은 과제서 자신이 금지한 행위다. 따라서 차선 후보로 넘어간다.
