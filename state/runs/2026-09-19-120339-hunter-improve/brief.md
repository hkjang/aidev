# 과제서 — 2026-09-19 (수정 과제)

- 과제: v1.12.0 릴리즈 워크플로(`Release offline image`) 실패 원인 확인·수정 — 태그 v1.12.0 의 첨부 자산이 비어 있는 상태 복구 (가치 5 / 위험 2 / 작업량 S~M)
- 왜: 2026-09-18-090342 회차의 `release.json` 은 `status: released, github_release: false, assets: []` 로 끝났고, 2026-09-18-164351 회차는 예산 hold(`회차 예산($22)이 오늘 남은 상한을 넘음`)로 시작조차 못 했다. 즉 태그 `v1.12.0` 은 밀려 있으나 GitHub 릴리즈·`hunter-v1.12.0.tar.gz` 첨부가 없다(러너 기록 기준). 정찰 세션은 `gh`·네트워크가 차단되어 워크플로 실패 로그를 직접 보지 못했으므로(미확인) 구현자가 첫 단계로 실패 단계를 확인해야 한다.
- 수용 기준:
  1) `gh run list --workflow release.yml --limit 5` 로 v1.12.0 태그의 실패 run 과 실패 단계(Validate tag / Build and package / Create release notes / Publish image archive)를 확인해 `journal.md` 에 원인을 인용한다.
  2) 원인이 저장소 코드·스크립트에 있으면 그 스크립트를 고치고(워크플로 조건을 느슨하게 하지 않음), 로컬에서 같은 단계를 재현해 통과시킨다: `test "v1.12.0" = "v$(cat VERSION)"`, `bash -n scripts/release.sh`, `python3 -m py_compile scripts/release-notes.py`, `bash scripts/release.sh "$(cat VERSION)"`(Docker 필요, 수 분), `python3 scripts/release-notes.py v1.12.0 dist/hunter-v1.12.0.tar.gz /tmp/notes.md`.
  3) 원인이 인프라(러너 디스크·GitHub 일시 오류·예산 hold 로 인한 러너 미기록)라면 코드를 바꾸지 말고 `gh run rerun <id>` 로 재실행한 뒤 `gh release view v1.12.0 --json tagName,assets,url` 에서 `hunter-v1.12.0.tar.gz` 하나만 첨부됐음을 확인해 원장에 기록한다. 릴리즈 완료는 첨부 자산 확인으로만 보고한다.
- 건드릴 파일(후보 — 실패 단계에 따라 하나만):
  - `.github/workflows/release.yml` — 단계 4개. `Publish image archive` 는 `gh release view` 실패 시 `gh release create --verify-tag`. 여기서 실패했다면 `permissions: contents: write` 와 태그 존재 여부를 본다. 조건을 지우거나 `continue-on-error` 를 넣지 말 것.
  - `scripts/release.sh` — `VERSION` 과 인자 일치 검사, `docker build --platform linux/amd64 --build-arg VERSION=`, `docker image save | gzip -n -9`, `gzip --test`. 러너 디스크 부족(`no space left on device`)이면 `docker build` 전 불필요 캐시 정리는 워크플로가 아니라 스크립트가 아닌 별도 step(`docker system prune -af` 는 금지 — 검증 완화 아님이지만 이미지 캐시만)으로 추가 검토.
  - `scripts/release-notes.py` — 아카이브 경로에서 sha256 을 계산해 본문에 넣는 것으로 추정(미확인: 정찰 세션에서 실행 불가). 아카이브가 없을 때 실패하는지 확인.
  - `Dockerfile` — 고정 digest 4개(node:26, golang:1.26, debian:bookworm-slim). digest 가 레지스트리에서 사라졌으면(`manifest unknown`) **같은 태그의 현재 digest** 로만 갱신하고 `go.mod`·`web/package-lock.json` 과 버전이 어긋나지 않는지 확인.
- 검증 명령:
  - `bash -n scripts/release.sh && python3 -m py_compile scripts/release-notes.py`
  - `node scripts/verify-pentagi.mjs`
  - `bash scripts/release.sh "$(cat VERSION)"` → `dist/hunter-v1.12.0.tar.gz` 생성·`sha256sum` 출력
  - `python3 scripts/release-notes.py v1.12.0 dist/hunter-v1.12.0.tar.gz /tmp/notes.md && cat /tmp/notes.md`
  - 코드 변경이 있으면 `go vet ./... && go build ./cmd/hunter`
  - 재실행 후: `gh run watch <id>`; `gh release view v1.12.0 --json tagName,assets,url`
- 위험과 피할 것:
  - 워크플로를 통과시키기 위해 `Validate tag`·`--verify-tag`·`gzip --test` 를 제거하거나 `continue-on-error` 를 넣는 것은 금지(운영자 규칙).
  - 릴리즈 자산은 `hunter-v1.12.0.tar.gz` 하나만. 체크섬 파일·PDF 를 추가 첨부하지 않는다(AGENTS.md §10).
  - 새 버전을 올리지 않는다. VERSION 은 1.12.0 그대로이고 태그도 이미 존재한다. `release.yml` 을 바꿔야 한다면 그 변경은 main 에 머지되어도 이미 밀린 태그에는 적용되지 않으므로, 태그를 옮기지 말고 `gh run rerun` 또는 `workflow_dispatch` 없이 처리 가능한지 먼저 판단(태그 재생성은 운영자 승인 대상).
  - Go 테스트 전체(`go test -race ./internal/app`)는 650초 이상 걸리고 `HUNTER_TEST_DSN` 이 없으면 DB 테스트가 skip 된다. 이 과제에는 필요 없으니 돌리지 말 것(예산 hold 재발 방지).
  - `third_party/pentagi` 원본 파일을 건드리지 않는다.
- 차선 후보: 표를 발급하는 보고서 본문 크기·사용자별 미사용 표 개수 상한(사용자당 20개) — `internal/app/handoff.go` 발급 트랜잭션에 COUNT 검사 한 줄과 통합 테스트 (가치 2 / 위험 1 / 작업량 S). 1순위가 인프라 원인으로 코드 변경 없이 끝나면 이것을 이어서 한다.
