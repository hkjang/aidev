## 2026-10-16
- 선택: 저장공간 증가 이상 탐지도 ‘처음 본 지 하루가 안 된 자원’은 건너뛴다 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: 직전 회차(e85afae)가 `detectHourly` 에만 넣은 `firstSeen` 가드를 `detectStorage` 에도 넣어(선택 매개변수 4번째, 같은 인수 순서) 어제 처음 본 R2 버킷·D1 DB 의 첫날 적재가 `storage_growth` 로 잡히지 않게 했고, `run.ts:evaluateOrg` 에서 인라인으로 만들던 맵을 `const firstSeen` 으로 한 번만 만들어 양쪽에 넘겼다. 같은 함수의 차선 후보도 별도 커밋으로 담았다 — 어제 값이 `storageGrowthMinBytes` 미만이면 `ratio` 가 수억 배로 튀어 +0.30GB 가 high 가 되던 것을 증가량만으로 판정하게 했다(임계값은 그대로, severity 유도만 변경). 검증은 `npm run test:unit` 51 passed(기준선 49 → 신규 2) 와 `npm run check`(tsc 4개) 통과. 가드 키 전제도 코드로 확인했다 — `usage_daily.resource` 와 `resources.key` 는 d1/r2/kv 모두 같은 문자열이고(`collect/analytics.ts` vs `collect/resources.ts`), `do:*` 만 집계용 가짜 자원이라 가드가 무동작한다.
- 실패 재현: `Expected length: 0 / Received length: 1 / Received array: [{"baseline": 50000000, "detail": "하루 사이 0.05GB → 0.40GB(+0.35GB)", "kind": "storage_growth", ..., "ratio": 8, "severity": "high"}]` — 두 번째 커밋 쪽은 `Expected: "low" / Received: "high"`.
- 보류 아이디어: 브라우저·접근성 시험 첫 한 개(ui-*.spec.ts, 개요 화면 스모크 + axe) / `do:*` 집계 가짜 자원은 `resources.key` 와 영원히 안 맞아 서비스 귀속도 없고 알림 제목에 `do:*` 가 그대로 나온다 / `playwright.config.ts` 의 mobile·shots 프로젝트는 대응 파일이 없는 죽은 설정 / CLAUDE.md 로 저장소 관례 고정
- 과제서: 채택 — 근거가 지금 코드와 정확히 맞았고(`detectStorage` 에 가드 없음, `run.ts:166` 에 맵이 이미 있음) 지정한 파일 3개·수용 기준 4개를 그대로 구현했으며, 명시적으로 허용된 차선 후보를 severity 쪽만 별도 커밋으로 더했다.
