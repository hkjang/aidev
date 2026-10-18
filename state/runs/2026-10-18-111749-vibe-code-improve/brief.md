# 과제서 (정찰, 2026-10-18, base main@6eaede8)

- 과제: `외부 통신 점검`(`vibe-code.auditNetwork`)이 `offlineMode = offline` 에서도 외부 호스트 6곳에 HEAD 요청을 보내는 것 막기 (가치 3 / 위험 2 / 작업량 S)

- 왜: 설정 `vibe-code.offlineMode` 의 `offline` 값은 자기 문서에서 **"폐쇄망 — 외부 호출(npx MCP, 웹 가져오기, 공개 URL git 등)을 막고 로컬 자료를 우선"** / "Air-gapped — block outside calls ... and prefer local sources" 라고 약속한다(`package.nls.ko.json:145`, `package.nls.json:145` — 직접 열어 확인). 그런데 `src/features/diagnostics.ts:168` 의 `registerAuditNetwork` 는 설정을 읽지 않고 `AUDIT_HOSTS`(`1.1.1.1`, `api.anthropic.com`, `api.openai.com`, `registry.npmjs.org`, `github.com`, `huggingface.co`)에 무조건 `https.request(... method:"HEAD")` 를 보낸다. 폐쇄망이라고 선언한 사용자가 점검 명령을 한 번 누르면 외부로 6건이 나간다 — 이 저장소의 다른 네트워크 경로는 이미 같은 설정을 지킨다(`src/features/usage.ts:111` 은 `offline` 이면 `null` 을 돌려주고 조회하지 않으며, `src/features/status-bar.ts:74-79` 는 주석에 "`offlineMode` decides whether we probe the network at all" 이라 적고 `offline`/`online` 을 설정에서 단정한다). 고치면 설정이 약속한 대로 `offline` 에서 외부 호출이 0건이 되고, 보고서가 "외부 호스트에 모두 연결됩니다" 같은 **빈 프로브 기반의 거짓 단언**(`src/features/reports.ts:201` 의 `reached === n.probes.length`, 빈 배열이면 `0 === 0` 으로 참)을 내지 않는다.

- 범위(명시): `offline` 일 때만 건너뛴다. `online`("인터넷이 된다고 보고 확인하지 않기")과 `auto` 는 **지금처럼 프로브한다** — 이 명령은 "실제로 되는지" 를 보는 진단이므로 `online` 에서 건너뛰면 명령 자체가 쓸모없어진다. 상태 표시줄·`usage.ts`·MCP·업데이트 확인은 건드리지 않는다.

- 수용 기준:
  1) `offlineMode = offline` 이면 외부 호스트 프로브가 **한 건도** 생성되지 않는다(프로브 대상 목록이 빈 배열이고, `vscode.window.withProgress("외부 호스트 확인 중")` 도 뜨지 않는다).
  2) 보고서의 판정 한 줄이 빈 프로브에서 "모두 연결됩니다" 가 아니라 "폐쇄망 모드라 확인하지 않았다" 는 뜻의 문장이 된다. 호스트 표 자리에는 빈 표가 아니라 한 줄 안내문이 들어간다(바로 옆 `## MCP 서버` 가 이미 쓰는 패턴: `n.mcp.length ? table(...) : "_등록된 MCP 서버가 없습니다_"`, `src/features/reports.ts:220`).
  3) 권장 목록이 이미 `offline` 인 사용자에게 `offlineMode` 를 `offline` 로 두라고 다시 권하지 않는다(`src/features/reports.ts:227` 의 `if (reached === 0)` 가 빈 프로브에서도 참이 된다 — 프로브가 있을 때만으로 좁힌다).
  4) `auto`/`online` 의 기존 보고서 출력(판정 세 갈래, 호스트 표, 프로바이더·MCP·권장 섹션 순서와 문구)은 **한 글자도 바뀌지 않는다** — 기존 `tests/unit/reports.test.ts` 의 기대값을 수정하지 않고 통과해야 한다.
  5) 테스트가 증명할 것: (a) `offline` 에서 프로브 대상이 `[]`, `auto`/`online`/빈 문자열/대문자 섞임에서는 6개 호스트 그대로 (b) `probes: []` + `offlineMode: "offline"` 인 `networkReport` 출력에 "모두 연결됩니다" 가 없고 폐쇄망 안내가 있으며 권장에 `offline` 로 두라는 줄이 없다 (c) 프로브가 있는 기존 경로의 세 판정(전부 성공 / 전부 실패 / 일부)은 그대로다.

- 건드릴 파일 (프로덕션 2개 + 테스트 1개):
  - `src/features/diagnostics.ts` — `AUDIT_HOSTS`(:151) 바로 아래에 순수 export 를 하나 추가: `export function auditHosts(offlineMode: string): string[]` → `String(offlineMode).trim().toLowerCase() === "offline" ? [] : AUDIT_HOSTS` (사본을 돌려줄지 상수를 돌려줄지는 구현자 판단, 호출부가 변형하지 않으므로 어느 쪽도 됨). `registerAuditNetwork`(:158-185)에서 `offlineMode` 를 **한 번만** 계산해(`const mode = String(config().get("offlineMode") || "auto")`) `auditHosts(mode)` 로 대상 목록을 얻고, 목록이 비면 `withProgress` 를 **호출하지 않고** `probes = []` 로 둔다. `networkReport({ offlineMode: mode, ... })`(:172)에 같은 `mode` 를 넘긴다. `probeHost`·`AUDIT_HOSTS` 내용·`healthCheck`·`showIndexInfo`·MCP 집계(`needsInternet` 판정)는 손대지 않는다.
  - `src/features/reports.ts` — `networkReport`(:200 부근)의 `verdict` 계산 앞에 빈 프로브 분기를 넣고(`n.probes.length === 0` → 폐쇄망 안내 문장), 호스트 표(:216)를 `n.probes.length ? table(...) : "_폐쇄망 모드라 확인하지 않았습니다_"` 꼴로 바꾸고, `:227` 의 `if (reached === 0)` 를 `if (n.probes.length > 0 && reached === 0)` 로 좁힌다. `NetworkInput` 타입·`healthReport`·`commandHistoryReport`·`table`/`cell` 헬퍼는 바꾸지 않는다(새 필드 추가 불필요 — `offlineMode` 가 이미 입력에 있다).
  - 테스트 — `tests/unit/reports.test.ts` 끝에 새 `describe` 를 덧붙이는 것이 1순위. `auditHosts` 를 위해 `src/features/diagnostics.ts` 를 import 해야 하는데 그 모듈은 `vscode`·`../settings`·`../ui/report` 를 끌어온다. vitest alias 가 공유 스텁을 물려 주므로 모듈 로드 자체는 되어야 하지만 **이번 세션에서 실제로 import 해 보지는 않았다(미확인)**. 로드가 깨지면 `auditHosts` 검사만 새 파일 `tests/unit/network-audit.test.ts` 로 분리하라(테스트 파일은 두 개까지 허용, 프로덕션은 2개 고정). 공유 `tests/unit/vscode-stub.ts` 는 고치지 말 것 — 필요하면 테스트 안에서 국소 캐스트로 대입하는 기존 방식(`goal-catalog.test.ts` 가 쓰는 방식)을 따를 것.

- 검증 명령 (Node 20.19.2 = `engines.node` 정확 핀):
  1) 빨간 로그 먼저: 프로덕션 수정 **전에** 테스트를 추가하고 `npx vitest run tests/unit/reports.test.ts` → 실제 값 불일치(`AssertionError`)가 떠야 한다. `auditHosts` 가 아직 없으면 그 테스트는 심볼 누락이 되므로, 순서는 **(1) `auditHosts` 를 `offline` 무시 버전(항상 `AUDIT_HOSTS` 반환)으로 먼저 추가해 배선하고 → (2) 빨강 → (3) 한 줄 수정** 으로 간다(이 저장소의 과거 회차가 쓴 "추출만 → 빨강 → 수정" 순서와 같다). 로그를 `validation/` 에 남길 것.
  2) 집중: `npx vitest run tests/unit/reports.test.ts`(및 새 파일), 전부 통과.
  3) 전체: `npm run check`(typecheck + vitest 전체 + esbuild). 체크포인트 테스트가 상위 `aidev` 저장소를 탐색하지 않도록 `GIT_CEILING_DIRECTORIES=<임시디렉터리>` 를 지정해 돌릴 것(2026-10-09·10-16 회차의 교훈).
  4) 번들 문법: `node --check dist/extension.js`, `node --check dist/extension.core.js`.
  5) 인과 확인: 테스트·새 export 는 그대로 두고 **프로덕션 두 파일만** `git checkout HEAD --` 로 되돌려 같은 `AssertionError` 가 재발하는지 확인.
  - 기준선(미실측, 이전 회차 실측치 인용): main@6eaede8 에서 `npx vitest run` → **19 files / 152 tests**(2026-10-16 회차가 실측). 이번 정찰에서는 테스트를 돌리지 않았으므로 구현자가 수정 전에 한 번 실측해 과제서 숫자를 교정할 것. 워크트리에 `node_modules` 가 없으면 `npm ci` 가 필요하다.

- 위험과 피할 것:
  - **행동 변경이므로 범위를 넓히지 말 것.** `online`/`auto` 를 건너뛰게 만들면 진단 명령이 무의미해지고 기준 4 를 깨뜨린다. `AUDIT_HOSTS` 목록 자체를 설정으로 빼거나 타임아웃·포트를 손보지 말 것.
  - 금지(최근 회차가 미머지로 물고 있는 접근·경로): `src/features/verification.ts`(`suggestTestCommands`·`gitChangedFiles`), `src/features/goal-metrics.ts`·`src/features/command-audit.ts` 의 "최근 N일" 집계, `src/features/journal-summary.ts` 의 exitCode 집계, `src/features/update-check.ts`(`latestVsixVersion`). 같은 접근 재제출 금지 규칙에 걸린다.
  - 보호 경로: `vendor/`, `src/core/`, `src/features/mcp/`, `src/features/checkpoints.ts`, `src/features/vibe-coders-proxy.ts`, `.github/workflows/`, `scripts/*.ps1`, `release/`. 버전·CHANGELOG 도 올리지 말 것.
  - 감사 기록에 새 값을 넣지 말 것(운영자 규칙: `writeAudit` details 에 원문 문자열 추가 금지). 이 과제는 `writeAudit` 를 건드릴 이유가 없다.
  - 문구를 새로 쓰면 `package.nls*.json` 쪽 번역 키와 어긋나지 않게 **보고서 본문 문자열만** 쓸 것(`reports.ts` 의 다른 문장들처럼 하드코딩 한국어). 새 설정·새 명령·새 nls 키를 만들지 말 것.
  - 과장 금지: 증명되는 것은 "프로브 대상 목록이 비고 보고서 문자열이 바뀐다" 까지다. 실제 VS Code UI 에서 알림이 안 뜨는지, Windows 패키징(`npm run vsix`/`verify`/`smoke:vscode`), 원격 CI 는 이 환경에서 확인 불가 — 보고서에 미확인으로 적을 것.

- 차선 후보: **로드맵의 릴리즈 자동화 현황을 현재 워크플로와 맞추기 (가치 2 / 위험 1 / 작업량 S)** — `docs/improvement-roadmap.md:9` 가 VSIX 검증 자동화를 "다음 단계" 로만 적는데 `.github/workflows/release-vsix.yml`·`ui-snapshots.yml` 이 이미 존재한다(워크플로 파일은 읽기만 할 것). 구현된 단계와 남은 단계(태그 push 자체로 release 생성은 미구현)를 분리하고 `docs/maintenance-guide.md` 와 정본이 둘로 갈리지 않게 한쪽에서만 서술할 것. 1순위가 성립하지 않을 때만 쓸 것.
