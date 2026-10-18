# 수리 요약 — 고치지 않음 (지적이 이 변경의 결함이 아니다)

**결론: 커밋 없음.** CI 실패는 PR #22 의 코드 결함이 아니라 계정 차원의 GitHub Actions 거부다.

**근거 1 — 물리적으로 불가능한 소요 시간.** `ci-3453f1e31e8d.json` 의 두 실패 잡은 모두
`started_at 2026-10-18T02:32:32Z → completed_at 02:32:34Z` = **2초**이고, `output.title/summary/text`
가 전부 `null`, 스텝 로그가 0건이다(회차가 캡처한 실패 로그도 빈 문자열). 두 워크플로의 첫 스텝은
`actions/checkout@v4` → `setup-node@v4` → `npm ci` 인데, 로컬에서 `npm ci` 만 수십 초가 걸린다.
2초 안에 실패할 수 있는 스텝이 존재하지 않는다 — 즉 **어떤 스텝도 실행되지 않았다**. 저장소 내용이
무엇이든 이 시간 안에 실패를 만들 수 없으므로, revert 로 재현/소거할 대상 자체가 없다.
`package VSIX (windows)` 는 `conclusion: skipped` 에 `completed_at(02:32:34) < started_at(02:32:35)` —
`needs: check` 가 먼저 거부되어 실행 전에 건너뛴 잡의 형태다.

**근거 2 — 비공개 저장소만 거부됐다.** 같은 계정·같은 시각의 다른 저장소 Actions 는 정상이었다:
`umm` verify success 02:32:11→02:43:02 (11분), `visitflow` test/e2e success 02:42:50→02:45:57.
미인증 `api.github.com/repos/hkjang/<repo>` 응답으로 공개/비공개를 가르면 2026-10-18 자료가 정확히
갈린다 — 공개(200) `umm`·`visitflow`·`madi`·`Invenqor` 는 수 분~수십 분 정상 실행, 비공개(404)
`vibe-code`·`yeopjari-your`·`nexabuilder`·`DartFly` 는 2~3초 즉시 실패. Actions 분(minutes)은 공개
저장소에서 무료, 비공개에서만 과금된다. 결제 실패 또는 지출 한도 초과가 비공개 저장소 잡만 즉시
거부하는 것과 일치하고, GitHub 전역 장애나 워크플로 YAML 오류(그러면 공개 저장소도 같이 멈추거나
`startup_failure` 로 뜬다)로는 이 분리를 설명할 수 없다. 참고로 같은 비공개 `qurio` 는 2026-10-08 에
19분간 정상 실행했다 — 저장소 내용이 아니라 시점(= 결제 상태)이 갈림선이다.

**근거 3 — 게이트는 로컬에서 통과한다.** 워크트리에서 `npm ci` 후 CI 와 같은 Node **20.19.2**
(`engines.node`, ci.yml `node-version: 20`)로 `npm run check` 실행: `tsc --noEmit` 무오류,
`vitest run` **19 files / 155 tests 통과**(main 152 + 이번 3), `npm run build` 성공, exit 0.
`dist/` 런타임 자산 누락 경고는 main 과 동일한 기존 동작이다. 이 변경이 건드린 파일은
`src/features/diagnostics.ts`, `src/features/reports.ts`, `tests/unit/reports.test.ts` 3개뿐이고
`.github/`·`package.json`·`package-lock.json` 은 손대지 않았다.

**확인하지 못한 것.** 잡마다 달린 `annotations_count: 2` 의 원문은 읽지 못했다 —
`vibe-code` 가 비공개라 미인증 API 는 404 이고 이 환경에 `gh` 토큰이 없다. 거부 문구를 직접
인용하지 못했으므로 "비공개 저장소 과금"은 위 세 갈래 정황의 최적 설명이지 인용된 사실은 아니다.
다만 **"이 변경 때문이 아니다"** 는 근거 1·3 만으로 성립한다. Windows 패키징, UI 스냅샷 실제 촬영,
실제 VS Code 설치는 이번에도 미확인(앞선 회차와 동일).

**다음 할 일(코드 아님).** 계정의 Actions 결제/지출 한도를 해제하고 두 체크를 재실행(re-run)하면
된다. 워크플로 파일이나 검증 명령을 고쳐 통과시키는 것은 금지 사항이라 하지 않았다.
