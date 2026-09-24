# PR 처리기 노트 2026-09-24-041142-madi-shepherd — madi PR #12
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-24-033411-madi-improve)
# 회차 노트 2026-09-24-033411-madi-improve — madi
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:34] base pinned — main@fd2c3b5
- [러너 03:34] autonomy release — 

## 정찰 노트
- 우선 과제가 지목한 `go build ./...` 실패를 직접 재현 확인: `web/dist` 부재 + `.gitignore:2` `web/dist/` + `web/embed.go:7` `all:dist`. 새 아이디어 대신 이 실패 복구를 과제로 적었다(보류 항목 중 CLAUDE.md·이미지 회귀 등은 이번 실패를 되돌리지 못해 제쳤다).
- 추측 아님: release.yml:46-51 이 go 단계보다 먼저 `npm run build` 를 돌린다. 따라서 "릴리즈가 같은 이유로 두 번 실패" 를 임베드 탓으로 옮기지 않았고, 워크플로가 실제로 깨지는 곳은 53행 govulncheck 와 110행 이미지 빌드로 분리해 적었다.
- 미확인: 이 환경에서 `go build`·govulncheck·docker 를 실행할 권한이 없어 명령 출력은 얻지 못했다(파일 상태·버전 핀은 모두 직접 읽어 확인). 수정 전/후 실제 출력 기록은 구현자가 남겨야 한다.
- 조심할 것: `ca-certificates` 와 `ca-certificates-bundle` 은 반드시 함께 올릴 것, 고지 파일은 `scripts/licenses.mjs` 재생성으로만, 워크플로·Dockerfile·internal/server 는 무변경. 검증 완화로 통과시키는 것은 금지.
- 반복 유형 회피: 09-21 의 "스킬 부재로 미선택" 은 기각됐으므로 세 스킬을 먼저 로드하고 과제서를 반드시 남겼다. Docker/govulncheck 불가 시를 대비해 차선 후보를 임베드 단독(S)으로 좁혀 두었다.
- [러너 03:37] scout done — 수정 과제 — 깨끗한 체크아웃에서 `go build ./...` 를 깨뜨리는 `all:dist` 임베드 자리표시자 부재를 고치고, �

## 구현 노트
- 무엇/왜: 러너 검증을 깨뜨린 `all:dist` 자리표시자 부재(bb4e155)와, 같은 회차에 반복 실패하던 govulncheck GO-2026-6348·apk lock 드리프트(f942218)를 cherry-pick 으로 통합. 두 커밋 모두 `git merge-base --is-ancestor` 로 fd2c3b5 의 비조상임을 먼저 확인했고, 세 실패를 각각 실제 출력으로 재현한 뒤 적용했다. 변경은 과제서가 지정한 9개 파일뿐이다.
- 확신 없는 곳/미검증: 전체 이미지 빌드(`docker build -t madi:ci .`)와 `scripts/verify-image.sh` 를 돌리지 않았다 — `--target runtime-base` 만 exit 0 으로 확인했으므로 소스 재고(69 APK/52 origins) 일치는 이번 회차에 직접 증명하지 못했다(직전 회차가 같은 lock 값으로 확인했을 뿐). `go test -race ./...` 전체, PostgreSQL 이 필요한 internal/server 시험, 브라우저 시험도 미실행. 원격 CI/릴리즈 성공은 확인할 수 없다.
- 일부러 하지 않은 것: 1de4d00(직전 회차의 runtime_lock_test.go)은 과제서 지시대로 끌어오지 않았다. `.github/workflows`, `Dockerfile`, `scripts/**`, `internal/**` 는 무변경(`git diff --stat fd2c3b5 HEAD -- .github Dockerfile scripts internal` 빈 출력). govulncheck 억제·lock 핀 완화 같은 검증 약화는 하지 않았다.
- 다음 역할이 조심할 것: `node scripts/licenses.mjs` 는 `npm ci --prefix web` 이 선행되지 않으면 원시 ENOENT 스택으로 죽는다(`--check` 는 node_modules 없이도 동작). `web/dist/.gitkeep` 은 추적 파일이며 Vite `keepDistPlaceholder()` 플러그인이 매 빌드마다 재생성한다 — 플러그인을 빼면 빌드 직후 `D web/dist/.gitkeep` 이 되는 것을 실제로 확인했다. `go test ./tests/deployment-contract` 는 DB 불필요하지만 이미지 검증은 Docker 필요.
- [러너 03:42] brief accepted — 채택 — bb4e155·f942218 이 HEAD 의 비조상임과 세 실패 재현 출력이 과제서와 그대로 일치했고, 지정된 9개 파일만 바꿨으며 
- [러너 03:42] verify passed — 검증 5개 통과 (auto)

## 비평 노트
- 확인함(직접 실행): go build/vet ./... , govulncheck v1.7.0 "No vulnerabilities found" exit 0, `go mod tidy` no-op, licenses --check {ok:true,423}, deployment-contract, `npm run build --prefix web` 후 web/dist/.gitkeep 잔존 + git status 깨끗, git check-ignore 로 .gitignore 3규칙 검증. x/sys·x/text 직접 import 도 파일에서 확인(sandbox_linux.go:11, search_operations.go:16) — 커밋 메시지와 코드가 일치한다.
- 못 본 것: 전체 `docker build -t madi:ci .` 와 verify-image.sh(69 APK/52 origins 단언), PostgreSQL/브라우저 시험, 원격 CI.
- 승인이어도 남는 우려 ①: d5cbdbc 의 "Verified after" 는 8451b5b 에서 옮겨 온 증거다. vite.config.ts 는 .dockerignore 에 안 걸려 web 스테이지에 실제로 들어가므로 이미지 증거는 이번 트리를 전부 덮지 않는다. 릴리즈 전 이미지 1회 실빌드 권장.
- 우려 ②: web/dist/.gitkeep 추적을 지키는 테스트가 없다(방어는 README 문구뿐). 지워지면 깨끗한 체크아웃 go build 가 다시 조용히 깨진다 — 다음 회차의 값싼 후보.
- 우려 ③: runtime-apk.lock 정확 핀은 Alpine 저장소 상태 의존이라 머지 전 재드리프트 시 같은 실패가 재발한다. 구조는 그대로 두는 것이 맞다(완화 금지). 보안·법무 차단 사유 없음.
- [러너 03:46] review approved — 리뷰 승인 (risk=low)
- [러너 03:47] pr created — https://github.com/hkjang/madi/pull/12
- [러너 04:05] ci timeout — 제한 시간 안에 CI 완료를 확인하지 못함

## 심사 노트
- 확인함(이 트리에서 직접 실행): docker build . exit 0 + scripts/verify-image.sh 8 PASS(69 APK/52 origins/eng+kor) — 비평 우려 ①을 해소했다. npm ci+build 후 web/dist/.gitkeep 잔존·git status 빈 출력, go build/vet, govulncheck "No vulnerabilities found", go mod tidy no-op, licenses --check {ok:true,423}, deployment-contract, git check-ignore 3규칙.
- 확인함(배선): x/sys·x/text 직접 승격이 sandbox_linux.go:11·search_operations.go:16 실제 import 과 일치. runtime-apk.lock 의 세 핀을 읽는 경로(runtime-base 설치 / runtime-sources 소스 재고)를 verify-image 로 함께 확인했고, 저 버전을 박아 둔 다른 파일은 없다(grep).
- 못 본 것: PostgreSQL 필요한 internal/server 전체 시험, 브라우저 시험, go test -race ./... 전체, 원격 CI 결과.
- 권고 근거: 보호 파일 무변경, 신규 공개 경로·권한 확대·비밀 노출·마이그레이션 없음, 순수 revert 가능. 모든 변경이 실제 출력을 바꾸며 대역 없이 프로덕션 배선으로 증명됨 → approve / merge (risk=low).
- 다음 회차 후보(차단 아님): web/dist/.gitkeep 추적을 지키는 회귀 테스트가 없다(방어는 README 문구뿐).
