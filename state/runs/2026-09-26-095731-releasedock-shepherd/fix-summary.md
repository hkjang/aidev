# 수리 요약 — PR #24 (커밋 1561c2f)

- 지적 두 건 모두 맞았다. streamLost 를 내리는 곳이 스트림 'open' 리스너뿐이라, 스트림이 끊긴 뒤 실행이 끝나면 '새로 고침' 으로 저장 로그를 전부 다시 읽어도 '끊긴 시점까지' 안내와 '다시 연결' 이 남았다(종료된 run 은 스트림을 열지 않아 'open' 이 영원히 안 옴).
- 고친 곳: (a) `loadStoredLogs` 성공 경로에서 `setStreamLost(false)` — 처음부터 다시 수집했으면 전제가 깨짐. (b) 안내 조건을 `streamLost && live` 로 좁힘 — 종료된 run 엔 잃을 연결도, 버튼이 열 스트림도 없음.
- 새 테스트 3건(프로덕션 라우트·App·EventSource 대역): 끊김 뒤 SUCCESS + 새로 고침 / 끊김 뒤 RUNNING 유지 + 새로 고침 / 끊김 뒤 SUCCESS + 로그 재수집 실패. 고치기 전 3건 모두 안내가 남아 실패함을 확인했다.
- (a)·(b) 를 각각 되돌려 매 수정이 서로 다른 테스트로 고정돼 있음을 확인했다((a)↔RUNNING 유지, (b)↔재수집 실패).
- 검증: `npx tsc -b --noEmit` 0, `npm test -- --run` 105/105, backend·runner `go test ./...` 통과(TEST_POSTGRES_DSN 미설정으로 통합 테스트는 Skip, 웹 전용 변경이라 그대로 둠).
