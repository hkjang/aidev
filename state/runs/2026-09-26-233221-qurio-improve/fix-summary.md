# fix-summary (수리 1회차 → 커밋 b715a45)

- 문제(재현됨): `postgresFromItemContext` 의 왼쪽 탐색이 FROM 목록의 최상위 콤마를 경계로 보지 않아 앞 from-item 의 `ON`/`USING`/`AS` 에 걸려 false → `FROM a JOIN b ON a.id=b.id, (SELECT 1) t(x)` 류가 RiskBlocked. 지적대로 맞았고 세 쿼리 모두 로컬에서 blocked 재현.
- 고침: 탐색 중 이 레벨의 콤마를 건넜는지(`crossedFromItemComma`)를 들고, 건넌 뒤에는 앞 from-item 소유 키워드(ON/USING/AS/CAST/IN…)가 탐색을 끝내지 않게 했다. 미탐 방지로 두 장치를 뒀다: (1) 짝 안 맞는 `(` 를 넘어 밖으로 나가면 플래그를 해제(그 콤마는 함수 인자 목록의 것) — `ON coalesce(x, (1) || evil_fn(y))` 는 계속 차단, (2) FROM 목록에 올 수 없는 절 키워드(SELECT/WHERE/BY/SET/VALUES/ORDER…)는 콤마 이후에도 항상 종료 — select/GROUP BY/ORDER BY 목록의 콤마가 FROM 문맥을 빌리지 못한다.
- 검증: `go test ./...` 전체 통과, gofmt·go vet 통과. 되돌림 검증 2방향 — 수리 전 sqlsafe.go 로는 새 allow 테스트가 실패하고, 괄호 해제 줄만 빼면 새 block 테스트(`ON COALESCE(a.x,(1)||fn(y))`)가 실패한다. 새로 허용한 6개 쿼리는 docker postgres:17-alpine(55481)에서 실제 실행 확인, 차단 쪽 공격 형태도 PG17 에서 유효 구문임을 확인(컨테이너 제거 완료).
- 테스트는 기존 두 테이블에 케이스만 추가했고 단언을 느슨하게 하거나 검증 명령을 고친 곳은 없다.
