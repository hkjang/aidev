# fix-summary (수리 2회차 → 커밋 5f3ca5e)

- 문제(재현됨): `postgresAliasColumnList` 가 호출 지점마다 왼쪽 끝까지 재탐색해 O(n^2) 이었다. 지적대로 `postgresFromItemContext` 가 한 원인이고, **두 번째 원인**은 `postgresCTEColumnList` — main 은 `) fn(` 에서 조기 return 해 도달하지 않았으나 이 브랜치에서는 fallthrough 되어 호출마다 `depths` 배열을 새로 만든다. 측정(`SELECT upper(x) || …`, 비-race): 192KB 5.95s, 384KB 4.37s(첫 원인만 고친 상태)/15.2s(memo 만 뺀 상태).
- 고침: 토큰 스트림당 `postgresTokenIndex` 1개(지연 생성)로 ① 괄호 짝(matchingLeft/Right)을 1회 O(n) 계산해 그룹 건너뛰기를 O(1) 점프로, ② 중첩 깊이 배열을 공유해 CTE 검사를 O(1) 로, ③ 왼쪽 탐색 결과를 (cursor, 콤마통과) 상태별로 메모이즈. 탐색 상한은 두지 않았다(정확도 손실 없이 선형이 됨).
- 검증: 384KB 23ms, 1MiB 59ms(지적의 3m53.9s, main 83ms 대비). 양방향 되돌림 확인 — memo 제거 시 15.2s 로, CTE 공유 제거 시 4.37s 로 새 테스트가 각각 실패한다. 판정 동등성은 b715a45 복사본 대 현재로 8949 질의 × 3 dialect A/B 프로브에서 **차이 0건**(저장소 SQL 리터럴 + 괄호/alias/CTE 조합 + 임의 토큰 4000개).
- 명령: `make check-go-format`, `go vet ./...`, `go build ./...`, `go test ./... -count=1` 전부 통과. `go test -race ./internal/domain/sqlsafe ./internal/domain/dbexec ./internal/runtimeapi` 통과(새 perf 테스트 -race 0.14s / 예산 2s).
- 미실행: dbexec 통합 테스트(`-tags=integration`, 폐기 DB 미준비)·e2e. 이번 변경은 sqlsafe 내부 계산 구조만 바꾸고 판정은 위 A/B 로 불변 확인했다.
