- 과제: Go 툴체인 핀 드리프트를 기존 CI 게이트(`scripts/check-version.sh`)에서 닫기 (가치 4 / 위험 2 / 작업량 S)
- 왜: `go.mod:8` 의 `toolchain go1.26.9` 와 `Dockerfile:9` 의 `FROM golang:1.26.9-alpine` 은 **같은 Go 패치여야 한다**는 계약인데, 이 계약을 지키는 검사가 저장소에 하나도 없습니다. 2026-10-16 회차가 바로 이 자리에서 반쪽을 잃었습니다(go.mod 만 올려 CI 게이트는 초록이 됐지만 릴리즈 이미지의 표준 라이브러리 취약점 12건이 남았고, 원인은 공식 golang 이미지가 `GOTOOLCHAIN=local` 이어서 `toolchain` 줄을 무시하는 것). 둘을 손으로 맞춘 직후라 지금은 일치하지만 구조적으로 재발합니다.
- 수용 기준:
  1) 지금 트리에서 `./scripts/check-version.sh` 가 **EXIT 0** 이고, 기존 출력(`version 0.76.8 is consistent across every file that carries it`)과 기존 검사 5건(web/package.json·web/package-lock.json·compose.yaml 2건·릴리즈 노트)이 그대로 남는다.
  2) **red/green 드릴로 증명**: `Dockerfile:9` 의 태그만 임시로 `golang:1.26.8-alpine` 으로 바꾸면 `./scripts/check-version.sh` 가 **EXIT 1** 로 떨어지고 `go.mod` 의 `toolchain` 값과 `Dockerfile` 의 빌더 태그를 둘 다 이름 붙여 말한다(어느 쪽이 무엇인지 사람이 읽고 고칠 수 있게). 되돌리면 다시 EXIT 0. 반대 방향(`go.mod` 의 `toolchain` 만 바꿈)도 같은 식으로 빨개진다. **두 드릴 모두 끝에 `git checkout -- Dockerfile go.mod` 로 원복하고 `git status --porcelain` 으로 확인.**
  3) 통제: 두 값이 일치하면 이 새 검사가 **아무 것도 출력하지 않고** 통과한다("전부 거절" 로 통과하는 검사가 아님을 보이는 자리).
  4) `git diff --name-only` 가 **정확히 `scripts/check-version.sh` 한 줄**. `.github/workflows/`·`Makefile`·`Dockerfile`·`go.mod`·`go.sum`·`VERSION`·`migrations/`·`internal/auth`·락파일·`web/` 은 **없다**.
- 건드릴 파일:
  - `scripts/check-version.sh` — 유일한 수정 파일. 기존 `check()` 헬퍼(라벨/값 비교 + `status=1`)와 릴리즈 노트 블록 사이, 또는 릴리즈 노트 블록 뒤에 **Go 핀 블록 하나**를 더합니다. `check()` 는 `$version`(VERSION 파일)과 비교하는 전용 헬퍼라 **재사용하지 말고** 별도 비교를 쓰세요(툴체인 패치는 VERSION 과 무관한 숫자입니다). 추출은 기존 스타일대로 `sed -n`:
    - `toolchain="$(sed -n 's/^toolchain go\([0-9][0-9.]*\)$/\1/p' "$root/go.mod" | head -1)"`
    - `builder="$(sed -n 's/^FROM golang:\([0-9][0-9.]*\)-alpine .*/\1/p' "$root/Dockerfile" | head -1)"`
    - **빈 값도 실패로 처리하세요** — 둘 중 하나가 비면 패턴이 어긋난 것이고, 빈 값 == 빈 값으로 조용히 통과하면 검사가 죽은 채 초록이 됩니다(이 저장소에서 반복된 사고 유형).
    - 실패 메시지는 기존 `printf ... >&2` + `status=1` 관례를 따르고, **왜** 같아야 하는지 한 줄 주석을 답니다(공식 golang 이미지의 `GOTOOLCHAIN=local` 이 `toolchain` 줄을 무시하므로 go.mod 만 올리면 릴리즈 이미지가 옛 표준 라이브러리를 담는다 — 2026-10-16 에 실제로 일어난 일).
  - 마지막 성공 메시지는 지금 VERSION 만 말합니다. 바꾸지 말거나, 바꾸더라도 **문구에 `version <X> is consistent` 조각을 남기세요** — 과거 회차 로그가 이 문자열을 통과 신호로 읽습니다.
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `./scripts/check-version.sh` (CI 의 `ci.yml:49` 가 그대로 돌리는 명령 — 이것이 배선 증거입니다)
  - `bash -n scripts/check-version.sh` (구문)
  - `make test-go` → `go vet ./...` + `go test ./...` (Go 소스 0줄 변경이지만 `go.mod` 를 건드리지 않았음을 확인하는 값싼 게이트. DSN 없으면 통합 SKIP 이고 그것을 통과로 포장하지 말 것)
  - `git diff --name-only` / `git status --porcelain` (수용 기준 4)
  - **범위 밖**: `make test-web`(웹 0줄), `docker build`, DB 통합, E2E. 쉘 한 파일 변경이므로 돌리지 않아도 됩니다 — 돌리지 않았다고 적으세요.
- 위험과 피할 것:
  - **보호 경로를 읽기만 하세요.** `Dockerfile`·`go.mod` 는 이 과제에서 **수정 대상이 아닙니다**(드릴 때 임시로 바꾸고 반드시 원복). `.github/workflows/`·`Makefile` 배선은 손대지 마세요 — 이미 `ci.yml:49` 가 이 스크립트를 돌리므로 **추가 배선이 필요 없습니다**.
  - **느슨하게 만들지 마세요**: 추출 패턴이 안 맞을 때 "경고만 하고 통과" 로 처리하면 검사가 무의미합니다(수용 기준 3·빈 값 처리 참조).
  - **일치하는 숫자를 상수로 하드코딩하지 마세요**(`1.26.9` 를 스크립트에 적지 말 것). 두 파일에서 읽어 **서로** 비교해야 다음 업그레이드 때 자동으로 따라갑니다.
  - 같은 계열의 **Node 하한 드리프트**(`engines`·`ci.yml:41` 의 `node-version: 24`·`Dockerfile:2` 의 `node:24-alpine`)는 이번 회차에 **담지 마세요**. 지금 실제 불일치가 없고, 한 회차에 두 계약을 넣으면 파일/판정이 늘어납니다. 다음 회차에 같은 스크립트에 얹으면 됩니다.
  - 마크다운 파서 통합(운영자 규칙 3번)·다운로드 파일명 통합(PR #167 되돌림)과는 무관한 과제입니다. 건드리지 마세요.
  - 이 체크아웃에 `web/node_modules` 가 **없습니다**(확인함). npm 계열 명령의 exit 127 / `not found` 는 저장소 결함이 아닙니다. 웹을 돌릴 일은 없지만 돌린다면 `npm ci --prefix web` 선행.
  - 커밋 트레일러(`Co-Authored-By` 등) 금지, `VERSION` 미변경, 커밋 메시지는 영어 동작 설명.
- **확인한 것 (정찰이 실제로 본 것)**:
  - `go.mod:8` = `toolchain go1.26.9`, `go.mod:3` = `go 1.26.0`, `go.mod:5-7` 주석은 "any Go 1.26 patch release can build with GOTOOLCHAIN=local" — **읽었습니다**.
  - `Dockerfile` 의 이미지 줄은 **정확히 두 개**: `:2 FROM node:24-alpine AS web-build`, `:9 FROM golang:1.26.9-alpine AS go-build`. `golang:` 은 **한 번만** 나옵니다 → 빌더 줄 특정 문제 없음.
  - `ci.yml:49` 가 `run: ./scripts/check-version.sh` — **이미 배선되어 있습니다**(추가 배선 불필요). 같은 job 의 `:37` 은 `go-version-file: go.mod`, `:41` 은 `node-version: 24`.
  - `scripts/` 에 **쉘 테스트 파일이 없습니다**(`scripts/**/*test*` 글롭 → 0건). 따라서 증거는 수용 기준 2·3 의 red/green 드릴입니다 — 소스 문자열 grep 을 증거로 삼지 마세요(운영자 규칙 5번).
  - `docs/releases/v0.76.8.md` **존재함** → check-version.sh 의 릴리즈 노트 블록은 통과합니다. `web/package.json:4` 의 `"version": "0.76.8"` 도 VERSION 과 일치.
  - 이 체크아웃에 `web/node_modules` **없음**.
- **미확인 (구현자가 실행으로 메울 것)**:
  - `./scripts/check-version.sh` 를 **직접 돌리지 못했습니다**(이 세션의 샌드박스가 스크립트 실행을 막았습니다). 위 두 블록(릴리즈 노트·web/package.json)은 글롭·읽기로 확인했지만 `web/package-lock.json`·`compose.yaml` 두 블록은 미확인입니다. 2026-10-18 직전 회차 로그가 같은 VERSION 에서 `version 0.76.8 is consistent across every file that carries it` 을 받았으므로 초록일 것으로 보지만, **가장 먼저 베이스에서 한 번 돌려 기준선을 찍으세요**. 베이스가 이미 빨갛다면 그 실패는 이번 과제와 무관한 선행 결함이므로 기록하고 차선 후보로 가세요.
  - `sed` 패턴이 실제 두 줄을 뽑는지 미확인 — 제안한 패턴은 위에서 읽은 줄 모양(`toolchain go1.26.9`, `FROM golang:1.26.9-alpine AS go-build`)에 맞춘 것이지만, 추출 결과를 **먼저 echo 로 찍어** 빈 값이 아닌지 확인한 뒤 비교 로직을 붙이세요.
- 차선 후보: **`format` 에도 설치 가드 한 줄 (`preformat`)** (가치 2 / 위험 1 / S) — `web/package.json:20` 의 `"format": "prettier --write src"` 는 `scripts` 중 **유일하게 `pre*` 가 없는 자리**입니다(확인함: predev·prebuild·pretypecheck·prelint·pree2e·prepreview 는 있고 format 만 없음). `"preformat": "node scripts/require-installed.mjs prettier"` 한 줄을 더하고, `web/scripts/require-installed.test.mjs` 의 기존 fixture 방식(실제 `web/package.json` 복사 + 실제 npm 자식 프로세스)으로 재현하세요. 소스를 **쓰는** 명령이므로 실제 `src/` 에 prettier 를 돌리는 드릴은 하지 말고 fixture 안에서만 확인할 것.
