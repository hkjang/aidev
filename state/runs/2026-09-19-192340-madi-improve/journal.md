# 회차 노트 2026-09-19-192340-madi-improve — madi
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:23] base pinned — main@fd2c3b5
- [러너 19:23] autonomy release — 

## 정찰 노트
- 원인 확정: 17:23 회차 `verify.txt` 한 줄 `web/embed.go:7:12: pattern all:dist: no matching files found`. 러너 auto verify 는 `go build ./...` 을 `npm run build` 보다 먼저 돌리는데(shepherd.log 99~103행 순서) 깨끗한 트리엔 gitignore 된 `web/dist` 가 없다. 11:22 shepherd(PR #6)도 같은 줄로 fix-failed. release.yml/ci.yml/Dockerfile 은 web 먼저 빌드하므로 워크플로 결함 아님 → 워크플로 무변경, 저장소 쪽 `web/dist/.gitkeep` + Vite emitFile 재생성으로 고친다(SAML return_to 재시도 등 다른 후보는 이 수정 없이는 전부 같은 자리에서 다시 죽으므로 제쳤다).
- 확신 없는 곳: Rollup `emitFile` 이 `.gitkeep` 같은 dot 파일명을 받는지 미확인(안 되면 비-dot 이름). Vite `emptyOutDir` 가 dotfile 도 지우는지는 소스 기억 기준(`.git` 만 보존) — 구현자가 빌드 뒤 `git status` 로 확인. 이 세션은 `go build` 실행 권한이 승인되지 않아 재현은 verify.txt 기록으로 대신했다.
- 구현자 주의: `.gitignore` 의 `web/dist/` 를 `web/dist/*` 로 바꿔야 `!web/dist/.gitkeep` 예외가 먹는다(`git check-ignore -v` 로 확인). 전체 `go test ./...`·브라우저 시험 금지(러너가 자체 검증으로 돌림). 83239b9 는 이 브랜치에 없으니 되살리지 말 것.
- [러너 19:28] scout done — 깨끗한 체크아웃에서 `go build ./...` 이 통과하도록 `web/dist` 자리표시자(`.gitkeep`)를 커밋하고 Vite 빌드가 그�

## 구현 노트
- 커밋 e8e0960: `web/dist/.gitkeep`(0바이트) 추적 + `.gitignore` 재포함(`dist/` 뒤에 `!web/dist/` → `web/dist/*` → `!web/dist/.gitkeep`) + `vite.config.ts` `keepDistPlaceholder()` 플러그인이 `generateBundle`에서 `.gitkeep`을 emitFile + README 로컬 개발 절 한 줄. 워크플로·Dockerfile·embed.go 무변경.
- 과제서와 다른 점: `.gitignore`를 `web/dist/*`로만 바꾸면 3행 generic `dist/`가 디렉터리를 통째로 무시해 예외가 안 먹었다(`git check-ignore -v`로 확인) → `!web/dist/` 재포함 줄을 추가. generic `dist/`는 release 아카이브·sdk dist가 쓰므로 유지.
- 검증 실제 출력: 수정 전 `go build ./...` → `web/embed.go:7:12: pattern all:dist: no matching files found` exit 1. 수정 후 `git clean -xfd web/dist && go build ./... && go vet ./...` exit 0, HEAD 로 만든 새 worktree 에서도 exit 0, `npm run build` 뒤 `git status --short web/dist` 빈 출력·`grep -c gitkeep web/dist/sw.js`=0·실제 산출물 위 `go build/vet` exit 0, `go test -count=1 ./web/ ./cmd/...` 통과, `prettier --check vite.config.ts` 통과.
- 인과 증명: 플러그인을 뺀 채 `npm run build` → `.gitkeep` 삭제됨(Vite emptyOutDir 가 dotfile 도 지움); 넣고 빌드 → 0바이트로 재생성. Rollup 은 dot 파일명 emitFile 을 거부하지 않았다.
- 확신 없는 곳: 없음에 가깝다. 굳이 꼽으면 `http.FileServer`가 `/.gitkeep`에 빈 200을 내는 것(과제서가 허용·차단 코드 금지) — 실제 요청으로 확인하지 않았고 정보 노출은 없다.
- 일부러 하지 않은 것: 전체 `go test ./...`·`-race`·브라우저 시험(러너 검증과 중복, 예산 홀드 전례); docs/admin-guide 수정과 manuals 재생성(README 한 줄로 충분, build-docs.mjs 는 README 를 읽지 않음); `.dockerignore` 변경(Dockerfile 이 web 스테이지 산출물을 복사하므로 효과 없음).
- 다음 역할 주의: 러너 검증 순서(go build → vet → test → npm build) 그대로 깨끗한 트리에서 통과해야 하며, `npm run build` 뒤 `git status`가 비어 있어야 정상. 커밋 전 `git clean -xfd web/dist`로 산출물을 지웠다(.gitkeep 은 추적돼 남는다). SAML return_to(83239b9)는 이 브랜치에 넣지 않았다.
- [러너 19:33] brief accepted — 채택 — 원인·파일·검증 명령이 모두 코드와 일치했고 verify.txt 의 한 줄이 로컬에서 그대로 재현됐다. 단 `.gitignore` 지시
- [러너 19:33] verify passed — 검증 5개 통과 (auto)

## 비평 노트
- 확인: diff 4파일 전부(gitignore·README·.gitkeep·vite.config.ts), embed.go `all:dist`, server.go 442~493 정적 서빙·503 폴백, Dockerfile·.dockerignore·ci/release.yml 빌드 순서, `git check-ignore -v` 실제 출력, 러너 `npm run build` 뒤 `git status --short` 빈 출력과 .gitkeep 19:33 재생성 — 인과 주장이 트리에서 그대로 확인됨. 판정 approve, risk low, blocking 없음.
- 못 본 것: `npm run build` 를 직접 재실행하진 않았고(러너 산출물로 대신), GET /.gitkeep 실제 응답도 요청하지 않았다(코드상 빈 200, 노출 없음).
- 남는 우려(릴리즈 노트): 웹 빌드 없이 만든 바이너리가 이제 조용히 컴파일되고 /app 은 503 '웹 빌드가 필요합니다' 를 낸다. 릴리즈·Docker 경로는 web 먼저 빌드라 무관, 로컬 개발자만 해당.
- 다음 회차: CI 에 web 빌드 전 `go build ./...` 한 줄이 없어 이 보호가 회귀해도 못 잡는다(워크플로 변경 제한 때문에 이번엔 지적만). 자동 테스트는 없으나 빌드 설정 변경이라 러너의 깨끗한 트리 빌드가 곧 검증이다.
- [러너 19:35] review approved — 리뷰 승인 (risk=low)
- [러너 19:35] pr created — https://github.com/hkjang/madi/pull/7
- [러너 19:39] ci failed — 성공이 아닌 검사: offline-image=failure, test=failure
