# 회차 노트 2026-10-06-105908-qurio-improve — qurio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 10:59] base pinned — main@bd3f310
- [러너 10:59] autonomy release — 

## 정찰 노트
- 고른 이유: `appendUnique` quadratic 을 **실측으로 승격**했다 — 보류 목록은 가치 3(16000개 0.26s)으로 적어 두었지만, 실제 요청 한도(1MiB, `runtimeapi/sql.go:50`)에서 재면 양 방언 모두 **10.93s / 10.95s** 에 사유 7.3MB(88,304건)였다. 추정이 아니라 측정이고 직전 두 회차가 같은 기준(DB 권한 없이 코어 점유)으로 머지됐으므로 가치를 4로 올려 1순위로 했다. 제친 후보: `-count=1`(4/1/S)은 두 번 미머지라 운영자 규칙상 재제출 금지에 가깝고 워크플로가 보호 경로, Oracle CTE(4/3/S)는 verify-failed 원인 미파악, 웹 flake(4/2/M)는 결정적 재현이 없다.
- 확신 없는 곳: 어느 호출부까지 `reasonSet` 으로 바꿀지는 **내 권고**다 — 실측으로 정한 것은 "두 계층 모두 고쳐야 한다"(118ms/128ms) 뿐이다. `postgresTypeLiteralViolation` 자리(444)의 카디널리티는 직접 재지 않고 코드 모양으로만 무한이라 판단했다.
- 구현자가 조심할 것: map 을 순회해 사유를 만들면 순서가 무작위가 되어 "판정 불변" 증명이 즉시 깨진다 — 슬라이스가 순서의 유일한 출처여야 한다. 사유 **개수를 자르는 것은 별개 과제**(ideas.json 에 분리)이니 섞지 말 것. 직전 회차 교훈대로 "기존 테스트를 깨서 인과 확인" 이 공허할 수 있으니, 중복 제거·순서를 고정하는 테스트를 새로 쓰고 그것이 base 에서도 통과함을 먼저 보일 것.
- [러너 11:06] scout done — sqlsafe 위반 사유 누적(`appendUnique`)의 quadratic 제거 — 서로 다른 비허용 이름이 많은 질의 하나가 **양 방언 

## 구현 노트
- 무엇을 왜: `appendUnique` 가 사유마다 기존 전부와 문자열 비교를 해, 질의에서 뽑은 식별자를 품는 사유 자리 네 곳(+`AnalyzeDialect` 병합 루프)이 서로 다른 사유 개수에 quadratic 이었다. 순서 보존 슬라이스 + lazy 멤버십 인덱스(`reasonSet.add`/`list`)로 교체. 병합은 `result.Reasons` 로 시드해 중복 제거·순서 불변. 1MiB 한도에서 pg 10.9s→113ms, oracle 11.0s→89ms. 프로덕션 파일 1개.
- 확신 없는 곳·검증 못 한 것: 실 Oracle 인스턴스가 없어 Oracle 은 게이트 판정 수준까지만 증명했다(단 384판정 덤프에서 Oracle 행이 base 와 바이트 동일, 퍼즈 60만 drift 0). 웹 스위트·E2E 미실행(Go 전용 변경). 스케일 테스트 3개는 **타이밍 단정**(64000개 1초 예산)이라 원리상 극단적으로 느린 머신에서 흔들릴 수 있다 — 다만 red 5.4~5.9s / green 0.06~0.08s 로 양쪽 마진이 10배 이상이고 `t.Parallel()` 없이 직렬로 돈다.
- 일부러 하지 않은 것: ① 사유 **개수 상한** — 기준 3(판정 불변)을 깨고 UI 문자열이 생기는 별개 과제라 ideas.json 에 분리(1MiB 질의가 여전히 사유 8만여 개·약 6.8MB 를 낸다). ② `appendUnique`·`removeReason` 삭제 — `Analyze` 호출부(76·88)는 키워드 표 크기로 묶여 있어 그대로 두는 것이 변경면을 줄인다. ③ 판정 로직(허용목록·`callAdjacent`·`postgresAliasColumnList`·`fromItemContext`·`postgresCTEColumnList`·`scanOraclePolicy`) 한 줄도 안 건드림.
- 다음 역할이 조심할 것: 통합 테스트는 DB 가 있어야 돈다(폐기 PostgreSQL 17 + `POSTGRES_DSN`·`QURIO_INTEGRATION_DSN`·`QURIO_TEST_POSTGRES_DSN` 세 env 동일 지정 → fresh-install → migrations). 이번엔 55617 로 3회 연속 exit 0 / 30 ok. 과제서가 예고한 credential_race 2회차 결함은 **재현되지 않았다**(3회 전부 녹색) — 선재 결함이지만 재현 조건 미특정으로 기록만 했다. `reasonSetIndexThreshold`(sqlsafe.go:209)를 maxint 로 바꾸면 스케일 테스트 3개만 red·정합성 전부 green 으로 인과를 다시 확인할 수 있다.
- [러너 11:22] brief accepted — 채택 — 지목한 행(`appendUnique` 192, `postgresReadOnlyViolations` 418 의 네 호출부, `oracleReadOnlyViolations` 1067 의 1071·1096·1105, `AnalyzeD
- [러너 11:25] verify failed — 실패한 검증: cd web && npm test --silent (exit 1)
