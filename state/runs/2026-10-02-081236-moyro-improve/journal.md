# 회차 노트 2026-10-02-081236-moyro-improve — moyro
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 08:12] base pinned — main@0439d5e
- [러너 08:12] autonomy release — 
- [러너 08:17] scout done — webapp 테스트 전체를 죽이는 jsdom 30.0.1 + undici 8.10.0 조합을 동작하는 버전 쌍으로 고정 (가치 5 / 위험 2 / 작�

## 구현 노트
- 바꾼 것: `webapp/scripts/run-vitest.mjs`(신규)가 vitest 를 `npm_node_execpath` 로 띄우고 `package.json` 의 `test` 가 경유 + 잠금파일 undici 8.10.0→8.11.2. 원인은 과제서가 적은 버전 조합 결함이 아니라, npm 이 상위 모든 디렉터리의 `node_modules/.bin` 을 PATH 앞에 붙여(`@npmcli/run-script/lib/set-path.js:22-28`) `/home/hkjang/node_modules/.bin/node`(Node 20.19.2)가 nvm v22.23.1 을 선점한 것이다. jsdom 30 engines 는 `^22.22.2` 라 engines 미충족이었고, 코드 변경 0으로 Node 22 에서는 그 트리가 그대로 통과한다(양방향 확인).
- **확신 없는 곳·검증 못 한 것**: ① CI 에서 실제로 돌려 보지 못했다(`gh` 미인증). CI 러너의 `$HOME` 에는 `node_modules/node` 가 없을 것이므로 **CI 의 `npm test` 는 base 에서도 통과했을 가능성이 높다** — 과제서의 "CI 도 같은 이유로 깨진다" 는 주장은 내가 확인하지 못했고 아마 틀렸다. 반면 `npm audit --audit-level=high` 는 잠금파일이 base 와 바이트 동일한 상태에서 깨졌으니 CI/릴리즈에서도 깨졌을 것이다. 즉 **릴리즈를 실제로 막고 있던 것은 audit 쪽일 수 있다.** ② Windows 에서 돌려 보지 못했다(`node scripts/run-vitest.mjs` 는 cmd/PowerShell 모두 동작해야 하지만 미검증). ③ `npm_node_execpath` 가 비어 있는 경로는 `process.execPath` 폴백을 직접 실행해 확인했지만, npm 이 그 변수를 주지 않는 npm 버전은 확인하지 않았다(npm 10.9.8 에서만 확인).
- 일부러 하지 않은 것: 과제서의 1·2순위(undici/jsdom 을 내려 핀)는 **기각**했다 — 지원되지 않는 Node 를 덮는 것이 되고, 인터프리터가 올바르면 원래 버전 그대로 통과한다. `/home/hkjang/node_modules/node` 삭제 등 **환경 변경은 하지 않았다**(되돌릴 수 없는 전역 변경 금지, 환경과 코드를 같은 단계에서 바꾸지 않음). tsconfig·`--maxWorkers`·pool·skip·워크플로 파일은 건드리지 않았다. 타입 범위도 넓히지 않았다 — `src` 테스트에 node 타입이 없어 `process` 는 `globalThis` 캐스트로 읽는다.
- 다음 역할이 조심할 것: 새 `src/node-runtime.test.ts` 는 DB 가 필요 없고 jsdom 환경도 아니다(그래서 Node 20 에서도 **실행되어 실패**한다 — 그게 의도다). 런처는 exit code/signal 을 전달하므로 게이트가 느슨해지지 않았다(일부러 실패하는 테스트로 exit 1 전달 확인). `npm audit` 는 실시간 DB 를 조회하므로 **다음 회차에 코드 변경 없이 다시 깨질 수 있다**; 그때도 저장소 결함이 아니다. 서버 Go 코드는 전혀 건드리지 않았다(`go build`/`go vet` 만 돌렸고 `go test ./...` 는 이번 회차 미실행 — webapp 전용 변경이라 생략했다, 이것도 검증 공백으로 적어 둔다).
- [러너 08:31] brief fallback — 차선 — 과제가 가리킨 증상(`npm test` 가 jsdom 테스트 35개에서 `markAsUncloneable` 로 죽는 것)과 수용 기준 1~6 은 그대로 채택�
- [러너 08:32] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- 판정 approve / risk low / blocking 없음. RED→GREEN 을 양방향 직접 재현했다: 섀도된 `~/node_modules/.bin/node`(v20.19.2)로는 `src/node-runtime.test.ts:35` 가 "expected 20 to be greater than or equal to 22" 로 실패하고 jsdom 테스트는 `jsdom/lib/api.js:12` 에서 죽는다. 런처 경유 49파일/224테스트 통과, typecheck·build·check-source-sizes 통과, `npm audit --audit-level=high` 는 dev 포함/미포함 모두 0건.
- 게이트가 느슨해지지 않았음을 파이프 없이 종료코드로 확인했다 — `npm_node_execpath` 를 Node 20 으로 강제하면 런처가 그것을 쓰고 exit 1 을 전달한다. undici 8.11.2 의 integrity 를 레지스트리 값과 비교해 일치시켰으니 CI 의 `npm ci` 가 손으로 쓴 해시로 깨질 위험은 없다(jsdom 이 `^8.9.0` 선언, override 불필요, MIT 유지).
- **승인이어도 남는 우려(릴리즈 노트·다음 회차)**: ① `node-runtime.test.ts:28-33` 은 major 만 비교하므로 Node 22.0.0 이나 23/25 에서는 가드가 초록인 채 같은 `markAsUncloneable` 증상이 재발할 수 있다(jsdom 은 `^22.22.2`, 그 API 는 22.14+). 지금 CI 는 `node-version: 22`→22.23.x 라 안전. ② `npm audit` 는 실시간 DB 조회라 코드 변경 없이 다시 붉어질 수 있고 그때도 저장소 결함이 아니다. ③ `webapp/scripts/run-vitest.mjs` 는 어떤 tsconfig include 에도 없어 타입검사를 받지 않으며 `require.resolve("vitest/package.json")` 은 장래 vitest 의 exports 맵에 취약하다.
- 못 본 것: Windows 실행(단 `scripts/verify.ps1` 은 `npm test` 를 돌리지 않으므로 Windows 검증 경로는 무영향), 실제 CI 실행(`gh` 미인증), `go test ./...`(diff 에 Go 변경 0 — `git diff --stat` 으로 확인하여 생략).
- 구현자의 자기 수정은 맞다: CI 의 `npm test` 는 base 에서도 통과했을 것이고 실제 차단은 audit 쪽이었다. 커밋 메시지는 "CI 도 깨진다" 고 주장하지 않으므로 설명·동작 불일치는 없다. 범위 이탈도 없다(두 커밋 모두 과제가 가리킨 jsdom/undici 조합 문제).
- [러너 08:37] review approved — 리뷰 승인 (risk=low)
- [러너 08:37] pr created — https://github.com/hkjang/moyro/pull/31
- [러너 08:49] ci passed — 검사 3개 모두 success
- [러너 08:50] merge done — e6234a9
- [러너 09:08] release published — v0.2.43
- [러너 09:21] assets verified — v0.2.43 자산 1개 (이전 v0.2.42: 1)
