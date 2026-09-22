# 수리 요약 (수리 시도 2)

- 지적이 맞다. 재현: `TZ=America/New_York node -e` 로 `new Date().toISOString().slice(0,16)` 기본값을 `new Date(...)` 로 되읽으니 now+239.9분(Asia/Seoul 은 -540.1분). 서버 5분 상한과 만나 아메리카 클라이언트는 기본값 그대로 누르면 항상 400.
- 고침: 순수 함수 `web/src/datetimeLocal.ts#toDatetimeLocalValue(at)` 를 만들어 로컬 구성요소(getFullYear/getHours…)로 `YYYY-MM-DDTHH:mm` 를 만들고 `PersonPage.tsx:435` 기본값이 이를 쓰게 했다(표시 시각 오류도 같이 사라짐). DST 경계에서도 표시=저장이 같다.
- 테스트: `web/src/datetimeLocal.test.ts` 3건(NY/Seoul 벽시계, 세 TZ 왕복이 -45s, 0 채움). 되돌림 확인 — 옛 UTC 구현으로 바꾸면 2건 FAIL(NY 왕복 +14,355,000ms), 되돌리면 PASS.
- 검증: `npm run test -- --run` 15파일 112건 PASS, `npm run build`(tsc -b + vite) 성공, `go vet ./...`·`go test ./...` PASS. 커밋 1dc3fe3 (서버 변경은 그대로 둠).
- 남는 것: 이미 저장된 미래 행 정리는 여전히 안 한다. MemoriesPage 의 datetime-local 은 기본값이 빈 문자열이라 같은 결함이 없어 건드리지 않았다.
