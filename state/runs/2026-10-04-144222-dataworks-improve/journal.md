# 회차 노트 2026-10-04-144222-dataworks-improve — dataworks
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:42] base pinned — main@98c5081
- [러너 14:42] autonomy release — 

## 정찰 노트
- 선택: `usableEntitlement`(dataworks_runtime.go:246)의 `err != nil || !found || ProductKey 불일치` 한 덩어리 분리. 최근 네 회차 연속 성공한 "조회 실패를 부재로 숨기지 않는다" 계열의 마지막 런타임 지점이고 프로덕션 1파일이다.
- 제친 후보: web 계열 3건(keep-dist logLevel·npm audit·루트 .mjs 린트)은 1d63b7c·5ed6876·f927cea 가 모두 main@98c5081 에 없어 web 방향 자체가 반려된 신호로 보고 기각. 게이트 의미가 바뀌는 2건은 운영자 판단 필요로 보류.
- 확신 없는 곳(추측): 재현 레버. 테이블 전체 rename 은 `:100` 이 이미 500 을 내므로 **재현이 안 된다**는 것만 코드로 확인했고, 한 행만 깨뜨리는 방법은 뷰 치환(1순위, 열 이름 14개를 `GetContractScope` 와 맞춰야 함)과 `rate_limit` 텍스트 UPDATE(2순위, SQLite affinity 동작을 이 환경에서 실행 확인 못 함 — `sqlite3` 실행 권한 없음)를 과제서에 둘 다 적었다.
- 구현자가 조심할 것: 오류를 즉시 반환하면 "다른 후보가 유효하면 서비스" 라는 :66-69 의 설계가 깨진다 — 사용 가능한 후보가 하나도 없을 때만 반환할 것. 후보 순서는 `updated_at DESC` 이므로 테스트에서 raw SQL 로 고정할 것.
- [러너 14:47] scout done — `usableEntitlement` 가 `GetContractScope` 조회 오류를 "계약 없음" 과 함께 `continue` 로 묶어, 서버 오류가 403 `contract

## 구현 노트
- `usableEntitlement` 에서 `GetContractScope` 오류만 `!found`/`ProductKey` 불일치와 분리했다. 첫 오류를 지역 변수에 보관하고 루프를 끝까지 돌려, 사용 가능한 후보가 있으면 그 후보로 서빙(오류는 버림)하고 하나도 없을 때만 오류를 반환 → 호출부(:71)가 기존 code 로 500 `contract_lookup_failed`. 프로덕션 1파일(`internal/proxy/dataworks_runtime.go`) + 테스트 1파일 + `docs/OPERATIONS.md`.
- 확신 없는 곳: (1) 감사 로그를 **일부러 남기지 않았다** — 다른 500 들과 일관되고, 실패한 후보가 `ent` 가 가리키는 후보가 아닐 수 있어 식별자를 남기면 오해를 준다. 감사가 필요하다는 판단이면 실패한 `ContractKey` 를 함께 반환하도록 넓혀야 한다. (2) PostgreSQL 에서는 이 경로를 못 돌렸다(SQLite 만). (3) 과금 `defer` 가 아직 설치되지 않았다는 것은 코드 순서로만 확인했고 미터링 행 부재를 DB 로 단언하지는 않았다.
- 일부러 하지 않은 것: `entitlementActive`·`contractScopeActive`·`contractScopeCanServe`·`entitlementAllowsQuery` 판정 로직, `errCode`/과금 흐름, `internal/store`(이번 재현 레버가 노출한 `GetContractScope` 의 COALESCE 누락은 ideas.json 에 신규 후보로만 적었다), web/ 전체.
- 다음 역할이 조심할 것: 새 테스트는 SQLite 파일 DB 가 필요하고(`store.Open` + 별도 `sql.Open` 핸들) `dw_contract_scopes` 를 **뷰로 치환**한다 — cleanup 은 반드시 `DROP VIEW` → `ALTER TABLE ... RENAME` 순서이고, 뷰에는 INSERT 가 안 되므로 모든 쓰기는 뷰 설치 전에 끝난다. 후보 순서는 `updated_at` raw UPDATE 로 고정했으니 그 값을 바꾸면 사례 A/B 의 의미가 뒤집힌다. `gofmt -l internal/proxy/dataworks_runtime.go` 는 HEAD 에서도 파일명을 출력한다(기존 CRLF) — 줄 끝을 건드리지 말 것. 웹 검증은 미실행(web 변경 없음, `web/node_modules` 부재).
- [러너 14:57] brief accepted — 채택 — 근거 세 지점(:252-255 의 한 덩어리 조건, :70 의 `candidates[0]` 되돌림, :100-104 의 기존 500)이 현재 코드와 정확히 일치
- [러너 14:58] verify failed — 실패한 검증: cd web && npm test --silent (exit 1)
