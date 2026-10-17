# 과제서 — 2026-10-17-211326-ops.irumx.app-improve

## 먼저: 자동 배정된 "오류 대응"의 진상 (확인함)
자동 적재된 지시는 "릴리즈 워크플로가 같은 이유로 두 번 실패했으니 워크플로 파일과 실패 단계의 스크립트·테스트를 고치라" 였다.
읽어 보니 **저장소에는 고칠 워크플로도, 실패하는 시험도 없다.** 근거는 전부 실제로 확인한 것이다:

1. 마지막 회차는 시험 실패가 아니라 **러너 단계 중단**이었다 — `state/runs/2026-10-17-124541-.../journal.md:5` =
   `[러너 13:30] resume closed — PR 이전 단계에서 중단 — 에이전트 단계는 재실행하지 않음`.
   그 회차 브랜치에는 **커밋이 0개**다: `git log --oneline main..auto/2026-10-17-1245` → 출력 없음. 즉 코드를 한 줄도 남기지 못하고 끊겼다.
2. base `main@e85afae` 에는 `.github/` 가 **없다** — `ls .github/workflows` → `No such file or directory`. GitHub Actions 워크플로 파일이 저장소에 존재하지 않으므로 "워크플로가 실패했다" 를 저장소 안에서 재현할 길이 없다.
3. 유일한 CI 파일과 유일한 시험-전제 수정은 **아직 머지 안 된 두 브랜치에 이미 올라가 있다**(둘 다 origin 에 푸시됨):
   - `origin/auto/2026-10-16-1704` → `7086771` `package.json` 에 `pretest: npm run build`
   - `origin/auto/2026-10-16-2028` → `5b19674` `.github/workflows/ci.yml`
   따라서 이 두 변경을 다시 올리는 것은 **중복 PR** 이다(운영자 규칙: 같은 접근 반복 금지 / 중복 PR 금지). **건드리지 말 것.**

구현자가 받아들여야 할 결론: 이번 회차에 "워크플로 고치기" 는 할 일이 없다(미확인이 아니라, 없음을 확인했다).
대신 **빌드가 필요 없는 빠른 검증(`npm run test:unit`)만으로 끝나는 작은 과제**를 한다 — 지난 두 회차가 전체 `npm test`(빌드 선행 필수) 때문에 시간을 태우고 중단된 패턴을 되풀이하지 않기 위해서다.

---

- 과제: 이상 탐지의 `do:*` 집계 가짜 자원을 사람이 읽는 라벨로 보여 준다 — 알림 제목·화면에 `do:*` 노출 제거 (가치 3 / 위험 1 / 작업량 S)
- 왜: `src/worker/collect/analytics.ts:184-185` 가 Durable Objects 저장공간을 자원 키 `'do:*'` 로 적재하는데, 실제 자원 행(`resources.key`)은 `do:<namespaceId>` 라 `'do:*'` 는 어떤 자원과도 매칭되지 않는다(이 전제는 직전 회차에서도 코드로 확인됨). 그 결과 `src/worker/engine/run.ts:126` 의 `resourceSvc.get(f.resource)` 가 `undefined` → `anomalies.service_id` 가 null 이고, `run.ts:326` 의 알림 제목은 폴백으로 **`do:*` 라는 내부 키를 그대로 사용자에게 보낸다**(`src/client/pages/anomalies.tsx:299` 도 같은 키를 그대로 출력). 라벨만 바꾸면 알림·화면 문구가 읽을 수 있게 되고, 계측 공식은 전혀 손대지 않으므로 되돌림 이력(a6eb4ea → d30940e → a47c108)의 지뢰를 피한다.
- 수용 기준:
  1) `src/shared/policy.ts` 에 순수 함수 `resourceLabel(key: string): string` 이 생기고, `'do:*'` → `'전체 Durable Objects'`, 그 밖의 키는 **입력 문자열을 그대로** 돌려준다(알 수 없는 키를 임의로 바꾸지 않는다).
  2) 이상 알림 제목에서 `do:*` 가 사라진다 — `run.ts:326` 의 `svcName.get(...) ?? f.resource` 폴백이 `resourceLabel(f.resource)` 를 쓴다. 서비스 이름이 있을 때의 동작은 **그대로**(서비스 이름 우선).
  3) 화면에서도 사라진다 — `src/client/pages/anomalies.tsx:299` 의 `{a.resource}` 가 `{resourceLabel(a.resource)}` 가 된다. `a.resource` 를 키로 쓰는 102·154행(`${a.resource}|${a.metric}` 스파크라인 조회)은 **절대 바꾸지 말 것** — 라벨이 아니라 조회 키다.
  4) 시험이 증명할 것: `tests/unit-policy.spec.ts` 에 새 `test('자원 라벨 — do:* 집계 자원', ...)` 이 들어가 (a) `resourceLabel('do:*') === '전체 Durable Objects'`, (b) `resourceLabel('do:abc123') === 'do:abc123'`, (c) `resourceLabel('r2:assets') === 'r2:assets'` 를 확인한다. `npm run test:unit` 기준선(2026-10-16 실측 51 passed)보다 늘어난 수로 통과한다.
- 건드릴 파일 (프로덕션 3 + 시험 1):
  - `src/shared/policy.ts` — `ANOMALY_LABEL`(346행) 옆에 `resourceLabel` 추가. 한 줄 주석으로 왜 `do:*` 가 존재하는지(GraphQL `durableObjectsStorageGroups` 는 namespace 별 분해가 없어 집계 한 줄로만 온다 — `collect/analytics.ts:183`) 남길 것. 저장소 관례대로 한국어 주석.
  - `src/worker/engine/run.ts` — 10~11행 import 에 `resourceLabel` 추가(이미 `ANOMALY_LABEL, SEVERITY_RANK` 를 `../../shared/policy` 에서 가져온다), 326행 제목 폴백 교체.
  - `src/client/pages/anomalies.tsx` — 15행 import 에 추가(이미 `ANOMALY_LABEL` 을 같은 모듈에서 가져온다), 299행 표시만 교체.
  - `tests/unit-policy.spec.ts` — 6행 import 에 `resourceLabel` 추가 + 위 시험 1개.
- 검증 명령:
  - `npm run test:unit` — 빌드 불필요, 외부 서버 불필요. **이것이 이번 회차의 주 검증이다.**
  - `npm run check` — tsc 4개(`--noEmit`). 클라이언트/워커 양쪽 import 를 다 보므로 반드시 돌릴 것.
  - 전체 `npm test` 는 **돌릴 필요 없다**: base `main` 의 `package.json` 에 `pretest` 가 없어 `npm run build` 를 먼저 수동으로 돌리지 않으면 webServer 가 `assets ... missing the required directory` 로 죽는다(2026-10-16 회차에서 확인된 증상). 돌릴 거라면 `npm run build && npm test` 로. 시간이 부족하면 생략하고 생략했다고 적을 것 — **없는 통과를 주장하지 말 것.**
- 위험과 피할 것:
  - `src/worker/collect/analytics.ts` 의 계측 공식·자원 키는 **바꾸지 말 것**. `'do:*'` 를 `do:<namespaceId>` 로 쪼개려는 시도는 최근 3회차가 되돌린 바로 그 자리다(a6eb4ea → d30940e → a47c108). 이번 과제는 **표시 라벨만**이다.
  - `anomalies.service_id` 가 null 로 남는 것은 그대로 둔다(스키마·마이그레이션 금지 — `migrations/` 는 위험 구역).
  - `package.json`, `.github/workflows/*` 를 건드리지 말 것 — 위 2·3번대로 이미 열린 PR 과 중복된다.
  - `anomalies.tsx` 의 102·154행 조회 키, `dedupe_key`(`run.ts:126`·`328`)에 들어가는 `f.resource` 원문도 그대로 — 라벨을 키로 쓰면 중복 억제가 깨진다.
  - 커밋 메시지는 한국어 `영역: 무엇을 왜` 형식. 예: `이상 탐지: 집계 자원 do:* 를 '전체 Durable Objects' 로 표시 — 내부 키가 알림 제목에 그대로 나갔다`.
- 차선 후보: **devDependencies 취약점 8건(moderate 4 · high 4) 선별** — 1순위가 이미 고쳐져 있거나 성립하지 않으면 이것을 하라. `npm audit` 출력으로 어떤 패키지·경로인지 먼저 적고, **패치/마이너 범위에서만** 올릴 것(메이저 업그레이드는 금지). CI 에 audit 단계를 넣지는 말 것(새 CVE 가 뜨면 무관한 PR 이 전부 빨개진다). `migrations/meta` 불일치는 위험 구역이라 차선으로도 고르지 말 것.
