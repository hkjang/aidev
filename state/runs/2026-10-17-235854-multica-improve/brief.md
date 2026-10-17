# 과제서 (정찰, 2026-10-18)

- 과제: 루트 `test` 스크립트가 전달 인자를 받지 못해 `pnpm test --silent` 가 turbo 사용법 오류로 죽는 것을 고친다 (가치 4 / 위험 2 / 작업량 S)

- 왜: 루트 `package.json` 의 `"test": "turbo test --filter=!@multica/mobile"` 에는 turbo 패스스루(`--`)가 없어, `pnpm test --silent` 처럼 스크립트 이름 뒤에 붙은 인자를 pnpm 이 그대로 turbo 에 넘기면 turbo 2.9.5 가 `ERROR unexpected argument '--silent' found` 로 즉시 종료한다(테스트는 한 개도 실행되지 않음, exit 1). 지난 회차 검증이 정확히 이 이유로 실패했고(아래 "실패 증거"), 사람이 `pnpm test --reporter=verbose` / `-t "<name>"` 같은 평범한 vitest 플래그를 붙여도 같은 사용법 덤프가 나온다.

## 실패 증거 (이미 확인됨 — 재조사 불필요)

이전 회차 `verify.txt` 115~161줄 그대로:

```
> multica@0.2.0 test /home/hkjang/.cache/auto-improve-wt/multica
> turbo test --filter=!@multica/mobile --silent

 ERROR  unexpected argument '--silent' found
  tip: to pass '--silent' as a value, use '-- --silent'
...
 ELIFECYCLE  Test failed. See above for more details.
```

검증 명령 목록(러너 자동 산출)은 `[ -d node_modules ] || pnpm install --frozen-lockfile` 과 `pnpm test --silent` 두 개뿐이고, 앞의 설치는 exit 0, 뒤가 exit 1(0초 — 테스트 자체는 시작도 못 함)이었다. turbo 버전은 `pnpm-lock.yaml:152-154` 에서 `specifier ^2.5.4 / version 2.9.5` 로 고정돼 있다(최근 변경 아님).

**이것은 테스트 내용의 실패가 아니라 인자 전달 결함이다. 어떤 테스트도 끄거나 느슨하게 하지 말 것.**

- 수용 기준:
  1) `pnpm test --silent` 가 exit 0 으로 끝나고, 모바일을 제외한 6개 워크스페이스(`@multica/core`, `@multica/views`, `@multica/web`, `@multica/desktop`, `@multica/docs`, `@multica/ui-lab`)의 `test` 태스크가 **실제로 실행된다**(turbo 요약의 `Tasks: N successful, N total` 로 확인; 0개 실행이나 전부 캐시 적중만으로 끝내지 말 것 — 필요하면 `--force` 로 한 번 더 확인).
  2) 인자 없는 `pnpm test` 가 기존과 똑같이 동작한다(빈 패스스루 `--` 가 turbo 에 거부되지 않는지 반드시 먼저 확인).
  3) 실패가 여전히 실패로 보인다: 아무 테스트 하나를 일부러 깨뜨린 상태에서 `pnpm test --silent` 가 exit 1 이 되고 실패한 테스트 이름이 출력에 남는다(vitest `--silent` 는 테스트 안의 console 출력만 죽이지 리포터·실패 요약은 죽이지 않는다). 확인 후 원복.
  4) 커밋에 테스트 스킵·필터 축소·`continue-on-error`·`|| true` 가 없다.

- 건드릴 파일:
  - `package.json`:`scripts.test` — `"turbo test --filter=!@multica/mobile"` → `"turbo test --filter=!@multica/mobile --"`. 뒤에 오는 인자가 각 패키지의 `test` 스크립트(`vitest run`)로 전달되어 `vitest run --silent` 가 된다.
  - (선택, 같은 파일) 왜 `--` 가 필요한지 한 줄 설명이 필요하면 `package.json` 에는 주석을 못 쓰므로 `AGENTS.md` 의 검증 표 각주 대신 **아무것도 쓰지 말 것** — 지금 문서가 적는 명령(`pnpm test`)은 변하지 않는다.
  - 건드릴 파일은 이 한 개가 정답이다. 두 개를 넘어가면 과제를 잘못 읽은 것이다.

- 검증 명령 (저장소 루트에서, 이 순서로 실제 실행하고 출력을 보고에 붙일 것):
  - `pnpm test`            ← 인자 없는 기존 경로가 깨지지 않았는지 (수용 기준 2)
  - `pnpm test --silent`   ← 실패 재현 명령 그대로. 이번엔 exit 0 (수용 기준 1)
  - `pnpm typecheck`, `pnpm lint` ← 루트 스크립트를 건드렸으니 나머지 두 검증도 한 번씩
  - Go/E2E 는 이 변경과 무관하므로 실행하지 않아도 되고, 보고에 "실행 안 함"이라고 적을 것.

- 위험과 피할 것:
  - **빈 패스스루 확인이 1순위.** `turbo test --filter=!@multica/mobile --` 뒤에 인자가 하나도 없을 때 turbo 2.9.5 가 이를 받아들이는지 가장 먼저 확인하라(정찰은 이 환경에서 pnpm 실행 권한이 없어 **미확인**). 거부한다면 이 접근을 버리고 차선 후보로 넘어가라 — 억지로 래퍼 스크립트(`scripts/*.mjs`)를 새로 만들어 파일 수를 늘리지 말 것.
  - **`typecheck` / `lint` 에는 같은 `--` 를 달지 말 것.** `tsc --silent`(TS5023)와 eslint 모두 `--silent` 를 모르므로 패스스루를 열어도 실패 지점이 turbo 에서 도구로 옮겨갈 뿐 아무것도 고쳐지지 않는다. `build` 등 나머지 스크립트도 그대로 둔다.
  - `turbo.json` 을 건드리지 말 것. `test` 태스크의 `dependsOn: ["^cache-inputs"]` 는 캐시 해시 정합성을 위한 의도된 설계이며(파일 내 주석에 이유가 적혀 있다) 이번 결함과 무관하다.
  - `.github/workflows/ci.yml` 은 `--silent` 를 쓰지 않는다(grep 확인). CI 를 고칠 필요가 없고, 워크플로를 느슨하게 만드는 수정은 금지다.
  - 각 패키지의 `test` 는 모두 `vitest run` 이다(`packages/core`, `packages/views`, `apps/web`, `apps/desktop`, `apps/docs`, `apps/ui-lab`). `apps/mobile` 만 `vitest run && bash scripts/ios-run.test.sh` 인데 `--filter=!@multica/mobile` 로 제외되므로 영향 없음.
  - 과거 교훈: **소스 문자열 검사를 증거로 삼지 말 것.** `package.json` 에 `--` 가 들어 있는지 확인하는 테스트를 새로 추가하지 말고, 위 두 명령의 실제 실행 출력을 증거로 제출하라.

- 차선 후보: **백엔드 요금표 Claude 4.x 규칙에 `claudeVersionEnd` 종결자 추가** — `server/internal/metrics/pricing.go` 의 Opus 4.8/4.7/4.6/4.5, Sonnet 4.6/4.5, Haiku 4.5 규칙만 `claudeVersionEnd` 종결자가 없어, 종결자 없는 부분 문자열 매칭이 후속 SKU(예: `claude-opus-4-5-1`)를 조용히 같은 등급으로 삼킨다. 같은 파일 상수 주석이 바로 그 위험을 설명하고 있다(`claude-fable-5` ↔ `5-1` 사례). 현재 4.x 단가가 동일해 금액 회귀 테스트로는 실패를 못 보이므로, 규칙 매칭 수준의 테이블 테스트(가짜 미래 SKU 문자열이 매칭되지 않을 것)를 쓸 것. 검증: `(cd server && go test ./internal/metrics -count=1)`.
