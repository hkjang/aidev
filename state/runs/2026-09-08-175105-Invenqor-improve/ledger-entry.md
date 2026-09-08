## 2026-09-08
- 선택: CSV 내보내기가 행 한도에 걸려도 완전한 파일처럼 보임 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `exportAssets`(기본 10,000행)와 `exportAudit`(기본 5,000행) 둘 다 한도만큼
  읽은 결과를 전부인 것처럼 써 내려갔다 — 자산 12,000건이면 `invenqor-assets.csv`
  가 10,000행으로 내려오고, 파일에도 이름에도 응답에도 나머지가 빠졌다는 표시가
  없다. 콘솔 밖에서 대조할 것 없이 읽히라고 만든 파일(인수인계·티켓·감사 증거)이라
  거절보다 나쁘다. 이제 둘 다 한도보다 한 행 더 읽어(정확히 한도에서 끝나는 결과는
  여전히 '완전'으로 보고) 잘린 경우에만 `X-Invenqor-Truncated`·`X-Invenqor-Row-Limit`
  헤더를 붙이고 첨부 파일 이름을 `...-partial.csv` 로 바꾼다(헤더가 사라진 뒤에도
  남는 유일한 신호 — 브라우저가 그 이름으로 저장하고 스프레드시트 제목 표시줄에
  뜬다). `audit.export` 감사 기록에도 `truncated` 를 남겨, 부분 추출이 "전체 로그를
  받아갔다"는 증거로 남지 않게 했다. 검증: `csv_truncation_test.go` 추가 — 두
  내보내기 각각에 대해 limit=3(5건 중 잘림)에서 행 수·두 헤더·partial 파일명을,
  limit=5(정확히 한도에서 끝나는 완전한 결과)에서 헤더 없음·평범한 파일명을 확인하고,
  `audit_logs.after_json` 의 `truncated` 가 [true false] 로 남는지 확인한다. 헛돌지
  않는지 보려고 `truncated` 계산을 `false` 로 바꿔 실패하는 것을 확인한 뒤 되돌렸다.
  `go test ./...` 를 SQLite fallback 과 실 PostgreSQL(`scripts/test-postgres.sh`)
  양쪽에서 전 패키지 통과, `go vet`·`go build`·`gofmt` 통과. `openapi.yaml` 의 두
  CSV 엔드포인트에 `limit` 파라미터와 새 응답 헤더를 기술하고 CI 와 같은
  `@redocly/cli@2.47.0 lint` 통과(경고 6개는 모두 기존 것, 내 줄과 무관). web·Rust
  는 손대지 않아 `npm`·`cargo` 는 돌리지 않았다. 문서 `.md` 와 버전 범프·릴리즈
  노트는 하지 않았다: `docs/*.md` 는 릴리즈 커밋에서 PDF 와 함께만 갱신된다.
- 보류 아이디어: MCP `asset_search` 가 0건일 때 실제 존재하는 type·status 값을 함께 돌려주어 '자산 없음'과 '필터 값 없음'을 구분하게 함 (가치 3 / 위험 2 / M) · Query DSL 에 `attributes.<키>` 존재/부재 연산자가 없어 `>= ""` 우회가 필요함 (가치 3 / 위험 2 / M) · `listAgents`·설정 목록/이력·자산 상세 루프·`executeQuery` 가 `rows.Err()` 를 확인하지 않아 부분 결과를 200 으로 돌려줌 — 드라이버 fault injection 없이는 테스트 불가 (가치 3 / 위험 1 / M) · 잘못된 API key 는 rate limit 을 전혀 소비하지 않아 키 추측만 무제한 (가치 2 / 위험 3 / M) · API key 로 한 행위도 감사 기록의 `actor_type` 이 `user` 라 소유자가 콘솔에서 한 일과 구분되지 않음 (가치 2 / 위험 3 / S)
