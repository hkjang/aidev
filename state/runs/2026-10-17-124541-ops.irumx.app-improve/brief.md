(초안 — 확인 중. 아래 내용은 이 시점의 최선이며 세션 중 덮어쓸 수 있다.)

- 과제: 이상 탐지 알림에 집계용 가짜 자원 `do:*` 가 그대로 노출되는 것을 사람이 읽을 라벨로 바꾼다 (가치 3 / 위험 2 / 작업량 S)
- 왜: `src/worker/collect/analytics.ts:184-185` 가 Durable Objects 저장공간을 `resource: 'do:*'` 로 넣지만 `resources.key` 는 `do:<namespaceId>` 라, 이 자원은 `resources` 와 영원히 매칭되지 않는다. 그래서 이상 항목의 `service_id` 가 null 이고 알림 제목에 `do:*` 라는 내부 문자열이 운영자에게 그대로 보인다.
- 수용 기준: 1) `do:*` 로 만들어진 이상 항목의 사용자 노출 문자열이 `전체 Durable Objects` 류의 한국어 라벨이 된다 2) 계측 수치(qty·metric·resource 키 자체)는 바뀌지 않는다 3) 단위 시험이 `do:*` 자원의 이상 항목 제목/라벨을 검증한다
- 건드릴 파일: (확인 중)
- 검증 명령: `npm run test:unit`, `npm run check`
- 위험과 피할 것: 계측 공식은 절대 건드리지 말 것(a6eb4ea → d30940e → a47c108 되돌림 이력). `migrations/`·`auth.ts`·`.github/` 는 건드리지 않는다. `pretest`(package.json)와 `.github/workflows/ci.yml` 은 직전 두 회차가 이미 PR 로 올렸고 이 base 에는 없다 — **같은 변경을 다시 올리지 말 것**.
- 차선 후보: (확인 중)
