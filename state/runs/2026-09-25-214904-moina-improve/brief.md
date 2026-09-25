- 과제: CI `Image and browser smoke` 실패가 어느 화면 때문인지 알 수 없는 것을 고친다 — 실패 상세를 `$GITHUB_STEP_SUMMARY`로 올리고, 시각 회귀 여유가 0.155%p 밖에 없는 베이스라인을 CI renderer 기준으로 재승인한다 (가치 4 / 위험 2 / 작업량 M)
- 왜: PR #31(`972113f`)은 `Source tests`는 통과하고 `Image and browser smoke`만 실패했는데(`ci-972113f6d535.json:15`), 남은 단서가 annotation "exit code 1" 하나뿐이라 지난 회차 수리자가 이미지 잡 전체를 로컬에서 2회 재현하고도(둘 다 통과) 원인을 특정하지 못했다 — artifact는 인증이 없어 403. `e2e/VISUAL_REGRESSION.md:45`가 "변경하지 않은 화면도 공식 renderer에서 0.000~0.35%"라고 적고 있고 임계값은 `visual-regression.mjs:21`의 0.5%이므로, 최악 화면(`dark-desktop-admin-smtp`, 로컬 재현 0.345%)은 여유가 0.155%p뿐이라 러너에서 한 화면이 넘는 실패가 반복될 조건이 이미 갖춰져 있다.

- 수용 기준:
  1) `.github/workflows/ci.yml`의 image 잡이 실패했을 때, artifact를 열지 않고 **check-run 출력(=job summary)만으로** 실패한 e2e 단계와 화면 id·diff 비율을 알 수 있다. 즉 `e2e/test-results/visual/visual-regression.json`·`e2e/test-results/browser-smoke.json`·`e2e/test-results/accessibility-regression.json` 중 존재하는 것의 실패 항목을 `$GITHUB_STEP_SUMMARY`에 적는 `if: failure()` 단계가 있고, 기존 `Upload failure diagnostics` artifact 업로드는 그대로 남는다.
  2) 시각 회귀 임계값(`MOINA_VISUAL_MAX_DIFF_RATIO` 기본 0.005)·재시도·`continue-on-error`·테스트 건너뛰기를 **늘리거나 추가하지 않는다**. 통과 조건은 지금과 동일해야 한다(느슨화 금지).
  3) 테스트가 증명할 것: 일부러 실패를 만든 상태(예: `MOINA_VISUAL_MAX_DIFF_RATIO=0` 또는 베이스라인 1장을 임시 변조)에서 새 요약 단계의 스크립트를 로컬에서 실행하면, 출력에 실패한 화면 id와 diff 비율이 **문자열로 실제 나타난다**. 정상 통과 상태에서는 이 단계가 아무것도 깨뜨리지 않는다.
  4) (여유가 남으면) 공식 `mcr.microsoft.com/playwright:v1.62.1-noble`에서 `npm run test:visual`을 돌려 0.15% 이상 어긋나는 화면을 찾아 그 화면만 `MOINA_VISUAL_ONLY=<slug>`로 재승인하고, 재승인 후 52/52 비교가 통과하는 것과 해당 화면 diff가 0.000%로 돌아온 것을 보인다. 찾지 못하면 재승인하지 말고 "재승인 대상 없음"으로 적는다.

- 건드릴 파일:
  - `.github/workflows/ci.yml` — image 잡 `Upload failure diagnostics`(166~174줄) **앞**에 `if: failure()`인 "e2e 실패 요약" 단계 추가. 기존 단계·타임아웃·이미지 digest·cleanup은 건드리지 말 것. 참고로 같은 파일 `Dependency audit`(89줄)이 이미 `${GITHUB_STEP_SUMMARY}`를 쓰는 선례다.
  - (선택) `e2e/visual-baselines/` 의 해당 PNG + `manifest.json` — 4)를 할 때만. `VISUAL_REGRESSION.md:20~30`의 명시적 `visual:update` 절차를 따를 것.
  - 읽어야 할 것(수정 대상 아님): `e2e/visual-regression.mjs:12,21,164~170,210~211,528`(결과 JSON·actual/diff PNG 경로와 실패 목록 형식), `e2e/browser-smoke.mjs:9~11,407~411`, `e2e/accessibility-regression.mjs:9~11,373~379`, `e2e/package.json`의 `test`(visual → accessibility → smoke 순서, `&&` 연쇄라 앞이 실패하면 뒤는 안 돈다).

- 검증 명령:
  - 요약 스크립트 단위: 실패 상태 JSON을 만든 뒤 새 단계와 같은 셸 조각을 직접 실행해 화면 id가 출력되는지 확인.
  - 시각 회귀 재현(느림, docker 필요): `make image` → postgres:17-alpine 새 DB → `--read-only` 앱을 `127.0.0.1:18080`에 띄우고
    `docker run --rm --network host --user "$(id -u):$(id -g)" -e HOME=/tmp -e MOINA_E2E_BASE_URL=http://127.0.0.1:18080 -e MOINA_E2E_USERNAME=... -e MOINA_E2E_PASSWORD=... -v "$PWD:$PWD" -w "$PWD/e2e" mcr.microsoft.com/playwright:v1.62.1-noble npm run test:visual`
    (`e2e/VISUAL_REGRESSION.md:48~52`의 명령 그대로)
  - 회귀 없음 확인: `make check` (OpenAPI route 120 유지), `bash -n scripts/*.sh`는 `make check`에 포함됨. 백엔드 무변경이면 `cd backend && go test -race -count=1 ./...`는 생략 가능하나 돌리면 `MOINA_TEST_POSTGRES_DSN`을 주고 `--- SKIP` 0줄을 확인할 것.

- 위험과 피할 것:
  - **워크플로를 느슨하게 만들어 통과시키는 것은 금지**다 — 임계값 상향·`continue-on-error`·재시도·`--retries`·테스트 제외는 전부 반려 사유. 추가하는 것은 진단 출력뿐이어야 한다.
  - `.github/workflows/release.yml`은 열지도 고치지도 말 것(정확한 commit의 CI 성공을 요구하는 보호 경로). `ci.yml`의 `source` 잡, postgres digest 고정, `concurrency`, `permissions: contents: read`도 그대로 둘 것.
  - `$GITHUB_STEP_SUMMARY`에 로그 전문을 쏟아붓지 말 것 — 실패 항목만. (운영자 지시: 감사·로그로 넘기는 값은 식별자만.)
  - 재현 불가한 상태로 베이스라인을 갱신하거나 임계치를 늘리는 것은 지난 회차 수리자가 이미 거부한 선택이다. 4)는 **공식 noble 이미지에서 실제로 0.15% 이상을 눈으로 본 화면**에만 적용하고, 그렇지 않으면 하지 말 것.
  - 로컬에서 WSL·macOS Chromium으로 시각 회귀를 돌리면 52장 전부 2~12% 어긋난다(`VISUAL_REGRESSION.md:45`). 그 결과를 증거로 쓰지 말 것.
  - `972113f`의 `social.go`·`openapi.yaml`·새 integration 테스트는 **되돌리지 말 것** — `Source tests`가 통과했고 수리자가 이미지 경로와 무관함을 확인했다(Dockerfile에 openapi 없음, `_test.go`는 바이너리에 없음, e2e/smoke에 `topic`·`follow` 참조 0건).

- 차선 후보: POST /media 거절 경로(415·인증·CSRF) 실제 multipart integration 테스트 — `media_upload_test.go`가 `uploadMedia`를 직접 호출하고 `media_upload_postgres_integration_test.go`는 성공 파일명만 검사해 거절 경로 커버리지가 0이다. 동작 변경 없는 테스트 보강이라 위험이 낮고, 1순위가 "요약할 실패 정보가 애초에 없다"로 성립하지 않을 때 고를 것. 이미 릴리즈된 413 `media_too_large`(1bd4408)는 반복 금지.

---

## 정찰이 확인한 것 / 추측인 것
- 확인: `Source tests`=success, `Image and browser smoke`=failure (`.../2026-09-25-192055-moina-improve/ci-972113f6d535.json:15,154`). annotation 1건, artifact 163KB.
- 확인: 지난 회차 수리자가 이미지 잡 전 구간을 로컬 2회 재현해 모두 통과시켰고 커밋 없이 끝냈다(`.../2026-09-25-204035-moina-shepherd/fix-summary.md`).
- 확인: `visual-regression.mjs:21` 기본 허용 0.005, `:210~211`에서 실패 목록을 `id: reason`으로 모아 assert, `:164~168`이 `<id>-actual.png`·`<id>-diff.png`를 결과 디렉터리에 쓴다. 세 e2e 스크립트 모두 `e2e/test-results` 아래에 결과 JSON을 남긴다.
- 확인: `ci.yml`에는 `$GITHUB_STEP_SUMMARY` 사용처가 `Dependency audit` 한 곳뿐이라, 실패한 e2e 단계 상세는 artifact를 열어야만 보인다.
- **추측(미확인)**: 실패한 e2e 단계가 셋 중 어느 것인지. 실패가 시각 회귀라는 것도 `dark-desktop-admin-smtp`의 0.155%p 여유라는 정황뿐이고 증명되지 않았다. 그래서 1순위를 "임의 추측 수정"이 아니라 "다음 실패가 스스로 이름을 말하게 하는 진단"으로 잡았다.
- **미실행**: 이 정찰 세션에서는 `docker run`·`git worktree add`·`gh`가 전부 권한 거부되어 테스트를 한 번도 돌리지 못했다. 위 명령은 저장소 문서·CI 파일에 실제로 있는 것을 옮긴 것이지 이번에 실행해 본 것이 아니다.
