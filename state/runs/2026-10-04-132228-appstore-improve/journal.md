# 회차 노트 2026-10-04-132228-appstore-improve — appstore
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 13:22] base pinned — main@eab31c6
- [러너 13:22] autonomy release — 

## 정찰 노트
- `/my/apps` 빈 상태를 골랐다: 이번 회차에 실행 없이 코드로 전제를 완전히 확인한 유일한 후보이고 프로덕션 파일 1개로 끝난다. 더 높아 보였던 4건(취소 버튼이 폼을 submit / 반려 사유 오류 미렌더 / 저장 실패 안내 없음 / 상태 배지)은 열어 보니 전부 이미 올바르거나 이미 고쳐져 있어 ideas.json 에 rejected·done 으로 기록했다.
- 차선(`app-form-page.tsx:69-73` 의 `undefined` 반환)이 가치는 더 높지만 TanStack Query v5 의 undefined 처리 동작을 실행해 확인하지 못했다(워크트리에 `web/node_modules` 없음). 과제서에 "미확인" 이라 명시했고 구현자가 재현 테스트로 먼저 관측한 뒤 착수하도록 썼다.
- 조심할 것 ①: `ui.tsx` 의 `EmptyState` 기본값은 bare `<EmptyState />` 6곳이 공유한다 — 공용 기본값을 고치면 무관한 화면 6개가 같이 바뀐다. ②: `public-pages.tsx` 는 열지 말 것 — 즐겨찾기 total·페이지 nav 결함이 2026-10-04 에도 main 에 있으나 `0898f7a` 가 다섯 회차째 review-pending 이라 중복 제출이 된다(새로 찾은 "/favorites 검색 시 빈 상태 문구가 틀림" 3/2/S 도 같은 이유로 보류만 했다).
- 기준선 Vitest 수는 미확인(직전 기록 97건) — 직접 실행해 확인할 것. 프로필(2026-10-03)은 1일 전이고 지금 코드와 어긋난 곳을 찾지 못해 새로 쓰지 않았다(HEAD 만 eab31c6 로 전진).
- [러너 13:27] scout done — `/my/apps` 의 빈 상태가 필터 없는 화면에서 "조건을 바꾸어" 보라고 안내한다 (가치 2 / 위험 1 / 작업량 S)

## 구현 노트
- `MyAppsPage` 의 `EmptyState`(personal-pages.tsx:119-124)에 `title="등록한 앱이 없습니다"` 와 화면에 맞는 description 을 전달했다 — 그 화면엔 필터가 없는데 공용 기본값이 "조건을 바꾸거나" 를 안내하고 있었다. 프로덕션 파일 1개, 두 줄.
- 확신 없는 곳: 없음에 가깝다. 다만 `ui.tsx` 기본값 공유처(bare `<EmptyState />` 6곳)는 코드로만 확인했고 그 6개 화면을 실제로 띄워 보지는 않았다 — 이번 변경이 그 기본값을 건드리지 않으므로 영향은 구조적으로 없다.
- E2E(`test:e2e`)는 돌리지 않았다. 변경이 빈 상태 문자열 두 개뿐이고 과제서 검증 목록에도 없다. Go 테스트도 미실행(`go build ./cmd/server` 만 exit 0) — Go 코드는 건드리지 않았다.
- 일부러 안 한 것: `/my` 대시보드의 빈 상태 description 은 여전히 공용 기본 문구지만 과제서의 "다른 화면까지 같이 고치지 말 것" 에 따라 두었다(ideas.json 에 1/1/S 로 기록). `public-pages.tsx` 는 열지 않았다.
- 다음 역할이 조심할 것: `StatePanel` 이 로딩·빈 상태·오류 모두에 `role="status"` 를 주므로 `findByRole("status")` 는 로딩 패널을 먼저 잡는다. 새 테스트는 제목(`findByRole("heading", ...)`)으로 앵커를 잡고 `closest(".state-panel")` 로 범위를 좁혔다 — 이 패턴을 유지할 것.
- 워크트리에 이제 `web/node_modules` 가 있다. 차선 후보(app-form-page 의 undefined 쿼리)를 다음 회차에 바로 재현해 볼 수 있다.
- [러너 13:31] brief accepted — 채택 — 지목한 `personal-pages.tsx:120` 의 bare `EmptyState`, `ui.tsx:258-260` 의 기본값, 대시보드 `:78-83` 의 "등록한 앱이 없습니다", 
- [러너 13:31] verify passed — 검증 8개 통과 (auto)

## 비평 노트
- 확인한 것: `git diff main...HEAD` 는 프로덕션 1파일 6줄 + 테스트 34줄뿐. `npm --prefix web test` 99 passed / `run lint` exit 0 / `git status` 깨끗(산출물 없음) — 원장 수치와 일치. 테스트 1 은 main 코드에서 반드시 실패한다(`ui.tsx:259-260` 기본값이 렌더되고 MyAppsPage 의 다른 heading 은 PageHeader 뿐이라 오탐 없음). `<EmptyState />` bare 6곳·`ui.tsx` 기본값은 미변경을 코드로 확인.
- 못 본 것: E2E·Go 테스트 미실행(변경에 Go 가 없고, `mock-api.ts:436` 이 비어 있지 않은 `/me/apps` 를 주므로 `visual.spec.ts:30` 캡처가 빈 상태를 지나지 않음을 코드로 확인해 생략 타당하다고 판단). 6개 공용 기본값 화면을 실제로 띄워 보지는 않았다.
- 승인이어도 남는 우려 ①: `/my` 대시보드(`personal-pages.tsx:78-83`)의 description 은 여전히 공용 기본값 "조건을 바꾸거나…" 다 — 같은 증상이 한 화면만 고쳐졌고, 새 테스트 2 의 이름이 그 불일치를 "unchanged" 로 고정하므로 다음 회차는 그 테스트도 함께 손봐야 한다.
- 우려 ②: 테스트 2 는 수정 전후 모두 통과하는 보존 테스트다(원장에 정직하게 명시됨) — 거절 사유로 보지 않았다. 릴리즈 노트는 "`/my/apps` 빈 상태 안내 문구" 한 줄로 충분.
- 보안·법무 차단 없음: 인가 경로·식별자·비밀값·의존성·개인정보 수집이 전혀 없고 위험 구역(internal/auth·migrations·workflows·webui/dist) 미접촉. 판정 approve / risk low.
- [러너 13:34] review approved — 리뷰 승인 (risk=low)
- [러너 13:34] pr created — https://github.com/hkjang/appstore/pull/37
- [러너 13:39] ci passed — 검사 2개 모두 success
- [러너 13:39] merge done — 892c1b8

## 릴리즈 노트
- v2.11.13 (patch, 이전 10회가 모두 patch) 을 `e9aee8a` 로 커밋하고 주석 태그 `v2.11.13` 을 달았다. detached HEAD 그대로이고 원격 전송은 하지 않았다 — 커밋·태그 push 와 GitHub Release 생성·자산 업로드는 `release.yml` 이 태그 push 에 반응해 전부 수행하므로 `github_release:false` / `assets:[]` 다.
- 버전 표기 17파일: web/package.json·package-lock.json(자기 version 2곳만 — line 2626 의 `baseline-browser-mapping: ^2.11.12` 는 무관한 의존성이라 건드리지 않았다), docker-compose.yml, README, docs/USER_GUIDE.md·ADMIN_GUIDE.md, docs/index.html, docs/guides 5곳, screenshot manifest, 캡처 2개, PDF 2개. ADMIN_GUIDE 롤백 예시는 관례대로 한 칸 밀어 `appstore-v2.11.12-stopped`.
- 캡처 90개 재생성: 88개 byte 동일. `/my/apps` 캡처도 동일 — fixture 소유자에게 앱이 있어 이번 수정이 지나는 빈 상태를 캡처가 밟지 않는다(구현·비평 노트가 코드로 예측한 그대로). 관리자 AI 캡처 2개만 변했고 Chromium 으로 디코딩해 비교했다: 공통 영역 평균 차 desktop 0.038/255·mobile 0.017/255, 최대 24~27 이 픽셀의 0.02% 이하(lossy WebP 재인코딩), 차이의 본체는 전체 페이지 아래 여백 높이(desktop 1905→1810, mobile 8867→9012). desktop 의 늘어난 쪽은 단색 배경(30,32,33), mobile 의 늘어난 145행은 luma 19~80 으로 배경·카드 경계 범위이며 글자 밝기(다크 테마 ~200+) 픽셀은 없다. v2.11.12 에서도 같은 두 캡처가 반대 방향으로 흔들렸고 이번 변경은 관리자 AI route 를 지나지 않는다 — 이 페이지의 full-page 높이가 실행마다 흔들리는 것은 이 저장소의 기존 성질로 보인다.
- 다음 역할이 조심할 것 ①: **이 런처 환경은 `HOME` 을 run 디렉터리(`.../home`)로 덮는다.** 그래서 Playwright 가 브라우저를 못 찾고, `test:e2e` 와 `publish-doc-screenshots.mjs` **둘 다** 실패한다(후자는 PNG→WebP 변환에 브라우저를 쓴다). `PLAYWRIGHT_BROWSERS_PATH=/home/hkjang/.cache/ms-playwright` 를 주면 된다 — 요구 revision chromium-1234 가 그곳에 설치돼 있다. `npx playwright install` 은 필요 없었다.
- 조심할 것 ②: `scripts/build-guide-pdfs.sh` 는 여전히 mode 644 라 `sh ./scripts/...` 로 불러야 한다(`GUIDE_TOOL` 기본 경로 `/mnt/c/Users/USER/projects/aidev/tools/guide` 는 존재했다). 이 스크립트도 브라우저를 쓰므로 같은 env 가 필요하다.
- 조심할 것 ③: manifest 의 `generatedForVersion` 이 semver 이기만 하면 `check-docs.sh` 는 통과한다 — 버전을 올리지 않고도 초록불이 나오므로 캡처 재생성 누락을 이 검사가 잡아 주지 않는다. 실제로 manifest 갱신 전에 check-docs 가 통과하는 것을 관측했다.
- 검증(모두 exit 0): gofmt·go vet·`go test -race` 전 패키지, React 99건, lint(`--max-warnings 0`), `prettier --check`, web build, check-offline-assets, check-env-contract, check-docs(manifest 갱신 후 재실행), `go build ./cmd/server`(산출물은 /tmp 로 빼 워크트리를 더럽히지 않았다), Playwright 81 passed/1 skipped(retry 0). PDF 는 표지 v2.11.13·태그 링크 2개씩·`file://` 0건 확인. DB DSN 미설정으로 DB 통합과 실제 Keycloak 은 검증하지 않았고, Docker build/load/smoke 와 SHA-256 은 태그 워크플로가 수행한다.
- `internal/webui/dist` 는 기존 관례대로 손대지 않았다(이전 릴리즈 커밋들도 포함하지 않는다). 커밋 파일 수 17개는 v2.11.12(19개, 캡처 4개 변경)와 같은 모양이다.
- [러너 13:53] release published — v2.11.13
- [러너 13:55] assets verified — v2.11.13 자산 1개 (이전 v2.11.12: 1)
