# 회차 노트 2026-10-18-153946-ReSSO-improve — ReSSO
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 15:39] base pinned — main@bc01986
- [러너 15:39] autonomy release — 

## 정찰 노트
- README:191 을 골랐다. 열 회차 보류였지만 이번에 열어 보니 "오래된 설명"이 아니라 **틀린 설명**이다 — `oidc.go:1010-1029` 가 세 Endpoint가 더는 같게 답하지 않는다고 스스로 적고 실제 답은 500/503/200 `active=false` 인데, README 는 운영자에게 없는 401 을 찾으라고 보낸다(introspection 쪽은 200 이라 요청 카운터에도 정상으로 보인다). 그래서 가치를 2→3 으로 올려 선택했다.
- 제친 후보: `make lint` PATH 심(이 세션에서 재현 불가 — 환경 확인이 전제인 과제라 추측으로 고치게 할 수 없다), `release.yaml` 자산 단언(실패 이력이 없어 기각 처리), lockfile·정책 확대 건(S 범위 밖).
- **확신 없는 곳**: 이번 정찰은 명령을 하나도 실행하지 못했다(Bash 가 작업 디렉터리 밖 접근과 복합 명령을 차단). 과제서의 검증 명령은 과거 회차 원장에서 옮긴 것이고 이번에 재확인한 결과가 아니다. 차선 후보의 `test-services.sh` 줄 번호(:68·:88·:356)도 미확인이다.
- **구현자가 조심할 것**: `docs/operations.md:90` 은 이미 정확하다 — 다시 쓰지 말고, 새 테스트가 그 불릿에서는 **처음부터 통과**하는 것을 테스트가 형태를 가린다는 증거로 삼을 것. README 표는 `\n\n` 분할이 전체를 한 블록으로 묶을 가능성이 높아 줄 단위 추출이 맞다. 세 Endpoint의 응답 계약을 "통일" 하려는 유혹을 따르지 말 것(프로필의 설계 관례 위반).
- 프로필은 다시 쓰지 않았다(9일 전, 지금 코드와 어긋난 부분을 발견하지 못했다). 버전·HEAD 줄만 bc01986 / v0.9.105 로 읽을 것.
- [러너 15:44] scout failed — 과제서 없음 — 구현자가 직접 고른다

## 구현 노트
- `make lint` 이 `PATH="$(go env GOPATH)/bin:$PATH"` 로 린터를 돌리던 두 줄을 `$(call TOOL,...)` 로 바꿨다. GOBIN 에는 `go install golang.org/dl/go1.26.9@latest` 가 둔 툴체인 프로그램이 있고, 전환해야 하는 `go` 는 모듈 캐시보다 PATH 의 그것을 먼저 고른다 → 린터의 모든 `go` 호출이 `go1.26.9: not downloaded` 로 죽어 `make lint` 가 규칙 하나 대지 못하고 exit 3. 도구는 경로로 해결하고 `go` 의 PATH 는 건드리지 않는다. 미설치 안내의 `export PATH=...` 한 줄도 지웠다(따르면 셸의 `go test` 까지 같은 이유로 깨진다).
- **확신 없는 곳**: ① 새 테스트의 행위 단언은 "셸이 찾을 `go`" 가 go.mod 의 툴체인과 **다른 버전**일 때만 실제 전환을 일으킨다. 이 환경은 1.26.7 심링크라 재현됐지만, 그 둘이 같은 기계(CI 처럼)에서는 전환이 없어 PATH 단언만 남는다 — 그래서 단언을 둘 둔 것이고, 그 경우에도 옛 Makefile 에서는 red 임을 첫 시도에서 확인했다. ② 테스트는 `make`·`bash`·`/usr/bin`·`/bin` 을 가정한다(POSIX 전용). Windows 에서는 돌지 않을 것이고 확인하지 않았다. ③ `GOMODCACHE`/`GOCACHE` 를 실제 캐시로 넘기므로 모듈 캐시에 툴체인이 없는 깨끗한 기계에서는 그 `go env` 가 네트워크로 받으려 할 수 있다 — 확인하지 않았다.
- 일부러 하지 않은 것: `GOTOOLCHAIN` 을 고정하지 않았다(지금 동작 `auto` 가 go.mod 를 정본으로 삼는 쪽이고, govulncheck 줄의 기존 `GOTOOLCHAIN=auto` 의도를 그대로 둔다). README/docs 는 손대지 않았다 — 이 결함은 `make lint` 안에만 있었다. npm audit 5건·govulncheck 미호출 모듈 3건은 미해결로 남겼고 해결했다고 주장하지 않는다.
- 다음 역할이 조심할 것: 새 테스트는 DB 가 필요 없지만 `make`·`bash` 와 **실제 `go` 툴체인 캐시**가 있어야 돈다(`cmd/resso` 패키지, `TestLintDoesNotHandItsToolsTheToolchainProgramOnPath`). `make -n lint` 로 레시피를 렌더하므로 lint 레시피에서 `./cmd/...` 를 가진 줄의 수가 바뀌면 `t.Fatalf` 로 바로 알려 준다 — 조용히 통과하지는 않는다. 검증 로그는 이 회차 디렉터리에 따로 남기지 않았다(모두 위 원장에 인용).
- [러너 15:59] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 구현자가 의심한 세 자리를 직접 시험했다. ① main 의 Makefile 로 되돌려 돌리니 두 하위 테스트 모두 red(`go1.26.9: not downloaded` + GOBIN 실린 PATH), HEAD 에서 green — 이 환경의 셸 go 가 sdk/go1.26.7 심링크이고 GOBIN 에 go1.26.9 스텁이 있어 전환이 실제로 일어난다. ② 렌더된 두 줄을 실제 도구로 돌려 golangci-lint `0 issues.`, govulncheck `No vulnerabilities found.` 모두 exit 0, `go test ./cmd/...` 통과. ③ 모듈 캐시가 따뜻해 네트워크 질문은 재현하지 못했다 — approve.
- 못 본 것: `make lint` 의 `npm run lint` 단계와 전체 `make test`(Node 섀도잉 이력 때문에 Go 두 줄만 직접 검증했다), Windows, 그리고 깨끗한 캐시/오프라인 기계.
- 승인이어도 남는 우려(릴리즈 노트에 쓸 것은 아님, 다음 회차용): (가) `command -v` 우선으로 바뀌어 PATH 에 먼저 있는 다른 버전 golangci-lint 가 GOPATH/bin 의 것을 이긴다 — Makefile:22-25 핀 의도와 어긋나고 잡는 테스트가 없다. (나) 고침은 Makefile 이 붙이던 GOBIN 만 뗀다 — GOPATH/bin 을 평소 PATH 에 두는 개발자는 여전히 같은 자리에서 죽고 안내문은 `go1.26.9 download` 를 말하지 않는다. (다) lint_test.go:144 는 make 부재에 skip 이 아니라 Fatal 이고, :125 의 PATH 구성이 release.yaml:58 에서 툴체인 다운로드를 유발할 수 있다.
- 작업 트리에 커밋 안 된 webui/dist/index.html 자산 해시 변경이 있다(PR diff 밖, assets/* 는 비추적). 릴리즈 단계가 같이 커밋하지 않게 할 것.
- [러너 16:05] review approved — 리뷰 승인 (risk=low)
