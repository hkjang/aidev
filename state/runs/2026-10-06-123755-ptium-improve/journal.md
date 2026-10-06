# 회차 노트 2026-10-06-123755-ptium-improve — ptium
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:38] base pinned — main@e81b9d9
- [러너 12:38] autonomy release — 

## 정찰 노트
- 자동 배정된 수정 과제라 후보 선택은 없었다. 대신 **원인을 어디까지 좁힐 수 있나**에 시간을 썼다: `cb30ae5` 는 docs 한 파일이므로 원인일 수 없고, `ci.yml:40` 의 `npm audit --audit-level=high` 만이 코드 변경 없이 날마다 판정이 뒤집히는 단계다. 수리 시도 `d174036` 의 커밋 제목이 "advisories the audit gate stops on" 이라 이 진단을 뒷받침한다.
- 확인 못 한 것: **어떤 권고인지**. 이 샌드박스는 `npm` 실행과 `WebSearch` 가 모두 차단되고 CI 로그도 런 디렉터리에 없다. 그래서 과제서를 "0단계로 사실을 먼저 확보하고 그 출력에 따라 갈라지는 절차" 로 썼다 — 추측한 세 가지를 과제서 끝에 명시했다.
- 락파일에서 **실측**한 것(추측 아님): 트리에 vite 가 두 벌(최상위 8.2.0 / `vitest`·`vite-node` 아래 중첩 7.3.6, `:3423`·`:3598`), `vitest` 3.2.7 이 vite 를 peer 가 아니라 **일반 dependency** 로 `^5||^6||^7` 선언(`:3525`), postcss 8.5.25, esbuild 0.28.2. 그래서 최상위 vite 를 올려도 중첩 사본은 안 바뀐다 — 수리 에이전트가 vitest 메이저로 뛴 이유가 이것으로 설명된다.
- 구현자가 조심할 것: ① `ci.yml` 은 한 글자도 바꾸지 말 것(게이트 완화 금지는 운영자가 되풀이한 반려 사유다) ② `d174036` 의 vitest 3→5 + config 재작성을 그대로 재제출하지 말 것 ③ `tsconfig.node.json:11` 이 `vite.config.ts` 만 include 하므로 **`vitest.config.ts` 는 `tsc -b` 가 보지 않는다** — 그 파일을 고치면 `npm test` 로만 검증되고, 설정이 틀리면 한쪽 환경이 조용히 0개 파일로 돈다.
- 길이 정말 없으면(`No fix available`) **고치지 말고 멈추고 `npm audit` 출력 전문을 남기라**고 과제서에 적었다. 게이트를 느슨하게 해 녹색을 만드는 것보다 사람에게 넘기는 쪽이 낫다.
- [러너 12:44] scout done — CI 의 `npm audit --audit-level=high` 가 막는 권고를 **락파일/overrides 범위에서** 해소해 PR #44 를 녹색으로 돌리기 

## 구현 노트
- 이 샌드박스는 `npm` 이 돌아가서 0단계를 실제로 했다. **정찰의 진단은 틀렸다**: vite 권고는 0건이고, 막던 것은 `source-map-js` 1.2.1(high)과 `tinypool` 1.1.1(critical) 둘이었다. 앞은 postcss 선언 `^1.2.1` 안의 1.2.2 라 `npm audit fix` 가 락파일만 고쳐 끝났고, 뒤는 패치된 1.x 가 없어(`<=2.1.1` 전부 취약) `overrides: {"tinypool":"^2.1.2"}` 로 2.2.0 으로 넘겼다. 프로덕션 2파일, `vitest.config.ts` 는 안 건드렸다. 커밋 0f06b6d.
- **확신 없는 곳**: overrides 가 vitest 3.2.7 의 선언 `^1.1.1` 을 거짓으로 만든다는 점 — 과제서가 경계한 바로 그 수단이다. 근거는 2.x 가 Node 하한 변경뿐(`^18||>=20`→`^20||>=22`)이고, tinypool 이 **vitest 가 모든 테스트를 돌리는 워커 풀**이라 48파일/271테스트가 1.1.1 때와 같은 수로 통과한 것이 API 호환의 실증이라는 것. 그래도 비평가가 먼저 볼 자리로 적어 둔다 — 워커 풀의 미묘한 동작(타임아웃·격리·종료 경로)까지 테스트가 덮는지는 확인 못 했다.
- **일부러 안 한 것** ① vitest 4.1.11 로 올리기: 더 깔끔하지만(tinypool 의존 소멸 + 중첩 vite 접힘 + moderate 해소) `npm install` 이 **npm 10.9.8 arborist 의 #loadPeerSet 에서 크래시**한다(`Cannot read properties of null (reading 'edgesOut')`). vitest 3.2.7 로는 정상이라 버전 특정 재현을 확인했다. 하필 그게 node 22 가 싣는 npm = CI 의 npm 이라, CI 가 재생성 못 하는 락파일을 남기게 되므로 뺐다. ② `@vitest/mocker` moderate 는 게이트(`high`) 아래라 남겼다 — 넘으려면 vitest 5 가 필요하고 그건 `d174036` 이 두 번 실패한 길이다. ③ `ci.yml` 은 한 글자도 안 바꿨다(`git diff --stat` 으로 `.github/`·`server/` 변경 0 확인).
- **다음 역할이 조심할 것** ① 이 과제의 게이트는 **네트워크 의존**이다 — 새 권고가 올라오면 코드 변경 없이 다시 빨개진다. 재검증은 반드시 `npm ci` 뒤 `npm audit --audit-level=high` 의 exit 코드로 할 것. ② `web/package-lock.json` 을 손으로 고치지 말 것(integrity 해시). ③ `npm install` 로 vitest 를 올리려 하면 위 arborist 크래시를 만난다 — 버그이고 내 변경 탓이 아니다. ④ docker 는 이 환경에서 돌았고 통과했다(web 스테이지를 `--no-cache-filter web-build` 로 강제 재빌드해 컨테이너 안 `npm ci` 가 이 락파일로 돌아 "2 moderate" 만 보고하는 것까지 확인). ⑤ 버전·릴리즈 노트·배포 매니페스트는 손대지 않았다.
- [러너 12:55] brief accepted — 채택 — 절차(0단계로 사실 확보 → 1단계 `npm audit fix` → 2단계 경로별 분기)와 "패치된 버전이 선언 범위 밖이면 overrides �
- [러너 12:55] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 승인. 원장의 주장을 모두 다시 돌려 확인했다: base 락파일로 `npm audit --audit-level=high` **exit 1**(source-map-js high + tinypool critical 2건), HEAD 로 **exit 0**(moderate 2건). `npm ci`·`npm test`(48/271)·typecheck·build 전부 통과.
- 구현자가 의심한 자리(overrides 가 vitest 의 `^1.1.1` 선언을 거짓으로 만드는 것)를 테스트 통과만으로 믿지 않고 타입으로 확인했다 — tinypool 2.2.0 의 Options/RunOptions 에 vitest 가 넘기는 키 전부가 남아 있다(teardown·isolateWorkers·terminateTimeout·useAtomics·channel 포함). 반려된 vitest 4.1.11 경로도 재현했다(npm 10.9.8 arborist 크래시, exit 1).
- 락파일 수작업 의심도 제거: HEAD 로 `npm install --package-lock-only` 재생성이 커밋본과 바이트 동일. 단 npm 10.9.8 은 overrides 를 락파일 루트에 안 적으므로 게이트 통과는 락파일 트리에만 의존한다.
- 못 본 것: 워커 풀의 타임아웃·취소·강제 종료 경로(`Failed to terminate worker`, cancelPendingTasks)는 이 테스트 묶음이 지나가지 않는다 — 2.x 에서 조용히 달라져도 지금 묶음은 녹색일 수 있다. Go·docker 는 변경이 없어 다시 돌리지 않았다(이번 diff 는 web 2파일).
- 릴리즈 담당에게: 이 게이트는 레지스트리 라이브 조회라 코드 변경 없이 다시 빨개진다 — 태깅 직전 `npm ci` 뒤 exit 코드로 한 번 더. @vitest/mocker moderate 는 dev 전용이라 남겨도 되고, vitest 4+ 로 올리는 회차가 오면 이 overrides 항목을 함께 지워야 한다.
- [러너 13:07] review approved — 리뷰 승인 (risk=low)
- [러너 13:07] pr created — https://github.com/hkjang/ptium/pull/45
- [러너 13:11] ci passed — 검사 1개 모두 success
- [러너 13:11] merge done — 0f06b6d
- [러너 13:23] release published — v1.69.58
- [러너 13:23] gh-release created — GitHub Release v1.69.58
- [러너 13:23] manifest ok — ptium-1.69.58.tar.gz ptium-1.69.58.tar.gz.sha256 docker-compose.ptium-1.69.58.yml ptium-1.69.58.env.example load-ptium-1.69.58.ps1 load-ptium-1.69.58.sh ptium-1.69.58.kubernetes.yaml 
- [러너 13:23] assets uploaded — 7개
- [러너 13:23] assets verified — v1.69.58 자산 7개 (이전 v1.69.57: 7)
