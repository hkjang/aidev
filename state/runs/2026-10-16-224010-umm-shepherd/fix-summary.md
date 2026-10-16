# fix-summary — umm PR #172 (수리: 커밋 없음)

- **지적은 이 변경의 결함이 아니다.** CI `verify` 는 `Go vulnerability scan`(ci.yml:66) 에서 죽었고, 이 PR 의 diff 는 `web/src/lib/edge-vocabulary.test.ts` **1파일 뿐**(`git diff origin/main...HEAD --stat`)이라 govulncheck 의 입력(Go 소스·go.mod)은 base 와 바이트 단위로 동일하다.
- **측정으로 귀속 확인:** HEAD(a5959a5)에서 `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` → exit 3, stdlib 취약점 12건. `origin/main`(745dcf3) 을 임시 worktree 로 떼어 **같은 명령** → 동일하게 exit 3, 동일한 12건(GO-2026-6599/6600 … `Fixed in: html/template@go1.26.9`). PR 유무와 무관하다 = 취약점 DB 가 판정을 바꿨다.
- **그래서 고치지 않았다.** 진짜 해결은 `go.mod` 의 `toolchain go1.26.7` → `go1.26.9` **와** `Dockerfile:9` 의 `golang:1.26.7-alpine` 동반 상향이고, 둘 다 이 PR 의 목적(프론트 시험 1파일)과 무관한 릴리즈 경로·위험 구역이며 실제 `docker build` 검증을 요구한다. 저장소 전례도 전용 PR(#170/#171, x/text)이다. 절대 규칙("무관한 파일 금지", "지적이 틀렸으면 커밋 없이 끝")에 따라 **커밋 0개**.
- **이 PR 자신의 게이트는 전부 초록** (`npm ci --prefix web` 선행 후): typecheck 통과, lint 통과(신규 파일 경고 0, 기존 AdminPage.tsx 경고만), `check-i18n` 1060키 통과, `npm --prefix web test` **24파일/260시험 통과**, prettier 통과. CI 는 govulncheck 에서 죽어 프론트 절을 **한 번도 돌리지 못했다**.
- **다음 사람에게:** #172 는 그대로 머지 가능하고, 막는 것은 base 의 Go 툴체인이다. govulncheck 녹화는 go1.26.9 상향 **전용 PR** 로 분리하라(go.mod + Dockerfile 동시, docker build 로 확인). 내가 안 돌린 것: Go 테스트·DB 절·e2e·`npm audit`·`verify:pwa`(이 변경이 닿지 않는 영역).
