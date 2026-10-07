## 2026-10-07
- 선택: 예산·시험 수 한도 동시 예약 시험 (설계 수용 시험 12번) (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `usage.ts:reserveSql` 은 한도 검사와 INSERT 를 조건부 INSERT 한 문장에 넣어 동시 예약을 막게 설계됐는데 그 동시성을 밟는 시험이 없었다. `tests/api-targets.spec.ts` 에 동시 예약 시험 1개를 더했다 — `estimate` 로 예약 없이 한 건의 시험 수를 재고, `GET /api/w/:ws/usage` 로 기준선을 읽어 딱 2건만 들어가는 한도를 맞춘 뒤 같은 업체 공간에 `POST .../experiments` 5건을 한꺼번에 보내 201 이 정확히 2개·나머지는 409 over_budget 인지, 끝으로 `trials` 합계가 기준선+성공분과 같은지 본다. 프로덕션 로직 변경 0개(util.ts 주석 1줄에서 없는 파일 `tests/guard.spec.ts` 참조만 실제 시험 경로로 고쳤다). 검증: `npm run build && npm test`.
- 실패 재현: `Error: 201,201,201,201,201` / `expect(received).toBe(expected) Expected: 2 Received: 5` — `reserveSql` 의 시험 수 조건을 빼고 같은 검사를 JS 로 먼저 읽게(SELECT 후 INSERT) 바꾼 상태에서 새 시험만 실패했고 **기존 순차 예산 시험은 그대로 통과**했다 — 새 시험이 기존 시험이 못 보는 바로 그 구멍을 막는다는 증거.
- 보류 아이디어: 큐 중복 전달 시험(수용 시험 10, 4/3/M — 큐에 두 번 넣는 시험 경로가 있는지 미확인) · 비밀 모양 문자열 가림을 원문·보고서 출력 경로로만 좁히기(수용 시험 23, 3/4/M) · build+unit 만 도는 최소 CI 워크플로(3/2/M, 보호 경로라 운영자 판단 필요) · D1 바인딩 100개 한도 배치 쪼개기 경계 단위 시험(3/1/S) · 시험 보조 수단 기록: `wrangler dev` 가 `dist/` 의 빌드 결과를 띄운다는 점을 docs 에 적기(2/1/S)
- 과제서: 채택 — 과제서의 근거(reserveSql 조건부 INSERT · 시험 공백 · estimate/usage 경로)는 코드와 그대로 맞았다. 다만 과제서가 적은 `api.raw` + `Promise.all` 로는 Playwright 요청 컨텍스트가 요청을 하나씩 보내 동시성을 전혀 밟지 못했고(회귀를 심어도 통과), 쿠키를 꺼내 Node 전역 `fetch` 로 보내야 비로소 재현됐다.
