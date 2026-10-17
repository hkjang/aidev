- 과제: 지난 회차가 PR 전에 끊겨 사라진 `splitSheetRange` 의 `''` 해제 수정을 main 위에 다시 올린다 (가치 3 / 위험 1 / 작업량 S)
- 왜: 2026-10-17-1219 회차는 구현·검증까지 모두 초록이었지만 PR 을 열기 전에 끊겨(`error: interrupted before PR`), 커밋 `e5b0eb4` 가 `auto/2026-10-17-1219` 브랜치에만 남고 main 에 들어오지 못했다 — 지금 main(`b0481a9`)의 `web/src/lib/compareLists.ts:130` 은 여전히 `.replace(/^'(.*)'$/,'$1')` 한 줄이라 `'Bob''s Data'!A1:B2` 의 시트 이름이 `Bob''s Data` 로 나온다. 같은 결함을 세 번째로 다시 분석하지 말고, 이미 검증된 그 변경을 그대로 되살려 회차를 닫는 것이 이번 회차의 일이다.
- 수용 기준:
  1) `splitSheetRange("'Bob''s Data'!A1:B2")` 의 `sheetName` 이 `Bob's Data` 다 (지금은 `Bob''s Data`).
  2) 따옴표로 감싸지 않은 이름은 글자 그대로 남는다 — `A''B!A1:B2` → `A''B`, `O'Brien!A1:B2` → `O'Brien`, `''!A1:B2` → `undefined`.
  3) `web/src/lib/compareLists.test.ts` 의 기존 `splitSheetRange` 6사례가 **한 줄도 바뀌지 않고** 통과한다. 새 테스트는 고치기 전에 빨강이고, 그 한 자리만 되돌리면 정확히 그 테스트만 다시 빨강이 된다.
- 건드릴 파일 (프로덕션 1 + 테스트 1):
  - `web/src/lib/compareLists.ts:122-132` `splitSheetRange` — 시트 이름 한 줄을 `raw`/`quoted`/`sheetName` 세 줄로 나눠 **바깥 따옴표가 실제로 벗겨진 경우에만** `replaceAll("''","'")` 를 돌린다. `internal/formula/structure.go` 의 `quotedQualifier` 를 가리키는 한국어 주석을 함께 남긴다.
  - `web/src/lib/compareLists.test.ts` — `splitSheetRange` describe 에 이중 따옴표 사례 1 + 따옴표 없는 경계 사례를 더한다(경계 사례는 수정 전에도 PASS 여야 한다 — 따옴표 없는 경로를 건드리지 않았다는 증거).
- 권장 절차: 같은 변경이 커밋 `e5b0eb4` 에 그대로 있고, **그 부모가 정확히 지금 base 다**(`git rev-parse e5b0eb4^` = `b0481a99e31d476026a640e70969f70a8c7034c7` = `HEAD`, 이번 회차에 확인). 즉 충돌 없이 그대로 얹힌다. `git cherry-pick e5b0eb4` 로 가져오고, 저자가 `hkjang <gagagiga@naver.com>` 인지 `./scripts/check-commit-identities.sh HEAD` 로 확인하라. 충돌이 나면 포기하고 위 "건드릴 파일" 대로 직접 쓰면 된다 — 변경은 4줄이다. 커밋 메시지는 `e5b0eb4` 의 것을 재사용하라(`git show -s --format=%B e5b0eb4`).
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd web && npm ci` — 이 워크트리에 `web/node_modules` 가 없다. 시간이 걸리니 먼저 시작해 둘 것.
  - `cd web && npx vitest run compareLists` (빨강 1 → 초록 21 이 지난 회차 기록), `npm run lint`(= `tsc -b`), `npm test`(74 파일 523 테스트), `npm run build`
  - `go build ./...`, `go vet ./...`, `gofmt -l ./cmd ./internal ./pkg`(출력 없음), `go test ./...`
  - `./scripts/check-release-docs.sh`(`release docs ok: v0.265.0`), `./scripts/check-commit-identities.sh HEAD`
- 위험과 피할 것:
  - `replaceAll` 은 `web/tsconfig.app.json:3,5` 의 `target`/`lib` 이 둘 다 `ES2022` 라 쓸 수 있다(이번 회차에 파일로 확인, 지난 회차는 `tsc -b` 통과까지 확인했다). 굳이 `.split().join()` 으로 바꾸지 말 것.
  - Go 쪽 파서(`internal/formula/reference.go`·`structure.go`·`evaluator.go`)는 **손대지 말 것** — 거울은 통합하지 않고 각각 맞추는 것이 이 저장소 관례다. Go 쪽은 이미 `''` 를 푼다.
  - 같은 파일의 `compareKey`·`looksLikeIdentifier`·`parseFilterRange`, 그리고 `EditorPage.tsx`(읽기만), docs/PDF, `.github/workflows`, migrations, auth 는 건드리지 말 것.
  - 워크플로를 느슨하게 고쳐 통과시키지 말 것. 이번 실패는 테스트·CI 가 거절한 것이 아니라 PR 단계에서 세션이 끊긴 것이므로 `.github/workflows/*` 와 `scripts/release*.sh` 는 고칠 데가 없다(아래 "미확인" 참조).
  - **이번 회차의 핵심 위험은 또 PR 전에 끊기는 것이다.** 재분석에 시간을 쓰지 말고 cherry-pick → 검증 → 커밋 → PR 을 먼저 끝내라.
- 이번 회차에 확인한 것과 미확인:
  - 확인: main 의 `splitSheetRange` 는 아직 옛 한 줄이다(`compareLists.ts:130`). `compareLists.test.ts:70-88` 의 `splitSheetRange` describe 에는 이중 따옴표 사례가 없다(기존 6사례만). `e5b0eb4^ == HEAD`. `web/node_modules` 없음. 태그 `v0.258.0~v0.265.0` 과 `docs/releases/v*.md` 가 1:1 로 일치하므로 `scripts/release-notes.sh` 의 "release notes are required" 로 릴리즈가 죽을 상태는 아니다. `.github/workflows/release.yml` 은 `v*.*.*` 태그 push 에서만 돌고, 단계는 docker build → bundle → `release-notes.sh` → gh-release 뿐이다.
  - **미확인**: GitHub Actions 의 실제 실패 로그를 못 봤다 — 이 세션에서 `gh run list` 가 승인 거절로 막혔다. 따라서 "릴리즈 워크플로가 두 번 실패" 를 CI 로그로 교차검증하지 못했고, 위 진단(테스트/CI 결함이 아니라 `interrupted before PR`, 즉 PR 생성 전 세션 중단)은 저장소 상태(커밋이 브랜치에만 남고 main 에 없음)와 회차 노트의 `error: interrupted before PR` 에만 근거한다. 구현자가 `gh run list -L 10` 을 볼 수 있다면 **먼저 한 번만** 보고(`gh run watch` 는 API 한도를 소진하므로 쓰지 말 것), 실제로 `release` 워크플로가 같은 단계에서 두 번 깨진 흔적이 있으면 그 단계를 고치는 쪽으로 과제를 바꿔라 — 그때도 워크플로를 느슨하게 만들어 통과시키는 것은 금지다.
- 차선 후보: `compareKey` 가 `0x10`·`0b10` 을 숫자로 합쳐 서로 다른 식별자를 같은 키로 만든다 (가치 3 / 위험 3 / S) — 단, 대사 키 계약 결정이 먼저이고 `looksLikeIdentifier` 와 묶어 판단해야 하므로 1순위가 성립하지 않을 때만.
