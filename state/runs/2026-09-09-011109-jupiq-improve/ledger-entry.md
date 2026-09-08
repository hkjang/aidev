## 2026-09-09
- 선택: 큰 페이지 번호가 OFFSET을 음수로 뒤집지 않도록 pageBounds 제한 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `queryInt`가 쿼리스트링의 `page`를 상한 없이 그대로 넘겨 `store.pageBounds`의 `(page-1)*pageSize`가 int를 넘겨 음수 OFFSET으로 감쌌고, PostgreSQL이 이를 거부해 인증된 사용자가 `?page=99999999999999999` 하나로 목록·감사로그·사용자 상세 API를 500으로 만들 수 있었다. `page`를 offset이 표현 가능한 마지막 페이지로 잘라(실제 테이블 끝을 지난 값이라 빈 페이지가 정직한 답) 모든 `pageBounds` 호출자와 자체적으로 offset을 다시 계산하는 `user_detail.go` 경로까지 한 번에 막았고, 지금까지 테스트가 없던 이 순수 함수에 클램프·오버플로 방지·경계 페이지 보존을 덮는 테스트 3개를 추가했다. 클램프를 빼면 두 테스트가 실제로 실패하는 것을 확인했으며 `go vet ./...`, `go test -race ./...`, `scripts/check-version.sh`, `scripts/check-screenshots.mjs`, `npm run lint`, `npm test`(18파일 59개) 모두 통과했다.
- 보류 아이디어: `internal/api/helpers.go`의 사용되지 않는 `parseTimeQuery` 제거(이번에도 호출자 없음 재확인) / `collectPrometheus`가 metric마다 `featureEnabled`로 features 설정을 다시 읽는 중복 조회 제거 / `internal/config`의 `Load()` 테스트 신설(현재 커버리지 23.1%, 필수 환경변수 누락 집계·비밀번호 최소 길이·오류 우선순위 미검증) / 로그인 리미터 `succeeded`가 ip 키를 의도적으로 유지하는 동작에 대한 테스트·문서화
