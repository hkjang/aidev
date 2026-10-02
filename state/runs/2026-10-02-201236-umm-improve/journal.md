# 회차 노트 2026-10-02-201236-umm-improve — umm
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 20:12] base pinned — main@715fd97
- [러너 20:12] autonomy release — 

## 구현 노트
- 고친 것: `web/package.json` 에 `engines.node ">=22.22.2"` 선언 + 새 `web/scripts/run-on-supported-node.mjs` 로 `test`·`test:watch`·`test:offline-queue` 를 하한을 넘는 Node 에서 돌림. npm 이 lifecycle PATH 에 조상 `node_modules/.bin` 을 루트까지 붙여 `/home/hkjang/node_modules` 의 `node@20.19.2` 가 vitest 를 Node 20 으로 띄우고, jsdom 30→undici 8 이 `markAsUncloneable`(Node 22.10+) 없이 죽던 것이 verify 실패의 원인. 계측으로 증명함(원본 복원).
- 확신 없는 곳: 래퍼가 `spawnSync(stdio:'inherit')` 라 `test:watch` 의 대화형 키 입력·Ctrl-C 는 돌려 보지 않았다(`test`·`test:offline-queue` 만 실제 실행). Windows(cmd) 에서의 동작도 미확인 — 다만 셸 문법을 안 썼고 경로는 전부 `node:path` 로 만든다.
- 일부러 안 한 것: `build`·`typecheck`·`lint` 는 래퍼에 넣지 않았다 — Node 20 에서도 PASS 했고, 릴리즈 경로(Dockerfile `npm ci` + `npm run build`)를 건드리지 않기 위해서. `.github/workflows/*`·Dockerfile 도 손대지 않았다(이미 node 24 로 하한을 넘음). `/home/hkjang/node_modules` 의 `node@20` 패키지는 저장소 밖 사용자 환경이라 지우지 않았다.
- 느슨해지지 않았다는 증거: 일부러 실패하는 시험 파일로 `npm test` EXIT=1(`Tests 1 failed | 186 passed`), 자식 SIGKILL EXIT=1, `engines.node` 가 `>=x.y.z` 모양을 벗어나면 가드가 꺼지지 않고 거절. 지원 Node 에서는 메시지 없이 18 파일 통과.
- 다음 역할 주의: Go DB 통합 시험은 `umm-test-pg` 가 안 떠 있어 SKIP 상태로만 돌렸다(Go 파일은 한 줄도 안 바꿨다). `npm test` 가 지금 한 줄 진단 메시지를 stderr 로 찍는데, 이는 PATH 가 하한 미달일 때만이며 CI 에서는 안 나온다.
- [러너 20:38] verify failed — 실패한 검증: npm run lint   # CI에서 가져옴 (exit 1)
