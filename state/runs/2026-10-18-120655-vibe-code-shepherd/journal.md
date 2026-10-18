# PR 처리기 노트 2026-10-18-120655-vibe-code-shepherd — vibe-code PR #22
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-18-111749-vibe-code-improve)
# 회차 노트 2026-10-18-111749-vibe-code-improve — vibe-code
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:18] base pinned — main@6eaede8
- [러너 11:18] autonomy release — 
- [러너 11:22] scout done — `외부 통신 점검`(`vibe-code.auditNetwork`)이 `offlineMode = offline` 에서도 외부 호스트 6곳에 HEAD 요청을 보내는 것 

## 구현 노트
- 무엇/왜: `auditNetwork` 가 `offlineMode` 를 읽지 않아 폐쇄망 선언 사용자에게서 외부 6곳으로 HEAD 가 나갔다. `diagnostics.ts` 에 순수 export `auditHosts(mode)` 를 넣어 `offline` 이면 빈 목록 → `withProgress` 미호출 + `probes=[]`, `reports.ts` 는 빈 프로브에서 "모두 연결됩니다"(`0===0`) 거짓 단언 대신 폐쇄망 안내를 내고 `offline` 재권유를 막았다. 커밋 `3453f1e`, 프로덕션 2파일 + 테스트 1파일.
- 확신 없는 곳·검증 못 한 것: **명령 핸들러를 실행하지 않았다** — `withProgress` 가 실제로 안 뜨는지와 소켓이 진짜 0건인지는 미확인이고, `hosts.length ? ... : []` 삼항은 타입 검사로만 보장된다(비평가는 여기를 먼저 보라). 증명 범위는 "프로브 목록이 비고 보고서 문자열이 바뀐다" 까지. Windows 패키징·UI 스냅샷·원격 CI 미확인. `npm run build` 의 런타임 자산 누락 경고는 기존 그대로.
- 일부러 하지 않은 것: `online`/`auto` 는 그대로 프로브한다(진단 명령의 목적이고 기준 4). `AUDIT_HOSTS` 목록·타임아웃·포트·`probeHost`·MCP `needsInternet`·`NetworkInput` 타입·공유 `vscode-stub.ts`·새 설정/명령/nls 키는 손대지 않았다. `offline` 을 안 지키는 다른 경로(웹 가져오기·공개 URL git·`update-check.ts`)는 범위를 넓히지 않으려 `ideas.json` 의 새 항목으로만 남겼다.
- 다음 역할이 조심할 것: 워크트리에 `node_modules` 가 없어 `npm ci` 가 필요하다. Node 20.19.2 는 `/tmp/n20/node-v20.19.2-linux-x64/bin` 에 뒀다(저장소 안에 두면 `npm ci` 가 지운다). 전체 검사는 `GIT_CEILING_DIRECTORIES`·`TMPDIR` 를 임시 디렉터리로 줘야 체크포인트 테스트가 상위 `aidev` 저장소를 탐색하지 않는다.
- 인과 확인 방법이 과제서와 다르다: 두 파일을 통째로 `git checkout HEAD --` 하면 `auditHosts` export 가 사라져 `TypeError`(심볼 누락)가 되므로, export·배선은 남기고 **동작 줄만** 되돌려 같은 `AssertionError` 2건을 재현했다. 두 방식 출력 모두 `validation/` 에 있다.
- [러너 11:27] brief accepted — 채택 — 근거가 지금 코드와 정확히 맞았다(`diagnostics.ts:168` 의 무조건 프로브, nls:145 의 약속, `reports.ts:201`·`:227` 의 빈 배
- [러너 11:28] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 승인. 구현자가 비워 둔 자리(명령 핸들러 미실행)를 직접 메웠다: `/tmp/rev-audit` 의 임시 vscode 스텁 + `node:https` 모킹으로 `vibe-code.auditNetwork` 핸들러를 실행 — offline 에서 https.request 0건·withProgress 0회·보고서에 폐쇄망 안내, auto 에서 6건·1회. 같은 테스트를 `git archive main src` 소스에 대고 돌리면 offline 에서 6개 호스트를 모두 찾아 실패한다. 인과 핸들러 수준 재현 완료.
- 저장소 단위 테스트도 원 checkout 의 Vitest 로 재실행: 19 files / 155 tests 통과(main 152 + 신규 3). 범위 이탈·revert 위험·보안/법무 차단 사유 없음(새 입력·권한·비밀값·개인정보 없음, egress 감소).
- 남는 우려 1: `reports.ts:195` 의 폐쇄망 문구는 `probes.length === 0` 만 보고 `offlineMode` 를 보지 않는다. 지금은 offline 경로만 빈 배열이라 무해하지만 다른 호출자가 빈 프로브를 넘기면 보고서가 거짓말을 한다.
- 남는 우려 2: `online` 은 여전히 프로브한다 — nls:144("확인하지 않기")·status-bar.ts:77 과 어긋나는, 이번에 offline 에서 고친 것과 같은 종류의 불일치. 의도라 적혀 있어 통과시켰으니 다음 회차 후보. `auditHosts` 만 trim/lowercase 정규화하고 usage.ts:111·status-bar.ts:77 은 원문 비교하는 점도 같이 보라.
- 못 본 것: Windows 패키징, UI 스냅샷, 원격 CI, 실제 VS Code 설치. 릴리즈 노트는 "폐쇄망(offline)에서 외부 통신 점검이 더 이상 외부 호스트를 찾지 않는다"로 쓰면 코드와 정확히 맞는다 — `online`/`auto` 는 그대로 찾는다는 단서를 반드시 붙일 것.
- [러너 11:32] review approved — 리뷰 승인 (risk=low)
- [러너 11:32] pr created — https://github.com/hkjang/vibe-code/pull/22
- [러너 11:33] ci failed — 성공이 아닌 검사: UI snapshots=failure, typecheck + unit tests + build=failure · 실패한 검사: ? 잡: UI snapshots 

## 수리 노트
- 지적이 **틀렸다**: 고칠 코드 결함이 없어 커밋하지 않았다. 두 실패 잡은 02:32:32→02:32:34 = **2초**에 스텝 로그 0건·output 전부 null — 첫 스텝이 checkout→setup-node→`npm ci` 인데 `npm ci` 만 수십 초다. 2초에 실패할 스텝이 없으니 **어떤 스텝도 돌지 않았다**. 특히 `UI snapshots` 는 이 diff(진단 보고서 문자열 3파일)와 무관한데 같은 2초에 같이 죽었고, `.github/`·`package.json`·lock 은 손대지 않았다.
- 원인은 계정 차원 Actions 거부로 보인다: 같은 계정·같은 분에 **공개** 저장소는 정상(`umm` 11분 success 02:32:11, `visitflow` success)인데 **비공개**만 2~3초 즉사(`vibe-code`·`yeopjari-your`·`nexabuilder`·`DartFly` 모두 미인증 API 404). Actions 분은 공개 무료·비공개 과금이라 결제/지출 한도와 맞고, 전역 장애나 YAML 오류로는 이 분리가 설명되지 않는다. 같은 비공개 `qurio` 는 10-08 에 19분 정상 — 갈림선은 코드가 아니라 시점이다.
- 게이트는 로컬에서 통과: 워크트리 `npm ci` 후 CI 와 같은 **Node 20.19.2** 로 `npm run check` → tsc 무오류, vitest **19 files / 155 tests** 통과, build 성공, exit 0. `dist/` 자산 누락 경고는 main 과 같은 기존 동작.
- **확신 없는 곳**: 잡별 `annotations_count: 2` 의 원문을 못 읽었다(비공개라 미인증 API 404, 이 환경에 `gh` 토큰 없음). 결제라는 *특정* 사유는 정황 추론이고 인용이 아니다 — 단 "이 변경 탓이 아니다" 는 2초 증거만으로 성립한다. 앞 회차의 남는 우려 1·2(`reports.ts:195` 가 `offlineMode` 를 안 봄, `online` 이 여전히 프로브)는 비평가가 통과시킨 범위 밖이라 손대지 않았다.
- 다음 역할: 결제/지출 한도를 풀고 두 체크를 **re-run** 하면 된다. 워크플로·검증 명령 수정으로 녹색을 만드는 길은 금지라 택하지 않았다.
