## 2026-09-18
- 선택: 콘솔 Query DSL 화면이 Server 의 `total`·`offset`·`has_more` 를 쓰지 않아 첫 페이지만 보여 주던 것을 이전·다음으로 넘겨 볼 수 있게 함 (가치 3 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: v0.2.31 부터 `POST /api/v1/query/execute` 가 `offset` 을 받고 `total`·`has_more`·`next_offset` 을
  돌려주는데 `operationsPages.tsx` 의 Query DSL 패널은 `items`·`truncated` 만 읽어 "조건을 좁히거나 limit 을
  올리라" 고만 안내했다. 패널을 순수 컴포넌트 `QueryResultPanel` 로 떼어 제목에 조건 일치 전체 수, 오른쪽에
  "1–100 표시 · limit 100" 범위를 그리고, 한도 너머에 행이 있으면 자산 목록과 같은 이전·다음 버튼으로 같은
  질의를 이어 읽게 했다(offset 은 Server 가 되돌려 준 값, 다음 = offset + 표시 건수, 질의 실행은 항상 0 부터).
  끝을 지난 빈 페이지("이 페이지에는 자산이 없습니다")와 0건("조건에 맞는 자산이 없습니다")을 구분하고 거부된
  실행은 건수를 보이지 않는다. Server 는 손대지 않았다. USER_GUIDE 12.5·SERVER_INSTALLATION 19.4 의
  "다음 페이지가 없다" 는 문장을 고치고 캡처 스크립트로 `query-result.png` 만 다시 찍었다(다른 20장은 되돌림).
  검증: vitest 에 QueryResultPanel 렌더 6개 추가(150개 통과), 실제 Server(SQLite)+재빌드한 임베디드 콘솔을
  headless Chrome(CDP, `scripts/lib/guide-capture.mjs` 재사용)으로 열어 자산 7개·limit 3 으로 1→2→3 페이지·
  이전·재실행·1 페이지 삭제 뒤 끝을 지난 페이지·0건·거부까지 화면 글자와 버튼 disabled 상태를 확인(임시
  스크립트, 미커밋), `npm run build` → webui/dist 동기화(재빌드해 diff 없음 확인), `tsc -b`, `go test ./...`,
  `go vet`, `gofmt`, `go build`. 버전 범프·릴리즈 노트·PDF 재생성은 하지 않았다.
- 보류 아이디어: 여러 목록 핸들러가 `rows.Err()` 를 확인하지 않아 부분 결과를 200 으로 돌려줌 — listAgents·settings·자산 상세 sources/history/relations, 실패 주입 수단이 선행 과제 (가치 3 / 위험 1 / M) · cargo audit·govulncheck 를 main 에 대해 매일 cron 으로도 돌리기 — ci.yml 에 schedule 트리거 추가 (가치 3 / 위험 1 / S) · MCP `asset_get` 이 병합된 자산에 "asset not found" 만 답함 — `merged_into` 로 primary 안내 (가치 3 / 위험 2 / M) · 콘솔·외부 REST `assetRelations` 가 상한 없이 모든 edge 를 돌려줌 — MCP 와 같은 `limit`/`offset`·`has_more`, 콘솔 관계 패널 수정 동반 (가치 3 / 위험 2 / M) · Query DSL 화면의 limit 입력이 0·빈값·501 을 그대로 보내 Server 가 100/500 으로 고쳐 돌려줌 — 입력 시점에 1~500 으로 맞추고 되돌려 준 `limit` 을 편집기에도 반영 (가치 2 / 위험 1 / S)
