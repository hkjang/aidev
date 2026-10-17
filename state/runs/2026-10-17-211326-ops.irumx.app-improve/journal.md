# 회차 노트 2026-10-17-211326-ops.irumx.app-improve — ops.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:13] base pinned — main@e85afae
- [러너 21:13] autonomy release — 

## 정찰 노트
- 자동 배정된 "워크플로 실패 고치기" 는 성립하지 않았다: base 에 `.github` 가 없고(ls 확인), 마지막 회차는 시험 실패가 아니라 러너 중단이었다(`2026-10-17-124541/journal.md:5`, 그 브랜치 커밋 0개). CI·pretest 수정은 이미 `origin/auto/2026-10-16-2028`·`-1704` 에 푸시돼 있어 다시 올리면 중복 PR — 그래서 두 파일을 금지 목록에 넣었다.
- 그 대신 고른 것: `do:*` 집계 자원의 **표시 라벨만** 교체(프로덕션 3 + 시험 1 파일). ui-*.spec.ts(가치 4)를 제친 이유는 브라우저 미설치 + dist 필요 + ci.yml 충돌로 45분 안에 끊길 위험이 커서다 — 직전 회차가 실제로 PR 전에 끊겼다.
- 추측으로 적은 것: `resourceLabel` 을 `shared/policy.ts` 에 두라는 배치는 내 판단이다(두 소비자가 이미 그 모듈을 import 한다는 것만 확인). `npm run test:unit` 기준선 51은 2026-10-16 회차의 실측치이며 이번 세션에서 직접 돌려 보지는 못했다(샌드박스 권한).
- 구현자가 조심할 것: `a.resource` 는 스파크라인 조회 키(`anomalies.tsx:102·154`)와 `dedupe_key`(`run.ts:126·328`)로도 쓰인다 — 라벨로 바꾸면 중복 억제가 깨진다. 표시 지점 2곳만 손댈 것.
- [러너 21:18] scout done — 이상 탐지의 `do:*` 집계 가짜 자원을 사람이 읽는 라벨로 보여 준다 — 알림 제목·화면에 `do:*` 노출 제거 (

## 구현 노트
- 바꾼 것: `shared/policy.ts` 에 순수 함수 `resourceLabel`(`'do:*'` → `'전체 Durable Objects'`, 그 밖은 입력 그대로) + 표시 지점 3곳 교체 — `run.ts:326` 알림 제목 폴백, `run.ts:327` 알림 본문 `자원:` 줄, `anomalies.tsx:299` 화면 `자원` 칸. 커밋 7a29bfa(프로덕션 3 + 시험 1 파일).
- 과제서보다 한 곳 넓혔다: 알림 **본문**도 같은 내부 키를 사용자에게 보내고 있었다. 키가 아니라 표시 문자열이라 `dedupe_key`·조회에 영향 없다 — 비평가가 "과제서에 없는 줄" 로 볼 수 있으니 여기 적어 둔다.
- **확신 없는 곳**: 표시 지점 3곳이 실제 출력까지 바뀌는지는 **tsc 와 코드 읽기로만** 확인했다. `do:*` 가 지나가는 실행 시험이 없다 — `tests/fake-cloudflare.mjs:183` 이 `durableObjectsStorageGroups: []` 를 돌려주므로 api 경로에 DO 저장공간 데이터가 아예 생기지 않는다. 단위 시험은 순수 함수만 덮는다. 여기를 먼저 보라.
- 일부러 안 한 것: 가짜 서버에 DO 저장공간 행 주입(비용 합계가 바뀌어 `api-b-connect.spec.ts` 의 '대사 —' 수치를 흔든다), `anomalies.service_id` null 과 `collect/analytics.ts` 자원 키(최근 3회차가 되돌린 자리 a6eb4ea→d30940e→a47c108), `package.json`·`.github/*`(미머지 브랜치와 중복).
- 검증: `npm run test:unit` 50 passed(기준선 49), `npm run check` exit 0, `npm run build && npm test` **91 passed (14.9s) exit 0**. red 는 항등 함수 뼈대로 받았다 — `Expected: "전체 Durable Objects" / Received: "do:*"`.
- 다음 역할이 조심할 것: ① 워크트리에 `node_modules` 가 없다 — `npm ci` 먼저(123 packages, 3초). ② `npm run build` 가 **추적 파일** `public/fonts/*.woff2`·`src/client/styles/fonts.css` 를 다시 쓴다. 이 환경의 서브셋터 판본 차이로 바이트가 달라져 커밋 전에 되돌렸다(2026-10-16 회차에는 동일했다고 기록돼 있다) — 빌드 후 `git status` 를 꼭 보라. ③ `a.resource`/`f.resource` 원문은 스파크라인 조회 키와 `dedupe_key` 다. 라벨로 바꾸지 말 것.
- [러너 21:23] brief accepted — 채택 — 근거 세 자리(`analytics.ts:184-185` 의 `'do:*'`, `run.ts:326` 폴백, `anomalies.tsx:299`)를 모두 코드로 확인했고 지정한 파일 4�
- [러너 21:24] verify passed — 검증 4개 통과 (auto)

## 비평 노트
- 확인함: diff 4파일 전부, `npm run check` exit 0, `npm run test:unit` **50 passed**(새 시험 `unit-policy.spec.ts:254` 가 실제로 돌고 항등 함수로는 통과 못 한다). 조회 키 보존도 코드로 확인 — `dedupe_key`(run.ts:328)·`byKey`(run.ts:112)·스파크라인(anomalies.tsx:102·154)·`governance.ts:420` 은 모두 원문을 쓴다. 구현자가 의심한 자리(`do:*` 실행 시험 없음)는 사실이나, 레이어가 맞아 결함은 그쪽이 아니었다.
- **거절 — 수리가 가장 먼저 볼 파일: `src/worker/engine/run.ts:174`.** 같은 함수가 cost_jump 을 `resource: 'account'` 로 만든다. 이 키도 `resources.key` 와 안 맞아 고친 폴백(run.ts:326·327)·화면(anomalies.tsx:299)을 그대로 지나가 `[이룸옵스] 예상 비용 급등 — account` / `자원: account` 로 나간다 — 커밋이 말한 증상이 **더 자주 터지는 경로에서 그대로 남았다**(severity 가 medium|high 뿐이라 알림 필터를 늘 통과). 저장소 표현은 이미 '계정 전체'(services.tsx:721·budgets.tsx:289). `resourceLabel` 에 분기 한 줄 + 시험 한 줄.
- 같이 볼 것: `collect/analytics.ts:135` 의 `'r2:*'` — 지금은 표시 경로에 닿지 않음을 확인했다(저장공간 지표 없어 `detectStorage` 밖, 시간별은 bucketName 필수라 `detectHourly` 밖). 차단 사유는 아니나 `resourceLabel` 을 다시 열 때 함께 넣을 자리. 주석의 `collect/analytics.ts:183` 은 실제 184-185 — 한 줄 밀렸다.
- 못 봄: 실제 브라우저·실제 알림 메일(`npm test` 미실행 — 폰트 추적 파일 재작성 때문에 피했다). `anomalies.tsx:299` 의 `className="mono"` 가 이제 한글 라벨에 등폭을 입힌다(취향, `ui-*.spec.ts` 없어 아무도 못 막음).
- 보안·법무 차단 없음: 인증·권한·결제·마이그레이션·비밀값·삭제 미접촉, 새 의존성·새 개인정보 없음. 방향은 반대로 내부 식별자 노출을 줄인다. revert 로 완전히 되돌아온다(스키마·데이터 변경 없음).
- [러너 21:28] review rejected — 리뷰 거절: src/shared/policy.ts:362 `resourceLabel` 은 주석에 '알려진 집계용 가짜 키만 바꾼다' 고 적었는데 `'account'` 를 빠뜨렸다. `src/worker/engine/run.ts:174` 가 co
- [러너 21:28] pr created — https://github.com/hkjang/ops.irumx.app/pull/3
