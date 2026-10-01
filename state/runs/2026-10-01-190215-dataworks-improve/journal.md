# 회차 노트 2026-10-01-190215-dataworks-improve — dataworks
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:02] base pinned — main@98c5081
- [러너 19:02] autonomy release — 

## 정찰 노트
- 과제는 자동 배정된 수정 과제(`cd web && npm test --silent` 실패)라 후보 선택은 없었다. 대신 원인 범위를 소거로 좁혔다: 설치 없는 상태의 실패는 **exit 127**(`vitest: not found`, 실행 확인)이고 러너 보고는 **exit 1** → vitest 가 돌고 테스트가 실제로 실패했다. package-lock 루트는 package.json 과 글자까지 동기이고 `@rolldown/binding-*` 16종이 lock 에 있어 설치·플랫폼 바이너리 문제도 소거했다.
- web 소스는 2026-09-29(9파일/27테스트 통과) 이후 릴리즈 버전 범프만 바뀌었고, 9개 테스트 파일 중 8개를 전부 읽어 시계·네트워크·FS 의존이 없음을 확인했다. 남은 `src/test/keep-dist-placeholder.test.ts` 가 유일하게 실제 `vite build`(Vite 8/Rolldown) + `os.tmpdir()` 에 의존한다 → 1순위 가설.
- **확신 없는 곳(추측)**: 그 1순위 가설을 실행으로 재현하지 못했다 — 이 샌드박스에서 `npm ci`·`npm install`·`node -e` 가 모두 승인 거부됐다. 어떤 사례가 어떤 메시지로 실패하는지는 **미확인**이며, 과제서 1단계를 "재현 먼저" 로 둔 이유다.
- 구현자가 조심할 것: `logLevel: 'silent'` 가 vite 오류를 삼키므로 진단 중에만 올리고 되돌릴 것. `emptyOutDir`·`include`·`it.skip`·CI 완화는 전부 금지(릴리즈·빌드 경로이고 운영자 규칙이 명시적이다). `npm run build` 뒤 `git status --short` 깨끗함과 `go build ./...` 까지 확인해야 릴리즈를 다시 깨지 않는다.
- 로컬 Node 는 22.23.1, CI 는 24 다. 버전 차이로 재현이 안 되면 그 사실을 원장에 적고 버전 비의존 수정을 택할 것(과제서 차선 후보).
- [러너 19:09] scout done — 릴리즈 검증 `cd web && npm test --silent` 실패 복구 — 실제 `vite build` 를 돌리는 `keep-dist-placeholder` 테스트 경로 

## 구현 노트
- 과제서의 1순위 가설은 **반증**됐다. `keep-dist-placeholder.test.ts` 는 `// @vitest-environment node` 라서 jsdom 을 로드하지 않고, 실패 출력의 "1 passed (1) / 2 passed (2)" 가 바로 그 파일이었다. 진짜 원인: npm 이 run-script PATH 앞에 상위 디렉터리의 `node_modules/.bin` 을 전부 붙이고, `/home/hkjang/node_modules/.bin/node`(Node 20.19.2, Sep 30 22:24 생성)가 nvm 22.23.1 을 가려 vitest 가 Node 20 에서 돌았다 → `worker_threads.markAsUncloneable`(22.10+) 없음 → jsdom 30 이 끌어오는 undici 8.10.0 이 로드 중 사망 → jsdom 환경 8파일 기동 실패. 고친 것: `web/scripts/run-with-supported-node.mjs` 가 `npm_node_execpath` 로 vitest 를 띄우고, 하한(`engines.node >=22.19.0`) 미달이면 후보 목록과 함께 멈춘다.
- **확신 없는 곳**: (1) `npm test` 호출부는 `scripts/`·`.github/`·`Dockerfile` 전체 검색으로 `.github/workflows/ci.yml:64`(무변경, Node 24) 한 곳뿐임을 확인했다 — `release.sh`·`golden-regression.sh` 는 npm 을 호출하지 않는다(해소됨). (2) 런처는 Windows 에서도 동작하도록 셸 없이 인터프리터를 직접 spawn 하지만 **Windows 에서 실행해 보지 않았다**. (3) `npm_node_execpath` 가 npm 의 공식 계약인지 문서로 확인하지 않고 이 환경에서 값이 올바름(22.23.1)만 실행으로 확인했다 — 미설정 시 `process.execPath` 폴백이 있어 깨지지는 않는다.
- 일부러 하지 않은 것: `npm run build`·`npm run lint` 는 같은 런처를 거치게 하지 않았다 — 둘은 Node 20.19.2 에서도 통과하고(실행 확인), 산출물을 만드는 build 를 같은 변경에서 더 건드리면 릴리즈 위험이 커진다. ideas.json 에 후보로 남겼다. `keep-dist-placeholder.test.ts` 의 `logLevel:'silent'` 도 범인이 아니어서 손대지 않았다(후보 유지). `/home/hkjang/node_modules/node` 삭제 같은 되돌릴 수 없는 환경 변경은 하지 않았다.
- 다음 역할이 조심할 것: 새 테스트 `web/src/test/test-script-interpreter.test.ts` 는 **PATH 앞에 `node` 라는 이름의 sh probe 를 끼워 package.json 의 `test` 스크립트를 자식 프로세스로 다시 돌린다** — POSIX 셸이 필요하고(CI 는 ubuntu), `npm ci` 가 끝나 `node_modules/.bin` 이 있어야 하며, 중첩 실행이 자신을 다시 집지 않도록 `src/lib/labels.ko.test.ts` 로 필터를 건다. 타임아웃은 120s 로 올려 뒀다. 이 테스트는 "러너가 PATH 의 node 로 기동되지 않는다" 를 단정하므로, `test` 스크립트를 다시 `vitest run` 으로 되돌리면 즉시 빨개진다(의도된 가드).
- [러너 19:24] brief fallback — 차선 — 과제서의 1순위 가설(`keep-dist-placeholder` 의 실제 `vite build` 가 실패한다)은 실행으로 **반증**됐다. 그 파일은 jsdom �
- [러너 19:24] verify failed — 실패한 검증: npm run lint   # CI에서 가져옴 (exit 1)
