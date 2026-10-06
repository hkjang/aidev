# 회차 노트 2026-10-07-005838-umm-improve — umm
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:58] base pinned — main@29149e5
- [러너 00:58] autonomy low-risk — 회귀(reverted) 2026-10-06T22:38:39+09:00
- [러너 01:07] scout timeout — 단계 제한 시간 초과
- [러너 01:08] scout done — 수정 과제 — 설치 누락을 이름 붙여 말해 주는 것이 프런트엔드 게이트 여덟 개 중 세 개뿐이다 (`typecheck`

## 구현 노트
- 무엇을 왜: 설치 없는 체크아웃에서 `typecheck`·`lint`·`build` 가 이름 없는 `sh: 1: tsc: not found` / exit 127 로 죽던 것을, 새 `web/scripts/require-installed.mjs` 를 `pretypecheck`·`prelint`·`prebuild` 로 걸어 `run-on-supported-node.mjs` 와 같은 한 문장(패키지 이름·manifest 경로·`npm ci --prefix`·"missing install, not an unsupported interpreter")으로 바꿨습니다. 프로덕션 2파일(가드 + `package.json` 3줄), 시험 1파일. 느슨하게 만든 것 없음 — 검사를 더하기만 했습니다. 커밋 9769548.
- 확신 없는 곳·검증 못 한 것: (a) 종료 코드가 **127 → 1** 로 바뀝니다. 비영이므로 수용 기준을 만족하고 게이트는 여전히 멈추지만, 127 을 보고 분기하는 외부 도구가 있다면 영향을 받습니다 — 러너 `verify.json` 과 `ci.yml` 은 비영만 보므로 괜찮다고 판단했지만 그 두 곳 밖은 확인하지 않았습니다. (b) 새 시험의 `runNpmScript` 가 자식 env 에서 `npm_*` 전부와 PATH 의 `node_modules` 항목을 걷어냅니다(바깥 `--prefix web` 이 임시 루트를 실제 체크아웃으로 되돌리고, 리프된 `.bin` 이 임시 루트에 없는 `tsc` 를 주기 때문). 이 격리는 이 환경에서만 확인했고 CI 러너의 env 에서도 같은지는 못 봤습니다 — 거기서 깨지면 시험이 거짓 통과가 아니라 **실패**하는 방향입니다. (c) `npm_execpath` 가 없는 환경에서는 PATH 의 `npm` 으로 폴백하는데, 그 경로는 이 환경에서 타지 않았습니다(여기서는 항상 set).
- 일부러 하지 않은 것: 워크플로·`Makefile`·`Dockerfile`·`engines`·`run-on-supported-node.mjs`·루트 `package.json`·`VERSION`·락파일 전부 무수정(과제서 금지, 읽기만 함). 두 파일의 같은 문장을 통합하지 않음(운영자 규칙 3번 — 한쪽은 로드 즉시 `spawnSync` 하는 진입점). `e2e`·`dev`·`preview` 는 CI·릴리즈 경로가 아니라 범위 밖(ideas.json 에 적음). 테스트 타깃이 `npm ci` 를 스스로 돌리게 만들지 않음(운영자 규칙 10번).
- 다음 역할이 조심할 것: 새 시험 10개는 **실제 `npm` 을 자식 프로세스로 띄웁니다** — DB 는 필요 없지만 `npm` 이 PATH 나 `npm_execpath` 로 닿아야 하고, 임시 루트를 `/tmp` 에 만들므로 `/tmp` 쓰기 권한이 필요하며, 한 시험당 타임아웃을 120s 로 잡았습니다(이 환경 실측 총 ~1.0s). 릴리즈 경로를 확인할 때는 `docker build` 의 EXIT=0 만 보지 마세요 — 제 첫 빌드는 거의 전부 CACHED 였습니다. `--no-cache-filter web-build --progress=plain` 으로 이미지 안에서 `prebuild` 가 조용히 통과한 뒤 `vite build` 가 도는 것을 로그로 봐야 증거가 됩니다. 가드는 성공 시 **아무것도 출력하지 않습니다**(게이트 로그를 더럽히지 않기 위한 의도) — 로그에 안 보이는 것이 안 돌았다는 뜻이 아닙니다.
- [러너 01:27] brief accepted — 채택 — 과제서가 **미확인**으로 남긴 세 가지를 지시대로 모두 실행으로 확인했고 전부 과제서의 추론과 맞았습니다: �
- [러너 01:27] verify passed — 검증 17개 통과 (auto)

## 비평 노트
- 확인: base 의 `web/package.json` 으로 임시 루트를 만들어 수정 전 증상(`sh: 1: tsc: not found` / `oxlint: not found` / EXIT=127)을 직접 찍었다 — 새 시험은 그 코드에서 반드시 깨진다. 구현 노트의 미확인 3건 전부 실행 확인: 127→1 로 분기하는 곳 없음(grep), `npm_*`·`npm_execpath` 를 지운 환경에서 10 passed, `npm_execpath` 부재 폴백도 10 passed.
- 확인: `npm --prefix web test` 23파일/240시험 EXIT=0, typecheck·lint·build EXIT=0, 새 파일 2개 prettier EXIT=0. 릴리즈 경로는 CACHED 를 믿지 않고 `--no-cache-filter web-build --progress=plain` 로그에서 `prebuild` 가 조용히 통과한 뒤 `✓ built in 950ms` 로 이어지는 것을 보았다.
- 못 본 것: GitHub Actions 러너의 실제 env(여기서는 흉내만), 전체 `docker build`(web-build 스테이지까지만), Playwright·Go 쪽(이번 diff 가 닿지 않음).
- 승인이어도 남는 우려: 커밋 제목의 "every frontend gate" 는 `make test-web` 여덟 게이트 기준이며 `npm run e2e`·`dev`·`preview` 는 여전히 이름 없는 127 로 죽는다(원장 보류 아이디어에 있음 — 릴리즈 노트에 "모든 스크립트" 로 적지 말 것). 가드는 `<dep>/package.json` 이 `exports` 로 노출되는 데 의존하므로 장래 의존성 범프가 그것을 빼면 설치돼 있는데도 게이트가 raw `ERR_PACKAGE_PATH_NOT_EXPORTED` 로 죽는다 — "설치된 체크아웃" 통제 시험이 그 시점에 잡아 줄 것이니 그 시험을 지우지 말 것.
- 보안·법무 차단 없음: 새 경로·인가·비밀값·암호 비교·외부 요청·의존성·개인정보 전무.
- [러너 01:41] review timeout — 단계 제한 시간 초과
- [러너 01:41] review approved — 리뷰 승인 (risk=low)
- [러너 01:41] pr created — https://github.com/hkjang/umm/pull/169
- [러너 01:53] ci passed — 검사 1개 모두 success
- [러너 01:53] merge done — 9769548
- [러너 01:53] release skipped — 자율화 단계 low-risk — 릴리즈는 사람이
