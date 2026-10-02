# PR 처리기 노트 2026-10-02-154651-postra-shepherd — postra PR #31
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-02-141219-postra-improve)
# 회차 노트 2026-10-02-141219-postra-improve — postra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:12] base pinned — main@1d382aa
- [러너 14:12] autonomy release — 

## 정찰 노트
- 우선 과제(verify-failed 수정)가 배정돼 새 아이디어는 고르지 않았다. 다른 후보(POP3 retrBody 상한 등)는 프런트 검증이 막힌 동안 어차피 verify 를 통과하지 못하므로 전부 뒤로 미뤘다.
- **재현하지 못했다**: 이 세션에서 `npm ci`/`npm` 실행이 권한 거부됐고 워크트리에 `web/node_modules` 가 없다. 어느 스펙이 실패하는지는 **미확인**이며, 과제서의 "실패 후보 순위" 는 가설이다. 구현자는 재현 출력을 가설보다 우선하라.
- 확인한 사실만: `release.yml` 은 `npm test` 를 돌리지 않는다(ci.yml frontend 잡과 `Makefile: frontend-test` 만 돌린다) / `frontend-test` 는 의존성 설치를 하지 않는다 / `test-setup.ts` 는 jest-dom import 한 줄뿐이라 전역 시각·TZ 고정이 없다 / 날짜 의존 후보 6개는 모두 시점 주입·`setSystemTime` 으로 안전했다 / 최근 프런트 변경은 d97577a(v0.25.0) 이고 1d382aa 는 버전 범프뿐.
- 구현자가 조심할 것: 워크플로·`npm test` 스크립트·`test.exclude`·스펙 skip 으로 통과시키는 것은 반려 사유다. 설치 누락이 원인으로 드러나면 YAML 이 아니라 `Makefile: frontend-test` 를 고치고, "프로덕션 결함이 아니었다" 를 정직하게 적어라.
- 프로필은 0일 전 기록이고 이번에 읽은 내용과 어긋나지 않아 다시 쓰지 않았다(web/npm 명령도 이미 적혀 있다).
- [러너 14:17] scout done — `cd web && npm test` 실패를 재현해 원인 지점을 고친다 (수정 과제 / 가치 5 / 위험 2 / 작업량 M)

## 구현 노트
- `cd web && npm test` 의 exit 1 은 스펙 결함이 아니었다: npm 이 상위 모든 `node_modules/.bin` 을 PATH 앞에 붙여 `/home/hkjang/node_modules/node`(v20.19.2)가 셸의 v22.23.1 을 가렸고, jsdom 30 의 undici 가 `worker_threads.markAsUncloneable`(Node 22.10+) 부재로 스펙마다 죽었다. 새 `web/scripts/engine-node.mjs` 가 `engines.node` 를 검사하고 `npm_node_execpath` 로 재실행한다. 파일 추가가 Tailwind v4 자동 탐지를 건드려 자산 드리프트 33건을 만든 두 번째 결함은 `src/styles.css` 의 `@source not "../scripts"` 로 닫았다(재빌드 산출물이 커밋된 자산과 바이트 동일).
- **확신 없는 곳·검증 못 한 것**: (1) "Node 20 에서 깨진다" 를 이식 가능한 자동 테스트로 묶지 못했다 — CI 에 Node 20 바이너리가 없다. 재현은 명령 출력이고 스펙은 실제 버전 문자열로 `planInterpreter` 분기만 묶는다. (2) `npm_node_execpath` 는 npm 이 설정하므로 yarn/pnpm 이나 vitest 직접 실행에는 없다 — 그때는 재실행 없이 한 줄로 실패한다(의도). (3) `parseMinimumVersion` 은 이 저장소가 쓰는 `>=X.Y` 모양만 읽고 그 밖의 range 는 검사 없이 통과시킨다. (4) Windows 에서 래퍼를 실행해 보지 못했다(`spawnSync`+절대 경로라 문제없을 것으로 보지만 미검증). (5) `web/scripts` 외의 경로(tests/e2e, 루트 설정 파일)도 Tailwind 탐지 대상인지는 확인하지 않았다.
- **일부러 하지 않은 것**: `typecheck`·`build` 스크립트는 감싸지 않았다 — 둘 다 Node 20 에서도 exit 0 이고 자산 드리프트도 없었으며, 릴리즈 경로(`release.yml` 은 `npm ci`→`npm run build`)를 건드리는 위험이 이득보다 크다. `Makefile: frontend-test` 의 설치 누락도 이번 원인이 아니라 묶지 않고 보류 아이디어로 남겼다. 워크플로 YAML·`test.exclude`·스펙 skip 은 일절 손대지 않았다.
- **다음 역할이 조심할 것**: 이 박스에서는 `npm`/`npx` 로 무엇을 돌려도 Node 20 이 선택되므로, vitest 를 직접 부를 때는 `/home/hkjang/.nvm/versions/node/v22.23.1/bin/node node_modules/vitest/vitest.mjs run` 처럼 인터프리터를 명시하라. `npm test` 는 이제 stderr 에 `engine-node: ... retrying with ...` 한 줄을 찍고 통과하는데, 이는 정상이며 CI(Node 22)에서는 출력되지 않는다. 자산 드리프트를 볼 때는 `web/` 아래 어떤 파일을 추가했는지도 의심하라 — Tailwind 가 주석을 읽는다. PostgreSQL(`POSTRA_TEST_PG`)·브라우저 e2e 는 이번에 돌리지 않았다.
- [러너 14:33] brief accepted — 채택 — 과제서가 "미확인" 으로 남긴 실패 스펙 확정을 `npm ci` 후 재현으로 끝냈고, 결론은 과제서의 가설 1(스펙이 실제�
- [러너 14:34] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인: Node 20 그림자 아래 vitest 직접 실행으로 ledger-entry.md:5 의 재현 출력을 독립 재현(undici markAsUncloneable)했고, 수정 후 `npm test` 50파일/426테스트 통과·새 스펙 12테스트가 vitest include 에 실제로 걸리는 것을 확인했다. `test.exclude`·skip·워크플로 YAML 은 손대지 않았다.
- 확인: `@source not "../scripts"` 를 프로브 파일(`mt-[13px] rotate-[7deg]` 주석)로 시험해 유틸리티가 생성되지 않음을 증명했고, 프로브 제거 후 재빌드로 자산 드리프트 0·워크트리 clean 을 확인했다. 구현자가 의심한 (1)~(5) 중 (5)는 scripts/ 밖은 여전히 탐지 대상이라는 뜻으로 남는다.
- 못 본 것: Windows 에서의 래퍼 실행, PostgreSQL(POSTRA_TEST_PG)·브라우저 e2e·go test(이 변경은 Go 를 건드리지 않음), CI 실제 러너에서의 실행.
- 승인 후에도 남는 우려: `web/package.json` 이 `node_modules/vitest/vitest.mjs` 를 직접 가리키므로 vitest 메이저 범프 때 같이 봐야 한다 / `typecheck`·`build` 는 래퍼 밖(지금은 Node 20 에서도 exit 0 실측) / web/ 아래 어디든 파일을 추가하면 `make frontend-check` 필수.
- 릴리즈 노트: 프로덕션 결함이 아니라 개발자 도구 수정으로 적을 것. 보안·법무 차단 사유 없음(인증·비밀값·개인정보·새 의존성 변경 없음).
- [러너 14:39] review approved — 리뷰 승인 (risk=low)
- [러너 14:39] pr created — https://github.com/hkjang/postra/pull/31
- [러너 14:44] ci failed — 성공이 아닌 검사: govulncheck=failure · 실패한 검사: ? 잡: govulncheck 

## 수리 노트
- 지적은 맞았다: `govulncheck ./...` 를 로컬 govulncheck@v1.6.0 으로 돌려 CI 와 같은 GO-2026-6505(exit 3)를 재현했다. 다만 이것은 PR #31 의 web/ 변경이 만든 결함이 아니다 — go.mod 는 이 PR 이 건드리지 않았고, 권고가 뒤늦게 공개돼 origin/main 도 같이 실패한다.
- 고친 방법: otel 계열을 v1.45.0(수정 버전)으로 올리고 `go mod tidy`. 새 커밋 7ae6372 는 go.mod/go.sum 만 바꾼다. 테스트·워크플로·검증 명령·`toolchain` 은 일절 손대지 않았다.
- 검증: govulncheck exit 0(v1.44.0 으로 되돌리면 exit 3 재현) · build · vet · `go test -race -count=1 ./...` 전부 통과 · contracts -check · `make lint` gosec 0 issues.
- 확신 없는 곳: 남은 "import 1개·require 4개" 미호출 취약점은 v1.44.0 때와 같은 숫자로 그대로이며 CI 는 이것으로 실패하지 않는다. 전이로 올라간 proto/otlp 1.11.0·genproto 는 빌드·테스트로만 확인했고 OTLP 를 실제 collector 로 내보내 보지는 않았다. PG·브라우저 e2e·npm 은 돌리지 않았다(이 커밋은 Go 소스와 web/ 를 바꾸지 않음).

## 심사 노트
- 독립 재현·확인: shadow `/home/hkjang/node_modules/node/bin/node`(v20.19.2)로 vitest 직접 실행 → undici `markAsUncloneable` 로 "no tests, 3 errors" 재현, `npm test` 는 래퍼가 v22.23.1 로 재실행해 50파일/426테스트 통과하고 새 스펙 12테스트가 실제로 걸린다(verbose 로 이름 확인). `npm ci`→typecheck→`npm run build` 후 `internal/transport/spa/assets` 드리프트 0·워크트리 clean 이라 `@source not "../scripts"` 가 브라우저로 가는 CSS 를 이전 커밋과 바이트 동일하게 유지함을 확인(= scripts/ 제외로 UI 유틸리티가 사라지지 않음).
- Go 쪽 전부 재실행: `go mod tidy -diff` clean · `go mod verify` · build · vet · `go test -race -count=1 ./...` 통과 · contracts `-check` · `make lint` gosec 0 · govulncheck@v1.6.0 **exit 0**("0 vulnerabilities", 미호출 import 1·require 4 잔존) · 릴리즈와 같은 `CGO_ENABLED=0 -trimpath` 정적 빌드 성공. `release.yml` 은 `npm test` 를 돌리지 않아(`npm ci`→`npm run build` 뿐) 래퍼가 릴리즈 경로에 닿지 않는 것도 확인했다.
- 보호 파일·차단 소견 없음: auth/oidc/session/migrations/.github/workflows 미변경, spa/assets 미변경, 새 의존성 없음(otel 1.44→1.45 동일 Apache-2.0), 새 공개 경로·권한 확대·개인정보 처리·비밀값 경로 없음. `spawnSync` 는 셸 없이 배열 인수만 쓴다.
- 못 본 것: Windows 에서의 래퍼 실행, PG(`POSTRA_TEST_PG`)·브라우저 e2e(Chromium), 7ae6372 에 대한 실제 GitHub CI 실행(이 세션에 GH 권한 없음 — 같은 명령을 로컬에서 대신 돌렸다), origin/main 도 govulncheck 로 실패하는지의 반대 검증(이 PR 은 Go 소스를 추가하지 않으므로 취약점을 들여올 수 없다).
- 권고 근거(approve/merge, risk=low): 재현된 개발자 도구 결함을 스펙 skip·exclude·워크플로 수정 없이 고쳤고, 프로덕션으로 나가는 산출물(Go 바이너리·CSS·JS 자산)이 바이트 수준으로 변하지 않았다. 남는 것은 전부 note 급 — `package.json` 이 `node_modules/vitest/vitest.mjs` 를 직접 가리켜 vitest 메이저 범프 때 같이 봐야 하고, `parseMinimumVersion` 은 `>=X.Y` 외 range 를 무검사 통과시키며(의도·테스트됨), 7ae6372 는 본문 없는 커밋이라 저장소 관례와 어긋난다.
