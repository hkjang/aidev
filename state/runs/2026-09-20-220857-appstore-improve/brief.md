- 과제: [수정 과제] PR #26 즐겨찾기 E2E의 모바일 메뉴 탐색 누락 수정 (가치 5 / 위험 1 / 작업량 S)
- 왜: PR #26의 새 즐겨찾기 E2E가 모바일에서 닫힌 사이드바의 링크를 클릭해 CI를 막고 있다. 실제 사용자의 메뉴 열기→즐겨찾기 이동을 테스트에 반영하면 기능과 CI 검증 강도를 유지하면서 미병합 개선을 검증할 수 있다.
- 수용 기준: 1) PR a4d9997의 원래 테스트가 mobile에서 core.spec.ts:584의 viewport 밖 링크 클릭으로 실패함을 재현·기록한다. 2) /my/apps·/my·/apps 각각에서 모바일은 실제 `메뉴 열기` 버튼→`주 메뉴` 열림→즐겨찾기 링크 클릭을 거치고 desktop은 기존 사이드바로 이동한다. 3) 두 프로젝트 모두 공개 앱 추가→aria-pressed/실제 localStorage→메뉴로 /favorites→해제→새로고침 후 빈 상태라는 기존 단언을 모두 통과하며, 소유자 비공개 카드 숨김·기존 저장 문자열 보존 테스트도 통과한다. 4) CI와 같은 전체 desktop/mobile E2E 및 아래 검증 명령이 exit 0이고 워크플로·재시도·timeout·skip·viewport·강제 클릭은 변경하지 않는다. 5) ledger-entry.md에 ‘수정 과제’, 원인/수정/수정 전후 명령과 결과를 적고 실제 완료 뒤에만 ideas 상태를 done으로 바꾼다.
- 건드릴 파일: `web/e2e/core.spec.ts` — PR head 567행의 `공개 앱 즐겨찾기는 소유자와 공개 목록에서 추가하고 즐겨찾기에서 해제한다` 테스트만 최소 수정. 반복문 안에서 메뉴 링크를 누르기 직전에 mobile 프로젝트이면 `page.getByRole("button", {name:"메뉴 열기", exact:true}).click()`하고 `page.getByRole("complementary", {name:"주 메뉴"})`의 open 상태/링크 viewport 진입을 확인한다. callback의 두 번째 testInfo 인자는 기존 101행 모바일 메뉴 테스트를 참고한다. 경로 이동·reload마다 메뉴가 닫히므로 반복문 밖에서 한 번만 열면 안 된다. `web/src/components/app-shell.tsx:Sidebar, AppShell`·`web/src/styles.css`·`web/src/features/apps/app-card.tsx:AppCard, publiclyViewable`·`web/src/features/apps/favorites.tsx:FavoritesProvider`는 원인/회귀 확인용이며 이번 수리로 변경하지 않는다.
- 검증 명령: 저장소 루트에서 `npm --prefix web ci --no-audit --no-fund`, `npm --prefix web run build`, `(cd web && npx playwright install chromium)`, `npm --prefix web run test:e2e -- core.spec.ts --project=mobile -g '공개 앱 즐겨찾기는'`, `npm --prefix web run test:e2e -- core.spec.ts -g '공개 앱 즐겨찾기는|게시되지 않은 내 앱'`, `CI=true npm --prefix web run test:e2e`, `npm --prefix web test`, `npm --prefix web run lint`, `(cd web && npx prettier --check e2e/core.spec.ts)`, `./scripts/check-offline-assets.sh web/dist`, `./scripts/check-env-contract.sh`, `./scripts/check-docs.sh`, `git diff --check`. CI 나머지 확인이 필요하면 `go test -race . ./cmd/... ./internal/... ./migrations/... ./openapi/...`. CI 브라우저 준비는 `(cd web && npx playwright install --with-deps chromium)`이며 Linux 시스템 의존성이 없을 때 사용한다. 기본 preview 4173이 점유되면 실행 명령에 `APPSTORE_PREVIEW_PORT=4317`을 주고 다른 앱 서버를 재사용하지 않는다.
- 위험과 피할 것: `.github/workflows/*`, `web/playwright.config.ts`, auth, migrations, 릴리즈/스모크 스크립트를 바꾸지 않는다. `force:true`, DOM click/evaluate, 직접 page.goto('/favorites'), 테스트 삭제/모바일 skip, timeout 증가로 탐색 결함을 우회하지 않는다. 닫힌 사이드바 링크도 Playwright toBeVisible이 참일 수 있으므로 가시성만 보고 메뉴 열기를 생략하지 않는다. 하트 표시 기능을 롤백하거나 기기 동기화·100개 제한·비활성 카테고리 수정까지 묶지 않는다. 캡처/manifest/PDF/임베드 번들은 이번 수정 커밋 대상이 아니다.
- 차선 후보: 같은 수정 과제의 실제 잔여 CI 실패 원인 재확정 — PR이 갱신되어 이 줄이 이미 수리됐다면 최신 SHA/실패 단계 로그를 다시 확인하고 같은 릴리즈 차단 문제 안에서 과제서를 갱신한다. 다른 기능 후보로 바꾸지 않는다.

근거와 작업 기준

- 정찰 작업 트리는 main `a858bf43d8c3e857e9737916f13166372bdc86dc`이고, PR #26은 open/unmerged, head `auto/2026-09-20-2144` / `a4d9997b17e3e2c0849187a38d499661ea28f15f`이다. 신규 테스트는 main에는 없으며 `git show a4d9997:web/e2e/core.spec.ts`로 읽었다. 구현 시작 시 `git log -1 --oneline`, `git merge-base --is-ancestor a4d9997 HEAD`로 작업 기반을 확인한다. PR 수리 컨텍스트이면 PR head 위에서 수정한다. 러너가 현재 pinned main에서 구현을 시작한다면 기존 PR 커밋의 변경을 먼저 해당 구현 브랜치에 가져와 기능과 회귀 테스트를 유지한 뒤 수리해야 한다(이미 포함됐으면 중복 적용 금지). 정찰은 checkout/cherry-pick/커밋/원격 변경을 하지 않았다. 다른 PR을 중복 개설하거나 기존 PR을 닫는 일은 이 과제의 범위가 아니다.
- GitHub CI run [35512038178](https://github.com/hkjang/appstore/actions/runs/35512038178), job [106081501201](https://github.com/hkjang/appstore/actions/runs/35512038178/job/106081501201), step 13 `Run desktop and mobile E2E tests`에서 실패했다. Go·React·빌드·환경/문서 계약·Chromium 설치 단계는 성공했고 image smoke job은 skipped다. 결과는 68 passed / 1 skipped / 1 failed. 실패 로그는 `element is outside of the viewport`, `core.spec.ts:584:65`, 30초 timeout이며 최초+Retry #1+#2 모두 동일하다. `ci-failure-excerpt.log`에 발췌를 남겼다.
- 자동 적재 문구의 ‘릴리즈 워크플로 두 번 실패’는 확인된 사실과 다르다. 이 PR에서 확인한 것은 CI 실행 1건과 그 안의 재시도 2회이며, 별도 release.yml 실패 2건은 미확인이다. release.yml의 Install and test에는 E2E가 없으므로 해당 파일을 손댈 근거가 없다. 직전 stages.json의 verify는 passed, ci는 failed다.
- 원인: styles.css의 820px 이하 `.sidebar`가 `translateX(-105%)`, `.sidebar.open`만 `translateX(0)`이다. app-shell.tsx의 AppShell은 mobileOpen=false로 시작하고 pathname 변경 시 닫으며 Sidebar NavLink도 close를 호출한다. 새 테스트는 열기 절차 없이 곧장 링크를 클릭했다. Chromium 설치 실패나 즐겨찾기 저장 실패가 아니다.

실행 순서와 체크포인트 (구현자가 상태를 갱신할 것)

1. [pending] 위 PR 기준 확인 및 수정 전 단일 mobile 테스트 실패 증거 확보. 증명: 위 mobile -g 명령. 사람 승인 체크포인트 없음; 로그가 다르면 이 계획부터 수정하고 진행한다.
2. [pending] core.spec.ts의 해당 반복문에 실제 메뉴 열기 동작을 추가. 증명: 단일 테스트 desktop/mobile와 비공개 소유자 테스트 통과. 사람 승인 체크포인트 없음; 공개 추가/해제·저장값 단언을 유지한다.
3. [pending] 전체 CI E2E와 React/lint/build/계약 검사 실행, 수정 과제 원장 기록. 증명: 정확한 실행 명령과 exit code/집계 기록. 사람 승인 체크포인트 없음; 이 단계 실패 시 done/성공으로 기록하지 않는다.

대안 검토와 예상 작업량

- 채택: 기존 테스트에서 실제 모바일 메뉴를 연다. 단일 테스트 수정이며 현재 사용자 탐색 계약을 그대로 검증한다.
- 공용 메뉴 이동 헬퍼: 여러 테스트가 같은 절차를 반복할 때 유효하지만 이번 실패는 한 곳이므로 추상화/호출처 확장은 후순위다.
- 기존 모바일 메뉴 테스트와 즐겨찾기 직접 URL 테스트를 분리: 저장 기능 단독 테스트에는 가능하지만 이번 추가→메뉴 이동의 통합 계약을 끊으므로 선택하지 않는다. 무수정 재시도는 같은 결정적 실패가 3회 나와 효과가 없다.
- 핵심 가정: 최신 PR도 동일 head/실패이며 실제 메뉴 버튼은 정상 동작한다. 구현 시작 재확인 및 단일 브라우저 테스트가 이 가정을 검증한다.
- Bottom-up 추정: 기반/재현 5–8분, 한 테스트 수정 3–5분, 전체 검증 10–17분, 기록 3–5분 = 21–35분. Chromium/포트 등 알려진 환경 편차 예비 5–10분을 별도로 두어 26–45분(S), 주관적 신뢰 중간이다. 이는 시간 약속/통계적 80% 보장이 아니다. 별도 관리 예비는 배정하지 않았으며 다른 CI 결함이 나오면 범위와 추정을 갱신한다. 직전 회차의 이미 작성된 E2E/CI 실행시간(원격 E2E 2.2분, timeout/retry 포함)과 비교했으나 통계적 유사 작업 표본은 없으므로 정밀 추정으로 해석하지 않는다.

적용 스킬

Skill 호출 도구는 이 세션에 없지만 아래 실제 원본을 찾아 읽고 대안 비교·작업별 증명/체크포인트·가정/범위/예비시간을 반영했다. 별도 강제 JSON 반환 형식은 해당 파일에 없다.
- [pmo:estimating-and-contingency](/mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md)
- [technology:implementation-planning](/mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md)
- [technology:solution-exploration](/mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md)
- 추정 근거의 범위·가정·위험·갱신 원칙은 스킬의 references/sources.md와 [GAO Cost Estimating Guide](https://www.gao.gov/products/gao-20-195g)를 확인했다. 위 분 단위 숫자는 저장소 근거에 따른 정찰자의 추정이다.

정찰 로컬 재현 결과

- 작업 트리에 쓰지 않고 지정 run 폴더의 `repro/`에 `git archive a4d9997 web` 원본만 추출했다. PR의 추적된 web 파일 전부를 git show 원문과 바이트 비교해 변경 없음 확인. 해당 스냅샷에서 lockfile 기준 npm ci 및 npm run build exit 0.
- 실행 cwd: `/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-220857-appstore-improve/repro/web`.
- `PLAYWRIGHT_BROWSERS_PATH=/home/hkjang/.cache/ms-playwright APPSTORE_PREVIEW_PORT=4317 npm run test:e2e -- core.spec.ts -g '공개 앱 즐겨찾기는' --reporter=line` → exit 1, desktop 1 passed / mobile 1 failed(34.9초). 원격과 똑같은 584행, viewport 밖 링크 클릭으로 timeout. `local-repro-browser-cache.log` 참조.
- 첫 실행은 격리 HOME에 Chromium이 없어 두 프로젝트 launch 실패(`local-repro.log`)했으므로 제품 실패 증거로 쓰지 않았다. 설치된 동일 revision Chromium 캐시를 PLAYWRIGHT_BROWSERS_PATH로 지정해 위 재현을 얻었다. HOME 등 전역 설정은 변경하지 않았다.
- 현재 main의 `./scripts/check-env-contract.sh`, `./scripts/check-docs.sh`는 exit 0. 정찰은 코드를 수정하지 않았으므로 수정 후 전체 검증은 아직 미실행이며 구현자의 수용 기준이다.
- 같은 PR 원본에서 `PLAYWRIGHT_BROWSERS_PATH=/home/hkjang/.cache/ms-playwright APPSTORE_PREVIEW_PORT=4317 npm run test:e2e -- core.spec.ts --project=mobile -g '모바일 메뉴는' --reporter=line` → exit 0, 1 passed(5.1초). 기존 테스트가 실제 메뉴 버튼을 눌러 주 메뉴 open/MCP 링크 표시를 증명한다(`local-existing-menu.log`). 이는 새 즐겨찾기 테스트 수정 후 통과를 대신하지 않는다.
