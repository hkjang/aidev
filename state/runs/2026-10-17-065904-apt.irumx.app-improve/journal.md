# 회차 노트 2026-10-17-065904-apt.irumx.app-improve — apt.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 06:59] base pinned — main@b14cc46
- [러너 06:59] autonomy release — 

## 정찰 노트
- 고른 이유: 가장 최근 두 회차(266bfce·b14cc46)가 매달린 DB 차단기에서, `baseTx` 가 호출 시작 시점의 breaker 를 캡처해 `trip` 에 넘기는 탓에 동시 요청에서 strikes 가 누적되지 않아 백오프 자체가 작동하지 않는다. 프로덕션 파일 1개, 기존 테스트 하네스(dropper/hole 서버, Date.now 모킹)를 그대로 쓸 수 있어 S.
- 제친 후보: inspection-attention 의 `in ${s([])}` 구문 오류 의심은 fieldRoles 와 recordAssigneeRoles('inspection') 이 같은 집합이라 도달 불가로 기각. grace_days 불일치는 제품 판단이 필요해 보류. herd 프로브 좁히기는 Promise 공유가 들어가 차선으로 남김.
- 확신 없는 곳: AC2 의 구현 방식(`closedAt` Map + startedAt 비교)은 내가 제안한 것으로, 저장소에 선례가 없다. 구현자가 같은 성질을 더 단순히 만족시키면 그 쪽이 낫다. 검증 명령은 scripts/test.mjs 와 ci.yml 에서 읽은 실제 명령이지만 **이 세션에서 실행하지 못했다(승인 제약)** — 로컬 포트 55462 DB 가 뜨는지도 미확인.
- 구조적 함정: `server/vendor/postgres/**`(패치 사본) 와 `baseTx` 안의 `set_config` RLS 줄은 건드리지 말 것. 대기 상수(10s/120s/connect_timeout 5s)는 기존 테스트가 숫자로 못 박아 두었다.
- 증명 방식: `Promise.allSettled` 로 두 호출을 실제로 겹쳐 돌릴 것. `trip()` 을 직접 호출하는 단위 테스트는 "baseTx 가 캡처한 값을 넘긴다"는 배선 결함을 구조적으로 못 본다.
- [러너 07:04] scout done — DB 차단기 백오프가 동시 요청에서 누적되지 않는 것 고치기 (가치 4 / 위험 2 / 작업량 S)

## 구현 노트
- 과제서 그대로 채택. `trip` 이 트립하는 순간 `breakers.get(key)` 를 다시 읽어 strikes 를 더하고, 키별 `closedAt` 으로 "차단기가 닫힌 뒤에 끝난 실패" 를 버린다. `breakers.delete(key)` 두 자리는 `close(key)` 로 묶었다. 프로덕션 파일 1개(server/db.ts) + 테스트 1개.
- 확신 없는 곳: AC2 의 `closedAt > startedAt` 비교는 같은 밀리초에 성공과 실패가 겹치면 보수적으로 트립한다(열리는 쪽). 의도한 선택이지만 선례가 없다. `closedAt` Map 은 성공마다 쓰이고 비워지지 않는다 — 키 수가 DB 주소 수로 묶여 있어 누수로 보지 않았다.
- AC2 테스트를 위해 드로퍼에 'hold' 모드(연결을 받고 답하지 않음)를 더했다. `held` 가 올라올 때까지 10ms 폴링해 기다린 다음 `mode='pass'` 로 바꾼다 — 느린 실패(5초 connect timeout)와 빠른 성공의 순서를 벽시계가 아니라 서버 쪽 사실로 고정하려는 것이다. 이 테스트는 5초 걸린다.
- 일부러 안 한 것: 차선 후보(재개 순간 단일 프로브)는 Promise 공유가 들어가 범위 밖으로 뺐다. 대기 상수·`connect_timeout`·"재시도 없음" 설계·`isDatabaseUnreachable` 정규식·`set_config` 줄은 손대지 않았다. b14cc46 이 남긴 prettier 미정렬 한 줄은 과제와 무관해 되돌렸다.
- 다음 역할이 조심할 것: 이 테스트들은 **로컬 127.0.0.1:55462 PostgreSQL 이 있어야 돈다**(README 의 `irumx-apt-test` 컨테이너). 그리고 **같은 DB 에 `npm test` 를 두 번 이상 돌리면 integration·예약·bill-analysis 에서 403/404·403/409 불일치로 6건쯤 깨진다** — 내 변경과 무관하고 누적 데이터 탓이다(새 컨테이너 + `db:migrate` 에서 216/216 통과, 베이스라인 214/214 도 같은 조건에서 통과 확인). 검증은 매번 DB 를 새로 만들어서 하라. ideas.json 에 후보로 적어 두었다.
- 확인: `node --test artifacts/db-breaker.mjs` 12/12(3회), `npm test` 216/216, `npm run typecheck`, `npm run build`, `npm run test:runtime` 2/2. e2e·test:assets 는 돌리지 않았다(이 과제와 무관, 각각 dev 서버·miniflare 필요).
