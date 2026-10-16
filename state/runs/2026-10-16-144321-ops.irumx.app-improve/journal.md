# 회차 노트 2026-10-16-144321-ops.irumx.app-improve — ops.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:43] base pinned — main@e85afae
- [러너 14:43] autonomy release — 

## 정찰 노트
- 최근 3회차가 전부 계측·이상 탐지 오탐 자리였고, 직전 e85afae 가 `detectHourly` 에만 firstSeen 가드를 넣고 `detectStorage` 를 남겨 둔 것을 확인했다 — 같은 결함 유형의 미완성 절반이라 가장 확실한 한 조각이다. 브라우저 시험 0개(ui-*.spec 파일 없음·axe 미사용)가 가치는 더 크지만 build+wrangler+가짜 서버 3개가 필요해 한 세션에 플래키 위험이 커서 2순위로 미뤘다.
- 추측으로 적은 것: `usage_daily.resource` 와 `resources.key` 가 같은 문자열이라는 전제(detectHourly:114 가 이미 그렇게 쓰고 있어 유추했을 뿐, 수집 경로에서 끝까지 따라가 확인하지는 않았다). 과제서에 "가드가 조용히 무동작할 수 있다" 로 적어 두었으니 구현자는 0/1 양쪽 시험으로 반드시 확인할 것.
- 이 세션에서는 샌드박스 권한 때문에 `npm run test:unit` 을 한 번도 돌려 보지 못했다 — 구현자는 변경 전에 먼저 한 번 돌려 기준선을 잡을 것.
- 조심할 것: 임계값을 같이 손대지 말 것(오탐 줄이려다 탐지 자체를 죽인 전례가 계측 쪽에 있다). 기존 시험 unit-policy.spec.ts:192-204 를 고쳐야 하면 설계가 틀린 신호다.
- [러너 14:47] scout done — 저장공간 증가 이상 탐지도 ‘처음 본 지 하루가 안 된 자원’은 건너뛴다 (가치 4 / 위험 2 / 작업량 S)

## 구현 노트
- 과제서 그대로: `detectStorage` 에 선택 매개변수 `firstSeen` 을 `detectHourly` 와 같은 순서로 넣어 어제 처음 본 자원의 첫날 적재를 건너뛰고(6e54d5e), `run.ts:evaluateOrg` 는 맵을 `const firstSeen` 으로 한 번 만들어 양쪽에 넘긴다. 명시적으로 허용된 차선 후보도 별도 커밋(6f117c3) — 어제 값이 `storageGrowthMinBytes` 미만이면 `ratio` 기반 severity 를 쓰지 않는다. 임계값은 하나도 건드리지 않았다.
- 정찰이 '추측' 으로 남긴 키 전제를 코드로 확인했다: `collect/analytics.ts` 가 넣는 `d1:<databaseId>`·`r2:<bucketName>`·`kv:<namespaceId>` 는 `collect/resources.ts` 의 `d1:<uuid>`·`r2:<name>`·`kv:<id>` 와 같은 문자열이라 가드가 실제로 걸린다. 시험도 같은 rows·firstSeen 만 바꾼 0/1 짝이라 '다른 이유로 0' 이 아님을 증명한다.
- 확신 없는 곳: (1) `do:*` 는 집계용 가짜 자원이라 `resources.key` 와 절대 안 맞고 가드가 무동작한다 — 일부러 그대로 뒀다(집계에는 first_seen 이 없다). 다만 같은 이유로 `resourceSvc.get('do:*')` 도 항상 undefined 라 DO 저장공간 이상은 서비스 귀속이 없고 알림 제목에 `do:*` 가 그대로 나온다 — 이번 범위 밖, ideas.json 에 적었다. (2) 가드 기준시각을 `dateMs(today) - 24h` 로 잡아 '어제 중간에 생긴 자원' 은 건너뛰고 '그제 생긴 자원' 은 잡는다(과제서 지정). 실제 D1 로 돌려 본 end-to-end 확인은 못 했다 — `evaluateOrg` 는 단위 시험이 닿지 않는다.
- 일부러 안 한 것: `queues.reads` 재시도·`workers.errors` 오류 비율 가드 제외 유지(`anomaly.ts:70` 주석대로), 임계값·`detail` 문장 불변(감사 details 에 원문 추가 금지 지시), `migrations/`·auth·배포 스크립트 무변경.
- 다음 역할이 조심할 것: 이 워크트리에는 `node_modules` 가 없어 `npm install` 을 먼저 돌려야 한다(5초). `npm install` 이 `package-lock.json` 에 dev 플래그 잡음을 만드므로 커밋 전에 되돌렸다 — 다시 설치하면 또 생긴다. 검증은 `npm run test:unit` 51 passed(기준선 49), `npm run check` 통과. `npm test`(API)는 build+wrangler+가짜 서버 3개가 필요해 돌리지 않았다.
- [러너 14:52] brief accepted — 채택 — 근거가 지금 코드와 정확히 맞았고(`detectStorage` 에 가드 없음, `run.ts:166` 에 맵이 이미 있음) 지정한 파일 3개·수�
- [러너 14:52] verify failed — 실패한 검증: npm test --silent (exit 1)
