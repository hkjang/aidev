# 수리 요약 — PR #172 (커밋 없음)

- **재현**: HEAD(a5959a5)에서 CI 와 같은 명령 `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` 를 돌려 같은 자리에서 같은 실패를 봤다 — stdlib 12건(GO-2026-6599/6600/6603/6605/6607/6608/6609/6610/6611/6612/6613/6617), `exit status 3` → 종료코드 1. 로컬 Go 는 CI 와 동일한 go1.26.7.
- **원인은 이 변경이 아니라 베이스**: `origin/main`(745dcf3)을 별도 worktree 로 꺼내 같은 명령을 돌려도 **똑같이 12건·exit 1**. 이 PR 의 diff 는 `web/src/lib/edge-vocabulary.test.ts` 한 파일뿐이고 Go 파일·go.mod·go.sum 변경이 0이라 Go 스캔의 입력이 베이스와 글자 하나까지 같다. 진짜 원인은 go.mod 의 `toolchain go1.26.7` 이 보안 패치 릴리즈(go1.26.9)보다 뒤진 것이며, 커밋이 아니라 취약점 DB 가 판정을 바꾼 경우다.
- **그래서 커밋 없이 끝냈다**: 고치려면 이 변경의 목적(연결 어휘 고정)과 무관한 go.mod·go.sum·Dockerfile 을 건드려야 하고, 그건 절대 규칙 2번 위반이자 #170/#171 처럼 자기 PR 로 가야 할 별건이다.
- **다음 PR 용 검증된 처방**(베이스 worktree 에서 실측): ① `toolchain go1.26.9` 만 올리면 12→4건(x/net v0.57.0 잔존) ② 여기에 `go get golang.org/x/net@v0.60.0`(crypto v0.57.0·sync v0.23.0·sys v0.48.0·text v0.42.0 동반 상승)까지 하면 `No vulnerabilities found.` **exit 0** ③ Dockerfile 9행의 `golang:1.26.7-alpine` 도 같이 `1.26.9-alpine` 으로(태그 존재는 Docker Hub 에서 확인, docker build 는 미실행).
- **이 변경 자체는 멀쩡하다**: `npm ci --prefix web` 후 `make test-web` 8게이트 전부 통과(exit 0) — offline-queue·audit(0건)·typecheck·lint·check-i18n·vitest **24파일/260시험 통과**·build·verify:pwa(150자산). CI 는 govulncheck 에서 멈춰 프런트 게이트에 도달조차 못 했다.
