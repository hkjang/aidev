# 회차 노트 2026-10-18-073859-umm-improve — umm
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:39] base pinned — main@745dcf3
- [러너 07:39] autonomy low-risk — 회귀(reverted) 2026-10-06T22:38:39+09:00

## 정찰 노트
- 고른 이유: 베이스 main@745dcf3 에 13b7e00(review-pending)이 안 들어와 `go.mod:8/:39` 가 취약한 채다 — 이 베이스에서 나가는 **어떤 PR 이든 CI 의 govulncheck(ci.yml:66)이 빨갛다**. 그래서 작은 DX 후보(preformat 등)를 고르면 과제와 무관하게 회차가 verify-failed 한다. 더구나 `ci.yml:37`(go-version-file: go.mod)과 `release.yml:26`(docker build → Dockerfile:9)이 Go 버전을 **다른 데서** 읽는 것을 파일로 확인했으므로, go.mod 만 고치면 릴리즈 이미지는 그대로 취약하다. 두 반쪽을 한 과제(프로덕션 3파일)로 묶었다.
- 확신 없는 곳(과제서에 "미확인" 으로 표시): ① **이 세션은 Bash 실행 승인이 읽기 전용에만 나서 govulncheck 을 직접 못 돌렸다** — 권고 ID 12건은 베이스 SHA 가 같은 2026-10-16 회차의 기록이고 내 실행 결과가 아니다. ② `golang:1.26.9-alpine` 태그와 `x/net v0.60.0` 의 실재. ③ setup-go@v6 가 go.mod 의 `toolchain` 줄을 읽는지.
- 구현자가 조심할 것: `docker build` EXIT=0 을 증거로 삼지 말 것 — 이 자리가 바로 운영자 규칙 4번이 실제로 걸린 곳이다(이미지에서 바이너리를 꺼내 `go version` 까지). `Dockerfile` 은 9행 한 줄만. `go mod tidy` 금지. x/crypto·x/sys·x/text·x/sync 가 따라 올라가는 것은 MVS 강제이며 기회주의적 업그레이드가 아니다. `13b7e00` 을 cherry-pick 하지 말고 독립적으로 고칠 것.
- 피한 유형: `make test-go` 를 CI 에 맞추는 논거는 2026-10-06 에 verify-failed 했고 운영자가 재제출을 금지했으므로 ideas.json 에서 `rejected` 로 내렸다. 파일 이름 새니타이즈도 PR #167 되돌림 자리라 `rejected`.
- [러너 07:44] scout done — Go 버전 핀 드리프트를 CI 게이트와 릴리즈 이미지 **양쪽**에서 닫기 — `go.mod` 의 `x/net`·`toolchain` + `Dockerfil

## 구현 노트
- 무엇을 왜: govulncheck 이 베이스에서 실제로 빨갰다(권고 12건, exit status 3). `x/net v0.57.0→v0.60.0` + `toolchain go1.26.7→go1.26.9`(CI 게이트 반쪽)와 `Dockerfile:9 golang:1.26.7→1.26.9-alpine`(릴리즈 이미지 반쪽)을 함께 닫았다 — 공식 golang 이미지의 `GOTOOLCHAIN=local`(`docker run … go env GOTOOLCHAIN` 으로 확인)이 `toolchain` 줄을 무시하므로 go.mod 만으로는 이미지가 안 고쳐진다. 프로덕션 3파일, 새 시험 0개. 커밋 26dd483.
- 확신 없는 곳·검증 못 한 것: ① **`actions/setup-go@v6` 가 `go-version-file: go.mod` 에서 `toolchain` 줄을 읽는지 끝까지 확인하지 못했다**(`gh` 미인증, 원격 CI 로그 접근 불가). 로컬은 `GOTOOLCHAIN` 기본값(auto)으로 `go version` → `go1.26.9` 였고 그 조건에서 게이트가 초록이다. setup-go 가 `go 1.26.0` 을 깔더라도 auto 가 toolchain 을 내려받아 같은 결과가 되리라 보지만 **실행으로 확인한 것은 로컬 경로뿐**이다. ② `-show verbose` 를 돌리지 않아, 고친 뒤에도 남는 "호출되지 않는" 권고 2건(import 1 + require 1)이 **어느 것인지 모른다**. ③ `make test-web`·웹 게이트·E2E 는 돌리지 않았다(웹·Go 소스 0줄, 이 체크아웃에 `web/node_modules` 없음) — 미설치 상태의 exit 127 은 저장소 결함이 아니다.
- 일부러 하지 않은 것: `go mod tidy`(diff 번짐 — `go.sum` 은 10줄 추가만), `13b7e00` 머지·cherry-pick(독립적으로 고쳤다 — 머지 시 go.mod/go.sum 충돌은 중재에 맡긴다), `.github/workflows/` 수정·스캔 끄기/핀하기(느슨하게 만드는 방향), `VERSION` 올리기, `Dockerfile` 의 9행 외 모든 줄(`node:24-alpine`·`alpine:3.24`·`-trimpath`/`-ldflags`·`CGO_ENABLED=0` 그대로), `go.mod:5-7` 주석(리터럴 `1.26.7` 이 없고 그 주장이 **여전히 참**임을 로컬 go1.26.7 + `GOTOOLCHAIN=local go build` EXIT=0 으로 확인했다 — 하한을 정하는 것은 그대로 둔 `go 1.26.0` 디렉티브다).
- 다음 역할이 조심할 것: `go test -p 1 ./...` 는 **DB 가 있어야 의미가 있다** — DSN 없이 돌리면 통합이 `--- SKIP` 이고 그 PASS 는 DB 를 돌린 것이 아니다(대조군을 `impl-dsn-control.log` 에 붙였다). `docker build` EXIT=0 을 증거로 쓰지 말 것 — 이미지에서 `/app/umm` 을 꺼내 `go version` 까지 가야 한다(`impl-image-goversion.log`: `go1.26.9`, `x/net v0.60.0`). 첫 빌드가 CACHED 면 `--no-cache-filter go-build` 로 다시 띄울 것. govulncheck 은 네트워크가 필요하고 **커밋이 아니라 취약점 DB 가 판정을 바꾼다** — 이 초록은 2026-10-18 시점의 DB 기준이다. x/crypto·x/sys·x/text·x/sync 동반 상승은 상류 go.mod 가 강제한 MVS 결과이며 되돌릴 수 없다.
- [러너 07:53] brief accepted — 채택 — 과제서의 근거가 지금 코드와 전부 맞았고, 과제서가 "**미확인**" 으로 표시한 세 가지를 지시대로 모두 1단계에�
- [러너 07:53] verify failed — 실패한 검증: npm test --silent (exit 1)
