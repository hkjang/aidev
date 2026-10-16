# 회차 노트 2026-10-16-220900-umm-improve — umm
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 22:09] base pinned — main@745dcf3
- [러너 22:09] autonomy low-risk — 회귀(reverted) 2026-10-06T22:38:39+09:00
- [러너 22:14] scout done — 수정 과제 — PR #172 의 CI 를 두 번 떨어뜨린 게이트를 특정해 그 원인을 고치기 (외부 권고 게이트가 1순위

## 구현 노트
- 바꾼 것: `go.mod`·`go.sum` 두 파일. `ci.yml:68` govulncheck 이 베이스(main@745dcf3)에서 `exit status 3` 이었고 원인이 **두 갈래**였다 — x/net@v0.57.0 의 HTTP/2 묶음(GO-2026-6617 등, 호출 경로 있음)과 `go.mod:8` `toolchain go1.26.7` 이 고르는 **표준 라이브러리 12건**. 각각 최소 패치로 x/net v0.60.0 · toolchain go1.26.9. x/crypto·x/sys·x/text·x/sync 가 함께 오른 것은 MVS 강제(상류 go.mod 로 확인), `go mod tidy` 미사용.
- **확신 없는 곳·검증 못 한 것**: ① **PR #172 의 CI 로그를 못 읽었다**(`gh` 미인증). 실패 게이트는 main 에서의 재현으로 특정했고 PR #172 가 이 고침만으로 초록이 된다고 단정하지 않는다. ② **릴리즈 이미지는 아직 표준 라이브러리 12건을 담는다** — `docker build` 는 EXIT=0 이지만 이미지에서 꺼낸 바이너리가 `go version` → `go1.26.7` 이었다. 공식 golang 이미지가 `GOTOOLCHAIN=local` 이라 `toolchain` 줄을 무시한다(`docker run --rm golang:1.26.7-alpine go env GOTOOLCHAIN` → `local`). 즉 **지금 go.mod 의 toolchain 과 `Dockerfile:9` 의 핀이 드리프트 상태**다. x/net 쪽은 바이너리에 실제로 들어갔다(`go version -m` 으로 확인).
- 일부러 하지 않은 것: `Dockerfile:9` 를 `golang:1.26.9-alpine` 로 올리는 것 — 보호 경로라 읽기만 했다. 이것이 다음 회차 최우선(ideas.json 2번). `Makefile`·`.github/workflows/`·`VERSION`·`migrations/`·`internal/auth` 도 무수정, 느슨하게 만든 것 0건, 앱 코드·새 시험 0줄.
- 다음 역할이 조심할 것: 검증 로그는 `impl-vuln-base.log`(실패) / `impl-vuln-reverted.log`(되돌림 재실패) / `impl-vuln-final.log`(커밋 트리 EXIT=0) / `impl-go-gates.log` / `impl-web-gates.log` / `impl-docker-build.log`. govulncheck 은 네트워크가 필요하고 **커밋이 아니라 취약점 DB 가 판정을 바꾼다** — 며칠 뒤 다시 빨개질 수 있다. Go 통합 시험은 DB 가 있어야 돈다(격리 PostgreSQL 17 포트 15438, 검증 뒤 삭제 — 남아 있는 `umm-test-pg` 컨테이너는 **제 것이 아니다**). `make test-web` 전에 `npm ci --prefix web` 선행 필수(새 체크아웃에 `node_modules` 없었다). 남는 노이즈: 호출되지 않는 권고 1패키지/1모듈, `TestGatewayEmbeddingQuality` SKIP(외부 URL 필요), 기존 `src/` oxlint 경고.
- [러너 22:23] brief accepted — 채택 — 1순위 혐의(외부 권고 게이트)가 **맞았고** 1단계에서 멈췄습니다. 과제서가 **미확인**으로 남긴 것을 지시대로 �
- [러너 22:24] verify passed — 검증 17개 통과 (auto)

## 비평 노트
- 판정 approve / risk medium / blocking 없음. 직접 돌려 확인: `go build ./...`·`go vet ./...`·`go test -count=1 ./internal/auth` EXIT=0, `govulncheck ./...` EXIT=0(No vulnerabilities found). MVS 강제 주장은 상류 go.mod 로 검증됨 — x/net v0.60.0 → crypto v0.57.0·sys v0.48.0·text v0.42.0, text v0.42.0 → sync v0.23.0. 기회주의적 상향 0건, 커밋 메시지와 일치.
- 구현자가 "확신 없다" 고 적은 두 곳을 먼저 봤다. ① PR #172 CI 로그 미확인 — 저도 gh 권한이 없어 못 봤으나, main 에서 게이트가 실제로 빨갛고 이 커밋으로 초록이 되는 것은 재현됨(impl-vuln-base/reverted/final.log + 제 재실행). ② 릴리즈 이미지 드리프트 — release.yml:26 → Dockerfile:9 `golang:1.26.7-alpine` 로 확인, 사실임. 다만 main 에서도 이미 참이라 이 PR 이 만든 노출이 아니고 x/net 수정은 바이너리에 들어가므로 차단하지 않았다.
- 승인이어도 남는 우려 둘: (a) **go.mod:4-7 주석이 이제 거짓** — "Pinned for CI and release builds" 인데 toolchain 줄은 릴리즈를 고정하지 못한다(golang 이미지가 GOTOOLCHAIN=local). Dockerfile:9 를 올릴 때 이 주석도 같이 고칠 것. (b) 머지 뒤 CI 는 표준 라이브러리 12건을 더 이상 알려주지 않는다 — **탐지 신호가 줄어든다**는 사실을 릴리즈 노트에 남겨야 다음 회차가 잊지 않는다.
- 못 본 것: 웹 게이트·E2E·docker build 를 재실행하지 않고 impl-web-gates.log / impl-docker-build.log(EXIT=0) 를 신뢰했다. Go 통합 시험도 DSN 을 세우지 않고 impl-go-gates.log 로 갈음했다(앱 코드 0줄 변경이므로 비용 대비 타당하다고 판단).
- 노이즈 2건: go.sum 에 tidy 가 지울 구버전 10줄이 남았다(과제서가 tidy 를 금지해 의도된 것, main 은 tidy 상태였음). 미호출 권고 GO-2026-6443(grpc v1.83.1→v1.83.2)·GO-2026-5932(x/crypto, 수정판 없음) — 전자가 DB 재평가 시 게이트를 다시 빨갛게 만들 가장 가까운 후보.
- [러너 22:29] review approved — 리뷰 승인 (risk=medium)
- [러너 22:29] pr created — https://github.com/hkjang/umm/pull/173
- [러너 22:29] autonomy held — low-risk 단계 — 위험도 medium, 파일 2: 사람 승인 필요
