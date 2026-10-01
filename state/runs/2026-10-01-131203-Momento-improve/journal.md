# 회차 노트 2026-10-01-131203-Momento-improve — Momento
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 13:12] base pinned — main@12ba629
- [러너 13:12] autonomy release — 

## 구현 노트
- 수정 과제. `cd web && npm test --silent` 실패의 원인은 앞 회차가 적은 '병렬 자식의 타입 스트리핑' 이 아니라 npm lifecycle PATH 가림이었다 — npm 은 상위 디렉터리의 `node_modules/.bin` 을 전부 PATH 앞에 붙이고, 이 환경 홈에 깔린 `~/node_modules/.bin/node`(v20.19.2)가 npm 자신의 v22.23.1 을 가려 `.ts` import 가 30개 파일에서 죽었다. 증거는 자식 출력의 `# Node.js v20.19.2`. web/package.json 의 test 를 `"${npm_node_execpath:-node}" --test …` 로 바꾸고 engines(`^22.18 || >=24`)를 적었다.
- 확신 없는 곳: (1) `${...:-...}` 는 POSIX 셸 문법이라 **Windows cmd 에서 npm run test 가 깨진다**. 저장소에 Windows 경로가 안 보여(Makefile·Docker·ubuntu CI) 감수했고, `.npmrc` 에 script-shell 을 넣는 쪽은 전역 영향이 있어 피했다. (2) GitHub CI 의 Node 24 에서는 못 돌렸다 — 대신 nvm v22.23.1/npm 10.9.8 과 `/usr/bin/npm` 11.12.1(Node v25.9.0) 두 조합, 그리고 `npm_node_execpath` 미설정 시 `node` 되돌림 경로까지 각각 178/178 로 확인했다. (3) testCommand.test.mjs 의 대조 단언은 `--no-experimental-strip-types` 플래그 이름에 의존한다 — 미래에 이름이 바뀌면 조용히 통과하지 않고 그 테스트가 떨어진다(실패하는 쪽으로 깨지므로 안전하다고 봤다).
- 일부러 안 한 것: `.github/workflows` 와 `node --test`·단언은 건드리지 않았다(느슨하게 만들기 금지). `sdk/package.json` 의 같은 `node --test` 는 그대로 뒀다 — sdk 테스트는 `.ts` 를 import 하지 않아 가림에 영향받지 않고, 재현된 실패가 없다. web 의 lint·build 도 `node_modules/.bin` shebang 으로 PATH 의 node 를 타지만 이번 가림(v20.19.2)에서 실제로 통과했으므로 재현 없이 손대지 않고 ideas.json 에 남겼다.
- 다음 역할이 조심할 것: 새 테스트는 자식 프로세스를 띄우고 `NODE_TEST_CONTEXT` 를 지운다 — 지우지 않으면 안쪽 `node --test` 가 아무것도 돌리지 않고 0 으로 끝나 단언이 공허해진다(실제로 처음 그렇게 썼다가 대조 단언이 걸러냈다). POSIX 셸(`sh`)과 쓰기 가능한 tmpdir 이 필요하다. Go 테스트와 Postgres 통합 테스트는 이번에 돌리지 않았다.
- [러너 13:20] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인함: 변경은 2파일뿐이고 테스트가 변경을 진짜 검증한다 — 대조 단언(testCommand.test.mjs:72)이 가림 PATH 에서 맨 `node` 의 ERR_UNKNOWN_FILE_EXTENSION 을 먼저 증명하므로 수정 전 코드에서는 test 2 가 반드시 떨어진다. 직접 돌림: web 178/178, lint, build(3.51s), `npm ci --dry-run`, git status 깨끗. `npm_node_execpath` 주입은 /tmp 별도 패키지로 npm 10.9.8·11.12.1 양쪽에서 확인, `--no-experimental-strip-types` 는 Node v25.9.0 에서 확인.
- 못 봄: CI 의 Node 24 실행, Go 테스트·Postgres 통합 테스트(DSN 없음), Windows 경로.
- 승인이어도 남는 우려 (릴리즈 노트·다음 회차): (1) `engines` 가 package.json 에만 있고 package-lock.json 루트에 없다 — `npm install` 이 lock 에 한 줄을 추가하므로 README 대로 설치한 개발자에게 stray diff 가 생긴다(`npm ci` 는 영향 없음, 차단 아님). 다음 회차에서 lock 을 함께 커밋할 것. (2) `${npm_node_execpath:-node}` 는 POSIX 전용 → Windows cmd 의 `npm run test` 가 깨진다. (3) 이 수정은 npm 이 스크립트 PATH 에 node_modules/.bin 을 덧붙여 생기는 가림만 고친다 — 가려진 node 가 npm 실행 시점의 대화형 PATH 에 먼저 있으면 npm_node_execpath 자체가 오염되어 듣지 않는다. (4) testCommand.test.mjs 가 /tmp 에 shim 디렉터리를 회당 2개 남긴다(정리 없음).
- 보안·법무 소견: 차단 없음. 인증·권한·개인정보·비밀값·외부 의존성에 닿지 않고, 새 테스트의 shim 은 mkdtemp 안에서만 산다.
- [러너 13:24] review approved — 리뷰 승인 (risk=low)
- [러너 13:25] pr created — https://github.com/hkjang/Momento/pull/21
- [러너 13:31] ci passed — 검사 1개 모두 success
- [러너 13:31] merge done — dda0d3e

## 릴리즈 노트
- v0.34.53 로 패치를 올렸다(최근 릴리즈가 전부 패치 증가, 이번 변경은 도구 수정 하나). 이전 릴리즈와 같은 10파일 모양 — `internal/version/version.go`, `web`·`sdk` 의 package.json/package-lock.json, `compose.yml`, `README.md`, `docs/OFFLINE.md`, `CHANGELOG.md`, 새 `.github/release-notes/v0.34.53.md`. 커밋 `release: v0.34.53`(본문 영어, 트레일러 없음), 경량 태그 `v0.34.53` — 둘 다 이전 관례 그대로. 푸시는 하지 않았다.
- 자산은 빈 배열이다. `.github/workflows/release.yml` 이 `v*` 태그 푸시로 이미지를 빌드하고 `momento-v<version>.tar.gz`·`.sha256` 을 만들어 smoke test 까지 한 뒤 Release 를 생성한다 — 사람이 올리는 자산이 없다.
- 검증(이전 릴리즈가 밟던 `make test` 전부 + CI 게이트): `go test ./cmd/... ./internal/...` 통과, `go vet` 통과, `TestDeclaredVersionsAgree`·`TestVersionIsAReleaseVersion`·`TestAGuideIsAtLeastAsRecentAsWhatItDescribes` 포함 `internal/version` 6건 통과(버전 일치 검사가 여기 있다). web: `npm ci`/`npm audit`(0 vulnerabilities)/`lint`/`test` **178/178**/`build` ✓ 5.97s — 이번 수정 덕에 `npm test` 가 가림 PATH 에서도 그대로 돈다. sdk: `npm ci`/`npm audit`(0)/`typecheck`/`build` 통과, 테스트 **27/27**.
- **다음 회차가 반드시 볼 것 — `sdk/package.json` 에 같은 결함이 그대로 있다.** 구현 노트는 "sdk 테스트는 `.ts` 를 import 하지 않아 가림에 영향받지 않는다" 고 적었는데 **사실이 아니다**: `sdk/test/*.test.mjs` 는 `src/index.ts` 를 import 하고, 이 기계에서 `cd sdk && npm test` 는 2개 파일 **둘 다** `ERR_UNKNOWN_FILE_EXTENSION ".ts"` 로 떨어진다(출력에 `# Node.js v20.19.2` — web 에서 증명된 바로 그 가림). 즉 `make test` 는 이 기계에서 여전히 sdk 단계에서 멈춘다. 같은 한 줄(`"${npm_node_execpath:-node}" --test test/*.test.mjs`)과 `engines` 를 sdk 에도 넣으면 끝난다 — 재현이 있으므로 이번엔 "재현 없음" 으로 미룰 수 없다. 릴리즈 세션이라 코드는 건드리지 않았고, 대신 npm 바깥의 올바른 인터프리터(v22.23.1)로 돌려 트리 자체는 멀쩡함을 확인했다(27/27). CI(Node 24, 상위 경로에 stray node 없음)는 영향받지 않으므로 릴리즈를 막을 사유는 아니라고 판단했다.
- 비평 노트가 남긴 우려 중 하나는 해소됐다 — PR #21 의 CI(Node 24)가 success 로 끝났으므로 "CI 의 Node 24 에서 못 돌렸다" 는 더 이상 열린 항목이 아니다. `engines` 가 package-lock.json 루트에 없는 건 그대로 남겼다(`npm ci` 는 영향 없음을 이번에 두 번 확인했고, 릴리즈가 고칠 범위가 아니다).
- [러너 13:46] release published — v0.34.53
- [러너 13:48] assets verified — v0.34.53 자산 2개 (이전 v0.34.48: 2)
