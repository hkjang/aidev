## 구현 체크포인트
- 1 완료: 테스트 먼저 추가. null 4건만 503/500으로 FAIL, 나머지 19건 PASS, SKIP 없음. chat-null-red.log (exit 1).
- 2 완료: 첫 Decode 조건에 payload == nil 추가. 23건 green(0.737s) → nil 제거 시 null 4건만 red(0.753s) → 복원 23건 green(0.780s), SKIP 없음. 기존 8건 및 기본 공급자/model 정상 전달 유지.
- 3 완료: API 문서 갱신. 전체 go test -race -count=1 -v ./... exit 0(서버 156.319s; TEST_KEYCLOAK_ISSUER 미설정 E2E 1건 SKIP), make lint/go build ./.../git diff --check 순서대로 exit 0. 전용 DB 컨테이너 제거 및 e12916a 커밋 완료. ledger-entry.md/ideas.json/journal.md 기록 완료. 웹은 범위 밖 미실행.
