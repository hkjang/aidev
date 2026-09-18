# 과제서 — 2026-09-19 (appstore)

## 우선 과제 진단 결과: 저장소에는 고칠 것이 없다 (러너 예산 보류였음)

- 과제: [수정 과제] '마지막 회차 error' 의 원인 확정 — 릴리즈 워크플로 실패가 아니라 러너의 예산 보류 (가치 3 / 위험 1 / 작업량 S)
- 왜: 러너가 "릴리즈 워크플로가 같은 이유로 두 번 실패" 라고 자동 적재했지만, 상태 파일을 읽어 보면 **마지막 회차(2026-09-18-163351)는 코드에 손대기 전에 `improve` 단계에서 `hold` 로 끝났다** — `stages.json`: `{"improve":{"state":"hold","reason":"회차 예산($22)이 오늘 남은 상한을 넘음"}}`, `run.json` 의 started 와 finished 가 같은 초(16:33:52). 워크플로·테스트·스크립트가 돌지도 않았다. 직전 실제 릴리즈 회차(2026-09-17-031312)는 `release: published v2.11.1`, `assets: verified (자산 1개)` 로 끝났고 main 은 `b3323ff chore(release): AppStore v2.11.1` 이다. 즉 "실패한 단계의 스크립트·테스트" 가 존재하지 않으므로 워크플로를 고치면 안 되고(효과 없는 변경은 반려 사유), 원장에 진단을 기록하는 것이 이번 수정 과제다.
- 확인한 것(2026-09-19 로컬 worktree, main@b3323ff 기준):
  - `.github/workflows/release.yml` 의 "Install and test" 단계와 같은 Go 패키지 목록 `go test . ./cmd/... ./internal/... ./migrations/... ./openapi/...` → 전부 `ok`(DSN 없이; 통합 테스트는 건너뜀).
  - `.github/workflows/ci.yml`·`release.yml`·`Makefile` 을 읽었고 v2.11.1 이후 워크플로 파일 변경 없음.
  - PR #23 (mcp-oauth 캠페인, 2026-09-17-180332 회차, `internal/auth/*` 변경으로 guard `held-expected`) 은 2026-09-19 06:20 shepherd 리뷰에서 `approve / merge` 판정을 받았으나 **아직 main 에 머지되지 않음**(git log 에 d74df77 없음). 이것은 사람이 머지할 보호 경로(auth) PR 이라 이 회차가 건드릴 일이 아니다.
  - 미확인: `gh run list` 는 이 세션에서 권한이 거부되어 GitHub Actions 실행 이력을 직접 보지 못했다. `check-env-contract.sh`·`check-docs.sh`·`npm test`·`npm run build` 도 이 세션에서는 실행 승인이 안 나 돌리지 못했다(구현자가 돌릴 것).
- 수용 기준:
  1) 코드·워크플로 변경 없음(`git status` 깨끗). 릴리즈 워크플로를 느슨하게 만들지 않는다.
  2) 릴리즈 워크플로 "Install and test" 단계를 로컬에서 그대로 재현해 통과: `go test . ./cmd/... ./internal/... ./migrations/... ./openapi/...`, `npm --prefix web ci --no-audit --no-fund`, `npm --prefix web test`, `npm --prefix web run build`, `./scripts/check-offline-assets.sh web/dist`, `./scripts/check-env-contract.sh`, `./scripts/check-docs.sh` — 모두 exit 0 을 원장에 적는다.
  3) 원장 항목을 '수정 과제' 로 기록: "2026-09-18 error 는 러너 예산 보류(improve 단계 hold, $22 > 일일 잔여 상한)였고 v2.11.1 릴리즈는 published+assets verified 로 정상. 저장소 결함 없음, 변경 없음."
  4) 시간이 남으면 아래 차선 후보를 같은 회차에 구현한다(별도 커밋). 차선을 구현했을 때만 코드 변경이 생긴다.
- 건드릴 파일: 없음(원장만). 차선 후보를 하면 아래 참조.
- 검증 명령: 위 수용 기준 2) 의 명령 그대로. Go 는 `-race` 없이가 release.yml 과 동일(CI 는 `-race`).
- 위험과 피할 것:
  - `.github/workflows/*`, `scripts/release-image.sh`, `scripts/smoke-image.sh` 는 손대지 말 것 — 실패 증거가 없다.
  - PR #23 브랜치(`auto/2026-09-17-1803`)·`internal/auth/*` 를 건드리지 말 것 — 사람이 머지할 보호 경로.
  - 검증 하드닝 계열(입력 길이·제어문자 검사)은 반려 전례(PR 6854375) — 고르지 말 것.

## 차선 후보 (1순위가 코드 변경 없이 끝나므로 이어서 할 것)

- 과제: `streamAiChat` 이 스트림 종료 시 `\n\n` 없이 끝난 마지막 SSE event 를 버림 (가치 2 / 위험 2 / 작업량 S)
- 왜: `web/src/lib/api.ts:533-561` — `reader.read()` 가 `done` 이면 곧바로 `break` 하고 남은 `buffer` 를 파싱하지 않으며 `decoder.decode()`(flush) 도 호출하지 않는다. 서버가 마지막 event 뒤 빈 줄 없이 연결을 닫으면 마지막 텍스트 조각이나 `[DONE]`(→ `finish` 이벤트) 이 사라져 관리 콘솔 AI 채팅(`admin-pages.tsx:2015`) 이 끝나지 않은 것처럼 보일 수 있다.
- 수용 기준: 1) `done` 뒤 `buffer += decoder.decode()` 하고 남은 buffer 가 비어 있지 않으면 같은 블록 파서를 한 번 더 통과시킨다(파서를 내부 함수로 빼서 두 곳이 같은 코드를 쓰게 — "같은 값을 읽는 경로가 여럿이면 모두 같게" 교훈). 2) `web/src/lib/api.test.ts:60` 의 기존 스트림 테스트 옆에 "마지막 event 가 `\n\n` 없이 끝나도 onEvent 가 호출된다" 와 "`data: [DONE]` 으로 끝나면 finish 가 온다" 케이스를 추가하고, HEAD 코드로는 실패하는지 확인한다. 3) 기존 케이스(정상 종료·중간 분할)는 그대로 통과.
- 건드릴 파일: `web/src/lib/api.ts:streamAiChat` — 블록 파서 추출 + done 뒤 flush; `web/src/lib/api.test.ts` — 케이스 2개.
- 검증 명령: `npm --prefix web test`, `npm --prefix web run lint`, `npx --prefix web prettier --check src`(web/ 안에서 `npm run lint` 뒤 `npx prettier --check .`), `npm --prefix web run build`, `./scripts/check-offline-assets.sh web/dist`. Go 변경 없으면 `go vet ./...` 만.
- 피할 것: 서버 쪽 SSE 작성기(`internal/httpapi` ai 핸들러)는 손대지 말 것 — 프런트 파서만.
- 그다음 후보: silent SSO 거절 뒤 `/login?sso=none` 안내 한 줄 — `web/src/pages` 의 LoginPage 가 이미 `useSearchParams` 를 쓴다(2026-09-17 정찰 기준, 이번 회차 미재확인).
