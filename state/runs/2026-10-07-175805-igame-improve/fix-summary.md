# 수리 요약 (시도 2)

- 문제: docs/api.md:10 이 `## 공통 규칙`(모든 limit/offset 목록) 자리에서 두 가지를 과장했다 — ① 유일 키 tiebreak 은 admin.go 의 세 목록뿐인데 catalog.go:130(`ORDER BY g.name`, games.name 은 UNIQUE 아님)·catalog.go:990·extended.go:581 에는 없다, ② "offset 을 끝까지 넘기면 각 행이 정확히 한 번씩" 은 페이지마다 다른 스냅샷이므로 고친 세 목록에서도 거짓이다. 둘 다 코드를 직접 열어 확인했고 지적이 맞았다.
- 고친 방법(문서 쪽으로 좁힘, SQL 무수정): 그 줄을 `/api/v1/admin/{users,audit,games}` 로 한정하고 "다른 목록은 아직 이 보증이 없다" 를 명시했으며, 보증을 "동점 행의 페이지 경계가 결정론" 까지로 낮추고 페이지 사이 INSERT/DELETE 는 어느 목록이든 창을 흔든다고 적었다.
- 같은 과장이 admin.go:576 주석("Every paged list here...")에도 있어 "this file 의 세 목록" 으로 좁히고 공개 카탈로그·플레이 기록·점수 모더레이션은 아직 아니라고 덧붙였다(주석 1곳, 동작 변화 없음).
- 검증: 일회용 PG17-alpine(포트 15441, 전용 pgcrypto 스키마)로 `make test-db` 전체 PASS, 새 세 테스트 `-run WhenTimestampsTie` 3/3 PASS, `go vet ./internal/...` 무출력, gofmt 깨끗, `api_test_` 잔여 스키마 0개, 컨테이너 제거.
- 남은 몫: catalog.go/extended.go 세 쿼리의 tiebreak 는 다음 회차(ideas.json). 문서가 더 이상 그것을 약속하지 않으므로 거짓 보증은 없다.
